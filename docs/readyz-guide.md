# Fetcher /readyz Operator Guide

Operational reference for SRE/DevOps running Lerian `fetcher` in production. Covers endpoint semantics, K8s probe wiring, TLS enforcement in SaaS mode, multi-tenant behaviour, metrics, and runbook entries for the canonical failure modes.

---

## 1. Overview

`/readyz` answers one question: **is this fetcher replica safe to serve traffic right now?** It fans out to every platform dependency the replica owns (Mongo, RabbitMQ, Redis, S3, Tenant Manager), aggregates per-dependency status, and returns `200` only when every probed dep is `up` or benignly off-band (`skipped` / `n/a`). Any `down` or `degraded` dep returns `503` so Kubernetes removes the replica from Service endpoints until it recovers.

This endpoint exists because of the **Monetarie incident**: a fetcher replica kept receiving traffic after RabbitMQ credentials rotated, silently dropping inbound messages for ~22 minutes before the on-call paged. The new contract closes that gap by making broker health a first-class readiness signal rather than a log line.

**Scope fence** (see §13 for the full list): fetcher probes only platform state it *owns* — the infrastructure required to accept a request and deliver it to a queue or an object store. It does **not** probe user-supplied datasources (Postgres, MySQL, Oracle, SQL Server), cert validity, or business-logic SLIs. Those belong to separate tooling.

---

## 2. Endpoints

| Path                    | Service                        | Auth | Purpose                                              |
| ----------------------- | ------------------------------ | ---- | ---------------------------------------------------- |
| `/readyz`               | manager:4006 + worker:4007     | none | K8s `readinessProbe` target                          |
| `/health`               | manager:4006 + worker:4007     | none | K8s `livenessProbe` target — gated by startup probe  |
| `/readyz/tenant/:id`    | manager:4006 + worker:4007     | none | Per-tenant readiness (MT mode only)                  |
| `/metrics`              | manager:4006 + worker:4007     | none | Prometheus scrape target                             |
| `/version`              | manager:4006                   | none | Pre-existing build/version info                      |

> **CRITICAL — all four endpoints are intentionally unauthenticated.** Kubernetes requires it; the kubelet has no way to present tokens. Operators **MUST** restrict external access at layer 7 (Ingress) or layer 3/4 (NetworkPolicy) before shipping to production. `/readyz/tenant/:id` is a tenant-existence oracle (attackers can probe tenant IDs) and `/metrics` leaks Go runtime shape + dep latencies — both are info-disclosure risks if exposed to the open internet.

Recommended posture: expose these ports only on `ClusterIP` services, scrape Prometheus over a dedicated NetworkPolicy, and keep the container ports off any public ingress.

---

## 3. Environment Variables Reference

### New variables introduced by /readyz

| Name                      | Default   | Accepted values            | Behavior                                                                                                           | When to override                                                                       |
| ------------------------- | --------- | -------------------------- | ------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------- |
| `DEPLOYMENT_MODE`         | `local`   | `local`, `byoc`, `saas`    | `saas` runs `ValidateSaaSTLS()` at boot — any plaintext DSN on platform deps aborts startup. `byoc`/`local` skip. | **Must be `saas` in every hosted environment.** `local` only for developer laptops.    |
| `HEALTH_PORT`             | `4007`    | integer 1024-65535         | Worker-only: binds a dedicated Fiber server that exposes `/readyz`, `/health`, `/metrics`.                         | Change only if 4007 clashes with another sidecar. Update K8s `containerPort` to match. |
| `READYZ_DRAIN_DELAY_SEC`  | `12`      | integer ≥ 1                | On SIGTERM: `/readyz` starts returning 503 for this many seconds before `server.Shutdown` begins.                  | Raise if `readinessProbe.periodSeconds * failureThreshold` > 12s (see §4).             |

### Related existing variables that interact with /readyz

