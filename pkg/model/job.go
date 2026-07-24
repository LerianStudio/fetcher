package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/model/job"
	"github.com/google/uuid"
)

// NestedFilters represents the nested filter format for data extraction jobs.
// Structure: map[datasource]map[table]map[field]FilterCondition
// This format is used for both API input/output and internal storage.
type NestedFilters = map[string]map[string]map[string]job.FilterCondition

const (
	// MaxDatasourcesPerJob is the maximum number of datasources allowed per job.
	MaxDatasourcesPerJob = 10
)

var (
	errMappedFieldsRequired  = errors.New("mappedFields is required")
	errMappedFieldsEmpty     = errors.New("mappedFields cannot be empty")
	errDatasourceFieldsEmpty = errors.New("datasource must have at least one table with fields")
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

var validJobStatuses = map[JobStatus]struct{}{
	JobStatusPending:    {},
	JobStatusProcessing: {},
	JobStatusCompleted:  {},
	JobStatusFailed:     {},
}

func NewJobStatusFromString(status string) (JobStatus, error) {
	normalized := JobStatus(strings.ToLower(strings.TrimSpace(status)))
	if !normalized.IsValid() {
		return "", errors.New("invalid job status")
	}

	return normalized, nil
}

func (js JobStatus) IsValid() bool {
	_, ok := validJobStatuses[js]
	return ok
}

// Job represents the API payload stored in the jobs collection.
type Job struct {
	ID           uuid.UUID
	Metadata     map[string]any
	MappedFields map[string]map[string][]string
	Filters      NestedFilters
	Status       JobStatus
	ResultPath   string
	ResultHMAC   string
	RequestHash  string
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

func NewJob(
	metadata map[string]any,
	mappedFields map[string]map[string][]string,
	filters NestedFilters,
	status JobStatus,
	resultPath string,
	requestHash string,
	createdAt time.Time,
	completedAt *time.Time,
) (*Job, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &Job{
		ID:           id,
		Metadata:     metadata,
		MappedFields: mappedFields,
		Filters:      filters,
		Status:       status,
		ResultPath:   resultPath,
		RequestHash:  requestHash,
		CreatedAt:    createdAt,
		CompletedAt:  completedAt,
	}, nil
}

// ValidateFilterReferences validates that all filter datasource references exist in mappedFields.
// Only validates that datasource names exist - table/field resolution is handled by DataSource adapters
// which can apply default schema fallback logic (e.g., "transactions" -> "public.transactions").
func ValidateFilterReferences(filters NestedFilters, mappedFields map[string]map[string][]string) error {
	if len(filters) == 0 {
		return nil
	}

	var validationErrors []string

	for datasourceName := range filters {
		// Check if datasource exists in mappedFields
		if _, exists := mappedFields[datasourceName]; !exists {
			validationErrors = append(validationErrors,
				fmt.Sprintf("filter references unknown datasource '%s' not found in mappedFields", datasourceName))
		}
	}

	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "; "))
	}

	return nil
}

// IsValid validates the FetcherRequest structure.
// It checks that mappedFields is present and non-empty, and validates all filters.
func (r *Job) IsValid() error {
	if r.MappedFields == nil {
		return pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrInvalidDataRequest.Error(),
			Title:      "Invalid Data Request",
			Message:    errMappedFieldsRequired.Error(),
			Err:        errMappedFieldsRequired,
		}
	}

	if len(r.MappedFields) == 0 {
		return pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrInvalidDataRequest.Error(),
			Title:      "Invalid Data Request",
			Message:    errMappedFieldsEmpty.Error(),
			Err:        errMappedFieldsEmpty,
		}
	}

	// Validate each datasource has at least one table with fields
	for _, tables := range r.MappedFields {
		hasFields := false

		for _, fields := range tables {
			if len(fields) > 0 {
				hasFields = true
				break
			}
		}

		if !hasFields {
			return pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrInvalidDataRequest.Error(),
				Title:      "Invalid Data Request",
				Message:    errDatasourceFieldsEmpty.Error(),
				Err:        errDatasourceFieldsEmpty,
			}
		}
	}

	// Get datasource names from the request
	datasourceNames := r.GetDatasourceNames()
	if len(datasourceNames) == 0 {
		return pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrInvalidDataRequest.Error(),
			Title:      "Invalid Data Request",
			Message:    "mappedFields must contain at least one datasource",
		}
	}

	// Validate maximum datasources per job
	if len(datasourceNames) > MaxDatasourcesPerJob {
		return pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrInvalidDataRequest.Error(),
			Title:      "Invalid Data Request",
			Message:    fmt.Sprintf("Maximum %d datasources allowed per job, got %d", MaxDatasourcesPerJob, len(datasourceNames)),
		}
	}

	return nil
}

