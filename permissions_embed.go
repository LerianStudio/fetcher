// Package fetcher is the repository-root anchor for the embedded access-manager
// permission-declaration manifest, and holds nothing else.
//
// It lives at the root because it has to: //go:embed resolves relative to the
// source file that carries the directive and rejects parent-directory
// references, so the only package that can embed the repo-root permissions.yaml
// is a package AT the repo root. The manifest stays at the root because that is
// where the platform's Responsibility-Inversion tooling looks for it.
package fetcher

import (
	// embed is imported for its side effect only: it enables the //go:embed
	// directive below, which compiles permissions.yaml into the binary.
	_ "embed"
)

// PermissionManifest is the byte content of the repo-root permissions.yaml — the
// authorization surface of the "fetcher" service slug, published to the access
// manager by the lib-auth declaration publisher when the Manager boots (see
// components/manager/internal/bootstrap/permission_declaration.go).
//
// The bytes are embedded rather than read from disk at run time so the running
// container declares the exact manifest its binary was built from: a file read
// would let an image with a stale, edited or missing permissions.yaml publish a
// surface the repository never reviewed.
//
//go:embed permissions.yaml
var PermissionManifest []byte
