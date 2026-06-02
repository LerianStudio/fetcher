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

// ConnectorBehavior is the injectable, COPY-by-value behavior of a fake
// connector: the canned data and the per-step failure switches. It is held on
// the template Connector a test configures (via seedSource) and is cloned into
// every connector the factory builds, so each built connector carries the SAME
// behavior but its OWN lifecycle counters. Separating behavior (shared, copied)
// from counters (per-instance) is what makes build-per-step honest: the runner's
// freshly-built connector records its own Close/Query independently of the
// planner's, instead of both mutating one shared instance.
type ConnectorBehavior struct {
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

	// BlockOnContext, when true, makes Query block in a select on ctx.Done()
	// instead of returning immediately, modelling a slow datasource. It returns
	// the context's error (context.DeadlineExceeded or context.Canceled) the
	// instant the context is done, so a timeout/cancel test is deterministic and
	// leaves no goroutine blocked. It still records the Query call first, so a
	// test can assert Query was entered. This is the context-respecting affordance
	// ST-T007-04 relies on for leak-free timeout testing.
	BlockOnContext bool
	// LargeRowFactory, when non-nil, lazily produces the canned Query rows. It
	// lets a test build an oversized result without holding the bytes in a struct
	// field. When set it takes precedence over Rows.
	LargeRowFactory func() map[string][]map[string]any
	// AfterQuery, when non-nil, runs AFTER a successful Query returns its rows. It
	// lets a test cancel the context (or otherwise mutate state) the instant a
	// step completes, so the runner's BETWEEN-steps / DURING-assembly context
	// guards fire deterministically rather than racily.
	AfterQuery func()
}

// Connector is an in-memory engine.Connector with deterministic, injectable
// behavior so runner tests (ST-T007-02/03/04) can exercise the success path and
// every failure path without a real datasource. It serves canned Rows from
// Query and RECORDS its lifecycle: CloseCount counts every Close call so a test
// can assert the runner's "close every opened connector" invariant on both the
// success and failure paths.
//
// A Connector plays two roles. A TEMPLATE connector (returned by seedSource) is
// configured by a test and never executed directly: the factory clones its
// embedded ConnectorBehavior into each connector it builds. The template's
// lifecycle accessors (CloseCount, QueryCalls) DELEGATE to
// the connector the factory built most recently — the one the RUNNER opened —
// so a test asserts against the runner's freshly-built connector, not a reused
// planner instance. A BUILT connector (returned by the factory) carries the
// cloned behavior and its own counters; it is the instance the runner drives.
//
// The error fields mirror the cache/store/sink doubles' *Err idiom: when set,
// the corresponding lifecycle step returns the injected error so a test can
// force that step to fail deterministically. They are inert (nil) by default.
type Connector struct {
	mu sync.Mutex

	ConnectorBehavior

	// factory, when set, makes the lifecycle accessors of this (template)
	// connector delegate to the factory's most-recently-built connector. It is
	// nil on a built connector, which records its own counters directly.
	factory *ConnectorFactory

	// queryCalls records Query invocations so a test can assert the runner drove
	// Query the expected number of times.
	queryCalls int
	// closeCount records every Close call. The runner MUST close every connector
	// it opens, on both the success and failure paths; CloseCount is how a test
	// proves the invariant.
	closeCount int
}

// NewTemplateConnector returns a TEMPLATE connector carrying the given behavior.
// A test configures it (further field assignments are allowed via the promoted
// ConnectorBehavior fields) and wires it into a ConnectorFactory; the factory
// clones the behavior into each connector it builds. The template is never
// executed directly — its lifecycle accessors delegate to the runner's built
// connector. A composite literal cannot set the promoted ConnectorBehavior
// fields, so this constructor is the seam that seeds them.
func NewTemplateConnector(behavior ConnectorBehavior) *Connector {
	return &Connector{ConnectorBehavior: behavior}
}

