# ST-T002-02 Create Engine Facade And Options

## Goal

Create an importable `Engine` facade with constructor options and validation for required capabilities.

## Prerequisites

- ST-T002-01 complete.

## Files

- Create `pkg/engine/engine.go`.
- Create `pkg/engine/options.go`.
- Create `pkg/engine/ports.go`.
- Create `pkg/engine/engine_test.go`.

## Implementation brief

- Add an `Engine` facade and `Options` model that accept host-provided ports.
- Validate required capabilities at construction time instead of allowing nil-pointer failures later.
- Apply safe default limits when hosts omit custom limits.
- Keep optional ports optional: result sink, execution store, schema cache, event sink, tenant resolver, and observability hooks.

## Test plan

1. RED: Add tests for `New` with missing connector registry, missing credential protector when encrypted persistence is enabled, safe default limits, and optional ports.
2. GREEN: Implement `Engine`, `Options`, `New`, and port interfaces with only constructor behavior.
3. REFACTOR: Keep constructor validation explicit; do not hide missing required capabilities behind nil-pointer runtime failures.

## Acceptance assertions

- Invalid options return stable Engine errors.
- Valid fake collaborators produce a usable Engine instance.
- Constructor behavior does not import optional adapter packages.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: host code can construct an Engine with fake collaborators and receive stable Engine errors for invalid options.

## Expected failure/pass patterns

- Failure: missing required options produce deterministic validation errors.
- Pass: a minimal host test constructs Engine without MongoDB, Redis, RabbitMQ, S3, SeaweedFS, or Fiber.

## Rollback

- Remove `pkg/engine/engine.go`, `options.go`, `ports.go`, and `engine_test.go`.
