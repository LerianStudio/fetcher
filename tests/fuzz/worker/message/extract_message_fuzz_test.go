//go:build go1.18
// +build go1.18

package message

import (
	"encoding/json"
	"testing"

	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/tests/fuzz/shared/generators"
	"github.com/google/uuid"
)

type ExtractExternalDataMessage struct {
	JobID          uuid.UUID                                                 `json:"jobId"`
	OrganizationID uuid.UUID                                                 `json:"organizationId"`
	MappedFields   map[string]map[string][]string                            `json:"mappedFields"`
	Filters        map[string]map[string]map[string]modelJob.FilterCondition `json:"filters"`
	Metadata       map[string]any                                            `json:"metadata"`
}

func FuzzExtractExternalDataMessageParsing(f *testing.F) {
	seeds := [][]byte{
		generators.GenerateExtractExternalDataSeed(),
		[]byte(`{"jobId":"00000000-0000-0000-0000-000000000000","organizationId":"00000000-0000-0000-0000-000000000000"}`),
		[]byte(`{}`),
		[]byte(`{"jobId":"invalid-uuid"}`),
		[]byte(`{"mappedFields":{}}`),
		[]byte(`{"filters":{}}`),
		[]byte(`{"metadata":{"key":"value"}}`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var message ExtractExternalDataMessage
		err := json.Unmarshal(data, &message)
		if err != nil {
			return
		}

		_ = message.JobID
		_ = message.OrganizationID
		_ = message.JobID == uuid.Nil

		if message.MappedFields != nil {
			for ds, tables := range message.MappedFields {
				_ = ds
				for t, fields := range tables {
					_ = t
					for _, f := range fields {
						_ = f
					}
				}
			}
		}

		if message.Filters != nil {
			for ds, tables := range message.Filters {
				_ = ds
				for t, fields := range tables {
					_ = t
					for f, cond := range fields {
						_ = f
						_ = cond.Equals
						_ = cond.GreaterThan
					}
				}
			}
		}
	})
}
