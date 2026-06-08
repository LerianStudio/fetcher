// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package tablenorm is the host-side canonicalizer that reconciles datasource
// table-name AND field-name conventions BEFORE they cross the embedded Engine
// boundary.
//
// It is the load-bearing half of owner decision Option 2 (T-010): the Engine core
// matches requested tables/fields against the discovered schema snapshot by LITERAL
// equality and carries no datasource-naming knowledge. The legacy Worker, by
// contrast, normalized identifiers per datasource type at query time:
//
//   - PostgreSQL: strip the "public." default-schema prefix (case-sensitive names).
//   - SQL Server: strip the "dbo." default-schema prefix (case-sensitive names).
//   - Oracle: fold table AND field identifiers to UPPERCASE — Oracle stores schema
//     identifiers uppercased and the legacy adapter matched them with EqualFold /
//     ToUpper (pkg/oracle: case-insensitive table+field matching). A lowercase or
//     mixed-case Oracle request must therefore match the uppercased snapshot.
//   - MySQL / MongoDB: no normalization (case-sensitive, no strippable default
//     schema).
//
// To preserve that byte-identical behavior WITHOUT teaching the Engine core about
// schema names or case rules, the host normalizes BOTH the discovered snapshot
// identifiers AND the requested identifiers to the SAME canonical form here, at the
// enginecompat seam. The Engine's literal match then behaves exactly like the legacy
// per-adapter normalization: prefix/case variants the legacy Worker accepted still
// pass, while genuinely-missing identifiers still fail validation.
//
// The default-schema prefix rule REUSES pkg/schemautil.NormalizeTableNameForLookup —
// the SAME function the legacy adapters call — so there is a single source of truth
// for it. Oracle case-folding mirrors the legacy oracle adapter's ToUpper.
package tablenorm

import (
	"strings"

	"github.com/LerianStudio/fetcher/pkg/model"
	datasourceModel "github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/schemautil"
)

// DefaultSchemaForType returns the default schema name used to normalize table
// identities for a datasource type, mirroring each legacy adapter's DefaultSchema:
//
//   - PostgreSQL -> "public"
//   - SQL Server -> "dbo"
//   - Oracle, MySQL, MongoDB -> "" (no default-schema stripping)
//
// An empty default schema makes the prefix-stripping step a no-op, which is correct
// for types that do not carry a strippable default schema: Oracle qualifies by
// owner, MySQL by database, and MongoDB by collection.
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

// FoldsFieldCase reports whether the datasource type folds identifier case during
// matching. Only Oracle does (it stores identifiers uppercased and the legacy
// adapter matched case-insensitively); PostgreSQL, MySQL, and SQL Server matched
// field names case-SENSITIVELY in the legacy path, so they must NOT be folded.
func FoldsFieldCase(dbType model.DBType) bool {
	return dbType == model.TypeOracle
}

// NormalizeTable canonicalizes a single table name for the given datasource type.
// It applies the legacy default-schema prefix rule (schemautil) and, for Oracle,
// folds the result to UPPERCASE — so a snapshot table and a requested table the
// legacy Worker treated as equal canonicalize to the same string here.
func NormalizeTable(dbType model.DBType, tableName string) string {
	normalized := schemautil.NormalizeTableNameForLookup(tableName, DefaultSchemaForType(dbType))

	if FoldsFieldCase(dbType) {
		return strings.ToUpper(normalized)
	}

	return normalized
}

// SchemaScopeForTables computes the schema-name scope live discovery should fetch
// for a datasource of the given type from the requested table keys. It is the
// host-seam equivalent of the Manager's schemaScopeForConfig and the legacy
// per-adapter ensureDefaultSchemaIncluded: the unique qualifying schemas are taken
// from the table keys (reusing datasource.GetUniqueSchemas — the single source of
// truth for schema parsing), and the type's default schema is appended when any
// requested table is unqualified, so a mixed qualified+unqualified request (e.g.
// "accounting.invoices" + "users") discovers BOTH "accounting" and "public".
//
// Types without a strippable default schema (Oracle/MySQL/MongoDB) get DefaultSchemaForType == ""
// and so receive no injected default, preserving their existing discovery behavior:
// only the explicitly-qualified schemas (if any) scope discovery. A request with no
// qualified tables and no default schema yields nil, which the connector treats as
// "no explicit scope" so the underlying adapter applies its own default.
func SchemaScopeForTables(dbType model.DBType, tables map[string][]string) []string {
	schemas := datasourceModel.GetUniqueSchemas(tables)

	defaultSchema := DefaultSchemaForType(dbType)
	if defaultSchema == "" {
		return schemas
	}

	if !hasUnqualifiedTable(tables) || containsSchema(schemas, defaultSchema) {
		return schemas
	}

	return append(schemas, defaultSchema)
}

// hasUnqualifiedTable reports whether any requested table key lacks a "schema."
// prefix, mirroring the legacy ensureDefaultSchema unqualified check.
func hasUnqualifiedTable(tables map[string][]string) bool {
	for tableName := range tables {
		if !strings.Contains(tableName, ".") {
			return true
		}
	}

	return false
}

// containsSchema reports whether schemas already includes target, so the default
// schema is never appended twice.
func containsSchema(schemas []string, target string) bool {
	for _, s := range schemas {
		if s == target {
			return true
		}
	}

	return false
}

// NormalizeField canonicalizes a single field/column name for the datasource type.
// It is the IDENTITY for case-sensitive types (PG/MySQL/SQLServer) and folds to
// UPPERCASE for Oracle, restoring the legacy Oracle case-insensitive field matching
// at the seam so the Engine's literal field membership succeeds.
func NormalizeField(dbType model.DBType, fieldName string) string {
	if FoldsFieldCase(dbType) {
		return strings.ToUpper(fieldName)
	}

	return fieldName
}
