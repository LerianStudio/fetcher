package sslmode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInjectionAttackVectors tests both MySQL and Oracle validators against
// a comprehensive suite of injection attack vectors to ensure they properly
// reject malicious input.
func TestInjectionAttackVectors(t *testing.T) {
	// Common injection attack vectors that should be rejected by all validators
	attackVectors := []struct {
		name   string
		vector string
	}{
		// SQL Injection attempts
		{name: "SQL injection DROP TABLE", vector: "true; DROP TABLE users;--"},
		{name: "SQL injection UNION SELECT", vector: "false' UNION SELECT * FROM passwords--"},
		{name: "SQL injection OR 1=1", vector: "true' OR '1'='1"},
		{name: "SQL injection comment", vector: "true--malicious"},
		{name: "SQL injection multi-line comment", vector: "true/*malicious*/"},
		{name: "SQL injection WAITFOR", vector: "true; WAITFOR DELAY '00:00:10'--"},
		{name: "SQL injection batch", vector: "true; EXEC xp_cmdshell('whoami')--"},

		// Connection string injection
		{name: "connection string parameter injection", vector: "true&password=hacked"},
		{name: "connection string host override", vector: "true&host=evil.com"},
		{name: "connection string port override", vector: "true&port=31337"},
		{name: "connection string database override", vector: "true&database=admin"},
		{name: "connection string user override", vector: "true&user=root"},
		{name: "connection string semicolon", vector: "true;server=evil.com"},
		{name: "connection string DSN override", vector: "false;DSN=evil"},

		// URL encoding attacks
		{name: "URL encoded ampersand", vector: "true%26password=hacked"},
		{name: "URL encoded semicolon", vector: "true%3Bserver=evil"},
		{name: "URL encoded space", vector: "true%20DROP%20TABLE"},
		{name: "URL encoded equals", vector: "true%3Dmalicious"},
		{name: "double URL encoding", vector: "true%2526password%253Dhacked"},
		{name: "mixed encoding attack", vector: "true&pass%77ord=hacked"},

		// Null byte injection
		{name: "null byte termination", vector: "true\x00malicious"},
		{name: "null byte in middle", vector: "tr\x00ue"},
		{name: "multiple null bytes", vector: "true\x00\x00\x00extra"},

		// Newline injection (CRLF)
		{name: "newline injection LF", vector: "true\nmalicious"},
		{name: "newline injection CRLF", vector: "true\r\nmalicious"},
		{name: "carriage return only", vector: "true\rmalicious"},
		{name: "newline with parameter", vector: "true\n&password=hacked"},

		// Path traversal
		{name: "path traversal unix", vector: "../../../etc/passwd"},
		{name: "path traversal windows", vector: "..\\..\\..\\windows\\system32"},
		{name: "path traversal URL encoded", vector: "..%2F..%2F..%2Fetc%2Fpasswd"},
		{name: "path traversal with file", vector: "/etc/passwd"},
		{name: "path traversal home", vector: "~/sensitive"},

		// Shell command injection
		{name: "shell command backticks", vector: "true`whoami`"},
		{name: "shell command subshell", vector: "true$(whoami)"},
		{name: "shell command pipe", vector: "true|cat /etc/passwd"},
		{name: "shell command redirect", vector: "true>>/tmp/evil"},
		{name: "shell command semicolon", vector: "true;whoami"},
		{name: "shell command AND", vector: "true&&whoami"},

		// Unicode/encoding attacks
		{name: "unicode homoglyph t", vector: "тrue"},  // Cyrillic т
		{name: "unicode homoglyph e", vector: "truе"},  // Cyrillic е
		{name: "unicode zero width", vector: "tru\u200be"},
		{name: "unicode BOM", vector: "\ufefftrue"},
		{name: "UTF-7 encoding", vector: "+AHQ-rue"},

		// Whitespace manipulation
		{name: "leading spaces", vector: "  true"},
		{name: "trailing spaces", vector: "true  "},
		{name: "tab character", vector: "true\t"},
		{name: "vertical tab", vector: "true\v"},
		{name: "form feed", vector: "true\f"},

		// Protocol smuggling
		{name: "file protocol", vector: "file:///etc/passwd"},
		{name: "ldap protocol", vector: "ldap://evil.com/dc=com"},
		{name: "javascript protocol", vector: "javascript:alert(1)"},
		{name: "data protocol", vector: "data:text/plain,malicious"},

		// Format string attacks
		{name: "format string %s", vector: "true%s%s%s"},
		{name: "format string %x", vector: "true%x%x%x"},
		{name: "format string %n", vector: "true%n%n%n"},

		// Extremely long inputs (potential buffer issues)
		{name: "long input", vector: "true" + string(make([]byte, 1000))},

		// Empty but tricky
		{name: "only whitespace", vector: "   "},
		{name: "only tabs", vector: "\t\t\t"},
		{name: "only newlines", vector: "\n\n\n"},
	}

	t.Run("MySQL validator rejects attack vectors", func(t *testing.T) {
		for _, tt := range attackVectors {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateMySQLMode(tt.vector)
				require.Error(t, err, "Expected MySQL validator to reject attack vector: %q", tt.vector)
				assert.Contains(t, err.Error(), "FET-0413", "Error should use ErrInvalidSSLMode code")
			})
		}
	})

	t.Run("Oracle validator rejects attack vectors", func(t *testing.T) {
		for _, tt := range attackVectors {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateOracleMode(tt.vector)
				require.Error(t, err, "Expected Oracle validator to reject attack vector: %q", tt.vector)
				assert.Contains(t, err.Error(), "FET-0413", "Error should use ErrInvalidSSLMode code")
			})
		}
	})
}

