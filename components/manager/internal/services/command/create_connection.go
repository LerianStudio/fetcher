package command

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"

	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	productRepo "github.com/LerianStudio/fetcher/pkg/mongodb/product"
	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type CreateConnection struct {
	connRepo    connRepo.Repository
	productRepo productRepo.Repository
	cryptor     crypto.Cryptor
}

func NewCreateConnection(connectionRepo connRepo.Repository, prodRepo productRepo.Repository, cryptor crypto.Cryptor) *CreateConnection {
	return &CreateConnection{
		connRepo:    connectionRepo,
		productRepo: prodRepo,
		cryptor:     cryptor,
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

	// Parse and validate productId
	productID, err := uuid.Parse(connInput.ProductID)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "invalid product id", err)

		return nil, pkg.ValidateBadRequestFieldsError(
			map[string]string{"productId": "productId is required and must be a valid UUID"},
			nil,
			"connection",
			nil,
		)
	}

	span.SetAttributes(attribute.String("app.request.product_id", productID.String()))

	// Validate product exists and belongs to the organization
	product, err := s.productRepo.FindByID(ctx, productID, organizationID)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to validate product", err)
		return nil, err
	}

	if product == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"product",
		)
	}

	sslMode, sslCA, sslCert, sslKey := s.extractSSLFields(connInput)

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
		connInput.Metadata,
		sslMode,
		sslCA,
		sslCert,
		sslKey,
	)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to create connection model", err)
		return nil, err
	}

	connection.ProductID = &productID

	existing, errRepo := s.connRepo.FindByOrganizationAndName(ctx, connection.OrganizationID, connection.ConfigName)
	if errRepo != nil {
		return nil, errRepo
	}

	if existing != nil {
		libOpentelemetry.HandleSpanError(&span, "Connection config name conflict", nil)
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityConflict,
			"connection",
		)
	}

	created, err := s.connRepo.Create(ctx, connection)
	if err != nil {
		return nil, err
	}

	return created, nil
}

// extractSSLFields extracts SSL configuration pointers from the connection input.
func (s *CreateConnection) extractSSLFields(input model.ConnectionInput) (sslMode, sslCA, sslCert, sslKey *string) {
	if input.SSL == nil || input.SSL.IsEmpty() {
		return nil, nil, nil, nil
	}

	sslMode = &input.SSL.Mode
	sslCA = &input.SSL.CA

	if input.SSL.Cert != nil {
		sslCert = input.SSL.Cert
	}

	if input.SSL.Key != nil {
		sslKey = input.SSL.Key
	}

	return sslMode, sslCA, sslCert, sslKey
}
