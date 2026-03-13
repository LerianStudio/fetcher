package sslmode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSQLServerMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		wantError bool
	}{
		// Valid modes - from go-mssqldb documentation
		{name: "empty string is valid (driver default behavior)", mode: "", wantError: false},
		{name: "disable is valid", mode: "disable", wantError: false},
		{name: "false is valid", mode: "false", wantError: false},
		{name: "true is valid", mode: "true", wantError: false},
		{name: "strict is valid", mode: "strict", wantError: false},
		// Case variations should be rejected
		{name: "DISABLE uppercase is invalid", mode: "DISABLE", wantError: true},
		{name: "TRUE uppercase is invalid", mode: "TRUE", wantError: true},
		{name: "True mixed case is invalid", mode: "True", wantError: true},
		{name: "STRICT uppercase is invalid", mode: "STRICT", wantError: true},
		{name: "FALSE uppercase is invalid", mode: "FALSE", wantError: true},
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
		{name: "require is invalid for SQL Server (PostgreSQL mode)", mode: "require", wantError: true},
		{name: "verify-ca is invalid for SQL Server (PostgreSQL mode)", mode: "verify-ca", wantError: true},
		{name: "verify-full is invalid for SQL Server (PostgreSQL mode)", mode: "verify-full", wantError: true},
		{name: "skip-verify is invalid for SQL Server (MySQL mode)", mode: "skip-verify", wantError: true},
		{name: "preferred is invalid for SQL Server (MySQL mode)", mode: "preferred", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSQLServerMode(tt.mode)
			if tt.wantError {
				require.Error(t, err, "Expected error for mode: %q", tt.mode)
				assert.Contains(t, err.Error(), "FET-0413", "Error should use ErrInvalidSSLMode code")
			} else {
				assert.NoError(t, err, "Expected no error for mode: %q", tt.mode)
			}
		})
	}
}

func TestGetValidSQLServerModes(t *testing.T) {
	modes := GetValidSQLServerModes()

	// Verify all expected modes are present
	expected := []string{"", "disable", "false", "true", "strict"}
	assert.ElementsMatch(t, expected, modes, "Should return all valid SQL Server encryption modes")
}
