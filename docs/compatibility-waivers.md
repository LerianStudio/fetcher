# Compatibility Waivers

## lib-commons/v5 pinned at v5.2.0

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Dependency | `github.com/LerianStudio/lib-commons/v5` |
| Current version | `v5.2.0` |
| Target version | `v5.3.0` or newer |
| Reason | `v5.3.0` currently breaks Fetcher MongoDB manager API compatibility in the tenant-manager integration path. Fetcher remains pinned until that compatibility break is resolved upstream or Fetcher receives the matching adapter migration. |
| Expiry / removal condition | Remove this waiver and upgrade once Fetcher compiles and passes worker/manager bootstrap, multi-tenant MongoDB manager, and readyz tests against `lib-commons/v5 >= v5.3.0`. Review no later than 2026-06-30. |
| Validation evidence | Current remediation keeps `go.mod` pinned to `v5.2.0`; verification must include `go test ./...`, changed-package tests, and changed-package `golangci-lint run`. |

## Resolved: Worker no longer depends on `tmconsumer.MultiTenantConsumer` hidden Tenant Manager client

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Dependency | `github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/consumer` |
| Current version | `v5.2.0` |
| Inspected APIs | `tmconsumer.NewMultiTenantConsumerWithError` and options in `lib-commons/v5` `v5.2.0`, `v5.2.1`, and `v5.3.0` expose only `WithPostgresManager`, `WithMongoManager`, `WithRabbitMQ`, and `WithEventDispatcher`; none exposes a preconfigured `*tmclient.Client` or `tmclient.ClientOption` propagation. `v5.3.0` is the latest available tag at remediation time. |
| Resolution | Fetcher no longer constructs `tmconsumer.MultiTenantConsumer` in Worker startup. Worker now uses a local consumer adapter that keeps lib-commons canonical building blocks: configured `tmclient.Client` with `client.WithCircuitBreaker`, `tenantcache.TenantCache`, `tenantcache.TenantLoader`, `event.EventDispatcher`, `tmrabbitmq.Manager` per-tenant vhost channels, `tmmongo.Manager` tenant DB resolution, and `redis.NewTenantPubSubRedisClient` / `event.NewTenantEventListener`. |
| Runtime signal | Removed the fail-fast startup guard. Multi-tenant Worker bootstrap can initialize with the circuit-breaker-compliant client and per-tenant RabbitMQ manager path. |
| Runtime blocking status | No longer blocks runtime. The upstream `tmconsumer` seam gap remains, but Worker does not rely on the hidden raw client path. |
| Upstream TODO | Still useful upstream: add `tmconsumer.WithTenantManagerClient(*client.Client)` or client option propagation so services can return to the canonical consumer wrapper without losing circuit-breaker compliance. |

## RabbitMQ AMQP security envelope uses temporary local HMAC adapter

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Dependency | `github.com/LerianStudio/lib-commons/v5/commons/webhook` / future queue-envelope primitive |
| Current version | `v5.2.0` |
| Target version / API | First lib-commons version that exposes a queue/message envelope signer and verifier able to bind timestamp, tenant ID, job ID, exchange, routing key, and body for AMQP messages. |
| Reason | lib-commons `webhook.VerifySignatureWithFreshness` verifies HTTP webhook signatures (`X-Webhook-Signature`) over webhook-specific payload formats and requires the raw secret. Fetcher's AMQP envelope currently receives a `crypto.Signer` abstraction and must bind queue-routing fields to prevent cross-tenant and cross-route replay. lib-commons v5.2.0 has no exported signing generator or AMQP envelope verifier that preserves those semantics. |
| Local adapter | `pkg/rabbitmq/security_envelope.go` keeps AMQP canonical payload construction local and marks the local signer usage as temporary until lib-commons ships a queue-envelope primitive. |
| Expiry / removal condition | Replace local HMAC signing/verification with lib-commons once the upstream queue-envelope primitive exists and behavior-preserving tests pass. Review no later than 2026-06-30. |
| Upstream TODO | Add lib-commons queue-envelope signing/verification APIs with freshness, versioning, constant-time comparison, and canonical metadata binding for tenant/routing/message identity. |

