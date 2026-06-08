# ST-T006-03 Enforce tenant product scope

## Goal

Prevent extraction plans from selecting connections outside the provided tenant/product context.

## Prerequisites

- ST-T006-02 complete.

## Files

- Update `pkg/engine/context.go`.
- Update `pkg/engine/planner.go`.
- Update `pkg/engine/planner_test.go`.
- Read `pkg/model/connection.go`.
- Read `pkg/resolver/resolver.go`.

## Implementation brief

- Resolve persisted connections through `connectionStore` or resolver using the execution context scope.
- Require product scope for persisted external connections used in extraction plans.
- Reject mismatched product ownership before schema validation or execution.
- Treat missing or mismatched scoped connections as safe scoped not-found or unauthorized-context errors; do not reveal whether another product owns the config name.
- Preserve the `PlanExtraction` capability chain: scoped resolution, credential protection when needed, connector registry resolution, optional schema cache, and limits.
- Carry tenant ID and organization ID into the plan for adapters and observability.
- Do not perform host authorization; Engine only checks scope consistency on supplied context and connections.

## Test plan

1. RED: Add tests for missing product name, matching product, mismatched product, unknown scoped config name, optional tenant ID propagation, credential/protector failure under matching scope, schema cache hit, schema cache miss, invalid table/field/filter under matching scope, safe unauthorized-context error, and no raw credential material in the valid plan.
2. GREEN: Add scope checks when resolving connection references by config name.
3. REFACTOR: Keep authorization outside Engine; Engine enforces scope consistency, not user permissions.

## Acceptance assertions

- Missing product scope fails for persisted external connection planning.
- Matching product scope succeeds.
- Mismatched product scope fails before connector access.
- Unknown config names fail before execution without leaking cross-product existence.
- Credential/protector failures return safe errors under otherwise valid scope.
- Schema cache hits avoid connector schema discovery.
- Schema cache misses use connector discovery.
- Invalid table, field, or filter references fail before execution.
- Valid plans contain no raw credential material.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./components/manager/internal/services/command`.
- Run `go test ./...`.
- Expected pass pattern: cross-product plans are rejected before execution.

## Expected failure/pass patterns

- Failure: mismatched product returns safe unauthorized-context or scoped not-found error.
- Pass: tenant and organization metadata propagate into the plan without authorizing the actor.

## Rollback

- Revert tenant/product scope checks in planner and tests.
