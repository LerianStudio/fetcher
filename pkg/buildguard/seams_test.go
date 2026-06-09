// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package buildguard

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
			Name:       "storage_tenant_scoping",
			SourcePath: "pkg/storage/s3.go; pkg/seaweedfs/external/external_data.go",
			Reason:     "tenant scoping for object storage keys is implemented in storage adapters where object keys must already be tenant-scoped before use, not in generic Engine core persistence logic",
		},
		{
			Name:       "schema_cache_tenant_scoping",
			SourcePath: "components/manager/internal/services/query/validate_schema.go; components/manager/internal/adapters/cache/schema_cache.go; components/manager/internal/adapters/cache/schema_cache_interface.go; pkg/redis/factory.go; pkg/redis/redis_cache.go; pkg/redis/fallback_cache.go; pkg/redis/memory_cache.go",
			Reason:     "schema cache access remains a Manager adapter seam: validate_schema orchestration, schema cache adapters, Redis factory selection, Redis cache, fallback cache, and memory cache paths must use raw keys only after tenant-scoped-before-use handling so fallback and in-memory cache behavior cannot leak tenant schema data into Engine core",
		},
		{
			Name:       "storage_result_handling",
			SourcePath: "components/worker/internal/services/extract_data.go",
			Reason:     "extracted results are marshaled, encrypted, written to configured object storage, and converted into result path and HMAC metadata for job updates",
		},
		{
			Name:       "notification_publishing",
			SourcePath: "components/worker/internal/services/job_notification.go",
			Reason:     "job status notifications are serialized with source in payload metadata and emitted through the mandatory lib-streaming contract using stable job.completed/job.failed event keys, never product/source routing-key segments",
		},
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
