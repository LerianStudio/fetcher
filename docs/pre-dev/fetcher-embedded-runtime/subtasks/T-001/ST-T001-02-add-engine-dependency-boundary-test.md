# ST-T001-02 Add Engine Dependency Boundary Test

## Goal

Make the Engine dependency boundary executable by failing tests if `pkg/engine` imports service-shell dependencies or Manager/Worker internals.

## Prerequisites

- `pkg/engine` exists from ST-T001-01.
- The boundary must apply to `github.com/LerianStudio/fetcher/pkg/engine`, not optional adapter packages under `pkg/enginecompat`.

## Files

- Create `pkg/engine/dependency_test.go`.
- Read `go.mod` for module path.
- Read `docs/pre-dev/fetcher-embedded-runtime/dependencies.md` for forbidden dependency classes.

## Implementation brief

- Use `go list -deps github.com/LerianStudio/fetcher/pkg/engine` from a Go test to inspect transitive imports.
- Keep forbidden dependency substrings explicit and grouped by shell concern: HTTP, queue, state stores, storage, deployment, auth, license, and service internals.
- Apply the boundary only to `pkg/engine`; do not block optional wrappers under `pkg/enginecompat/*`.
- Fail with the forbidden import and its class so future violations are cheap to understand.

## Test plan

1. RED: Add a test that shells out to `go list -deps github.com/LerianStudio/fetcher/pkg/engine` and checks forbidden import substrings.
2. GREEN: Ensure current minimal `pkg/engine` has no forbidden imports.
3. REFACTOR: Keep the forbidden list explicit and readable: `components/manager/internal`, `components/worker/internal`, `github.com/gofiber/fiber`, `github.com/rabbitmq/amqp091-go`, `go.mongodb.org/mongo-driver`, `github.com/redis/go-redis`, AWS S3 packages, SeaweedFS packages, `github.com/LerianStudio/lib-auth`, and `github.com/LerianStudio/lib-license-go`.

## Acceptance assertions

- The test fails if `pkg/engine` imports Manager internals, Worker internals, Fiber, RabbitMQ, MongoDB, Redis, AWS S3, SeaweedFS, lib-auth, or lib-license.
- The test passes for optional adapter packages that are outside `pkg/engine`.
- The test reads the module path from the repository instead of hardcoding a different module.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: tests pass and will fail on future forbidden core imports.

## Expected failure/pass patterns

- Failure: adding a forbidden import to `pkg/engine` produces a failure naming the import path.
- Pass: `go list -deps` for `pkg/engine` contains only allowed core dependencies.

## Rollback

- Remove `pkg/engine/dependency_test.go`.
