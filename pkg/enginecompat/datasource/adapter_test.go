// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package enginecompatdatasource_test

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	enginecompatdatasource "github.com/LerianStudio/fetcher/v2/pkg/enginecompat/datasource"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/tablenorm"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/v2/pkg/model/job"
	"github.com/LerianStudio/lib-observability/log"

	"github.com/google/uuid"
)

// queryStreamRows opens the connector's QueryStream and materializes the cursor
// back into the legacy map[table][]rows shape so the existing row-shape
// assertions stay meaningful after the eager Query was replaced by a streaming
// cursor. It fails the test on an open or iteration error.
func queryStreamRows(t *testing.T, conn engine.Connector, req engine.ExtractionRequest) map[string][]map[string]any {
	t.Helper()

	cursor, err := conn.QueryStream(context.Background(), req)
	if err != nil {
		t.Fatalf("QueryStream: unexpected error: %v", err)
	}

	defer func() { _ = cursor.Close(context.Background()) }()

	rows := make(map[string][]map[string]any)
	for cursor.Next(context.Background()) {
		table, row := cursor.Row()
		rows[table] = append(rows[table], row)
	}

	if err := cursor.Err(); err != nil {
		t.Fatalf("cursor.Err: unexpected error: %v", err)
	}

	return rows
}

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

	closed        bool
	queryTables   map[string][]string
	queryFilters  map[string]map[string]job.FilterCondition
	schemaScopeIn []string
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

func (f *fakeDataSource) GetSchemaInfo(_ context.Context, schemas []string) (*model.DataSourceSchema, error) {
	f.schemaScopeIn = schemas

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

	rows := queryStreamRows(t, conn, req)

	if len(rows["public.accounts"]) != 1 {
		t.Fatalf("QueryStream did not return underlying rows; got %#v", rows)
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

	_, err = conn.QueryStream(context.Background(), engine.ExtractionRequest{})
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
	// Option 2 (T-010): the snapshot table identity is canonicalized host-side via
	// tablenorm, so the PostgreSQL public-schema table "public.accounts" is keyed by
	// its canonical "accounts". This is what lets the Engine's literal match accept a
	// request for either "accounts" or "public.accounts" (both canonicalize equally),
	// preserving the legacy normalizeTableNameForLookup behavior.
	if !snapshot.HasTable("accounts") {
		t.Fatalf("snapshot missing canonical table accounts; got %#v", snapshot.Tables)
	}
	if snapshot.HasTable("public.accounts") {
		t.Fatalf("snapshot should carry the canonical name, not public.accounts; got %#v", snapshot.Tables)
	}

	var fields []string
	for _, table := range snapshot.Tables {
		if table.Name == "accounts" {
			fields = table.Fields
		}
	}
	if len(fields) != 2 {
		t.Errorf("snapshot field count = %d, want 2 (got %#v)", len(fields), fields)
	}
}

// TestConnector_DiscoverSchemaHonorsSeededScope is the regression guard for the
// non-default-schema extraction defect: when the host seeds a per-request schema
// scope (schemacompat.WithSchemaScope) carrying the schemas the requested tables
// reference, DiscoverSchema MUST pass that scope to the underlying datasource so
// PostgreSQL/SQLServer discovery does not narrow to the default schema and drop
// qualified tables like "accounting.invoices".
func TestConnector_DiscoverSchemaHonorsSeededScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		descriptor engine.ConnectionDescriptor
		seedScope  []string
		wantScope  []string
	}{
		{
			name:       "seeded scope takes precedence over descriptor schema",
			descriptor: sampleDescriptor(), // Schema: "public"
			seedScope:  []string{"accounting", "reporting", "public"},
			wantScope:  []string{"accounting", "public", "reporting"},
		},
		{
			name:       "falls back to descriptor schema when no scope seeded",
			descriptor: sampleDescriptor(), // Schema: "public"
			seedScope:  nil,
			wantScope:  []string{"public"},
		},
		{
			name: "nil when neither scope nor descriptor schema is set",
			descriptor: engine.ConnectionDescriptor{
				ID:         "11111111-1111-1111-1111-111111111111",
				ConfigName: "pg-main",
				Type:       "POSTGRESQL",
				Host:       "db.internal",
				Port:       5432,
			},
			seedScope: nil,
			wantScope: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			captured := &capturedFactoryCall{}
			ds := &fakeDataSource{
				config:       datasource.DataSourceConfig{Type: "POSTGRESQL"},
				schemaResult: model.NewDataSourceSchema("pg-main"),
			}
			factory := newFakeFactory(ds, nil, captured)

			adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

			conn, err := adapter.Build(context.Background(), tc.descriptor)
			if err != nil {
				t.Fatalf("Build: unexpected error: %v", err)
			}

			ctx := context.Background()
			if tc.seedScope != nil {
				ctx = schemacompat.WithSchemaScope(ctx, tc.descriptor.ConfigName, tc.seedScope)
			}

			if _, err := conn.DiscoverSchema(ctx); err != nil {
				t.Fatalf("DiscoverSchema: unexpected error: %v", err)
			}

			got := append([]string(nil), ds.schemaScopeIn...)
			sort.Strings(got)
			want := append([]string(nil), tc.wantScope...)
			sort.Strings(want)

			if !reflect.DeepEqual(got, want) {
				t.Errorf("GetSchemaInfo scope = %#v, want %#v", got, want)
			}
		})
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

	queryStreamRows(t, conn, req)

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

