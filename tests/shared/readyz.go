package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Readyz status vocabulary — kept as exported constants so e2e tests can
// reference them by name without importing pkg/bootstrap/readyz (an internal
// package would be a layering violation from the test side, and we want the
// e2e suite to validate the wire contract independently).
const (
	ReadyzTopHealthy   = "healthy"
	ReadyzTopUnhealthy = "unhealthy"

	ReadyzStatusUp       = "up"
	ReadyzStatusDown     = "down"
	ReadyzStatusDegraded = "degraded"
	ReadyzStatusSkipped  = "skipped"
	ReadyzStatusNA       = "n/a"
)

// Canonical dependency names used in /readyz responses. The chaos suite hard-
// codes the same strings (tests/chaos/readyz_chaos_test.go) — keeping a single
// source of truth in shared/ avoids drift between the two suites.
const (
	ReadyzDepMongoDB       = "mongodb"
	ReadyzDepRabbitMQ      = "rabbitmq"
	ReadyzDepRedis         = "redis"
	ReadyzDepS3            = "s3"
	ReadyzDepTenantManager = "tenant_manager"
	ReadyzDepDraining      = "draining"
)

// ReadyzCheck mirrors readyz.DependencyCheck on the wire side. It is
// intentionally redefined here rather than imported so the e2e tests catch any
// drift in the JSON contract — if the field shape changes upstream, these
// tests fail with a clear decode error.
type ReadyzCheck struct {
	Status       string `json:"status"`
	LatencyMs    int64  `json:"latency_ms,omitempty"`
	TLS          *bool  `json:"tls,omitempty"`
	Error        string `json:"error,omitempty"`
	Reason       string `json:"reason,omitempty"`
	BreakerState string `json:"breaker_state,omitempty"`
}

// ReadyzResponse mirrors readyz.ReadyzResponse on the wire side. Same
// rationale as ReadyzCheck.
type ReadyzResponse struct {
	Status         string                 `json:"status"`
	Checks         map[string]ReadyzCheck `json:"checks"`
	Version        string                 `json:"version"`
	DeploymentMode string                 `json:"deployment_mode"`
	TenantID       string                 `json:"tenant_id,omitempty"`
}

// readyzHTTPTimeout caps every /readyz request — the handler enforces 2s
// per-dep timeouts and is expected to respond well below this even with all
// checks running serially. A 10s budget gives ample slack for CI cold starts
// without masking pathological hangs.
const readyzHTTPTimeout = 10 * time.Second

// readyzClient is a single shared *http.Client. resty is overkill for these
// raw probes — we want the bare HTTP semantics K8s' readinessProbe sees.
var readyzClient = &http.Client{Timeout: readyzHTTPTimeout}

// GetReadyz issues GET <baseURL>/readyz and returns (status, body, latency).
// It is the canonical helper used by every e2e readyz test. Decode failures
// fail the test fataly because they signal a contract violation, not a
// recoverable runtime error.
func GetReadyz(t *testing.T, ctx context.Context, baseURL string) (int, ReadyzResponse, time.Duration) {
	t.Helper()
	return getReadyzAt(t, ctx, baseURL, "/readyz")
}

// GetReadyzTenant issues GET <baseURL>/readyz/tenant/<tenantID>. Returns the
// raw status code, decoded body (may be empty when the handler responds with
// a non-readyz error envelope), and elapsed latency.
func GetReadyzTenant(t *testing.T, ctx context.Context, baseURL, tenantID string) (int, ReadyzResponse, time.Duration) {
	t.Helper()
	return getReadyzAt(t, ctx, baseURL, "/readyz/tenant/"+tenantID)
}

// GetHealth issues GET <baseURL>/health. /health returns 200 once the
// startup self-probe has flipped — used to assert liveness behaviour
// independently of /readyz (which gates on per-dep state, not boot state).
func GetHealth(t *testing.T, ctx context.Context, baseURL string) (int, time.Duration) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
	require.NoError(t, err, "build /health request")

	start := time.Now()
	resp, err := readyzClient.Do(req)
	elapsed := time.Since(start)

	require.NoError(t, err, "GET /health")

	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, elapsed
}

// GetMetricsBody issues GET <baseURL>/metrics and returns the response body
// as a string. Tests grep this for readyz_check_duration_ms,
// readyz_check_status, and selfprobe_result line prefixes.
func GetMetricsBody(t *testing.T, ctx context.Context, baseURL string) string {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/metrics", nil)
	require.NoError(t, err, "build /metrics request")

	resp, err := readyzClient.Do(req)
	require.NoError(t, err, "GET /metrics")

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode, "/metrics must return 200")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "read /metrics body")

	return string(body)
}

// AssertMetricFamilyPresent asserts that the Prometheus exposition body
// contains at least one sample for the given metric family name. It matches
// against the HELP comment line ("# HELP <name>") which is emitted exactly
// once per registered metric — far more reliable than substring matching the
// sample lines, which may be absent when no sample has been recorded yet.
func AssertMetricFamilyPresent(t *testing.T, body, metricName string) {
	t.Helper()

	helpLine := "# HELP " + metricName
	assert.True(t, strings.Contains(body, helpLine),
		"metrics body should expose %q (looked for %q)", metricName, helpLine)
}

