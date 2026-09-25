package bootstrap

import (
	"encoding/json"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

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
