// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package memory

import (
	"sync"

	"github.com/LerianStudio/fetcher/pkg/engine"
)

// ConnectorRegistry is an in-memory engine.ConnectorRegistry. It resolves
// connector factories registered by tests through Register, guarded by an
// RWMutex. The registry maps a datasource type to the engine.ConnectorFactory
// the host registered; the factory builds a connector without opening a
// connection, preserving the build/connect separation of the connector contract.
type ConnectorRegistry struct {
	mu        sync.RWMutex
	factories map[string]engine.ConnectorFactory
}

// NewConnectorRegistry returns an empty in-memory connector registry.
func NewConnectorRegistry() *ConnectorRegistry {
	return &ConnectorRegistry{
		factories: make(map[string]engine.ConnectorFactory),
	}
}

// Register associates a connector factory with a datasource type. A later
// registration for the same type overwrites the earlier one.
func (r *ConnectorRegistry) Register(datasourceType string, factory engine.ConnectorFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.factories[datasourceType] = factory
}

// Connector implements engine.ConnectorRegistry. It returns the factory
// registered for the datasource type and whether one exists. Resolution is
// deterministic by type and performs no I/O.
func (r *ConnectorRegistry) Connector(datasourceType string) (engine.ConnectorFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, ok := r.factories[datasourceType]

	return factory, ok
}

// LookupOrError returns the factory for the datasource type or the stable
// engine.UnknownConnectorTypeError when none is registered. It is a harness
// affordance for tests that want the error path, not part of the port.
func (r *ConnectorRegistry) LookupOrError(datasourceType string) (engine.ConnectorFactory, error) {
	factory, ok := r.Connector(datasourceType)
	if !ok {
		return nil, engine.UnknownConnectorTypeError(datasourceType)
	}

	return factory, nil
}