// TestConnector_QueryReconstructsNestedAnyFilters covers the FIX-1 production shape:
// after the planner round-trips host filters, the connector receives the per-config
// filters as fully-nested map[string]any (table -> field -> FilterCondition-as-any),
// NOT the typed map. filtersForConfig must reconstruct the typed
// map[table]map[field]FilterCondition the DataSource expects. It also covers the
// skip branches: a non-map table value and a non-FilterCondition leaf are dropped
// without dropping the whole datasource's valid filters.
func TestConnector_QueryReconstructsNestedAnyFilters(t *testing.T) {
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

	// The nested any shape the planner produces (datasourceFilters -> map[string]any
	// all the way down, FilterCondition at the leaf).
	req := engine.ExtractionRequest{
		Filters: map[string]any{
			"pg-main": map[string]any{
				"accounts": map[string]any{
					"status": job.FilterCondition{Equals: []any{"active"}},
					"bogus":  "not-a-filter-condition", // non-FilterCondition leaf -> skipped
				},
				"badtable": "not-a-map", // non-map table value -> skipped
			},
		},
	}

	queryStreamRows(t, conn, req)

	if ds.queryFilters == nil {
		t.Fatalf("nested any filters were not reconstructed for the DataSource")
	}
	cond, ok := ds.queryFilters["accounts"]["status"]
	if !ok {
		t.Fatalf("status filter not reconstructed; got %#v", ds.queryFilters)
	}
	if len(cond.Equals) != 1 || cond.Equals[0] != "active" {
		t.Fatalf("filter value not preserved; got %#v", cond)
	}
	if _, leaked := ds.queryFilters["accounts"]["bogus"]; leaked {
		t.Fatalf("non-FilterCondition leaf must be skipped; got %#v", ds.queryFilters["accounts"])
	}
	if _, leaked := ds.queryFilters["badtable"]; leaked {
		t.Fatalf("non-map table value must be skipped; got %#v", ds.queryFilters)
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

	queryStreamRows(t, conn, req)
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

	_, err = conn.QueryStream(context.Background(), engine.ExtractionRequest{})
	assertEngineError(t, err, engine.CategoryUnavailable)
	if strings.Contains(err.Error(), "reader") {
		t.Fatalf("query error leaked driver internals: %q", err.Error())
	}
}

// TestConnector_DiscoverSchemaLazilyConnects proves DiscoverSchema connects on
// first use WITHOUT a prior TestConnection. The Engine's plan/validation lifecycle
// is build -> discover -> close (no explicit TestConnection), so the extraction
// connector must lazily connect there, matching the schemacompat connector. A
// factory error on that lazy connect surfaces as a safe unavailable error.
func TestConnector_DiscoverSchemaLazilyConnects(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	schema := model.NewDataSourceSchema("pg-main")
	ds := &fakeDataSource{config: datasource.DataSourceConfig{Type: "POSTGRESQL"}, schemaResult: schema}
	factory := newFakeFactory(ds, nil, captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	// No TestConnection called first — DiscoverSchema must connect itself.
	snapshot, err := conn.DiscoverSchema(context.Background())
	if err != nil {
		t.Fatalf("DiscoverSchema should lazily connect, got error: %v", err)
	}
	if snapshot.ConfigName != "pg-main" {
		t.Fatalf("snapshot ConfigName = %q, want pg-main", snapshot.ConfigName)
	}
	if captured.conn == nil {
		t.Fatal("expected the factory to be invoked by the lazy connect")
	}
}

// TestConnector_DiscoverSchemaLazyConnectFactoryErrorIsSafe proves a factory failure
// on the lazy connect surfaces as a safe unavailable error carrying no driver
// internals.
func TestConnector_DiscoverSchemaLazyConnectFactoryErrorIsSafe(t *testing.T) {
	t.Parallel()

	captured := &capturedFactoryCall{}
	factory := newFakeFactory(nil, errors.New("dial tcp db.internal:5432: connection refused for reader"), captured)

	adapter := enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil)

	conn, err := adapter.Build(context.Background(), sampleDescriptor())
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	_, err = conn.DiscoverSchema(context.Background())
	assertEngineError(t, err, engine.CategoryUnavailable)
	if strings.Contains(err.Error(), "reader") {
		t.Fatalf("schema connect error leaked driver internals: %q", err.Error())
	}
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

// newNormalizationEngine wires a REAL engine over the extraction ConnectorFactory
// and the schemacompat request-scoped ConnectionStore — the exact production
// topology the Worker runner builds — so a test can drive PlanExtraction through
// the genuine normalization seam (snapshot side in DiscoverSchema, request side in
// the worker mapper) rather than a fake.
func newNormalizationEngine(t *testing.T, ds datasource.DataSource) *engine.Engine {
	t.Helper()

	captured := &capturedFactoryCall{}
	factory := newFakeFactory(ds, nil, captured)

	registry := singleConnectorRegistry{
		factory: enginecompatdatasource.NewConnectorFactory(factory, constResolver("p", nil), nil),
	}

	eng, err := engine.New(
		engine.WithConnectorRegistry(registry),
		engine.WithConnectionStore(schemacompat.NewConnectionStore()),
	)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	return eng
}

// singleConnectorRegistry resolves one ConnectorFactory for any type, mirroring the
// worker/manager registries (type validation happens in the factory's Build).
type singleConnectorRegistry struct {
	factory engine.ConnectorFactory
}

func (r singleConnectorRegistry) Connector(string) (engine.ConnectorFactory, bool) {
	return r.factory, true
}

// pgConnection builds a rich PostgreSQL *model.Connection for config "pg-main", the
// record the Worker resolves and seeds into the context.
func pgConnection() *model.Connection {
	conn := &model.Connection{
		ID:           uuid.New(),
		ConfigName:   "pg-main",
		Type:         model.TypePostgreSQL,
		Host:         "db.internal",
		Port:         5432,
		DatabaseName: "ledger",
		Username:     "reader",
	}
	conn.SetPlaintextPassword("p")

	return conn
}

// publicUsersSchema is a PostgreSQL schema whose only table is the public-schema
// "users", which GetSchemaInfo returns unqualified (the legacy storage convention
// for public-schema tables). The host snapshot normalization keys it as "users".
func publicUsersSchema() *model.DataSourceSchema {
	schema := model.NewDataSourceSchema("pg-main")
	schema.AddTable("users", []string{"id", "name"})

	return schema
}

// TestNormalizationSeam_PublicPrefixedRequestStillExecutes is the MANDATORY
// behavior-preservation test for owner decision Option 2 (T-010). A job that
// addresses a PostgreSQL public-schema table with the "public." prefix
// ("public.users") must validate and execute against a snapshot the host normalized
// to "users" — exactly as the legacy Worker's normalizeTableNameForLookup accepted
// it. It proves the strict execution-layer validation does NOT regress the legacy
// prefix tolerance: the request side (worker mapper, modeled here by normalizing the
// requested table key with tablenorm) and the snapshot side (adapter DiscoverSchema)
// canonicalize to the same identity, so the Engine's literal match succeeds.
func TestNormalizationSeam_PublicPrefixedRequestStillExecutes(t *testing.T) {
	t.Parallel()

	ds := &fakeDataSource{
		config:       datasource.DataSourceConfig{Type: "POSTGRESQL"},
		schemaResult: publicUsersSchema(),
		queryResult:  map[string][]map[string]any{"users": {{"id": 1, "name": "Ada"}}},
	}

	eng := newNormalizationEngine(t, ds)

	ctx := schemacompat.WithResolvedConnections(context.Background(), []*model.Connection{pgConnection()})

	tenant, err := engine.NewTenantContext("tenant-norm")
	if err != nil {
		t.Fatalf("NewTenantContext: %v", err)
	}

	// The request carries the ACTUAL "public." prefix a caller would send. It is run
	// through tablenorm.NormalizeTable exactly as the Worker mapper does, producing
	// the canonical "users" that must match the snapshot the adapter normalized from
	// "users". This is the genuine cross-side reconciliation: if NormalizeTable were
	// reduced to identity, the request key would stay "public.users", the snapshot key
	// is "users", the literal match would FAIL, and this test would fail — so the
	// no-op mutation is caught (FIX-4: non-vacuous seam proof).
	const requestedTable = "public.users" // PG public-schema table addressed WITH the prefix
	canonicalTable := tablenorm.NormalizeTable(model.TypePostgreSQL, requestedTable)
	if canonicalTable == requestedTable {
		t.Fatalf("normalization must strip the public. prefix; got %q (identity no-op?)", canonicalTable)
	}

	request := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"pg-main": {canonicalTable: {"id", "name"}},
		},
	}

	plan, err := eng.PlanExtraction(ctx, tenant, request)
	if err != nil {
		t.Fatalf("PlanExtraction rejected a public-schema table the legacy worker accepted: %v", err)
	}

	result, err := eng.ExecuteExtraction(ctx, plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: %v", err)
	}
	if result.Direct == nil {
		t.Fatalf("expected a direct-mode result, got %#v", result)
	}
	if !strings.Contains(string(result.Direct.Data), "Ada") {
		t.Fatalf("expected extracted rows in direct payload, got %q", string(result.Direct.Data))
	}
}

