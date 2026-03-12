package sslmode

import (
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
)

// validMongoDBModes contains the allowlist of valid MongoDB TLS mode values.
// MongoDB uses the tls and tlsInsecure connection string parameters.
//
// Reference: https://www.mongodb.com/docs/drivers/go/current/fundamentals/connection/tls/
//
// Valid values:
//   - "" (empty): TLS disabled (default for localhost, enabled for Atlas)
//   - "disable": TLS explicitly disabled
//   - "false": TLS disabled (alias for disable)
//   - "true": TLS enabled with certificate verification
//   - "enable": TLS enabled with certificate verification (alias for true)
//   - "insecure": TLS enabled without certificate verification (tlsInsecure=true)
//
// Note: MongoDB Go driver uses tlsInsecure=true to skip verification,
// which maps to our "insecure" mode.
var validMongoDBModes = map[string]struct{}{
	"disable":  {}, // Explicitly disable TLS
	"false":    {}, // Disable TLS (same as disable)
	"true":     {}, // Enable TLS with verification
	"enable":   {}, // Enable TLS with verification (alias for true)
	"insecure": {}, // Enable TLS, skip certificate verification (INSECURE)
}

// ValidateMongoDBMode validates that the provided SSL/TLS mode is in the allowlist
// of valid MongoDB TLS modes. Returns an error if the mode is not valid.
//
// This function is designed to prevent injection attacks by rejecting any
// value that is not explicitly in the allowlist. The validation is case-sensitive
// for consistency and security.
//
// Example:
//
//	if err := ValidateMongoDBMode(conn.SSL.Mode); err != nil {
//	    return nil, err // Invalid mode, potential injection
//	}
func ValidateMongoDBMode(mode string) error {
	if _, valid := validMongoDBModes[mode]; !valid {
		return pkg.ValidateBusinessError(constant.ErrInvalidSSLMode, "", mode)
	}

	return nil
}

// GetValidMongoDBModes returns a copy of all valid MongoDB TLS mode values.
// This can be used for documentation or error messages.
func GetValidMongoDBModes() []string {
	modes := make([]string, 0, len(validMongoDBModes))
	for mode := range validMongoDBModes {
		modes = append(modes, mode)
	}

	return modes
}
