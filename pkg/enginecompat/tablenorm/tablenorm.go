// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package tablenorm is the host-side canonicalizer that reconciles datasource
// table-name AND field-name conventions BEFORE they cross the embedded Engine
// boundary.
//
// It is the load-bearing half of owner decision Option 2 (T-010): the Engine core
// matches requested tables/fields against the discovered schema snapshot by LITERAL
// equality and carries no datasource-naming knowledge. The host normalizes
// identifiers per datasource type at the seam:
//
//   - PostgreSQL: strip the "public." default-schema prefix (case-sensitive names).
//   - SQL Server: strip the "dbo." default-schema prefix (case-sensitive names).
//   - Oracle: fold table AND field identifiers to UPPERCASE. This is the
//     UPPERCASE-CANONICAL contract, and it is the one that aligns the SCHEMA identity
//     to the physical DATA identity for free. Oracle's data dictionary stores
//     identifiers UPPERCASED; the extracted result rows are keyed verbatim by
//     rows.Columns() (pkg/oracle.createRowMap), i.e. by the physical UPPERCASE column
//     names, and nothing normalizes those data keys. Uppercasing the snapshot and the
//     request keys therefore makes snapshot == validation == result table keys ==
//     result column keys == physical catalog, all UPPERCASE. It also preserves the
//     signed-off result-key compatibility waiver ("Oracle identifiers are uppercased",
//     docs/compatibility-waivers.md). GetSchemaInfo internally lowercases what it
//     returns, but ToUpper is idempotent so the canonical fold re-uppercases it; the
//     adapter's ToLower is harmless and left untouched.
//   - MySQL / MongoDB: no normalization (case-sensitive, no strippable default
//     schema).
//
// PHYSICAL-CASE MAPPING DOES NOT HAPPEN HERE. The Oracle query path emits UNQUOTED
// identifiers (squirrel writes column/table names verbatim with no quoting), and
// pkg/oracle.ValidateTableAndFields resolves the requested (case-insensitive) field
// names back to the PHYSICAL column case taken from the freshly-discovered schema
// before building the SELECT. Oracle then folds the unquoted identifiers to uppercase
// itself. So an UPPERCASE canonical identity at this seam matches the physical data
// directly, and request-vs-physical reconciliation lives at the SQL-building seam in
// pkg/oracle, untouched by this package.
//
// To preserve case-insensitive matching WITHOUT teaching the Engine core about
// schema names or case rules, the host normalizes BOTH the discovered snapshot
// identifiers AND the requested identifiers to the SAME canonical form here, at the
// enginecompat seam. The Engine's literal match then behaves like the legacy
// per-adapter case-insensitive matching: prefix/case variants still pass, while
// genuinely-missing identifiers still fail validation.
//
// The default-schema prefix rule REUSES pkg/schemautil.NormalizeTableNameForLookup —
// the SAME function the legacy adapters call — so there is a single source of truth
// for it. Oracle case-folding uppercases to match the physical Oracle catalog and the
// result-key waiver.
package tablenorm

import (
	"strings"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	datasourceModel "github.com/LerianStudio/fetcher/v2/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/v2/pkg/schemautil"
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
// matching. Only Oracle does (its physical catalog is UPPERCASE and it matches
// case-insensitively); PostgreSQL, MySQL, and SQL Server match field names
// case-SENSITIVELY, so they must NOT be folded.
func FoldsFieldCase(dbType model.DBType) bool {
	return dbType == model.TypeOracle
}

// NormalizeTable canonicalizes a single table name for the given datasource type.
// It applies the legacy default-schema prefix rule (schemautil) and, for Oracle,
// folds the result to UPPERCASE — matching the physical Oracle catalog AND the
// extracted result keys — so a snapshot table and a requested table that differ only
// in case canonicalize to the same string here. The physical-case request resolution
// stays in pkg/oracle.ValidateTableAndFields.
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
// UPPERCASE for Oracle — matching the physical Oracle catalog AND the extracted
// result column keys — so the Engine's literal field membership succeeds for any-case
// Oracle request. The physical column case is resolved later in
// pkg/oracle.ValidateTableAndFields, not here.
func NormalizeField(dbType model.DBType, fieldName string) string {
	if FoldsFieldCase(dbType) {
		return strings.ToUpper(fieldName)
	}

	return fieldName
}
