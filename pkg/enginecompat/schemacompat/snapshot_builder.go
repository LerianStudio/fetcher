// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package schemacompat

import (
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/tablenorm"
	"github.com/LerianStudio/fetcher/pkg/model"
)

// SnapshotOptions parameterizes the single forward DataSourceSchema -> SchemaSnapshot
// builder so every host call site shares ONE normalization implementation.
//
//   - FilterSystemTables drops system tables (pg_*, Oracle SYS, db_*, Mongo system.*)
//     via IsSystemTable, so the snapshot that crosses the Engine boundary never
//     carries a system table as a valid one.
//   - Normalize canonicalizes table keys (default-schema prefix rule + Oracle
//     lowercase fold) and field names (Oracle lowercase fold) via tablenorm, so the
//     snapshot side and the request side reconcile to the SAME identity before the
//     Engine's LITERAL match. It is a no-op for the case-sensitive types with no
//     strippable default schema (MySQL, MongoDB) and strips only the default-schema
//     prefix for PostgreSQL/SQLServer.
type SnapshotOptions struct {
	FilterSystemTables bool
	Normalize          bool
}

// BuildSnapshot is the SINGLE forward converter from a host *model.DataSourceSchema
// into a secret-free Engine SchemaSnapshot. Every forward call site routes through
// it with the flags appropriate to that seam, so the filter rule and the
// normalization rule each have exactly one implementation:
//
//   - SnapshotFromDataSourceSchema -> {filter:false, normalize:false} (cache round-trip)
//   - Connector.snapshot           -> {filter:true,  normalize:false} (Manager discovery)
//   - snapshotFromSchema (Worker)  -> {filter:true,  normalize:true}  (extraction path)
//   - snapshotForCRM               -> {filter:false, normalize:false} (CRM projection)
//
// A nil schema yields an empty snapshot carrying only configName. The schema's own
// ConfigName overrides configName when present, and CapturedAt is carried through,
// matching every prior converter's behavior.
func BuildSnapshot(
	configName string,
	dbType model.DBType,
	schema *model.DataSourceSchema,
	opts SnapshotOptions,
) engine.SchemaSnapshot {
	snapshot := engine.SchemaSnapshot{ConfigName: configName}
	if schema == nil {
		return snapshot
	}

	if schema.ConfigName != "" {
		snapshot.ConfigName = schema.ConfigName
	}

	snapshot.CapturedAt = schema.CachedAt

	if len(schema.Tables) == 0 {
		return snapshot
	}

	tables := make([]engine.TableSnapshot, 0, len(schema.Tables))

	for name, table := range schema.Tables {
		tableName := name
		if table != nil && table.TableName != "" {
			tableName = table.TableName
		}

		if opts.FilterSystemTables && IsSystemTable(dbType, tableName) {
			continue
		}

		var fields []string
		if table != nil {
			fields = table.GetColumnsList()
		}

		if opts.Normalize {
			tableName = tablenorm.NormalizeTable(dbType, tableName)
			fields = normalizeSnapshotFields(dbType, fields)
		}

		tables = append(tables, engine.TableSnapshot{Name: tableName, Fields: fields})
	}

	snapshot.Tables = tables

	return snapshot
}

// normalizeSnapshotFields canonicalizes a snapshot's field names for the datasource
// type so the snapshot side and the request side reconcile to the SAME identity
// before the Engine's literal field match. It is the IDENTITY for case-sensitive
// types (PG/MySQL/SQLServer) and folds to LOWERCASE for Oracle, mirroring
// tablenorm.NormalizeField on the request side. A nil input yields nil.
func normalizeSnapshotFields(dbType model.DBType, fields []string) []string {
	if !tablenorm.FoldsFieldCase(dbType) || fields == nil {
		return fields
	}

	out := make([]string, len(fields))
	for i, field := range fields {
		out[i] = tablenorm.NormalizeField(dbType, field)
	}

	return out
}
