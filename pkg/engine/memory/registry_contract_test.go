// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
)

// recordingFactory is a host-side ConnectorFactory double for registry tests.
// Build performs no I/O; it just records the type it was registered under.
type recordingFactory struct {
	typeName string
}

func (f *recordingFactory) Build(_ context.Context, _ engine.ConnectionDescriptor) (engine.Connector, error) {
	return &recordingConnector{}, nil
}

// recordingConnector is a minimal Connector that satisfies the lifecycle
// contract for registry resolution tests.
type recordingConnector struct{ connected bool }

func (c *recordingConnector) TestConnection(_ context.Context) error { c.connected = true; return nil }
func (c *recordingConnector) DiscoverSchema(_ context.Context) (engine.SchemaSnapshot, error) {
	return engine.SchemaSnapshot{}, nil
}

func (c *recordingConnector) Query(_ context.Context, _ engine.ExtractionRequest) (map[string][]map[string]any, error) {
	return nil, nil
}
func (c *recordingConnector) Close(_ context.Context) error { c.connected = false; return nil }

// The registry must satisfy the Engine ConnectorRegistry port exactly.
var _ engine.ConnectorRegistry = (*memory.ConnectorRegistry)(nil)

func TestConnectorRegistry_ResolveFactoryByType(t *testing.T) {
	t.Parallel()

	registry := memory.NewConnectorRegistry()
	registry.Register("postgres", &recordingFactory{typeName: "postgres"})
	registry.Register("mongodb", &recordingFactory{typeName: "mongodb"})

	// Deterministic resolution by type via the port method.
	factory, ok := registry.Connector("postgres")
	if !ok {
		t.Fatalf("Connector(postgres): expected ok=true")
	}
	if rf, isRecording := factory.(*recordingFactory); !isRecording || rf.typeName != "postgres" {
		t.Fatalf("Connector(postgres): resolved wrong factory %#v", factory)
	}

	// Build through the resolved factory yields a live-capable connector with no
	// hidden connect on build.
	conn, err := factory.Build(context.Background(), engine.ConnectionDescriptor{Type: "postgres"})
	if err != nil {
		t.Fatalf("Build: unexpected error: %v", err)
	}
	if rc, isRecording := conn.(*recordingConnector); !isRecording || rc.connected {
		t.Fatalf("Build must not connect; got %#v", conn)
	}
}

func TestConnectorRegistry_UnknownType_StableEngineError(t *testing.T) {
	t.Parallel()

	registry := memory.NewConnectorRegistry()
	registry.Register("postgres", &recordingFactory{typeName: "postgres"})

	// Port-level miss reports ok=false without an I/O excursion.
	if _, ok := registry.Connector("oracle"); ok {
		t.Fatalf("Connector(oracle): expected ok=false for unknown type")
	}

	// Error-returning resolution yields a stable *engine.EngineError.
	_, err := registry.LookupOrError("oracle")
	if err == nil {
		t.Fatalf("LookupOrError(oracle): expected error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("LookupOrError(oracle): error is not *engine.EngineError: %T", err)
	}
	if engErr.Category != engine.CategoryNotFound {
		t.Fatalf("LookupOrError(oracle): category = %q, want %q", engErr.Category, engine.CategoryNotFound)
	}
}
