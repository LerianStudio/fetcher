package mongodb

import (
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/stretchr/testify/assert"
)

func TestDataSourceConfigMongoDB_GetType(t *testing.T) {
	t.Run("returns correct type", func(t *testing.T) {
		ds := &DataSourceConfigMongoDB{
			DataSourceConfig: datasource.DataSourceConfig{
				Type: "MONGODB",
			},
		}

		result := ds.GetType()

		assert.Equal(t, "MONGODB", result)
	})

	t.Run("returns empty type when not set", func(t *testing.T) {
		ds := &DataSourceConfigMongoDB{}

		result := ds.GetType()

		assert.Equal(t, "", result)
	})
}

func TestGetCollectionFilters(t *testing.T) {
	tests := []struct {
		name            string
		databaseFilters map[string]map[string]job.FilterCondition
		collection      string
		expectedFilter  map[string]job.FilterCondition
	}{
		{
			name: "with filters for collection",
			databaseFilters: map[string]map[string]job.FilterCondition{
				"users": {"status": {Equals: []any{"active"}}},
			},
			collection:     "users",
			expectedFilter: map[string]job.FilterCondition{"status": {Equals: []any{"active"}}},
		},
		{
			name: "no filters for collection",
			databaseFilters: map[string]map[string]job.FilterCondition{
				"orders": {"status": {Equals: []any{"pending"}}},
			},
			collection:     "users",
			expectedFilter: nil,
		},
		{
			name:            "nil filters",
			databaseFilters: nil,
			collection:      "users",
			expectedFilter:  nil,
		},
		{
			name:            "empty filters map",
			databaseFilters: map[string]map[string]job.FilterCondition{},
			collection:      "users",
			expectedFilter:  nil,
		},
		{
			name: "multiple collections returns correct one",
			databaseFilters: map[string]map[string]job.FilterCondition{
				"users":    {"status": {Equals: []any{"active"}}},
				"orders":   {"amount": {GreaterThan: []any{100}}},
				"products": {"category": {In: []any{"electronics", "books"}}},
			},
			collection:     "orders",
			expectedFilter: map[string]job.FilterCondition{"amount": {GreaterThan: []any{100}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCollectionFilters(tt.databaseFilters, tt.collection)

			assert.Equal(t, tt.expectedFilter, result)
		})
	}
}

func TestDataSourceConfigMongoDB_GetConfig(t *testing.T) {
	t.Run("returns embedded DataSourceConfig", func(t *testing.T) {
		expectedConfig := datasource.DataSourceConfig{
			ID:           "test-id",
			ConfigName:   "test-config",
			Type:         "MONGODB",
			DatabaseName: "testdb",
		}

		ds := &DataSourceConfigMongoDB{
			DataSourceConfig: expectedConfig,
			MongoURI:         "mongodb://localhost:27017",
		}

		result := ds.GetConfig()

		assert.Equal(t, expectedConfig, result)
	})
}
