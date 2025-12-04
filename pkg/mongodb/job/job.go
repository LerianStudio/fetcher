package job

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// JobStatus enumerates supported job statuses.
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

var generateJobUUID = uuid.NewV7

// normalizeJobStatus trims, lowercases, and maps aliases (e.g., finished -> completed).
func normalizeJobStatus(status JobStatus) (JobStatus, error) {
	normalized := JobStatus(strings.ToLower(strings.TrimSpace(string(status))))

	if normalized == "" {
		return JobStatusPending, nil
	}

	if normalized == "finished" {
		normalized = JobStatusCompleted
	}

	if !normalized.IsValid() {
		return "", errors.New("invalid job status")
	}

	return normalized, nil
}

// IsValid reports whether the status is recognized by the enum.
func (js JobStatus) IsValid() bool {
	_, ok := validJobStatuses[js]
	return ok
}

// Job represents the API payload stored in the jobs collection.
type Job struct {
	ID             uuid.UUID      `json:"id"`
	OrganizationID uuid.UUID      `json:"organizationId"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	MappedFields   map[string]any `json:"mappedFields"`
	Filters        map[string]any `json:"filters,omitempty"`
	Status         JobStatus      `json:"status"`
	ResultPath     string         `json:"resultPath,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	CompletedAt    *time.Time     `json:"completedAt,omitempty"`
}

// ValidateForCreate ensures required fields are present for insertion.
func (job *Job) ValidateForCreate() error {
	if job == nil {
		return errors.New("job entity is required")
	}

	job.ResultPath = strings.TrimSpace(job.ResultPath)

	if job.OrganizationID == uuid.Nil {
		return errors.New("organization ID is required")
	}

	if job.MappedFields == nil {
		return errors.New("mappedFields is required")
	}

	normalizedStatus, err := normalizeJobStatus(job.Status)
	if err != nil {
		return err
	}
	job.Status = normalizedStatus

	return nil
}

// ValidateForUpdate ensures a job has the fields required for updates.
func (job *Job) ValidateForUpdate() error {
	if err := job.ValidateForCreate(); err != nil {
		return err
	}

	if job.ID == uuid.Nil {
		return errors.New("job ID is required")
	}

	return nil
}

// JobMongoDBModel represents how a job is stored in MongoDB.
type JobMongoDBModel struct {
	ID             uuid.UUID      `bson:"_id"`
	OrganizationID uuid.UUID      `bson:"organization_id"`
	ConnectionID   uuid.UUID      `bson:"connection_id"`
	Metadata       map[string]any `bson:"metadata,omitempty"`
	MappedFields   map[string]any `bson:"mapped_fields"`
	Filters        map[string]any `bson:"filters,omitempty"`
	Status         JobStatus      `bson:"status"`
	ResultPath     string         `bson:"result_path,omitempty"`
	CreatedAt      time.Time      `bson:"created_at"`
	CompletedAt    *time.Time     `bson:"completed_at"`
}

// ToEntity converts a MongoDB model into the API entity representation.
func (jm *JobMongoDBModel) ToEntity() *Job {
	if jm == nil {
		return nil
	}

	return &Job{
		ID:             jm.ID,
		OrganizationID: jm.OrganizationID,
		Metadata:       jm.Metadata,
		MappedFields:   jm.MappedFields,
		Filters:        jm.Filters,
		Status:         jm.Status,
		ResultPath:     jm.ResultPath,
		CreatedAt:      jm.CreatedAt,
		CompletedAt:    jm.CompletedAt,
	}
}

// FromEntity prepares the MongoDB model for persistence.
// Note: This method mutates the input entity by setting ID (if uuid.Nil) and timestamps if empty.
func (jm *JobMongoDBModel) FromEntity(job *Job) error {
	if job == nil {
		return errors.New("job entity is required")
	}

	normalizedStatus, err := normalizeJobStatus(job.Status)
	if err != nil {
		return err
	}
	job.Status = normalizedStatus

	id := job.ID
	if id == uuid.Nil {
		generated, err := generateJobUUID()
		if err != nil {
			return err
		}
		id = generated
		job.ID = generated
	}

	now := time.Now().UTC()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}

	if (job.Status == JobStatusCompleted || job.Status == JobStatusFailed) && job.CompletedAt == nil {
		job.CompletedAt = &now
	}

	jm.ID = id
	jm.OrganizationID = job.OrganizationID
	jm.Metadata = job.Metadata
	jm.MappedFields = job.MappedFields
	jm.Filters = job.Filters
	jm.Status = job.Status
	jm.ResultPath = job.ResultPath
	jm.CreatedAt = job.CreatedAt
	jm.CompletedAt = job.CompletedAt

	return nil
}
