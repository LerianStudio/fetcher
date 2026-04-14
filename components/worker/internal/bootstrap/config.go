package bootstrap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
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
	"github.com/LerianStudio/fetcher/pkg/resolver"
	pkgStorage "github.com/LerianStudio/fetcher/pkg/storage"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	mongoDB "github.com/LerianStudio/lib-commons/v4/commons/mongo"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libRabbitMQ "github.com/LerianStudio/lib-commons/v4/commons/rabbitmq"
	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	tmconsumer "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/consumer"
	tmevent "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/event"
	tmmongo "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/mongo"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/rabbitmq"
	"github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/tenantcache"
	libZap "github.com/LerianStudio/lib-commons/v4/commons/zap"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
	"github.com/redis/go-redis/v9"
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
	RabbitMQTLS                 bool   `env:"RABBITMQ_TLS" default:"false"`
	// Otel Collector configurations
	OtelServiceName         string `env:"OTEL_RESOURCE_SERVICE_NAME"`
	OtelLibraryName         string `env:"OTEL_LIBRARY_NAME"`
	OtelServiceVersion      string `env:"OTEL_RESOURCE_SERVICE_VERSION"`
	OtelDeploymentEnv       string `env:"OTEL_RESOURCE_DEPLOYMENT_ENVIRONMENT"`
	OtelColExporterEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	EnableTelemetry         bool   `env:"ENABLE_TELEMETRY"`
	OtelInsecureExporter    bool   `env:"OTEL_INSECURE_EXPORTER"`
	// S3-compatible object storage configuration (AWS S3, MinIO, SeaweedFS S3 API).
	// SSL is controlled by the URL scheme of ObjectStorageEndpoint.
	ObjectStorageEndpoint     string `env:"OBJECT_STORAGE_ENDPOINT"`
	ObjectStorageRegion       string `env:"OBJECT_STORAGE_REGION"`
	ObjectStorageBucket       string `env:"OBJECT_STORAGE_BUCKET"`
	ObjectStorageKeyPrefix    string `env:"OBJECT_STORAGE_KEY_PREFIX"`
	ObjectStorageAccessKeyID  string `env:"OBJECT_STORAGE_ACCESS_KEY_ID"`
	ObjectStorageSecretKey    string `env:"OBJECT_STORAGE_SECRET_KEY"`
	ObjectStorageUsePathStyle bool   `env:"OBJECT_STORAGE_USE_PATH_STYLE"`
	// ObjectStorageTTL is the file TTL for storage backends that support it (e.g., SeaweedFS).
	// S3 ignores this — use lifecycle policies instead. Format: "1h", "7d", "6M".
	ObjectStorageTTL string `env:"OBJECT_STORAGE_TTL"`
	// OBJECT_STORAGE_DISABLE_SSL omitted — SSL controlled by endpoint URL scheme.
	// MongoDB
	MongoURI          string `env:"MONGO_URI"`
	MongoDBHost       string `env:"MONGO_HOST"`
	MongoDBName       string `env:"MONGO_NAME"`
	MongoDBUser       string `env:"MONGO_USER"`
	MongoDBPassword   string `env:"MONGO_PASSWORD"`
	MongoDBPort       string `env:"MONGO_PORT"`
	MongoDBParameters string `env:"MONGO_PARAMETERS"`
	MongoTLSCACert    string `env:"MONGO_TLS_CA_CERT"`
	MaxPoolSize       int    `env:"MONGO_MAX_POOL_SIZE"`
	// License configuration envs
	LicenseKey      string `env:"LICENSE_KEY"`
	OrganizationIDs string `env:"ORGANIZATION_IDS"`
	// Encryption
	AppEncryptionKey        string `env:"APP_ENC_KEY"`
	AppEncryptionKeyVersion string `env:"APP_ENC_KEY_VERSION"`
	// CRM plugin encryption keys
	CryptoEncryptSecretKeyPluginCRM string `env:"CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM"`
	CryptoHashSecretKeyPluginCRM    string `env:"CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM"`
	// Multi-Tenant configuration
	MultiTenantEnabled                  bool   `env:"MULTI_TENANT_ENABLED"`
	MultiTenantURL                      string `env:"MULTI_TENANT_URL"`
	MultiTenantRedisHost                string `env:"MULTI_TENANT_REDIS_HOST"`
	MultiTenantRedisPort                string `env:"MULTI_TENANT_REDIS_PORT" default:"6379"`
	MultiTenantRedisPassword            string `env:"MULTI_TENANT_REDIS_PASSWORD"`
	MultiTenantRedisTLS                 bool   `env:"MULTI_TENANT_REDIS_TLS" default:"false"`
	MultiTenantRedisCACert              string `env:"MULTI_TENANT_REDIS_CA_CERT"`
	MultiTenantMaxTenantPools           int    `env:"MULTI_TENANT_MAX_TENANT_POOLS" default:"100"`
	MultiTenantIdleTimeoutSec           int    `env:"MULTI_TENANT_IDLE_TIMEOUT_SEC" default:"300"`
	MultiTenantCircuitBreakerThreshold  int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD" default:"5"`
	MultiTenantCircuitBreakerTimeoutSec int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC" default:"30"`
	MultiTenantServiceAPIKey            string `env:"MULTI_TENANT_SERVICE_API_KEY"`
	MultiTenantCacheTTLSec              int    `env:"MULTI_TENANT_CACHE_TTL_SEC" default:"120"`
	MultiTenantTimeout                  int    `env:"MULTI_TENANT_TIMEOUT" default:"30"`
	MultiTenantAllowInsecureHTTP        bool   `env:"MULTI_TENANT_ALLOW_INSECURE_HTTP" default:"false"`
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

	logger, telemetry, err := initObservability(cfg)
	if err != nil {
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

	logger.Log(ctx, libLog.LevelInfo, "Storage initialized (S3)",
		libLog.String("bucket", cfg.ObjectStorageBucket),
		libLog.String("region", cfg.ObjectStorageRegion))

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
		FileTTL:              cfg.ObjectStorageTTL,
		JobEventsExchange:    cfg.RabbitMQJobEventsExchange,
	}
	service.SetStorageEncryptDerivedKey(keyDeriver.GetStorageEncryptKey())
	service.SetCRMSecrets(cfg.CryptoEncryptSecretKeyPluginCRM, cfg.CryptoHashSecretKeyPluginCRM)
	service.SetDataSourceFactory(datasource.NewDataSourceFromConnectionWithLogger(logger))

	// Create ConnectionResolver based on multi-tenant mode
	dsRegistry := resolver.NewInternalDatasourceRegistry()

	if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
		resolverTMClient, resolverTMErr := initTenantManagerClient(cfg, logger)
		if resolverTMErr != nil {
			return nil, fmt.Errorf("create tenant manager client for resolver: %w", resolverTMErr)
		}

		tenantAdapter := resolver.NewTenantManagerAdapter(resolverTMClient)
		service.ConnectionResolver = resolver.NewMultiTenantResolver(connectionRepository, dsRegistry, tenantAdapter)
	} else {
		// Single-tenant: load internal datasource connections from DATASOURCE_* env vars.
		envConnections := resolver.LoadInternalConnectionsFromEnv(dsRegistry, logger)
		service.ConnectionResolver = resolver.NewSingleTenantResolver(connectionRepository, dsRegistry, envConnections)
	}

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
		mtConsumer, tmClient, mtCleanup, mtErr := initMultiTenantStack(ctx, cfg, logger, mongoManager, rabbitMQManager)
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

		// Discover existing tenants on startup so consumers start immediately.
		// Called AFTER handler registration so EnsureConsumerStarted can spawn
		// consumer goroutines for all registered queues.
		performInitialTenantSync(ctx, logger, tmClient, mtConsumer)

		return &Service{
			MultiQueueConsumer: multiQueueConsumer,
			Logger:             logger,
			licenseShutdown:    licenseClient.GetLicenseManagerShutdown(),
			mtCleanup:          mtCleanup,
		}, nil
	}

	multiQueueConsumer, err := initSingleTenantRabbitMQ(cfg, logger, telemetry, keyDeriver, cryptoWithExternalHMAC, service, mongoManager)
	if err != nil {
		return nil, err
	}

	return &Service{
		MultiQueueConsumer: multiQueueConsumer,
		Logger:             logger,
		licenseShutdown:    licenseClient.GetLicenseManagerShutdown(),
	}, nil
}

