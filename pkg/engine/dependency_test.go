// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

type forbiddenDependencyClass struct {
	concern         string
	name            string
	patterns        []string
	allowedPatterns []string
}

const readonlyGoFlags = "-mod=readonly"

func TestEngineDependencyBoundary_BlocksForbiddenImports(t *testing.T) {
	t.Parallel()

	repoRoot := mustRepositoryRoot(t)
	modulePath := mustModulePathFromGoMod(t, repoRoot)
	requiredClassNames := requiredForbiddenClassNames()
	configuredClasses := configuredForbiddenDependencyClasses(modulePath)

	assertForbiddenConfigComplete(t, requiredClassNames, configuredClasses)

	dependencies := collectEngineDependencyGraph(t, repoRoot)
	for _, dependencyClass := range configuredClasses {
		dependencyClass := dependencyClass
		t.Run(dependencyClass.name, func(t *testing.T) {
			t.Parallel()

			for _, dependency := range dependencies {
				if dependencyClass.matches(dependency) {
					t.Fatalf("pkg/engine dependency boundary violation: go list -deps found forbidden %s dependency %q", dependencyClass.name, dependency)
				}
			}
		})
	}
}

func TestEngineDependencyBoundary_ReadsModulePathFromGoMod(t *testing.T) {
	t.Parallel()

	repoRoot := mustRepositoryRoot(t)
	modulePath := mustModulePathFromGoMod(t, repoRoot)
	if modulePath == "" {
		t.Fatalf("module path read from go.mod is empty")
	}

	configuredClasses := configuredForbiddenDependencyClasses(modulePath)
	for _, dependencyClass := range configuredClasses {
		if dependencyClass.name != "local_infrastructure_shells" {
			continue
		}
		if !slices.Contains(dependencyClass.patterns, modulePath+"/pkg/rabbitmq") {
			t.Fatalf("local infrastructure boundary patterns do not use module path read from go.mod: %#v", dependencyClass.patterns)
		}
		return
	}

	t.Fatalf("local infrastructure dependency boundary class is not configured")
}

func TestEngineDependencyBoundary_TenantRuntimeShells_RequiredPatternsConfigured(t *testing.T) {
	t.Parallel()

	modulePath := mustModulePathFromGoMod(t, mustRepositoryRoot(t))
	configuredClasses := configuredForbiddenDependencyClasses(modulePath)
	tenantRuntimeShells := mustForbiddenDependencyClass(t, configuredClasses, "tenant_runtime_shells")

	for _, requiredPattern := range requiredTenantRuntimeShellPatterns() {
		requiredPattern := requiredPattern
		t.Run(requiredPattern, func(t *testing.T) {
			t.Parallel()

			if !slices.Contains(tenantRuntimeShells.patterns, requiredPattern) {
				t.Fatalf("tenant_runtime_shells missing required forbidden import pattern %q", requiredPattern)
			}
		})
	}
}

func TestEngineDependencyBoundary_ExternalDeploymentRuntime_RequiredPatternsConfigured(t *testing.T) {
	t.Parallel()

	modulePath := mustModulePathFromGoMod(t, mustRepositoryRoot(t))
	configuredClasses := configuredForbiddenDependencyClasses(modulePath)
	deploymentRuntime := mustForbiddenDependencyClass(t, configuredClasses, "external_deployment_runtime")

	for _, requiredPattern := range requiredExternalDeploymentRuntimePatterns() {
		requiredPattern := requiredPattern
		t.Run(requiredPattern, func(t *testing.T) {
			t.Parallel()

			if !slices.Contains(deploymentRuntime.patterns, requiredPattern) {
				t.Fatalf("external_deployment_runtime missing required forbidden import pattern %q", requiredPattern)
			}
		})
	}
}

