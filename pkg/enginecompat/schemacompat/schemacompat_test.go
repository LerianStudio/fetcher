// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package schemacompat_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	modelDatasource "github.com/LerianStudio/fetcher/v2/pkg/model/datasource"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func mustTenant(t *testing.T) engine.TenantContext {
	t.Helper()

	tenant, err := engine.NewTenantContext("tenant-a")
	require.NoError(t, err)

	return tenant
}

// --- SchemaCache adapter -----------------------------------------------------

type fakeCachePort struct {
	getSchema *model.DataSourceSchema
	getErr    error
	setErr    error

	setCalledKey string
	setCalledTTL time.Duration
}

func (f *fakeCachePort) Get(_ context.Context, _ string) (*model.DataSourceSchema, error) {
	return f.getSchema, f.getErr
}

func (f *fakeCachePort) Set(_ context.Context, configName string, _ *model.DataSourceSchema, ttl time.Duration) error {
	f.setCalledKey = configName
	f.setCalledTTL = ttl

	return f.setErr
}

func (f *fakeCachePort) Delete(context.Context, string) error { return nil }
func (f *fakeCachePort) Clear(context.Context) error          { return nil }
func (f *fakeCachePort) IsHealthy(context.Context) bool       { return true }
func (f *fakeCachePort) Close() error                         { return nil }

func TestSchemaCache_NilCacheYieldsNilAdapter(t *testing.T) {
	t.Parallel()

	assert.Nil(t, schemacompat.NewSchemaCache(nil, time.Minute))
}

func TestSchemaCache_GetSchema_Hit(t *testing.T) {
	t.Parallel()

	stored := model.NewDataSourceSchema("db1")
	stored.AddTable("users", []string{"id", "name"})

	adapter := schemacompat.NewSchemaCache(&fakeCachePort{getSchema: stored}, time.Minute)

	snapshot, ok, err := adapter.GetSchema(context.Background(), mustTenant(t), "db1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "db1", snapshot.ConfigName)
	require.Len(t, snapshot.Tables, 1)
	assert.Equal(t, "users", snapshot.Tables[0].Name)
	assert.ElementsMatch(t, []string{"id", "name"}, snapshot.Tables[0].Fields)
}

func TestSchemaCache_GetSchema_Miss(t *testing.T) {
	t.Parallel()

	adapter := schemacompat.NewSchemaCache(&fakeCachePort{getSchema: nil}, time.Minute)

	_, ok, err := adapter.GetSchema(context.Background(), mustTenant(t), "db1")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSchemaCache_GetSchema_ErrorSurfaces(t *testing.T) {
	t.Parallel()

	adapter := schemacompat.NewSchemaCache(&fakeCachePort{getErr: errors.New("redis down")}, time.Minute)

	_, ok, err := adapter.GetSchema(context.Background(), mustTenant(t), "db1")
	require.Error(t, err)
	assert.False(t, ok)
}

func TestSchemaCache_PutSchema_WritesThroughWithTTL(t *testing.T) {
	t.Parallel()

	port := &fakeCachePort{}
	adapter := schemacompat.NewSchemaCache(port, 7*time.Minute)

	snapshot := engine.SchemaSnapshot{
		ConfigName: "db1",
		Tables:     []engine.TableSnapshot{{Name: "users", Fields: []string{"id"}}},
	}

	require.NoError(t, adapter.PutSchema(context.Background(), mustTenant(t), snapshot))
	assert.Equal(t, "db1", port.setCalledKey)
	assert.Equal(t, 7*time.Minute, port.setCalledTTL)
}

// --- ConnectionStore (request-scoped) ---------------------------------------

func TestConnectionStore_FindConnection_FromSeed(t *testing.T) {
	t.Parallel()

	conn := &model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL, Host: "db.example.com"}
	ctx := schemacompat.WithResolvedConnections(context.Background(), []*model.Connection{conn})

	store := schemacompat.NewConnectionStore()
	desc, found, err := store.FindConnection(ctx, mustTenant(t), "db1")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "db1", desc.ConfigName)

	// The rich record round-trips through the opaque host payload.
	unpacked := connectioncompat.ConnectionFromDescriptor(desc)
	require.NotNil(t, unpacked)
	assert.Equal(t, "db.example.com", unpacked.Host)
}

