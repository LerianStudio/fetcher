package bootstrap

import (
	"testing"
	"time"

	cacheRepo "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
	"github.com/LerianStudio/lib-commons/v4/commons/zap"
	"github.com/stretchr/testify/assert"
)

func TestConfigStruct(t *testing.T) {
	cfg := &Config{
		EnvName:       "test",
		ServerAddress: "localhost:8080",
		LogLevel:      "info",
	}

	if cfg.EnvName != "test" {
		t.Errorf("Expected EnvName to be 'test', got '%s'", cfg.EnvName)
	}

	if cfg.ServerAddress != "localhost:8080" {
		t.Errorf("Expected ServerAddress to be 'localhost:8080', got '%s'", cfg.ServerAddress)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be 'info', got '%s'", cfg.LogLevel)
	}
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
