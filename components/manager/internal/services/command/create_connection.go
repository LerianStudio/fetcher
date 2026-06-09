package command

import (
	"context"
	"fmt"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/v2/pkg/model"

	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"go.opentelemetry.io/otel/attribute"
)

type CreateConnection struct {
	cryptor crypto.Cryptor
	engine  *engine.Engine
}

func NewCreateConnection(cryptor crypto.Cryptor, eng *engine.Engine) *CreateConnection {
	return &CreateConnection{
		cryptor: cryptor,
		engine:  eng,
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

	// The Engine is the AUTHORITY for the connection-create rules: it validates
	// the per-request tenant scope and enforces (tenantID, configName)
	// uniqueness, then persists the rich record through the connectioncompat
	// ConnectionStore adapter. The Manager keeps the rich model, credential
	// encryption, ProductName, and response mapping; the rich record rides to
	// persistence inside the Engine descriptor's opaque host payload, so no
	// field is dropped and ProductName never becomes an Engine scope dimension.
	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to derive tenant scope", err)
		return nil, fmt.Errorf("failed to derive tenant scope: %w", err)
	}

	descriptor, err := s.engine.CreateConnection(ctx, tenant, engineInputFromConnection(connection))
	if err != nil {
		if conflict := mapEngineCreateError(err); conflict != err {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection config name conflict", nil)
			return nil, conflict
		}

		libOpentelemetry.HandleSpanError(span, "Failed to create connection", err)

		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// The Engine returns the secret-free descriptor carrying the rich record it
	// persisted in the opaque payload. Unpack it so the response is byte-identical
	// to the pre-delegation create result.
	created := connectioncompat.ConnectionFromDescriptor(descriptor)
	if created == nil {
		created = connection
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
