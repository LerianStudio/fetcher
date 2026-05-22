package services

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
	jobPort "github.com/LerianStudio/fetcher/pkg/ports/job"
	streaming "github.com/LerianStudio/lib-streaming"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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

type countingEmitter struct{ count int }

func (e *countingEmitter) Emit(context.Context, streaming.EmitRequest) error {
	e.count++
	return nil
}

func (e *countingEmitter) Close() error { return nil }

func (e *countingEmitter) Healthy(context.Context) error { return nil }

type pendingTerminalRepo struct {
	jobs      []*model.Job
	clearedID uuid.UUID
}

func (r *pendingTerminalRepo) ListPendingTerminalEvents(context.Context, int) ([]*model.Job, error) {
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
