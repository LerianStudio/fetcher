package sslmode

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
)

// ValidateWithHint validates a mode against valid modes and provides helpful error messages.
// It detects case mismatches and suggests the correct lowercase version.
func ValidateWithHint(mode string, validModes map[string]struct{}, dbType string) error {
	// Check if mode is valid
	if _, valid := validModes[mode]; valid {
		return nil
	}

	// Check if it's a case mismatch
	lowerMode := strings.ToLower(mode)
	if _, valid := validModes[lowerMode]; valid {
		return fmt.Errorf("%w: %q is invalid for %s (did you mean %q? SSL modes are case-sensitive)",
			constant.ErrInvalidSSLMode, mode, dbType, lowerMode)
	}

	// Get list of valid modes for error message
	modes := make([]string, 0, len(validModes))
	for m := range validModes {
		if m != "" {
			modes = append(modes, fmt.Sprintf("%q", m))
		}
	}

	sort.Strings(modes)

	return pkg.ValidateBusinessError(constant.ErrInvalidSSLMode, "",
		fmt.Sprintf("%s (valid modes for %s: %s)", mode, dbType, strings.Join(modes, ", ")))
}
