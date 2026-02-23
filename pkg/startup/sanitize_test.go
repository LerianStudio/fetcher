package startup

import (
	"fmt"
	"testing"
)

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "redacts mongodb uri with credentials",
			err:      fmt.Errorf("failed to connect: mongodb://admin:secret@localhost:27017/db"),
			expected: "failed to connect: [redacted-uri]",
		},
		{
			name:     "redacts amqp uri with credentials",
			err:      fmt.Errorf("dial: amqp://guest:guest@rabbitmq:5672/"),
			expected: "dial: [redacted-uri]",
		},
		{
			name:     "redacts multiple uris",
			err:      fmt.Errorf("sources: mongodb://u:p@h1 and amqp://u:p@h2"),
			expected: "sources: [redacted-uri] and [redacted-uri]",
		},
		{
			name:     "preserves message without uris",
			err:      fmt.Errorf("missing environment variable APP_KEY"),
			expected: "missing environment variable APP_KEY",
		},
		{
			name:     "redacts https uri",
			err:      fmt.Errorf("tls: https://token:secret@api.example.com/v1"),
			expected: "tls: [redacted-uri]",
		},
		{
			name:     "nil error returns empty string",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeError(tt.err)
			if got != tt.expected {
				t.Errorf("SanitizeError() = %q, want %q", got, tt.expected)
			}
		})
	}
}
