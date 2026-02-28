package bootstrap

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/components/worker/internal/services"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	mongoDB "github.com/LerianStudio/lib-commons/v3/commons/mongo"

	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	pkgStorage "github.com/LerianStudio/fetcher/pkg/storage"

	libCommons "github.com/LerianStudio/lib-commons/v3/commons"
	"github.com/LerianStudio/lib-commons/v3/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	libRabbitMQ "github.com/LerianStudio/lib-commons/v3/commons/rabbitmq"
	tmclient "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/client"
	tmconsumer "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/consumer"
	tmmongo "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/mongo"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/rabbitmq"
	libZap "github.com/LerianStudio/lib-commons/v3/commons/zap"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
	"github.com/redis/go-redis/v9"
)

// Config holds the application's configurable parameters read from environment variables.
type Config struct {
	EnvName  string `env:"ENV_NAME"`
	LogLevel string `env:"LOG_LEVEL"`
	// RabbitMQ envs
	RabbitURI                   string `env:"RABBITMQ_URI"`
	RabbitMQHost                string `env:"RABBITMQ_HOST"`
	RabbitMQPortHost            string `env:"RABBITMQ_PORT_HOST"`
	RabbitMQPortAMQP            string `env:"RABBITMQ_PORT_AMQP"`
	RabbitMQUser                string `env:"RABBITMQ_DEFAULT_USER"`
	RabbitMQPass                string `env:"RABBITMQ_DEFAULT_PASS"`
	RabbitMQNumWorkers          int    `env:"RABBITMQ_NUMBERS_OF_WORKERS"`
	RabbitMQHealthCheckURL      string `env:"RABBITMQ_HEALTH_CHECK_URL"`
	RabbitMQGenerateReportQueue string `env:"RABBITMQ_FETCHER_WORK_QUEUE"`
	RabbitMQJobEventsExchange   string `env:"RABBITMQ_JOB_EVENTS_EXCHANGE"`
	// Otel Collector configurations
	OtelServiceName         string `env:"OTEL_RESOURCE_SERVICE_NAME"`
	OtelLibraryName         string `env:"OTEL_LIBRARY_NAME"`
	OtelServiceVersion      string `env:"OTEL_RESOURCE_SERVICE_VERSION"`
	OtelDeploymentEnv       string `env:"OTEL_RESOURCE_DEPLOYMENT_ENVIRONMENT"`
	OtelColExporterEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	EnableTelemetry         bool   `env:"ENABLE_TELEMETRY"`
	// SeaweedFS configuration envs
	SeaweedFSHost      string `env:"SEAWEEDFS_HOST"`
	SeaweedFSFilerPort string `env:"SEAWEEDFS_FILER_PORT"`
	SeaweedFSTTL       string `env:"SEAWEEDFS_TTL"`
	// Storage provider selection ("seaweedfs" or "s3", defaults to "seaweedfs")
	StorageProvider string `env:"STORAGE_PROVIDER"`
	// S3-compatible object storage configuration (used when STORAGE_PROVIDER=s3)
	// SSL is controlled by the URL scheme of ObjectStorageEndpoint.
	ObjectStorageEndpoint     string `env:"OBJECT_STORAGE_ENDPOINT"`
	ObjectStorageRegion       string `env:"OBJECT_STORAGE_REGION"`
	ObjectStorageBucket       string `env:"OBJECT_STORAGE_BUCKET"`
	ObjectStorageKeyPrefix    string `env:"OBJECT_STORAGE_KEY_PREFIX"`
	ObjectStorageAccessKeyID  string `env:"OBJECT_STORAGE_ACCESS_KEY_ID"`
	ObjectStorageSecretKey    string `env:"OBJECT_STORAGE_SECRET_KEY"`
	ObjectStorageUsePathStyle bool   `env:"OBJECT_STORAGE_USE_PATH_STYLE"`
	// OBJECT_STORAGE_DISABLE_SSL omitted — SSL controlled by endpoint URL scheme.
	// MongoDB
	MongoURI        string `env:"MONGO_URI"`
	MongoDBHost     string `env:"MONGO_HOST"`
	MongoDBName     string `env:"MONGO_NAME"`
	MongoDBUser     string `env:"MONGO_USER"`
	MongoDBPassword string `env:"MONGO_PASSWORD"`
	MongoDBPort     string `env:"MONGO_PORT"`
	MaxPoolSize     int    `env:"MONGO_MAX_POOL_SIZE"`
	// License configuration envs
	LicenseKey      string `env:"LICENSE_KEY"`
	OrganizationIDs string `env:"ORGANIZATION_IDS"`
	// Encryption
	AppEncryptionKey        string `env:"APP_ENC_KEY"`
	AppEncryptionKeyVersion string `env:"APP_ENC_KEY_VERSION"`
	// SeaweedFS encryption keys
	CryptoEncryptFileStorage string `env:"CRYPTO_ENCRYPT_FILE_STORAGE"`
	CryptoHashFileStorage    string `env:"CRYPTO_HASH_SECRET_KEY_FILE_STORAGE"`
	// CRM plugin encryption keys
	CryptoEncryptSecretKeyPluginCRM string `env:"CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM"`
	CryptoHashSecretKeyPluginCRM    string `env:"CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM"`
	// Multi-Tenant configuration
	MultiTenantEnabled                  bool   `env:"MULTI_TENANT_ENABLED"`
	MultiTenantURL                      string `env:"MULTI_TENANT_URL"`
	MultiTenantEnvironment              string `env:"MULTI_TENANT_ENVIRONMENT"`
	MultiTenantMaxTenantPools           int    `env:"MULTI_TENANT_MAX_TENANT_POOLS"`
	MultiTenantIdleTimeoutSec           int    `env:"MULTI_TENANT_IDLE_TIMEOUT_SEC"`
	MultiTenantCircuitBreakerThreshold  int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD"`
	MultiTenantCircuitBreakerTimeoutSec int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC"`
	// Redis configuration for multi-tenant consumer (tenant discovery)
	RedisHost     string `env:"REDIS_HOST"`
	RedisPort     string `env:"REDIS_PORT"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB       int    `env:"REDIS_DB"`
}

// InitWorker initializes and configures the application's dependencies and returns the Service instance.
func InitWorker() (*Service, error) {
	cfg := &Config{}
	if err := libCommons.SetConfigFromEnvVars(cfg); err != nil {
		return nil, fmt.Errorf("load environment configuration: %w", err)
	}

	ctx := context.Background()

	logger, err := libZap.InitializeLoggerWithError()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	if cfg.MultiTenantEnabled {
		logger.Info("Multi-tenant mode ENABLED")
	} else {
		logger.Info("Running in SINGLE-TENANT MODE")
	}

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

	// Init rabbitmq connection for consumer
	// Consumer and Publisher use SEPARATE connections to avoid channel interference.
	// A shared connection causes both to share the same AMQP channel, leading to issues
	// when one invalidates/closes the channel (affects the other).
	// URL-encode credentials to handle special characters (@ : / etc.)
	escapedUserRMQ := url.PathEscape(cfg.RabbitMQUser)
	escapedPassRMQ := url.QueryEscape(cfg.RabbitMQPass)
	rabbitSource := fmt.Sprintf("%s://%s:%s@%s:%s",
		cfg.RabbitURI, escapedUserRMQ, escapedPassRMQ, cfg.RabbitMQHost, cfg.RabbitMQPortAMQP)
	consumerConnection := &libRabbitMQ.RabbitMQConnection{
		ConnectionStringSource: rabbitSource,
		HealthCheckURL:         cfg.RabbitMQHealthCheckURL,
		Host:                   cfg.RabbitMQHost,
		Port:                   cfg.RabbitMQPortHost,
		User:                   cfg.RabbitMQUser,
		Pass:                   cfg.RabbitMQPass,
		Queue:                  cfg.RabbitMQGenerateReportQueue,
		Logger:                 logger,
	}

	// Separate connection for Publisher to isolate channel lifecycle
	publisherConnection := &libRabbitMQ.RabbitMQConnection{
		ConnectionStringSource: rabbitSource,
		HealthCheckURL:         cfg.RabbitMQHealthCheckURL,
		Host:                   cfg.RabbitMQHost,
		Port:                   cfg.RabbitMQPortHost,
		User:                   cfg.RabbitMQUser,
		Pass:                   cfg.RabbitMQPass,
		Logger:                 logger,
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
	cryptoService, errCrypto := crypto.NewAESGCMService(keyDeriver.GetCredentialKey(), cfg.AppEncryptionKeyVersion)
	if errCrypto != nil {
		return nil, fmt.Errorf("initialize crypto service: %w", errCrypto)
	}

	// Init message signer for RabbitMQ with derived internal HMAC key
	cryptoWithInternalHMAC, errSigner := crypto.NewHMACSigner(keyDeriver.GetInternalHMACKey(), crypto.SignatureVersion)
	if errSigner != nil {
		return nil, fmt.Errorf("initialize internal message signer: %w", errSigner)
	}

	// Init document signer for external verification with derived external HMAC key
	cryptoWithExternalHMAC, errSigner := crypto.NewHMACSigner(keyDeriver.GetExternalHMACKey(), crypto.SignatureVersion)
	if errSigner != nil {
		return nil, fmt.Errorf("initialize external document signer: %w", errSigner)
	}

	// Initialize RabbitMQ consumer and publisher with separate connections.
	// When MULTI_TENANT_ENABLED=true, use tmrabbitmq.Manager and tmconsumer.MultiTenantConsumer
	// for per-tenant vhost isolation (Layer 1). Single-tenant path remains unchanged.
	var consumerRoutes *rabbitmq.ConsumerRoutes
	var multiTenantConsumer *tmconsumer.MultiTenantConsumer
	var rmqManager *tmrabbitmq.Manager
	var publisherRoutes rabbitmq.PublisherRepository

	if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
		// Multi-tenant path: use tmrabbitmq.Manager for per-tenant vhost isolation
		logger.Info("Initializing RabbitMQ with multi-tenant vhost isolation")

		// Create Tenant Manager client (reusing same pattern as MongoDB manager)
		tmClient := initTenantManagerClient(cfg, logger)

		// Create RabbitMQ Manager for per-tenant vhost connections
		rmqManager = initMultiTenantRabbitMQManager(tmClient, cfg, logger)

		// Create Redis client for tenant discovery
		redisClient := initRedisClient(cfg, logger)

		// Create multi-tenant consumer configuration
		mtConfig := tmconsumer.MultiTenantConfig{
			SyncInterval:     30 * time.Second,
			PrefetchCount:    cfg.RabbitMQNumWorkers,
			MultiTenantURL:   cfg.MultiTenantURL,
			Service:          constant.ApplicationName,
			Environment:      cfg.MultiTenantEnvironment,
			DiscoveryTimeout: 500 * time.Millisecond,
		}

		// Create multi-tenant consumer with MongoDB manager for connection cleanup
		mongoManager := initMultiTenantMongoManager(cfg, logger)
		multiTenantConsumer = tmconsumer.NewMultiTenantConsumer(
			rmqManager,
			redisClient,
			mtConfig,
			logger,
			tmconsumer.WithMongoManager(mongoManager),
		)

		// Create multi-tenant publisher
		publisherRoutes = rabbitmq.NewPublisherRoutesMultiTenant(rmqManager, logger, telemetry, cryptoWithExternalHMAC)

		logger.Infof("Multi-tenant RabbitMQ initialized: environment=%s, service=%s", cfg.MultiTenantEnvironment, constant.ApplicationName)
	} else {
		// Single-tenant path: use existing static RabbitMQ connections (unchanged)
		var errRoutes error
		consumerRoutes, errRoutes = rabbitmq.NewConsumerRoutes(consumerConnection, cfg.RabbitMQNumWorkers, logger, telemetry, cryptoWithInternalHMAC, cfg.EnvName)
		if errRoutes != nil {
			return nil, fmt.Errorf("initialize consumer routes: %w", errRoutes)
		}

		publisherRoutes = rabbitmq.NewPublisherRoutes(publisherConnection, logger, telemetry, cryptoWithExternalHMAC)
	}

	// Initialize storage repository (SeaweedFS or S3) via factory
	storageProvider := cfg.StorageProvider
	if storageProvider == "" {
		storageProvider = pkgStorage.ProviderSeaweedFS
	}

	storageRepository, err := pkgStorage.NewRepository(ctx, pkgStorage.ProviderConfig{
		Provider:          storageProvider,
		SeaweedFSEndpoint: fmt.Sprintf("http://%s:%s", cfg.SeaweedFSHost, cfg.SeaweedFSFilerPort),
		Bucket:            constant.ExternalDataBucketName,
		S3Endpoint:        cfg.ObjectStorageEndpoint,
		S3Region:          cfg.ObjectStorageRegion,
		S3Bucket:          cfg.ObjectStorageBucket,
		S3KeyPrefix:       cfg.ObjectStorageKeyPrefix,
		S3AccessKeyID:     cfg.ObjectStorageAccessKeyID,
		S3SecretAccessKey: cfg.ObjectStorageSecretKey,
		S3UsePathStyle:    cfg.ObjectStorageUsePathStyle,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize storage repository: %w", err)
	}

	logger.Infof("Storage initialized with provider: %s", storageProvider)

	// Init mongo DB connection
	escapedPass := url.QueryEscape(cfg.MongoDBPassword)
	mongoSource := fmt.Sprintf("%s://%s:%s@%s:%s",
		cfg.MongoURI, cfg.MongoDBUser, escapedPass, cfg.MongoDBHost, cfg.MongoDBPort)

	if cfg.MaxPoolSize <= 0 {
		cfg.MaxPoolSize = 100
	}

	mongoConnection := &mongoDB.MongoConnection{
		ConnectionStringSource: mongoSource,
		Database:               cfg.MongoDBName,
		Logger:                 logger,
		MaxPoolSize:            uint64(cfg.MaxPoolSize),
	}

	// Initialize MongoDB repositories
	jobRepository, errJobRepo := job.NewJobMongoDBRepository(ctx, mongoConnection)
	if errJobRepo != nil {
		return nil, fmt.Errorf("initialize job repository: %w", errJobRepo)
	}

	connectionRepository, errConnectRepo := connection.NewConnectionMongoDBRepository(ctx, mongoConnection)
	if errConnectRepo != nil {
		return nil, fmt.Errorf("initialize connection repository: %w", errConnectRepo)
	}

	service := &services.UseCase{
		ExternalDataStorage:  storageRepository,
		JobRepository:        jobRepository,
		ConnectionRepository: connectionRepository,
		Cryptor:              cryptoService,
		DocumentSigner:       cryptoWithExternalHMAC,
		FileTTL:              cfg.SeaweedFSTTL,
		RabbitMQPublisher:    publisherRoutes,
		JobEventsExchange:    cfg.RabbitMQJobEventsExchange,
	}
	service.SetStorageSecrets(cfg.CryptoEncryptFileStorage, cfg.CryptoHashFileStorage)
	service.SetCRMSecrets(cfg.CryptoEncryptSecretKeyPluginCRM, cfg.CryptoHashSecretKeyPluginCRM)
	service.SetDataSourceFactory(datasource.NewDataSourceFromConnectionWithLogger(logger))

	if cfg.SeaweedFSTTL != "" {
		logger.Infof("Reports will expire after: %s", cfg.SeaweedFSTTL)
	} else {
		logger.Infof("Reports will be stored permanently (no TTL)")
	}

	licenseClient := libLicense.NewLicenseClient(
		constant.ApplicationName,
		cfg.LicenseKey,
		cfg.OrganizationIDs,
		&logger,
	)

	// Create consumer abstraction based on multi-tenant mode
	var consumer Consumer
	if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
		// Multi-tenant: use MultiTenantConsumerAdapter that wraps tmconsumer.MultiTenantConsumer
		mongoManager := initMultiTenantMongoManager(cfg, logger)
		consumer = NewMultiTenantConsumerAdapter(
			multiTenantConsumer,
			service,
			cfg.RabbitMQGenerateReportQueue,
			mongoManager,
			rmqManager,
			logger,
		)
	} else {
		// Single-tenant: use existing MultiQueueConsumer (unchanged)
		mongoManager := initMultiTenantMongoManager(cfg, logger)
		consumer = NewMultiQueueConsumer(consumerRoutes, service, cfg.RabbitMQGenerateReportQueue, mongoManager)
	}

	return &Service{
		Consumer:        consumer,
		Logger:          logger,
		licenseShutdown: licenseClient.GetLicenseManagerShutdown(),
	}, nil
}

// initMultiTenantMongoManager creates a MongoDB Manager for tenant connection pool management
// if multi-tenant mode is enabled and configured. Returns nil when multi-tenant is disabled.
// The Worker does not need HTTP middleware (no HTTP server) -- tenant context comes from
// RabbitMQ message headers (to be implemented in Gate 6).
//
// Per multi-tenant.md standards:
//   - Circuit breaker is MANDATORY for the Tenant Manager client
//   - Uses constant.ApplicationName and constant.ModuleWorker for service/module identity
//   - WithMongoManager configures MongoDB connection pool management
func initMultiTenantMongoManager(cfg *Config, logger log.Logger) *tmmongo.Manager {
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
		tmmongo.WithModule(constant.ModuleWorker),
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

	logger.Infof("Multi-tenant MongoDB manager initialized: url=%s, module=%s", cfg.MultiTenantURL, constant.ModuleWorker)

	return mongoManager
}

// initTenantManagerClient creates a Tenant Manager HTTP client with circuit breaker.
// Used by both MongoDB and RabbitMQ managers.
func initTenantManagerClient(cfg *Config, logger log.Logger) *tmclient.Client {
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

	return tmclient.NewClient(cfg.MultiTenantURL, logger, clientOpts...)
}

// initMultiTenantRabbitMQManager creates a RabbitMQ Manager for per-tenant vhost isolation.
// Each tenant has a dedicated RabbitMQ vhost with separate queues, exchanges, and connections.
//
// Per multi-tenant.md standards:
//   - Layer 1 (Vhost Isolation): tmrabbitmq.Manager → GetChannel(ctx, tenantID)
//   - Layer 2 (X-Tenant-ID Header): Already implemented in publisher/consumer
func initMultiTenantRabbitMQManager(tmClient *tmclient.Client, cfg *Config, logger log.Logger) *tmrabbitmq.Manager {
	var rmqOpts []tmrabbitmq.Option

	rmqOpts = append(rmqOpts,
		tmrabbitmq.WithModule(constant.ModuleWorker),
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

	rmqManager := tmrabbitmq.NewManager(tmClient, constant.ApplicationName, rmqOpts...)

	logger.Infof("Multi-tenant RabbitMQ manager initialized: module=%s", constant.ModuleWorker)

	return rmqManager
}

// initRedisClient creates a Redis client for multi-tenant consumer tenant discovery.
// The MultiTenantConsumer uses Redis to discover active tenants without connecting
// to RabbitMQ at startup (lazy mode).
func initRedisClient(cfg *Config, logger log.Logger) redis.UniversalClient {
	redisAddr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	logger.Infof("Redis client initialized for multi-tenant consumer: addr=%s", redisAddr)

	return client
}
