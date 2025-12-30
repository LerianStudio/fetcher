package mysql

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/mysql"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// mockLogger is a simplified mock for log.Logger
type mockLogger struct{}

func (m *mockLogger) Info(args ...any)                                      {}
func (m *mockLogger) Infof(format string, args ...any)                      {}
func (m *mockLogger) Infoln(args ...any)                                    {}
func (m *mockLogger) Warn(args ...any)                                      {}
func (m *mockLogger) Warnf(format string, args ...any)                      {}
func (m *mockLogger) Warnln(args ...any)                                    {}
func (m *mockLogger) Warning(args ...any)                                   {}
func (m *mockLogger) Warningf(format string, args ...any)                   {}
func (m *mockLogger) Warningln(args ...any)                                 {}
func (m *mockLogger) Error(args ...any)                                     {}
func (m *mockLogger) Errorf(format string, args ...any)                     {}
func (m *mockLogger) Errorln(args ...any)                                   {}
func (m *mockLogger) Debug(args ...any)                                     {}
func (m *mockLogger) Debugf(format string, args ...any)                     {}
func (m *mockLogger) Debugln(args ...any)                                   {}
func (m *mockLogger) Fatal(args ...any)                                     {}
func (m *mockLogger) Fatalf(format string, args ...any)                     {}
func (m *mockLogger) Fatalln(args ...any)                                   {}
func (m *mockLogger) Panic(args ...any)                                     {}
func (m *mockLogger) Panicf(format string, args ...any)                     {}
func (m *mockLogger) Panicln(args ...any)                                   {}
func (m *mockLogger) Trace(args ...any)                                     {}
func (m *mockLogger) Tracef(format string, args ...any)                     {}
func (m *mockLogger) Traceln(args ...any)                                   {}
func (m *mockLogger) Print(args ...any)                                     {}
func (m *mockLogger) Printf(format string, args ...any)                     {}
func (m *mockLogger) Println(args ...any)                                   {}
func (m *mockLogger) Log(level string, args ...any)                         {}
func (m *mockLogger) Logf(level string, format string, args ...any)         {}
func (m *mockLogger) Logln(level string, args ...any)                       {}
func (m *mockLogger) WithFields(fields ...any) log.Logger                   { return m }
func (m *mockLogger) WithField(key string, value any) log.Logger            { return m }
func (m *mockLogger) WithError(err error) log.Logger                        { return m }
func (m *mockLogger) WithDefaultMessageTemplate(template string) log.Logger { return m }
func (m *mockLogger) GetLevel() string                                      { return "" }
func (m *mockLogger) SetLevel(level string) error                           { return nil }
func (m *mockLogger) IsLevelEnabled(level string) bool                      { return false }
func (m *mockLogger) GetLogger() any                                        { return nil }
func (m *mockLogger) GetOutput() any                                        { return nil }
func (m *mockLogger) SetOutput(output any) error                            { return nil }
func (m *mockLogger) GetFormatter() any                                     { return nil }
func (m *mockLogger) SetFormatter(formatter any) error                      { return nil }
func (m *mockLogger) GetHooks() any                                         { return nil }
func (m *mockLogger) AddHook(hook any) error                                { return nil }
func (m *mockLogger) Clone() any                                            { return m }
func (m *mockLogger) GetContext() any                                       { return nil }
func (m *mockLogger) SetContext(ctx any) error                              { return nil }
func (m *mockLogger) GetCallerInfo() bool                                   { return false }
func (m *mockLogger) SetCallerInfo(enabled bool)                            {}
func (m *mockLogger) GetReportCaller() bool                                 { return false }
func (m *mockLogger) SetReportCaller(enabled bool)                          {}
func (m *mockLogger) GetExitFunc() any                                      { return nil }
func (m *mockLogger) SetExitFunc(exitFunc any) error                        { return nil }
func (m *mockLogger) GetBufferPool() any                                    { return nil }
func (m *mockLogger) SetBufferPool(pool any) error                          { return nil }
func (m *mockLogger) Sync() error                                           { return nil }

func newMockLogger() log.Logger {
	return &mockLogger{}
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mysql.NewMockRepository(ctrl)

	t.Run("successful close", func(t *testing.T) {
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mysql.NewMockRepository(ctrl)
	logger := newMockLogger()
	ctx := context.Background()

	schemaResult := []mysql.TableSchema{
		{TableName: "users", Columns: []mysql.ColumnInformation{{Name: "id"}, {Name: "name"}}},
	}

	t.Run("successful query without filters", func(t *testing.T) {
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
}

func TestDataSourceConfigMySQL_GetSchemaInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mysql.NewMockRepository(ctrl)

	t.Run("successful schema retrieval", func(t *testing.T) {
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
		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any()).Return(nil, errors.New("db error"))

		ds := &DataSourceConfigMySQL{
			MySQLRepository: mockRepo,
		}

		ctx := context.Background()
		_, err := ds.GetSchemaInfo(ctx, nil)
		assert.Error(t, err)
	})
}
