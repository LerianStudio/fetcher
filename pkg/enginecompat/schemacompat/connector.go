// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package schemacompat

import (
	"context"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/model"
	datasourceModel "github.com/LerianStudio/fetcher/pkg/model/datasource"
)

// ConnectorFactory is the schema Engine's engine.ConnectorFactory. It rebuilds
// the rich *model.Connection the host resolved (carried through the descriptor's
// opaque host payload) and constructs a datasource through the host's existing
// factory — preserving the legacy decrypt + connect behavior exactly. Build is
// I/O-free; the eager-connecting factory call is deferred to DiscoverSchema's
// lazy connect, matching the Engine's build -> discover -> close lifecycle for
// schema (the Engine does NOT call TestConnection on the discovery path).
type ConnectorFactory struct {
	factory datasource.DataSourceFactory
	cryptor crypto.Cryptor
}

// NewConnectorFactory builds the schema connector factory over the host's
// datasource factory and cryptor. The cryptor is required for encrypted
// (external) connections and ignored for in-memory (internal) ones; the host
// factory already enforces that contract.
func NewConnectorFactory(factory datasource.DataSourceFactory, cryptor crypto.Cryptor) *ConnectorFactory {
	return &ConnectorFactory{factory: factory, cryptor: cryptor}
}

// Build constructs a Connector for the descriptor without opening a connection.
// It validates only the datasource type offline and unpacks the rich connection
// from the opaque host payload; a descriptor missing its host record is a
// malformed validation error rather than a silent nil connector.
func (f *ConnectorFactory) Build(_ context.Context, descriptor engine.ConnectionDescriptor) (engine.Connector, error) {
	if _, err := model.NewTypeFromString(descriptor.Type); err != nil {
		return nil, engine.UnknownConnectorTypeError(descriptor.Type)
	}

	conn := connectioncompat.ConnectionFromDescriptor(descriptor)
	if conn == nil {
		return nil, engine.NewEngineError(engine.CategoryValidation, "connection record is missing from descriptor")
	}

	return &Connector{
		conn:    conn,
		factory: f.factory,
		cryptor: f.cryptor,
	}, nil
}

// Connector is the schema Engine's engine.Connector. It lazily creates the
// underlying datasource on first DiscoverSchema (the Engine omits the explicit
// TestConnection step on the discovery path) and excludes system tables from the
// snapshot BEFORE it crosses the Engine boundary, preserving the legacy Manager's
// type-aware system-table filtering as a host concern (the Engine validates
// literal snapshot names and performs no datasource-type filtering).
type Connector struct {
	conn    *model.Connection
	factory datasource.DataSourceFactory
	cryptor crypto.Cryptor

	ds datasourceModel.DataSource
}

// TestConnection is the explicit connect step. The schema discovery path does
// not call it, but it is provided for contract completeness and shares the lazy
// connect so a host that does call it gets the connected datasource.
func (c *Connector) TestConnection(ctx context.Context) error {
	return c.ensureConnected(ctx)
}

// DiscoverSchema reads the datasource schema through the host datasource, filters
// system tables by datasource type, and returns a secret-free Engine snapshot.
// It lazily connects on first use because the Engine's discovery lifecycle is
// build -> discover -> close without a separate TestConnection step. Failures are
// mapped to safe Engine errors that carry no secret, DSN, or driver internals.
func (c *Connector) DiscoverSchema(ctx context.Context) (engine.SchemaSnapshot, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return engine.SchemaSnapshot{}, err
	}

	schema, err := c.ds.GetSchemaInfo(ctx, c.schemaArg(ctx))
	if err != nil {
		return engine.SchemaSnapshot{}, engine.NewEngineError(engine.CategoryUnavailable, "failed to discover datasource schema")
	}

	return c.snapshot(schema), nil
}

// Query is unsupported on the schema path; the schema Engine performs discovery
// only. Extraction runs through the dedicated enginecompat/datasource connector.
func (c *Connector) Query(context.Context, engine.ExtractionRequest) (map[string][]map[string]any, error) {
	return nil, engine.NewEngineError(engine.CategoryValidation, "schema connector supports discovery only")
}

