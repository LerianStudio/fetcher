package datasource

import (
	"context"
	"errors"
	"testing"
	"time"

	cryptoPkg "github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	modeldatasource "github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type stubDataSource struct {
	config modeldatasource.DataSourceConfig
}

func (s *stubDataSource) GetConfig() modeldatasource.DataSourceConfig {
	return s.config
}

func (s *stubDataSource) Connect(ctx context.Context, logger libLog.Logger) error {
	return nil
}

func (s *stubDataSource) Close(ctx context.Context) error {
	return nil
}

func (s *stubDataSource) GetType() string {
	return s.config.Type
}

func (s *stubDataSource) Query(ctx context.Context, tables map[string][]string, filters map[string]map[string]job.FilterCondition, logger libLog.Logger) (map[string][]map[string]any, error) {
	return nil, nil
}

func (s *stubDataSource) GetSchemaInfo(ctx context.Context, schemas []string) (*model.DataSourceSchema, error) {
	return nil, nil
}

func TestNewDataSourceFromConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		conn          *model.Connection
		cryptor       func(t *testing.T) cryptoPkg.Cryptor
		builders      map[model.DBType]dataSourceConfigBuilder
		assertOutcome func(t *testing.T, got modeldatasource.DataSource, err error)
	}{
		{
			name:    "rejects nil connection",
			conn:    nil,
			cryptor: func(t *testing.T) cryptoPkg.Cryptor { return newMockCryptor(t) },
			assertOutcome: func(t *testing.T, got modeldatasource.DataSource, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "connection cannot be nil")
				assert.Nil(t, got)
			},
		},
		{
			name:    "rejects nil cryptor",
			conn:    testConnection(model.TypePostgreSQL),
			cryptor: func(t *testing.T) cryptoPkg.Cryptor { return nil },
			assertOutcome: func(t *testing.T, got modeldatasource.DataSource, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "cryptor cannot be nil")
				assert.Nil(t, got)
			},
		},
		{
			name:    "rejects unsupported database type",
			conn:    testConnection(model.DBType("UNSUPPORTED")),
			cryptor: func(t *testing.T) cryptoPkg.Cryptor { return newMockCryptor(t) },
			assertOutcome: func(t *testing.T, got modeldatasource.DataSource, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "unsupported database type: UNSUPPORTED")
				assert.Nil(t, got)
			},
		},
		{
			name:    "routes supported type through deterministic builder seam",
			conn:    testConnection(model.TypePostgreSQL),
			cryptor: func(t *testing.T) cryptoPkg.Cryptor { return newMockCryptor(t) },
			builders: func() map[model.DBType]dataSourceConfigBuilder {
				return map[model.DBType]dataSourceConfigBuilder{
					model.TypePostgreSQL: func(ctx context.Context, base modeldatasource.DataSourceConfig, conn *model.Connection, cryptor cryptoPkg.Cryptor, logger libLog.Logger) (modeldatasource.DataSource, error) {
						return &stubDataSource{config: base}, nil
					},
				}
			}(),
			assertOutcome: func(t *testing.T, got modeldatasource.DataSource, err error) {
				require.NoError(t, err)
				require.NotNil(t, got)
				cfg := got.GetConfig()
				assert.Equal(t, "POSTGRESQL", cfg.Type)
				assert.Equal(t, "analytics", cfg.ConfigName)
				assert.Equal(t, "db.internal", cfg.Host)
				assert.Equal(t, "5432", cfg.Port)
				assert.Equal(t, "warehouse", cfg.DatabaseName)
				assert.Equal(t, "fetcher", cfg.Username)
				assert.Equal(t, "enc-password", cfg.PasswordEncrypted)
				assert.Empty(t, cfg.SSL.Mode)
			},
		},
		{
			name:    "propagates builder errors without downstream connection attempts",
			conn:    testConnection(model.TypeMongoDB),
			cryptor: func(t *testing.T) cryptoPkg.Cryptor { return newMockCryptor(t) },
			builders: map[model.DBType]dataSourceConfigBuilder{
				model.TypeMongoDB: func(ctx context.Context, base modeldatasource.DataSourceConfig, conn *model.Connection, cryptor cryptoPkg.Cryptor, logger libLog.Logger) (modeldatasource.DataSource, error) {
					return nil, errors.New("deterministic constructor failure")
				},
			},
			assertOutcome: func(t *testing.T, got modeldatasource.DataSource, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "deterministic constructor failure")
				assert.Nil(t, got)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builders := tt.builders
			if builders == nil {
				builders = map[model.DBType]dataSourceConfigBuilder{}
			}

			got, err := newDataSourceFromConnection(context.Background(), tt.conn, tt.cryptor(t), nil, builders)
			tt.assertOutcome(t, got, err)
		})
	}
}

func testConnection(dbType model.DBType) *model.Connection {
	return &model.Connection{
		ID:                uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		OrganizationID:    uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		ConfigName:        "analytics",
		Type:              dbType,
		Host:              "db.internal",
		Port:              5432,
		DatabaseName:      "warehouse",
		Username:          "fetcher",
		PasswordEncrypted: "enc-password",
		CreatedAt:         time.Unix(1700000000, 0).UTC(),
		UpdatedAt:         time.Unix(1700000000, 0).UTC(),
	}
}

func newMockCryptor(t *testing.T) cryptoPkg.Cryptor {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	return cryptoPkg.NewMockCryptor(ctrl)
}
