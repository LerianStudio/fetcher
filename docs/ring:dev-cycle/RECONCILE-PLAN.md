# Reconcile Plan — Fetcher Embedded Runtime (tenant-model gap + develop reconciliation)

> Created 2026-06-01, on branch `feat/fetcher-as-runtime-engine`, before a context compact.
> This file is the durable source of truth for the next steps. The dev-cycle state lives in
> `docs/ring:dev-cycle/current-cycle.json`; task specs in `docs/pre-dev/fetcher-embedded-runtime/`.

## Where we are (factual snapshot)

Dev-cycle: `ring:dev-cycle` lean backend flow, feature `fetcher-embedded-runtime`, mode `automatic`, 10 tasks.

| Task | State | Notes |
|------|-------|-------|
| T-001 | ✅ Done (accepted as-is) | Characterization + dependency boundary tests. Gate 8 had 25 iterations of platform remediation (worker MT, rabbitmq, streaming, lib-commons) accepted by user on 2026-06-01. Ended at commit `de2fb12`. |
| T-002 | ✅ Done (validated) | Engine contracts + facade + in-memory harness. Gate 8 scoped, 9/9 PASS in 2 iters (1 MEDIUM typed-nil fixed `ada7032`). Commits `e21f613`, `888d402`, `e1c5134`, `ada7032`. |
| T-003 | 🔄 Gate 0 done (3/3), **Gate 8 BLOCKED** | Connection lifecycle + credential protection + active-job conflict hook. Commits `baf2286`, `045c8b6`, `3df6af9` (= HEAD). Gate 8 iteration 1 = ISSUES_FOUND. |
| T-004..T-010 | pending | Not started. |

