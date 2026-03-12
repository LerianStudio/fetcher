package bootstrap

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	in2 "github.com/LerianStudio/fetcher/components/manager/internal/adapters/http/in"
	connectionCommand "github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	connectionQuery "github.com/LerianStudio/fetcher/components/manager/internal/services/query"

	cacheRepo "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	datasourceFactory "github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	"github.com/LerianStudio/fetcher/pkg/mongodb/product"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/ratelimit"
	redisCache "github.com/LerianStudio/fetcher/pkg/redis"

	"github.com/LerianStudio/lib-auth/v2/auth/middleware"
	libZapV2 "github.com/LerianStudio/lib-commons/v2/commons/zap"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libMongo "github.com/LerianStudio/lib-commons/v4/commons/mongo"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v4/commons/rabbitmq"
	"github.com/LerianStudio/lib-commons/v4/commons/zap"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
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
}

type managerRepositories struct {
	connection *connection.ConnectionMongoDBRepository
	job        *job.JobMongoDBRepository
	product    *product.ProductMongoDBRepository
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
	newProductRepository    = product.NewProductMongoDBRepository
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
	), nil
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

	connectionRepository, err := newConnectionRepository(mongoConnection, cfg.MongoDBName)
	if err != nil {
		return nil, wrapBootstrapError("create MongoDB connection repository", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "Ensuring MongoDB indexes exist for connections...")

	if err := connectionRepository.EnsureIndexes(ctx); err != nil {
		return nil, wrapBootstrapError("ensure MongoDB connection indexes", err)
	}

	jobRepository, err := newJobRepository(mongoConnection, cfg.MongoDBName)
	if err != nil {
		return nil, wrapBootstrapError("create MongoDB job repository", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "Ensuring MongoDB indexes exist for jobs...")

	if err := jobRepository.EnsureIndexes(ctx); err != nil {
		return nil, wrapBootstrapError("ensure MongoDB job indexes", err)
	}

	productRepository, err := newProductRepository(mongoConnection, cfg.MongoDBName)
	if err != nil {
		return nil, wrapBootstrapError("create MongoDB product repository", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "Ensuring MongoDB indexes exist for products...")

	if err := productRepository.EnsureIndexes(ctx); err != nil {
		return nil, wrapBootstrapError("ensure MongoDB product indexes", err)
	}

	return &managerRepositories{
		connection: connectionRepository,
		job:        jobRepository,
		product:    productRepository,
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

	authLogger := libZapV2.InitializeLogger()
	licenseLogger := libZapV2.InitializeLogger()
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
		schemaCache:         cacheRepo.NewSchemaCache(genericCache, schemaCacheTTL),
	}, nil
}

func assembleService(
	cfg *Config,
	logger libLog.Logger,
	telemetry *libOtel.Telemetry,
	repositories *managerRepositories,
	cryptoService *crypto.AESGCMService,
	platformDependencies *managerPlatformDependencies,
) *Service {
	createConnectionCmd := connectionCommand.NewCreateConnection(repositories.connection, repositories.product, cryptoService)
	updateConnectionCmd := connectionCommand.NewUpdateConnection(repositories.connection, repositories.job, cryptoService)
	deleteConnectionCmd := connectionCommand.NewDeleteConnection(repositories.connection, repositories.job)
	getConnectionQuery := connectionQuery.NewGetConnection(repositories.connection)
	listConnectionsQuery := connectionQuery.NewListConnections(repositories.connection, repositories.product)
	testConnectionQuery := connectionQuery.NewTestConnection(repositories.connection, cryptoService, platformDependencies.connectionTestStore)
	validateSchemaQuery := connectionQuery.NewValidateSchema(repositories.connection, cryptoService, platformDependencies.schemaCache)
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

	productHandler := in2.NewProductHandler(
		connectionCommand.NewCreateProduct(repositories.product),
		connectionCommand.NewUpdateProduct(repositories.product),
		connectionCommand.NewDeleteProduct(repositories.product, repositories.connection),
		connectionQuery.NewGetProduct(repositories.product),
		connectionQuery.NewListProducts(repositories.product),
	)

	migrationHandler := in2.NewMigrationHandler(
		connectionCommand.NewAssignConnection(repositories.connection, repositories.product),
		connectionQuery.NewListUnassignedConnections(repositories.connection),
	)

	fetcherHandler := in2.NewFetcherHandler(
		connectionCommand.NewCreateFetcherJob(
			repositories.connection,
			repositories.job,
			repositories.product,
			cryptoService,
			platformDependencies.rabbitAdapter,
			cfg.RabbitMQGenerateReportQueue,
		),
		connectionQuery.NewGetJob(repositories.job),
	)

	httpApp := in2.NewRoutes(
		logger,
		telemetry,
		platformDependencies.authClient,
		platformDependencies.licenseClient,
		connectionHandler,
		productHandler,
		migrationHandler,
		fetcherHandler,
	)

	return &Service{
		Server: NewServer(cfg, httpApp, logger, telemetry, platformDependencies.licenseClient),
		Logger: logger,
	}
}

func buildMongoSource(cfg *Config) string {
	escapedUser := url.PathEscape(cfg.MongoDBUser)
	escapedPass := url.QueryEscape(cfg.MongoDBPassword)

	return fmt.Sprintf("%s://%s:%s@%s:%s", cfg.MongoURI, escapedUser, escapedPass, cfg.MongoDBHost, cfg.MongoDBPort)
}

func buildRabbitMQSource(cfg *Config) string {
	escapedUser := url.PathEscape(cfg.RabbitMQUser)
	escapedPass := url.PathEscape(cfg.RabbitMQPass)

	return fmt.Sprintf("%s://%s:%s@%s:%s", cfg.RabbitURI, escapedUser, escapedPass, cfg.RabbitMQHost, cfg.RabbitMQPortAMQP)
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
