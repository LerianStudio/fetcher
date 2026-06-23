# ST-T008-03 Preserve Manager Error And Rate Limit Boundaries

## Goal

Ensure Engine errors map to current Manager business/HTTP behavior and rate limiting remains outside Engine core.

## Prerequisites

- ST-T008-02 complete.

## Files

- Update `components/manager/internal/services/query/test_connection.go` if needed.
- Update `components/manager/internal/adapters/http/in/connection.go` only if error mapping requires it.
- Update relevant Manager tests.
- Read `pkg/net/http/errors.go`.
- Read `pkg/errors.go`.
- Read `pkg/ratelimit/ratelimit.go`.

## Mapping table

| Engine error category | Current Manager behavior to preserve |
| --- | --- |
| validation | Existing validation business error mapping |
| notFound | Current not-found response for missing connection/job resources |
| conflict | Current active-job conflict behavior |
| connection | Current connection-test failure behavior |
| unauthorizedContext | Current scoped not-found or forbidden behavior, matching existing tests |
| timeout | Current safe internal or gateway-style mapping if already present |

## Implementation brief

- Add a Manager-local error mapper only if direct mapping would leak Engine internals into handlers.
- Keep rate-limit checks in Manager service or adapter code.
- Preserve current `pkg/errors.go` and `pkg/net/http/errors.go` behavior for public responses.
- Do not add rate-limit dependencies to `pkg/engine`.

## Test plan

1. RED: Add tests for Engine validation, not-found, conflict, connection, and unauthorized-context errors mapping to existing Manager error behavior.
2. GREEN: Add a small adapter mapping function in Manager if direct error mapping would leak Engine internals into handlers.
3. REFACTOR: Keep rate-limit store checks in Manager service/adapter code, not `pkg/engine`.

## Acceptance assertions

- Engine validation, not-found, conflict, connection, unauthorized-context, and timeout errors map to current Manager-compatible behavior.
- Existing rate-limit tests remain Manager-owned.
- Engine dependency tests still pass.

## Commands

- Run `go test ./components/manager/internal/services/query ./components/manager/internal/services/command`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: error compatibility holds and boundary tests still pass.

## Expected failure/pass patterns

- Failure: a rate-limit dependency in `pkg/engine` fails the boundary test.
- Pass: HTTP-facing errors remain compatible while Engine errors stay stable and transport-neutral.

## Rollback

- Revert Manager error mapping changes made in this subtask.
