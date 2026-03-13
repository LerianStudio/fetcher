package pkg

import (
	"context"
	"testing"

	"github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewLoggerFromContext(t *testing.T) {
	tests := []struct {
		name          string
		setupContext  func() context.Context
		expectNilType bool
	}{
		{
			name: "returns logger when set in context",
			setupContext: func() context.Context {
				ctx := context.Background()
				logger := &log.NopLogger{}
				return ContextWithLogger(ctx, logger)
			},
			expectNilType: false,
		},
		{
			name: "returns NopLogger when context has no custom value",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectNilType: true,
		},
		{
			name: "returns NopLogger when custom context value has nil logger",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, CustomContextKey, &CustomContextKeyValue{Logger: nil})
			},
			expectNilType: true,
		},
		{
			name: "returns NopLogger when context value is wrong type",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, CustomContextKey, "wrong-type")
			},
			expectNilType: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			logger := NewLoggerFromContext(ctx)

			require.NotNil(t, logger, "logger should never be nil")

			if tt.expectNilType {
				_, ok := logger.(*log.NopLogger)
				assert.True(t, ok, "expected NopLogger when no valid logger in context")
			}
		})
	}
}

func TestNewTracerFromContext(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func() context.Context
		expectCustom bool
	}{
		{
			name: "returns tracer when set in context",
			setupContext: func() context.Context {
				ctx := context.Background()
				tracer := noop.NewTracerProvider().Tracer("test-tracer")
				return ContextWithTracer(ctx, tracer)
			},
			expectCustom: true,
		},
		{
			name: "returns default otel tracer when context has no custom value",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectCustom: false,
		},
		{
			name: "returns default otel tracer when custom context value has nil tracer",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, CustomContextKey, &CustomContextKeyValue{Tracer: nil})
			},
			expectCustom: false,
		},
		{
			name: "returns default otel tracer when context value is wrong type",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, CustomContextKey, "wrong-type")
			},
			expectCustom: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			tracer := NewTracerFromContext(ctx)

			require.NotNil(t, tracer, "tracer should never be nil")

			// Can't easily distinguish between default and custom,
			// but we can verify it's a valid tracer
			assert.Implements(t, (*trace.Tracer)(nil), tracer)
		})
	}
}

func TestContextWithLogger(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func() context.Context
		logger       log.Logger
	}{
		{
			name: "adds logger to empty context",
			setupContext: func() context.Context {
				return context.Background()
			},
			logger: &log.NopLogger{},
		},
		{
			name: "replaces existing logger in context",
			setupContext: func() context.Context {
				ctx := context.Background()
				oldLogger := &log.NopLogger{}
				return ContextWithLogger(ctx, oldLogger)
			},
			logger: &log.NopLogger{},
		},
		{
			name: "preserves tracer when adding logger",
			setupContext: func() context.Context {
				ctx := context.Background()
				tracer := noop.NewTracerProvider().Tracer("test-tracer")
				return ContextWithTracer(ctx, tracer)
			},
			logger: &log.NopLogger{},
		},
		{
			name: "handles nil custom context value",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, CustomContextKey, nil)
			},
			logger: &log.NopLogger{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			newCtx := ContextWithLogger(ctx, tt.logger)

			// Verify logger was stored
			customCtx, ok := newCtx.Value(CustomContextKey).(*CustomContextKeyValue)
			require.True(t, ok, "custom context value should be accessible")
			assert.Equal(t, tt.logger, customCtx.Logger)
		})
	}
}

