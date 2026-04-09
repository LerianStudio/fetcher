package job

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/mongodb"

	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libMongo "github.com/LerianStudio/lib-commons/v4/commons/mongo"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tryvium-travels/memongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

var (
	jobTestMongoServer *memongo.Server
	jobTestMongoConn   *libMongo.Client
)

const jobTestDatabaseName = "fetcher_job_test"

func TestMain(m *testing.M) {
	server, err := memongo.Start("6.0.6")
	if err != nil {
		// memongo doesn't support all platforms (e.g., Fedora 42)
		// Skip tests gracefully instead of failing
		log.Printf("SKIP: memongo not available on this platform: %v", err)
		os.Exit(0)
	}
	jobTestMongoServer = server
	client, err := libMongo.NewClient(context.Background(), libMongo.Config{
		URI:         server.URI(),
		Database:    jobTestDatabaseName,
		Logger:      &libLog.GoLogger{Level: libLog.LevelError},
		MaxPoolSize: 5,
	})
	if err != nil {
		log.Fatalf("failed to create mongo client: %v", err)
	}
	jobTestMongoConn = client

	code := m.Run()

	server.Stop()
	os.Exit(code)
}

func newJobRepository(t *testing.T) *JobMongoDBRepository {
	t.Helper()
	clearJobsCollection(t)
	repo, err := NewJobMongoDBRepository(context.Background(), jobTestMongoConn, jobTestDatabaseName)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	return repo
}

func clearJobsCollection(t *testing.T) {
	t.Helper()
	if jobTestMongoConn == nil {
		t.Fatalf("mongo connection not initialized")
	}
	client, err := jobTestMongoConn.Client(context.Background())
	if err != nil {
		t.Fatalf("failed to get db: %v", err)
	}
	dbName, err := jobTestMongoConn.DatabaseName()
	if err != nil {
		t.Fatalf("failed to get db name: %v", err)
	}
	coll := client.Database(strings.ToLower(dbName)).Collection(strings.ToLower(constant.MongoCollectionJob))
	if err := coll.Drop(context.Background()); err != nil {
		var cmdErr mongo.CommandError
		if errors.As(err, &cmdErr) && cmdErr.Code == 26 {
			return
		}
		t.Fatalf("failed to drop collection: %v", err)
	}
}

func jobFixture() *model.Job {
	id, _ := uuid.NewV7()
	return &model.Job{
		ID: id,
		Metadata: map[string]any{
			"source": "unit-test",
		},
		MappedFields: map[string]map[string][]string{
			"datasource1": {
				"table1": {"field1", "field2"},
			},
		},
		Filters:     model.NestedFilters{},
		Status:      model.JobStatusPending,
		ResultPath:  "/tmp/result.csv",
		RequestHash: "dummyhashvalue1234567890abcdef1234567890abcdef1234567890abcdef12",
		CreatedAt:   time.Now(),
	}
}

