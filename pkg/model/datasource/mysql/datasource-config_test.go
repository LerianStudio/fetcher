package mysql

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/mysql"
	"github.com/LerianStudio/fetcher/pkg/testutil"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newMockLogger() log.Logger {
	return &testutil.MockLogger{}
}

func TestDataSourceConfigMySQL_GetConfig(t *testing.T) {
	cfg := datasource.DataSourceConfig{
		ConfigName: "test-config",
		Type:       "mysql",
		Host:       "localhost",
		Port:       "3306",
	}

	ds := &DataSourceConfigMySQL{
		DataSourceConfig: cfg,
	}

	got := ds.GetConfig()
	assert.Equal(t, cfg, got)
}

func TestDataSourceConfigMySQL_GetType(t *testing.T) {
	ds := &DataSourceConfigMySQL{
		DataSourceConfig: datasource.DataSourceConfig{
			Type: "mysql",
		},
	}

	assert.Equal(t, "mysql", ds.GetType())
}

func TestDataSourceConfigMySQL_Connect(t *testing.T) {
	ds := &DataSourceConfigMySQL{
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

func TestDataSourceConfigMySQL_Close(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mysql.NewMockRepository(ctrl)

		mockRepo.EXPECT().CloseConnection().Return(nil)

		ds := &DataSourceConfigMySQL{
			MySQLRepository: mockRepo,
		}

		err := ds.Close(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, libConstant.DataSourceStatusUnavailable, ds.Status)
	})

	t.Run("close with nil repository", func(t *testing.T) {
		ds := &DataSourceConfigMySQL{
			MySQLRepository: nil,
		}

		err := ds.Close(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, libConstant.DataSourceStatusUnavailable, ds.Status)
	})

	t.Run("close with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mysql.NewMockRepository(ctrl)

		mockRepo.EXPECT().CloseConnection().Return(errors.New("close error"))

		ds := &DataSourceConfigMySQL{
			MySQLRepository: mockRepo,
		}

		err := ds.Close(context.Background())
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
			name: "table in filters",
			filters: map[string]map[string]job.FilterCondition{
				"users": {
					"id":     {Equals: []any{1}},
					"status": {Equals: []any{"active"}},
				},
			},
			tableName:   "users",
			wantNil:     false,
			wantColumns: 2,
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

func TestDataSourceConfigMySQL_Query(t *testing.T) {
	logger := newMockLogger()
	ctx := context.Background()

	schemaResult := []mysql.TableSchema{
		{TableName: "users", Columns: []mysql.ColumnInformation{{Name: "id"}, {Name: "name"}}},
	}

	t.Run("successful query without filters", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mysql.NewMockRepository(ctrl)

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any()).Return(schemaResult, nil)
		mockRepo.EXPECT().Query(gomock.Any(), schemaResult, "users", []string{"id", "name"}, nil).
			Return([]map[string]any{{"id": 1, "name": "John"}}, nil)

		ds := &DataSourceConfigMySQL{
			MySQLRepository: mockRepo,
		}

		tables := map[string][]string{
			"users": {"id", "name"},
		}

		result, err := ds.Query(ctx, tables, nil, logger)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result["users"]))
	})

	t.Run("query with filters", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mysql.NewMockRepository(ctrl)

		filters := map[string]map[string]job.FilterCondition{
			"users": {"status": {Equals: []any{"active"}}},
		}

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any()).Return(schemaResult, nil)
		mockRepo.EXPECT().QueryWithAdvancedFilters(gomock.Any(), schemaResult, "users", []string{"id"}, gomock.Any()).
			Return([]map[string]any{{"id": 1}}, nil)

		ds := &DataSourceConfigMySQL{
			MySQLRepository: mockRepo,
		}

		tables := map[string][]string{
			"users": {"id"},
		}

		result, err := ds.Query(ctx, tables, filters, logger)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("query with schema error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mysql.NewMockRepository(ctrl)

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any()).Return(nil, errors.New("schema error"))

		ds := &DataSourceConfigMySQL{
			MySQLRepository: mockRepo,
		}

		tables := map[string][]string{
			"users": {"id"},
		}

		_, err := ds.Query(ctx, tables, nil, logger)
		assert.Error(t, err)
	})

	t.Run("query execution error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mysql.NewMockRepository(ctrl)

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any()).Return(schemaResult, nil)
		mockRepo.EXPECT().Query(gomock.Any(), schemaResult, "users", []string{"id"}, nil).
			Return(nil, errors.New("query execution failed"))

		ds := &DataSourceConfigMySQL{
			MySQLRepository: mockRepo,
		}

		tables := map[string][]string{
			"users": {"id"},
		}

		_, err := ds.Query(ctx, tables, nil, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "query execution failed")
	})

	t.Run("query with advanced filters error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mysql.NewMockRepository(ctrl)

		filters := map[string]map[string]job.FilterCondition{
			"users": {"status": {Equals: []any{"active"}}},
		}

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any()).Return(schemaResult, nil)
		mockRepo.EXPECT().QueryWithAdvancedFilters(gomock.Any(), schemaResult, "users", []string{"id"}, gomock.Any()).
			Return(nil, errors.New("advanced query failed"))

		ds := &DataSourceConfigMySQL{
			MySQLRepository: mockRepo,
		}

		tables := map[string][]string{
			"users": {"id"},
		}

		_, err := ds.Query(ctx, tables, filters, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "advanced query failed")
	})
}

func TestDataSourceConfigMySQL_GetSchemaInfo(t *testing.T) {
	t.Run("successful schema retrieval", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mysql.NewMockRepository(ctrl)

		schemaResult := []mysql.TableSchema{
			{
				TableName: "users",
				Columns: []mysql.ColumnInformation{
					{Name: "id"},
					{Name: "name"},
					{Name: "email"},
				},
			},
			{
				TableName: "orders",
				Columns: []mysql.ColumnInformation{
					{Name: "id"},
					{Name: "user_id"},
				},
			},
		}

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any()).Return(schemaResult, nil)

		ds := &DataSourceConfigMySQL{
			DataSourceConfig: datasource.DataSourceConfig{
				ConfigName: "test-mysql",
			},
			MySQLRepository: mockRepo,
		}

		ctx := context.Background()

		schema, err := ds.GetSchemaInfo(ctx, nil)
		assert.NoError(t, err)
		assert.Equal(t, "test-mysql", schema.ConfigName)
		assert.Equal(t, 2, len(schema.Tables))
	})

	t.Run("schema retrieval error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mysql.NewMockRepository(ctrl)

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any()).Return(nil, errors.New("db error"))

		ds := &DataSourceConfigMySQL{
			MySQLRepository: mockRepo,
		}

		ctx := context.Background()
		_, err := ds.GetSchemaInfo(ctx, nil)
		assert.Error(t, err)
	})
}
