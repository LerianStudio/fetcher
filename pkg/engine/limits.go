// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "time"

// Stable default Engine limits. These values are part of the public contract;
// changing any of them is a breaking change and must update the contract test
// that pins them. They mirror the historical Manager/Worker bounds so the
// embedded Engine preserves behavior during the strangler migration.
const (
	// DefaultMaxDatasources bounds the number of datasources per extraction.
	DefaultMaxDatasources = 10
	// DefaultMaxTablesPerDatasource bounds tables/collections per datasource.
	DefaultMaxTablesPerDatasource = 20
	// DefaultMaxFieldsPerTable bounds selected fields per table.
	DefaultMaxFieldsPerTable = 50
	// DefaultMaxConcurrency bounds parallel datasource extraction workers.
	DefaultMaxConcurrency = 4
	// DefaultTimeout bounds a single extraction operation.
	DefaultTimeout = 5 * time.Minute
	// DefaultMaxResultBytes bounds the serialized result size (256 MiB).
	DefaultMaxResultBytes int64 = 256 * 1024 * 1024
)

// Limits describes the resource bounds an Engine operation must respect.
// The zero value is intentionally not a usable configuration — callers obtain
// safe defaults via DefaultLimits and may override individual fields.
//
// Limits is NOT comparable with `==`: it carries a map field
// (ConnectorHardLimits), so comparing two Limits values with `==` is a compile
// error. Use reflect.DeepEqual (or field-wise comparison) to compare them.
type Limits struct {
	// MaxDatasources bounds datasources per extraction request.
	MaxDatasources int
	// MaxTablesPerDatasource bounds tables/collections per datasource.
	MaxTablesPerDatasource int
	// MaxFieldsPerTable bounds selected fields per table.
	MaxFieldsPerTable int
	// MaxConcurrency bounds parallel datasource workers.
	MaxConcurrency int
	// Timeout bounds a single extraction operation.
	Timeout time.Duration
	// MaxResultBytes bounds the serialized result size in bytes.
	MaxResultBytes int64
	// ConnectorHardLimits carries connector-type-specific hard bounds (keyed by
	// datasource type, e.g. a max-rows or max-batch ceiling) for the HOST adapter
	// to enforce at execute time. It is a documented contract the core carries but
	// does NOT interpret: keeping it driver-free here is what lets pkg/engine stay
	// free of any concrete driver import. The planner copies it into the plan so
	// the host can read it without re-deriving it.
	ConnectorHardLimits map[string]int
}

// limitField is a stable, safe identifier for a single limit dimension. It is
// used as the field path in override-rejection validation errors so a caller can
// see exactly which limit it breached — never a raw value that could leak data.
type limitField string

const (
	limitFieldMaxDatasources limitField = "limits.maxDatasources"
	limitFieldMaxTables      limitField = "limits.maxTablesPerDatasource"
	limitFieldMaxFields      limitField = "limits.maxFieldsPerTable"
	limitFieldMaxConcurrency limitField = "limits.maxConcurrency"
	limitFieldTimeout        limitField = "limits.timeout"
	limitFieldMaxResultBytes limitField = "limits.maxResultBytes"
)

