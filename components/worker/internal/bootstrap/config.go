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
	"github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	portConnection "github.com/LerianStudio/fetcher/pkg/ports/connection"
	portJob "github.com/LerianStudio/fetcher/pkg/ports/job"
	portStorage "github.com/LerianStudio/fetcher/pkg/ports/storage"
	pkgStorage "github.com/LerianStudio/fetcher/pkg/storage"
	mongoDB "github.com/LerianStudio/lib-commons/v3/commons/mongo"

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

// Validate checks that all required configuration fields are present.
// Returns a descriptive multi-error message listing all missing fields.
func (c *Config) Validate() error {
	var errs []string

	if c.RabbitMQHost == "" {
		errs = append(errs, "RABBITMQ_HOST is required")
	}

	if c.RabbitMQPortAMQP == "" {
		errs = append(errs, "RABBITMQ_PORT_AMQP is required")
	}

	if c.RabbitMQUser == "" {
		errs = append(errs, "RABBITMQ_DEFAULT_USER is required")
	}

	if c.RabbitMQPass == "" {
		errs = append(errs, "RABBITMQ_DEFAULT_PASS is required")
	}

	if c.RabbitMQGenerateReportQueue == "" {
		errs = append(errs, "RABBITMQ_FETCHER_WORK_QUEUE is required")
	}

	if c.MongoDBHost == "" {
		errs = append(errs, "MONGO_HOST is required")
	}

	if c.MongoDBName == "" {
		errs = append(errs, "MONGO_NAME is required")
	}

	if c.AppEncryptionKey == "" {
		errs = append(errs, "APP_ENC_KEY is required")
	}

	if c.MultiTenantEnabled && c.MultiTenantURL == "" {
		errs = append(errs, "MULTI_TENANT_URL is required when MULTI_TENANT_ENABLED=true")
	}

	if c.MultiTenantEnabled {
		if c.RedisHost == "" {
			errs = append(errs, "REDIS_HOST is required when MULTI_TENANT_ENABLED=true")
		}

		if c.RedisPort == "" {
			errs = append(errs, "REDIS_PORT is required when MULTI_TENANT_ENABLED=true")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n- %s", strings.Join(errs, "\n- "))
	}

	return nil
}

// cryptoComponents holds the initialized cryptographic services for the worker.
type cryptoComponents struct {
	service      *crypto.AESGCMService
	internalHMAC *crypto.HMACSigner
	externalHMAC *crypto.HMACSigner
}

// rabbitMQComponents holds the initialized RabbitMQ consumer and publisher for the worker.
type rabbitMQComponents struct {
	consumerRoutes      *rabbitmq.ConsumerRoutes
	multiTenantConsumer *tmconsumer.MultiTenantConsumer
	rmqManager          *tmrabbitmq.Manager
	publisherRoutes     rabbitmq.PublisherRepository
}

// mongoComponents holds the initialized MongoDB connection and repositories.
type mongoComponents struct {
	connection *mongoDB.MongoConnection
	jobRepo    portJob.Repository
	connRepo   portConnection.Repository
}

// InitWorker initializes and configures the application's dependencies and returns the Service instance.
func InitWorker() (_ *Service, err error) {
	cfg := &Config{}
	if err := libCommons.SetConfigFromEnvVars(cfg); err != nil {
		return nil, fmt.Errorf("load environment configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ctx := context.Background()

	logger, err := libZap.InitializeLoggerWithError()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	logTenantMode(cfg, logger)

	// Cleanup stack: on failure, close resources in reverse order
	var cleanups []func()

	defer func() {
		if err != nil {
			runCleanups(cleanups, logger)
		}
	}()

	telemetry, err := initTelemetry(cfg, logger)
	if err != nil {
		return nil, err
	}

	cleanups = append(cleanups, func() {
		logger.Info("Cleanup: shutting down telemetry")
		telemetry.ShutdownTelemetry()
	})

	cryptoComps, err := initCryptoServices(cfg, logger)
	if err != nil {
		return nil, err
	}

	// Create shared Tenant Manager client for MT mode (nil in single-tenant mode).
	var tmClient *tmclient.Client
	if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
		tmClient = initTenantManagerClient(cfg, logger)
	}

	// Initialize MongoDB manager once -- reused by consumer setup below.
	// In single-tenant mode initMultiTenantMongoManager returns nil (no-op).
	mongoManager := initMultiTenantMongoManager(tmClient, cfg, logger)

	rmqComps, err := initRabbitMQComponents(cfg, tmClient, mongoManager, cryptoComps, logger, telemetry)
	if err != nil {
		return nil, err
	}

	storageRepository, err := initStorageRepository(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	mongoComps, err := initMongoRepositories(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	cleanups = append(cleanups, func() {
		if mongoComps.connection.DB != nil {
			logger.Info("Cleanup: disconnecting MongoDB")

			if disconnectErr := mongoComps.connection.DB.Disconnect(context.Background()); disconnectErr != nil {
				logger.Errorf("Cleanup: failed to disconnect MongoDB: %v", disconnectErr)
			}
		}
	})

	service := assembleUseCase(cfg, mongoComps, storageRepository, cryptoComps, rmqComps.publisherRoutes, logger)

	licenseClient := libLicense.NewLicenseClient(
		constant.ApplicationName,
		cfg.LicenseKey,
		cfg.OrganizationIDs,
		&logger,
	)

	consumer := initConsumer(cfg, rmqComps, service, mongoManager, logger)

	return &Service{
		Consumer:        consumer,
		Logger:          logger,
		licenseShutdown: licenseClient.GetLicenseManagerShutdown(),
	}, nil
}

// logTenantMode logs whether the worker is running in multi-tenant or single-tenant mode.
func logTenantMode(cfg *Config, logger log.Logger) {
	if cfg.MultiTenantEnabled {
		logger.Info("Multi-tenant mode ENABLED")
	} else {
		logger.Info("Running in SINGLE-TENANT MODE")
	}
}

// runCleanups executes cleanup functions in reverse order on initialization failure.
func runCleanups(cleanups []func(), logger log.Logger) {
	logger.Infof("Initialization failed, cleaning up %d resources...", len(cleanups))

	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}
}

// initTelemetry initializes the OpenTelemetry provider.
func initTelemetry(cfg *Config, logger log.Logger) (*libOtel.Telemetry, error) {
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

	return telemetry, nil
}

// initCryptoServices initializes the master key, key deriver, AES-GCM crypto service,
// and HMAC signers (internal for RabbitMQ messages, external for document verification).
func initCryptoServices(cfg *Config, logger log.Logger) (*cryptoComponents, error) {
	masterKey, err := crypto.DecodeMasterKey(cfg.AppEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decode master encryption key: %w", err)
	}

	keyDeriver, err := crypto.NewHKDFKeyDeriver(masterKey)
	if err != nil {
		return nil, fmt.Errorf("initialize key deriver: %w", err)
	}

	logger.Info("Key derivation initialized successfully")

	cryptoService, err := crypto.NewAESGCMService(keyDeriver.GetCredentialKey(), cfg.AppEncryptionKeyVersion)
	if err != nil {
		return nil, fmt.Errorf("initialize crypto service: %w", err)
	}

	internalHMAC, err := crypto.NewHMACSigner(keyDeriver.GetInternalHMACKey(), crypto.SignatureVersion)
	if err != nil {
		return nil, fmt.Errorf("initialize internal message signer: %w", err)
	}

	externalHMAC, err := crypto.NewHMACSigner(keyDeriver.GetExternalHMACKey(), crypto.SignatureVersion)
	if err != nil {
		return nil, fmt.Errorf("initialize external document signer: %w", err)
	}

	return &cryptoComponents{
		service:      cryptoService,
		internalHMAC: internalHMAC,
		externalHMAC: externalHMAC,
	}, nil
}

// initRabbitMQComponents initializes RabbitMQ consumer and publisher, choosing between
// multi-tenant vhost isolation and single-tenant static connections based on configuration.
func initRabbitMQComponents(
	cfg *Config,
	tmClient *tmclient.Client,
	mongoManager *tmmongo.Manager,
	cryptoComps *cryptoComponents,
	logger log.Logger,
	telemetry *libOtel.Telemetry,
) (*rabbitMQComponents, error) {
	if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
		return initMultiTenantRabbitMQ(cfg, tmClient, mongoManager, cryptoComps, logger, telemetry)
	}

	return initSingleTenantRabbitMQ(cfg, cryptoComps, logger, telemetry)
}

// initMultiTenantRabbitMQ sets up RabbitMQ with per-tenant vhost isolation using
// tmrabbitmq.Manager and tmconsumer.MultiTenantConsumer.
func initMultiTenantRabbitMQ(
	cfg *Config,
	tmClient *tmclient.Client,
	mongoManager *tmmongo.Manager,
	cryptoComps *cryptoComponents,
	logger log.Logger,
	telemetry *libOtel.Telemetry,
) (*rabbitMQComponents, error) {
	logger.Info("Initializing RabbitMQ with multi-tenant vhost isolation")

	rmqManager := initMultiTenantRabbitMQManager(tmClient, cfg, logger)

	redisClient, err := initRedisClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize redis client: %w", err)
	}

	mtConfig := tmconsumer.MultiTenantConfig{
		SyncInterval:     30 * time.Second,
		PrefetchCount:    cfg.RabbitMQNumWorkers,
		MultiTenantURL:   cfg.MultiTenantURL,
		Service:          constant.ApplicationName,
		Environment:      cfg.MultiTenantEnvironment,
		DiscoveryTimeout: 500 * time.Millisecond,
	}

	var consumerOpts []tmconsumer.Option
	if mongoManager != nil {
		consumerOpts = append(consumerOpts, tmconsumer.WithMongoManager(mongoManager))
	}

	multiTenantConsumer := tmconsumer.NewMultiTenantConsumer(
		rmqManager,
		redisClient,
		mtConfig,
		logger,
		consumerOpts...,
	)

	publisherRoutes := rabbitmq.NewPublisherRoutesMultiTenant(rmqManager, logger, telemetry, cryptoComps.externalHMAC)

	logger.Infof("Multi-tenant RabbitMQ initialized: environment=%s, service=%s", cfg.MultiTenantEnvironment, constant.ApplicationName)

	return &rabbitMQComponents{
		multiTenantConsumer: multiTenantConsumer,
		rmqManager:          rmqManager,
		publisherRoutes:     publisherRoutes,
	}, nil
}

// initSingleTenantRabbitMQ sets up RabbitMQ with static consumer and publisher connections.
// Consumer and Publisher use SEPARATE connections to avoid channel interference.
func initSingleTenantRabbitMQ(
	cfg *Config,
	cryptoComps *cryptoComponents,
	logger log.Logger,
	telemetry *libOtel.Telemetry,
) (*rabbitMQComponents, error) {
	// URL-encode credentials to handle special characters (@ : / etc.)
	rabbitSource := buildRabbitMQSource(cfg)

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

	publisherConnection := &libRabbitMQ.RabbitMQConnection{
		ConnectionStringSource: rabbitSource,
		HealthCheckURL:         cfg.RabbitMQHealthCheckURL,
		Host:                   cfg.RabbitMQHost,
		Port:                   cfg.RabbitMQPortHost,
		User:                   cfg.RabbitMQUser,
		Pass:                   cfg.RabbitMQPass,
		Logger:                 logger,
	}

	consumerRoutes, err := rabbitmq.NewConsumerRoutes(consumerConnection, cfg.RabbitMQNumWorkers, logger, telemetry, cryptoComps.internalHMAC, cfg.EnvName)
	if err != nil {
		return nil, fmt.Errorf("initialize consumer routes: %w", err)
	}

	publisherRoutes := rabbitmq.NewPublisherRoutes(publisherConnection, logger, telemetry, cryptoComps.externalHMAC)

	return &rabbitMQComponents{
		consumerRoutes:  consumerRoutes,
		publisherRoutes: publisherRoutes,
	}, nil
}

// buildRabbitMQSource builds the AMQP connection string with URL-encoded credentials.
func buildRabbitMQSource(cfg *Config) string {
	escapedUser := url.PathEscape(cfg.RabbitMQUser)
	escapedPass := url.QueryEscape(cfg.RabbitMQPass)

	return fmt.Sprintf("%s://%s:%s@%s:%s",
		cfg.RabbitURI, escapedUser, escapedPass, cfg.RabbitMQHost, cfg.RabbitMQPortAMQP)
}

// initStorageRepository initializes the storage backend (SeaweedFS or S3) via factory.
func initStorageRepository(ctx context.Context, cfg *Config, logger log.Logger) (portStorage.Repository, error) {
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

	return storageRepository, nil
}

// initMongoRepositories initializes the MongoDB connection and creates job and connection repositories.
func initMongoRepositories(ctx context.Context, cfg *Config, logger log.Logger) (*mongoComponents, error) {
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
		MaxPoolSize:            uint64(cfg.MaxPoolSize), //nolint:gosec // MaxPoolSize is validated > 0 above
	}

	jobRepository, err := job.NewJobMongoDBRepository(ctx, mongoConnection)
	if err != nil {
		return nil, fmt.Errorf("initialize job repository: %w", err)
	}

	connectionRepository, err := connection.NewConnectionMongoDBRepository(ctx, mongoConnection)
	if err != nil {
		return nil, fmt.Errorf("initialize connection repository: %w", err)
	}

	return &mongoComponents{
		connection: mongoConnection,
		jobRepo:    jobRepository,
		connRepo:   connectionRepository,
	}, nil
}

