package readyz

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
)

// RunSelfProbe runs every checker once at startup in parallel under
// PerDepTimeout, then flips the process-wide self-probe flag based on the
// aggregated outcome. The parallel model matches Handler.Run so probe
// timing is identical at boot and at runtime; a misbehaving dep delays the
// probe only by its own budget.
//
// Entry sets the flag to false so /health stays unhealthy until proven
// otherwise — a process that crashes mid-probe never reports healthy by
// accident. Each dep emits selfprobe_result; the function logs but never
// panics, os.Exit, or log.Fatal — failure is advisory and callers must
// keep the pod up so /health can serve 503 and Kubernetes can collect
// logs. An empty checker slice succeeds trivially.
func RunSelfProbe(ctx context.Context, checkers []DependencyChecker, logger libLog.Logger) error {
	if logger == nil {
		logger = libLog.NewNop()
	}

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

// isSelfProbeHealthyStatus mirrors aggregateStatus: skipped and n/a count
// as healthy because the probe did not need to run. Only down or degraded
// fail the self-probe.
func isSelfProbeHealthyStatus(status string) bool {
	switch status {
	case StatusUp, StatusSkipped, StatusNA:
		return true
	default:
		return false
	}
}
