# Handoff: Fetcher Multi-Tenant Event-Driven Discovery Fix

**Session:** 2026-03-27 | **Branch:** `feature/multi-tenant-orgid-removal` | **Service:** fetcher (Manager + Worker)

---

## 1. Session Summary

This session ran the `/dev-multi-tenant` compliance cycle on the fetcher service. The Gate 0 audit found the implementation mostly compliant with the Ring canonical standard, but identified **9 env var and architecture gaps**. All 9 were fixed and verified (build + tests pass). During the fix, a **critical bug** was discovered in the Worker's event-driven tenant discovery: Redis Pub/Sub events never reach the consumer's `EnsureConsumerStarted()`. An implementation plan was created but not yet executed.

---

## 2. What Was Completed

### Compliance Audit (Gate 0)
- Full stack detection: MongoDB, Redis, RabbitMQ, S3 (no PostgreSQL for service data)
- lib-commons v4.5.0-beta.19, lib-auth v2.5.0 confirmed
- All 11 audit checks (A1-A11) passed COMPLIANT

### 9 Gaps Fixed
| # | Gap | Components | Status |
|---|-----|-----------|--------|
| 1 | Added `MULTI_TENANT_REDIS_HOST/PORT/PASSWORD` to Config structs | Manager + Worker | Done |
| 2 | Removed non-canonical `MULTI_TENANT_ENVIRONMENT` | Manager + Worker | Done |
| 3 | Added `default:` struct tags to all int Config fields (7 fields each) | Manager + Worker | Done |
| 4 | Wired `WithTenantCache` + `WithTenantLoader` on TenantMiddleware | Manager | Done |
| 5 | Implemented `EventListener` + `EventDispatcher` for Redis Pub/Sub | Manager | Done |
| 6 | Implemented `EventListener` + `EventDispatcher` for Redis Pub/Sub | Worker | Done (but has bug) |
| 7 | Fixed validation from `REDIS_HOST` to `MULTI_TENANT_REDIS_HOST` | Worker | Done |
| 8 | Updated `.env.example` files (added 3 Redis vars, removed ENVIRONMENT) | Manager + Worker | Done |
| 9 | Updated test structs and assertions across 4 test files | pkg/multitenant, pkg/metrics, manager, worker | Done |

### Plan Created
- Path: `/home/z1l/dev/work/lerian/workspace/plans/fetcher/2026-03-27-unify-worker-event-driven-tenant-discovery.md`
- 4 tasks: implement `initMultiTenantStack` -> run tests -> code review -> commit

---

## 3. Challenges Remaining

### Challenge 1: Worker Dual-Dispatcher Bug (CRITICAL)

**The Problem:** Two independent `EventDispatchers` exist in the Worker bootstrap that don't communicate:

```
initMultiTenantConsumer() creates:
  MultiTenantConsumer
    -> internal EventDispatcher (has EnsureConsumerStarted callbacks)
    -> internal TenantCache, TenantLoader
    -> BUT no Redis listener feeds events to it

initWorkerEventDiscovery() creates:
  SEPARATE tmClient (duplicated!)
  SEPARATE TenantCache, TenantLoader
  SEPARATE EventDispatcher (has Redis listener via TenantEventListener)
    -> BUT no consumer callbacks (never calls EnsureConsumerStarted)
```

**Impact:** When `tenant.service.associated` arrives via Redis Pub/Sub, the external dispatcher updates its cache but never spawns a consumer goroutine. Tenant messages in RabbitMQ queues go unprocessed.

**The Fix:** Replace both functions with unified `initMultiTenantStack`:
1. Create ONE shared `tmClient`, `TenantCache`, `TenantLoader`
2. Create ONE `EventDispatcher` with infra managers (mongo, rabbitmq)
3. Inject dispatcher into `MultiTenantConsumer` via `tmconsumer.WithEventDispatcher(dispatcher)`
   - lib-commons internally calls `wireDispatcherCallbacks()` which wires:
     - `OnTenantAdded` -> `knownTenants[id]=true` + `EnsureConsumerStarted(ctx, tenantID)`
     - `OnTenantRemoved` -> `cancel goroutine` + `delete knownTenants`
4. Wire `tenantLoader.SetOnTenantLoaded` -> `consumer.EnsureConsumerStarted` (restart recovery)
5. Create `TenantEventListener` using `dispatcher.HandleEvent` (same dispatcher!)

