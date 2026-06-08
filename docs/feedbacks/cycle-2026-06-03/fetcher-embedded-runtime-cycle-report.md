# Dev Cycle Report: fetcher-embedded-runtime

**Cycle ID:** 4de9ba09-1807-4062-87e2-f588e8275f19
**Completed:** 2026-06-03
**Flow:** ring:dev-cycle lean backend (Gate 0 → Gate 8 → Gate 9, per task)
**Agent:** ring:backend-engineer-golang (all tasks)
**Language:** go | **Service type:** api (Manager) + worker (Worker) + library (pkg/engine)

## Outcome

The strangler extraction is complete. `pkg/engine` is now an importable, infra-free Engine core; the Manager (connections, schema, job-creation) and the Worker (extraction) both run over it; the legacy Worker extraction path is deleted. Products (matcher, reporter) can now embed the Engine in-process instead of calling Fetcher as a network service. The migration introduced intentional, documented breaking changes — tracked as breaking for the v2.0.0 cut: normalized extraction result table keys, indented (changed) stored result bytes, and a source-less notification routing-key change (`job.<status>` instead of `job.<status>.<source>`). Outside those deltas, the HTTP response shapes, queue payloads, and HMAC/status semantics were preserved. See `docs/compatibility-waivers.md` for the full breaking-change log.

## Per-Task Scorecard

All 10 tasks hit the scored bar (TDD RED+GREEN completed, coverage ≥ 85%, delivery PASS, lint 0, no file-size/license violations) → **score 100/100 each**. The score formula does not capture review-iteration cost; that is reported separately below as the real process signal.

| Task | Title | Gate 8 result | Gate 8 iters | Notable |
|------|-------|---------------|:---:|---------|
| T-001 | Characterize seams + dependency boundary | ACCEPTED_BY_USER | 25 | Origin/main cumulative-diff spiral (see Root Cause) |
| T-002 | Engine contracts, facade, in-memory harness | PASS 9/9 | 2 | isNilPort typed-nil guard |
| T-003 | Connection lifecycle + credential protection | PASS 9/9 | 2 | B2 tenant model landed (org/product dropped) |
| T-004 | Connector registry/factory + test semantics | PASS 9/9 | 2 | credential-decrypt-at-host-seam blessed |
| T-005 | Schema discovery + validation | PASS 9/9 | 2 | 2 business-logic HIGH (nested-field, CRM union) |
| T-006 | Extraction planner + limits + tenant scope | PASS 9/9 | 2 | CWE-770 probe bound; existence-oracle preserved |
| T-007 | Sync runner + result model + sink | PASS 9/9 | 3 | close-every-connector invariant; result-protection model |
| T-008 | Migrate Manager connection/schema | PASS 10/10 | 2 | FIRST migration; lib-observability triggered |
| T-009 | Migrate Manager job-creation/planning | PASS 9/9 | 2 | Option 2 (map, validate-at-execution) |
| T-010 | Migrate Worker extraction | PASS 11/11 | 2 | 1 CRITICAL + 3 HIGH remediated; strangler completion |

**Aggregate:** 10/10 tasks complete, 30 subtasks + 1 strangler-completion cleanup. Final suite 5551 pass / 84 packages, vet 0, golangci-lint 0, engine boundary clean (dependency_test 398 pass + forbidden-import grep empty). Coverage: engine ~97%, worker services 87.2%, enginecompat 96.9%, tablenorm 100%.

## Issues Found (by severity, across all Gate 8 reviews)

- **CRITICAL ×1** (T-010): generic-path filters silently dropped → full-table extraction (data-scoping defect). Caught by the parallel adversarial review; remediated (mapFilters nested-map shape + real-engine e2e guard).
- **HIGH ×~13** (across tasks): Oracle case-folding dropped (T-010), CRM span host:port leak (T-010), config-name uniqueness divergence (T-003), IsZero-drops-ConnectorHardLimits (T-006), connector build-failure untested (T-007), nested-field matching dropped (T-005), CWE-770 unbounded probe (T-006), typed-nil guards (T-002/T-004), + the T-001 platform-compliance set. All remediated.
- **MEDIUM/LOW ×many**: all remediated or carried-forward with rationale.

