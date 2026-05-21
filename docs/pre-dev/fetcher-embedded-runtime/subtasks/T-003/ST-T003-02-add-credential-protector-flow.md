# ST-T003-02 Add Credential Protector Flow

## Goal

Ensure Engine connection operations protect credentials before persistence and never return raw credentials.

## Prerequisites

- ST-T003-01 complete.

## Files

- Update `pkg/engine/ports.go`.
- Update `pkg/engine/connection_ops.go`.
- Update `pkg/engine/connection_ops_test.go`.
- Read `pkg/crypto/crypto.go`.
- Read `components/manager/internal/services/command/create_connection.go`.

## Implementation brief

- Add a `CredentialProtector` port if it is not already present.
- Call the protector before persisted create/update writes that include passwords.
- Store only protected credential material and key-version metadata.
- Return safe errors when protection fails, with no raw password or encrypted value in messages.

## Test plan

1. RED: Add fake `CredentialProtector` tests proving create encrypts password, update encrypts changed password, outputs redact password fields, and protector errors return safe Engine errors.
2. GREEN: Wire credential protection into create/update operations.
3. REFACTOR: Keep host key management outside Engine; Engine only calls the injected protector.

## Acceptance assertions

- Create encrypts the password before storage.
- Update encrypts only when a password change is supplied.
- Outputs and errors never include raw credentials.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./...`.
- Expected pass pattern: credential behavior is covered without logging or returning secrets.

## Expected failure/pass patterns

- Failure: fake protector error returns a safe Engine error and no store mutation.
- Pass: stored credential differs from the plaintext test password.

## Rollback

- Revert credential-protector changes from `ports.go`, `connection_ops.go`, and tests.
