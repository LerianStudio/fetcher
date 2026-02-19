package bootstrap

import (
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"

	cacheRepo "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
)

func TestConfig_StructFields(t *testing.T) {
	cfg := &Config{
		EnvName:       "test",
		ServerAddress: "localhost:8080",
		LogLevel:      "info",
	}

	if cfg.EnvName != "test" {
		t.Errorf("Expected EnvName to be 'test', got '%s'", cfg.EnvName)
	}

	if cfg.ServerAddress != "localhost:8080" {
		t.Errorf("Expected ServerAddress to be 'localhost:8080', got '%s'", cfg.ServerAddress)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be 'info', got '%s'", cfg.LogLevel)
	}
}

func TestConfig_LoadFromEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "loads service configuration",
			envVars: map[string]string{
				"ENV_NAME":       "staging",
				"SERVER_ADDRESS": "0.0.0.0:3000",
				"LOG_LEVEL":      "warn",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.EnvName != "staging" {
					t.Errorf("EnvName = %q, want %q", cfg.EnvName, "staging")
				}
				if cfg.ServerAddress != "0.0.0.0:3000" {
					t.Errorf("ServerAddress = %q, want %q", cfg.ServerAddress, "0.0.0.0:3000")
				}
				if cfg.LogLevel != "warn" {
					t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
				}
			},
		},
		{
			name: "loads MongoDB configuration",
			envVars: map[string]string{
				"MONGO_URI":      "mongodb",
				"MONGO_HOST":     "mongo-host",
				"MONGO_NAME":     "fetcher",
				"MONGO_USER":     "root",
				"MONGO_PASSWORD": "pass123",
				"MONGO_PORT":     "27017",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.MongoURI != "mongodb" {
					t.Errorf("MongoURI = %q, want %q", cfg.MongoURI, "mongodb")
				}
				if cfg.MongoDBHost != "mongo-host" {
					t.Errorf("MongoDBHost = %q, want %q", cfg.MongoDBHost, "mongo-host")
				}
				if cfg.MongoDBName != "fetcher" {
					t.Errorf("MongoDBName = %q, want %q", cfg.MongoDBName, "fetcher")
				}
				if cfg.MongoDBPassword != "pass123" {
					t.Errorf("MongoDBPassword = %q, want %q", cfg.MongoDBPassword, "pass123")
				}
			},
		},
		{
			name: "loads RabbitMQ configuration",
			envVars: map[string]string{
				"RABBITMQ_URI":                "amqp",
				"RABBITMQ_HOST":               "rabbit-host",
				"RABBITMQ_PORT_AMQP":          "5672",
				"RABBITMQ_DEFAULT_USER":       "rmq-user",
				"RABBITMQ_DEFAULT_PASS":       "rmq-pass",
				"RABBITMQ_FETCHER_WORK_QUEUE": "fetcher-queue",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.RabbitURI != "amqp" {
					t.Errorf("RabbitURI = %q, want %q", cfg.RabbitURI, "amqp")
				}
				if cfg.RabbitMQHost != "rabbit-host" {
					t.Errorf("RabbitMQHost = %q, want %q", cfg.RabbitMQHost, "rabbit-host")
				}
				if cfg.RabbitMQGenerateReportQueue != "fetcher-queue" {
					t.Errorf("RabbitMQGenerateReportQueue = %q, want %q", cfg.RabbitMQGenerateReportQueue, "fetcher-queue")
				}
			},
		},
		{
			name: "loads boolean fields",
			envVars: map[string]string{
				"ENABLE_TELEMETRY":    "true",
				"PLUGIN_AUTH_ENABLED": "true",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.EnableTelemetry {
					t.Error("EnableTelemetry should be true")
				}
				if !cfg.AuthEnabled {
					t.Error("AuthEnabled should be true")
				}
			},
		},
		{
			name: "loads Redis configuration",
			envVars: map[string]string{
				"REDIS_HOST":     "redis-host",
				"REDIS_PORT":     "6379",
				"REDIS_PASSWORD": "redis-pass",
				"REDIS_DB":       "2",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.RedisHost != "redis-host" {
					t.Errorf("RedisHost = %q, want %q", cfg.RedisHost, "redis-host")
				}
				if cfg.RedisPort != "6379" {
					t.Errorf("RedisPort = %q, want %q", cfg.RedisPort, "6379")
				}
				if cfg.RedisPassword != "redis-pass" {
					t.Errorf("RedisPassword = %q, want %q", cfg.RedisPassword, "redis-pass")
				}
				if cfg.RedisDB != "2" {
					t.Errorf("RedisDB = %q, want %q", cfg.RedisDB, "2")
				}
			},
		},
		{
			name:    "empty env vars produce zero values",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.EnvName != "" {
					t.Errorf("EnvName should be empty, got %q", cfg.EnvName)
				}
				if cfg.ServerAddress != "" {
					t.Errorf("ServerAddress should be empty, got %q", cfg.ServerAddress)
				}
				if cfg.EnableTelemetry {
					t.Error("EnableTelemetry should default to false")
				}
				if cfg.AuthEnabled {
					t.Error("AuthEnabled should default to false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg := &Config{}
			if err := pkg.SetConfigFromEnvVars(cfg); err != nil {
				t.Fatalf("SetConfigFromEnvVars() error: %v", err)
			}

			tt.validate(t, cfg)
		})
	}
}

func TestGetSchemaCacheTTL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
	}{
		{
			name:  "valid seconds",
			input: "300",
			want:  300 * time.Second,
		},
		{
			name:  "one second",
			input: "1",
			want:  1 * time.Second,
		},
		{
			name:  "empty string returns default",
			input: "",
			want:  cacheRepo.DefaultSchemaCacheTTL,
		},
		{
			name:  "invalid string returns default",
			input: "not-a-number",
			want:  cacheRepo.DefaultSchemaCacheTTL,
		},
		{
			name:  "zero seconds",
			input: "0",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSchemaCacheTTL(tt.input)
			if got != tt.want {
				t.Errorf("getSchemaCacheTTL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetRedisDB(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "valid db number",
			input: "3",
			want:  3,
		},
		{
			name:  "zero",
			input: "0",
			want:  0,
		},
		{
			name:  "empty string returns 0",
			input: "",
			want:  0,
		},
		{
			name:  "invalid string returns 0",
			input: "abc",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRedisDB(tt.input)
			if got != tt.want {
				t.Errorf("getRedisDB(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
