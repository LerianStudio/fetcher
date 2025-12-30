package sqlserver

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/sqlserver"
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := sqlserver.NewMockRepository(ctrl)

	t.Run("successful close", func(t *testing.T) {
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
