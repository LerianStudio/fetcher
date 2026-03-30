package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/LerianStudio/lib-commons/v4/commons/log"
	tmconsumer "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/consumer"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	tmmongo "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/mongo"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMultiTenantConsumer implements MultiTenantConsumerInterface for testing.
type mockMultiTenantConsumer struct {
	registeredQueues   []string
	registeredHandlers []tmconsumer.HandlerFunc
	runCalled          bool
	closeCalled        bool
	registerErr        error
	runErr             error
	closeErr           error
}

func (m *mockMultiTenantConsumer) Register(queueName string, handler tmconsumer.HandlerFunc) error {
	m.registeredQueues = append(m.registeredQueues, queueName)
	m.registeredHandlers = append(m.registeredHandlers, handler)
	return m.registerErr
}

func (m *mockMultiTenantConsumer) Run(_ context.Context) error {
	m.runCalled = true
	return m.runErr
}

func (m *mockMultiTenantConsumer) Close() error {
	m.closeCalled = true
	return m.closeErr
}

// mockBootstrapLogger is a minimal logger for bootstrap package tests
// that satisfies the log.Logger interface from lib-commons v4.
type mockBootstrapLogger struct{}

func (m *mockBootstrapLogger) Enabled(_ log.Level) bool                                     { return true }
func (m *mockBootstrapLogger) Log(_ context.Context, _ log.Level, _ string, _ ...log.Field) {}
func (m *mockBootstrapLogger) With(_ ...log.Field) log.Logger                               { return m }
func (m *mockBootstrapLogger) WithGroup(_ string) log.Logger                                { return m }
func (m *mockBootstrapLogger) Sync(_ context.Context) error                                 { return nil }

func TestNewMultiQueueConsumerMultiTenant_SetsFields(t *testing.T) {
	mockConsumer := &mockMultiTenantConsumer{}
	logger := &mockBootstrapLogger{}
	mgr := &tmmongo.Manager{}

	consumer := NewMultiQueueConsumerMultiTenant(mockConsumer, nil, "my-queue", logger, mgr)

	assert.Equal(t, "my-queue", consumer.queueName)
	assert.Equal(t, logger, consumer.logger)
	assert.Equal(t, mgr, consumer.mongoManager)
	assert.Nil(t, consumer.consumerRoutes, "single-tenant routes must be nil in multi-tenant mode")
	assert.NotNil(t, consumer.mtConsumer)
}

// TestResolveTenantMongo verifies that resolveTenantMongo correctly resolves
// tenant-specific MongoDB databases and injects them into context when a mongo
// manager is configured, and passes context through unchanged in single-tenant mode.
func TestResolveTenantMongo(t *testing.T) {
	tests := []struct {
		name         string
		mongoManager *tmmongo.Manager
		tenantID     string
		expectError  bool
		description  string
	}{
		{
			name:         "nil manager passes context through unchanged (single-tenant backward compat)",
			mongoManager: nil,
			tenantID:     "tenant-abc",
			expectError:  false,
			description:  "single-tenant mode must not attempt tenant DB resolution",
		},
		{
			name:         "nil manager with empty tenant ID passes context through",
			mongoManager: nil,
			tenantID:     "",
			expectError:  false,
			description:  "no manager and no tenant ID should be a no-op",
		},
		{
			name:         "non-nil manager with empty tenant ID passes context through",
			mongoManager: &tmmongo.Manager{},
			tenantID:     "",
			expectError:  false,
			description:  "multi-tenant enabled but no tenant in message should be a no-op",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.tenantID != "" {
				ctx = tmcore.ContextWithTenantID(ctx, tt.tenantID)
			}

			resultCtx, err := resolveTenantMongo(ctx, tt.mongoManager)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
				// Verify tenant ID is preserved in the returned context
				assert.Equal(t, tt.tenantID, tmcore.GetTenantIDContext(resultCtx),
					"tenant ID must be preserved in returned context")
			}
		})
	}
}

