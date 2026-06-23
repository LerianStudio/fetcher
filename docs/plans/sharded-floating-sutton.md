# Plan: Extract `pkg/engine` into its own Go module + dual-module release CI

## Context

`pkg/engine` is infrastructure-free at the **package** level (the allowlist boundary test
proves its non-test build graph is stdlib + module-local only), but it physically lives
inside the `github.com/LerianStudio/fetcher/v2` module. Any consumer importing it inherits
fetcher's entire `go.mod` via MVS — lib-commons v5.5.0, mongo-driver v2, Fiber, every DB
driver, AWS SDK — despite the engine using none of them.

The midaz **reporter** is about to embed the engine in-process and that import must carry
**zero** third-party dependencies. This plan makes the *module* boundary match the *package*
boundary, and wires the release pipeline to cut both modules together so they never drift.

Two deliverables: **(A)** the module extraction, **(B)** the dual-module release CI.

### Confirmed decisions (from the user)
- **goleak stays.** The engine's only third-party import is `go.uber.org/goleak` (2 test
  files, 6 goroutine-leak assertions). It will appear in the engine module's `require` as a
  **test-only** dependency. The consumer guarantee (zero third-party in a consumer's
  `go.sum`) still holds because `go 1.26.4` module-graph pruning drops the engine's test-only
  deps from consumers. The boundary test runs `go list -deps` **without** `-test`, so it
  stays green. → The acceptance line "EMPTY require block" is reinterpreted as **"no non-test
  third-party in the build graph; `require` contains only the test-scoped goleak."** The
  smoke test is the real proof.
- **Release CI is in-repo only.** `release.yml` is a one-line call to the shared org workflow
  `LerianStudio/github-actions-shared-workflows@v1.32.0`, which cannot be edited from this
  repo. All dual-module logic lives in `.releaserc.yml` (the `@semantic-release/exec` plugin
  is already wired but empty) + a committed Node script + one new in-repo workflow for
  verification. No cross-repo change.

### Ground-truth facts
- Module path: `github.com/LerianStudio/fetcher/v2`, `go 1.26.4`. No `go.work` today.
- **83 Go files** import `fetcher/v2/pkg/engine` (engine internal tests, `pkg/engine/memory`,
  `pkg/enginecompat/*`, manager, worker).
- Latest parent tag: `v2.0.0-beta.5` → next parent: **`v2.0.0-beta.6`**.
- Engine non-test build graph: stdlib + module-local only (verified via `go list -deps`).
- `dependency_test.go` (780 lines) mixes two concerns: the engine dependency-boundary tests
  **and** repo-root doc-hygiene guards (`CLAUDE.md`/`README.md`/`PROJECT_RULES.md` go-version
  guards). The doc guards reference repo-root files and **cannot move** into the engine module.

---

## A. Module extraction

### A1. Create `pkg/engine/go.mod`
- `module github.com/LerianStudio/fetcher/pkg/engine` (**no `/v2`** — the submodule's fetchable
  path is repo-root + subdir; it starts its own version line at v1).
- `go 1.26.4` (match parent).
- `cd pkg/engine && go mod tidy`. Expected `require`: **only** `go.uber.org/goleak` (+ any
  indirect of goleak, also test-only). If anything *non-test* third-party appears, that is a
  real boundary violation — stop and remove the offending import; do not pin it.
- `pkg/engine/memory/` stays inside this module.

### A2. Rewrite every engine/memory import path repo-wide
- `github.com/LerianStudio/fetcher/v2/pkg/engine` → `github.com/LerianStudio/fetcher/pkg/engine`
  (and `/memory` likewise) across all 83 files.
- **Precision constraint:** anchor the replacement on `engine"` and `engine/` so it never
  touches `fetcher/v2/pkg/engine`**`compat`** (stays in `/v2`). Concretely two replacements:
  `fetcher/v2/pkg/engine"` → `fetcher/pkg/engine"` and `fetcher/v2/pkg/engine/` → `fetcher/pkg/engine/`.
- Verify afterward: `grep -rn 'fetcher/v2/pkg/engine"' && grep -rn 'fetcher/v2/pkg/engine/'`
  return nothing, and `fetcher/v2/pkg/enginecompat` is untouched.
- The repo legitimately ends up mixing `fetcher/v2/...` (host) and `fetcher/pkg/engine`
  (engine) imports — that is correct.

### A3. Split `dependency_test.go` across the two modules
The file currently lives in package `engine` but covers two unrelated concerns. After the
split, `mustRepositoryRoot` resolves to the **engine** module dir, which breaks the doc guards.

