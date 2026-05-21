package bootstrap

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"

	workerRabbitMQ "github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/components/worker/internal/services"
	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	pkgRabbitMQ "github.com/LerianStudio/fetcher/pkg/rabbitmq"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/mock/gomock"
)

type fakeLicenseTerminator struct {
	called bool
	msg    string
}

func (f *fakeLicenseTerminator) Terminate(msg string) {
	f.called = true
	f.msg = msg
}

func TestNewMultiQueueConsumerRegistersQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Setenv("RABBITMQ_FETCHER_WORK_QUEUE", "worker.jobs")

	adapter := pkgRabbitMQ.NewMockAdapter(ctrl)
	adapter.EXPECT().
		ConsumerLoop(gomock.Any(), "worker.jobs", 2, gomock.Any()).
		DoAndReturn(func(context.Context, string, int, func(context.Context, []byte, map[string]any) error) error {
			return nil
		})
	routes := workerRabbitMQ.NewConsumerRoutesWithAdapter(adapter, 2, testBootstrapLogger(), &libOtel.Telemetry{})
	useCase := &services.UseCase{}

	consumer := NewMultiQueueConsumer(routes, useCase, "worker.jobs", testBootstrapLogger(), nil, 0)

	if consumer.UseCase != useCase {
		t.Fatal("expected use case to be assigned")
	}

	var wg sync.WaitGroup
	if err := routes.RunConsumers(context.Background(), &wg); err != nil {
		t.Fatalf("expected no error running consumers, got %v", err)
	}
	wg.Wait()
}

