package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	workerRabbitMQ "github.com/LerianStudio/fetcher/v2/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/v2/pkg/bootstrap/readyz"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	pkgRabbitMQ "github.com/LerianStudio/fetcher/v2/pkg/rabbitmq"
	"github.com/LerianStudio/fetcher/v2/pkg/testutil"
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	mongoDB "github.com/LerianStudio/lib-commons/v5/commons/mongo"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
	libZap "github.com/LerianStudio/lib-observability/zap"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func TestFetcherOperationalMongoModule_UsesSharedFetcherModule(t *testing.T) {
	t.Parallel()

	assert.Equal(t, constant.ModuleFetcherOperationalState, fetcherOperationalMongoModule())
	assert.NotEqual(t, constant.ModuleWorker, fetcherOperationalMongoModule())
}

func TestWorkerMultiTenantConsumer_UsesConfiguredTenantManagerClientWithCircuitBreaker(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	cfg := &Config{
		MultiTenantURL:                      server.URL,
		MultiTenantServiceAPIKey:            "test-service-key",
		MultiTenantAllowInsecureHTTP:        true,
		MultiTenantCircuitBreakerThreshold:  1,
		MultiTenantCircuitBreakerTimeoutSec: 30,
		MultiTenantTimeout:                  1,
		MultiTenantCacheTTLSec:              1,
	}

	tmClient, err := initTenantManagerClient(cfg, testBootstrapLogger())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tmClient.Close()) })

	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		TenantClient: tmClient,
		Service:      constant.ApplicationName,
		Logger:       testBootstrapLogger(),
	})
	require.NotNil(t, consumer)
	assert.Same(t, tmClient, consumer.tenantClient)

	_, firstErr := tmClient.GetTenantConfig(testutil.TestContext(), "tenant-cb", constant.ApplicationName)
	require.Error(t, firstErr)

	_, secondErr := tmClient.GetTenantConfig(testutil.TestContext(), "tenant-cb", constant.ApplicationName)
	require.ErrorIs(t, secondErr, tmcore.ErrCircuitBreakerOpen)
}

func TestWorkerMultiTenantConsumer_RegisterRejectsEmptyQueue(t *testing.T) {
	t.Parallel()

	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		Service: constant.ApplicationName,
		Logger:  testBootstrapLogger(),
	})

	err := consumer.Register(" ", func(context.Context, amqp.Delivery) error { return nil })
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "queue name is required"))
}

func TestStreamingRabbitMQPublisher_UsesConfiguredRouteDestination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adapter := pkgRabbitMQ.NewMockAdapter(ctrl)
	adapter.EXPECT().
		ProducerDefault(gomock.Any(), "job.events", "job.completed", []byte(`{"status":"completed","metadata":{"source":"api"}}`), gomock.Any()).
		Return(nil)

	publisher := workerRabbitMQ.NewPublisherRoutesWithAdapter(adapter, testBootstrapLogger(), &libOtel.Telemetry{})
	target := streamingRabbitMQPublisher{publisher: publisher}

	err := target.Publish(context.Background(), "job.events", "job.completed", "application/json", []byte(`{"status":"completed","metadata":{"source":"api"}}`), nil)
	require.NoError(t, err)
}

func TestStreamingRabbitMQPublisher_PingReportsBrokerHealth(t *testing.T) {
	t.Parallel()

	t.Run("healthy broker propagates success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		adapter := pkgRabbitMQ.NewMockAdapter(ctrl)
		adapter.EXPECT().IsHealthy().Return(true)

		publisher := workerRabbitMQ.NewPublisherRoutesWithAdapter(adapter, testBootstrapLogger(), &libOtel.Telemetry{})
		target := streamingRabbitMQPublisher{publisher: publisher}

		require.NoError(t, target.Ping(context.Background()))
	})

	t.Run("unhealthy broker propagates failure", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		adapter := pkgRabbitMQ.NewMockAdapter(ctrl)
		adapter.EXPECT().IsHealthy().Return(false)

		publisher := workerRabbitMQ.NewPublisherRoutesWithAdapter(adapter, testBootstrapLogger(), &libOtel.Telemetry{})
		target := streamingRabbitMQPublisher{publisher: publisher}

		require.Error(t, target.Ping(context.Background()))
	})

	t.Run("nil publisher reports error", func(t *testing.T) {
		t.Parallel()

		target := streamingRabbitMQPublisher{}
		require.Error(t, target.Ping(context.Background()))
	})
}

