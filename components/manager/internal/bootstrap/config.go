package bootstrap

import (
	"context"
	"fmt"
	"log"
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

	"github.com/LerianStudio/lib-auth/v2/auth/middleware"
	mongoDB "github.com/LerianStudio/lib-commons/v2/commons/mongo"
	libOtel "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v2/commons/rabbitmq"
	"github.com/LerianStudio/lib-commons/v2/commons/zap"
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

// InitServers initiate http and grpc servers.
func InitServers() *Service {
	cfg := &Config{}
	if err := pkg.SetConfigFromEnvVars(cfg); err != nil {
		// log.Fatalf is used here because this runs before the structured logger
		// (zap) is initialized. Returning an error is not possible since InitServers
		// is called from main() and the application cannot start without valid config.
		log.Fatalf("Failed to load configuration from environment variables: %v", err)
	}

	ctx := context.Background()

	// Init Logger
	logger := zap.InitializeLogger()

	// Init OpenTelemetry telemetry
	telemetry := libOtel.InitializeTelemetry(&libOtel.TelemetryConfig{
		LibraryName:               cfg.OtelLibraryName,
		ServiceName:               cfg.OtelServiceName,
		ServiceVersion:            cfg.OtelServiceVersion,
		DeploymentEnv:             cfg.OtelDeploymentEnv,
		CollectorExporterEndpoint: cfg.OtelColExporterEndpoint,
		EnableTelemetry:           cfg.EnableTelemetry,
		Logger:                    logger,
	})

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
		logger.Fatalf("Failed to create MongoDB repository: %v", err)
	}

	logger.Info("Ensuring MongoDB indexes exist for connections...")

	if errConnRepo := connectionRepository.EnsureIndexes(ctx); errConnRepo != nil {
		logger.Fatalf("Failed to ensure MongoDB indexes: %v", errConnRepo)
	}

	// Init Job repository
	jobRepository, err := job.NewJobMongoDBRepository(ctx, mongoConnection)
	if err != nil {
		logger.Fatalf("Failed to create Job MongoDB repository: %v", err)
	}

	logger.Info("Ensuring MongoDB indexes exist for jobs...")

	if errJobRepo := jobRepository.EnsureIndexes(ctx); errJobRepo != nil {
		logger.Fatalf("Failed to ensure Job indexes: %v", errJobRepo)
	}

	// Init key deriver for cryptographic key segregation
	masterKey, err := crypto.DecodeMasterKey(cfg.AppEncryptionKey)
	if err != nil {
		logger.Fatalf("Failed to decode master encryption key: %v", err)
	}

	keyDeriver, err := crypto.NewHKDFKeyDeriver(masterKey)
	if err != nil {
		logger.Fatalf("Failed to initialize key deriver: %v", err)
	}

	logger.Info("Key derivation initialized successfully")

	// Init crypto service with derived credential key
	cryptoService, err := crypto.NewAESGCMService(keyDeriver.GetCredentialKey(), cfg.AppEncryptionKeyVersion)
	if err != nil {
		logger.Fatalf("Failed to initialize crypto service: %v", err)
	}

	// Init message signer for RabbitMQ with derived internal HMAC key
	cryptoWithInternalHMAC, err := crypto.NewHMACSigner(keyDeriver.GetInternalHMACKey(), crypto.SignatureVersion)
	if err != nil {
		logger.Fatalf("Failed to initialize message signer: %v", err)
	}

	// Init RabbitMQ
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

	rabbitMQAdapter := rabbitmq.NewRabbitMQAdapterWithOptions(rabbitMQConnection, rabbitMQOptions)

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
		logger.Fatalf("Failed to initialize cache: %v", errCache)
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
		rabbitMQAdapter,
		cfg.RabbitMQGenerateReportQueue,
		datasourceFactory.NewDataSourceFromConnectionWithLogger(logger),
	)

	getJobQuery := connectionQuery.NewGetJob(jobRepository)
	fetcherHandler := in2.NewFetcherHandler(createFetcherJobCmd, getJobQuery)

	// Init HTTP server
	httpApp := in2.NewRoutes(logger, telemetry, authClient, licenseClient, connectionHandler, migrationHandler, fetcherHandler)
	serverAPI := NewServer(cfg, httpApp, logger, telemetry, licenseClient)

	return &Service{
		Server: serverAPI,
		Logger: logger,
	}
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
