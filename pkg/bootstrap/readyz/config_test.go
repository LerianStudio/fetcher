package readyz

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveDeploymentMode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty defaults to local", in: "", want: DeploymentModeLocal},
		{name: "saas lowercase", in: "saas", want: DeploymentModeSaaS},
		{name: "SAAS uppercase", in: "SAAS", want: DeploymentModeSaaS},
		{name: "byoc", in: "byoc", want: DeploymentModeBYOC},
		{name: "local explicit", in: "local", want: DeploymentModeLocal},
		{name: "typo falls back to local", in: "sass", want: DeploymentModeLocal},
		{name: "whitespace trimmed", in: "  saas ", want: DeploymentModeSaaS},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, resolveDeploymentMode(tc.in))
		})
	}
}

func TestResolveHealthPort(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{name: "empty defaults to 4007", in: "", want: 4007},
		{name: "valid port", in: "8080", want: 8080},
		{name: "zero rejected", in: "0", want: 4007},
		{name: "negative rejected", in: "-1", want: 4007},
		{name: "too large rejected", in: "70000", want: 4007},
		{name: "non-numeric rejected", in: "abc", want: 4007},
		{name: "whitespace trimmed", in: "  8081 ", want: 8081},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, resolveHealthPort(tc.in))
		})
	}
}

func TestResolveDrainDelay(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want time.Duration
	}{
		{name: "empty defaults to 12s", in: "", want: 12 * time.Second},
		{name: "valid 5s", in: "5", want: 5 * time.Second},
		{name: "zero clamped to 1s", in: "0", want: 1 * time.Second},
		{name: "negative clamped to 1s", in: "-4", want: 1 * time.Second},
		{name: "non-numeric falls back to default", in: "xx", want: 12 * time.Second},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, resolveDrainDelay(tc.in))
		})
	}
}

func TestLoadConfig_AppliesDefaults(t *testing.T) {
	t.Setenv("DEPLOYMENT_MODE", "")
	t.Setenv("HEALTH_PORT", "")
	t.Setenv("READYZ_DRAIN_DELAY_SEC", "")

	cfg := LoadConfig()

	assert.Equal(t, DeploymentModeLocal, cfg.DeploymentMode)
	assert.Equal(t, 4007, cfg.HealthPort)
	assert.Equal(t, 12*time.Second, cfg.DrainDelay)
	// The test binary carries no -X stamp, so the identity is the FC-1 fallback.
	assert.Equal(t, Identity{Version: "dev", Revision: "unknown", BuildTime: "unknown"}, cfg.Identity)
}

func TestLoadConfig_ReadsEnv(t *testing.T) {
	t.Setenv("DEPLOYMENT_MODE", "saas")
	t.Setenv("HEALTH_PORT", "5555")
	t.Setenv("READYZ_DRAIN_DELAY_SEC", "7")

	cfg := LoadConfig()

	assert.Equal(t, DeploymentModeSaaS, cfg.DeploymentMode)
	assert.Equal(t, 5555, cfg.HealthPort)
	assert.Equal(t, 7*time.Second, cfg.DrainDelay)
}
