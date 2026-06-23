# ST-T001-04 Reconcile Go version source of truth

## Goal

Block Engine implementation until dev-cycle reconciles Go version documentation or tooling. `go.mod` is the source of truth and declares Go `1.25.9`.

## Prerequisites

- Complete ST-T001-02 so the Engine boundary test already uses the module path from `go.mod`.
- Do not edit `docs/PROJECT_RULES.md` during this pre-dev pass.

## Files

- Read `go.mod`.
- Read `docs/PROJECT_RULES.md`.
- Read `CLAUDE.md`.
- Modify the docs or tooling file chosen by dev-cycle during implementation to align with Go `1.25.9`.

## Implementation brief

- Treat `go.mod` as authoritative for the Go toolchain version.
- Before writing Engine code, decide whether implementation should update architecture documentation, local setup guidance, CI tooling, or generated environment checks.
- Preserve `docs/PROJECT_RULES.md` during this pre-dev recalibration. This subtask only records the implementation obligation.
- If dev-cycle updates documentation later, keep the update narrow: replace the stale Go version and explain that `go.mod` owns the executable toolchain contract.

## Test plan

- Add or update a documentation/tooling consistency check only if the repository already has a natural place for that check.
- Run version-sensitive tests after reconciliation.

## Acceptance assertions

- A developer entering Engine implementation sees one Go version decision: Go `1.25.9` from `go.mod`.
- No implementation task depends on the stale `1.25.6` value from `docs/PROJECT_RULES.md`.
- This pre-dev pass does not modify `docs/PROJECT_RULES.md`.

## Commands

- `go version`
- `go test ./pkg/engine/...`
- `go test ./...`

## Expected failure/pass patterns

- Before reconciliation, a reviewer can point to the mismatch between `go.mod` and architecture docs.
- After reconciliation, version references used by implementation agree with `go.mod` or explicitly defer to it.

## Rollback

- Revert only the documentation or tooling version-alignment file changed during dev-cycle implementation.
