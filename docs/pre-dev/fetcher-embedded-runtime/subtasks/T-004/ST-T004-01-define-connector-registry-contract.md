# ST-T004-01 Define Connector Registry Contract

## Goal

Define Engine connector and registry contracts with explicit construction, connection testing, query, schema discovery, and close behavior.

## Prerequisites

- T-003 complete.

## Files

- Update `pkg/engine/ports.go`.
- Create `pkg/engine/connector.go`.
- Create `pkg/engine/connector_test.go`.
- Update `pkg/engine/memory/registry.go`.
- Read `pkg/model/datasource/datasource-config.go`.

## Implementation brief

- Define connector contracts around explicit lifecycle steps: build, connect or test, discover schema, query, and close.
- Make registry lookup deterministic by datasource type.
- Return stable unknown-type errors for unsupported datasource types.
- Keep current concrete datasource implementations behind adapters, not inside core contracts.

## Test plan

1. RED: Add tests for registry lookup by datasource type, unknown type errors, fake connector lifecycle, and explicit test-connection behavior.
2. GREEN: Implement the interfaces and memory fake registry.
3. REFACTOR: Keep connector contracts independent from current concrete datasource structs.

## Acceptance assertions

- Fake connectors can prove construction, test, query, schema discovery, and close calls.
- Unknown datasource type returns a stable Engine error.
- Engine core remains independent from concrete database drivers.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./pkg/engine/memory/...`.
- Run `go test ./...`.
- Expected pass pattern: Engine can resolve fake connectors without concrete database drivers.

## Expected failure/pass patterns

- Failure: missing registry entry returns an unknown datasource error.
- Pass: a fake registry resolves all supported test connector types without network access.

## Rollback

- Revert connector contract and memory registry changes.
