package sqlserver

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/sqlserver"
	"github.com/LerianStudio/fetcher/pkg/testutil"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newMockLogger() log.Logger {
	return &testutil.MockLogger{}
}

func TestDataSourceConfigSQLServer_GetConfig(t *testing.T) {
	cfg := datasource.DataSourceConfig{
		ConfigName: "test-config",
		Type:       "sqlserver",
	}

	ds := &DataSourceConfigSQLServer{
		DataSourceConfig: cfg,
	}

	got := ds.GetConfig()
	assert.Equal(t, cfg, got)
}

func TestDataSourceConfigSQLServer_GetType(t *testing.T) {
	ds := &DataSourceConfigSQLServer{
		DataSourceConfig: datasource.DataSourceConfig{
			Type: "sqlserver",
		},
	}

	assert.Equal(t, "sqlserver", ds.GetType())
}

func TestDataSourceConfigSQLServer_Connect(t *testing.T) {
	ds := &DataSourceConfigSQLServer{
		DataSourceConfig: datasource.DataSourceConfig{
			ConfigName: "test-config",
		},
	}

	ctx := context.Background()
	logger := newMockLogger()

	err := ds.Connect(ctx, logger)
	assert.NoError(t, err)
	assert.Equal(t, libConstant.DataSourceStatusAvailable, ds.Status)
}

func TestDataSourceConfigSQLServer_Close(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := sqlserver.NewMockRepository(ctrl)

		mockRepo.EXPECT().CloseConnection().Return(nil)

		ds := &DataSourceConfigSQLServer{
			SQLServerRepository: mockRepo,
		}

		err := ds.Close(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, libConstant.DataSourceStatusUnavailable, ds.Status)
	})

	t.Run("close with nil repository", func(t *testing.T) {
		ds := &DataSourceConfigSQLServer{
			SQLServerRepository: nil,
		}

		err := ds.Close(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, libConstant.DataSourceStatusUnavailable, ds.Status)
	})

	t.Run("close with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := sqlserver.NewMockRepository(ctrl)

		mockRepo.EXPECT().CloseConnection().Return(errors.New("close error"))

		ds := &DataSourceConfigSQLServer{
			SQLServerRepository: mockRepo,
		}

		err := ds.Close(context.Background())
		assert.Error(t, err)
	})
}

func TestDataSourceConfigSQLServer_Query(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := sqlserver.NewMockRepository(ctrl)
	logger := newMockLogger()
	ctx := context.Background()

	schemaResult := []sqlserver.TableSchema{
		{TableName: "dbo.users", Columns: []sqlserver.ColumnInformation{{Name: "id"}, {Name: "name"}}},
	}

	t.Run("successful query without filters", func(t *testing.T) {
		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(schemaResult, nil)
		mockRepo.EXPECT().Query(gomock.Any(), schemaResult, "dbo.users", []string{"id", "name"}, nil).
			Return([]map[string]any{{"id": 1, "name": "John"}}, nil)

		ds := &DataSourceConfigSQLServer{
			SQLServerRepository: mockRepo,
		}

		tables := map[string][]string{
			"dbo.users": {"id", "name"},
		}

		result, err := ds.Query(ctx, tables, nil, logger)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result["dbo.users"]))
	})

	t.Run("query with filters", func(t *testing.T) {
		filters := map[string]map[string]job.FilterCondition{
			"dbo.users": {"status": {Equals: []any{"active"}}},
		}

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(schemaResult, nil)
		mockRepo.EXPECT().QueryWithAdvancedFilters(gomock.Any(), schemaResult, "dbo.users", []string{"id"}, gomock.Any()).
			Return([]map[string]any{{"id": 1}}, nil)

		ds := &DataSourceConfigSQLServer{
			SQLServerRepository: mockRepo,
		}

		tables := map[string][]string{
			"dbo.users": {"id"},
		}

		result, err := ds.Query(ctx, tables, filters, logger)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("query with schema error", func(t *testing.T) {
		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(nil, errors.New("schema error"))

		ds := &DataSourceConfigSQLServer{
			SQLServerRepository: mockRepo,
		}

		tables := map[string][]string{
			"dbo.users": {"id"},
		}

		_, err := ds.Query(ctx, tables, nil, logger)
		assert.Error(t, err)
	})
}

func TestGetTableFilters(t *testing.T) {
	tests := []struct {
		name        string
		filters     map[string]map[string]job.FilterCondition
		tableName   string
		wantNil     bool
		wantColumns int
	}{
		{
			name:      "nil filters",
			filters:   nil,
			tableName: "users",
			wantNil:   true,
		},
		{
			name:      "empty filters",
			filters:   map[string]map[string]job.FilterCondition{},
			tableName: "users",
			wantNil:   true,
		},
		{
			name: "table not in filters",
			filters: map[string]map[string]job.FilterCondition{
				"orders": {"id": {Equals: []any{1}}},
			},
			tableName: "users",
			wantNil:   true,
		},
		{
			name: "exact match with schema prefix",
			filters: map[string]map[string]job.FilterCondition{
				"dbo.users": {
					"id":     {Equals: []any{1}},
					"status": {Equals: []any{"active"}},
				},
			},
			tableName:   "dbo.users",
			wantNil:     false,
			wantColumns: 2,
		},
		{
			name: "match without schema prefix when filters have schema",
			filters: map[string]map[string]job.FilterCondition{
				"users": {
					"id": {Equals: []any{1}},
				},
			},
			tableName:   "dbo.users",
			wantNil:     false,
			wantColumns: 1,
		},
		{
			name: "match with default schema when tableName has no schema",
			filters: map[string]map[string]job.FilterCondition{
				"dbo.orders": {
					"total": {GreaterThan: []any{100}},
				},
			},
			tableName:   "orders",
			wantNil:     false,
			wantColumns: 1,
		},
		{
			name: "no match with different schema",
			filters: map[string]map[string]job.FilterCondition{
				"sales.orders": {
					"total": {GreaterThan: []any{100}},
				},
			},
			tableName: "dbo.orders",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTableFilters(tt.filters, tt.tableName)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			assert.NotNil(t, got)
			assert.Equal(t, tt.wantColumns, len(got))
		})
	}
}

func TestDataSourceConfigSQLServer_GetSchemaInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := sqlserver.NewMockRepository(ctrl)

	t.Run("successful schema retrieval", func(t *testing.T) {
		schemaResult := []sqlserver.TableSchema{
			{
				TableName: "dbo.users",
				Columns: []sqlserver.ColumnInformation{
					{Name: "id"},
					{Name: "name"},
				},
			},
		}

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(schemaResult, nil)

		ds := &DataSourceConfigSQLServer{
			DataSourceConfig: datasource.DataSourceConfig{
				ConfigName: "test-sqlserver",
			},
			SQLServerRepository: mockRepo,
		}

		ctx := context.Background()

		schema, err := ds.GetSchemaInfo(ctx, nil)
		assert.NoError(t, err)
		assert.Equal(t, "test-sqlserver", schema.ConfigName)
	})

	t.Run("schema retrieval error", func(t *testing.T) {
		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

		ds := &DataSourceConfigSQLServer{
			SQLServerRepository: mockRepo,
		}

		ctx := context.Background()
		_, err := ds.GetSchemaInfo(ctx, nil)
		assert.Error(t, err)
	})
}
