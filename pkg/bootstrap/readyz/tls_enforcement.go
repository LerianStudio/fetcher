package readyz

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// SaaSTLSConfig carries the connection posture used by ValidateSaaSTLS.
//
// User-owned data sources (PostgreSQL / MySQL / Oracle / SQL Server) are
// intentionally absent: their TLS posture is the tenant's responsibility,
// not the platform's. Platform deps that are checked: MongoDB (metadata
// store), Redis (schema cache), Multi-Tenant Redis (tenant discovery),
// RabbitMQ (job queue + publisher), S3 (worker-only), Tenant Manager
// HTTP client (multi-tenant only).
//
// Callers compose RedisURL / MultiTenantRedisURL / RabbitMQURL via the
// ComposeRedisURL / ComposeRabbitMQURL helpers below.
type SaaSTLSConfig struct {
	// DeploymentMode triggers enforcement only when "saas".
	DeploymentMode string

	MongoURI string

	RedisURL string

	MultiTenantRedisURL string

	RabbitMQURL string

	// S3Endpoint empty means "AWS default" (always HTTPS).
	S3Endpoint string

	// TenantManagerURL is meaningful only when MultiTenantEnabled.
	TenantManagerURL string

	// MultiTenantEnabled skips MultiTenantRedis and TenantManager checks
	// when false.
	MultiTenantEnabled bool

	// HasS3 distinguishes worker (true) from manager (false).
	HasS3 bool

	// AllowInsecureHTTPTM is an operator escape hatch (mirrors
	// MULTI_TENANT_ALLOW_INSECURE_HTTP) for plaintext HTTP TM stubs behind
	// a trusted network. Skips the Tenant Manager TLS check.
	AllowInsecureHTTPTM bool
}

// ErrSaaSTLSRequired wraps the bootstrap error returned when SaaS mode
// finds a non-TLS platform dependency. Callers use errors.Is to
// discriminate it from malformed-URL parse failures.
var ErrSaaSTLSRequired = errors.New("SaaS mode requires TLS on all platform dependencies")

type tlsCheck struct {
	name   string
	url    string
	detect func(string) (bool, error)
	skip   bool
}

// ValidateSaaSTLS is the single, centralized SaaS TLS gate; it must run
// before any platform connection is opened. Inline DEPLOYMENT_MODE checks
// at connection sites are not permitted — this is the only location for
// the "saas" comparison.
//
// Behavior:
//   - non-"saas" DeploymentMode → nil (no checks).
//   - empty URL → dep treated as not configured and skipped (S3 is the
//     exception: empty endpoint means "AWS default", which is HTTPS).
//   - parse error → wrapped error naming the dep (not ErrSaaSTLSRequired —
//     malformed URL is a config bug, not a TLS posture issue).
//   - non-TLS dep → wrapped ErrSaaSTLSRequired.
//
// The check order is fixed (mongodb, rabbitmq, redis, multi_tenant_redis,
// s3, tenant_manager) so the first failure is predictable for operators
// and tests.
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
			skip:   !cfg.MultiTenantEnabled,
		},
		{
			name:   "s3",
			url:    cfg.S3Endpoint,
			detect: detectS3TLS,
			skip:   !cfg.HasS3,
		},
		{
			name:   "tenant_manager",
			url:    cfg.TenantManagerURL,
			detect: detectHTTPUpstreamTLS,
			skip:   !cfg.MultiTenantEnabled || cfg.AllowInsecureHTTPTM,
		},
	}

	for _, c := range checks {
		if c.skip {
			continue
		}

		// S3 is the exception: empty endpoint means AWS default (HTTPS).
		// Other detectors return (false, nil) for empty, which we treat
		// as "dep not in use".
		if c.url == "" && c.name != "s3" {
			continue
		}

		tlsOn, err := c.detect(c.url)
		if err != nil {
			return fmt.Errorf("readyz: validate %s TLS: %w", c.name, err)
		}

		if !tlsOn {
			return fmt.Errorf("%w: %s is not using TLS (DEPLOYMENT_MODE=saas)",
				ErrSaaSTLSRequired, c.name)
		}
	}

	return nil
}

// ComposeRedisURL emits "redis://" or "rediss://" depending on tls. Empty
// host yields "" so ValidateSaaSTLS treats the dep as not configured.
// IPv6 literals are bracketed via net.JoinHostPort.
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

// ComposeRabbitMQURL emits "amqp://" or "amqps://", omits userinfo when
// user and pass are both empty, and URL-escapes credentials so "@", ":",
// and "/" survive intact. Empty host yields "".
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
