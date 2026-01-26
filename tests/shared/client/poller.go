package client

import (
	"context"
	"fmt"
	"time"
)

// Poller provides a generic polling mechanism for waiting on conditions.
// It replaces duplicated polling logic in ManagerClient.WaitForJobCompletion
// and SeaweedFSClient.WaitForFileSync.
type Poller struct {
	interval time.Duration
}

// NewPoller creates a new poller with the specified polling interval.
func NewPoller(interval time.Duration) *Poller {
	if interval <= 0 {
		panic("poller: interval must be positive")
	}
	return &Poller{interval: interval}
}

// ConditionFunc is a function that checks a condition.
// Returns (result, done, error) where:
// - result is the final value when done
// - done indicates if the condition is met
// - error indicates a failure that should stop polling
type ConditionFunc[T any] func() (T, bool, error)

// WaitFor polls the condition until it returns done=true, an error occurs,
// or the timeout is reached.
func (p *Poller) WaitFor(ctx context.Context, timeout time.Duration, condition func() (string, bool, error)) (string, error) {
	return WaitFor(p, ctx, timeout, condition)
}

// WaitFor is a generic function that polls the condition until it returns done=true,
// an error occurs, or the timeout is reached.
func WaitFor[T any](p *Poller, ctx context.Context, timeout time.Duration, condition ConditionFunc[T]) (T, error) {
	var zero T

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("polling canceled: %w", ctx.Err())

		case <-timer.C:
			return zero, fmt.Errorf("timeout waiting for condition after %v", timeout)

		case <-ticker.C:
			result, done, err := condition()
			if err != nil {
				return zero, fmt.Errorf("condition check failed: %w", err)
			}

			if done {
				return result, nil
			}
		}
	}
}

// WaitForWithRetry polls with configurable retry behavior for transient errors.
func WaitForWithRetry[T any](
	p *Poller,
	ctx context.Context,
	timeout time.Duration,
	maxRetries int,
	condition ConditionFunc[T],
) (T, error) {
	var zero T

	if maxRetries < 1 {
		return zero, fmt.Errorf("maxRetries must be at least 1, got %d", maxRetries)
	}

	retries := 0

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("polling canceled: %w", ctx.Err())

		case <-timer.C:
			return zero, fmt.Errorf("timeout waiting for condition after %v", timeout)

		case <-ticker.C:
			result, done, err := condition()
			if err != nil {
				retries++
				if retries >= maxRetries {
					return zero, fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, err)
				}

				continue // Retry on error
			}

			retries = 0 // Reset on success

			if done {
				return result, nil
			}
		}
	}
}