| Name                                | Default | Behavior                                                                                                                 |
| ----------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------ |
| `SERVER_PORT`                       | `4006`  | Manager Fiber port. `/readyz`/`/health`/`/metrics` share this port with business endpoints.                              |
| `MULTI_TENANT_ENABLED`              | `false` | When `true`: Mongo + RabbitMQ global checks report `n/a`. Enables `/readyz/tenant/:id`. Requires Tenant Manager reachable. |
| `MULTI_TENANT_ALLOW_INSECURE_HTTP`  | `false` | SaaS escape hatch: allows Tenant Manager URL to be plain `http://` even when `DEPLOYMENT_MODE=saas`. See §15.            |
| `MULTI_TENANT_CACHE_TTL_SEC`        | `120`   | Tenant-existence cache TTL. Tenant create/disable can lag in `/readyz/tenant/:id` up to this window.                    |
| `REDIS_HOST`                        | (unset) | When unset, `/readyz` reports `redis: skipped` (not a failure). When set, Redis becomes a hard dep.                      |
| `RABBITMQ_TLS`                      | `false` | Forces `amqps://` in SaaS validation. Must be `true` in SaaS with a public broker.                                       |
| `MONGO_TLS_CA_CERT`                 | (unset) | CA cert path for Mongo TLS verification. Validated at boot in SaaS mode.                                                 |

---

## 4. K8s Probe Configuration

### Manager (port 4006)

```yaml
ports:
- name: http
  containerPort: 4006
readinessProbe:
  httpGet:
    path: /readyz
    port: http
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 2
  timeoutSeconds: 3
livenessProbe:
  httpGet:
    path: /health
    port: http
  initialDelaySeconds: 30
  periodSeconds: 10
  failureThreshold: 3
  timeoutSeconds: 3
```

### Worker (port 4007)

The worker's health server is **separate** from any business listener — it binds only when `HEALTH_PORT` is set (default `4007`). The deployment template must declare the container port and point both probes at it.

```yaml
ports:
- name: health
  containerPort: 4007
readinessProbe:
  httpGet:
    path: /readyz
    port: health
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 2
  timeoutSeconds: 3
livenessProbe:
  httpGet:
    path: /health
    port: health
  initialDelaySeconds: 30
  periodSeconds: 10
  failureThreshold: 3
  timeoutSeconds: 3
```

### terminationGracePeriodSeconds

The pod spec must allow the full drain window plus ServerManager teardown:

```
terminationGracePeriodSeconds ≥ READYZ_DRAIN_DELAY_SEC + 30
```

**Recommended: `45` for the default 12s drain.** If you bump `READYZ_DRAIN_DELAY_SEC`, bump this alongside it — K8s will SIGKILL the container the instant the grace period expires, even mid-shutdown.

```yaml
spec:
  terminationGracePeriodSeconds: 45
  containers:
  - name: fetcher-manager
    # ...
```

---

## 5. Pre-Production Coordination Checklist

Five blocking items live **outside this repo** and must be shipped before fetcher /readyz is effective in production. Treat this section as a pre-flight — `/readyz` ships code-complete in fetcher but is inert without the infrastructure side.

| # | Repo                                                            | File                                                          | Action                                                                                                                          |
| - | --------------------------------------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| 1 | `repos/helm`                                                    | `charts/fetcher/templates/manager/deployment.yaml`            | Split liveness/readiness (currently both target `/health`); add `terminationGracePeriodSeconds: 45`.                            |
| 2 | `repos/helm`                                                    | `charts/fetcher/templates/worker/deployment.yaml`             | Add `ports`, `livenessProbe`, `readinessProbe`, and a `Service` entry for the worker — no probes exist today.                   |
| 3 | `repos/helm`                                                    | `charts/fetcher/values.yaml`                                  | Expose `DEPLOYMENT_MODE`, `HEALTH_PORT`, `READYZ_DRAIN_DELAY_SEC` in the chart's configmap so overlays can set them.            |
| 4 | `repos/midaz-firmino-gitops` + `repos/lerian-aws-gitops`        | `environments/{firmino-prd,firmino-stg,staging-fetcher-mt}/fetcher/values.yaml` | Set `DEPLOYMENT_MODE: "saas"` per env. **Gate 4 TLS enforcement is silently inactive when this env is absent.** |
| 5 | `repos/fetcher` (already done)                                  | `components/worker/docker-compose.yml`                        | Publishes port 4007 + healthcheck for local dev. No action — reference for parity.                                              |