func createJob(t *testing.T, repo *JobMongoDBRepository, job *model.Job) *model.Job {
	t.Helper()
	created, err := repo.Create(context.Background(), job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	return created
}

func stubJobSpanAttributes(t *testing.T, retErr error) {
	t.Helper()
	original := setSpanAttributesFromValue
	setSpanAttributesFromValue = func(span trace.Span, key string, value any) error {
		return retErr
	}
	t.Cleanup(func() {
		setSpanAttributesFromValue = original
	})
}

func TestJobMongoDBRepository_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newJobRepository(t)
		job := jobFixture()
		originalID := job.ID
		created, err := repo.Create(context.Background(), job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.ID != originalID {
			t.Fatalf("expected ID %s, got %s", originalID, created.ID)
		}
	})

	t.Run("nil mapped_fields inserts successfully", func(t *testing.T) {
		repo := newJobRepository(t)
		job := jobFixture()
		job.MappedFields = nil
		// Repository no longer validates - validation is at model/service layer
		// Repository should insert successfully even with nil MappedFields
		created, err := repo.Create(context.Background(), job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.MappedFields != nil {
			t.Fatalf("expected nil MappedFields, got %v", created.MappedFields)
		}
	})

	t.Run("nil payload", func(t *testing.T) {
		repo := newJobRepository(t)
		if _, err := repo.Create(context.Background(), nil); err == nil {
			t.Fatalf("expected error for nil payload")
		}
	})

	t.Run("span attribute error ignored", func(t *testing.T) {
		repo := newJobRepository(t)
		stubJobSpanAttributes(t, errors.New("span failure"))
		if _, err := repo.Create(context.Background(), jobFixture()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("preserves valid status", func(t *testing.T) {
		repo := newJobRepository(t)
		job := jobFixture()
		// Status must be a valid JobStatus constant
		job.Status = model.JobStatusCompleted
		created, err := repo.Create(context.Background(), job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.Status != model.JobStatusCompleted {
			t.Fatalf("expected status completed, got %s", created.Status)
		}
	})

	t.Run("database error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &JobMongoDBRepository{
			connection: mockConn,
			Database:   jobTestDatabaseName,
		}
		_, err := repo.Create(context.Background(), jobFixture())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "db down") {
			t.Fatalf("expected error containing 'db down', got %v", err)
		}
	})
}

func TestJobMongoDBRepository_Update(t *testing.T) {
	t.Run("success preserves completed_at for terminal status", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())
		created.Status = model.JobStatusCompleted
		// Repository does NOT auto-set CompletedAt - caller must set it
		now := time.Now().UTC()
		created.CompletedAt = &now
		updated, err := repo.Update(context.Background(), created)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.CompletedAt == nil {
			t.Fatalf("expected completed_at to be preserved")
		}
	})

	t.Run("preserves nil completed_at for non terminal status", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())
		created.Status = model.JobStatusProcessing
		// Explicitly set CompletedAt to nil (caller responsibility)
		created.CompletedAt = nil
		updated, err := repo.Update(context.Background(), created)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.CompletedAt != nil {
			t.Fatalf("expected completed_at to remain nil")
		}
	})

	t.Run("nil payload", func(t *testing.T) {
		repo := newJobRepository(t)
		if _, err := repo.Update(context.Background(), nil); err == nil {
			t.Fatalf("expected error for nil payload")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newJobRepository(t)
		missing := jobFixture()
		missing.ID = uuid.New()

		if _, err := repo.Update(context.Background(), missing); err == nil {
			t.Fatalf("expected not found error")
		}
	})

	t.Run("span attribute error ignored", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())
		stubJobSpanAttributes(t, errors.New("span failure"))
		created.Status = model.JobStatusPending
		if _, err := repo.Update(context.Background(), created); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &JobMongoDBRepository{
			connection: mockConn,
			Database:   jobTestDatabaseName,
		}
		job := jobFixture()
		job.ID = uuid.New()
		_, err := repo.Update(context.Background(), job)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "db down") {
			t.Fatalf("expected error containing 'db down', got %v", err)
		}
	})
}

func TestJobMongoDBRepository_FindByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())
		found, err := repo.FindByID(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected IDs to match")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newJobRepository(t)
		job, err := repo.FindByID(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if job != nil {
			t.Fatalf("expected nil job for not found, got %+v", job)
		}
	})

	t.Run("database error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &JobMongoDBRepository{
			connection: mockConn,
			Database:   jobTestDatabaseName,
		}
		_, err := repo.FindByID(context.Background(), uuid.New())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "db down") {
			t.Fatalf("expected error containing 'db down', got %v", err)
		}
	})
}

