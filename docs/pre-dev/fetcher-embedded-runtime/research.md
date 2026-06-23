---
feature: fetcher-embedded-runtime
gate: 0
status: COMPLETED
track: Full
repository: /Users/fredamaral/repos/lerianstudio/fetcher
topology:
  current: Manager HTTP API plus Worker RabbitMQ consumer
  target: Importable Fetcher Engine with current services as compatibility adapters
  source_of_truth:
    architecture_rules: docs/PROJECT_RULES.md
    go_toolchain: go.mod
ui: none
---

# Gate 0 Research: Fetcher Embedded Runtime

## Codebase Patterns

Fetcher is already a developed service, not a greenfield library. The module is `github.com/LerianStudio/fetcher` and `go.mod` currently declares Go `1.25.9` (`go.mod:1-3`). That conflicts with the architecture docs, which still say Go `1.25.6` (`docs/PROJECT_RULES.md:11`). Gate 7 treats `go.mod` as the source of truth and adds a first-task implementation brief to reconcile documentation or tooling before Engine code starts.

The current product shape is Manager plus Worker. Manager owns HTTP routes, auth/license middleware, health/readiness/metrics, telemetry middleware, and Fiber routing (`components/manager/internal/adapters/http/in/routes.go:23-51`, `components/manager/internal/adapters/http/in/routes.go:53-92`). Worker owns asynchronous extraction through a `UseCase`, with queue-specific message parsing and job state mutation (`components/worker/internal/services/service.go:16-59`, `components/worker/internal/services/extract_data.go:51-137`).

The Manager composition root is locked under `components/manager/internal`. It loads env config, validates SaaS TLS, opens Mongo/Rabbit/Redis/platform dependencies, assembles commands and queries, and mounts HTTP routes (`components/manager/internal/bootstrap/config.go:199-255`, `components/manager/internal/bootstrap/config.go:550-706`). This is the central reason embedded usage cannot import the current behavior cleanly.

Connection lifecycle behavior already exists. `CreateConnection` builds the domain model, encrypts credentials, enforces config-name uniqueness, and persists through a connection repository (`components/manager/internal/services/command/create_connection.go:31-96`). The domain model carries product isolation, datasource type, host, port, database/schema, username, encrypted password, key version, SSL, metadata, and timestamps (`pkg/model/connection.go:18-36`).

Job creation behavior already exists. `CreateFetcherJob` computes request hash, deduplicates within a five-minute window, resolves connections, validates product ownership, tests connections, persists the job, and publishes to RabbitMQ (`components/manager/internal/services/command/create_fetcher_job.go:31-64`, `components/manager/internal/services/command/create_fetcher_job.go:123-191`). The current job model has status, request hash, mapped fields, filters, result path, HMAC, and timestamps (`pkg/model/job.go:35-77`, `pkg/model/job.go:255-297`).

Schema discovery and validation already exist. Validation resolves connections, applies request limits, reads schema from cache or datasource, handles default schema conventions, transforms `plugin_crm`, and returns detailed validation errors (`components/manager/internal/services/query/validate_schema.go:35-65`, `components/manager/internal/services/query/validate_schema.go:80-235`). Direct schema discovery creates a datasource and calls `GetSchemaInfo`, then filters system tables before returning a response (`components/manager/internal/services/query/get_connection_schema.go:54-152`).

Datasource capability is currently represented by `DataSource`, with `Connect`, `Close`, `Query`, and `GetSchemaInfo` (`pkg/model/datasource/datasource-config.go:28-50`). The current factory is a switch over supported DB types and includes password resolution plus early connection side effects, including MongoDB ping during construction (`pkg/datasource/datasource_factory.go:35-103`, `pkg/datasource/datasource_factory.go:139-185`, `pkg/datasource/datasource_factory.go:238-260`). This is useful behavior, but it is too concrete for Engine core as-is.

Extraction lifecycle already exists in the Worker. It parses the message, skips non-pending jobs, resolves connections, queries each datasource, encrypts JSON results, writes to storage, updates job status, and publishes optional completion/failure notifications (`components/worker/internal/services/extract_data.go:51-137`, `components/worker/internal/services/extract_data.go:383-575`). Event publication is already optional when no RabbitMQ publisher or exchange is configured (`components/worker/internal/services/job_notification.go:78-95`).

Ports exist but are still shaped by the service. Connection, job, storage, cache, and publisher ports are present (`pkg/ports/connection/repository.go:14-28`, `pkg/ports/job/repository.go:27-39`, `pkg/ports/storage/repository.go:8-14`, `pkg/ports/cache/repository.go:14-39`, `pkg/ports/publisher/repository.go:8-14`, `pkg/ports/messaging/publisher.go:8-15`). The Engine can reuse many of these ideas, but should not inherit service-only assumptions such as MongoDB-backed jobs or RabbitMQ enqueue as mandatory runtime semantics.

Current adapters are clear boundaries. MongoDB repositories and datasource implementations live under `pkg/mongodb/*`; storage selection is already provider-based (`pkg/storage/factory.go:20-91`); SeaweedFS and S3 are concrete storage adapters (`pkg/seaweedfs/external/external_data.go:20-99`, `pkg/storage/s3.go:25-107`); RabbitMQ is a resilient adapter with retry/circuit-breaker/signing concerns (`pkg/rabbitmq/rabbitmq.go:119-195`); crypto provides AES-GCM credential encryption (`pkg/crypto/crypto.go:16-23`, `pkg/crypto/crypto.go:30-60`); resolver handles internal/external datasource resolution (`pkg/resolver/resolver.go:9-37`, `pkg/resolver/registry.go:30-45`, `pkg/resolver/single_tenant.go:11-69`).

