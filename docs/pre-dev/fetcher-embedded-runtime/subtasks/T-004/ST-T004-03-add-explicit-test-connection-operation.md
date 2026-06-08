# ST-T004-03 Add Explicit Test Connection Operation

## Goal

Implement Engine `TestConnection` so connector construction and connectivity testing are separate and host-controlled.

## Prerequisites

- ST-T004-02 complete.

## Files

- Create `pkg/engine/test_connection.go`.
- Create `pkg/engine/test_connection_test.go`.
- Read `components/manager/internal/services/query/test_connection.go`.
- Read `components/manager/internal/services/query/test_connection_test.go`.

## Implementation brief

- Implement `TestConnection` as an Engine operation that resolves a scoped connection, decrypts credentials through the protector, obtains a connector, calls explicit connectivity test behavior, and closes the connector.
- Keep rate limiting in Manager compatibility code.
- Treat connector construction errors and connectivity errors as safe Engine connection errors.
- Enforce product scope before connector access.

## Test plan

1. RED: Add tests for successful fake connector test, unknown connection, wrong product scope, connector test failure, and close-after-test behavior.
2. GREEN: Implement `TestConnection` using `ConnectionStore`, `CredentialProtector`, and `ConnectorRegistry`.
3. REFACTOR: Keep rate limiting out of Engine; Manager remains responsible for rate-limit middleware or service checks.

## Acceptance assertions

- Successful fake connector test returns a success result.
- Unknown or wrong-product connections fail before connector construction.
- Connector close is attempted on success and failure.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./components/manager/internal/services/query`.
- Run `go test ./...`.
- Expected pass pattern: Engine can test connections through connector contracts and Manager tests remain unchanged.

## Expected failure/pass patterns

- Failure: wrong product returns unauthorized-context or scoped not-found error.
- Pass: Manager rate-limit tests remain outside Engine.

## Rollback

- Remove `pkg/engine/test_connection.go` and `test_connection_test.go`.
