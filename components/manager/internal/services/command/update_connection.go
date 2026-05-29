package command

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/ports/job"

	"github.com/LerianStudio/lib-commons/v5/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"

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

func (s *UpdateConnection) Execute(ctx context.Context, connectionID uuid.UUID, connInput model.ConnectionUpdateInput) (*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.update_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", connInput.ToMapWithMask(), nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert fetcher input to JSON string", err)
	}

	current, err := s.connRepo.FindByID(ctx, connectionID)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to find connection by ID", err)
		return nil, fmt.Errorf("failed to find connection by id: %w", err)
	}

	if current == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection not found", constant.ErrEntityNotFound)

		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	active, err := s.jobRepo.ExistsRunningByMappedFieldKey(ctx, current.ConfigName)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to check for active jobs", err)
		return nil, fmt.Errorf("failed to check for active jobs: %w", err)
	}

	if active {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection has active jobs", constant.ErrJobInProgress)

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
		connInput.Schema,
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
		libOpentelemetry.HandleSpanError(span, "Failed to apply connection patch", errPatch)
		return nil, fmt.Errorf("failed to apply connection patch: %w", errPatch)
	}

	updated, err := s.connRepo.Update(ctx, current)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to update connection", err)
		return nil, fmt.Errorf("failed to update connection: %w", err)
	}

	if updated == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Updated connection not found", constant.ErrEntityNotFound)

		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	return updated, nil
}
