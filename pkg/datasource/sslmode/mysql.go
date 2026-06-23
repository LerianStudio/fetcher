package sslmode

import (
	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
)

// validMySQLModes contains the allowlist of valid MySQL TLS mode values.
// These are the only values accepted by the go-sql-driver/mysql driver.
//
// Reference: https://pkg.go.dev/github.com/go-sql-driver/mysql#RegisterTLSConfig
//
// Valid values:
//   - "" (empty): Same as "false", TLS disabled
//   - "false": TLS disabled (default)
//   - "true": TLS enabled with certificate verification
//   - "skip-verify": TLS enabled without certificate verification (insecure)
//   - "preferred": TLS if available, fallback to unencrypted (insecure)
//
// Note: Custom TLS configs registered via mysql.RegisterTLSConfig() are NOT
// supported through this validation as they require code changes to register.
var validMySQLModes = map[string]struct{}{
	"":            {}, // Driver default (same as false)
	"false":       {}, // Explicitly disable TLS
	"true":        {}, // Enable TLS with certificate verification
	"skip-verify": {}, // Enable TLS, skip certificate verification (INSECURE)
	"preferred":   {}, // Use TLS if server supports it, otherwise plaintext (INSECURE)
}

// ValidateMySQLMode validates that the provided SSL mode is in the allowlist
// of valid MySQL TLS modes. Returns an error if the mode is not valid.
//
// This function is designed to prevent injection attacks by rejecting any
// value that is not explicitly in the allowlist. The validation is case-sensitive
// because the MySQL driver is case-sensitive.
//
// Example:
//
//	if err := ValidateMySQLMode(conn.SSL.Mode); err != nil {
//	    return nil, err // Invalid mode, potential injection
//	}
func ValidateMySQLMode(mode string) error {
	if _, valid := validMySQLModes[mode]; !valid {
		return pkg.ValidateBusinessError(constant.ErrInvalidSSLMode, "", mode)
	}

	return nil
}

// GetValidMySQLModes returns a copy of all valid MySQL SSL mode values.
// This can be used for documentation or error messages.
func GetValidMySQLModes() []string {
	modes := make([]string, 0, len(validMySQLModes))
	for mode := range validMySQLModes {
		modes = append(modes, mode)
	}

	return modes
}
