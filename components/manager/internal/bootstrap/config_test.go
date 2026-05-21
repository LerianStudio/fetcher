package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	cacheRepo "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	portCache "github.com/LerianStudio/fetcher/pkg/ports/cache"
	redisCache "github.com/LerianStudio/fetcher/pkg/redis"
	libMongo "github.com/LerianStudio/lib-commons/v5/commons/mongo"
	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
	"github.com/LerianStudio/lib-observability/zap"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testManagerBootstrapLogger() *libLog.GoLogger {
	return &libLog.GoLogger{Level: libLog.LevelError}
}

type stubSchemaCacheStore struct{}

func (stubSchemaCacheStore) Get(context.Context, string) (model.DataSourceSchema, bool, error) {
	return model.DataSourceSchema{}, false, nil
}

func (stubSchemaCacheStore) Set(context.Context, string, model.DataSourceSchema, time.Duration) error {
	return nil
}

func (stubSchemaCacheStore) Delete(context.Context, string) error {
	return nil
}

func (stubSchemaCacheStore) Clear(context.Context) error {
	return nil
}

func (stubSchemaCacheStore) IsHealthy(context.Context) bool {
	return true
}

type closableSchemaCache struct {
	closed bool
}

var _ portCache.SchemaCacheRepository = (*closableSchemaCache)(nil)

func (*closableSchemaCache) Get(context.Context, string) (*model.DataSourceSchema, error) {
	return nil, nil
}

func (*closableSchemaCache) Set(context.Context, string, *model.DataSourceSchema, time.Duration) error {
	return nil
}
func (*closableSchemaCache) Delete(context.Context, string) error { return nil }
func (*closableSchemaCache) Clear(context.Context) error          { return nil }
func (*closableSchemaCache) IsHealthy(context.Context) bool       { return true }
func (c *closableSchemaCache) Close() error {
	c.closed = true
	return nil
}

func TestRegisterSchemaCacheCloseHook_ClosesCache(t *testing.T) {
	t.Parallel()

	cache := &closableSchemaCache{}
	hooks := []func(context.Context) error{}

	registerSchemaCacheCloseHook(&hooks, cache)

	require.Len(t, hooks, 1)
	require.NoError(t, hooks[0](context.Background()))
	assert.True(t, cache.closed)
}

