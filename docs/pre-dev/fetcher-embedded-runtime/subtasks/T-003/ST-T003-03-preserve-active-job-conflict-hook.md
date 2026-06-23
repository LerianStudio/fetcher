# ST-T003-03 Preserve Active Job Conflict Hook

## Goal

Expose an Engine-side optional conflict check so update/delete can preserve Manager behavior that blocks changes while active jobs exist.

## Prerequisites

- ST-T003-02 complete.

## Files

- Update `pkg/engine/ports.go`.
- Update `pkg/engine/options.go`.
- Update `pkg/engine/connection_ops.go`.
- Update `pkg/engine/connection_ops_test.go`.
- Read `components/manager/internal/services/command/update_connection.go`.
- Read `components/manager/internal/services/command/delete_connection.go`.
- Read `pkg/ports/job/repository.go`.

## Implementation brief

- Add an optional logical conflict checker port for active executions.
- Call the checker before update and delete mutations.
- Return a conflict Engine error when active work blocks mutation.
- Do not import the current job repository into `pkg/engine`.

## Test plan

1. RED: Add tests where an injected active-execution checker blocks update and delete with a conflict Engine error.
2. GREEN: Add a small optional port such as `ActiveExecutionChecker` and call it before update/delete mutation.
3. REFACTOR: Keep the port logical; do not call the current job repository directly from `pkg/engine`.

## Acceptance assertions

- Update and delete are blocked when the injected checker reports active work.
- Operations proceed when the checker is absent or reports no active work.
- The port remains logical and does not make durable job storage mandatory in Engine.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./components/manager/internal/services/command`.
- Run `go test ./...`.
- Expected pass pattern: conflict behavior is reusable without making job storage mandatory in core.

## Expected failure/pass patterns

- Failure: active work returns conflict before store mutation.
- Pass: Manager compatibility tests still observe the active-job block.

## Rollback

- Remove the optional conflict-check port and related tests.
