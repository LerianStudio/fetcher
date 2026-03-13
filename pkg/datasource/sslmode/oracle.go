package sslmode

import (
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
)

// validOracleModes contains the allowlist of valid Oracle SSL mode values.
// These values map to the go-ora driver's ssl and ssl_verify parameters.
//
// Reference: https://github.com/sijms/go-ora
//
// Valid values:
//   - "" (empty): SSL disabled (no encryption)
//   - "disable": SSL explicitly disabled (no encryption)
//   - "false": SSL disabled (alias for disable, no encryption)
//   - "true": SSL enabled - encrypts connection, server cert validated against system CA store
//   - "enable": SSL enabled (alias for true) - same behavior as "true"
//   - "verify": SSL enabled with STRICT verification - requires valid CA chain AND hostname match
//   - "skip-verify": SSL enabled but certificate NOT validated (INSECURE - vulnerable to MITM)
//
// The go-ora driver uses these to construct connection options:
//   - ssl=true|enable enables SSL
//   - ssl_verify=false with ssl=true enables skip-verify mode
var validOracleModes = map[string]struct{}{
	"":            {}, // Default: SSL disabled
	"disable":     {}, // Explicitly disable SSL
	"false":       {}, // Disable SSL (same as disable)
	"true":        {}, // Enable SSL with verification
	"enable":      {}, // Enable SSL with verification (alias for true)
	"verify":      {}, // Enable SSL with strict verification
	"skip-verify": {}, // Enable SSL, skip certificate verification (INSECURE)
}

// ValidateOracleMode validates that the provided SSL mode is in the allowlist
// of valid Oracle SSL modes. Returns an error if the mode is not valid.
//
// This function is designed to prevent injection attacks by rejecting any
// value that is not explicitly in the allowlist. The validation is case-sensitive
// for consistency and security.
//
// Example:
//
//	if err := ValidateOracleMode(conn.SSL.Mode); err != nil {
//	    return nil, err // Invalid mode, potential injection
//	}
func ValidateOracleMode(mode string) error {
	if _, valid := validOracleModes[mode]; !valid {
		return pkg.ValidateBusinessError(constant.ErrInvalidSSLMode, "", mode)
	}

	return nil
}

// GetValidOracleModes returns a copy of all valid Oracle SSL mode values.
// This can be used for documentation or error messages.
func GetValidOracleModes() []string {
	modes := make([]string, 0, len(validOracleModes))
	for mode := range validOracleModes {
		modes = append(modes, mode)
	}

	return modes
}
