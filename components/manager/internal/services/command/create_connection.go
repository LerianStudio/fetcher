package command

import (
	"context"
	"fmt"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"

	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

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

func (s *CreateConnection) Execute(ctx context.Context, connInput model.ConnectionInput, productName string) (*model.Connection, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.create_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.product_name", productName),
	)

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", connInput.ToMapWithMask(), nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert fetcher input to JSON string", err)
	}

	sslMode, sslCA, sslCert, sslKey := s.extractSSLFields(connInput)

	var schema *string
	if connInput.Schema != "" {
		schema = &connInput.Schema
	}

	connection, err := model.NewConnection(
		ctx, s.cryptor,
		productName,
		connInput.ConfigName,
		connInput.Type,
		connInput.Host,
		connInput.Port,
		connInput.DatabaseName,
		schema,
		connInput.Username,
		connInput.Password,
		connInput.Metadata,
		sslMode,
		sslCA,
		sslCert,
		sslKey,
	)
	if err != nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Failed to create connection model", err)
		return nil, fmt.Errorf("failed to create connection model: %w", err)
	}

	existing, errRepo := s.connRepo.FindByName(ctx, connection.ConfigName)
	if errRepo != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to check existing connection", errRepo)
		return nil, fmt.Errorf("failed to check for existing connection: %w", errRepo)
	}

	if existing != nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection config name conflict", nil)

		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityConflict,
			"connection",
		)
	}

	created, err := s.connRepo.Create(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
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
