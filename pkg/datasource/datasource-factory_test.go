package datasource

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	datasourceConfig "github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDataSourceFromConnection_NilConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	ctx := context.Background()

	ds, err := NewDataSourceFromConnection(ctx, nil, mockCryptor, nil)

	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection cannot be nil")
}

func TestNewDataSourceFromConnection_NilCryptor(t *testing.T) {
	ctx := context.Background()
	conn := &model.Connection{
		ID:   uuid.New(),
		Type: model.TypePostgreSQL,
	}

	ds, err := NewDataSourceFromConnection(ctx, conn, nil, nil)

	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cryptor cannot be nil")
}

func TestNewDataSourceFromConnection_UnsupportedType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	ctx := context.Background()

	conn := &model.Connection{
		ID:   uuid.New(),
		Type: model.DBType("UNSUPPORTED"),
	}

	ds, err := NewDataSourceFromConnection(ctx, conn, mockCryptor, nil)

	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database type")
}

func TestNewDataSourceFromConnection_DecryptionError(t *testing.T) {
	tests := []struct {
		name    string
		dbType  model.DBType
		wantErr string
	}{
		{
			name:    "MongoDB decryption error",
			dbType:  model.TypeMongoDB,
			wantErr: "failed to decrypt password for MongoDB connection",
		},
		{
			name:    "PostgreSQL decryption error",
			dbType:  model.TypePostgreSQL,
			wantErr: "failed to decrypt password for PostgreSQL connection",
		},
		{
			name:    "Oracle decryption error",
			dbType:  model.TypeOracle,
			wantErr: "failed to decrypt password for Oracle connection",
		},
		{
			name:    "MySQL decryption error",
			dbType:  model.TypeMySQL,
			wantErr: "failed to decrypt password for MySQL connection",
		},
		{
			name:    "SQL Server decryption error",
			dbType:  model.TypeSQLServer,
			wantErr: "failed to decrypt password for SQL Server connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockCryptor := crypto.NewMockCryptor(ctrl)
			ctx := context.Background()

			conn := &model.Connection{
				ID:                   uuid.New(),
				OrganizationID:       uuid.New(),
				ConfigName:           "test-connection",
				Type:                 tt.dbType,
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
			}

			mockCryptor.EXPECT().
				Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion).
				Return("", errors.New("decryption failed"))

			ds, err := NewDataSourceFromConnection(ctx, conn, mockCryptor, nil)

			assert.Nil(t, ds)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestNewDataSourceConfigFromConnection(t *testing.T) {
	tests := []struct {
		name     string
		conn     *model.Connection
		validate func(t *testing.T, config datasourceConfig.DataSourceConfig)
	}{
		{
			name: "with all fields",
			conn: &model.Connection{
				ID:                uuid.MustParse("11111111-1111-1111-1111-111111111111"),
				OrganizationID:    uuid.MustParse("22222222-2222-2222-2222-222222222222"),
				ConfigName:        "production-db",
				Type:              model.TypePostgreSQL,
				Host:              "db.example.com",
				Port:              5432,
				DatabaseName:      "mydb",
				Username:          "admin",
				PasswordEncrypted: "encrypted",
				SSL: &model.SSLConfig{
					Mode: "require",
					CA:   "cert-data",
				},
			},
			validate: func(t *testing.T, config datasourceConfig.DataSourceConfig) {
				assert.Equal(t, "11111111-1111-1111-1111-111111111111", config.ID)
				assert.Equal(t, "22222222-2222-2222-2222-222222222222", config.OrganizationID)
				assert.Equal(t, "production-db", config.ConfigName)
				assert.Equal(t, "POSTGRESQL", config.Type)
				assert.Equal(t, "db.example.com", config.Host)
				assert.Equal(t, "5432", config.Port)
				assert.Equal(t, "mydb", config.DatabaseName)
				assert.Equal(t, "admin", config.Username)
				assert.Equal(t, "encrypted", config.PasswordEncrypted)
				assert.Equal(t, "require", config.SSL.Mode)
				assert.Equal(t, "cert-data", config.SSL.CA)
			},
		},
		{
			name: "without SSL config",
			conn: &model.Connection{
				ID:                uuid.MustParse("33333333-3333-3333-3333-333333333333"),
				OrganizationID:    uuid.MustParse("44444444-4444-4444-4444-444444444444"),
				ConfigName:        "dev-db",
				Type:              model.TypeMySQL,
				Host:              "localhost",
				Port:              3306,
				DatabaseName:      "devdb",
				Username:          "dev",
				PasswordEncrypted: "pwd",
				SSL:               nil,
			},
			validate: func(t *testing.T, config datasourceConfig.DataSourceConfig) {
				assert.Equal(t, "33333333-3333-3333-3333-333333333333", config.ID)
				assert.Equal(t, "MYSQL", config.Type)
				assert.Equal(t, "3306", config.Port)
				assert.Equal(t, "", config.SSL.Mode)
				assert.Equal(t, "", config.SSL.CA)
			},
		},
		{
			name: "MongoDB type",
			conn: &model.Connection{
				ID:                uuid.MustParse("55555555-5555-5555-5555-555555555555"),
				OrganizationID:    uuid.MustParse("66666666-6666-6666-6666-666666666666"),
				ConfigName:        "mongo-db",
				Type:              model.TypeMongoDB,
				Host:              "mongo.example.com",
				Port:              27017,
				DatabaseName:      "mongodb",
				Username:          "mongouser",
				PasswordEncrypted: "mongopwd",
			},
			validate: func(t *testing.T, config datasourceConfig.DataSourceConfig) {
				assert.Equal(t, "MONGODB", config.Type)
				assert.Equal(t, "27017", config.Port)
			},
		},
		{
			name: "Oracle type",
			conn: &model.Connection{
				ID:                uuid.MustParse("77777777-7777-7777-7777-777777777777"),
				OrganizationID:    uuid.MustParse("88888888-8888-8888-8888-888888888888"),
				ConfigName:        "oracle-db",
				Type:              model.TypeOracle,
				Host:              "oracle.example.com",
				Port:              1521,
				DatabaseName:      "orcl",
				Username:          "oracleuser",
				PasswordEncrypted: "oraclepwd",
			},
			validate: func(t *testing.T, config datasourceConfig.DataSourceConfig) {
				assert.Equal(t, "ORACLE", config.Type)
				assert.Equal(t, "1521", config.Port)
			},
		},
		{
			name: "SQL Server type",
			conn: &model.Connection{
				ID:                uuid.MustParse("99999999-9999-9999-9999-999999999999"),
				OrganizationID:    uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				ConfigName:        "sqlserver-db",
				Type:              model.TypeSQLServer,
				Host:              "sqlserver.example.com",
				Port:              1433,
				DatabaseName:      "sqldb",
				Username:          "sqluser",
				PasswordEncrypted: "sqlpwd",
			},
			validate: func(t *testing.T, config datasourceConfig.DataSourceConfig) {
				assert.Equal(t, "SQL_SERVER", config.Type)
				assert.Equal(t, "1433", config.Port)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := newDataSourceConfigFromConnection(tt.conn)
			tt.validate(t, config)
			assert.Equal(t, "", config.Status)
		})
	}
}

func TestNewDataSourceConfigOracle_MissingServiceName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	ctx := context.Background()

	tests := []struct {
		name     string
		metadata *map[string]any
	}{
		{
			name:     "nil metadata",
			metadata: nil,
		},
		{
			name:     "empty metadata",
			metadata: &map[string]any{},
		},
		{
			name: "metadata without serviceName",
			metadata: &map[string]any{
				"otherKey": "value",
			},
		},
		{
			name: "metadata with empty serviceName",
			metadata: &map[string]any{
				"serviceName": "",
			},
		},
		{
			name: "metadata with non-string serviceName",
			metadata: &map[string]any{
				"serviceName": 12345,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &model.Connection{
				ID:                   uuid.New(),
				OrganizationID:       uuid.New(),
				ConfigName:           "oracle-test",
				Type:                 model.TypeOracle,
				Host:                 "oracle.example.com",
				Port:                 1521,
				DatabaseName:         "orcl",
				Username:             "admin",
				PasswordEncrypted:    "encrypted",
				EncryptionKeyVersion: "v1",
				Metadata:             tt.metadata,
			}

			mockCryptor.EXPECT().
				Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion).
				Return("decrypted-password", nil)

			ds, err := NewDataSourceFromConnection(ctx, conn, mockCryptor, nil)

			assert.Nil(t, ds)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "serviceName is required")
		})
	}
}

