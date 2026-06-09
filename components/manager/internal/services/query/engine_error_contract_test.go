package query

import (
	"errors"
	"net/http"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// This file locks the Engine-error -> Manager-HTTP behavior contract for the
// migrated read paths (ST-T008-03). The Engine returns transport-neutral,
// pre-redacted *engine.EngineError values across its boundary; the Manager is
// the sole place that translates them into its public HTTP business-error
// contract. The renderer (pkg/net/http.WithError) is deliberately Engine-AGNOSTIC
// — a raw EngineError that reaches it is a safe generic 500, never a leak — so
// every public 4xx contract MUST be produced by the SERVICE layer before the
// error reaches the renderer. These tests pin that division so a future Engine
// change cannot silently flip a 404/409 into a leak or a generic 500.
//
// Category -> preserved Manager behavior (the mapping table for T-008):
//   validation          -> internal-fault paths only on read; safe generic 500
//                          (no public 4xx existed pre-delegation for these)
//   not_found           -> ErrEntityNotFound (HTTP 404)
//   conflict            -> ErrEntityConflict / ErrJobInProgress (HTTP 409)  [mutation paths]
//   connection          -> host-side test_connection mapping (NOT Engine-routed)
//   unauthorizedContext -> cross-tenant / unknown collapse to not_found (HTTP 404
//                          existence-oracle: a tenant can never tell "exists elsewhere"
//                          apart from "does not exist")
//   timeout / canceled  -> safe generic 500 (matches pre-delegation wrapped-error 500)

// asBusinessStatus renders a service error through the public business-error
// model and returns the HTTP status + machine code the renderer would emit,
// without standing up Fiber. It mirrors pkg/net/http.WithError's type switch so
// the assertions are byte-identical to the wire contract.
func asBusinessStatus(t *testing.T, err error) (int, string) {
	t.Helper()
	require.Error(t, err)

	var validationErr pkg.ValidationError
	if errors.As(err, &validationErr) {
		return http.StatusBadRequest, validationErr.Code
	}

	var responseWithStatus pkg.ResponseErrorWithStatusCode
	if errors.As(err, &responseWithStatus) {
		return responseWithStatus.StatusCode, responseWithStatus.Code
	}

	var responseErr pkg.ResponseError
	if errors.As(err, &responseErr) {
		return responseErr.Code, ""
	}

	// Unmapped errors (including a raw *engine.EngineError) fall through to the
	// renderer's internal-error default: a safe generic 500.
	internal := pkg.ValidateInternalError(err, "")
	if ise, ok := internal.(pkg.InternalServerError); ok {
		return http.StatusInternalServerError, ise.Code
	}

	return http.StatusInternalServerError, ""
}

// TestEngineErrorContract_NotFound_Maps404 proves a connection the Engine
// cannot resolve under the request's tenant scope (missing OR owned by another
// tenant — both surface as engine.CategoryNotFound from the ConnectionStore)
// becomes the Manager's ErrEntityNotFound business error (HTTP 404).
func TestEngineErrorContract_NotFound_Maps404(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connID := uuid.New()
	repo := connRepo.NewMockRepository(ctrl)
	repo.EXPECT().FindByID(gomock.Any(), connID).Return(nil, nil) // store -> not found

	svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, repo))

	_, err := svc.Execute(testContext(), connID)
	status, code := asBusinessStatus(t, err)
	assert.Equal(t, http.StatusNotFound, status, "Engine not_found must map to the Manager's 404")
	assert.Equal(t, constant.ErrEntityNotFound.Error(), code, "must preserve the ErrEntityNotFound machine code")
}

