# Gate 3 TRD: Fetcher Embedded Runtime

## Technical Thesis

Fetcher Embedded Runtime is a strangler extraction of an importable Engine from existing Manager and Worker internals. The goal is not to invent a larger abstraction hierarchy. The goal is to move reusable orchestration behind an importable composition root while leaving operational shell concerns in host apps and compatibility adapters.

## Current-State Summary

Manager currently wires commands, queries, repositories, cryptography, datasource factory, resolver, schema cache, RabbitMQ publisher, auth/license clients, readiness, telemetry, and Fiber routes under `components/manager/internal/bootstrap` (`components/manager/internal/bootstrap/config.go:550-706`).

Worker currently wires extraction dependencies into `UseCase`, including storage, job repository, connection repository, cryptor, document signer, RabbitMQ publisher, datasource factory, and connection resolver (`components/worker/internal/services/service.go:16-59`).

Reusable behavior exists, but the import boundary is wrong. Host apps cannot import Manager/Worker internals, and importing the current app shell would drag Fiber, RabbitMQ, MongoDB, Redis, env loading, and process lifecycle into embedded use.

## Target Architecture

Create an importable Engine layer outside `components/*/internal`. Gate 7 fixes the implementation package boundary as `pkg/engine` for Engine core and `pkg/enginecompat/*` for optional compatibility wrappers, while the capability boundary remains the architectural source of truth.

Engine core provides:

- Embedded operation surface for connection lifecycle, schema discovery, schema validation, planning, and extraction execution.
- Composition root that accepts host-provided ports and optional adapters.
- Connector registry/factory for supported datasource connectors.
- Canonical request, execution, result, schema, and error contracts.
- Limits, cancellation, tenant-safety primitives, and observability hooks.

Compatibility adapters provide:

- Manager HTTP adapter over Engine operations.
- Worker queue adapter over Engine execution.
- MongoDB/Redis/RabbitMQ/S3/SeaweedFS/Fiber/lib-auth/lib-license integration where current standalone service behavior needs it.

Host applications provide:

- Auth, license, HTTP/API shape, process lifecycle, env loading, state stores, caches, storage, events, telemetry exporters, and deployment topology.

## Capability-Level Components

| Component | Responsibility | Current Source |
| --- | --- | --- |
| Engine composition root | Assemble Engine operations from ports, connectors, limits, hooks, and optional adapters | Proposed importable layer |
| Connection operations | Create/update/delete/list/get/test connection definitions | `create_connection.go`, update/delete/list/get/test services |
| Credential protection | Encrypt/decrypt connection credentials; host controls key source and rotation | `pkg/crypto/crypto.go` |
| Result protection and integrity | Define canonical result protection/integrity contracts; compatibility adapters preserve current encrypted storage payload and HMAC behavior | Worker storage encryption and document signer behavior |
| Connector registry/factory | Create datasource connectors without tying core to service shell | `pkg/model/datasource/datasource-config.go`, `pkg/datasource/datasource_factory.go` |
| Schema operations | Discover and validate datasource schema | `validate_schema.go`, `get_connection_schema.go`, `pkg/model/schema.go` |
| Planner | Normalize mapped fields/filters into executable datasource work | Current job validation and extraction loops |
| Runner | Execute planned extraction and normalize results | `components/worker/internal/services/extract_data.go` |
| Result handling | Return direct payload or write to host-provided sink | Worker save/result flow and storage port |
| Error taxonomy | Stable business, validation, connection, extraction, timeout, storage errors | `pkg/errors.go` and current service errors |
| Observability hooks | Host-connectable logs/metrics/traces/events | Current lib-commons/lib-observability usage |
| Tenant-safety primitives | Product/tenant context and scope enforcement | Connection product fields and resolver behavior |

## Technology Boundaries

Engine core may depend on Go standard library, current domain models where compatible, connector contracts, and minimal shared Lerian libraries required by standards. It must not require Fiber, RabbitMQ, MongoDB, Redis, Docker Compose, KEDA, S3, or SeaweedFS.

Optional adapters may depend on current infrastructure libraries. For example, MongoDB repositories remain adapter dependencies; RabbitMQ remains a queue/event adapter; S3/SeaweedFS remain result sink adapters; Fiber/lib-auth/lib-license remain Manager compatibility adapter dependencies.

