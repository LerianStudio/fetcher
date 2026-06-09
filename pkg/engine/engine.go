// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "reflect"

// Engine is the embedded runtime facade. A host constructs one Engine with the
// capability ports it provides and then drives extraction operations through it
// in later subtasks. ST-T002-02 implements CONSTRUCTOR behavior only: New
// validates required capabilities and applies safe defaults. No connect,
// discover, plan, or execute logic exists yet — that arrives in subsequent
// tasks.
//
// The fields are unexported so an Engine can only originate from New, which
// guarantees the validated, defaulted invariants hold for the lifetime of the
// instance.
type Engine struct {
	options Options
}

// New constructs an Engine from the supplied options. It validates required
// capabilities AT CONSTRUCTION and returns a stable *EngineError instead of
// deferring failures to nil-pointer panics at operation time.
//
// Validation rules:
//   - ConnectorRegistry is REQUIRED; a missing or nil registry yields a
//     CategoryValidation error.
//   - When encrypted persistence is enabled, CredentialProtector is REQUIRED;
//     a missing or nil protector yields a CategoryValidation error.
//
// Safe defaults:
//   - When no limits are supplied (or a zero-value Limits is given), New
//     substitutes DefaultLimits so the Engine never runs unbounded.
//
// Optional ports (connection store, execution store, result sink, schema cache,
// active-execution checker, observability) are genuinely optional: New succeeds
// without them.
func New(opts ...Option) (*Engine, error) {
	options := Options{}

	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	if isNilPort(options.connectorRegistry) {
		return nil, NewEngineError(
			CategoryValidation,
			"connector registry is required",
		)
	}

	if options.encryptedPersistence && isNilPort(options.credentialProtector) {
		return nil, NewEngineError(
			CategoryValidation,
			"credential protector is required when encrypted persistence is enabled",
		)
	}

	if options.limits.IsZero() {
		options.limits = DefaultLimits()
	}

	return &Engine{options: options}, nil
}

// Limits returns the effective resource limits the Engine was constructed with.
func (e *Engine) Limits() Limits {
	return e.options.limits
}

// isNilPort reports whether v is an absent capability port. It catches both a
// literal nil interface and a TYPED nil — an interface value that wraps a nil
// pointer (or nil map/chan/func/slice). The latter slips past a plain `== nil`
// check, so without this New would accept a non-nil interface backed by a nil
// pointer and only fail with a nil-pointer panic at first use, defeating the
// constructor's "validate required capabilities at construction" guarantee.
//
// reflect.Interface is intentionally absent from the Kind switch: a value passed
// through the `any` parameter is always concrete, so its Kind is never Interface,
// and the literal-nil case is already covered by the v == nil guard above.
func isNilPort(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.Func, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