// TestConnection implements engine.Connector. It records the call and returns
// the injected TestErr when set.
func (c *Connector) TestConnection(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

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
// QueryErr when set, optionally blocks on ctx (BlockOnContext) to model a slow
// datasource so a timeout/cancel test is deterministic, and otherwise returns a
// defensive copy of the canned Rows (or the LargeRowFactory output). The runner
// drives the BUILT connector, so the call is recorded on its own counter.
func (c *Connector) Query(ctx context.Context, _ engine.ExtractionRequest) (map[string][]map[string]any, error) {
	c.mu.Lock()
	c.queryCalls++
	queryErr := c.QueryErr
	block := c.BlockOnContext
	rowFactory := c.LargeRowFactory
	canned := c.Rows
	afterQuery := c.AfterQuery
	c.mu.Unlock()

	if queryErr != nil {
		// Run the post-query hook even on the error path so a test can cancel the
		// context and THEN observe a generic (non-context) query error — exercising
		// the runner's "generic error while context is done" mapping guard.
		if afterQuery != nil {
			afterQuery()
		}

		return nil, queryErr
	}

	// Respect the context: when asked to block, wait until the context is done
	// and return its error. This makes the timeout/cancel path deterministic and
	// leaves NO goroutine parked — the wait ends the instant the deadline fires
	// or the context is cancelled.
	if block {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	var out map[string][]map[string]any
	if rowFactory != nil {
		out = rowFactory()
	} else {
		out = make(map[string][]map[string]any, len(canned))
		for table, rows := range canned {
			copied := make([]map[string]any, len(rows))
			copy(copied, rows)
			out[table] = copied
		}
	}

	// Run the post-query hook AFTER the rows are produced so a test can cancel the
	// context the instant a step succeeds, exercising the runner's between-steps
	// and during-assembly context guards deterministically.
	if afterQuery != nil {
		afterQuery()
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

// CloseCount returns the number of times Close was called. On a TEMPLATE
// connector it DELEGATES to the factory's most-recently-built connector — the
// one the runner opened — so the close-every-opened-connector invariant is
// asserted against the runner's freshly-built connector, not the planner's.
// On a built connector it returns its own count.
func (c *Connector) CloseCount() int {
	if target := c.lifecycleTarget(); target != c {
		return target.CloseCount()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.closeCount
}

// QueryCalls returns the number of times Query was called, delegating to the
// runner's freshly-built connector when called on a template (see CloseCount).
func (c *Connector) QueryCalls() int {
	if target := c.lifecycleTarget(); target != c {
		return target.QueryCalls()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.queryCalls
}

// lifecycleTarget returns the connector whose counters a lifecycle accessor
// should read. A template connector (factory set) defers to the factory's
// most-recently-built connector — the runner's — falling back to itself when
// nothing has been built yet. A built connector returns itself.
func (c *Connector) lifecycleTarget() *Connector {
	c.mu.Lock()
	factory := c.factory
	c.mu.Unlock()

	if factory == nil {
		return c
	}

	if last := factory.LastBuilt(); last != nil {
		return last
	}

	return c
}

// ConnectorFactory is an in-memory engine.ConnectorFactory. Build is side-effect
// free with respect to the network: it CLONES the template connector's behavior
// into a FRESH connector on every call (build-per-step), WITHOUT opening any
// connection, preserving the connector contract's build/connect separation. Each
// built connector is distinct (its own pointer and counters), so a bug where the
// runner closed the planner's connector instead of its own would be visible. A
// BuildErr, when set, makes Build fail so a test can exercise the runner's
// build-failure path. A NilConnector flag forces the (nil, nil) buggy-host
// branch so a test can drive the runner's nil-connector guard.
type ConnectorFactory struct {
	mu sync.Mutex

	// template carries the behavior every built connector clones. Tests configure
	// it through the *Connector seedSource returns; its lifecycle accessors
	// delegate to the most-recently-built connector below.
	template *Connector
	// built records every connector this factory produced, in build order, so a
	// test can assert against the runner's connector (the last one built).
	built []*Connector

	// BuildErr, when non-nil, makes Build fail before producing a connector.
	BuildErr error
	// NilConnector, when true, makes Build return (nil, nil) — the buggy-host
	// branch the runner's isNilPort guard must catch without panicking. It takes
	// precedence over producing a real connector but not over BuildErr.
	NilConnector bool
}

// NewConnectorFactory returns a factory that clones the supplied connector's
// behavior into each connector it builds. A nil connector makes the factory
// build behavior-less connectors. The supplied connector is wired as the
// factory's template so its lifecycle accessors delegate to the runner's built
// connector.
func NewConnectorFactory(conn *Connector) *ConnectorFactory {
	f := &ConnectorFactory{template: conn}
	if conn != nil {
		conn.factory = f
	}

	return f
}

// Build implements engine.ConnectorFactory. It performs NO I/O: it clones the
// template's behavior into a FRESH connector, records it, and returns it,
// deferring all connectivity to TestConnection. When BuildErr is set it fails
// before producing a connector; when NilConnector is set it returns (nil, nil)
// to exercise the runner's nil-connector guard.
func (f *ConnectorFactory) Build(_ context.Context, _ engine.ConnectionDescriptor) (engine.Connector, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.BuildErr != nil {
		// A failed build produces NO connector to record or close: the runner must
		// never close something it never built.
		return nil, f.BuildErr
	}

	if f.NilConnector {
		// The buggy-host branch: a real *Connector value is never produced, so the
		// returned engine.Connector is an untyped nil the runner must reject.
		return nil, nil
	}

	conn := &Connector{}

	if f.template != nil {
		// snapshotBehavior locks the TEMPLATE connector's own mutex, which is a
		// distinct mutex from f.mu, so taking it while holding f.mu cannot deadlock.
		conn.ConnectorBehavior = f.template.snapshotBehavior()
	}

	f.built = append(f.built, conn)

	return conn, nil
}

// snapshotBehavior returns a copy of the template's injectable behavior under
// its lock, so Build clones the data and failure switches the test configured.
func (c *Connector) snapshotBehavior() ConnectorBehavior {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.ConnectorBehavior
}

// BuildCount returns the number of connectors this factory built. It excludes
// failed builds (BuildErr / NilConnector produce no connector), so it measures
// exactly how many connectors the runner could have opened.
func (f *ConnectorFactory) BuildCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.built)
}

// LastBuilt returns the most-recently-built connector, or nil when none built.
// In a runner test the planner builds first and the runner builds last, so the
// last built connector is the one the runner opened.
func (f *ConnectorFactory) LastBuilt() *Connector {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.built) == 0 {
		return nil
	}

	return f.built[len(f.built)-1]
}

// compile-time proof the harness doubles satisfy the Engine connector contracts.
var (
	_ engine.Connector        = (*Connector)(nil)
	_ engine.ConnectorFactory = (*ConnectorFactory)(nil)
)
