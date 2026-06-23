package bootstrap

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	workerRabbitMQ "github.com/LerianStudio/fetcher/v2/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/v2/components/worker/internal/services"
	"github.com/LerianStudio/fetcher/v2/pkg/bootstrap/readyz"
	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	pkgRabbitMQ "github.com/LerianStudio/fetcher/v2/pkg/rabbitmq"
	libOutbox "github.com/LerianStudio/lib-commons/v5/commons/outbox"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

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

func TestHandlerGenerateReportDelivery_VerifiesTenantBoundSignatures(t *testing.T) {
	originalExtractExternalData := extractExternalData
	t.Cleanup(func() { extractExternalData = originalExtractExternalData })

	signer, err := crypto.NewHMACSigner([]byte("0123456789abcdef0123456789abcdef"), crypto.SignatureVersion)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	body := []byte(`{"jobId":"123e4567-e89b-12d3-a456-426614174000"}`)
	now := time.Now().Unix()
	signatureFor := func(tenantID string) string {
		return signer.Sign(pkgRabbitMQ.BuildMessageSignaturePayload(now, signer.SignatureVersion(), tenantID, "123e4567-e89b-12d3-a456-426614174000", "worker.exchange", "worker.key", body))
	}
	legacySignature := signer.Sign(crypto.BuildSignaturePayload(now, body))
	oldTimestamp := time.Now().Add(-(pkgRabbitMQ.DefaultSignatureTimestampTolerance + time.Minute)).Unix()
	oldLegacySignature := signer.Sign(crypto.BuildSignaturePayload(oldTimestamp, body))

	tests := []struct {
		name        string
		headers     amqp.Table
		ctxTenant   string
		allowLegacy bool
		wantHandled bool
	}{
		{
			name: "valid tenant-bound signature",
			headers: amqp.Table{
				pkgRabbitMQ.HeaderTenantID:           "tenant-a",
				pkgRabbitMQ.HeaderMessageSignature:   signatureFor("tenant-a"),
				pkgRabbitMQ.HeaderSignatureTimestamp: strconv.FormatInt(now, 10),
				pkgRabbitMQ.HeaderSignatureVersion:   signer.SignatureVersion(),
			},
			ctxTenant:   "tenant-a",
			wantHandled: true,
		},
		{
			name: "legacy body-only signature disabled rejects before handler",
			headers: amqp.Table{
				pkgRabbitMQ.HeaderTenantID:           "tenant-a",
				pkgRabbitMQ.HeaderMessageSignature:   legacySignature,
				pkgRabbitMQ.HeaderSignatureTimestamp: strconv.FormatInt(now, 10),
				pkgRabbitMQ.HeaderSignatureVersion:   signer.SignatureVersion(),
			},
			ctxTenant: "tenant-a",
		},
		{
			name: "legacy body-only signature enabled accepts matching tenant header",
			headers: amqp.Table{
				pkgRabbitMQ.HeaderTenantID:           "tenant-a",
				pkgRabbitMQ.HeaderMessageSignature:   legacySignature,
				pkgRabbitMQ.HeaderSignatureTimestamp: strconv.FormatInt(now, 10),
				pkgRabbitMQ.HeaderSignatureVersion:   signer.SignatureVersion(),
			},
			ctxTenant:   "tenant-a",
			allowLegacy: true,
			wantHandled: true,
		},
		{
			name: "tenant mismatch rejected before verification",
			headers: amqp.Table{
				pkgRabbitMQ.HeaderTenantID:           "tenant-b",
				pkgRabbitMQ.HeaderMessageSignature:   signatureFor("tenant-b"),
				pkgRabbitMQ.HeaderSignatureTimestamp: strconv.FormatInt(now, 10),
				pkgRabbitMQ.HeaderSignatureVersion:   signer.SignatureVersion(),
			},
			ctxTenant: "tenant-a",
		},
		{
			name: "missing tenant header rejected",
			headers: amqp.Table{
				pkgRabbitMQ.HeaderMessageSignature:   legacySignature,
				pkgRabbitMQ.HeaderSignatureTimestamp: strconv.FormatInt(now, 10),
				pkgRabbitMQ.HeaderSignatureVersion:   signer.SignatureVersion(),
			},
			ctxTenant: "tenant-a",
		},
		{
			name: "legacy body-only signature enabled still rejects missing tenant header",
			headers: amqp.Table{
				pkgRabbitMQ.HeaderMessageSignature:   legacySignature,
				pkgRabbitMQ.HeaderSignatureTimestamp: strconv.FormatInt(now, 10),
				pkgRabbitMQ.HeaderSignatureVersion:   signer.SignatureVersion(),
			},
			ctxTenant:   "tenant-a",
			allowLegacy: true,
		},
		{
			name: "missing authoritative context rejected before verification",
			headers: amqp.Table{
				pkgRabbitMQ.HeaderTenantID:           "tenant-a",
				pkgRabbitMQ.HeaderMessageSignature:   signatureFor("tenant-a"),
				pkgRabbitMQ.HeaderSignatureTimestamp: strconv.FormatInt(now, 10),
				pkgRabbitMQ.HeaderSignatureVersion:   signer.SignatureVersion(),
			},
		},
		{
			name: "tampered signature rejected",
			headers: amqp.Table{
				pkgRabbitMQ.HeaderTenantID:           "tenant-a",
				pkgRabbitMQ.HeaderMessageSignature:   "tampered",
				pkgRabbitMQ.HeaderSignatureTimestamp: strconv.FormatInt(now, 10),
				pkgRabbitMQ.HeaderSignatureVersion:   signer.SignatureVersion(),
			},
			ctxTenant: "tenant-a",
		},
		{
			name: "old legacy body-only signature rejected after tolerance",
			headers: amqp.Table{
				pkgRabbitMQ.HeaderTenantID:           "tenant-a",
				pkgRabbitMQ.HeaderMessageSignature:   oldLegacySignature,
				pkgRabbitMQ.HeaderSignatureTimestamp: strconv.FormatInt(oldTimestamp, 10),
				pkgRabbitMQ.HeaderSignatureVersion:   signer.SignatureVersion(),
			},
			ctxTenant: "tenant-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handled atomic.Bool
			extractExternalData = func(*services.UseCase, context.Context, []byte, map[string]any) error {
				handled.Store(true)
				return nil
			}

			consumer := &MultiQueueConsumer{UseCase: &services.UseCase{}, logger: testBootstrapLogger(), messageVerifier: signer, allowLegacyBodySignatureFallback: tt.allowLegacy}
			ctx := contextWithBootstrapTracking(t)
			if tt.ctxTenant != "" {
				ctx = tmcore.ContextWithTenantID(ctx, tt.ctxTenant)
			}
			err := consumer.handlerGenerateReportDelivery(ctx, amqp.Delivery{
				Exchange:   "worker.exchange",
				RoutingKey: "worker.key",
				Body:       body,
				Headers:    tt.headers,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := handled.Load(); got != tt.wantHandled {
				t.Fatalf("handled = %v, want %v", got, tt.wantHandled)
			}
		})
	}
}

func TestHandlerGenerateReportDelivery_ExtractsQueueTraceBeforeProcessing(t *testing.T) {
	originalExtractExternalData := extractExternalData
	originalTracerProvider := otel.GetTracerProvider()
	originalPropagator := otel.GetTextMapPropagator()
	tp := tracesdk.NewTracerProvider()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		extractExternalData = originalExtractExternalData
		otel.SetTracerProvider(originalTracerProvider)
		otel.SetTextMapPropagator(originalPropagator)
		if err := tp.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown tracer provider: %v", err)
		}
	})

	parentTraceID := "00112233445566778899aabbccddeeff"
	var gotTraceID string

	extractExternalData = func(_ *services.UseCase, gotCtx context.Context, _ []byte, _ map[string]any) error {
		spanCtx := trace.SpanContextFromContext(gotCtx)
		if spanCtx.IsValid() {
			gotTraceID = spanCtx.TraceID().String()
		}

		return nil
	}

	consumer := &MultiQueueConsumer{
		UseCase: &services.UseCase{},
		logger:  testBootstrapLogger(),
	}
	ctx := tmcore.ContextWithTenantID(contextWithBootstrapTracking(t), "tenant-a")

	err := consumer.handlerGenerateReportDelivery(ctx, amqp.Delivery{
		Exchange:   "worker.exchange",
		RoutingKey: "worker.key",
		Body:       []byte(`{"jobId":"123e4567-e89b-12d3-a456-426614174000"}`),
		Headers: amqp.Table{
			pkgRabbitMQ.HeaderTenantID: "tenant-a",
			"traceparent":              "00-" + parentTraceID + "-0123456789abcdef-01",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTraceID != parentTraceID {
		t.Fatalf("trace ID = %q, want %q", gotTraceID, parentTraceID)
	}
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

	t.Run("invokes launcher with consumer and logger", func(t *testing.T) {
		consumer := &MultiQueueConsumer{}
		logger := testBootstrapLogger()

		called := false
		runLauncher = func(gotLogger libLog.Logger, gotConsumer *MultiQueueConsumer, _ *HealthServer, _ *libOutbox.Dispatcher, _ *services.TerminalEventRepairer, _ *services.TenantConsumerReconciler) {
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
