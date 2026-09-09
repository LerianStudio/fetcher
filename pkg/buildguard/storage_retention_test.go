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

// retentionPromisePatterns match the shapes in which a surface promises that
// stored extraction results expire on their own.
var retentionPromisePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)configurable\s+ttl`),
	regexp.MustCompile(`OBJECT_STORAGE_TTL`),
	regexp.MustCompile(`\bFileTTL\b`),
	regexp.MustCompile(`(?i)store\([^)]*\bttl\b`),
	regexp.MustCompile(`(?i)files?\s+will\s+expire`),
}

// retentionClaimSurfaces are the operator-facing surfaces that must not promise
// an expiry the product does not apply: the two documents a customer reads, the
// Worker's configuration template, and the Worker's own configuration types.
var retentionClaimSurfaces = []string{
	"README.md",
	filepath.Join("docs", "PROJECT_RULES.md"),
	filepath.Join("components", "worker", ".env.example"),
	filepath.Join("components", "worker", "internal", "bootstrap", "config.go"),
	filepath.Join("components", "worker", "internal", "services", "service.go"),
}

// TestNoRetentionPromiseWithoutAnExpiryMechanism keeps the retention story
// honest. The object-storage port (pkg/ports/storage.Repository) takes no expiry
// argument and no provider applies one, so extracted results are kept until the
// bucket's own lifecycle policy removes them. While that is true, no surface may
// advertise a TTL: an operator who believes a knob expires customer data and is
// wrong gets unbounded retention of data pulled out of their databases.
//
// The guard retires itself: the day the storage port carries an expiry, the
// promise becomes legal and this test steps aside.
func TestNoRetentionPromiseWithoutAnExpiryMechanism(t *testing.T) {
	t.Parallel()

	repoRoot := mustRepositoryRoot(t)

	portPath := filepath.Join(repoRoot, "pkg", "ports", "storage", "repository.go")

	port, err := os.ReadFile(portPath)
	if err != nil {
		t.Fatalf("read object storage port: %v", err)
	}

	portSpelling := strings.ToLower(string(port))
	if strings.Contains(portSpelling, "ttl") || strings.Contains(portSpelling, "expir") {
		t.Skip("object storage port now carries an expiry; retention promises are backed by a mechanism")
	}

	for _, surface := range retentionClaimSurfaces {
		surface := surface
		t.Run(surface, func(t *testing.T) {
			t.Parallel()

			contentBytes, err := os.ReadFile(filepath.Join(repoRoot, surface))
			if err != nil {
				t.Fatalf("read retention claim surface %s: %v", surface, err)
			}

			content := string(contentBytes)
			for _, promise := range retentionPromisePatterns {
				if match := promise.FindString(content); match != "" {
					t.Fatalf(
						"%s promises result expiry (%q) but the object storage port applies none; "+
							"describe retention as a bucket lifecycle policy instead",
						surface, match,
					)
				}
			}
		})
	}
}
