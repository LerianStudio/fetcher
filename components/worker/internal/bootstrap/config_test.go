package bootstrap

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libZap "github.com/LerianStudio/lib-commons/v4/commons/zap"
)

func testBootstrapLogger() *libLog.GoLogger {
	return &libLog.GoLogger{Level: libLog.LevelError}
}

func TestResolveZapEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  libZap.Environment
	}{
		{name: "production aliases", input: " PROD ", want: libZap.EnvironmentProduction},
		{name: "staging alias", input: "stage", want: libZap.EnvironmentStaging},
		{name: "uat alias", input: "UAT", want: libZap.EnvironmentUAT},
		{name: "development alias", input: "development", want: libZap.EnvironmentDevelopment},
		{name: "local alias", input: "local", want: libZap.EnvironmentLocal},
		{name: "unknown defaults to local", input: "qa", want: libZap.EnvironmentLocal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := resolveZapEnvironment(tt.input); got != tt.want {
				t.Fatalf("resolveZapEnvironment(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMust(t *testing.T) {
	t.Parallel()

	t.Run("does not panic on nil error", func(t *testing.T) {
		t.Parallel()
		must("noop", nil)
	})

	t.Run("wraps and panics on error", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("boom")

		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic, got nil")
			}

			panicErr, ok := r.(error)
			if !ok {
				t.Fatalf("expected error panic, got %T", r)
			}

			if !errors.Is(panicErr, wantErr) {
				t.Fatalf("expected panic to wrap %v, got %v", wantErr, panicErr)
			}

			if got := panicErr.Error(); got != "decode key: boom" {
				t.Fatalf("unexpected panic message: %s", got)
			}
		}()

		must("decode key", wantErr)
	})
}

func TestInitWorker_PanicsWhenConfigLoadFails(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
	})

	setConfigFromEnvVars = func(any) error {
		return errors.New("config load failed")
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got nil")
		}

		if !strings.Contains(fmt.Sprint(r), "config load failed") {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	InitWorker()
}

func TestInitWorker_PanicsWhenLoggerInitFails(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "debug"
		return nil
	}

	newZapLogger = func(libZap.Config) (libLog.Logger, error) {
		return nil, errors.New("logger init failed")
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got nil")
		}

		if !strings.Contains(fmt.Sprint(r), "logger init failed") {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	InitWorker()
}

func TestInitWorker_PanicsWhenTelemetryGlobalsFail(t *testing.T) {
	originalSetConfigFromEnvVars := setConfigFromEnvVars
	originalNewZapLogger := newZapLogger
	originalNewTelemetry := newTelemetry
	originalApplyTelemetryGlobals := applyTelemetryGlobals
	t.Cleanup(func() {
		setConfigFromEnvVars = originalSetConfigFromEnvVars
		newZapLogger = originalNewZapLogger
		newTelemetry = originalNewTelemetry
		applyTelemetryGlobals = originalApplyTelemetryGlobals
	})

	setConfigFromEnvVars = func(target any) error {
		cfg := target.(*Config)
		cfg.EnvName = "local"
		cfg.LogLevel = "debug"
		return nil
	}

	newZapLogger = func(libZap.Config) (libLog.Logger, error) {
		return testBootstrapLogger(), nil
	}

	newTelemetry = func(libOtel.TelemetryConfig) (*libOtel.Telemetry, error) {
		return &libOtel.Telemetry{}, nil
	}

	applyTelemetryGlobals = func(*libOtel.Telemetry) error {
		return errors.New("apply globals failed")
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got nil")
		}

		if !strings.Contains(fmt.Sprint(r), "apply globals failed") {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	InitWorker()
}
