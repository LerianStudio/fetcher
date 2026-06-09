// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package enginecompatplugincrm is the EXPLICIT, product-scoped compatibility
// adapter that preserves the legacy Manager `plugin_crm` schema behavior for the
// first Engine release. It is deliberately named for the product policy it
// encodes — `plugin_crm` / CRMCompatibility — so the special-casing is visible
// at the call site and can never be mistaken for generic datasource behavior.
//
// WHY THIS LIVES OUTSIDE pkg/engine. The Engine core validates against the
// LITERAL SchemaSnapshot and performs NO datasource-type-specific normalization
// (no Oracle lowercasing, no default-schema injection, no collection
// auto-discovery). CRM collection auto-discovery is product policy, not a
// generic core extension, so it stays in this adapter. The import direction is
// one-way: this package MAY import pkg/engine; pkg/engine MUST NOT import this
// package. The dependency boundary tests in pkg/engine and in this package keep
// that invariant enforced.
//
// WHAT IT PRESERVES. The legacy Manager (validate_schema.go) transformed CRM
// table names ONLY when the datasource config name was "plugin_crm", AFTER the
// real schema was resolved: each logical collection name (e.g. "holders")
// resolved to the first physical collection matching the "<name>_" prefix (e.g.
// "holders_06c4f684"), a fully-qualified physical name already present passed
// through unchanged, and an unmatched logical name passed through as-is so
// downstream validation reports TABLE_NOT_FOUND. A reverse map
// (physical -> logical) is produced for error display. This adapter reproduces
// exactly that mapping against the Engine's SchemaSnapshot.
package enginecompatplugincrm

import (
	"sort"
	"strings"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
)

// ConfigName is the literal datasource config name that selects CRM
// compatibility behavior. It is a product-policy constant, not a generic
// datasource identifier, and intentionally carries the `plugin_crm` name so the
// special-casing is explicit wherever the selection predicate is read.
const ConfigName = "plugin_crm"

// IsPluginCRM reports whether a datasource config name selects the CRM
// compatibility mapping. The match is EXACT: only the literal "plugin_crm"
// source is CRM-compatible, so a generic datasource — including names that merely
// contain or differ in case from "plugin_crm" — never triggers CRM behavior.
func IsPluginCRM(configName string) bool {
	return configName == ConfigName
}

// MapTablesForCRMCompatibility resolves a logical CRM table mapping against the
// already-discovered SchemaSnapshot, preserving the legacy Manager `plugin_crm`
// auto-discovery behavior.
//
// It is a NO-OP for any non-CRM source: when configName is not the literal
// "plugin_crm", the input tables are returned UNCHANGED and the reverse map is
// nil, so generic datasource validation never executes CRM policy.
//
// For the CRM source it returns the transformed table mapping plus a reverse map
// (physical -> logical name) for error display. For each logical table name:
//   - if a physical collection of that exact name exists in the snapshot, it
//     passes through unchanged (maps to itself);
//   - otherwise the first physical collection whose name has the "<logical>_"
//     prefix is selected (matches sorted for a deterministic first match, so the
//     seam does not depend on snapshot ordering);
//   - if no physical collection matches, the logical name passes through as-is so
//     downstream validation reports TABLE_NOT_FOUND.
func MapTablesForCRMCompatibility(
	configName string,
	tables map[string][]string,
	snapshot engine.SchemaSnapshot,
) (map[string][]string, map[string]string) {
	if !IsPluginCRM(configName) {
		return tables, nil
	}

	physicalCollections := sortedTableNames(snapshot)

	transformed := make(map[string][]string, len(tables))
	reverse := make(map[string]string, len(tables))

	// Iterate logical names in sorted order so collision resolution (field union +
	// reverse mapping) is fully deterministic regardless of Go map-iteration order.
	for _, logicalName := range sortedKeys(tables) {
		fields := tables[logicalName]
		physicalName := resolvePhysicalCollection(logicalName, snapshot, physicalCollections)

		if existing, collision := transformed[physicalName]; collision {
			// Two distinct logical names resolved to the same physical collection.
			// Union the field lists (de-duplicated, sorted) so no table's selected or
			// filter fields are silently dropped — union is the conservative-correct
			// validation behavior. The reverse map keeps the lexicographically-first
			// logical name (already established by the sorted iteration), so it stays
			// deterministic.
			//
			// This deterministic field-list UNION is an INTENTIONAL improvement over
			// the legacy Manager behavior, which was a nondeterministic last-write-wins
			// (the surviving field list depended on Go map-iteration order, so one
			// logical name's fields could silently overwrite another's). No stable
			// contract is broken: the legacy outcome was never deterministic, and the
			// union is strictly safer — it can only ADD fields to validate, never drop
			// a referenced field. The behavior is therefore changed on purpose.
			transformed[physicalName] = unionFields(existing, fields)

			continue
		}

		transformed[physicalName] = fields
		reverse[physicalName] = logicalName
	}

	return transformed, reverse
}

// sortedKeys returns the map keys in lexicographic order for deterministic
// iteration.
func sortedKeys(tables map[string][]string) []string {
	keys := make([]string, 0, len(tables))
	for key := range tables {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

// unionFields returns the de-duplicated, sorted union of two field lists so a
// physical-collection collision merges every referenced field deterministically.
func unionFields(a, b []string) []string {
	set := make(map[string]struct{}, len(a)+len(b))
	for _, f := range a {
		set[f] = struct{}{}
	}

	for _, f := range b {
		set[f] = struct{}{}
	}

	union := make([]string, 0, len(set))
	for f := range set {
		union = append(union, f)
	}

	sort.Strings(union)

	return union
}

// resolvePhysicalCollection maps one logical CRM collection name to its physical
// collection name in the snapshot. An exact match or no match yields the name
// unchanged; otherwise the first (sorted) prefix match "<logical>_" wins.
func resolvePhysicalCollection(logicalName string, snapshot engine.SchemaSnapshot, sortedNames []string) string {
	if snapshot.HasTable(logicalName) {
		return logicalName
	}

	prefix := logicalName + "_"
	for _, physicalName := range sortedNames {
		if strings.HasPrefix(physicalName, prefix) {
			return physicalName
		}
	}

	return logicalName
}

// sortedTableNames returns the snapshot's physical collection names in a stable
// sorted order so prefix auto-discovery selects a deterministic first match
// regardless of the order the host populated the snapshot.
func sortedTableNames(snapshot engine.SchemaSnapshot) []string {
	names := make([]string, 0, len(snapshot.Tables))
	for _, table := range snapshot.Tables {
		names = append(names, table.Name)
	}

	sort.Strings(names)

	return names
}
