package bootstrap

import (
	"testing"

	libCommons "github.com/LerianStudio/lib-commons/v3/commons"
	"github.com/LerianStudio/lib-commons/v3/commons/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			name: "multi-tenant enabled without Redis returns error",
			cfg: &Config{
				MultiTenantEnabled: true,
				RedisHost:          "",
			},
			wantErr:   true,
			errSubstr: "REDIS_HOST is required",
		},
		{
			name: "multi-tenant enabled with Redis succeeds",
			cfg: &Config{
				MultiTenantEnabled: true,
				RedisHost:          "localhost:6379",
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

	logger := &mockConfigTestLogger{}

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

// mockConfigTestLogger satisfies log.Logger for config tests.
type mockConfigTestLogger struct{}

func (m *mockConfigTestLogger) Info(_ ...any)                                  {}
func (m *mockConfigTestLogger) Infof(_ string, _ ...any)                       {}
func (m *mockConfigTestLogger) Infoln(_ ...any)                                {}
func (m *mockConfigTestLogger) Warn(_ ...any)                                  {}
func (m *mockConfigTestLogger) Warnf(_ string, _ ...any)                       {}
func (m *mockConfigTestLogger) Warnln(_ ...any)                                {}
func (m *mockConfigTestLogger) Error(_ ...any)                                 {}
func (m *mockConfigTestLogger) Errorf(_ string, _ ...any)                      {}
func (m *mockConfigTestLogger) Errorln(_ ...any)                               {}
func (m *mockConfigTestLogger) Debug(_ ...any)                                 {}
func (m *mockConfigTestLogger) Debugf(_ string, _ ...any)                      {}
func (m *mockConfigTestLogger) Debugln(_ ...any)                               {}
func (m *mockConfigTestLogger) Fatal(_ ...any)                                 {}
func (m *mockConfigTestLogger) Fatalf(_ string, _ ...any)                      {}
func (m *mockConfigTestLogger) Fatalln(_ ...any)                               {}
func (m *mockConfigTestLogger) WithFields(_ ...any) log.Logger                 { return m }
func (m *mockConfigTestLogger) WithDefaultMessageTemplate(_ string) log.Logger { return m }
func (m *mockConfigTestLogger) Sync() error                                    { return nil }