func TestJobMongoDBRepository_List(t *testing.T) {
	t.Run("applies filters and pagination", func(t *testing.T) {
		repo := newJobRepository(t)

		older := jobFixture()

		older.Status = model.JobStatusPending
		older.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
		createJob(t, repo, older)

		newer := jobFixture()

		newer.Status = model.JobStatusProcessing
		createJob(t, repo, newer)

		other := jobFixture()

		createJob(t, repo, other)

		since := time.Now().UTC().Add(-90 * time.Minute)
		filters := &ListFilter{
			Statuses:    []model.JobStatus{model.JobStatusProcessing},
			CreatedFrom: &since,
			Limit:       1,
			Page:        1,
			SortOrder:   constant.Desc,
		}

		list, err := repo.List(context.Background(), filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 || list[0].Status != model.JobStatusProcessing {
			t.Fatalf("expected only processing job")
		}
	})

	t.Run("limit boundaries and nil filters", func(t *testing.T) {
		repo := newJobRepository(t)
		createJob(t, repo, jobFixture())
		if _, err := repo.List(context.Background(), &ListFilter{Limit: -1}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := repo.List(context.Background(), &ListFilter{Limit: 999}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := repo.List(context.Background(), nil); err != nil {
			t.Fatalf("unexpected error listing with nil filters: %v", err)
		}
	})

	t.Run("span attribute error ignored", func(t *testing.T) {
		repo := newJobRepository(t)
		createJob(t, repo, jobFixture())
		stubJobSpanAttributes(t, errors.New("span failure"))
		if _, err := repo.List(context.Background(), &ListFilter{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &JobMongoDBRepository{
			connection: mockConn,
			Database:   jobTestDatabaseName,
		}
		_, err := repo.List(context.Background(), &ListFilter{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "db down") {
			t.Fatalf("expected error containing 'db down', got %v", err)
		}
	})

	t.Run("find failure surfaces", func(t *testing.T) {
		repo := newJobRepository(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := repo.List(ctx, &ListFilter{}); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestJobMongoDBRepository_UpdateStatus(t *testing.T) {
	t.Run("updates status to completed and sets completed_at", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())

		err := repo.UpdateStatus(context.Background(), created.ID, model.JobStatusCompleted, "/external-data/test.json", "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the job was updated
		found, err := repo.FindByID(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("failed to find job: %v", err)
		}
		if found.Status != model.JobStatusCompleted {
			t.Fatalf("expected status completed, got %s", found.Status)
		}
		if found.CompletedAt == nil {
			t.Fatalf("expected completed_at to be set")
		}
		if found.ResultPath != "/external-data/test.json" {
			t.Fatalf("expected result_path to be set, got %s", found.ResultPath)
		}
	})

	t.Run("updates status to failed and sets completed_at", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())

		err := repo.UpdateStatus(context.Background(), created.ID, model.JobStatusFailed, "", "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found, err := repo.FindByID(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("failed to find job: %v", err)
		}
		if found.Status != model.JobStatusFailed {
			t.Fatalf("expected status failed, got %s", found.Status)
		}
		if found.CompletedAt == nil {
			t.Fatalf("expected completed_at to be set for failed status")
		}
	})

	t.Run("updates status to processing and clears completed_at", func(t *testing.T) {
		repo := newJobRepository(t)
		job := jobFixture()
		job.Status = model.JobStatusCompleted
		now := time.Now()
		job.CompletedAt = &now
		created := createJob(t, repo, job)

		err := repo.UpdateStatus(context.Background(), created.ID, model.JobStatusProcessing, "", "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found, err := repo.FindByID(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("failed to find job: %v", err)
		}
		if found.Status != model.JobStatusProcessing {
			t.Fatalf("expected status processing, got %s", found.Status)
		}
		if found.CompletedAt != nil {
			t.Fatalf("expected completed_at to be cleared for non-terminal status")
		}
	})

	t.Run("updates metadata when provided", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())

		metadata := map[string]any{
			"error_message": "something went wrong",
			"retry_count":   3,
		}
		err := repo.UpdateStatus(context.Background(), created.ID, model.JobStatusFailed, "", "", metadata)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found, err := repo.FindByID(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("failed to find job: %v", err)
		}
		if found.Metadata["error_message"] != "something went wrong" {
			t.Fatalf("expected metadata to be updated, got %v", found.Metadata)
		}
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())

		err := repo.UpdateStatus(context.Background(), created.ID, "invalid-status", "", "", nil)
		if err == nil {
			t.Fatalf("expected error for invalid status")
		}
	})

	t.Run("returns error for non-existent job", func(t *testing.T) {
		repo := newJobRepository(t)

		err := repo.UpdateStatus(context.Background(), uuid.New(), model.JobStatusCompleted, "", "", nil)
		if err == nil {
			t.Fatalf("expected error for non-existent job")
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &JobMongoDBRepository{
			connection: mockConn,
			Database:   jobTestDatabaseName,
		}
		err := repo.UpdateStatus(context.Background(), uuid.New(), model.JobStatusCompleted, "", "", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "db down") {
			t.Fatalf("expected error containing 'db down', got %v", err)
		}
	})

	t.Run("updates result_hmac when provided", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())

		resultHMAC := "hmac-sha256:abc123def456"
		err := repo.UpdateStatus(context.Background(), created.ID, model.JobStatusCompleted, "/external-data/test.json", resultHMAC, nil)
		require.NoError(t, err, "unexpected error updating status")

		found, err := repo.FindByID(context.Background(), created.ID)
		require.NoError(t, err, "failed to find job")
		require.Equal(t, resultHMAC, found.ResultHMAC, "expected result_hmac to be set")
	})
}

func TestJobMongoDBRepository_FindByRequestHashWithinWindow(t *testing.T) {
	t.Run("finds job within time window", func(t *testing.T) {
		repo := newJobRepository(t)

		hash := "abc123def456"

		job := jobFixture()

		job.RequestHash = hash
		job.CreatedAt = time.Now().UTC()
		created := createJob(t, repo, job)

		// Look for job within 60 minute window
		found, err := repo.FindByRequestHashWithinWindow(context.Background(), hash, 60)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found == nil {
			t.Fatalf("expected to find job")
		}
		if found.ID != created.ID {
			t.Fatalf("expected same job ID")
		}
	})

	t.Run("returns nil for empty request hash", func(t *testing.T) {
		repo := newJobRepository(t)

		found, err := repo.FindByRequestHashWithinWindow(context.Background(), "", 60)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil for empty hash")
		}
	})

	t.Run("returns nil when job is outside time window", func(t *testing.T) {
		repo := newJobRepository(t)

		hash := "oldhashold123"

		job := jobFixture()

		job.RequestHash = hash
		// Force created_at to be older than the lookup window
		job.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
		createJob(t, repo, job)

		// Look for job within a 60 minute window – should not find this older job
		found, err := repo.FindByRequestHashWithinWindow(context.Background(), hash, 60)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil for job outside time window")
		}
	})

	t.Run("returns most recent job when multiple exist", func(t *testing.T) {
		repo := newJobRepository(t)

		hash := "duplicatehash123"

		// Create two jobs with same hash, different times
		older := jobFixture()

		older.RequestHash = hash
		older.ResultPath = "/older"
		older.CreatedAt = time.Now().UTC().Add(-30 * time.Minute)
		createJob(t, repo, older)

		newer := jobFixture()

		newer.RequestHash = hash
		newer.ResultPath = "/newer"
		newer.CreatedAt = time.Now().UTC()
		createJob(t, repo, newer)

		found, err := repo.FindByRequestHashWithinWindow(context.Background(), hash, 60)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found == nil {
			t.Fatalf("expected to find job")
		}
		if found.ResultPath != "/newer" {
			t.Fatalf("expected most recent job, got %s", found.ResultPath)
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &JobMongoDBRepository{
			connection: mockConn,
			Database:   jobTestDatabaseName,
		}
		_, err := repo.FindByRequestHashWithinWindow(context.Background(), "hash", 60)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "db down") {
			t.Fatalf("expected error containing 'db down', got %v", err)
		}
	})
}

func TestJobMongoDBRepository_FindActiveByRequestHash(t *testing.T) {
	t.Run("finds most recent active job", func(t *testing.T) {
		repo := newJobRepository(t)

		hash := "active-hash-123"

		older := jobFixture()

		older.RequestHash = hash
		older.Status = model.JobStatusPending
		older.CreatedAt = time.Now().UTC().Add(-2 * time.Minute)
		createJob(t, repo, older)

		newer := jobFixture()

		newer.RequestHash = hash
		newer.Status = model.JobStatusProcessing
		newer.CreatedAt = time.Now().UTC().Add(-1 * time.Minute)
		createdNewer := createJob(t, repo, newer)

		found, err := repo.FindActiveByRequestHash(context.Background(), hash)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, createdNewer.ID, found.ID)
	})

	t.Run("returns nil when only terminal jobs exist", func(t *testing.T) {
		repo := newJobRepository(t)

		hash := "terminal-hash-123"

		failedJob := jobFixture()

		failedJob.RequestHash = hash
		failedJob.Status = model.JobStatusFailed
		now := time.Now().UTC()
		failedJob.CompletedAt = &now
		createJob(t, repo, failedJob)

		found, err := repo.FindActiveByRequestHash(context.Background(), hash)
		require.NoError(t, err)
		require.Nil(t, found)
	})
}

