package bootstrap

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/bootstrap/readyz"
	"github.com/LerianStudio/lib-commons/v7/commons/buildinfo"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sentinel values nothing else in this repo produces, so an assertion against
// them can only pass if the surface read the compiled identity.
const (
	sentinelVersion   = "9.9.9-sentinel"
	sentinelRevision  = "5e9f7c1a3b8d604e2f17a9c05b3d8e6417f2a90c"
	sentinelBuildTime = "2001-02-03T04:05:06Z"
)

// setSentinelBuildIdentity injects the sentinel and restores the values read
// before it, not the zero Build: restoring zeros would leave the rest of the
// package reading fallbacks instead of the real identity.
func setSentinelBuildIdentity(t *testing.T) {
	t.Helper()

	prior := buildinfo.Get()

	t.Cleanup(func() {
		buildinfo.Set(buildinfo.Build{
			Version:   prior.Version,
			Revision:  prior.Revision,
			BuildTime: prior.BuildTime,
		})
	})

	buildinfo.Set(buildinfo.Build{
		Version:   sentinelVersion,
		Revision:  sentinelRevision,
		BuildTime: sentinelBuildTime,
	})
}

// TestBuildIdentity_ReachesManagerSurfaces proves the two manager surfaces
// wired in this package — the /readyz body and the OTel resource — report
// whatever identity the binary was compiled with, rather than a value that
// happens to match because both sides evaluate buildinfo.Get(). Sequential on
// purpose: the identity is process-global (Decision 8).
func TestBuildIdentity_ReachesManagerSurfaces(t *testing.T) {
	setSentinelBuildIdentity(t)

	cfg := &Config{OtelServiceName: "fetcher", DeploymentMode: readyz.DeploymentModeLocal}

	app := fiber.New()
	app.Get("/readyz", readyz.NewHandler(newReadyzConfig(cfg)).Fiber())

	res, err := app.Test(httptest.NewRequest("GET", "/readyz", nil),
		fiber.TestConfig{Timeout: 2 * time.Second, FailOnTimeout: true})
	require.NoError(t, err)

	defer res.Body.Close()

	require.Equal(t, fiber.StatusOK, res.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))

	assert.Equal(t, sentinelVersion, body["version"])
	assert.Equal(t, sentinelRevision, body["revision"])
	assert.Equal(t, sentinelBuildTime, body["buildTime"])

	otel := managerTelemetryConfig(cfg, nil)
	assert.Equal(t, sentinelVersion, otel.ServiceVersion)
	assert.Equal(t, sentinelRevision, otel.ServiceRevision)
}
