# ST-T002-03 Create In-Memory Harness

## Goal

Create `pkg/engine/memory` so Engine tests and embedded examples can run without MongoDB, Redis, RabbitMQ, S3, SeaweedFS, Fiber, Manager, or Worker.

## Prerequisites

- ST-T002-02 complete.

## Files

- Create `pkg/engine/memory/store.go`.
- Create `pkg/engine/memory/sink.go`.
- Create `pkg/engine/memory/cache.go`.
- Create `pkg/engine/memory/registry.go`.
- Create `pkg/engine/memory/memory_test.go`.

## Implementation brief

- Implement in-memory collaborators that satisfy Engine ports for tests and embedded examples.
- Protect shared maps with mutexes where concurrent access is possible.
- Keep ordering deterministic when listing records.
- Avoid production persistence semantics; this harness is for local embedded use and tests.

## Test plan

1. RED: Add tests for in-memory connection storage, schema cache get/set, result sink put/get, fake connector registry lookup, and concurrent access where maps are shared.
2. GREEN: Implement small mutex-protected in-memory types satisfying Engine ports.
3. REFACTOR: Keep memory harness deterministic and no-op friendly; it is a test/embedded harness, not a production database.

## Acceptance assertions

- In-memory connection store, schema cache, result sink, execution store, and connector registry satisfy Engine ports.
- Tests pass without external services or environment variables.
- Concurrent access tests do not race under normal `go test` execution.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./pkg/engine/memory/...`.
- Run `go test ./...`.
- Expected pass pattern: all Engine packages pass without external services.

## Expected failure/pass patterns

- Failure: unsynchronized map access appears under concurrent harness tests.
- Pass: memory harness behavior is deterministic and boundary tests remain green.

## Rollback

- Remove `pkg/engine/memory` files created by this subtask.
