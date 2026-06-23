//go:build go1.18
// +build go1.18

package schema

import (
	"encoding/json"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/tests/fuzz/shared/generators"
)

func FuzzSchemaValidationRequestParsing(f *testing.F) {
	seeds := [][]byte{
		generators.GenerateSchemaValidationSeed(),
		[]byte(`{"mappedFields":{}}`),
		[]byte(`{"mappedFields":null}`),
		[]byte(`{}`),
		[]byte(`{"mappedFields":{"ds1":{"t1":["f1","f2"]}}}`),
		[]byte(`{"mappedFields":{"ds1":{"t1":[]}}}`),
		[]byte(`{"mappedFields":{"":{"t1":["f1"]}}}`),
		[]byte(`{"mappedFields":{"ds1":{}}}`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var request model.SchemaValidationRequest
		err := json.Unmarshal(data, &request)
		if err != nil {
			return
		}

		_ = request.ToMapWithMask()

		if request.MappedFields != nil {
			for ds, tables := range request.MappedFields {
				_ = ds
				for t, fields := range tables {
					_ = t
					for _, f := range fields {
						_ = f
					}
				}
			}
		}
	})
}

func FuzzSchemaValidationSpecValidation(f *testing.F) {
	f.Add("ds1", "table1", "field1")
	f.Add("", "table1", "field1")
	f.Add("   ", "table1", "field1")
	f.Add("ds1", "", "field1")
	f.Add("ds1", "table1", "")

	f.Fuzz(func(t *testing.T, datasource, table, field string) {
		request := model.SchemaValidationRequest{
			MappedFields: map[string]map[string][]string{
				datasource: {
					table: {field},
				},
			},
		}

		spec := model.NewSchemaValidationSpec(request)

		_ = spec.Validate()
		_ = spec.GetConfigNames()
		_ = spec.GetTablesByConfigName(datasource)
	})
}

func FuzzSchemaValidationLimits(f *testing.F) {
	// Test at and around actual domain model limits:
	// MaxDataSourcesPerRequest = 10, MaxTablesPerDataSource = 20, MaxFieldsPerTable = 50
	f.Add(1, 1, 1)
	f.Add(model.MaxDataSourcesPerRequest, 1, 1)   // At DS limit
	f.Add(model.MaxDataSourcesPerRequest+1, 1, 1) // Over DS limit
	f.Add(1, model.MaxTablesPerDataSource, 1)     // At tables limit
	f.Add(1, model.MaxTablesPerDataSource+1, 1)   // Over tables limit
	f.Add(1, 1, model.MaxFieldsPerTable)          // At fields limit
	f.Add(1, 1, model.MaxFieldsPerTable+1)        // Over fields limit

	f.Fuzz(func(t *testing.T, numDS, numTables, numFields int) {
		// Cap to slightly above limits to test boundary behavior without memory exhaustion
		if numDS > model.MaxDataSourcesPerRequest+2 {
			numDS = model.MaxDataSourcesPerRequest + 2
		}
		if numDS < 0 {
			numDS = 0
		}
		if numTables > model.MaxTablesPerDataSource+2 {
			numTables = model.MaxTablesPerDataSource + 2
		}
		if numTables < 0 {
			numTables = 0
		}
		if numFields > model.MaxFieldsPerTable+2 {
			numFields = model.MaxFieldsPerTable + 2
		}
		if numFields < 0 {
			numFields = 0
		}

		mappedFields := make(map[string]map[string][]string)
		for i := 0; i < numDS; i++ {
			dsName := string(rune('a' + i))
			mappedFields[dsName] = make(map[string][]string)
			for j := 0; j < numTables; j++ {
				tableName := string(rune('A' + j))
				fields := make([]string, numFields)
				for k := 0; k < numFields; k++ {
					fields[k] = string(rune('x' + k%26))
				}
				mappedFields[dsName][tableName] = fields
			}
		}

		spec := &model.SchemaValidationSpec{MappedFields: mappedFields}
		_ = spec.Validate()
	})
}
