package services

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
	jobPort "github.com/LerianStudio/fetcher/pkg/ports/job"
	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	streaming "github.com/LerianStudio/lib-streaming"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestTerminalEventRepairer_RepairOnce_WithPendingTerminalEvent_RetriesWithoutMessageRedelivery(t *testing.T) {
	t.Parallel()

	jobID := uuid.New()
	repo := &pendingTerminalRepo{
		jobs: []*model.Job{{
			ID:     jobID,
			Status: model.JobStatusCompleted,
			Metadata: map[string]any{
				terminalEventPendingMetadataKey: true,
				terminalEventStatusMetadataKey:  "completed",
				terminalEventPayloadMetadataKey: `{"status":"completed"}`,
			},
		}},
	}
	emitter := &countingEmitter{}
	uc := &UseCase{
		JobRepository:            repo,
		JobEventEmitter:          emitter,
		JobEventStreamingEnabled: true,
	}
	repairer := NewTerminalEventRepairer(uc, testLogger())

	err := repairer.RepairOnce(testContext())
	require.NoError(t, err)
	require.Equal(t, 1, emitter.count)
	require.Equal(t, jobID, repo.clearedID)
}

func TestTerminalEventRepairer_RepairOnce_WithTenantScope_InjectsTenantContext(t *testing.T) {
	t.Parallel()

	jobID := uuid.New()
	repo := &pendingTerminalRepo{
		requireTenantContext: true,
		jobs: []*model.Job{{
			ID:     jobID,
			Status: model.JobStatusCompleted,
			Metadata: map[string]any{
				terminalEventPendingMetadataKey: true,
				terminalEventStatusMetadataKey:  "completed",
				terminalEventPayloadMetadataKey: `{"status":"completed"}`,
			},
		}},
	}
	emitter := &countingEmitter{}
	uc := &UseCase{JobRepository: repo, JobEventEmitter: emitter, JobEventStreamingEnabled: true, JobEventStreamingRequireTenant: true}
	repairer := NewTerminalEventRepairerWithTenantScope(
		uc,
		testLogger(),
		"fetcher",
		&activeTenantRepo{tenants: []*tmclient.TenantSummary{{ID: "tenant-a"}}},
		&tenantMongoResolver{},
	)

	err := repairer.RepairOnce(testContext())
	require.NoError(t, err)
	require.Equal(t, 1, emitter.count)
	require.Equal(t, "tenant-a", emitter.tenantIDs[0])
	require.Equal(t, "tenant-a", repo.seenTenantIDs[0])
	require.True(t, repo.seenTenantDB[0])
}

func TestTerminalEventRepairer_RepairOnce_WithTenantScope_ContinuesAfterTenantFailure(t *testing.T) {
	t.Parallel()

	jobID := uuid.New()
	repo := &pendingTerminalRepo{
		requireTenantContext: true,
		jobs: []*model.Job{{
			ID:     jobID,
			Status: model.JobStatusCompleted,
			Metadata: map[string]any{
				terminalEventPendingMetadataKey: true,
				terminalEventStatusMetadataKey:  "completed",
				terminalEventPayloadMetadataKey: `{"status":"completed"}`,
			},
		}},
	}
	emitter := &countingEmitter{}
	uc := &UseCase{JobRepository: repo, JobEventEmitter: emitter, JobEventStreamingEnabled: true, JobEventStreamingRequireTenant: true}
	repairer := NewTerminalEventRepairerWithTenantScope(
		uc,
		testLogger(),
		"fetcher",
		&activeTenantRepo{tenants: []*tmclient.TenantSummary{{ID: "tenant-a"}, {ID: "tenant-b"}}},
		&tenantMongoResolver{failTenants: map[string]error{"tenant-a": errors.New("tenant database unavailable")}},
	)

	err := repairer.RepairOnce(testContext())
	require.Error(t, err)
	require.Contains(t, err.Error(), "tenant-a")
	require.Contains(t, err.Error(), "tenant database unavailable")
	require.Equal(t, 1, emitter.count)
	require.Equal(t, "tenant-b", emitter.tenantIDs[0])
	require.Equal(t, "tenant-b", repo.seenTenantIDs[0])
	require.True(t, repo.seenTenantDB[0])
}

