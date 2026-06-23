# ST-T003-01 Implement Connection Store Operations

## Goal

Implement Engine connection create, get, list, update, and delete operations over `ConnectionStore`.

## Prerequisites

- T-002 complete.

## Files

- Create `pkg/engine/connection_ops.go`.
- Create `pkg/engine/connection_ops_test.go`.
- Update `pkg/engine/memory/store.go`.
- Read `components/manager/internal/services/command/create_connection.go`.
- Read `components/manager/internal/services/command/update_connection.go`.
- Read `components/manager/internal/services/command/delete_connection.go`.
- Read `components/manager/internal/services/query/list_connections.go`.

## Implementation brief

- Implement create, get, list, update, and delete through the `ConnectionStore` port.
- Enforce config-name uniqueness within `productName` scope.
- Treat deleted connections as unavailable for get/list/extraction selection.
- Preserve Manager-compatible patch semantics while keeping HTTP response mapping outside Engine.

## Test plan

1. RED: Add Engine tests for create, duplicate config name in product, get by ID, list by product, update patch behavior, and deleted connection exclusion.
2. GREEN: Implement minimal operations using `ConnectionStore` and `TenantContext`.
3. REFACTOR: Keep store behavior interface-driven; do not import MongoDB repositories into `pkg/engine`.

## Acceptance assertions

- Create stores a scoped connection and returns a redacted output.
- Duplicate config names fail only within the same product scope.
- Deleted connections do not appear in normal get/list results.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./components/manager/internal/services/command ./components/manager/internal/services/query`.
- Run `go test ./...`.
- Expected pass pattern: Engine connection operations pass and Manager behavior remains unchanged.

## Expected failure/pass patterns

- Failure: duplicate config name in the same product returns conflict.
- Pass: same config name in a different product remains isolated.

## Rollback

- Revert `pkg/engine/connection_ops.go`, `connection_ops_test.go`, and related memory-store changes.