func TestGetUniqueSchemas(t *testing.T) {
	tests := []struct {
		name   string
		tables map[string][]string
		want   []string
	}{
		{
			name:   "nil tables",
			tables: nil,
			want:   nil,
		},
		{
			name:   "empty tables",
			tables: map[string][]string{},
			want:   nil,
		},
		{
			name: "single schema",
			tables: map[string][]string{
				"public.users":    {"id", "name"},
				"public.orders":   {"id", "user_id"},
				"public.products": {"id", "name"},
			},
			want: []string{"public"},
		},
		{
			name: "multiple schemas",
			tables: map[string][]string{
				"public.users":  {"id"},
				"sales.orders":  {"id"},
				"hr.employees":  {"id"},
				"public.logs":   {"id"},
				"sales.reports": {"id"},
			},
			want: []string{"public", "sales", "hr"},
		},
		{
			name: "tables without schema (no dot)",
			tables: map[string][]string{
				"users":   {"id"},
				"orders":  {"id"},
				"configs": {"id"},
			},
			want: nil,
		},
		{
			name: "mixed with and without schema",
			tables: map[string][]string{
				"public.users": {"id"},
				"orders":       {"id"},
				"hr.employees": {"id"},
			},
			want: []string{"public", "hr"},
		},
		{
			name: "empty table names",
			tables: map[string][]string{
				"":     {"id"},
				"   ":  {"id"},
				".":    {"id"},
				".b":   {"id"},
				"  . ": {"id"},
			},
			want: nil,
		},
		{
			name: "schema with empty table name",
			tables: map[string][]string{
				"a.": {"id"},
			},
			want: []string{"a"},
		},
		{
			name: "whitespace around schema and table",
			tables: map[string][]string{
				"  public.users  ": {"id"},
				"sales.orders":     {"id"},
			},
			want: []string{"public", "sales"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := datasourceConfig.GetUniqueSchemas(tt.tables)

			if tt.want == nil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			// Since maps don't guarantee order, check length and contents
			assert.Equal(t, len(tt.want), len(got))
			for _, schema := range tt.want {
				assert.Contains(t, got, schema)
			}
		})
	}
}

