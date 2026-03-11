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

func (s *UpdateConnection) Execute(ctx context.Context, organizationID, connectionID uuid.UUID, connInput model.ConnectionUpdateInput) (*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.update_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", connInput.ToMapWithMask())
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert fetcher input to JSON string", err)
	}

	current, err := s.connRepo.FindByID(ctx, connectionID, organizationID)
	if err != nil {
		return nil, err
	}

	if current == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	active, err := s.jobRepo.ExistsRunningByMappedFieldKey(ctx, organizationID, current.ConfigName)
	if err != nil {
		return nil, err
	}

	if active {
		return nil, pkg.ValidateBusinessError(
			constant.ErrJobInProgress,
			"connection",
			"cannot update connection with active jobs",
		)
	}

	if errPatch := current.ApplyPatch(
		ctx,
		s.cryptor,
		connInput.ConfigName,
		connInput.Type,
		connInput.Host,
		connInput.Port,
		connInput.DatabaseName,
		connInput.Username,
		connInput.Password,
		connInput.Metadata,
		func() *string {
			if connInput.SSL != nil {
				return connInput.SSL.Mode
			}

			return nil
		}(),
		func() *string {
			if connInput.SSL != nil {
				return connInput.SSL.CA
			}

			return nil
		}(),
		func() *string {
			if connInput.SSL != nil && connInput.SSL.Cert != nil {
				return connInput.SSL.Cert
			}

			return nil
		}(),
		func() *string {
			if connInput.SSL != nil && connInput.SSL.Key != nil {
				return connInput.SSL.Key
			}

			return nil
		}(),
	); errPatch != nil {
		return nil, errPatch
	}

	updated, err := s.connRepo.Update(ctx, current)
	if err != nil {
		return nil, err
	}

	if updated == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	return updated, nil
}
