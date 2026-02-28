package bootstrap

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
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
	"github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	cacheRepo "github.com/LerianStudio/fetcher/pkg/ports/cache"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/ratelimit"
	redisCache "github.com/LerianStudio/fetcher/pkg/redis"

	mgrRabbitMQ "github.com/LerianStudio/fetcher/components/manager/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/ports/messaging"
	"github.com/LerianStudio/lib-auth/v2/auth/middleware"
	"github.com/LerianStudio/lib-commons/v3/commons/log"
	mongoDB "github.com/LerianStudio/lib-commons/v3/commons/mongo"
	libOtel "github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v3/commons/rabbitmq"
	tmclient "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/client"
	tmmiddleware "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/middleware"
	tmmongo "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/mongo"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/rabbitmq"
	"github.com/LerianStudio/lib-commons/v3/commons/zap"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
	"github.com/gofiber/fiber/v2"
)

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
	MultiTenantEnabled                  bool   `env:"MULTI_TENANT_ENABLED"`
	MultiTenantURL                      string `env:"MULTI_TENANT_URL"`
	MultiTenantEnvironment              string `env:"MULTI_TENANT_ENVIRONMENT"`
	MultiTenantMaxTenantPools           int    `env:"MULTI_TENANT_MAX_TENANT_POOLS"`
	MultiTenantIdleTimeoutSec           int    `env:"MULTI_TENANT_IDLE_TIMEOUT_SEC"`
	MultiTenantCircuitBreakerThreshold  int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD"`
	MultiTenantCircuitBreakerTimeoutSec int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC"`
}