// Close releases the underlying connection. It is safe before connect and
// idempotent: after the first close the cached datasource is dropped.
func (c *Connector) Close(ctx context.Context) error {
	if c.ds == nil {
		return nil
	}

	ds := c.ds
	c.ds = nil

	if err := ds.Close(ctx); err != nil {
		return engine.NewEngineError(engine.CategoryUnavailable, "failed to close datasource")
	}

	return nil
}

// ensureConnected lazily creates the underlying datasource through the host
// factory. This is the CONNECT stage of the schema discovery path: the Engine's
// discovery lifecycle is build -> discover -> close without a separate
// TestConnection, so the connect happens here on first use.
//
// A factory failure is wrapped as a CategoryConnect EngineError so the Engine's
// resolveSchema passes the STAGE through unchanged (a connect failure is told
// apart from a discovery-read failure, which stays CategoryUnavailable) and the
// Manager can render it as "Database Connection Error" — the legacy two-title
// contract. The wrapper keeps the underlying cause REACHABLE for errors.As /
// errors.Is so the host still recognizes a typed pkg.ValidationError (e.g.
// FET-0414 host-safety rejection) and maps it to HTTP 400, while the wrapper's
// rendered message stays redacted: the raw cause MAY embed a DSN or credential,
// so it never appears in the safe boundary message. A nil datasource with no
// error is a factory misconfiguration mapped to the same connect category.
func (c *Connector) ensureConnected(ctx context.Context) error {
	if c.ds != nil {
		return nil
	}

	if c.factory == nil {
		return engine.NewEngineError(engine.CategoryConnect, "datasource factory is not configured")
	}

	ds, err := c.factory(ctx, c.conn, c.cryptor)
	if err != nil {
		return engine.NewWrappedEngineError(engine.CategoryConnect, "failed to connect to datasource", err)
	}

	if ds == nil {
		return engine.NewEngineError(engine.CategoryConnect, "datasource factory returned nil datasource")
	}

	c.ds = ds

	return nil
}

// schemaArg derives the schema-name argument the host datasource expects.
// Priority: the per-request schema scope seeded by the host (which preserves the
// legacy multi-schema discovery) > an explicit connection Schema > nil (the
// adapter applies its default, e.g. PostgreSQL "public").
func (c *Connector) schemaArg(ctx context.Context) []string {
	if scope := SchemaScope(ctx, c.conn.ConfigName); len(scope) > 0 {
		return scope
	}

	if c.conn.Schema != nil && strings.TrimSpace(*c.conn.Schema) != "" {
		return []string{*c.conn.Schema}
	}

	return nil
}

// snapshot maps the host *model.DataSourceSchema into a secret-free Engine
// snapshot, EXCLUDING system tables by datasource type. This is the host layer
// that keeps system-table conventions (pg_*, Oracle SYS, db_*, Mongo system.*)
// out of the Engine core: the snapshot that crosses the boundary is already
// filtered, so the Engine never sees a system table as a valid one.
func (c *Connector) snapshot(schema *model.DataSourceSchema) engine.SchemaSnapshot {
	snapshot := engine.SchemaSnapshot{ConfigName: c.conn.ConfigName}
	if schema == nil || schema.Tables == nil {
		return snapshot
	}

	if schema.ConfigName != "" {
		snapshot.ConfigName = schema.ConfigName
	}

	snapshot.CapturedAt = schema.CachedAt

	tables := make([]engine.TableSnapshot, 0, len(schema.Tables))
	for name, table := range schema.Tables {
		tableName := name
		if table != nil && table.TableName != "" {
			tableName = table.TableName
		}

		if IsSystemTable(c.conn.Type, tableName) {
			continue
		}

		var fields []string
		if table != nil {
			fields = table.GetColumnsList()
		}

		tables = append(tables, engine.TableSnapshot{Name: tableName, Fields: fields})
	}

	snapshot.Tables = tables

	return snapshot
}
