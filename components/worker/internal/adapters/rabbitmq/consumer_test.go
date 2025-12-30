package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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

	done := make(chan struct{})
	go func() {
		// Simulate waiting
		time.Sleep(50 * time.Millisecond)
		cr.shutdownWg.Done()
	}()

	go func() {
		err := cr.Shutdown(context.Background())
		assert.NoError(t, err)
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("Shutdown wait timed out")
	}
}
