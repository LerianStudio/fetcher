package job

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newValidJob() *Job {
	return &Job{
		OrganizationID: uuid.New(),
		ConnectionID:   uuid.New(),
		MappedFields:   map[string]any{"k": "v"},
		Filters:        map[string]any{"f": "v"},
		Metadata:       map[string]any{"m": "v"},
		Status:         JobStatusProcessing,
		ResultPath:     "  /tmp/result  ",
	}
}

func cloneJob(src *Job) *Job {
	if src == nil {
		return nil
	}
	cp := *src
	return &cp
}

func TestJobStatusIsValid(t *testing.T) {
	if !JobStatusCompleted.IsValid() {
		t.Fatalf("expected JobStatusCompleted to be valid")
	}
	if JobStatus("unknown").IsValid() {
		t.Fatalf("expected unknown status to be invalid")
	}
}

func TestJobValidateForCreate(t *testing.T) {
	t.Run("success defaults status and trims path", func(t *testing.T) {
		job := newValidJob()
		job.Status = ""
		if err := job.ValidateForCreate(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if job.Status != JobStatusPending {
			t.Fatalf("expected default status pending, got %s", job.Status)
		}
		if job.ResultPath != "/tmp/result" {
			t.Fatalf("expected trimmed result path, got %q", job.ResultPath)
		}
	})

	t.Run("accepts mixed-case and finished alias", func(t *testing.T) {
		job := newValidJob()
		job.Status = "Processing"
		if err := job.ValidateForCreate(); err != nil {
			t.Fatalf("expected no error: %v", err)
		}
		if job.Status != JobStatusProcessing {
			t.Fatalf("expected processing normalized, got %s", job.Status)
		}

		job = newValidJob()
		job.Status = "Finished"
		if err := job.ValidateForCreate(); err != nil {
			t.Fatalf("expected no error: %v", err)
		}
		if job.Status != JobStatusCompleted {
			t.Fatalf("expected finished normalized to completed, got %s", job.Status)
		}
	})

	tests := []struct {
		name string
		job  *Job
		err  string
	}{
		{"nil job", nil, "job entity is required"},
		{"missing org", func() *Job { j := newValidJob(); j.OrganizationID = uuid.Nil; return j }(), "organization ID is required"},
		{"missing connection", func() *Job { j := newValidJob(); j.ConnectionID = uuid.Nil; return j }(), "connection ID is required"},
		{"missing mappedFields", func() *Job { j := newValidJob(); j.MappedFields = nil; return j }(), "mappedFields is required"},
		{"invalid status", func() *Job { j := newValidJob(); j.Status = JobStatus("bad"); return j }(), "invalid job status"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.job == nil {
				var nilJob *Job
				err = nilJob.ValidateForCreate()
			} else {
				err = cloneJob(tt.job).ValidateForCreate()
			}
			if err == nil || err.Error() != tt.err {
				t.Fatalf("expected error %q, got %v", tt.err, err)
			}
		})
	}
}

func TestJobValidateForUpdate(t *testing.T) {
	job := newValidJob()
	job.ID = uuid.New()
	if err := job.ValidateForUpdate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jobNoID := newValidJob()
	if err := jobNoID.ValidateForUpdate(); err == nil || err.Error() != "job ID is required" {
		t.Fatalf("expected job ID error, got %v", err)
	}
}

func TestJobMongoDBModelFromEntity(t *testing.T) {
	t.Run("nil entity", func(t *testing.T) {
		model := &JobMongoDBModel{}
		if err := model.FromEntity(nil); err == nil {
			t.Fatalf("expected error for nil entity")
		}
	})

	t.Run("generates defaults and completes terminal status", func(t *testing.T) {
		originalGen := generateJobUUID
		defer func() { generateJobUUID = originalGen }()

		expectedID := uuid.New()
		generateJobUUID = func() (uuid.UUID, error) {
			return expectedID, nil
		}

		job := newValidJob()
		job.ID = uuid.Nil
		job.Status = JobStatusCompleted
		job.CreatedAt = time.Time{}
		model := &JobMongoDBModel{}
		if err := model.FromEntity(job); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.ID != expectedID {
			t.Fatalf("expected generated ID %v, got %v", expectedID, model.ID)
		}
		if job.CreatedAt.IsZero() {
			t.Fatalf("expected created_at to be set")
		}
		if model.CompletedAt == nil {
			t.Fatalf("expected completed_at to be set for completed status")
		}
	})

	t.Run("propagates uuid error", func(t *testing.T) {
		originalGen := generateJobUUID
		defer func() { generateJobUUID = originalGen }()

		generateJobUUID = func() (uuid.UUID, error) {
			return uuid.Nil, errors.New("uuid failure")
		}

		job := newValidJob()
		job.ID = uuid.Nil
		model := &JobMongoDBModel{}
		if err := model.FromEntity(job); err == nil || err.Error() != "uuid failure" {
			t.Fatalf("expected uuid failure, got %v", err)
		}
	})

	t.Run("preserves provided values even when non terminal", func(t *testing.T) {
		now := time.Now().Add(-time.Hour)
		completed := now.Add(time.Minute)
		job := newValidJob()
		job.ID = uuid.New()
		job.CreatedAt = now
		job.CompletedAt = &completed
		job.Status = JobStatusProcessing

		model := &JobMongoDBModel{}
		if err := model.FromEntity(job); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.CreatedAt != now {
			t.Fatalf("expected created_at preserved")
		}
		if model.CompletedAt == nil || *model.CompletedAt != completed {
			t.Fatalf("expected completed_at to be preserved")
		}
		if model.ID != job.ID {
			t.Fatalf("expected IDs to match")
		}
	})
}

func TestJobMongoDBModelToEntity(t *testing.T) {
	t.Run("nil model", func(t *testing.T) {
		var model *JobMongoDBModel
		if model.ToEntity() != nil {
			t.Fatalf("expected nil entity for nil model")
		}
	})

	completed := time.Now()
	model := &JobMongoDBModel{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Metadata:       map[string]any{"m": "v"},
		MappedFields:   map[string]any{"k": "v"},
		Filters:        map[string]any{"f": "v"},
		Status:         JobStatusCompleted,
		ResultPath:     "/res",
		CreatedAt:      completed.Add(-time.Hour),
		CompletedAt:    &completed,
	}

	entity := model.ToEntity()
	if entity == nil {
		t.Fatalf("expected entity")
	}
	if entity.ID != model.ID || entity.OrganizationID != model.OrganizationID {
		t.Fatalf("expected IDs to match")
	}
	if entity.Status != model.Status || entity.ResultPath != model.ResultPath {
		t.Fatalf("expected status and result path to match")
	}
	if entity.CompletedAt == nil || *entity.CompletedAt != *model.CompletedAt {
		t.Fatalf("expected completed_at to match")
	}
}
