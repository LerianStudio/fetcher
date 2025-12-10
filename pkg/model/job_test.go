package model

import (
	"testing"
)

// TestFetcherRequest_ComputeRequestHash tests the ComputeRequestHash method.
func TestFetcherRequest_ComputeRequestHash(t *testing.T) {
	t.Run("same requests produce same hash", func(t *testing.T) {
		request1 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {
						"table1": {"field1", "field2"},
					},
				},
				Filters: []FilterRequest{
					{
						Field:    "datasource1.table1.field1",
						Operator: "eq",
						Value:    []any{"value1"},
					},
				},
			},
		}

		request2 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {
						"table1": {"field1", "field2"},
					},
				},
				Filters: []FilterRequest{
					{
						Field:    "datasource1.table1.field1",
						Operator: "eq",
						Value:    []any{"value1"},
					},
				},
			},
		}

		hash1, err1 := request1.ComputeRequestHash()
		if err1 != nil {
			t.Fatalf("unexpected error computing hash1: %v", err1)
		}

		hash2, err2 := request2.ComputeRequestHash()
		if err2 != nil {
			t.Fatalf("unexpected error computing hash2: %v", err2)
		}

		if hash1 != hash2 {
			t.Fatalf("expected same hash for identical requests, got %s and %s", hash1, hash2)
		}
	})

	t.Run("different requests produce different hashes", func(t *testing.T) {
		request1 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {
						"table1": {"field1"},
					},
				},
			},
		}

		request2 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource2": {
						"table2": {"field2"},
					},
				},
			},
		}

		hash1, err1 := request1.ComputeRequestHash()
		if err1 != nil {
			t.Fatalf("unexpected error computing hash1: %v", err1)
		}

		hash2, err2 := request2.ComputeRequestHash()
		if err2 != nil {
			t.Fatalf("unexpected error computing hash2: %v", err2)
		}

		if hash1 == hash2 {
			t.Fatalf("expected different hashes for different requests, both got %s", hash1)
		}
	})

	t.Run("hash is 64 characters SHA-256 hex", func(t *testing.T) {
		request := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {
						"table1": {"field1"},
					},
				},
			},
		}

		hash, err := request.ComputeRequestHash()
		if err != nil {
			t.Fatalf("unexpected error computing hash: %v", err)
		}

		if len(hash) != 64 {
			t.Fatalf("expected hash length 64 (SHA-256 hex), got %d", len(hash))
		}

		// Verify it's valid hex
		for _, c := range hash {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Fatalf("hash contains invalid hex character: %c", c)
			}
		}
	})

	t.Run("metadata does not affect hash", func(t *testing.T) {
		request1 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {"table1": {"field1"}},
				},
			},
			Metadata: map[string]any{"key1": "value1"},
		}

		request2 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {"table1": {"field1"}},
				},
			},
			Metadata: map[string]any{"key2": "value2"},
		}

		hash1, _ := request1.ComputeRequestHash()
		hash2, _ := request2.ComputeRequestHash()

		if hash1 != hash2 {
			t.Fatalf("metadata should not affect hash, got different hashes: %s vs %s", hash1, hash2)
		}
	})
}
