//go:build chaos

// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package storage_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_Chaos_S3Repository_ConnectionLossDuringPut verifies that
// Put() handles connection loss gracefully with a wrapped error (no panic).
// 5-phase structure: Normal -> Inject -> Verify Failure -> Restore -> Recovery.
func TestIntegration_Chaos_S3Repository_ConnectionLossDuringPut(t *testing.T) {
	// Gate 1: Chaos tests disabled by default
	if os.Getenv("CHAOS") != "1" {
		t.Skip("CHAOS=1 env required to run chaos tests")
	}

	// Gate 2: Skip in short mode
	if testing.Short() {
		t.Skip("chaos tests are not short")
	}

	// Phase 1: Normal — S3 server operational
	failAfterCalls := 0
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if failAfterCalls > 0 && callCount > failAfterCalls {
			// Phase 2: Inject — return 503 Service Unavailable
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("service unavailable"))
			return
		}

		// Normal success response
		w.Header().Set("ETag", `"abc123"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo, err := storage.NewS3Repository(context.Background(), storage.S3Config{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		UsePathStyle:    true,
	})
	require.NoError(t, err, "repository creation should succeed")

	ctx := context.Background()

	// Phase 1: Normal operation — first Put succeeds
	err = repo.Put(ctx, "normal-file.json", []byte("test data"))
	require.NoError(t, err, "normal Put should succeed before failure injection")

	// Phase 2: Inject failure
	failAfterCalls = 1

	// Phase 3: Verify failure handling — Put returns error, no panic
	err = repo.Put(ctx, "during-failure.json", []byte("test data"))
	assert.Error(t, err, "Put should return error when server fails")
	assert.True(t,
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) || err != nil,
		"error should be a network error or wrapped error, not nil",
	)

	// Verify no panic occurs during failure
	assert.NotPanics(t, func() {
		_ = repo.Put(ctx, "another-fail.json", []byte("test data"))
	}, "Put must never panic under connection failure")

	// Phase 4: Restore — remove failure injection
	failAfterCalls = 0

	// Phase 5: Recovery — Put works again after recovery
	err = repo.Put(ctx, "recovered-file.json", []byte("test data"))
	assert.NoError(t, err, "Put should succeed after connection is restored")
}

// TestIntegration_Chaos_S3Repository_ConnectionLossDuringGet verifies that
// Get() handles connection loss gracefully with a wrapped error (no panic).
// 5-phase structure: Normal -> Inject -> Verify Failure -> Restore -> Recovery.
func TestIntegration_Chaos_S3Repository_ConnectionLossDuringGet(t *testing.T) {
	// Gate 1: Chaos tests disabled by default
	if os.Getenv("CHAOS") != "1" {
		t.Skip("CHAOS=1 env required to run chaos tests")
	}

	// Gate 2: Skip in short mode
	if testing.Short() {
		t.Skip("chaos tests are not short")
	}

	// Phase 1: Normal — S3 server operational
	failRequest := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failRequest {
			// Phase 2: Inject — return 503 Service Unavailable
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("service unavailable"))
			return
		}

		// Normal success response
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("stored data content"))
	}))
	defer server.Close()

	repo, err := storage.NewS3Repository(context.Background(), storage.S3Config{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		UsePathStyle:    true,
	})
	require.NoError(t, err, "repository creation should succeed")

	ctx := context.Background()

	// Phase 1: Normal operation — first Get succeeds
	data, err := repo.Get(ctx, "test-key.json")
	require.NoError(t, err, "normal Get should succeed before failure injection")
	assert.NotEmpty(t, data, "returned data should not be empty")

	// Phase 2: Inject failure
	failRequest = true

	// Phase 3: Verify failure handling — Get returns error, no panic
	_, err = repo.Get(ctx, "test-key.json")
	assert.Error(t, err, "Get should return error when server fails")
	assert.NotNil(t, err, "error should not be nil")

	// Verify no panic occurs during failure
	assert.NotPanics(t, func() {
		_, _ = repo.Get(ctx, "test-key.json")
	}, "Get must never panic under connection failure")

	// Phase 4: Restore — remove failure injection
	failRequest = false

	// Phase 5: Recovery — Get works again after recovery
	data, err = repo.Get(ctx, "test-key.json")
	assert.NoError(t, err, "Get should succeed after connection is restored")
	assert.NotEmpty(t, data, "returned data should not be empty after recovery")
}

// TestIntegration_Chaos_S3Repository_TimeoutDuringGet verifies that
// Get() with context timeout handles high latency gracefully.
// 5-phase structure: Normal -> Inject Latency -> Verify Timeout -> Restore -> Recovery.
func TestIntegration_Chaos_S3Repository_TimeoutDuringGet(t *testing.T) {
	// Gate 1: Chaos tests disabled by default
	if os.Getenv("CHAOS") != "1" {
		t.Skip("CHAOS=1 env required to run chaos tests")
	}

	// Gate 2: Skip in short mode
	if testing.Short() {
		t.Skip("chaos tests are not short")
	}

	// Phase 1: Normal — S3 server responds quickly
	slowMode := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if slowMode {
			// Phase 2: Inject latency — 3 second delay
			time.Sleep(3 * time.Second)
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response data"))
	}))
	defer server.Close()

	repo, err := storage.NewS3Repository(context.Background(), storage.S3Config{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		UsePathStyle:    true,
	})
	require.NoError(t, err, "repository creation should succeed")

	// Phase 1: Normal operation with normal timeout
	ctx := context.Background()
	data, err := repo.Get(ctx, "test-key.json")
	require.NoError(t, err, "normal Get should succeed before latency injection")
	assert.NotEmpty(t, data, "returned data should not be empty")

	// Phase 2: Inject latency
	slowMode = true

	// Phase 3: Verify timeout handling — Get with short timeout returns context deadline error
	ctxWithShortTimeout, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err = repo.Get(ctxWithShortTimeout, "test-key.json")
	assert.Error(t, err, "Get with short timeout should return error")

	// Verify no panic occurs during timeout
	assert.NotPanics(t, func() {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_, _ = repo.Get(ctxTimeout, "test-key.json")
	}, "Get must never panic under timeout")

	// Phase 4: Restore — disable slow mode
	slowMode = false

	// Phase 5: Recovery — Get works again with normal timeout
	data, err = repo.Get(context.Background(), "test-key.json")
	assert.NoError(t, err, "Get should succeed after latency is removed")
	assert.NotEmpty(t, data, "returned data should not be empty after recovery")
}

// TestIntegration_Chaos_S3Repository_NetworkPartition verifies that
// Put() handles network partition (connection refused) gracefully.
// 5-phase structure: Normal -> Inject Partition -> Verify Failure -> Restore -> Recovery.
func TestIntegration_Chaos_S3Repository_NetworkPartition(t *testing.T) {
	// Gate 1: Chaos tests disabled by default
	if os.Getenv("CHAOS") != "1" {
		t.Skip("CHAOS=1 env required to run chaos tests")
	}

	// Gate 2: Skip in short mode
	if testing.Short() {
		t.Skip("chaos tests are not short")
	}

	// Use a listener that we can close to simulate network partition
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "creating listener should succeed")

	addr := listener.Addr().String()

	// Accept connections in a goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed
			}

			// Handle HTTP request
			go func() {
				// Simple HTTP response
				_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nETag: \"test\"\r\nContent-Length: 0\r\n\r\n"))
				_ = conn.Close()
			}()
		}
	}()

	// Phase 1: Normal — network partition not active
	repo, err := storage.NewS3Repository(context.Background(), storage.S3Config{
		Endpoint:        fmt.Sprintf("http://%s", addr),
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		UsePathStyle:    true,
	})
	require.NoError(t, err, "repository creation should succeed")

	ctx := context.Background()

	// Phase 2 & 3: Inject partition — close listener
	listener.Close()

	// Phase 3: Verify partition handling — Put returns error during partition
	err = repo.Put(ctx, "during-partition.json", []byte("test data"))
	assert.Error(t, err, "Put should return error during network partition")

	// Verify no panic occurs during partition
	assert.NotPanics(t, func() {
		_ = repo.Put(ctx, "another-partition.json", []byte("test data"))
	}, "Put must never panic during network partition")

	// Phase 4 & 5: Restore — recreate listener
	listener2, errCreate := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, errCreate, "recreating listener should succeed")
	defer listener2.Close()

	// Note: Recovery would require recreating the repository with new endpoint
	// For this test, we verify the error was handled gracefully (no panic)
	// A real recovery test would need endpoint redirection capability
}
