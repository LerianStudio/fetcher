// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package memory provides in-memory collaborators that satisfy the embedded
// Engine ports (pkg/engine). They exist for Engine tests and embedded examples
// so the runtime can be exercised without MongoDB, Redis, RabbitMQ, S3,
// SeaweedFS, Fiber, the Manager, or the Worker.
//
// These types are a TEST and EMBEDDED harness, not production persistence: they
// hold state in mutex-protected maps, perform no I/O, read no environment
// variables, and contact no external services. Shared maps are guarded so the
// harness is race-free under concurrent access. List output is deterministically
// ordered. Where a port method returns an error, the harness returns a stable
// *engine.EngineError rather than panicking.
package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/LerianStudio/fetcher/pkg/engine"
)

// tenantScope is the isolation key for stored records. The in-memory harness
// keys every collection by tenant so records owned by one tenant are never
// visible to another, mirroring the ownership boundary the real stores enforce.
type tenantScope struct {
	organizationID string
	productName    string
}

func scopeOf(tenant engine.TenantContext) tenantScope {
	return tenantScope{
		organizationID: tenant.OrganizationID,
		productName:    tenant.ProductName,
	}
}

// connectionKey identifies a stored connection within its tenant scope.
type connectionKey struct {
	scope      tenantScope
	configName string
}

// storedConnection pairs the secret-free descriptor with the protected
// credential sidecar so the harness can model encrypt-before-store persistence
// without serializing or logging the secret material.
type storedConnection struct {
	descriptor engine.ConnectionDescriptor
	credential *engine.ProtectedCredential
}

// ConnectionStore is an in-memory engine.ConnectionStore. Beyond the port's
// FindConnection lookup it offers Create/List/Update/Delete helpers so tests and
// embedded examples can seed and manage connection descriptors. All access is
// guarded by a single RWMutex.
type ConnectionStore struct {
	mu          sync.RWMutex
	connections map[connectionKey]storedConnection
}

// NewConnectionStore returns an empty in-memory connection store.
func NewConnectionStore() *ConnectionStore {
	return &ConnectionStore{
		connections: make(map[connectionKey]storedConnection),
	}
}

// FindConnection implements engine.ConnectionStore. It returns the descriptor
// for the named connection within the tenant scope and whether it exists. It
// never returns secret material because ConnectionDescriptor carries none.
func (s *ConnectionStore) FindConnection(
	_ context.Context,
	tenant engine.TenantContext,
	configName string,
) (engine.ConnectionDescriptor, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, ok := s.connections[connectionKey{scope: scopeOf(tenant), configName: configName}]

	return stored.descriptor, ok, nil
}

// Create stores a new connection descriptor for the tenant. It returns a
// CategoryValidation error if a connection with the same config name already
// exists for that tenant. The protected credential, when supplied, is persisted
// alongside the descriptor as opaque material and never serialized.
func (s *ConnectionStore) Create(
	_ context.Context,
	tenant engine.TenantContext,
	descriptor engine.ConnectionDescriptor,
	credential *engine.ProtectedCredential,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := connectionKey{scope: scopeOf(tenant), configName: descriptor.ConfigName}
	if _, exists := s.connections[key]; exists {
		return engine.NewEngineError(engine.CategoryValidation, "connection already exists")
	}

	s.connections[key] = storedConnection{descriptor: descriptor, credential: cloneCredential(credential)}

	return nil
}

// Update replaces an existing connection descriptor for the tenant. It returns a
// CategoryNotFound error when no connection with that config name exists. A nil
// credential means no password change was supplied: the existing stored secret
// is preserved and never wiped.
func (s *ConnectionStore) Update(
	_ context.Context,
	tenant engine.TenantContext,
	descriptor engine.ConnectionDescriptor,
	credential *engine.ProtectedCredential,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := connectionKey{scope: scopeOf(tenant), configName: descriptor.ConfigName}

	existing, exists := s.connections[key]
	if !exists {
		return engine.NewEngineError(engine.CategoryNotFound, "connection not found")
	}

	updated := storedConnection{descriptor: descriptor, credential: existing.credential}
	if credential != nil {
		updated.credential = cloneCredential(credential)
	}

	s.connections[key] = updated

	return nil
}

// ProtectedCredential returns the protected credential stored for the named
// connection within the tenant scope and whether one is present. It is a TEST
// and embedded-harness accessor — the production port never returns secret
// material across the Engine boundary.
func (s *ConnectionStore) ProtectedCredential(
	tenant engine.TenantContext,
	configName string,
) (engine.ProtectedCredential, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, ok := s.connections[connectionKey{scope: scopeOf(tenant), configName: configName}]
	if !ok || stored.credential == nil {
		return engine.ProtectedCredential{}, false
	}

	return *cloneCredential(stored.credential), true
}

// cloneCredential deep-copies a protected credential so stored material is
// insulated from later mutation of the caller's slice.
func cloneCredential(credential *engine.ProtectedCredential) *engine.ProtectedCredential {
	if credential == nil {
		return nil
	}

	return &engine.ProtectedCredential{
		Ciphertext: append([]byte(nil), credential.Ciphertext...),
		KeyVersion: credential.KeyVersion,
	}
}

// Delete removes a connection descriptor for the tenant. It returns a
// CategoryNotFound error when no connection with that config name exists.
func (s *ConnectionStore) Delete(
	_ context.Context,
	tenant engine.TenantContext,
	configName string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := connectionKey{scope: scopeOf(tenant), configName: configName}
	if _, exists := s.connections[key]; !exists {
		return engine.NewEngineError(engine.CategoryNotFound, "connection not found")
	}

	delete(s.connections, key)

	return nil
}

// List returns the connection descriptors owned by the tenant, sorted by config
// name for deterministic output.
func (s *ConnectionStore) List(
	_ context.Context,
	tenant engine.TenantContext,
) ([]engine.ConnectionDescriptor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scope := scopeOf(tenant)
	descriptors := make([]engine.ConnectionDescriptor, 0)

	for key, stored := range s.connections {
		if key.scope == scope {
			descriptors = append(descriptors, stored.descriptor)
		}
	}

	sort.Slice(descriptors, func(i, j int) bool {
		return descriptors[i].ConfigName < descriptors[j].ConfigName
	})

	return descriptors, nil
}

// executionKey identifies a stored execution within its tenant scope.
type executionKey struct {
	scope tenantScope
	jobID string
}

// ExecutionStore is an in-memory engine.ExecutionStore. It upserts and reads
// execution lifecycle state, guarded by an RWMutex.
type ExecutionStore struct {
	mu         sync.RWMutex
	executions map[executionKey]engine.ExecutionState
}

// NewExecutionStore returns an empty in-memory execution store.
func NewExecutionStore() *ExecutionStore {
	return &ExecutionStore{
		executions: make(map[executionKey]engine.ExecutionState),
	}
}

// SaveExecution implements engine.ExecutionStore by upserting the execution
// state for the tenant. Repeated saves model status transitions.
func (s *ExecutionStore) SaveExecution(
	_ context.Context,
	tenant engine.TenantContext,
	state engine.ExecutionState,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.executions[executionKey{scope: scopeOf(tenant), jobID: state.JobID}] = state

	return nil
}

// FindExecution implements engine.ExecutionStore. It returns the execution
// state for the job within the tenant scope and whether it exists.
func (s *ExecutionStore) FindExecution(
	_ context.Context,
	tenant engine.TenantContext,
	jobID string,
) (engine.ExecutionState, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.executions[executionKey{scope: scopeOf(tenant), jobID: jobID}]

	return state, ok, nil
}
