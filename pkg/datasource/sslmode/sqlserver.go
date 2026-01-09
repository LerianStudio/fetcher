package sslmode

import (
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
)

// validSQLServerModes contains the allowlist of valid SQL Server encryption mode values.
// These values map to the encrypt parameter in the go-mssqldb driver.
//
// Reference: https://github.com/microsoft/go-mssqldb#connection-parameters-and-dsn
//
// Valid values:
//   - "" (empty): Same as "false", encryption disabled
//   - "disable": Encryption explicitly disabled
//   - "false": Encryption disabled (default)
//   - "true": Encryption enabled with certificate verification
//   - "strict": Encryption with TDS 8.0 (SQL Server 2022+)
//
// Note: The go-mssqldb driver also uses trustServerCertificate parameter
// to skip certificate verification when encrypt=true.
var validSQLServerModes = map[string]struct{}{
	"":        {}, // Empty string defaults to "false" (no encryption)
	"disable": {}, // Explicitly disable encryption
	"false":   {}, // Disable encryption (default)
	"true":    {}, // Enable encryption with certificate verification
	"strict":  {}, // TDS 8.0 strict encryption (SQL Server 2022+)
}

// ValidateSQLServerMode validates that the provided encryption mode is in the allowlist
// of valid SQL Server encryption modes. Returns an error if the mode is not valid.
//
// This function is designed to prevent injection attacks by rejecting any
// value that is not explicitly in the allowlist. The validation is case-sensitive
// for consistency and security.
//
// Example:
//
//	if err := ValidateSQLServerMode(conn.SSL.Mode); err != nil {
//	    return nil, err // Invalid mode, potential injection
//	}
func ValidateSQLServerMode(mode string) error {
	if _, valid := validSQLServerModes[mode]; !valid {
		return pkg.ValidateBusinessError(constant.ErrInvalidSSLMode, "", mode)
	}

	return nil
}

// GetValidSQLServerModes returns a copy of all valid SQL Server encryption mode values.
// This can be used for documentation or error messages.
func GetValidSQLServerModes() []string {
	modes := make([]string, 0, len(validSQLServerModes))
	for mode := range validSQLServerModes {
		modes = append(modes, mode)
	}

	return modes
}
