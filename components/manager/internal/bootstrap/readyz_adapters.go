package bootstrap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"net"

	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	"github.com/LerianStudio/fetcher/pkg/constant"
	pkgRabbitmq "github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// rabbitMQAdapterProbe adapts pkgRabbitmq.Adapter to
// readyz.RabbitMQAdapterProbe without forcing the readyz package to import
// the fetcher-specific CircuitState enum. Unknown values collapse to
// BreakerClosed so a transient out-of-range read does not wrongly mark
// the dep as open.
type rabbitMQAdapterProbe struct {
	adapter pkgRabbitmq.Adapter
}

// newRabbitMQAdapterProbe returns nil for a nil adapter; the checker falls
// back to a safe "not initialized" response.
func newRabbitMQAdapterProbe(adapter pkgRabbitmq.Adapter) *rabbitMQAdapterProbe {
	if adapter == nil {
		return nil
	}

	return &rabbitMQAdapterProbe{adapter: adapter}
}

func (r *rabbitMQAdapterProbe) State() readyz.BreakerState {
	if r == nil || r.adapter == nil {
		return readyz.BreakerClosed
	}

	switch r.adapter.CircuitBreakerState() {
	case pkgRabbitmq.CircuitClosed:
		return readyz.BreakerClosed
	case pkgRabbitmq.CircuitOpen:
		return readyz.BreakerOpen
	case pkgRabbitmq.CircuitHalfOpen:
		return readyz.BreakerHalfOpen
	default:
		return readyz.BreakerClosed
	}
}

// Ping reads the adapter's IsHealthy snapshot — a cheap mutex-protected
// field that reflects channel + connection liveness from the watcher. A
// fresh AMQP channel is intentionally not opened here: the watcher already
// reconnects, and probing on every /readyz would thrash the server's
// channel limit on a misconfigured pod.
func (r *rabbitMQAdapterProbe) Ping(ctx context.Context) error {
	if r == nil || r.adapter == nil {
		return errAdapterNil
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if !r.adapter.IsHealthy() {
		return errNotHealthy
	}

	return nil
}

var (
	errAdapterNil = &adapterError{msg: "rabbitmq adapter not initialized"}
	errNotHealthy = &adapterError{msg: "rabbitmq connection not healthy"}
)

type adapterError struct{ msg string }

func (e *adapterError) Error() string { return e.msg }

// newReadyzRedisClient builds a standalone *redis.Client for /readyz so
// the probe bypasses the schema cache's in-memory fallback wrapper, which
// would otherwise always report success even when Redis is down. Returns
// nil when REDIS_HOST is unset; the caller must omit the dep in that case.
//
// When cfg.RedisTLS is true, opts.TLSConfig is populated with TLS 1.2 as
// the floor and the optional base64-encoded REDIS_CA_CERT bundle. This
// mirrors the multi-tenant Redis client in initManagerEventDiscovery —
// without it, the readyz Ping speaks plaintext on a TLS-only Redis and
// reports redis=down even when Redis is healthy. CA-cert decode failures
// here intentionally fall back to a system-trust-store TLS config: the
// bootstrap caller (initPlatformDependencies) does not propagate errors
// from the readyz client, so erroring out would crash the manager. The
// downstream Ping will still surface a TLS handshake failure as redis=down
// if the system trust store cannot validate the broker.
func newReadyzRedisClient(cfg *Config) *redis.Client {
	if cfg == nil || cfg.RedisHost == "" {
		return nil
	}

	port := cfg.RedisPort
	if port == "" {
		port = "6379"
	}

	opts := &redis.Options{
		Addr:     net.JoinHostPort(cfg.RedisHost, port),
		Password: cfg.RedisPassword,
		DB:       getRedisDB(cfg.RedisDB),
	}

	if cfg.RedisTLS {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}

		if cfg.RedisCACert != "" {
			if caCert, err := base64.StdEncoding.DecodeString(cfg.RedisCACert); err == nil {
				pool := x509.NewCertPool()
				if pool.AppendCertsFromPEM(caCert) {
					tlsCfg.RootCAs = pool
				}
			}
		}

		opts.TLSConfig = tlsCfg
	}

	return redis.NewClient(opts)
}

// redisURLFromCfg composes a URL used only for TLS posture detection.
// Password and DB are intentionally omitted — only scheme and host:port
// matter to detectRedisTLS.
func redisURLFromCfg(host, port string, useTLS bool) string {
	if host == "" {
		return ""
	}

	return readyz.ComposeRedisURL(host, port, useTLS)
}

// multiTenantRedisURLFromCfg is the MT-Redis variant — same as
// redisURLFromCfg but reads the MT-specific fields.
func multiTenantRedisURLFromCfg(cfg *Config) string {
	if cfg == nil {
		return ""
	}

	return readyz.ComposeRedisURL(cfg.MultiTenantRedisHost, cfg.MultiTenantRedisPort, cfg.MultiTenantRedisTLS)
}

