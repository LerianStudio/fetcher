// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

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
	concern  string
	name     string
	patterns []string
}

func TestEngineDependencyBoundary_BlocksForbiddenImports(t *testing.T) {
	t.Parallel()

	repoRoot := mustRepositoryRoot(t)
	modulePath := mustModulePathFromGoMod(t, repoRoot)
	requiredClassNames := requiredForbiddenClassNames()
	configuredClasses := configuredForbiddenDependencyClasses(modulePath)

	assertForbiddenConfigComplete(t, requiredClassNames, configuredClasses)

	dependencies := collectEngineDependencyGraph(t, repoRoot, modulePath)
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

		// Tenant runtime shells: tenant primitives may become ports, but concrete runtime/middleware/managers stay outside Engine core.
		{
			concern: "tenant_runtime",
			name:    "tenant_runtime_shells",
			patterns: []string{
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client",
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/consumer",
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/event",
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/middleware",
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/mongo",
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/rabbitmq",
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/s3",
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/tenantcache",
				"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/valkey",
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
		"aws_s3",
		"seaweedfs",
		"local_infrastructure_shells",
		"tenant_runtime_shells",
		"deployment",
		"lib_auth",
		"lib_license",
	}
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

func collectEngineDependencyGraph(t *testing.T, repoRoot string, modulePath string) []string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "list", "-deps", "-f", "{{.ImportPath}}", modulePath+"/pkg/engine")
	cmd.Dir = repoRoot

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
			return true
		}
	}

	return false
}
