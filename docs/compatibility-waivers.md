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
