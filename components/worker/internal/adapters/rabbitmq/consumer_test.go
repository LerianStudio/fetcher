package rabbitmq

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v2/commons/rabbitmq"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// mockLogger is a simplified mock for log.Logger
type mockLogger struct{}

func (m *mockLogger) Info(args ...any)                                      {}
func (m *mockLogger) Infof(format string, args ...any)                      {}
func (m *mockLogger) Infoln(args ...any)                                    {}
func (m *mockLogger) Warn(args ...any)                                      {}
func (m *mockLogger) Warnf(format string, args ...any)                      {}
func (m *mockLogger) Warnln(args ...any)                                    {}
func (m *mockLogger) Warning(args ...any)                                   {}
func (m *mockLogger) Warningf(format string, args ...any)                   {}
func (m *mockLogger) Warningln(args ...any)                                 {}
func (m *mockLogger) Error(args ...any)                                     {}
func (m *mockLogger) Errorf(format string, args ...any)                     {}
func (m *mockLogger) Errorln(args ...any)                                   {}
func (m *mockLogger) Debug(args ...any)                                     {}
func (m *mockLogger) Debugf(format string, args ...any)                     {}
func (m *mockLogger) Debugln(args ...any)                                   {}
func (m *mockLogger) Fatal(args ...any)                                     {}
func (m *mockLogger) Fatalf(format string, args ...any)                     {}
func (m *mockLogger) Fatalln(args ...any)                                   {}
func (m *mockLogger) Panic(args ...any)                                     {}
func (m *mockLogger) Panicf(format string, args ...any)                     {}
func (m *mockLogger) Panicln(args ...any)                                   {}
func (m *mockLogger) Trace(args ...any)                                     {}
func (m *mockLogger) Tracef(format string, args ...any)                     {}
func (m *mockLogger) Traceln(args ...any)                                   {}
func (m *mockLogger) Print(args ...any)                                     {}
func (m *mockLogger) Printf(format string, args ...any)                     {}
func (m *mockLogger) Println(args ...any)                                   {}
func (m *mockLogger) Log(level string, args ...any)                         {}
func (m *mockLogger) Logf(level string, format string, args ...any)         {}
func (m *mockLogger) Logln(level string, args ...any)                       {}
func (m *mockLogger) WithFields(fields ...any) log.Logger                   { return m }
func (m *mockLogger) WithField(key string, value any) log.Logger            { return m }
func (m *mockLogger) WithError(err error) log.Logger                        { return m }
func (m *mockLogger) WithDefaultMessageTemplate(template string) log.Logger { return m }
func (m *mockLogger) GetLevel() string                                      { return "" }
func (m *mockLogger) SetLevel(level string) error                           { return nil }
func (m *mockLogger) IsLevelEnabled(level string) bool                      { return false }
func (m *mockLogger) GetLogger() any                                        { return nil }
func (m *mockLogger) GetOutput() any                                        { return nil }
func (m *mockLogger) SetOutput(output any) error                            { return nil }
func (m *mockLogger) GetFormatter() any                                     { return nil }
func (m *mockLogger) SetFormatter(formatter any) error                      { return nil }
func (m *mockLogger) GetHooks() any                                         { return nil }
func (m *mockLogger) AddHook(hook any) error                                { return nil }
func (m *mockLogger) Clone() any                                            { return m }
func (m *mockLogger) GetContext() any                                       { return nil }
func (m *mockLogger) SetContext(ctx any) error                              { return nil }
func (m *mockLogger) GetCallerInfo() bool                                   { return false }
func (m *mockLogger) SetCallerInfo(enabled bool)                            {}
func (m *mockLogger) GetReportCaller() bool                                 { return false }
func (m *mockLogger) SetReportCaller(enabled bool)                          {}
func (m *mockLogger) GetExitFunc() any                                      { return nil }
func (m *mockLogger) SetExitFunc(exitFunc any) error                        { return nil }
func (m *mockLogger) GetBufferPool() any                                    { return nil }
func (m *mockLogger) SetBufferPool(pool any) error                          { return nil }
func (m *mockLogger) Sync() error                                           { return nil }

func TestNewConsumerRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	logger := &mockLogger{}
	telemetry := &opentelemetry.Telemetry{}

	t.Run("default workers when zero", func(t *testing.T) {
		cr := NewConsumerRoutesWithAdapter(mockAdapter, 0, logger, telemetry)
		assert.Equal(t, 5, cr.numWorkers)
	})

	t.Run("custom workers preserved", func(t *testing.T) {
		cr := NewConsumerRoutesWithAdapter(mockAdapter, 10, logger, telemetry)
		assert.Equal(t, 10, cr.numWorkers)
	})

	t.Run("nil telemetry does not panic", func(t *testing.T) {
		cr := NewConsumerRoutesWithAdapter(mockAdapter, 1, logger, nil)
		assert.NotNil(t, cr)
	})

	t.Run("non-dev env with nil signer returns error", func(t *testing.T) {
		conn := &libRabbitmq.RabbitMQConnection{
			ConnectionStringSource: "invalid-uri",
			Logger:                 logger,
		}

		cr, err := NewConsumerRoutes(conn, 1, logger, telemetry, nil, "production")
		assert.Error(t, err)
		assert.Nil(t, cr)
	})

	t.Run("dev env with nil signer is allowed", func(t *testing.T) {
		conn := &libRabbitmq.RabbitMQConnection{
			ConnectionStringSource: "invalid-uri",
			Logger:                 logger,
		}

		cr, err := NewConsumerRoutes(conn, 1, logger, telemetry, nil, "dev")
		assert.NoError(t, err)
		assert.NotNil(t, cr)
	})

	t.Run("non-dev env with signer succeeds", func(t *testing.T) {
		conn := &libRabbitmq.RabbitMQConnection{
			ConnectionStringSource: "invalid-uri",
			Logger:                 logger,
		}

		signer := crypto.NewMockSigner(ctrl)

		cr, err := NewConsumerRoutes(conn, 1, logger, telemetry, signer, "production")
		assert.NoError(t, err)
		assert.NotNil(t, cr)
	})

	t.Run("empty env name defaults to strict (fail-closed)", func(t *testing.T) {
		conn := &libRabbitmq.RabbitMQConnection{
			ConnectionStringSource: "invalid-uri",
			Logger:                 logger,
		}

		cr, err := NewConsumerRoutes(conn, 1, logger, telemetry, nil, "")
		assert.Error(t, err)
		assert.Nil(t, cr)
	})
}

func TestIsNonDevelopmentEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envName  string
		expected bool
	}{
		{name: "production", envName: "production", expected: true},
		{name: "empty defaults to strict", envName: "", expected: true},
		{name: "dev", envName: "dev", expected: false},
		{name: "development", envName: "development", expected: false},
		{name: "local", envName: "local", expected: false},
		{name: "test", envName: "test", expected: false},
		{name: "testing", envName: "testing", expected: false},
		{name: "DEV uppercase", envName: "DEV", expected: false},
		{name: "Production mixed case", envName: "Production", expected: true},
		{name: "staging is non-dev", envName: "staging", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isNonDevelopmentEnvironment(tt.envName))
		})
	}
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
		Logger:  &mockLogger{},
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
		Logger:  &mockLogger{},
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
		logger := &mockLogger{}
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
		logger := &mockLogger{}
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
		logger := &mockLogger{}
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