**Item 4 is the most dangerous to miss.** `ValidateSaaSTLS()` reads `DEPLOYMENT_MODE`; when unset it returns immediately with no warning. Plaintext broker URIs in production won't fail boot — they'll just silently bypass the check. Audit each env's `values.yaml` before merging this to a production cluster.

---

## 6. Response Contract

`/readyz` returns `application/json`. Every dependency slot present in the response is one the replica actually owns — worker responses include `s3`, manager responses don't. The top-level `status` is `"healthy"` when every check resolved to `up`/`skipped`/`n/a`, `"unhealthy"` otherwise (HTTP 503).

### a) Manager, single-tenant, all healthy

```json
{
  "status": "healthy",
  "checks": {
    "mongodb":  { "status": "up", "latency_ms": 3, "tls": true },
    "rabbitmq": { "status": "up", "latency_ms": 5, "tls": true, "breaker_state": "closed" },
    "redis":    { "status": "up", "latency_ms": 1, "tls": true }
  },
  "version": "1.4.2",
  "deployment_mode": "saas"
}
```

### b) Manager, multi-tenant, Tenant Manager breaker open

Global mongo/rabbitmq report `n/a` because those are per-tenant; detail lives behind `/readyz/tenant/:id`.

```json
{
  "status": "unhealthy",
  "checks": {
    "mongodb":        { "status": "n/a", "tls": true, "reason": "multi-tenant: see /readyz/tenant/:id" },
    "rabbitmq":       { "status": "n/a", "tls": true, "reason": "multi-tenant: see /readyz/tenant/:id" },
    "redis":          { "status": "up", "latency_ms": 1, "tls": true },
    "tenant_manager": { "status": "degraded", "tls": true, "breaker_state": "open", "error": "circuit breaker open" }
  },
  "version": "1.4.2",
  "deployment_mode": "saas"
}
```

### c) Worker, single-tenant, RabbitMQ timing out

```json
{
  "status": "unhealthy",
  "checks": {
    "mongodb":  { "status": "up", "latency_ms": 4, "tls": true },
    "rabbitmq": { "status": "down", "latency_ms": 2003, "tls": true, "error": "context deadline exceeded" },
    "redis":    { "status": "skipped", "reason": "REDIS_HOST not configured" },
    "s3":       { "status": "up", "latency_ms": 18, "tls": true }
  },
  "version": "1.4.2",
  "deployment_mode": "saas"
}
```

---

## 7. Status Vocabulary Reference

Every check produces one of five values. Understand the distinction — treating `skipped` as `down` will cause false pages.

| Value     | Meaning                                                             | When emitted                                                                                      | K8s impact                          |
| --------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | ----------------------------------- |
| `up`      | Dep responded within timeout, all assertions passed.                | Probe succeeded.                                                                                  | Counts healthy. 200.                |
| `down`    | Dep did not respond or returned an error.                           | Probe failed — connection refused, timeout, auth error, wrong CA.                                 | Forces 503.                         |
| `degraded`| Dep reachable but protective state engaged.                         | Circuit breaker is `open` or `half-open`. Pod stays alive but drops from Service until recovery.  | Forces 503.                         |
| `skipped` | Dep intentionally not configured for this replica.                  | `REDIS_HOST` unset, `MULTI_TENANT_ENABLED=false` for Tenant Manager, S3 on manager.               | Counts healthy. 200 if all others up.|
| `n/a`     | Dep exists but is tenant-scoped; global answer is meaningless.      | `MULTI_TENANT_ENABLED=true` for Mongo/RabbitMQ.                                                   | Counts healthy. Per-tenant endpoint must be used for detail. |

**Why both `skipped` and `n/a`?** They convey different operational meaning: `skipped` means "this replica will never use this dep" (configuration decision), `n/a` means "this replica uses this dep but the answer is per-tenant" (architectural fact). Treating them identically would hide configuration drift — for example, an unset `REDIS_HOST` in an env that requires Redis would look indistinguishable from tenant-scoping.

---

## 8. Metrics Reference

