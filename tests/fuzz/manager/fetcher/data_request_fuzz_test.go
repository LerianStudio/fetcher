//go:build go1.18
// +build go1.18

package fetcher

import (
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/model/job"
)

func FuzzDataRequestValidation(f *testing.F) {
	f.Add("ds1", "table1", "field1")
	f.Add("", "table1", "field1")
	f.Add("ds1", "", "field1")
	f.Add("ds1", "table1", "")
	f.Add("ds1.schema", "table1", "f1")
	f.Add(string(make([]byte, 1000)), "t", "f")

	f.Fuzz(func(t *testing.T, datasource, table, field string) {
		mappedFields := map[string]map[string][]string{
			datasource: {
				table: {field},
			},
		}

		job := &model.Job{
			MappedFields: mappedFields,
		}

		_ = job.IsValid()
		_ = job.GetDatasourceNames()
		_ = job.ToMappedFieldsMap()
	})
}

func FuzzFilterReferencesValidation(f *testing.F) {
	f.Add("ds1", "table1", "field1", "ds1")

	f.Fuzz(func(t *testing.T, datasource, table, field, mappedDS string) {
		filters := model.NestedFilters{
			datasource: {
				table: {
					field: job.FilterCondition{
						Equals: []any{"value"},
					},
				},
			},
		}

		mappedFields := map[string]map[string][]string{
			mappedDS: {
				"table1": {"field1"},
			},
		}

		_ = model.ValidateFilterReferences(filters, mappedFields)
	})
}
