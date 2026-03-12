package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	cacheRepo "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	redisCache "github.com/LerianStudio/fetcher/pkg/redis"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"github.com/LerianStudio/lib-commons/v4/commons/zap"
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
	assembleServiceFn = func(*Config, libLog.Logger, *libOtel.Telemetry, *managerRepositories, *crypto.AESGCMService, *managerPlatformDependencies) *Service {
		return expectedService
	}

	service, err := InitServers()
	require.NoError(t, err)
	assert.Same(t, expectedService, service)
}

func TestInitServers_PropagatesHelperErrors(t *testing.T) {
	originalLoadConfig := loadConfigFn
	originalLoggerAndTelemetry := initLoggerAndTelemetryFn
	t.Cleanup(func() {
		loadConfigFn = originalLoadConfig
		initLoggerAndTelemetryFn = originalLoggerAndTelemetry
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
				"MULTI_TENANT_ENVIRONMENT":                 "staging",
				"MULTI_TENANT_MAX_TENANT_POOLS":            "200",
				"MULTI_TENANT_IDLE_TIMEOUT_SEC":            "600",
				"MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD":   "10",
				"MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC": "60",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.MultiTenantEnabled {
					t.Error("MultiTenantEnabled should be true")
				}
				if cfg.MultiTenantURL != "http://tenant-manager:8080" {
					t.Errorf("MultiTenantURL = %q, want %q", cfg.MultiTenantURL, "http://tenant-manager:8080")
				}
				if cfg.MultiTenantEnvironment != "staging" {
					t.Errorf("MultiTenantEnvironment = %q, want %q", cfg.MultiTenantEnvironment, "staging")
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
				if cfg.MultiTenantEnvironment != "" {
					t.Errorf("MultiTenantEnvironment should be empty, got %q", cfg.MultiTenantEnvironment)
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

func TestInitMultiTenantMiddleware(t *testing.T) {
	logger, _ := zap.New(zap.Config{})

	tests := []struct {
		name        string
		cfg         *Config
		wantNil     bool
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
				MultiTenantMaxTenantPools:           100,
				MultiTenantIdleTimeoutSec:           300,
				MultiTenantCircuitBreakerThreshold:  5,
				MultiTenantCircuitBreakerTimeoutSec: 30,
			},
			wantNil:     false,
			description: "fully configured multi-tenant should return middleware",
		},
		{
			name: "returns middleware with zero circuit breaker threshold and applies default",
			cfg: &Config{
				MultiTenantEnabled:                  true,
				MultiTenantURL:                      "http://tenant-manager:8080",
				MultiTenantCircuitBreakerThreshold:  0,
				MultiTenantCircuitBreakerTimeoutSec: 0,
			},
			wantNil:     false,
			description: "zero threshold should apply default circuit breaker (5 failures, 30s timeout)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := initMultiTenantMiddleware(tt.cfg, logger)
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
