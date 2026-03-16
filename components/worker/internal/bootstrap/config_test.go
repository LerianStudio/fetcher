package bootstrap

import (
	"context"
	"errors"
	"testing"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libZap "github.com/LerianStudio/lib-commons/v4/commons/zap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestConfig_RedisFields(t *testing.T) {
	t.Setenv("REDIS_HOST", "localhost:6379")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_PROTOCOL", "3")

	cfg := &Config{}
	err := libCommons.SetConfigFromEnvVars(cfg)
	require.NoError(t, err, "Failed to load config")

	assert.Equal(t, "localhost:6379", cfg.RedisHost)
	assert.Equal(t, "secret", cfg.RedisPassword)
	assert.Equal(t, 2, cfg.RedisDB)
	assert.Equal(t, 3, cfg.RedisProtocol)
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
				RedisHost:                "localhost:6379",
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
				RedisHost:                "localhost:6379",
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
				RedisHost:                "",
			},
			wantErr:   true,
			errSubstr: "REDIS_HOST is required",
		},
		{
			name: "multi-tenant enabled with all required config succeeds",
			cfg: &Config{
				MultiTenantEnabled:       true,
				MultiTenantURL:           "http://tenant-manager:8080",
				MultiTenantServiceAPIKey: "test-api-key",
				RedisHost:                "localhost:6379",
			},
			wantErr: false,
		},
		{
			name: "multi-tenant disabled succeeds without Redis",
			cfg: &Config{
				MultiTenantEnabled: false,
				RedisHost:          "",
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

// Dummy usage of context to avoid import issues
var _ = context.Background
