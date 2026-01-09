package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrInvalidSSLModeExists(t *testing.T) {
	// Verify the error constant exists and has the expected format
	assert.NotNil(t, ErrInvalidSSLMode)
	assert.Contains(t, ErrInvalidSSLMode.Error(), "FET-")
}
