# ST-T008-01 Wire Manager Connection Services To Engine

## Goal

Refactor Manager connection command/query services to delegate shared connection behavior to Engine while preserving current service interfaces and HTTP compatibility.

## Prerequisites

- T-003 complete.
- Do not change route paths or response contracts.

## Files

- Update `components/manager/internal/services/command/create_connection.go`.
- Update `components/manager/internal/services/command/update_connection.go`.
- Update `components/manager/internal/services/command/delete_connection.go`.
- Update `components/manager/internal/services/query/get_connection.go`.
- Update `components/manager/internal/services/query/list_connections.go`.
- Update tests alongside those files.
- Read `components/manager/internal/bootstrap/config.go`.

## Mapping table

| Current Manager behavior | Engine operation | Compatibility owner |
| --- | --- | --- |
| `CreateConnection.Execute` validates input, protects password, persists connection | Engine create connection | Manager keeps request parsing and response mapping |
| `UpdateConnection.Execute` applies patch and active-job conflict check | Engine update connection with optional conflict checker | Manager keeps job repository adapter wiring |
| `DeleteConnection.Execute` soft-deletes and blocks active jobs | Engine delete connection with optional conflict checker | Manager maps delete result to current HTTP behavior |
| `GetConnection.Execute` and `ListConnections.Execute` scope by organization/product | Engine get/list connection | Manager keeps `X-Product-Name` and organization extraction |

## Implementation brief

- Inject or adapt Engine behind existing Manager service constructors with the smallest bootstrap change.
- Preserve current service method signatures and handler contracts.
- Map current Manager models to Engine connection inputs and Engine outputs back to current response models.
- Keep Fiber handlers, Swagger annotations, auth, license, and tenant middleware outside Engine.

## Test plan

1. RED: Adjust or add Manager service tests proving current inputs and outputs remain unchanged while Engine mocks/fakes receive delegated calls.
2. GREEN: Inject Engine into Manager services or compose adapter helpers with the smallest bootstrap change.
3. REFACTOR: Keep Fiber handlers untouched unless constructor wiring requires parameter changes.

## Acceptance assertions

- Existing Manager connection service tests pass with Engine delegation.
- Public response fields and status behavior remain unchanged.
- Engine boundary tests still prove no Manager internals enter `pkg/engine`.

## Commands

- Run `go test ./components/manager/internal/services/command ./components/manager/internal/services/query`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: Manager connection behavior remains compatible and Engine tests pass.

## Expected failure/pass patterns

- Failure: response shape drift appears in existing Manager tests.
- Pass: Engine fake or mock receives delegated calls while handlers remain untouched unless constructor wiring requires it.

## Rollback

- Revert Manager service wiring and tests changed in this subtask.