// AssertMetricSamplePresent asserts the body contains at least one sample
// line whose label set contains every key=value pair in labels. Useful for
// asserting selfprobe_result{dep="mongodb"} 1 etc. without parsing the full
// Prometheus exposition format.
func AssertMetricSamplePresent(t *testing.T, body, metricName string, labels map[string]string) {
	t.Helper()

	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.HasPrefix(line, metricName) {
			continue
		}

		// Quick label match — for each k=v we ensure the literal `k="v"`
		// substring appears in the line. This is good enough for the labels
		// the readyz checkers emit (dep, status) which never contain quotes.
		matched := true

		for k, v := range labels {
			needle := fmt.Sprintf(`%s=%q`, k, v)
			if !strings.Contains(line, needle) {
				matched = false
				break
			}
		}

		if matched {
			return
		}
	}

	t.Fatalf("metrics body did not contain a %s sample with labels %v", metricName, labels)
}

// AssertMetricGaugeValue asserts the body contains a sample line for the
// named gauge with the exact value, matching the labels. Differs from
// AssertMetricSamplePresent in that it parses the trailing numeric value and
// compares it — necessary for boolean-style gauges like
// `readyz_check_status{dep="mongodb",status="down"} 1` where the existence
// of the line is not enough (the same metric+labels exists with value 0
// when the dep is in a different status).
//
// Float comparison uses ParseFloat and equality at full precision; the
// readyz handler emits integer-valued doubles (0/1) so this is exact.
func AssertMetricGaugeValue(t *testing.T, body, metricName string, labels map[string]string, expected float64) {
	t.Helper()

	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.HasPrefix(line, metricName) {
			continue
		}

		matched := true

		for k, v := range labels {
			needle := fmt.Sprintf(`%s=%q`, k, v)
			if !strings.Contains(line, needle) {
				matched = false
				break
			}
		}

		if !matched {
			continue
		}

		// Sample-line shape: `<metric>{<labels>} <value>` — value is the
		// last whitespace-separated token. (No timestamp — the Go client
		// library's text exposition omits it.)
		idx := strings.LastIndex(line, " ")
		if idx < 0 {
			continue
		}

		value, err := strconv.ParseFloat(strings.TrimSpace(line[idx+1:]), 64)
		if err != nil {
			t.Fatalf("could not parse value from sample line %q: %v", line, err)
			return
		}

		if value != expected {
			t.Fatalf("metric %s with labels %v has value %v, expected %v (line: %q)",
				metricName, labels, value, expected, line)
		}

		return
	}

	t.Fatalf("metrics body did not contain a %s sample with labels %v", metricName, labels)
}

// IsValidReadyzStatus returns true iff s is one of the five vocabulary values
// that readyz.DependencyCheck.Status may carry on the wire.
func IsValidReadyzStatus(s string) bool {
	switch s {
	case ReadyzStatusUp, ReadyzStatusDown, ReadyzStatusDegraded,
		ReadyzStatusSkipped, ReadyzStatusNA:
		return true
	}

	return false
}

// IsValidReadyzTopStatus returns true iff s is one of the two values
// readyz.ReadyzResponse.Status may carry on the wire.
func IsValidReadyzTopStatus(s string) bool {
	return s == ReadyzTopHealthy || s == ReadyzTopUnhealthy
}

// WaitForReadyzHealthy polls GET /readyz until it returns 200 + healthy or
// the timeout fires. Returns the elapsed wait time. Used during recovery
// assertions where the handler eventually flips back to healthy after a dep
// recovers.
func WaitForReadyzHealthy(t *testing.T, ctx context.Context, baseURL string, timeout time.Duration) time.Duration {
	t.Helper()

	deadline := time.Now().Add(timeout)
	start := time.Now()

	for time.Now().Before(deadline) {
		status, body, _ := GetReadyz(t, ctx, baseURL)
		if status == http.StatusOK && body.Status == ReadyzTopHealthy {
			return time.Since(start)
		}

		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled while waiting for /readyz healthy: %v", ctx.Err())
			return 0
		case <-time.After(250 * time.Millisecond):
		}
	}

	t.Fatalf("/readyz did not become healthy within %v", timeout)

	return 0
}

// getReadyzAt is the shared implementation behind GetReadyz and
// GetReadyzTenant. It deliberately does NOT set X-Organization-Id — readyz
// endpoints are unauthenticated by contract (Kubernetes kubelet has no way
// to present credentials).
func getReadyzAt(t *testing.T, ctx context.Context, baseURL, path string) (int, ReadyzResponse, time.Duration) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	require.NoError(t, err, "build %s request", path)

	start := time.Now()
	resp, err := readyzClient.Do(req)
	elapsed := time.Since(start)

	require.NoError(t, err, "GET %s", path)

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "read %s body", path)

	var decoded ReadyzResponse
	if len(body) > 0 {
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("decode %s body: %v, body=%s", path, err, string(body))
		}
	}

	return resp.StatusCode, decoded, elapsed
}