// InitServers initiate http and grpc servers.
func InitServers() (*Service, error) {
	cfg := &Config{}
	if err := pkg.SetConfigFromEnvVars(cfg); err != nil {
		return nil, fmt.Errorf("load environment configuration: %w", err)
	}

	ctx := context.Background()

	// Init Logger
	logger, err := zap.InitializeLoggerWithError()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	if cfg.MultiTenantEnabled {
		logger.Info("Multi-tenant mode ENABLED")
	} else {
		logger.Info("Running in SINGLE-TENANT MODE")
	}

	// Init OpenTelemetry telemetry
	telemetry, err := libOtel.InitializeTelemetryWithError(&libOtel.TelemetryConfig{
		LibraryName:               cfg.OtelLibraryName,
		ServiceName:               cfg.OtelServiceName,
		ServiceVersion:            cfg.OtelServiceVersion,
		DeploymentEnv:             cfg.OtelDeploymentEnv,
		CollectorExporterEndpoint: cfg.OtelColExporterEndpoint,
		EnableTelemetry:           cfg.EnableTelemetry,
		Logger:                    logger,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize telemetry: %w", err)
	}

	// Init MongoDB
	escapedUser := url.PathEscape(cfg.MongoDBUser)
	escapedPass := url.QueryEscape(cfg.MongoDBPassword)
	mongoSource := fmt.Sprintf("%s://%s:%s@%s:%s",
		cfg.MongoURI, escapedUser, escapedPass, cfg.MongoDBHost, cfg.MongoDBPort)

	mongoConnection := &mongoDB.MongoConnection{
		ConnectionStringSource: mongoSource,
		Database:               cfg.MongoDBName,
		Logger:                 logger,
	}

	connectionRepository, err := connection.NewConnectionMongoDBRepository(ctx, mongoConnection)
	if err != nil {
		return nil, fmt.Errorf("create connection repository: %w", err)
	}

	logger.Info("Ensuring MongoDB indexes exist for connections...")

	if errConnRepo := connectionRepository.EnsureIndexes(ctx); errConnRepo != nil {
		return nil, fmt.Errorf("ensure connection indexes: %w", errConnRepo)
	}

	// Init Job repository
	jobRepository, err := job.NewJobMongoDBRepository(ctx, mongoConnection)
	if err != nil {
		return nil, fmt.Errorf("create job repository: %w", err)
	}

	logger.Info("Ensuring MongoDB indexes exist for jobs...")

	if errJobRepo := jobRepository.EnsureIndexes(ctx); errJobRepo != nil {
		return nil, fmt.Errorf("ensure job indexes: %w", errJobRepo)
	}

	// Init key deriver for cryptographic key segregation
	masterKey, err := crypto.DecodeMasterKey(cfg.AppEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decode master encryption key: %w", err)
	}

	keyDeriver, err := crypto.NewHKDFKeyDeriver(masterKey)
	if err != nil {
		return nil, fmt.Errorf("initialize key deriver: %w", err)
	}

	logger.Info("Key derivation initialized successfully")

	// Init crypto service with derived credential key
	cryptoService, err := crypto.NewAESGCMService(keyDeriver.GetCredentialKey(), cfg.AppEncryptionKeyVersion)
	if err != nil {
		return nil, fmt.Errorf("initialize crypto service: %w", err)
	}

	// Init message signer for RabbitMQ with derived internal HMAC key
	cryptoWithInternalHMAC, err := crypto.NewHMACSigner(keyDeriver.GetInternalHMACKey(), crypto.SignatureVersion)
	if err != nil {
		return nil, fmt.Errorf("initialize internal message signer: %w", err)
	}

	// Init RabbitMQ - choose single-tenant or multi-tenant adapter based on configuration.
	// When MULTI_TENANT_ENABLED=true, use tmrabbitmq.Manager for per-tenant vhost isolation (Layer 1).
	// Single-tenant path remains unchanged.
	var rabbitMQPublisher messaging.MessagePublisher

	if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
		// Multi-tenant path: use tmrabbitmq.Manager for per-tenant vhost isolation
		logger.Info("Initializing RabbitMQ with multi-tenant vhost isolation")

		rmqManager := initMultiTenantRabbitMQManager(cfg, logger)
		rabbitMQPublisher = mgrRabbitMQ.NewMultiTenantPublisher(rmqManager, logger, telemetry)

		logger.Infof("Multi-tenant RabbitMQ publisher initialized: module=%s", constant.ModuleManager)
	} else {
		// Single-tenant path: use existing static RabbitMQ connection (unchanged)
		escapedUserRMQ := url.PathEscape(cfg.RabbitMQUser)
		escapedPassRMQ := url.PathEscape(cfg.RabbitMQPass)
		rabbitSource := fmt.Sprintf("%s://%s:%s@%s:%s", cfg.RabbitURI, escapedUserRMQ, escapedPassRMQ, cfg.RabbitMQHost, cfg.RabbitMQPortAMQP)

		rabbitMQConnection := &libRabbitmq.RabbitMQConnection{
			ConnectionStringSource: rabbitSource,
			HealthCheckURL:         cfg.RabbitMQHealthCheckURL,
			Host:                   cfg.RabbitMQHost,
			Port:                   cfg.RabbitMQPortHost,
			User:                   cfg.RabbitMQUser,
			Pass:                   cfg.RabbitMQPass,
			Queue:                  cfg.RabbitMQGenerateReportQueue,
			Logger:                 logger,
		}

		rabbitMQOptions := rabbitmq.DefaultOptions()
		rabbitMQOptions.Signer = cryptoWithInternalHMAC

		rabbitMQPublisher = rabbitmq.NewRabbitMQAdapterWithOptions(rabbitMQConnection, rabbitMQOptions)
	}

	// Init Auth middleware client
	authClient := middleware.NewAuthClient(cfg.AuthAddress, cfg.AuthEnabled, &logger)

	// Init License middleware client
	licenseClient := libLicense.NewLicenseClient(
		constant.ApplicationName,
		cfg.LicenseKey,
		cfg.OrganizationIDs,
		&logger,
	)

	// Init rate limiter for connection tests
	// 10 tokens per minute per connection
	connectionTestStore := ratelimit.New(10, time.Minute)

	// Init Redis and schema cache
	schemaCacheTTL := getSchemaCacheTTL(cfg.SchemaCacheTTLSeconds)

	var schemaCache cacheRepo.SchemaCacheRepository

	redisConfig := redisCache.RedisConfig{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       getRedisDB(cfg.RedisDB),
	}

	// Use graceful degradation - if Redis fails, use memory-only cache
	genericCache, errCache := redisCache.NewCacheWithFallback[model.DataSourceSchema](
		redisConfig,
		logger,
		schemaCacheTTL,
		"fetcher:schema:",
	)
	if errCache != nil {
		// This should never happen as NewCacheWithFallback handles Redis failures gracefully
		return nil, fmt.Errorf("initialize schema cache: %w", errCache)
	}

	schemaCache = cacheAdapter.NewSchemaCache(genericCache, schemaCacheTTL)

	// Init services and handlers
	createConnectionCmd := connectionCommand.NewCreateConnection(connectionRepository, cryptoService)
	updateConnectionCmd := connectionCommand.NewUpdateConnection(connectionRepository, jobRepository, cryptoService)
	deleteConnectionCmd := connectionCommand.NewDeleteConnection(connectionRepository, jobRepository)
	getConnectionQuery := connectionQuery.NewGetConnection(connectionRepository)
	listConnectionsQuery := connectionQuery.NewListConnections(connectionRepository)
	testConnectionQuery := connectionQuery.NewTestConnection(connectionRepository, cryptoService, connectionTestStore, datasourceFactory.NewDataSourceFromConnectionWithLogger(logger))
	validateSchemaQuery := connectionQuery.NewValidateSchema(connectionRepository, cryptoService, schemaCache)
	getConnectionSchemaQuery := connectionQuery.NewGetConnectionSchema(
		connectionRepository,
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

	// Init Migration services and handler
	assignConnectionCmd := connectionCommand.NewAssignConnection(connectionRepository)
	listUnassignedQuery := connectionQuery.NewListUnassignedConnections(connectionRepository)

	migrationHandler := in2.NewMigrationHandler(assignConnectionCmd, listUnassignedQuery)

	// Init Fetcher services and handler
	createFetcherJobCmd := connectionCommand.NewCreateFetcherJob(
		connectionRepository,
		jobRepository,
		cryptoService,
		rabbitMQPublisher,
		cfg.RabbitMQGenerateReportQueue,
		datasourceFactory.NewDataSourceFromConnectionWithLogger(logger),
	)

	getJobQuery := connectionQuery.NewGetJob(jobRepository)
	fetcherHandler := in2.NewFetcherHandler(createFetcherJobCmd, getJobQuery)

	// Init multi-tenant middleware (nil if disabled)
	tenantHandler := initMultiTenantMiddleware(cfg, logger)

	// Init HTTP server
	httpApp := in2.NewRoutes(logger, telemetry, authClient, licenseClient, connectionHandler, migrationHandler, fetcherHandler, tenantHandler)
	serverAPI := NewServer(cfg, httpApp, logger, telemetry, licenseClient)

	return &Service{
		Server: serverAPI,
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
func initMultiTenantMiddleware(cfg *Config, logger log.Logger) fiber.Handler {
	if !cfg.MultiTenantEnabled || cfg.MultiTenantURL == "" {
		return nil
	}

	// Create Tenant Manager HTTP client with circuit breaker (MANDATORY per multi-tenant.md).
	// Default: 5 consecutive failures, 30s half-open timeout.
	var clientOpts []tmclient.ClientOption
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

	tmClient := tmclient.NewClient(cfg.MultiTenantURL, logger, clientOpts...)

	// Create MongoDB Manager for tenant connection pool management
	var mongoOpts []tmmongo.Option

	mongoOpts = append(mongoOpts,
		tmmongo.WithModule(constant.ModuleManager),
		tmmongo.WithLogger(logger),
	)

	if cfg.MultiTenantMaxTenantPools > 0 {
		mongoOpts = append(mongoOpts, tmmongo.WithMaxTenantPools(cfg.MultiTenantMaxTenantPools))
	}

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

	logger.Infof("Multi-tenant middleware initialized: url=%s, module=%s", cfg.MultiTenantURL, constant.ModuleManager)

	return tenantMid.WithTenantDB
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

// initMultiTenantRabbitMQManager creates a RabbitMQ Manager for per-tenant vhost isolation.
// Each tenant has a dedicated RabbitMQ vhost with separate queues, exchanges, and connections.
//
// Per multi-tenant.md standards:
//   - Layer 1 (Vhost Isolation): tmrabbitmq.Manager → GetChannel(ctx, tenantID)
//   - Layer 2 (X-Tenant-ID Header): Injected by the publisher
func initMultiTenantRabbitMQManager(cfg *Config, logger log.Logger) *tmrabbitmq.Manager {
	// Create Tenant Manager HTTP client with circuit breaker (MANDATORY per multi-tenant.md)
	var clientOpts []tmclient.ClientOption
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

	tmClient := tmclient.NewClient(cfg.MultiTenantURL, logger, clientOpts...)

	// Create RabbitMQ Manager for per-tenant vhost connections
	var rmqOpts []tmrabbitmq.Option

	rmqOpts = append(rmqOpts,
		tmrabbitmq.WithModule(constant.ModuleManager),
		tmrabbitmq.WithLogger(logger),
	)

	if cfg.MultiTenantMaxTenantPools > 0 {
		rmqOpts = append(rmqOpts, tmrabbitmq.WithMaxTenantPools(cfg.MultiTenantMaxTenantPools))
	}

	if cfg.MultiTenantIdleTimeoutSec > 0 {
		rmqOpts = append(rmqOpts, tmrabbitmq.WithIdleTimeout(
			time.Duration(cfg.MultiTenantIdleTimeoutSec)*time.Second,
		))
	}

	return tmrabbitmq.NewManager(tmClient, constant.ApplicationName, rmqOpts...)
}
