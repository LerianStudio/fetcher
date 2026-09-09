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

// enterpriseFraming matches the word used to sell a paid tier, in any casing and
// in the compound forms it usually arrives in ("Enterprise-Grade", "enterprise
// security").
var enterpriseFraming = regexp.MustCompile(`(?i)\benterprise`)

// productStatusSurfaces are the two documents a reader meets first on GitHub.
var productStatusSurfaces = []string{"README.md", "CLAUDE.md"}

// TestSourceAvailableProductIsNotFramedAsEnterprise keeps Fetcher's own front
// page consistent with its licence. Fetcher ships under the Elastic License 2.0
// with SPDX headers across the tree: it is one of only two source-available
// Lerian products, and Enterprise framing on it reads as a paid tier that does
// not exist. Describe what the product does instead.
func TestSourceAvailableProductIsNotFramedAsEnterprise(t *testing.T) {
	t.Parallel()

	repoRoot := mustRepositoryRoot(t)

	license, err := os.ReadFile(filepath.Join(repoRoot, "LICENSE"))
	if err != nil {
		t.Fatalf("read LICENSE: %v", err)
	}

	if !strings.Contains(string(license), "Elastic License 2.0") {
		t.Skip("LICENSE is no longer the Elastic License 2.0; the source-available framing rule no longer applies here")
	}

	for _, surface := range productStatusSurfaces {
		surface := surface
		t.Run(surface, func(t *testing.T) {
			t.Parallel()

			contentBytes, err := os.ReadFile(filepath.Join(repoRoot, surface))
			if err != nil {
				t.Fatalf("read product status surface %s: %v", surface, err)
			}

			if match := enterpriseFraming.FindString(string(contentBytes)); match != "" {
				t.Fatalf(
					"%s frames a source-available (Elastic License 2.0) product as %q; "+
						"describe the capability instead of the tier",
					surface, match,
				)
			}
		})
	}
}
