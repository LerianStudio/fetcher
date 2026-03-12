package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	cacheRepo "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
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

func TestGetSchemaCacheTTL(t *testing.T) {
	assert.Equal(t, cacheRepo.DefaultSchemaCacheTTL, getSchemaCacheTTL(""))
	assert.Equal(t, cacheRepo.DefaultSchemaCacheTTL, getSchemaCacheTTL("invalid"))
	assert.Equal(t, 30*time.Second, getSchemaCacheTTL("30"))
}

func TestGetRedisDB(t *testing.T) {
	assert.Equal(t, 2, getRedisDB("2"))
	assert.Equal(t, 0, getRedisDB(""))
	assert.Equal(t, 0, getRedisDB("invalid"))
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
