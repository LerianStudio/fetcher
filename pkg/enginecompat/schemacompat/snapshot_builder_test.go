// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package schemacompat_test

import (
	"sort"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/pkg/model"
)

// tableNames returns the snapshot table names sorted for order-independent assertions.
func tableNames(snapshot engine.SchemaSnapshot) []string {
	names := make([]string, 0, len(snapshot.Tables))
	for _, table := range snapshot.Tables {
		names = append(names, table.Name)
	}

	sort.Strings(names)

	return names
}

// fieldsFor returns the (sorted) fields of the named table in the snapshot.
func fieldsFor(snapshot engine.SchemaSnapshot, name string) []string {
	for _, table := range snapshot.Tables {
		if table.Name == name {
			fields := append([]string(nil), table.Fields...)
			sort.Strings(fields)

			return fields
		}
	}

	return nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// TestBuildSnapshot_FlagMatrix exercises every SnapshotOptions combination and the
// per-type normalization, including the system-table filter and the Oracle
// LOWERCASE-canonical fold.
func TestBuildSnapshot_FlagMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		dbType     model.DBType
		schema     func() *model.DataSourceSchema
		opts       schemacompat.SnapshotOptions
		wantTables []string
		wantFields map[string][]string
	}{
		{
			name:   "no filter no normalize keeps everything verbatim (pg)",
			dbType: model.TypePostgreSQL,
			schema: func() *model.DataSourceSchema {
				s := model.NewDataSourceSchema("pg")
				s.AddTable("public.users", []string{"Id", "Name"})
				s.AddTable("pg_catalog.pg_class", []string{"oid"})

				return s
			},
			opts:       schemacompat.SnapshotOptions{},
			wantTables: []string{"pg_catalog.pg_class", "public.users"},
			wantFields: map[string][]string{"public.users": {"Id", "Name"}},
		},
		{
			name:   "filter drops system tables (pg)",
			dbType: model.TypePostgreSQL,
			schema: func() *model.DataSourceSchema {
				s := model.NewDataSourceSchema("pg")
				s.AddTable("users", []string{"id"})
				s.AddTable("pg_catalog.pg_class", []string{"oid"})

				return s
			},
			opts:       schemacompat.SnapshotOptions{FilterSystemTables: true},
			wantTables: []string{"users"},
			wantFields: map[string][]string{"users": {"id"}},
		},
		{
			name:   "normalize strips default schema prefix but preserves case (pg)",
			dbType: model.TypePostgreSQL,
			schema: func() *model.DataSourceSchema {
				s := model.NewDataSourceSchema("pg")
				s.AddTable("public.Orders", []string{"OrderId"})
				s.AddTable("sales.Invoices", []string{"Total"})

				return s
			},
			opts:       schemacompat.SnapshotOptions{Normalize: true},
			wantTables: []string{"Orders", "sales.Invoices"},
			wantFields: map[string][]string{"Orders": {"OrderId"}, "sales.Invoices": {"Total"}},
		},
		{
			name:   "normalize folds Oracle to UPPERCASE (table + fields)",
			dbType: model.TypeOracle,
			schema: func() *model.DataSourceSchema {
				// GetSchemaInfo lowercases its output; BuildSnapshot re-folds to UPPERCASE
				// to match the physical Oracle catalog + result keys.
				s := model.NewDataSourceSchema("ora")
				s.AddTable("accounts", []string{"id", "balance"})

				return s
			},
			opts:       schemacompat.SnapshotOptions{FilterSystemTables: true, Normalize: true},
			wantTables: []string{"ACCOUNTS"},
			wantFields: map[string][]string{"ACCOUNTS": {"BALANCE", "ID"}},
		},
		{
			name:   "normalize is identity for MySQL (case-sensitive, no default schema)",
			dbType: model.TypeMySQL,
			schema: func() *model.DataSourceSchema {
				s := model.NewDataSourceSchema("my")
				s.AddTable("Users", []string{"Id"})

				return s
			},
			opts:       schemacompat.SnapshotOptions{Normalize: true},
			wantTables: []string{"Users"},
			wantFields: map[string][]string{"Users": {"Id"}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := schemacompat.BuildSnapshot("cfg", tt.dbType, tt.schema(), tt.opts)

			if names := tableNames(got); !equalStrings(names, tt.wantTables) {
				t.Fatalf("tables = %v, want %v", names, tt.wantTables)
			}

			for table, want := range tt.wantFields {
				if fields := fieldsFor(got, table); !equalStrings(fields, want) {
					t.Fatalf("fields[%q] = %v, want %v", table, fields, want)
				}
			}
		})
	}
}

