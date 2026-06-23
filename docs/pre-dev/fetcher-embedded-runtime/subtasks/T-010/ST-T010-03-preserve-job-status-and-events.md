# ST-T010-03 Preserve job status and events

## Goal

Preserve Worker job status transitions and optional RabbitMQ notification behavior when configured and execution is delegated to Engine.

## Prerequisites

- ST-T010-02 complete.

## Files

- Update `components/worker/internal/services/extract_data.go`.
- Update `components/worker/internal/services/job_notification.go` only if mapping requires it.
- Update `components/worker/internal/services/job_notification_test.go`.
- Update `components/worker/internal/services/extract_data_test.go`.
- Read `pkg/ports/publisher/repository.go`.

## Mapping table

| Current Worker lifecycle behavior | Engine result/error input | Preservation rule |
| --- | --- | --- |
| `PENDING` to `IN_PROGRESS` | Execution start | Worker repository update remains before execution |
| `IN_PROGRESS` to `COMPLETED` | Successful Engine result plus storage success | Preserve completed status and result metadata |
| `IN_PROGRESS` to `FAILED` | Engine error or adapter storage/signing error | Preserve failed status and safe error metadata |
| Completion notification | Engine result mapped to current event payload | RabbitMQ publisher remains Worker adapter |
| Failure notification | Engine error mapped to current event payload | Preserve optional publisher behavior |
| Publisher absent | No notification emitted | Execution still updates job status |

## Implementation brief

- Map Engine execution success and errors into current job repository updates.
- Preserve optional notification publisher behavior.
- Keep RabbitMQ publisher and event routing outside Engine.
- Verify status transitions in order, not only final state.

## Test plan

1. RED: Add tests for pending-to-in-progress-to-completed, pending-to-in-progress-to-failed, optional notification publisher, completion event payload, and failure event payload.
2. GREEN: Map Engine runner results/errors into existing job repository updates and notification calls.
3. REFACTOR: Keep RabbitMQ publisher optional and outside Engine core.

## Acceptance assertions

- Pending jobs transition to in-progress before Engine execution.
- Successful jobs transition to completed after result storage/signing succeeds.
- Engine or adapter failures transition to failed and produce current failure event behavior.

## Commands

- Run `go test ./components/worker/internal/services`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: current job and event semantics remain compatible.

## Expected failure/pass patterns

- Failure: completion event emits before completed job persistence when current tests expect the opposite.
- Pass: publisher nil path still completes job updates without notification calls.

## Rollback

- Revert status/event mapping changes in Worker files.
