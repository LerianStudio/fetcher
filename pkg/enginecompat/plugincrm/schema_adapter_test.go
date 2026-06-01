// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package enginecompatplugincrm_test

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	plugincrm "github.com/LerianStudio/fetcher/pkg/enginecompat/plugincrm"
)

// snapshot is a tiny helper that builds an engine.SchemaSnapshot from a set of
// physical collection names so the CRM compatibility behavior can be exercised
// against the literal, already-discovered schema the way the Engine core sees it.
func snapshot(configName string, collections ...string) engine.SchemaSnapshot {
	tables := make([]engine.TableSnapshot, 0, len(collections))
	for _, name := range collections {
		tables = append(tables, engine.TableSnapshot{Name: name})
	}

	return engine.SchemaSnapshot{ConfigName: configName, Tables: tables}
}

// TestPluginCRMConfigName_IsExplicitPolicyConstant pins the product-specific
// config name the compatibility seam keys on. It must remain the literal
// "plugin_crm" so the policy is visible and never collapses into a generic
// datasource name.
func TestPluginCRMConfigName_IsExplicitPolicyConstant(t *testing.T) {
	t.Parallel()

	if plugincrm.ConfigName != "plugin_crm" {
		t.Fatalf("CRM compatibility config name = %q, want %q", plugincrm.ConfigName, "plugin_crm")
	}
}

// TestCRMCompatibility_IsPluginCRM_OnlySelectsTheCRMSource proves the selection
// predicate: the CRM compatibility mapping is invoked ONLY for the plugin_crm
// config name, never for a generic datasource.
func TestCRMCompatibility_IsPluginCRM_OnlySelectsTheCRMSource(t *testing.T) {
	t.Parallel()

	if !plugincrm.IsPluginCRM("plugin_crm") {
		t.Fatalf("IsPluginCRM(%q) = false, want true", "plugin_crm")
	}

	for _, configName := range []string{"postgres_main", "mongo_orders", "PLUGIN_CRM", "plugin_crm_extra", ""} {
		if plugincrm.IsPluginCRM(configName) {
			t.Fatalf("IsPluginCRM(%q) = true, want false (non-CRM source must not select CRM mapping)", configName)
		}
	}
}

// TestCRMCompatibility_NonCRMSource_SkipsMapping proves that a non-CRM
// datasource is returned UNCHANGED: the CRM compatibility transformation does
// not execute for it, and no reverse map is produced.
func TestCRMCompatibility_NonCRMSource_SkipsMapping(t *testing.T) {
	t.Parallel()

	tables := map[string][]string{"users": {"id", "email"}}
	snap := snapshot("postgres_main", "users")

	transformed, reverse := plugincrm.MapTablesForCRMCompatibility("postgres_main", tables, snap)

	if !reflect.DeepEqual(transformed, tables) {
		t.Fatalf("non-CRM source mutated tables: got %v, want %v", transformed, tables)
	}
	if reverse != nil {
		t.Fatalf("non-CRM source produced reverse map %v, want nil (CRM mapping must not run)", reverse)
	}
}

// TestCRMCompatibility_PluginCRM_AutoDiscoversCollectionByPrefix preserves the
// legacy Manager behavior: a logical CRM collection name resolves to the first
// physical collection matching the "<name>_" prefix, and the reverse map carries
// the physical->logical name for error display.
func TestCRMCompatibility_PluginCRM_AutoDiscoversCollectionByPrefix(t *testing.T) {
	t.Parallel()

	tables := map[string][]string{"holders": {"document", "name"}}
	snap := snapshot("plugin_crm", "holders_06c4f684", "accounts_abc123")

	transformed, reverse := plugincrm.MapTablesForCRMCompatibility("plugin_crm", tables, snap)

	wantTransformed := map[string][]string{"holders_06c4f684": {"document", "name"}}
	if !reflect.DeepEqual(transformed, wantTransformed) {
		t.Fatalf("CRM prefix auto-discovery transformed = %v, want %v", transformed, wantTransformed)
	}

	wantReverse := map[string]string{"holders_06c4f684": "holders"}
	if !reflect.DeepEqual(reverse, wantReverse) {
		t.Fatalf("CRM reverse map = %v, want %v", reverse, wantReverse)
	}
}

// TestCRMCompatibility_PluginCRM_FullCollectionNamePassesThrough preserves the
// legacy behavior where a fully-qualified physical collection name already
// present in the schema is passed through unchanged and maps to itself.
func TestCRMCompatibility_PluginCRM_FullCollectionNamePassesThrough(t *testing.T) {
	t.Parallel()

	tables := map[string][]string{"holders_06c4f684": {"document"}}
	snap := snapshot("plugin_crm", "holders_06c4f684")

	transformed, reverse := plugincrm.MapTablesForCRMCompatibility("plugin_crm", tables, snap)

	if !reflect.DeepEqual(transformed, tables) {
		t.Fatalf("CRM full-name pass-through transformed = %v, want %v", transformed, tables)
	}
	if reverse["holders_06c4f684"] != "holders_06c4f684" {
		t.Fatalf("CRM full-name reverse[%q] = %q, want self", "holders_06c4f684", reverse["holders_06c4f684"])
	}
}

