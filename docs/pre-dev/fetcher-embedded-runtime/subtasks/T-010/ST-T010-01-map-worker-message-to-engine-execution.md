# ST-T010-01 Map Worker Message To Engine Execution

## Goal

Map current Worker queued job messages and persisted job records into Engine execution requests.

## Prerequisites

- T-007 complete.
- T-009 complete.

## Files

- Update `components/worker/internal/services/extract_data.go`.
- Update `components/worker/internal/services/extract_data_test.go`.
- Update `components/worker/internal/services/extract_data_additional_test.go`.
- Read `components/worker/internal/services/service.go`.
- Read `pkg/model/job/job_queue.go`.
- Read `pkg/model/job.go`.

## Mapping table

| Current Worker input | Engine execution input | Preservation rule |
| --- | --- | --- |
| Queue job ID | `executionId` or compatibility metadata | Preserve lookup of persisted job by ID |
| Persisted job mapped fields | Extraction plan or execution request fields | Preserve datasource/table/field selection exactly |
| Persisted job filters | Execution request filters | Preserve filter nesting and operators |
| `metadata.source` | Execution metadata | Preserve for adapter-level `plugin_crm` selection |
| Product name | `TenantContext.productName` | Preserve product isolation |
| Organization ID | `TenantContext.organizationId` | Preserve compatibility scoping |
| Completed job check | Worker adapter guard | Skip Engine execution when current behavior skips completed jobs |

## Implementation brief

- Add a Worker adapter mapper from queue payload plus persisted job into Engine execution input.
- Keep message parsing, job repository lookup, ack/nack behavior, and completed-job skip logic in Worker.
- Call Engine runner only after current compatibility guards pass.
- Preserve product and organization context in every Engine call.

## Test plan

1. RED: Add tests proving job ID, mapped fields, filters, metadata source, product name, organization ID, and tenant context are mapped into Engine execution inputs.
2. GREEN: Add a Worker adapter mapping layer and call Engine runner for execution while preserving current skip behavior for completed jobs.
3. REFACTOR: Keep message parsing, ack/nack expectations, and job repository lookup in Worker adapter code.

## Acceptance assertions

- Worker tests prove job ID, mapped fields, filters, metadata source, product name, organization ID, and tenant context reach Engine execution.
- Completed jobs are still skipped without Engine execution.
- Queue parsing and repository lookup remain Worker-owned.

## Commands

- Run `go test ./components/worker/internal/services`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: Worker tests pass with Engine execution mapping.

## Expected failure/pass patterns

- Failure: missing product or organization context fails mapping tests before execution.
- Pass: Engine fake receives one execution call for an eligible pending job.

## Rollback

- Revert Worker mapping changes in `extract_data.go` and related tests.
