package command

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/engine"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/ports/job"
	observability "github.com/LerianStudio/lib-observability"

	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type DeleteConnection struct {
	connRepo connRepo.Repository
	jobRepo  job.Repository
	engine   *engine.Engine
}

func NewDeleteConnection(connectionRepo connRepo.Repository, jobRepo job.Repository, eng *engine.Engine) *DeleteConnection {
	return &DeleteConnection{
		connRepo: connectionRepo,
		jobRepo:  jobRepo,
		engine:   eng,
	}
}

func (s *DeleteConnection) Execute(ctx context.Context, connectionID uuid.UUID) error {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.delete_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	// Route the per-request tenant-scope authority AND persistence through the
	// Engine, symmetric with UpdateConnection. Delete's read flows through the
	// Engine's ID-addressed FindByID and the SOFT delete through DeleteByID (the
	// connectioncompat adapter maps it to repo.Delete with a deleted_at stamp).
	// The Manager keeps its conflict gate and HTTP mapping.
	if err := authorizeConnectionAccess(ctx, s.engine); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to authorize tenant scope", err)
		return err
	}

	current, err := getConnectionByIDViaEngine(ctx, s.engine, connectionID)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to find connection by ID", err)
		return fmt.Errorf("failed to find connection by id: %w", err)
	}

	if current == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection not found", constant.ErrEntityNotFound)

		return pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	if err := checkActiveJobsViaEngine(ctx, s.engine, current.ConfigName); err != nil {
		if conflict := asActiveJobConflict(err, "cannot delete connection with active jobs"); conflict != nil {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection has active jobs", constant.ErrJobInProgress)

			return conflict
		}

		libOpentelemetry.HandleSpanError(span, "Failed to check for active jobs", err)

		return fmt.Errorf("failed to check for active jobs: %w", err)
	}

	if err := deleteConnectionByIDViaEngine(ctx, s.engine, connectionID); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to delete connection", err)
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "deleted connection",
		libLog.String("connection_id", connectionID.String()),
	)

	return nil
}