func TestEngineDependencyBoundary_TenantRuntimeShells_PolicyIsFutureSafe(t *testing.T) {
	t.Parallel()

	modulePath := mustModulePathFromGoMod(t, mustRepositoryRoot(t))
	tenantRuntimeShells := mustForbiddenDependencyClass(t, configuredForbiddenDependencyClasses(modulePath), "tenant_runtime_shells")

	tests := []struct {
		name       string
		importPath string
		wantBlock  bool
	}{
		{
			name:       "safe tenant manager core primitive is explicitly allowed",
			importPath: "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core",
			wantBlock:  false,
		},
		{
			name:       "future tenant manager core subpackage is blocked",
			importPath: "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core/runtime/new-shell",
			wantBlock:  true,
		},
		{
			name:       "safe dispatch layer core primitive is explicitly allowed",
			importPath: "github.com/LerianStudio/lib-commons/v5/commons/dispatch-layer/core",
			wantBlock:  false,
		},
		{
			name:       "future dispatch layer core subpackage is blocked",
			importPath: "github.com/LerianStudio/lib-commons/v5/commons/dispatch-layer/core/runtime/new-shell",
			wantBlock:  true,
		},
		{
			name:       "known concrete tenant manager shell is blocked",
			importPath: "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/rabbitmq",
			wantBlock:  true,
		},
		{
			name:       "known concrete dispatch layer shell is blocked",
			importPath: "github.com/LerianStudio/lib-commons/v5/commons/dispatch-layer/middleware",
			wantBlock:  true,
		},
		{
			name:       "future tenant manager concrete shell is blocked by prefix",
			importPath: "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/runtime/new-shell",
			wantBlock:  true,
		},
		{
			name:       "future dispatch layer concrete shell is blocked by prefix",
			importPath: "github.com/LerianStudio/lib-commons/v5/commons/dispatch-layer/runtime/new-shell",
			wantBlock:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tenantRuntimeShells.matches(tt.importPath); got != tt.wantBlock {
				t.Fatalf("tenant runtime policy match for %q = %v, want %v", tt.importPath, got, tt.wantBlock)
			}
		})
	}
}

func TestEngineDependencyBoundary_GoListRunsReadonly(t *testing.T) {
	t.Parallel()

	cmd := newEngineDependencyGraphCommand(context.Background(), mustRepositoryRoot(t))
	if !slices.Contains(cmd.Args, "./pkg/engine/...") {
		t.Fatalf("go list command must enumerate every pkg/engine subpackage, got args: %#v", cmd.Args)
	}

	goFlagsCount, goFlagsValue := goFlagsEnvStatus(cmd.Env)
	if goFlagsCount != 1 || goFlagsValue != readonlyGoFlags {
		t.Fatalf(
			"go list command must run in readonly module mode with exactly one GOFLAGS=%s, got GOFLAGS count=%d value=%q",
			readonlyGoFlags,
			goFlagsCount,
			goFlagsValue,
		)
	}
}

func TestEngineDependencyBoundary_GoListEnvReplacesInheritedGoFlags(t *testing.T) {
	t.Parallel()

	env := goListReadonlyEnv([]string{
		"GOFLAGS=-mod=mod",
		"PATH=/bin",
		"GOFLAGS=-tags=stale",
		"HOME=/tmp/fetcher-test",
	})

	goFlagsCount, goFlagsValue := goFlagsEnvStatus(env)
	if goFlagsCount != 1 || goFlagsValue != readonlyGoFlags {
		t.Fatalf(
			"go list readonly environment must replace inherited GOFLAGS with exactly one value, got GOFLAGS count=%d value=%q",
			goFlagsCount,
			goFlagsValue,
		)
	}
}

func TestClaudeGoVersionGuidance_UsesGoModAsSourceOfTruth(t *testing.T) {
	t.Parallel()

	claudePath := filepath.Join(mustRepositoryRoot(t), "CLAUDE.md")
	claude, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}

	content := string(claude)
	if !strings.Contains(content, "Go version:** Source of truth is `go.mod`") {
		t.Fatalf("CLAUDE.md must direct agents to use go.mod as the Go version source of truth")
	}

	staleGoVersionPattern := regexp.MustCompile(`(?im)go\s+(version|toolchain|minimum|required|recommended)[^\n]*\b1\.[0-9]+(?:\.[0-9]+)?\b|\bgo\s*1\.[0-9]+(?:\.[0-9]+)?\b`)
	if match := staleGoVersionPattern.FindString(content); match != "" {
		t.Fatalf("CLAUDE.md must not reintroduce stale literal Go/toolchain guidance; found %q", match)
	}
}

