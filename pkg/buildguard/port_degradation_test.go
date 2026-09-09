// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package buildguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConnectionStoreDegradationRowIsNotCRUDOnly keeps the published port table
// honest about the optional ConnectionStore. An embedder reads that table to
// decide which ports to implement, and pkg/engine gates far more than connection
// CRUD on the store: extraction planning, extraction execution, schema
// discovery, schema validation and connection testing all fail without it
// (pinned by TestEngineWithoutConnectionStore_DegradationSurface in the engine
// module). A row that mentions only CRUD sends an embedder into an engine that
// constructs and then fails on nearly every call.
func TestConnectionStoreDegradationRowIsNotCRUDOnly(t *testing.T) {
	t.Parallel()

	readmePath := filepath.Join(mustRepositoryRoot(t), "README.md")

	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	row := ""

	for _, line := range strings.Split(string(readme), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "| `ConnectionStore` |") {
			row = line

			break
		}
	}

	if row == "" {
		t.Fatal("README.md must keep a port-table row for `ConnectionStore`")
	}

	spelling := strings.ToLower(row)
	for _, gatedConcern := range []string{"extraction", "schema"} {
		if !strings.Contains(spelling, gatedConcern) {
			t.Fatalf(
				"README.md ConnectionStore row omits %s, which the engine also gates on the store: %s",
				gatedConcern, strings.TrimSpace(row),
			)
		}
	}
}
