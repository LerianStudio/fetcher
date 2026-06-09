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
// keys every collection by tenantId so records owned by one tenant are never
// visible to another, mirroring the ownership boundary the real stores enforce.
// tenantId is the sole isolation dimension; the Engine carries no org/product
// scope.
type tenantScope struct {
	tenantID string
}

func scopeOf(tenant engine.TenantContext) tenantScope {
	return tenantScope{tenantID: tenant.TenantID}
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

	return s.scopedDescriptorsLocked(scopeOf(tenant)), nil
}

// FindByID implements the ID-addressed engine.ConnectionStore lookup. The
// harness addresses by the opaque descriptor ID within the tenant scope; a
// connection whose descriptor carries no ID is never matched, and a soft-deleted
// connection reports found=false (the harness models a hard map delete, so an
// absent key is the deleted state). The Engine never interprets the ID — the
// harness treats it as an opaque token exactly as the production adapter does.
func (s *ConnectionStore) FindByID(
	_ context.Context,
	tenant engine.TenantContext,
	id string,
) (engine.ConnectionDescriptor, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, _, found := s.findByIDLocked(scopeOf(tenant), id)
	if !found {
		return engine.ConnectionDescriptor{}, false, nil
	}

	return stored.descriptor, true, nil
}

// UpdateByID replaces the connection addressed by the opaque ID within the
// tenant scope. A nil credential preserves the existing stored secret. A missing
// ID yields the Engine's not-found rule.
func (s *ConnectionStore) UpdateByID(
	_ context.Context,
	tenant engine.TenantContext,
	id string,
	descriptor engine.ConnectionDescriptor,
	credential *engine.ProtectedCredential,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, key, found := s.findByIDLocked(scopeOf(tenant), id)
	if !found {
		return engine.NewEngineError(engine.CategoryNotFound, "connection not found")
	}

	updated := storedConnection{descriptor: descriptor, credential: existing.credential}
	if credential != nil {
		updated.credential = cloneCredential(credential)
	}

	// The descriptor's config name may change under an ID-addressed update; the
	// map is keyed by config name, so re-key when it does to keep both addressing
	// dimensions (ID and config name) consistent.
	if descriptor.ConfigName != key.configName {
		delete(s.connections, key)
		key = connectionKey{scope: key.scope, configName: descriptor.ConfigName}
	}

	s.connections[key] = updated

	return nil
}

// DeleteByID removes the connection addressed by the opaque ID within the tenant
// scope. A missing ID yields the Engine's not-found rule. The harness models a
// hard map delete; the production adapter maps this to the Manager's SOFT delete.
func (s *ConnectionStore) DeleteByID(
	_ context.Context,
	tenant engine.TenantContext,
	id string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, key, found := s.findByIDLocked(scopeOf(tenant), id)
	if !found {
		return engine.NewEngineError(engine.CategoryNotFound, "connection not found")
	}

	delete(s.connections, key)

	return nil
}

// ListPaged implements the OPAQUE-params paginated list. The harness does not
// interpret the params (it has no real pagination concept); it returns the full
// tenant set with Total set to the item count, which is enough to exercise the
// Engine's tenant-scope-only delegation. The production adapter reproduces the
// Manager's exact pagination behavior.
func (s *ConnectionStore) ListPaged(
	_ context.Context,
	tenant engine.TenantContext,
	_ engine.ConnectionListParams,
) (engine.ConnectionPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.scopedDescriptorsLocked(scopeOf(tenant))

	return engine.ConnectionPage{Items: items, Total: int64(len(items))}, nil
}

// scopedDescriptorsLocked returns the tenant's descriptors sorted by config name
// for deterministic output. Callers must hold at least the read lock.
func (s *ConnectionStore) scopedDescriptorsLocked(scope tenantScope) []engine.ConnectionDescriptor {
	descriptors := make([]engine.ConnectionDescriptor, 0)

	for key, stored := range s.connections {
		if key.scope == scope {
			descriptors = append(descriptors, stored.descriptor)
		}
	}

	sort.Slice(descriptors, func(i, j int) bool {
		return descriptors[i].ConfigName < descriptors[j].ConfigName
	})

	return descriptors
}

// findByIDLocked resolves a stored connection by the opaque descriptor ID within
// the tenant scope, returning it with its map key. An empty id never matches.
// Callers must hold at least the read lock.
func (s *ConnectionStore) findByIDLocked(scope tenantScope, id string) (storedConnection, connectionKey, bool) {
	if id == "" {
		return storedConnection{}, connectionKey{}, false
	}

	for key, stored := range s.connections {
		if key.scope == scope && stored.descriptor.ID == id {
			return stored, key, true
		}
	}

	return storedConnection{}, connectionKey{}, false
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

// Find returns the saved execution state for the job within the tenant scope
// and whether it exists. It is a harness affordance for tests to assert that
// SaveExecution landed, not part of the Engine port.
func (s *ExecutionStore) Find(tenant engine.TenantContext, jobID string) (engine.ExecutionState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.executions[executionKey{scope: scopeOf(tenant), jobID: jobID}]

	return state, ok
}

// RecordingExecutionStore is an in-memory engine.ExecutionStore that records the
// ORDER of status transitions the runner drives, so tests can assert the
// processing -> completed / failed / cancelled call ordering deterministically.
//
// SaveErr is a test affordance mirroring the sink's PutErr idiom: when set,
// SaveExecution returns a safe Engine error so tests can verify that a
// best-effort status-write failure does not corrupt the extraction result. The
// transition is still recorded so ordering remains observable.
type RecordingExecutionStore struct {
	mu       sync.RWMutex
	statuses []engine.ExecutionStatus

	// SaveErr, when non-nil, makes SaveExecution return a safe Engine error after
	// recording the attempted status.
	SaveErr error
}

// NewRecordingExecutionStore returns an empty recording execution store.
func NewRecordingExecutionStore() *RecordingExecutionStore {
	return &RecordingExecutionStore{}
}

// SaveExecution implements engine.ExecutionStore by recording the status in
// transition order. When SaveErr is set it returns a safe error WITHOUT echoing
// the injected detail, after recording the attempted transition.
func (s *RecordingExecutionStore) SaveExecution(
	_ context.Context,
	_ engine.TenantContext,
	state engine.ExecutionState,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.statuses = append(s.statuses, state.Status)

	if s.SaveErr != nil {
		return engine.NewEngineError(engine.CategoryUnavailable, "execution store unavailable")
	}

	return nil
}

// Statuses returns a copy of the recorded status transitions in order. It is a
// harness affordance for tests, not part of the Engine port.
func (s *RecordingExecutionStore) Statuses() []engine.ExecutionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]engine.ExecutionStatus, len(s.statuses))
	copy(out, s.statuses)

	return out
}
