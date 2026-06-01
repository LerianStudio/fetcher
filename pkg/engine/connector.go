// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "context"

// The connector contracts model the explicit lifecycle of a datasource
// connector as INTERFACES only. The Engine core never imports a concrete
// database driver; the host wraps its drivers (pkg/postgres, pkg/mongodb, ...)
// behind these seams in pkg/enginecompat. Keeping construction (ConnectorFactory)
// separate from connectivity (Connector) is the load-bearing invariant: building
// a connector MUST NOT open a network connection, so the Engine can resolve and
// inspect connectors deterministically before deciding to connect.
//
// Lifecycle, in order:
//  1. ConnectorFactory.Build   — construct a Connector from a descriptor (no I/O).
//  2. Connector.TestConnection — explicit connectivity check (the only connect step).
//  3. Connector.DiscoverSchema — read the datasource schema snapshot.
//  4. Connector.Query          — execute an extraction request.
//  5. Connector.Close          — release the connection.
//
// All scope is tenant scope. The connector contracts carry no organization or
// product concept: the host supplies an already tenant-scoped descriptor, and
// any tenant gating happens at the Engine's connection-operation layer, never
// inside the connector contract.

// ConnectorFactory builds a Connector from a secret-free ConnectionDescriptor.
// Build MUST be side-effect free with respect to the network: it constructs the
// connector value and defers all connectivity to Connector.TestConnection. The
// HOST owns how the descriptor's credential is resolved (via CredentialProtector)
// and supplied to the concrete driver; the factory contract takes only the
// secret-free descriptor so it stays freely loggable.
type ConnectorFactory interface {
	// Build constructs a Connector for the given descriptor without opening a
	// connection. It returns a CategoryValidation *EngineError when the
	// descriptor is malformed for the target datasource type.
	Build(ctx context.Context, descriptor ConnectionDescriptor) (Connector, error)
}

// Connector is a host-provided datasource connector with an explicit lifecycle.
// It is an INTERFACE: concrete drivers live in the host and are wrapped behind
// this seam, never inside Engine core. A Connector is obtained from a
// ConnectorFactory and is NOT connected until TestConnection succeeds.
//
// Implementations MUST NOT embed credentials, DSNs, or any secret material in
// returned errors; errors crossing the Engine boundary are pre-redacted.
//
// A Connector instance is SINGLE-FLIGHT: it is not safe for concurrent use, and
// its build, test, discover/query, and close steps all belong to one goroutine.
type Connector interface {
	// TestConnection performs the explicit connectivity check. It is the single
	// step that opens the underlying connection. It returns a CategoryUnavailable
	// *EngineError when the datasource is unreachable.
	TestConnection(ctx context.Context) error
	// DiscoverSchema reads the datasource's schema and returns a secret-free
	// snapshot of tables and fields.
	DiscoverSchema(ctx context.Context) (SchemaSnapshot, error)
	// Query executes the extraction request and returns rows keyed by qualified
	// table name. The result carries data only — no secrets.
	Query(ctx context.Context, request ExtractionRequest) (map[string][]map[string]any, error)
	// Close releases the underlying connection. It is safe to call after a failed
	// TestConnection and idempotent enough that a double Close does not panic.
	Close(ctx context.Context) error
}

// UnknownConnectorTypeError returns the stable Engine error for a datasource
// type with no registered connector factory. It is CategoryNotFound — the
// referenced connector resource does not exist — and carries no secret material.
// The Engine and the in-memory registry share this single constructor so the
// unknown-type failure is identical everywhere a registry resolves by type.
func UnknownConnectorTypeError(datasourceType string) *EngineError {
	return NewEngineError(CategoryNotFound, "no connector registered for datasource type "+datasourceType)
}
