package rabbitmq

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	tmcore "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/core"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewPublisherRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	logger := &mockLogger{}
	telemetry := &opentelemetry.Telemetry{}

	pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
	assert.NotNil(t, pr)
	assert.Equal(t, mockAdapter, pr.adapter)

	prNilTelemetry := NewPublisherRoutesWithAdapter(mockAdapter, logger, nil)
	assert.NotNil(t, prNilTelemetry)
}

func TestPublisherRoutes_Publish(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	logger := &mockLogger{}
	telemetry := &opentelemetry.Telemetry{}
	ctx := context.Background()

	t.Run("successful publish", func(t *testing.T) {
		mockAdapter.EXPECT().
			ProducerDefault(gomock.Any(), "test-exchange", "test-key", gomock.Any(), nil).
			Return(nil)

		pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
		err := pr.Publish(ctx, "test-exchange", "test-key", []byte("test body"))
		assert.NoError(t, err)
	})

	t.Run("publish error", func(t *testing.T) {
		mockAdapter.EXPECT().
			ProducerDefault(gomock.Any(), "test-exchange", "test-key", gomock.Any(), nil).
			Return(errors.New("publish error"))

		pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
		err := pr.Publish(ctx, "test-exchange", "test-key", []byte("test body"))
		assert.Error(t, err)
	})
}

func TestPublisherRoutes_Shutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	logger := &mockLogger{}
	telemetry := &opentelemetry.Telemetry{}
	ctx := context.Background()

	t.Run("successful shutdown", func(t *testing.T) {
		mockAdapter.EXPECT().Shutdown(gomock.Any()).Return(nil)

		pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
		err := pr.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("shutdown error", func(t *testing.T) {
		mockAdapter.EXPECT().Shutdown(gomock.Any()).Return(errors.New("shutdown error"))

		pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
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
			logger := &mockLogger{}
			telemetry := &opentelemetry.Telemetry{}

			ctx := context.Background()
			if tt.tenantID != "" {
				ctx = tmcore.SetTenantIDInContext(ctx, tt.tenantID)
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
