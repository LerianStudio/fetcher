//go:build go1.18
// +build go1.18

package message

import (
	"encoding/json"
	"testing"

	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
)

func FuzzFilterConditionParsing(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{}`),
		[]byte(`{"eq":["value"]}`),
		[]byte(`{"gt":[100]}`),
		[]byte(`{"gte":["2025-01-01"]}`),
		[]byte(`{"lt":[1000]}`),
		[]byte(`{"lte":["2025-12-31"]}`),
		[]byte(`{"between":[100,1000]}`),
		[]byte(`{"in":["a","b","c"]}`),
		[]byte(`{"nin":["x","y"]}`),
		[]byte(`{"ne":["excluded"]}`),
		[]byte(`{"like":["%pattern%"]}`),
		[]byte(`{"gte":["2025-01-01"],"lte":["2025-12-31"]}`),
		[]byte(`{"eq":[]}`),
		[]byte(`{"eq":null}`),
		[]byte(`{"between":[100]}`),
		[]byte(`{"unknown":["value"]}`),
		[]byte(`{"eq":[123, "string", true, null]}`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var condition modelJob.FilterCondition
		err := json.Unmarshal(data, &condition)
		if err != nil {
			return
		}

		_ = condition.Equals
		_ = condition.GreaterThan
		_ = condition.GreaterOrEqual
		_ = condition.LessThan
		_ = condition.LessOrEqual
		_ = condition.Between
		_ = condition.In
		_ = condition.NotIn
		_ = condition.NotEquals
		_ = condition.Like

		for _, v := range condition.Equals {
			_ = v
		}
		for _, v := range condition.In {
			_ = v
		}
	})
}

func FuzzFilterConditionMap(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"field1":{"eq":["value"]}}`),
		[]byte(`{}`),
		[]byte(`{"":{"eq":["value"]}}`),
		[]byte(`{"field1":{}}`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var filterMap map[string]modelJob.FilterCondition
		err := json.Unmarshal(data, &filterMap)
		if err != nil {
			return
		}

		for field, cond := range filterMap {
			_ = field
			_ = cond.Equals
		}
	})
}