Three Prometheus metrics are emitted. All are registered on the same `/metrics` endpoint as the existing process metrics.

### readyz_check_duration_ms

- **Type:** Histogram
- **Labels:** `dep` (mongodb|rabbitmq|redis|s3|tenant_manager), `status` (up|down|degraded|skipped|n/a)
- **Emitted:** Every `/readyz` call, once per dep probed.
- **Buckets:** 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000 ms.

### readyz_check_status

- **Type:** Gauge
- **Labels:** `dep`, `status`
- **Value:** `1` if current status matches the label pair, `0` otherwise.
- **Emitted:** Every `/readyz` call.

### selfprobe_result

- **Type:** Gauge
- **Labels:** `dep`
- **Value:** `1` if startup self-probe succeeded for that dep, `0` otherwise.
- **Emitted:** Once at boot by `RunSelfProbe`; retained as long as the replica lives.

### Sample PromQL alerts

```promql
# Any dep has reported down within the last 2 minutes (instant-fire)
max by (dep, service_name) (readyz_check_status{status="down"}) > 0

# P99 check latency exceeds 1.5s for any dep over 5m
histogram_quantile(0.99,
  sum by (dep, le) (rate(readyz_check_duration_ms_bucket[5m]))
) > 1500

# Startup self-probe never succeeded for a dep (boot failure canary)
selfprobe_result == 0

# Drain in progress for >30s — shutdown is stuck
count_over_time(readyz_check_status{dep="draining",status="down"}[1m]) > 6

# Tenant Manager breaker flapping
changes(readyz_check_status{dep="tenant_manager",status="degraded"}[10m]) > 3
```

---

## 9. Multi-Tenant Mode

When `MULTI_TENANT_ENABLED=true`, fetcher's data plane switches from a single shared Mongo/RabbitMQ connection to per-tenant connections resolved through `tmmongo.Manager` and `tmrabbitmq.Manager`. The global `/readyz` answer changes accordingly:

- **Mongo + RabbitMQ → `n/a`** in the global response. A global `up` would be a lie — the dep is 47 different databases, not one.
- **Redis stays global.** It's a shared schema cache (manager) or multi-tenant discovery store (worker), not per-tenant.
- **`/readyz/tenant/:id` is enabled.** This endpoint validates the tenant ID against `tmclient.GetActiveTenantsByService("fetcher")`, then runs the same dep probes scoped to that tenant's resources.

### Tenant cache semantics

`tmclient` caches the tenant-existence answer for `MULTI_TENANT_CACHE_TTL_SEC` (default 120s). This means:

- A tenant just created in Tenant Manager may return 404 from `/readyz/tenant/:id` for up to 120s.
- A tenant just disabled may continue returning 200 for up to 120s.
- Operators diagnosing tenant onboarding should wait out the TTL before assuming a bug, or bounce the replica to force a cache refresh.

### Breaker interaction

When the Tenant Manager circuit breaker is `open`, **every** call to `/readyz/tenant/:id` returns 503 during the tenant-validation phase — even for tenants that are definitely active. This is by design: fetcher cannot safely authorize a tenant request without reaching TM.

**Alert on the global `/readyz` dep `tenant_manager`, not on per-tenant 503s.** A single TM outage will generate one alert from the global endpoint and potentially thousands from per-tenant endpoints; the former is actionable, the latter is noise.

---

## 10. SaaS Mode TLS Enforcement

`DEPLOYMENT_MODE=saas` activates `ValidateSaaSTLS()` during bootstrap — before any dep is dialed. It walks every platform-dep config and rejects plaintext.

### What gets checked

| Dep             | Failure condition                                                       |
| --------------- | ----------------------------------------------------------------------- |
| MongoDB         | `MONGO_URI` missing `tls=true` OR `MONGO_TLS_CA_CERT` is empty          |
| RabbitMQ        | URI scheme is `amqp://` OR `RABBITMQ_TLS=false`                         |
| Redis           | Redis TLS disabled AND `REDIS_HOST` is non-empty                        |
| S3              | Endpoint scheme is `http://` (unless AWS default, which is implicit TLS)|
| Tenant Manager  | URL scheme is `http://` AND `MULTI_TENANT_ALLOW_INSECURE_HTTP != true`  |

