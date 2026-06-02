// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package memory

import (
	"context"
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

// Connector is an in-memory engine.Connector with deterministic, injectable
// behavior so runner tests (ST-T007-02/03/04) can exercise the success path and
// every failure path without a real datasource. It serves canned Rows from
// Query and RECORDS its lifecycle: CloseCount counts every Close call so a test
// can assert the runner's "close every opened connector" invariant on both the
// success and failure paths.
//
// The error fields mirror the cache/store/sink doubles' *Err idiom: when set,
// the corresponding lifecycle step returns the injected error so a test can
// force that step to fail deterministically. They are inert (nil) by default.
type Connector struct {
	mu sync.Mutex

	// Rows is the canned Query result keyed by qualified table name. Query
	// returns a defensive shallow copy so callers cannot mutate the canned data.
	Rows map[string][]map[string]any

	// Schema is the canned snapshot DiscoverSchema returns. It lets a test seed
	// the schema the planner validates a request against before the runner
	// executes the resulting plan.
	Schema engine.SchemaSnapshot

	// TestErr, when non-nil, makes TestConnection fail.
	TestErr error
	// QueryErr, when non-nil, makes Query fail. The injected error is the raw
	// host error the runner is responsible for redacting at its boundary.
	QueryErr error
	// CloseErr, when non-nil, makes Close return an error AFTER recording the
	// call, so a Close failure never hides the fact that Close was attempted.
	CloseErr error

	// testConnectionCalls and queryCalls record lifecycle invocations so a test
	// can assert the runner drove TestConnection before Query.
	testConnectionCalls int
	queryCalls          int
	// closeCount records every Close call. The runner MUST close every connector
	// it opens, on both the success and failure paths; CloseCount is how a test
	// proves the invariant.
	closeCount int
}

// TestConnection implements engine.Connector. It records the call and returns
// the injected TestErr when set.
func (c *Connector) TestConnection(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.testConnectionCalls++

	return c.TestErr
}

// DiscoverSchema implements engine.Connector. It returns the canned Schema so a
// test can drive PlanExtraction (which validates against the discovered schema)
// before the runner executes the resulting plan. The runner itself never calls
// DiscoverSchema — planning already validated the mapping.
func (c *Connector) DiscoverSchema(_ context.Context) (engine.SchemaSnapshot, error) {
	return c.Schema, nil
}

// Query implements engine.Connector. It records the call, returns the injected
// QueryErr when set, and otherwise returns a defensive copy of the canned Rows.
func (c *Connector) Query(_ context.Context, _ engine.ExtractionRequest) (map[string][]map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.queryCalls++

	if c.QueryErr != nil {
		return nil, c.QueryErr
	}

	out := make(map[string][]map[string]any, len(c.Rows))
	for table, rows := range c.Rows {
		copied := make([]map[string]any, len(rows))
		copy(copied, rows)
		out[table] = copied
	}

	return out, nil
}

// Close implements engine.Connector. It ALWAYS records the call before
// returning CloseErr, so a Close error never masks the fact that the runner
// attempted to close the connector. It is safe to call more than once.
func (c *Connector) Close(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closeCount++

	return c.CloseErr
}

// CloseCount returns the number of times Close was called. It is a harness
// affordance for tests asserting the close-every-opened-connector invariant.
func (c *Connector) CloseCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.closeCount
}

// ResetLifecycle zeroes the recorded lifecycle counts (build is recorded on the
// factory, not here). It lets a test discard the lifecycle of a SETUP operation —
// e.g. PlanExtraction, which builds and closes a connector to discover schema —
// so a later assertion measures only the operation under test (the runner).
func (c *Connector) ResetLifecycle() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.testConnectionCalls = 0
	c.queryCalls = 0
	c.closeCount = 0
}

// QueryCalls returns the number of times Query was called.
func (c *Connector) QueryCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.queryCalls
}

// TestConnectionCalls returns the number of times TestConnection was called.
func (c *Connector) TestConnectionCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.testConnectionCalls
}

// ConnectorFactory is an in-memory engine.ConnectorFactory. Build is side-effect
// free with respect to the network: it returns the pre-seeded Connector (or a
// fresh empty one) WITHOUT opening any connection, preserving the connector
// contract's build/connect separation. A BuildErr, when set, makes Build fail so
// a test can exercise the runner's build-failure path.
type ConnectorFactory struct {
	mu sync.Mutex

	// Conn is the connector Build returns. When nil, Build lazily allocates and
	// retains a fresh empty Connector so a test can inspect it after execution.
	Conn *Connector
	// BuildErr, when non-nil, makes Build fail before producing a connector.
	BuildErr error
	// buildCount records how many times Build was invoked.
	buildCount int
}

// NewConnectorFactory returns a factory that builds the supplied connector. A
// nil connector makes Build allocate a fresh empty Connector on first use.
func NewConnectorFactory(conn *Connector) *ConnectorFactory {
	return &ConnectorFactory{Conn: conn}
}

// Build implements engine.ConnectorFactory. It performs NO I/O: it records the
// call and returns the seeded connector, deferring all connectivity to
// TestConnection. When BuildErr is set it fails before producing a connector.
func (f *ConnectorFactory) Build(_ context.Context, _ engine.ConnectionDescriptor) (engine.Connector, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.buildCount++

	if f.BuildErr != nil {
		return nil, f.BuildErr
	}

	if f.Conn == nil {
		f.Conn = &Connector{}
	}

	return f.Conn, nil
}

// BuildCount returns the number of times Build was invoked.
func (f *ConnectorFactory) BuildCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.buildCount
}

// compile-time proof the harness doubles satisfy the Engine connector contracts.
var (
	_ engine.Connector        = (*Connector)(nil)
	_ engine.ConnectorFactory = (*ConnectorFactory)(nil)
)
