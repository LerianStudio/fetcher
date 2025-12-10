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

	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	libMongo "github.com/LerianStudio/lib-commons/v2/commons/mongo"

	"github.com/google/uuid"
	"github.com/tryvium-travels/memongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
)

var (
	jobTestMongoServer *memongo.Server
	jobTestMongoConn   *libMongo.MongoConnection
)

const jobTestDatabaseName = "fetcher_job_test"

func TestMain(m *testing.M) {
	server, err := memongo.Start("6.0.6")
	if err != nil {
		log.Fatalf("failed to start memongo: %v", err)
	}
	jobTestMongoServer = server
	jobTestMongoConn = &libMongo.MongoConnection{
		ConnectionStringSource: server.URI(),
		Database:               jobTestDatabaseName,
		Logger:                 &libLog.GoLogger{Level: libLog.ErrorLevel},
		MaxPoolSize:            5,
	}

	code := m.Run()

	server.Stop()
	os.Exit(code)
}

func newJobRepository(t *testing.T) *JobMongoDBRepository {
	t.Helper()
	clearJobsCollection(t)
	repo, err := NewJobMongoDBRepository(jobTestMongoConn)
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
	client, err := jobTestMongoConn.GetDB(context.Background())
	if err != nil {
		t.Fatalf("failed to get db: %v", err)
	}
	coll := client.Database(strings.ToLower(jobTestMongoConn.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))
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
		ID:             id,
		OrganizationID: uuid.New(),
		Metadata: map[string]any{
			"source": "unit-test",
		},
		MappedFields: map[string]map[string][]string{
			"datasource1": {
				"table1": {"field1", "field2"},
			},
		},
		Filters:     []model.Filter{},
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
	original := setSpanAttributesFromStruct
	setSpanAttributesFromStruct = func(span *trace.Span, key string, valueStruct any) error {
		return retErr
	}
	t.Cleanup(func() {
		setSpanAttributesFromStruct = original
	})
}

type fakeJobMongoConnection struct {
	err error
}

func (f *fakeJobMongoConnection) GetDB(ctx context.Context) (*mongo.Client, error) {
	return nil, f.err
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
		repo := &JobMongoDBRepository{
			connection: &fakeJobMongoConnection{err: errors.New("db down")},
			Database:   jobTestDatabaseName,
		}
		if _, err := repo.Create(context.Background(), jobFixture()); err == nil || err.Error() != "db down" {
			t.Fatalf("expected db error, got %v", err)
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

	t.Run("validation error", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())
		created.OrganizationID = uuid.Nil
		if _, err := repo.Update(context.Background(), created); err == nil {
			t.Fatalf("expected validation error")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newJobRepository(t)
		missing := jobFixture()
		missing.ID = uuid.New()
		missing.OrganizationID = uuid.New()
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
		repo := &JobMongoDBRepository{
			connection: &fakeJobMongoConnection{err: errors.New("db down")},
			Database:   jobTestDatabaseName,
		}
		job := jobFixture()
		job.ID = uuid.New()
		if _, err := repo.Update(context.Background(), job); err == nil || err.Error() != "db down" {
			t.Fatalf("expected db error, got %v", err)
		}
	})
}

func TestJobMongoDBRepository_FindByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newJobRepository(t)
		created := createJob(t, repo, jobFixture())
		found, err := repo.FindByID(context.Background(), created.ID, created.OrganizationID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected IDs to match")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newJobRepository(t)
		job, err := repo.FindByID(context.Background(), uuid.New(), uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if job != nil {
			t.Fatalf("expected nil job for not found, got %+v", job)
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo := &JobMongoDBRepository{
			connection: &fakeJobMongoConnection{err: errors.New("db down")},
			Database:   jobTestDatabaseName,
		}
		if _, err := repo.FindByID(context.Background(), uuid.New(), uuid.New()); err == nil || err.Error() != "db down" {
			t.Fatalf("expected db error, got %v", err)
		}
	})
}