Any failure aborts boot with a log line naming the dep and the specific constraint violated. Fix the DSN or — for user-datasources and dev envs — lower `DEPLOYMENT_MODE`.

### Escape hatch

`MULTI_TENANT_ALLOW_INSECURE_HTTP=true` bypasses the TM URL scheme check **only**. It exists for private-network stubs (e.g. TM reached via in-cluster `http://tenant-manager.tenant-manager.svc:8080`) where mTLS is enforced at a layer below the URL. Document every activation in the deployment manifest and audit it at release time.

### Silent-inactive pitfall

If `DEPLOYMENT_MODE` is unset (or set to anything other than `saas`), validation is skipped with **no warning**. Plaintext broker URIs in production will boot fine, expose the service, and wait for an incident. This is the reason item 4 in §5 is a blocker — every production GitOps manifest must set `DEPLOYMENT_MODE: "saas"` explicitly.

---

## 11. Operational Runbook

Keep this section open in the pager window. Each scenario starts with the visible symptom, then the three-step diagnosis, then the fix.

### Why is /readyz returning 503?

1. `curl -s :4006/readyz | jq '.checks'` — identify the failing dep by its `status` field.
2. Read the `error` field; cross-reference with `readyz_check_status{status="down"}` in Grafana to see if this is flapping or persistent.
3. Fix the underlying dep: TLS cert, credentials, network path, or upstream service. /readyz will flip back to 200 on the next successful probe.

### Why is /health returning 503?

Startup self-probe failed — a required dep did not respond at boot. K8s will restart the pod on the configured `failureThreshold`.

1. `kubectl logs <pod>` — look for `startup_self_probe_failed` and preceding per-dep `self_probe_check` lines.
2. Identify which dep never reached `status=up` during boot.
3. Fix that dep's connectivity or credentials. Pod restarts clean if self-probe clears.

> The distinction between `/readyz` and `/health` is deliberate: `/readyz` gates steady-state traffic, `/health` gates pod lifecycle. A pod can become temporarily unready (503 on `/readyz`) without being restarted. A pod that fails self-probe (503 on `/health`) is considered broken beyond recovery and is recycled.

### Service won't start — "TLS required for X"

`DEPLOYMENT_MODE=saas` and a platform dep DSN is plaintext.

1. Log line names the failing dep (e.g. `saas mode requires TLS: rabbitmq scheme must be amqps`).
2. Update the offending DSN/env in GitOps.
3. If this is a pre-prod env without TLS terminators, set `DEPLOYMENT_MODE=byoc` for that env instead (never `local` in Kubernetes).

### In-flight requests killed during deploy

Drain grace is shorter than K8s reads the new readiness state.

1. Check `READYZ_DRAIN_DELAY_SEC` vs `readinessProbe.periodSeconds * failureThreshold`. Drain must be longer.
2. Bump `READYZ_DRAIN_DELAY_SEC` (e.g. `12` → `20`).
3. Bump `terminationGracePeriodSeconds` by the same amount (e.g. `45` → `53`). K8s SIGKILL respects no drain.

### /readyz/tenant/:id returns 404 for a known-active tenant

Tenant cache is stale (up to `MULTI_TENANT_CACHE_TTL_SEC`, default 120s).

1. Wait 2 minutes and retry.
2. If still failing, `kubectl delete pod <fetcher-replica>` to force a fresh cache population.
3. If still failing post-restart, the tenant is genuinely not active in TM — verify via `tmclient` admin UI or direct TM API.

### Every /readyz/tenant/:id returns 503

Tenant Manager breaker is open. The tenant-validation step can't run.

1. Check global `/readyz`'s `tenant_manager` dep — should show `degraded` with `breaker_state: open`.
2. Investigate TM itself: logs, dashboards, latency. Breaker auto-recovers on success.
3. Do not page on per-tenant 503s in this scenario — they are symptoms, not root cause.

### /metrics has no readyz_* series

Registry is not exposed or middleware order swallowed the route.