## Behavior delta: extraction result table keys are normalized

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Since | Embedded-engine migration |
| Scope | Worker generic extraction path (`pkg/enginecompat/tablenorm`); does not apply to `plugin_crm` |
| Legacy behavior | The stored/encrypted result artifact was keyed by the verbatim requested table name. |
| New behavior | Table keys are normalized at the engine seam: default-schema prefixes are stripped (PostgreSQL `public.users` -> `users`, SQL Server `dbo.x` -> `x`) and Oracle identifiers are uppercased. The stored/encrypted result artifact is keyed by the NORMALIZED name. |
| Unaffected | The persisted job spec (Manager, Mongo) keeps the verbatim requested name. Non-default schemas (e.g. `accounting.invoices`), MySQL, and MongoDB names are not normalized. |
| Decision | Accepted as the new contract. No external result-key consumers existed at decision time (2026-06-07). |

## Behavior delta: job notification routing key drops the source segment

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Since | lib-streaming migration |
| Scope | Job status notifications (`components/infra/rabbitmq/etc/definitions.json`) |
| Legacy behavior | Published raw RabbitMQ routing key `job.<status>.<source>` (e.g. `job.completed.plugin_crm`). |
| New behavior | Events are emitted with DefinitionKey `job.<status>` (exact bindings `job.completed` / `job.failed`). `source` is available ONLY in the event payload metadata. |
| Impact | Topic subscribers using `job.<status>.<source>` or `job.<status>.*` patterns will not match the new key. Routing-level filtering by source now requires payload inspection or a future Subject/attribute change. |
| Decision | Accepted. No consumers bind by source at decision time (2026-06-07). |

## Breaking change: streaming env vars are now mandatory for the Worker

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Since | lib-streaming migration (v2.0.0) |
| Scope | Worker startup (`components/worker`) |
| New requirement | `STREAMING_ENABLED` must be `true` for the Worker to start. Terminal job-event notifications (`job.completed` / `job.failed`) are mandatory and emitted via lib-streaming. |
| Exchange | `RABBITMQ_JOB_EVENTS_EXCHANGE` (default `fetcher.job.events`) is the job-events exchange used by the streaming RabbitMQ route target. |
| Behavioral impact if unset | Worker startup fails fast (fail-closed wiring). There is no silent degradation and no legacy fallback — a missing or `false` `STREAMING_ENABLED` blocks the Worker from starting. |
| Decision | Accepted as the new v2.0.0 contract. Operators must set `STREAMING_ENABLED=true` and provision the `fetcher.job.events` exchange before upgrade. |

## Behavior delta: Manager schema discovery + validation use UPPERCASE-canonical Oracle identifiers

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Since | Embedded-runtime GA (v2.0.0) |
| Scope | Manager `GET .../schema` response and schema validation for Oracle datasources only. Does not affect PostgreSQL/MySQL/SQL Server/MongoDB. |
| Legacy behavior | The Manager surfaced Oracle table/field identifiers in lowercase (from `oracle.GetSchemaInfo`), while the Worker extraction path normalized to UPPERCASE — the two services disagreed, and the Manager's lowercase identity diverged from the physical UPPERCASE column keys the extracted rows actually carry (`pkg/oracle.createRowMap` keys verbatim by the physical catalog). |
| New behavior | Oracle is UPPERCASE-canonical end-to-end: discovery snapshot, `/schema` response, validation identity, the normalized result table keys, and the physical data-key casing all agree (UPPERCASE). The Manager now aligns UP to the already-correct Worker. Pinned by the cross-path parity test (`components/manager/internal/services/query/oracle_casing_parity_test.go`). |
| Unaffected | The result-key normalization contract above ("Oracle identifiers are uppercased") is preserved byte-for-byte. `oracle.GetSchemaInfo` still lowercases internally; the canonical fold (`ToUpper`, idempotent) re-normalizes before any user-visible surface. The persisted job spec keeps the verbatim requested name; request matching stays case-insensitive at `pkg/oracle.ValidateTableAndFields`. |
| Deploy note | The Redis schema cache may hold pre-upgrade lowercase Oracle snapshots. During the cache TTL drain window after upgrade, an Oracle schema validation could compare an UPPERCASE-normalized request against a stale lowercase cached snapshot and fail. The window is transient (TTL-bounded) and Oracle-only; operators can flush the schema cache on upgrade to avoid it. |
| Decision | Accepted as the v2.0.0 contract. The pre-GA Manager-vs-Worker casing disagreement was the latent bug; this closes it by making every Oracle identity surface match the physical data. |
