# ST-T005-03 Preserve CRM schema adapter compatibility

## Goal

Preserve `plugin_crm` schema behavior as adapter-level compatibility for the first Engine release. Do not add a generic Engine core extension API for CRM in this subtask.

## Prerequisites

- ST-T005-02 complete.
- T-001 CRM seam characterization exists.

## Files

- Create or update `pkg/enginecompat/plugincrm/schema_adapter.go` if an adapter seam is needed.
- Create or update `pkg/enginecompat/plugincrm/schema_adapter_test.go`.
- Read `components/manager/internal/services/query/validate_schema.go`.
- Read `components/worker/internal/services/extract_crm_data.go`.

## Implementation brief

- Keep `plugin_crm` out of generic Engine schema validation behavior.
- If current Manager schema compatibility needs special handling, place it in an explicit adapter package such as `pkg/enginecompat/plugincrm` or in Manager compatibility mapping.
- Name every public test/helper with `plugin_crm` or `CRMCompatibility` so product-specific behavior is visible.
- Preserve current Manager validation behavior for `metadata.source=plugin_crm`.

## Test plan

1. RED: Add tests proving `plugin_crm` behavior is invoked only through the explicit compatibility adapter or Manager mapping.
2. GREEN: Implement the smallest adapter-level mapping that preserves behavior without hardcoding CRM as generic datasource behavior.
3. REFACTOR: Keep names explicit: use terms like `CRMCompatibility` or `plugin_crm`, not generic names that hide product policy.

## Acceptance assertions

- Non-CRM validation does not execute CRM compatibility mapping.
- CRM compatibility remains outside generic Engine core contracts.
- Existing Manager schema tests still pass.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./components/manager/internal/services/query ./components/worker/internal/services`.
- Run `go test ./...`.
- Expected pass pattern: CRM compatibility is deliberate, adapter-owned, and test-covered.

## Expected failure/pass patterns

- Failure: a direct CRM branch inside generic Engine validation fails review and should fail boundary-oriented tests if it imports adapter code.
- Pass: CRM behavior is preserved only when the compatibility adapter or Manager mapping selects it.

## Rollback

- Remove CRM compatibility adapter and tests added in this subtask.
