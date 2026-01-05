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

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/google/uuid"
)

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
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Metadata       map[string]any
	MappedFields   map[string]map[string][]string
	Filters        []Filter
	Status         JobStatus
	ResultPath     string
	RequestHash    string
	CreatedAt      time.Time
	CompletedAt    *time.Time
}

func NewJob(
	organizationID uuid.UUID,
	metadata map[string]any,
	mappedFields map[string]map[string][]string,
	filters []Filter,
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
		ID:             id,
		OrganizationID: organizationID,
		Metadata:       metadata,
		MappedFields:   mappedFields,
		Filters:        filters,
		Status:         status,
		ResultPath:     resultPath,
		RequestHash:    requestHash,
		CreatedAt:      createdAt,
		CompletedAt:    completedAt,
	}, nil
}

type Filter struct {
	Field    string `json:"field" validate:"required"`
	Operator string `json:"operator" validate:"required"`
	Value    []any  `json:"value" validate:"required"`
}

// ParsedFilterField represents the components of a qualified filter field.
// Supports formats:
// - configName.tableName.fieldName (3 parts)
// - configName.schema.tableName.fieldName (4 parts)
type ParsedFilterField struct {
	ConfigName string
	TableName  string
	FieldName  string
}

// ParseFilterField parses a qualified filter field string into its components.
// Valid formats:
// - "configName.tableName.fieldName" → ConfigName="configName", TableName="tableName", FieldName="fieldName"
// - "configName.schema.tableName.fieldName" → ConfigName="configName", TableName="schema.tableName", FieldName="fieldName"
func ParseFilterField(field string) (*ParsedFilterField, error) {
	if field == "" {
		return nil, errors.New("filter field cannot be empty")
	}

	parts := strings.Split(field, ".")

	switch len(parts) {
	case 3:
		// configName.tableName.fieldName
		if parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return nil, fmt.Errorf("invalid filter field format: empty component in '%s'", field)
		}

		return &ParsedFilterField{
			ConfigName: parts[0],
			TableName:  parts[1],
			FieldName:  parts[2],
		}, nil
	case 4:
		// configName.schema.tableName.fieldName
		if parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
			return nil, fmt.Errorf("invalid filter field format: empty component in '%s'", field)
		}

		return &ParsedFilterField{
			ConfigName: parts[0],
			TableName:  parts[1] + "." + parts[2], // Reconstruct schema.tableName
			FieldName:  parts[3],
		}, nil
	default:
		return nil, fmt.Errorf("invalid filter field format: expected 'configName.tableName.fieldName' or 'configName.schema.tableName.fieldName', got '%s' with %d parts", field, len(parts))
	}
}

// ValidateFilterReferences validates that all filter field references exist in mappedFields.
// Only validates that configName exists - tableName resolution is handled by DataSource adapters
// which can apply default schema fallback logic (e.g., "transactions" → "public.transactions").
func ValidateFilterReferences(filters []Filter, mappedFields map[string]map[string][]string) error {
	if len(filters) == 0 {
		return nil
	}

	var validationErrors []string

	for i, f := range filters {
		parsed, err := ParseFilterField(f.Field)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("filter[%d]: %v", i, err))
			continue
		}

		// Check if configName exists in mappedFields
		// NOTE: We do NOT validate tableName here - the DataSource adapter will handle
		// schema resolution with fallback logic (e.g., trying "public.table" if "table" not found)
		if _, exists := mappedFields[parsed.ConfigName]; !exists {
			validationErrors = append(validationErrors,
				fmt.Sprintf("filter[%d]: datasource '%s' not found in mappedFields", i, parsed.ConfigName))
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
//
// swagger:model FetcherRequest
//
// @Description FetcherRequest represents the request body for creating a new data extraction job.
type FetcherRequest struct {
	DataRequest DataRequest    `json:"dataRequest" validate:"required"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// DataRequest represents the data extraction request structure.
//
// swagger:model DataRequest
//
// @Description DataRequest encapsulates field mappings and optional filters for data extraction.
type DataRequest struct {
	MappedFields map[string]map[string][]string `json:"mappedFields" validate:"required"`
	Filters      []FilterRequest                `json:"filters,omitempty"`
}

// FilterRequest represents a single filter condition.
//
// swagger:model FilterRequest
//
// @Description FilterRequest defines a filter condition with field, operator, and value(s).
type FilterRequest struct {
	Field    string `json:"field" validate:"required"`
	Operator string `json:"operator" validate:"required"`
	Value    []any  `json:"value" validate:"required"`
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
//
// swagger:model FetcherResponse
//
// @Description FetcherResponse represents the response after successfully creating a data extraction job.
type FetcherResponse struct {
	JobID     uuid.UUID `json:"jobId"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	Message   string    `json:"message"`
}

// JobResponse represents the GET /v1/fetcher/:id response body.
//
// swagger:model JobResponse
//
// @Description JobResponse represents the complete information about a data extraction job.
type JobResponse struct {
	ID             uuid.UUID                      `json:"id"`
	OrganizationID uuid.UUID                      `json:"organizationId"`
	Metadata       map[string]any                 `json:"metadata,omitempty"`
	MappedFields   map[string]map[string][]string `json:"mappedFields"`
	Filters        []Filter                       `json:"filters,omitempty"`
	Status         string                         `json:"status"`
	ResultPath     string                         `json:"resultPath,omitempty"`
	RequestHash    string                         `json:"requestHash,omitempty"`
	CreatedAt      time.Time                      `json:"createdAt"`
	CompletedAt    *time.Time                     `json:"completedAt,omitempty"`
}

// NewJobResponseFrom creates a JobResponse from a Job entity.
func NewJobResponseFrom(job *Job) *JobResponse {
	if job == nil {
		return nil
	}

	return &JobResponse{
		ID:             job.ID,
		OrganizationID: job.OrganizationID,
		Metadata:       job.Metadata,
		MappedFields:   job.MappedFields,
		Filters:        job.Filters,
		Status:         string(job.Status),
		ResultPath:     job.ResultPath,
		RequestHash:    job.RequestHash,
		CreatedAt:      job.CreatedAt,
		CompletedAt:    job.CompletedAt,
	}
}