func TestConfigStruct(t *testing.T) {
	cfg := &Config{EnvName: "test", ServerAddress: "localhost:8080", LogLevel: "info"}
	assert.Equal(t, "test", cfg.EnvName)
	assert.Equal(t, "localhost:8080", cfg.ServerAddress)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestResolveZapEnvironment(t *testing.T) {
	assert.Equal(t, zap.EnvironmentProduction, resolveZapEnvironment("prod"))
	assert.Equal(t, zap.EnvironmentStaging, resolveZapEnvironment("stage"))
	assert.Equal(t, zap.EnvironmentDevelopment, resolveZapEnvironment("dev"))
	assert.Equal(t, zap.EnvironmentLocal, resolveZapEnvironment("unknown"))
}

func TestWrapBootstrapError(t *testing.T) {
	assert.NoError(t, wrapBootstrapError("noop", nil))

	err := wrapBootstrapError("decode key", errors.New("boom"))
	require.Error(t, err)
	assert.EqualError(t, err, "decode key: boom")
}

func TestLoadConfig_ReturnsError(t *testing.T) {
	original := setConfigFromEnvVars
	t.Cleanup(func() { setConfigFromEnvVars = original })

	setConfigFromEnvVars = func(any) error {
		return errors.New("config load failed")
	}

	cfg, err := loadConfig()
	assert.Nil(t, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config load failed")
}

func TestInitLoggerAndTelemetry_ReturnsApplyGlobalsError(t *testing.T) {
	originalLogger := newManagerLogger
	originalTelemetry := newManagerTelemetry
	originalApply := applyTelemetryGlobals
	t.Cleanup(func() {
		newManagerLogger = originalLogger
		newManagerTelemetry = originalTelemetry
		applyTelemetryGlobals = originalApply
	})

	newManagerLogger = func(zap.Config) (libLog.Logger, error) {
		return testManagerBootstrapLogger(), nil
	}
	newManagerTelemetry = func(libOtel.TelemetryConfig) (*libOtel.Telemetry, error) {
		return &libOtel.Telemetry{}, nil
	}
	applyTelemetryGlobals = func(*libOtel.Telemetry) error {
		return errors.New("apply globals failed")
	}

	logger, telemetry, err := initLoggerAndTelemetry(&Config{EnvName: "local", LogLevel: "debug"})
	assert.Nil(t, logger)
	assert.Nil(t, telemetry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apply globals failed")
}

func TestInitCrypto_ReturnsWrappedError(t *testing.T) {
	deps, err := initCrypto(&Config{AppEncryptionKey: "invalid", AppEncryptionKeyVersion: "v1"}, testManagerBootstrapLogger())
	assert.Nil(t, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode master encryption key")
}

func TestInitPlatformDependencies_ReturnsCacheError(t *testing.T) {
	original := newSchemaCacheStore
	t.Cleanup(func() { newSchemaCacheStore = original })

	newSchemaCacheStore = func(redisCache.RedisConfig, libLog.Logger, time.Duration, string) (redisCache.Cache[model.DataSourceSchema], error) {
		return nil, errors.New("cache init failed")
	}

	deps, err := initPlatformDependencies(&Config{}, testManagerBootstrapLogger(), nil)
	assert.Nil(t, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initialize schema cache")
}

func TestInitServers_ReturnsAssembledService(t *testing.T) {
	originalLoadConfig := loadConfigFn
	originalLoggerAndTelemetry := initLoggerAndTelemetryFn
	originalMongoRepos := initMongoRepositoriesFn
	originalCrypto := initCryptoFn
	originalPlatform := initPlatformDependenciesFn
	originalAssemble := assembleServiceFn
	t.Cleanup(func() {
		loadConfigFn = originalLoadConfig
		initLoggerAndTelemetryFn = originalLoggerAndTelemetry
		initMongoRepositoriesFn = originalMongoRepos
		initCryptoFn = originalCrypto
		initPlatformDependenciesFn = originalPlatform
		assembleServiceFn = originalAssemble
	})

	expectedService := &Service{}
	loadConfigFn = func() (*Config, error) { return &Config{}, nil }
	initLoggerAndTelemetryFn = func(*Config) (libLog.Logger, *libOtel.Telemetry, error) {
		return testManagerBootstrapLogger(), &libOtel.Telemetry{}, nil
	}
	initMongoRepositoriesFn = func(context.Context, *Config, libLog.Logger) (*managerRepositories, error) {
		return &managerRepositories{}, nil
	}
	initCryptoFn = func(*Config, libLog.Logger) (*managerCrypto, error) {
		return &managerCrypto{service: &crypto.AESGCMService{}}, nil
	}
	initPlatformDependenciesFn = func(*Config, libLog.Logger, crypto.Signer) (*managerPlatformDependencies, error) {
		return &managerPlatformDependencies{}, nil
	}
	assembleServiceFn = func(*Config, libLog.Logger, *libOtel.Telemetry, *managerRepositories, *crypto.AESGCMService, *managerPlatformDependencies) (*Service, error) {
		return expectedService, nil
	}

	service, err := InitServers()
	require.NoError(t, err)
	assert.Same(t, expectedService, service)
}

func TestInitServers_PropagatesHelperErrors(t *testing.T) {
	originalLoadConfig := loadConfigFn
	originalLoggerAndTelemetry := initLoggerAndTelemetryFn
	originalMongoRepos := initMongoRepositoriesFn
	originalCrypto := initCryptoFn
	originalPlatform := initPlatformDependenciesFn
	originalAssemble := assembleServiceFn
	t.Cleanup(func() {
		loadConfigFn = originalLoadConfig
		initLoggerAndTelemetryFn = originalLoggerAndTelemetry
		initMongoRepositoriesFn = originalMongoRepos
		initCryptoFn = originalCrypto
		initPlatformDependenciesFn = originalPlatform
		assembleServiceFn = originalAssemble
	})

	loadConfigFn = func() (*Config, error) {
		return nil, errors.New("config load failed")
	}

	service, err := InitServers()
	assert.Nil(t, service)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config load failed")

	loadConfigFn = func() (*Config, error) {
		return &Config{}, nil
	}
	initLoggerAndTelemetryFn = func(*Config) (libLog.Logger, *libOtel.Telemetry, error) {
		return nil, nil, errors.New("logger init failed")
	}

	service, err = InitServers()
	assert.Nil(t, service)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logger init failed")

	loadConfigFn = func() (*Config, error) {
		return &Config{}, nil
	}
	initLoggerAndTelemetryFn = func(*Config) (libLog.Logger, *libOtel.Telemetry, error) {
		return testManagerBootstrapLogger(), &libOtel.Telemetry{}, nil
	}
	initMongoRepositoriesFn = func(context.Context, *Config, libLog.Logger) (*managerRepositories, error) {
		return &managerRepositories{}, nil
	}
	initCryptoFn = func(*Config, libLog.Logger) (*managerCrypto, error) {
		return &managerCrypto{service: &crypto.AESGCMService{}}, nil
	}
	initPlatformDependenciesFn = func(*Config, libLog.Logger, crypto.Signer) (*managerPlatformDependencies, error) {
		return &managerPlatformDependencies{}, nil
	}
	assembleServiceFn = func(*Config, libLog.Logger, *libOtel.Telemetry, *managerRepositories, *crypto.AESGCMService, *managerPlatformDependencies) (*Service, error) {
		return nil, errors.New("tenant middleware init failed")
	}

	service, err = InitServers()
	assert.Nil(t, service)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant middleware init failed")
}

func TestInitMongoRepositories_PassesTLSConfig(t *testing.T) {
	originalMongoClient := newManagerMongoClient
	t.Cleanup(func() { newManagerMongoClient = originalMongoClient })

	var capturedConfig libMongo.Config

	newManagerMongoClient = func(ctx context.Context, cfg libMongo.Config, opts ...libMongo.Option) (*libMongo.Client, error) {
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
		MongoTLSCACert:  "dGVzdC1jYS1jZXJ0",
	}

	_, _ = initMongoRepositories(context.Background(), cfg, testManagerBootstrapLogger())

	require.NotNil(t, capturedConfig.TLS, "TLS config should be set when MongoTLSCACert is non-empty")
	assert.Equal(t, "dGVzdC1jYS1jZXJ0", capturedConfig.TLS.CACertBase64)
}

func TestInitMongoRepositories_NoTLSWhenCertEmpty(t *testing.T) {
	originalMongoClient := newManagerMongoClient
	t.Cleanup(func() { newManagerMongoClient = originalMongoClient })

	var capturedConfig libMongo.Config

	newManagerMongoClient = func(ctx context.Context, cfg libMongo.Config, opts ...libMongo.Option) (*libMongo.Client, error) {
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
		MongoTLSCACert:  "",
	}

	_, _ = initMongoRepositories(context.Background(), cfg, testManagerBootstrapLogger())

	assert.Nil(t, capturedConfig.TLS, "TLS config should be nil when MongoTLSCACert is empty")
}

func TestConfig_LoadFromEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "loads service configuration",
			envVars: map[string]string{
				"ENV_NAME":       "staging",
				"SERVER_ADDRESS": "0.0.0.0:3000",
				"LOG_LEVEL":      "warn",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.EnvName != "staging" {
					t.Errorf("EnvName = %q, want %q", cfg.EnvName, "staging")
				}
				if cfg.ServerAddress != "0.0.0.0:3000" {
					t.Errorf("ServerAddress = %q, want %q", cfg.ServerAddress, "0.0.0.0:3000")
				}
				if cfg.LogLevel != "warn" {
					t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
				}
			},
		},
		{
			name: "loads MongoDB configuration",
			envVars: map[string]string{
				"MONGO_URI":      "mongodb",
				"MONGO_HOST":     "mongo-host",
				"MONGO_NAME":     "fetcher",
				"MONGO_USER":     "root",
				"MONGO_PASSWORD": "pass123",
				"MONGO_PORT":     "27017",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.MongoURI != "mongodb" {
					t.Errorf("MongoURI = %q, want %q", cfg.MongoURI, "mongodb")
				}
				if cfg.MongoDBHost != "mongo-host" {
					t.Errorf("MongoDBHost = %q, want %q", cfg.MongoDBHost, "mongo-host")
				}
				if cfg.MongoDBName != "fetcher" {
					t.Errorf("MongoDBName = %q, want %q", cfg.MongoDBName, "fetcher")
				}
				if cfg.MongoDBPassword != "pass123" {
					t.Errorf("MongoDBPassword = %q, want %q", cfg.MongoDBPassword, "pass123")
				}
			},
		},
		{
			name: "loads RabbitMQ configuration",
			envVars: map[string]string{
				"RABBITMQ_URI":                "amqp",
				"RABBITMQ_HOST":               "rabbit-host",
				"RABBITMQ_PORT_AMQP":          "5672",
				"RABBITMQ_DEFAULT_USER":       "rmq-user",
				"RABBITMQ_DEFAULT_PASS":       "rmq-pass",
				"RABBITMQ_FETCHER_WORK_QUEUE": "fetcher-queue",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.RabbitURI != "amqp" {
					t.Errorf("RabbitURI = %q, want %q", cfg.RabbitURI, "amqp")
				}
				if cfg.RabbitMQHost != "rabbit-host" {
					t.Errorf("RabbitMQHost = %q, want %q", cfg.RabbitMQHost, "rabbit-host")
				}
				if cfg.RabbitMQGenerateReportQueue != "fetcher-queue" {
					t.Errorf("RabbitMQGenerateReportQueue = %q, want %q", cfg.RabbitMQGenerateReportQueue, "fetcher-queue")
				}
			},
		},
		{
			name: "loads boolean fields",
			envVars: map[string]string{
				"ENABLE_TELEMETRY":    "true",
				"PLUGIN_AUTH_ENABLED": "true",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.EnableTelemetry {
					t.Error("EnableTelemetry should be true")
				}
				if !cfg.AuthEnabled {
					t.Error("AuthEnabled should be true")
				}
			},
		},
		{
			name: "loads Redis configuration",
			envVars: map[string]string{
				"REDIS_HOST":     "redis-host",
				"REDIS_PORT":     "6379",
				"REDIS_PASSWORD": "redis-pass",
				"REDIS_DB":       "2",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.RedisHost != "redis-host" {
					t.Errorf("RedisHost = %q, want %q", cfg.RedisHost, "redis-host")
				}
				if cfg.RedisPort != "6379" {
					t.Errorf("RedisPort = %q, want %q", cfg.RedisPort, "6379")
				}
				if cfg.RedisPassword != "redis-pass" {
					t.Errorf("RedisPassword = %q, want %q", cfg.RedisPassword, "redis-pass")
				}
				if cfg.RedisDB != "2" {
					t.Errorf("RedisDB = %q, want %q", cfg.RedisDB, "2")
				}
			},
		},
		{
			name: "loads MongoDB TLS CA cert configuration",
			envVars: map[string]string{
				"MONGO_TLS_CA_CERT": "dGVzdC1jYS1jZXJ0LWJhc2U2NA==",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.MongoTLSCACert != "dGVzdC1jYS1jZXJ0LWJhc2U2NA==" {
					t.Errorf("MongoTLSCACert = %q, want %q", cfg.MongoTLSCACert, "dGVzdC1jYS1jZXJ0LWJhc2U2NA==")
				}
			},
		},
		{
			name: "loads Redis TLS configuration",
			envVars: map[string]string{
				"REDIS_TLS":     "true",
				"REDIS_CA_CERT": "dGVzdC1jYS1jZXJ0",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.RedisTLS {
					t.Error("RedisTLS should be true")
				}
				if cfg.RedisCACert != "dGVzdC1jYS1jZXJ0" {
					t.Errorf("RedisCACert = %q, want %q", cfg.RedisCACert, "dGVzdC1jYS1jZXJ0")
				}
			},
		},
		{
			name:    "Redis TLS defaults to false when not set",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.RedisTLS {
					t.Error("RedisTLS should default to false")
				}
				if cfg.RedisCACert != "" {
					t.Errorf("RedisCACert should be empty, got %q", cfg.RedisCACert)
				}
			},
		},
		{
			name: "loads multi-tenant Redis TLS configuration",
			envVars: map[string]string{
				"MULTI_TENANT_REDIS_TLS": "true",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.MultiTenantRedisTLS {
					t.Error("MultiTenantRedisTLS should be true")
				}
			},
		},
		{
			name:    "empty env vars produce zero values",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.EnvName != "" {
					t.Errorf("EnvName should be empty, got %q", cfg.EnvName)
				}
				if cfg.ServerAddress != "" {
					t.Errorf("ServerAddress should be empty, got %q", cfg.ServerAddress)
				}
				if cfg.EnableTelemetry {
					t.Error("EnableTelemetry should default to false")
				}
				if cfg.AuthEnabled {
					t.Error("AuthEnabled should default to false")
				}
			},
		},
		{
			name: "loads multi-tenant configuration with all fields",
			envVars: map[string]string{
				"MULTI_TENANT_ENABLED":                     "true",
				"MULTI_TENANT_URL":                         "http://tenant-manager:8080",
				"MULTI_TENANT_REDIS_HOST":                  "redis-host",
				"MULTI_TENANT_REDIS_PORT":                  "6380",
				"MULTI_TENANT_REDIS_PASSWORD":              "redis-secret",
				"MULTI_TENANT_MAX_TENANT_POOLS":            "200",
				"MULTI_TENANT_IDLE_TIMEOUT_SEC":            "600",
				"MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD":   "10",
				"MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC": "60",
				"MULTI_TENANT_SERVICE_API_KEY":             "test-api-key-123",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.MultiTenantEnabled {
					t.Error("MultiTenantEnabled should be true")
				}
				if cfg.MultiTenantURL != "http://tenant-manager:8080" {
					t.Errorf("MultiTenantURL = %q, want %q", cfg.MultiTenantURL, "http://tenant-manager:8080")
				}
				if cfg.MultiTenantRedisHost != "redis-host" {
					t.Errorf("MultiTenantRedisHost = %q, want %q", cfg.MultiTenantRedisHost, "redis-host")
				}
				if cfg.MultiTenantRedisPort != "6380" {
					t.Errorf("MultiTenantRedisPort = %q, want %q", cfg.MultiTenantRedisPort, "6380")
				}
				if cfg.MultiTenantRedisPassword != "redis-secret" {
					t.Errorf("MultiTenantRedisPassword = %q, want %q", cfg.MultiTenantRedisPassword, "redis-secret")
				}
				if cfg.MultiTenantMaxTenantPools != 200 {
					t.Errorf("MultiTenantMaxTenantPools = %d, want 200", cfg.MultiTenantMaxTenantPools)
				}
				if cfg.MultiTenantIdleTimeoutSec != 600 {
					t.Errorf("MultiTenantIdleTimeoutSec = %d, want 600", cfg.MultiTenantIdleTimeoutSec)
				}
				if cfg.MultiTenantCircuitBreakerThreshold != 10 {
					t.Errorf("MultiTenantCircuitBreakerThreshold = %d, want 10", cfg.MultiTenantCircuitBreakerThreshold)
				}
				if cfg.MultiTenantCircuitBreakerTimeoutSec != 60 {
					t.Errorf("MultiTenantCircuitBreakerTimeoutSec = %d, want 60", cfg.MultiTenantCircuitBreakerTimeoutSec)
				}
				if cfg.MultiTenantServiceAPIKey != "test-api-key-123" {
					t.Errorf("MultiTenantServiceAPIKey = %q, want %q", cfg.MultiTenantServiceAPIKey, "test-api-key-123")
				}
			},
		},
		{
			name:    "multi-tenant defaults to disabled when env vars not set",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.MultiTenantEnabled {
					t.Error("MultiTenantEnabled should default to false")
				}
				if cfg.MultiTenantURL != "" {
					t.Errorf("MultiTenantURL should be empty, got %q", cfg.MultiTenantURL)
				}
				if cfg.MultiTenantRedisHost != "" {
					t.Errorf("MultiTenantRedisHost should be empty, got %q", cfg.MultiTenantRedisHost)
				}
				if cfg.MultiTenantMaxTenantPools != 0 {
					t.Errorf("MultiTenantMaxTenantPools should be 0, got %d", cfg.MultiTenantMaxTenantPools)
				}
				if cfg.MultiTenantIdleTimeoutSec != 0 {
					t.Errorf("MultiTenantIdleTimeoutSec should be 0, got %d", cfg.MultiTenantIdleTimeoutSec)
				}
				if cfg.MultiTenantCircuitBreakerThreshold != 0 {
					t.Errorf("MultiTenantCircuitBreakerThreshold should be 0, got %d", cfg.MultiTenantCircuitBreakerThreshold)
				}
				if cfg.MultiTenantCircuitBreakerTimeoutSec != 0 {
					t.Errorf("MultiTenantCircuitBreakerTimeoutSec should be 0, got %d", cfg.MultiTenantCircuitBreakerTimeoutSec)
				}
				if cfg.MultiTenantServiceAPIKey != "" {
					t.Errorf("MultiTenantServiceAPIKey should be empty, got %q", cfg.MultiTenantServiceAPIKey)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg := &Config{}
			if err := pkg.SetConfigFromEnvVars(cfg); err != nil {
				t.Fatalf("SetConfigFromEnvVars() error: %v", err)
			}

			tt.validate(t, cfg)
		})
	}
}

