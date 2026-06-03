# Gate 7 Task Breakdown: Fetcher Embedded Runtime

## Summary

The implementation is a strangler extraction. The first safe slice characterizes and enforces the dependency boundary before moving Manager or Worker behavior. The importable Engine core lives at `pkg/engine`; no mandatory service-shell dependency may enter that package. Optional compatibility wrappers live outside core, under paths such as `pkg/enginecompat/*`, and current Manager/Worker packages remain compatibility adapters.

Target for every task: backend.

Working directory for every task: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Recommended implementation agent for every task: `ring:backend-engineer-golang`.

## Summary Table

| Task | Title | Type | Hours | Confidence | Blocks | Status |
| --- | --- | --- | ---: | --- | --- | --- |
| T-001 | Characterize runtime seams and enforce Engine dependency boundary | backend | 8h | Medium | none | ✅ Done |
| T-002 | Create importable Engine contracts, facade, and in-memory harness | backend | 8h | Medium | T-001 | ✅ Done |
| T-003 | Extract embedded connection lifecycle and credential protection | backend | 10h | Medium | T-002 | ✅ Done |
| T-004 | Introduce connector registry/factory with explicit connect/test semantics | backend | 12h | Medium | T-002, T-003 | ✅ Done |
| T-005 | Extract schema discovery and validation into Engine operations | backend | 12h | Medium | T-003, T-004 | ✅ Done |
| T-006 | Add extraction planner with limits and tenant/product scope checks | backend | 10h | Medium | T-005 | ✅ Done |
| T-007 | Add synchronous execution runner, canonical result model, and result sink port | backend | 14h | Medium | T-006 | ✅ Done |
| T-008 | Migrate Manager connection/schema compatibility paths over Engine | backend | 12h | Medium | T-003, T-005 | ✅ Done |
| T-009 | Migrate Manager job creation/planning path over Engine while preserving RabbitMQ dispatch | backend | 14h | Medium | T-006, T-008 | 🔄 Doing |
| T-010 | Migrate Worker extraction path over Engine while preserving job status/results/events | backend | 16h | Medium | T-007, T-009 | READY_FOR_DEV_CYCLE |

## Business Deliverables

| Deliverable | First Task That Delivers It | Business Value |
| --- | --- | --- |
| Proven Engine core dependency boundary | T-001 | Prevents embedded mode from inheriting mandatory service deployment dependencies. |
| Importable Engine API with no-infra harness | T-002 | Host products can compile against Fetcher Engine without Manager or Worker internals. |
| Embedded connection management and credential protection | T-003 | Keeps datasource setup canonical while host apps own persistence and keys. |
| Explicit connector registry and test semantics | T-004 | Separates connector construction from connectivity checks and supports host-controlled runtime. |
| Embedded schema discovery and validation | T-005 | Lets host products reuse Fetcher schema safety instead of reimplementing it. |
| Tenant-safe extraction planning | T-006 | Prevents cross-product datasource access before execution begins. |
| Synchronous execution and canonical results | T-007 | Enables in-process extraction without RabbitMQ, Worker, or mandatory object storage. |
| Manager compatibility for connection/schema flows | T-008 | Existing HTTP consumers keep working while behavior moves behind Engine. |
| Manager compatibility for job planning and dispatch | T-009 | Existing async API keeps RabbitMQ dispatch while planning becomes Engine-owned. |
| Worker compatibility over Engine runner | T-010 | Existing queue execution, job status, result storage, and events remain intact. |

## T-001 Characterize runtime seams and enforce Engine dependency boundary

Target: backend.

Working Directory: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Agent: `ring:backend-engineer-golang`.

Deliverable: A compiling boundary test and seam characterization proving future Engine core code can stay importable without Manager/Worker internals or mandatory shell dependencies.

Scope Includes:

- Add dependency-boundary tests for `pkg/engine` before substantial Engine behavior exists.
- Characterize current runtime seams in datasource factory, Manager command/query services, Worker extraction, resolver, storage, cache, and notification flows.
- Preserve `plugin_crm` as adapter-level compatibility behavior for the first Engine release; do not promote it into a generic Engine core extension yet.
- Add a dev-cycle blocker subtask that reconciles the Go version source of truth before implementation starts: `go.mod` declares Go `1.25.9`, so docs or tooling must align with that source without changing `docs/PROJECT_RULES.md` during this pre-dev pass.

