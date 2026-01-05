package bootstrap

import (
	"testing"
)

func TestConfigStruct(t *testing.T) {
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
