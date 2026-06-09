#!/usr/bin/env bash
# Copyright (c) 2026 Lerian Studio. All rights reserved.
# SPDX-License-Identifier: Elastic-2.0
#
# smoke-consumer.sh — THE proof of the module extraction.
#
# Builds a throwaway module that imports github.com/LerianStudio/fetcher/pkg/engine
# and asserts the consumer's module graph carries ZERO third-party dependencies — only
# the engine module itself. If the engine ever grows a non-test third-party require, or
# module-graph pruning fails to drop its test-only deps (e.g. go.uber.org/goleak), this
# script fails.
#
# Modes:
#   --local           Resolve the engine via a local replace to this repo's pkg/engine.
#                     Use in PR CI, before any pkg/engine/v* tag exists.
#   --version <vX>    Resolve the engine from the module proxy/VCS at semver <vX>
#                     (maps to tag pkg/engine/<vX>). Use after a release.
#
# Exit non-zero on any third-party leak.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENGINE_MODULE="github.com/LerianStudio/fetcher/pkg/engine"

MODE="local"
VERSION=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --local) MODE="local"; shift ;;
    --version) MODE="version"; VERSION="${2:?--version needs a value}"; shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
cd "$WORKDIR"

echo "[smoke] scratch consumer in $WORKDIR (mode=$MODE)"

# A consumer module deliberately OUTSIDE the fetcher repo and workspace.
export GOWORK=off
go mod init example.com/engine-smoke >/dev/null

cat > main.go <<EOF
package main

import (
	"fmt"

	"${ENGINE_MODULE}"
	"${ENGINE_MODULE}/memory"
)

func main() {
	// Touch both packages so the linker keeps them in the build.
	_ = memory.NewConnectorRegistry
	fmt.Println(engine.DefaultLimits())
}
EOF

if [[ "$MODE" == "local" ]]; then
  go mod edit -require="${ENGINE_MODULE}@v0.0.0-00010101000000-000000000000"
  go mod edit -replace="${ENGINE_MODULE}=${REPO_ROOT}/pkg/engine"
  go mod tidy
else
  go get "${ENGINE_MODULE}@${VERSION}"
  go mod tidy
fi

echo "[smoke] building consumer..."
go build ./...

echo "[smoke] consumer module graph (go list -m all):"
go list -m all | sed 's/^/    /'

# Assertion: every module in the graph is either the consumer itself, the Go toolchain
# pseudo-module, or the engine module. Anything else is a third-party leak.
LEAK="$(go list -m -f '{{.Path}}' all \
  | grep -v '^example.com/engine-smoke$' \
  | grep -v '^go$' \
  | grep -v "^${ENGINE_MODULE}\$" || true)"

if [[ -n "$LEAK" ]]; then
  echo "[smoke] FAIL — third-party dependencies leaked into the consumer:" >&2
  echo "$LEAK" | sed 's/^/    /' >&2
  exit 1
fi

echo "[smoke] PASS — consumer imports ${ENGINE_MODULE} with ZERO third-party deps."