// TestCRMCompatibility_PluginCRM_UnmatchedLogicalNamePassesThrough preserves the
// legacy behavior where a logical name with no matching physical prefix is left
// as-is so downstream validation reports TABLE_NOT_FOUND.
func TestCRMCompatibility_PluginCRM_UnmatchedLogicalNamePassesThrough(t *testing.T) {
	t.Parallel()

	tables := map[string][]string{"missing": {"x"}}
	snap := snapshot("plugin_crm", "holders_06c4f684")

	transformed, reverse := plugincrm.MapTablesForCRMCompatibility("plugin_crm", tables, snap)

	if !reflect.DeepEqual(transformed, map[string][]string{"missing": {"x"}}) {
		t.Fatalf("CRM unmatched transformed = %v, want pass-through", transformed)
	}
	if reverse["missing"] != "missing" {
		t.Fatalf("CRM unmatched reverse[%q] = %q, want self", "missing", reverse["missing"])
	}
}

// TestCRMCompatibility_PluginCRM_FirstPrefixMatchWins preserves the legacy
// deterministic-first-match behavior: when several physical collections share a
// logical prefix, the lexicographically-first match is selected, so the seam is
// deterministic rather than map-iteration-order dependent.
func TestCRMCompatibility_PluginCRM_FirstPrefixMatchWins(t *testing.T) {
	t.Parallel()

	tables := map[string][]string{"holders": {"document"}}
	snap := snapshot("plugin_crm", "holders_zzz", "holders_aaa", "holders_mmm")

	transformed, reverse := plugincrm.MapTablesForCRMCompatibility("plugin_crm", tables, snap)

	got := make([]string, 0, len(transformed))
	for name := range transformed {
		got = append(got, name)
	}
	sort.Strings(got)

	if len(got) != 1 || got[0] != "holders_aaa" {
		t.Fatalf("CRM first-prefix-match transformed keys = %v, want [holders_aaa]", got)
	}
	if reverse["holders_aaa"] != "holders" {
		t.Fatalf("CRM first-match reverse[%q] = %q, want %q", "holders_aaa", reverse["holders_aaa"], "holders")
	}
}

// TestCRMCompatibility_PluginCRM_PhysicalNameCollision_UnionsFields proves that
// when two distinct logical names resolve to the SAME physical collection, the
// adapter MERGES (unions) their field lists deterministically rather than letting
// one map-iteration order silently overwrite the other and drop a table's fields.
func TestCRMCompatibility_PluginCRM_PhysicalNameCollision_UnionsFields(t *testing.T) {
	t.Parallel()

	// Both the logical name "holders" (prefix-resolved to "holders_06c4f684") and
	// the already-physical name "holders_06c4f684" collapse onto the same physical
	// collection. Their field lists must be unioned, not overwritten.
	tables := map[string][]string{
		"holders":          {"document", "name"},
		"holders_06c4f684": {"name", "email"},
	}
	snap := snapshot("plugin_crm", "holders_06c4f684")

	transformed, reverse := plugincrm.MapTablesForCRMCompatibility("plugin_crm", tables, snap)

	if len(transformed) != 1 {
		t.Fatalf("CRM collision: expected a single physical entry, got %v", transformed)
	}

	got := transformed["holders_06c4f684"]
	want := []string{"document", "email", "name"} // de-duplicated, sorted
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CRM collision: field union = %v, want %v (no field dropped, deterministic)", got, want)
	}

	// The reverse map must resolve deterministically; the lexicographically-first
	// logical name wins.
	if reverse["holders_06c4f684"] != "holders" {
		t.Fatalf("CRM collision: reverse[%q] = %q, want deterministic %q", "holders_06c4f684", reverse["holders_06c4f684"], "holders")
	}
}

// TestPluginCRMCompatibility_IsAdapterScoped_NotImportedByEngineCore enforces the
// hard dependency boundary: pkg/engine MUST NOT import the plugin_crm
// compatibility adapter. CRM stays adapter-scoped, never core.
func TestPluginCRMCompatibility_IsAdapterScoped_NotImportedByEngineCore(t *testing.T) {
	t.Parallel()

	deps, listErr := exec.Command(
		"go", "list", "-mod=readonly", "-deps",
		"github.com/LerianStudio/fetcher/pkg/engine/...",
	).CombinedOutput()
	if listErr != nil {
		t.Fatalf("go list -deps pkg/engine failed: %v\n%s", listErr, deps)
	}

	if strings.Contains(string(deps), "enginecompat/plugincrm") {
		t.Fatalf("pkg/engine MUST NOT import the plugin_crm compatibility adapter; deps:\n%s", deps)
	}
}

// TestPluginCRMCompatibility_AdapterImportsOnlyEngineCore enforces the one-way
// import direction: the compatibility adapter source may depend on pkg/engine
// but must not reach into Manager/Worker components or concrete drivers, so the
// adapter cannot smuggle infrastructure back across the seam.
func TestPluginCRMCompatibility_AdapterImportsOnlyEngineCore(t *testing.T) {
	t.Parallel()

	repoRoot := repositoryRoot(t)
	adapterPath := filepath.Join(repoRoot, "pkg", "enginecompat", "plugincrm", "schema_adapter.go")

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, adapterPath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("failed to parse adapter source %q: %v", adapterPath, err)
	}

	forbidden := []string{
		"components/manager",
		"components/worker",
		"pkg/datasource",
		"pkg/mongodb",
		"pkg/postgres",
		"go.mongodb.org",
		"database/sql",
	}

	for _, imp := range parsed.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		for _, bad := range forbidden {
			if strings.Contains(path, bad) {
				t.Fatalf("CRM compatibility adapter imports forbidden dependency %q (matched %q)", path, bad)
			}
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repository root from %q", dir)
		}

		dir = parent
	}
}
