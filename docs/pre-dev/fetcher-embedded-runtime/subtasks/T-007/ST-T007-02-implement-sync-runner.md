# ST-T007-02 Implement Sync Runner

## Goal

Implement synchronous Engine execution over an extraction plan using connector contracts.

## Prerequisites

- ST-T007-01 complete.

## Files

- Create `pkg/engine/runner.go`.
- Create `pkg/engine/runner_test.go`.
- Update `pkg/engine/memory/registry.go`.
- Read `components/worker/internal/services/extract_data.go`.

## Implementation brief

- Execute each planned datasource work item through connector contracts.
- Ensure connectors close on success and failure.
- Aggregate rows into the canonical direct result model for direct mode.
- Keep Worker job status, result storage encryption, and notifications outside the Engine runner.

## Test plan

1. RED: Add tests for single-source execution, multi-source execution, connector query failure, close-on-success, close-on-failure, and result row counting.
2. GREEN: Implement `ExecuteExtraction` for synchronous direct mode using fake connectors.
3. REFACTOR: Keep Worker job status and notification logic outside Engine runner.

## Acceptance assertions

- Single-source and multi-source fake connector plans execute successfully.
- Connector query failures return safe extraction errors.
- Close is attempted for every connector that is opened.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./components/worker/internal/services`.
- Run `go test ./...`.
- Expected pass pattern: Engine runner executes fake planned work and existing Worker tests are unaffected.

## Expected failure/pass patterns

- Failure: connector query error stops or marks execution according to the selected failure policy.
- Pass: row counts equal the sum of fake connector results.

## Rollback

- Remove `pkg/engine/runner.go` and `runner_test.go`.
