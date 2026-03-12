package bootstrap

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/components/worker/internal/services"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/mongodb/connection"

	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	simpleClient "github.com/LerianStudio/fetcher/pkg/seaweedfs"
	"github.com/LerianStudio/fetcher/pkg/seaweedfs/external"

	libZapV2 "github.com/LerianStudio/lib-commons/v2/commons/zap"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libMongo "github.com/LerianStudio/lib-commons/v4/commons/mongo"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libRabbitMQ "github.com/LerianStudio/lib-commons/v4/commons/rabbitmq"
	libZap "github.com/LerianStudio/lib-commons/v4/commons/zap"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
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
}

var (
	setConfigFromEnvVars  = libCommons.SetConfigFromEnvVars
	newZapLogger          = func(cfg libZap.Config) (libLog.Logger, error) { return libZap.New(cfg) }
	newTelemetry          = libOtel.NewTelemetry
	applyTelemetryGlobals = func(telemetry *libOtel.Telemetry) error {
		return telemetry.ApplyGlobals()
	}
)

// InitWorker initializes and configures the application's dependencies and returns the Service instance.
func InitWorker() *Service {
	cfg := &Config{}
	if err := setConfigFromEnvVars(cfg); err != nil {
		panic(err)
	}

	logger, err := newZapLogger(libZap.Config{
		Environment:     resolveZapEnvironment(cfg.EnvName),
		Level:           cfg.LogLevel,
		OTelLibraryName: cfg.OtelLibraryName,
	})
	if err != nil {
		panic(err)
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
		panic(err)
	}

	if err := applyTelemetryGlobals(telemetry); err != nil {
		panic(err)
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
	must("decode master encryption key", err)

	keyDeriver, err := crypto.NewHKDFKeyDeriver(masterKey)
	must("initialize key deriver", err)

	logger.Log(context.Background(), libLog.LevelInfo, "Key derivation initialized successfully")

	// Init crypto service with derived credential key
	cryptoService, errCrypto := crypto.NewAESGCMService(keyDeriver.GetCredentialKey(), cfg.AppEncryptionKeyVersion)
	must("initialize crypto service", errCrypto)

	// Init message signer for RabbitMQ with derived internal HMAC key
	cryptoWithInternalHMAC, errSigner := crypto.NewHMACSigner(keyDeriver.GetInternalHMACKey(), crypto.SignatureVersion)
	must("initialize message signer", errSigner)

	// Init document signer for external verification with derived external HMAC key
	cryptoWithExternalHMAC, errSigner := crypto.NewHMACSigner(keyDeriver.GetExternalHMACKey(), crypto.SignatureVersion)
	must("initialize document signer", errSigner)

	// Initialize RabbitMQ consumer and publisher with separate connections
	consumerRoutes := rabbitmq.NewConsumerRoutes(consumerConnection, cfg.RabbitMQNumWorkers, logger, telemetry, cryptoWithInternalHMAC)
	publisherRoutes := rabbitmq.NewPublisherRoutes(publisherConnection, logger, telemetry, cryptoWithExternalHMAC)

	// Config SeaweedFS connection
	seaweedFSEndpoint := fmt.Sprintf("http://%s:%s", cfg.SeaweedFSHost, cfg.SeaweedFSFilerPort)
	seaweedFSClient := simpleClient.NewSeaweedFSClient(seaweedFSEndpoint)

	// Init mongo DB connection
	escapedPass := url.QueryEscape(cfg.MongoDBPassword)
	mongoSource := fmt.Sprintf("%s://%s:%s@%s:%s",
		cfg.MongoURI, cfg.MongoDBUser, escapedPass, cfg.MongoDBHost, cfg.MongoDBPort)

	if cfg.MaxPoolSize <= 0 {
		cfg.MaxPoolSize = 100
	}

	mongoConnection, errConnectMongo := libMongo.NewClient(context.Background(), libMongo.Config{
		URI:         mongoSource,
		Database:    cfg.MongoDBName,
		Logger:      logger,
		MaxPoolSize: uint64(cfg.MaxPoolSize),
	})
	must("initialize MongoDB client", errConnectMongo)

	externalDataSeaweedFSRepository := external.NewSimpleRepository(seaweedFSClient, constant.ExternalDataBucketName)

	// Initialize MongoDB repositories
	jobRepository, errJobRepo := job.NewJobMongoDBRepository(mongoConnection, cfg.MongoDBName)
	must("initialize job repository", errJobRepo)

	connectionRepository, errConnectRepo := connection.NewConnectionMongoDBRepository(mongoConnection, cfg.MongoDBName)
	must("initialize connection repository", errConnectRepo)

	service := &services.UseCase{
		ExternalDataSeaweedFS: externalDataSeaweedFSRepository,
		JobRepository:         jobRepository,
		ConnectionRepository:  connectionRepository,
		Cryptor:               cryptoService,
		DocumentSigner:        cryptoWithExternalHMAC,
		FileTTL:               cfg.SeaweedFSTTL,
		RabbitMQPublisher:     publisherRoutes,
		JobEventsExchange:     cfg.RabbitMQJobEventsExchange,
	}

	if cfg.SeaweedFSTTL != "" {
		logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Reports will expire after: %s", cfg.SeaweedFSTTL))
	} else {
		logger.Log(context.Background(), libLog.LevelInfo, "Reports will be stored permanently (no TTL)")
	}

	licenseLogger := libZapV2.InitializeLogger()
	licenseClient := libLicense.NewLicenseClient(
		constant.ApplicationName,
		cfg.LicenseKey,
		cfg.OrganizationIDs,
		&licenseLogger,
	)
	multiQueueConsumer := NewMultiQueueConsumer(consumerRoutes, service)

	return &Service{
		MultiQueueConsumer: multiQueueConsumer,
		Logger:             logger,
		licenseShutdown:    licenseClient.GetLicenseManagerShutdown(),
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

func must(action string, err error) {
	if err != nil {
		panic(fmt.Errorf("%s: %w", action, err))
	}
}
