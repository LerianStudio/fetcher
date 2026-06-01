// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package enginecompatdatasource_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/engine"
	enginecompatdatasource "github.com/LerianStudio/fetcher/pkg/enginecompat/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/lib-observability/log"

	"github.com/google/uuid"
)

// fakeDataSource is a host-side datasource.DataSource double. It records the
// lifecycle calls the adapter drives through it and returns canned data so a
// test can prove the Engine Connector delegates to the underlying DataSource
// without touching a real driver.
type fakeDataSource struct {
	config       datasource.DataSourceConfig
	queryResult  map[string][]map[string]any
	schemaResult *model.DataSourceSchema
	queryErr     error
	schemaErr    error
	closeErr     error

	closed       bool
	queryTables  map[string][]string
	queryFilters map[string]map[string]job.FilterCondition
}

func (f *fakeDataSource) GetConfig() datasource.DataSourceConfig { return f.config }
func (f *fakeDataSource) GetType() string                        { return f.config.Type }

func (f *fakeDataSource) Connect(_ context.Context, _ log.Logger) error { return nil }

func (f *fakeDataSource) Close(_ context.Context) error {
	f.closed = true
	return f.closeErr
}

func (f *fakeDataSource) Query(
	_ context.Context,
	tables map[string][]string,
	filters map[string]map[string]job.FilterCondition,
	_ log.Logger,
) (map[string][]map[string]any, error) {
	f.queryTables = tables
	f.queryFilters = filters

	if f.queryErr != nil {
		return nil, f.queryErr
	}

	return f.queryResult, nil
}

func (f *fakeDataSource) GetSchemaInfo(_ context.Context, _ []string) (*model.DataSourceSchema, error) {
	if f.schemaErr != nil {
		return nil, f.schemaErr
	}

	return f.schemaResult, nil
}

// capturedFactoryCall records the inputs the adapter passed into the injected
// DataSourceFactory, so a test can assert the descriptor + credential mapping.
type capturedFactoryCall struct {
	conn    *model.Connection
	cryptor crypto.Cryptor
}

// newFakeFactory returns a DataSourceFactory that records its inputs and returns
// the supplied datasource (or error). It performs NO real I/O.
func newFakeFactory(ds datasource.DataSource, err error, captured *capturedFactoryCall) enginecompatdatasource.DataSourceFactory {
	return func(_ context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		captured.conn = conn
		captured.cryptor = cryptor

		if err != nil {
			return nil, err
		}

		return ds, nil
	}
}

func sampleDescriptor() engine.ConnectionDescriptor {
	return engine.ConnectionDescriptor{
		ID:           "11111111-1111-1111-1111-111111111111",
		ConfigName:   "pg-main",
		Type:         "POSTGRESQL",
		Host:         "db.internal",
		Port:         5432,
		DatabaseName: "ledger",
		Schema:       "public",
		Username:     "reader",
		SSLMode:      "require",
	}
}

// constResolver returns a credential resolver that always yields the given
// password, recording that it was asked for a credential.
func constResolver(password string, called *bool) enginecompatdatasource.CredentialResolver {
	return func(_ context.Context, _ engine.ConnectionDescriptor) (string, error) {
		if called != nil {
			*called = true
		}

		return password, nil
	}
}

