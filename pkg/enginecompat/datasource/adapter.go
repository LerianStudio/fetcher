// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package enginecompatdatasource is the OPTIONAL compatibility bridge that
// adapts the host's existing pkg/datasource factory to the Engine's
// ConnectorFactory/Connector contracts.
//
// IMPORT DIRECTION IS ONE-WAY AND LOAD-BEARING. This package MAY import
// pkg/engine, pkg/datasource, pkg/model, pkg/model/datasource, and pkg/crypto —
// it is the single seam that is ALLOWED to know about model.Connection and the
// concrete driver factory. pkg/engine MUST NOT import this package or any
// concrete driver; the dependency boundary test in pkg/engine/dependency_test.go
// keeps that invariant enforced.
//
// The impedance it resolves: the Engine speaks in the secret-free
// ConnectionDescriptor, while the legacy factory wants a *model.Connection
// carrying a credential. The host supplies the secret through an injected
// CredentialResolver, and the adapter assembles an in-memory (plaintext)
// *model.Connection from descriptor + resolved password. The Engine core never
// learns about model.Connection.
//
// Construction (Build) is deliberately I/O-free: it only captures the descriptor
// and the injected seams. The legacy factory connects eagerly during
// construction (PostgreSQL dials, MongoDB pings), so the adapter defers the
// factory call to TestConnection — the contract's single, explicit connect step.
// This preserves the existing "MongoDB pings early" behavior while exposing it
// through the explicit Connector lifecycle.
package enginecompatdatasource

import (
	"context"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"

	"github.com/google/uuid"
)

// DataSourceFactory mirrors datasource.DataSourceFactory. It is redeclared here
// as the adapter's injection seam so unit tests can supply a FAKE factory
// without depending on a live driver, while production wiring passes
// datasource.NewDataSourceFromConnectionWithLogger(logger) directly (the
// signatures are identical).
type DataSourceFactory func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error)

// CredentialResolver resolves the plaintext credential for a connection from the
// secret-free descriptor. The HOST owns how the secret is recovered (e.g. via
// the Engine's CredentialProtector.Reveal path or an out-of-band secret store);
// the adapter only calls this seam at connect time and never persists the
// returned secret. A resolver returning the empty string yields a credential-free
// connection, which the underlying factory rejects per its own validation.
type CredentialResolver func(ctx context.Context, descriptor engine.ConnectionDescriptor) (string, error)

// ConnectorFactory is the Engine ConnectorFactory implementation that delegates
// to the host's pkg/datasource factory. It holds the injected factory, the
// credential resolver, and the optional cryptor the factory expects.
type ConnectorFactory struct {
	factory  DataSourceFactory
	resolver CredentialResolver
	cryptor  crypto.Cryptor
}

// NewConnectorFactory builds the compatibility ConnectorFactory. The factory and
// resolver are required injection seams; cryptor may be nil because the adapter
// always builds plaintext (in-memory) connections, for which the legacy factory
// tolerates a nil cryptor.
func NewConnectorFactory(factory DataSourceFactory, resolver CredentialResolver, cryptor crypto.Cryptor) *ConnectorFactory {
	return &ConnectorFactory{
		factory:  factory,
		resolver: resolver,
		cryptor:  cryptor,
	}
}

// Build constructs a Connector for the descriptor WITHOUT opening a connection.
// It validates only what can be checked offline (the datasource type) and defers
// the credential resolution and the eager-connecting factory call to
// TestConnection. A descriptor with an unrecognized type fails fast here as a
// CategoryValidation error rather than constructing an unusable connector.
func (f *ConnectorFactory) Build(_ context.Context, descriptor engine.ConnectionDescriptor) (engine.Connector, error) {
	if _, err := model.NewTypeFromString(descriptor.Type); err != nil {
		return nil, engine.UnknownConnectorTypeError(descriptor.Type)
	}

	return &Connector{
		descriptor: descriptor,
		factory:    f.factory,
		resolver:   f.resolver,
		cryptor:    f.cryptor,
	}, nil
}

// Connector is the Engine Connector implementation backed by a lazily-created
// datasource.DataSource. The underlying DataSource is created on TestConnection
// and reused by DiscoverSchema, Query, and Close.
type Connector struct {
	descriptor engine.ConnectionDescriptor
	factory    DataSourceFactory
	resolver   CredentialResolver
	cryptor    crypto.Cryptor

	ds datasource.DataSource
}

// TestConnection performs the explicit connectivity check. It resolves the
// credential, assembles an in-memory *model.Connection, and invokes the legacy
// factory — which connects eagerly (and, for MongoDB, pings). The resulting
// DataSource is cached for subsequent lifecycle calls. All failures are mapped
// to safe Engine errors that carry no secret, DSN, or driver internals.
func (c *Connector) TestConnection(ctx context.Context) error {
	conn, err := c.buildConnection(ctx)
	if err != nil {
		return err
	}

	ds, factoryErr := c.factory(ctx, conn, c.cryptor)
	if factoryErr != nil {
		// The factory's raw error may embed the DSN, credential, or driver text.
		// It is DELIBERATELY discarded from the returned message so no secret
		// crosses the Engine boundary, mirroring protectSecret in the Engine core.
		return engine.NewEngineError(engine.CategoryUnavailable, "failed to connect to datasource")
	}

	c.ds = ds

	return nil
}

