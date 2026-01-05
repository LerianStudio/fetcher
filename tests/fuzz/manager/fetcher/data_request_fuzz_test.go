//go:build go1.18
// +build go1.18

package fetcher

import (
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
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

func FuzzFilterFieldParsing(f *testing.F) {
	f.Add("config.table.field")
	f.Add("config.schema.table.field")
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add("...")
	f.Add("config.")
	f.Add(".table.field")
	f.Add("a.b.c.d.e")
	f.Add("ab")

	f.Fuzz(func(t *testing.T, field string) {
		parsed, err := model.ParseFilterField(field)
		if err == nil && parsed != nil {
			_ = parsed.ConfigName
			_ = parsed.TableName
			_ = parsed.FieldName
		}
	})
}

func FuzzFilterReferencesValidation(f *testing.F) {
	f.Add("ds1.table1.field1", "eq", "ds1")

	f.Fuzz(func(t *testing.T, filterField, operator, mappedDS string) {
		filters := []model.Filter{
			{
				Field:    filterField,
				Operator: operator,
				Value:    []any{"value"},
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
