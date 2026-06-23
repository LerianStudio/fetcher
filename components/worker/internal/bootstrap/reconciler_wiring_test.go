package bootstrap

import (
	"testing"

	"github.com/LerianStudio/fetcher/v2/components/worker/internal/services"
	libOutbox "github.com/LerianStudio/lib-commons/v5/commons/outbox"
	libLog "github.com/LerianStudio/lib-observability/log"
	"github.com/stretchr/testify/assert"
)

// TestRunLauncher_RegistersReconcilerWhenPresent asserts the launcher seam
// receives the reconciler when it is non-nil (multi-tenant mode), mirroring
// how terminalRepairer is conditionally registered.
func TestRunLauncher_RegistersReconcilerWhenPresent(t *testing.T) {
	originalRunLauncher := runLauncher
	t.Cleanup(func() { runLauncher = originalRunLauncher })

	logger := testBootstrapLogger()
	reconciler := services.NewTenantConsumerReconciler(nil, nil, "fetcher", logger)

	var gotReconciler *services.TenantConsumerReconciler

	called := false
	runLauncher = func(_ libLog.Logger, _ *MultiQueueConsumer, _ *HealthServer, _ *libOutbox.Dispatcher, _ *services.TerminalEventRepairer, r *services.TenantConsumerReconciler) {
		called = true
		gotReconciler = r
	}

	service := &Service{
		MultiQueueConsumer: &MultiQueueConsumer{},
		Logger:             logger,
		reconciler:         reconciler,
	}

	service.Run()

	assert.True(t, called, "expected launcher to be invoked")
	assert.Same(t, reconciler, gotReconciler, "expected reconciler to be passed to launcher")
}

// TestRunLauncher_NilReconcilerInSingleTenant asserts the single-tenant path
// passes a nil reconciler to the launcher seam (no reconciler registered).
func TestRunLauncher_NilReconcilerInSingleTenant(t *testing.T) {
	originalRunLauncher := runLauncher
	t.Cleanup(func() { runLauncher = originalRunLauncher })

	var gotReconciler *services.TenantConsumerReconciler

	called := false
	runLauncher = func(_ libLog.Logger, _ *MultiQueueConsumer, _ *HealthServer, _ *libOutbox.Dispatcher, _ *services.TerminalEventRepairer, r *services.TenantConsumerReconciler) {
		called = true
		gotReconciler = r
	}

	service := &Service{
		MultiQueueConsumer: &MultiQueueConsumer{},
		Logger:             testBootstrapLogger(),
		// reconciler intentionally left nil (single-tenant mode)
	}

	service.Run()

	assert.True(t, called, "expected launcher to be invoked")
	assert.Nil(t, gotReconciler, "expected nil reconciler in single-tenant mode")
}
