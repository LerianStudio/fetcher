# ST-T008-02 Wire Manager Schema Services To Engine

## Goal

Refactor Manager schema discovery and validation services to call Engine schema operations while preserving current API behavior.

## Prerequisites

- ST-T008-01 complete.
- T-005 complete.

## Files

- Update `components/manager/internal/services/query/get_connection_schema.go`.
- Update `components/manager/internal/services/query/validate_schema.go`.
- Update `components/manager/internal/services/query/get_connection_schema_test.go`.
- Update `components/manager/internal/services/query/validate_schema_test.go`.
- Read `components/manager/internal/adapters/cache/schema_cache.go`.

## Mapping table

| Current Manager behavior | Engine operation | Compatibility owner |
| --- | --- | --- |
| `GetConnectionSchema` reads connection, uses datasource, returns current schema response | Engine discover schema | Manager maps canonical schema to current response |
| `ValidateSchema` checks mapped fields and filters | Engine validate schema | Manager maps validation report to current success/failure response |
| Redis schema cache is optional infrastructure | Engine `SchemaCache` port | Manager wires Redis adapter |
| `plugin_crm` schema compatibility | Adapter-level CRM compatibility mapping | Manager or `pkg/enginecompat/plugincrm`, not generic Engine core |

## Implementation brief

- Delegate schema discovery and validation to Engine operations.
- Keep current response shapes, business errors, and cache wiring in Manager.
- Preserve `plugin_crm` behavior only through explicit adapter-level compatibility mapping.
- Do not move Redis or HTTP response formatting into Engine.

## Test plan

1. RED: Add compatibility tests for current schema response shape, validation success, validation failure, cache behavior, and CRM compatibility source handling.
2. GREEN: Delegate discovery and validation to Engine, mapping Engine reports/errors back to existing service responses.
3. REFACTOR: Keep Redis cache as Manager adapter wiring; Engine sees only the `SchemaCache` port.

## Acceptance assertions

- Current schema discovery tests pass unchanged or with only constructor wiring changes.
- Current validation success and failure responses remain compatible.
- CRM schema behavior is explicit and not part of generic Engine core.

## Commands

- Run `go test ./components/manager/internal/services/query`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: schema routes retain behavior through Manager service tests.

## Expected failure/pass patterns

- Failure: CRM behavior triggered for non-CRM requests fails compatibility tests.
- Pass: Redis cache is exercised through the Engine cache port without becoming a core dependency.

## Rollback

- Revert schema service wiring and tests changed in this subtask.
