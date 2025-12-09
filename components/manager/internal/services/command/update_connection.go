package command

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type UpdateConnection struct {
	connRepo connRepo.Repository
	jobRepo  job.Repository
	cryptor  crypto.Cryptor
}

func NewUpdateConnection(connectionRepo connRepo.Repository, jobRepo job.Repository, cryptor crypto.Cryptor) *UpdateConnection {
	return &UpdateConnection{
		connRepo: connectionRepo,
		jobRepo:  jobRepo,
		cryptor:  cryptor,
	}
}

func (s *UpdateConnection) Execute(ctx context.Context, organizationID, connectionID uuid.UUID, connInput model.ConnectionInput) (*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.update_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	if err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", connInput); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert fetcher input to JSON string", err)
	}

	if err := s.validateNoActiveJobs(ctx, organizationID, connectionID); err != nil {
		return nil, err
	}

	current, err := s.findConnection(ctx, connectionID, organizationID)
	if err != nil {
		return nil, err
	}

	sslMode, sslCA, sslCert, sslKey := resolveSSLFields(connInput.SSL)

	if errPatch := current.ApplyPatch(
		ctx,
		s.cryptor,
		&connInput.ConfigName,
		&connInput.Type,
		&connInput.Host,
		&connInput.Port,
		&connInput.DatabaseName,
		&connInput.Username,
		&connInput.Password,
		sslMode,
		sslCA,
		sslCert,
		sslKey,
	); errPatch != nil {
		return nil, errPatch
	}

	// TODO: Test the database connection with the new data before persisting. If it fails, return the failure to the user; only update when the connection test passes.

	return s.persistConnection(ctx, current)
}

func (s *UpdateConnection) validateNoActiveJobs(ctx context.Context, organizationID, connectionID uuid.UUID) error {
	active, err := s.jobRepo.ExistsRunningByConnection(ctx, organizationID, connectionID)
	if err != nil {
		return err
	}

	if active {
		return pkg.ValidateBusinessError(constant.ErrJobInProgress, "connection", "cannot update connection with active jobs")
	}

	return nil
}

func (s *UpdateConnection) findConnection(ctx context.Context, connectionID, organizationID uuid.UUID) (*model.Connection, error) {
	current, err := s.connRepo.FindByID(ctx, connectionID, organizationID)
	if err != nil {
		return nil, err
	}

	if current == nil {
		return nil, pkg.EntityNotFoundError{
			EntityType: "connection",
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		}
	}

	return current, nil
}

func (s *UpdateConnection) persistConnection(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
	updated, err := s.connRepo.Update(ctx, conn)
	if err != nil {
		return nil, err
	}

	if updated == nil {
		return nil, pkg.EntityNotFoundError{
			EntityType: "connection",
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		}
	}

	return updated, nil
}

func resolveSSLFields(ssl *model.SSLInput) (mode, ca, cert, key *string) {
	if ssl == nil || ssl.IsEmpty() {
		return nil, nil, nil, nil
	}

	return &ssl.Mode, &ssl.CA, ssl.Cert, ssl.Key
}
