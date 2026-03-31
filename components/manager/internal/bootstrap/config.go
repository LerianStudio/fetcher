package bootstrap

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	in2 "github.com/LerianStudio/fetcher/components/manager/internal/adapters/http/in"
	connectionCommand "github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	connectionQuery "github.com/LerianStudio/fetcher/components/manager/internal/services/query"

	cacheAdapter "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	datasourceFactory "github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	cacheRepo "github.com/LerianStudio/fetcher/pkg/ports/cache"
	"github.com/LerianStudio/fetcher/pkg/ports/messaging"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/ratelimit"
	redisCache "github.com/LerianStudio/fetcher/pkg/redis"
	"github.com/LerianStudio/fetcher/pkg/resolver"

	"github.com/LerianStudio/lib-auth/v2/auth/middleware"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libMongo "github.com/LerianStudio/lib-commons/v4/commons/mongo"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v4/commons/rabbitmq"
	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	tmevent "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/event"
	tmmiddleware "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/middleware"
	tmmongo "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/mongo"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/rabbitmq"
	"github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/tenantcache"
	"github.com/LerianStudio/lib-commons/v4/commons/zap"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
	"github.com/gofiber/fiber/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

// defaultMaxTenantPools is the fallback soft limit for tenant connection pools
// when MULTI_TENANT_MAX_TENANT_POOLS is unset or zero. This prevents unbounded
// pool growth. The value can be overridden via the environment variable.
const defaultMaxTenantPools = 100

// Config is the top-level configuration struct for the entire application.
type Config struct {
	// Service envs
	EnvName       string `env:"ENV_NAME"`
	ServerAddress string `env:"SERVER_ADDRESS"`
	LogLevel      string `env:"LOG_LEVEL"`
	// Otel and telemetry configuration envs
	OtelServiceName         string `env:"OTEL_RESOURCE_SERVICE_NAME"`
	OtelLibraryName         string `env:"OTEL_LIBRARY_NAME"`
	OtelServiceVersion      string `env:"OTEL_RESOURCE_SERVICE_VERSION"`
	OtelDeploymentEnv       string `env:"OTEL_RESOURCE_DEPLOYMENT_ENVIRONMENT"`
	OtelColExporterEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	EnableTelemetry         bool   `env:"ENABLE_TELEMETRY"`
	OtelInsecureExporter    bool   `env:"OTEL_INSECURE_EXPORTER"`
	// Mongo configuration envs
	MongoURI        string `env:"MONGO_URI"`
	MongoDBHost     string `env:"MONGO_HOST"`
	MongoDBName     string `env:"MONGO_NAME"`
	MongoDBUser     string `env:"MONGO_USER"`
	MongoDBPassword string `env:"MONGO_PASSWORD"`
	MongoDBPort     string `env:"MONGO_PORT"`
	// RabbitMQ configuration envs
	RabbitURI                   string `env:"RABBITMQ_URI"`
	RabbitMQHost                string `env:"RABBITMQ_HOST"`
	RabbitMQHealthCheckURL      string `env:"RABBITMQ_HEALTH_CHECK_URL"`
	RabbitMQPortHost            string `env:"RABBITMQ_PORT_HOST"`
	RabbitMQPortAMQP            string `env:"RABBITMQ_PORT_AMQP"`
	RabbitMQUser                string `env:"RABBITMQ_DEFAULT_USER"`
	RabbitMQPass                string `env:"RABBITMQ_DEFAULT_PASS"`
	RabbitMQGenerateReportQueue string `env:"RABBITMQ_FETCHER_WORK_QUEUE"`
	// Auth envs
	AuthAddress string `env:"PLUGIN_AUTH_ADDRESS"`
	AuthEnabled bool   `env:"PLUGIN_AUTH_ENABLED"`
	// License configuration envs
	LicenseKey      string `env:"LICENSE_KEY"`
	OrganizationIDs string `env:"ORGANIZATION_IDS"`
	// Encryption
	AppEncryptionKey        string `env:"APP_ENC_KEY"`
	AppEncryptionKeyVersion string `env:"APP_ENC_KEY_VERSION"`
	// Redis configuration envs
	RedisHost     string `env:"REDIS_HOST"`
	RedisPort     string `env:"REDIS_PORT"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB       string `env:"REDIS_DB"`
	// Schema cache TTL
	SchemaCacheTTLSeconds string `env:"SCHEMA_CACHE_TTL_SECONDS"`
	// Multi-Tenant configuration
	MultiTenantEnabled                  bool   `env:"MULTI_TENANT_ENABLED"`
	MultiTenantURL                      string `env:"MULTI_TENANT_URL"`
	MultiTenantRedisHost                string `env:"MULTI_TENANT_REDIS_HOST"`
	MultiTenantRedisPort                string `env:"MULTI_TENANT_REDIS_PORT" default:"6379"`
	MultiTenantRedisPassword            string `env:"MULTI_TENANT_REDIS_PASSWORD"`
	MultiTenantMaxTenantPools           int    `env:"MULTI_TENANT_MAX_TENANT_POOLS" default:"100"`
	MultiTenantIdleTimeoutSec           int    `env:"MULTI_TENANT_IDLE_TIMEOUT_SEC" default:"300"`
	MultiTenantCircuitBreakerThreshold  int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD" default:"5"`
	MultiTenantCircuitBreakerTimeoutSec int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC" default:"30"`
	MultiTenantServiceAPIKey            string `env:"MULTI_TENANT_SERVICE_API_KEY"`
	MultiTenantCacheTTLSec              int    `env:"MULTI_TENANT_CACHE_TTL_SEC" default:"120"`
	MultiTenantTimeout                  int    `env:"MULTI_TENANT_TIMEOUT" default:"30"`
}

