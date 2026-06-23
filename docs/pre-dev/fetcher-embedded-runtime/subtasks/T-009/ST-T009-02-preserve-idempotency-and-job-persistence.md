# ST-T009-02 Preserve Idempotency And Job Persistence

## Goal

Keep current five-minute duplicate detection, request hash, and job persistence behavior while moving planning to Engine.

## Prerequisites

- ST-T009-01 complete.

## Files

- Update `components/manager/internal/services/command/create_fetcher_job.go`.
- Update `components/manager/internal/services/command/create_fetcher_job_test.go`.
- Read `pkg/ports/job/repository.go`.
- Read `pkg/model/job.go`.

## Mapping table

| Current job behavior | Owner after migration | Preservation rule |
| --- | --- | --- |
| Request hash calculation | Manager compatibility service | Hash stays stable for equivalent requests |
| Five-minute duplicate lookup | Manager job repository adapter | Duplicate returns existing job behavior |
| New job persistence | Manager job repository adapter | New job remains `PENDING` before publish |
| Planning validation | Engine planner | Planning failure creates no job |
| Durable async job state | Manager/Worker compatibility stores | Engine core does not require job repository |

## Implementation brief

- Preserve request hash calculation order and duplicate window semantics.
- Run Engine planning before creating a new job when planning can reject the request.
- Keep durable job persistence in Manager compatibility code.
- Ensure planning failure, repository failure, and duplicate success have distinct tests.

## Test plan

1. RED: Add tests for duplicate request returning existing job, new request creating pending job, Engine planning failure not creating job, and hash stability for equivalent requests.
2. GREEN: Preserve current repository calls around Engine planning.
3. REFACTOR: Do not move durable job store semantics into mandatory Engine core; Manager owns compatibility job persistence.

## Acceptance assertions

- Duplicate request returns the existing job without publishing a new message.
- New valid request creates a pending job after successful planning.
- Engine planning failure does not create or publish a job.

## Commands

- Run `go test ./components/manager/internal/services/command`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: idempotency and job persistence behavior match current tests.

## Expected failure/pass patterns

- Failure: hash drift causes duplicate-detection tests to fail.
- Pass: repository call order preserves current idempotency semantics.

## Rollback

- Revert idempotency and persistence changes in `create_fetcher_job.go` and tests.
