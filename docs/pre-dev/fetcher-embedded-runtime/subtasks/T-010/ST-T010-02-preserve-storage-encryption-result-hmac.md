# ST-T010-02 Preserve storage encryption result HMAC

## Goal

Preserve Worker result encryption, storage write, result path, and HMAC behavior while using Engine execution output.

## Prerequisites

- ST-T010-01 complete.

## Files

- Update `components/worker/internal/services/extract_data.go`.
- Update `components/worker/internal/services/extract_data_test.go`.
- Read `pkg/ports/storage/repository.go`.
- Read `pkg/crypto/crypto.go`.
- Read `components/worker/internal/services/job_notification.go`.

## Mapping table

| Current Worker result behavior | Engine output or adapter input | Preservation rule |
| --- | --- | --- |
| Plain extraction rows | Engine direct result data | Worker adapter receives canonical result data |
| Payload encryption | Worker cryptor flow | Preserve encryption before storage |
| Result storage path or URL | Worker storage repository output | Preserve current result reference shape |
| HMAC signing | Worker document signer flow | Preserve HMAC on completed job/result event |
| Storage provider | Worker/storage adapter | Keep S3 and SeaweedFS outside Engine core |
| Storage failure | Worker job failure path | Preserve current failed status and safe error behavior |

## Canonical metadata mapping

| Current Worker behavior | Canonical Engine-compatible metadata |
| --- | --- |
| Document signer HMAC | `integrity.algorithm=HMAC-SHA256`, `integrity.signature=<current HMAC>` |
| Encrypted stored payload | `protection.encrypted=true` |
| Worker/storage adapter applies encryption | `protection.appliedBy=adapter` |
| Result encryption key version, when available | `protection.keyVersion=<version>` |
| Adapter-managed encryption | `protection.mode=adapter-managed` |

## Implementation brief

- Adapt Engine direct result data into the current Worker encryption and storage flow.
- Keep cryptor, signer, storage repository, and provider-specific behavior in Worker or storage adapters.
- Preserve result path/URL fields used by current job records and notifications.
- Populate canonical `integrity` and `protection` metadata from existing Worker behavior without changing the stored payload, signer, or job fields.
- Keep credential protection separate from result protection. The canonical metadata in this task describes stored extraction results only.
- Test that encrypted stored bytes differ from plaintext JSON.

## Test plan

1. RED: Add tests for successful result storage, storage error failure, HMAC set on completed job, encrypted payload not equal plaintext JSON, result URL/path compatibility, canonical integrity metadata, canonical protection metadata, and unchanged current job/result HMAC behavior.
2. GREEN: Adapt Engine result data/reference into current Worker storage and job update behavior.
3. REFACTOR: Keep S3/SeaweedFS storage dependencies in Worker/storage adapters, not `pkg/engine`.

## Acceptance assertions

- Successful Engine result is encrypted, stored, signed, and attached to the completed job as before.
- Current job/result HMAC behavior remains unchanged.
- Engine-compatible metadata is populated with `integrity.algorithm=HMAC-SHA256` and `integrity.signature` set to the current HMAC value.
- Engine-compatible metadata is populated with `protection.encrypted=true`, `protection.appliedBy=adapter`, `protection.mode=adapter-managed`, and `protection.keyVersion` when available.
- Storage failure marks the job failed according to current Worker tests.
- Engine core has no mandatory S3 or SeaweedFS dependency.

## Commands

- Run `go test ./components/worker/internal/services`.
- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: result compatibility remains intact and core boundary tests pass.

## Expected failure/pass patterns

- Failure: stored payload equals plaintext fixture data.
- Pass: HMAC and result URL/path match current Worker expectations.

## Rollback

- Revert storage/encryption/HMAC adaptation changes in Worker files.
