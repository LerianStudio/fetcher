package bootstrap

import (
	"testing"
)

func TestConfigStruct(t *testing.T) {
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

func TestConfigDefaults(t *testing.T) {
	cfg := &Config{}

	if cfg.RabbitMQNumWorkers != 0 {
		t.Errorf("Expected RabbitMQNumWorkers default to be 0, got %d", cfg.RabbitMQNumWorkers)
	}

	if cfg.MaxPoolSize != 0 {
		t.Errorf("Expected MaxPoolSize default to be 0, got %d", cfg.MaxPoolSize)
	}
}
