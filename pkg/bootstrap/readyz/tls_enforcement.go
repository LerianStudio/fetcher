package readyz

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// This file implements Gate 4 of the ring:dev-readyz contract for the Lerian
// Fetcher service: a single, centralized SaaS TLS enforcement function that
// MUST run at bootstrap BEFORE any platform connection is opened. It refuses
// to start the service when DEPLOYMENT_MODE=saas and any configured
// dependency lacks TLS.
//
// Design notes:
//
//   - The function is pure: inputs come through SaaSTLSConfig, outputs are
//     (nil | error). No globals, no env reads inside ValidateSaaSTLS. The
//     caller (manager/worker bootstrap) is responsible for composing
//     SaaSTLSConfig from its own Config struct.
//
//   - TLS detection is delegated to the Gate 3 detectXxxTLS helpers that live
//     alongside in tls_detection.go. Substring matching is FORBIDDEN by the
//     ring:dev-readyz contract (anti-pattern #4) — everything goes through
//     net/url.
//
//   - Inline DEPLOYMENT_MODE=="saas" checks scattered across bootstrap code
//     are FORBIDDEN by the contract (anti-pattern #6). This file is the ONLY
//     place where that check may appear.
//
//   - The dep-check order is documented and STABLE so operators get a
//     predictable first-failure message: mongodb, rabbitmq, redis,
//     multi_tenant_redis, s3, tenant_manager. Tests rely on this order.
//
//   - The function never panics, never calls os.Exit, never log.Fatal. Errors
//     are returned and wrapped with %w so the caller can use errors.Is to
//     discriminate the SaaS-required sentinel from a malformed-URL parse
//     error.

// SaaSTLSConfig carries the connection posture required to enforce SaaS TLS.
//
// Fetcher-specific notes:
//
//   - PostgreSQL / MySQL / Oracle / SQL Server are intentionally omitted:
//     those are user-owned data sources (Fetcher reads from them on behalf
//     of a tenant), not platform dependencies. Their TLS posture is the
//     tenant's responsibility, not the platform's.
//
//   - Platform dependencies that ARE checked: MongoDB (metadata store),
//     Redis (schema cache), Multi-Tenant Redis (tenant discovery), RabbitMQ
//     (job queue + publisher), S3 (object storage, worker-only), Tenant
//     Manager HTTP client (multi-tenant only).
//
//   - The caller is expected to compose the RedisURL / MultiTenantRedisURL /
//     RabbitMQURL strings from their host/port/user/pass/tls fields before
//     invoking ValidateSaaSTLS. ComposeRedisURL / ComposeRabbitMQURL are
//     provided in this file as the canonical helpers for that.
type SaaSTLSConfig struct {
	// DeploymentMode is the raw DEPLOYMENT_MODE value ("saas" / "byoc" /
	// "local" / ""). Only "saas" (case-insensitive) triggers enforcement.
	DeploymentMode string

	// MongoURI is the MongoDB connection URI used by the metadata store.
	MongoURI string

	// RedisURL is the primary Redis URL (schema cache). Compose it from the
	// REDIS_* envs via ComposeRedisURL.
	RedisURL string

	// MultiTenantRedisURL is the Redis URL used for tenant event-driven
	// discovery. Compose it from MULTI_TENANT_REDIS_* via ComposeRedisURL.
	MultiTenantRedisURL string

	// RabbitMQURL is the RabbitMQ AMQP URL. Compose it via
	// ComposeRabbitMQURL.
	RabbitMQURL string

	// S3Endpoint is OBJECT_STORAGE_ENDPOINT. Empty means "use AWS default
	// endpoint" which is always HTTPS.
	S3Endpoint string

	// TenantManagerURL is MULTI_TENANT_URL. Only meaningful when
	// MultiTenantEnabled == true.
	TenantManagerURL string

	// MultiTenantEnabled gates the MultiTenantRedisURL + TenantManagerURL
	// checks. When false, those deps are not in use and are skipped.
	MultiTenantEnabled bool

	// HasS3 is true for the worker (which uses S3) and false for the
	// manager (which does not). When false, the S3 check is skipped.
	HasS3 bool

	// AllowInsecureHTTPTM mirrors MULTI_TENANT_ALLOW_INSECURE_HTTP. When
	// true, the Tenant Manager TLS check is skipped. This is an operator
	// escape hatch for local / BYOC stubs exposed over plaintext HTTP
	// behind a trusted network; in SaaS mode, setting this flag is a
	// deliberate (logged and audited) decision by the operator.
	AllowInsecureHTTPTM bool
}

// ErrSaaSTLSRequired is the sentinel returned (wrapped with %w) when
// DEPLOYMENT_MODE=saas and a configured dependency lacks TLS. Callers can
// use errors.Is to discriminate it from other bootstrap errors such as
// malformed-URL parse failures.
var ErrSaaSTLSRequired = errors.New("SaaS mode requires TLS on all platform dependencies")

// tlsCheck binds a human-readable dep name to its detection function and the
// URL captured from SaaSTLSConfig. The skip flag lets us express the
// multi-tenant / HasS3 / AllowInsecureHTTPTM carve-outs declaratively rather
// than with scattered if-continue statements inside the main loop.
type tlsCheck struct {
	name   string
	url    string
	detect func(string) (bool, error)
	skip   bool
}