func TestEnsureIndexes_DuplicateActiveJobsPreventsUniqueIndex(t *testing.T) {
	repo := newJobRepository(t)

	hash := "dup-active-hash-123"

	older := jobFixture()

	older.RequestHash = hash
	older.Status = model.JobStatusPending
	older.CreatedAt = time.Now().UTC().Add(-2 * time.Minute)
	createJob(t, repo, older)

	newer := jobFixture()

	newer.RequestHash = hash
	newer.Status = model.JobStatusProcessing
	newer.CreatedAt = time.Now().UTC().Add(-1 * time.Minute)
	createJob(t, repo, newer)

	// With the unique active hash index restored, EnsureIndexes returns an error
	// when duplicate active jobs exist (same org_id + request_hash).
	// Manual cleanup is required before the index can be created.
	err := repo.EnsureIndexes(context.Background())
	require.Error(t, err, "EnsureIndexes should fail when duplicate active jobs prevent unique index creation")
}

func TestJobMongoDBRepository_ExistsRunningByMappedFieldKey(t *testing.T) {
	t.Run("returns true when pending job exists with key", func(t *testing.T) {
		repo := newJobRepository(t)

		job := jobFixture()

		job.Status = model.JobStatusPending
		job.MappedFields = map[string]map[string][]string{
			"my-config": {
				"table1": {"field1"},
			},
		}
		createJob(t, repo, job)

		exists, err := repo.ExistsRunningByMappedFieldKey(context.Background(), "my-config")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatalf("expected to find running job with key")
		}
	})

	t.Run("returns true when processing job exists with key", func(t *testing.T) {
		repo := newJobRepository(t)

		job := jobFixture()

		job.Status = model.JobStatusProcessing
		job.MappedFields = map[string]map[string][]string{
			"processing-config": {
				"table1": {"field1"},
			},
		}
		createJob(t, repo, job)

		exists, err := repo.ExistsRunningByMappedFieldKey(context.Background(), "processing-config")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatalf("expected to find processing job with key")
		}
	})

	t.Run("returns false when job is completed", func(t *testing.T) {
		repo := newJobRepository(t)

		job := jobFixture()

		job.Status = model.JobStatusCompleted
		now := time.Now()
		job.CompletedAt = &now
		job.MappedFields = map[string]map[string][]string{
			"completed-config": {
				"table1": {"field1"},
			},
		}
		createJob(t, repo, job)

		exists, err := repo.ExistsRunningByMappedFieldKey(context.Background(), "completed-config")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Fatalf("expected false for completed job")
		}
	})

	t.Run("returns false when key does not exist", func(t *testing.T) {
		repo := newJobRepository(t)

		job := jobFixture()

		job.Status = model.JobStatusPending
		job.MappedFields = map[string]map[string][]string{
			"other-config": {
				"table1": {"field1"},
			},
		}
		createJob(t, repo, job)

		exists, err := repo.ExistsRunningByMappedFieldKey(context.Background(), "nonexistent-config")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Fatalf("expected false for non-matching key")
		}
	})

	t.Run("returns error for invalid key pattern", func(t *testing.T) {
		repo := newJobRepository(t)

		_, err := repo.ExistsRunningByMappedFieldKey(context.Background(), "invalid.key.with.dots")
		if err == nil {
			t.Fatalf("expected error for invalid key pattern")
		}
	})

	t.Run("accepts valid key patterns", func(t *testing.T) {
		repo := newJobRepository(t)

		// These should not error
		validPatterns := []string{"config", "my-config", "config_name", "config123", "Config-Name_123"}
		for _, pattern := range validPatterns {
			_, err := repo.ExistsRunningByMappedFieldKey(context.Background(), pattern)
			if err != nil {
				t.Fatalf("unexpected error for pattern %s: %v", pattern, err)
			}
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &JobMongoDBRepository{
			connection: mockConn,
			Database:   jobTestDatabaseName,
		}
		_, err := repo.ExistsRunningByMappedFieldKey(context.Background(), "config")
		if err == nil {
			t.Fatalf("expected db error")
		}
	})
}

