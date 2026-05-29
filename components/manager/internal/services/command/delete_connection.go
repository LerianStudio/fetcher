package command

import (
	"context"
	"fmt"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/ports/job"

	"github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type DeleteConnection struct {
	connRepo connRepo.Repository
	jobRepo  job.Repository
}

func NewDeleteConnection(connectionRepo connRepo.Repository, jobRepo job.Repository) *DeleteConnection {
	return &DeleteConnection{
		connRepo: connectionRepo,
		jobRepo:  jobRepo,
	}
}

func (s *DeleteConnection) Execute(ctx context.Context, connectionID uuid.UUID) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.delete_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	current, err := s.connRepo.FindByID(ctx, connectionID)
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

	active, err := s.jobRepo.ExistsRunningByMappedFieldKey(ctx, current.ConfigName)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to check for active jobs", err)
		return fmt.Errorf("failed to check for active jobs: %w", err)
	}

	if active {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection has active jobs", constant.ErrJobInProgress)

		return pkg.ValidateBusinessError(constant.ErrJobInProgress, "connection", "cannot delete connection with active jobs")
	}

	if err := s.connRepo.Delete(ctx, connectionID, time.Now().UTC()); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to delete connection", err)
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "deleted connection",
		libLog.String("connection_id", connectionID.String()),
	)

	return nil
}