type managerRepositories struct {
	connection *connection.ConnectionMongoDBRepository
	job        *job.JobMongoDBRepository
}

type managerCrypto struct {
	service       *crypto.AESGCMService
	messageSigner crypto.Signer
}

type managerPlatformDependencies struct {
	rabbitPublisher     messaging.MessagePublisher
	rabbitMQCleanup     func()
	authClient          *middleware.AuthClient
	licenseClient       *libLicense.LicenseClient
	connectionTestStore *ratelimit.RateLimiter
	schemaCache         cacheRepo.SchemaCacheRepository
}

var (
	setConfigFromEnvVars  = pkg.SetConfigFromEnvVars
	newManagerLogger      = func(cfg zap.Config) (libLog.Logger, error) { return zap.New(cfg) }
	newManagerTelemetry   = libOtel.NewTelemetry
	applyTelemetryGlobals = func(telemetry *libOtel.Telemetry) error {
		return telemetry.ApplyGlobals()
	}
	newManagerMongoClient   = libMongo.NewClient
	newConnectionRepository = connection.NewConnectionMongoDBRepository
	newJobRepository        = job.NewJobMongoDBRepository
	newTenantManagerClient  = tmclient.NewClient
	newSchemaCacheStore     = func(cfg redisCache.RedisConfig, logger libLog.Logger, ttl time.Duration, prefix string) (redisCache.Cache[model.DataSourceSchema], error) {
		return redisCache.NewCacheWithFallback[model.DataSourceSchema](cfg, logger, ttl, prefix)
	}
	loadConfigFn               = loadConfig
	initLoggerAndTelemetryFn   = initLoggerAndTelemetry
	initMongoRepositoriesFn    = initMongoRepositories
	initCryptoFn               = initCrypto
	initPlatformDependenciesFn = initPlatformDependencies
	assembleServiceFn          = assembleService
)

