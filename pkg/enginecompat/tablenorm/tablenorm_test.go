// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package tablenorm_test

import (
	"testing"

	"github.com/LerianStudio/fetcher/pkg/enginecompat/tablenorm"
	"github.com/LerianStudio/fetcher/pkg/model"
)

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
		{name: "oracle no stripping", dbType: model.TypeOracle, input: "public.x", want: "public.x"},
		{name: "mongodb no stripping", dbType: model.TypeMongoDB, input: "db.collection", want: "db.collection"},
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