1. `curl -s :4006/metrics | grep readyz_` — should return three metric families.
2. If empty, check `routes.go`: `/metrics` must be mounted **before** any auth middleware and the Prometheus registry handler must be attached.
3. Worker-only: verify `HEALTH_PORT` Fiber app was started — check startup log for `health server listening on :4007`.

### selfprobe_result stays 0 forever for a dep

The startup self-probe never reached that dep.

1. Check startup logs for `startup_self_probe_started` followed by per-dep `self_probe_check` entries.
2. If the dep's entry is missing, the adapter didn't register — check for a config panic/early return before `RunSelfProbe`.
3. If the entry is present but `result=failed`, the dep is unreachable at boot. Fix connectivity before expecting /health to clear.

---

## 12. Performance & Cost Considerations

### Tenant Manager request rate

In MT mode, every `/readyz` call makes one `tmclient.GetActiveTenantsByService("fetcher")` request. At default `periodSeconds: 5` and N replicas:

```
TM RPM = N × (60 / 5) = 12N
```

A 10-replica fetcher fleet drives ~120 RPM to Tenant Manager **just from /readyz**. Factor this into TM's HPA and rate-limit sizing. If TM is close to saturation, raise fetcher's `periodSeconds` to `10` — cheaper than scaling TM.

### S3 HeadBucket rate

Worker `/readyz` calls `HeadBucket` on every probe. AWS S3 pricing:

```
$0.0004 per 1k requests × 12 req/min × 60 × 24 × 30 × N replicas
= $0.21 per replica per month (rounded)
```

A 10-replica worker fleet costs ~$2/month in S3 HeadBucket traffic. MinIO/SeaweedFS are free. This is a non-issue at current scale but documented for finance when fleets grow past 100 replicas.

### Redis connection pool

The readiness-only Redis client uses default `go-redis` settings, which means `PoolSize=10` per replica. At fleet scale (e.g. 50 replicas) that's 500 idle Redis connections just for probes.

> **Follow-up PR flagged:** override to `PoolSize=2` for the probe-only client. Not shipped in this gate.

### Worker CPU limit

Helm default is `200m`. With Go 1.25's default `GOMAXPROCS=1` under container limits **and** the /readyz fanout spawning one goroutine per dep, the worker can starve during probe bursts.

> **Follow-up PR flagged:** bump worker default to `500m`. For prod, set explicitly in GitOps values.

### MT-Redis double-client

In MT mode, workers hold **two** Redis connections — one for the event-listener (long-lived pub/sub), one for /readyz probes. Capacity planners: count workers twice when sizing Redis connection limits.

---

## 13. Scope Fence

`/readyz` does **not** cover:

- **Synthetic business-logic transactions** — it's infrastructure readiness, not end-to-end validation. Synthetic tests belong to a separate probe layer.
- **Certificate validity and expiry windows** — /readyz reports the current TLS handshake outcome, not certificate freshness. Cert rotation alerting lives in cert-manager metrics.
- **Performance SLIs (p99 latency, error rate)** — belong to RED dashboards, not readiness. A slow service is still ready; an unavailable one is not.
- **User datasources (Postgres, MySQL, Oracle, SQL Server)** — these are the tenant's data sources, not fetcher's. Their health is the tenant's concern and is surfaced through ingestion error logs, not readiness.
- **Per-tenant business rules** — the global `/readyz` is tenant-agnostic. Business-rule validation (e.g. "does tenant X have ingestion quota remaining") belongs to the request path, not readiness.

Scope creep on this endpoint is the failure mode most likely to re-introduce the exact class of bug the Monetarie incident exposed. Keep it tight.

---

## 14. Common Errors

Quick reference for error strings operators will see, in order of likelihood.

