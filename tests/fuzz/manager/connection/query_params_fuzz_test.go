//go:build go1.18
// +build go1.18

package connection

import (
	"testing"

	httpUtils "github.com/LerianStudio/fetcher/pkg/net/http"
)

func FuzzQueryParameterValidation(f *testing.F) {
	f.Add("10", "1", "desc", "2025-01-01", "2025-01-31")
	f.Add("0", "0", "asc", "", "")
	f.Add("100", "1", "desc", "", "")
	f.Add("101", "1", "desc", "", "")
	f.Add("-1", "1", "desc", "", "")
	f.Add("abc", "1", "desc", "", "")
	f.Add("10", "-1", "desc", "", "")
	f.Add("10", "1", "invalid", "", "")
	f.Add("10", "1", "desc", "invalid", "")

	f.Fuzz(func(t *testing.T, limit, page, sortOrder, startDate, endDate string) {
		params := map[string]string{
			"limit":     limit,
			"page":      page,
			"sortOrder": sortOrder,
			"startDate": startDate,
			"endDate":   endDate,
		}
		queryHeader, err := httpUtils.ValidateParameters(params)
		_ = err
		if queryHeader != nil {
			_ = queryHeader.Limit
			_ = queryHeader.Page
			_ = queryHeader.ToOffsetPagination()
		}
	})
}

func FuzzMetadataQueryParams(f *testing.F) {
	f.Add("metadata.key1", "value1")
	f.Add("metadata.", "value")
	f.Add("metadata.key", "")

	f.Fuzz(func(t *testing.T, key, value string) {
		params := map[string]string{key: value}
		queryHeader, _ := httpUtils.ValidateParameters(params)
		if queryHeader != nil && queryHeader.Metadata != nil {
			for k, v := range *queryHeader.Metadata {
				_ = k
				_ = v
			}
		}
	})
}
