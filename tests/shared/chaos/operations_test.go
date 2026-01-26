package chaos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChaosOperations_DisableService_NilProxy(t *testing.T) {
	registry := NewProxyRegistry()
	ops := NewChaosOperations(registry)

	// Should not error for nil proxy (service not registered)
	err := ops.DisableService(ServicePostgres)
	assert.NoError(t, err, "DisableService should handle nil proxy gracefully")
}

func TestChaosOperations_EnableService_NilProxy(t *testing.T) {
	registry := NewProxyRegistry()
	ops := NewChaosOperations(registry)

	err := ops.EnableService(ServicePostgres)
	assert.NoError(t, err, "EnableService should handle nil proxy gracefully")
}

func TestChaosOperations_AddChaos_NilProxy(t *testing.T) {
	registry := NewProxyRegistry()
	ops := NewChaosOperations(registry)

	config := DefaultLatencyConfig(100, 10)
	toxic, err := ops.AddChaos(ServicePostgres, config)
	assert.Error(t, err, "AddChaos should error for nil proxy")
	assert.Nil(t, toxic)
}

func TestChaosOperations_RemoveChaos_NilProxy(t *testing.T) {
	registry := NewProxyRegistry()
	ops := NewChaosOperations(registry)

	err := ops.RemoveChaos(ServicePostgres, "test-toxic")
	assert.NoError(t, err, "RemoveChaos should handle nil proxy gracefully")
}

func TestChaosOperations_ResetAll_EmptyRegistry(t *testing.T) {
	registry := NewProxyRegistry()
	ops := NewChaosOperations(registry)

	err := ops.ResetAll()
	assert.NoError(t, err, "ResetAll should handle empty registry")
}

func TestNewChaosOperations_NilRegistry_Panics(t *testing.T) {
	assert.Panics(t, func() {
		NewChaosOperations(nil)
	}, "NewChaosOperations should panic with nil registry")
}