- **Keep in `pkg/engine/dependency_test.go` (engine module):** the dependency-boundary tests
  (`BlocksForbiddenImports`, `AllowlistStdlibAndModuleLocalOnly`, `AllowlistPolicyClassification`,
  `ReadsModulePathFromGoMod`, the tenant/docker/deployment pattern tests, the GOFLAGS/readonly
  tests). Required fixes for the new module identity:
  - `newEngineDependencyGraphCommand`: `./pkg/engine/...` → `./...` (engine pkgs are now at
    module root), and the assertion in `GoListRunsReadonly` likewise.
  - `AllowlistPolicyClassification`: update the hardcoded `const modulePath` to
    `github.com/LerianStudio/fetcher/pkg/engine` and adjust the module-local sample case
    (`/pkg/engine/memory` → `/memory`); the `/v2` and `/v3` lookalike cases stay (still
    correctly classified as non-local).
  - The module-local denylist patterns (`modulePath + "/pkg/rabbitmq"`, `/components/...`,
    etc.) become structurally unreachable (those packages are in the *parent* module now).
    Leave the denylist intact as belt-and-suspenders for third-party families (fiber,
    rabbitmq, mongo, …) — the **allowlist** test is the real guard and trips the instant any
    non-test third-party `require` is added.
- **Move to a new parent-module test (`pkg/buildguard/buildguard_test.go`, package
  `buildguard`):** `TestClaudeGoVersionGuidance_UsesGoModAsSourceOfTruth` and
  `TestActiveGoVersionGuidance_UsesGoModAsSourceOfTruth`, plus the `mustRepositoryRoot` /
  `mustModulePathFromGoMod` helpers they need. In the parent module these resolve the repo
  root correctly and keep guarding repo-root docs.

### A4. Committed `go.work` at repo root
```
go 1.26.4

use .
use ./pkg/engine
```
Enables local/CI multi-module dev (engine resolves from the working tree). Release uses
`GOWORK=off` to prove require-resolution (see B).

### A5. Parent `go.mod`
- Add `require github.com/LerianStudio/fetcher/pkg/engine v1.0.0-beta.1`.
- **Transient only on this feature branch:** add `replace github.com/LerianStudio/fetcher/pkg/engine => ./pkg/engine`
  so `GOWORK=off` builds pass *before* the first engine tag exists. The release prepare hook
  (B) removes this `replace` and pins the `require` in the release commit. The committed
  parent `go.mod` on `main`/`develop` post-release carries **no replace**.

### A6. Docs
- `README.md`: install snippet `go get github.com/LerianStudio/fetcher/v2/pkg/engine` →
  `go get github.com/LerianStudio/fetcher/pkg/engine`; update the two import lines in the
  embedding example (`pkg/engine` and `pkg/engine/memory`).
- New `docs/RELEASING.md`: document the dual-module scheme (path-prefixed engine tags,
  engine-first ordering, the require-pin + GOWORK=off gate, the transient replace bootstrap).

---

## B. Release CI: cut BOTH modules, dependency-ordered, in-repo only

### B1. `.releaserc.yml` changes
- **Reorder plugins** so `@semantic-release/exec` runs **before** `@semantic-release/git`
  (so the exec prepare hook can mutate `go.mod`/`go.sum` *before* the release commit is made).
- Add `go.mod` and `go.sum` to the `@semantic-release/git` `assets` list (so the require-pin
  + replace-removal land in the release commit).
