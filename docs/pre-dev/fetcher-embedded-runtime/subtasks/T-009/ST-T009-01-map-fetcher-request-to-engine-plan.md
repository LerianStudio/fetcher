# ST-T009-01 Map Fetcher Request To Engine Plan

## Goal

Map current Manager `FetcherRequest` inputs into Engine extraction requests and plans without changing public API structs.

## Prerequisites

- T-006 complete.
- T-008 complete.

## Files

- Update `components/manager/internal/services/command/create_fetcher_job.go`.
- Update `components/manager/internal/services/command/create_fetcher_job_test.go`.
- Read `pkg/model/job.go`.
- Read `pkg/model/job/job_queue.go`.

## Mapping table

| Current `FetcherRequest` field or behavior | Engine planning input | Preservation rule |
| --- | --- | --- |
| `data` / mapped fields | `ExtractionRequest.mappedFields` | Preserve datasource, table, and field names exactly |
| `filters` | `ExtractionRequest.filters` | Preserve filter nesting and operators |
| `metadata.source` | `ExtractionRequest.metadata.source` | Preserve for Worker compatibility and `plugin_crm` adapter selection |
| `X-Product-Name` | `TenantContext.productName` | Required for connection scope |
| organization ID from context | `TenantContext.organizationId` | Preserve current Manager scoping behavior |
| request validation errors | Engine planning errors mapped by Manager | Preserve public error behavior |

## Implementation brief

- Add a Manager-local mapper from current request structs to Engine planning input.
- Keep `CreateFetcherJob.Execute` signature and public API structs unchanged.
- Use Engine planning for request normalization and scope validation.
- Keep queue payload creation and job persistence out of Engine.

## Test plan

1. RED: Add tests proving mapped fields, filters, metadata source, product name, and organization ID are passed to Engine planning correctly.
2. GREEN: Add request-to-plan mapping while keeping current `CreateFetcherJob` public method signature.
3. REFACTOR: Keep adapter mapping close to Manager service; do not move HTTP/request compatibility structs into `pkg/engine`.

## Acceptance assertions

- Manager tests prove mapped fields, filters, metadata source, product name, and organization ID reach Engine planning.
- Invalid Engine planning output prevents job creation and publishing.
- Public Manager API structs remain unchanged.

## Commands

- Run `go test ./components/manager/internal/services/command`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: job creation uses Engine planning for request validation.

## Expected failure/pass patterns

- Failure: missing product name or unknown config name prevents job persistence.
- Pass: a valid request produces an Engine plan and continues through existing Manager compatibility logic.

## Rollback

- Revert `create_fetcher_job.go` and tests changed in this subtask.
