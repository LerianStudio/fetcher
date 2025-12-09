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

// ToFiltersMap converts the filters to a generic map[string]any format
// suitable for storage or serialization.
// The structure follows: { "datasource.table.field": { "operator": values } }
func (r *Job) ToFiltersMap() map[string]any {
	if len(r.Filters) == 0 {
		return nil
	}

	result := make(map[string]any, len(r.Filters))
	for _, filter := range r.Filters {
		filterCondition := map[string]any{
			filter.Operator: filter.Value,
		}
		result[filter.Field] = filterCondition
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
// The hash is computed from a deterministic JSON representation of the DataRequest.
func (r *FetcherRequest) ComputeRequestHash() (string, error) {
	data, err := json.Marshal(r.DataRequest)
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
