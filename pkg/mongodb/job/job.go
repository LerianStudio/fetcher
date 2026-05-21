package job

import (
	"errors"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	portsJob "github.com/LerianStudio/fetcher/pkg/ports/job"

	"github.com/google/uuid"
)

// JobMongoDBModel represents how a job is stored in MongoDB.
//
// DedupActive is a derived field that mirrors whether Status is pending or
// processing. It exists so the unique dedup index can use an equality filter
// (compatible with both MongoDB and AWS DocumentDB) instead of a $in expression.
// It MUST be kept in sync with Status in every write path — see isDedupActive.
type JobMongoDBModel struct {
	ID           uuid.UUID                      `bson:"_id"`
	Metadata     map[string]any                 `bson:"metadata,omitempty"`
	MappedFields map[string]map[string][]string `bson:"mapped_fields"`
	Filters      model.NestedFilters            `bson:"filters,omitempty"`
	Status       string                         `bson:"status"`
	DedupActive  bool                           `bson:"dedup_active"`
	ResultPath   string                         `bson:"result_path,omitempty"`
	ResultHMAC   string                         `bson:"result_hmac,omitempty"`
	RequestHash  string                         `bson:"request_hash,omitempty"`
	CreatedAt    time.Time                      `bson:"created_at"`
	CompletedAt  *time.Time                     `bson:"completed_at"`
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
		ID:           jm.ID,
		Metadata:     jm.Metadata,
		MappedFields: jm.MappedFields,
		Filters:      jm.Filters,
		Status:       jobStatus,
		ResultPath:   jm.ResultPath,
		ResultHMAC:   jm.ResultHMAC,
		RequestHash:  jm.RequestHash,
		CreatedAt:    jm.CreatedAt,
		CompletedAt:  jm.CompletedAt,
	}, nil
}

// FromEntity prepares the MongoDB model for persistence.
// Note: This method mutates the input entity by setting ID (if uuid.Nil) and timestamps if empty.
func (jm *JobMongoDBModel) FromEntity(job *model.Job) error {
	if job == nil {
		return errors.New("job entity is required")
	}

	jm.ID = job.ID
	jm.Metadata = job.Metadata
	jm.MappedFields = job.MappedFields
	jm.Filters = job.Filters
	jm.Status = string(job.Status)
	jm.DedupActive = isDedupActive(job.Status)
	jm.ResultPath = job.ResultPath
	jm.ResultHMAC = job.ResultHMAC
	jm.RequestHash = job.RequestHash
	jm.CreatedAt = job.CreatedAt
	jm.CompletedAt = job.CompletedAt

	return nil
}

// ListFilter is an alias for the domain filter type defined in pkg/ports/job.
type ListFilter = portsJob.ListFilter

// isDedupActive reports whether a job in the given status participates in the
// active deduplication window enforced by the uniq_job_hash_active index.
// A job is active while pending or processing; on completed/failed it leaves
// the window and a new job with the same request_hash may be created.
//
// Every write path that touches Status MUST also write the result of
// isDedupActive(status) — otherwise the unique partial index silently stops
// enforcing the invariant.
func isDedupActive(status model.JobStatus) bool {
	return status == model.JobStatusPending || status == model.JobStatusProcessing
}
