package schemautil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseQualifiedTableName(t *testing.T) {
	tests := []struct {
		name          string
		qualifiedName string
		defaultSchema string
		wantSchema    string
		wantTable     string
		wantErr       error
	}{
		{
			name:          "qualified name with custom schema",
			qualifiedName: "accounting.invoices",
			defaultSchema: "public",
			wantSchema:    "accounting",
			wantTable:     "invoices",
		},
		{
			name:          "unqualified name returns default schema",
			qualifiedName: "invoices",
			defaultSchema: "public",
			wantSchema:    "public",
			wantTable:     "invoices",
		},
		{
			name:          "dbo qualified name for SQL Server",
			qualifiedName: "dbo.users",
			defaultSchema: "dbo",
			wantSchema:    "dbo",
			wantTable:     "users",
		},
		{
			name:          "unqualified name with dbo default",
			qualifiedName: "users",
			defaultSchema: "dbo",
			wantSchema:    "dbo",
			wantTable:     "users",
		},
		{
			name:          "Oracle uppercase qualified name",
			qualifiedName: "HR.EMPLOYEES",
			defaultSchema: "SYSTEM",
			wantSchema:    "HR",
			wantTable:     "EMPLOYEES",
		},
		{
			name:          "empty default schema",
			qualifiedName: "users",
			defaultSchema: "",
			wantSchema:    "",
			wantTable:     "users",
		},
		{
			name:          "3-part SQL Server name extracts schema and table",
			qualifiedName: "mydb.dbo.users",
			defaultSchema: "public",
			wantSchema:    "dbo",
			wantTable:     "users",
		},
		{
			name:          "3-part name with custom schema",
			qualifiedName: "catalog.accounting.invoices",
			defaultSchema: "dbo",
			wantSchema:    "accounting",
			wantTable:     "invoices",
		},
		{
			name:          "empty table name returns error",
			qualifiedName: "",
			defaultSchema: "public",
			wantErr:       ErrEmptyTableName,
		},
		{
			name:          "whitespace only table name returns error",
			qualifiedName: "   ",
			defaultSchema: "public",
			wantErr:       ErrEmptyTableName,
		},
		{
			name:          "4-part name returns error",
			qualifiedName: "server.catalog.schema.table",
			defaultSchema: "public",
			wantErr:       ErrInvalidTableName,
		},
		{
			name:          "more than 4 parts returns error",
			qualifiedName: "a.b.c.d.e",
			defaultSchema: "public",
			wantErr:       ErrInvalidTableName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSchema, gotTable, err := ParseQualifiedTableName(tt.qualifiedName, tt.defaultSchema)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSchema, gotSchema, "schema mismatch")
			assert.Equal(t, tt.wantTable, gotTable, "table mismatch")
		})
	}
}

func TestNormalizeTableNameForLookup(t *testing.T) {
	tests := []struct {
		name          string
		tableName     string
		defaultSchema string
		want          string
	}{
		{
			name:          "strips default public schema prefix",
			tableName:     "public.users",
			defaultSchema: "public",
			want:          "users",
		},
		{
			name:          "preserves non-default schema prefix",
			tableName:     "accounting.invoices",
			defaultSchema: "public",
			want:          "accounting.invoices",
		},
		{
			name:          "unqualified name unchanged",
			tableName:     "users",
			defaultSchema: "public",
			want:          "users",
		},
		{
			name:          "strips default dbo schema prefix for SQL Server",
			tableName:     "dbo.users",
			defaultSchema: "dbo",
			want:          "users",
		},
		{
			name:          "preserves custom schema for SQL Server",
			tableName:     "sales.orders",
			defaultSchema: "dbo",
			want:          "sales.orders",
		},
		{
			name:          "strips default SYSTEM schema for Oracle",
			tableName:     "SYSTEM.transactions",
			defaultSchema: "SYSTEM",
			want:          "transactions",
		},
		{
			name:          "preserves HR schema for Oracle",
			tableName:     "HR.EMPLOYEES",
			defaultSchema: "SYSTEM",
			want:          "HR.EMPLOYEES",
		},
		{
			name:          "empty default schema preserves qualified name",
			tableName:     "schema.table",
			defaultSchema: "",
			want:          "schema.table",
		},
		{
			name:          "case sensitive comparison",
			tableName:     "PUBLIC.users",
			defaultSchema: "public",
			want:          "users",
		},
		{
			name:          "sqlserver 3-part name with default schema",
			tableName:     "mydb.dbo.users",
			defaultSchema: "dbo",
			want:          "users",
		},
		{
			name:          "sqlserver 3-part name with custom schema",
			tableName:     "mydb.sales.orders",
			defaultSchema: "dbo",
			want:          "sales.orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTableNameForLookup(tt.tableName, tt.defaultSchema)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestQualifyTableName(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		table         string
		defaultSchema string
		want          string
	}{
		{
			name:          "default schema returns just table name",
			schema:        "public",
			table:         "users",
			defaultSchema: "public",
			want:          "users",
		},
		{
			name:          "non-default schema returns qualified name",
			schema:        "accounting",
			table:         "invoices",
			defaultSchema: "public",
			want:          "accounting.invoices",
		},
		{
			name:          "empty schema returns just table name",
			schema:        "",
			table:         "users",
			defaultSchema: "public",
			want:          "users",
		},
		{
			name:          "dbo default schema for SQL Server",
			schema:        "dbo",
			table:         "orders",
			defaultSchema: "dbo",
			want:          "orders",
		},
		{
			name:          "sales schema for SQL Server",
			schema:        "sales",
			table:         "orders",
			defaultSchema: "dbo",
			want:          "sales.orders",
		},
		{
			name:          "Oracle SYSTEM schema",
			schema:        "SYSTEM",
			table:         "USERS",
			defaultSchema: "SYSTEM",
			want:          "USERS",
		},
		{
			name:          "Oracle HR schema",
			schema:        "HR",
			table:         "EMPLOYEES",
			defaultSchema: "SYSTEM",
			want:          "HR.EMPLOYEES",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QualifyTableName(tt.schema, tt.table, tt.defaultSchema)
			assert.Equal(t, tt.want, got)
		})
	}
}