Scope Excludes:

- No Manager or Worker behavior migration.
- No connector rewrite.
- No new HTTP routes, queue topology, or deployment changes.

Success Criteria:

- `go test ./pkg/engine/...` passes with an empty or minimal Engine package.
- Boundary tests fail if `pkg/engine` imports Fiber, RabbitMQ, MongoDB, Redis, S3/SeaweedFS, Docker Compose/KEDA packages, lib-auth, lib-license, or `components/*/internal`.
- Characterization tests or docs name the current seams that later tasks will strangle.
- Dev-cycle has an explicit first-task instruction to reconcile the Go version mismatch before code implementation starts, using `go.mod` Go `1.25.9` as the source of truth.

User/Technical Value: De-risks the whole feature by making the dependency boundary executable instead of advisory prose. Advisory prose has the structural strength of wet cardboard.

Technical Components:

- Proposed `pkg/engine` package.
- Proposed `pkg/engine/dependency_test.go`.
- Proposed version reconciliation guard in T-001 subtasks.
- Existing anchors: `pkg/datasource/datasource_factory.go`, `components/manager/internal/services/command/create_fetcher_job.go`, `components/manager/internal/services/query/validate_schema.go`, `components/worker/internal/services/extract_data.go`, `components/worker/internal/services/extract_crm_data.go`, `pkg/resolver/resolver.go`, `pkg/storage/factory.go`.

Dependencies: none.

Effort Estimate: 8h.

Risks:

- Boundary tests that are too broad could block legitimate optional connector adapters.
- Boundary tests that are too narrow could let service shell dependencies leak into core.

Testing Strategy:

- Add package-level import graph tests using `go list -deps` for `github.com/LerianStudio/fetcher/pkg/engine`.
- Run `go test ./pkg/engine/...` and `go test ./...`.

Definition of Done:

- Boundary tests are committed as source in the implementation task, not only described in docs.
- The first task leaves working software: an importable, dependency-guarded Engine shell.

## T-002 Create importable Engine contracts, facade, and in-memory harness

Target: backend.

Working Directory: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Agent: `ring:backend-engineer-golang`.

Deliverable: A minimal public Engine facade with stable contracts and a no-infra in-memory harness usable by host products and tests.

Scope Includes:

- Create `pkg/engine` contracts for Engine construction, tenant context, connections, schemas, planning, execution, results, errors, limits, and ports.
- Create `pkg/engine/memory` in-memory stores/sinks/cache for tests and embedded examples.
- Provide constructor validation and safe defaults.

Scope Excludes:

- No Manager or Worker migration.
- No real datasource execution.
- No dependency on MongoDB, Redis, RabbitMQ, Fiber, S3, or SeaweedFS.

Success Criteria:

- Host code can import `github.com/LerianStudio/fetcher/pkg/engine` and construct an Engine with in-memory collaborators.
- Core contracts model `ConnectionStore`, `ExecutionStore`, `ResultSink`, `SchemaCache`, `EventSink`, `CredentialProtector`, `Connector`, `ConnectorRegistry`, `TenantResolver`, and observability hooks.
- Missing required capabilities return safe Engine errors.

User/Technical Value: Establishes the embedded API surface that Reporter and Matcher can compile against before behavior is migrated.

Technical Components:

- New `pkg/engine` files for facade, options, errors, limits, context, connection, schema, planning, execution, ports.
- New `pkg/engine/memory` harness.
- Existing anchors: `pkg/model/connection.go`, `pkg/model/schema.go`, `pkg/model/job.go`, `pkg/errors.go`, `pkg/testutil/context.go`.

Dependencies: T-001.

Effort Estimate: 8h.

Risks:

- Over-modeling contracts before behavior migrates.
- Reusing current service-shaped models too directly and freezing adapter assumptions into core.

Testing Strategy:

- TDD constructor validation, error contracts, and memory harness behavior.
- Run `go test ./pkg/engine/...` and `go test ./...`.

Definition of Done:

- Engine core compiles independently.
- In-memory harness proves host usage without service infrastructure.
- Boundary tests from T-001 still pass.

## T-003 Extract embedded connection lifecycle and credential protection

Target: backend.

