package multitenant_test

import (
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/multitenant"
	"github.com/LerianStudio/fetcher/v2/pkg/resolver"
	"github.com/stretchr/testify/assert"
)

// dashedTenantID and dashlessTenantID are the two spellings of ONE tenant UUID.
// Access Manager tokens carry the dashless form, so that is the spelling every
// derived key holds today; the dashed form is what a caller reading a tenant ID
// back out of lib-commons through a different seam can end up with.
const (
	dashedTenantID   = "9b2f4c1e-3d5a-4f8b-9c7d-1e2f3a4b5c6d"
	dashlessTenantID = "9b2f4c1e3d5a4f8b9c7d1e2f3a4b5c6d"
)

func TestCanonicalCollapsesBothUUIDSpellings(t *testing.T) {
	t.Parallel()

	assert.Equal(t, dashlessTenantID, multitenant.Canonical(dashedTenantID))
	assert.Equal(t, dashlessTenantID, multitenant.Canonical(dashlessTenantID),
		"canonicalizing the form real tokens already carry must be a no-op")
	assert.Equal(t, multitenant.Canonical(dashedTenantID), multitenant.Canonical(dashlessTenantID))
}

func TestCanonicalPreservesNonUUIDTenants(t *testing.T) {
	t.Parallel()

	// Fetcher's single-tenant sentinel and slug-style tenants have no canonical
	// UUID form and must survive verbatim.
	assert.Equal(t, "single-tenant", multitenant.Canonical("single-tenant"))
	assert.Equal(t, "tenant-123-abc", multitenant.Canonical("tenant-123-abc"))

	// An ID lib-commons rejects outright is returned unchanged rather than
	// blanked — collapsing it to "" would turn a malformed tenant into a wildcard.
	assert.Equal(t, "", multitenant.Canonical(""))
	assert.Equal(t, "not/a/tenant", multitenant.Canonical("not/a/tenant"))
}

// TestDeterministicConnectionIDIsSpellingIndependent pins the consequence that
// matters most: the internal-datasource UUID is an identity, so two spellings of
// one tenant deriving two different UUIDs would mean a caller holding the dashed
// form resolves nothing.
func TestDeterministicConnectionIDIsSpellingIndependent(t *testing.T) {
	t.Parallel()

	registry := resolver.NewInternalDatasourceRegistry()

	for _, configName := range registry.ListInternal() {
		t.Run(configName, func(t *testing.T) {
			t.Parallel()

			dashed := registry.GetDeterministicID(dashedTenantID, configName)
			dashless := registry.GetDeterministicID(dashlessTenantID, configName)

			assert.Equal(t, dashless, dashed, "deterministic connection id must not depend on UUID spelling")

			// The reverse lookup must agree with the forward derivation for both
			// spellings, or a connection resolves one way and not the other.
			nameFromDashed, _, okDashed := registry.FindConfigByID(dashed, dashedTenantID)
			assert.True(t, okDashed)
			assert.Equal(t, configName, nameFromDashed)

			nameFromDashless, _, okDashless := registry.FindConfigByID(dashed, dashlessTenantID)
			assert.True(t, okDashless, "an id derived from the dashed spelling must resolve under the dashless one")
			assert.Equal(t, configName, nameFromDashless)
		})
	}

	// Different tenants must still diverge — canonicalization must not flatten
	// tenant scoping.
	other := registry.GetDeterministicID("11111111111111111111111111111111", "plugin_crm")
	assert.NotEqual(t, registry.GetDeterministicID(dashlessTenantID, "plugin_crm"), other)
}
