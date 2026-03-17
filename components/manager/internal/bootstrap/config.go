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
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/ratelimit"
	redisCache "github.com/LerianStudio/fetcher/pkg/redis"

	"github.com/LerianStudio/lib-auth/v2/auth/middleware"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libMongo "github.com/LerianStudio/lib-commons/v4/commons/mongo"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v4/commons/rabbitmq"
	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	tmmiddleware "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/middleware"
	tmmongo "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/mongo"
	"github.com/LerianStudio/lib-commons/v4/commons/zap"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
	"github.com/gofiber/fiber/v2"
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
	// Mongo configuration envs
	MongoURI        string `env:"MONGO_URI"`
	MongoDBHost     string `env:"MONGO_HOST"`
	MongoDBName     string `env:"MONGO_NAME"`
	MongoDBUser     string `env:"MONGO_USER"`
	MongoDBPassword string `env:"MONGO_PASSWORD"`
	MongoDBPort     string `env:"MONGO_PORT"`
	// SeaweedFS configuration envs
	SeaweedFSHost      string `env:"SEAWEEDFS_HOST"`
	SeaweedFSFilerPort string `env:"SEAWEEDFS_FILER_PORT"`
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
	MultiTenantEnabled bool   `env:"MULTI_TENANT_ENABLED"`
	MultiTenantURL     string `env:"MULTI_TENANT_URL"`
	// TODO(multi-tenant): Wire MultiTenantEnvironment into RabbitMQ lazy consumer when full multi-tenant RabbitMQ is implemented.
	MultiTenantEnvironment              string `env:"MULTI_TENANT_ENVIRONMENT"`
	MultiTenantMaxTenantPools           int    `env:"MULTI_TENANT_MAX_TENANT_POOLS"`
	MultiTenantIdleTimeoutSec           int    `env:"MULTI_TENANT_IDLE_TIMEOUT_SEC"`
	MultiTenantCircuitBreakerThreshold  int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD"`
	MultiTenantCircuitBreakerTimeoutSec int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC"`
	MultiTenantServiceAPIKey            string `env:"MULTI_TENANT_SERVICE_API_KEY"`
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
	rabbitAdapter       rabbitmq.Adapter
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
		rabbitAdapter: rabbitmq.NewRabbitMQAdapterWithOptions(rabbitMQConnection, rabbitMQOptions),
		authClient:    middleware.NewAuthClient(cfg.AuthAddress, cfg.AuthEnabled, &authLogger),
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
	createConnectionCmd := connectionCommand.NewCreateConnection(repositories.connection, cryptoService)
	updateConnectionCmd := connectionCommand.NewUpdateConnection(repositories.connection, repositories.job, cryptoService)
	deleteConnectionCmd := connectionCommand.NewDeleteConnection(repositories.connection, repositories.job)
	getConnectionQuery := connectionQuery.NewGetConnection(repositories.connection)
	listConnectionsQuery := connectionQuery.NewListConnections(repositories.connection)
	testConnectionQuery := connectionQuery.NewTestConnection(repositories.connection, cryptoService, platformDependencies.connectionTestStore, datasourceFactory.NewDataSourceFromConnectionWithLogger(logger))
	validateSchemaQuery := connectionQuery.NewValidateSchema(repositories.connection, cryptoService, platformDependencies.schemaCache, datasourceFactory.NewDataSourceFromConnectionWithLogger(logger))
	getConnectionSchemaQuery := connectionQuery.NewGetConnectionSchema(
		repositories.connection,
		cryptoService,
		datasourceFactory.NewDataSourceFromConnectionWithLogger(logger),
	)

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
		platformDependencies.rabbitAdapter,
		cfg.RabbitMQGenerateReportQueue,
		datasourceFactory.NewDataSourceFromConnectionWithLogger(logger),
	)

	fetcherHandler := in2.NewFetcherHandler(
		createFetcherJobCmd,
		connectionQuery.NewGetJob(repositories.job),
	)

	// Init multi-tenant middleware (nil if disabled)
	tenantHandler, err := initMultiTenantMiddleware(cfg, logger)
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

	return &Service{
		Server: NewServer(cfg, httpApp, logger, telemetry, platformDependencies.licenseClient),
		Logger: logger,
	}, nil
}

// initMultiTenantMiddleware creates a TenantMiddleware Fiber handler if multi-tenant
// mode is enabled and configured. Returns nil when multi-tenant is disabled (single-tenant mode).
// The middleware resolves tenant-specific MongoDB connections from JWT claims.
//
// Per multi-tenant.md standards:
//   - Circuit breaker is MANDATORY for the Tenant Manager client
//   - Uses constant.ApplicationName and constant.ModuleManager for service/module identity
//   - WithMongoManager configures MongoDB connection pool management
func initMultiTenantMiddleware(cfg *Config, logger libLog.Logger) (fiber.Handler, error) {
	if !cfg.MultiTenantEnabled || cfg.MultiTenantURL == "" {
		return nil, nil
	}

	// Create Tenant Manager HTTP client with circuit breaker (MANDATORY per multi-tenant.md).
	// Default: 5 consecutive failures, 30s half-open timeout.
	var clientOpts []tmclient.ClientOption

	// Allow plaintext HTTP for local/dev environments where TLS is not configured.
	if strings.HasPrefix(cfg.MultiTenantURL, "http://") {
		clientOpts = append(clientOpts, tmclient.WithAllowInsecureHTTP())
	}

	clientOpts = append(clientOpts,
		tmclient.WithServiceAPIKey(cfg.MultiTenantServiceAPIKey),
	)

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

		return nil, wrapBootstrapError("create tenant manager client", err)
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

	// Create TenantMiddleware with MongoDB manager (fetcher uses MongoDB only for internal DB)
	tenantMid := tmmiddleware.NewTenantMiddleware(
		tmmiddleware.WithMongoManager(mongoManager),
	)

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Multi-tenant middleware initialized: url=%s, module=%s", cfg.MultiTenantURL, constant.ModuleManager))

	return tenantMid.WithTenantDB, nil
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

func wrapBootstrapError(action string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return nil
}