// TestValidModesAccepted ensures valid modes are still accepted after security hardening
func TestValidModesAccepted(t *testing.T) {
	t.Run("MySQL valid modes accepted", func(t *testing.T) {
		validModes := []string{"", "false", "true", "skip-verify", "preferred"}
		for _, mode := range validModes {
			t.Run("mode_"+mode, func(t *testing.T) {
				err := ValidateMySQLMode(mode)
				assert.NoError(t, err, "Expected MySQL validator to accept valid mode: %q", mode)
			})
		}
	})

	t.Run("Oracle valid modes accepted", func(t *testing.T) {
		validModes := []string{"", "disable", "false", "true", "enable", "verify", "skip-verify"}
		for _, mode := range validModes {
			t.Run("mode_"+mode, func(t *testing.T) {
				err := ValidateOracleMode(mode)
				assert.NoError(t, err, "Expected Oracle validator to accept valid mode: %q", mode)
			})
		}
	})
}

// TestCaseSensitivity verifies that validation is case-sensitive
func TestCaseSensitivity(t *testing.T) {
	caseMutations := []struct {
		name  string
		input string
	}{
		{name: "all uppercase TRUE", input: "TRUE"},
		{name: "all uppercase FALSE", input: "FALSE"},
		{name: "mixed case True", input: "True"},
		{name: "mixed case False", input: "False"},
		{name: "CamelCase SkipVerify", input: "SkipVerify"},
		{name: "SKIP-VERIFY uppercase", input: "SKIP-VERIFY"},
		{name: "Preferred capitalized", input: "Preferred"},
		{name: "DISABLE uppercase", input: "DISABLE"},
		{name: "Enable capitalized", input: "Enable"},
		{name: "VERIFY uppercase", input: "VERIFY"},
	}

	t.Run("MySQL case sensitivity", func(t *testing.T) {
		for _, tt := range caseMutations {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateMySQLMode(tt.input)
				require.Error(t, err, "Expected MySQL validator to reject case variant: %q", tt.input)
			})
		}
	})

	t.Run("Oracle case sensitivity", func(t *testing.T) {
		for _, tt := range caseMutations {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateOracleMode(tt.input)
				require.Error(t, err, "Expected Oracle validator to reject case variant: %q", tt.input)
			})
		}
	})
}

// TestCrossDatabaseModeRejection ensures modes valid for one DB are rejected for another
func TestCrossDatabaseModeRejection(t *testing.T) {
	t.Run("PostgreSQL modes rejected by MySQL", func(t *testing.T) {
		postgresOnlyModes := []string{"disable", "allow", "require", "verify-ca", "verify-full"}
		for _, mode := range postgresOnlyModes {
			t.Run(mode, func(t *testing.T) {
				err := ValidateMySQLMode(mode)
				require.Error(t, err, "Expected MySQL to reject PostgreSQL mode: %q", mode)
			})
		}
	})

	t.Run("PostgreSQL modes rejected by Oracle", func(t *testing.T) {
		postgresOnlyModes := []string{"allow", "require", "verify-ca", "verify-full"}
		for _, mode := range postgresOnlyModes {
			t.Run(mode, func(t *testing.T) {
				err := ValidateOracleMode(mode)
				require.Error(t, err, "Expected Oracle to reject PostgreSQL mode: %q", mode)
			})
		}
	})

	t.Run("MySQL-only mode rejected by Oracle", func(t *testing.T) {
		err := ValidateOracleMode("preferred")
		require.Error(t, err, "Expected Oracle to reject MySQL-only mode: preferred")
	})

	t.Run("Oracle-only modes rejected by MySQL", func(t *testing.T) {
		oracleOnlyModes := []string{"disable", "enable", "verify"}
		for _, mode := range oracleOnlyModes {
			t.Run(mode, func(t *testing.T) {
				err := ValidateMySQLMode(mode)
				require.Error(t, err, "Expected MySQL to reject Oracle-only mode: %q", mode)
			})
		}
	})
}
