package bootstrap

import (
	"testing"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
)

func TestConfig_StructFields(t *testing.T) {
	cfg := &Config{
		EnvName:  "test",
		LogLevel: "info",
	}

	if cfg.EnvName != "test" {
		t.Errorf("Expected EnvName to be 'test', got '%s'", cfg.EnvName)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be 'info', got '%s'", cfg.LogLevel)
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := &Config{}

	if cfg.RabbitMQNumWorkers != 0 {
		t.Errorf("Expected RabbitMQNumWorkers default to be 0, got %d", cfg.RabbitMQNumWorkers)
	}

	if cfg.MaxPoolSize != 0 {
		t.Errorf("Expected MaxPoolSize default to be 0, got %d", cfg.MaxPoolSize)
	}
}

func TestConfig_LoadFromEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "loads string fields from env vars",
			envVars: map[string]string{
				"ENV_NAME":  "production",
				"LOG_LEVEL": "debug",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.EnvName != "production" {
					t.Errorf("EnvName = %q, want %q", cfg.EnvName, "production")
				}
				if cfg.LogLevel != "debug" {
					t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
				}
			},
		},
		{
			name: "loads RabbitMQ configuration",
			envVars: map[string]string{
				"RABBITMQ_URI":                "amqp",
				"RABBITMQ_HOST":               "rabbit-host",
				"RABBITMQ_PORT_AMQP":          "5672",
				"RABBITMQ_DEFAULT_USER":       "guest",
				"RABBITMQ_DEFAULT_PASS":       "guest",
				"RABBITMQ_FETCHER_WORK_QUEUE": "work-queue",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.RabbitURI != "amqp" {
					t.Errorf("RabbitURI = %q, want %q", cfg.RabbitURI, "amqp")
				}
				if cfg.RabbitMQHost != "rabbit-host" {
					t.Errorf("RabbitMQHost = %q, want %q", cfg.RabbitMQHost, "rabbit-host")
				}
				if cfg.RabbitMQUser != "guest" {
					t.Errorf("RabbitMQUser = %q, want %q", cfg.RabbitMQUser, "guest")
				}
				if cfg.RabbitMQGenerateReportQueue != "work-queue" {
					t.Errorf("RabbitMQGenerateReportQueue = %q, want %q", cfg.RabbitMQGenerateReportQueue, "work-queue")
				}
			},
		},
		{
			name: "loads MongoDB configuration",
			envVars: map[string]string{
				"MONGO_URI":      "mongodb",
				"MONGO_HOST":     "mongo-host",
				"MONGO_NAME":     "fetcher-db",
				"MONGO_USER":     "admin",
				"MONGO_PASSWORD": "secret",
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
				if cfg.MongoDBName != "fetcher-db" {
					t.Errorf("MongoDBName = %q, want %q", cfg.MongoDBName, "fetcher-db")
				}
			},
		},
		{
			name: "loads boolean and int fields",
			envVars: map[string]string{
				"ENABLE_TELEMETRY":            "true",
				"RABBITMQ_NUMBERS_OF_WORKERS": "5",
				"MONGO_MAX_POOL_SIZE":         "50",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.EnableTelemetry {
					t.Error("EnableTelemetry should be true")
				}
				if cfg.RabbitMQNumWorkers != 5 {
					t.Errorf("RabbitMQNumWorkers = %d, want 5", cfg.RabbitMQNumWorkers)
				}
				if cfg.MaxPoolSize != 50 {
					t.Errorf("MaxPoolSize = %d, want 50", cfg.MaxPoolSize)
				}
			},
		},
		{
			name: "loads encryption keys",
			envVars: map[string]string{
				"APP_ENC_KEY":                          "master-key-123",
				"APP_ENC_KEY_VERSION":                  "v1",
				"CRYPTO_ENCRYPT_FILE_STORAGE":          "seaweed-encrypt",
				"CRYPTO_HASH_SECRET_KEY_FILE_STORAGE":  "seaweed-hash",
				"CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM": "crm-encrypt",
				"CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM":    "crm-hash",
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.AppEncryptionKey != "master-key-123" {
					t.Errorf("AppEncryptionKey = %q, want %q", cfg.AppEncryptionKey, "master-key-123")
				}
				if cfg.CryptoEncryptFileStorage != "seaweed-encrypt" {
					t.Errorf("CryptoEncryptFileStorage = %q, want %q", cfg.CryptoEncryptFileStorage, "seaweed-encrypt")
				}
				if cfg.CryptoHashFileStorage != "crm-hash" {
					t.Errorf("CryptoHashFileStorage = %q, want %q", cfg.CryptoHashFileStorage, "crm-hash")
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
				if cfg.EnableTelemetry {
					t.Error("EnableTelemetry should default to false")
				}
				if cfg.RabbitMQNumWorkers != 0 {
					t.Errorf("RabbitMQNumWorkers should be 0, got %d", cfg.RabbitMQNumWorkers)
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
			if err := libCommons.SetConfigFromEnvVars(cfg); err != nil {
				t.Fatalf("SetConfigFromEnvVars() error: %v", err)
			}

			tt.validate(t, cfg)
		})
	}
}
