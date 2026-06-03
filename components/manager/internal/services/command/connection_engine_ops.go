package command

import (
	"context"
	"errors"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/model"

	"github.com/google/uuid"
)

// engineInputFromConnection builds the Engine's credential-bearing connection
// input from the Manager's rich record, packing the FULL record into the opaque
// host payload (connectioncompat.DescriptorFromConnection) so the Engine carries
// ProductName / full SSL / uuid identity / metadata / timestamps verbatim
// through its (tenantID, configName)-scoped ConnectionStore without those host
// fields becoming Engine scoping dimensions. The Engine reads only ConfigName
// for scoping; everything else rides as opaque bytes.
func engineInputFromConnection(conn *model.Connection) engine.ConnectionInput {
	descriptor := connectioncompat.DescriptorFromConnection(conn)

	return engine.NewConnectionInput(engine.ConnectionInputParams{
		ConfigName:     conn.ConfigName,
		Type:           string(conn.Type),
		Host:           conn.Host,
		Port:           conn.Port,
		DatabaseName:   conn.DatabaseName,
		Schema:         descriptor.Schema,
		Username:       conn.Username,
		SSLMode:        descriptor.SSLMode,
		HostAttributes: descriptor.HostAttributes,
	})
}

// updateConnectionByIDViaEngine routes the connection write through the Engine's
// ID-addressed op (Engine tenant-scope + ConnectionStore.UpdateByID ->
// repo.Update). The Manager applies its own domain patch (cryptor re-encryption)
// BEFORE this call and packs the patched rich record into the descriptor's
// opaque host payload, so the Engine persists the host-authored record without
// interpreting it and the UUID identity is preserved.
func updateConnectionByIDViaEngine(ctx context.Context, eng *engine.Engine, connectionID uuid.UUID, patched *model.Connection) (*model.Connection, error) {
	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	descriptor := connectioncompat.DescriptorFromConnection(patched)

	updated, err := eng.UpdateConnectionByID(ctx, tenant, connectionID.String(), descriptor, engine.ConnectionPatch{})
	if err != nil {
		var engErr *engine.EngineError
		if errors.As(err, &engErr) && engErr.Category == engine.CategoryNotFound {
			// The record vanished between read and write (e.g. soft-deleted via a
			// concurrent delete). Return (nil, nil) so the caller surfaces the
			// Manager's existing ErrEntityNotFound business error — byte-identical to
			// the pre-delegation "repo.Update returned nil" contract.
			return nil, nil
		}

		return nil, err
	}

	return connectioncompat.ConnectionFromDescriptor(updated), nil
}

// deleteConnectionByIDViaEngine routes the SOFT delete through the Engine's
// ID-addressed op (Engine tenant-scope + ConnectionStore.DeleteByID ->
// repo.Delete(soft)). The Manager keeps its conflict gate and HTTP mapping; the
// Engine owns the scope authority and the delete persistence.
func deleteConnectionByIDViaEngine(ctx context.Context, eng *engine.Engine, connectionID uuid.UUID) error {
	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	if err != nil {
		return err
	}

	return eng.DeleteConnectionByID(ctx, tenant, connectionID.String())
}

// mapEngineCreateError translates an Engine create failure into the Manager's
// existing business-error HTTP contract. A duplicate (tenantID, configName) is
// the Engine's CategoryConflict, which the Manager surfaces as ErrEntityConflict
// (HTTP 409) — byte-identical to the pre-delegation behavior. Any other Engine
// error is returned as-is for the caller's generic wrapping.
func mapEngineCreateError(err error) error {
	var engErr *engine.EngineError
	if errors.As(err, &engErr) && engErr.Category == engine.CategoryConflict {
		return pkg.ValidateBusinessError(constant.ErrEntityConflict, "connection")
	}

	return err
}