func TestInitJobEventEmitter_DisabledFailsStartup(t *testing.T) {
	t.Setenv("STREAMING_ENABLED", "false")
	t.Setenv("STREAMING_BROKERS", "")
	t.Setenv("STREAMING_CLOUDEVENTS_SOURCE", "")

	emitter, enabled, err := initJobEventEmitter(testutil.TestContext(), &Config{}, testBootstrapLogger(), &libOtel.Telemetry{}, nil, nil)
	require.Error(t, err)
	assert.Nil(t, emitter)
	assert.False(t, enabled)
	assert.Contains(t, err.Error(), "STREAMING_ENABLED=true is required")
}

func TestInitJobEventEmitter_EnabledRequiresOutboxRepository(t *testing.T) {
	t.Setenv("STREAMING_ENABLED", "true")
	t.Setenv("STREAMING_BROKERS", "broker:9092")
	t.Setenv("STREAMING_CLOUDEVENTS_SOURCE", "//lerian.fetcher/worker")

	telemetry, err := libOtel.NewTelemetry(libOtel.TelemetryConfig{Logger: testBootstrapLogger()})
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adapter := pkgRabbitMQ.NewMockAdapter(ctrl)
	publisher := workerRabbitMQ.NewPublisherRoutesWithAdapter(adapter, testBootstrapLogger(), telemetry)

	emitter, enabled, err := initJobEventEmitter(testutil.TestContext(), &Config{RabbitMQJobEventsExchange: "job.events", OtelLibraryName: "test"}, testBootstrapLogger(), telemetry, publisher, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "streaming outbox repository is required")
	assert.False(t, enabled)
	assert.Nil(t, emitter)
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
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assert.Contains(t, err.Error(), "config load failed")
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
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assert.Contains(t, err.Error(), "logger init failed")
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
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assert.Contains(t, err.Error(), "telemetry init failed")
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
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assert.Contains(t, err.Error(), "decode master encryption key")
}

func TestValidateMultiTenantConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		wantErr   bool
		errSubstr string
	}{
		{
			name: "multi-tenant enabled without tenant manager URL returns error",
			cfg: &Config{
				MultiTenantEnabled:       true,
				MultiTenantURL:           "",
				MultiTenantServiceAPIKey: "test-api-key",
				MultiTenantRedisHost:     "localhost",
			},
			wantErr:   true,
			errSubstr: "MULTI_TENANT_URL is required",
		},
		{
			name: "multi-tenant enabled without service API key returns error",
			cfg: &Config{
				MultiTenantEnabled:       true,
				MultiTenantURL:           "http://tenant-manager:8080",
				MultiTenantServiceAPIKey: "",
				MultiTenantRedisHost:     "localhost",
			},
			wantErr:   true,
			errSubstr: "MULTI_TENANT_SERVICE_API_KEY is required",
		},
		{
			name: "multi-tenant enabled without Redis returns error",
			cfg: &Config{
				MultiTenantEnabled:       true,
				MultiTenantURL:           "http://tenant-manager:8080",
				MultiTenantServiceAPIKey: "test-api-key",
				MultiTenantRedisHost:     "",
			},
			wantErr:   true,
			errSubstr: "MULTI_TENANT_REDIS_HOST is required",
		},
		{
			name: "multi-tenant enabled with all required config succeeds",
			cfg: &Config{
				MultiTenantEnabled:       true,
				MultiTenantURL:           "http://tenant-manager:8080",
				MultiTenantServiceAPIKey: "test-api-key",
				MultiTenantRedisHost:     "redis-host",
			},
			wantErr: false,
		},
		{
			name: "multi-tenant disabled succeeds without Redis",
			cfg: &Config{
				MultiTenantEnabled:   false,
				MultiTenantRedisHost: "",
			},
			wantErr: false,
		},
	}

	logger := testBootstrapLogger()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMultiTenantConfig(tt.cfg, logger)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestResolvedMaxTenantPools(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected int
	}{
		{
			name:     "zero uses default",
			cfg:      &Config{MultiTenantMaxTenantPools: 0},
			expected: defaultMaxTenantPools,
		},
		{
			name:     "negative uses default",
			cfg:      &Config{MultiTenantMaxTenantPools: -1},
			expected: defaultMaxTenantPools,
		},
		{
			name:     "positive uses configured value",
			cfg:      &Config{MultiTenantMaxTenantPools: 50},
			expected: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, resolvedMaxTenantPools(tt.cfg))
		})
	}
}

