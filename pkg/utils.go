package pkg

import (
	"errors"
	"math"
	"regexp"
	"strings"
)

// SafeIntToInt32 Function to safely convert int to int32 with overflow check
func SafeIntToInt32(val int) (int32, error) {
	if val > math.MaxInt32 || val < math.MinInt32 {
		return 0, errors.New("integer overflow: value out of range for int32")
	}

	return int32(val), nil
}

// IsNilOrEmpty returns a boolean indicating if a *string is nil or empty.
// It's use TrimSpace so, a string "  " and "" and "null" and "nil" will be considered empty
func IsNilOrEmpty(s *string) bool {
	return s == nil || strings.TrimSpace(*s) == "" || strings.TrimSpace(*s) == "null" || strings.TrimSpace(*s) == "nil"
}

// ValidateServerAddress checks if the value matches the pattern <some-address>:<some-port> and returns the value if it does.
func ValidateServerAddress(value string) string {
	matched, _ := regexp.MatchString(`^[^:]+:\d+$`, value)
	if !matched {
		return ""
	}

	return value
}

// MaskSecret masks the given secret value by returning "[REDACTED]" if the value is not empty.
func MaskSecret(value string) string {
	if value == "" {
		return ""
	}

	return "[REDACTED]"
}

// MaskSecretPtr masks the given secret value by returning a pointer to "[REDACTED]" if the value is not nil or empty.
func MaskSecretPtr(value *string) *string {
	if value == nil || *value == "" {
		return value
	}

	masked := "[REDACTED]"
	return &masked
}
