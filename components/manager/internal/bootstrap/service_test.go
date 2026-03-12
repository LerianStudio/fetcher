package bootstrap

import (
	"testing"
)

func TestService_StructComposition(t *testing.T) {
	// Service can be partially constructed without infrastructure.
	// Run() requires a real Server and Logger, so we test the struct directly.
	svc := &Service{}

	if svc.Server != nil {
		t.Error("Server should be nil by default")
	}

	if svc.Logger != nil {
		t.Error("Logger should be nil by default")
	}
}
