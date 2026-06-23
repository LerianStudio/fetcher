# ST-T009-03 Preserve RabbitMQ Dispatch Boundary

## Goal

Ensure Manager continues publishing the existing queue payload after Engine planning, while RabbitMQ remains outside Engine core.

## Prerequisites

- ST-T009-02 complete.

## Files

- Update `components/manager/internal/services/command/create_fetcher_job.go`.
- Update `components/manager/internal/services/command/create_fetcher_job_test.go`.
- Read `pkg/rabbitmq/rabbitmq.go`.
- Read `pkg/model/job/job_queue.go`.

## Mapping table

| Current dispatch behavior | Owner after migration | Preservation rule |
| --- | --- | --- |
| Queue payload type | Manager compatibility service | Preserve `pkg/model/job/job_queue.go` shape |
| RabbitMQ publisher call | Manager RabbitMQ adapter | Publish only after plan and job persistence succeed |
| Queue topology | Infra/RabbitMQ compatibility | No topology changes in this task |
| Engine plan | Engine planner | Used as validation/planning input, not serialized as queue payload unless compatibility requires it later |
| Publish failure handling | Manager service | Preserve current error and persistence behavior from tests |

## Implementation brief

- Publish the existing queue message shape after successful Engine planning and job persistence.
- Do not import RabbitMQ into `pkg/engine`.
- Add negative tests for planning failure and persistence failure to prove publish is not called.
- Keep queue routing keys and topology unchanged.

## Test plan

1. RED: Add tests proving publish is called with the existing job message shape after successful plan and not called on planning or persistence failure.
2. GREEN: Keep RabbitMQ publisher invocation in Manager after Engine plan success.
3. REFACTOR: Run `go test ./pkg/engine/...` to prove RabbitMQ is still absent from core dependencies.

## Acceptance assertions

- Existing queue payload fields are preserved.
- Publish occurs exactly once on successful new job creation.
- Publish does not occur on duplicate, planning failure, or persistence failure unless current behavior says otherwise and tests prove it.

## Commands

- Run `go test ./components/manager/internal/services/command`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: dispatch behavior is preserved and Engine dependency boundary holds.

## Expected failure/pass patterns

- Failure: RabbitMQ appears in `pkg/engine` dependency output.
- Pass: Manager dispatch tests verify existing queue payload preservation.

## Rollback

- Revert RabbitMQ dispatch changes in `create_fetcher_job.go` and tests.
