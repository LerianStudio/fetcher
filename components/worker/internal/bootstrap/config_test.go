package bootstrap

import (
	"context"
	"errors"
	"testing"

	workerRabbitMQ "github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	pkgRabbitMQ "github.com/LerianStudio/fetcher/pkg/rabbitmq"
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	mongoDB "github.com/LerianStudio/lib-commons/v5/commons/mongo"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
	libZap "github.com/LerianStudio/lib-observability/zap"
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

func TestInitJobEventEmitter_DisabledFailsStartup(t *testing.T) {
	t.Setenv("STREAMING_ENABLED", "false")
	t.Setenv("STREAMING_BROKERS", "")
	t.Setenv("STREAMING_CLOUDEVENTS_SOURCE", "")

	emitter, enabled, err := initJobEventEmitter(context.Background(), &Config{}, testBootstrapLogger(), &libOtel.Telemetry{}, nil)
	require.Error(t, err)
	assert.Nil(t, emitter)
	assert.False(t, enabled)
	assert.Contains(t, err.Error(), "STREAMING_ENABLED=true is required")
}

func TestInitJobEventEmitter_EnabledSupportsSingleTenant(t *testing.T) {
	t.Setenv("STREAMING_ENABLED", "true")
	t.Setenv("STREAMING_BROKERS", "broker:9092")
	t.Setenv("STREAMING_CLOUDEVENTS_SOURCE", "//lerian.fetcher/worker")

	telemetry, err := libOtel.NewTelemetry(libOtel.TelemetryConfig{Logger: testBootstrapLogger()})
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adapter := pkgRabbitMQ.NewMockAdapter(ctrl)
	publisher := workerRabbitMQ.NewPublisherRoutesWithAdapter(adapter, testBootstrapLogger(), telemetry)

	emitter, enabled, err := initJobEventEmitter(context.Background(), &Config{RabbitMQJobEventsExchange: "job.events", OtelLibraryName: "test"}, testBootstrapLogger(), telemetry, publisher)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.NotNil(t, emitter)
	require.NoError(t, emitter.Close())
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

	_, _ = initMongoConnection(context.Background(), cfg, testBootstrapLogger())

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

	_, _ = initMongoConnection(context.Background(), cfg, testBootstrapLogger())

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
