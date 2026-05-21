package bootstrap

import (
	"context"
	"crypto/tls"
	"net"

	workerRabbitAdapters "github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	"github.com/LerianStudio/fetcher/pkg/constant"
	portStorage "github.com/LerianStudio/fetcher/pkg/ports/storage"
	pkgRabbitmq "github.com/LerianStudio/fetcher/pkg/rabbitmq"
	pkgStorage "github.com/LerianStudio/fetcher/pkg/storage"
	libMongo "github.com/LerianStudio/lib-commons/v5/commons/mongo"
	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	tmmongo "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/mongo"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/rabbitmq"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// workerReadyzDeps bundles the handles the worker bootstrap forwards to
// NewHealthServer. All fields are nullable — the checker-building logic
// either skips a missing dep or carves out an NA entry. closers are
// invoked by the top-level Service on shutdown.
type workerReadyzDeps struct {
	cfg            *Config
	mongoClient    *libMongo.Client
	rabbitAdapter  pkgRabbitmq.Adapter
	s3Client       *s3.Client
	mtRedisClient  *redis.Client
	tmClient       readyz.TMClient
	tmMongoManager *tmmongo.Manager
	tmRabbitMgr    *tmrabbitmq.Manager

	closers []func() error
}

// newWorkerReadyzDepsST builds deps for the single-tenant path. Tenant-
// scoped fields stay nil and /readyz/tenant/:id falls back to the disabled
// handler.
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

// newWorkerReadyzDepsMT builds deps for the multi-tenant path. The
// single-tenant Rabbit adapter is nil (the global /readyz emits an
// NAChecker; tenant-scoped probes go through tmrabbitmq.Manager). The
// MT-Redis client is owned by readyz so its lifecycle stays independent
// of the event listener's own Redis client.
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

// close runs every closer in order, ignoring errors so a single cleanup
// failure does not abort shutdown.
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

// newReadyzMTRedis builds the standalone MT-Redis client used by /readyz.
// Returns nil when MT-Redis is not configured.
//
// When cfg.MultiTenantRedisTLS is true, opts.TLSConfig is populated with TLS
// 1.2 as the floor. Custom CA bundles via MULTI_TENANT_REDIS_CA_CERT are
// rejected during bootstrap; readyz uses the runtime system trust store.
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

	if cfg.MultiTenantRedisTLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	return redis.NewClient(opts)
}

// workerRabbitMQAdapterProbe duplicates the manager-side adapter so each
// component's readyz wiring stays self-contained — the adapter is small
// enough that a cross-component utility package would not pay off.
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

// s3HeadBucketShim adapts *s3.Client to readyz.S3HeadBucketAPI without
// exposing the worker's S3Repository internals.
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

func workerApplicationServiceName() string { return constant.ApplicationName }

// buildWorkerReadyzCheckers composes the worker's /readyz checkers.
// Compared to the manager: worker always probes S3, the only Redis dep is
// multi-tenant Redis (no schema cache), and the Mongo / RabbitMQ MT
// carve-outs match.
func buildWorkerReadyzCheckers(deps *workerReadyzDeps) []readyz.DependencyChecker {
	if deps == nil || deps.cfg == nil {
		return nil
	}

	checkers := make([]readyz.DependencyChecker, 0, 6)

	if deps.cfg.MultiTenantEnabled {
		// CA-cert fallback covers legacy deployments that supply a CA
		// without a TLS-signaling URI (mongodb+srv or tls=true).
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

	if deps.mtRedisClient != nil {
		client := deps.mtRedisClient

		checkers = append(checkers, readyz.NewRedisClientCheckerFromFn(
			"multi_tenant_redis",
			func(ctx context.Context) error { return client.Ping(ctx).Err() },
			readyz.ComposeRedisURL(deps.cfg.MultiTenantRedisHost, deps.cfg.MultiTenantRedisPort, deps.cfg.MultiTenantRedisTLS),
		))
	}

	if deps.s3Client != nil {
		checkers = append(checkers, readyz.NewS3BucketChecker(
			&s3HeadBucketShim{client: deps.s3Client},
			deps.cfg.ObjectStorageBucket,
			deps.cfg.ObjectStorageEndpoint,
		))
	}

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

// buildWorkerTenantHandler returns a 400 handler when MT is off or any MT
// prerequisite is missing, so the route stays mounted.
func buildWorkerTenantHandler(readyzCfg *readyz.Config, deps *workerReadyzDeps) fiber.Handler {
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