- Set explicit `tagFormat: "v${version}"` for the parent so semantic-release's tag matcher
  **ignores** `pkg/engine/v*` tags (req 4 — path-prefixed engine tags never collide; this is
  structural since they don't match `v${version}`, but pin it explicitly).
- Wire `@semantic-release/exec`:
  - `prepareCmd`: `node scripts/release/engine-release.mjs prepare --parent-version ${nextRelease.version} --channel ${nextRelease.channel}`
  - (no `verifyConditionsCmd` engine cut — engine is cut only when the parent is actually
    releasing, i.e. inside prepare, satisfying "cut both always" *per releasing run*.)

### B2. `scripts/release/engine-release.mjs` (zero npm deps — Node + git only)
`prepare` subcommand, run inside the releasing pipeline:
1. **Compute engine next version.** Find latest `pkg/engine/v*` tag (semver sort); inspect
   commits since it touching `pkg/engine/**`; apply the **same** conventional-commit bump
   rules as `.releaserc.yml` (feat/perf/build/refactor→minor, fix/chore/ci/test/docs→patch,
   breaking→major). Honor the channel: on `beta`/`rc`, produce the next prerelease counter.
   **If no `pkg/engine/**` commits since the last engine tag, still bump** (increment the
   prerelease counter, or patch on stable) — req 1: an unchanged engine still gets a fresh
   tag. First-ever run with no prior `pkg/engine/v*` tag → `v1.0.0-beta.1`.
3. **Engine-first tag.** `git tag pkg/engine/vX.Y.Z` at HEAD and `git push origin` that tag,
   **before** touching the parent commit, so the parent `require` is always resolvable.
4. **Pin parent + drop replace.** Rewrite parent `go.mod`: `require .../pkg/engine vX.Y.Z`
   (exact tag, no `replace`); remove any transient `replace` line.
5. **Refresh go.sum + GOWORK=off gate.** `GOWORK=off go mod download github.com/LerianStudio/fetcher/pkg/engine`
   then `GOWORK=off go build ./...` and `GOWORK=off go test ./...`. A stale/missing engine
   require fails the release here (req 3) instead of being masked by `go.work`.
6. Append an engine changelog entry to `pkg/engine/CHANGELOG.md` (add it to git assets too).
Then `@semantic-release/git` commits (CHANGELOG.md, pkg/engine/CHANGELOG.md, go.mod, go.sum),
and semantic-release core tags the parent `v2.0.0-beta.6` on that commit.

### B3. New in-repo workflow `.github/workflows/engine-module.yml`
The shared `go-pr-analysis` filters to `components/*` only, so the engine module is otherwise
uncovered. This new workflow (allowed — it's a new file in *this* repo) covers req 6 + the
smoke test on PRs to `develop`/`main`:
- `cd pkg/engine && go test ./...` (boundary tests included).
- `cd pkg/engine && golangci-lint run` (engine module).
- **Parent GOWORK=off build gate:** `GOWORK=off go build ./...` (with the transient replace
  pre-tag; via proxy post-tag).
- **Consumer smoke test** (`scripts/release/smoke-consumer.sh`): scratch throwaway module in
  a temp dir importing `github.com/LerianStudio/fetcher/pkg/engine`; `go build`; assert the
  consumer `go.sum` references **only** the engine module path and **zero** third-party
  modules. PR mode uses `replace => <repo>/pkg/engine` (pre-tag); post-tag mode uses
  `go get ...@pkg/engine/vX.Y.Z`. **This is THE proof of the whole change.**

### B4. Makefile
- `test-unit` / `test-all`: also `cd pkg/engine && go test ./...`.
- `lint`: also lint the engine module.
- `tidy`: tidy both modules.
- Add `make test-engine` and `make smoke-engine` (runs the smoke script).

---

## Cut it in THIS beta (held behind explicit go-ahead)
- Engine first tag: `pkg/engine/v1.0.0-beta.1`.
- Parent next: `v2.0.0-beta.6`.
- Parent `go.mod` require pinned to `v1.0.0-beta.1`.
- **Bootstrap order (first cut only):** the feature branch carries the transient `replace`;
  merging to `develop` runs the enhanced pipeline, whose prepare hook pushes the engine tag
  first, removes the replace, pins+tidies under `GOWORK=off`, then the parent tag is cut on
  the release commit. Both tags land in one pipeline run, engine first.
- **Hold point:** I will NOT push any tag without your explicit go-ahead. I'll get everything
  green locally first and present the exact cut commands / pipeline trigger.

---

## Out of scope
- No engine API/behavior changes — pure import-path rewrite + module split + CI.
- No `enginecompat` logic changes beyond the mechanical import-path update.

---

## Verification
1. `cd pkg/engine && go mod tidy` → `require` contains only `go.uber.org/goleak` (test-only);
   non-test build graph stdlib-only.
2. `cd pkg/engine && go test ./...` → green (boundary + allowlist tests pass under new identity).
3. Root via go.work: `go build ./... && go test ./...` (manager + worker still embed engine).
4. `GOWORK=off go build ./...` (with transient replace pre-tag) → green.
5. `scripts/release/smoke-consumer.sh` → consumer `go.sum` has zero third-party modules.
6. `make lint` (both modules) clean.
7. Dry-run the release logic (`node scripts/release/engine-release.mjs prepare --parent-version 2.0.0-beta.6 --channel beta` in a throwaway clone) → produces `pkg/engine/v1.0.0-beta.1`,
   pins the parent require, passes the GOWORK=off gate.
8. Parent module guards (`pkg/buildguard`) still pass against repo-root docs.

## Critical files
- New: `pkg/engine/go.mod`, `go.work`, `pkg/buildguard/buildguard_test.go`,
  `scripts/release/engine-release.mjs`, `scripts/release/smoke-consumer.sh`,
  `.github/workflows/engine-module.yml`, `docs/RELEASING.md`, `pkg/engine/CHANGELOG.md`.
- Modified: `go.mod` (require + transient replace), `.releaserc.yml`, `Makefile`,
  `pkg/engine/dependency_test.go` (split + new-identity fixes), `README.md`, and 83 files
  for the import-path rewrite (mechanical).