// initSingleTenantRabbitMQ initializes RabbitMQ consumer and publisher for single-tenant mode.
// Consumer and Publisher use SEPARATE connections to avoid channel interference.
func initSingleTenantRabbitMQ(
	cfg *Config,
	logger libLog.Logger,
	telemetry *libOtel.Telemetry,
	keyDeriver *crypto.HKDFKeyDeriver,
	cryptoWithExternalHMAC *crypto.HMACSigner,
	service *services.UseCase,
	mongoManager *tmmongo.Manager,
) (*MultiQueueConsumer, error) {
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

	return NewMultiQueueConsumer(consumerRoutes, service, cfg.RabbitMQGenerateReportQueue, logger, mongoManager), nil
}

// initObservability initializes the logger and telemetry pipeline.
func initObservability(cfg *Config) (libLog.Logger, *libOtel.Telemetry, error) {
	logger, err := newZapLogger(libZap.Config{
		Environment:     resolveZapEnvironment(cfg.EnvName),
		Level:           cfg.LogLevel,
		OTelLibraryName: cfg.OtelLibraryName,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	if err := validateMultiTenantConfig(cfg, logger); err != nil {
		return nil, nil, err
	}

	telemetry, err := newTelemetry(libOtel.TelemetryConfig{
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
		return nil, nil, fmt.Errorf("initialize telemetry: %w", err)
	}

	if err := applyTelemetryGlobals(telemetry); err != nil {
		return nil, nil, err
	}

	return logger, telemetry, nil
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

		if cfg.MultiTenantRedisHost == "" {
			return fmt.Errorf("MULTI_TENANT_REDIS_HOST is required when MULTI_TENANT_ENABLED=true (used for tenant event-driven discovery)")
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

// initStorageRepository initializes the S3-compatible storage repository.
func initStorageRepository(ctx context.Context, cfg *Config) (portStorage.Repository, error) {
	return pkgStorage.NewRepository(ctx, pkgStorage.ProviderConfig{
		Provider:          pkgStorage.ProviderS3,
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

	if cfg.MongoDBParameters != "" {
		mongoSource += "/?" + cfg.MongoDBParameters
	}

	if cfg.MaxPoolSize <= 0 {
		cfg.MaxPoolSize = 100
	}

	mongoCfg := mongoDB.Config{
		URI:         mongoSource,
		Database:    cfg.MongoDBName,
		Logger:      logger,
		MaxPoolSize: uint64(cfg.MaxPoolSize),
	}

	if cfg.MongoTLSCACert != "" {
		mongoCfg.TLS = &mongoDB.TLSConfig{CACertBase64: cfg.MongoTLSCACert}
	}

	return newMongoClient(ctx, mongoCfg)
}

// logFileTTL logs the configured file TTL for storage.
func logFileTTL(logger libLog.Logger, cfg *Config) {
	if cfg.ObjectStorageTTL != "" {
		logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Files will expire after: %s", cfg.ObjectStorageTTL))
	} else {
		logger.Log(context.Background(), libLog.LevelInfo, "Files will be stored permanently (no TTL — use S3 lifecycle policies for expiration)")
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

	// Allow plaintext HTTP when explicitly configured via MULTI_TENANT_ALLOW_INSECURE_HTTP.
	if cfg.MultiTenantAllowInsecureHTTP {
		clientOpts = append(clientOpts, tmclient.WithAllowInsecureHTTP())
	}

	if cfg.MultiTenantTimeout > 0 {
		clientOpts = append(clientOpts, tmclient.WithTimeout(time.Duration(cfg.MultiTenantTimeout)*time.Second))
	}

	if cfg.MultiTenantCacheTTLSec > 0 {
		clientOpts = append(clientOpts, tmclient.WithCacheTTL(time.Duration(cfg.MultiTenantCacheTTLSec)*time.Second))
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

	if cfg.RabbitMQTLS {
		rabbitOpts = append(rabbitOpts, tmrabbitmq.WithTLS())
	}

	rabbitManager := tmrabbitmq.NewManager(tmClient, constant.ApplicationName, rabbitOpts...)

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Multi-tenant managers initialized: url=%s, module=%s", cfg.MultiTenantURL, constant.ModuleWorker))

	return mongoManager, rabbitManager, nil
}

// initMultiTenantStack creates the unified multi-tenant consumer stack:
// shared TenantCache, TenantLoader, EventDispatcher, MultiTenantConsumer,
// and TenantEventListener. The EventDispatcher is shared between the consumer
// and the event listener so that Redis Pub/Sub events reach
// MultiTenantConsumer.EnsureConsumerStarted via wireDispatcherCallbacks.
func initMultiTenantStack(
	ctx context.Context,
	cfg *Config,
	logger libLog.Logger,
	tenantMongoManager *tmmongo.Manager,
	rabbitMQManager *tmrabbitmq.Manager,
) (*tmconsumer.MultiTenantConsumer, *tmclient.Client, func(), error) {
	// 1. Create shared Tenant Manager client
	tmClient, err := initTenantManagerClient(cfg, logger)
	if err != nil {
		return nil, nil, nil, wrapBootstrapError("create tenant manager client for multi-tenant stack", err)
	}

	// 2. Create shared TenantCache and TenantLoader
	var cacheTTL time.Duration
	if cfg.MultiTenantCacheTTLSec > 0 {
		cacheTTL = time.Duration(cfg.MultiTenantCacheTTLSec) * time.Second
	}

	tenantCache := tenantcache.NewTenantCache()
	tenantLoader := tenantcache.NewTenantLoader(tmClient, tenantCache, constant.ApplicationName, cacheTTL, logger)

	// 3. Create ONE EventDispatcher with infrastructure managers
	var dispatcherOpts []tmevent.DispatcherOption

	dispatcherOpts = append(dispatcherOpts,
		tmevent.WithDispatcherLogger(logger),
		tmevent.WithCacheTTL(cacheTTL),
	)

	if tenantMongoManager != nil {
		dispatcherOpts = append(dispatcherOpts, tmevent.WithMongo(tenantMongoManager))
	}

	if rabbitMQManager != nil {
		dispatcherOpts = append(dispatcherOpts, tmevent.WithRabbitMQ(rabbitMQManager))
	}

	dispatcher := tmevent.NewEventDispatcher(
		tenantCache,
		tenantLoader,
		constant.ApplicationName,
		dispatcherOpts...,
	)

	// 4. Create MultiTenantConsumer with the shared dispatcher injected.
	// The consumer's constructor calls wireDispatcherCallbacks() which wires:
	//   - onTenantAdded  -> knownTenants + EnsureConsumerStarted
	//   - onTenantRemoved -> cancel goroutine + remove from knownTenants
	//   - cache sync (consumer uses same cache as dispatcher)
	mtConfig := tmconsumer.DefaultMultiTenantConfig()
	mtConfig.Service = constant.ApplicationName
	mtConfig.Environment = cfg.EnvName
	mtConfig.MultiTenantURL = cfg.MultiTenantURL
	mtConfig.ServiceAPIKey = cfg.MultiTenantServiceAPIKey
	mtConfig.PrefetchCount = constant.DefaultPrefetchCount
	mtConfig.AllowInsecureHTTP = cfg.MultiTenantAllowInsecureHTTP

	var consumerOpts []tmconsumer.Option

	if rabbitMQManager != nil {
		consumerOpts = append(consumerOpts, tmconsumer.WithRabbitMQ(rabbitMQManager))
	}

	consumerOpts = append(consumerOpts, tmconsumer.WithEventDispatcher(dispatcher))

	if tenantMongoManager != nil {
		consumerOpts = append(consumerOpts, tmconsumer.WithMongoManager(tenantMongoManager))
	}

	mtConsumer, err := tmconsumer.NewMultiTenantConsumerWithError(
		mtConfig,
		logger,
		consumerOpts...,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create multi-tenant consumer: %w", err)
	}

	// 5. Wire restart recovery: when TenantLoader lazy-loads a tenant from the API
	// (e.g., on cache miss after restart), also ensure a consumer goroutine is started.
	tenantLoader.SetOnTenantLoaded(func(loadCtx context.Context, tenantID string) {
		mtConsumer.EnsureConsumerStarted(loadCtx, tenantID)
	})

	logger.Log(ctx, libLog.LevelInfo, "MultiTenantConsumer initialized with shared EventDispatcher and per-tenant vhost isolation")

	// 6. Create TenantEventListener (Redis Pub/Sub -> dispatcher.HandleEvent)
	var listenerCleanup func()

	redisPort := cfg.MultiTenantRedisPort
	if redisPort == "" {
		redisPort = "6379"
	}

	redisOpts := &redis.Options{
		Addr:     net.JoinHostPort(cfg.MultiTenantRedisHost, redisPort),
		Password: cfg.MultiTenantRedisPassword,
	}

	if cfg.MultiTenantRedisTLS {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}

		if cfg.MultiTenantRedisCACert != "" {
			caCert, err := base64.StdEncoding.DecodeString(cfg.MultiTenantRedisCACert)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to decode multi-tenant Redis CA certificate: %w", err)
			}

			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(caCert) {
				return nil, nil, nil, fmt.Errorf("failed to parse multi-tenant Redis CA certificate")
			}

			tlsCfg.RootCAs = pool
		}

		redisOpts.TLSConfig = tlsCfg
	}

	redisClient := redis.NewClient(redisOpts)

	listener, listenerErr := tmevent.NewTenantEventListener(
		redisClient,
		dispatcher.HandleEvent,
		tmevent.WithListenerLogger(logger),
		tmevent.WithService(constant.ApplicationName),
	)
	if listenerErr != nil {
		return nil, nil, nil, wrapBootstrapError("create worker tenant event listener", listenerErr)
	}

	if startErr := listener.Start(context.Background()); startErr != nil {
		return nil, nil, nil, wrapBootstrapError("start worker tenant event listener", startErr)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Worker multi-tenant event listener started: redis=%s", net.JoinHostPort(cfg.MultiTenantRedisHost, redisPort)))

	listenerCleanup = func() {
		logger.Log(context.Background(), libLog.LevelInfo, "Stopping worker multi-tenant event listener")

		if stopErr := listener.Stop(); stopErr != nil {
			logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to stop worker tenant event listener: %v", stopErr))
		}

		if closeErr := redisClient.Close(); closeErr != nil {
			logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to close worker tenant event Redis client: %v", closeErr))
		}
	}

	// Compose cleanup: event listener resources
	// Note: mtConsumer.Close() is handled by MultiQueueConsumer.Run() on context cancellation.
	cleanup := listenerCleanup

	return mtConsumer, tmClient, cleanup, nil
}

// performInitialTenantSync fetches all active tenants from the Tenant Manager API
// and calls EnsureConsumerStarted for each one. This ensures the worker starts
// consuming messages for all known tenants immediately on startup, rather than
// waiting for a Redis Pub/Sub event or lazy-load trigger.
func performInitialTenantSync(
	ctx context.Context,
	logger libLog.Logger,
	tmClient *tmclient.Client,
	mtConsumer *tmconsumer.MultiTenantConsumer,
) {
	if tmClient == nil {
		logger.Log(ctx, libLog.LevelWarn,
			"Initial tenant sync skipped: tenant manager client is nil")

		return
	}

	tenants, err := tmClient.GetActiveTenantsByService(ctx, constant.ApplicationName)
	if err != nil {
		logger.Log(ctx, libLog.LevelWarn,
			"Initial tenant sync failed; tenants will be discovered via events or lazy-load",
			libLog.Err(err))

		return
	}

	for _, t := range tenants {
		logger.Log(ctx, libLog.LevelDebug, "Initial tenant sync: starting consumer",
			libLog.String("tenant_id", t.ID),
			libLog.String("tenant_name", t.Name),
			libLog.String("tenant_status", t.Status))

		mtConsumer.EnsureConsumerStarted(ctx, t.ID)
	}

	logger.Log(ctx, libLog.LevelInfo, "Initial tenant sync completed",
		libLog.Int("tenant_count", len(tenants)))
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
