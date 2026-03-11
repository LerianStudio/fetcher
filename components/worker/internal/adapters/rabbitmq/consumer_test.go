package rabbitmq

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewConsumerRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	logger := log.NewNop()
	telemetry := &opentelemetry.Telemetry{}

	t.Run("default workers when zero", func(t *testing.T) {
		cr := NewConsumerRoutesWithAdapter(mockAdapter, 0, logger, telemetry)
		assert.Equal(t, 5, cr.numWorkers)
	})

	t.Run("custom workers preserved", func(t *testing.T) {
		cr := NewConsumerRoutesWithAdapter(mockAdapter, 10, logger, telemetry)
		assert.Equal(t, 10, cr.numWorkers)
	})
}

func TestConsumerRoutes_Register(t *testing.T) {
	cr := &ConsumerRoutes{
		routes: make(map[string]QueueHandlerFunc),
	}

	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		return nil
	}

	cr.Register("test-queue", handler)

	assert.NotNil(t, cr.routes["test-queue"])
}

func TestQueueHandlerFunc(t *testing.T) {
	var called bool
	var receivedBody []byte
	var receivedHeaders map[string]any

	handler := QueueHandlerFunc(func(ctx context.Context, body []byte, headers map[string]any) error {
		called = true
		receivedBody = body
		receivedHeaders = headers
		return nil
	})

	ctx := context.Background()
	body := []byte("test body")
	headers := map[string]any{"key": "value"}

	err := handler(ctx, body, headers)
	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, body, receivedBody)
	assert.Equal(t, "value", receivedHeaders["key"])
}

func TestConsumerRoutes_Shutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	mockAdapter.EXPECT().Shutdown(gomock.Any()).Return(nil)

	cr := &ConsumerRoutes{
		adapter: mockAdapter,
		Logger:  log.NewNop(),
	}

	cr.shutdownWg.Add(1)

	errCh := make(chan error, 1)
	go func() {
		// Simulate waiting
		time.Sleep(50 * time.Millisecond)
		cr.shutdownWg.Done()
	}()

	go func() {
		err := cr.Shutdown(context.Background())
		errCh <- err
	}()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(500 * time.Millisecond):
		t.Error("Shutdown wait timed out")
	}
}

func TestConsumerRoutes_Shutdown_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	expectedErr := assert.AnError
	mockAdapter.EXPECT().Shutdown(gomock.Any()).Return(expectedErr)

	cr := &ConsumerRoutes{
		adapter: mockAdapter,
		Logger:  log.NewNop(),
	}

	err := cr.Shutdown(context.Background())
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestConsumerRoutes_RunConsumers(t *testing.T) {
	t.Run("starts consumers for registered queues", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAdapter := rabbitmq.NewMockAdapter(ctrl)
		logger := log.NewNop()
		telemetry := &opentelemetry.Telemetry{}

		cr := NewConsumerRoutesWithAdapter(mockAdapter, 5, logger, telemetry)

		handler := func(ctx context.Context, body []byte, headers map[string]any) error {
			return nil
		}

		cr.Register("test-queue", handler)

		// Setup mock to simulate consumer loop that exits on context cancel
		ctx, cancel := context.WithCancel(context.Background())
		mockAdapter.EXPECT().ConsumerLoop(gomock.Any(), "test-queue", 5, gomock.Any()).DoAndReturn(
			func(ctx context.Context, queue string, concurrency int, h func(context.Context, []byte, map[string]any) error) error {
				<-ctx.Done()
				return ctx.Err()
			},
		)

		var wg sync.WaitGroup
		err := cr.RunConsumers(ctx, &wg)
		assert.NoError(t, err)

		// Give goroutine time to start
		time.Sleep(50 * time.Millisecond)

		// Cancel context to stop consumer
		cancel()

		// Wait for consumer to finish
		wg.Wait()
	})

	t.Run("starts multiple consumers", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAdapter := rabbitmq.NewMockAdapter(ctrl)
		logger := log.NewNop()
		telemetry := &opentelemetry.Telemetry{}

		cr := NewConsumerRoutesWithAdapter(mockAdapter, 3, logger, telemetry)

		handler1 := func(ctx context.Context, body []byte, headers map[string]any) error { return nil }
		handler2 := func(ctx context.Context, body []byte, headers map[string]any) error { return nil }

		cr.Register("queue-1", handler1)
		cr.Register("queue-2", handler2)

		ctx, cancel := context.WithCancel(context.Background())

		// Expect ConsumerLoop to be called for each queue
		mockAdapter.EXPECT().ConsumerLoop(gomock.Any(), gomock.Any(), 3, gomock.Any()).DoAndReturn(
			func(ctx context.Context, queue string, concurrency int, h func(context.Context, []byte, map[string]any) error) error {
				<-ctx.Done()
				return ctx.Err()
			},
		).Times(2)

		var wg sync.WaitGroup
		err := cr.RunConsumers(ctx, &wg)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		cancel()
		wg.Wait()
	})

	t.Run("no registered queues", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAdapter := rabbitmq.NewMockAdapter(ctrl)
		logger := log.NewNop()
		telemetry := &opentelemetry.Telemetry{}

		cr := NewConsumerRoutesWithAdapter(mockAdapter, 5, logger, telemetry)

		ctx := context.Background()
		var wg sync.WaitGroup

		err := cr.RunConsumers(ctx, &wg)
		assert.NoError(t, err)

		// Should complete immediately with no consumers
		wg.Wait()
	})
}