**File:** `components/worker/internal/bootstrap/config.go`
- Delete `initMultiTenantConsumer` (lines 586-636)
- Delete `initWorkerEventDiscovery` (lines 656-end)
- Add `initMultiTenantStack` (unified function)
- Update bootstrap caller (lines 274-308)

### Challenge 2: Manager OnTenantRemoved Callback (LOW)

The Manager's `initManagerEventDiscovery` correctly shares TenantCache/TenantLoader with the middleware but lacks an `OnTenantRemoved` callback for closing mongo connections and invalidating the pmClient cache. The midaz CRM reference implementation (`/home/z1l/dev/work/lerian/workspace/repos/midaz/components/crm/internal/bootstrap/config.tenant.go`) does this. Non-blocking but recommended.

### Challenge 3: Uncommitted Changes

All 9 gap fixes from this session are uncommitted. Files modified:
- `components/manager/internal/bootstrap/config.go`
- `components/manager/internal/bootstrap/config_test.go`
- `components/manager/.env.example`
- `components/worker/internal/bootstrap/config.go`
- `components/worker/internal/bootstrap/config_test.go`
- `components/worker/.env.example`
- `pkg/multitenant/multitenant_test.go`
- `pkg/metrics/backward_compat_test.go`

Build passes, all tests pass. Needs commit before or after the worker fix.

---

## 4. Key Decisions Made

| Decision | Rationale |
|----------|-----------|
| `MULTI_TENANT_REDIS_*` are separate from application `REDIS_*` | Tenant-manager Redis (Pub/Sub discovery) may be a different instance than the application Redis (schema cache) |
| Removed `MULTI_TENANT_ENVIRONMENT` | Not in the canonical 13 env vars; replaced usage with `cfg.EnvName` (`ENV_NAME`) |
| Manager pattern follows midaz CRM | HTTP-only component: shared cache/loader/dispatcher, no consumer callbacks needed |
| Worker needs `WithEventDispatcher` injection | Consumer component: dispatcher must have `EnsureConsumerStarted` wired via `wireDispatcherCallbacks()` |
| `SetOnTenantLoaded` for restart recovery | After restart, first lazy-load of a tenant also starts its consumer goroutine |

---

## 5. Key Reference Files

| File | Purpose |
|------|---------|
| `plans/fetcher/2026-03-27-unify-worker-event-driven-tenant-discovery.md` | Implementation plan with exact code |
| `repos/ring/dev-team/docs/standards/golang/multi-tenant.md` | Canonical multi-tenant standard |
| `repos/midaz/components/crm/internal/bootstrap/config.tenant.go` | Reference: HTTP component event-driven discovery |
| `repos/reporter/components/worker/internal/bootstrap/config_multitenant.go` | Reference: Worker component (has same gap as fetcher) |
| lib-commons `tenant-manager/consumer/multi_tenant.go` line 138 | `WithEventDispatcher` option |
| lib-commons `tenant-manager/consumer/multi_tenant_compat.go` line 67 | `wireDispatcherCallbacks()` — wires OnTenantAdded/Removed |
| lib-commons `tenant-manager/consumer/multi_tenant.go` line 382 | `buildEventDispatcher()` — internal dispatcher builder |
| lib-commons `tenant-manager/event/dispatcher.go` | `EventDispatcher` with `WithOnTenantAdded/Removed` |
| lib-commons `tenant-manager/event/listener.go` | `TenantEventListener` for Redis Pub/Sub |
| lib-commons `tenant-manager/tenantcache/` | `TenantCache` + `TenantLoader` |

---

## 6. Next Steps (Priority Order)

1. **Execute the implementation plan** — Task 1 through Task 4 in the plan file
2. **Commit all changes** — both the 9 gap fixes and the worker dispatcher unification
3. **Consider Manager OnTenantRemoved** — add mongo connection cleanup + pmClient cache invalidation callback (like midaz CRM)

---

## 7. How to Resume

```bash
cd /home/z1l/dev/work/lerian/workspace/repos/worktree/fetcher-multi-tenant
# Read the plan:
cat /home/z1l/dev/work/lerian/workspace/plans/fetcher/2026-03-27-unify-worker-event-driven-tenant-discovery.md
# Then execute Task 1 through Task 4
```

Or use: `/ring:execute-plan /home/z1l/dev/work/lerian/workspace/plans/fetcher/2026-03-27-unify-worker-event-driven-tenant-discovery.md`
