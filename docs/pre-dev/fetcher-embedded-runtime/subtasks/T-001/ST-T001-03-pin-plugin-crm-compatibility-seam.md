# ST-T001-03 Pin plugin_crm adapter compatibility seam

## Goal

Prevent `plugin_crm` behavior from silently becoming generic Engine core behavior. The first Engine release preserves it as adapter-level compatibility behavior only.

## Prerequisites

- ST-T001-01 completed.
- No schema or runner migration has started.

## Files

- Read `components/manager/internal/services/query/validate_schema.go`.
- Read `components/worker/internal/services/extract_crm_data.go`.
- Update `pkg/engine/seams_test.go`.

## Implementation brief

- Name the seam `plugin_crm_adapter_compatibility`.
- Anchor the seam to Manager validation and Worker extraction files.
- State that `plugin_crm` remains a Manager/Worker compatibility behavior in the first Engine release.
- Do not add a generic Engine extension API in this subtask.

## Test plan

1. RED: Add a characterization test that requires a seam named `plugin_crm_adapter_compatibility` with source anchors for Manager validation and Worker extraction.
2. GREEN: Add the seam metadata to the seam registry used by the test.
3. REFACTOR: Keep the seam metadata test-only unless production code already has a natural home; do not create a CRM extension API in this subtask.

## Acceptance assertions

- The seam name makes CRM behavior visibly adapter-specific.
- No Engine core contract treats `plugin_crm` as a generic datasource extension.
- Manager and Worker behavior remains unchanged.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./components/manager/internal/services/query ./components/worker/internal/services`.
- Run `go test ./...`.
- Expected pass pattern: behavior is unchanged and the adapter compatibility seam is explicitly named.

## Expected failure/pass patterns

- Failure: a generic CRM extension name or missing source anchor fails the seam test.
- Pass: `plugin_crm` is documented as compatibility behavior without changing execution behavior.

## Rollback

- Revert only the seam metadata and assertion added in this subtask.
