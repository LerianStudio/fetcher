# ST-T005-01 Implement Discover Schema Operation

## Goal

Add Engine schema discovery over connector contracts with optional schema cache support.

## Prerequisites

- T-004 complete.

## Files

- Create `pkg/engine/schema_ops.go`.
- Create `pkg/engine/schema_ops_test.go`.
- Update `pkg/engine/memory/cache.go`.
- Read `components/manager/internal/services/query/get_connection_schema.go`.
- Read `pkg/model/schema.go`.

## Implementation brief

- Resolve the connection by scoped config name or ID through Engine ports.
- Use optional cache before connector discovery when cache policy allows it.
- Discover schema through connector contracts and normalize it into the canonical Engine schema model.
- Keep Redis cache behavior behind an adapter; Engine sees only `SchemaCache`.

## Test plan

1. RED: Add tests for cache miss discovery, cache hit, unknown connection, connector discovery failure, and system-table filtering if current behavior requires it.
2. GREEN: Implement `DiscoverSchema` using `ConnectionStore`, `CredentialProtector`, `ConnectorRegistry`, and optional `SchemaCache`.
3. REFACTOR: Keep returned schema canonical and independent from Manager HTTP response formatting.

## Acceptance assertions

- Cache hit returns schema without connector discovery.
- Cache miss discovers schema and writes through the cache when configured.
- Connector discovery failures become safe Engine schema errors.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./components/manager/internal/services/query`.
- Run `go test ./...`.
- Expected pass pattern: discovery works with fake connectors and optional cache.

## Expected failure/pass patterns

- Failure: unknown connection returns a scoped not-found error.
- Pass: schema discovery works with no Redis dependency in core.

## Rollback

- Remove schema discovery implementation and tests from this subtask.
