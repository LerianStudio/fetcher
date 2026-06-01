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
		l.MaxResultBytes == 0
}