func TestConnectionStore_FindConnection_NotSeeded(t *testing.T) {
	t.Parallel()

	store := schemacompat.NewConnectionStore()
	_, found, err := store.FindConnection(context.Background(), mustTenant(t), "missing")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestConnectionStore_LifecycleMethodsAreUnsupported(t *testing.T) {
	t.Parallel()

	store := schemacompat.NewConnectionStore()
	ctx := context.Background()
	tenant := mustTenant(t)
	desc := engine.ConnectionDescriptor{ConfigName: "db1"}

	assert.Error(t, store.Create(ctx, tenant, desc, nil))
	assert.Error(t, store.Update(ctx, tenant, desc, nil))
	assert.Error(t, store.Delete(ctx, tenant, "db1"))

	_, listErr := store.List(ctx, tenant)
	assert.Error(t, listErr)

	_, _, byIDErr := store.FindByID(ctx, tenant, "id")
	assert.Error(t, byIDErr)

	assert.Error(t, store.UpdateByID(ctx, tenant, "id", desc, nil))
	assert.Error(t, store.DeleteByID(ctx, tenant, "id"))

	_, pagedErr := store.ListPaged(ctx, tenant, engine.ConnectionListParams{})
	assert.Error(t, pagedErr)
}

func TestWithSchemaScope_SeedsAndOverrides(t *testing.T) {
	t.Parallel()

	conn := &model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL}
	ctx := schemacompat.WithResolvedConnections(context.Background(), []*model.Connection{conn})
	ctx = schemacompat.WithSchemaScope(ctx, "db1", []string{"public", "finance"})
	// A second scope for another config must not clobber the first.
	ctx = schemacompat.WithSchemaScope(ctx, "db2", []string{"dbo"})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ds := modelDatasource.NewMockDataSource(ctrl)
	ds.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Eq([]string{"public", "finance"})).
		Return(model.NewDataSourceSchema("db1"), nil)
	ds.EXPECT().Close(gomock.Any()).Return(nil)

	cf := schemacompat.NewConnectorFactory(func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		return ds, nil
	}, nil)

	connector, err := cf.Build(ctx, descriptorFor(conn))
	require.NoError(t, err)

	_, err = connector.DiscoverSchema(ctx)
	require.NoError(t, err)
	require.NoError(t, connector.Close(ctx))
}

func TestConnector_TestConnectionAndQuery(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ds := modelDatasource.NewMockDataSource(ctrl)

	cf := schemacompat.NewConnectorFactory(func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		return ds, nil
	}, nil)

	connector, err := cf.Build(context.Background(), descriptorFor(&model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL}))
	require.NoError(t, err)

	// TestConnection connects (lazy) without error.
	require.NoError(t, connector.TestConnection(context.Background()))

	// QueryStream is unsupported on the schema path.
	_, queryErr := connector.QueryStream(context.Background(), engine.ExtractionRequest{})
	require.Error(t, queryErr)
}

func TestConnector_DiscoverSchema_NilFactoryIsUnavailable(t *testing.T) {
	t.Parallel()

	cf := schemacompat.NewConnectorFactory(nil, nil)
	connector, err := cf.Build(context.Background(), descriptorFor(&model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL}))
	require.NoError(t, err)

	_, err = connector.DiscoverSchema(context.Background())
	require.Error(t, err)
}

func TestConnector_DiscoverSchema_NilDatasourceIsUnavailable(t *testing.T) {
	t.Parallel()

	cf := schemacompat.NewConnectorFactory(func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		return nil, nil
	}, nil)
	connector, err := cf.Build(context.Background(), descriptorFor(&model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL}))
	require.NoError(t, err)

	_, err = connector.DiscoverSchema(context.Background())
	require.Error(t, err)
}

func TestConnector_Close_BeforeConnectIsNoop(t *testing.T) {
	t.Parallel()

	cf := schemacompat.NewConnectorFactory(func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		return nil, nil
	}, nil)
	connector, err := cf.Build(context.Background(), descriptorFor(&model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL}))
	require.NoError(t, err)

	require.NoError(t, connector.Close(context.Background()))
}

