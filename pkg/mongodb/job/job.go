package job

import (
	"errors"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	portsJob "github.com/LerianStudio/fetcher/pkg/ports/job"

	"github.com/google/uuid"
)

// JobMongoDBModel represents how a job is stored in MongoDB.
type JobMongoDBModel struct {
	ID             uuid.UUID                      `bson:"_id"`
	OrganizationID uuid.UUID                      `bson:"organization_id"`
	Metadata       map[string]any                 `bson:"metadata,omitempty"`
	MappedFields   map[string]map[string][]string `bson:"mapped_fields"`
	Filters        model.NestedFilters            `bson:"filters,omitempty"`
	Status         string                         `bson:"status"`
	ResultPath     string                         `bson:"result_path,omitempty"`
	ResultHMAC     string                         `bson:"result_hmac,omitempty"`
	RequestHash    string                         `bson:"request_hash,omitempty"`
	CreatedAt      time.Time                      `bson:"created_at"`
	CompletedAt    *time.Time                     `bson:"completed_at"`
}

// ToEntity converts a MongoDB model into the API entity representation.
func (jm *JobMongoDBModel) ToEntity() (*model.Job, error) {
	if jm == nil {
		return nil, nil
	}

	jobStatus, err := model.NewJobStatusFromString(jm.Status)
	if err != nil {
		return nil, err
	}

	return &model.Job{
		ID:             jm.ID,
		OrganizationID: jm.OrganizationID,
		Metadata:       jm.Metadata,
		MappedFields:   jm.MappedFields,
		Filters:        jm.Filters,
		Status:         jobStatus,
		ResultPath:     jm.ResultPath,
		ResultHMAC:     jm.ResultHMAC,
		RequestHash:    jm.RequestHash,
		CreatedAt:      jm.CreatedAt,
		CompletedAt:    jm.CompletedAt,
	}, nil
}

// FromEntity prepares the MongoDB model for persistence.
// Note: This method mutates the input entity by setting ID (if uuid.Nil) and timestamps if empty.
func (jm *JobMongoDBModel) FromEntity(job *model.Job) error {
	if job == nil {
		return errors.New("job entity is required")
	}

	jm.ID = job.ID
	jm.OrganizationID = job.OrganizationID
	jm.Metadata = job.Metadata
	jm.MappedFields = job.MappedFields
	jm.Filters = job.Filters
	jm.Status = string(job.Status)
	jm.ResultPath = job.ResultPath
	jm.ResultHMAC = job.ResultHMAC
	jm.RequestHash = job.RequestHash
	jm.CreatedAt = job.CreatedAt
	jm.CompletedAt = job.CompletedAt

	return nil
}

// ListFilter is an alias for the domain filter type defined in pkg/ports/job.
type ListFilter = portsJob.ListFilter
