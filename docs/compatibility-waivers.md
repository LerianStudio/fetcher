# Compatibility Waivers

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

## Breaking change: job event `ce-source` and `ce-type` both change

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Since | lib-streaming v3 migration |
| Scope | The `ce-source` **and `ce-type`** CloudEvents headers on every `job.completed` / `job.failed` event, and the `STREAMING_CLOUDEVENTS_SOURCE` variable. Both headers reach the AMQP message headers verbatim. |
| Legacy behavior | Source was the URI-shaped `//lerian.fetcher/worker`, which lib-streaming v2 silently folded into a topic segment. `ce-type` was `studio.lerian.<resourceType>.<eventType>` — `studio.lerian.job.completed` and `studio.lerian.job.failed`. |
| New behavior | Source must be exactly `fetcher` — Fetcher's roster name. v3 REJECTS a non-conforming source instead of folding it, and Fetcher additionally pins it to the roster name at startup (`pkg/streaming.RequireRosterSource`), enforced whether or not streaming is enabled; any other value refuses to boot. v3 also inserts the source into `ce-type`, which becomes `studio.lerian.<source>.<resourceType>.<eventType>` — `studio.lerian.fetcher.job.completed` and `studio.lerian.fetcher.job.failed`. |
| Impact | Consumers that filter, route, or audit on `ce-source` **or `ce-type`** stop matching until they accept the new values. Consumers bound on the exchange and routing key, or deduping on `ce-id`, are unaffected: `fetcher.job.events`, `job.completed` / `job.failed`, and the `ce-id` format are all unchanged. |
| Why the `ce-type` change happened upstream | Without the source segment, two services publishing the same resource and event names produce byte-identical `ce-type` values — a homonym collision the v3 topic collapse makes reachable in practice. |
| Why pinned to equality | The ce-source is the sole input to the topic, DLQ and ACL names the platform provisions, and those grants are literal patterns on the roster name. A grammar-legal near-miss (`fetcher-worker`) passes lib-streaming's own validation while being unpublishable by construction. Fetcher's RabbitMQ transport makes today's failure mode a corrupted event identity — on both headers — rather than total loss; the equality gate forecloses the loss case before a Kafka route ever exists. |
| Regression guard | `TestJobEventStreamingContract_CeTypeValues` pins both literal `ce-type` values through the exported `streaming.CloudEventsType` facade. No `ce-*` header was pinned before v3, which is why this change was nearly missed. |
| Deploy note | Operators MUST set `STREAMING_CLOUDEVENTS_SOURCE=fetcher` as part of the v3 deploy, and consumers MUST accept the new `ce-source` and `ce-type` values first. Widen consumer matching rather than switching it, so the same consumer survives a rollback. Ordering, including the mandatory outbox drain, is in [`docs/streaming/lib-streaming-v3-rollout.md`](streaming/lib-streaming-v3-rollout.md). |
| Decision | Accepted. Source pinning is fleet policy across Lerian services, not a Fetcher-local choice. |

## Breaking change: streaming outbox envelope version 1 -> 2 requires a drain

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Since | lib-streaming v3 migration |
| Scope | Persisted rows in the `streaming_outbox_events` collection with `event_type` `lerian.streaming.publish`. |
| Legacy behavior | lib-streaming v1 and v2 wrote outbox envelope version `1`. |
| New behavior | v3 writes version `2` and validates the persisted version by strict equality. A version-1 row is rejected as `ErrInvalidOutboxEnvelope`. Fetcher wires no retry classifier on the outbox dispatcher, so that error is treated as RETRYABLE: the row is retried each dispatch cycle until it exhausts the dispatch-attempt ceiling (10 by default), then marked failed. The version check is deterministic, so the retries cannot succeed. |
| Impact | Any undelivered job event at deploy time is ultimately lost, not delayed. The terminal-event repairer does NOT cover it: the `terminalEventPending` flag is cleared once the outbox row is durably written, so a stuck row whose job flag is already cleared has no repair path. |
| Repair window | Because the error is retryable, an affected row stays in the collection as `PENDING`/`FAILED` with a climbing `attempts` count and its payload intact until the ceiling is reached. Leaving it retryable is deliberate: for mandatory job events a narrower window is the wrong direction, since classifying the error non-retryable would convert a recoverable state into an immediate silent write-off. |
| Notable | Unlike services whose destination also changed, Fetcher's persisted destination is identical across versions (same exchange, same routing key), and the envelope and event structs are unchanged. A version-1 row would have delivered correctly under v2; under v3 it is refused on the version integer, and its stale `event.source` is the only other field needing repair — `ce-source` and `ce-type` are both derived from it at publish time. |
| Pre-existing `INVALID` rows | `INVALID` is terminal and is counted by the drain predicate, so a deployment that already holds such rows can never drain to zero by waiting. They need manual disposition before the deploy. |
| Required action | Drain the outbox to zero on the pre-v3 build before deploying, in every tenant database. Full procedure in [`docs/streaming/lib-streaming-v3-rollout.md`](streaming/lib-streaming-v3-rollout.md). |
| Decision | Accepted. Strict version equality is the upstream contract; the drain is the deploy-time cost of it. |

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

## Breaking change: TLS required by default for Mongo/Redis/Postgres/RabbitMQ connections

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Since | lib-commons v5.2.0 -> v5.5.0 bump (v2.0.0 GA) |
| Scope | All backing-store connections constructed through lib-commons helpers (Mongo, Redis, Postgres, RabbitMQ), Manager and Worker. |
| Legacy behavior | lib-commons v5.2.0 did not enforce TLS on Mongo/Redis connection URIs; plaintext connections were accepted silently. |
| New behavior | lib-commons v5.5.0 fails CLOSED on insecure (non-TLS) connection URIs unless `ALLOW_INSECURE_TLS=true`. A Manager/Worker pointed at a plaintext Mongo or Redis will refuse to start. |
| Impact | Existing deployments using non-TLS Mongo/Redis (including the default local docker-compose infra) must either enable TLS or set `ALLOW_INSECURE_TLS=true`. Local development is handled: the committed `components/{manager,worker}/.env.example` set `ALLOW_INSECURE_TLS=true` with an explicit production warning, matching the relaxed local posture (`DEPLOYMENT_MODE=local` also disables license enforcement). SSRF host-safety is gated separately by `MULTI_TENANT_ENABLED`, not by `DEPLOYMENT_MODE`. |
| Production guidance | Production MUST use TLS and MUST NOT set `ALLOW_INSECURE_TLS`. The fail-closed default is the intended secure posture for a fintech infrastructure product. |
| Decision | Accepted as the v2.0.0 contract. Fail-closed-on-insecure is the correct default; the override exists only for throwaway local/dev instances. |