// TestNormalizationSeam_GenuinelyMissingTableStillFails proves the normalization
// seam does NOT weaken strict validation: a table that does not exist in the
// snapshot under ANY canonical form is still rejected by PlanExtraction. This is
// the T-009-deferred strict-validation target — preserved alongside the prefix
// tolerance, not traded away for it.
func TestNormalizationSeam_GenuinelyMissingTableStillFails(t *testing.T) {
	t.Parallel()

	ds := &fakeDataSource{
		config:       datasource.DataSourceConfig{Type: "POSTGRESQL"},
		schemaResult: publicUsersSchema(),
	}

	eng := newNormalizationEngine(t, ds)

	ctx := schemacompat.WithResolvedConnections(context.Background(), []*model.Connection{pgConnection()})

	tenant, err := engine.NewTenantContext("tenant-norm")
	if err != nil {
		t.Fatalf("NewTenantContext: %v", err)
	}

	request := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"pg-main": {"ghost_table": {"id"}},
		},
	}

	_, err = eng.PlanExtraction(ctx, tenant, request)
	if err == nil {
		t.Fatal("expected PlanExtraction to reject a genuinely missing table, got nil")
	}

	assertEngineError(t, err, engine.CategoryValidation)
}

// oracleConnection builds a rich Oracle *model.Connection for config "ora-main".
func oracleConnection() *model.Connection {
	conn := &model.Connection{
		ID:           uuid.New(),
		ConfigName:   "ora-main",
		Type:         model.TypeOracle,
		Host:         "ora.internal",
		Port:         1521,
		DatabaseName: "ORCL",
		Username:     "reader",
	}
	conn.SetPlaintextPassword("p")

	return conn
}