// GetDatasourceNames extracts and returns all datasource names from mappedFields.
// The returned slice is sorted alphabetically for consistent ordering.
func (r *Job) GetDatasourceNames() []string {
	if r.MappedFields == nil {
		return nil
	}

	names := make([]string, 0, len(r.MappedFields))
	for name := range r.MappedFields {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// ToMappedFieldsMap converts the mappedFields to a generic map[string]any format
// suitable for storage or serialization.
func (r *Job) ToMappedFieldsMap() map[string]any {
	if r.MappedFields == nil {
		return nil
	}

	result := make(map[string]any, len(r.MappedFields))
	for datasource, tables := range r.MappedFields {
		tablesMap := make(map[string]any, len(tables))
		for table, fields := range tables {
			tablesMap[table] = fields
		}

		result[datasource] = tablesMap
	}

	return result
}

// SetFailedStatus updates the job status to FAILED, sets the CompletedAt timestamp,
// and records the failure message in the Metadata.
func (r *Job) SetFailedStatus(failedMsg string) {
	r.Status = JobStatusFailed

	// Set CompletedAt to current time
	completedAt := time.Now().UTC()
	r.CompletedAt = &completedAt

	// Record failure message in Metadata
	if r.Metadata == nil {
		r.Metadata = make(map[string]any)
	}

	r.Metadata["error"] = failedMsg
}

// FetcherRequest represents the POST /v1/fetcher request body.
type FetcherRequest struct {
	DataRequest DataRequest    `json:"dataRequest" validate:"required" doc:"Fields and filters to extract from configured data sources." example:"{\"mappedFields\":{\"ledger\":{\"transactions\":[\"id\",\"amount\"]}}}"`
	Metadata    map[string]any `json:"metadata,omitempty" doc:"Caller-defined metadata stored with the job; source identifies the owning product." example:"{\"source\":\"payments\",\"correlationId\":\"settlement-2026-07-13\"}"`
}

// DataRequest represents the data extraction request structure.
type DataRequest struct {
	MappedFields map[string]map[string][]string `json:"mappedFields" validate:"required" doc:"Fields to extract, grouped by data source and table." example:"{\"ledger\":{\"transactions\":[\"id\",\"amount\"]}}"`
	Filters      NestedFilters                  `json:"filters,omitempty" doc:"Optional filter conditions grouped by data source, table, and field." example:"{\"ledger\":{\"transactions\":{\"status\":{\"eq\":[\"approved\"]}}}}"`
}

// ComputeRequestHash generates a SHA-256 hash of the request for idempotency checks.
// The hash is computed from a deterministic JSON representation of both DataRequest and Metadata.
// Including metadata ensures that requests with the same data but different metadata
// (e.g., different correlation IDs or timestamps) are treated as distinct requests.
func (r *FetcherRequest) ComputeRequestHash() (string, error) {
	// Create a composite structure for hashing that includes both DataRequest and Metadata
	hashInput := struct {
		DataRequest DataRequest    `json:"dataRequest"`
		Metadata    map[string]any `json:"metadata,omitempty"`
	}{
		DataRequest: r.DataRequest,
		Metadata:    r.Metadata,
	}

	data, err := json.Marshal(hashInput)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:]), nil
}

// FetcherResponse represents the POST /v1/fetcher response body.
type FetcherResponse struct {
	JobID     uuid.UUID `json:"jobId" doc:"Unique identifier of the extraction job." example:"01980a89-21f0-7d7e-a109-564b5c6f53ac"`
	Status    string    `json:"status" doc:"Current processing status of the job." example:"pending"`
	CreatedAt time.Time `json:"createdAt" doc:"UTC timestamp when the job was created." example:"2026-07-13T12:00:00Z"`
	Message   string    `json:"message" doc:"Outcome of the create request." example:"Job created and queued for processing"`
}

// JobResponse represents the GET /v1/fetcher/:id response body.
type JobResponse struct {
	ID           uuid.UUID                      `json:"id" doc:"Unique identifier of the extraction job." example:"01980a89-21f0-7d7e-a109-564b5c6f53ac"`
	Metadata     map[string]any                 `json:"metadata,omitempty" doc:"Caller-defined public metadata stored with the job." example:"{\"source\":\"payments\",\"correlationId\":\"settlement-2026-07-13\"}"`
	MappedFields map[string]map[string][]string `json:"mappedFields" doc:"Fields requested for extraction, grouped by data source and table." example:"{\"ledger\":{\"transactions\":[\"id\",\"amount\"]}}"`
	Filters      NestedFilters                  `json:"filters,omitempty" doc:"Filter conditions applied to the extraction." example:"{\"ledger\":{\"transactions\":{\"status\":{\"eq\":[\"approved\"]}}}}"`
	Status       string                         `json:"status" doc:"Current processing status of the job." example:"completed"`
	ResultPath   string                         `json:"resultPath,omitempty" doc:"Storage path of the generated extraction result." example:"s3://fetcher-results/jobs/01980a89-21f0-7d7e-a109-564b5c6f53ac.json"`
	ResultHmac   string                         `json:"resultHmac,omitempty" doc:"HMAC digest used to verify the result integrity." example:"9dc118e1d4efdcf0f45bb646c38c2a02e9c20b0fc2c75c1a9be0fae200bd9f93"`
	RequestHash  string                         `json:"requestHash,omitempty" doc:"SHA-256 digest used to detect idempotent duplicate requests." example:"d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"`
	CreatedAt    time.Time                      `json:"createdAt" doc:"UTC timestamp when the job was created." example:"2026-07-13T12:00:00Z"`
	CompletedAt  *time.Time                     `json:"completedAt,omitempty" doc:"UTC timestamp when the job reached a terminal state." example:"2026-07-13T12:05:00Z"`
}

// NewJobResponseFrom creates a JobResponse from a Job entity.
func NewJobResponseFrom(j *Job) *JobResponse {
	if j == nil {
		return nil
	}

	return &JobResponse{
		ID:           j.ID,
		Metadata:     publicJobMetadata(j.Metadata),
		MappedFields: j.MappedFields,
		Filters:      j.Filters,
		Status:       string(j.Status),
		ResultPath:   j.ResultPath,
		ResultHmac:   j.ResultHMAC,
		RequestHash:  j.RequestHash,
		CreatedAt:    j.CreatedAt,
		CompletedAt:  j.CompletedAt,
	}
}

func publicJobMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}

	public := make(map[string]any, len(metadata))
	for key, value := range metadata {
		switch key {
		case "terminalEventPending", "terminalEventStatus", "terminalEventPayload":
			continue
		default:
			public[key] = value
		}
	}

	if len(public) == 0 {
		return nil
	}

	return public
}