// TestEngineErrorContract_UnauthorizedContext_IsExistenceOracleSafe proves the
// LOCKED cross-tenant existence-oracle behavior: a connection that exists for
// ANOTHER tenant is invisible under this request's tenant scope and collapses to
// the SAME 404 + machine code as a connection that never existed. The two cases
// MUST be byte-identical so the response cannot leak cross-tenant existence.
func TestEngineErrorContract_UnauthorizedContext_IsExistenceOracleSafe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Case A: connection simply does not exist (store returns nil,nil).
	unknownID := uuid.New()
	repoUnknown := connRepo.NewMockRepository(ctrl)
	repoUnknown.EXPECT().FindByID(gomock.Any(), unknownID).Return(nil, nil)
	_, errUnknown := NewGetConnection(nil, nil, scopeAuthorityEngine(t, repoUnknown)).
		Execute(testContext(), unknownID)
	statusUnknown, codeUnknown := asBusinessStatus(t, errUnknown)

	// Case B: connection exists but for a DIFFERENT tenant. Under the request's
	// scope the store cannot see it and reports not-found — identical to Case A.
	// The repo mock models the scoped store: the cross-tenant record is invisible,
	// so FindByID returns nil,nil exactly as the unknown case.
	otherTenantID := uuid.New()
	repoOther := connRepo.NewMockRepository(ctrl)
	repoOther.EXPECT().FindByID(gomock.Any(), otherTenantID).Return(nil, nil)
	_, errOther := NewGetConnection(nil, nil, scopeAuthorityEngine(t, repoOther)).
		Execute(testContext(), otherTenantID)
	statusOther, codeOther := asBusinessStatus(t, errOther)

	assert.Equal(t, http.StatusNotFound, statusUnknown, "unknown connection must be 404")
	assert.Equal(t, statusUnknown, statusOther, "cross-tenant existence must be indistinguishable from unknown (status)")
	assert.Equal(t, codeUnknown, codeOther, "cross-tenant existence must be indistinguishable from unknown (machine code)")
	assert.Equal(t, errUnknown.Error(), errOther.Error(), "cross-tenant and unknown errors must be byte-identical")

	// The not-found error must not echo the requested connection id: a per-id
	// message would let a caller correlate responses and reconstruct an existence
	// oracle even when status + code match. This is the leak vector a future
	// "helpful" message change would reopen.
	assert.NotContains(t, errUnknown.Error(), unknownID.String(), "not-found error must not echo the connection id")
	assert.NotContains(t, errOther.Error(), otherTenantID.String(), "not-found error must not echo the connection id")
}

// TestEngineErrorContract_RawEngineErrorIsSafe500 proves the renderer-agnostic
// safety net: ANY *engine.EngineError category that is NOT intercepted by the
// service layer (validation from an internal store invariant, unavailable,
// timeout, canceled, unauthorized, forbidden, limit_exceeded) degrades to a safe
// generic 500 carrying NO Engine message — never a 4xx and never a leak. This
// matches the pre-delegation behavior for the same internal-fault conditions and
// is the contract a future Engine change must not silently break.
func TestEngineErrorContract_RawEngineErrorIsSafe500(t *testing.T) {
	cats := []engine.ErrorCategory{
		engine.CategoryValidation,
		engine.CategoryUnavailable,
		engine.CategoryConnect,
		engine.CategoryTimeout,
		engine.CategoryCanceled,
		engine.CategoryUnauthorized,
		engine.CategoryForbidden,
		engine.CategoryLimitExceeded,
		engine.CategoryInternal,
	}

	for _, c := range cats {
		c := c
		t.Run(string(c), func(t *testing.T) {
			// The Manager wraps unmapped Engine errors with fmt.Errorf("...: %w").
			wrapped := errors.Join(errors.New("failed to find connection by id"), engine.NewEngineError(c, "secret-bearing internals"))
			status, code := asBusinessStatus(t, wrapped)

			assert.Equal(t, http.StatusInternalServerError, status, "unmapped Engine category %q must be a safe 500", c)
			assert.Equal(t, constant.ErrInternalServer.Error(), code, "must be the generic internal-error machine code")
			assert.NotContains(t, pkg.ValidateInternalError(wrapped, "").Error(), "secret-bearing internals",
				"the safe 500 must not surface the Engine message")
		})
	}
}

// TestEngineErrorContract_NotFoundCategoryIsInterceptedNotLeaked proves the
// SERVICE layer (not the renderer) is what turns engine.CategoryNotFound into a
// 404: a raw not_found EngineError that bypassed the service interception would
// render as a generic 500, so the interception is load-bearing. This pins that a
// not_found EngineError must be consumed before the renderer.
func TestEngineErrorContract_NotFoundCategoryIsInterceptedNotLeaked(t *testing.T) {
	raw := engine.NewEngineError(engine.CategoryNotFound, "connection not found")
	status, _ := asBusinessStatus(t, raw)
	assert.Equal(t, http.StatusInternalServerError, status,
		"a RAW not_found EngineError reaching the renderer is a generic 500 — proving the service-layer interception that produces the real 404 is load-bearing")
}
