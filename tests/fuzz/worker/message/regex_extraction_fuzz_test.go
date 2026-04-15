//go:build go1.18
// +build go1.18

package message

import (
	"regexp"
	"testing"

	"github.com/google/uuid"
)

// Limit whitespace to {0,10} to prevent ReDoS (matches production code)
var (
	jobIDRegex = regexp.MustCompile(`"jobId"\s{0,10}:\s{0,10}"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)
	orgIDRegex = regexp.MustCompile(`"organizationId"\s{0,10}:\s{0,10}"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)
)

func FuzzRegexJobIDExtraction(f *testing.F) {
	validUUID := uuid.New().String()

	seeds := []string{
		`{"jobId":"` + validUUID + `"}`,
		`{"jobId" : "` + validUUID + `"}`,
		`malformed json with "jobId":"` + validUUID + `" in it`,
		`{"jobId":"invalid-uuid"}`,
		`{"jobId":""}`,
		`no uuid here`,
		``,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, body string) {
		// Limit input size to realistic message payload (1KB max)
		// Production messages are typically small JSON with UUIDs
		if len(body) > 1024 {
			return
		}

		// Skip inputs with excessive whitespace sequences that could cause slowdowns
		// even with bounded quantifiers
		whitespaceCount := 0
		for _, c := range body {
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				whitespaceCount++
				if whitespaceCount > 50 {
					return
				}
			}
		}

		matches := jobIDRegex.FindStringSubmatch(body)
		if len(matches) > 1 {
			_, _ = uuid.Parse(matches[1])
		}

		orgMatches := orgIDRegex.FindStringSubmatch(body)
		if len(orgMatches) > 1 {
			_, _ = uuid.Parse(orgMatches[1])
		}
	})
}
