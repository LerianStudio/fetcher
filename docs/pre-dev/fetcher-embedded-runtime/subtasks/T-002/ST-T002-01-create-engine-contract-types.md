# ST-T002-01 Create Engine Contract Types

## Goal

Create the first importable Engine core contracts for context, errors, limits, connection, schema, extraction request, plan, execution, and result models.

## Prerequisites

- T-001 complete and boundary tests passing.
- Use `pkg/engine` as the public core package.

## Files

- Create `pkg/engine/context.go`.
- Create `pkg/engine/errors.go`.
- Create `pkg/engine/limits.go`.
- Create `pkg/engine/connection.go`.
- Create `pkg/engine/schema.go`.
- Create `pkg/engine/extraction.go`.
- Create `pkg/engine/contracts_test.go`.
- Read `pkg/model/connection.go`, `pkg/model/schema.go`, and `pkg/model/job.go`.

## Implementation brief

- Define contracts only; do not implement operation behavior in this subtask.
- Include tenant context, safe Engine errors, limits, connection inputs/outputs, schema snapshots, extraction requests, plans, execution state, results, and result references.
- Redact credentials from normal output types and error strings.
- Keep contracts importable without Manager, Worker, queue, storage, HTTP, or database packages.

## Test plan

1. RED: Write tests asserting zero-value safety, password redaction, stable error categories, default limits, and valid status constants.
2. GREEN: Add minimal structs and constants to satisfy tests without implementing operations.
3. REFACTOR: Keep contracts free of JSON-heavy service response assumptions unless they are part of the embedded API.

## Acceptance assertions

- Contract tests prove default limits and status/error constants are stable.
- Password-bearing input types do not leak secrets through normal output formatting.
- Boundary tests from T-001 still pass.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: contracts compile and boundary tests still pass.

## Expected failure/pass patterns

- Failure: importing a shell dependency into a contract file fails the dependency boundary test.
- Pass: host code can compile against `pkg/engine` contracts without external services.

## Rollback

- Remove the new `pkg/engine` contract files and `contracts_test.go`.
