# ST-T010-04 Preserve plugin_crm extraction adapter compatibility

## Goal

Preserve `plugin_crm` extraction behavior through explicit Worker adapter compatibility mapping. Do not promote `plugin_crm` into generic Engine core execution behavior in the first Engine release.

## Prerequisites

- ST-T010-03 complete.
- T-005 CRM schema compatibility decision implemented.

## Files

- Update `components/worker/internal/services/extract_data.go`.
- Update `components/worker/internal/services/extract_crm_data.go` only if the adapter boundary changes.
- Update `components/worker/internal/services/extract_crm_data_test.go`.
- Update `components/worker/internal/services/extract_data_test.go`.

## Mapping table

| Current CRM behavior | Compatibility location | Preservation rule |
| --- | --- | --- |
| `metadata.source=plugin_crm` selects CRM extraction behavior | Worker adapter mapping or `pkg/enginecompat/plugincrm` | Preserve explicit selection only for CRM source |
| Non-CRM extraction uses normal Worker path | Worker adapter mapping | Must not execute CRM compatibility code |
| CRM-specific schema expectations | Manager/CRM compatibility adapter | Align with ST-T005-03 adapter decision |
| Engine generic runner | `pkg/engine` | Receives generic execution input and stays CRM-agnostic |

## Implementation brief

- Keep `plugin_crm` routing explicit in Worker adapter compatibility code.
- Ensure non-CRM requests use normal Engine runner mapping.
- Align Worker CRM extraction behavior with the schema compatibility decision from ST-T005-03.
- Name tests and helpers with `plugin_crm` or `CRMCompatibility` so the product-specific seam stays visible.

## Test plan

1. RED: Add tests proving `plugin_crm` requests still use CRM extraction behavior and non-CRM requests do not.
2. GREEN: Wire explicit CRM adapter mapping into Worker execution.
3. REFACTOR: Keep the name explicit so product-specific behavior does not masquerade as generic Engine core behavior.

## Acceptance assertions

- `plugin_crm` requests preserve current CRM extraction behavior.
- Non-CRM requests never use CRM compatibility mapping.
- Generic Engine runner remains CRM-agnostic.

## Commands

- Run `go test ./components/worker/internal/services`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: CRM behavior is preserved deliberately as adapter compatibility.

## Expected failure/pass patterns

- Failure: a non-CRM job executes CRM extraction compatibility.
- Pass: CRM tests show adapter mapping while `pkg/engine` boundary tests stay green.

## Rollback

- Revert CRM compatibility mapping changes in Worker and tests.
