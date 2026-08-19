package in

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bridgeSentinelHeader = "X-Bridge-Sentinel"

type bridgeSentinelContextKey struct{}

type bridgeProbeOutput struct {
	Body bridgeProbeBody
}

type bridgeProbeBody struct {
	HeaderBefore  string `json:"headerBefore"`
	HeaderAfter   string `json:"headerAfter"`
	ContextBefore string `json:"contextBefore"`
	ContextAfter  string `json:"contextAfter"`
}

type bridgeProbeResult struct {
	Sentinel string
	Status   int
	Body     bridgeProbeBody
	Err      error
	CloseErr error
}

func TestCaptureFiberContext_IsolatesConcurrentRequests(t *testing.T) {
	const requestCount = 16

	app := fiber.New()
	app.Use(func(fiberCtx fiber.Ctx) error {
		sentinel := fiberCtx.Get(bridgeSentinelHeader)
		requestContext := context.WithValue(
			fiberCtx.Context(),
			bridgeSentinelContextKey{},
			sentinel,
		)
		fiberCtx.SetContext(requestContext)

		return fiberCtx.Next()
	})

	api := BuildHumaAPI(app, false)
	entered := make(chan struct{}, requestCount)
	release := make(chan struct{})
	registerTypedOperation(
		app,
		api,
		huma.Operation{
			OperationID: "probeConcurrentFiberContext",
			Method:      http.MethodGet,
			Path:        "/bridge-probe",
		},
		"bridge-probe",
		"read",
		nil,
		func(ctx context.Context, _ *emptyInput) (*bridgeProbeOutput, error) {
			fiberCtx, err := fiberContext(ctx)
			if err != nil {
				return nil, err
			}

			body := bridgeProbeBody{
				HeaderBefore:  fiberCtx.Get(bridgeSentinelHeader),
				ContextBefore: contextString(ctx, bridgeSentinelContextKey{}),
			}
			entered <- struct{}{}
			<-release

			fiberCtx, err = fiberContext(ctx)
			if err != nil {
				return nil, err
			}
			body.HeaderAfter = fiberCtx.Get(bridgeSentinelHeader)
			body.ContextAfter = contextString(ctx, bridgeSentinelContextKey{})

			return &bridgeProbeOutput{Body: body}, nil
		},
	)

	results := make(chan bridgeProbeResult, requestCount)
	var waitGroup sync.WaitGroup
	waitGroup.Add(requestCount)

	for index := range requestCount {
		sentinel := fmt.Sprintf("request-%02d", index)
		go func() {
			defer waitGroup.Done()

			request := httptest.NewRequest(http.MethodGet, "/bridge-probe", nil)
			request.Header.Set(bridgeSentinelHeader, sentinel)
			response, err := app.Test(request)
			result := bridgeProbeResult{Sentinel: sentinel, Err: err}
			if err == nil {
				result.Status = response.StatusCode
				result.Err = json.NewDecoder(response.Body).Decode(&result.Body)
				result.CloseErr = response.Body.Close()
			}

			results <- result
		}()
	}

	released := false
	defer func() {
		if !released {
			close(release)
		}
		waitGroup.Wait()
	}()

	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for range requestCount {
		select {
		case <-entered:
		case result := <-results:
			require.NoError(t, result.Err)
			t.Fatalf("request %q completed before every request reached the barrier", result.Sentinel)
		case <-deadline.C:
			t.Fatal("concurrent requests did not reach the barrier")
		}
	}

	close(release)
	released = true

	for range requestCount {
		result := <-results
		require.NoError(t, result.Err, result.Sentinel)
		require.NoError(t, result.CloseErr, result.Sentinel)
		assert.Equal(t, http.StatusOK, result.Status, result.Sentinel)
		assert.Equal(t, result.Sentinel, result.Body.HeaderBefore, result.Sentinel)
		assert.Equal(t, result.Sentinel, result.Body.HeaderAfter, result.Sentinel)
		assert.Equal(t, result.Sentinel, result.Body.ContextBefore, result.Sentinel)
		assert.Equal(t, result.Sentinel, result.Body.ContextAfter, result.Sentinel)
	}
}

func contextString(ctx context.Context, key any) string {
	value, _ := ctx.Value(key).(string)
	return value
}
