# ST-T006-01 Create extraction plan model

## Goal

Create deterministic Engine extraction plan models that normalize mapped fields and filters into per-datasource work items.

## Prerequisites

- T-005 complete.

## Files

- Create `pkg/engine/planner.go`.
- Create `pkg/engine/planner_test.go`.
- Read `pkg/model/job.go`.
- Read `components/manager/internal/services/command/create_fetcher_job.go`.

## Implementation brief

- Build `PlanExtraction` around the capabilities defined in `api-design.md`: `connectionStore` or resolver, `credentialProtector` when persisted encrypted credentials are used, `connectorRegistry`, optional `schemaCache`, and `limits`.
- Resolve config names to scoped connection definitions before building executable work items.
- Use schema validation results to turn request selections into executable plan steps; do not skip schema validation for persisted connections.
- Normalize mapped fields and filters into deterministic per-datasource work items.
- Preserve metadata needed by compatibility paths, including source metadata.
- Sort datasource, table, and field keys before producing plans.
- Do not execute connectors or publish jobs in the planner.

## Planning flow

| Stage | Input | Required capability | Output |
| --- | --- | --- | --- |
| Request normalization | `mappedFields`, `filters`, request metadata | `limits` | Deterministic normalized request |
| Connection resolution | Config names from normalized request | `connectionStore` or resolver | Scoped connection definitions |
| Credential preparation | Persisted encrypted credentials | `credentialProtector` | Connector-safe credential view with no raw credential material in the plan |
| Schema validation | Scoped connections plus mapped tables, fields, and filters | `connectorRegistry`, optional `schemaCache` | Validated schema selections |
| Executable planning | Validated selections and effective limits | `limits` | Deterministic executable plan |

## Test plan

1. RED: Add tests for deterministic plan creation from mapped fields, stable source ordering, filter attachment, metadata preservation, empty request rejection, unknown config name, credential/protector failure, cache hit, cache miss, invalid table/field/filter, and no raw credential material in the valid plan.
2. GREEN: Implement minimal plan building without connector execution.
3. REFACTOR: Keep plan structures immutable by convention after creation; expose copies where maps could be mutated by callers.

## Acceptance assertions

- Equivalent requests produce byte-stable or deeply equal plans.
- Empty mapped fields fail with a validation error.
- Unknown config names fail before execution.
- Credential/protector failures return safe errors.
- Schema cache hits avoid connector schema discovery.
- Schema cache misses use connector discovery.
- Invalid table, field, or filter references fail before execution.
- Filters attach only to their matching datasource/table/field paths.
- Valid plans contain no raw credential material.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: valid requests produce deterministic plans.

## Expected failure/pass patterns

- Failure: map iteration order causes nondeterministic test output.
- Pass: repeated runs produce the same plan ordering.

## Rollback

- Remove `pkg/engine/planner.go` and `planner_test.go`.