Tests already protect current behavior. Manager and Worker service tests cover command/query/extraction behavior, including `create_fetcher_job_test.go`, `validate_schema_test.go`, `get_connection_schema_test.go`, `extract_data_test.go`, and `job_notification_test.go`. E2E tests cover connection lifecycle, schema validation, job status, and extraction across PostgreSQL, MySQL, MongoDB, Oracle, SQL Server, S3, filters, multi-schema, and multi-datasource flows under `tests/e2e/*`. Fuzz tests cover connection validation, schema validation, fetcher requests, extraction messages, filters, headers, and regex extraction under `tests/fuzz/*`.

## Best Practices

Use a strangler extraction. The first-class outcome is not a new abstraction layer; it is an importable Engine composition root that can assemble existing connection, schema, planning, extraction, result, error, limit, tenant-safety, and observability capabilities without importing `components/*/internal`.

Separate core capability from operational shell. Engine core should own datasource/source semantics. Host apps should own HTTP route shape, auth, license enforcement, env loading, health/readyz, telemetry exporters, queues, state stores, object storage choices, deployment, process lifecycle, and product policy.

Keep compatibility adapters explicit. Current Manager and Worker should remain usable as optional adapters over the Engine. Their current Fiber, RabbitMQ, MongoDB, Redis, S3/SeaweedFS, Docker Compose, and KEDA dependencies must stay adapter/shell dependencies, not Engine core dependencies.

Prefer ports that model business capabilities, not infrastructure products. `ConnectionStore`, `ExecutionStore`, `ResultSink`, `EventSink`, `SchemaCache`, `ConnectorRegistry`, and `CredentialProtector` are safer concepts than MongoDB, RabbitMQ, Redis, or SeaweedFS in core API design.

Preserve tenant and data ownership semantics. Tenant/product context must be explicit in Engine requests or carried through a documented execution context contract. The Engine must never require extracted data to leave the host deployment boundary.

## Framework Constraints

Go `internal` package rules are the hard blocker. Host applications cannot import reusable behavior currently under `components/manager/internal` and `components/worker/internal`. The importable composition root must live outside `components/*/internal`.

Fiber, lib-auth, and lib-license are Manager adapter concerns. Routes currently apply lib-auth authorization and optional tenant middleware to HTTP endpoints (`components/manager/internal/adapters/http/in/routes.go:76-92`). The Engine must expose contracts that the adapter can protect, not take middleware dependencies.

RabbitMQ is an execution transport, not an Engine requirement. Current job creation publishes messages (`components/manager/internal/services/command/create_fetcher_job.go:183-191`), and Worker consumes them, but embedded hosts must be able to run synchronous or host-scheduled execution without RabbitMQ.

MongoDB and Redis are current stores/caches, not Engine requirements. Current repositories and schema cache are wired in Manager bootstrap (`components/manager/internal/bootstrap/config.go:529-547`), but embedded hosts may use their own state and cache infrastructure.

Datasource creation currently includes side effects. The existing factory constructs concrete datasource implementations and may test connectivity during creation (`pkg/datasource/datasource_factory.go:168-185`, `pkg/datasource/datasource_factory.go:238-260`). Engine design should distinguish connector construction from connection testing.

## User Research

Target consumers are internal engineering teams embedding extraction into products such as Reporter and Matcher. The PRD frames the pain as deployment and integration overhead for products that already own their operational lifecycle (`prd.md:5-19`, `prd.md:59-65`).

Operations teams need fewer separately deployed components across cloud, on-premises, and self-hosted models. The deployment model is host-controlled by requirement, so Engine success is measured by removing a standalone Fetcher service from host product deployment paths (`prd.md:161-170`, `prd.md:174-183`).

Security and compliance stakeholders need consistent credential handling, tenant isolation, and data ownership. The Engine must centralize datasource semantics while leaving authorization policy to host applications (`prd.md:149-158`).

## Key Findings

The problem is not lack of abstractions. Fetcher already has domain models, ports, adapters, resolver concepts, tests, and service-layer orchestration. The problem is that the importable Engine composition root does not exist and key orchestration is locked under `components/*/internal`.

The Engine should own datasource/source capabilities: connection model, connector registry/factory, schema discovery/validation, query planning, extraction runner, canonical result model, error taxonomy, limits, observability hooks, and tenant-safety primitives.

Host apps should own the operational shell: HTTP routes, auth/license middleware, env loading, health/readyz, telemetry exporters, queues, state stores, storage choices, deployment, process lifecycle, and product-specific policy.

Current Manager and Worker are valuable compatibility adapters. They should be refactored to call the Engine rather than deleted or rewritten wholesale.

## Risks & Unknowns

The first migration slice can easily become too broad. Gate 7 should force the first slice to create the importable composition root and move only the minimum reusable orchestration needed to prove embedded extraction.

Compatibility guarantees are now first-release constraints. Current API behavior, job status semantics, request-hash dedupe with the five-minute window, optional notifications, event payloads, error mapping, and storage path/HMAC behavior are preserved unless explicitly migrated.

Engine package boundaries are fixed for the implementation track: Engine core lives under `pkg/engine`, and optional compatibility wrappers live under `pkg/enginecompat/*`.

Credential responsibility is split: Engine owns credential protection semantics through `CredentialProtector`; host applications own key provisioning and rotation policy. The remaining implementation detail is the exact key-provider adapter shape.

Plugin-specific behavior is still partly hardcoded. `plugin_crm` exists in schema validation and extraction flows (`components/manager/internal/services/query/validate_schema.go:28-31`, `components/worker/internal/services/extract_data.go:463-470`). Gate 7 decides that the first Engine release preserves it as adapter-level compatibility behavior only, not as a generic Engine core extension.