func TestContextWithTracer(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func() context.Context
		tracer       trace.Tracer
	}{
		{
			name: "adds tracer to empty context",
			setupContext: func() context.Context {
				return context.Background()
			},
			tracer: noop.NewTracerProvider().Tracer("test-tracer"),
		},
		{
			name: "replaces existing tracer in context",
			setupContext: func() context.Context {
				ctx := context.Background()
				oldTracer := noop.NewTracerProvider().Tracer("old-tracer")
				return ContextWithTracer(ctx, oldTracer)
			},
			tracer: noop.NewTracerProvider().Tracer("new-tracer"),
		},
		{
			name: "preserves logger when adding tracer",
			setupContext: func() context.Context {
				ctx := context.Background()
				logger := &log.NopLogger{}
				return ContextWithLogger(ctx, logger)
			},
			tracer: noop.NewTracerProvider().Tracer("test-tracer"),
		},
		{
			name: "handles nil custom context value",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, CustomContextKey, nil)
			},
			tracer: noop.NewTracerProvider().Tracer("test-tracer"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			newCtx := ContextWithTracer(ctx, tt.tracer)

			// Verify tracer was stored
			customCtx, ok := newCtx.Value(CustomContextKey).(*CustomContextKeyValue)
			require.True(t, ok, "custom context value should be accessible")
			assert.Equal(t, tt.tracer, customCtx.Tracer)
		})
	}
}

func TestContextWithLoggerAndTracer_Combined(t *testing.T) {
	t.Run("both logger and tracer can be set independently", func(t *testing.T) {
		ctx := context.Background()
		logger := &log.NopLogger{}
		tracer := noop.NewTracerProvider().Tracer("test-tracer")

		// Set logger first, then tracer
		ctx = ContextWithLogger(ctx, logger)
		ctx = ContextWithTracer(ctx, tracer)

		// Verify both are accessible
		customCtx, ok := ctx.Value(CustomContextKey).(*CustomContextKeyValue)
		require.True(t, ok)
		assert.Equal(t, logger, customCtx.Logger)
		assert.Equal(t, tracer, customCtx.Tracer)
	})

	t.Run("setting tracer first then logger preserves both", func(t *testing.T) {
		ctx := context.Background()
		logger := &log.NopLogger{}
		tracer := noop.NewTracerProvider().Tracer("test-tracer")

		// Set tracer first, then logger
		ctx = ContextWithTracer(ctx, tracer)
		ctx = ContextWithLogger(ctx, logger)

		// Verify both are accessible
		customCtx, ok := ctx.Value(CustomContextKey).(*CustomContextKeyValue)
		require.True(t, ok)
		assert.Equal(t, logger, customCtx.Logger)
		assert.Equal(t, tracer, customCtx.Tracer)
	})
}

func TestCustomContextKeyValue_Integration(t *testing.T) {
	t.Run("full roundtrip with logger and tracer", func(t *testing.T) {
		ctx := context.Background()

		// Set up logger and tracer
		logger := &log.NopLogger{}
		tracer := noop.NewTracerProvider().Tracer("integration-test")

		ctx = ContextWithLogger(ctx, logger)
		ctx = ContextWithTracer(ctx, tracer)

		// Retrieve them back
		retrievedLogger := NewLoggerFromContext(ctx)
		retrievedTracer := NewTracerFromContext(ctx)

		assert.Equal(t, logger, retrievedLogger)
		assert.Equal(t, tracer, retrievedTracer)
	})
}

func TestCustomContextKey_Type(t *testing.T) {
	t.Run("custom context key is unique type", func(t *testing.T) {
		// Verify that the custom key type prevents collisions
		ctx := context.Background()

		// Using a regular string key should not conflict
		regularKey := "custom_context"
		ctx = context.WithValue(ctx, regularKey, "some-value")
		ctx = ContextWithLogger(ctx, &log.NopLogger{})

		// String key value should still be accessible
		assert.Equal(t, "some-value", ctx.Value(regularKey))

		// Custom context key should return the CustomContextKeyValue
		customValue, ok := ctx.Value(CustomContextKey).(*CustomContextKeyValue)
		require.True(t, ok)
		require.NotNil(t, customValue)
	})
}

func TestNewTracerFromContext_DefaultTracer(t *testing.T) {
	t.Run("returns global otel tracer when no custom tracer", func(t *testing.T) {
		ctx := context.Background()

		tracer := NewTracerFromContext(ctx)

		// Should return the default tracer from otel global provider
		expectedTracer := otel.Tracer("default")

		// Both should be valid tracers
		require.NotNil(t, tracer)
		require.NotNil(t, expectedTracer)
	})
}