func configuredForbiddenDependencyClasses(modulePath string) []forbiddenDependencyClass {
	return []forbiddenDependencyClass{
		// Service internals: Engine must not reach into compatibility shells.
		{
			concern: "service_internals",
			name:    "manager_internals",
			patterns: []string{
				modulePath + "/components/manager/internal",
			},
		},
		{
			concern: "service_internals",
			name:    "worker_internals",
			patterns: []string{
				modulePath + "/components/worker/internal",
			},
		},

		// HTTP shell: HTTP transport and documentation stay outside Engine core.
		{
			concern: "http",
			name:    "fiber",
			patterns: []string{
				"github.com/gofiber/fiber",
			},
		},
		{
			concern: "http",
			name:    "swagger",
			patterns: []string{
				"github.com/swaggo/swag",
				"github.com/swaggo/fiber-swagger",
			},
		},

		// Queue shell: brokers are optional adapters, not Engine core dependencies.
		{
			concern: "queue",
			name:    "rabbitmq",
			patterns: []string{
				"github.com/rabbitmq/amqp091-go",
			},
		},
		{
			concern: "queue",
			name:    "lib_streaming",
			patterns: []string{
				"github.com/LerianStudio/lib-streaming",
			},
		},

		// State stores: concrete persistence/cache clients stay behind ports.
		{
			concern: "state_stores",
			name:    "mongodb",
			patterns: []string{
				"go.mongodb.org/mongo-driver",
			},
		},
		{
			concern: "state_stores",
			name:    "redis",
			patterns: []string{
				"github.com/redis/go-redis",
			},
		},
		{
			concern: "state_stores",
			name:    "external_sql_drivers",
			patterns: []string{
				"github.com/jackc/pgx",
				"github.com/go-sql-driver/mysql",
				"github.com/microsoft/go-mssqldb",
				"github.com/sijms/go-ora",
				"github.com/lib/pq",
			},
		},
		{
			concern: "stdlib_shells",
			name:    "stdlib_persistence_shells",
			patterns: []string{
				"database/sql",
			},
		},
		{
			concern: "stdlib_shells",
			name:    "stdlib_process_shells",
			patterns: []string{
				"os/exec",
				"plugin",
			},
		},
		{
			concern: "stdlib_shells",
			name:    "stdlib_network_shells",
			patterns: []string{
				"net/http",
				"net/rpc",
			},
		},

		// Storage shell: result sinks are optional adapters, not Engine core.
		{
			concern: "storage",
			name:    "aws_s3",
			patterns: []string{
				"github.com/aws/aws-sdk-go-v2",
			},
		},
		{
			concern: "storage",
			name:    "seaweedfs",
			patterns: []string{
				modulePath + "/pkg/seaweedfs",
			},
		},
		{
			concern: "deployment",
			name:    "external_deployment_runtime",
			patterns: []string{
				"github.com/docker/docker",
				"github.com/docker/compose",
				"github.com/kedacore/keda",
				"github.com/kedacore/keda/v2",
				"sigs.k8s.io/keda",
			},
		},

		// Local shell packages: Engine core may depend on ports/primitives only, not concrete adapters/factories.
		{
			concern: "local_shells",
			name:    "local_infrastructure_shells",
			patterns: []string{
				modulePath + "/pkg/rabbitmq",
				modulePath + "/pkg/storage",
				modulePath + "/pkg/mongodb",
				modulePath + "/pkg/redis",
				modulePath + "/pkg/net/http",
				modulePath + "/pkg/seaweedfs",
				modulePath + "/pkg/postgres",
				modulePath + "/pkg/mysql",
				modulePath + "/pkg/oracle",
				modulePath + "/pkg/sqlserver",
				modulePath + "/pkg/datasource",
				modulePath + "/pkg/ratelimit",
				modulePath + "/pkg/bootstrap/readyz",
			},
		},

		// Tenant runtime shells: only safe tenant core primitives may cross the Engine boundary.
		// Concrete tenant-manager/dispatch-layer runtime, middleware, storage, queue, cache,
		// client, and any future shell packages stay outside Engine core by prefix policy.
		{
			concern: "tenant_runtime",
			name:    "tenant_runtime_shells",
			patterns: []string{
				"github.com/LerianStudio/lib-commons/v5/commons/dispatch-layer",
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager",
			},
			allowedPatterns: []string{
				"github.com/LerianStudio/lib-commons/v5/commons/dispatch-layer/core",
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core",
			},
		},

		// Deployment shell: infrastructure artifacts must not become runtime core.
		{
			concern: "deployment",
			name:    "deployment",
			patterns: []string{
				modulePath + "/components/infra",
				modulePath + "/deployments",
				modulePath + "/helm",
			},
		},

		// Auth shell: host authorization stays outside Engine core.
		{
			concern: "auth",
			name:    "lib_auth",
			patterns: []string{
				"github.com/LerianStudio/lib-auth",
			},
		},

		// License shell: host licensing stays outside Engine core.
		{
			concern: "license",
			name:    "lib_license",
			patterns: []string{
				"github.com/LerianStudio/lib-license-go",
			},
		},
	}
}