func TestGetSchemaCacheTTL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Duration
	}{
		{
			name:  "valid seconds",
			input: "300",
			want:  300 * time.Second,
		},
		{
			name:  "one second",
			input: "1",
			want:  1 * time.Second,
		},
		{
			name:  "empty string returns default",
			input: "",
			want:  cacheRepo.DefaultSchemaCacheTTL,
		},
		{
			name:  "invalid string returns default",
			input: "not-a-number",
			want:  cacheRepo.DefaultSchemaCacheTTL,
		},
		{
			name:  "zero seconds",
			input: "0",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSchemaCacheTTL(tt.input)
			if got != tt.want {
				t.Errorf("getSchemaCacheTTL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetRedisDB(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "valid db number",
			input: "3",
			want:  3,
		},
		{
			name:  "zero",
			input: "0",
			want:  0,
		},
		{
			name:  "empty string returns 0",
			input: "",
			want:  0,
		},
		{
			name:  "invalid string returns 0",
			input: "abc",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRedisDB(tt.input)
			if got != tt.want {
				t.Errorf("getRedisDB(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolvedMaxTenantPools(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want int
	}{
		{
			name: "configured value",
			cfg:  &Config{MultiTenantMaxTenantPools: 200},
			want: 200,
		},
		{
			name: "zero uses default",
			cfg:  &Config{MultiTenantMaxTenantPools: 0},
			want: defaultMaxTenantPools,
		},
		{
			name: "negative uses default",
			cfg:  &Config{MultiTenantMaxTenantPools: -1},
			want: defaultMaxTenantPools,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolvedMaxTenantPools(tt.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMultiTenantPublisher_ProducerDefault_RequiresTenantID(t *testing.T) {
	publisher := &multiTenantPublisher{
		manager: &stubRabbitMQManager{},
		logger:  testManagerBootstrapLogger(),
	}

	err := publisher.ProducerDefault(context.Background(), "", "test-queue", []byte(`{}`), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no tenant ID in context")
}

func TestMultiTenantPublisher_ProducerDefault_GetChannelError(t *testing.T) {
	publisher := &multiTenantPublisher{
		manager: &stubRabbitMQManager{err: errors.New("connection refused")},
		logger:  testManagerBootstrapLogger(),
	}

	ctx := stubTenantContext("tenant-1")

	err := publisher.ProducerDefault(ctx, "", "test-queue", []byte(`{}`), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get RabbitMQ channel for tenant tenant-1")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestMultiTenantPublisher_ProducerDefault_PublishesMessage(t *testing.T) {
	ch := &stubRabbitMQChannel{}
	publisher := &multiTenantPublisher{
		manager: &stubRabbitMQManager{channel: ch},
		logger:  testManagerBootstrapLogger(),
	}

	ctx := stubTenantContext("tenant-1")
	headers := map[string]any{"X-Custom": "value"}

	err := publisher.ProducerDefault(ctx, "exchange", "routing-key", []byte(`{"job":"1"}`), &headers)
	require.NoError(t, err)
	assert.True(t, ch.published, "expected message to be published")
	assert.True(t, ch.closed, "expected channel to be closed")
	assert.Equal(t, "exchange", ch.lastExchange)
	assert.Equal(t, "routing-key", ch.lastKey)
}

func TestMultiTenantPublisher_ProducerDefault_NilHeaders(t *testing.T) {
	ch := &stubRabbitMQChannel{}
	publisher := &multiTenantPublisher{
		manager: &stubRabbitMQManager{channel: ch},
		logger:  testManagerBootstrapLogger(),
	}

	ctx := stubTenantContext("tenant-1")

	err := publisher.ProducerDefault(ctx, "", "queue", []byte(`{}`), nil)
	require.NoError(t, err)
	assert.True(t, ch.published)
}

func TestManagerRabbitMQAdapter_GetChannel(t *testing.T) {
	// Verify the adapter wraps the manager correctly (structural test)
	adapter := newManagerRabbitMQAdapter(nil)
	assert.NotNil(t, adapter)
}

func TestInitPlatformDependencies_MultiTenant_CreatesTMPublisher(t *testing.T) {
	original := newSchemaCacheStore
	originalTM := newTenantManagerClient
	t.Cleanup(func() {
		newSchemaCacheStore = original
		newTenantManagerClient = originalTM
	})

	newSchemaCacheStore = func(redisCache.RedisConfig, libLog.Logger, time.Duration, string) (redisCache.Cache[model.DataSourceSchema], error) {
		return stubSchemaCacheStore{}, nil
	}

	newTenantManagerClient = func(string, libLog.Logger, ...tmclient.ClientOption) (*tmclient.Client, error) {
		return &tmclient.Client{}, nil
	}

	cfg := &Config{
		MultiTenantEnabled:       true,
		MultiTenantURL:           "http://tenant-manager:8080",
		MultiTenantServiceAPIKey: "test-key",
	}

	deps, err := initPlatformDependencies(cfg, testManagerBootstrapLogger(), nil)
	require.NoError(t, err)
	assert.NotNil(t, deps.rabbitPublisher)
	assert.NotNil(t, deps.rabbitMQCleanup)

	// Verify it is a multiTenantPublisher
	_, ok := deps.rabbitPublisher.(*multiTenantPublisher)
	assert.True(t, ok, "expected rabbitPublisher to be *multiTenantPublisher in multi-tenant mode")
}

func TestInitPlatformDependencies_MultiTenant_TMClientError(t *testing.T) {
	original := newSchemaCacheStore
	originalTM := newTenantManagerClient
	t.Cleanup(func() {
		newSchemaCacheStore = original
		newTenantManagerClient = originalTM
	})

	newSchemaCacheStore = func(redisCache.RedisConfig, libLog.Logger, time.Duration, string) (redisCache.Cache[model.DataSourceSchema], error) {
		return stubSchemaCacheStore{}, nil
	}

	newTenantManagerClient = func(string, libLog.Logger, ...tmclient.ClientOption) (*tmclient.Client, error) {
		return nil, errors.New("tm client init failed")
	}

	cfg := &Config{
		MultiTenantEnabled:       true,
		MultiTenantURL:           "http://tenant-manager:8080",
		MultiTenantServiceAPIKey: "test-key",
	}

	deps, err := initPlatformDependencies(cfg, testManagerBootstrapLogger(), nil)
	assert.Nil(t, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create tenant manager client for RabbitMQ")
}

// --- test stubs for multiTenantPublisher ---

type stubRabbitMQManager struct {
	channel managerRabbitMQChannel
	err     error
}

func (s *stubRabbitMQManager) GetChannel(_ context.Context, _ string) (managerRabbitMQChannel, error) {
	return s.channel, s.err
}

type stubRabbitMQChannel struct {
	published    bool
	closed       bool
	lastExchange string
	lastKey      string
	publishErr   error
}

func (s *stubRabbitMQChannel) PublishWithContext(_ context.Context, exchange, key string, _, _ bool, _ amqp.Publishing) error {
	s.published = true
	s.lastExchange = exchange
	s.lastKey = key

	return s.publishErr
}

func (s *stubRabbitMQChannel) Close() error {
	s.closed = true
	return nil
}

// stubTenantContext creates a context with tenant ID set via tmcore.
func stubTenantContext(tenantID string) context.Context {
	return tmcore.ContextWithTenantID(context.Background(), tenantID)
}

func TestInitMultiTenantMiddleware(t *testing.T) {
	logger := testManagerBootstrapLogger()

	originalNewTenantManagerClient := newTenantManagerClient
	t.Cleanup(func() {
		newTenantManagerClient = originalNewTenantManagerClient
	})

	tests := []struct {
		name        string
		cfg         *Config
		wantNil     bool
		wantErr     string
		description string
	}{
		{
			name: "returns nil when multi-tenant disabled",
			cfg: &Config{
				MultiTenantEnabled: false,
				MultiTenantURL:     "http://tenant-manager:8080",
			},
			wantNil:     true,
			description: "single-tenant mode should return nil middleware",
		},
		{
			name: "returns nil when multi-tenant URL is empty",
			cfg: &Config{
				MultiTenantEnabled: true,
				MultiTenantURL:     "",
			},
			wantNil:     true,
			description: "enabled without URL should return nil middleware",
		},
		{
			name: "returns middleware when fully configured",
			cfg: &Config{
				MultiTenantEnabled:                  true,
				MultiTenantURL:                      "http://tenant-manager:8080",
				MultiTenantAllowInsecureHTTP:        true,
				MultiTenantMaxTenantPools:           100,
				MultiTenantIdleTimeoutSec:           300,
				MultiTenantCircuitBreakerThreshold:  5,
				MultiTenantCircuitBreakerTimeoutSec: 30,
				MultiTenantServiceAPIKey:            "test-api-key",
			},
			wantNil:     false,
			description: "fully configured multi-tenant should return middleware",
		},
		{
			name: "returns middleware with zero circuit breaker threshold and applies default",
			cfg: &Config{
				MultiTenantEnabled:                  true,
				MultiTenantURL:                      "http://tenant-manager:8080",
				MultiTenantAllowInsecureHTTP:        true,
				MultiTenantCircuitBreakerThreshold:  0,
				MultiTenantCircuitBreakerTimeoutSec: 0,
				MultiTenantServiceAPIKey:            "test-api-key",
			},
			wantNil:     false,
			description: "zero threshold should apply default circuit breaker (5 failures, 30s timeout)",
		},
		{
			name: "returns error when tenant manager client initialization fails",
			cfg: &Config{
				MultiTenantEnabled:       true,
				MultiTenantURL:           "http://tenant-manager:8080",
				MultiTenantServiceAPIKey: "test-api-key",
			},
			wantNil:     true,
			wantErr:     "tenant client init failed",
			description: "configured multi-tenant must fail fast when client creation fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newTenantManagerClient = originalNewTenantManagerClient
			if tt.wantErr != "" {
				newTenantManagerClient = func(string, libLog.Logger, ...tmclient.ClientOption) (*tmclient.Client, error) {
					return nil, errors.New(tt.wantErr)
				}
			}

			handler, cleanup, err := initMultiTenantMiddleware(tt.cfg, logger)
			if cleanup != nil {
				t.Cleanup(cleanup)
			}

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "create tenant manager client")
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, handler)
				return
			}

			require.NoError(t, err)
			if tt.wantNil {
				if handler != nil {
					t.Errorf("initMultiTenantMiddleware() = non-nil, want nil: %s", tt.description)
				}
			} else {
				if handler == nil {
					t.Errorf("initMultiTenantMiddleware() = nil, want non-nil: %s", tt.description)
				}
			}
		})
	}
}

// --- Gate 4 of ring:dev-readyz --------------------------------------------
//
// ValidateSaaSTLS MUST run before any platform connection opens. The
// following tests assert that:
//
//  1. InitServers refuses to start when DEPLOYMENT_MODE=saas and a
//     dependency (here: MongoDB URI "mongodb") is non-TLS. The returned
//     error MUST wrap readyz.ErrSaaSTLSRequired so alerting can key off
//     it.
//
//  2. The TLS check runs BEFORE initMongoRepositoriesFn is invoked, proving
//     the hard-gate ordering contract holds.
//
//  3. buildSaaSTLSConfig maps the Config fields into the right
//     SaaSTLSConfig fields — notably HasS3 is false for the manager and
//     AllowInsecureHTTPTM mirrors MultiTenantAllowInsecureHTTP.

func TestInitServers_SaaSMode_RefusesNonTLSMongo(t *testing.T) {
	originalLoadConfig := loadConfigFn
	originalLoggerAndTelemetry := initLoggerAndTelemetryFn
	originalMongoRepos := initMongoRepositoriesFn
	originalCrypto := initCryptoFn
	originalPlatform := initPlatformDependenciesFn
	originalAssemble := assembleServiceFn

	t.Cleanup(func() {
		loadConfigFn = originalLoadConfig
		initLoggerAndTelemetryFn = originalLoggerAndTelemetry
		initMongoRepositoriesFn = originalMongoRepos
		initCryptoFn = originalCrypto
		initPlatformDependenciesFn = originalPlatform
		assembleServiceFn = originalAssemble
	})

	loadConfigFn = func() (*Config, error) {
		return &Config{
			DeploymentMode: "saas",
			MongoURI:       "mongodb",
			MongoDBHost:    "host",
			MongoDBPort:    "27017",
		}, nil
	}
	initLoggerAndTelemetryFn = func(*Config) (libLog.Logger, *libOtel.Telemetry, error) {
		return testManagerBootstrapLogger(), &libOtel.Telemetry{}, nil
	}

	mongoCalled := false

	initMongoRepositoriesFn = func(context.Context, *Config, libLog.Logger) (*managerRepositories, error) {
		mongoCalled = true
		return &managerRepositories{}, nil
	}

	service, err := InitServers()
	assert.Nil(t, service)
	require.Error(t, err)
	require.ErrorIs(t, err, readyz.ErrSaaSTLSRequired,
		"error must wrap ErrSaaSTLSRequired so operators can detect the SaaS-gate failure")
	assert.False(t, mongoCalled,
		"Gate 4 must fire BEFORE initMongoRepositoriesFn — Mongo client must not be dialed")
}

func TestInitServers_LocalMode_SkipsSaaSTLSEnforcement(t *testing.T) {
	originalLoadConfig := loadConfigFn
	originalLoggerAndTelemetry := initLoggerAndTelemetryFn
	originalMongoRepos := initMongoRepositoriesFn
	originalCrypto := initCryptoFn
	originalPlatform := initPlatformDependenciesFn
	originalAssemble := assembleServiceFn

	t.Cleanup(func() {
		loadConfigFn = originalLoadConfig
		initLoggerAndTelemetryFn = originalLoggerAndTelemetry
		initMongoRepositoriesFn = originalMongoRepos
		initCryptoFn = originalCrypto
		initPlatformDependenciesFn = originalPlatform
		assembleServiceFn = originalAssemble
	})

	loadConfigFn = func() (*Config, error) {
		return &Config{
			DeploymentMode: "local",
			MongoURI:       "mongodb",
			MongoDBHost:    "host",
			MongoDBPort:    "27017",
		}, nil
	}
	initLoggerAndTelemetryFn = func(*Config) (libLog.Logger, *libOtel.Telemetry, error) {
		return testManagerBootstrapLogger(), &libOtel.Telemetry{}, nil
	}
	initMongoRepositoriesFn = func(context.Context, *Config, libLog.Logger) (*managerRepositories, error) {
		return &managerRepositories{}, nil
	}
	initCryptoFn = func(*Config, libLog.Logger) (*managerCrypto, error) {
		return &managerCrypto{service: &crypto.AESGCMService{}}, nil
	}
	initPlatformDependenciesFn = func(*Config, libLog.Logger, crypto.Signer) (*managerPlatformDependencies, error) {
		return &managerPlatformDependencies{}, nil
	}
	assembleServiceFn = func(*Config, libLog.Logger, *libOtel.Telemetry, *managerRepositories, *crypto.AESGCMService, *managerPlatformDependencies) (*Service, error) {
		return &Service{}, nil
	}

	service, err := InitServers()
	require.NoError(t, err, "local mode must bypass SaaS TLS enforcement")
	require.NotNil(t, service)
}

func TestBuildSaaSTLSConfig_ManagerNeverClaimsS3(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		DeploymentMode:               "saas",
		MongoURI:                     "mongodb+srv",
		MongoDBHost:                  "host",
		RedisHost:                    "redis",
		RedisPort:                    "6379",
		RedisTLS:                     true,
		MultiTenantRedisHost:         "mt-redis",
		MultiTenantRedisPort:         "6379",
		MultiTenantRedisTLS:          true,
		RabbitURI:                    "amqps",
		RabbitMQHost:                 "rabbit",
		RabbitMQPortAMQP:             "5671",
		RabbitMQUser:                 "user",
		RabbitMQPass:                 "pass",
		MultiTenantURL:               "https://tm",
		MultiTenantEnabled:           true,
		MultiTenantAllowInsecureHTTP: true,
	}

	got := buildSaaSTLSConfig(cfg, false)
	assert.Equal(t, "saas", got.DeploymentMode)
	assert.Equal(t, "https://tm", got.TenantManagerURL)
	assert.True(t, got.MultiTenantEnabled)
	assert.True(t, got.AllowInsecureHTTPTM)
	assert.False(t, got.HasS3, "manager must not claim S3 ownership")
	assert.Equal(t, "", got.S3Endpoint)
	assert.Equal(t, "rediss://redis:6379", got.RedisURL)
	assert.Equal(t, "rediss://mt-redis:6379", got.MultiTenantRedisURL)
}
