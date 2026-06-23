package command

import (
	"context"
	"errors"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
)

// checkActiveJobsViaEngine delegates the "block mutation while jobs run against
// the connection" gate to the Engine, scoped by the request's tenant. The
// Engine consults the host's ActiveExecutionChecker (backed by the Manager job
// repository) under the derived tenant, so the decision is the SHARED policy
// the Engine owns while persistence and HTTP mapping stay in the Manager.
//
// The Engine tenant is derived per request from the tmcore tenant id in ctx
// (single-tenant falls back to connectioncompat.SingleTenantID). It is read
// fresh from this request's context every call, never cached, so the gate is
// always scoped to the request's actual tenant — no ambient or global tenant.
//
// A nil engine means the service was constructed without delegation; callers
// that pass a nil engine get a no-op gate. In the assembled Manager the engine
// is always present, so this only relaxes test-only constructions.
func checkActiveJobsViaEngine(ctx context.Context, eng *engine.Engine, configName string) error {
	if eng == nil {
		return nil
	}

	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	if err != nil {
		return err
	}

	return eng.CheckActiveExecutions(ctx, tenant, configName)
}

// asActiveJobConflict maps an Engine active-execution conflict to the Manager's
// existing business-error HTTP behavior (409 via ErrJobInProgress). It returns
// nil for any non-conflict error so the caller can fall through to its generic
// error handling, preserving the pre-delegation contract (including errors.Is
// on the underlying repository error, which CheckActiveExecutions wraps rather
// than replaces).
func asActiveJobConflict(err error, args ...any) error {
	var engErr *engine.EngineError
	if errors.As(err, &engErr) && engErr.Category == engine.CategoryConflict {
		return pkg.ValidateBusinessError(constant.ErrJobInProgress, "connection", args...)
	}

	return nil
}
