package sslmode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMySQLMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		wantError bool
	}{
		// Valid modes - from go-sql-driver/mysql documentation
		{name: "empty string is valid (driver default behavior)", mode: "", wantError: false},
		{name: "false is valid", mode: "false", wantError: false},
		{name: "true is valid", mode: "true", wantError: false},
		{name: "skip-verify is valid", mode: "skip-verify", wantError: false},
		{name: "preferred is valid", mode: "preferred", wantError: false},
		// Case variations should be rejected (driver is case-sensitive)
		{name: "FALSE uppercase is invalid", mode: "FALSE", wantError: true},
		{name: "TRUE uppercase is invalid", mode: "TRUE", wantError: true},
		{name: "True mixed case is invalid", mode: "True", wantError: true},
		{name: "SKIP-VERIFY uppercase is invalid", mode: "SKIP-VERIFY", wantError: true},
		{name: "PREFERRED uppercase is invalid", mode: "PREFERRED", wantError: true},
		// Injection attempts
		{name: "injection with semicolon", mode: "false;DROP TABLE users", wantError: true},
		{name: "injection with ampersand", mode: "false&password=hacked", wantError: true},
		{name: "injection with space", mode: "false DROP TABLE", wantError: true},
		{name: "injection with URL encoding", mode: "false%26password=hacked", wantError: true},
		{name: "path traversal attempt", mode: "../../../etc/passwd", wantError: true},
		{name: "null byte injection", mode: "false\x00malicious", wantError: true},
		{name: "newline injection", mode: "false\nmalicious", wantError: true},
		// Invalid modes
		{name: "random string is invalid", mode: "random", wantError: true},
		{name: "require is invalid for MySQL", mode: "require", wantError: true},
		{name: "disable is invalid for MySQL", mode: "disable", wantError: true},
		{name: "verify-ca is invalid for MySQL (PostgreSQL mode)", mode: "verify-ca", wantError: true},
		{name: "verify-full is invalid for MySQL (PostgreSQL mode)", mode: "verify-full", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMySQLMode(tt.mode)
			if tt.wantError {
				require.Error(t, err, "Expected error for mode: %q", tt.mode)
				assert.Contains(t, err.Error(), "FET-0413", "Error should use ErrInvalidSSLMode code")
			} else {
				assert.NoError(t, err, "Expected no error for mode: %q", tt.mode)
			}
		})
	}
}

func TestGetValidMySQLModes(t *testing.T) {
	modes := GetValidMySQLModes()

	// Verify all expected modes are present
	expected := []string{"", "false", "true", "skip-verify", "preferred"}
	assert.ElementsMatch(t, expected, modes, "Should return all valid MySQL SSL modes")
}