func TestJobMongoDBRepository_isIndexConflictError(t *testing.T) {
	t.Run("returns true for index options conflict (code 85)", func(t *testing.T) {
		err := mongo.CommandError{Code: 85, Message: "Index options conflict"}
		if !mongodb.IsIndexConflictError(err) {
			t.Fatalf("expected true for code 85")
		}
	})

	t.Run("returns true for index key specs conflict (code 86)", func(t *testing.T) {
		err := mongo.CommandError{Code: 86, Message: "Index key specs conflict"}
		if !mongodb.IsIndexConflictError(err) {
			t.Fatalf("expected true for code 86")
		}
	})

	t.Run("returns false for other command errors", func(t *testing.T) {
		err := mongo.CommandError{Code: 11000, Message: "Duplicate key"}
		if mongodb.IsIndexConflictError(err) {
			t.Fatalf("expected false for code 11000")
		}
	})

	t.Run("returns false for non-command errors", func(t *testing.T) {
		err := errors.New("some other error")
		if mongodb.IsIndexConflictError(err) {
			t.Fatalf("expected false for non-command error")
		}
	})
}

func TestJobMongoDBRepository_EnsureIndexes(t *testing.T) {
	repo := newJobRepository(t)
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("unexpected error ensuring indexes: %v", err)
	}
}