func TestTerminalEventRepairer_RepairOnce_ErrorPaths(t *testing.T) {
	t.Parallel()

	listErr := errors.New("list failed")
	publishErr := errors.New("publish failed")
	jobID := uuid.New()

	tests := []struct {
		name          string
		repo          *pendingTerminalRepo
		emitter       *countingEmitter
		wantErr       string
		wantCleared   bool
		wantPublished int
	}{
		{
			name:    "list failure returns error",
			repo:    &pendingTerminalRepo{listErr: listErr},
			emitter: &countingEmitter{},
			wantErr: "list pending terminal events",
		},
		{
			name: "malformed metadata returns error",
			repo: &pendingTerminalRepo{jobs: []*model.Job{{ID: jobID, Status: model.JobStatusCompleted, Metadata: map[string]any{
				terminalEventPendingMetadataKey: true,
				terminalEventStatusMetadataKey:  "completed",
			}}}},
			emitter: &countingEmitter{},
			wantErr: "missing terminal event payload metadata",
		},
		{
			name: "non terminal status returns error",
			repo: &pendingTerminalRepo{jobs: []*model.Job{{ID: jobID, Status: model.JobStatusProcessing, Metadata: map[string]any{
				terminalEventPendingMetadataKey: true,
				terminalEventStatusMetadataKey:  "completed",
				terminalEventPayloadMetadataKey: `{"status":"completed"}`,
			}}}},
			emitter: &countingEmitter{},
			wantErr: "non-terminal status",
		},
		{
			name: "publish failure does not clear metadata",
			repo: &pendingTerminalRepo{jobs: []*model.Job{{ID: jobID, Status: model.JobStatusCompleted, Metadata: map[string]any{
				terminalEventPendingMetadataKey: true,
				terminalEventStatusMetadataKey:  "completed",
				terminalEventPayloadMetadataKey: `{"status":"completed"}`,
			}}}},
			emitter:       &countingEmitter{err: publishErr},
			wantErr:       "publish required job completed notification",
			wantPublished: 1,
		},
		{
			name: "success clears metadata after publish",
			repo: &pendingTerminalRepo{jobs: []*model.Job{{ID: jobID, Status: model.JobStatusCompleted, Metadata: map[string]any{
				terminalEventPendingMetadataKey: true,
				terminalEventStatusMetadataKey:  "completed",
				terminalEventPayloadMetadataKey: `{"status":"completed"}`,
			}}}},
			emitter:       &countingEmitter{},
			wantCleared:   true,
			wantPublished: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &UseCase{JobRepository: tt.repo, JobEventEmitter: tt.emitter, JobEventStreamingEnabled: true}
			repairer := NewTerminalEventRepairer(uc, testLogger())

			err := repairer.RepairOnce(testContext())
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.wantPublished, tt.emitter.count)
			require.Equal(t, tt.wantCleared, tt.repo.clearedID != uuid.Nil)
		})
	}
}

type countingEmitter struct {
	count     int
	err       error
	tenantIDs []string
}

func (e *countingEmitter) Emit(ctx context.Context, _ streaming.EmitRequest) error {
	e.count++
	e.tenantIDs = append(e.tenantIDs, tmcore.GetTenantIDContext(ctx))
	return e.err
}

func (e *countingEmitter) Close() error { return nil }

func (e *countingEmitter) Healthy(context.Context) error { return nil }

type pendingTerminalRepo struct {
	jobs                 []*model.Job
	listErr              error
	clearedID            uuid.UUID
	requireTenantContext bool
	seenTenantIDs        []string
	seenTenantDB         []bool
}

func (r *pendingTerminalRepo) ListPendingTerminalEvents(ctx context.Context, _ int) ([]*model.Job, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}

	if r.requireTenantContext {
		if tmcore.GetTenantIDContext(ctx) == "" || tmcore.GetMBContext(ctx) == nil {
			return nil, tmcore.ErrTenantContextRequired
		}
	}

	r.seenTenantIDs = append(r.seenTenantIDs, tmcore.GetTenantIDContext(ctx))
	r.seenTenantDB = append(r.seenTenantDB, tmcore.GetMBContext(ctx) != nil)

	return r.jobs, nil
}

func (r *pendingTerminalRepo) ClearTerminalEventMetadata(_ context.Context, id uuid.UUID) error {
	r.clearedID = id
	return nil
}

func (r *pendingTerminalRepo) Create(context.Context, *model.Job) (*model.Job, error) {
	return nil, nil
}

func (r *pendingTerminalRepo) Update(context.Context, *model.Job) (*model.Job, error) {
	return nil, nil
}

func (r *pendingTerminalRepo) UpdateStatus(context.Context, uuid.UUID, model.JobStatus, string, string, map[string]any) error {
	return nil
}

func (r *pendingTerminalRepo) FindByID(context.Context, uuid.UUID) (*model.Job, error) {
	return nil, nil
}

func (r *pendingTerminalRepo) FindByRequestHashWithinWindow(context.Context, string, int) (*model.Job, error) {
	return nil, nil
}

func (r *pendingTerminalRepo) FindActiveByRequestHash(context.Context, string) (*model.Job, error) {
	return nil, nil
}

func (r *pendingTerminalRepo) List(context.Context, *jobPort.ListFilter) ([]*model.Job, error) {
	return nil, nil
}

func (r *pendingTerminalRepo) ExistsRunningByMappedFieldKey(context.Context, string) (bool, error) {
	return false, nil
}

var _ jobPort.Repository = (*pendingTerminalRepo)(nil)

type activeTenantRepo struct {
	tenants []*tmclient.TenantSummary
	err     error
}

func (r *activeTenantRepo) GetActiveTenantsByService(context.Context, string) ([]*tmclient.TenantSummary, error) {
	return r.tenants, r.err
}

type tenantMongoResolver struct {
	failTenants map[string]error
}

func (r *tenantMongoResolver) GetDatabaseForTenant(_ context.Context, tenantID string) (*mongo.Database, error) {
	if err := r.failTenants[tenantID]; err != nil {
		return nil, err
	}

	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return nil, err
	}

	return client.Database(tenantID), nil
}