func TestJobMongoDBRepository_List(t *testing.T) {
	t.Run("applies filters and pagination", func(t *testing.T) {
		repo := newJobRepository(t)
		orgID := uuid.New()
		otherOrg := uuid.New()

		older := jobFixture()
		older.OrganizationID = orgID
		older.Status = model.JobStatusPending
		older.CreatedAt = time.Now().Add(-2 * time.Hour)
		createJob(t, repo, older)

		newer := jobFixture()
		newer.OrganizationID = orgID
		newer.Status = model.JobStatusProcessing
		createJob(t, repo, newer)

		other := jobFixture()
		other.OrganizationID = otherOrg
		createJob(t, repo, other)

		since := time.Now().Add(-90 * time.Minute)
		filters := &ListFilter{
			OrganizationID: orgID,
			Statuses:       []model.JobStatus{model.JobStatusProcessing},
			CreatedFrom:    &since,
			Limit:          1,
			Page:           1,
			SortOrder:      constant.Desc,
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
		if _, err := repo.List(context.Background(), &ListFilter{OrganizationID: uuid.New(), Limit: -1}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := repo.List(context.Background(), &ListFilter{OrganizationID: uuid.New(), Limit: 999}); err != nil {
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
		if _, err := repo.List(context.Background(), &ListFilter{OrganizationID: uuid.New()}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo := &JobMongoDBRepository{
			connection: &fakeJobMongoConnection{err: errors.New("db down")},
			Database:   jobTestDatabaseName,
		}
		if _, err := repo.List(context.Background(), &ListFilter{OrganizationID: uuid.New()}); err == nil || err.Error() != "db down" {
			t.Fatalf("expected db error, got %v", err)
		}
	})

	t.Run("find failure surfaces", func(t *testing.T) {
		repo := newJobRepository(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := repo.List(ctx, &ListFilter{OrganizationID: uuid.New()}); err == nil {
			t.Fatalf("expected error")
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

func TestEnsureIndexesHandlesConflicts(t *testing.T) {
	repo := newJobRepository(t)
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("expected idempotent ensure indexes, got %v", err)
	}
}

func TestEnsureIndexesDatabaseError(t *testing.T) {
	repo := &JobMongoDBRepository{
		connection: &fakeJobMongoConnection{err: errors.New("db down")},
		Database:   jobTestDatabaseName,
	}
	if err := repo.EnsureIndexes(context.Background()); err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestDropIndexesDatabaseError(t *testing.T) {
	repo := &JobMongoDBRepository{
		connection: &fakeJobMongoConnection{err: errors.New("db down")},
		Database:   jobTestDatabaseName,
	}
	if err := repo.DropIndexes(context.Background()); err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestRepositoryConstructorValidatesDB(t *testing.T) {
	repo, err := NewJobMongoDBRepository(jobTestMongoConn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo == nil {
		t.Fatalf("expected repository instance")
	}
}

func TestListCompletedRangeFilter(t *testing.T) {
	repo := newJobRepository(t)
	org := uuid.New()

	completedAt := time.Now().Add(-time.Hour)
	completedJob := jobFixture()
	completedJob.OrganizationID = org
	completedJob.Status = model.JobStatusCompleted
	completedJob.CompletedAt = &completedAt
	createJob(t, repo, completedJob)

	from := completedAt.Add(-time.Minute)
	to := completedAt.Add(time.Minute)
	filters := &ListFilter{OrganizationID: org, CompletedFrom: &from, CompletedTo: &to}
	list, err := repo.List(context.Background(), filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].Status != model.JobStatusCompleted {
		t.Fatalf("expected completed job")
	}
}

func TestListUsesDescendingByDefault(t *testing.T) {
	repo := newJobRepository(t)
	org := uuid.New()

	first := jobFixture()
	first.OrganizationID = org
	first.CreatedAt = time.Now().Add(-time.Hour)
	createJob(t, repo, first)

	second := jobFixture()
	second.OrganizationID = org
	createJob(t, repo, second)

	list, err := repo.List(context.Background(), &ListFilter{OrganizationID: org, Limit: 2})
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

func TestListPartialFilters(t *testing.T) {
	repo := newJobRepository(t)
	org := uuid.New()

	job := jobFixture()
	job.OrganizationID = org
	job.Status = model.JobStatusFailed
	createJob(t, repo, job)

	filters := &ListFilter{OrganizationID: org, Status: model.JobStatusFailed}
	list, err := repo.List(context.Background(), filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].Status != model.JobStatusFailed {
		t.Fatalf("expected failed job")
	}
}

func TestCreateSetsDefaults(t *testing.T) {
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

func TestUpdateWithoutCompletedAtWhenFailed(t *testing.T) {
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

func TestListWithPaginationSecondPageEmpty(t *testing.T) {
	repo := newJobRepository(t)
	org := uuid.New()
	for i := 0; i < 2; i++ {
		job := jobFixture()
		job.OrganizationID = org
		job.ResultPath = fmt.Sprintf("/res-%d", i)
		createJob(t, repo, job)
	}
	list, err := repo.List(context.Background(), &ListFilter{OrganizationID: org, Limit: 2, Page: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty page")
	}
}
