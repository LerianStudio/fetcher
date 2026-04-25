package readyz

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
)

// RunSelfProbe executes every registered DependencyChecker exactly once at
// startup, aggregates the outcomes, and flips the process-wide self-probe
// flag accordingly. This is Gate 7 of ring:dev-readyz.
//
// Execution model: checkers run in PARALLEL, one goroutine per checker, under
// a per-dep deadline equal to PerDepTimeout(name). This matches Handler.Run
// (handler.go) so dependency-probe timing is the same at boot and at runtime,
// and it keeps startup fast — a misbehaving dep holds up the probe only by
// its own budget, not the sum of every dep's budget.
//
// Contract:
//
//   - Entry: SetSelfProbe(false) to guarantee /health reports unhealthy while
//     the probe is running. A service that crashes mid-probe therefore never
//     reports healthy by accident.
//   - Per dep: log the outcome via the provided Logger (Info on success,
//     Error on failure), and ALWAYS emit the selfprobe_result gauge via
//     emitSelfProbeResult so Grafana dashboards show the last boot outcome.
//   - Success: every check in {up, skipped, n/a} → SetSelfProbe(true),
//     log "startup_self_probe_passed", return nil.
//   - Failure: at least one check in {down, degraded} → SetSelfProbe(false),
//     log "startup_self_probe_failed", return an error listing the failing
//     deps. The error is advisory — callers MUST NOT panic / os.Exit / log.Fatal
//     on it (ring:dev-readyz zero-panic policy). The pod must stay up so
//     /health can serve 503 and Kubernetes can collect the logs.
//
// The function does not itself call os.Exit, log.Fatal or panic. It is safe
// to invoke from bootstrap code that expects a standard error return.
//
// Empty checker slice is treated as "nothing to probe" and succeeds with
// SetSelfProbe(true) — a service with zero external dependencies is trivially
// ready. No metrics are emitted in that case (nothing to label).
func RunSelfProbe(ctx context.Context, checkers []DependencyChecker, logger libLog.Logger) error {
	if logger == nil {
		logger = libLog.NewNop()
	}

	// Pessimistic initialization: stay unhealthy until proven otherwise. If
	// the caller panics mid-probe, /health keeps returning 503 and the pod
	// can be restarted by the kubelet.
	SetSelfProbe(false)

	logger.Log(ctx, libLog.LevelInfo, "startup_self_probe_started",
		libLog.Int("checker_count", len(checkers)),
	)

	if len(checkers) == 0 {
		SetSelfProbe(true)
		logger.Log(ctx, libLog.LevelInfo, "startup_self_probe_passed",
			libLog.String("reason", "no checkers registered"),
		)

		return nil
	}

	type probeResult struct {
		name    string
		status  string
		err     string
		elapsed time.Duration
	}

	results := make(chan probeResult, len(checkers))

	var wg sync.WaitGroup

	for _, checker := range checkers {
		wg.Add(1)

		go func(c DependencyChecker) {
			defer wg.Done()

			name := c.Name()

			depCtx, cancel := context.WithTimeout(ctx, PerDepTimeout(name))
			defer cancel()

			start := time.Now()
			check := runWithDeadline(depCtx, c)
			elapsed := time.Since(start)

			results <- probeResult{
				name:    name,
				status:  check.Status,
				err:     check.Error,
				elapsed: elapsed,
			}
		}(checker)
	}

	wg.Wait()
	close(results)

	var failing []string

	for r := range results {
		up := isSelfProbeHealthyStatus(r.status)

		emitSelfProbeResult(r.name, up)

		if up {
			logger.Log(ctx, libLog.LevelInfo, "self_probe_check",
				libLog.String("dep", r.name),
				libLog.String("status", r.status),
				libLog.Int("elapsed_ms", int(r.elapsed.Milliseconds())),
			)

			continue
		}

		failing = append(failing, r.name)

		logger.Log(ctx, libLog.LevelError, "self_probe_check",
			libLog.String("dep", r.name),
			libLog.String("status", r.status),
			libLog.String("error", r.err),
			libLog.Int("elapsed_ms", int(r.elapsed.Milliseconds())),
		)
	}

	if len(failing) == 0 {
		SetSelfProbe(true)
		logger.Log(ctx, libLog.LevelInfo, "startup_self_probe_passed",
			libLog.Int("checker_count", len(checkers)),
		)

		return nil
	}

	SetSelfProbe(false)
	logger.Log(ctx, libLog.LevelError, "startup_self_probe_failed",
		libLog.String("failing_deps", strings.Join(failing, ",")),
		libLog.Int("failing_count", len(failing)),
	)

	return fmt.Errorf("startup self-probe failed: %s", strings.Join(failing, ", "))
}

// isSelfProbeHealthyStatus returns true for the statuses that count as "probe
// passed" for the purposes of the startup self-probe aggregation rule. It
// mirrors aggregateStatus in handler.go — a dep that reports skipped or n/a
// is treated as healthy because the probe did not need to run. Only down or
// degraded count as a self-probe failure.
func isSelfProbeHealthyStatus(status string) bool {
	switch status {
	case StatusUp, StatusSkipped, StatusNA:
		return true
	default:
		return false
	}
}