func TestInitMongoConnection_PassesTLSConfig(t *testing.T) {
	originalMongoClient := newMongoClient
	t.Cleanup(func() { newMongoClient = originalMongoClient })

	var capturedConfig mongoDB.Config

	newMongoClient = func(ctx context.Context, cfg mongoDB.Config, opts ...mongoDB.Option) (*mongoDB.Client, error) {
		capturedConfig = cfg
		return nil, errors.New("intentional stop after capturing config")
	}

	cfg := &Config{
		MongoURI:        "mongodb",
		MongoDBHost:     "localhost",
		MongoDBName:     "testdb",
		MongoDBUser:     "user",
		MongoDBPassword: "pass",
		MongoDBPort:     "27017",
		MaxPoolSize:     100,
		MongoTLSCACert:  "dGVzdC1jYS1jZXJ0",
	}

	_, _ = initMongoConnection(testutil.TestContext(), cfg, testBootstrapLogger())

	require.NotNil(t, capturedConfig.TLS, "TLS config should be set when MongoTLSCACert is non-empty")
	assert.Equal(t, "dGVzdC1jYS1jZXJ0", capturedConfig.TLS.CACertBase64)
}

func TestInitMongoConnection_NoTLSWhenCertEmpty(t *testing.T) {
	originalMongoClient := newMongoClient
	t.Cleanup(func() { newMongoClient = originalMongoClient })

	var capturedConfig mongoDB.Config

	newMongoClient = func(ctx context.Context, cfg mongoDB.Config, opts ...mongoDB.Option) (*mongoDB.Client, error) {
		capturedConfig = cfg
		return nil, errors.New("intentional stop after capturing config")
	}

	cfg := &Config{
		MongoURI:        "mongodb",
		MongoDBHost:     "localhost",
		MongoDBName:     "testdb",
		MongoDBUser:     "user",
		MongoDBPassword: "pass",
		MongoDBPort:     "27017",
		MaxPoolSize:     100,
		MongoTLSCACert:  "",
	}

	_, _ = initMongoConnection(testutil.TestContext(), cfg, testBootstrapLogger())

	assert.Nil(t, capturedConfig.TLS, "TLS config should be nil when MongoTLSCACert is empty")
}

func TestConfig_MultiTenantRedisTLS(t *testing.T) {
	t.Setenv("MULTI_TENANT_REDIS_TLS", "true")

	cfg := &Config{}
	err := libCommons.SetConfigFromEnvVars(cfg)
	require.NoError(t, err, "Failed to load config")

	assert.True(t, cfg.MultiTenantRedisTLS, "MultiTenantRedisTLS should be true")
}

// Dummy usage of context to avoid import issues
var _ = context.Background

// --- Gate 4 of ring:dev-readyz --------------------------------------------
//
// Worker-side integration tests: ValidateSaaSTLS MUST fire before
// initStorageRepository opens the S3 client, and its failure must stop
// InitWorker with a readyz.ErrSaaSTLSRequired-wrapped error.
//
// Because InitWorker composes many dependencies inline, we stub only the
// minimum needed to drive execution up to the Gate 4 call — logger,
// telemetry, crypto — then assert that bootstrap refuses to proceed.

func TestInitWorker_SaaSMode_RefusesNonTLSS3(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	originalNewTelemetry := newTelemetry
	originalApplyTelemetryGlobals := applyTelemetryGlobals
	originalDecodeMasterKey := decodeMasterKey
	originalNewKeyDeriver := newKeyDeriver

	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
		newTelemetry = originalNewTelemetry
		applyTelemetryGlobals = originalApplyTelemetryGlobals
		decodeMasterKey = originalDecodeMasterKey
		newKeyDeriver = originalNewKeyDeriver
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "error"
		cfg.DeploymentMode = "saas"
		// Non-TLS S3 endpoint triggers the Gate 4 failure.
		cfg.ObjectStorageEndpoint = "http://minio:9000"

		return nil
	}
	newZapLogger = func(libZap.Config) (libLog.Logger, error) { return testBootstrapLogger(), nil }
	newTelemetry = func(libOtel.TelemetryConfig) (*libOtel.Telemetry, error) { return &libOtel.Telemetry{}, nil }
	applyTelemetryGlobals = func(*libOtel.Telemetry) error { return nil }
	// Provide a master key long enough to satisfy decodeMasterKey + keyDeriver.
	decodeMasterKey = func(string) ([]byte, error) { return make([]byte, 32), nil }

	service, err := InitWorker()
	assert.Nil(t, service)
	require.Error(t, err)
	require.ErrorIs(t, err, readyz.ErrSaaSTLSRequired,
		"worker must refuse to start when DEPLOYMENT_MODE=saas and S3 is non-TLS")
	assert.Contains(t, err.Error(), "s3")
}