// InitServers initiate http and grpc servers.
func InitServers() (*Service, error) {
	cfg, err := loadConfigFn()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	logger, telemetry, err := initLoggerAndTelemetryFn(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.MultiTenantEnabled {
		logger.Log(ctx, libLog.LevelInfo, "Multi-tenant mode ENABLED")

		if cfg.MultiTenantURL == "" {
			return nil, fmt.Errorf("MULTI_TENANT_URL is required when MULTI_TENANT_ENABLED=true")
		}

		if cfg.MultiTenantServiceAPIKey == "" {
			return nil, fmt.Errorf("MULTI_TENANT_SERVICE_API_KEY is required when MULTI_TENANT_ENABLED=true")
		}
	} else {
		logger.Log(ctx, libLog.LevelInfo, "Running in SINGLE-TENANT MODE")
	}

	repositories, err := initMongoRepositoriesFn(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	cryptoDependencies, err := initCryptoFn(cfg, logger)
	if err != nil {
		return nil, err
	}

	platformDependencies, err := initPlatformDependenciesFn(cfg, logger, cryptoDependencies.messageSigner)
	if err != nil {
		return nil, err
	}

	return assembleServiceFn(
		cfg,
		logger,
		telemetry,
		repositories,
		cryptoDependencies.service,
		platformDependencies,
	)
}

func loadConfig() (*Config, error) {
	cfg := &Config{}
	if err := setConfigFromEnvVars(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func initLoggerAndTelemetry(cfg *Config) (libLog.Logger, *libOtel.Telemetry, error) {
	logger, err := newManagerLogger(zap.Config{
		Environment:     resolveZapEnvironment(cfg.EnvName),
		Level:           cfg.LogLevel,
		OTelLibraryName: cfg.OtelLibraryName,
	})
	if err != nil {
		return nil, nil, err
	}

	telemetry, err := newManagerTelemetry(libOtel.TelemetryConfig{
		LibraryName:               cfg.OtelLibraryName,
		ServiceName:               cfg.OtelServiceName,
		ServiceVersion:            cfg.OtelServiceVersion,
		DeploymentEnv:             cfg.OtelDeploymentEnv,
		CollectorExporterEndpoint: cfg.OtelColExporterEndpoint,
		EnableTelemetry:           cfg.EnableTelemetry,
		InsecureExporter:          cfg.OtelInsecureExporter,
		Logger:                    logger,
	})
	if err != nil {
		return nil, nil, err
	}

	if err := applyTelemetryGlobals(telemetry); err != nil {
		return nil, nil, err
	}

	return logger, telemetry, nil
}

func initMongoRepositories(ctx context.Context, cfg *Config, logger libLog.Logger) (*managerRepositories, error) {
	mongoConnection, err := newManagerMongoClient(ctx, libMongo.Config{
		URI:      buildMongoSource(cfg),
		Database: cfg.MongoDBName,
		Logger:   logger,
	})
	if err != nil {
		return nil, wrapBootstrapError("initialize MongoDB client", err)
	}

	// Wrap the MongoDB connection to implement multi-tenant checker.
	// When multi-tenant mode is enabled, tenant context is required
	// instead of silently falling back to the default DB.
	mongoProvider := mongodb.NewMultiTenantMongoProvider(mongoConnection, cfg.MultiTenantEnabled)

	connectionRepository, err := newConnectionRepository(ctx, mongoProvider, cfg.MongoDBName)
	if err != nil {
		return nil, wrapBootstrapError("create MongoDB connection repository", err)
	}

	jobRepository, err := newJobRepository(ctx, mongoProvider, cfg.MongoDBName)
	if err != nil {
		return nil, wrapBootstrapError("create MongoDB job repository", err)
	}

	// Ensure indexes only in single-tenant mode. In multi-tenant mode, there is no
	// tenant context available at bootstrap time (it is injected at runtime by the
	// TenantMiddleware or RabbitMQ consumer), so EnsureIndexes would fail with
	// ErrTenantContextRequired. Tenant-scoped indexes are managed externally.
	if !cfg.MultiTenantEnabled {
		logger.Log(ctx, libLog.LevelInfo, "Ensuring MongoDB indexes exist for connections...")

		if err := connectionRepository.EnsureIndexes(ctx); err != nil {
			return nil, wrapBootstrapError("ensure MongoDB connection indexes", err)
		}

		logger.Log(ctx, libLog.LevelInfo, "Ensuring MongoDB indexes exist for jobs...")

		if err := jobRepository.EnsureIndexes(ctx); err != nil {
			return nil, wrapBootstrapError("ensure MongoDB job indexes", err)
		}
	} else {
		logger.Log(ctx, libLog.LevelInfo, "Multi-tenant mode: skipping index creation at bootstrap (indexes are managed per-tenant)")
	}

	return &managerRepositories{
		connection: connectionRepository,
		job:        jobRepository,
	}, nil
}

func initCrypto(cfg *Config, logger libLog.Logger) (*managerCrypto, error) {
	masterKey, err := crypto.DecodeMasterKey(cfg.AppEncryptionKey)
	if err != nil {
		return nil, wrapBootstrapError("decode master encryption key", err)
	}

	keyDeriver, err := crypto.NewHKDFKeyDeriver(masterKey)
	if err != nil {
		return nil, wrapBootstrapError("initialize key deriver", err)
	}

	logger.Log(context.Background(), libLog.LevelInfo, "Key derivation initialized successfully")

	cryptoService, err := crypto.NewAESGCMService(keyDeriver.GetCredentialKey(), cfg.AppEncryptionKeyVersion)
	if err != nil {
		return nil, wrapBootstrapError("initialize crypto service", err)
	}

	messageSigner, err := crypto.NewHMACSigner(keyDeriver.GetInternalHMACKey(), crypto.SignatureVersion)
	if err != nil {
		return nil, wrapBootstrapError("initialize message signer", err)
	}

	return &managerCrypto{
		service:       cryptoService,
		messageSigner: messageSigner,
	}, nil
}

func initPlatformDependencies(cfg *Config, logger libLog.Logger, messageSigner crypto.Signer) (*managerPlatformDependencies, error) {
	var rabbitPublisher messaging.MessagePublisher

	var rabbitMQCleanup func()

	if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
		// Multi-tenant mode: use tmrabbitmq.Manager for per-tenant vhost isolation.
		// Do NOT create a static connection to the default vhost — the RabbitMQ user
		// only has permissions on tenant-specific vhosts.
		tmClient, err := newTenantManagerClient(cfg.MultiTenantURL, logger, resolverTMClientOpts(cfg)...)
		if err != nil {
			return nil, wrapBootstrapError("create tenant manager client for RabbitMQ", err)
		}

		maxPools := resolvedMaxTenantPools(cfg)

		var rabbitOpts []tmrabbitmq.Option

		rabbitOpts = append(rabbitOpts,
			tmrabbitmq.WithModule(constant.ModuleManager),
			tmrabbitmq.WithLogger(logger),
			tmrabbitmq.WithMaxTenantPools(maxPools),
		)

		if cfg.MultiTenantIdleTimeoutSec > 0 {
			rabbitOpts = append(rabbitOpts, tmrabbitmq.WithIdleTimeout(
				time.Duration(cfg.MultiTenantIdleTimeoutSec)*time.Second,
			))
		}

		rabbitMQManager := tmrabbitmq.NewManager(tmClient, constant.ApplicationName, rabbitOpts...)

		rabbitPublisher = &multiTenantPublisher{
			manager: newManagerRabbitMQAdapter(rabbitMQManager),
			signer:  messageSigner,
			logger:  logger,
		}

		rabbitMQCleanup = func() {
			logger.Log(context.Background(), libLog.LevelInfo, "Cleanup: closing multi-tenant RabbitMQ manager")

			if closeErr := rabbitMQManager.Close(context.Background()); closeErr != nil {
				logger.Log(context.Background(), libLog.LevelError, "Cleanup: failed to close RabbitMQ manager", libLog.Err(closeErr))
			}
		}

		logger.Log(context.Background(), libLog.LevelInfo, "RabbitMQ: multi-tenant publisher initialized with tmrabbitmq.Manager")
	} else {
		// Single-tenant mode: use static connection (existing behavior)
		rabbitMQConnection := &libRabbitmq.RabbitMQConnection{
			ConnectionStringSource: buildRabbitMQSource(cfg),
			HealthCheckURL:         cfg.RabbitMQHealthCheckURL,
			Host:                   cfg.RabbitMQHost,
			Port:                   cfg.RabbitMQPortHost,
			User:                   cfg.RabbitMQUser,
			Pass:                   cfg.RabbitMQPass,
			Queue:                  cfg.RabbitMQGenerateReportQueue,
			Logger:                 logger,
		}

		rabbitMQOptions := rabbitmq.DefaultOptions()
		rabbitMQOptions.Signer = messageSigner

		rabbitPublisher = rabbitmq.NewRabbitMQAdapterWithOptions(rabbitMQConnection, rabbitMQOptions)
	}

	authLoggerV4, authLogErr := zap.New(zap.Config{
		Environment:     resolveZapEnvironment(cfg.EnvName),
		Level:           cfg.LogLevel,
		OTelLibraryName: constant.ApplicationName + "-auth",
	})
	if authLogErr != nil {
		return nil, wrapBootstrapError("initialize auth logger", authLogErr)
	}

	var authLogger libLog.Logger = authLoggerV4

	licenseLoggerV4, licenseLogErr := zap.New(zap.Config{
		Environment:     resolveZapEnvironment(cfg.EnvName),
		Level:           cfg.LogLevel,
		OTelLibraryName: constant.ApplicationName + "-license",
	})
	if licenseLogErr != nil {
		return nil, wrapBootstrapError("initialize license logger", licenseLogErr)
	}

	var licenseLogger libLog.Logger = licenseLoggerV4

	schemaCacheTTL := getSchemaCacheTTL(cfg.SchemaCacheTTLSeconds)

	genericCache, errCache := newSchemaCacheStore(
		redisCache.RedisConfig{
			Host:     cfg.RedisHost,
			Port:     cfg.RedisPort,
			Password: cfg.RedisPassword,
			DB:       getRedisDB(cfg.RedisDB),
		},
		logger,
		schemaCacheTTL,
		"fetcher:schema:",
	)
	if errCache != nil {
		return nil, wrapBootstrapError("initialize schema cache", errCache)
	}

	return &managerPlatformDependencies{
		rabbitPublisher: rabbitPublisher,
		rabbitMQCleanup: rabbitMQCleanup,
		authClient:      middleware.NewAuthClient(cfg.AuthAddress, cfg.AuthEnabled, &authLogger),
		licenseClient: libLicense.NewLicenseClient(
			constant.ApplicationName,
			cfg.LicenseKey,
			cfg.OrganizationIDs,
			&licenseLogger,
		),
		connectionTestStore: ratelimit.New(10, time.Minute),
		schemaCache:         cacheAdapter.NewSchemaCache(genericCache, schemaCacheTTL),
	}, nil
}

func assembleService(
	cfg *Config,
	logger libLog.Logger,
	telemetry *libOtel.Telemetry,
	repositories *managerRepositories,
	cryptoService *crypto.AESGCMService,
	platformDependencies *managerPlatformDependencies,
) (*Service, error) {
	// Create ConnectionResolver based on multi-tenant mode (needed by query services below)
	var connResolver resolver.ConnectionResolver

	registry := resolver.NewInternalDatasourceRegistry()

	if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
		tmClient, tmErr := newTenantManagerClient(cfg.MultiTenantURL, logger, resolverTMClientOpts(cfg)...)
		if tmErr != nil {
			return nil, wrapBootstrapError("create tenant manager client for resolver", tmErr)
		}

		tenantAdapter := resolver.NewTenantManagerAdapter(tmClient)
		connResolver = resolver.NewMultiTenantResolver(repositories.connection, registry, tenantAdapter)
	} else {
		// Single-tenant: env-based connections deferred to follow-up task.
		// Empty map means internal datasources must still be registered via Connection API.
		envConnections := make(map[string]*model.Connection)
		connResolver = resolver.NewSingleTenantResolver(repositories.connection, registry, envConnections)
	}

	dsFactory := datasourceFactory.NewDataSourceFromConnectionWithLogger(logger)

	createConnectionCmd := connectionCommand.NewCreateConnection(repositories.connection, cryptoService)
	updateConnectionCmd := connectionCommand.NewUpdateConnection(repositories.connection, repositories.job, cryptoService)
	deleteConnectionCmd := connectionCommand.NewDeleteConnection(repositories.connection, repositories.job)
	getConnectionQuery := connectionQuery.NewGetConnection(repositories.connection, connResolver, registry)
	listConnectionsQuery := connectionQuery.NewListConnections(repositories.connection, connResolver)
	testConnectionQuery := connectionQuery.NewTestConnection(repositories.connection, cryptoService, platformDependencies.connectionTestStore, dsFactory, connResolver, registry)
	validateSchemaQuery := connectionQuery.NewValidateSchema(repositories.connection, cryptoService, platformDependencies.schemaCache, dsFactory, connResolver)
	getConnectionSchemaQuery := connectionQuery.NewGetConnectionSchema(repositories.connection, cryptoService, dsFactory, connResolver, registry)

	connectionHandler := in2.NewConnectionHandler(
		createConnectionCmd,
		updateConnectionCmd,
		deleteConnectionCmd,
		getConnectionQuery,
		listConnectionsQuery,
		testConnectionQuery,
		validateSchemaQuery,
		getConnectionSchemaQuery,
	)

	migrationHandler := in2.NewMigrationHandler(
		connectionCommand.NewAssignConnection(repositories.connection),
		connectionQuery.NewListUnassignedConnections(repositories.connection),
	)

	createFetcherJobCmd := connectionCommand.NewCreateFetcherJob(
		repositories.connection,
		repositories.job,
		cryptoService,
		platformDependencies.rabbitPublisher,
		cfg.RabbitMQGenerateReportQueue,
		dsFactory,
		connResolver,
	)

	fetcherHandler := in2.NewFetcherHandler(
		createFetcherJobCmd,
		connectionQuery.NewGetJob(repositories.job),
	)

	// Init multi-tenant middleware (nil if disabled)
	tenantHandler, mtEventCleanup, err := initMultiTenantMiddleware(cfg, logger)
	if err != nil {
		return nil, err
	}

	httpApp := in2.NewRoutes(
		logger,
		telemetry,
		platformDependencies.authClient,
		platformDependencies.licenseClient,
		connectionHandler,
		migrationHandler,
		fetcherHandler,
		tenantHandler,
	)

	var shutdownHooks []func(context.Context) error

	if platformDependencies.rabbitMQCleanup != nil {
		cleanup := platformDependencies.rabbitMQCleanup

		shutdownHooks = append(shutdownHooks, func(context.Context) error {
			cleanup()
			return nil
		})
	}

	if mtEventCleanup != nil {
		eventCleanup := mtEventCleanup

		shutdownHooks = append(shutdownHooks, func(context.Context) error {
			eventCleanup()
			return nil
		})
	}

	return &Service{
		Server: NewServer(cfg, httpApp, logger, telemetry, platformDependencies.licenseClient, shutdownHooks...),
		Logger: logger,
	}, nil
}

// initMultiTenantMiddleware creates a TenantMiddleware Fiber handler if multi-tenant
// mode is enabled and configured. Returns (nil, nil, nil) when multi-tenant is disabled.
// The middleware resolves tenant-specific MongoDB connections from JWT claims.
//
// Per multi-tenant.md standards:
//   - Circuit breaker is MANDATORY for the Tenant Manager client
//   - Uses constant.ApplicationName and constant.ModuleManager for service/module identity
//   - WithMongoManager configures MongoDB connection pool management
//   - WithTenantCache + WithTenantLoader for cache-first strategy
//   - EventListener + EventDispatcher for event-driven tenant discovery via Redis Pub/Sub
//
// Returns:
//   - fiber.Handler: the middleware handler (nil when disabled)
//   - func(): cleanup function to stop the event listener (nil when disabled)
//   - error: initialization error
func initMultiTenantMiddleware(cfg *Config, logger libLog.Logger) (fiber.Handler, func(), error) {
	if !cfg.MultiTenantEnabled || cfg.MultiTenantURL == "" {
		return nil, nil, nil
	}

	// Create Tenant Manager HTTP client with circuit breaker (MANDATORY per multi-tenant.md).
	// Default: 5 consecutive failures, 30s half-open timeout.
	var clientOpts []tmclient.ClientOption

	// Allow plaintext HTTP for local/dev environments where TLS is not configured.
	if strings.HasPrefix(strings.ToLower(cfg.MultiTenantURL), "http://") && strings.ToLower(cfg.EnvName) != "production" {
		clientOpts = append(clientOpts, tmclient.WithAllowInsecureHTTP())
	}

	clientOpts = append(clientOpts,
		tmclient.WithServiceAPIKey(cfg.MultiTenantServiceAPIKey),
	)

	if cfg.MultiTenantTimeout > 0 {
		clientOpts = append(clientOpts, tmclient.WithTimeout(time.Duration(cfg.MultiTenantTimeout)*time.Second))
	}

	if cfg.MultiTenantCacheTTLSec > 0 {
		clientOpts = append(clientOpts, tmclient.WithCacheTTL(time.Duration(cfg.MultiTenantCacheTTLSec)*time.Second))
	}

	if cfg.MultiTenantCircuitBreakerThreshold > 0 {
		clientOpts = append(clientOpts,
			tmclient.WithCircuitBreaker(
				cfg.MultiTenantCircuitBreakerThreshold,
				time.Duration(cfg.MultiTenantCircuitBreakerTimeoutSec)*time.Second,
			),
		)
	} else {
		clientOpts = append(clientOpts,
			tmclient.WithCircuitBreaker(5, 30*time.Second),
		)
	}

	tmClient, err := newTenantManagerClient(cfg.MultiTenantURL, logger, clientOpts...)
	if err != nil {
		logger.Log(context.Background(), libLog.LevelError, "failed to create tenant manager client", libLog.Err(err))

		return nil, nil, wrapBootstrapError("create tenant manager client", err)
	}

	// Create MongoDB Manager for tenant connection pool management
	var mongoOpts []tmmongo.Option

	mongoOpts = append(mongoOpts,
		tmmongo.WithModule(constant.ModuleManager),
		tmmongo.WithLogger(logger),
	)

	// Apply tenant pool limit. Use configured value if > 0, otherwise apply a sensible
	// default to prevent unbounded pool growth. The value matches the value in .env.example.
	maxTenantPools := cfg.MultiTenantMaxTenantPools
	if maxTenantPools <= 0 {
		maxTenantPools = defaultMaxTenantPools
	}

	mongoOpts = append(mongoOpts, tmmongo.WithMaxTenantPools(maxTenantPools))

	if cfg.MultiTenantIdleTimeoutSec > 0 {
		mongoOpts = append(mongoOpts, tmmongo.WithIdleTimeout(
			time.Duration(cfg.MultiTenantIdleTimeoutSec)*time.Second,
		))
	}

	mongoManager := tmmongo.NewManager(tmClient, constant.ApplicationName, mongoOpts...)

	// Create TenantCache and TenantLoader for cache-first strategy per multi-tenant.md
	tenantCache := tenantcache.NewTenantCache()

	var cacheTTL time.Duration
	if cfg.MultiTenantCacheTTLSec > 0 {
		cacheTTL = time.Duration(cfg.MultiTenantCacheTTLSec) * time.Second
	}

	tenantLoader := tenantcache.NewTenantLoader(tmClient, tenantCache, constant.ApplicationName, cacheTTL, logger)

	// Create TenantMiddleware with MongoDB manager, TenantCache, and TenantLoader
	tenantMid := tmmiddleware.NewTenantMiddleware(
		tmmiddleware.WithMB(mongoManager),
		tmmiddleware.WithTenantCache(tenantCache),
		tmmiddleware.WithTenantLoader(tenantLoader),
	)

	// Set up event-driven tenant discovery via Redis Pub/Sub
	cleanup, eventErr := initManagerEventDiscovery(cfg, logger, tenantCache, tenantLoader, mongoManager)
	if eventErr != nil {
		return nil, nil, eventErr
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Multi-tenant middleware initialized: url=%s, module=%s", cfg.MultiTenantURL, constant.ModuleManager))

	return tenantMid.WithTenantDB, cleanup, nil
}

// initManagerEventDiscovery creates and starts the EventDispatcher and EventListener
// for event-driven tenant discovery via Redis Pub/Sub. Returns a cleanup function
// to stop the listener on shutdown.
func initManagerEventDiscovery(
	cfg *Config,
	logger libLog.Logger,
	cache *tenantcache.TenantCache,
	loader *tenantcache.TenantLoader,
	mongoManager *tmmongo.Manager,
) (func(), error) {
	if cfg.MultiTenantRedisHost == "" {
		// No Redis configured for event discovery; return a no-op cleanup.
		logger.Log(context.Background(), libLog.LevelInfo, "Multi-tenant event discovery: MULTI_TENANT_REDIS_HOST not set, skipping event listener")

		return func() {}, nil
	}

	redisPort := cfg.MultiTenantRedisPort
	if redisPort == "" {
		redisPort = "6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(cfg.MultiTenantRedisHost, redisPort),
		Password: cfg.MultiTenantRedisPassword,
	})

	var cacheTTL time.Duration
	if cfg.MultiTenantCacheTTLSec > 0 {
		cacheTTL = time.Duration(cfg.MultiTenantCacheTTLSec) * time.Second
	}

	dispatcher := tmevent.NewEventDispatcher(
		cache,
		loader,
		constant.ApplicationName,
		tmevent.WithMongo(mongoManager),
		tmevent.WithDispatcherLogger(logger),
		tmevent.WithCacheTTL(cacheTTL),
	)

	listener, err := tmevent.NewTenantEventListener(
		redisClient,
		dispatcher.HandleEvent,
		tmevent.WithListenerLogger(logger),
		tmevent.WithService(constant.ApplicationName),
	)
	if err != nil {
		return nil, wrapBootstrapError("create tenant event listener", err)
	}

	if err := listener.Start(context.Background()); err != nil {
		return nil, wrapBootstrapError("start tenant event listener", err)
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Multi-tenant event listener started: redis=%s", net.JoinHostPort(cfg.MultiTenantRedisHost, redisPort)))

	cleanup := func() {
		logger.Log(context.Background(), libLog.LevelInfo, "Stopping multi-tenant event listener")

		if stopErr := listener.Stop(); stopErr != nil {
			logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to stop tenant event listener: %v", stopErr))
		}

		if closeErr := redisClient.Close(); closeErr != nil {
			logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to close tenant event Redis client: %v", closeErr))
		}
	}

	return cleanup, nil
}

func buildMongoSource(cfg *Config) string {
	return buildCredentialURL(cfg.MongoURI, cfg.MongoDBUser, cfg.MongoDBPassword, cfg.MongoDBHost, cfg.MongoDBPort)
}

func buildRabbitMQSource(cfg *Config) string {
	return buildCredentialURL(cfg.RabbitURI, cfg.RabbitMQUser, cfg.RabbitMQPass, cfg.RabbitMQHost, cfg.RabbitMQPortAMQP)
}

func buildCredentialURL(scheme, user, password, host, port string) string {
	targetURL := &url.URL{Scheme: scheme}

	if user != "" || password != "" {
		targetURL.User = url.UserPassword(user, password)
	}

	if port != "" {
		targetURL.Host = net.JoinHostPort(host, port)
	} else {
		targetURL.Host = host
	}

	return targetURL.String()
}

// getSchemaCacheTTL parses the TTL from string and returns a time.Duration.
// Returns DefaultSchemaCacheTTL if the string is empty or invalid.
func getSchemaCacheTTL(ttlStr string) time.Duration {
	if ttlStr == "" {
		return cacheRepo.DefaultSchemaCacheTTL
	}

	ttlSeconds, err := strconv.Atoi(ttlStr)
	if err != nil {
		return cacheRepo.DefaultSchemaCacheTTL
	}

	return time.Duration(ttlSeconds) * time.Second
}

// getRedisDB parses the Redis database number from string.
// Returns 0 if the string is empty or invalid.
func getRedisDB(dbStr string) int {
	if dbStr == "" {
		return 0
	}

	db, err := strconv.Atoi(dbStr)
	if err != nil {
		return 0
	}

	return db
}

func resolveZapEnvironment(env string) zap.Environment {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "production", "prod":
		return zap.EnvironmentProduction
	case "staging", "stage":
		return zap.EnvironmentStaging
	case "uat":
		return zap.EnvironmentUAT
	case "development", "dev":
		return zap.EnvironmentDevelopment
	case "local":
		return zap.EnvironmentLocal
	default:
		return zap.EnvironmentLocal
	}
}

// resolverTMClientOpts builds tmclient options for the ConnectionResolver's tenant-manager client.
// Reuses the same circuit breaker and insecure HTTP settings as the middleware client.
func resolverTMClientOpts(cfg *Config) []tmclient.ClientOption {
	var opts []tmclient.ClientOption

	if strings.HasPrefix(strings.ToLower(cfg.MultiTenantURL), "http://") && strings.ToLower(cfg.EnvName) != "production" {
		opts = append(opts, tmclient.WithAllowInsecureHTTP())
	}

	opts = append(opts, tmclient.WithServiceAPIKey(cfg.MultiTenantServiceAPIKey))

	if cfg.MultiTenantTimeout > 0 {
		opts = append(opts, tmclient.WithTimeout(time.Duration(cfg.MultiTenantTimeout)*time.Second))
	}

	if cfg.MultiTenantCacheTTLSec > 0 {
		opts = append(opts, tmclient.WithCacheTTL(time.Duration(cfg.MultiTenantCacheTTLSec)*time.Second))
	}

	if cfg.MultiTenantCircuitBreakerThreshold > 0 {
		opts = append(opts,
			tmclient.WithCircuitBreaker(
				cfg.MultiTenantCircuitBreakerThreshold,
				time.Duration(cfg.MultiTenantCircuitBreakerTimeoutSec)*time.Second,
			),
		)
	} else {
		opts = append(opts, tmclient.WithCircuitBreaker(5, 30*time.Second))
	}

	return opts
}

// resolvedMaxTenantPools returns the configured value if > 0, or the default.
func resolvedMaxTenantPools(cfg *Config) int {
	if cfg.MultiTenantMaxTenantPools > 0 {
		return cfg.MultiTenantMaxTenantPools
	}

	return defaultMaxTenantPools
}

// managerRabbitMQChannel abstracts an AMQP channel for multi-tenant publishing.
type managerRabbitMQChannel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Close() error
}

// managerRabbitMQManagerInterface abstracts the tenant-aware RabbitMQ connection manager.
type managerRabbitMQManagerInterface interface {
	GetChannel(ctx context.Context, tenantID string) (managerRabbitMQChannel, error)
}

// managerRabbitMQAdapter wraps tmrabbitmq.Manager to satisfy managerRabbitMQManagerInterface.
type managerRabbitMQAdapter struct {
	manager *tmrabbitmq.Manager
}

func newManagerRabbitMQAdapter(manager *tmrabbitmq.Manager) *managerRabbitMQAdapter {
	return &managerRabbitMQAdapter{manager: manager}
}

// GetChannel wraps tmrabbitmq.Manager.GetChannel and returns the channel as managerRabbitMQChannel.
func (a *managerRabbitMQAdapter) GetChannel(ctx context.Context, tenantID string) (managerRabbitMQChannel, error) {
	return a.manager.GetChannel(ctx, tenantID)
}

// multiTenantPublisher implements messaging.MessagePublisher using per-tenant RabbitMQ channels.
// In multi-tenant mode, each ProducerDefault call resolves the tenant from context, obtains
// a channel on the tenant-specific vhost via tmrabbitmq.Manager, publishes, and closes the channel.
type multiTenantPublisher struct {
	manager managerRabbitMQManagerInterface
	signer  crypto.Signer
	logger  libLog.Logger
}

// ProducerDefault publishes a message to a tenant-specific RabbitMQ vhost.
// The tenant ID is extracted from context (set by the TenantMiddleware).
func (p *multiTenantPublisher) ProducerDefault(ctx context.Context, exchange, key string, queueMessage []byte, header *map[string]any) error {
	tenantID := tmcore.GetTenantIDContext(ctx)
	if tenantID == "" {
		return fmt.Errorf("multi-tenant RabbitMQ: no tenant ID in context")
	}

	ch, err := p.manager.GetChannel(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("get RabbitMQ channel for tenant %s: %w", tenantID, err)
	}

	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			p.logger.Log(ctx, libLog.LevelError, fmt.Sprintf("error closing RabbitMQ channel for tenant %s: %v", tenantID, closeErr))
		}
	}()

	amqpHeaders := amqp.Table{}

	if header != nil {
		for k, v := range *header {
			amqpHeaders[k] = v
		}
	}

	// Sign message if signer is configured (preserves message signing from single-tenant mode)
	if p.signer != nil {
		timestamp := time.Now().UTC().Unix()
		payload := crypto.BuildSignaturePayload(timestamp, queueMessage)
		signature := p.signer.Sign(payload)

		amqpHeaders[rabbitmq.HeaderMessageSignature] = signature
		amqpHeaders[rabbitmq.HeaderSignatureTimestamp] = strconv.FormatInt(timestamp, 10)
		amqpHeaders[rabbitmq.HeaderSignatureVersion] = p.signer.SignatureVersion()
	}

	return ch.PublishWithContext(ctx, exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        queueMessage,
		Headers:     amqpHeaders,
	})
}

func wrapBootstrapError(action string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return nil
}