func TestBuild_IsSideEffectFree_NoFactoryCallUntilTestConnection(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	factory := newFakeFactory(&fakeDataSource{}, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("secret", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatalf("Build: returned nil connector")
	}

	// Build MUST NOT call the factory (which would open a connection): the
	// factory call is deferred to TestConnection.
	if captured.conn != nil {
		t.Fatalf("Build must be side-effect free; factory was invoked with %#v", captured.conn)
	}
}

func TestTestConnection_MapsDescriptorAndCredentialIntoFactoryConnection(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	cryptor := crypto.Cryptor(nil)
	factory := newFakeFactory(&fakeDataSource{}, nil, captured)
	resolverCalled := false

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("s3cr3t", &resolverCalled), cryptor)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}

	if !resolverCalled {
		t.Fatalf("TestConnection must resolve the credential via the injected resolver")
	}
	if captured.conn == nil {
		t.Fatalf("TestConnection must invoke the factory with a *model.Connection")
	}

	got := captured.conn
	if got.ConfigName != "pg-main" {
		t.Errorf("ConfigName = %q, want pg-main", got.ConfigName)
	}
	if got.Type != model.TypePostgreSQL {
		t.Errorf("Type = %q, want %q", got.Type, model.TypePostgreSQL)
	}
	if got.Host != "db.internal" {
		t.Errorf("Host = %q, want db.internal", got.Host)
	}
	if got.Port != 5432 {
		t.Errorf("Port = %d, want 5432", got.Port)
	}
	if got.DatabaseName != "ledger" {
		t.Errorf("DatabaseName = %q, want ledger", got.DatabaseName)
	}
	if got.Username != "reader" {
		t.Errorf("Username = %q, want reader", got.Username)
	}
	if got.GetPlaintextPassword() != "s3cr3t" {
		t.Errorf("resolved password not mapped into connection; got %q", got.GetPlaintextPassword())
	}
	// Secret-free descriptor produces an in-memory (plaintext) connection: the
	// EncryptionKeyVersion must be empty so the factory treats the resolved
	// password as plaintext, not as ciphertext to decrypt.
	if got.EncryptionKeyVersion != "" {
		t.Errorf("EncryptionKeyVersion = %q, want empty (plaintext path)", got.EncryptionKeyVersion)
	}
	if got.Schema == nil || *got.Schema != "public" {
		t.Errorf("Schema not mapped; got %v", got.Schema)
	}
	if got.SSL == nil || got.SSL.Mode != "require" {
		t.Errorf("SSL mode not mapped; got %v", got.SSL)
	}
	if got.ProductName != "" {
		t.Errorf("ProductName must stay empty (Engine has no product concept); got %q", got.ProductName)
	}
}

