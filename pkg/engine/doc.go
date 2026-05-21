// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package engine contains the embedded runtime seam contracts that will
// strangle the existing Fetcher manager and worker orchestration paths.
package engine

// Seam describes an existing runtime boundary that the embedded Engine must
// preserve while it replaces behavior incrementally.
type Seam struct {
	Name       string
	SourcePath string
	Reason     string
}

// RequiredSeams returns the characterized runtime seams.
func RequiredSeams() []Seam {
	return []Seam{
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
			SourcePath: "components/worker/internal/bootstrap/consumer.go; components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go",
			Reason:     "tenant identifiers are propagated through queue consumption and notification publishing shells and must remain outside Engine core transport contracts",
		},
		{
			Name:       "storage_cache_tenant_scoping",
			SourcePath: "pkg/storage/s3.go; pkg/seaweedfs/external/external_data.go; pkg/redis/redis_cache.go",
			Reason:     "tenant scoping for object storage keys and cache access is implemented in storage and cache adapters, not in generic Engine core persistence logic",
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
