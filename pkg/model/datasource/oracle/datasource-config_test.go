package oracle

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/tablenorm"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/v2/pkg/model/job"
	"github.com/LerianStudio/fetcher/v2/pkg/oracle"
	"github.com/LerianStudio/fetcher/v2/pkg/testutil"
	libConstant "github.com/LerianStudio/lib-commons/v5/commons/constants"
	"github.com/LerianStudio/lib-observability/log"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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
	t.Run("successful close", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := oracle.NewMockRepository(ctrl)

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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := oracle.NewMockRepository(ctrl)

		mockRepo.EXPECT().CloseConnection().Return(errors.New("close error"))

		ds := &DataSourceConfigOracle{
			OracleRepository: mockRepo,
		}

		err := ds.Close(context.Background())
		assert.Error(t, err)
	})
}

func TestDataSourceConfigOracle_Query(t *testing.T) {
	logger := newMockLogger()
	ctx := context.Background()

	schemaResult := []oracle.TableSchema{
		{TableName: "USERS", Columns: []oracle.ColumnInformation{{Name: "ID"}, {Name: "NAME"}}},
	}

	t.Run("successful query without filters", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := oracle.NewMockRepository(ctrl)

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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := oracle.NewMockRepository(ctrl)

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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := oracle.NewMockRepository(ctrl)

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
	t.Run("successful schema retrieval", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := oracle.NewMockRepository(ctrl)

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

	// TestDataSourceConfigOracle_GetSchemaInfo / "canonical normalization yields physical
	// UPPERCASE" PINS THE UPPERCASE-CANONICAL CONTRACT. This is the exact divergence the
	// contract closes: GetSchemaInfo INTERNALLY lowercases the physical UPPERCASE catalog
	// names (all_tab_columns) it reads — but the extracted result rows are keyed by the
	// PHYSICAL UPPERCASE columns (pkg/oracle.createRowMap). A lowercase schema against
	// UPPERCASE data is the mismatch. The canonical normalizer (tablenorm, used by the
	// Manager snapshot/validation AND the Worker extraction) RE-FOLDS GetSchemaInfo's
	// lowercase output back to UPPERCASE, realigning schema == data.
	//
	// !!! IF THIS TEST FAILS, DO NOT "FIX" IT BY LOOSENING THE ASSERTION. The
	// !!! UPPERCASE-canonical contract (tablenorm Oracle fold = ToUpper, the Manager
	// !!! normalize*ForValidation delegating to tablenorm, the Oracle-gated snapshot
	// !!! normalize=true, and the result-key waiver "Oracle identifiers are uppercased")
	// !!! must be re-evaluated TOGETHER — otherwise the Manager will validate one
	// !!! identity and the Worker/data will carry another.
	t.Run("canonical normalization yields physical UPPERCASE", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := oracle.NewMockRepository(ctrl)

		// The adapter reports physical Oracle identifiers UPPERCASED, as the data
		// dictionary does.
		schemaResult := []oracle.TableSchema{
			{
				TableName: "ACCOUNTS",
				Columns: []oracle.ColumnInformation{
					{Name: "ID"},
					{Name: "BALANCE"},
				},
			},
		}

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(schemaResult, nil)

		ds := &DataSourceConfigOracle{
			DataSourceConfig: datasource.DataSourceConfig{ConfigName: "ora-pin"},
			OracleRepository: mockRepo,
		}

		schema, err := ds.GetSchemaInfo(context.Background(), nil)
		assert.NoError(t, err)

		// FACT (unchanged, documented): GetSchemaInfo lowercases its output.
		assert.True(t, schema.HasTable("accounts"),
			"GetSchemaInfo lowercases the table name (this is WHY the canonical fold must re-uppercase)")
		assert.True(t, schema.HasField("accounts", "id"),
			"GetSchemaInfo lowercases column names (this is WHY the canonical fold must re-uppercase)")

		// CONTRACT: the canonical normalizer re-folds GetSchemaInfo's lowercase output to
		// the PHYSICAL UPPERCASE identity, realigning the schema identity with the
		// UPPERCASE data keys createRowMap emits.
		assert.Equal(t, "ACCOUNTS", tablenorm.NormalizeTable(model.TypeOracle, "accounts"),
			"canonical Oracle table identity must be physical UPPERCASE")
		assert.Equal(t, "ID", tablenorm.NormalizeField(model.TypeOracle, "id"),
			"canonical Oracle field identity must be physical UPPERCASE")
		assert.Equal(t, "BALANCE", tablenorm.NormalizeField(model.TypeOracle, "balance"),
			"canonical Oracle field identity must be physical UPPERCASE")
	})

	t.Run("schema retrieval error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := oracle.NewMockRepository(ctrl)

		mockRepo.EXPECT().GetDatabaseSchema(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

		ds := &DataSourceConfigOracle{
			OracleRepository: mockRepo,
		}

		ctx := context.Background()
		_, err := ds.GetSchemaInfo(ctx, nil)
		assert.Error(t, err)
	})
}
