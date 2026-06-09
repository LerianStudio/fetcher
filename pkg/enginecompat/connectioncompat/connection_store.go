// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package connectioncompat

import (
	"context"
	"time"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	netHTTP "github.com/LerianStudio/fetcher/v2/pkg/net/http"
	connPort "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"

	"github.com/google/uuid"
)

// listAllPageSize is the page size the port-completeness List uses to fetch the
// tenant's full connection set. It is intentionally large because the Engine's
// flat List has no pagination concept; the Manager's HTTP List endpoint owns
// real pagination on its own path and never calls this method.
const listAllPageSize = 10000

func listAllFilters() netHTTP.QueryHeader {
	return netHTTP.QueryHeader{Limit: listAllPageSize, Page: 1}
}

// hostRecordKey is the single opaque key under which the adapter carries the
// Manager's RICH connection record through the Engine's secret-free
// ConnectionDescriptor.HostAttributes payload. The Engine never reads this key
// (it treats HostAttributes as a black box), so the rich record — ProductName,
// full SSL CA/Cert/Key, uuid identity, metadata, timestamps,
// EncryptionKeyVersion — round-trips losslessly WITHOUT becoming an Engine
// scoping dimension and WITHOUT field loss.
//
// The value is the *model.Connection itself (a host-side Go pointer carried in
// memory), not a serialized projection: there is no JSON round-trip and thus no
// field erosion. This mirrors the ExtractionPlan.Metadata pass-through
// precedent, scaled to a full record.
const hostRecordKey = "fetcher.connection.record"

// ConnectionStore adapts the Manager's RICH, UUID-keyed connection repository to
// the Engine's thin, (tenantID, configName)-keyed engine.ConnectionStore port.
// It is the persistence seam through which the Engine's connection RULES
// (tenant-scope validation, config-name uniqueness, credential protection,
// active-execution conflict gating, soft-delete) flow for Create/Update/Delete,
// while the Manager keeps the rich model, its UUID identity, and its MongoDB
// repository.
//
// The adapter lives on the HOST side (pkg/enginecompat): the Engine imports the
// port interface, never this adapter or the Manager. That keeps the pkg/engine
// dependency boundary intact.
type ConnectionStore struct {
	repo connPort.Repository
}

// NewConnectionStore builds the adapter over the Manager's connection
// repository. A nil repository yields a nil adapter so the caller can treat
// "no repo" as "no connection store" (the Engine port is optional).
func NewConnectionStore(repo connPort.Repository) *ConnectionStore {
	if repo == nil {
		return nil
	}

	return &ConnectionStore{repo: repo}
}

// DescriptorFromConnection projects a rich *model.Connection onto the Engine's
// secret-free descriptor, packing the FULL record into the opaque
// HostAttributes payload. The typed descriptor fields mirror the record's
// non-secret shape so the Engine can scope by (tenantID, configName); the
// opaque payload carries everything the descriptor cannot type (ProductName,
// full SSL, UUID, metadata, timestamps). The secret (PasswordEncrypted) rides
// inside the host record, never in a typed descriptor field — the descriptor
// stays freely loggable.
func DescriptorFromConnection(conn *model.Connection) engine.ConnectionDescriptor {
	if conn == nil {
		return engine.ConnectionDescriptor{}
	}

	desc := engine.ConnectionDescriptor{
		ID:           conn.ID.String(),
		ConfigName:   conn.ConfigName,
		Type:         string(conn.Type),
		Host:         conn.Host,
		Port:         conn.Port,
		DatabaseName: conn.DatabaseName,
		Username:     conn.Username,
		HostAttributes: map[string]any{
			hostRecordKey: conn,
		},
	}

	if conn.Schema != nil {
		desc.Schema = *conn.Schema
	}

	if conn.SSL != nil {
		desc.SSLMode = conn.SSL.Mode
	}

	return desc
}

// ConnectionFromDescriptor unpacks the rich *model.Connection the Engine carried
// verbatim through the opaque HostAttributes payload. It returns nil when no
// host record is present, so callers can distinguish a packed descriptor from a
// bare one.
func ConnectionFromDescriptor(desc engine.ConnectionDescriptor) *model.Connection {
	if desc.HostAttributes == nil {
		return nil
	}

	conn, ok := desc.HostAttributes[hostRecordKey].(*model.Connection)
	if !ok {
		return nil
	}

	return conn
}

