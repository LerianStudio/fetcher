# ST-T001-01 Characterize Runtime Seams

## Goal

Create executable characterization around the current seams that the Engine will strangle: datasource factory side effects, Manager orchestration, Worker extraction, resolver, storage, and CRM compatibility behavior.

## Prerequisites

- Read `docs/pre-dev/fetcher-embedded-runtime/tasks.md` T-001.
- Do not modify source outside tests and the new Engine package skeleton required by this task.

## Files

- Read `pkg/datasource/datasource_factory.go`.
- Read `components/manager/internal/services/command/create_fetcher_job.go`.
- Read `components/manager/internal/services/query/validate_schema.go`.
- Read `components/worker/internal/services/extract_data.go`.
- Read `components/worker/internal/services/extract_crm_data.go`.
- Create `pkg/engine/doc.go` if `pkg/engine` does not exist.
- Create `pkg/engine/seams_test.go`.

## Implementation brief

- Record each seam as named metadata with a source path and a short compatibility reason.
- Cover datasource factory side effects, Manager job orchestration, Manager schema validation, Worker extraction, Worker CRM extraction, resolver behavior, storage result handling, and notification publishing.
- Keep the seam registry test-owned unless Engine already has a smaller natural contract home.
- Do not import Manager or Worker internals from `pkg/engine`.

## Test plan

1. RED: Add tests in `pkg/engine/seams_test.go` that document seam names as constants and fail if any required seam name is empty.
2. GREEN: Add the minimal seam registry in test code or a tiny unexported helper so the test passes without production behavior migration.
3. REFACTOR: Keep this as characterization only; do not import Manager or Worker internals into `pkg/engine`.

## Acceptance assertions

- `pkg/engine` exists and compiles with only the minimal skeleton needed for tests.
- `pkg/engine/seams_test.go` fails if a required seam name, source path, or reason is empty.
- The test output names the missing seam when characterization is incomplete.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: tests pass with no dependency on Fiber, RabbitMQ, MongoDB, Redis, S3, SeaweedFS, lib-auth, lib-license, or `components/*/internal` from `pkg/engine`.

## Expected failure/pass patterns

- Failure: a missing seam produces a direct assertion failure naming the absent seam key.
- Pass: the Engine package remains a characterization shell with no behavior migration.

## Rollback

- Remove `pkg/engine/seams_test.go` and `pkg/engine/doc.go` if this subtask is reverted.
