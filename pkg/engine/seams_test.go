// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seam describes an existing runtime boundary that the embedded Engine must
// preserve while it replaces behavior incrementally. It intentionally remains
// test-local until T-002 introduces a production Engine API that consumes it.
type seam struct {
	Name       string
	SourcePath string
	Reason     string
}

func requiredSeams() []seam {
	return []seam{
		{
			Name:       "datasource_factory_side_effects",
			SourcePath: "pkg/datasource/datasource_factory.go",
			Reason:     "datasource construction currently decrypts credentials, validates SSL options, tests live connections, and creates adapter repositories as side effects",
		},
		{
			Name:       "manager_create_fetcher_job_orchestration",
			SourcePath: "components/manager/internal/services/command/create_fetcher_job.go",
			Reason:     "manager job creation owns deduplication, connection resolution, ownership validation, connection testing, persistence, and queue publishing orchestration",
		},
		{
			Name:       "manager_validate_schema_orchestration",
			SourcePath: "components/manager/internal/services/query/validate_schema.go",
			Reason:     "schema validation resolves configured datasources, fetches schema metadata, applies datasource-specific defaults, and preserves legacy table compatibility",
		},
		{
			Name:       "worker_extract_external_data_orchestration",
			SourcePath: "components/worker/internal/services/extract_data.go",
			Reason:     "worker extraction owns message parsing, idempotent job state transitions, connection resolution, datasource querying, storage persistence, and completion flow",
		},
		{
			Name:       "worker_plugin_crm_compatibility",
			SourcePath: "components/worker/internal/services/extract_crm_data.go",
			Reason:     "plugin_crm compatibility extraction remains Worker adapter-scoped behavior for physical collection prefix matching, filter transformation, field decryption, and merged logical collection output",
		},
		{
			Name:       "plugin_crm_adapter_compatibility",
			SourcePath: "components/manager/internal/services/query/validate_schema.go; components/worker/internal/services/extract_crm_data.go",
			Reason:     "plugin_crm remains Manager/Worker compatibility behavior for the first Engine release and must remain adapter-specific, never core",
		},
		{
			Name:       "connection_resolver_behavior",
			SourcePath: "pkg/resolver/resolver.go",
			Reason:     "connection resolution abstracts internal tenant-managed datasources and external configured connections behind a stable ResolveConnections seam",
		},
		{
			Name:       "tenant_context_product_boundary",
			SourcePath: "components/manager/internal/services/command/create_fetcher_job.go; components/manager/internal/services/query/get_connection.go; components/manager/internal/services/query/get_connection_schema.go",
			Reason:     "tenant context and product ownership boundaries are enforced before Engine orchestration may resolve connections, read schema, or create jobs",
		},
		{
			Name:       "queue_tenant_propagation",
			SourcePath: "components/manager/internal/services/command/create_fetcher_job.go; components/worker/internal/bootstrap/consumer.go; components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go",
			Reason:     "tenant identifiers are propagated from Manager publish path through X-Tenant-ID AMQP headers, worker queue consume paths, and notification publishing shells and must remain outside Engine core transport contracts",
		},
		{
			Name:       "storage_cache_tenant_scoping",
			SourcePath: "pkg/storage/s3.go; pkg/seaweedfs/external/external_data.go; pkg/redis/redis_cache.go; pkg/redis/fallback_cache.go; pkg/redis/memory_cache.go",
			Reason:     "tenant scoping for object storage keys and cache access is implemented in storage and cache adapters, including fallback and in-memory cache paths where keys must already be tenant-scoped before use, not in generic Engine core persistence logic",
		},
		{
			Name:       "storage_result_handling",
			SourcePath: "components/worker/internal/services/extract_data.go",
			Reason:     "extracted results are marshaled, encrypted, written to configured object storage, and converted into result path and HMAC metadata for job updates",
		},
		{
			Name:       "notification_publishing",
			SourcePath: "components/worker/internal/services/job_notification.go",
			Reason:     "job status notifications are serialized with source metadata, routed by status and product, and published to the RabbitMQ job events exchange",
		},
	}
}

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
			sourcePath: "components/manager/internal/services/command/create_fetcher_job.go; components/worker/internal/bootstrap/consumer.go; components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go",
			keywords:   []string{"tenant", "X-Tenant-ID", "AMQP headers", "publish", "consume", "notification"},
		},
		"storage_cache_tenant_scoping": {
			sourcePath: "pkg/storage/s3.go; pkg/seaweedfs/external/external_data.go; pkg/redis/redis_cache.go; pkg/redis/fallback_cache.go; pkg/redis/memory_cache.go",
			keywords:   []string{"tenant", "storage", "cache", "fallback", "in-memory", "tenant-scoped before use"},
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
	for _, seam := range requiredSeams() {
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

	for _, seam := range requiredSeams() {
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

func seamsByName(t *testing.T) map[string]seam {
	t.Helper()

	actual := requiredSeams()
	seams := make(map[string]seam, len(actual))
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