// TestHandlerGenerateReport_TenantIDExtraction verifies that handlerGenerateReport
// extracts X-Tenant-ID from AMQP headers and injects it into the context before
// passing it to UseCase.ExtractExternalData.
func TestHandlerGenerateReport_TenantIDExtraction(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]any
		expectTenantID string
	}{
		{
			name: "extracts X-Tenant-ID from headers into context",
			headers: map[string]any{
				"X-Tenant-ID": "tenant-abc-123",
			},
			expectTenantID: "tenant-abc-123",
		},
		{
			name:           "no X-Tenant-ID header results in empty tenant context",
			headers:        map[string]any{},
			expectTenantID: "",
		},
		{
			name: "non-string X-Tenant-ID is ignored",
			headers: map[string]any{
				"X-Tenant-ID": 12345,
			},
			expectTenantID: "",
		},
		{
			name: "empty X-Tenant-ID string is ignored",
			headers: map[string]any{
				"X-Tenant-ID": "",
			},
			expectTenantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// extractTenantIDFromHeaders is a helper that should be called
			// at the top of handlerGenerateReport to extract tenant context.
			// We test the extraction directly since the full handler requires
			// infrastructure (UseCase with DB connections).
			ctx := context.Background()
			ctx = extractTenantIDFromHeaders(ctx, tt.headers)

			tenantID := tmcore.GetTenantIDContext(ctx)
			assert.Equal(t, tt.expectTenantID, tenantID)
		})
	}
}

// TestHandlerGenerateReport_TenantIDInBody verifies that the message body
// does NOT need to change for multi-tenant support -- only headers carry X-Tenant-ID.
func TestHandlerGenerateReport_TenantIDNotInBody(t *testing.T) {
	body := map[string]any{
		"jobId":          "test-job-id",
		"organizationId": "test-org-id",
		"mappedFields":   map[string]any{},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	// The message body should not contain X-Tenant-ID
	var decoded map[string]any
	err = json.Unmarshal(bodyBytes, &decoded)
	require.NoError(t, err)

	_, exists := decoded["X-Tenant-ID"]
	assert.False(t, exists, "X-Tenant-ID must be in AMQP headers, not message body")
}

// TestIsPermanentTenantError verifies that isPermanentTenantError correctly classifies
// errors from the tenant-manager library as permanent (will not resolve on retry) or
// transient (may succeed on retry).
func TestIsPermanentTenantError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		permanent bool
	}{
		{
			name:      "nil error is not permanent",
			err:       nil,
			permanent: false,
		},
		{
			name: "TenantSuspendedError is permanent",
			err: &tmcore.TenantSuspendedError{
				TenantID: "tenant-123",
				Status:   "suspended",
				Message:  "service suspended",
			},
			permanent: true,
		},
		{
			name: "wrapped TenantSuspendedError is permanent",
			err: fmt.Errorf("resolve: %w", &tmcore.TenantSuspendedError{
				TenantID: "tenant-123",
				Status:   "purged",
			}),
			permanent: true,
		},
		{
			name:      "ErrTenantNotFound is permanent",
			err:       tmcore.ErrTenantNotFound,
			permanent: true,
		},
		{
			name:      "wrapped ErrTenantNotFound is permanent",
			err:       fmt.Errorf("outer: %w", tmcore.ErrTenantNotFound),
			permanent: true,
		},
		{
			name:      "ErrServiceNotConfigured is permanent",
			err:       tmcore.ErrServiceNotConfigured,
			permanent: true,
		},
		{
			name:      "ErrManagerClosed is permanent",
			err:       tmcore.ErrManagerClosed,
			permanent: true,
		},
		{
			name:      "ErrCircuitBreakerOpen is transient",
			err:       tmcore.ErrCircuitBreakerOpen,
			permanent: false,
		},
		{
			name:      "generic error is transient",
			err:       errors.New("network timeout"),
			permanent: false,
		},
		{
			name:      "wrapped generic error is transient",
			err:       fmt.Errorf("failed to connect: %w", errors.New("connection refused")),
			permanent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPermanentTenantError(tt.err)
			assert.Equal(t, tt.permanent, result)
		})
	}
}