func TestJobMongoDBRepository_DropIndexes(t *testing.T) {
	repo := newJobRepository(t)
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("unexpected error ensuring indexes: %v", err)
	}
	if err := repo.DropIndexes(context.Background()); err != nil {
		t.Fatalf("unexpected error dropping indexes: %v", err)
	}
}

func TestEnsureIndexes_HandlesConflicts(t *testing.T) {
	repo := newJobRepository(t)
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("expected idempotent ensure indexes, got %v", err)
	}
}

func TestEnsureIndexes_DatabaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mongodb.NewMockMongoClientProvider(ctrl)
	mockConn.EXPECT().
		Client(gomock.Any()).
		Return(nil, errors.New("db down"))

	repo := &JobMongoDBRepository{
		connection: mockConn,
		Database:   jobTestDatabaseName,
	}
	if err := repo.EnsureIndexes(context.Background()); err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestDropIndexes_DatabaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mongodb.NewMockMongoClientProvider(ctrl)
	mockConn.EXPECT().
		Client(gomock.Any()).
		Return(nil, errors.New("db down"))

	repo := &JobMongoDBRepository{
		connection: mockConn,
		Database:   jobTestDatabaseName,
	}
	if err := repo.DropIndexes(context.Background()); err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestNewJobMongoDBRepository_ValidatesDB(t *testing.T) {
	repo, err := NewJobMongoDBRepository(context.Background(), jobTestMongoConn, jobTestDatabaseName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo == nil {
		t.Fatalf("expected repository instance")
	}
}

func TestList_CompletedRangeFilter(t *testing.T) {
	repo := newJobRepository(t)

	completedAt := time.Now().UTC().Add(-time.Hour)
	completedJob := jobFixture()

	completedJob.Status = model.JobStatusCompleted
	completedJob.CompletedAt = &completedAt
	createJob(t, repo, completedJob)

	from := completedAt.Add(-time.Minute)
	to := completedAt.Add(time.Minute)
	filters := &ListFilter{CompletedFrom: &from, CompletedTo: &to}
	list, err := repo.List(context.Background(), filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].Status != model.JobStatusCompleted {
		t.Fatalf("expected completed job")
	}
}

func TestList_UsesDescendingByDefault(t *testing.T) {
	repo := newJobRepository(t)

	first := jobFixture()

	first.CreatedAt = time.Now().UTC().Add(-time.Hour)
	createJob(t, repo, first)

	second := jobFixture()

	createJob(t, repo, second)

	list, err := repo.List(context.Background(), &ListFilter{Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 jobs")
	}
	if list[0].CreatedAt.Before(list[1].CreatedAt) {
		t.Fatalf("expected desc ordering by created_at")
	}
}

func TestList_PartialFilters(t *testing.T) {
	repo := newJobRepository(t)

	job := jobFixture()

	job.Status = model.JobStatusFailed
	createJob(t, repo, job)

	filters := &ListFilter{Status: model.JobStatusFailed}
	list, err := repo.List(context.Background(), filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].Status != model.JobStatusFailed {
		t.Fatalf("expected failed job")
	}
}

func TestCreate_SetsDefaults(t *testing.T) {
	repo := newJobRepository(t)
	job := jobFixture()
	// Status must be valid - repository no longer sets defaults
	// Defaults should be set at service/model layer before repository
	job.Status = model.JobStatusPending
	job.ResultPath = "  /path  "
	created, err := repo.Create(context.Background(), job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Status != model.JobStatusPending {
		t.Fatalf("expected pending status, got %s", created.Status)
	}
	// Repository stores as-is without trimming
	if created.ResultPath != "  /path  " {
		t.Fatalf("expected ResultPath preserved as-is, got %s", created.ResultPath)
	}
}

func TestUpdate_WithoutCompletedAtWhenFailed(t *testing.T) {
	repo := newJobRepository(t)
	job := jobFixture()
	created := createJob(t, repo, job)
	created.Status = model.JobStatusFailed
	// Repository does NOT auto-set CompletedAt - caller must set it
	// This test verifies that explicitly setting CompletedAt works
	now := time.Now().UTC()
	created.CompletedAt = &now
	updated, err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.CompletedAt == nil {
		t.Fatalf("expected completed_at to be preserved when failed")
	}
}

func TestList_PaginationSecondPageEmpty(t *testing.T) {
	repo := newJobRepository(t)

	for i := 0; i < 2; i++ {
		job := jobFixture()

		job.ResultPath = fmt.Sprintf("/res-%d", i)
		createJob(t, repo, job)
	}
	list, err := repo.List(context.Background(), &ListFilter{Limit: 2, Page: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty page")
	}
}

func TestJobMongoDBRepository_getDatabase(t *testing.T) {
	t.Run("returns tenant database when tenant context is set", func(t *testing.T) {
		repo := newJobRepository(t)

		client, err := jobTestMongoConn.Client(context.Background())
		if err != nil {
			t.Fatalf("failed to get db client: %v", err)
		}

		tenantDB := client.Database("tenant_xyz789")
		ctx := tmcore.ContextWithMB(context.Background(), tenantDB)

		db, err := repo.getDatabase(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assert.Equal(t, "tenant_xyz789", db.Name())
	})

	t.Run("falls back to static connection when no tenant context", func(t *testing.T) {
		repo := newJobRepository(t)

		db, err := repo.getDatabase(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assert.Equal(t, strings.ToLower(jobTestDatabaseName), db.Name())
	})

	t.Run("returns error when no tenant and static connection fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := NewMockmongoDatabaseProvider(ctrl)
		mockConn.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &JobMongoDBRepository{
			connection: mockConn,
			Database:   jobTestDatabaseName,
		}

		_, err := repo.getDatabase(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db down")
	})
}