Working Directory: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Agent: `ring:backend-engineer-golang`.

Deliverable: Engine-owned connection create, update, delete, get, list, and credential protection semantics running against the in-memory harness.

Scope Includes:

- Implement Engine connection lifecycle operations over `ConnectionStore`.
- Add `CredentialProtector` behavior with host-provided implementation.
- Preserve product ownership, config-name uniqueness, password non-return, encryption key version, soft-delete compatibility semantics, and active-job conflict hooks where applicable.

Scope Excludes:

- No Manager handler migration.
- No Mongo repository changes unless an adapter test needs compatibility proof.
- No real datasource connectivity testing; that is T-004.

Success Criteria:

- Engine connection operations pass service-equivalent tests using in-memory store.
- Credentials are encrypted before persistence and decrypted only through runtime connector paths in later tasks.
- Returned connection outputs never include raw password or encrypted secret unless the store-facing contract explicitly requires it.

User/Technical Value: Host apps gain canonical connection lifecycle behavior without importing Manager internals.

Technical Components:

- `pkg/engine` connection operation files.
- `pkg/engine/memory` connection store.
- Existing anchors: `components/manager/internal/services/command/create_connection.go`, `update_connection.go`, `delete_connection.go`, `components/manager/internal/services/query/get_connection.go`, `list_connections.go`, `pkg/crypto/crypto.go`, `pkg/ports/connection/repository.go`.

Dependencies: T-002.

Effort Estimate: 10h.

Risks:

- Accidentally changing compatibility rules for uniqueness, soft delete, or active jobs.
- Logging or returning credentials through new Engine errors.

Testing Strategy:

- Port current Manager connection tests into Engine-level tests where behavior should be shared.
- Add credential redaction assertions.
- Run `go test ./pkg/engine/...`, `go test ./components/manager/internal/services/command ./components/manager/internal/services/query`, and `go test ./...`.

Definition of Done:

- Engine connection lifecycle works without Manager internals.
- Existing Manager tests still pass before Manager migration.

## T-004 Introduce connector registry/factory with explicit connect/test semantics

Target: backend.

Working Directory: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Agent: `ring:backend-engineer-golang`.

Deliverable: Engine connector registry and compatibility datasource adapter that separates connector construction from explicit connectivity testing.

Scope Includes:

- Define Engine connector and registry contracts.
- Add explicit `BuildConnector`, `Connect`, `TestConnection`, `Query`, `DiscoverSchema`, and `Close` semantics.
- Add optional compatibility wrapper under `pkg/enginecompat/datasource` around current datasource implementations.

Scope Excludes:

- No SQL query planner changes.
- No Manager/Worker migration.
- No change to concrete datasource driver behavior beyond adapter shape.

Success Criteria:

- Engine core depends only on connector interfaces, not concrete drivers.
- Compatibility adapter can wrap current `pkg/datasource` factory behavior without importing service shell packages into `pkg/engine`.
- Tests prove construction can occur without a network connection where the connector supports deferred connect.

User/Technical Value: Host apps can register connectors explicitly and test connectivity deliberately, instead of receiving side effects hidden inside construction.

Technical Components:

- `pkg/engine` connector contracts.
- `pkg/enginecompat/datasource` wrapper.
- Existing anchors: `pkg/model/datasource/datasource-config.go`, `pkg/datasource/datasource_factory.go`, `pkg/postgres`, `pkg/mysql`, `pkg/oracle`, `pkg/sqlserver`, `pkg/mongodb`.

Dependencies: T-002, T-003.

Effort Estimate: 12h.

Risks:

- Current factory side effects may be hard to separate without a compatibility shim.
- MongoDB construction currently pings early; preserving behavior while exposing explicit test semantics needs care.

Testing Strategy:

- Unit-test registry resolution and unknown datasource errors.
- Add fake connectors for explicit construction/connect/test behavior.
- Run `go test ./pkg/engine/...`, `go test ./pkg/enginecompat/...`, `go test ./pkg/datasource/...`, and `go test ./...`.

Definition of Done:

- Engine core can plan against connector contracts.
- Compatibility wrapper preserves existing connector behavior without polluting core dependencies.

## T-005 Extract schema discovery and validation into Engine operations

Target: backend.

Working Directory: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Agent: `ring:backend-engineer-golang`.

