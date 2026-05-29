package readyz

import (
	"fmt"
	"net/url"
	"strings"
)

// URL-based TLS detection helpers. Every detector takes a raw connection
// string and returns (bool, error):
//
//	empty input  -> (false, nil)   // dep not configured; not an error
//	parse error  -> (false, err)
//	TLS detected -> (true,  nil)
//	otherwise    -> (false, nil)
//
// Implementation must use net/url, not substring matching, because
// substring checks are wrong for URL-encoded query parameters and DSNs that
// embed hints in comments. The detectors inspect the configured posture
// only — they never touch live sockets, log, panic, or os.Exit.

// truthyQueryValue accepts the loose ("true", "1") truthy set used by
// Mongo-style TLS parameters. Anything else, including "", counts as off.
func truthyQueryValue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1":
		return true
	default:
		return false
	}
}

// falseyQueryValue is the explicit "off" counterpart to truthyQueryValue.
// Empty string is intentionally NOT falsey — empty means "param unset"
// and the caller falls through to scheme-default rules. Mongo driver spec
// allows ?tls=false (or ?ssl=false) to override the implicit TLS that
// mongodb+srv enables; this helper detects that override.
func falseyQueryValue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "0":
		return true
	default:
		return false
	}
}

// parseOrWrap names the failing dependency in the parse error so the
// operator-visible error can identify which connection string was bad.
func parseOrWrap(dep, raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("readyz: parse %s connection string: %w", dep, err)
	}

	return u, nil
}

// DetectMongoTLS:
//   - Explicit "tls=false" or "ssl=false" → false, regardless of scheme.
//     The MongoDB driver spec allows operators to opt out of the implicit
//     TLS that mongodb+srv enables; the readyz posture must reflect that.
//   - Explicit "tls=true" / "ssl=true" → true (covers the redundant case
//     on srv schemes and the only way to enable TLS on plain mongodb).
//   - "mongodb+srv" with no explicit tls/ssl override → true (driver
//     upgrades automatically per the SRV spec).
//   - "mongodb" with no explicit override → false.
//   - "tlsCACert" alone is not TLS — the driver still requires tls=true.
//
// Exported because the multi-tenant carve-out at the bootstrap layer needs
// to surface TLS posture even when no probe runs.
func DetectMongoTLS(uri string) (bool, error) {
	if uri == "" {
		return false, nil
	}

	u, err := parseOrWrap("mongodb", uri)
	if err != nil {
		return false, err
	}

	q := u.Query()
	tlsParam := q.Get("tls")
	sslParam := q.Get("ssl")

	// Explicit overrides win over scheme defaults.
	if falseyQueryValue(tlsParam) || falseyQueryValue(sslParam) {
		return false, nil
	}

	if truthyQueryValue(tlsParam) || truthyQueryValue(sslParam) {
		return true, nil
	}

	// Scheme default: mongodb+srv implies TLS per the SRV spec.
	if strings.EqualFold(u.Scheme, "mongodb+srv") {
		return true, nil
	}

	return false, nil
}

func detectMongoTLS(uri string) (bool, error) { return DetectMongoTLS(uri) }

// detectRedisTLS: scheme "rediss" → true.
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

// detectAMQPTLS: scheme "amqps" → true.
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

// detectS3TLS: empty endpoint → true (AWS default endpoints are HTTPS;
// S3 is never truly unconfigured here). Custom endpoint → scheme "https".
func detectS3TLS(endpoint string) (bool, error) {
	if endpoint == "" {
		return true, nil
	}

	u, err := parseOrWrap("s3", endpoint)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(u.Scheme, "https"), nil
}

// detectHTTPUpstreamTLS: scheme "https" → true.
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

// detectPostgresTLS: absent/empty sslmode → false (we force operators to be
// explicit instead of relying on libpq's silent sslmode=prefer fallback);
// "disable" → false; any other value → true.
//
//nolint:unused // included for library completeness; not currently wired.
func detectPostgresTLS(dsn string) (bool, error) {
	if dsn == "" {
		return false, nil
	}

	u, err := parseOrWrap("postgres", dsn)
	if err != nil {
		return false, err
	}

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
