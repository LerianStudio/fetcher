package readyz

import (
	"regexp"
	"strings"
)

// sanitize redacts credential patterns from probe error strings before they
// reach /readyz responses (and from there, Loki / Grafana). It only targets
// the three forms seen in practice from pgx, mongo-driver, redis, and
// amqp091 — strings that do not match are passed through unchanged so the
// real failure cause stays visible:
//
//  1. URI userinfo:  "user:pass@host" -> "***@host"
//  2. Query / DSN:   "password=secret" -> "password=***"
//  3. Query / DSN:   "pwd=secret"      -> "pwd=***"
//
// Output is capped at 512 bytes to prevent an error that embedded the full
// connection string from flooding the response.
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

// userinfoRegex matches the userinfo portion of a URL-shaped string per
// RFC 3986. The character class on the user side excludes "/" and ":" runs
// so it does not eat fragments like "file.go:123"; the non-greedy match up
// to "@" tolerates passwords containing ":".
var userinfoRegex = regexp.MustCompile(`([A-Za-z0-9._~%!$&'()*+,;=-]+):[^@\s/]+@`)

// passwordKVRegex matches "password=..." / "pwd=..." / "passwd=...". The
// captured key is preserved in the replacement so operators can tell which
// field was redacted.
var passwordKVRegex = regexp.MustCompile(`(?i)\b(password|passwd|pwd)=[^&\s]+`)

// tlsOrFalse drops the error from a TLS detector's (bool, error) result.
// TLS posture on /readyz is informational — a parse error is surfaced via
// the probe's Error field, and SaaS-mode TLS enforcement runs separately,
// so reporting TLS=false on parse failure is acceptable.
func tlsOrFalse(b bool, _ error) bool { return b }
