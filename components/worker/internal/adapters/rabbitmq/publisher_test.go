package rabbitmq

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	libConstants "github.com/LerianStudio/lib-commons/v5/commons/constants"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	observability "github.com/LerianStudio/lib-observability"
	obsConstants "github.com/LerianStudio/lib-observability/constants"
	"github.com/LerianStudio/lib-observability/log"
	opentelemetry "github.com/LerianStudio/lib-observability/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/mock/gomock"
)

// mockRabbitMQManager is a mock implementation of RabbitMQManagerInterface.
type mockRabbitMQManager struct {
	channel       RabbitMQChannel
	getChannelErr error
}

func (m *mockRabbitMQManager) GetChannel(_ context.Context, _ string) (RabbitMQChannel, error) {
	if m.getChannelErr != nil {
		return nil, m.getChannelErr
	}

	return m.channel, nil
}

// mockChannel is a mock implementation of RabbitMQChannel.
type mockChannel struct {
	exchangeDeclareErr error
	publishErr         error
	published          amqp.Publishing
	closed             bool
}

func (m *mockChannel) ExchangeDeclare(_, _ string, _, _, _, _ bool, _ amqp.Table) error {
	return m.exchangeDeclareErr
}

func (m *mockChannel) PublishWithContext(_ context.Context, _, _ string, _, _ bool, msg amqp.Publishing) error {
	m.published = msg
	return m.publishErr
}

func (m *mockChannel) Close() error {
	m.closed = true
	return nil
}

func TestNewPublisherRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	nopLogger := log.NewNop()
	telemetry := &opentelemetry.Telemetry{}

	pr := NewPublisherRoutesWithAdapter(mockAdapter, nopLogger, telemetry)
	assert.NotNil(t, pr)
	assert.Equal(t, mockAdapter, pr.adapter)

	prNilTelemetry := NewPublisherRoutesWithAdapter(mockAdapter, nopLogger, nil)
	assert.NotNil(t, prNilTelemetry)
}

func TestPublisherRoutes_Publish(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	nopLogger := log.NewNop()
	telemetry := &opentelemetry.Telemetry{}
	ctx := context.Background()

	t.Run("successful publish", func(t *testing.T) {
		mockAdapter.EXPECT().
			ProducerDefault(gomock.Any(), "test-exchange", "test-key", gomock.Any(), nil).
			Return(nil)

		pr := NewPublisherRoutesWithAdapter(mockAdapter, nopLogger, telemetry)
		err := pr.Publish(ctx, "test-exchange", "test-key", []byte("test body"))
		assert.NoError(t, err)
	})

	t.Run("publish error", func(t *testing.T) {
		mockAdapter.EXPECT().
			ProducerDefault(gomock.Any(), "test-exchange", "test-key", gomock.Any(), nil).
			Return(errors.New("publish error"))

		pr := NewPublisherRoutesWithAdapter(mockAdapter, nopLogger, telemetry)
		err := pr.Publish(ctx, "test-exchange", "test-key", []byte("test body"))
		assert.Error(t, err)
	})
}