// FindConnection implements engine.ConnectionStore. It resolves the connection
// by config name within the tenant scope (the Manager repo is already
// tenant-scoped through the request context) and packs the rich record into the
// returned descriptor's opaque payload. A missing connection reports found=false
// so the Engine can map it to its not-found rule.
func (s *ConnectionStore) FindConnection(
	ctx context.Context,
	_ engine.TenantContext,
	configName string,
) (engine.ConnectionDescriptor, bool, error) {
	conn, err := s.repo.FindByName(ctx, configName)
	if err != nil {
		return engine.ConnectionDescriptor{}, false, err
	}

	if conn == nil {
		return engine.ConnectionDescriptor{}, false, nil
	}

	return DescriptorFromConnection(conn), true, nil
}

// Create implements engine.ConnectionStore. It reconstructs the rich record from
// the opaque payload and persists it through the Manager repo. The Engine's
// CreateConnection already runs the config-name uniqueness PRE-CHECK through
// this adapter's FindConnection before calling Create, so this method does NOT
// repeat the existence read — that preserves the Manager's pre-delegation call
// shape (exactly one FindByName + one Create). The Manager repo's unique index
// remains the atomic race backstop for two concurrent creates.
func (s *ConnectionStore) Create(
	ctx context.Context,
	_ engine.TenantContext,
	descriptor engine.ConnectionDescriptor,
	_ *engine.ProtectedCredential,
) error {
	conn := ConnectionFromDescriptor(descriptor)
	if conn == nil {
		return engine.NewEngineError(engine.CategoryValidation, "connection record is missing from descriptor")
	}

	if _, err := s.repo.Create(ctx, conn); err != nil {
		return err
	}

	return nil
}

// Update implements engine.ConnectionStore. The rich record carried in the
// opaque payload already holds the patched, re-encrypted state and the original
// UUID identity, so the adapter persists it through the Manager's UUID-keyed
// repo Update directly — no second read, no identity loss.
func (s *ConnectionStore) Update(
	ctx context.Context,
	_ engine.TenantContext,
	descriptor engine.ConnectionDescriptor,
	_ *engine.ProtectedCredential,
) error {
	conn := ConnectionFromDescriptor(descriptor)
	if conn == nil {
		return engine.NewEngineError(engine.CategoryValidation, "connection record is missing from descriptor")
	}

	if _, err := s.repo.Update(ctx, conn); err != nil {
		return err
	}

	return nil
}

// Delete implements engine.ConnectionStore. The Engine addresses connections by
// config name, but the Manager soft-deletes by UUID, so the adapter resolves the
// UUID through the repo and maps the Engine's delete to a SOFT delete
// (deleted_at = now) — never a hard delete. A missing connection is the Engine's
// not-found rule.
func (s *ConnectionStore) Delete(
	ctx context.Context,
	_ engine.TenantContext,
	configName string,
) error {
	conn, err := s.repo.FindByName(ctx, configName)
	if err != nil {
		return err
	}

	if conn == nil {
		return engine.NewEngineError(engine.CategoryNotFound, "connection not found")
	}

	return s.repo.Delete(ctx, conn.ID, time.Now().UTC())
}

// List implements engine.ConnectionStore for port-completeness, packing each
// rich record into its descriptor's opaque payload. The Manager's HTTP List
// endpoint keeps its own paginated, filtered, resolver-merged read (a host
// presentation concern the Engine's flat list does not model), so this method
// is not on the Manager's List hot path; it exists so the port is fully
// satisfied and so an embedded host that wants the flat tenant set can use it.
func (s *ConnectionStore) List(
	ctx context.Context,
	_ engine.TenantContext,
) ([]engine.ConnectionDescriptor, error) {
	conns, _, err := s.repo.List(ctx, listAllFilters())
	if err != nil {
		return nil, err
	}

	descriptors := make([]engine.ConnectionDescriptor, 0, len(conns))
	for _, conn := range conns {
		descriptors = append(descriptors, DescriptorFromConnection(conn))
	}

	return descriptors, nil
}

