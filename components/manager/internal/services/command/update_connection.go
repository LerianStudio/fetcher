package command

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/model"
	observability "github.com/LerianStudio/lib-observability"

	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type UpdateConnection struct {
	cryptor crypto.Cryptor
	engine  *engine.Engine
}

func NewUpdateConnection(cryptor crypto.Cryptor, eng *engine.Engine) *UpdateConnection {
	return &UpdateConnection{
		cryptor: cryptor,
		engine:  eng,
	}
}

func (s *UpdateConnection) Execute(ctx context.Context, connectionID uuid.UUID, connInput model.ConnectionUpdateInput) (*model.Connection, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

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

	// The Engine is the AUTHORITY for the per-request tenant scope AND the
	// connection persistence. Update routes its read and write through the
	// Engine's ID-addressed ops (FindByID/UpdateByID via the connectioncompat
	// adapter over the Manager's UUID-keyed repo); the Manager keeps its rich
	// model, domain patch, cryptor re-encryption, and HTTP response mapping. The
	// active-execution conflict gate also flows through the Engine.
	if err := authorizeConnectionAccess(ctx, s.engine); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to authorize tenant scope", err)
		return nil, err
	}

	current, err := getConnectionByIDViaEngine(ctx, s.engine, connectionID)
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

	if err := checkActiveJobsViaEngine(ctx, s.engine, current.ConfigName); err != nil {
		if conflict := asActiveJobConflict(err, "cannot update connection with active jobs"); conflict != nil {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection has active jobs", constant.ErrJobInProgress)

			return nil, conflict
		}

		libOpentelemetry.HandleSpanError(span, "Failed to check for active jobs", err)

		return nil, fmt.Errorf("failed to check for active jobs: %w", err)
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

	updated, err := updateConnectionByIDViaEngine(ctx, s.engine, connectionID, current)
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
