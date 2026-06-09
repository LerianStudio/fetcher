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
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/tablenorm"
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
// connection, which the underlying factory passes to the driver; connectivity
// then fails unless the target datasource permits passwordless auth.
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

// TestConnection performs the explicit connectivity check by lazily creating the
// underlying datasource through the host factory (which connects eagerly and, for
// MongoDB, pings). The resulting DataSource is cached for subsequent lifecycle
// calls. All failures are mapped to safe Engine errors that carry no secret, DSN,
// or driver internals.
func (c *Connector) TestConnection(ctx context.Context) error {
	return c.ensureConnected(ctx)
}

// ensureConnected lazily creates and caches the underlying datasource through the
// host factory. It is idempotent: a second call is a no-op once connected. It backs
// BOTH lifecycle entry points — TestConnection (the explicit execute-path connect)
// and DiscoverSchema (the plan-path connect, on which the Engine does NOT call
// TestConnection first), mirroring the schemacompat connector's lazy-connect so the
// extraction connector serves the discovery/validation path too.
func (c *Connector) ensureConnected(ctx context.Context) error {
	if c.ds != nil {
		return nil
	}

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

// DiscoverSchema reads the datasource schema through the underlying DataSource and
// maps it into a secret-free Engine SchemaSnapshot. It lazily connects on first use
// because the Engine's discovery/validation lifecycle is build -> discover -> close
// without a separate TestConnection step.
func (c *Connector) DiscoverSchema(ctx context.Context) (engine.SchemaSnapshot, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return engine.SchemaSnapshot{}, err
	}

	schemaArg := c.schemaArg(ctx)

	schema, err := c.ds.GetSchemaInfo(ctx, schemaArg)
	if err != nil {
		return engine.SchemaSnapshot{}, engine.NewEngineError(engine.CategoryUnavailable, "failed to discover datasource schema")
	}

	// The descriptor type is already validated at Build time, so a parse error here
	// is a defensive no-op default (empty type -> no default-schema stripping).
	dbType, _ := model.NewTypeFromString(c.descriptor.Type)

	return snapshotFromSchema(c.descriptor.ConfigName, dbType, schema), nil
}

