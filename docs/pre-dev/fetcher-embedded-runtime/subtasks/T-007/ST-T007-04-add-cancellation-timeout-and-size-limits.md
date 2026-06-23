# ST-T007-04 Add Cancellation Timeout And Size Limits

## Goal

Make Engine execution respect context cancellation, timeouts, and result size limits.

## Prerequisites

- ST-T007-03 complete.

## Files

- Update `pkg/engine/runner.go`.
- Update `pkg/engine/runner_test.go`.
- Update `pkg/engine/errors.go`.
- Update `pkg/engine/limits.go`.

## Implementation brief

- Check context cancellation before connector calls and during result assembly.
- Enforce execution timeout through context deadlines or resolved Engine limits.
- Track result size against configured limits before returning or sinking payloads.
- Use fake connectors that respect context to avoid goroutine leaks in tests.

## Test plan

1. RED: Add tests for cancelled context before query, timeout during fake connector query, result size exceeded, and safe timeout/cancelled error categories.
2. GREEN: Implement context checks and size accounting around connector query and result assembly.
3. REFACTOR: Avoid goroutine leaks in tests; fake connectors should respect context.

## Acceptance assertions

- Cancelled context exits without connector query work when cancellation happens first.
- Timeout during fake query returns a timeout-category Engine error.
- Oversized results fail before sink write or direct return.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: runner exits cleanly on cancellation, timeout, and size-limit failures.

## Expected failure/pass patterns

- Failure: a fake connector that ignores context causes the timeout test to expose a stuck execution path.
- Pass: cancellation and timeout tests complete deterministically.

## Rollback

- Revert cancellation, timeout, and size-limit runner changes.