// FindByID implements the ID-addressed engine.ConnectionStore lookup. The Engine
// hands the opaque host id (the Manager's uuid.UUID rendered as a string); the
// adapter parses it back to a uuid.UUID and resolves through the Manager's
// UUID-keyed repo (already tenant-scoped via the request context). A parse
// failure or a missing/soft-deleted connection reports found=false so the Engine
// maps it to its byte-safe not-found rule — routing by id never resurfaces a
// soft-deleted record because repo.FindByID applies the deleted_at filter.
func (s *ConnectionStore) FindByID(
	ctx context.Context,
	_ engine.TenantContext,
	id string,
) (engine.ConnectionDescriptor, bool, error) {
	connID, parseErr := uuid.Parse(id)
	if parseErr != nil {
		// A malformed opaque id is treated as "not found", NOT a propagated error:
		// the existence oracle must never distinguish a malformed id from a missing
		// one. The parse error carries no tenant-safe signal, so it is deliberately
		// discarded (the Engine maps found=false to its byte-safe not-found rule).
		//nolint:nilerr // intentional: malformed id is an absence, not a surfaced error (existence oracle).
		return engine.ConnectionDescriptor{}, false, nil
	}

	conn, err := s.repo.FindByID(ctx, connID)
	if err != nil {
		return engine.ConnectionDescriptor{}, false, err
	}

	if conn == nil {
		return engine.ConnectionDescriptor{}, false, nil
	}

	return DescriptorFromConnection(conn), true, nil
}

// UpdateByID implements the ID-addressed engine.ConnectionStore write. The rich
// record carried in the opaque payload already holds the host-patched,
// re-encrypted state and the original UUID identity, so the adapter persists it
// through the Manager's UUID-keyed repo Update directly — no second read, no
// identity loss. The id argument is the same opaque token; the persisted record
// carries the authoritative UUID.
func (s *ConnectionStore) UpdateByID(
	ctx context.Context,
	_ engine.TenantContext,
	_ string,
	descriptor engine.ConnectionDescriptor,
	_ *engine.ProtectedCredential,
) error {
	conn := ConnectionFromDescriptor(descriptor)
	if conn == nil {
		return engine.NewEngineError(engine.CategoryValidation, "connection record is missing from descriptor")
	}

	updated, err := s.repo.Update(ctx, conn)
	if err != nil {
		return err
	}

	// repo.Update returns (nil, nil) when the record no longer matches the
	// non-deleted filter — e.g. it was soft-deleted between the read and the write
	// (TOCTOU). The Manager's pre-delegation contract surfaced this as not-found
	// rather than a fake-success returning the stale record, so the adapter maps it
	// to the Engine's not-found rule to keep the public response byte-identical.
	if updated == nil {
		return engine.NewEngineError(engine.CategoryNotFound, "connection not found")
	}

	return nil
}

// DeleteByID implements the ID-addressed engine.ConnectionStore delete. The
// adapter parses the opaque id back to the Manager's uuid.UUID and maps the
// Engine's delete to a SOFT delete (deleted_at = now) — never a hard delete. An
// unparseable id is the Engine's not-found rule.
func (s *ConnectionStore) DeleteByID(
	ctx context.Context,
	_ engine.TenantContext,
	id string,
) error {
	connID, err := uuid.Parse(id)
	if err != nil {
		return engine.NewEngineError(engine.CategoryNotFound, "connection not found")
	}

	return s.repo.Delete(ctx, connID, time.Now().UTC())
}

// ListPaged implements the OPAQUE-params paginated list. It unpacks the host's
// native net/http.QueryHeader from the opaque ConnectionListParams.Filter and
// runs the Manager's UUID-keyed, tenant-scoped repo List with the EXACT filters
// and pagination the Manager built — so the public pagination behavior (page
// size, total, ordering, product/type filter) is reproduced byte-identically.
// Each rich record is packed into its descriptor's opaque payload for a lossless
// round-trip; the Manager unpacks them and applies its resolver-merge + total
// math on top. A missing or wrong-typed filter is a programming error mapped to
// the Engine's validation rule rather than a silent unfiltered list.
func (s *ConnectionStore) ListPaged(
	ctx context.Context,
	_ engine.TenantContext,
	params engine.ConnectionListParams,
) (engine.ConnectionPage, error) {
	filters, ok := params.Filter.(netHTTP.QueryHeader)
	if !ok {
		return engine.ConnectionPage{}, engine.NewEngineError(engine.CategoryValidation, "connection list params are missing or malformed")
	}

	conns, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return engine.ConnectionPage{}, err
	}

	items := make([]engine.ConnectionDescriptor, 0, len(conns))
	for _, conn := range conns {
		items = append(items, DescriptorFromConnection(conn))
	}

	return engine.ConnectionPage{Items: items, Total: total}, nil
}
