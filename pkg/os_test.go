package pkg

import (
	"os"
	"testing"
)

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		want         string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_ENV_VAR",
			defaultValue: "default",
			envValue:     "actual",
			setEnv:       true,
			want:         "actual",
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_ENV_VAR_NOT_SET",
			defaultValue: "default",
			envValue:     "",
			setEnv:       false,
			want:         "default",
		},
		{
			name:         "returns default when env is empty",
			key:          "TEST_ENV_VAR_EMPTY",
			defaultValue: "default",
			envValue:     "",
			setEnv:       true,
			want:         "default",
		},
		{
			name:         "returns default when env is whitespace only",
			key:          "TEST_ENV_VAR_WHITESPACE",
			defaultValue: "default",
			envValue:     "   ",
			setEnv:       true,
			want:         "default",
		},
		{
			name:         "preserves value with leading/trailing spaces",
			key:          "TEST_ENV_VAR_SPACES",
			defaultValue: "default",
			envValue:     "  value  ",
			setEnv:       true,
			want:         "  value  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnvOrDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvOrDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetenvBoolOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		setEnv       bool
		want         bool
	}{
		{
			name:         "returns true when env is 'true'",
			key:          "TEST_BOOL_TRUE",
			defaultValue: false,
			envValue:     "true",
			setEnv:       true,
			want:         true,
		},
		{
			name:         "returns true when env is '1'",
			key:          "TEST_BOOL_ONE",
			defaultValue: false,
			envValue:     "1",
			setEnv:       true,
			want:         true,
		},
		{
			name:         "returns false when env is 'false'",
			key:          "TEST_BOOL_FALSE",
			defaultValue: true,
			envValue:     "false",
			setEnv:       true,
			want:         false,
		},
		{
			name:         "returns false when env is '0'",
			key:          "TEST_BOOL_ZERO",
			defaultValue: true,
			envValue:     "0",
			setEnv:       true,
			want:         false,
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_BOOL_NOT_SET",
			defaultValue: true,
			envValue:     "",
			setEnv:       false,
			want:         true,
		},
		{
			name:         "returns default when env is invalid",
			key:          "TEST_BOOL_INVALID",
			defaultValue: true,
			envValue:     "invalid",
			setEnv:       true,
			want:         true,
		},
		{
			name:         "returns default when env is empty",
			key:          "TEST_BOOL_EMPTY",
			defaultValue: false,
			envValue:     "",
			setEnv:       true,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := GetenvBoolOrDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetenvBoolOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetenvIntOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int64
		envValue     string
		setEnv       bool
		want         int64
	}{
		{
			name:         "returns int value when valid",
			key:          "TEST_INT_VALID",
			defaultValue: 0,
			envValue:     "42",
			setEnv:       true,
			want:         42,
		},
		{
			name:         "returns negative int value",
			key:          "TEST_INT_NEGATIVE",
			defaultValue: 0,
			envValue:     "-100",
			setEnv:       true,
			want:         -100,
		},
		{
			name:         "returns large int64 value",
			key:          "TEST_INT_LARGE",
			defaultValue: 0,
			envValue:     "9223372036854775807",
			setEnv:       true,
			want:         9223372036854775807,
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_INT_NOT_SET",
			defaultValue: 99,
			envValue:     "",
			setEnv:       false,
			want:         99,
		},
		{
			name:         "returns default when env is invalid",
			key:          "TEST_INT_INVALID",
			defaultValue: 50,
			envValue:     "not_a_number",
			setEnv:       true,
			want:         50,
		},
		{
			name:         "returns default when env is empty",
			key:          "TEST_INT_EMPTY",
			defaultValue: 25,
			envValue:     "",
			setEnv:       true,
			want:         25,
		},
		{
			name:         "returns default when env is float",
			key:          "TEST_INT_FLOAT",
			defaultValue: 10,
			envValue:     "3.14",
			setEnv:       true,
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := GetenvIntOrDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetenvIntOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetConfigFromEnvVars(t *testing.T) {
	type TestConfig struct {
		StringField string `env:"TEST_STRING"`
		BoolField   bool   `env:"TEST_BOOL"`
		IntField    int    `env:"TEST_INT"`
		Int8Field   int8   `env:"TEST_INT8"`
		Int16Field  int16  `env:"TEST_INT16"`
		Int32Field  int32  `env:"TEST_INT32"`
		Int64Field  int64  `env:"TEST_INT64"`
		NoTagField  string
	}

	t.Run("populates struct from env vars", func(t *testing.T) {
		// Set env vars
		os.Setenv("TEST_STRING", "hello")
		os.Setenv("TEST_BOOL", "true")
		os.Setenv("TEST_INT", "42")
		os.Setenv("TEST_INT8", "8")
		os.Setenv("TEST_INT16", "16")
		os.Setenv("TEST_INT32", "32")
		os.Setenv("TEST_INT64", "64")
		defer func() {
			os.Unsetenv("TEST_STRING")
			os.Unsetenv("TEST_BOOL")
			os.Unsetenv("TEST_INT")
			os.Unsetenv("TEST_INT8")
			os.Unsetenv("TEST_INT16")
			os.Unsetenv("TEST_INT32")
			os.Unsetenv("TEST_INT64")
		}()

		cfg := &TestConfig{}
		err := SetConfigFromEnvVars(cfg)

		if err != nil {
			t.Fatalf("SetConfigFromEnvVars() error = %v", err)
		}

		if cfg.StringField != "hello" {
			t.Errorf("StringField = %q, want %q", cfg.StringField, "hello")
		}
		if cfg.BoolField != true {
			t.Errorf("BoolField = %v, want %v", cfg.BoolField, true)
		}
		if cfg.IntField != 42 {
			t.Errorf("IntField = %d, want %d", cfg.IntField, 42)
		}
		if cfg.Int64Field != 64 {
			t.Errorf("Int64Field = %d, want %d", cfg.Int64Field, 64)
		}
	})

	t.Run("returns error for non-pointer", func(t *testing.T) {
		cfg := TestConfig{}
		err := SetConfigFromEnvVars(cfg)

		if err == nil {
			t.Error("SetConfigFromEnvVars() expected error for non-pointer, got nil")
		}
	})

	t.Run("handles missing env vars with defaults", func(t *testing.T) {
		// Ensure env vars are not set
		os.Unsetenv("TEST_STRING")
		os.Unsetenv("TEST_BOOL")
		os.Unsetenv("TEST_INT")

		cfg := &TestConfig{}
		err := SetConfigFromEnvVars(cfg)

		if err != nil {
			t.Fatalf("SetConfigFromEnvVars() error = %v", err)
		}

		// Should have zero values
		if cfg.StringField != "" {
			t.Errorf("StringField = %q, want empty string", cfg.StringField)
		}
		if cfg.BoolField != false {
			t.Errorf("BoolField = %v, want false", cfg.BoolField)
		}
		if cfg.IntField != 0 {
			t.Errorf("IntField = %d, want 0", cfg.IntField)
		}
	})
}

func TestEnsureConfigFromEnvVars(t *testing.T) {
	type TestConfig struct {
		Field string `env:"TEST_ENSURE_FIELD"`
	}

	t.Run("returns populated struct", func(t *testing.T) {
		os.Setenv("TEST_ENSURE_FIELD", "value")
		defer os.Unsetenv("TEST_ENSURE_FIELD")

		cfg := &TestConfig{}
		result, err := EnsureConfigFromEnvVars(cfg)

		if err != nil {
			t.Fatalf("EnsureConfigFromEnvVars() unexpected error: %v", err)
		}

		if result != cfg {
			t.Error("EnsureConfigFromEnvVars() should return same pointer")
		}
		if cfg.Field != "value" {
			t.Errorf("Field = %q, want %q", cfg.Field, "value")
		}
	})

	t.Run("returns error on non-pointer", func(t *testing.T) {
		cfg := TestConfig{}
		result, err := EnsureConfigFromEnvVars(cfg)

		if err == nil {
			t.Error("EnsureConfigFromEnvVars() expected error for non-pointer, got nil")
		}

		if result != nil {
			t.Errorf("EnsureConfigFromEnvVars() expected nil result on error, got %#v", result)
		}
	})
}
