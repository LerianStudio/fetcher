package oracle

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/oracle"
	"github.com/LerianStudio/fetcher/pkg/testutil"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newMockLogger() log.Logger {
	return &testutil.MockLogger{}
}

func TestDataSourceConfigOracle_GetConfig(t *testing.T) {
	cfg := datasource.DataSourceConfig{
		ConfigName: "test-config",
		Type:       "oracle",
	}

	ds := &DataSourceConfigOracle{
		DataSourceConfig: cfg,
	}

	got := ds.GetConfig()
	assert.Equal(t, cfg, got)
}

func TestDataSourceConfigOracle_GetType(t *testing.T) {
	ds := &DataSourceConfigOracle{
		DataSourceConfig: datasource.DataSourceConfig{
			Type: "oracle",
		},
	}

	assert.Equal(t, "oracle", ds.GetType())
}

func TestDataSourceConfigOracle_Connect(t *testing.T) {
	ds := &DataSourceConfigOracle{
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

func TestDataSourceConfigOracle_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := oracle.NewMockRepository(ctrl)

	t.Run("successful close", func(t *testing.T) {
		mockRepo.EXPECT().CloseConnection().Return(nil)

		ds := &DataSourceConfigOracle{
			OracleRepository: mockRepo,
		}

		err := ds.Close(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, libConstant.DataSourceStatusUnavailable, ds.Status)
	})

	t.Run("close with nil repository", func(t *testing.T) {
		ds := &DataSourceConfigOracle{
			OracleRepository: nil,
		}

		err := ds.Close(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, libConstant.DataSourceStatusUnavailable, ds.Status)
	})

	t.Run("close with error", func(t *testing.T) {
		mockRepo.EXPECT().CloseConnection().Return(errors.New("close error"))

		ds := &DataSourceConfigOracle{
			OracleRepository: mockRepo,
		}

		err := ds.Close(context.Background())
		assert.Error(t, err)
	})
}

func TestDataSourceConfigOracle_Query(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := oracle.NewMockRepository(ctrl)
	logger := newMockLogger()
	ctx := context.Background()

	schemaResult := []oracle.TableSchema{
		{TableName: "USERS", Columns: []oracle.ColumnInformation{{Name: "ID"}, {Name: "NAME"}}},
	}

	t.Run("successful query without filters", func(t *testing.T) {
		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(schemaResult, nil)
		mockRepo.EXPECT().Query(gomock.Any(), schemaResult, "USERS", []string{"ID", "NAME"}, nil).
			Return([]map[string]any{{"ID": 1, "NAME": "John"}}, nil)

		ds := &DataSourceConfigOracle{
			OracleRepository: mockRepo,
		}

		tables := map[string][]string{
			"USERS": {"ID", "NAME"},
		}

		result, err := ds.Query(ctx, tables, nil, logger)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result["USERS"]))
	})

	t.Run("query with filters", func(t *testing.T) {
		filters := map[string]map[string]job.FilterCondition{
			"USERS": {"STATUS": {Equals: []any{"active"}}},
		}

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(schemaResult, nil)
		mockRepo.EXPECT().QueryWithAdvancedFilters(gomock.Any(), schemaResult, "USERS", []string{"ID"}, gomock.Any()).
			Return([]map[string]any{{"ID": 1}}, nil)

		ds := &DataSourceConfigOracle{
			OracleRepository: mockRepo,
		}

		tables := map[string][]string{
			"USERS": {"ID"},
		}

		result, err := ds.Query(ctx, tables, filters, logger)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("query with schema error", func(t *testing.T) {
		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(nil, errors.New("schema error"))

		ds := &DataSourceConfigOracle{
			OracleRepository: mockRepo,
		}

		tables := map[string][]string{
			"USERS": {"ID"},
		}

		_, err := ds.Query(ctx, tables, nil, logger)
		assert.Error(t, err)
	})
}

func TestDataSourceConfigOracle_GetSchemaInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := oracle.NewMockRepository(ctrl)

	t.Run("successful schema retrieval", func(t *testing.T) {
		schemaResult := []oracle.TableSchema{
			{
				TableName: "USERS",
				Columns: []oracle.ColumnInformation{
					{Name: "ID"},
					{Name: "NAME"},
				},
			},
		}

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(schemaResult, nil)

		ds := &DataSourceConfigOracle{
			DataSourceConfig: datasource.DataSourceConfig{
				ConfigName: "test-oracle",
			},
			OracleRepository: mockRepo,
		}

		ctx := context.Background()

		schema, err := ds.GetSchemaInfo(ctx, nil)
		assert.NoError(t, err)
		assert.Equal(t, "test-oracle", schema.ConfigName)
	})

	t.Run("schema retrieval error", func(t *testing.T) {
		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

		ds := &DataSourceConfigOracle{
			OracleRepository: mockRepo,
		}

		ctx := context.Background()
		_, err := ds.GetSchemaInfo(ctx, nil)
		assert.Error(t, err)
	})
}