func TestSplitSchemaTable(t *testing.T) {
	tests := []struct {
		name       string
		qualified  string
		wantSchema string
		wantTable  string
	}{
		{
			name:       "schema and table",
			qualified:  "public.users",
			wantSchema: "public",
			wantTable:  "users",
		},
		{
			name:       "table only (no dot)",
			qualified:  "users",
			wantSchema: "",
			wantTable:  "users",
		},
		{
			name:       "empty string",
			qualified:  "",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "whitespace only",
			qualified:  "   ",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "with leading and trailing whitespace",
			qualified:  "  public.users  ",
			wantSchema: "public",
			wantTable:  "users",
		},
		{
			name:       "dot at start",
			qualified:  ".users",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "dot at end",
			qualified:  "schema.",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "just a dot",
			qualified:  ".",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "multiple dots",
			qualified:  "catalog.schema.table",
			wantSchema: "catalog",
			wantTable:  "schema.table",
		},
		{
			name:       "whitespace around dot",
			qualified:  "  schema  .  table  ",
			wantSchema: "schema",
			wantTable:  "table",
		},
		{
			name:       "schema with numbers",
			qualified:  "schema123.table456",
			wantSchema: "schema123",
			wantTable:  "table456",
		},
		{
			name:       "schema with underscore",
			qualified:  "my_schema.my_table",
			wantSchema: "my_schema",
			wantTable:  "my_table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSchema, gotTable := datasourceConfig.SplitSchemaTable(tt.qualified)
			assert.Equal(t, tt.wantSchema, gotSchema)
			assert.Equal(t, tt.wantTable, gotTable)
		})
	}
}

func TestDataSourceConfig_GetConfig(t *testing.T) {
	config := datasourceConfig.DataSourceConfig{
		ID:                "test-id",
		OrganizationID:    "org-id",
		ConfigName:        "test-config",
		Type:              "POSTGRESQL",
		Host:              "localhost",
		Port:              "5432",
		DatabaseName:      "testdb",
		Username:          "user",
		PasswordEncrypted: "encrypted",
		SSL: model.SSLConfig{
			Mode: "require",
		},
		Status: "active",
	}

	assert.Equal(t, "test-id", config.ID)
	assert.Equal(t, "org-id", config.OrganizationID)
	assert.Equal(t, "test-config", config.ConfigName)
	assert.Equal(t, "POSTGRESQL", config.Type)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "5432", config.Port)
	assert.Equal(t, "testdb", config.DatabaseName)
	assert.Equal(t, "user", config.Username)
	assert.Equal(t, "encrypted", config.PasswordEncrypted)
	assert.Equal(t, "require", config.SSL.Mode)
	assert.Equal(t, "active", config.Status)
}
