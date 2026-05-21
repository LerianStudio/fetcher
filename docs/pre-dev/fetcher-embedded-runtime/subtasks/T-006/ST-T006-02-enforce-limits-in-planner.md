# ST-T006-02 Enforce limits in planner

## Goal

Enforce Engine limits during planning before any datasource query can execute.

## Prerequisites

- ST-T006-01 complete.

## Files

- Update `pkg/engine/limits.go`.
- Update `pkg/engine/planner.go`.
- Update `pkg/engine/planner_test.go`.

## Implementation brief

- Require `PlanExtraction` to receive effective `limits` from Engine configuration plus allowed request overrides.
- Apply limits before connection resolution where possible, before connector discovery where schema shape is already known, and before execution in all cases.
- Keep limit checks integrated with the request -> resolved connection -> schema validation -> executable plan flow from ST-T006-01.
- Resolve effective limits from Engine defaults plus allowed per-request overrides.
- Reject requests that exceed datasource, table, field, concurrency, timeout, or result-size limits before execution.
- Return validation errors with field paths that point to the violated limit.
- Keep connector-specific hard limits available for later adapter enforcement without requiring concrete drivers in core.

## Test plan

1. RED: Add tests for max datasource count, max table count, max field count, max filter count, max concurrency, timeout defaulting, invalid per-request override rejection, cache hit under limits, cache miss under limits, and valid plans containing no raw credential material after limit checks.
2. GREEN: Implement limit resolution and validation in `PlanExtraction`.
3. REFACTOR: Return canonical validation errors with safe field paths.

## Acceptance assertions

- Oversized requests fail during planning.
- Valid overrides within maximums are applied to the plan.
- Invalid overrides do not mutate default Engine limits.
- Unknown config names still fail before execution after limit validation.
- Schema cache hits avoid connector schema discovery when the request remains within effective limits.
- Schema cache misses use connector discovery when the request remains within effective limits.
- Invalid table, field, or filter references fail before execution when within limit boundaries.
- Valid plans contain no raw credential material.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: unsafe or oversized requests fail during planning.

## Expected failure/pass patterns

- Failure: a max-field violation returns before connector execution.
- Pass: a valid request at the configured limit boundary plans successfully.

## Rollback

- Revert planner limit enforcement and tests.
