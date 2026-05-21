// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package engine

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
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
	requiredClassNames := requiredForbiddenClassNames(t, repoRoot)
	configuredClasses := configuredForbiddenDependencyClasses(modulePath)

	assertForbiddenConfigComplete(t, requiredClassNames, configuredClasses)

	imports := collectEngineImports(t, repoRoot)
	for _, dependencyClass := range configuredClasses {
		dependencyClass := dependencyClass
		t.Run(dependencyClass.name, func(t *testing.T) {
			t.Parallel()

			for sourcePath, importPaths := range imports {
				for _, importPath := range importPaths {
					if dependencyClass.matches(importPath) {
						t.Fatalf("pkg/engine dependency boundary violation: %s imports forbidden %s dependency %q", sourcePath, dependencyClass.name, importPath)
					}
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
		if dependencyClass.name != "manager_internals" {
			continue
		}
		if !slices.Contains(dependencyClass.patterns, modulePath+"/components/manager/internal") {
			t.Fatalf("manager internal boundary pattern does not use module path read from go.mod: %#v", dependencyClass.patterns)
		}
		return
	}

	t.Fatalf("manager internal dependency boundary class is not configured")
}

func TestEngineDependencyBoundary_Scope_AllowsEngineCompatibilityAdaptersOutsideEngine(t *testing.T) {
	t.Parallel()

	repoRoot := mustRepositoryRoot(t)
	engineRoot := filepath.Join(repoRoot, "pkg", "engine")
	optionalAdapterPath := filepath.Join(repoRoot, "pkg", "enginecompat", "rabbitmq", "adapter.go")
	if isUnderDirectory(optionalAdapterPath, engineRoot) {
		t.Fatalf("engine compatibility adapter path %q should be outside pkg/engine boundary", optionalAdapterPath)
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

		// Queue shell: brokers are optional adapters, not Engine core dependencies.
		{
			concern: "queue",
			name:    "rabbitmq",
			patterns: []string{
				"github.com/rabbitmq/amqp091-go",
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

func requiredForbiddenClassNames(t *testing.T, repoRoot string) []string {
	t.Helper()

	dependencyDocPath := filepath.Join(repoRoot, "docs", "pre-dev", "fetcher-embedded-runtime", "dependencies.md")
	dependencyDoc, err := os.ReadFile(dependencyDocPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fallbackForbiddenClassNames()
		}
		t.Fatalf("read dependency map %s: %v", dependencyDocPath, err)
	}

	doc := string(dependencyDoc)
	if !strings.Contains(doc, "Engine core must not depend on:") {
		t.Fatalf("dependency map %s does not define Engine core forbidden dependencies", dependencyDocPath)
	}

	requirements := []struct {
		name     string
		evidence []string
	}{
		{name: "manager_internals", evidence: []string{"Manager or Worker `internal` packages", "components/*/internal"}},
		{name: "worker_internals", evidence: []string{"Manager or Worker `internal` packages", "components/*/internal"}},
		{name: "fiber", evidence: []string{"Fiber or Swagger", "github.com/gofiber/fiber/v2"}},
		{name: "rabbitmq", evidence: []string{"RabbitMQ/amqp091-go", "github.com/rabbitmq/amqp091-go"}},
		{name: "mongodb", evidence: []string{"MongoDB drivers", "go.mongodb.org/mongo-driver"}},
		{name: "redis", evidence: []string{"Redis as mandatory", "github.com/redis/go-redis/v9"}},
		{name: "aws_s3", evidence: []string{"S3/SeaweedFS", "github.com/aws/aws-sdk-go-v2/service/s3"}},
		{name: "seaweedfs", evidence: []string{"S3/SeaweedFS", "SeaweedFS local client"}},
		{name: "deployment", evidence: []string{"Docker Compose, KEDA, or deployment artifacts"}},
		{name: "lib_auth", evidence: []string{"lib-auth", "github.com/LerianStudio/lib-auth/v2"}},
		{name: "lib_license", evidence: []string{"lib-license", "github.com/LerianStudio/lib-license-go/v2"}},
	}

	names := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		if !containsAny(doc, requirement.evidence) {
			t.Fatalf("dependency map %s is missing evidence for forbidden dependency class %q", dependencyDocPath, requirement.name)
		}
		names = append(names, requirement.name)
	}

	return names
}

func fallbackForbiddenClassNames() []string {
	return []string{
		"manager_internals",
		"worker_internals",
		"fiber",
		"rabbitmq",
		"mongodb",
		"redis",
		"aws_s3",
		"seaweedfs",
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

func collectEngineImports(t *testing.T, repoRoot string) map[string][]string {
	t.Helper()

	engineRoot := filepath.Join(repoRoot, "pkg", "engine")
	importsBySource := make(map[string][]string)
	fset := token.NewFileSet()

	err := filepath.WalkDir(engineRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		parsedFile, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}

		for _, importedPackage := range parsedFile.Imports {
			importsBySource[relativePath] = append(importsBySource[relativePath], strings.Trim(importedPackage.Path.Value, "\""))
		}

		return nil
	})
	if err != nil {
		t.Fatalf("collect pkg/engine imports: %v", err)
	}

	return importsBySource
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

func containsAny(value string, candidates []string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}

	return false
}

func isUnderDirectory(path string, directory string) bool {
	relativePath, err := filepath.Rel(directory, path)
	if err != nil {
		return false
	}

	return relativePath == "." || (!strings.HasPrefix(relativePath, "..") && !filepath.IsAbs(relativePath))
}
