// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package tablenorm_test

import (
	"sort"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/enginecompat/tablenorm"
	"github.com/LerianStudio/fetcher/pkg/model"
)

func TestSchemaScopeForTables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		dbType model.DBType
		tables map[string][]string
		want   []string
	}{
		{
			name:   "PostgreSQL qualified tables yield their schemas",
			dbType: model.TypePostgreSQL,
			tables: map[string][]string{"accounting.invoices": nil, "reporting.daily_summary": nil},
			want:   []string{"accounting", "reporting"},
		},
		{
			name:   "PostgreSQL mixed qualified + unqualified injects public default",
			dbType: model.TypePostgreSQL,
			tables: map[string][]string{"accounting.invoices": nil, "users": nil},
			want:   []string{"accounting", "public"},
		},
		{
			name:   "PostgreSQL only unqualified yields just public",
			dbType: model.TypePostgreSQL,
			tables: map[string][]string{"users": nil},
			want:   []string{"public"},
		},
		{
			name:   "PostgreSQL qualified public is not duplicated",
			dbType: model.TypePostgreSQL,
			tables: map[string][]string{"public.users": nil, "orders": nil},
			want:   []string{"public"},
		},
		{
			name:   "SQLServer mixed qualified + unqualified injects dbo default",
			dbType: model.TypeSQLServer,
			tables: map[string][]string{"sales.orders": nil, "users": nil},
			want:   []string{"dbo", "sales"},
		},
		{
			name:   "Oracle has no default injection (qualified schemas only)",
			dbType: model.TypeOracle,
			tables: map[string][]string{"HR.EMPLOYEES": nil, "DUAL": nil},
			want:   []string{"HR"},
		},
		{
			name:   "Oracle all-unqualified yields nil (adapter default applies)",
			dbType: model.TypeOracle,
			tables: map[string][]string{"DUAL": nil},
			want:   nil,
		},
		{
			name:   "MySQL no default injection",
			dbType: model.TypeMySQL,
			tables: map[string][]string{"app.users": nil, "orders": nil},
			want:   []string{"app"},
		},
		{
			name:   "empty tables yields nil",
			dbType: model.TypePostgreSQL,
			tables: map[string][]string{},
			want:   nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tablenorm.SchemaScopeForTables(tc.dbType, tc.tables)
			sort.Strings(got)
			want := append([]string(nil), tc.want...)
			sort.Strings(want)

			if len(got) != len(want) {
				t.Fatalf("SchemaScopeForTables = %#v, want %#v", got, want)
			}

			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("SchemaScopeForTables = %#v, want %#v", got, want)
				}
			}
		})
	}
}

func TestDefaultSchemaForType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		dbType model.DBType
		want   string
	}{
		{name: "postgres -> public", dbType: model.TypePostgreSQL, want: "public"},
		{name: "sqlserver -> dbo", dbType: model.TypeSQLServer, want: "dbo"},
		{name: "oracle -> empty", dbType: model.TypeOracle, want: ""},
		{name: "mysql -> empty", dbType: model.TypeMySQL, want: ""},
		{name: "mongodb -> empty", dbType: model.TypeMongoDB, want: ""},
		{name: "unknown -> empty", dbType: model.DBType("WAT"), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tablenorm.DefaultSchemaForType(tt.dbType); got != tt.want {
				t.Fatalf("DefaultSchemaForType(%q) = %q, want %q", tt.dbType, got, tt.want)
			}
		})
	}
}

func TestNormalizeTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		dbType model.DBType
		input  string
		want   string
	}{
		{name: "pg public prefix stripped", dbType: model.TypePostgreSQL, input: "public.transactions", want: "transactions"},
		{name: "pg unqualified unchanged", dbType: model.TypePostgreSQL, input: "transactions", want: "transactions"},
		{name: "pg non-default schema preserved", dbType: model.TypePostgreSQL, input: "accounting.invoices", want: "accounting.invoices"},
		{name: "sqlserver dbo prefix stripped", dbType: model.TypeSQLServer, input: "dbo.users", want: "users"},
		{name: "sqlserver custom schema preserved", dbType: model.TypeSQLServer, input: "sales.orders", want: "sales.orders"},
		{name: "sqlserver case preserved", dbType: model.TypeSQLServer, input: "dbo.Users", want: "Users"},
		// Oracle is UPPERCASE-CANONICAL (matches the physical Oracle catalog + result keys); no public-prefix concept.
		{name: "oracle lowercase folds upper", dbType: model.TypeOracle, input: "accounts", want: "ACCOUNTS"},
		{name: "oracle mixed-case folds upper", dbType: model.TypeOracle, input: "Accounts", want: "ACCOUNTS"},
		{name: "oracle uppercase stays upper", dbType: model.TypeOracle, input: "ACCOUNTS", want: "ACCOUNTS"},
		{name: "oracle owner-qualified folds upper", dbType: model.TypeOracle, input: "hr.employees", want: "HR.EMPLOYEES"},
		{name: "mongodb no stripping, case preserved", dbType: model.TypeMongoDB, input: "db.Collection", want: "db.Collection"},
		{name: "pg case preserved", dbType: model.TypePostgreSQL, input: "public.Users", want: "Users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tablenorm.NormalizeTable(tt.dbType, tt.input); got != tt.want {
				t.Fatalf("NormalizeTable(%q, %q) = %q, want %q", tt.dbType, tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		dbType model.DBType
		input  string
		want   string
	}{
		// Case-sensitive types: field name preserved exactly (FIX-2 must NOT alter them).
		{name: "pg field case preserved", dbType: model.TypePostgreSQL, input: "CreatedAt", want: "CreatedAt"},
		{name: "mysql field case preserved", dbType: model.TypeMySQL, input: "userName", want: "userName"},
		{name: "sqlserver field case preserved", dbType: model.TypeSQLServer, input: "OrderId", want: "OrderId"},
		{name: "mongodb field case preserved", dbType: model.TypeMongoDB, input: "primaryEmail", want: "primaryEmail"},
		// Oracle is UPPERCASE-CANONICAL (matches the physical ALL_TAB_COLUMNS catalog + result column keys).
		{name: "oracle field lowercase folds upper", dbType: model.TypeOracle, input: "balance", want: "BALANCE"},
		{name: "oracle field mixed folds upper", dbType: model.TypeOracle, input: "AccountId", want: "ACCOUNTID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tablenorm.NormalizeField(tt.dbType, tt.input); got != tt.want {
				t.Fatalf("NormalizeField(%q, %q) = %q, want %q", tt.dbType, tt.input, got, tt.want)
			}
		})
	}
}

func TestFoldsFieldCase(t *testing.T) {
	t.Parallel()

	if !tablenorm.FoldsFieldCase(model.TypeOracle) {
		t.Fatal("Oracle must fold field case")
	}

	for _, dbType := range []model.DBType{model.TypePostgreSQL, model.TypeMySQL, model.TypeSQLServer, model.TypeMongoDB} {
		if tablenorm.FoldsFieldCase(dbType) {
			t.Fatalf("%s must NOT fold field case (was case-sensitive in the legacy path)", dbType)
		}
	}
}
