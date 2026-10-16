package bootstrap

import (
	"strings"
	"testing"

	fetcher "github.com/LerianStudio/fetcher/v2"
	"github.com/LerianStudio/lib-auth/v5/auth/declaration"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// seedGrants is what the access-manager seed (caradhras
// deploy/pam-seed/init_data.json) grants each fetcher group today, restricted to
// the pairs that have a live route. Publishing the manifest replaces the seeded
// RBAC, so every group must keep at least these pairs.
var seedGrants = map[string][]string{
	"admin-group":               {"connections:get", "connections:post", "connections:patch", "connections:delete", "fetcher:get", "fetcher:post"},
	"fetcher-editor-group":      {"connections:get", "connections:post", "connections:patch", "connections:delete", "fetcher:get", "fetcher:post"},
	"fetcher-contributor-group": {"connections:get", "connections:post", "connections:patch", "fetcher:get", "fetcher:post"},
	"fetcher-viewer-group":      {"connections:get", "fetcher:get"},
}

func TestPermissionManifestKeepsSeedGrants(t *testing.T) {
	var m declaration.DeclarationManifest
	require.NoError(t, yaml.Unmarshal(fetcher.PermissionManifest, &m))
	require.NoError(t, m.Validate())

	roleGroups := map[string][]string{}
	for _, r := range m.Roles {
		for _, g := range r.GrantedTo {
			require.True(t, strings.HasSuffix(g.Group, "-group"),
				"role %q binds %q: without the -group suffix the reconciler creates a new, empty group", r.Name, g.Group)
			roleGroups[r.Name] = append(roleGroups[r.Name], g.Group)
		}
	}

	granted := map[string]map[string]bool{}
	for _, p := range m.Permissions {
		for _, role := range p.Roles {
			for _, g := range roleGroups[role] {
				if granted[g] == nil {
					granted[g] = map[string]bool{}
				}
				granted[g][p.Resource+":"+p.Action] = true
			}
		}
	}

	for group, pairs := range seedGrants {
		for _, pair := range pairs {
			require.True(t, granted[group][pair], "group %q loses %q that the seed grants today", group, pair)
		}
	}
}
