// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequiredSeams_Characterization_IsComplete(t *testing.T) {
	t.Parallel()

	required := map[string]struct {
		sourcePath string
		keywords   []string
	}{
		"datasource_factory_side_effects": {
			sourcePath: "pkg/datasource/datasource_factory.go",
			keywords:   []string{"datasource", "credentials", "adapter"},
		},
		"manager_create_fetcher_job_orchestration": {
			sourcePath: "components/manager/internal/services/command/create_fetcher_job.go",
			keywords:   []string{"deduplication", "ownership", "queue"},
		},
		"manager_validate_schema_orchestration": {
			sourcePath: "components/manager/internal/services/query/validate_schema.go",
			keywords:   []string{"schema", "datasource", "legacy"},
		},
		"worker_extract_external_data_orchestration": {
			sourcePath: "components/worker/internal/services/extract_data.go",
			keywords:   []string{"message", "datasource", "storage"},
		},
		"connection_resolver_behavior": {
			sourcePath: "pkg/resolver/resolver.go",
			keywords:   []string{"tenant", "connections", "ResolveConnections"},
		},
		"tenant_context_product_boundary": {
			sourcePath: "components/manager/internal/services/command/create_fetcher_job.go; components/manager/internal/services/query/get_connection.go; components/manager/internal/services/query/get_connection_schema.go",
			keywords:   []string{"tenant", "product", "ownership"},
		},
		"queue_tenant_propagation": {
			sourcePath: "components/worker/internal/bootstrap/consumer.go; components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go",
			keywords:   []string{"tenant", "queue", "publishing"},
		},
		"storage_cache_tenant_scoping": {
			sourcePath: "pkg/storage/s3.go; pkg/seaweedfs/external/external_data.go; pkg/redis/redis_cache.go",
			keywords:   []string{"tenant", "storage", "cache"},
		},
		"storage_result_handling": {
			sourcePath: "components/worker/internal/services/extract_data.go",
			keywords:   []string{"encrypted", "storage", "HMAC"},
		},
		"worker_plugin_crm_compatibility": {
			sourcePath: "components/worker/internal/services/extract_crm_data.go",
			keywords:   []string{"plugin_crm", "compatibility", "Worker"},
		},
		"plugin_crm_adapter_compatibility": {
			sourcePath: "components/manager/internal/services/query/validate_schema.go; components/worker/internal/services/extract_crm_data.go",
			keywords:   []string{"plugin_crm", "Manager/Worker", "compatibility"},
		},
		"notification_publishing": {
			sourcePath: "components/worker/internal/services/job_notification.go",
			keywords:   []string{"notifications", "product", "RabbitMQ"},
		},
	}

	seamsByName := seamsByName(t)
	for name, want := range required {
		name, want := name, want
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, ok := seamsByName[name]
			if !ok {
				t.Fatalf("missing runtime seam characterization %q", name)
			}
			if got.SourcePath != want.sourcePath {
				t.Fatalf("runtime seam characterization %q source path = %q, want %q", name, got.SourcePath, want.sourcePath)
			}
			if got.Reason == "" {
				t.Fatalf("runtime seam characterization %q reason is empty", name)
			}
			for _, keyword := range want.keywords {
				if !strings.Contains(got.Reason, keyword) {
					t.Fatalf("runtime seam characterization %q reason = %q, want keyword %q", name, got.Reason, keyword)
				}
			}
		})
	}
}

func TestRequiredSeams_SourcePathsExist(t *testing.T) {
	t.Parallel()

	repoRoot := mustRepositoryRoot(t)
	for _, seam := range RequiredSeams() {
		seam := seam
		t.Run(seam.Name, func(t *testing.T) {
			t.Parallel()

			for _, sourcePath := range splitSourcePaths(seam.SourcePath) {
				resolvedPath := filepath.Join(repoRoot, filepath.FromSlash(sourcePath))
				if _, err := os.Stat(resolvedPath); err != nil {
					t.Fatalf("runtime seam characterization %q references stale/missing source path %q: %v", seam.Name, sourcePath, err)
				}
			}
		})
	}
}

func TestRequiredSeams_PluginCRMCompatibility_IsAdapterScoped(t *testing.T) {
	t.Parallel()

	for _, seam := range RequiredSeams() {
		seam := seam
		if !strings.Contains(strings.ToLower(seam.Name+" "+seam.Reason), "crm") {
			continue
		}

		t.Run(seam.Name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(seam.Name, "compatibility") && !strings.Contains(seam.Name, "adapter") {
				t.Fatalf("CRM seam %q must be named as adapter/compatibility scoped", seam.Name)
			}

			for _, sourcePath := range splitSourcePaths(seam.SourcePath) {
				if !strings.HasPrefix(sourcePath, "components/manager/") && !strings.HasPrefix(sourcePath, "components/worker/") {
					t.Fatalf("CRM seam %q source path %q must remain Manager/Worker adapter scoped", seam.Name, sourcePath)
				}
			}

			lowerReason := strings.ToLower(seam.Reason)
			if !strings.Contains(lowerReason, "compatibility") {
				t.Fatalf("CRM seam %q reason = %q, want explicit compatibility scope", seam.Name, seam.Reason)
			}
			if strings.Contains(lowerReason, "generic engine datasource") || strings.Contains(lowerReason, "core datasource extension") {
				t.Fatalf("CRM seam %q reason must not position CRM as a generic Engine/core datasource extension: %q", seam.Name, seam.Reason)
			}
		})
	}
}

func seamsByName(t *testing.T) map[string]Seam {
	t.Helper()

	actual := RequiredSeams()
	seams := make(map[string]Seam, len(actual))
	for _, seam := range actual {
		if seam.Name == "" {
			t.Fatalf("runtime seam characterization contains an empty seam name: %#v", seam)
		}
		if seam.SourcePath == "" {
			t.Fatalf("runtime seam characterization %q has an empty source path", seam.Name)
		}
		if seam.Reason == "" {
			t.Fatalf("runtime seam characterization %q has an empty reason", seam.Name)
		}
		if _, exists := seams[seam.Name]; exists {
			t.Fatalf("runtime seam characterization %q is duplicated", seam.Name)
		}

		seams[seam.Name] = seam
	}

	return seams
}

func splitSourcePaths(sourcePath string) []string {
	parts := strings.Split(sourcePath, ";")
	sourcePaths := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			sourcePaths = append(sourcePaths, trimmed)
		}
	}

	return sourcePaths
}
