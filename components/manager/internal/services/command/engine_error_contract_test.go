package command

import (
	"errors"
	"net/http"
	"testing"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/engine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file locks the Engine-conflict -> Manager-409 contract for the migrated
// mutation paths (ST-T008-03). The conflict category is the one public 4xx the
// mutation ops can produce from the Engine: a duplicate (tenantID, configName)
// on create, and active work referencing a connection on delete/update. Both
// MUST surface as the Manager's existing HTTP 409 business errors. Any other
// Engine category must fall through unmapped so the renderer applies its safe
// generic 500 — never a misclassified 4xx.

func conflictStatus(t *testing.T, err error) (int, string) {
	t.Helper()
	require.Error(t, err)

	var responseWithStatus pkg.ResponseErrorWithStatusCode
	if errors.As(err, &responseWithStatus) {
		return responseWithStatus.StatusCode, responseWithStatus.Code
	}

	internal := pkg.ValidateInternalError(err, "")
	if ise, ok := internal.(pkg.InternalServerError); ok {
		return http.StatusInternalServerError, ise.Code
	}

	return http.StatusInternalServerError, ""
}

// TestEngineErrorContract_CreateConflict_Maps409 proves a duplicate-config-name
// create (engine.CategoryConflict) becomes ErrEntityConflict (HTTP 409),
// byte-identical to the pre-delegation unique-key conflict.
func TestEngineErrorContract_CreateConflict_Maps409(t *testing.T) {
	mapped := mapEngineCreateError(engine.NewEngineError(engine.CategoryConflict, "connection already exists"))
	status, code := conflictStatus(t, mapped)
	assert.Equal(t, http.StatusConflict, status, "create conflict must map to 409")
	assert.Equal(t, constant.ErrEntityConflict.Error(), code, "must preserve the ErrEntityConflict machine code")
}

// TestEngineErrorContract_ActiveJobConflict_Maps409 proves an active-execution
// conflict (engine.CategoryConflict from the Engine's execution gate) becomes
// ErrJobInProgress (HTTP 409), preserving the pre-delegation active-job block.
func TestEngineErrorContract_ActiveJobConflict_Maps409(t *testing.T) {
	mapped := asActiveJobConflict(engine.NewEngineError(engine.CategoryConflict, "connection has active executions"))
	require.NotNil(t, mapped, "an active-execution conflict must be mapped, not passed through")
	status, code := conflictStatus(t, mapped)
	assert.Equal(t, http.StatusConflict, status, "active-job conflict must map to 409")
	assert.Equal(t, constant.ErrJobInProgress.Error(), code, "must preserve the ErrJobInProgress machine code")
}

// TestEngineErrorContract_NonConflictCategoriesFallThrough proves the conflict
// mappers translate ONLY engine.CategoryConflict. Every other category — and a
// plain error — must NOT be coerced into a 409: the create mapper returns the
// input unchanged and the active-job mapper returns nil, so the caller's generic
// handling (safe 500) applies. This stops a future Engine change from silently
// laundering an internal fault into a misleading 409.
func TestEngineErrorContract_NonConflictCategoriesFallThrough(t *testing.T) {
	others := []engine.ErrorCategory{
		engine.CategoryValidation,
		engine.CategoryNotFound,
		engine.CategoryUnavailable,
		engine.CategoryTimeout,
		engine.CategoryCanceled,
		engine.CategoryUnauthorized,
		engine.CategoryForbidden,
		engine.CategoryLimitExceeded,
		engine.CategoryInternal,
	}

	for _, c := range others {
		c := c
		t.Run(string(c), func(t *testing.T) {
			in := engine.NewEngineError(c, "internal")

			out := mapEngineCreateError(in)
			assert.Equal(t, error(in), out, "create mapper must pass non-conflict category %q through unchanged", c)

			assert.Nil(t, asActiveJobConflict(in), "active-job mapper must return nil for non-conflict category %q", c)
		})
	}

	// A plain (non-Engine) error must also pass through both mappers.
	plain := errors.New("boom")
	assert.Equal(t, plain, mapEngineCreateError(plain), "create mapper must pass plain errors through")
	assert.Nil(t, asActiveJobConflict(plain), "active-job mapper must return nil for plain errors")
}
