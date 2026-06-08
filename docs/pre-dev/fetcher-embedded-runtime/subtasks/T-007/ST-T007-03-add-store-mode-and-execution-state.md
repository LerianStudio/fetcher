# ST-T007-03 Add store mode and execution state

## Goal

Support store-and-reference execution mode and optional execution-store status transitions.

## Prerequisites

- ST-T007-02 complete.

## Files

- Update `pkg/engine/runner.go`.
- Update `pkg/engine/runner_test.go`.
- Update `pkg/engine/ports.go`.
- Update `pkg/engine/memory/store.go`.
- Read `pkg/model/job.go`.

## Implementation brief

- Add store mode that writes extracted data to a host-provided `ResultSink` and returns a `ResultReference`.
- Carry canonical `integrity` and `protection` metadata through both direct and store modes.
- Require store mode to preserve metadata returned by the `ResultSink` or adapter, including `integrity.algorithm`, `integrity.digest` or `integrity.signature`, `protection.encrypted`, optional `protection.keyVersion`, optional `protection.mode`, and `protection.appliedBy`.
- Allow only `engine`, `adapter`, or `host` for `protection.appliedBy`.
- Keep credential protection separate from result protection. Store-mode result metadata describes extracted result bytes, not persisted datasource credentials.
- Add optional `ExecutionStore` calls for processing, completed, failed, and cancelled states.
- Keep durable scheduling and async retry policy host-owned.
- Ensure direct mode remains available without a result sink.

## Test plan

1. RED: Add tests for processing, completed, failed, and cancelled execution state transitions when an `ExecutionStore` is configured, plus direct-mode integrity/protection metadata, store-mode integrity/protection metadata, invalid `protection.appliedBy`, and credential protection metadata not appearing as result protection.
2. GREEN: Implement optional execution state updates and result sink writes in store mode.
3. REFACTOR: Treat durable async behavior as host-owned; Engine only defines status semantics and optional persistence calls.

## Acceptance assertions

- Store mode fails clearly when no result sink is configured.
- Store mode returns a result reference after successful sink write.
- Direct mode preserves canonical integrity and protection metadata on the returned result.
- Store mode preserves canonical integrity and protection metadata for the stored result reference.
- Integrity metadata includes `algorithm` and either `digest` or `signature` when integrity applies.
- Protection metadata includes `encrypted` and valid `appliedBy` when protection state is known.
- Credential protection metadata is not reused as result protection metadata.
- Optional execution-store transitions occur in the expected order.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: store mode returns `ResultReference` and direct mode still returns payload.

## Expected failure/pass patterns

- Failure: sink write error marks execution failed when execution store is configured.
- Pass: direct mode tests remain green without an execution store.

## Rollback

- Revert store-mode and execution-store changes.