// DiscoverSchema reads the datasource schema through the underlying DataSource
// and maps it into a secret-free Engine SchemaSnapshot.
func (c *Connector) DiscoverSchema(ctx context.Context) (engine.SchemaSnapshot, error) {
	if c.ds == nil {
		return engine.SchemaSnapshot{}, engine.NewEngineError(engine.CategoryUnavailable, "datasource is not connected")
	}

	schemaArg := c.schemaArg()

	schema, err := c.ds.GetSchemaInfo(ctx, schemaArg)
	if err != nil {
		return engine.SchemaSnapshot{}, engine.NewEngineError(engine.CategoryUnavailable, "failed to discover datasource schema")
	}

	return snapshotFromSchema(c.descriptor.ConfigName, schema), nil
}

// Query executes the extraction request through the underlying DataSource. It
// maps the Engine ExtractionRequest into the (tables, filters) inputs the
// DataSource expects, keyed by this connector's config name.
func (c *Connector) Query(ctx context.Context, request engine.ExtractionRequest) (map[string][]map[string]any, error) {
	if c.ds == nil {
		return nil, engine.NewEngineError(engine.CategoryUnavailable, "datasource is not connected")
	}

	tables := tablesForConfig(c.descriptor.ConfigName, request)
	filters := filtersForConfig(c.descriptor.ConfigName, request)

	rows, err := c.ds.Query(ctx, tables, filters, nil)
	if err != nil {
		return nil, engine.NewEngineError(engine.CategoryUnavailable, "failed to query datasource")
	}

	return rows, nil
}

// Close releases the underlying connection. It is safe before TestConnection
// (no datasource yet) and idempotent: after the first close the cached
// datasource is dropped so a second call is a no-op.
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

// buildConnection resolves the credential and assembles an in-memory
// *model.Connection from the secret-free descriptor. EncryptionKeyVersion is
// left empty so the legacy factory treats the resolved password as plaintext
// (its in-memory connection contract) rather than ciphertext to decrypt.
func (c *Connector) buildConnection(ctx context.Context) (*model.Connection, error) {
	password, err := c.resolver(ctx, c.descriptor)
	if err != nil {
		// The resolver's raw error may carry key versions or ciphertext; discard
		// it so nothing secret-adjacent crosses the Engine boundary.
		return nil, engine.NewEngineError(engine.CategoryUnavailable, "failed to resolve datasource credential")
	}

	dbType, err := model.NewTypeFromString(c.descriptor.Type)
	if err != nil {
		return nil, engine.UnknownConnectorTypeError(c.descriptor.Type)
	}

	conn := &model.Connection{
		ID:           parseConnectionID(c.descriptor.ID),
		ConfigName:   c.descriptor.ConfigName,
		Type:         dbType,
		Host:         c.descriptor.Host,
		Port:         c.descriptor.Port,
		DatabaseName: c.descriptor.DatabaseName,
		Username:     c.descriptor.Username,
		// EncryptionKeyVersion intentionally empty: plaintext/in-memory path.
	}

	if c.descriptor.Schema != "" {
		schema := c.descriptor.Schema
		conn.Schema = &schema
	}

	if c.descriptor.SSLMode != "" {
		conn.SSL = &model.SSLConfig{Mode: c.descriptor.SSLMode}
	}

	conn.SetPlaintextPassword(password)

	return conn, nil
}

// schemaArg derives the schema-name argument the underlying DataSource expects
// from the descriptor. An empty schema yields a nil slice, matching the legacy
// "all schemas" behavior.
func (c *Connector) schemaArg() []string {
	if strings.TrimSpace(c.descriptor.Schema) == "" {
		return nil
	}

	return []string{c.descriptor.Schema}
}

// parseConnectionID converts the descriptor's string ID into a uuid.UUID. A
// blank or malformed ID yields uuid.Nil; identity for the in-memory connection
// is anchored by config name within the tenant scope, not the UUID, so a
// non-parseable host ID is non-fatal here.
func parseConnectionID(id string) uuid.UUID {
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return uuid.Nil
	}

	return parsed
}

// tablesForConfig extracts the table/field selection for this connector's config
// name from the Engine request, in the map[table][]fields shape the DataSource
// Query expects.
func tablesForConfig(configName string, request engine.ExtractionRequest) map[string][]string {
	selection, ok := request.MappedFields[configName]
	if !ok || len(selection) == 0 {
		return map[string][]string{}
	}

	tables := make(map[string][]string, len(selection))
	for table, fields := range selection {
		tables[table] = fields
	}

	return tables
}

// filtersForConfig extracts the per-table filter conditions for this connector's
// config name from the Engine request's opaque Filters payload. The contract
// keeps Filters untyped; the adapter interprets the host-supplied shape
// map[config]map[table]map[field]job.FilterCondition and returns the inner
// map[table]map[field]FilterCondition the DataSource expects. A missing or
// mismatched shape yields a nil filter map (no filtering).
func filtersForConfig(configName string, request engine.ExtractionRequest) map[string]map[string]job.FilterCondition {
	raw, ok := request.Filters[configName]
	if !ok {
		return nil
	}

	typed, ok := raw.(map[string]map[string]job.FilterCondition)
	if !ok {
		return nil
	}

	return typed
}

// snapshotFromSchema maps the legacy *model.DataSourceSchema into a secret-free
// Engine SchemaSnapshot. A nil schema yields an empty snapshot carrying only the
// config name.
func snapshotFromSchema(configName string, schema *model.DataSourceSchema) engine.SchemaSnapshot {
	snapshot := engine.SchemaSnapshot{ConfigName: configName}
	if schema == nil {
		return snapshot
	}

	if schema.ConfigName != "" {
		snapshot.ConfigName = schema.ConfigName
	}

	snapshot.CapturedAt = schema.CachedAt

	if len(schema.Tables) == 0 {
		return snapshot
	}

	tables := make([]engine.TableSnapshot, 0, len(schema.Tables))
	for name, table := range schema.Tables {
		tableName := name
		if table != nil && table.TableName != "" {
			tableName = table.TableName
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