// TestNormalizationSeam_OracleCaseInsensitiveMatching is the case-insensitivity
// regression guard under the UPPERCASE-CANONICAL contract. The Oracle adapter's
// GetSchemaInfo lowercases what it returns, but the seam re-folds to UPPERCASE so the
// snapshot matches the physical Oracle catalog AND the extracted result keys (which
// pkg/oracle.createRowMap keys verbatim by the physical UPPERCASE columns). The engine
// matches literally, so the host folds Oracle request identifiers to UPPERCASE
// (tablenorm.NormalizeTable / NormalizeField) on both the request and snapshot sides.
// A mixed/lower-case Oracle request must therefore execute against the UPPERCASE
// snapshot. The physical-case request resolution happens later in pkg/oracle
// (ValidateTableAndFields), not at this seam.
func TestNormalizationSeam_OracleCaseInsensitiveMatching(t *testing.T) {
	t.Parallel()

	// The DISCOVERED Oracle schema reports identifiers LOWERCASED, as GetSchemaInfo does;
	// the seam (snapshotFromSchema, normalize=true) re-folds them to UPPERCASE.
	schema := model.NewDataSourceSchema("ora-main")
	schema.AddTable("accounts", []string{"id", "balance"})

	// The extracted DATA is keyed by the physical UPPERCASE columns (createRowMap), so
	// the snapshot identity (UPPERCASE) equals the data-key identity (UPPERCASE).
	ds := &fakeDataSource{
		config:       datasource.DataSourceConfig{Type: string(model.TypeOracle)},
		schemaResult: schema,
		queryResult:  map[string][]map[string]any{"ACCOUNTS": {{"ID": 1, "BALANCE": 100}}},
	}

	eng := newNormalizationEngine(t, ds)
	ctx := schemacompat.WithResolvedConnections(context.Background(), []*model.Connection{oracleConnection()})

	tenant, err := engine.NewTenantContext("tenant-ora")
	if err != nil {
		t.Fatalf("NewTenantContext: %v", err)
	}

	// The caller addresses the table and fields in mixed/lower case. The worker mapper
	// folds them to UPPERCASE via tablenorm; we mirror that canonical form, which must
	// match the UPPERCASE snapshot. An identity-no-op for Oracle would leave "accounts" /
	// "id" lowercase and FAIL the literal match — so this proves the fold is real.
	canonTable := tablenorm.NormalizeTable(model.TypeOracle, "accounts")
	if canonTable != "ACCOUNTS" {
		t.Fatalf("Oracle table must fold to UPPERCASE; got %q", canonTable)
	}
	canonField := tablenorm.NormalizeField(model.TypeOracle, "id")
	if canonField != "ID" {
		t.Fatalf("Oracle field must fold to UPPERCASE; got %q", canonField)
	}

	request := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"ora-main": {canonTable: {canonField, tablenorm.NormalizeField(model.TypeOracle, "balance")}},
		},
	}

	plan, err := eng.PlanExtraction(ctx, tenant, request)
	if err != nil {
		t.Fatalf("PlanExtraction rejected a case-folded Oracle request the legacy worker accepted: %v", err)
	}

	result, err := eng.ExecuteExtraction(ctx, plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: %v", err)
	}
	// The extracted data carries the physical UPPERCASE keys (snapshot == data identity).
	if result.Direct == nil || !strings.Contains(string(result.Direct.Data), "BALANCE") {
		t.Fatalf("expected extracted Oracle rows keyed UPPERCASE, got %#v", result)
	}
}

