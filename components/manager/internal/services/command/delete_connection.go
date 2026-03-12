package command

import (
	"context"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"

	"github.com/LerianStudio/lib-commons/v4/commons"

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

func (s *DeleteConnection) Execute(ctx context.Context, organizationID, connectionID uuid.UUID) error {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.delete_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	current, err := s.connRepo.FindByID(ctx, connectionID, organizationID)
	if err != nil {
		return err
	}

	if current == nil {
		return pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	active, err := s.jobRepo.ExistsRunningByMappedFieldKey(ctx, organizationID, current.ConfigName)
	if err != nil {
		return err
	}

	if active {
		return pkg.ValidateBusinessError(constant.ErrJobInProgress, "connection", "cannot delete connection with active jobs")
	}

	if err := s.connRepo.Delete(ctx, connectionID, organizationID, time.Now().UTC()); err != nil {
		return err
	}

	return nil
}