**Key cycle mechanism established at T-001/T-002:** `state.review_scope_policy` — Gate 8 reviews ONLY `task.review_baseline_commit..HEAD` (the task's own commits), NOT `origin/main...HEAD`. This fixed the T-001 runaway (207 files) → T-002/T-003 each reviewed ~10-16 pkg/engine files only.
- T-002 baseline: `de2fb12`. T-003 baseline: `ada7032`.

## The blocker (why we stopped)

T-003 Gate 8 found **1 HIGH + 3 MEDIUM**. The HIGH escalated into a domain-model problem confirmed by the repo owner:

**Correct multi-tenancy hierarchy (authoritative):**
```
tenantId            ← top-level SaaS isolation boundary
  ├── organization  ← business org (N per tenant)
  └── product       ← matcher, reporter, others (N per tenant)
```
`organization ≠ tenant`. The Engine's `TenantContext` (pkg/engine/context.go) models only `OrganizationID + ProductName + RequestID` — **tenantId is absent**. The in-memory store scope key is `(organizationID, productName)` — **no tenant**. This is a cross-tenant collision by construction in an embeddable library.

**develop investigation (origin/develop tip `9449b41`, we are 26 behind):**
- develop HAS tenant-in-context: lib-commons `tmmiddleware` → `tmcore.ContextWithTenantID()` on every route; `tmcore.GetTenantIDContext()` in services; `X-Tenant-ID` header Manager→Worker (`create_fetcher_job.go:690`, `consumer.go:356`); fail-closed tenant Mongo (`pkg/mongodb/tenant.go:30` `ErrTenantContextRequired`); DB-per-tenant via `GetDatabaseForTenant(ctx, tenantId)`.
- develop isolates tenants **physically (database-per-tenant)**, NOT via a tenantId field on the connection. Connection scoping there is still `(org, product)` + global config_name unique index — which is effectively `(tenant, config_name)` because the DB *is* the tenant boundary.
- `pkg/engine` does NOT exist on develop. No prior art for "tenantId in connection model" — develop solves it by DB routing.
- **Implication:** the Engine, being an in-process embeddable library, CANNOT rely on DB-per-tenant routing. It MUST carry tenantId in `TenantContext` and in the scope key. Owner confirmed: tenant TEM que estar no contexto.

**The 3 MEDIUM (fold into the same remediation):**
1. code-reviewer: `DeleteConnection` calls active-execution checker BEFORE existence check (Update checks existence first). Make Delete symmetric: Find → not-found → guard → Delete.
2. test-reviewer: missing test — duplicate Create WITH protector enabled (assert existing ciphertext untouched, no second record).
3. test-reviewer: missing tenant/org isolation test through the Engine (all current tests use `org-1`).

## The plan

### Phase A — Reconcile with origin/develop (FIRST, before any more engine work)
Rationale: T-004..T-010 migrate Manager/Worker onto the Engine. Doing that 26 commits behind develop's canonical tmcore multi-tenancy = painful reconciliation + a divergent tenant model. Reconcile now so the Engine is built against develop's real conventions.

1. **Checkpoint:** commit the pending dev-cycle state edits (`current-cycle.json`, `tasks.md`) so the working tree is clean. Suggested: `chore(dev-cycle): checkpoint T-003 gate-8 state before develop reconciliation`.
2. **Safety net:** `git branch backup/pre-develop-merge-20260601 HEAD` (cheap rollback point).
3. **Merge** (recommended over rebase — preserves our 40 commits' history; rebase would replay 40 commits over 26 = conflict hell): `git merge origin/develop`.
   - Expected conflict hotspots: our T-001 platform remediation (`components/worker/*` MT consumer, `pkg/rabbitmq/*`, streaming, `components/manager/*` config) vs develop's own multi-tenant hardening (consumer gating, publisher, tenant event listener, SSRF guard). `pkg/engine/*` will NOT conflict (absent on develop).
   - Resolve favoring develop's tmcore conventions where they overlap our remediation.
4. **Verify:** `go build ./...`, `go vet ./...`, `make lint`, `go test ./...` green post-merge. Fix fallout.
5. Commit the merge.

> DECISION PENDING from owner: merge vs rebase; separate work branch vs current. Default = merge on current branch with the backup branch as safety net.

### Phase B — Fix the tenant model (dispatch ring:backend-engineer-golang, TDD)
Re-opens "completed" contract surface from T-002 + T-003. Treat as a remediation pass.

- **B1 — TenantContext (T-002, pkg/engine/context.go):** add `TenantID string` as the top-level boundary. `IsMultiTenant()` keys off `TenantID != ""` (not OrganizationID). `NewTenantContext(tenantId, organizationID, productName, ...)` signature gains tenantId. Mirror lib-commons `tmcore` tenantId semantics as a VALUE (opaque string, length/format validation à la `IsValidTenantID`/`MaxTenantIDLength`) — do NOT import `tenant-manager` into pkg/engine (boundary: it pulls mongo-driver). Update constructor validation + all call sites/tests.
- **B2 — Scope key + uniqueness (T-003, pkg/engine/memory/store.go + connection_ops.go):** re-key `tenantScope` to `{tenantId, organizationID, productName}`. Config-name uniqueness → `(tenantId, organizationID, productName, configName)`.
  - SUB-DECISION for owner: does a connection live at `(tenant, org, product)` or `(tenant, product)` with org as metadata? Default assumed: `(tenant, org, product, configName)`. Confirm.
  - This resolves the original HIGH: uniqueness is rooted at tenant (matches develop's intent), not global, not bare per-product.
- **B3 — Host bridge (document, wire for real in T-008/T-010):** host adapter maps `tmcore.GetTenantIDContext(ctx)` → `engine.NewTenantContext(tenantId, org, product)`. Document the pattern now.
- **B4 — Fold in the 3 T-003 MEDIUMs:** Delete-guard symmetry; dup-with-protector test; tenant+org isolation test through the Engine.

### Phase C — Re-review + re-validate
1. **Recompute Gate 8 scope** post-merge: the baseline `ada7032` may no longer be a clean ancestor after merge. Scope Gate 8 to the pkg/engine diff (T-002+T-003+B remediation) — i.e. engine files changed vs the pre-engine point. Recompute the correct baseline at execution time.
2. **Re-run scoped Gate 8 for T-003**, MUST include: `multi-tenant-reviewer` (its prior PASS assumed org==tenant — INVALID, must re-verify under tenantId model), `code-reviewer` (Delete fix), `test-reviewer` (new tests), `business-logic-reviewer` (uniqueness resolution). Full 9-pool if scope warrants.
3. **Re-validate T-002** lightly — its `TenantContext` contract changed; confirm its Gate 9 criteria still hold (the contract is stronger now, not weaker).
4. **Gate 9 for T-003** once Gate 8 PASS: aggregate the 3 subtasks' criteria + user APPROVED.

### Phase D — Resume the cycle at T-004
On a develop-reconciled branch with the correct tenant model. T-004 = connector registry/factory. Continue the proven flow: Gate 0 per subtask → scoped Gate 8 → Gate 9.

## Carry-forward open items (do not lose)
- **T-001 boundary-test gap (HIGH-value):** `pkg/engine/dependency_test.go` whitelists `tenant-manager/core` by import path, but that package transitively pulls `mongo-driver` + `dbresolver`. The test checks import statements, not the build graph. Harden to assert on `go list -deps`. Re-verify after B1 that adding tenantId-as-value did NOT import tmcore into pkg/engine.
- **LOW items carried (from T-002/T-003 Gate 8):** partial `WithLimits` defaulting (IsZero all-or-nothing) → fix when limits consumed (T-006/T-007); `ResultReference.HMAC` placeholder naming → T-007; keyVersion-0 ambiguity (unprotected vs key-v0) → pin before T-008 Reveal; `ConnectionPatch` can't express field-clearing → pin omit/clear/null mapping at T-008; `fakeProtector` ignores TenantContext (can't assert correct tenant passed) → add when touching B4 tests.
- **Uncommitted now:** dev-cycle state edits in `current-cycle.json` + `tasks.md` (orchestration metadata) — commit in Phase A step 1.

## Commit/ref cheat-sheet
- T-001 end: `de2fb12` · T-002 end: `ada7032` · T-003 end / current HEAD: `3df6af9`
- origin/develop tip: `9449b41` (we are 40 ahead / 26 behind)
- Cycle state: `docs/ring:dev-cycle/current-cycle.json` (T-003 gate8_iteration_1 records the blocking issues)
