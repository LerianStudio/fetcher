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
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	mongoDB "github.com/LerianStudio/lib-commons/v3/commons/mongo"

	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	portStorage "github.com/LerianStudio/fetcher/pkg/ports/storage"
	pkgStorage "github.com/LerianStudio/fetcher/pkg/storage"

	libCommons "github.com/LerianStudio/lib-commons/v3/commons"
	"github.com/LerianStudio/lib-commons/v3/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	libRabbitMQ "github.com/LerianStudio/lib-commons/v3/commons/rabbitmq"
	libRedis "github.com/LerianStudio/lib-commons/v3/commons/redis"
	tmclient "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/client"
	tmconsumer "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/consumer"
	tmmongo "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/mongo"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/rabbitmq"
	libZap "github.com/LerianStudio/lib-commons/v3/commons/zap"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
)

// defaultMaxTenantPools is the fallback soft limit for tenant connection pools
// when MULTI_TENANT_MAX_TENANT_POOLS is unset or zero. This prevents unbounded
// pool growth. The value can be overridden via the environment variable.
const defaultMaxTenantPools = 100

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
	// RabbitMQ multi-tenant consumer tuning (active when MULTI_TENANT_ENABLED=true)
	RabbitMQMultiTenantSyncInterval     int `env:"RABBITMQ_MULTI_TENANT_SYNC_INTERVAL"`     // Stored in seconds; default 30
	RabbitMQMultiTenantDiscoveryTimeout int `env:"RABBITMQ_MULTI_TENANT_DISCOVERY_TIMEOUT"` // Stored in milliseconds; default 500
	// Redis/Valkey configuration (required for multi-tenant tenant discovery cache)
	RedisHost     string `env:"REDIS_HOST"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB       int    `env:"REDIS_DB"`
	RedisProtocol int    `env:"REDIS_PROTOCOL"`
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

	if err := validateMultiTenantConfig(cfg, logger); err != nil {
		return nil, err
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

	cryptoService, cryptoWithExternalHMAC, keyDeriver, err := initCryptoServices(cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Key derivation initialized successfully")

	storageRepository, err := initStorageRepository(ctx, cfg)
	if err != nil {
		return nil, err
	}

	logger.Infof("Storage initialized with provider: %s", cfg.StorageProvider)

	mongoConnection := initMongoConnection(cfg, logger)

	// Initialize multi-tenant managers (nil when MULTI_TENANT_ENABLED=false)
	mongoManager, rabbitMQManager := initMultiTenantManagers(cfg, logger)

	// Wrap the MongoDB connection to implement tmcore.MultiTenantChecker.
	// When multi-tenant mode is enabled, this causes tmcore.ResolveMongo to return
	// ErrTenantContextRequired instead of silently falling back to the default DB
	// when no tenant context is present.
	mongoProvider := mongodb.NewMultiTenantMongoProvider(mongoConnection, cfg.MultiTenantEnabled)

	// Initialize MongoDB repositories
	jobRepository, errJobRepo := job.NewJobMongoDBRepository(ctx, mongoProvider, mongoConnection.Database)
	if errJobRepo != nil {
		return nil, fmt.Errorf("initialize job repository: %w", errJobRepo)
	}

	connectionRepository, errConnectRepo := connection.NewConnectionMongoDBRepository(ctx, mongoProvider, mongoConnection.Database)
	if errConnectRepo != nil {
		return nil, fmt.Errorf("initialize connection repository: %w", errConnectRepo)
	}

	// Create service use case (publisher set below per mode)
	service := &services.UseCase{
		ExternalDataStorage:  storageRepository,
		JobRepository:        jobRepository,
		ConnectionRepository: connectionRepository,
		Cryptor:              cryptoService,
		DocumentSigner:       cryptoWithExternalHMAC,
		FileTTL:              cfg.SeaweedFSTTL,
		JobEventsExchange:    cfg.RabbitMQJobEventsExchange,
	}
	service.SetStorageSecrets(cfg.CryptoEncryptFileStorage, cfg.CryptoHashFileStorage)
	service.SetCRMSecrets(cfg.CryptoEncryptSecretKeyPluginCRM, cfg.CryptoHashSecretKeyPluginCRM)
	service.SetDataSourceFactory(datasource.NewDataSourceFromConnectionWithLogger(logger))

	logFileTTL(logger, cfg)

	licenseClient := libLicense.NewLicenseClient(
		constant.ApplicationName,
		cfg.LicenseKey,
		cfg.OrganizationIDs,
		&logger,
	)

	// Branch: multi-tenant mode uses tmconsumer.MultiTenantConsumer with per-tenant vhosts
	// Single-tenant mode uses existing ConsumerRoutes with static RabbitMQ connection
	if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
		mtConsumer, mtCleanup, mtErr := initMultiTenantConsumer(ctx, cfg, logger, mongoManager, rabbitMQManager)
		if mtErr != nil {
			return nil, mtErr
		}

		// Use shared RabbitMQ manager for publisher (same pool as consumer)
		publisherRoutes := rabbitmq.NewPublisherRoutesMultiTenant(
			newRabbitMQManagerAdapter(rabbitMQManager),
			logger,
			telemetry,
		)

		service.RabbitMQPublisher = publisherRoutes

		multiQueueConsumer := NewMultiQueueConsumerMultiTenant(mtConsumer, service, cfg.RabbitMQGenerateReportQueue, logger, mongoManager)

		return &Service{
			MultiQueueConsumer: multiQueueConsumer,
			Logger:             logger,
			licenseShutdown:    licenseClient.GetLicenseManagerShutdown(),
			mtCleanup:          mtCleanup,
		}, nil
	}

	// Single-tenant mode: use existing ConsumerRoutes with static RabbitMQ connection
	// Consumer and Publisher use SEPARATE connections to avoid channel interference.
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

	// Init message signer for RabbitMQ with derived internal HMAC key
	cryptoWithInternalHMAC, errSigner := crypto.NewHMACSigner(keyDeriver.GetInternalHMACKey(), crypto.SignatureVersion)
	if errSigner != nil {
		return nil, fmt.Errorf("initialize internal message signer: %w", errSigner)
	}

	// Initialize RabbitMQ consumer and publisher with separate connections
	consumerRoutes, errRoutes := rabbitmq.NewConsumerRoutes(consumerConnection, cfg.RabbitMQNumWorkers, logger, telemetry, cryptoWithInternalHMAC, cfg.EnvName)
	if errRoutes != nil {
		return nil, fmt.Errorf("initialize consumer routes: %w", errRoutes)
	}

	publisherRoutes := rabbitmq.NewPublisherRoutes(publisherConnection, logger, telemetry, cryptoWithExternalHMAC)

	service.RabbitMQPublisher = publisherRoutes

	multiQueueConsumer := NewMultiQueueConsumer(consumerRoutes, service, cfg.RabbitMQGenerateReportQueue, logger, mongoManager)

	return &Service{
		MultiQueueConsumer: multiQueueConsumer,
		Logger:             logger,
		licenseShutdown:    licenseClient.GetLicenseManagerShutdown(),
	}, nil
}

// validateMultiTenantConfig validates multi-tenant configuration and logs the mode.
func validateMultiTenantConfig(cfg *Config, logger log.Logger) error {
	if cfg.MultiTenantEnabled {
		logger.Info("Multi-tenant mode ENABLED")

		if cfg.RedisHost == "" {
			return fmt.Errorf("REDIS_HOST is required when MULTI_TENANT_ENABLED=true (used for tenant discovery cache)")
		}
	} else {
		logger.Info("Running in SINGLE-TENANT MODE")
	}

	return nil
}

// initCryptoServices initializes the key deriver, crypto service, and external HMAC signer.
func initCryptoServices(cfg *Config) (*crypto.AESGCMService, *crypto.HMACSigner, *crypto.HKDFKeyDeriver, error) {
	masterKey, err := crypto.DecodeMasterKey(cfg.AppEncryptionKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decode master encryption key: %w", err)
	}

	keyDeriver, err := crypto.NewHKDFKeyDeriver(masterKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("initialize key deriver: %w", err)
	}

	cryptoService, errCrypto := crypto.NewAESGCMService(keyDeriver.GetCredentialKey(), cfg.AppEncryptionKeyVersion)
	if errCrypto != nil {
		return nil, nil, nil, fmt.Errorf("initialize crypto service: %w", errCrypto)
	}

	cryptoWithExternalHMAC, errSigner := crypto.NewHMACSigner(keyDeriver.GetExternalHMACKey(), crypto.SignatureVersion)
	if errSigner != nil {
		return nil, nil, nil, fmt.Errorf("initialize external document signer: %w", errSigner)
	}

	return cryptoService, cryptoWithExternalHMAC, keyDeriver, nil
}

// initStorageRepository initializes the storage repository (SeaweedFS or S3).
func initStorageRepository(ctx context.Context, cfg *Config) (portStorage.Repository, error) {
	storageProvider := cfg.StorageProvider
	if storageProvider == "" {
		storageProvider = pkgStorage.ProviderSeaweedFS
		cfg.StorageProvider = storageProvider
	}

	return pkgStorage.NewRepository(ctx, pkgStorage.ProviderConfig{
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
}

// initMongoConnection creates and returns a MongoDB connection configuration.
func initMongoConnection(cfg *Config, logger log.Logger) *mongoDB.MongoConnection {
	escapedPass := url.QueryEscape(cfg.MongoDBPassword)
	mongoSource := fmt.Sprintf("%s://%s:%s@%s:%s",
		cfg.MongoURI, cfg.MongoDBUser, escapedPass, cfg.MongoDBHost, cfg.MongoDBPort)

	if cfg.MaxPoolSize <= 0 {
		cfg.MaxPoolSize = 100
	}

	return &mongoDB.MongoConnection{
		ConnectionStringSource: mongoSource,
		Database:               cfg.MongoDBName,
		Logger:                 logger,
		MaxPoolSize:            uint64(cfg.MaxPoolSize),
	}
}

// logFileTTL logs the configured file TTL for storage.
func logFileTTL(logger log.Logger, cfg *Config) {
	if cfg.SeaweedFSTTL != "" {
		logger.Infof("Reports will expire after: %s", cfg.SeaweedFSTTL)
	} else {
		logger.Infof("Reports will be stored permanently (no TTL)")
	}
}

// initTenantManagerClient creates a Tenant Manager HTTP client with circuit breaker.
// This is shared across MongoDB manager and MultiTenantConsumer to avoid duplicate instances.
func initTenantManagerClient(cfg *Config, logger log.Logger) *tmclient.Client {
	var clientOpts []tmclient.ClientOption

	if cfg.MultiTenantCircuitBreakerThreshold > 0 {
		cbTimeout := time.Duration(cfg.MultiTenantCircuitBreakerTimeoutSec) * time.Second
		clientOpts = append(clientOpts,
			tmclient.WithCircuitBreaker(
				cfg.MultiTenantCircuitBreakerThreshold,
				cbTimeout,
			),
		)
	} else {
		clientOpts = append(clientOpts,
			tmclient.WithCircuitBreaker(5, 30*time.Second),
		)
	}

	return tmclient.NewClient(cfg.MultiTenantURL, logger, clientOpts...)
}

// resolvedMaxTenantPools returns the configured value if > 0, or the default.
func resolvedMaxTenantPools(cfg *Config) int {
	if cfg.MultiTenantMaxTenantPools > 0 {
		return cfg.MultiTenantMaxTenantPools
	}

	return defaultMaxTenantPools
}

// initMultiTenantManagers creates the Tenant Manager client and MongoDB manager when
// multi-tenant mode is enabled and configured. Returns (nil, nil, nil) when multi-tenant is disabled.
//
// Per multi-tenant.md standards:
//   - Circuit breaker is MANDATORY for the Tenant Manager client
//   - Uses constant.ApplicationName and constant.ModuleWorker for service/module identity
//   - WithMongoManager configures MongoDB connection pool management
//   - A single tmrabbitmq.Manager is shared between consumer and publisher
func initMultiTenantManagers(cfg *Config, logger log.Logger) (*tmmongo.Manager, *tmrabbitmq.Manager) {
	if !cfg.MultiTenantEnabled || cfg.MultiTenantURL == "" {
		return nil, nil
	}

	tmClient := initTenantManagerClient(cfg, logger)

	maxPools := resolvedMaxTenantPools(cfg)

	// Create MongoDB Manager for tenant connection pool management
	var mongoOpts []tmmongo.Option

	mongoOpts = append(mongoOpts,
		tmmongo.WithModule(constant.ModuleWorker),
		tmmongo.WithLogger(logger),
		tmmongo.WithMaxTenantPools(maxPools),
	)

	if cfg.MultiTenantIdleTimeoutSec > 0 {
		mongoOpts = append(mongoOpts, tmmongo.WithIdleTimeout(
			time.Duration(cfg.MultiTenantIdleTimeoutSec)*time.Second,
		))
	}

	mongoManager := tmmongo.NewManager(tmClient, constant.ApplicationName, mongoOpts...)

	// Create a single shared RabbitMQ Manager for per-tenant vhost connections.
	// Both consumer and publisher use this manager to avoid duplicate connection pools.
	var rabbitOpts []tmrabbitmq.Option

	rabbitOpts = append(rabbitOpts,
		tmrabbitmq.WithModule(constant.ModuleWorker),
		tmrabbitmq.WithLogger(logger),
		tmrabbitmq.WithMaxTenantPools(maxPools),
	)

	if cfg.MultiTenantIdleTimeoutSec > 0 {
		rabbitOpts = append(rabbitOpts, tmrabbitmq.WithIdleTimeout(
			time.Duration(cfg.MultiTenantIdleTimeoutSec)*time.Second,
		))
	}

	rabbitManager := tmrabbitmq.NewManager(tmClient, constant.ApplicationName, rabbitOpts...)

	logger.Infof("Multi-tenant managers initialized: url=%s, module=%s", cfg.MultiTenantURL, constant.ModuleWorker)

	return mongoManager, rabbitManager
}

// initMultiTenantConsumer creates the tmconsumer.MultiTenantConsumer for per-tenant
// vhost isolation with lazy initialization. This is the preferred pattern for multi-tenant
// RabbitMQ consumption as it dynamically discovers tenants from Redis cache and spawns
// consumer goroutines per-tenant vhost.
//
// Returns:
//   - mtConsumer: the MultiTenantConsumer instance to register handlers and run
//   - cleanup: a function to close Redis and MultiTenantConsumer resources
//   - error: if Redis connection or consumer creation fails
func initMultiTenantConsumer(
	ctx context.Context,
	cfg *Config,
	logger log.Logger,
	tenantMongoManager *tmmongo.Manager,
	rabbitMQManager *tmrabbitmq.Manager,
) (*tmconsumer.MultiTenantConsumer, func(), error) {
	// Create Redis connection for tenant discovery cache
	redisConn := &libRedis.RedisConnection{
		Address:  []string{cfg.RedisHost},
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		Protocol: cfg.RedisProtocol,
		Logger:   logger,
	}

	redisClient, err := redisConn.GetClient(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to Redis for tenant discovery: %w", err)
	}

	logger.Infof("Redis connected for multi-tenant consumer (host: %s, db: %d)", cfg.RedisHost, cfg.RedisDB)

	// Configure MultiTenantConsumer
	mtConfig := tmconsumer.DefaultMultiTenantConfig()
	mtConfig.Service = constant.ApplicationName
	mtConfig.Environment = cfg.MultiTenantEnvironment
	mtConfig.MultiTenantURL = cfg.MultiTenantURL
	mtConfig.PrefetchCount = constant.DefaultPrefetchCount

	// SyncInterval: use configured value (seconds) or default 30s
	if cfg.RabbitMQMultiTenantSyncInterval > 0 {
		mtConfig.SyncInterval = time.Duration(cfg.RabbitMQMultiTenantSyncInterval) * time.Second
	}

	// DiscoveryTimeout: use configured value (milliseconds) or default 500ms
	if cfg.RabbitMQMultiTenantDiscoveryTimeout > 0 {
		mtConfig.DiscoveryTimeout = time.Duration(cfg.RabbitMQMultiTenantDiscoveryTimeout) * time.Millisecond
	}

	// Build options
	var consumerOpts []tmconsumer.Option
	if tenantMongoManager != nil {
		consumerOpts = append(consumerOpts, tmconsumer.WithMongoManager(tenantMongoManager))
	}

	// Create MultiTenantConsumer
	mtConsumer := tmconsumer.NewMultiTenantConsumer(
		rabbitMQManager,
		redisClient,
		mtConfig,
		logger,
		consumerOpts...,
	)

	logger.Info("MultiTenantConsumer initialized with per-tenant vhost isolation")

	// Cleanup function to close Redis connection only.
	// mtConsumer.Close() is handled by MultiQueueConsumer.Run() on context cancellation.
	cleanup := func() {
		logger.Info("Cleanup: closing Redis connection")

		if closeErr := redisConn.Close(); closeErr != nil {
			logger.Errorf("Cleanup: failed to close Redis connection: %v", closeErr)
		}
	}

	return mtConsumer, cleanup, nil
}

// rabbitMQManagerAdapter wraps tmrabbitmq.Manager to satisfy rabbitmq.RabbitMQManagerInterface.
type rabbitMQManagerAdapter struct {
	manager *tmrabbitmq.Manager
}

func newRabbitMQManagerAdapter(manager *tmrabbitmq.Manager) *rabbitMQManagerAdapter {
	return &rabbitMQManagerAdapter{manager: manager}
}

// GetChannel wraps tmrabbitmq.Manager.GetChannel and converts the returned *amqp091.Channel
// to the RabbitMQChannel interface.
func (a *rabbitMQManagerAdapter) GetChannel(ctx context.Context, tenantID string) (rabbitmq.RabbitMQChannel, error) {
	return a.manager.GetChannel(ctx, tenantID)
}