func TestConnectorFactory_Build_MissingHostRecordFails(t *testing.T) {
	t.Parallel()

	cf := schemacompat.NewConnectorFactory(func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		return nil, nil
	}, nil)

	// A bare descriptor with a valid type but no packed host record must fail.
	_, err := cf.Build(context.Background(), engine.ConnectionDescriptor{ConfigName: "db1", Type: string(model.TypePostgreSQL)})
	require.Error(t, err)
}

// --- ConnectorFactory + Connector -------------------------------------------

func descriptorFor(conn *model.Connection) engine.ConnectionDescriptor {
	return connectioncompat.DescriptorFromConnection(conn)
}

func TestConnector_DiscoverSchema_FiltersSystemTablesAndLazilyConnects(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	schema := model.NewDataSourceSchema("db1")
	schema.AddTable("users", []string{"id", "name"})
	schema.AddTable("pg_catalog", []string{"oid"}) // system table — must be excluded

	ds := modelDatasource.NewMockDataSource(ctrl)
	ds.EXPECT().GetSchemaInfo(gomock.Any(), gomock.Any()).Return(schema, nil)
	ds.EXPECT().Close(gomock.Any()).Return(nil)

	factory := func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		return ds, nil
	}

	conn := &model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL}
	cf := schemacompat.NewConnectorFactory(factory, nil)

	connector, err := cf.Build(context.Background(), descriptorFor(conn))
	require.NoError(t, err)

	// Engine discovery lifecycle is build -> discover -> close, without an explicit
	// TestConnection: DiscoverSchema must lazily connect.
	snapshot, err := connector.DiscoverSchema(context.Background())
	require.NoError(t, err)

	names := make([]string, 0, len(snapshot.Tables))
	for _, table := range snapshot.Tables {
		names = append(names, table.Name)
	}

	assert.Contains(t, names, "users")
	assert.NotContains(t, names, "pg_catalog", "system tables must be excluded before crossing the Engine boundary")

	require.NoError(t, connector.Close(context.Background()))
}

func TestConnector_DiscoverSchema_FactoryErrorPropagatesVerbatim(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("host safety rejection")
	factory := func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		return nil, sentinel
	}

	conn := &model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL}
	cf := schemacompat.NewConnectorFactory(factory, nil)

	connector, err := cf.Build(context.Background(), descriptorFor(conn))
	require.NoError(t, err)

	_, err = connector.DiscoverSchema(context.Background())
	require.ErrorIs(t, err, sentinel, "the factory error cause must stay reachable so the host can recognize typed errors")
}

// TestConnector_DiscoverSchema_ConnectFailure_IsConnectCategory proves the
// stage signal: a CONNECT-stage failure (the factory failing to establish the
// datasource) surfaces as a CategoryConnect EngineError so resolveSchema passes
// it through and the Manager can render "Database Connection Error" — distinct
// from a discovery-read failure (CategoryUnavailable). The raw cause MUST NOT
// appear in the safe message, but MUST stay reachable via errors.Is.
func TestConnector_DiscoverSchema_ConnectFailure_IsConnectCategory(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("dial tcp 10.0.0.1:5432: password=s3cr3t connection refused")
	factory := func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		return nil, sentinel
	}

	conn := &model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL}
	cf := schemacompat.NewConnectorFactory(factory, nil)

	connector, err := cf.Build(context.Background(), descriptorFor(conn))
	require.NoError(t, err)

	_, err = connector.DiscoverSchema(context.Background())
	require.Error(t, err)

	var engErr *engine.EngineError
	require.ErrorAs(t, err, &engErr)
	require.Equal(t, engine.CategoryConnect, engErr.Category, "a connect-stage failure must carry CategoryConnect")

	// The cause stays reachable for typed-error recognition, but the safe
	// boundary message must NOT leak the raw connect error.
	require.ErrorIs(t, err, sentinel)

	for _, leak := range []string{"s3cr3t", "10.0.0.1", "password=", "dial tcp"} {
		require.NotContains(t, engErr.Error(), leak, "connect error must not leak %q", leak)
	}
}

