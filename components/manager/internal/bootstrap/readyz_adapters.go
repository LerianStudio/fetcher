package bootstrap

import (
	"context"
	"net"

	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	"github.com/LerianStudio/fetcher/pkg/constant"
	pkgRabbitmq "github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/gofiber/fiber/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

// rabbitMQAdapterProbe adapts the fetcher's rabbitmq.Adapter to the
// readyz.RabbitMQAdapterProbe interface without forcing an import cycle or
// exposing the fetcher-specific CircuitState enum on the readyz package.
//
// State mapping is a tight 1:1 with readyz.BreakerState. A defensive default
// collapses any unknown value to BreakerClosed so a transient out-of-range
// read doesn't wrongly mark the dep as open.
type rabbitMQAdapterProbe struct {
	adapter pkgRabbitmq.Adapter
}

// newRabbitMQAdapterProbe wraps the adapter. Returns nil when the adapter is
// nil so bootstrap can pass the result straight to NewRabbitMQAdapterChecker
// — the checker falls back to a safe "not initialized" response.
func newRabbitMQAdapterProbe(adapter pkgRabbitmq.Adapter) *rabbitMQAdapterProbe {
	if adapter == nil {
		return nil
	}

	return &rabbitMQAdapterProbe{adapter: adapter}
}

// State maps the fetcher adapter's circuit-breaker state onto the
// package-local readyz enum.
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

// Ping uses the adapter's IsHealthy snapshot as the liveness probe. The
// snapshot is cheap (a mutex-protected field read) and reflects the channel
// + connection liveness maintained by the adapter's channel watcher. We do
// NOT open a fresh AMQP channel here because the watcher already does so on
// reconnect and doing it again from /readyz would thrash the server's
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

// Sentinel errors used by the probe. They are exported only via the error
// interface; callers should not type-assert.
var (
	errAdapterNil = &adapterError{msg: "rabbitmq adapter not initialized"}
	errNotHealthy = &adapterError{msg: "rabbitmq connection not healthy"}
)

type adapterError struct{ msg string }

func (e *adapterError) Error() string { return e.msg }

// newReadyzRedisClient builds a standalone *redis.Client for the sole
// purpose of serving the /readyz schema-cache probe. The schema cache
// itself embeds Redis under a fallback wrapper (redis.NewCacheWithFallback)
// that hides the live state — probing through the wrapper would always
// succeed (in-memory fallback). A dedicated probe client bypasses the
// fallback so the operator sees the real Redis state.
//
// Returns nil when no Redis host is configured — the caller must omit the
// dep from /readyz in that case.
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

	return redis.NewClient(opts)
}

// redisURLFromCfg composes a redis://host:port URL used ONLY for TLS
// posture detection on the /readyz response. The scheme reflects cfg.RedisTLS.
// We deliberately do not encode the password — only the scheme and host:port
// matter for the readyz.detectRedisTLS helper.
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

// applicationServiceName returns the fetcher application name exactly as it
// must appear on tmclient calls. Kept as a one-liner so the readyz wiring
// does not import the HTTP-layer constant directly.
func applicationServiceName() string { return constant.ApplicationName }

// amqpChannelCloser is implemented by *amqp.Channel. A tiny shim keeps our
// code free of the direct amqp import where not needed.
type amqpChannelCloser interface {
	Close() error
}

// ensureCloseAMQP is a nil-safe close for an AMQP channel. Used in the
// per-tenant RabbitMQ checker path where we obtain a fresh channel, run the
// probe, then close.
func ensureCloseAMQP(ch amqpChannelCloser) error {
	if ch == nil {
		return nil
	}

	return ch.Close()
}

// amqpChannelFactory adapts tmrabbitmq.Manager.GetChannel, which returns
// *amqp.Channel, into the readyz.TenantRabbitMQResolver surface. The
// indirection isolates the amqp import to this adapter file.
type amqpChannelFactory = func(ctx context.Context, tenantID string) (*amqp.Channel, error)

// buildManagerReadyzCheckers assembles the ordered set of DependencyCheckers
// the manager registers on GET /readyz. The membership rules (from Gate 6 of
// ring:dev-readyz):
//
//   - MongoDB: single-tenant -> real MongoClientChecker against the platform
//     *mongo.Client; multi-tenant -> NAChecker pointing to /readyz/tenant/:id.
//   - RabbitMQ: single-tenant -> real RabbitMQAdapterChecker that surfaces
//     breaker state; multi-tenant -> NAChecker.
//   - Redis (schema cache): always global. If REDIS_HOST is unset the dep is
//     skipped rather than omitted.
//   - Multi-tenant Redis (event discovery): only when MT is enabled AND
//     MULTI_TENANT_REDIS_HOST is set.
//   - tenant_manager: only when MT is enabled.
//
// Keeping this pure-data helper makes the wiring testable without spinning
// up a Fiber app.
func buildManagerReadyzCheckers(
	cfg *Config,
	repos *managerRepositories,
	plat *managerPlatformDependencies,
) []readyz.DependencyChecker {
	// Nil-cfg guard: under test seams or misconfigured callers, returning an
	// empty checker set is preferable to panicking on the cfg.* dereferences
	// that follow. Mirrors the nil-tolerance pattern used by
	// newReadyzRedisClient and resolveServerDrainDelay elsewhere in bootstrap.
	if cfg == nil {
		return nil
	}

	checkers := make([]readyz.DependencyChecker, 0, 6)

	// --- MongoDB ---
	if cfg.MultiTenantEnabled {
		// TLS posture for the NAChecker must reflect the URI scheme/query
		// (mongodb+srv is TLS-implicit; tls=true on plain mongodb signals TLS).
		// CA-cert presence alone is a fallback — Atlas-style deployments do not
		// require a CA cert because the system trust store covers Atlas certs.
		// Parse error is non-fatal here: fall back to CA-cert presence.
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

	// --- RabbitMQ ---
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

	// --- Redis (schema cache / primary Redis) ---
	if plat != nil && plat.readyzRedisClient != nil {
		client := plat.readyzRedisClient

		checkers = append(checkers, readyz.NewRedisClientCheckerFromFn(
			"redis",
			func(ctx context.Context) error { return client.Ping(ctx).Err() },
			redisURLFromCfg(cfg.RedisHost, cfg.RedisPort, cfg.RedisTLS),
		))
	}

	// --- Tenant Manager (HTTP client) ---
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

// buildManagerTenantHandler returns the /readyz/tenant/:id Fiber handler. In
// single-tenant mode, or when any MT prerequisite is missing, the handler
// returns 400 with a stable body — the route stays mounted so operators can
// distinguish "MT disabled" from "route not wired".
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
