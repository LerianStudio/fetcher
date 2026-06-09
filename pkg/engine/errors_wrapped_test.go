// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
)

// These tests pin the SECRET-HIDING contract of NewWrappedEngineError / Unwrap:
// a wrapped engine error renders only the safe "engine: [category] message"
// string via Error(), NEVER the secret-bearing cause — yet the cause stays
// reachable through errors.Unwrap / errors.As / errors.Is so a host can still
// recognize a typed error it raised (e.g. a host-safety validation rejection)
// for transport mapping.

// secretSentinel is a string that must NEVER appear in any rendered Error()
// string. It models a DSN / credential / driver-internal a cause might carry.
const secretSentinel = "postgres://user:SUPERSECRETPASSWORD@db.internal:5432/prod"

// sentinelCause is a typed error carrying the secret, so the test can prove BOTH
// that the secret is hidden by Error() AND that the typed cause is recoverable
// via errors.As (the host's recognition path).
type sentinelCause struct {
	secret string
}

func (e *sentinelCause) Error() string { return "connect failed: " + e.secret }

func TestNewWrappedEngineError_HidesCauseFromErrorString(t *testing.T) {
	t.Parallel()

	cause := &sentinelCause{secret: secretSentinel}
	err := engine.NewWrappedEngineError(engine.CategoryConnect, "failed to connect to datasource", cause)

	rendered := err.Error()

	// The public boundary string is ONLY the safe category + message.
	if rendered != "engine: [connect] failed to connect to datasource" {
		t.Fatalf("unexpected boundary string: %q", rendered)
	}

	if strings.Contains(rendered, secretSentinel) {
		t.Fatalf("Error() leaked the secret cause: %q", rendered)
	}

	if strings.Contains(rendered, "SUPERSECRETPASSWORD") {
		t.Fatalf("Error() leaked a credential fragment: %q", rendered)
	}
}

func TestNewWrappedEngineError_CauseReachableViaUnwrap(t *testing.T) {
	t.Parallel()

	cause := &sentinelCause{secret: secretSentinel}
	err := engine.NewWrappedEngineError(engine.CategoryConnect, "failed to connect to datasource", cause)

	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		t.Fatal("errors.Unwrap returned nil; the cause must stay reachable")
	}

	if unwrapped != error(cause) {
		t.Fatalf("errors.Unwrap did not return the original cause: got %v", unwrapped)
	}
}

func TestNewWrappedEngineError_CauseReachableViaErrorsAs(t *testing.T) {
	t.Parallel()

	cause := &sentinelCause{secret: secretSentinel}
	err := engine.NewWrappedEngineError(engine.CategoryConnect, "failed to connect to datasource", cause)

	var target *sentinelCause
	if !errors.As(err, &target) {
		t.Fatal("errors.As could not recover the typed cause through the wrapped engine error")
	}

	if target.secret != secretSentinel {
		t.Fatalf("recovered cause lost its payload: got %q", target.secret)
	}

	// The host CAN see the secret by deliberately unwrapping to the typed cause —
	// that is the recognition seam. What matters is the engine never renders it by
	// default; the host owns whether to inspect deeper.
	if !strings.Contains(target.Error(), secretSentinel) {
		t.Fatalf("the recovered cause should still carry its own (secret) text: %q", target.Error())
	}
}

func TestNewWrappedEngineError_CauseReachableViaErrorsIs(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("host-safety: blocked private host " + secretSentinel)
	err := engine.NewWrappedEngineError(engine.CategoryValidation, "datasource host is not permitted", sentinel)

	if !errors.Is(err, sentinel) {
		t.Fatal("errors.Is could not match the wrapped sentinel cause")
	}

	if strings.Contains(err.Error(), secretSentinel) {
		t.Fatalf("Error() leaked the sentinel's secret: %q", err.Error())
	}
}

func TestNewEngineError_HasNoCauseToUnwrap(t *testing.T) {
	t.Parallel()

	// A plain (non-wrapped) engine error carries no cause: Unwrap is nil and the
	// secret-hiding contract holds trivially.
	err := engine.NewEngineError(engine.CategoryInternal, "failed to serialize extraction result")

	if errors.Unwrap(err) != nil {
		t.Fatal("a plain NewEngineError must have no cause to unwrap")
	}
}
