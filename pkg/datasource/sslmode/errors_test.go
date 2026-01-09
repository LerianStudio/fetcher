package sslmode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWithHint_CaseMismatch(t *testing.T) {
	t.Parallel()

	validModes := map[string]struct{}{
		"":       {},
		"true":   {},
		"false":  {},
		"strict": {},
	}

	tests := []struct {
		name         string
		mode         string
		expectError  bool
		expectHint   bool
		expectedHint string
	}{
		{
			name:        "valid lowercase",
			mode:        "true",
			expectError: false,
		},
		{
			name:         "uppercase suggests lowercase",
			mode:         "TRUE",
			expectError:  true,
			expectHint:   true,
			expectedHint: "did you mean \"true\"",
		},
		{
			name:         "mixed case suggests lowercase",
			mode:         "True",
			expectError:  true,
			expectHint:   true,
			expectedHint: "did you mean \"true\"",
		},
		{
			name:        "invalid mode shows valid options",
			mode:        "invalid",
			expectError: true,
			expectHint:  false,
		},
		{
			name:        "empty string is valid",
			mode:        "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateWithHint(tt.mode, validModes, "TestDB")

			if !tt.expectError {
				assert.NoError(t, err)

				return
			}

			require.Error(t, err)
			if tt.expectHint {
				assert.Contains(t, err.Error(), tt.expectedHint)
				assert.Contains(t, err.Error(), "case-sensitive")
			}
		})
	}
}

func TestValidateWithHint_ShowsValidModes(t *testing.T) {
	t.Parallel()

	validModes := map[string]struct{}{
		"":        {},
		"disable": {},
		"require": {},
	}

	err := ValidateWithHint("invalid-mode", validModes, "PostgreSQL")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid modes for PostgreSQL")
	assert.Contains(t, err.Error(), "disable")
	assert.Contains(t, err.Error(), "require")
}