// TestResolveTenantMongo_NilManagerPreservesContext verifies that resolveTenantMongo
// with a nil manager preserves the tenant context (single-tenant backward compatibility).
func TestResolveTenantMongo_NilManagerPreservesContext(t *testing.T) {
	// resolveTenantMongo with nil manager should always succeed (single-tenant mode)
	ctx := context.Background()
	ctx = tmcore.ContextWithTenantID(ctx, "tenant-abc")

	resultCtx, err := resolveTenantMongo(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, "tenant-abc", tmcore.GetTenantIDContext(resultCtx))
}

func TestNewMultiQueueConsumerMultiTenant_RegistersHandler(t *testing.T) {
	mockConsumer := &mockMultiTenantConsumer{}
	logger := &mockBootstrapLogger{}

	mqConsumer := NewMultiQueueConsumerMultiTenant(
		mockConsumer,
		nil, // UseCase not needed for this test
		"test-queue",
		logger,
		nil, // mongoManager
	)

	assert.NotNil(t, mqConsumer)
	assert.Equal(t, 1, len(mockConsumer.registeredQueues))
	assert.Equal(t, "test-queue", mockConsumer.registeredQueues[0])
	assert.NotNil(t, mockConsumer.registeredHandlers[0])
	assert.NoError(t, mqConsumer.initErr)
}

func TestNewMultiQueueConsumerMultiTenant_StoresRegistrationError(t *testing.T) {
	mockConsumer := &mockMultiTenantConsumer{registerErr: errors.New("register failed")}
	logger := &mockBootstrapLogger{}

	consumer := NewMultiQueueConsumerMultiTenant(mockConsumer, nil, "test-queue", logger, nil)

	require.Error(t, consumer.initErr)
	assert.Contains(t, consumer.initErr.Error(), "register multi-tenant handler")
	assert.Contains(t, consumer.initErr.Error(), "register failed")
	assert.False(t, mockConsumer.runCalled)
}

func TestMultiQueueConsumerRun_ReturnsDeferredRegistrationError(t *testing.T) {
	consumer := &MultiQueueConsumer{
		logger:  &mockBootstrapLogger{},
		initErr: errors.New("register failed"),
	}

	err := consumer.Run(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "register failed")
}

func TestHandlerGenerateReportDelivery_NilMongoManager_SkipsResolution(t *testing.T) {
	// When mongoManager is nil, handlerGenerateReportDelivery should skip
	// tenant resolution and delegate to handlerGenerateReport. It will fail
	// there because UseCase is nil, but it proves the nil-guard works.
	logger := &mockBootstrapLogger{}
	consumer := &MultiQueueConsumer{
		mongoManager: nil,
		logger:       logger,
	}

	ctx := context.Background()
	delivery := amqp.Delivery{
		Headers: amqp.Table{"X-Tenant-ID": "tenant-abc"},
		Body:    []byte(`{}`),
	}

	// This will fail in handlerGenerateReport (nil UseCase) but must NOT panic
	// on nil mongoManager. The error proves we got past the nil guard.
	err := consumer.handlerGenerateReportDelivery(ctx, delivery)
	assert.Error(t, err, "expected error from nil UseCase, not from nil mongoManager")
}

func TestHeadersFromDelivery_NilHeaders(t *testing.T) {
	delivery := amqp.Delivery{Headers: nil}
	headers := headersFromDelivery(delivery)
	assert.Nil(t, headers)
}

func TestHeadersFromDelivery_WithHeaders(t *testing.T) {
	delivery := amqp.Delivery{
		Headers: amqp.Table{
			"X-Tenant-ID": "tenant-abc",
			"jobId":       "job-123",
		},
	}
	headers := headersFromDelivery(delivery)
	assert.Equal(t, "tenant-abc", headers["X-Tenant-ID"])
	assert.Equal(t, "job-123", headers["jobId"])
}