func TestBuild_UnknownTypeReturnsNotFoundWithoutCallingFactory(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	factory := newFakeFactory(&fakeDataSource{}, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	descriptor := sampleDescriptor()
	descriptor.Type = "CASSANDRA"

	// An unknown datasource type is resolvable offline, so Build fails fast with
	// the stable Engine UnknownConnectorTypeError (CategoryNotFound) rather than
	// constructing an unusable connector.
	conn, err := adapter.Build(context.Background(), descriptor)
	assertEngineError(t, err, engine.CategoryNotFound)

	if conn != nil {
		t.Fatalf("Build must not return a connector for an unknown type")
	}
	if captured.conn != nil {
		t.Fatalf("unknown type must not reach the factory")
	}
}

func TestTestConnection_FactoryErrorMapsToUnavailableWithoutLeakingSecret(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	// A driver error that embeds DSN/credential material — the adapter MUST NOT
	// surface any of this text across the Engine boundary.
	driverErr := errors.New("dial tcp db.internal:5432: password=s3cr3t authentication failed for user reader")
	factory := newFakeFactory(nil, driverErr, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("s3cr3t", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	err = conn.TestConnection(context.Background())
	assertEngineError(t, err, engine.CategoryUnavailable)

	msg := err.Error()
	for _, leak := range []string{"s3cr3t", "db.internal", "reader", "password="} {
		if strings.Contains(msg, leak) {
			t.Fatalf("Engine error leaked secret material %q in %q", leak, msg)
		}
	}
}

func TestTestConnection_ResolverErrorMapsToUnavailableWithoutLeakingSecret(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	factory := newFakeFactory(&fakeDataSource{}, nil, captured)

	resolver := func(_ context.Context, _ engine.ConnectionDescriptor) (string, error) {
		return "", errors.New("kms reveal failed for key version 7 ciphertext deadbeef")
	}

	adapter := enginecompatdatasource.NewConnectorFactory(factory, resolver, nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	err = conn.TestConnection(context.Background())
	assertEngineError(t, err, engine.CategoryUnavailable)

	if captured.conn != nil {
		t.Fatalf("factory must not be reached when credential resolution fails")
	}
	if strings.Contains(err.Error(), "deadbeef") || strings.Contains(err.Error(), "key version 7") {
		t.Fatalf("Engine error leaked resolver internals: %q", err.Error())
	}
}

func TestConnector_QueryDelegatesToUnderlyingDataSource(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	ds := &fakeDataSource{
		config:      datasource.DataSourceConfig{Type: "POSTGRESQL"},
		queryResult: map[string][]map[string]any{"public.accounts": {{"id": 1}}},
	}
	factory := newFakeFactory(ds, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"pg-main": {"public.accounts": {"id", "name"}},
		},
	}

	rows, err := conn.Query(context.Background(), req)
	if err != nil {
		t.Fatalf("Query: unexpected error: %v", err)
	}

	if len(rows["public.accounts"]) != 1 {
		t.Fatalf("Query did not return underlying rows; got %#v", rows)
	}
	if ds.queryTables == nil {
		t.Fatalf("Query must delegate the table selection to the underlying DataSource")
	}
	if _, ok := ds.queryTables["public.accounts"]; !ok {
		t.Fatalf("table selection not mapped into the DataSource; got %#v", ds.queryTables)
	}
}

func TestConnector_QueryBeforeTestConnectionFails(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	factory := newFakeFactory(&fakeDataSource{config: datasource.DataSourceConfig{Type: "POSTGRESQL"}}, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	_, err = conn.Query(context.Background(), engine.ExtractionRequest{})
	assertEngineError(t, err, engine.CategoryUnavailable)
}

func TestConnector_DiscoverSchemaDelegatesAndMapsSnapshot(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	schema := model.NewDataSourceSchema("pg-main")
	schema.AddTable("public.accounts", []string{"id", "name"})

	ds := &fakeDataSource{
		config:       datasource.DataSourceConfig{Type: "POSTGRESQL"},
		schemaResult: schema,
	}
	factory := newFakeFactory(ds, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}

	snapshot, err := conn.DiscoverSchema(context.Background())
	if err != nil {
		t.Fatalf("DiscoverSchema: unexpected error: %v", err)
	}

	if snapshot.ConfigName != "pg-main" {
		t.Errorf("snapshot ConfigName = %q, want pg-main", snapshot.ConfigName)
	}
	if !snapshot.HasTable("public.accounts") {
		t.Fatalf("snapshot missing table public.accounts; got %#v", snapshot.Tables)
	}

	var fields []string
	for _, table := range snapshot.Tables {
		if table.Name == "public.accounts" {
			fields = table.Fields
		}
	}
	if len(fields) != 2 {
		t.Errorf("snapshot field count = %d, want 2 (got %#v)", len(fields), fields)
	}
}

func TestConnector_CloseDelegatesAndIsIdempotent(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	ds := &fakeDataSource{config: datasource.DataSourceConfig{Type: "POSTGRESQL"}}
	factory := newFakeFactory(ds, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	// Close before TestConnection (no underlying datasource yet) must be safe.
	if err := conn.Close(context.Background()); err != nil {
		t.Fatalf("Close before connect: unexpected error: %v", err)
	}

	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}
	if err := conn.Close(context.Background()); err != nil {
		t.Fatalf("Close: unexpected error: %v", err)
	}
	if !ds.closed {
		t.Fatalf("Close must delegate to the underlying DataSource")
	}

	// Double close must not panic and must stay safe.
	if err := conn.Close(context.Background()); err != nil {
		t.Fatalf("double Close: unexpected error: %v", err)
	}
}

func TestConnector_QueryMapsTypedFiltersForConfig(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	ds := &fakeDataSource{config: datasource.DataSourceConfig{Type: "POSTGRESQL"}}
	factory := newFakeFactory(ds, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"pg-main": {"public.accounts": {"id"}},
		},
		Filters: map[string]any{
			"pg-main": map[string]map[string]job.FilterCondition{
				"public.accounts": {"status": {Equals: []any{"active"}}},
			},
			// A mismatched-shape entry for another config must be ignored, not panic.
			"other": "not-a-filter-map",
		},
	}

	if _, err := conn.Query(context.Background(), req); err != nil {
		t.Fatalf("Query: unexpected error: %v", err)
	}

	if ds.queryFilters == nil {
		t.Fatalf("typed filters for config were not mapped into the DataSource")
	}
	cond, ok := ds.queryFilters["public.accounts"]["status"]
	if !ok {
		t.Fatalf("filter condition not mapped; got %#v", ds.queryFilters)
	}
	if len(cond.Equals) != 1 || cond.Equals[0] != "active" {
		t.Fatalf("filter condition value not preserved; got %#v", cond)
	}
}

func TestConnector_QueryMismatchedFilterShapeYieldsNoFilters(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	ds := &fakeDataSource{config: datasource.DataSourceConfig{Type: "POSTGRESQL"}}
	factory := newFakeFactory(ds, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}

	req := engine.ExtractionRequest{
		Filters: map[string]any{"pg-main": 42}, // wrong type for this config
	}

	if _, err := conn.Query(context.Background(), req); err != nil {
		t.Fatalf("Query: unexpected error: %v", err)
	}
	if ds.queryFilters != nil {
		t.Fatalf("mismatched filter shape must yield nil filters; got %#v", ds.queryFilters)
	}
}

func TestConnector_QueryErrorMapsToUnavailable(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	ds := &fakeDataSource{
		config:   datasource.DataSourceConfig{Type: "POSTGRESQL"},
		queryErr: errors.New("relation public.accounts does not exist for user reader"),
	}
	factory := newFakeFactory(ds, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}

	_, err = conn.Query(context.Background(), engine.ExtractionRequest{})
	assertEngineError(t, err, engine.CategoryUnavailable)
	if strings.Contains(err.Error(), "reader") {
		t.Fatalf("query error leaked driver internals: %q", err.Error())
	}
}

func TestConnector_DiscoverSchemaBeforeConnectFails(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	factory := newFakeFactory(&fakeDataSource{}, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	_, err = conn.DiscoverSchema(context.Background())
	assertEngineError(t, err, engine.CategoryUnavailable)
}

func TestConnector_DiscoverSchemaErrorMapsToUnavailable(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	ds := &fakeDataSource{
		config:    datasource.DataSourceConfig{Type: "POSTGRESQL"},
		schemaErr: errors.New("permission denied for schema public, user reader"),
	}
	factory := newFakeFactory(ds, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}

	_, err = conn.DiscoverSchema(context.Background())
	assertEngineError(t, err, engine.CategoryUnavailable)
	if strings.Contains(err.Error(), "reader") {
		t.Fatalf("schema error leaked driver internals: %q", err.Error())
	}
}

func TestConnector_DiscoverSchemaWithoutSchemaSelectorRequestsAllSchemas(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	schema := model.NewDataSourceSchema("pg-main")
	ds := &fakeDataSource{config: datasource.DataSourceConfig{Type: "POSTGRESQL"}, schemaResult: schema}
	factory := newFakeFactory(ds, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	descriptor := sampleDescriptor()
	descriptor.Schema = "" // no schema selector

	conn, err := adapter.Build(context.Background(), descriptor)
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}

	snapshot, err := conn.DiscoverSchema(context.Background())
	if err != nil {
		t.Fatalf("DiscoverSchema: unexpected error: %v", err)
	}
	if snapshot.ConfigName != "pg-main" {
		t.Fatalf("snapshot ConfigName = %q, want pg-main", snapshot.ConfigName)
	}
}

func TestConnector_CloseErrorMapsToUnavailable(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	ds := &fakeDataSource{
		config:   datasource.DataSourceConfig{Type: "POSTGRESQL"},
		closeErr: errors.New("connection reset by peer at db.internal:5432"),
	}
	factory := newFakeFactory(ds, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}

	err = conn.Close(context.Background())
	assertEngineError(t, err, engine.CategoryUnavailable)
	if strings.Contains(err.Error(), "db.internal") {
		t.Fatalf("close error leaked driver internals: %q", err.Error())
	}
}

func TestBuild_BlankIDProducesNilUUIDConnection(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	factory := newFakeFactory(&fakeDataSource{}, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	descriptor := sampleDescriptor()
	descriptor.ID = "not-a-uuid"

	conn, err := adapter.Build(context.Background(), descriptor)
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if err := conn.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}

	// A malformed host ID is non-fatal: identity is anchored by config name, so
	// the assembled connection carries the nil UUID rather than failing.
	if captured.conn.ID != uuid.Nil {
		t.Fatalf("malformed ID should map to uuid.Nil; got %v", captured.conn.ID)
	}
}

// compile-time proof the adapter satisfies the Engine contracts.
var (
	_ engine.ConnectorFactory = (*enginecompatdatasource.ConnectorFactory)(nil)
	_ engine.Connector        = (*enginecompatdatasource.Connector)(nil)
)

func assertEngineError(t *testing.T, err error, want engine.ErrorCategory) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected *engine.EngineError with category %q, got nil", want)
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("expected *engine.EngineError, got %T: %v", err, err)
	}
	if engErr.Category != want {
		t.Fatalf("error category = %q, want %q (msg %q)", engErr.Category, want, engErr.Message)
	}
}