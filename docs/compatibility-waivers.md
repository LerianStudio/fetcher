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

## Worker `tmconsumer.MultiTenantConsumer` hidden Tenant Manager client lacks circuit breaker injection seam

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Dependency | `github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/consumer` |
| Current version | `v5.2.0` |
| Target version / API | First lib-commons version that exposes either `tmconsumer.WithTenantManagerClient(*client.Client)`, `tmconsumer.WithTenantManagerClientOptions(...client.ClientOption)`, or equivalent support for `client.WithCircuitBreaker(...)` on the consumer-owned Tenant Manager HTTP client. |
| Reason | Fetcher creates its shared Tenant Manager client through `initTenantManagerClient`, with circuit breaker and service API key, and reuses it for MongoDB/RabbitMQ managers. `tmconsumer.NewMultiTenantConsumerWithError` in `v5.2.0`, `v5.2.1`, and `v5.3.0` still constructs a private `pmClient` internally using only service API key / insecure HTTP / cache TTL options; there is no option to inject the preconfigured client or circuit breaker settings. Duplicating the consumer stack locally would fork lib-commons lifecycle, cache, and callback behavior. |
| Runtime signal | Worker bootstrap fails startup in multi-tenant mode with an explicit error: `multi-tenant worker startup blocked: lib-commons tmconsumer creates an internal Tenant Manager client without a circuit-breaker injection seam`. |
| Expiry / removal condition | Remove this waiver once Fetcher can pass its circuit-breaker configured `tmclient.Client` or circuit-breaker options into `tmconsumer.MultiTenantConsumer`; add a regression test proving the configured path is used. Review no later than 2026-06-30. |
| Upstream TODO | Add a lib-commons `tmconsumer` option for preconfigured Tenant Manager client injection or client options propagation, including `client.WithCircuitBreaker` and `client.WithTimeout`. |

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
