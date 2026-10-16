package bootstrap

import (
	"context"
	"fmt"

	fetcher "github.com/LerianStudio/fetcher/v2"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"

	"github.com/LerianStudio/lib-auth/v5/auth/declaration"
	"github.com/LerianStudio/lib-auth/v5/auth/middleware"
	libLog "github.com/LerianStudio/lib-observability/v4/log"
)

// Access Manager permission declaration (responsibility inversion): when the
// Manager boots, it can publish the repo-root permissions.yaml to the identity
// service through lib-auth's declaration publisher, so the access manager
// learns fetcher's RBAC from fetcher instead of from its own seed.
//
// lib-auth OWNS the env contract and reads it inside WireFromEnv; none of it is
// added to Config, because a second parser here would only drift from the one
// that decides:
//
//   - IDP_DECLARATION_ENABLED — master switch. Anything other than "true" (the
//     default, unset) is a no-op: nothing else is read or validated, no goroutine
//     starts, and the boot is exactly what it was before this file existed.
//   - IDP_HOST, IDP_M2M_CLIENT_ID, IDP_M2M_CLIENT_SECRET — required when the
//     switch is on: the identity base URL and the M2M credential the publish
//     authenticates with.
//
// The token is minted with the Manager's own AuthClient (the one its routes
// authorize with), injected through WireInput.AuthClient. lib-auth therefore
// does not read PLUGIN_AUTH_HOST/PLUGIN_AUTH_ENABLED itself: the client's
// Address (PLUGIN_AUTH_ADDRESS) and Enabled (PLUGIN_AUTH_ENABLED) stand in for
// them. The identity endpoint is M2M-guarded, so with the switch on the Manager
// REQUIRES PLUGIN_AUTH_ENABLED=true and a non-empty PLUGIN_AUTH_ADDRESS.
//
// Failure policy, split by failure class:
//
//   - configuration error -> FAIL CLOSED. Switch on with a missing IDP_* value,
//     auth disabled, an empty auth address, or an embedded manifest that does
//     not parse, validate or name this slug: the boot aborts with the variable
//     named. A green pod that announced RI and declared nothing would be silent
//     policy drift; under a rolling update this stalls the rollout instead.
//   - runtime error -> FAIL OPEN. An unreachable identity host, a 5xx, a rejected
//     publish or an unmintable token is logged and the pod keeps serving:
//     whatever the access manager already holds keeps enforcing.
//
// Only the Manager publishes. The Worker serves none of the routes the manifest
// declares, so it has nothing to declare and needs no M2M credential.

// wireDeclarationFromEnv is overridable so tests can observe the input the
// Manager hands to lib-auth without reaching an identity service.
var wireDeclarationFromEnv = declaration.WireFromEnv

// declarationInput assembles the only per-service values WireFromEnv needs.
//
// The slug is constant.ApplicationName — the same string the routes pass to
// Authorize as the application name and the one permissions.yaml declares in
// its `service:` field. All three must agree: lib-auth rejects a slug that is
// not manifest.service, and a slug that disagreed with the routes would publish
// permissions nobody asks for.
func declarationInput(logger libLog.Logger, authClient *middleware.AuthClient) declaration.WireInput {
	return declaration.WireInput{
		Slug:       constant.ApplicationName,
		Manifest:   fetcher.PermissionManifest,
		Logger:     logger,
		AuthClient: authClient,
	}
}

// initPermissionDeclaration wires and starts the declaration publisher and
// returns its stop hook for the shutdown sequence. The stop hook is never nil,
// including on error, so a caller can always register or defer it.
func initPermissionDeclaration(ctx context.Context, logger libLog.Logger, authClient *middleware.AuthClient) (func(), error) {
	stop, err := wireDeclarationFromEnv(ctx, declarationInput(logger, authClient))
	if err != nil {
		return func() {}, fmt.Errorf("wire access manager permission declaration: %w", err)
	}

	if stop == nil {
		stop = func() {}
	}

	return stop, nil
}
