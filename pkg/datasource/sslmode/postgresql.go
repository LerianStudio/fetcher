package sslmode

import (
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
)

// validPostgreSQLModes contains the allowlist of valid PostgreSQL SSL mode values.
// These are the only values accepted by the lib/pq and pgx drivers.
//
// Reference: https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-SSLMODE-STATEMENTS
//
// Valid values:
//   - "" (empty): Same as "disable", SSL disabled
//   - "disable": SSL disabled (default)
//   - "allow": Try non-SSL first, then SSL if server requires it
//   - "prefer": Try SSL first, fallback to non-SSL (default for pgx)
//   - "require": Require SSL but don't verify certificate
//   - "verify-ca": Require SSL and verify that server cert is signed by trusted CA
//   - "verify-full": Require SSL, verify CA, and verify server hostname matches cert
var validPostgreSQLModes = map[string]struct{}{
	"":            {}, // Driver default (same as disable)
	"disable":     {}, // Explicitly disable SSL
	"allow":       {}, // Try non-SSL first, SSL if required
	"prefer":      {}, // Try SSL first, fallback to non-SSL
	"require":     {}, // Require SSL, don't verify certificate
	"verify-ca":   {}, // Require SSL, verify CA
	"verify-full": {}, // Require SSL, verify CA and hostname
}

// ValidatePostgreSQLMode validates that the provided SSL mode is in the allowlist
// of valid PostgreSQL SSL modes. Returns an error if the mode is not valid.
//
// This function is designed to prevent injection attacks by rejecting any
// value that is not explicitly in the allowlist. The validation is case-sensitive
// because the PostgreSQL driver is case-sensitive.
//
// Example:
//
//	if err := ValidatePostgreSQLMode(conn.SSL.Mode); err != nil {
//	    return nil, err // Invalid mode, potential injection
//	}
func ValidatePostgreSQLMode(mode string) error {
	if _, valid := validPostgreSQLModes[mode]; !valid {
		return pkg.ValidateBusinessError(constant.ErrInvalidSSLMode, "", mode)
	}

	return nil
}

// GetValidPostgreSQLModes returns a copy of all valid PostgreSQL SSL mode values.
// This can be used for documentation or error messages.
func GetValidPostgreSQLModes() []string {
	modes := make([]string, 0, len(validPostgreSQLModes))
	for mode := range validPostgreSQLModes {
		modes = append(modes, mode)
	}

	return modes
}