func TestInitWorker_SaaSMode_RefusesNonTLSMongo(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	originalNewTelemetry := newTelemetry
	originalApplyTelemetryGlobals := applyTelemetryGlobals
	originalDecodeMasterKey := decodeMasterKey
	originalNewKeyDeriver := newKeyDeriver

	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
		newTelemetry = originalNewTelemetry
		applyTelemetryGlobals = originalApplyTelemetryGlobals
		decodeMasterKey = originalDecodeMasterKey
		newKeyDeriver = originalNewKeyDeriver
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "error"
		cfg.DeploymentMode = "saas"
		// MongoURI "mongodb" (plaintext) triggers the Gate 4 failure before
		// Mongo is ever dialed.
		cfg.MongoURI = "mongodb"
		cfg.MongoDBHost = "host"
		cfg.MongoDBPort = "27017"
		// Keep S3 empty so the Mongo check fires first per stable order.
		cfg.ObjectStorageEndpoint = ""

		return nil
	}
	newZapLogger = func(libZap.Config) (libLog.Logger, error) { return testBootstrapLogger(), nil }
	newTelemetry = func(libOtel.TelemetryConfig) (*libOtel.Telemetry, error) { return &libOtel.Telemetry{}, nil }
	applyTelemetryGlobals = func(*libOtel.Telemetry) error { return nil }
	decodeMasterKey = func(string) ([]byte, error) { return make([]byte, 32), nil }

	service, err := InitWorker()
	assert.Nil(t, service)
	require.Error(t, err)
	require.ErrorIs(t, err, readyz.ErrSaaSTLSRequired)
	assert.Contains(t, err.Error(), "mongodb")
}

func TestBuildSaaSTLSConfig_WorkerClaimsS3(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		DeploymentMode:               "saas",
		MongoURI:                     "mongodb+srv",
		MongoDBHost:                  "host",
		MongoDBPort:                  "27017",
		MongoDBUser:                  "user",
		MongoDBPassword:              "pass",
		MultiTenantRedisHost:         "mt-redis",
		MultiTenantRedisPort:         "6379",
		MultiTenantRedisTLS:          true,
		RabbitURI:                    "amqps",
		RabbitMQHost:                 "rabbit",
		RabbitMQPortAMQP:             "5671",
		RabbitMQUser:                 "user",
		RabbitMQPass:                 "pass",
		ObjectStorageEndpoint:        "https://s3.amazonaws.com",
		MultiTenantURL:               "https://tm",
		MultiTenantEnabled:           true,
		MultiTenantAllowInsecureHTTP: false,
	}

	got := buildSaaSTLSConfig(cfg, true)
	assert.Equal(t, "saas", got.DeploymentMode)
	assert.True(t, got.HasS3, "worker must claim S3 ownership")
	assert.Equal(t, "https://s3.amazonaws.com", got.S3Endpoint)
	assert.Equal(t, "rediss://mt-redis:6379", got.MultiTenantRedisURL)
	assert.Equal(t, "", got.RedisURL, "worker has no direct Redis dep")
	assert.Equal(t, "https://tm", got.TenantManagerURL)
	assert.False(t, got.AllowInsecureHTTPTM)
}

func TestBuildWorkerMongoURI_EmptyWhenURISchemeMissing(t *testing.T) {
	t.Parallel()

	cfg := &Config{MongoURI: ""}
	assert.Equal(t, "", buildWorkerMongoURI(cfg),
		"empty scheme must yield empty URI so Gate 4 skips the dep")
}

func TestBuildWorkerRabbitMQURL_EmptyWhenURISchemeMissing(t *testing.T) {
	t.Parallel()

	cfg := &Config{RabbitURI: ""}
	assert.Equal(t, "", buildWorkerRabbitMQURL(cfg),
		"empty scheme must yield empty URL so Gate 4 skips the dep")
}