Zero unresolved Critical/High at cycle end.

## Root Cause (the one process failure worth analysis)

**T-001's 25 review iterations.** Gate 8 was run against the `origin/main...HEAD` cumulative diff (207 files, +11,560 lines) and spiraled remediating Ring-mandated platform compliance (lib-commons/observability/multi-tenant/streaming) unrelated to the 961-line characterization deliverable. **Fix applied mid-cycle:** the `review_scope_policy` — from T-002 onward, Gate 8 reviews ONLY `{task.review_baseline_commit}..HEAD` restricted to the task's own changes. Result: every subsequent task converged in 2-3 iterations. This single policy change is the highest-leverage learning of the cycle.

## Patterns Worth Carrying Forward

1. **Stale-spec correction recurred and the injection worked.** The pre-dev subtask specs predated the B2 owner decision (tenantId-only Engine), and also predated the streaming-hardening. Stale guidance surfaced at T-004, T-006, T-009 (org/product scoping), and T-010 (publisher-nil fallback). The orchestrator injected the locked correction into each agent dispatch and the agents honored it. Recommendation: when a pre-dev decision invalidates spec text, record it once in state and inject per-dispatch — do not trust the agent to reconcile silently.

2. **Carry-forward tracking had operational value, not just bookkeeping.** The T-010 CRITICAL (filter shape unreadable by the planner) was predicted verbatim as a T-009 carry-forward ("the filter projection MUST be pinned against PlanStep.Filters, not the typed shape"). It still shipped GREEN at the subtask level because the agent's tests validated the mapper and adapter halves in isolation. The parallel Gate 8 review crossing the real engine caught it. Recommendation: for migration tasks, require at least one end-to-end test that crosses the real boundary the carry-forward warns about.

3. **Adversarial parallel review beats TDD-in-isolation for integration defects.** Every CRITICAL/HIGH that the implementing agent's own green suite missed was an integration-seam defect (filter shape, Oracle case, span redaction) — exactly the class isolated unit tests bypass. The 9-default + conditional-specialist panel earned its cost here.

4. **The rule-vs-mechanic split resolved every strangler-depth fork.** "Is this a RULE (→ Engine) or a MECHANIC/normalization/presentation (→ host seam)?" cleanly decided Option 2 (schema-name normalization at the enginecompat seam, engine stays literal) at both T-008 and T-010.

## Accepted Deferrals (owner-timestamped 2026-06-03)

- **Performance: double-connect + schema round-trip** (plan-then-execute, no SchemaCache in the worker engine) — accepted as a known resource regression vs legacy; SchemaCache wiring or plan/execute connection reuse is a post-cycle follow-up.
- **Performance: peak-heap (~4 payload copies)** — cannot collapse without breaking the HMAC-over-MarshalIndent-plaintext byte-identity (ST-02); row/byte pushdown into connector Query is the structural follow-up.
- **HMAC semantics** — Fetcher signs plaintext (confirmed in code); the matcher-ciphertext divergence reconciles at matcher's embedding, not in Fetcher.

## Improvements for Next Cycle

1. **Scope Gate 8 to the task baseline from task 1.** The review_scope_policy was reactive (born from T-001's spiral). Establish it at cycle init.
2. **Reconcile pre-dev specs against locked owner decisions before Gate 0.** A one-pass spec-vs-decision diff at cycle start would have pre-empted the recurring stale-spec injections.
3. **Mandate a real-boundary e2e test for every migration task's carry-forward.** Would have caught the T-010 filter CRITICAL at Gate 0 instead of Gate 8.