// ValidateSaaSTLS enforces the SaaS TLS requirement at bootstrap. It MUST be
// invoked before any platform connection is opened.
//
// Behaviour:
//
//   - DeploymentMode != "saas" (case-insensitive, or empty) → returns nil
//     without performing any checks.
//
//   - DeploymentMode == "saas":
//
//   - Each configured (non-empty URL) dependency is TLS-checked via the
//     Gate 3 url.Parse-based helpers.
//
//   - An empty URL for a dep means "dep not configured" — it is skipped
//     silently. A misconfigured SaaS service will fail to connect later
//     anyway, and that diagnosis belongs to Gate 7 (self-probe), not here.
//
//   - A non-TLS dep returns an error wrapping ErrSaaSTLSRequired with the
//     dep name. Use errors.Is(err, ErrSaaSTLSRequired) to detect.
//
//   - A parse error on a configured URL returns a wrapped parse error
//     naming the dep. It does NOT wrap ErrSaaSTLSRequired — a malformed
//     URL is a configuration bug, not a TLS posture issue.
//
// The check order is STABLE and documented: mongodb, rabbitmq, redis,
// multi_tenant_redis, s3, tenant_manager. The first failing dep is returned
// so operators fix configuration one error at a time.
//
// This function is the ONLY permitted location for DEPLOYMENT_MODE=="saas"
// comparisons in the codebase. Scattering inline checks at connection sites
// is ring:dev-readyz anti-pattern #6 and is rejected at Gate 0.
//
// Gate 4 of ring:dev-readyz.
func ValidateSaaSTLS(cfg SaaSTLSConfig) error {
	if !strings.EqualFold(strings.TrimSpace(cfg.DeploymentMode), DeploymentModeSaaS) {
		return nil
	}

	checks := []tlsCheck{
		{
			name:   "mongodb",
			url:    cfg.MongoURI,
			detect: detectMongoTLS,
		},
		{
			name:   "rabbitmq",
			url:    cfg.RabbitMQURL,
			detect: detectAMQPTLS,
		},
		{
			name:   "redis",
			url:    cfg.RedisURL,
			detect: detectRedisTLS,
		},
		{
			name:   "multi_tenant_redis",
			url:    cfg.MultiTenantRedisURL,
			detect: detectRedisTLS,
			// Multi-tenant Redis is only in use when the MT feature is on.
			// When MT is off, this dep is irrelevant, no matter what URL
			// was set via MULTI_TENANT_REDIS_HOST.
			skip: !cfg.MultiTenantEnabled,
		},
		{
			name:   "s3",
			url:    cfg.S3Endpoint,
			detect: detectS3TLS,
			// Only the worker uses S3. For the manager (HasS3=false), the
			// S3 endpoint is meaningless even when present.
			skip: !cfg.HasS3,
		},
		{
			name:   "tenant_manager",
			url:    cfg.TenantManagerURL,
			detect: detectHTTPUpstreamTLS,
			// Tenant Manager is only used in MT mode, and operators may
			// opt out via AllowInsecureHTTPTM. The escape hatch exists for
			// stubs / private-network BYOC-style deployments that still
			// set DEPLOYMENT_MODE=saas for consistency.
			skip: !cfg.MultiTenantEnabled || cfg.AllowInsecureHTTPTM,
		},
	}

	for _, c := range checks {
		if c.skip {
			continue
		}

		// detectS3TLS is the single helper whose empty-input contract means
		// "AWS default (TLS on)" rather than "not configured". Every other
		// detector returns (false, nil) for empty — which we interpret here
		// as "skip, dep not in use". Funnel both behaviours through the
		// detector; the empty-string contract is the detector's concern.
		if c.url == "" && c.name != "s3" {
			continue
		}

		tlsOn, err := c.detect(c.url)
		if err != nil {
			// Parse error is a misconfiguration, not a TLS posture issue.
			// Return it directly (with dep context) so callers can
			// distinguish via errors.Is.
			return fmt.Errorf("readyz: validate %s TLS: %w", c.name, err)
		}

		if !tlsOn {
			return fmt.Errorf("%w: %s is not using TLS (DEPLOYMENT_MODE=saas)",
				ErrSaaSTLSRequired, c.name)
		}
	}

	return nil
}

// ComposeRedisURL returns a canonical "redis://host[:port]" or
// "rediss://host[:port]" string given the three building blocks Lerian
// services store individually in env vars.
//
// Returns "" when host is empty: the caller treats that as "dep not
// configured" and ValidateSaaSTLS skips it.
//
// The scheme is selected by the tls flag — true yields "rediss", false
// yields "redis". Host/port are joined via net.JoinHostPort so IPv6 literals
// are bracketed correctly.
func ComposeRedisURL(host, port string, tls bool) string {
	if host == "" {
		return ""
	}

	scheme := "redis"
	if tls {
		scheme = "rediss"
	}

	hostPort := host
	if port != "" {
		hostPort = net.JoinHostPort(host, port)
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   hostPort,
	}

	return u.String()
}

// ComposeRabbitMQURL returns a canonical "amqp://user:pass@host[:port]" or
// "amqps://user:pass@host[:port]" string. When user and pass are both empty
// the URL is emitted without userinfo.
//
// Returns "" when host is empty.
//
// Credentials are URL-escaped via url.UserPassword so special characters (@
// : / etc.) survive composition intact. Host/port are joined via
// net.JoinHostPort.
func ComposeRabbitMQURL(host, port, user, pass string, tls bool) string {
	if host == "" {
		return ""
	}

	scheme := "amqp"
	if tls {
		scheme = "amqps"
	}

	hostPort := host
	if port != "" {
		hostPort = net.JoinHostPort(host, port)
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   hostPort,
	}

	if user != "" || pass != "" {
		u.User = url.UserPassword(user, pass)
	}

	return u.String()
}
