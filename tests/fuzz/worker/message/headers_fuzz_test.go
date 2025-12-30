//go:build go1.18
// +build go1.18

package message

import (
	"testing"

	"github.com/google/uuid"
)

func FuzzMessageHeadersParsing(f *testing.F) {
	validUUID := uuid.New().String()

	f.Add("jobId", validUUID)
	f.Add("jobId", "invalid-uuid")
	f.Add("jobId", "")
	f.Add("organizationId", validUUID)
	f.Add("X-Job-ID", validUUID)
	f.Add("unknown-header", "value")

	f.Fuzz(func(t *testing.T, key, value string) {
		headers := map[string]any{
			key: value,
		}

		if jobIDHeader, exists := headers["jobId"]; exists {
			if jobIDStr, ok := jobIDHeader.(string); ok {
				_, _ = uuid.Parse(jobIDStr)
			}
		}

		if orgIDHeader, exists := headers["organizationId"]; exists {
			if orgIDStr, ok := orgIDHeader.(string); ok {
				_, _ = uuid.Parse(orgIDStr)
			}
		}
	})
}
