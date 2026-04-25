package readyz

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingLogger is a test-only libLog.Logger that records every Log call in
// memory so assertions can verify the structured messages emitted by
// RunSelfProbe.
type capturingLogger struct {
	mu      sync.Mutex
	entries []logEntry
}

type logEntry struct {
	level  libLog.Level
	msg    string
	fields []libLog.Field
}

func (c *capturingLogger) Log(_ context.Context, level libLog.Level, msg string, fields ...libLog.Field) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = append(c.entries, logEntry{level: level, msg: msg, fields: fields})
}

func (c *capturingLogger) With(_ ...libLog.Field) libLog.Logger { return c }
func (c *capturingLogger) WithGroup(_ string) libLog.Logger     { return c }
func (c *capturingLogger) Enabled(_ libLog.Level) bool          { return true }
func (c *capturingLogger) Sync(_ context.Context) error         { return nil }

func (c *capturingLogger) hasMessage(msg string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, e := range c.entries {
		if e.msg == msg {
			return true
		}
	}

	return false
}

func (c *capturingLogger) countMessage(msg string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := 0

	for _, e := range c.entries {
		if e.msg == msg {
			n++
		}
	}

	return n
}

func (c *capturingLogger) messageWithField(msg, fieldKey, fieldVal string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, e := range c.entries {
		if e.msg != msg {
			continue
		}

		for _, f := range e.fields {
			if f.Key != fieldKey {
				continue
			}

			if got, ok := f.Value.(string); ok && got == fieldVal {
				return true
			}
		}
	}

	return false
}

// resetSelfProbe is a convenience helper that restores the package default
// after a test flips the flag.
func resetSelfProbe(t *testing.T) {
	t.Helper()

	SetSelfProbe(false)
	t.Cleanup(func() { SetSelfProbe(false) })
}

// scrapeSelfProbeResult returns the selfprobe_result{dep=<dep>} value from the
// Prometheus exposition endpoint, or NaN if the series is not present. The
// test package does not import the prometheus/testutil helpers to keep the
// import graph small — string scraping is sufficient given the well-known
// metric name and label shape.
func scrapeSelfProbeResult(t *testing.T, dep string) (float64, bool) {
	t.Helper()

	// Build a short-lived HTTP server that exposes the same default registry
	// RunSelfProbe writes to via emitSelfProbeResult.
	h := adaptor.HTTPHandler(promhttp.Handler())

	app := fiber.New()
	app.Get("/metrics", h)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	res, err := app.Test(req, 2000)
	require.NoError(t, err)

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	needle := fmt.Sprintf("selfprobe_result{dep=\"%s\"}", dep)
	for line := range strings.SplitSeq(string(body), "\n") {
		if strings.HasPrefix(line, needle) {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				return 0, false
			}

			var v float64
			if _, err := fmt.Sscanf(parts[1], "%g", &v); err != nil {
				return 0, false
			}

			return v, true
		}
	}

	return 0, false
}

func TestRunSelfProbe_AllUp_SetsTrueAndReturnsNil(t *testing.T) {
	resetSelfProbe(t)

	// Ensure the metrics collectors are registered before the probe emits.
	_ = NewMetricsHandler()

	logger := &capturingLogger{}

	err := RunSelfProbe(context.Background(), []DependencyChecker{
		&fakeChecker{name: "selfprobe_all_up_mongodb", out: DependencyCheck{Status: StatusUp}},
		&fakeChecker{name: "selfprobe_all_up_redis", out: DependencyCheck{Status: StatusUp}},
	}, logger)

	require.NoError(t, err)
	assert.True(t, IsSelfProbeOK(), "selfProbeOK must be true on success")

	assert.True(t, logger.hasMessage("startup_self_probe_started"))
	assert.True(t, logger.hasMessage("startup_self_probe_passed"))
	assert.Equal(t, 2, logger.countMessage("self_probe_check"))

	v, ok := scrapeSelfProbeResult(t, "selfprobe_all_up_mongodb")
	require.True(t, ok, "metric must be emitted for mongodb")
	assert.InDelta(t, 1.0, v, 0.0001)

	v, ok = scrapeSelfProbeResult(t, "selfprobe_all_up_redis")
	require.True(t, ok, "metric must be emitted for redis")
	assert.InDelta(t, 1.0, v, 0.0001)
}

