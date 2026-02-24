package sslmode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePostgreSQLMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		wantError bool
	}{
		// Valid modes - from PostgreSQL documentation
		{name: "empty string is invalid (must use explicit mode)", mode: "", wantError: true},
		{name: "disable is valid", mode: "disable", wantError: false},
		{name: "allow is valid", mode: "allow", wantError: false},
		{name: "prefer is valid", mode: "prefer", wantError: false},
		{name: "require is valid", mode: "require", wantError: false},
		{name: "verify-ca is valid", mode: "verify-ca", wantError: false},
		{name: "verify-full is valid", mode: "verify-full", wantError: false},
		// Case variations should be rejected (driver is case-sensitive)
		{name: "DISABLE uppercase is invalid", mode: "DISABLE", wantError: true},
		{name: "REQUIRE uppercase is invalid", mode: "REQUIRE", wantError: true},
		{name: "Require mixed case is invalid", mode: "Require", wantError: true},
		{name: "VERIFY-CA uppercase is invalid", mode: "VERIFY-CA", wantError: true},
		{name: "VERIFY-FULL uppercase is invalid", mode: "VERIFY-FULL", wantError: true},
		// Injection attempts
		{name: "injection with semicolon", mode: "disable;DROP TABLE users", wantError: true},
		{name: "injection with ampersand", mode: "disable&password=hacked", wantError: true},
		{name: "injection with space", mode: "disable DROP TABLE", wantError: true},
		{name: "injection with URL encoding", mode: "disable%26password=hacked", wantError: true},
		{name: "path traversal attempt", mode: "../../../etc/passwd", wantError: true},
		{name: "null byte injection", mode: "disable\x00malicious", wantError: true},
		{name: "newline injection", mode: "disable\nmalicious", wantError: true},
		// Invalid modes
		{name: "random string is invalid", mode: "random", wantError: true},
		{name: "true is invalid for PostgreSQL (MySQL mode)", mode: "true", wantError: true},
		{name: "false is invalid for PostgreSQL (MySQL mode)", mode: "false", wantError: true},
		{name: "skip-verify is invalid for PostgreSQL (MySQL mode)", mode: "skip-verify", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePostgreSQLMode(tt.mode)
			if tt.wantError {
				require.Error(t, err, "Expected error for mode: %q", tt.mode)
				assert.Contains(t, err.Error(), "FET-0413", "Error should use ErrInvalidSSLMode code")
			} else {
				assert.NoError(t, err, "Expected no error for mode: %q", tt.mode)
			}
		})
	}
}

func TestGetValidPostgreSQLModes(t *testing.T) {
	modes := GetValidPostgreSQLModes()

	// Verify all expected modes are present
	expected := []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}
	assert.ElementsMatch(t, expected, modes, "Should return all valid PostgreSQL SSL modes")
}
