// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package tablenorm is the host-side canonicalizer that reconciles datasource
// table-name conventions BEFORE they cross the embedded Engine boundary.
//
// It is the load-bearing half of owner decision Option 2 (T-010): the Engine core
// matches requested tables against the discovered schema snapshot by LITERAL
// equality and carries no datasource-naming knowledge. The legacy Worker, by
// contrast, normalized names per datasource type at query time (PostgreSQL
// "public." prefix stripping, SQL Server "dbo." stripping, Oracle case folding),
// so a request for "public.transactions" matched a snapshot keyed "transactions".
//
// To preserve that byte-identical behavior WITHOUT teaching the Engine core about
// schema names, the host normalizes BOTH the discovered snapshot table names AND
// the requested table names to the SAME canonical form here, at the enginecompat
// seam. The Engine's literal match then behaves exactly like the legacy
// per-adapter normalization: prefix/case variants the legacy Worker accepted still
// pass, while genuinely-missing tables still fail validation.
//
// The canonicalization REUSES pkg/schemautil.NormalizeTableNameForLookup — the
// SAME function the legacy adapters call — so there is a single source of truth for
// the rule and no second, divergent implementation. This package only adds the
// datasource-type -> default-schema mapping the rule needs.
package tablenorm

import (
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/schemautil"
)

// DefaultSchemaForType returns the default schema name used to normalize table
// identities for a datasource type, mirroring each legacy adapter's DefaultSchema:
//
//   - PostgreSQL -> "public"
//   - SQL Server -> "dbo"
//   - Oracle, MySQL, MongoDB -> "" (no default-schema stripping)
//
// An empty default schema makes NormalizeTable a no-op for prefix stripping, which
// is the correct behavior for types that do not carry a strippable default schema:
// Oracle qualifies by owner, MySQL by database, and MongoDB by collection. Those
// types relied on no public-prefix normalization in the legacy path, so leaving the
// name untouched preserves their behavior exactly.
func DefaultSchemaForType(dbType model.DBType) string {
	switch dbType {
	case model.TypePostgreSQL:
		return schemautil.DefaultSchemaPostgreSQL
	case model.TypeSQLServer:
		return schemautil.DefaultSchemaSQLServer
	default:
		return ""
	}
}

// NormalizeTable canonicalizes a single table name for the given datasource type
// using the legacy lookup rule. It is the exact rule the legacy adapters applied
// (schemautil.NormalizeTableNameForLookup with the type's default schema), so a
// snapshot table and a requested table that the legacy Worker treated as equal
// canonicalize to the same string here.
func NormalizeTable(dbType model.DBType, tableName string) string {
	return schemautil.NormalizeTableNameForLookup(tableName, DefaultSchemaForType(dbType))
}