func TestConnectorFactory_Build_UnknownTypeFails(t *testing.T) {
	t.Parallel()

	conn := &model.Connection{ConfigName: "db1", Type: model.DBType("NOPE")}
	cf := schemacompat.NewConnectorFactory(func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		return nil, nil
	}, nil)

	_, err := cf.Build(context.Background(), descriptorFor(conn))
	require.Error(t, err)
}

// --- IsSystemTable (moved from the Manager) ---------------------------------

func TestIsSystemTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		dbType    model.DBType
		tableName string
		expected  bool
	}{
		// PostgreSQL
		{"PostgreSQL pg_catalog", model.TypePostgreSQL, "pg_catalog", true},
		{"PostgreSQL information_schema", model.TypePostgreSQL, "information_schema", true},
		{"PostgreSQL pg_toast", model.TypePostgreSQL, "pg_toast", true},
		{"PostgreSQL pg_temp_1", model.TypePostgreSQL, "pg_temp_1", true},
		{"PostgreSQL user table", model.TypePostgreSQL, "users", false},

		// MySQL
		{"MySQL mysql", model.TypeMySQL, "mysql", true},
		{"MySQL information_schema", model.TypeMySQL, "information_schema", true},
		{"MySQL performance_schema", model.TypeMySQL, "performance_schema", true},
		{"MySQL sys", model.TypeMySQL, "sys", true},
		{"MySQL user table", model.TypeMySQL, "products", false},

		// Oracle - uppercase (standard)
		{"Oracle SYS", model.TypeOracle, "SYS", true},
		{"Oracle SYSTEM", model.TypeOracle, "SYSTEM", true},
		{"Oracle OUTLN", model.TypeOracle, "OUTLN", true},
		{"Oracle XDB", model.TypeOracle, "XDB", true},
		{"Oracle MDSYS", model.TypeOracle, "MDSYS", true},
		{"Oracle CTXSYS", model.TypeOracle, "CTXSYS", true},
		{"Oracle DBSNMP", model.TypeOracle, "DBSNMP", true},
		{"Oracle user table", model.TypeOracle, "EMPLOYEES", false},
		// Oracle - lowercase (driver-dependent)
		{"Oracle sys lowercase", model.TypeOracle, "sys", true},
		{"Oracle system lowercase", model.TypeOracle, "system", true},
		{"Oracle mixed case Sys", model.TypeOracle, "Sys", true},
		{"Oracle mixed case System", model.TypeOracle, "System", true},

		// SQL Server - exact matches
		{"SQLServer sys", model.TypeSQLServer, "sys", true},
		{"SQLServer INFORMATION_SCHEMA", model.TypeSQLServer, "INFORMATION_SCHEMA", true},
		{"SQLServer user table", model.TypeSQLServer, "Customers", false},
		// SQL Server - db_ prefix (per business requirement)
		{"SQLServer db_owner", model.TypeSQLServer, "db_owner", true},
		{"SQLServer db_backup", model.TypeSQLServer, "db_backup", true},
		{"SQLServer db_accessadmin", model.TypeSQLServer, "db_accessadmin", true},
		{"SQLServer db_backup.audit_logs", model.TypeSQLServer, "db_backup.audit_logs", true},
		{"SQLServer dbo.users", model.TypeSQLServer, "dbo.users", false},
		{"SQLServer sales.orders", model.TypeSQLServer, "sales.orders", false},

		// MongoDB - databases
		{"MongoDB admin", model.TypeMongoDB, "admin", true},
		{"MongoDB local", model.TypeMongoDB, "local", true},
		{"MongoDB config", model.TypeMongoDB, "config", true},
		{"MongoDB user collection", model.TypeMongoDB, "users", false},
		// MongoDB - system.* prefix (per business requirement)
		{"MongoDB system.indexes", model.TypeMongoDB, "system.indexes", true},
		{"MongoDB system.users", model.TypeMongoDB, "system.users", true},
		{"MongoDB system.profile", model.TypeMongoDB, "system.profile", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := schemacompat.IsSystemTable(tt.dbType, tt.tableName)
			assert.Equal(t, tt.expected, result, "IsSystemTable(%s, %s)", tt.dbType, tt.tableName)
		})
	}
}