// TestNormalizationSeam_OracleGenuinelyMissingTableStillFails proves Oracle case
// folding does NOT weaken strict validation: a table absent from the snapshot under
// its UPPERCASE canonical form is still rejected.
func TestNormalizationSeam_OracleGenuinelyMissingTableStillFails(t *testing.T) {
	t.Parallel()

	schema := model.NewDataSourceSchema("ora-main")
	schema.AddTable("accounts", []string{"id"})

	ds := &fakeDataSource{config: datasource.DataSourceConfig{Type: string(model.TypeOracle)}, schemaResult: schema}
	eng := newNormalizationEngine(t, ds)
	ctx := schemacompat.WithResolvedConnections(context.Background(), []*model.Connection{oracleConnection()})

	tenant, err := engine.NewTenantContext("tenant-ora")
	if err != nil {
		t.Fatalf("NewTenantContext: %v", err)
	}

	request := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"ora-main": {tablenorm.NormalizeTable(model.TypeOracle, "ghosts"): {"ID"}},
		},
	}

	_, err = eng.PlanExtraction(ctx, tenant, request)
	if err == nil {
		t.Fatal("expected PlanExtraction to reject a genuinely missing Oracle table, got nil")
	}

	assertEngineError(t, err, engine.CategoryValidation)
}
