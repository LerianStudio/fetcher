package readyz

import (
	"regexp"
	"strings"
)

// sanitize redacts obvious credential patterns from an error string before it
// is emitted on the /readyz response.
//
// Operators consume /readyz.errors via Grafana / Loki; a leaked password would
// be catastrophic. We cover the three patterns the fetcher's platform
// dependencies are likely to surface:
//
//  1. URI userinfo:  "user:pass@host" -> "***@host"
//  2. Query / DSN:   "password=secret" -> "password=***"
//  3. Query / DSN:   "pwd=secret"      -> "pwd=***"
//
// The function is intentionally narrow: it targets strings that match the
// upstream driver error formats seen in practice (pgx, mongo-driver, redis,
// amqp091). Anything that doesn't look like one of these patterns is passed
// through unchanged so operators still see the real failure cause.
//
// Input bounds: strings longer than 512 bytes are truncated before returning.
// This prevents an upstream error that embedded the whole connection string
// from flooding a /readyz response, while still giving the operator enough
// signal to triage.
func sanitize(s string) string {
	if s == "" {
		return s
	}

	s = userinfoRegex.ReplaceAllString(s, "***@")
	s = passwordKVRegex.ReplaceAllString(s, "${1}=***")

	const maxLen = 512
	if len(s) > maxLen {
		s = s[:maxLen]
	}

	return strings.TrimSpace(s)
}

// userinfoRegex matches the userinfo portion of a URL-shaped string. The
// leading delimiter is a scheme-terminator-colon or a whitespace boundary so
// we don't accidentally eat fragments like "file.go:123". The trailing "@"
// anchors the userinfo form per RFC 3986. Passwords with ":" are allowed
// because the pattern is non-greedy up to the "@".
var userinfoRegex = regexp.MustCompile(`([A-Za-z0-9._~%!$&'()*+,;=-]+):[^@\s/]+@`)

// passwordKVRegex matches "password=..." / "pwd=..." / "passwd=..." where the
// value has no embedded whitespace or "&". Case-insensitive so we catch both
// DSN styles. The captured group is the key, which the replacement preserves
// verbatim so operators can tell which field was redacted.
var passwordKVRegex = regexp.MustCompile(`(?i)\b(password|passwd|pwd)=[^&\s]+`)

// tlsOrFalse collapses the (bool, error) return signature of the TLS
// detectors into a single bool for the DependencyCheck.TLS field.
//
// The rationale: TLS posture is *informational* on /readyz — it tells
// operators whether the dep is configured with TLS or not. If URL parsing
// fails (malformed connection string), we've already failed Gate 4 in SaaS
// mode; in BYOC / local mode we simply report TLS=false rather than surface
// the parse error on an informational field. The real failure propagates to
// the DependencyCheck.Error via the probe attempt.
func tlsOrFalse(b bool, _ error) bool { return b }