| Error fragment                                 | Root cause                                                    | First-step fix                                                                   |
| ---------------------------------------------- | ------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `401 on /readyz`                               | Endpoint mounted behind auth middleware (should be impossible)| Check `routes.go` ordering — /readyz must register before any auth middleware.   |
| `tls: handshake failure`                       | Wrong CA cert, expired cert, or TLS version mismatch          | Verify `MONGO_TLS_CA_CERT` points to a valid PEM; check cert expiry.             |
| `dial tcp: connection refused`                 | Dep not running or wrong host/port                            | Verify dep is up via direct `nc -zv host port`; check DNS.                       |
| `circuit breaker open`                         | Upstream is failing repeatedly; breaker tripped protectively  | Check upstream service health. Breaker auto-recovers after cool-down window.     |
| `tenant manager unreachable`                   | TM down, TM breaker open, or network path broken              | Global `/readyz` will show `tenant_manager: degraded`. Investigate TM.           |
| `multi-tenant mode is disabled` (400)          | `/readyz/tenant/:id` hit while `MULTI_TENANT_ENABLED=false`   | Either enable MT or don't call this endpoint.                                    |
| `saas mode requires TLS: ...`                  | Plaintext DSN while `DEPLOYMENT_MODE=saas`                    | Fix DSN to use `mongodb+srv://`, `amqps://`, `rediss://`, or `https://`.         |
| `context deadline exceeded` with latency ~2000ms | Dep probe hit its 2s per-dep timeout                        | Dep is overloaded or unreachable. Investigate network path and dep capacity.     |

---

## 15. Security Considerations

- **All probe endpoints are unauthenticated by design.** K8s kubelet has no way to present credentials. Restrict at NetworkPolicy/Ingress layer.
- **`/readyz/tenant/:id` is a tenant-existence oracle.** An attacker hitting this endpoint can enumerate valid tenant IDs by observing 200 vs 404 responses. Restrict network access accordingly.
- **`/metrics` exposes Go runtime + readyz timings.** Reveals dep latency, restart frequency, GC pressure. Restrict to Prometheus scrapers via a dedicated NetworkPolicy with pod selectors.
- **Error fields are routed through `sanitize()`** which redacts `user:pass@`, `password=*`, `pwd=*`. **Caveats:**
  - Bearer tokens, JWTs, AWS access keys are **not yet covered** — follow-up PR tracks this.
  - DO NOT paste raw `/readyz` responses into public channels; treat output as internal-only until sanitize coverage is extended.
- **`MULTI_TENANT_ALLOW_INSECURE_HTTP=true` bypasses TM TLS enforcement.** Permitted only for private-network TM stubs where lower-layer mTLS protects the connection. Each activation must be documented in the deployment manifest with a justification and an audit entry.
- **Future work:** an audit log entry for `MULTI_TENANT_ALLOW_INSECURE_HTTP=true` at boot is flagged as a follow-up (§16).

---

## 16. Follow-up PRs / Known Limitations

These items are **tracked, not blocking.** Operators should be aware of them when reasoning about behavior or planning capacity:

- **Dead code cleanup** in `readyz_adapters.go`: `ensureCloseAMQP` call chain, `multiTenantRedisURLFromCfg`, the `TenantHandler` alias, and the `panickingChecker` test scaffold are unused post-refactor.
- **Sanitize coverage extension** — bearer tokens, JWTs, AWS access keys, and generic secret-looking strings are not yet redacted from `error` fields.
- **Probe-only Redis `PoolSize=2`** — override the default 10 to avoid fleet-wide idle connection bloat (§12).
- **Worker CPU limit bump** from `200m` to `500m` default in the helm chart (§12).
- **Audit log entry** on `MULTI_TENANT_ALLOW_INSECURE_HTTP=true` activation at boot.
- **`runtime.SafeGo` wrapping** for /readyz fanout goroutines — catches per-dep panics without tearing down the probe.
- **Refactor `LoadConfig` to use `commons.SetConfigFromEnvVars`** — aligns fetcher config loading with the rest of the Lerian fleet.
- **Migration to lib-commons v5** — pending platform-wide rollout; fetcher tracks the current v4 baseline.

---

## References

Monetarie incident post-mortem lives in `docs/fetcher/incidents/` (to be written by incident review). The `/readyz` implementation PR is tagged `gate11-complete`. Upstream contract spec: [ring:dev-readyz][ring-dev-readyz].

[ring-dev-readyz]: https://github.com/LerianStudio/ring — 12-gate readiness orchestrator