// Effective resolves the limits in force for a single request: it starts from
// the receiver (the Engine default), applies the allowed per-request overrides,
// and returns a NEW Limits value. It NEVER mutates the receiver or the override
// argument — the receiver is the Engine's shared default, so mutating it would
// leak one request's bounds into every later request.
//
// Merge semantics, per field: a positive override value replaces the default; a
// zero (or negative) override field inherits the default. An override that
// exceeds the Engine maximum (the default value) is REJECTED with a
// CategoryValidation EngineError whose message names the violated limit field —
// the Engine default is the ceiling a request may not raise. ConnectorHardLimits
// are not request-overridable; they are copied from the default unchanged.
//
// A nil override returns a copy of the default with no validation work.
func (l Limits) Effective(overrides *Limits) (Limits, error) {
	effective := l
	effective.ConnectorHardLimits = copyConnectorHardLimits(l.ConnectorHardLimits)

	if overrides == nil {
		return effective, nil
	}

	if err := applyIntOverride(&effective.MaxDatasources, overrides.MaxDatasources, l.MaxDatasources, limitFieldMaxDatasources); err != nil {
		return Limits{}, err
	}

	if err := applyIntOverride(&effective.MaxTablesPerDatasource, overrides.MaxTablesPerDatasource, l.MaxTablesPerDatasource, limitFieldMaxTables); err != nil {
		return Limits{}, err
	}

	if err := applyIntOverride(&effective.MaxFieldsPerTable, overrides.MaxFieldsPerTable, l.MaxFieldsPerTable, limitFieldMaxFields); err != nil {
		return Limits{}, err
	}

	if err := applyIntOverride(&effective.MaxConcurrency, overrides.MaxConcurrency, l.MaxConcurrency, limitFieldMaxConcurrency); err != nil {
		return Limits{}, err
	}

	if err := applyDurationOverride(&effective.Timeout, overrides.Timeout, l.Timeout); err != nil {
		return Limits{}, err
	}

	if err := applyResultBytesOverride(&effective.MaxResultBytes, overrides.MaxResultBytes, l.MaxResultBytes); err != nil {
		return Limits{}, err
	}

	return effective, nil
}

// applyIntOverride applies a single integer override onto target: a positive
// requested value within the maximum replaces it, a non-positive request inherits
// the default (no-op), and a request above the maximum is rejected with a safe
// field-pathed error. maximum 0 means "unbounded default": any positive override
// is accepted because there is no ceiling to breach.
func applyIntOverride(target *int, requested, maximum int, field limitField) error {
	if requested <= 0 {
		return nil
	}

	if maximum > 0 && requested > maximum {
		return overrideExceededError(field)
	}

	*target = requested

	return nil
}

// applyDurationOverride applies a timeout override: a positive request within the
// maximum replaces it, a non-positive request inherits the default, and a request
// above the maximum is rejected.
func applyDurationOverride(target *time.Duration, requested, maximum time.Duration) error {
	if requested <= 0 {
		return nil
	}

	if maximum > 0 && requested > maximum {
		return overrideExceededError(limitFieldTimeout)
	}

	*target = requested

	return nil
}

// applyResultBytesOverride applies a result-size override: a positive request
// within the maximum replaces it, a non-positive request inherits the default,
// and a request above the maximum is rejected.
func applyResultBytesOverride(target *int64, requested, maximum int64) error {
	if requested <= 0 {
		return nil
	}

	if maximum > 0 && requested > maximum {
		return overrideExceededError(limitFieldMaxResultBytes)
	}

	*target = requested

	return nil
}

// overrideExceededError builds the canonical override-rejection validation error.
// The message names the violated limit field path and never echoes the offending
// value, so the error stays safe and stable.
func overrideExceededError(field limitField) error {
	return NewEngineError(CategoryValidation, "limit override exceeds the configured maximum: "+string(field))
}

// copyConnectorHardLimits returns an independent copy so the plan's limits cannot
// be mutated through the Engine's shared default map (or vice versa). It returns
// nil for an empty source.
func copyConnectorHardLimits(src map[string]int) map[string]int {
	if len(src) == 0 {
		return nil
	}

	out := make(map[string]int, len(src))
	for k, v := range src {
		out[k] = v
	}

	return out
}

// DefaultLimits returns the stable default Engine limits.
func DefaultLimits() Limits {
	return Limits{
		MaxDatasources:         DefaultMaxDatasources,
		MaxTablesPerDatasource: DefaultMaxTablesPerDatasource,
		MaxFieldsPerTable:      DefaultMaxFieldsPerTable,
		MaxConcurrency:         DefaultMaxConcurrency,
		Timeout:                DefaultTimeout,
		MaxResultBytes:         DefaultMaxResultBytes,
	}
}

// IsZero reports whether the limits carry no configured bounds. A zero Limits
// must never be used to authorize an operation; callers should fall back to
// DefaultLimits instead.
func (l Limits) IsZero() bool {
	return l.MaxDatasources == 0 &&
		l.MaxTablesPerDatasource == 0 &&
		l.MaxFieldsPerTable == 0 &&
		l.MaxConcurrency == 0 &&
		l.Timeout == 0 &&
		l.MaxResultBytes == 0 &&
		len(l.ConnectorHardLimits) == 0
}
