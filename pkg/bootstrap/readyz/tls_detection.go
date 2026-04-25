package readyz

import (
	"fmt"
	"net/url"
	"strings"
)

// This file implements URL-based TLS detection helpers for the /readyz
// machinery. Every detector takes a raw connection string (URI/DSN/URL) and
// returns (bool, error). They follow the same behavioural contract enforced by
// the ring:dev-readyz skill:
//
//   (1) Empty input  -> (false, nil)   — dependency not configured; not an error.
//   (2) Parse error  -> (false, err)   — caller must surface the operator-visible failure.
//   (3) TLS detected -> (true,  nil)
//   (4) Otherwise    -> (false, nil)
//
// Implementation rules (HARD GATE from the skill):
//
//   * Use net/url — never strings.Contains on the raw string. Substring
//     matching is demonstrably wrong for URL-encoded query params and for DSNs
//     that embed hints inside comments or note= values.
//   * Never reflect on live connection objects. These helpers operate on the
//     configured posture, not the negotiated state of an open socket.
//   * Never log, panic, or os.Exit. Errors travel back to the caller.
//
// Scheme and query-parameter comparisons are case-insensitive where the
// underlying protocol is case-insensitive (URL schemes per RFC 3986 §3.1; the
// truthy literal "true"/"TRUE" in query values).
//
// The functions are package-scoped (unexported) because Gate 4
// (ValidateSaaSTLS) and Gate 6 (real checkers) both live inside
// package readyz and call them directly. Keeping them unexported avoids
// leaking an internal contract that future work may tighten.

// truthyQueryValue reports whether a Mongo-style TLS query-parameter value
// means "TLS on". It matches the loose truthy set ("true", "1") in a
// case-insensitive fashion. Explicit "false", "0", "" and any other value
// counts as not-TLS.
func truthyQueryValue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1":
		return true
	default:
		return false
	}
}

// parseOrWrap parses a URL-ish string and wraps any parse error with a named
// context so the caller (Gate 4 ValidateSaaSTLS, operators reading /readyz
// error fields) can tell which dependency failed.
func parseOrWrap(dep, raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("readyz: parse %s connection string: %w", dep, err)
	}

	return u, nil
}

// DetectMongoTLS returns true when the Mongo connection URI indicates TLS.
//
// Rules (from the ring:dev-readyz contract):
//   - Scheme "mongodb+srv" is TLS-implicit (the driver upgrades automatically).
//   - For "mongodb", the query parameter "tls" OR "ssl" set to a truthy value
//     ("true"/"1", case-insensitive) signals TLS.
//   - "tlsCACert" alone is informational; the driver still requires an
//     explicit tls=true to enable TLS. We mirror that behaviour here.
//
// Returns (false, nil) when the URI is empty (dep not configured). Returns
// (false, err) on parse failure.
//
// Exported for use by Gate 6 NAChecker construction at the bootstrap layer
// (components/{manager,worker}/internal/bootstrap/readyz_adapters.go), where
// the multi-tenant carve-out needs to surface accurate TLS posture even when
// no probe runs. CA-cert presence alone is NOT TLS — operators on
// mongodb+srv (Atlas) need URI-derived detection.
func DetectMongoTLS(uri string) (bool, error) {
	if uri == "" {
		return false, nil
	}

	u, err := parseOrWrap("mongodb", uri)
	if err != nil {
		return false, err
	}

	if strings.EqualFold(u.Scheme, "mongodb+srv") {
		return true, nil
	}

	q := u.Query()
	if truthyQueryValue(q.Get("tls")) || truthyQueryValue(q.Get("ssl")) {
		return true, nil
	}

	return false, nil
}

// detectMongoTLS is the unexported alias retained for in-package callers
// (tls_enforcement.go, checker_mongo.go, tls_detection_test.go). Behaviour
// is identical to DetectMongoTLS.
func detectMongoTLS(uri string) (bool, error) { return DetectMongoTLS(uri) }

// detectRedisTLS returns true when the Redis URL's scheme is "rediss". A
// plain "redis" scheme returns false.
//
// Returns (false, nil) for an empty URL (dep not configured). Returns
// (false, err) on parse failure.
func detectRedisTLS(rawURL string) (bool, error) {
	if rawURL == "" {
		return false, nil
	}

	u, err := parseOrWrap("redis", rawURL)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(u.Scheme, "rediss"), nil
}

// detectAMQPTLS returns true when the RabbitMQ URL's scheme is "amqps".
// A plain "amqp" scheme returns false.
//
// Returns (false, nil) for an empty URL (dep not configured). Returns
// (false, err) on parse failure.
func detectAMQPTLS(rawURL string) (bool, error) {
	if rawURL == "" {
		return false, nil
	}

	u, err := parseOrWrap("rabbitmq", rawURL)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(u.Scheme, "amqps"), nil
}

// detectS3TLS returns true when the S3 endpoint is TLS-capable.
//
// Rules:
//   - Empty endpoint means "use the AWS default endpoint", which is always
//     HTTPS — so empty returns (true, nil). This is the one helper whose
//     empty case means TLS=on rather than "dep not configured"; S3 is never
//     truly unconfigured in a Fetcher deployment, there is always either a
//     custom endpoint or the implicit AWS default.
//   - A custom endpoint MUST be explicit: "https" returns true, "http"
//     returns false.
//
// Returns (false, err) on parse failure.
func detectS3TLS(endpoint string) (bool, error) {
	if endpoint == "" {
		// AWS default endpoints are HTTPS.
		return true, nil
	}

	u, err := parseOrWrap("s3", endpoint)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(u.Scheme, "https"), nil
}

// detectHTTPUpstreamTLS returns true when the upstream base URL uses the
// "https" scheme. Used for Tenant Manager and any other HTTP dependency.
//
// Returns (false, nil) for an empty URL (dep not configured). Returns
// (false, err) on parse failure.
func detectHTTPUpstreamTLS(baseURL string) (bool, error) {
	if baseURL == "" {
		return false, nil
	}

	u, err := parseOrWrap("http_upstream", baseURL)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(u.Scheme, "https"), nil
}

// detectPostgresTLS returns true when the Postgres DSN's sslmode is set to
// anything other than "disable" (or absent).
//
// Rules:
//   - Absent or empty sslmode -> false (Fetcher treats "not declared" as
//     "not TLS" to force operators to be explicit; libpq's default of
//     sslmode=prefer is a terrible silent fallback).
//   - sslmode=disable -> false.
//   - Any other value ("require", "verify-full", "verify-ca", "prefer",
//     "allow") -> true. Case-insensitive.
//
// Included for library completeness. Fetcher itself does not currently use
// Postgres as a platform dependency (internal data is MongoDB), but sibling
// Lerian services do.
//
// Returns (false, nil) for an empty DSN. Returns (false, err) on parse
// failure.
//
//nolint:unused // included for library completeness; Gate 4 may wire it in.
func detectPostgresTLS(dsn string) (bool, error) {
	if dsn == "" {
		return false, nil
	}

	u, err := parseOrWrap("postgres", dsn)
	if err != nil {
		return false, err
	}

	// net/url preserves original key casing, so iterate to find sslmode
	// regardless of how the operator wrote it.
	var sslmode string

	for k, v := range u.Query() {
		if strings.EqualFold(k, "sslmode") {
			if len(v) > 0 {
				sslmode = v[0]
			}

			break
		}
	}

	sslmode = strings.ToLower(strings.TrimSpace(sslmode))
	if sslmode == "" || sslmode == "disable" {
		return false, nil
	}

	return true, nil
}
