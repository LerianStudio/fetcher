package command

import (
	"context"
	"errors"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/model"
)

// authorizeConnectionAccess routes the per-request tenant-scope authority
// decision through the Engine for connection mutations that keep their own
// (UUID-keyed) persistence. The tenant is derived fresh from this request's
// context (single-tenant falls back to connectioncompat.SingleTenantID), never
// ambient or cached. A nil engine (test-only construction) is a no-op so the
// operation proceeds, mirroring the conflict gate's defensive guard.
func authorizeConnectionAccess(ctx context.Context, eng *engine.Engine) error {
	if eng == nil {
		return nil
	}

	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	if err != nil {
		return err
	}

	return eng.AuthorizeConnectionAccess(ctx, tenant)
}

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
