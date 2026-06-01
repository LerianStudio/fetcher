// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package memory

import (
	"sync"

	"github.com/LerianStudio/fetcher/pkg/engine"
)

// ConnectorRegistry is an in-memory engine.ConnectorRegistry. It resolves fake
// connectors registered by tests through Register, guarded by an RWMutex. The
// connectors themselves are opaque (engine.Connector is `any`); the registry
// only maps a datasource type to whatever value the host registered.
type ConnectorRegistry struct {
	mu         sync.RWMutex
	connectors map[string]engine.Connector
}

// NewConnectorRegistry returns an empty in-memory connector registry.
func NewConnectorRegistry() *ConnectorRegistry {
	return &ConnectorRegistry{
		connectors: make(map[string]engine.Connector),
	}
}

// Register associates a fake connector with a datasource type. A later
// registration for the same type overwrites the earlier one.
func (r *ConnectorRegistry) Register(datasourceType string, connector engine.Connector) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.connectors[datasourceType] = connector
}

// Connector implements engine.ConnectorRegistry. It returns the connector
// registered for the datasource type and whether one exists. It performs no I/O.
func (r *ConnectorRegistry) Connector(datasourceType string) (engine.Connector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	connector, ok := r.connectors[datasourceType]

	return connector, ok
}

// LookupOrError returns the connector for the datasource type or a stable
// *engine.EngineError with CategoryNotFound when none is registered. It is a
// harness affordance for tests that want the error path, not part of the port.
func (r *ConnectorRegistry) LookupOrError(datasourceType string) (engine.Connector, error) {
	connector, ok := r.Connector(datasourceType)
	if !ok {
		return nil, engine.NewEngineError(engine.CategoryNotFound, "connector not registered for datasource type")
	}

	return connector, nil
}