## Key Design Decisions

ADR-001: Strangler extraction over rewrite.

Decision: Extract an importable Engine around current behavior and adapt Manager/Worker to it incrementally.

Reason: Current behavior is broad and tested. A rewrite would destroy compatibility evidence and create a large regression surface.

ADR-002: Engine owns datasource semantics; hosts own operations.

Decision: Engine includes connection, connector, schema, validation, planning, extraction, result, errors, limits, hooks, and tenant-safety semantics. Host apps own HTTP, auth, license, env, queues, stores, storage choices, telemetry exporters, health, deployment, and process lifecycle.

Reason: Embedding exists to remove mandatory service operations, not to package them differently.

ADR-003: Async is a host execution mode, not a RabbitMQ requirement.

Decision: Engine should expose execution operations and optional event/execution-store ports. RabbitMQ remains an adapter.

Reason: Reporter/Matcher and self-hosted deployments may choose direct, scheduled, queue-backed, or hybrid execution.

ADR-004: Connector construction and connection testing should be separable.

Decision: Engine should distinguish connector creation from explicit connection testing.

Reason: Current factory can test connectivity during construction, which is service-friendly but awkward for planning, validation, testing, and host-controlled execution (`pkg/datasource/datasource_factory.go:168-185`).

ADR-005: Compatibility adapters preserve current API behavior.

Decision: Current Manager and Worker should remain as adapters over Engine, preserving route contracts, queue payloads, request-hash dedupe including the five-minute duplicate window, job status transitions, optional job notifications, event payloads, result path/HMAC semantics, and error mappings unless explicitly migrated.

Reason: Existing consumers should not pay the migration cost on day one.

## Data Flow: Embedded Synchronous Execution

Host application receives an already-authorized request.

Host application passes tenant/product context and extraction request to Engine.

Engine validates request shape, limits, mapped fields, and filters.

Engine resolves connections using host-provided connection store/resolver.

Engine optionally discovers or validates schema through connectors and schema cache port.

Engine plans datasource work.

Engine runs extraction with host-provided context for cancellation and timeout.

Engine returns a canonical result or writes to host-provided result sink and returns a reference.

Host application maps result/error to its own API or workflow.

## Data Flow: Compatibility Async Execution

Manager adapter receives existing HTTP request and applies existing auth/license/tenant middleware.

Manager adapter calls Engine to validate/build execution and persists compatibility job state through current job repository.

Manager adapter publishes existing queue message through RabbitMQ adapter.

Worker adapter consumes existing message and calls Engine runner.

Engine executes extraction and produces canonical result.

Worker adapter stores compatibility job status and publishes existing optional job notifications when configured.

## Security Model

Authorization remains outside Engine. The Engine receives context and operation inputs after host policy has allowed the call.

License enforcement remains outside Engine. Current Manager adapter continues to own lib-license middleware.

Credential protection semantics remain inside Engine, but key source and rotation policy are host-controlled.

Tenant/product context is required for operations that touch connections, internal datasource resolution, result paths, events, or observability attributes.

Logs, traces, metrics, and errors must never include raw credentials or extracted sensitive payloads.

## Assumptions

- Current app uses lib-auth and lib-license; Engine core keeps both outside.
- API contracts use camelCase for embedded request/response schemas.
- Logical data-model names are used in docs; backing stores keep their own naming conventions.
- `docs/PROJECT_RULES.md` remains the architecture source of truth, except Go toolchain version where `go.mod` is authoritative. Engine implementation must reconcile the stale architecture-doc Go version before code work starts. This feature must not regenerate architecture docs beyond that explicit reconciliation.

## Risks

Package extraction could accidentally import service internals or adapter dependencies into Engine core. Gate 7 must add explicit dependency tests or static checks.

Existing tests may protect service behavior but not embedded behavior. Gate 7 must add Engine-level tests that run without Fiber, RabbitMQ, MongoDB, Redis, S3, or SeaweedFS.

The plugin CRM behavior is currently special-cased. Gate 7 preserves it deliberately as adapter-level compatibility behavior for the first Engine release, not as a generic Engine core extension.
