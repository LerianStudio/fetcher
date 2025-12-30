package job

import (
	"encoding/json"
	"testing"
)

func TestJobQueuePayload_JSONSerialization(t *testing.T) {
	tests := []struct {
		name    string
		payload JobQueuePayload
	}{
		{
			name: "full payload",
			payload: JobQueuePayload{
				DataRequest: DataRequest{
					Filters: map[string]map[string]map[string]FilterCondition{
						"config1": {
							"table1": {
								"column1": {
									Equals: []any{"value1", "value2"},
								},
							},
						},
					},
					MappedFields: map[string]map[string][]string{
						"config1": {
							"table1": {"field1", "field2"},
						},
					},
				},
				Metadata: map[string]any{
					"key1": "value1",
					"key2": 123,
				},
			},
		},
		{
			name: "empty payload",
			payload: JobQueuePayload{
				DataRequest: DataRequest{},
				Metadata:    nil,
			},
		},
		{
			name: "payload with only filters",
			payload: JobQueuePayload{
				DataRequest: DataRequest{
					Filters: map[string]map[string]map[string]FilterCondition{
						"config1": {
							"table1": {
								"id": {
									GreaterThan: []any{100},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			// Deserialize
			var got JobQueuePayload
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			// Re-serialize for comparison
			gotData, _ := json.Marshal(got)
			wantData, _ := json.Marshal(tt.payload)

			if string(gotData) != string(wantData) {
				t.Errorf("Roundtrip failed:\ngot:  %s\nwant: %s", gotData, wantData)
			}
		})
	}
}

func TestFilterCondition_AllOperators(t *testing.T) {
	tests := []struct {
		name   string
		filter FilterCondition
		json   string
	}{
		{
			name:   "equals operator",
			filter: FilterCondition{Equals: []any{"active", "pending"}},
			json:   `{"eq":["active","pending"]}`,
		},
		{
			name:   "greater than operator",
			filter: FilterCondition{GreaterThan: []any{100}},
			json:   `{"gt":[100]}`,
		},
		{
			name:   "greater or equal operator",
			filter: FilterCondition{GreaterOrEqual: []any{"2025-01-01"}},
			json:   `{"gte":["2025-01-01"]}`,
		},
		{
			name:   "less than operator",
			filter: FilterCondition{LessThan: []any{1000}},
			json:   `{"lt":[1000]}`,
		},
		{
			name:   "less or equal operator",
			filter: FilterCondition{LessOrEqual: []any{"2025-12-31"}},
			json:   `{"lte":["2025-12-31"]}`,
		},
		{
			name:   "between operator",
			filter: FilterCondition{Between: []any{100, 1000}},
			json:   `{"between":[100,1000]}`,
		},
		{
			name:   "in operator",
			filter: FilterCondition{In: []any{"active", "pending", "approved"}},
			json:   `{"in":["active","pending","approved"]}`,
		},
		{
			name:   "not in operator",
			filter: FilterCondition{NotIn: []any{"deleted", "archived"}},
			json:   `{"nin":["deleted","archived"]}`,
		},
		{
			name:   "not equals operator",
			filter: FilterCondition{NotEquals: []any{"inactive"}},
			json:   `{"ne":["inactive"]}`,
		},
		{
			name:   "like operator",
			filter: FilterCondition{Like: []any{"%active%"}},
			json:   `{"like":["%active%"]}`,
		},
		{
			name: "combined operators",
			filter: FilterCondition{
				GreaterOrEqual: []any{"2025-01-01"},
				LessOrEqual:    []any{"2025-12-31"},
			},
			json: `{"gte":["2025-01-01"],"lte":["2025-12-31"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.filter)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			// Unmarshal the expected JSON and re-marshal for normalized comparison
			var expected FilterCondition
			if err := json.Unmarshal([]byte(tt.json), &expected); err != nil {
				t.Fatalf("json.Unmarshal(expected) error = %v", err)
			}
			expectedData, _ := json.Marshal(expected)

			if string(data) != string(expectedData) {
				t.Errorf("Marshal mismatch:\ngot:  %s\nwant: %s", data, expectedData)
			}
		})
	}
}

func TestQueueMessage_JSONSerialization(t *testing.T) {
	msg := QueueMessage{
		Name: "test-queue",
		Body: `{"job_id":"123","status":"pending"}`,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got QueueMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.Name != msg.Name {
		t.Errorf("Name = %q, want %q", got.Name, msg.Name)
	}
	if got.Body != msg.Body {
		t.Errorf("Body = %q, want %q", got.Body, msg.Body)
	}
}

func TestDataRequest_EmptyMaps(t *testing.T) {
	dr := DataRequest{
		Filters:      nil,
		MappedFields: nil,
	}

	data, err := json.Marshal(dr)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got DataRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Nil maps should serialize as null
	if got.Filters != nil && len(got.Filters) != 0 {
		t.Errorf("Filters should be nil or empty, got %v", got.Filters)
	}
}
