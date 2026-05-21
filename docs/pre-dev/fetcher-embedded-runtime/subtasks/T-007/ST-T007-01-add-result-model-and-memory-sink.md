# ST-T007-01 Add result model and memory sink

## Goal

Implement canonical result and result reference behavior with an in-memory result sink.

## Prerequisites

- T-006 complete.

## Files

- Update `pkg/engine/extraction.go`.
- Create `pkg/engine/result_test.go`.
- Update `pkg/engine/memory/sink.go`.
- Read `pkg/ports/storage/repository.go`.
- Read `components/worker/internal/services/job_notification.go`.

## Implementation brief

- Model direct result data and stored result references separately.
- Include format, row count, plaintext size, canonical `integrity`, canonical `protection`, and completion timestamp fields.
- Model `integrity.algorithm` plus one of `integrity.digest` or `integrity.signature` when integrity applies.
- Model `protection.encrypted`, optional `protection.keyVersion`, optional `protection.mode`, and `protection.appliedBy` with allowed values `engine`, `adapter`, and `host`.
- Treat HMAC as one possible integrity signature, not as the result model itself.
- Keep credential protection separate from result protection. Do not reuse credential encryption fields or terminology to describe extracted result bytes.
- Implement a memory result sink that can store and retrieve payloads by logical reference.
- Keep physical storage backend details out of `pkg/engine`.

## Test plan

1. RED: Add tests for direct result data, stored result reference, format, row count, size bytes, canonical integrity metadata, canonical protection metadata, allowed `protection.appliedBy` values, HMAC as optional integrity signature, and no sensitive data in errors.
2. GREEN: Implement result structs and memory sink behavior.
3. REFACTOR: Keep physical storage details out of `pkg/engine`; use logical `ResultReference` fields.

## Acceptance assertions

- Direct mode result carries data and no required sink reference.
- Direct mode result can carry canonical integrity and protection metadata when the Engine or host applies it.
- Store mode result carries a result reference and no mandatory physical backend type.
- Store mode result carries canonical integrity and protection metadata for the bytes written or referenced.
- Integrity metadata includes `algorithm` and either `digest` or `signature` when integrity applies.
- Protection metadata includes `encrypted` and valid `appliedBy` when protection state is known.
- Credential protection metadata is not reused as result protection metadata.
- Result errors and references do not include sensitive payload data.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./pkg/engine/memory/...`.
- Run `go test ./...`.
- Expected pass pattern: result model supports direct and sink-backed execution modes.

## Expected failure/pass patterns

- Failure: result sink errors return safe storage-category Engine errors.
- Pass: memory sink tests retrieve exactly the bytes written through the logical reference.

## Rollback

- Revert result model and memory sink changes.