Deliverable: Engine schema discovery and validation operations with optional cache and explicit adapter-level `plugin_crm` compatibility handling.

Scope Includes:

- Implement `DiscoverSchema` and `ValidateSchema` Engine operations.
- Reuse or map current schema models into canonical Engine schema contracts.
- Support optional `SchemaCache`.
- Preserve `plugin_crm` behavior as adapter-level compatibility for the first Engine release, not as a generic Engine core extension.

Scope Excludes:

- No Manager HTTP migration; that is T-008.
- No extraction execution; that is T-007.
- No new connector feature support beyond current behavior.

Success Criteria:

- Engine validation catches missing datasource, missing table, missing field, invalid filter, source-down, and limit errors.
- Cache use is optional and tested.
- `plugin_crm` handling is behind a clearly named Manager or `pkg/enginecompat/plugincrm` compatibility adapter.

User/Technical Value: Host apps can rely on the same schema safety rules as Fetcher Manager without deploying Manager.

Technical Components:

- `pkg/engine` schema operation files.
- `pkg/engine/memory` schema cache.
- Existing anchors: `components/manager/internal/services/query/validate_schema.go`, `get_connection_schema.go`, `pkg/model/schema.go`, `pkg/ports/cache/repository.go`, `components/worker/internal/services/extract_crm_data.go`.

Dependencies: T-003, T-004.

Effort Estimate: 12h.

Risks:

- Schema behavior is broad and could drift from current Manager responses.
- `plugin_crm` can easily leak product-specific behavior into the core API if the adapter boundary is not explicit.

Testing Strategy:

- Port Manager validation cases to Engine-level tests using fake connectors.
- Add cache hit/miss tests.
- Add explicit tests for adapter-level CRM compatibility behavior.
- Run `go test ./pkg/engine/...`, `go test ./components/manager/internal/services/query`, and `go test ./...`.

Definition of Done:

- Engine schema operations work through contracts and fake connectors.
- Existing Manager validation tests still pass before migration.

## T-006 Add extraction planner with limits and tenant/product scope checks

Target: backend.

Working Directory: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Agent: `ring:backend-engineer-golang`.

Deliverable: Engine extraction planner that resolves connections, validates schemas, enforces limits, and prevents cross-product execution before runtime.

Scope Includes:

- Implement `PlanExtraction` over Engine contracts.
- Resolve mapped fields and filters into per-datasource work items.
- Enforce datasource count, table count, field count, timeout, result-size, and concurrency limits.
- Enforce product/tenant scope using `TenantContext` and connection ownership.

Scope Excludes:

- No query execution.
- No RabbitMQ dispatch.
- No durable job persistence.

Success Criteria:

- Planner produces deterministic plans for valid requests.
- Planner rejects unknown config names, cross-product connections, invalid filters, and limit violations.
- Plan output is independent of Manager request structs even if compatibility adapters map to it.

User/Technical Value: Prevents unsafe work before any connector query runs and gives host apps a reusable planning surface.

Technical Components:

- `pkg/engine` planner files.
- Existing anchors: `components/manager/internal/services/command/create_fetcher_job.go`, `components/manager/internal/services/query/validate_schema.go`, `pkg/model/job.go`, `pkg/model/connection.go`, `pkg/resolver/resolver.go`.

Dependencies: T-005.

Effort Estimate: 10h.

Risks:

- Current job creation mixes validation, connection tests, idempotency, persistence, and RabbitMQ; planner extraction must not accidentally own queue behavior.
- Tenant/product checks must not be optional by accident for persisted external connections.

Testing Strategy:

- TDD planner success and rejection cases with in-memory store and fake schema.
- Add deterministic plan tests for map ordering.
- Run `go test ./pkg/engine/...`, `go test ./components/manager/internal/services/command`, and `go test ./...`.

Definition of Done:

- Valid requests produce executable plans.
- Invalid or unsafe requests fail before execution.

## T-007 Add synchronous execution runner, canonical result model, and result sink port

Target: backend.

Working Directory: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Agent: `ring:backend-engineer-golang`.

Deliverable: Engine synchronous runner that executes plans through connectors and returns direct results or result references through a host-provided sink, including canonical result protection and integrity metadata.

Scope includes:

