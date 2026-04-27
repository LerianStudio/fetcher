package bootstrap

import (
	"testing"
)

func TestService_StructComposition(t *testing.T) {
	// Service can be partially constructed without infrastructure.
	// Run() requires real dependencies, so we test the struct directly.
	svc := &Service{}

	if svc.MultiQueueConsumer != nil {
		t.Error("MultiQueueConsumer should be nil by default")
	}

	if svc.Logger != nil {
		t.Error("Logger should be nil by default")
	}

	if svc.licenseShutdown != nil {
		t.Error("licenseShutdown should be nil by default")
	}

	if svc.mtCleanup != nil {
		t.Error("mtCleanup should be nil by default")
	}

	if svc.healthServer != nil {
		t.Error("healthServer should be nil by default")
	}
}
