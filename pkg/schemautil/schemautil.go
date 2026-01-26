// Package schemautil provides utility functions for handling database schema operations.
package schemautil

import (
	"errors"
	"strings"
)

// Errors returned by ParseQualifiedTableName.
var (
	ErrEmptyTableName   = errors.New("table name cannot be empty")
	ErrInvalidTableName = errors.New("invalid table name format")
)

// ParseQualifiedTableName splits a qualified table name into schema and table parts.
// If no schema is present (no dot separator), it returns the defaultSchema and the original name.
// Handles SQL Server 3-part names (catalog.schema.table) by extracting schema and table.
// Examples:
//   - "accounting.invoices" -> ("accounting", "invoices", nil)
//   - "invoices" with defaultSchema="public" -> ("public", "invoices", nil)
//   - "dbo.users" -> ("dbo", "users", nil)
//   - "mydb.dbo.users" -> ("dbo", "users", nil) // 3-part SQL Server name
//   - "" -> ("", "", ErrEmptyTableName)
//   - "a.b.c.d" -> ("", "", ErrInvalidTableName)
func ParseQualifiedTableName(qualifiedName, defaultSchema string) (schema, table string, err error) {
	trimmed := strings.TrimSpace(qualifiedName)
	if trimmed == "" {
		return "", "", ErrEmptyTableName
	}

	parts := strings.Split(trimmed, ".")
	switch len(parts) {
	case 1:
		// Simple table name
		return defaultSchema, parts[0], nil
	case 2:
		// schema.table
		return parts[0], parts[1], nil
	case 3:
		// catalog.schema.table (SQL Server full qualification)
		return parts[1], parts[2], nil
	default:
		// More than 3 parts is invalid
		return "", "", ErrInvalidTableName
	}
}

// NormalizeTableNameForLookup normalizes a table name for schema lookup.
// If the table name includes the default schema prefix, it removes it.
// For non-default schemas, it preserves the qualified name.
// Examples with defaultSchema="public":
//   - "public.users" -> "users" (default schema stripped)
//   - "accounting.invoices" -> "accounting.invoices" (non-default preserved)
//   - "users" -> "users" (no change)
func NormalizeTableNameForLookup(tableName, defaultSchema string) string {
	schemaName, simpleTable, err := ParseQualifiedTableName(tableName, defaultSchema)
	if err != nil {
		return tableName
	}

	if schemaName == "" {
		return simpleTable
	}

	if defaultSchema != "" && strings.EqualFold(schemaName, defaultSchema) {
		return simpleTable
	}

	return schemaName + "." + simpleTable
}

// QualifyTableName returns a schema-qualified table name.
// If the schema is the default schema, it returns just the table name.
// Otherwise, it returns "schema.table".
func QualifyTableName(schema, table, defaultSchema string) string {
	if schema == defaultSchema || schema == "" {
		return table
	}

	return schema + "." + table
}
