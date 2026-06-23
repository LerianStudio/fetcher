// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package buildguard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// staleGoVersionPattern matches any literal Go/toolchain version guidance, so the
// active docs can't reintroduce a hardcoded version that drifts from go.mod.
var staleGoVersionPattern = regexp.MustCompile(`(?im)go\s+(version|toolchain|minimum|required|recommended)[^\n]*\b1\.[0-9]+(?:\.[0-9]+)?\b|\bgo\s*1\.[0-9]+(?:\.[0-9]+)?\b`)

func TestClaudeGoVersionGuidance_UsesGoModAsSourceOfTruth(t *testing.T) {
	t.Parallel()

	claudePath := filepath.Join(mustRepositoryRoot(t), "CLAUDE.md")
	claude, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}

	content := string(claude)
	if !strings.Contains(content, "Go version:** Source of truth is `go.mod`") {
		t.Fatalf("CLAUDE.md must direct agents to use go.mod as the Go version source of truth")
	}

	if match := staleGoVersionPattern.FindString(content); match != "" {
		t.Fatalf("CLAUDE.md must not reintroduce stale literal Go/toolchain guidance; found %q", match)
	}
}

func TestActiveGoVersionGuidance_UsesGoModAsSourceOfTruth(t *testing.T) {
	t.Parallel()

	repoRoot := mustRepositoryRoot(t)
	activeDocs := []string{"CLAUDE.md", "README.md", filepath.Join("docs", "PROJECT_RULES.md")}

	for _, activeDoc := range activeDocs {
		activeDoc := activeDoc
		t.Run(activeDoc, func(t *testing.T) {
			t.Parallel()

			contentBytes, err := os.ReadFile(filepath.Join(repoRoot, activeDoc))
			if err != nil {
				t.Fatalf("read active Go guidance %s: %v", activeDoc, err)
			}

			content := string(contentBytes)
			if !strings.Contains(content, "go.mod") {
				t.Fatalf("%s must defer Go toolchain guidance to go.mod", activeDoc)
			}

			if match := staleGoVersionPattern.FindString(content); match != "" {
				t.Fatalf("%s must not contain stale literal Go/toolchain guidance; found %q", activeDoc, match)
			}
		})
	}
}

// mustRepositoryRoot walks up from the test working directory to the first
// directory containing a go.mod. From the parent module's pkg/buildguard package
// this resolves to the repository root (the parent module's go.mod), which is
// where the guarded docs live.
func mustRepositoryRoot(t *testing.T) string {
	t.Helper()

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	for current := workingDir; ; current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		if parent := filepath.Dir(current); parent == current {
			t.Fatalf("could not locate repository root from %q", workingDir)
		}
	}
}
