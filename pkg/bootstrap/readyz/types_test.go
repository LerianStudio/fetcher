package readyz

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyCheck_JSON_OmitsEmptyOptionalFields(t *testing.T) {
	c := DependencyCheck{Status: StatusUp}

	body, err := json.Marshal(c)
	require.NoError(t, err)

	// Required.
	assert.Contains(t, string(body), `"status":"up"`)
	// Everything else must be omitted when zero-valued.
	assert.NotContains(t, string(body), "latency_ms")
	assert.NotContains(t, string(body), "tls")
	assert.NotContains(t, string(body), "error")
	assert.NotContains(t, string(body), "reason")
	assert.NotContains(t, string(body), "breaker_state")
}

func TestDependencyCheck_JSON_TLSPointerSemantics(t *testing.T) {
	t.Run("nil pointer omits the field entirely", func(t *testing.T) {
		c := DependencyCheck{Status: StatusUp}

		body, err := json.Marshal(c)
		require.NoError(t, err)
		assert.NotContains(t, string(body), "tls")
	})

	t.Run("false pointer emits tls:false", func(t *testing.T) {
		c := DependencyCheck{Status: StatusUp, TLS: TLSPtr(false)}

		body, err := json.Marshal(c)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"tls":false`)
	})

	t.Run("true pointer emits tls:true", func(t *testing.T) {
		c := DependencyCheck{Status: StatusUp, TLS: TLSPtr(true)}

		body, err := json.Marshal(c)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"tls":true`)
	})
}

func TestDependencyCheck_JSON_LatencyOmittedWhenZero(t *testing.T) {
	c := DependencyCheck{Status: StatusUp, LatencyMs: 0}

	body, err := json.Marshal(c)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "latency_ms")

	c.LatencyMs = 42
	body, err = json.Marshal(c)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"latency_ms":42`)
}

func TestDependencyCheck_JSON_BreakerStateOptional(t *testing.T) {
	c := DependencyCheck{Status: StatusDegraded, BreakerState: "half-open"}

	body, err := json.Marshal(c)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"breaker_state":"half-open"`)
}

func TestDependencyCheck_JSON_RoundTrip(t *testing.T) {
	original := DependencyCheck{
		Status:       StatusDegraded,
		LatencyMs:    12,
		TLS:          TLSPtr(true),
		Error:        "breaker half-open",
		BreakerState: "half-open",
	}

	body, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded DependencyCheck
	require.NoError(t, json.Unmarshal(body, &decoded))

	assert.Equal(t, original.Status, decoded.Status)
	assert.Equal(t, original.LatencyMs, decoded.LatencyMs)
	require.NotNil(t, decoded.TLS)
	assert.Equal(t, *original.TLS, *decoded.TLS)
	assert.Equal(t, original.Error, decoded.Error)
	assert.Equal(t, original.BreakerState, decoded.BreakerState)
}

func TestReadyzResponse_JSON_HealthyShape(t *testing.T) {
	resp := ReadyzResponse{
		Status: TopStatusHealthy,
		Checks: map[string]DependencyCheck{
			"mongodb": {Status: StatusUp, LatencyMs: 3, TLS: TLSPtr(true)},
			"redis":   {Status: StatusSkipped, Reason: "REDIS_ENABLED=false"},
		},
		Version:        "1.2.3",
		DeploymentMode: DeploymentModeSaaS,
	}

	body, err := json.Marshal(resp)
	require.NoError(t, err)

	s := string(body)
	assert.True(t, strings.Contains(s, `"status":"healthy"`), s)
	assert.Contains(t, s, `"version":"1.2.3"`)
	assert.Contains(t, s, `"deployment_mode":"saas"`)
	assert.Contains(t, s, `"mongodb"`)
	assert.Contains(t, s, `"redis"`)
}

func TestStatusConstants_UseExactContractStrings(t *testing.T) {
	// Lock the vocabulary — any change here is a contract break.
	assert.Equal(t, "up", StatusUp)
	assert.Equal(t, "down", StatusDown)
	assert.Equal(t, "degraded", StatusDegraded)
	assert.Equal(t, "skipped", StatusSkipped)
	assert.Equal(t, "n/a", StatusNA)
	assert.Equal(t, "healthy", TopStatusHealthy)
	assert.Equal(t, "unhealthy", TopStatusUnhealthy)
}
