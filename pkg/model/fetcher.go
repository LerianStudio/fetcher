package model

// JobQueuePayload represents the message structure for an extraction job.
//
// swagger:model JobQueuePayload
//
// @Description JobQueuePayload represents the message structure for an extraction job in RabbitMQ.
type JobQueuePayload struct {
	DataRequest DataRequest    `json:"data_request"`
	Metadata    map[string]any `json:"metadata"`
}

// DataRequest encapsulates filtering criteria and field mappings for data extraction requests.
//
// swagger:model DataRequest
//
// @Description DataRequest encapsulates filtering criteria and field mappings for data extraction requests.
type DataRequest struct {
	Filters      map[string]map[string]map[string]FilterCondition `json:"filters"`
	MappedFields map[string]map[string][]string                   `json:"mappedFields"`
}

// FilterCondition defines advanced filtering conditions for report generation.
// Supports multiple operators for complex queries including range, equality, and list-based filters.
type FilterCondition struct {
	// Equals specifies exact value matches. Multiple values treated as OR conditions.
	// Example: {"eq": ["active", "pending"]} matches records where field equals "active" OR "pending"
	Equals []any `json:"eq,omitempty"`

	// GreaterThan specifies values that must be greater than the provided value.
	// Should contain exactly one value for comparison.
	// Example: {"gt": [100]} matches records where field > 100
	GreaterThan []any `json:"gt,omitempty"`

	// GreaterOrEqual specifies values that must be greater than or equal to the provided value.
	// Should contain exactly one value for comparison.
	// Example: {"gte": ["2025-06-01"]} matches records where field >= "2025-06-01"
	GreaterOrEqual []any `json:"gte,omitempty"`

	// LessThan specifies values that must be less than the provided value.
	// Should contain exactly one value for comparison.
	// Example: {"lt": [1000]} matches records where field < 1000
	LessThan []any `json:"lt,omitempty"`

	// LessOrEqual specifies values that must be less than or equal to the provided value.
	// Should contain exactly one value for comparison.
	// Example: {"lte": ["2025-06-30"]} matches records where field <= "2025-06-30"
	LessOrEqual []any `json:"lte,omitempty"`

	// Between specifies a range condition with exactly two values [min, max].
	// Matches records where min <= field <= max
	// Example: {"between": [100, 1000]} matches records where 100 <= field <= 1000
	Between []any `json:"between,omitempty"`

	// In specifies a list of values where the field must match any one of them.
	// Multiple values treated as OR conditions.
	// Example: {"in": ["active", "pending", "suspended"]} matches any of these statuses
	In []any `json:"in,omitempty"`

	// NotIn specifies a list of values where the field must NOT match any of them.
	// Multiple values treated as AND NOT conditions.
	// Example: {"nin": ["deleted", "archived"]} excludes these statuses
	NotIn []any `json:"nin,omitempty"`
}

// QueueMessage represents the structure for generating messages in the queue.
//
// swagger:model QueueMessage
//
// @Description QueueMessage represents the structure for generating messages in the queue.
type QueueMessage struct {
	Name string `json:"queue_name"`
	Body string `json:"queue_body"`
}