func TestHandlerGenerateReport_DelegatesToUseCase(t *testing.T) {
	originalExtractExternalData := extractExternalData
	t.Cleanup(func() {
		extractExternalData = originalExtractExternalData
	})

	consumer := &MultiQueueConsumer{UseCase: &services.UseCase{}}
	ctx := contextWithBootstrapTracking(t)
	headers := map[string]any{"x-test": "value"}
	body := []byte(`{"jobId":"123"}`)

	t.Run("success", func(t *testing.T) {
		var called atomic.Bool
		extractExternalData = func(uc *services.UseCase, gotCtx context.Context, gotBody []byte, gotHeaders map[string]any) error {
			called.Store(true)
			if uc != consumer.UseCase {
				t.Fatal("unexpected use case passed to extractor")
			}
			if string(gotBody) != string(body) {
				t.Fatalf("unexpected body: %s", gotBody)
			}
			if gotHeaders["x-test"] != "value" {
				t.Fatalf("unexpected headers: %+v", gotHeaders)
			}
			if gotCtx == nil {
				t.Fatal("expected non-nil context")
			}
			return nil
		}

		if err := consumer.handlerGenerateReport(ctx, body, headers); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !called.Load() {
			t.Fatal("expected extractExternalData to be called")
		}
	})

	t.Run("propagates error", func(t *testing.T) {
		wantErr := errors.New("extract failed")
		extractExternalData = func(*services.UseCase, context.Context, []byte, map[string]any) error {
			return wantErr
		}

		if err := consumer.handlerGenerateReport(ctx, body, headers); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestMultiQueueConsumerRun(t *testing.T) {
	originalNotifySignals := notifySignals
	t.Cleanup(func() {
		notifySignals = originalNotifySignals
	})

	notifySignals = func(c chan<- os.Signal, _ ...os.Signal) {
		go func() {
			c <- syscall.SIGTERM
		}()
	}

	t.Setenv("RABBITMQ_FETCHER_WORK_QUEUE", "worker.jobs")

	t.Run("graceful shutdown", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		adapter := pkgRabbitMQ.NewMockAdapter(ctrl)
		adapter.EXPECT().
			ConsumerLoop(gomock.Any(), "worker.jobs", 2, gomock.Any()).
			DoAndReturn(func(ctx context.Context, _ string, _ int, _ func(context.Context, []byte, map[string]any) error) error {
				<-ctx.Done()
				return nil
			})
		adapter.EXPECT().Shutdown(gomock.Any()).Return(nil)

		routes := workerRabbitMQ.NewConsumerRoutesWithAdapter(adapter, 2, testBootstrapLogger(), &libOtel.Telemetry{})
		consumer := NewMultiQueueConsumer(routes, &services.UseCase{}, "worker.jobs", testBootstrapLogger(), nil, 0)

		if err := consumer.Run(nil); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("shutdown error is returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		adapter := pkgRabbitMQ.NewMockAdapter(ctrl)
		adapter.EXPECT().
			ConsumerLoop(gomock.Any(), "worker.jobs", 2, gomock.Any()).
			DoAndReturn(func(ctx context.Context, _ string, _ int, _ func(context.Context, []byte, map[string]any) error) error {
				<-ctx.Done()
				return nil
			})
		adapter.EXPECT().Shutdown(gomock.Any()).Return(errors.New("shutdown failed"))

		routes := workerRabbitMQ.NewConsumerRoutesWithAdapter(adapter, 2, testBootstrapLogger(), &libOtel.Telemetry{})
		consumer := NewMultiQueueConsumer(routes, &services.UseCase{}, "worker.jobs", testBootstrapLogger(), nil, 0)

		err := consumer.Run(nil)
		if err == nil || !strings.Contains(err.Error(), "shutdown failed") {
			t.Fatalf("expected shutdown failed error, got %v", err)
		}
	})

	// On SIGTERM the consumer must flip the draining flag BEFORE cancelling
	// the consumer context — otherwise in-flight deliveries get nacked
	// while kube-proxy still sees /readyz=200.
	t.Run("sets readyz draining before cancelling context", func(t *testing.T) {
		readyz.SetDraining(false)
		t.Cleanup(func() { readyz.SetDraining(false) })

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// drainObservedAt records the value of IsDraining() at the moment
		// ctx is cancelled — if the flag is still false, the ordering is
		// wrong and the test fails.
		var drainObservedAt bool

		adapter := pkgRabbitMQ.NewMockAdapter(ctrl)
		adapter.EXPECT().
			ConsumerLoop(gomock.Any(), "worker.jobs", 2, gomock.Any()).
			DoAndReturn(func(ctx context.Context, _ string, _ int, _ func(context.Context, []byte, map[string]any) error) error {
				<-ctx.Done()
				drainObservedAt = readyz.IsDraining()
				return nil
			})
		adapter.EXPECT().Shutdown(gomock.Any()).Return(nil)

		routes := workerRabbitMQ.NewConsumerRoutesWithAdapter(adapter, 2, testBootstrapLogger(), &libOtel.Telemetry{})
		consumer := NewMultiQueueConsumer(routes, &services.UseCase{}, "worker.jobs", testBootstrapLogger(), nil, 0)

		if err := consumer.Run(nil); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !drainObservedAt {
			t.Fatal("draining flag MUST be true before consumer context is cancelled")
		}
	})
}

func TestServiceRun(t *testing.T) {
	originalRunLauncher := runLauncher
	t.Cleanup(func() {
		runLauncher = originalRunLauncher
	})

	t.Run("terminates license manager after launcher", func(t *testing.T) {
		terminator := &fakeLicenseTerminator{}
		consumer := &MultiQueueConsumer{}
		logger := testBootstrapLogger()

		called := false
		runLauncher = func(gotLogger libLog.Logger, gotConsumer *MultiQueueConsumer, _ *HealthServer) {
			called = true
			if gotLogger != logger {
				t.Fatal("unexpected logger passed to launcher")
			}
			if gotConsumer != consumer {
				t.Fatal("unexpected consumer passed to launcher")
			}
		}

		service := &Service{
			MultiQueueConsumer: consumer,
			Logger:             logger,
			licenseShutdown:    terminator,
		}

		service.Run()

		if !called {
			t.Fatal("expected launcher to be invoked")
		}
		if !terminator.called {
			t.Fatal("expected license terminator to be called")
		}
		if terminator.msg != "Consumers are done." {
			t.Fatalf("unexpected terminate message: %s", terminator.msg)
		}
	})

	t.Run("nil license terminator is allowed", func(t *testing.T) {
		called := false
		runLauncher = func(libLog.Logger, *MultiQueueConsumer, *HealthServer) {
			called = true
		}

		service := &Service{
			MultiQueueConsumer: &MultiQueueConsumer{},
			Logger:             testBootstrapLogger(),
		}

		service.Run()

		if !called {
			t.Fatal("expected launcher to be invoked")
		}
	})
}

func contextWithBootstrapTracking(t *testing.T) context.Context {
	t.Helper()

	requestID := uuid.New()
	ctx := observability.ContextWithHeaderID(context.Background(), requestID.String())
	ctx = observability.ContextWithLogger(ctx, testBootstrapLogger())

	return observability.ContextWithTracer(ctx, otel.Tracer("bootstrap-test"))
}
