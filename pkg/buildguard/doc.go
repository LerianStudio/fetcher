// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package buildguard holds repo-root build/documentation hygiene guards that
// belong to the parent module (github.com/LerianStudio/fetcher/v2).
//
// These guards used to live in pkg/engine/dependency_test.go, but once pkg/engine
// became its own module the test's repository-root walk resolves to the engine
// module directory, not the repo root — so the guards that assert on root-level
// files (CLAUDE.md, README.md, docs/PROJECT_RULES.md) must run from the parent
// module instead. The engine dependency-boundary tests stay in the engine module.
package buildguard