- Implement canonical `ExtractionResult`, `ResultReference`, and `ResultSink` semantics.
- Implement canonical result `integrity` metadata with algorithm plus digest or signature.
- Implement canonical result `protection` metadata with encrypted status, optional key version, optional mode, and `appliedBy` ownership.
- Execute planned datasource work through connector contracts.
- Support direct and store result modes.
- Support cancellation, timeouts, row/size accounting, safe errors, and optional execution-store status transitions.

Scope excludes:

- No Worker migration.
- No RabbitMQ notification behavior.
- No mandatory S3/SeaweedFS dependency in core.

Success criteria:

- Synchronous execution works with fake connectors and memory result sink.
- Store mode returns a reference and direct mode returns payload without requiring object storage.
- Direct and stored results expose canonical protection and integrity metadata before Worker migration starts.
- Cancellation and timeout paths produce canonical errors and safe execution state.

User/technical value: This is the first complete embedded extraction path without standalone Fetcher Manager, Worker, RabbitMQ, or object storage.

Technical components:

- `pkg/engine` runner/result files.
- `pkg/engine` result metadata types for integrity and protection semantics.
- `pkg/engine/memory` result sink and execution store.
- Existing anchors: `components/worker/internal/services/extract_data.go`, `components/worker/internal/services/job_notification.go`, `pkg/ports/storage/repository.go`, `pkg/crypto/crypto.go`, `pkg/model/job.go`.

Dependencies: T-006.

Effort Estimate: 14h.

Risks:

- Worker behavior has storage encryption, HMAC, result paths, and notifications; runner must expose hooks without absorbing compatibility shell concerns.
- Large result handling can become memory-heavy if limits are not enforced early.

Testing strategy:

- TDD runner success, partial connector failure, cancellation, timeout, direct result, store result, and safe error cases.
- TDD metadata cases proving `integrity.algorithm`, `integrity.digest` or `integrity.signature`, `protection.encrypted`, optional `protection.keyVersion`, optional `protection.mode`, and `protection.appliedBy` are populated consistently for direct and stored result modes.
- Compatibility characterization proving current Worker encrypted storage payload and HMAC behavior can map into Engine result metadata without moving Worker dependencies into core.
- Run `go test ./pkg/engine/...`, `go test ./components/worker/internal/services`, and `go test ./...`.

Definition of done:

- Engine can execute extraction synchronously in-process.
- Engine core owns canonical result protection and integrity metadata before Worker migration.
- Core still has no mandatory storage or queue dependency.

## T-008 Migrate Manager connection/schema compatibility paths over Engine

Target: backend.

Working Directory: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Agent: `ring:backend-engineer-golang`.

Deliverable: Manager connection, test, schema discovery, and schema validation flows delegate to Engine while preserving existing HTTP route behavior.

Scope Includes:

- Adapt Manager command/query services to call Engine operations for connection lifecycle and schema paths.
- Keep Fiber handlers, Swagger, auth, license, rate limiting, tenant middleware, and HTTP response shapes in Manager.
- Preserve existing error mappings and compatibility response models.

Scope Excludes:

- No job creation migration; that is T-009.
- No Worker migration.
- No route changes.

Success Criteria:

- Existing Manager tests for connection and schema paths pass.
- Engine-level tests remain green.
- HTTP contract behavior is preserved for current clients.

User/Technical Value: Starts reducing Manager to a compatibility adapter without making clients migrate.

Technical Components:

- Existing anchors: `components/manager/internal/services/command/create_connection.go`, `update_connection.go`, `delete_connection.go`, `components/manager/internal/services/query/get_connection.go`, `list_connections.go`, `test_connection.go`, `get_connection_schema.go`, `validate_schema.go`, `components/manager/internal/bootstrap/config.go`, `components/manager/internal/adapters/http/in/connection.go`.

Dependencies: T-003, T-005.

Effort Estimate: 12h.

Risks:

- Error mapping drift can break API compatibility even if Engine behavior is correct.
- Rate limiting belongs in Manager and must not move into core.

Testing Strategy:

- Run targeted Manager command/query tests.
- Add adapter-level tests where mapping behavior changes.
- Run `go test ./components/manager/internal/services/command ./components/manager/internal/services/query`, `go test ./pkg/engine/...`, and `go test ./...`.

Definition of Done:

