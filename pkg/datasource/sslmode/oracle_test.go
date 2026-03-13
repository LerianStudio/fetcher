package sslmode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOracleMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		wantError bool
	}{
		// Valid modes - from go-ora documentation
		// The go-ora driver uses "ssl" and "ssl verify" URL parameters
		{name: "empty string is valid (driver default behavior)", mode: "", wantError: false},
		{name: "disable is valid", mode: "disable", wantError: false},
		{name: "false is valid", mode: "false", wantError: false},
		{name: "true is valid (SSL enabled)", mode: "true", wantError: false},
		{name: "enable is valid (SSL enabled)", mode: "enable", wantError: false},
		{name: "verify is valid (SSL with verification)", mode: "verify", wantError: false},
		{name: "skip-verify is valid (SSL without verification)", mode: "skip-verify", wantError: false},
		// Case variations should be rejected for consistency
		{name: "DISABLE uppercase is invalid", mode: "DISABLE", wantError: true},
		{name: "TRUE uppercase is invalid", mode: "TRUE", wantError: true},
		{name: "ENABLE uppercase is invalid", mode: "ENABLE", wantError: true},
		// Injection attempts
		{name: "injection with semicolon", mode: "true;DROP TABLE users", wantError: true},
		{name: "injection with ampersand", mode: "true&password=hacked", wantError: true},
		{name: "injection with space", mode: "true DROP TABLE", wantError: true},
		{name: "injection with equals", mode: "true=malicious", wantError: true},
		{name: "path traversal attempt", mode: "../../../etc/passwd", wantError: true},
		{name: "null byte injection", mode: "true\x00malicious", wantError: true},
		{name: "newline injection", mode: "true\nmalicious", wantError: true},
		// Invalid modes (PostgreSQL modes not valid for Oracle)
		{name: "require is invalid for Oracle (PostgreSQL mode)", mode: "require", wantError: true},
		{name: "verify-ca is invalid for Oracle (PostgreSQL mode)", mode: "verify-ca", wantError: true},
		{name: "verify-full is invalid for Oracle (PostgreSQL mode)", mode: "verify-full", wantError: true},
		{name: "preferred is invalid for Oracle (MySQL mode)", mode: "preferred", wantError: true},
		{name: "random string is invalid", mode: "random", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOracleMode(tt.mode)
			if tt.wantError {
				require.Error(t, err, "Expected error for mode: %q", tt.mode)
				assert.Contains(t, err.Error(), "FET-0413", "Error should use ErrInvalidSSLMode code")
			} else {
				assert.NoError(t, err, "Expected no error for mode: %q", tt.mode)
			}
		})
	}
}

func TestGetValidOracleModes(t *testing.T) {
	modes := GetValidOracleModes()

	// Verify all expected modes are present
	expected := []string{"", "disable", "false", "true", "enable", "verify", "skip-verify"}
	assert.ElementsMatch(t, expected, modes, "Should return all valid Oracle SSL modes")
}