// newReadyzMultiTenantRedisClient builds a standalone *redis.Client for the
// /readyz multi_tenant_redis dep so the probe is not coupled to the event
// listener's Redis client lifecycle (Start/Stop). Returns nil when MT is
// disabled or MULTI_TENANT_REDIS_HOST is unset; the caller must omit the
// dep in that case.
//
// Mirrors initManagerEventDiscovery's TLS branch (config.go around the
// MultiTenantRedisTLS block): TLS 1.2 floor, optional base64-encoded
// MULTI_TENANT_REDIS_CA_CERT bundle. CA-cert decode failures fall back to
// the system trust store with no error — bootstrap callers do not propagate
// errors from the readyz client and erroring out would crash the manager.
// A real TLS handshake failure surfaces as multi_tenant_redis=down via the
// downstream Ping.
func newReadyzMultiTenantRedisClient(cfg *Config) *redis.Client {
	if cfg == nil || !cfg.MultiTenantEnabled || cfg.MultiTenantRedisHost == "" {
		return nil
	}

	port := cfg.MultiTenantRedisPort
	if port == "" {
		port = "6379"
	}

	opts := &redis.Options{
		Addr:     net.JoinHostPort(cfg.MultiTenantRedisHost, port),
		Password: cfg.MultiTenantRedisPassword,
	}

	if cfg.MultiTenantRedisTLS {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}

		if cfg.MultiTenantRedisCACert != "" {
			if caCert, err := base64.StdEncoding.DecodeString(cfg.MultiTenantRedisCACert); err == nil {
				pool := x509.NewCertPool()
				if pool.AppendCertsFromPEM(caCert) {
					tlsCfg.RootCAs = pool
				}
			}
		}

		opts.TLSConfig = tlsCfg
	}

	return redis.NewClient(opts)
}

func applicationServiceName() string { return constant.ApplicationName }

// buildManagerReadyzCheckers assembles the DependencyCheckers registered on
// GET /readyz. Membership:
//
//   - MongoDB: single-tenant → MongoClientChecker; multi-tenant → NAChecker.
//   - RabbitMQ: single-tenant → RabbitMQAdapterChecker (surfaces breaker
//     state); multi-tenant → NAChecker.
//   - Redis (schema cache): always emitted; skipped when REDIS_HOST unset.
//   - Multi-tenant Redis (event discovery): only with MT enabled and a host.
//   - tenant_manager: only with MT enabled.
func buildManagerReadyzCheckers(
	cfg *Config,
	repos *managerRepositories,
	plat *managerPlatformDependencies,
) []readyz.DependencyChecker {
	if cfg == nil {
		return nil
	}

	checkers := make([]readyz.DependencyChecker, 0, 6)

	if cfg.MultiTenantEnabled {
		// CA-cert presence is a fallback for non-Atlas deployments; Atlas
		// uses the system trust store and signals TLS via the URI scheme
		// or tls=true query parameter.
		mongoTLS, _ := readyz.DetectMongoTLS(buildMongoSource(cfg))
		checkers = append(checkers, readyz.NewNAChecker(
			"mongodb",
			"multi-tenant: see /readyz/tenant/:id",
			readyz.TLSPtr(mongoTLS || cfg.MongoTLSCACert != ""),
		))
	} else if repos != nil && repos.mongoClient != nil {
		checkers = append(checkers, readyz.NewMongoClientChecker(
			repos.mongoClient, buildMongoSource(cfg),
		))
	}

	if cfg.MultiTenantEnabled {
		checkers = append(checkers, readyz.NewNAChecker(
			"rabbitmq",
			"multi-tenant: see /readyz/tenant/:id",
			readyz.TLSPtr(cfg.RabbitMQTLS),
		))
	} else if plat != nil && plat.rabbitMQAdapter != nil {
		checkers = append(checkers, readyz.NewRabbitMQAdapterChecker(
			newRabbitMQAdapterProbe(plat.rabbitMQAdapter),
			buildRabbitMQSource(cfg),
		))
	}

	if plat != nil && plat.readyzRedisClient != nil {
		client := plat.readyzRedisClient

		checkers = append(checkers, readyz.NewRedisClientCheckerFromFn(
			"redis",
			func(ctx context.Context) error { return client.Ping(ctx).Err() },
			redisURLFromCfg(cfg.RedisHost, cfg.RedisPort, cfg.RedisTLS),
		))
	}

	// Multi-tenant event-discovery Redis: surfaces tenant Pub/Sub Redis
	// health on the global /readyz so an MT deployment with a dead MT-Redis
	// is visible as multi_tenant_redis=down (the schema-cache redis dep is
	// a different host, so a healthy schema-cache cannot mask MT-Redis).
	if cfg.MultiTenantEnabled && plat != nil && plat.multiTenantReadyzRedisClient != nil {
		mtClient := plat.multiTenantReadyzRedisClient

		checkers = append(checkers, readyz.NewRedisClientCheckerFromFn(
			"multi_tenant_redis",
			func(ctx context.Context) error { return mtClient.Ping(ctx).Err() },
			multiTenantRedisURLFromCfg(cfg),
		))
	}

	if cfg.MultiTenantEnabled && plat != nil {
		checkers = append(checkers, readyz.NewTenantManagerClientChecker(
			plat.tmClient,
			applicationServiceName(),
			cfg.MultiTenantURL,
			true,
		))
	}

	return checkers
}

// buildManagerTenantHandler returns the /readyz/tenant/:id handler.
// Outside MT mode, or when any MT prerequisite is missing, it serves 400
// with a stable body so operators can distinguish "MT disabled" from
// "route not wired".
func buildManagerTenantHandler(
	readyzCfg *readyz.Config,
	cfg *Config,
	plat *managerPlatformDependencies,
) fiber.Handler {
	if !cfg.MultiTenantEnabled || plat == nil || plat.tmClient == nil ||
		plat.tmMongoManager == nil || plat.tmRabbitMQManager == nil {
		return readyz.NewDisabledTenantHandler()
	}

	th := readyz.NewTenantHandler(
		readyzCfg,
		plat.tmClient,
		applicationServiceName(),
		readyz.NewTenantMongoChecker(plat.tmMongoManager),
		readyz.NewTenantRabbitMQChecker(plat.tmRabbitMQManager),
	)

	return th.Fiber()
}
