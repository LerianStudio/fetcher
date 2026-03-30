# Fetcher Multi-Tenant Activation Guide

## Components

| Component | Service Const | Module Const | Resources | Adapted |
|-----------|--------------|--------------|-----------|---------|
| Manager | `fetcher` | `fetcher-manager` | Connections, Jobs (MongoDB), Schema Cache (Redis), Job Publishing (RabbitMQ) | TenantMiddleware + WithMB, valkey key prefixing, tmrabbitmq.Manager |
| Worker | `fetcher` | `fetcher-worker` | Jobs (MongoDB), External Data (S3), Job Consuming (RabbitMQ), Job Events (RabbitMQ) | MultiTenantConsumer, tmrabbitmq.Manager, tms3 key prefixing |

## Environment Variables

All variables apply to **both** Manager and Worker unless noted.

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `MULTI_TENANT_ENABLED` | bool | No | `false` | Enable multi-tenant mode |
| `MULTI_TENANT_URL` | string | When enabled | — | Tenant Manager service URL |
| `MULTI_TENANT_MAX_TENANT_POOLS` | int | No | `100` | Soft limit for tenant connection pools (LRU eviction) |
| `MULTI_TENANT_IDLE_TIMEOUT_SEC` | int | No | `300` | Seconds before idle tenant connection is eviction-eligible |
| `MULTI_TENANT_TIMEOUT` | int | No | lib-commons default | HTTP client timeout for Tenant Manager API calls (seconds) |
| `MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD` | int | No | `5` | Consecutive failures before circuit breaker opens |
| `MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC` | int | No | `30` | Seconds before circuit breaker resets (half-open) |
| `MULTI_TENANT_SERVICE_API_KEY` | string | When enabled | — | API key for authenticating with Tenant Manager `/settings` endpoint |
| `MULTI_TENANT_CACHE_TTL_SEC` | int | No | lib-commons default | In-memory cache TTL for tenant config (seconds) |

### Worker-only variables

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `MULTI_TENANT_REDIS_HOST` | string | When enabled | — | Redis host for event-driven tenant discovery (Pub/Sub) |
| `MULTI_TENANT_REDIS_PORT` | string | No | `6379` | Redis port |
| `MULTI_TENANT_REDIS_PASSWORD` | string | No | — | Redis password |
| `RABBITMQ_MULTI_TENANT_SYNC_INTERVAL` | int | No | `30` | Seconds between tenant list synchronizations |
| `RABBITMQ_MULTI_TENANT_DISCOVERY_TIMEOUT` | int | No | `500` | Milliseconds for initial tenant discovery timeout |

## How to Activate

1. Ensure the **Tenant Manager** service is running and accessible.

2. Set environment variables for both Manager and Worker:

```env
MULTI_TENANT_ENABLED=true
MULTI_TENANT_URL=http://tenant-manager:8080
MULTI_TENANT_SERVICE_API_KEY=<your-api-key>
```

3. For the Worker, ensure Redis is configured (required for event-driven tenant discovery):

```env
MULTI_TENANT_REDIS_HOST=fetcher-valkey
MULTI_TENANT_REDIS_PORT=6379
MULTI_TENANT_REDIS_PASSWORD=<your-password>
```

4. Start the services. Both Manager and Worker will log:

```
Multi-tenant middleware initialized: url=http://tenant-manager:8080, module=fetcher-manager
```

## How to Verify

1. **Check startup logs** for multi-tenant initialization messages.

2. **Send a request with a JWT containing `tenantId`** to any Manager endpoint:

```bash
curl -H "Authorization: Bearer <jwt-with-tenantId>" \
     -H "X-Product-Name: my-product" \
     http://localhost:4006/v1/management/connections
```

3. **Verify tenant isolation** by creating connections under different tenants and confirming they don't see each other's data.

4. **Verify RabbitMQ vhost isolation** by checking that messages are published to tenant-specific vhosts (visible in RabbitMQ management UI).

5. **Verify Redis key prefixing** by inspecting Redis keys — they should be prefixed with the tenant ID.

6. **Verify S3 key prefixing** by checking stored objects — they should be under `{tenantId}/` prefix.

## How to Deactivate

Set `MULTI_TENANT_ENABLED=false` (or remove it entirely — default is `false`).

When disabled:
- No TenantMiddleware is registered
- No JWT parsing or tenant resolution occurs
- All database connections use the static MongoDB connection
- Redis keys are unprefixed
- S3 keys are unprefixed
- RabbitMQ connects directly (no per-tenant vhosts)
- All existing tests pass unchanged

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `MULTI_TENANT_URL is required when MULTI_TENANT_ENABLED=true` | Missing URL | Set `MULTI_TENANT_URL` |
| `MULTI_TENANT_SERVICE_API_KEY is required when MULTI_TENANT_ENABLED=true` | Missing API key | Set `MULTI_TENANT_SERVICE_API_KEY` |
| `MULTI_TENANT_REDIS_HOST is required when MULTI_TENANT_ENABLED=true` (Worker only) | Missing Redis for tenant discovery | Set `MULTI_TENANT_REDIS_HOST` |
| `401 Unauthorized` from Tenant Manager | Invalid or missing API key | Verify `MULTI_TENANT_SERVICE_API_KEY` matches the key configured in Tenant Manager |
| `tenant context required but no database injected` | Tenant ID in JWT but TenantMiddleware didn't resolve the DB | Check Tenant Manager has the tenant association configured |
| Circuit breaker open | Tenant Manager unavailable | Check Tenant Manager health; circuit resets after `MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC` |