// assembleUseCase creates and configures the worker UseCase with all required dependencies.
func assembleUseCase(
	cfg *Config,
	mongoComps *mongoComponents,
	storageRepository portStorage.Repository,
	cryptoComps *cryptoComponents,
	publisherRoutes rabbitmq.PublisherRepository,
	logger log.Logger,
) *services.UseCase {
	service := &services.UseCase{
		ExternalDataStorage:  storageRepository,
		JobRepository:        mongoComps.jobRepo,
		ConnectionRepository: mongoComps.connRepo,
		Cryptor:              cryptoComps.service,
		DocumentSigner:       cryptoComps.externalHMAC,
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

	return service
}

// initConsumer creates the appropriate consumer (multi-tenant or single-tenant) based on configuration.
func initConsumer(
	cfg *Config,
	rmqComps *rabbitMQComponents,
	service *services.UseCase,
	mongoManager *tmmongo.Manager,
	logger log.Logger,
) Consumer {
	if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
		return NewMultiTenantConsumerAdapter(
			rmqComps.multiTenantConsumer,
			service,
			cfg.RabbitMQGenerateReportQueue,
			mongoManager,
			rmqComps.rmqManager,
			logger,
		)
	}

	return NewMultiQueueConsumer(rmqComps.consumerRoutes, service, cfg.RabbitMQGenerateReportQueue, mongoManager)
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
func initMultiTenantMongoManager(tmClient *tmclient.Client, cfg *Config, logger log.Logger) *tmmongo.Manager {
	if !cfg.MultiTenantEnabled || cfg.MultiTenantURL == "" || tmClient == nil {
		return nil
	}

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
// Uses lib-commons RedisConnection for standardized connection management.
func initRedisClient(cfg *Config, logger log.Logger) (redis.UniversalClient, error) {
	redisAddr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)

	redisConn := &libRedis.RedisConnection{
		Address:  []string{redisAddr},
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		Logger:   logger,
	}

	client, err := redisConn.GetClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis for tenant discovery: %w", err)
	}

	logger.Infof("Redis client initialized for multi-tenant consumer: addr=%s", redisAddr)

	return client, nil
}
