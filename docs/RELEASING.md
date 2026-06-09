# Releasing Fetcher (dual-module)

Fetcher ships **two Go modules from one repository**, released together in lockstep on
every release event but versioned independently:

| Module | Import path | Tag format | Version line |
|--------|-------------|------------|--------------|
| Parent (services) | `github.com/LerianStudio/fetcher/v2` | `v${version}` (e.g. `v2.0.0-beta.6`) | `v2.x` |
| Engine (embedded runtime) | `github.com/LerianStudio/fetcher/pkg/engine` | `pkg/engine/v${version}` (e.g. `pkg/engine/v1.0.0-beta.1`) | `v1.x` |

The engine numbers and the parent numbers **will never match** (engine `v1`, parent `v2`).
That is correct, not a problem. The engine path has **no `/v2`**: a submodule's fetchable
path is repo-root + subdir, and it carries its own major-version line.

## Why two modules

`pkg/engine` is infrastructure-free and carries a **zero third-party** dependency footprint
(verified by `scripts/release/smoke-consumer.sh` and the empty `require` block in
`pkg/engine/go.mod`). Keeping it inside the parent `/v2` module would force every embedding
consumer (e.g. the midaz reporter) to inherit Fetcher's entire dependency graph (lib-commons,
mongo-driver, Fiber, DB drivers, AWS SDK) via MVS. A separate module makes the **module**
boundary match the already-enforced **package** boundary.

## Local / CI development: `go.work`

A committed `go.work` at the repo root wires both modules for local and in-repo CI work, so
the host resolves the engine from the working tree:

```
go 1.26.4
use .
use ./pkg/engine
```

Most `go` commands respect the workspace automatically. Release resolution is verified with
`GOWORK=off` (below) so a stale/missing engine `require` can't be masked by the workspace.

## The release run (engine-first, one event, two tags)

`release.yml` calls the shared org workflow, which runs semantic-release using `.releaserc.yml`.
semantic-release owns the **parent** line. The **engine** is cut in the same run by the
`@semantic-release/exec` **prepare** hook (`scripts/release/engine-release.mjs prepare`), which
is listed **before** `@semantic-release/git` so it runs first:

1. **Compute** the engine's next version from commits touching `pkg/engine/**` since the last
   `pkg/engine/v*` tag, using the same conventional-commit rules as the parent
   (`feat/perf/build/refactor`→minor, `fix/chore/ci/test/docs`→patch, breaking→major). On a
   prerelease channel (`beta`/`rc`) the version follows semantic-release semantics: a
   **same-channel prerelease series ticks the counter** (`…-beta.3` → `…-beta.4`) regardless of
   bump type — the base only advances at the stable cut or when opening a prerelease from a
   stable tag. An unchanged engine still gets a fresh counter tick — the accepted **no-op bump**
   price of zero drift. First-ever release starts the line at `1.0.0-beta.1`.
2. **Engine-first tag**: `git tag pkg/engine/vX.Y.Z` and push it **before** the parent commit, so
   a `go get` of the parent can never observe an unresolvable engine `require`.
3. **Pin the parent**: rewrite the root `go.mod` `require github.com/LerianStudio/fetcher/pkg/engine`
   to the exact engine tag and **drop the transient `replace`** (see Bootstrap).
4. **`GOWORK=off` gate**: `go mod download` the engine at its tag, then `GOWORK=off go build ./...`
   and `GOWORK=off go test ./...`. This proves the parent resolves the engine via the proxy/VCS,
   not via the workspace. A stale require fails the release here. (Set `ENGINE_RELEASE_SKIP_TESTS=true`
   to skip the heavy test run if it already ran earlier in the pipeline.)
5. Prepend an entry to `pkg/engine/CHANGELOG.md`.

`@semantic-release/git` then commits `CHANGELOG.md`, `go.mod`, `go.sum`, and
`pkg/engine/CHANGELOG.md` as the release commit, and semantic-release tags the parent
`v${version}` on it. The engine tag and the parent tag sit on adjacent commits; the
`pkg/engine/` subtree is byte-identical across them.

No `replace` is ever committed on `main`/`develop` after a release.

> **Private modules:** the parent `GOWORK=off` build fetches private `github.com/LerianStudio/*`
> deps and therefore requires the org token (`secrets: inherit` in the shared workflow / `GOPRIVATE`).
> The standalone `engine-module.yml` PR workflow intentionally does **not** build the full parent —
> the engine is dependency-free, so its jobs need no token; the parent resolution gate lives in
> this release hook.

## Tag-matching: no collision

The parent `tagFormat` is `v${version}`. Path-prefixed engine tags (`pkg/engine/v*`) do not
match that glob, so semantic-release never mistakes an engine tag for a parent release.

## Bootstrap (first engine release only)

Before the first `pkg/engine/v*` tag exists, the parent `go.mod` carries a **transient**
`replace github.com/LerianStudio/fetcher/pkg/engine => ./pkg/engine` (clearly commented) so
`GOWORK=off` builds resolve the engine locally. The release prepare hook removes it and pins
the real tag in the release commit. A `replace` is acceptable **only** on a feature branch
before the first tag.

## CI coverage

- `.github/workflows/engine-module.yml` — engine module build+test (incl. the dependency
  boundary + leak guard), engine lint, the consumer **zero-third-party smoke test**, the
  release version-logic unit tests, and an assertion that `pkg/engine/go.mod` has an empty
  `require` block.
- `make test-engine`, `make smoke-engine`, `make tidy`, `make lint`, `make format` all cover
  both modules.

## Verifying locally

```bash
# engine module, in isolation
cd pkg/engine && GOWORK=off go test ./...        # or: make test-engine

# zero-third-party proof + release version logic
make smoke-engine

# parent resolves the engine (workspace + release modes)
go build ./...
GOWORK=off go build ./...                        # pre-tag: via the transient replace

# dry-run the engine version the next release would cut
node scripts/release/engine-release.mjs compute --channel beta
```