func TestPublisherRoutes_Shutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	nopLogger := log.NewNop()
	telemetry := &opentelemetry.Telemetry{}
	ctx := context.Background()

	t.Run("successful shutdown", func(t *testing.T) {
		mockAdapter.EXPECT().Shutdown(gomock.Any()).Return(nil)

		pr := NewPublisherRoutesWithAdapter(mockAdapter, nopLogger, telemetry)
		err := pr.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("shutdown error", func(t *testing.T) {
		mockAdapter.EXPECT().Shutdown(gomock.Any()).Return(errors.New("shutdown error"))

		pr := NewPublisherRoutesWithAdapter(mockAdapter, nopLogger, telemetry)
		err := pr.Shutdown(ctx)
		assert.Error(t, err)
	})
}

// TestPublisherRoutes_Publish_TenantIDPropagation tests that Publish forwards X-Tenant-ID
// from context to AMQP headers when tenant context is present.
func TestPublisherRoutes_Publish_TenantIDPropagation(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		expectTenantID bool
	}{
		{
			name:           "includes X-Tenant-ID in headers when tenant context present",
			tenantID:       "tenant-xyz-456",
			expectTenantID: true,
		},
		{
			name:           "passes nil headers when no tenant context",
			tenantID:       "",
			expectTenantID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAdapter := rabbitmq.NewMockAdapter(ctrl)
			logger := log.NewNop()
			telemetry := &opentelemetry.Telemetry{}

			ctx := context.Background()
			if tt.tenantID != "" {
				ctx = tmcore.ContextWithTenantID(ctx, tt.tenantID)
			}

			// Capture the headers argument passed to ProducerDefault
			var capturedHeaders *map[string]any
			mockAdapter.EXPECT().
				ProducerDefault(gomock.Any(), "test-exchange", "test-key", gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, _ string, _ []byte, h *map[string]any) error {
					capturedHeaders = h
					return nil
				})

			pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
			err := pr.Publish(ctx, "test-exchange", "test-key", []byte("test body"))
			assert.NoError(t, err)

			if tt.expectTenantID {
				assert.NotNil(t, capturedHeaders, "expected non-nil headers when tenant context is present")
				headers := *capturedHeaders
				assert.Equal(t, tt.tenantID, headers["X-Tenant-ID"], "expected X-Tenant-ID header to match tenant ID from context")
			} else {
				// When no tenant context, headers should remain nil (backward compat)
				assert.Nil(t, capturedHeaders, "expected nil headers when no tenant context")
			}
		})
	}
}

