package bootstrap

import (
	"context"
	"errors"
	"testing"

	workerRabbitMQ "github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	workerCrypto "github.com/LerianStudio/fetcher/pkg/crypto"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libMongo "github.com/LerianStudio/lib-commons/v4/commons/mongo"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libRabbitMQ "github.com/LerianStudio/lib-commons/v4/commons/rabbitmq"
	libZap "github.com/LerianStudio/lib-commons/v4/commons/zap"
)

func testBootstrapLogger() *libLog.GoLogger {
	return &libLog.GoLogger{Level: libLog.LevelError}
}

func TestResolveZapEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  libZap.Environment
	}{
		{name: "production aliases", input: " PROD ", want: libZap.EnvironmentProduction},
		{name: "staging alias", input: "stage", want: libZap.EnvironmentStaging},
		{name: "uat alias", input: "UAT", want: libZap.EnvironmentUAT},
		{name: "development alias", input: "development", want: libZap.EnvironmentDevelopment},
		{name: "local alias", input: "local", want: libZap.EnvironmentLocal},
		{name: "unknown defaults to local", input: "qa", want: libZap.EnvironmentLocal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := resolveZapEnvironment(tt.input); got != tt.want {
				t.Fatalf("resolveZapEnvironment(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWrapBootstrapError(t *testing.T) {
	t.Parallel()

	if err := wrapBootstrapError("noop", nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	err := wrapBootstrapError("decode key", errors.New("boom"))
	if err == nil {
		t.Fatal("expected wrapped error, got nil")
	}
	if got := err.Error(); got != "decode key: boom" {
		t.Fatalf("unexpected wrapped error: %s", got)
	}
}

func TestInitWorker_ReturnsErrorWhenConfigLoadFails(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	t.Cleanup(func() { setConfigFromEnvVars = originalSetConfigFromEnvVars })

	setConfigFromEnvVars = func(any) error {
		return errors.New("config load failed")
	}

	service, err := InitWorker()
	if service != nil {
		t.Fatalf("expected nil service, got %#v", service)
	}
	if err == nil || err.Error() != "config load failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitWorker_ReturnsErrorWhenLoggerInitFails(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "debug"
		return nil
	}

	newZapLogger = func(libZap.Config) (libLog.Logger, error) {
		return nil, errors.New("logger init failed")
	}

	service, err := InitWorker()
	if service != nil {
		t.Fatalf("expected nil service, got %#v", service)
	}
	if err == nil || err.Error() != "logger init failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitWorker_ReturnsErrorWhenTelemetryInitFails(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	originalNewTelemetry := newTelemetry
	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
		newTelemetry = originalNewTelemetry
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "debug"
		return nil
	}

	newZapLogger = func(libZap.Config) (libLog.Logger, error) {
		return testBootstrapLogger(), nil
	}

	newTelemetry = func(libOtel.TelemetryConfig) (*libOtel.Telemetry, error) {
		return nil, errors.New("telemetry init failed")
	}

	service, err := InitWorker()
	if service != nil {
		t.Fatalf("expected nil service, got %#v", service)
	}
	if err == nil || err.Error() != "telemetry init failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitWorker_ReturnsErrorWhenTelemetryGlobalsFail(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	originalNewTelemetry := newTelemetry
	originalApplyTelemetryGlobals := applyTelemetryGlobals
	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
		newTelemetry = originalNewTelemetry
		applyTelemetryGlobals = originalApplyTelemetryGlobals
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "debug"
		return nil
	}

	newZapLogger = func(libZap.Config) (libLog.Logger, error) {
		return testBootstrapLogger(), nil
	}

	newTelemetry = func(libOtel.TelemetryConfig) (*libOtel.Telemetry, error) {
		return &libOtel.Telemetry{}, nil
	}

	applyTelemetryGlobals = func(*libOtel.Telemetry) error {
		return errors.New("apply globals failed")
	}

	service, err := InitWorker()
	if service != nil {
		t.Fatalf("expected nil service, got %#v", service)
	}
	if err == nil || err.Error() != "apply globals failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitWorker_ReturnsErrorWhenCryptoInitFails(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	originalNewTelemetry := newTelemetry
	originalApplyTelemetryGlobals := applyTelemetryGlobals
	originalDecodeMasterKey := decodeMasterKey
	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
		newTelemetry = originalNewTelemetry
		applyTelemetryGlobals = originalApplyTelemetryGlobals
		decodeMasterKey = originalDecodeMasterKey
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "debug"
		return nil
	}
	newZapLogger = func(libZap.Config) (libLog.Logger, error) { return testBootstrapLogger(), nil }
	newTelemetry = func(libOtel.TelemetryConfig) (*libOtel.Telemetry, error) { return &libOtel.Telemetry{}, nil }
	applyTelemetryGlobals = func(*libOtel.Telemetry) error { return nil }
	decodeMasterKey = func(string) ([]byte, error) { return nil, errors.New("bad key") }

	service, err := InitWorker()
	if service != nil {
		t.Fatalf("expected nil service, got %#v", service)
	}
	if err == nil || err.Error() != "decode master encryption key: bad key" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitWorker_ReturnsErrorWhenMongoClientFails(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	originalNewTelemetry := newTelemetry
	originalApplyTelemetryGlobals := applyTelemetryGlobals
	originalDecodeMasterKey := decodeMasterKey
	originalNewKeyDeriver := newKeyDeriver
	originalNewWorkerCryptoService := newWorkerCryptoService
	originalNewWorkerHMACSigner := newWorkerHMACSigner
	originalNewWorkerMongoClient := newWorkerMongoClient
	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
		newTelemetry = originalNewTelemetry
		applyTelemetryGlobals = originalApplyTelemetryGlobals
		decodeMasterKey = originalDecodeMasterKey
		newKeyDeriver = originalNewKeyDeriver
		newWorkerCryptoService = originalNewWorkerCryptoService
		newWorkerHMACSigner = originalNewWorkerHMACSigner
		newWorkerMongoClient = originalNewWorkerMongoClient
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "debug"
		cfg.MaxPoolSize = 10
		return nil
	}
	newZapLogger = func(libZap.Config) (libLog.Logger, error) { return testBootstrapLogger(), nil }
	newTelemetry = func(libOtel.TelemetryConfig) (*libOtel.Telemetry, error) { return &libOtel.Telemetry{}, nil }
	applyTelemetryGlobals = func(*libOtel.Telemetry) error { return nil }
	decodeMasterKey = func(string) ([]byte, error) { return make([]byte, 32), nil }
	newKeyDeriver = func([]byte) (*workerCrypto.HKDFKeyDeriver, error) {
		return workerCrypto.NewHKDFKeyDeriver(make([]byte, 32))
	}
	newWorkerCryptoService = func([]byte, string) (*workerCrypto.AESGCMService, error) {
		return workerCrypto.NewAESGCMService(make([]byte, 32), "v1")
	}
	newWorkerHMACSigner = func([]byte, string) (*workerCrypto.HMACSigner, error) {
		return workerCrypto.NewHMACSigner(make([]byte, 32), "v1")
	}
	newWorkerMongoClient = func(context.Context, libMongo.Config, ...libMongo.Option) (*libMongo.Client, error) {
		return nil, errors.New("mongo connect failed")
	}

	service, err := InitWorker()
	if service != nil {
		t.Fatalf("expected nil service, got %#v", service)
	}
	if err == nil || err.Error() != "initialize MongoDB client: mongo connect failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitWorker_ReturnsErrorWhenJobRepositoryFails(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	originalNewTelemetry := newTelemetry
	originalApplyTelemetryGlobals := applyTelemetryGlobals
	originalDecodeMasterKey := decodeMasterKey
	originalNewKeyDeriver := newKeyDeriver
	originalNewWorkerCryptoService := newWorkerCryptoService
	originalNewWorkerHMACSigner := newWorkerHMACSigner
	originalNewWorkerMongoClient := newWorkerMongoClient
	originalNewWorkerJobRepository := newWorkerJobRepository
	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
		newTelemetry = originalNewTelemetry
		applyTelemetryGlobals = originalApplyTelemetryGlobals
		decodeMasterKey = originalDecodeMasterKey
		newKeyDeriver = originalNewKeyDeriver
		newWorkerCryptoService = originalNewWorkerCryptoService
		newWorkerHMACSigner = originalNewWorkerHMACSigner
		newWorkerMongoClient = originalNewWorkerMongoClient
		newWorkerJobRepository = originalNewWorkerJobRepository
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "debug"
		cfg.MaxPoolSize = 10
		return nil
	}
	newZapLogger = func(libZap.Config) (libLog.Logger, error) { return testBootstrapLogger(), nil }
	newTelemetry = func(libOtel.TelemetryConfig) (*libOtel.Telemetry, error) { return &libOtel.Telemetry{}, nil }
	applyTelemetryGlobals = func(*libOtel.Telemetry) error { return nil }
	decodeMasterKey = func(string) ([]byte, error) { return make([]byte, 32), nil }
	newKeyDeriver = func([]byte) (*workerCrypto.HKDFKeyDeriver, error) {
		return workerCrypto.NewHKDFKeyDeriver(make([]byte, 32))
	}
	newWorkerCryptoService = func([]byte, string) (*workerCrypto.AESGCMService, error) {
		return workerCrypto.NewAESGCMService(make([]byte, 32), "v1")
	}
	newWorkerHMACSigner = func([]byte, string) (*workerCrypto.HMACSigner, error) {
		return workerCrypto.NewHMACSigner(make([]byte, 32), "v1")
	}
	newWorkerMongoClient = func(context.Context, libMongo.Config, ...libMongo.Option) (*libMongo.Client, error) {
		return &libMongo.Client{}, nil
	}
	newWorkerJobRepository = func(*libMongo.Client, string, ...jobRepo.RepositoryConfig) (*jobRepo.JobMongoDBRepository, error) {
		return nil, errors.New("job repo failed")
	}

	service, err := InitWorker()
	if service != nil {
		t.Fatalf("expected nil service, got %#v", service)
	}
	if err == nil || err.Error() != "initialize job repository: job repo failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitWorker_Success(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	originalNewTelemetry := newTelemetry
	originalApplyTelemetryGlobals := applyTelemetryGlobals
	originalDecodeMasterKey := decodeMasterKey
	originalNewKeyDeriver := newKeyDeriver
	originalNewWorkerCryptoService := newWorkerCryptoService
	originalNewWorkerHMACSigner := newWorkerHMACSigner
	originalNewWorkerMongoClient := newWorkerMongoClient
	originalNewWorkerJobRepository := newWorkerJobRepository
	originalNewWorkerConnectionRepo := newWorkerConnectionRepo
	originalNewWorkerConsumerRoutes := newWorkerConsumerRoutes
	originalNewWorkerPublisherRoutes := newWorkerPublisherRoutes
	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
		newTelemetry = originalNewTelemetry
		applyTelemetryGlobals = originalApplyTelemetryGlobals
		decodeMasterKey = originalDecodeMasterKey
		newKeyDeriver = originalNewKeyDeriver
		newWorkerCryptoService = originalNewWorkerCryptoService
		newWorkerHMACSigner = originalNewWorkerHMACSigner
		newWorkerMongoClient = originalNewWorkerMongoClient
		newWorkerJobRepository = originalNewWorkerJobRepository
		newWorkerConnectionRepo = originalNewWorkerConnectionRepo
		newWorkerConsumerRoutes = originalNewWorkerConsumerRoutes
		newWorkerPublisherRoutes = originalNewWorkerPublisherRoutes
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "debug"
		cfg.RabbitMQGenerateReportQueue = "fetcher.extract-external-data.queue"
		cfg.MaxPoolSize = 10
		return nil
	}
	newZapLogger = func(libZap.Config) (libLog.Logger, error) { return testBootstrapLogger(), nil }
	newTelemetry = func(libOtel.TelemetryConfig) (*libOtel.Telemetry, error) { return &libOtel.Telemetry{}, nil }
	applyTelemetryGlobals = func(*libOtel.Telemetry) error { return nil }
	decodeMasterKey = func(string) ([]byte, error) { return make([]byte, 32), nil }
	newKeyDeriver = func([]byte) (*workerCrypto.HKDFKeyDeriver, error) {
		return workerCrypto.NewHKDFKeyDeriver(make([]byte, 32))
	}
	newWorkerCryptoService = func([]byte, string) (*workerCrypto.AESGCMService, error) {
		return workerCrypto.NewAESGCMService(make([]byte, 32), "v1")
	}
	newWorkerHMACSigner = func([]byte, string) (*workerCrypto.HMACSigner, error) {
		return workerCrypto.NewHMACSigner(make([]byte, 32), "v1")
	}
	newWorkerMongoClient = func(context.Context, libMongo.Config, ...libMongo.Option) (*libMongo.Client, error) {
		return &libMongo.Client{}, nil
	}
	newWorkerJobRepository = func(*libMongo.Client, string, ...jobRepo.RepositoryConfig) (*jobRepo.JobMongoDBRepository, error) {
		return &jobRepo.JobMongoDBRepository{}, nil
	}
	newWorkerConnectionRepo = func(*libMongo.Client, string, ...connRepo.RepositoryConfig) (*connRepo.ConnectionMongoDBRepository, error) {
		return &connRepo.ConnectionMongoDBRepository{}, nil
	}
	newWorkerConsumerRoutes = func(_ *libRabbitMQ.RabbitMQConnection, _ int, logger libLog.Logger, telemetry *libOtel.Telemetry, _ workerCrypto.Signer) *workerRabbitMQ.ConsumerRoutes {
		return workerRabbitMQ.NewConsumerRoutesWithAdapter(nil, 1, logger, telemetry)
	}
	newWorkerPublisherRoutes = func(_ *libRabbitMQ.RabbitMQConnection, logger libLog.Logger, telemetry *libOtel.Telemetry, _ workerCrypto.Signer) *workerRabbitMQ.PublisherRoutes {
		return workerRabbitMQ.NewPublisherRoutesWithAdapter(nil, logger, telemetry)
	}

	service, err := InitWorker()
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if service == nil || service.MultiQueueConsumer == nil || service.Logger == nil {
		t.Fatalf("expected initialized service, got %#v", service)
	}
}