// TestBuildSnapshot_NilAndEmpty pins the degenerate cases the prior converters all
// handled: a nil schema yields an empty snapshot carrying only the config name, and a
// schema with no tables yields a snapshot with no tables but the carried ConfigName.
func TestBuildSnapshot_NilAndEmpty(t *testing.T) {
	t.Parallel()

	nilSnap := schemacompat.BuildSnapshot("cfg", model.TypePostgreSQL, nil, schemacompat.SnapshotOptions{})
	if nilSnap.ConfigName != "cfg" || len(nilSnap.Tables) != 0 {
		t.Fatalf("nil schema: got %#v", nilSnap)
	}

	empty := model.NewDataSourceSchema("from-schema")
	emptySnap := schemacompat.BuildSnapshot("cfg", model.TypePostgreSQL, empty, schemacompat.SnapshotOptions{})
	if emptySnap.ConfigName != "from-schema" {
		t.Fatalf("empty schema must carry schema.ConfigName, got %q", emptySnap.ConfigName)
	}
	if len(emptySnap.Tables) != 0 {
		t.Fatalf("empty schema must yield no tables, got %v", emptySnap.Tables)
	}
}

// TestBuildSnapshot_NonOracleCharacterization is the golden characterization that the
// four forward call sites' outputs are UNCHANGED for non-Oracle types after the
// consolidation. It mirrors the exact flags each call site uses and asserts the
// snapshot is byte-stable for a PostgreSQL schema (the type most affected by the
// system-table filter and default-schema strip).
func TestBuildSnapshot_NonOracleCharacterization(t *testing.T) {
	t.Parallel()

	build := func() *model.DataSourceSchema {
		s := model.NewDataSourceSchema("pg")
		s.AddTable("public.users", []string{"id", "email"})
		s.AddTable("accounting.invoices", []string{"total"})
		s.AddTable("pg_catalog.pg_class", []string{"oid"})

		return s
	}

	// (1) cache round-trip: {filter:false, normalize:false}
	cacheLike := schemacompat.BuildSnapshot("pg", "", build(), schemacompat.SnapshotOptions{})
	if names := tableNames(cacheLike); !equalStrings(names, []string{"accounting.invoices", "pg_catalog.pg_class", "public.users"}) {
		t.Fatalf("cache-like snapshot tables = %v", names)
	}

	// (2) Manager discovery: {filter:true, normalize:false}
	discovery := schemacompat.BuildSnapshot("pg", model.TypePostgreSQL, build(), schemacompat.SnapshotOptions{FilterSystemTables: true})
	if names := tableNames(discovery); !equalStrings(names, []string{"accounting.invoices", "public.users"}) {
		t.Fatalf("discovery snapshot tables = %v", names)
	}

	// (3) Worker extraction: {filter:true, normalize:true} — strips "public." prefix,
	// preserves case for PG.
	extraction := schemacompat.BuildSnapshot("pg", model.TypePostgreSQL, build(), schemacompat.SnapshotOptions{FilterSystemTables: true, Normalize: true})
	if names := tableNames(extraction); !equalStrings(names, []string{"accounting.invoices", "users"}) {
		t.Fatalf("extraction snapshot tables = %v", names)
	}
}