// QueryStream executes the extraction request through the underlying DataSource
// and returns a RowCursor over the result. It maps the Engine ExtractionRequest
// into the (tables, filters) inputs the DataSource expects, keyed by this
// connector's config name.
//
// TODO(streaming): back this with real DB cursors. The underlying pkg/datasource
// Query fetches the whole result eagerly, so this materializes it and wraps it
// with engine.NewEagerCursor to satisfy the streaming contract. True DB-side
// streaming is a later optimization that does not change this seam's signature.
func (c *Connector) QueryStream(ctx context.Context, request engine.ExtractionRequest) (engine.RowCursor, error) {
	if c.ds == nil {
		return nil, engine.NewEngineError(engine.CategoryUnavailable, "datasource is not connected")
	}

	tables := tablesForConfig(c.descriptor.ConfigName, request)
	filters := filtersForConfig(c.descriptor.ConfigName, request)

	rows, err := c.ds.Query(ctx, tables, filters, nil)
	if err != nil {
		return nil, engine.NewEngineError(engine.CategoryUnavailable, "failed to query datasource")
	}

	return engine.NewEagerCursor(rows), nil
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

// buildConnection produces the *model.Connection the host factory connects with.
//
// It has TWO faithful paths:
//
//   - RICH-RECORD path (Worker wiring): when the descriptor carries the host's full
//     connection record in its opaque HostAttributes payload (seeded via
//     connectioncompat.DescriptorFromConnection), that record is used VERBATIM —
//     preserving the exact SSL config, schema, and ENCRYPTED password + key version
//     the host resolved, so the factory decrypts via the cryptor exactly like the
//     legacy queryDatabase path. This is the byte-identical extraction credential
//     path the Worker needs.
//   - RESOLVER path (Manager-style wiring): when no rich record is present, the
//     adapter assembles an in-memory connection from the secret-free descriptor and
//     the CredentialResolver's plaintext password (EncryptionKeyVersion left empty so
//     the factory treats it as plaintext).
func (c *Connector) buildConnection(ctx context.Context) (*model.Connection, error) {
	if rich := connectioncompat.ConnectionFromDescriptor(c.descriptor); rich != nil {
		return rich, nil
	}

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

// schemaArg derives the schema-name argument the underlying DataSource expects for
// DISCOVERY. Priority mirrors the schemacompat connector so the extraction and the
// schema-discovery paths resolve scope identically:
//
//  1. the per-request schema scope the host seeded (schemacompat.WithSchemaScope) —
//     this carries the schemas the requested tables reference (e.g. "accounting",
//     "reporting"), which the PostgreSQL/SQLServer discovery would otherwise narrow
//     to the default "public"/"dbo" and never surface the qualified tables;
//  2. the connection-level descriptor.Schema, when set;
//  3. nil — the underlying adapter applies its own default ("public"/"dbo").
//
// Reusing the schemacompat scope reader keeps ONE scope plumbing across the Manager
// validate path and the Worker extraction path. The extraction Query path is NOT
// affected: the PostgreSQL/SQLServer Query derives its own schemas from the
// qualified table keys, so only discovery needed the seeded scope.
func (c *Connector) schemaArg(ctx context.Context) []string {
	if scope := schemacompat.SchemaScope(ctx, c.descriptor.ConfigName); len(scope) > 0 {
		return scope
	}

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
// config name from the Engine request's opaque Filters payload and RECONSTRUCTS the
// typed map[table]map[field]job.FilterCondition the DataSource expects.
//
// The planner deep-copies host filters into a nested map[string]any (config -> table
// -> field -> value, value left opaque), so by the time the runner hands a step's
// filters back to the connector, the shape is map[string]any all the way down with
// the leaf field VALUE being whatever the host put there. The Worker mapper
// (mapFilters) puts a job.FilterCondition at each leaf, so this walks the nested
// any-maps and asserts each leaf back to job.FilterCondition. A missing config, a
// non-map level, or a non-FilterCondition leaf is skipped (no filtering for that
// path) rather than dropping the whole datasource's filters silently — the same
// conservative behavior the legacy path had for unrecognized filter shapes.
//
// This reconstruction is the EXECUTE half of the plan->execute filter round-trip;
// emitting the typed shape directly from the mapper would fail the planner's
// map[string]any assertion and drop filters entirely (the generic datasource would
// extract unfiltered).
func filtersForConfig(configName string, request engine.ExtractionRequest) map[string]map[string]job.FilterCondition {
	raw, ok := request.Filters[configName]
	if !ok {
		return nil
	}

	// Fast path: a caller that already supplied the typed shape (e.g. a test
	// constructing the request directly) is honored as-is.
	if typed, ok := raw.(map[string]map[string]job.FilterCondition); ok {
		return typed
	}

	tables, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	out := make(map[string]map[string]job.FilterCondition, len(tables))

	for table, tableRaw := range tables {
		fields, ok := tableRaw.(map[string]any)
		if !ok {
			continue
		}

		conditions := make(map[string]job.FilterCondition, len(fields))

		for field, value := range fields {
			if condition, ok := value.(job.FilterCondition); ok {
				conditions[field] = condition
			}
		}

		if len(conditions) > 0 {
			out[table] = conditions
		}
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// snapshotFromSchema maps the legacy *model.DataSourceSchema into a secret-free
// Engine SchemaSnapshot. A nil schema yields an empty snapshot carrying only the
// config name.
//
// It performs the two host-side reconciliations that keep the embedded Engine's
// LITERAL table match byte-identical to the legacy Worker's extraction path
// (owner decision Option 2, T-010):
//
//   - SYSTEM-TABLE EXCLUSION: reuses schemacompat.IsSystemTable so the snapshot the
//     extraction Engine validates against carries the same non-system tables the
//     schema-discovery Engine (Manager) does. Without it the extraction path and the
//     discovery path could disagree on what counts as a valid table.
//   - TABLE-NAME NORMALIZATION: reuses tablenorm (which wraps the SAME
//     schemautil.NormalizeTableNameForLookup the legacy adapters call) so a table the
//     legacy Worker addressed as "public.transactions" canonicalizes to the same
//     "transactions" the requested name canonicalizes to in the mapper, and the
//     Engine's exact match succeeds exactly where the legacy normalization did.
//
// Both reconciliations live HERE, at the enginecompat seam, never in the Engine
// core — the Engine stays literal and free of datasource-naming knowledge.
func snapshotFromSchema(configName string, dbType model.DBType, schema *model.DataSourceSchema) engine.SchemaSnapshot {
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

		if schemacompat.IsSystemTable(dbType, tableName) {
			continue
		}

		var fields []string
		if table != nil {
			fields = normalizeSnapshotFields(dbType, table.GetColumnsList())
		}

		tables = append(tables, engine.TableSnapshot{Name: tablenorm.NormalizeTable(dbType, tableName), Fields: fields})
	}

	snapshot.Tables = tables

	return snapshot
}

// normalizeSnapshotFields canonicalizes a snapshot's field names for the datasource
// type so the snapshot side and the request side (worker mapper) reconcile to the
// SAME identity before the Engine's literal field match. It is the IDENTITY for
// case-sensitive types (PG/MySQL/SQLServer) and folds to UPPERCASE for Oracle,
// mirroring tablenorm.NormalizeField on the request side. A nil input yields nil.
func normalizeSnapshotFields(dbType model.DBType, fields []string) []string {
	if !tablenorm.FoldsFieldCase(dbType) || fields == nil {
		return fields
	}

	out := make([]string, len(fields))
	for i, field := range fields {
		out[i] = tablenorm.NormalizeField(dbType, field)
	}

	return out
}
