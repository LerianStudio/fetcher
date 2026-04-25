package bootstrap

import (
	"context"
	"net"

	workerRabbitAdapters "github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	"github.com/LerianStudio/fetcher/pkg/constant"
	pkgRabbitmq "github.com/LerianStudio/fetcher/pkg/rabbitmq"
	portStorage "github.com/LerianStudio/fetcher/pkg/ports/storage"
	pkgStorage "github.com/LerianStudio/fetcher/pkg/storage"
	libMongo "github.com/LerianStudio/lib-commons/v4/commons/mongo"
	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	tmmongo "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/mongo"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/rabbitmq"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// workerReadyzDeps bundles the concrete handles the worker bootstrap passes
// to NewHealthServer for /readyz wiring. Fields are all nullable — the
// checker-building logic handles missing deps by either skipping or carving
// out the corresponding dep entry.
//
// This indirection keeps NewHealthServer's signature small and lets the
// single-tenant and multi-tenant construction paths share the same wiring
// code.
type workerReadyzDeps struct {
	cfg            *Config
	mongoClient    *libMongo.Client
	rabbitAdapter  pkgRabbitmq.Adapter
	s3Client       *s3.Client
	mtRedisClient  *redis.Client
	tmClient       readyz.TMClient
	tmMongoManager *tmmongo.Manager
	tmRabbitMgr    *tmrabbitmq.Manager

	// closers are cleanup hooks attached to deps the readyz_adapters layer
	// owns (e.g. a dedicated redis client). The top-level Service is
	// responsible for invoking these on shutdown.
	closers []func() error
}

// newWorkerReadyzDepsST constructs readyz deps for the single-tenant path.
// The tenant-scoped fields remain nil — the /readyz/tenant/:id handler will
// fall back to the disabled variant.
func newWorkerReadyzDepsST(
	cfg *Config,
	mongoClient *libMongo.Client,
	storage portStorage.Repository,
	consumer *workerRabbitAdapters.ConsumerRoutes,
) *workerReadyzDeps {
	deps := &workerReadyzDeps{cfg: cfg, mongoClient: mongoClient}

	if consumer != nil {
		deps.rabbitAdapter = consumer.Adapter()
	}

	if s3Repo, ok := storage.(*pkgStorage.S3Repository); ok {
		deps.s3Client = s3Repo.Client()
	}

	return deps
}

// newWorkerReadyzDepsMT constructs readyz deps for the multi-tenant path.
// The single-tenant rabbit adapter is nil (the global /readyz will register
// an NAChecker); tenant-scoped probes go through the tmrabbitmq.Manager.
//
// We also create an independent *redis.Client for the multi-tenant-Redis
// probe because the event listener's redis client is constructed inside
// initMultiTenantStack and is cleaned up there — sharing it would tangle
// the two lifecycles.
func newWorkerReadyzDepsMT(
	cfg *Config,
	mongoClient *libMongo.Client,
	storage portStorage.Repository,
	tmClient *tmclient.Client,
	tmMongo *tmmongo.Manager,
	tmRabbit *tmrabbitmq.Manager,
) *workerReadyzDeps {
	deps := &workerReadyzDeps{
		cfg:            cfg,
		mongoClient:    mongoClient,
		tmClient:       tmClient,
		tmMongoManager: tmMongo,
		tmRabbitMgr:    tmRabbit,
	}

	if s3Repo, ok := storage.(*pkgStorage.S3Repository); ok {
		deps.s3Client = s3Repo.Client()
	}

	rdb := newReadyzMTRedis(cfg)
	if rdb != nil {
		deps.mtRedisClient = rdb
		deps.closers = append(deps.closers, func() error { return rdb.Close() })
	}

	return deps
}

// close invokes every registered closer in order. Kept as a method so
// callers can assign it to Service.readyzCloser without writing another
// wrapper. Returns nothing because we want the shutdown path to keep going
// on individual cleanup errors.
func (d *workerReadyzDeps) close() {
	if d == nil {
		return
	}

	for _, fn := range d.closers {
		if fn != nil {
			_ = fn()
		}
	}
}

// newReadyzMTRedis builds the standalone multi-tenant-Redis client used
// exclusively by /readyz. Returns nil when MT Redis is not configured.
func newReadyzMTRedis(cfg *Config) *redis.Client {
	if cfg == nil || cfg.MultiTenantRedisHost == "" {
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

	return redis.NewClient(opts)
}

// workerRabbitMQAdapterProbe adapts the fetcher's rabbitmq.Adapter interface
// onto readyz.RabbitMQAdapterProbe. This mirrors the manager-side adapter,
// but lives here so the readyz wiring is localised to each component's
// bootstrap package (the adapter is a tiny nothing — duplicating it is
// cheaper than introducing a cross-component utility package).
type workerRabbitMQAdapterProbe struct {
	adapter pkgRabbitmq.Adapter
}

func newWorkerRabbitMQAdapterProbe(adapter pkgRabbitmq.Adapter) *workerRabbitMQAdapterProbe {
	if adapter == nil {
		return nil
	}

	return &workerRabbitMQAdapterProbe{adapter: adapter}
}

func (r *workerRabbitMQAdapterProbe) State() readyz.BreakerState {
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

// Ping uses the adapter's IsHealthy snapshot — mirrors the manager adapter.
// See manager/internal/bootstrap/readyz_adapters.go for the rationale.
func (r *workerRabbitMQAdapterProbe) Ping(ctx context.Context) error {
	if r == nil || r.adapter == nil {
		return errWorkerAdapterNil
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if !r.adapter.IsHealthy() {
		return errWorkerNotHealthy
	}

	return nil
}

var (
	errWorkerAdapterNil = &workerAdapterError{msg: "rabbitmq adapter not initialized"}
	errWorkerNotHealthy = &workerAdapterError{msg: "rabbitmq connection not healthy"}
)

type workerAdapterError struct{ msg string }

func (e *workerAdapterError) Error() string { return e.msg }

// s3HeadBucketShim wraps *s3.Client's HeadBucket method so we can pass the
// real client where readyz.S3HeadBucketAPI is expected without needing to
// expose the S3Repository's internals. The wrapper is a thin passthrough —
// most of the value lives in constructing it against the worker's
// storageRepository.
type s3HeadBucketShim struct {
	client *s3.Client
}

func (s *s3HeadBucketShim) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	if s == nil || s.client == nil {
		return nil, errS3ClientNil
	}

	return s.client.HeadBucket(ctx, params, optFns...)
}

var errS3ClientNil = &workerAdapterError{msg: "s3 client not initialized"}

// workerApplicationServiceName returns the fetcher service name used on
// tmclient calls. Mirrors the manager's helper.
func workerApplicationServiceName() string { return constant.ApplicationName }

// buildWorkerReadyzCheckers assembles the ordered set of DependencyCheckers
// the worker's /readyz micro-server registers.
//
// Composition differences versus the manager:
//   - Worker has an S3 bucket dep → S3BucketChecker always present.
//   - Worker's multi-tenant Redis is the ONLY Redis dep (no schema cache).
//   - Same Mongo / RabbitMQ MT carve-out rules.
func buildWorkerReadyzCheckers(deps *workerReadyzDeps) []readyz.DependencyChecker {
	// Nil-cfg guard: extending the existing deps == nil guard so a deps with
	// a nil cfg cannot panic on the cfg.* dereferences below.
	if deps == nil || deps.cfg == nil {
		return nil
	}

	checkers := make([]readyz.DependencyChecker, 0, 6)

	// --- MongoDB ---
	if deps.cfg.MultiTenantEnabled {
		// TLS posture: prefer URI-derived detection (mongodb+srv is implicit TLS,
		// query tls=true on plain mongodb is explicit). Fall back to CA-cert
		// presence so explicit operator-supplied CA configs still surface as
		// TLS-on for legacy deployments. Parse error is non-fatal — a malformed
		// URI cannot disprove TLS.
		mongoTLS, _ := readyz.DetectMongoTLS(buildWorkerMongoURI(deps.cfg))
		checkers = append(checkers, readyz.NewNAChecker(
			"mongodb",
			"multi-tenant: see /readyz/tenant/:id",
			readyz.TLSPtr(mongoTLS || deps.cfg.MongoTLSCACert != ""),
		))
	} else if deps.mongoClient != nil {
		checkers = append(checkers, readyz.NewMongoClientChecker(
			deps.mongoClient, buildWorkerMongoURI(deps.cfg),
		))
	}

	// --- RabbitMQ ---
	if deps.cfg.MultiTenantEnabled {
		checkers = append(checkers, readyz.NewNAChecker(
			"rabbitmq",
			"multi-tenant: see /readyz/tenant/:id",
			readyz.TLSPtr(deps.cfg.RabbitMQTLS),
		))
	} else if deps.rabbitAdapter != nil {
		checkers = append(checkers, readyz.NewRabbitMQAdapterChecker(
			newWorkerRabbitMQAdapterProbe(deps.rabbitAdapter),
			buildWorkerRabbitMQURL(deps.cfg),
		))
	}

	// --- Multi-tenant Redis (event discovery, MT mode only) ---
	if deps.mtRedisClient != nil {
		client := deps.mtRedisClient

		checkers = append(checkers, readyz.NewRedisClientCheckerFromFn(
			"multi_tenant_redis",
			func(ctx context.Context) error { return client.Ping(ctx).Err() },
			readyz.ComposeRedisURL(deps.cfg.MultiTenantRedisHost, deps.cfg.MultiTenantRedisPort, deps.cfg.MultiTenantRedisTLS),
		))
	}

	// --- S3 (worker-only) ---
	if deps.s3Client != nil {
		checkers = append(checkers, readyz.NewS3BucketChecker(
			&s3HeadBucketShim{client: deps.s3Client},
			deps.cfg.ObjectStorageBucket,
			deps.cfg.ObjectStorageEndpoint,
		))
	}

	// --- Tenant Manager ---
	if deps.cfg.MultiTenantEnabled {
		checkers = append(checkers, readyz.NewTenantManagerClientChecker(
			deps.tmClient,
			workerApplicationServiceName(),
			deps.cfg.MultiTenantURL,
			true,
		))
	}

	return checkers
}

// buildWorkerTenantHandler mirrors the manager's helper. Returns a 400
// handler when MT is off or any MT prerequisite is missing.
func buildWorkerTenantHandler(readyzCfg *readyz.Config, deps *workerReadyzDeps) fiber.Handler {
	// Extend nil safety: deps.cfg may be nil under test seams; short-circuit
	// to the disabled handler before dereferencing deps.cfg.MultiTenantEnabled.
	if deps == nil || deps.cfg == nil || !deps.cfg.MultiTenantEnabled ||
		deps.tmClient == nil || deps.tmMongoManager == nil || deps.tmRabbitMgr == nil {
		return readyz.NewDisabledTenantHandler()
	}

	th := readyz.NewTenantHandler(
		readyzCfg,
		deps.tmClient,
		workerApplicationServiceName(),
		readyz.NewTenantMongoChecker(deps.tmMongoManager),
		readyz.NewTenantRabbitMQChecker(deps.tmRabbitMgr),
	)

	return th.Fiber()
}
