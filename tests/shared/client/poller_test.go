package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoller_WaitFor_Success(t *testing.T) {
	callCount := 0
	condition := func() (string, bool, error) {
		callCount++
		if callCount >= 3 {
			return "done", true, nil
		}
		return "", false, nil
	}

	poller := NewPoller(10 * time.Millisecond)
	result, err := poller.WaitFor(context.Background(), 1*time.Second, condition)

	require.NoError(t, err)
	assert.Equal(t, "done", result)
	assert.Equal(t, 3, callCount)
}

func TestPoller_WaitFor_Timeout(t *testing.T) {
	condition := func() (string, bool, error) {
		return "", false, nil // Never completes
	}

	poller := NewPoller(10 * time.Millisecond)
	_, err := poller.WaitFor(context.Background(), 50*time.Millisecond, condition)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestPoller_WaitFor_Error(t *testing.T) {
	condition := func() (string, bool, error) {
		return "", false, errors.New("some error")
	}

	poller := NewPoller(10 * time.Millisecond)
	_, err := poller.WaitFor(context.Background(), 1*time.Second, condition)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "some error")
}

func TestPoller_WaitFor_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	condition := func() (string, bool, error) {
		return "", false, nil // Never completes
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	poller := NewPoller(10 * time.Millisecond)
	_, err := poller.WaitFor(ctx, 1*time.Second, condition)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "canceled")
}

func TestWaitForWithRetry_Success(t *testing.T) {
	callCount := 0
	condition := func() (string, bool, error) {
		callCount++
		if callCount >= 3 {
			return "done", true, nil
		}
		return "", false, nil
	}

	poller := NewPoller(10 * time.Millisecond)
	result, err := WaitForWithRetry(poller, context.Background(), 1*time.Second, 5, condition)

	require.NoError(t, err)
	assert.Equal(t, "done", result)
	assert.Equal(t, 3, callCount)
}

func TestWaitForWithRetry_RetryOnError(t *testing.T) {
	callCount := 0
	condition := func() (string, bool, error) {
		callCount++
		// First 2 calls return errors, then succeed
		if callCount <= 2 {
			return "", false, errors.New("transient error")
		}
		return "success", true, nil
	}

	poller := NewPoller(10 * time.Millisecond)
	result, err := WaitForWithRetry(poller, context.Background(), 1*time.Second, 5, condition)

	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount)
}

func TestWaitForWithRetry_MaxRetriesExceeded(t *testing.T) {
	callCount := 0
	condition := func() (string, bool, error) {
		callCount++
		return "", false, errors.New("persistent error")
	}

	poller := NewPoller(10 * time.Millisecond)
	_, err := WaitForWithRetry(poller, context.Background(), 1*time.Second, 3, condition)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max retries (3) exceeded")
	assert.Equal(t, 3, callCount)
}