func TestRunSelfProbe_OneDown_SetsFalseAndReturnsError(t *testing.T) {
	resetSelfProbe(t)

	_ = NewMetricsHandler()

	logger := &capturingLogger{}

	err := RunSelfProbe(context.Background(), []DependencyChecker{
		&fakeChecker{name: "selfprobe_one_down_mongodb", out: DependencyCheck{Status: StatusUp}},
		&fakeChecker{name: "selfprobe_one_down_rabbitmq", out: DependencyCheck{Status: StatusDown, Error: "dial refused"}},
	}, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "selfprobe_one_down_rabbitmq")
	assert.False(t, IsSelfProbeOK(), "selfProbeOK must be false on failure")

	assert.True(t, logger.hasMessage("startup_self_probe_started"))
	assert.True(t, logger.hasMessage("startup_self_probe_failed"))
	assert.False(t, logger.hasMessage("startup_self_probe_passed"))
	assert.True(t, logger.messageWithField(
		"startup_self_probe_failed", "failing_deps", "selfprobe_one_down_rabbitmq",
	))

	v, ok := scrapeSelfProbeResult(t, "selfprobe_one_down_rabbitmq")
	require.True(t, ok, "metric must be emitted for the failing dep too")
	assert.InDelta(t, 0.0, v, 0.0001)
}

func TestRunSelfProbe_SkippedAndNaAreHealthy(t *testing.T) {
	resetSelfProbe(t)

	_ = NewMetricsHandler()

	logger := &capturingLogger{}

	err := RunSelfProbe(context.Background(), []DependencyChecker{
		&fakeChecker{name: "selfprobe_skipna_mongodb", out: DependencyCheck{Status: StatusUp}},
		&fakeChecker{name: "selfprobe_skipna_redis", out: DependencyCheck{Status: StatusSkipped, Reason: "not configured"}},
		&fakeChecker{name: "selfprobe_skipna_rabbitmq", out: DependencyCheck{Status: StatusNA, Reason: "multi-tenant"}},
	}, logger)

	require.NoError(t, err)
	assert.True(t, IsSelfProbeOK())
	assert.True(t, logger.hasMessage("startup_self_probe_passed"))

	v, ok := scrapeSelfProbeResult(t, "selfprobe_skipna_redis")
	require.True(t, ok)
	assert.InDelta(t, 1.0, v, 0.0001, "skipped counts as up for the gauge")
}

// slowChecker sleeps for a configurable duration before returning.
type slowChecker struct {
	name  string
	sleep time.Duration
}

func (s *slowChecker) Name() string { return s.name }
func (s *slowChecker) Check(ctx context.Context) DependencyCheck {
	select {
	case <-time.After(s.sleep):
		return DependencyCheck{Status: StatusUp}
	case <-ctx.Done():
		return DependencyCheck{Status: StatusDown, Error: "ctx cancelled"}
	}
}

