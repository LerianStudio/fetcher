package command

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"

	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type CreateConnection struct {
	connRepo connRepo.Repository
	cryptor  crypto.Cryptor
}

func NewCreateConnection(connectionRepo connRepo.Repository, cryptor crypto.Cryptor) *CreateConnection {
	return &CreateConnection{
		connRepo: connectionRepo,
		cryptor:  cryptor,
	}
}

func (s *CreateConnection) Execute(ctx context.Context, organizationID uuid.UUID, connInput model.ConnectionInput) (*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.create_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
	)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", connInput.ToMapWithMask())
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert fetcher input to JSON string", err)
	}

	connection, err := model.NewConnection(
		ctx, s.cryptor,
		organizationID,
		connInput.ConfigName,
		connInput.Type,
		connInput.Host,
		connInput.Port,
		connInput.DatabaseName,
		connInput.Username,
		connInput.Password,
		func() *string {
			if connInput.SSL != nil && !connInput.SSL.IsEmpty() {
				return &connInput.SSL.Mode
			}

			return nil
		}(),
		func() *string {
			if connInput.SSL != nil && !connInput.SSL.IsEmpty() {
				return &connInput.SSL.CA
			}

			return nil
		}(),
		func() *string {
			if connInput.SSL != nil && !connInput.SSL.IsEmpty() && connInput.SSL.Cert != nil {
				return connInput.SSL.Cert
			}

			return nil
		}(),
		func() *string {
			if connInput.SSL != nil && !connInput.SSL.IsEmpty() && connInput.SSL.Key != nil {
				return connInput.SSL.Key
			}

			return nil
		}(),
	)
	if err != nil {
		return nil, err
	}

	existing, errRepo := s.connRepo.FindByOrganizationAndName(ctx, connection.OrganizationID, connection.ConfigName)
	if errRepo != nil {
		return nil, errRepo
	}

	if existing != nil {
		return nil, pkg.EntityConflictError{
			EntityType: "connection",
			Code:       constant.ErrEntityConflict.Error(),
			Title:      "Conflict",
			Message:    "connection with the same name already exists",
		}
	}

	created, err := s.connRepo.Create(ctx, connection)
	if err != nil {
		return nil, err
	}

	return created, nil
}
