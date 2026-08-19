// Package multitenant holds fetcher's tenant-identity helpers.
package multitenant

import (
	tmcore "github.com/LerianStudio/lib-commons/v6/commons/tenant-manager/core"
)

// Canonical normalizes a tenant ID to the one spelling fetcher keys everything
// by: a UUID tenant becomes dashless lowercase hex, and a non-UUID tenant slug
// (including fetcher's single-tenant sentinel) is returned verbatim.
//
// # Why this is safe today
//
// Access Manager tokens already carry the dashless form — the Tenant Manager
// mints tenant IDs into the Casdoor application's token attributes with the
// dashes stripped — so on every real token this is a no-op and no existing key,
// header or derived identifier changes value.
//
// # What it defends against
//
// lib-commons is asymmetric about the spelling. Its tenant middleware
// canonicalizes what it puts in the context, while values obtained through the
// Tenant Manager client are used as returned. Anything that compares a
// context-derived tenant ID against a client-derived one, or keys a map by one
// and looks it up by the other, is correct only for as long as both sides happen
// to agree. The two spellings of one UUID are not equal as strings, so the
// failure mode is not an error: a tenant simply looks like a different tenant.
// Depending on the call site that is a rejected message, a duplicate consumer,
// or a datasource UUID that no longer resolves.
//
// Normalizing at fetcher's own boundaries removes the dependency on that
// coincidence.
//
// An ID that lib-commons rejects outright — empty, over-length, or carrying
// characters outside the permitted set — is returned unchanged rather than
// blanked. Callers here are comparing and keying, not validating, and silently
// collapsing an invalid ID to the empty string would turn a malformed tenant
// into a wildcard.
func Canonical(tenantID string) string {
	canonical, err := tmcore.CanonicalTenantID(tenantID)
	if err != nil {
		return tenantID
	}

	return canonical
}