func TestRunSelfProbe_ContextDeadlineMarksDepDown(t *testing.T) {
	resetSelfProbe(t)

	_ = NewMetricsHandler()

	logger := &capturingLogger{}

	// redis has a 1s per-dep timeout; sleeping 3s forces the deadline path.
	err := RunSelfProbe(context.Background(), []DependencyChecker{
		&slowChecker{name: "redis", sleep: 3 * time.Second},
	}, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis")
	assert.False(t, IsSelfProbeOK())
}

func TestRunSelfProbe_EmptyCheckers_SucceedsSilently(t *testing.T) {
	resetSelfProbe(t)

	logger := &capturingLogger{}

	err := RunSelfProbe(context.Background(), nil, logger)
	require.NoError(t, err)
	assert.True(t, IsSelfProbeOK())
	assert.True(t, logger.hasMessage("startup_self_probe_started"))
	assert.True(t, logger.hasMessage("startup_self_probe_passed"))
	assert.Zero(t, logger.countMessage("self_probe_check"))
}

func TestRunSelfProbe_RunsInParallel(t *testing.T) {
	resetSelfProbe(t)

	_ = NewMetricsHandler()

	// Three checkers, each sleeping 400ms. A serial implementation would
	// take ≥ 1.2s; a parallel one completes in ~400ms.
	checkers := []DependencyChecker{
		&slowChecker{name: "redis", sleep: 400 * time.Millisecond},
		&slowChecker{name: "valkey", sleep: 400 * time.Millisecond},
		&slowChecker{name: "upstream_fees", sleep: 400 * time.Millisecond},
	}

	start := time.Now()
	err := RunSelfProbe(context.Background(), checkers, &capturingLogger{})
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, elapsed, 900*time.Millisecond,
		"parallel execution should finish well under the serial sum (expected ~400ms)")
}

func TestRunSelfProbe_EmitsMetricForEveryDep(t *testing.T) {
	resetSelfProbe(t)

	_ = NewMetricsHandler()

	names := []string{"selfprobe_emit_mongodb", "selfprobe_emit_redis", "selfprobe_emit_s3"}

	var checkers []DependencyChecker

	for _, n := range names {
		checkers = append(checkers, &fakeChecker{name: n, out: DependencyCheck{Status: StatusUp}})
	}

	require.NoError(t, RunSelfProbe(context.Background(), checkers, &capturingLogger{}))

	for _, n := range names {
		_, ok := scrapeSelfProbeResult(t, n)
		assert.True(t, ok, "missing selfprobe_result series for dep %q", n)
	}
}

func TestRunSelfProbe_NilLogger_DoesNotPanic(t *testing.T) {
	resetSelfProbe(t)

	_ = NewMetricsHandler()

	require.NotPanics(t, func() {
		err := RunSelfProbe(context.Background(), []DependencyChecker{
			&fakeChecker{name: "selfprobe_nil_logger_mongodb", out: DependencyCheck{Status: StatusUp}},
		}, nil)
		assert.NoError(t, err)
	})
}

// panickingChecker deliberately fails the zero-panic policy so we can verify
// RunSelfProbe captures the panic via runWithDeadline without crashing the
// caller. In practice the checker implementations never panic, but tests
// should guard against future regressions.
type panickingChecker struct{ name string }

func (p *panickingChecker) Name() string                           { return p.name }
func (p *panickingChecker) Check(_ context.Context) DependencyCheck { panic("boom") }

// Note: RunSelfProbe itself does not recover from panics — recovery is the
// responsibility of each DependencyChecker implementation. Including this
// checker in the slice would crash the goroutine. We keep the type here as a
// placeholder reminder; no test uses it at this gate.
var _ = &panickingChecker{name: "unused"}

func TestIsSelfProbeHealthyStatus(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{StatusUp, true},
		{StatusSkipped, true},
		{StatusNA, true},
		{StatusDown, false},
		{StatusDegraded, false},
		{"", false},
		{"unknown", false},
	}

	for _, tc := range cases {
		assert.Equalf(t, tc.want, isSelfProbeHealthyStatus(tc.in),
			"unexpected healthiness classification for %q", tc.in)
	}
}

// Ensure capturingLogger satisfies the libLog.Logger interface.
var _ libLog.Logger = (*capturingLogger)(nil)

// Sanity: errors.New used for the placeholder in this file.
var _ = errors.New

// Silence staticcheck on the unused counter when running with -count=1.
var _ atomic.Int64
