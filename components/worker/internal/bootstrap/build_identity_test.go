package bootstrap

import (
	"encoding/json"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/LerianStudio/lib-commons/v7/commons/buildinfo"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionRoute_ServesTheBuildIdentityContract pins the body a client of
// GET /version reads on the worker probe app — same port and surface as
// /health and /readyz. Sequential on purpose: the build identity is
// process-global (Decision 8).
func TestVersionRoute_ServesTheBuildIdentityContract(t *testing.T) {
	cfg := &Config{HealthPort: 4007, OtelServiceName: "fetcher-worker"}
	srv := NewHealthServer(cfg, nil, nil, nil)

	res, err := srv.App().Test(httptest.NewRequest("GET", "/version", nil),
		fiber.TestConfig{Timeout: 2 * time.Second, FailOnTimeout: true})
	require.NoError(t, err)

	defer res.Body.Close()

	assert.Equal(t, fiber.StatusOK, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "application/json")

	var body map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))

	keys := make([]string, 0, len(body))
	for k := range body {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	// Exactly these: no dependencyManifest (that one is --version only), and no
	// requestDate, which the deprecated commons version handler used to return.
	assert.Equal(t, []string{
		"buildTime", "goVersion", "modified", "revision", "schemaVersion", "service", "version",
	}, keys)

	assert.Equal(t, "v1", body["schemaVersion"])
	assert.Equal(t, "fetcher-worker", body["service"])
	// The test binary carries no -X stamp, so the identity reports its fallback.
	assert.Equal(t, "dev", body["version"])
}

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

// TestBuildIdentity_ReachesWorkerSurfaces proves the worker's three surfaces —
// GET /version, the /readyz body and the OTel resource — report whatever
// identity the binary was compiled with, rather than a value that happens to
// match because both sides evaluate buildinfo.Get(). Sequential on purpose:
// the identity is process-global (Decision 8).
func TestBuildIdentity_ReachesWorkerSurfaces(t *testing.T) {
	setSentinelBuildIdentity(t)

	cfg := &Config{HealthPort: 4007, OtelServiceName: defaultWorkerServiceName}
	srv := NewHealthServer(cfg, nil, nil, nil)

	for _, route := range []string{"/version", "/readyz"} {
		res, err := srv.App().Test(httptest.NewRequest("GET", route, nil),
			fiber.TestConfig{Timeout: 2 * time.Second, FailOnTimeout: true})
		require.NoError(t, err)

		require.Equal(t, fiber.StatusOK, res.StatusCode, route)

		var body map[string]any
		require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
		res.Body.Close()

		assert.Equal(t, sentinelVersion, body["version"], route)
		assert.Equal(t, sentinelRevision, body["revision"], route)
		assert.Equal(t, sentinelBuildTime, body["buildTime"], route)
	}

	otel := workerTelemetryConfig(cfg, nil)
	assert.Equal(t, sentinelVersion, otel.ServiceVersion)
	assert.Equal(t, sentinelRevision, otel.ServiceRevision)
}
