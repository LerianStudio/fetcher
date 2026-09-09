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

var (
	// hkdfContextValue matches the non-secret HKDF context strings in
	// pkg/crypto/key_deriver.go — one per key the master key derives.
	hkdfContextValue = regexp.MustCompile(`"fetcher-[a-z-]+-v[0-9]+"`)

	// derivedKeyCountClaim captures the number word in the README's security
	// statement about how many keys the master key derives.
	derivedKeyCountClaim = regexp.MustCompile(`derive\s+(\w+)\s+cryptographically independent keys`)

	numberWords = map[int]string{1: "one", 2: "two", 3: "three", 4: "four", 5: "five", 6: "six"}
)

// TestDerivedKeyTableCoversEveryHKDFContext keeps the security section's account
// of the master key complete. APP_ENC_KEY derives one key per HKDF context, and
// a reader uses that table to know what the single secret protects — so a
// context missing from the table is a part of the blast radius the operator
// cannot see. The table shipped with three rows while the code derived four: the
// fourth encrypts every extracted result at rest.
func TestDerivedKeyTableCoversEveryHKDFContext(t *testing.T) {
	t.Parallel()

	repoRoot := mustRepositoryRoot(t)

	deriver, err := os.ReadFile(filepath.Join(repoRoot, "pkg", "crypto", "key_deriver.go"))
	if err != nil {
		t.Fatalf("read key deriver: %v", err)
	}

	contexts := hkdfContextValue.FindAllString(string(deriver), -1)
	if len(contexts) == 0 {
		t.Fatal("found no HKDF context strings in pkg/crypto/key_deriver.go")
	}

	readmeBytes, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	readme := string(readmeBytes)

	rows := derivedKeyTableRows(readme)
	if rows != len(contexts) {
		t.Fatalf(
			"README.md derived-key table lists %d keys but the master key derives %d (contexts: %s)",
			rows, len(contexts), strings.Join(contexts, ", "),
		)
	}

	claim := derivedKeyCountClaim.FindStringSubmatch(readme)
	if claim == nil {
		t.Fatal("README.md must state how many keys APP_ENC_KEY derives")
	}

	if want := numberWords[len(contexts)]; !strings.EqualFold(claim[1], want) {
		t.Fatalf("README.md says the master key derives %q keys, want %q", claim[1], want)
	}
}

// derivedKeyTableRows counts the body rows of the README's "Derived Key" table.
func derivedKeyTableRows(readme string) int {
	rows := 0
	inTable := false

	for _, line := range strings.Split(readme, "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "| Derived Key |"):
			inTable = true
		case !inTable:
		case strings.HasPrefix(trimmed, "|---") || strings.HasPrefix(trimmed, "|--"):
		case strings.HasPrefix(trimmed, "|"):
			rows++
		default:
			return rows
		}
	}

	return rows
}