- Manager connection/schema services are compatibility adapters over Engine.
- Current public behavior remains preserved.

## T-009 Migrate Manager job creation/planning path over Engine while preserving RabbitMQ dispatch

Target: backend.

Working Directory: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Agent: `ring:backend-engineer-golang`.

Deliverable: Manager job creation delegates request validation and planning to Engine while preserving dedupe, persisted job records, and RabbitMQ dispatch semantics.

Scope Includes:

- Map `FetcherRequest` into Engine extraction request and plan.
- Preserve five-minute idempotency, job persistence, active status semantics, request hash behavior, and current queue payloads.
- Keep RabbitMQ publishing as Manager compatibility shell behavior.

Scope Excludes:

- No Worker execution migration; that is T-010.
- No queue topology changes.
- No result storage changes.

Success Criteria:

- `CreateFetcherJob` tests pass with Engine planning behind the service.
- Existing RabbitMQ message shape is preserved.
- Engine remains free of RabbitMQ and job repository dependencies as mandatory core dependencies.

User/Technical Value: Async API users keep the same Manager contract while planning becomes reusable for embedded hosts.

Technical Components:

- Existing anchors: `components/manager/internal/services/command/create_fetcher_job.go`, `create_fetcher_job_test.go`, `pkg/model/job.go`, `pkg/model/job/job_queue.go`, `pkg/ports/job/repository.go`, `pkg/rabbitmq/rabbitmq.go`.

Dependencies: T-006, T-008.

Effort Estimate: 14h.

Risks:

- Request hash and duplicate behavior are compatibility-sensitive.
- Product ownership validation can be accidentally performed twice or skipped if adapter and Engine responsibilities blur.

Testing Strategy:

- TDD adapter mapping and preserve existing `CreateFetcherJob` tests.
- Add tests proving RabbitMQ publish is not called on Engine planning failure.
- Run `go test ./components/manager/internal/services/command`, `go test ./pkg/engine/...`, and `go test ./...`.

Definition of Done:

- Manager job creation uses Engine planning.
- Queue dispatch remains an adapter concern.

## T-010 Migrate Worker extraction path over Engine while preserving job status/results/events

Target: backend.

Working Directory: `/Users/fredamaral/repos/lerianstudio/fetcher`.

Agent: `ring:backend-engineer-golang`.

Deliverable: Worker delegates extraction execution to Engine runner while preserving job state transitions, result encryption/storage, HMAC, notifications, ack/nack behavior, and product-specific CRM compatibility.

Scope Includes:

- Map current queued job payloads and persisted job state into Engine execution requests.
- Preserve pending/in-progress/completed/failed behavior, storage path/result URL, HMAC, optional notifications, and failure event behavior.
- Keep RabbitMQ consumer and publisher behavior in Worker adapter.
- Preserve `plugin_crm` behavior through explicit Worker adapter compatibility mapping only.

Scope Excludes:

- No queue topology changes.
- No storage provider replacement.
- No public API changes.

Success Criteria:

- Worker extraction tests pass with Engine runner behind the service.
- Existing job notification tests pass.
- Engine core remains free of RabbitMQ, MongoDB repository, S3, and SeaweedFS mandatory dependencies.

User/Technical Value: Completes the strangler path for current standalone async execution while exposing the same execution semantics to embedded hosts.

Technical Components:

- Existing anchors: `components/worker/internal/services/extract_data.go`, `extract_data_test.go`, `extract_data_additional_test.go`, `extract_crm_data.go`, `job_notification.go`, `service.go`, `pkg/ports/storage/repository.go`, `pkg/ports/job/repository.go`, `pkg/model/job.go`, `pkg/model/job/job_queue.go`.

Dependencies: T-007, T-009.

Effort Estimate: 16h.

Risks:

- Worker currently bundles execution, storage encryption, job persistence, and notifications; extracting runner calls without changing behavior is delicate.
- Partial failure semantics across multiple datasources must match current expectations.

Testing Strategy:

- Preserve and expand Worker unit tests for success, skip, failure, storage error, notification optionality, and CRM extraction.
- Run `go test ./components/worker/internal/services`, `go test ./pkg/engine/...`, selected E2E when images are available, and `go test ./...`.

Definition of Done:

- Worker acts as a compatibility adapter over Engine execution.
- Existing async behavior remains intact.
