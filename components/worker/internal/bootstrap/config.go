package bootstrap

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/components/worker/internal/services"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	portStorage "github.com/LerianStudio/fetcher/pkg/ports/storage"
	pkgStorage "github.com/LerianStudio/fetcher/pkg/storage"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	mongoDB "github.com/LerianStudio/lib-commons/v4/commons/mongo"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libRabbitMQ "github.com/LerianStudio/lib-commons/v4/commons/rabbitmq"
	libRedis "github.com/LerianStudio/lib-commons/v4/commons/redis"
	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	tmconsumer "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/consumer"
	tmmongo "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/mongo"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/rabbitmq"
	libZap "github.com/LerianStudio/lib-commons/v4/commons/zap"
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
	MultiTenantServiceAPIKey            string `env:"MULTI_TENANT_SERVICE_API_KEY"`
	// RabbitMQ multi-tenant consumer tuning (active when MULTI_TENANT_ENABLED=true)
	RabbitMQMultiTenantSyncInterval     int `env:"RABBITMQ_MULTI_TENANT_SYNC_INTERVAL"`     // Stored in seconds; default 30
	RabbitMQMultiTenantDiscoveryTimeout int `env:"RABBITMQ_MULTI_TENANT_DISCOVERY_TIMEOUT"` // Stored in milliseconds; default 500
	// Redis/Valkey configuration (required for multi-tenant tenant discovery cache)
	RedisHost     string `env:"REDIS_HOST"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB       int    `env:"REDIS_DB"`
	RedisProtocol int    `env:"REDIS_PROTOCOL"`
}

var (
	setConfigFromEnvVars  = libCommons.SetConfigFromEnvVars
	newZapLogger          = func(cfg libZap.Config) (libLog.Logger, error) { return libZap.New(cfg) }
	newTelemetry          = libOtel.NewTelemetry
	newMongoClient        = mongoDB.NewClient
	applyTelemetryGlobals = func(telemetry *libOtel.Telemetry) error {
		return telemetry.ApplyGlobals()
	}
	decodeMasterKey = crypto.DecodeMasterKey
	newKeyDeriver   = crypto.NewHKDFKeyDeriver
)

// InitWorker initializes and configures the application's dependencies and returns the Service instance.
func InitWorker() (*Service, error) {
	cfg := &Config{}
	if err := setConfigFromEnvVars(cfg); err != nil {
		return nil, fmt.Errorf("load environment configuration: %w", err)
	}

	ctx := context.Background()

	logger, err := newZapLogger(libZap.Config{
		Environment:     resolveZapEnvironment(cfg.EnvName),
		Level:           cfg.LogLevel,
		OTelLibraryName: cfg.OtelLibraryName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	if err := validateMultiTenantConfig(cfg, logger); err != nil {
		return nil, err
	}

	telemetry, err := newTelemetry(libOtel.TelemetryConfig{
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

	if err := applyTelemetryGlobals(telemetry); err != nil {
		return nil, err
	}

	cryptoService, cryptoWithExternalHMAC, keyDeriver, err := initCryptoServices(cfg)
	if err != nil {
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Key derivation initialized successfully")

	storageRepository, err := initStorageRepository(ctx, cfg)
	if err != nil {
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Storage initialized with provider: %s", cfg.StorageProvider))

	mongoConnection, err := initMongoConnection(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize MongoDB client: %w", err)
	}

	// Initialize multi-tenant managers (nil when MULTI_TENANT_ENABLED=false)
	mongoManager, rabbitMQManager, err := initMultiTenantManagers(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize multi-tenant managers: %w", err)
	}

	// Wrap the MongoDB connection to implement tmcore.MultiTenantChecker.
	// When multi-tenant mode is enabled, this causes tmcore.ResolveMongo to return
	// ErrTenantContextRequired instead of silently falling back to the default DB
	// when no tenant context is present.
	mongoProvider := mongodb.NewMultiTenantMongoProvider(mongoConnection, cfg.MultiTenantEnabled)

	// Initialize MongoDB repositories
	jobRepository, errJobRepo := job.NewJobMongoDBRepository(ctx, mongoProvider, cfg.MongoDBName)
	if errJobRepo != nil {
		return nil, fmt.Errorf("initialize job repository: %w", errJobRepo)
	}

	connectionRepository, errConnectRepo := connection.NewConnectionMongoDBRepository(ctx, mongoProvider, cfg.MongoDBName)
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

	licenseLoggerV4, licenseLogErr := libZap.New(libZap.Config{
		Environment:     resolveZapEnvironment(cfg.EnvName),
		Level:           cfg.LogLevel,
		OTelLibraryName: constant.ApplicationName + "-license",
	})
	if licenseLogErr != nil {
		return nil, wrapBootstrapError("initialize license logger", licenseLogErr)
	}

	var licenseLogger libLog.Logger = licenseLoggerV4

	licenseClient := libLicense.NewLicenseClient(
		constant.ApplicationName,
		cfg.LicenseKey,
		cfg.OrganizationIDs,
		&licenseLogger,
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
func validateMultiTenantConfig(cfg *Config, logger libLog.Logger) error {
	if cfg.MultiTenantEnabled {
		logger.Log(context.Background(), libLog.LevelInfo, "Multi-tenant mode ENABLED")

		if cfg.MultiTenantURL == "" {
			return fmt.Errorf("MULTI_TENANT_URL is required when MULTI_TENANT_ENABLED=true")
		}

		if cfg.MultiTenantServiceAPIKey == "" {
			return fmt.Errorf("MULTI_TENANT_SERVICE_API_KEY is required when MULTI_TENANT_ENABLED=true")
		}

		if cfg.RedisHost == "" {
			return fmt.Errorf("REDIS_HOST is required when MULTI_TENANT_ENABLED=true (used for tenant discovery cache)")
		}
	} else {
		logger.Log(context.Background(), libLog.LevelInfo, "Running in SINGLE-TENANT MODE")
	}

	return nil
}

// initCryptoServices initializes the key deriver, crypto service, and external HMAC signer.
func initCryptoServices(cfg *Config) (*crypto.AESGCMService, *crypto.HMACSigner, *crypto.HKDFKeyDeriver, error) {
	masterKey, err := decodeMasterKey(cfg.AppEncryptionKey)
	if err != nil {
		return nil, nil, nil, wrapBootstrapError("decode master encryption key", err)
	}

	keyDeriver, err := newKeyDeriver(masterKey)
	if err != nil {
		return nil, nil, nil, wrapBootstrapError("initialize key deriver", err)
	}

	cryptoService, errCrypto := crypto.NewAESGCMService(keyDeriver.GetCredentialKey(), cfg.AppEncryptionKeyVersion)
	if errCrypto != nil {
		return nil, nil, nil, wrapBootstrapError("initialize crypto service", errCrypto)
	}

	cryptoWithExternalHMAC, errSigner := crypto.NewHMACSigner(keyDeriver.GetExternalHMACKey(), crypto.SignatureVersion)
	if errSigner != nil {
		return nil, nil, nil, wrapBootstrapError("initialize external document signer", errSigner)
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

// initMongoConnection creates and returns a MongoDB client.
func initMongoConnection(ctx context.Context, cfg *Config, logger libLog.Logger) (*mongoDB.Client, error) {
	escapedPass := url.QueryEscape(cfg.MongoDBPassword)
	mongoSource := fmt.Sprintf("%s://%s:%s@%s:%s",
		cfg.MongoURI, cfg.MongoDBUser, escapedPass, cfg.MongoDBHost, cfg.MongoDBPort)

	if cfg.MaxPoolSize <= 0 {
		cfg.MaxPoolSize = 100
	}

	return newMongoClient(ctx, mongoDB.Config{
		URI:         mongoSource,
		Database:    cfg.MongoDBName,
		Logger:      logger,
		MaxPoolSize: uint64(cfg.MaxPoolSize),
	})
}

// logFileTTL logs the configured file TTL for storage.
func logFileTTL(logger libLog.Logger, cfg *Config) {
	if cfg.SeaweedFSTTL != "" {
		logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Reports will expire after: %s", cfg.SeaweedFSTTL))
	} else {
		logger.Log(context.Background(), libLog.LevelInfo, "Reports will be stored permanently (no TTL)")
	}
}

func resolveZapEnvironment(env string) libZap.Environment {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "production", "prod":
		return libZap.EnvironmentProduction
	case "staging", "stage":
		return libZap.EnvironmentStaging
	case "uat":
		return libZap.EnvironmentUAT
	case "development", "dev":
		return libZap.EnvironmentDevelopment
	case "local":
		return libZap.EnvironmentLocal
	default:
		return libZap.EnvironmentLocal
	}
}

// initTenantManagerClient creates a Tenant Manager HTTP client with circuit breaker.
// This is shared across MongoDB manager and MultiTenantConsumer to avoid duplicate instances.
func initTenantManagerClient(cfg *Config, logger libLog.Logger) (*tmclient.Client, error) {
	var clientOpts []tmclient.ClientOption

	clientOpts = append(clientOpts,
		tmclient.WithServiceAPIKey(cfg.MultiTenantServiceAPIKey),
	)

	// Allow plaintext HTTP for local/dev environments where TLS is not configured.
	if strings.HasPrefix(cfg.MultiTenantURL, "http://") {
		clientOpts = append(clientOpts, tmclient.WithAllowInsecureHTTP())
	}

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

	client, err := tmclient.NewClient(cfg.MultiTenantURL, logger, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("create tenant manager client: %w", err)
	}

	return client, nil
}

// resolvedMaxTenantPools returns the configured value if > 0, or the default.
func resolvedMaxTenantPools(cfg *Config) int {
	if cfg.MultiTenantMaxTenantPools > 0 {
		return cfg.MultiTenantMaxTenantPools
	}

	return defaultMaxTenantPools
}

// initMultiTenantManagers creates the Tenant Manager client and MongoDB manager when
// multi-tenant mode is enabled and configured. Returns (nil, nil) when multi-tenant is disabled.
func initMultiTenantManagers(cfg *Config, logger libLog.Logger) (*tmmongo.Manager, *tmrabbitmq.Manager, error) {
	if !cfg.MultiTenantEnabled || cfg.MultiTenantURL == "" {
		return nil, nil, nil
	}

	tmClient, err := initTenantManagerClient(cfg, logger)
	if err != nil {
		return nil, nil, err
	}

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

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Multi-tenant managers initialized: url=%s, module=%s", cfg.MultiTenantURL, constant.ModuleWorker))

	return mongoManager, rabbitManager, nil
}

// initMultiTenantConsumer creates the tmconsumer.MultiTenantConsumer for per-tenant
// vhost isolation with lazy initialization.
func initMultiTenantConsumer(
	ctx context.Context,
	cfg *Config,
	logger libLog.Logger,
	tenantMongoManager *tmmongo.Manager,
	rabbitMQManager *tmrabbitmq.Manager,
) (*tmconsumer.MultiTenantConsumer, func(), error) {
	// Create Redis connection for tenant discovery cache
	redisConn, err := libRedis.New(ctx, libRedis.Config{
		Topology: libRedis.Topology{
			Standalone: &libRedis.StandaloneTopology{
				Address: cfg.RedisHost,
			},
		},
		Auth: libRedis.Auth{
			StaticPassword: &libRedis.StaticPasswordAuth{
				Password: cfg.RedisPassword,
			},
		},
		Logger: logger,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to Redis for tenant discovery: %w", err)
	}

	redisClient, err := redisConn.GetClient(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Redis client for tenant discovery: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Redis connected for multi-tenant consumer (host: %s, db: %d)", cfg.RedisHost, cfg.RedisDB))

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
	mtConsumer, err := tmconsumer.NewMultiTenantConsumerWithError(
		rabbitMQManager,
		redisClient,
		mtConfig,
		logger,
		consumerOpts...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create multi-tenant consumer: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "MultiTenantConsumer initialized with per-tenant vhost isolation")

	// Cleanup function to close Redis connection only.
	// mtConsumer.Close() is handled by MultiQueueConsumer.Run() on context cancellation.
	cleanup := func() {
		logger.Log(context.Background(), libLog.LevelInfo, "Cleanup: closing Redis connection")

		if closeErr := redisConn.Close(); closeErr != nil {
			logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Cleanup: failed to close Redis connection: %v", closeErr))
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

func wrapBootstrapError(action string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return nil
}