func TestPublish_MultiTenant_RequiresTenantID(t *testing.T) {
	t.Parallel()

	mockMgr := &mockRabbitMQManager{channel: &mockChannel{}}
	logger := log.NewNop()
	publisher := NewPublisherRoutesMultiTenant(mockMgr, logger, nil, nil)

	err := publisher.Publish(context.Background(), "test-exchange", "test.key", []byte(`{}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID")
}

func TestPublish_MultiTenant_Success(t *testing.T) {
	t.Parallel()

	mockCh := &mockChannel{}
	mockMgr := &mockRabbitMQManager{channel: mockCh}
	logger := log.NewNop()
	publisher := NewPublisherRoutesMultiTenant(mockMgr, logger, nil, nil)

	ctx := tmcore.ContextWithTenantID(context.Background(), "tenant-123")
	err := publisher.Publish(ctx, "test-exchange", "test.key", []byte(`{"status":"completed"}`))

	require.NoError(t, err)
	assert.True(t, mockCh.closed, "channel should be closed after publish")
}

func TestPublish_MultiTenant_AddsTenantIDHeader(t *testing.T) {
	t.Parallel()

	tenantID := "tenant-header-123"
	mockCh := &mockChannel{}
	mockMgr := &mockRabbitMQManager{channel: mockCh}
	logger := log.NewNop()
	publisher := NewPublisherRoutesMultiTenant(mockMgr, logger, nil, nil)

	ctx := tmcore.ContextWithTenantID(context.Background(), tenantID)
	err := publisher.Publish(ctx, "test-exchange", "test.key", []byte(`{"status":"completed"}`))

	require.NoError(t, err)
	require.NotNil(t, mockCh.published.Headers)
	assert.Equal(t, tenantID, mockCh.published.Headers["X-Tenant-ID"])
}

func TestPublish_MultiTenant_UsesCanonicalSecureEnvelope(t *testing.T) {
	t.Parallel()

	signer, err := crypto.NewHMACSigner([]byte("12345678901234567890123456789012"), crypto.SignatureVersion)
	require.NoError(t, err)

	tenantID := "tenant-envelope-123"
	body := []byte(`{"status":"completed"}`)
	mockCh := &mockChannel{}
	mockMgr := &mockRabbitMQManager{channel: mockCh}
	publisher := NewPublisherRoutesMultiTenant(mockMgr, log.NewNop(), nil, signer)

	tp := sdktrace.NewTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()
	otel.SetTextMapPropagator(propagation.TraceContext{})

	ctx := observability.ContextWithTracer(context.Background(), tp.Tracer("publisher-test"))
	ctx, span := tp.Tracer("publisher-test").Start(ctx, "test.publish")
	defer span.End()
	ctx = tmcore.ContextWithTenantID(ctx, tenantID)
	err = publisher.Publish(ctx, "test-exchange", "test.key", body)

	require.NoError(t, err)
	require.NotNil(t, mockCh.published.Headers)
	assert.Equal(t, amqp.Persistent, mockCh.published.DeliveryMode)
	assert.Equal(t, "application/json", mockCh.published.ContentType)
	assert.Equal(t, tenantID, mockCh.published.Headers["X-Tenant-ID"])
	assert.NotEmpty(t, mockCh.published.Headers[libConstants.HeaderID])
	assert.Equal(t, 0, mockCh.published.Headers["x-retry-count"])
	assert.NotEmpty(t, mockCh.published.Headers[rabbitmq.HeaderMessageSignature])
	assert.NotEmpty(t, mockCh.published.Headers[rabbitmq.HeaderSignatureTimestamp])
	assert.Equal(t, crypto.SignatureVersion, mockCh.published.Headers[rabbitmq.HeaderSignatureVersion])
	assert.NotEmpty(t, mockCh.published.Headers[obsConstants.HeaderTraceparentPascal])
}

func TestPublish_MultiTenant_GetChannelError(t *testing.T) {
	t.Parallel()

	mockMgr := &mockRabbitMQManager{getChannelErr: errors.New("vhost connection failed")}
	logger := log.NewNop()
	publisher := NewPublisherRoutesMultiTenant(mockMgr, logger, nil, nil)

	ctx := tmcore.ContextWithTenantID(context.Background(), "tenant-123")
	err := publisher.Publish(ctx, "test-exchange", "test.key", []byte(`{}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get channel")
}

func TestPublish_MultiTenant_ExchangeDeclareError(t *testing.T) {
	t.Parallel()

	mockCh := &mockChannel{exchangeDeclareErr: errors.New("exchange declare failed")}
	mockMgr := &mockRabbitMQManager{channel: mockCh}
	logger := log.NewNop()
	publisher := NewPublisherRoutesMultiTenant(mockMgr, logger, nil, nil)

	ctx := tmcore.ContextWithTenantID(context.Background(), "tenant-123")
	err := publisher.Publish(ctx, "test-exchange", "test.key", []byte(`{}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to declare exchange")
}

func TestShutdown_MultiTenant_NilAdapter(t *testing.T) {
	t.Parallel()

	// In multi-tenant mode, adapter is nil — Shutdown must not panic
	mockMgr := &mockRabbitMQManager{channel: &mockChannel{}}
	logger := log.NewNop()
	publisher := NewPublisherRoutesMultiTenant(mockMgr, logger, nil, nil)

	err := publisher.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestNewPublisherRoutesMultiTenant_SetsFields(t *testing.T) {
	t.Parallel()

	mockMgr := &mockRabbitMQManager{channel: &mockChannel{}}
	logger := log.NewNop()
	publisher := NewPublisherRoutesMultiTenant(mockMgr, logger, nil, nil)

	assert.NotNil(t, publisher)
	assert.NotNil(t, publisher.rabbitMQManager)
	assert.Nil(t, publisher.adapter, "adapter must be nil in multi-tenant mode")
}

func TestPublish_MultiTenant_PublishError(t *testing.T) {
	t.Parallel()

	mockCh := &mockChannel{publishErr: errors.New("publish failed")}
	mockMgr := &mockRabbitMQManager{channel: mockCh}
	logger := log.NewNop()
	publisher := NewPublisherRoutesMultiTenant(mockMgr, logger, nil, nil)

	ctx := tmcore.ContextWithTenantID(context.Background(), "tenant-123")
	err := publisher.Publish(ctx, "test-exchange", "test.key", []byte(`{}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish message")
}
