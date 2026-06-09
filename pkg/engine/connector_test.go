// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
)

// lifecycleRecord captures which Connector lifecycle methods a fake connector
// observed and in what order, so a test can prove the explicit lifecycle is
// driven through the contract rather than via hidden side effects.
type lifecycleRecord struct {
	calls []string
}

func (r *lifecycleRecord) note(step string) { r.calls = append(r.calls, step) }

// fakeConnector is a host-side connector double. It records every lifecycle
// call. Construction (via fakeFactory.Build) is deliberately side-effect free:
// the connector does NOT open any network connection until TestConnection is
// invoked, proving build is separable from connectivity.
type fakeConnector struct {
	record    *lifecycleRecord
	connected bool
}

func (c *fakeConnector) TestConnection(_ context.Context) error {
	c.record.note("test")
	c.connected = true

	return nil
}

func (c *fakeConnector) DiscoverSchema(_ context.Context) (engine.SchemaSnapshot, error) {
	c.record.note("discover")
	if !c.connected {
		return engine.SchemaSnapshot{}, errors.New("discover before connect")
	}

	return engine.SchemaSnapshot{ConfigName: "fake", Tables: []engine.TableSnapshot{{Name: "public.t"}}}, nil
}

func (c *fakeConnector) QueryStream(_ context.Context, _ engine.ExtractionRequest) (engine.RowCursor, error) {
	c.record.note("query")
	if !c.connected {
		return nil, errors.New("query before connect")
	}

	return engine.NewEagerCursor(map[string][]map[string]any{"public.t": {{"id": 1}}}), nil
}

func (c *fakeConnector) Close(_ context.Context) error {
	c.record.note("close")
	c.connected = false

	return nil
}

// fakeFactory builds fakeConnectors. Build performs NO I/O: it only constructs
// the connector value, deferring all connectivity to TestConnection.
type fakeFactory struct {
	record    *lifecycleRecord
	buildErr  error
	descrSeen engine.ConnectionDescriptor
}

func (f *fakeFactory) Build(_ context.Context, descriptor engine.ConnectionDescriptor) (engine.Connector, error) {
	f.record.note("build")
	f.descrSeen = descriptor
	if f.buildErr != nil {
		return nil, f.buildErr
	}

	return &fakeConnector{record: f.record}, nil
}

// compile-time proof the fakes satisfy the Engine contracts.
var (
	_ engine.Connector        = (*fakeConnector)(nil)
	_ engine.ConnectorFactory = (*fakeFactory)(nil)
)

func TestConnectorLifecycle_BuildIsSeparableFromConnect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &lifecycleRecord{}
	factory := &fakeFactory{record: record}

	conn, err := factory.Build(ctx, engine.ConnectionDescriptor{ConfigName: "pg-main", Type: "postgres"})
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	// After build and BEFORE TestConnection, no connectivity must have occurred:
	// only the build step is recorded.
	if len(record.calls) != 1 || record.calls[0] != "build" {
		t.Fatalf("Build must be side-effect free; recorded calls = %v, want [build]", record.calls)
	}
	if factory.descrSeen.ConfigName != "pg-main" {
		t.Fatalf("Build did not receive descriptor; got %#v", factory.descrSeen)
	}

	// Now drive the full lifecycle in the order the Connector godoc documents:
	// test -> discover -> query -> close.
	if err := conn.TestConnection(ctx); err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}
	if _, err := conn.DiscoverSchema(ctx); err != nil {
		t.Fatalf("DiscoverSchema: unexpected error: %v", err)
	}
	if _, err := conn.QueryStream(ctx, engine.ExtractionRequest{}); err != nil {
		t.Fatalf("QueryStream: unexpected error: %v", err)
	}
	if err := conn.Close(ctx); err != nil {
		t.Fatalf("Close: unexpected error: %v", err)
	}

	want := []string{"build", "test", "discover", "query", "close"}
	if len(record.calls) != len(want) {
		t.Fatalf("lifecycle calls = %v, want %v", record.calls, want)
	}
	for i, step := range want {
		if record.calls[i] != step {
			t.Fatalf("lifecycle call[%d] = %q, want %q (full sequence %v)", i, record.calls[i], step, record.calls)
		}
	}
}

func TestConnectorContract_DiscoverAndQueryRequireConnect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &lifecycleRecord{}
	factory := &fakeFactory{record: record}

	conn, err := factory.Build(ctx, engine.ConnectionDescriptor{Type: "postgres"})
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}

	// Discover/query before TestConnection must fail in the fake — proving the
	// contract makes connectivity an explicit, separate step.
	if _, err := conn.DiscoverSchema(ctx); err == nil {
		t.Fatalf("DiscoverSchema before connect: expected error, got nil")
	}
	if _, err := conn.QueryStream(ctx, engine.ExtractionRequest{}); err == nil {
		t.Fatalf("QueryStream before connect: expected error, got nil")
	}
}
