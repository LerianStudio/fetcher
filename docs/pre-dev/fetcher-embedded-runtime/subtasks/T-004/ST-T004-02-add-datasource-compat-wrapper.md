# ST-T004-02 Add Datasource Compatibility Wrapper

## Goal

Create an optional adapter under `pkg/enginecompat/datasource` that wraps the current datasource factory behind Engine connector contracts.

## Prerequisites

- ST-T004-01 complete.
- Do not import this adapter from `pkg/engine`.

## Files

- Create `pkg/enginecompat/datasource/adapter.go`.
- Create `pkg/enginecompat/datasource/adapter_test.go`.
- Read `pkg/datasource/datasource_factory.go`.
- Read `pkg/model/datasource/datasource-config.go`.
- Read `pkg/crypto/crypto.go`.

## Implementation brief

- Create a wrapper outside core that adapts Engine connection definitions to the current datasource factory.
- Inject the current factory function so unit tests can use fakes instead of live databases.
- Map factory errors into safe Engine connector errors.
- Keep the wrapper import direction one-way: `pkg/enginecompat/datasource` may import `pkg/engine`, but `pkg/engine` must not import the wrapper.

## Test plan

1. RED: Add tests using fake or nil-safe inputs to prove the adapter maps Engine connection definitions into current datasource factory inputs and returns Engine connector errors safely.
2. GREEN: Implement the wrapper with dependency injection for the existing factory function so tests do not need live databases.
3. REFACTOR: Ensure `go list -deps github.com/LerianStudio/fetcher/pkg/engine` does not include this adapter or database drivers.

## Acceptance assertions

- The wrapper compiles separately from core.
- Tests prove mapping from Engine connection fields into current factory inputs.
- `go test ./pkg/engine/...` still passes the dependency boundary test.

## Commands

- Run `go test ./pkg/enginecompat/datasource/...`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: compatibility adapter compiles separately and core boundary tests still pass.

## Expected failure/pass patterns

- Failure: importing the wrapper from `pkg/engine` trips the boundary test.
- Pass: adapter tests use an injected fake factory and require no live datasource.

## Rollback

- Remove `pkg/enginecompat/datasource`.