func requiredForbiddenClassNames() []string {
	return []string{
		"manager_internals",
		"worker_internals",
		"fiber",
		"swagger",
		"rabbitmq",
		"lib_streaming",
		"mongodb",
		"redis",
		"external_sql_drivers",
		"stdlib_persistence_shells",
		"stdlib_process_shells",
		"stdlib_network_shells",
		"aws_s3",
		"seaweedfs",
		"external_deployment_runtime",
		"local_infrastructure_shells",
		"tenant_runtime_shells",
		"deployment",
		"lib_auth",
		"lib_license",
	}
}

func requiredTenantRuntimeShellPatterns() []string {
	return []string{
		"github.com/LerianStudio/lib-commons/v5/commons/dispatch-layer",
		"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager",
	}
}

func requiredExternalDeploymentRuntimePatterns() []string {
	return []string{
		"github.com/docker/docker",
		"github.com/docker/compose",
		"github.com/kedacore/keda",
		"github.com/kedacore/keda/v2",
	}
}

func mustForbiddenDependencyClass(t *testing.T, configuredClasses []forbiddenDependencyClass, name string) forbiddenDependencyClass {
	t.Helper()

	for _, configuredClass := range configuredClasses {
		if configuredClass.name == name {
			return configuredClass
		}
	}

	t.Fatalf("forbidden dependency class %q is not configured", name)
	return forbiddenDependencyClass{}
}

func assertForbiddenConfigComplete(t *testing.T, requiredClassNames []string, configuredClasses []forbiddenDependencyClass) {
	t.Helper()

	configuredNames := make(map[string]struct{}, len(configuredClasses))
	for _, configuredClass := range configuredClasses {
		if configuredClass.concern == "" {
			t.Fatalf("forbidden dependency class %q has no shell concern", configuredClass.name)
		}
		configuredNames[configuredClass.name] = struct{}{}
		if len(configuredClass.patterns) == 0 {
			t.Fatalf("forbidden dependency class %q has no import patterns", configuredClass.name)
		}
	}

	missing := make([]string, 0)
	for _, requiredClassName := range requiredClassNames {
		if _, ok := configuredNames[requiredClassName]; !ok {
			missing = append(missing, requiredClassName)
		}
	}

	if len(missing) > 0 {
		slices.Sort(missing)
		t.Fatalf("dependency boundary test is incomplete: missing forbidden dependency classes: %s", strings.Join(missing, ", "))
	}
}

func collectEngineDependencyGraph(t *testing.T, repoRoot string) []string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := newEngineDependencyGraphCommand(ctx, repoRoot)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("collect pkg/engine dependency graph with go list -deps: %v\n%s", err, output)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	dependencies := make([]string, 0, len(lines))
	for _, line := range lines {
		dependency := strings.TrimSpace(line)
		if dependency != "" {
			dependencies = append(dependencies, dependency)
		}
	}

	return dependencies
}

func newEngineDependencyGraphCommand(ctx context.Context, repoRoot string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "go", "list", "-deps", "-f", "{{.ImportPath}}", "./pkg/engine/...")
	cmd.Dir = repoRoot
	cmd.Env = goListReadonlyEnv(os.Environ())

	return cmd
}

func goListReadonlyEnv(environ []string) []string {
	env := make([]string, 0, len(environ)+1)
	for _, envVar := range environ {
		if strings.HasPrefix(envVar, "GOFLAGS=") {
			continue
		}
		env = append(env, envVar)
	}

	return append(env, "GOFLAGS="+readonlyGoFlags)
}

func goFlagsEnvStatus(environ []string) (int, string) {
	count := 0
	value := ""
	for _, envVar := range environ {
		goFlags, ok := strings.CutPrefix(envVar, "GOFLAGS=")
		if !ok {
			continue
		}
		count++
		value = goFlags
	}

	return count, value
}

func mustRepositoryRoot(t *testing.T) string {
	t.Helper()

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	for current := workingDir; ; current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		if parent := filepath.Dir(current); parent == current {
			t.Fatalf("could not locate repository root from %q", workingDir)
		}
	}
}

func mustModulePathFromGoMod(t *testing.T, repoRoot string) string {
	t.Helper()

	goModPath := filepath.Join(repoRoot, "go.mod")
	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	match := regexp.MustCompile(`(?m)^module\s+(\S+)`).FindSubmatch(goMod)
	if len(match) != 2 {
		t.Fatalf("go.mod does not contain a module directive")
	}

	return string(match[1])
}

func (dependencyClass forbiddenDependencyClass) matches(importPath string) bool {
	for _, pattern := range dependencyClass.patterns {
		if importPath == pattern || strings.HasPrefix(importPath, pattern+"/") {
			for _, allowedPattern := range dependencyClass.allowedPatterns {
				if importPath == allowedPattern {
					return false
				}
			}

			return true
		}
	}

	return false
}
