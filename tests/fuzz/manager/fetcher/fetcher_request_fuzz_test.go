//go:build go1.18
// +build go1.18

package fetcher

import (
	"encoding/json"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/tests/fuzz/shared/generators"
)

func FuzzFetcherRequestParsing(f *testing.F) {
	seeds := [][]byte{
		generators.GenerateFetcherRequestSeed(),
		[]byte(`{"dataRequest":{"mappedFields":{}}}`),
		[]byte(`{"dataRequest":{"mappedFields":null}}`),
		[]byte(`{"dataRequest":null}`),
		[]byte(`{}`),
		[]byte(`{"dataRequest":{"mappedFields":{"ds1":{"t1":["f1"]}}}}`),
		[]byte(`{"dataRequest":{"mappedFields":{"ds1":{"t1":[]}}}}`),
		[]byte(`{"dataRequest":{"mappedFields":{"":{"t1":["f1"]}}}}`),
		[]byte(`{"dataRequest":{"filters":[{"field":"ds.t.f","operator":"eq","value":["x"]}]}}`),
		[]byte(`{"metadata":{"key":"value"}}`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var request model.FetcherRequest
		err := json.Unmarshal(data, &request)
		if err != nil {
			return
		}

		_, _ = request.ComputeRequestHash()

		if request.DataRequest.MappedFields != nil {
			for ds, tables := range request.DataRequest.MappedFields {
				_ = ds
				for t, fields := range tables {
					_ = t
					for _, f := range fields {
						_ = f
					}
				}
			}
		}

		if request.DataRequest.Filters != nil {
			for _, filter := range request.DataRequest.Filters {
				_ = filter.Field
				_ = filter.Operator
				_ = filter.Value
			}
		}

		if request.Metadata != nil {
			for k, v := range request.Metadata {
				_ = k
				_ = v
			}
		}
	})
}
