package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyError_Timeout(t *testing.T) {
	testCases := []string{
		"connection timeout",
		"context deadline exceeded",
		"i/o timeout",
		"request timed out",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			assert.Equal(t, ErrorCategoryTimeout, ClassifyError(tc))
		})
	}
}

func TestClassifyError_Connection(t *testing.T) {
	testCases := []string{
		"connection refused",
		"connection reset by peer",
		"broken pipe",
		"dial tcp: connection refused",
		"EOF",
		"connection timed out", // TCP connection establishment timeout
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			assert.Equal(t, ErrorCategoryConnection, ClassifyError(tc))
		})
	}
}

func TestClassifyError_Network(t *testing.T) {
	testCases := []string{
		"dns lookup failed",
		"no route to host",
		"network is down",
		"no such host", // DNS resolution failure
		"lookup database-host on 127.0.0.1:53: no such host", // Full DNS error
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			assert.Equal(t, ErrorCategoryNetwork, ClassifyError(tc))
		})
	}
}

func TestClassifyError_Application(t *testing.T) {
	testCases := []string{
		"HTTP 500 Internal Server Error",
		"400 Bad Request",
		"503 Service Unavailable",
		"unauthorized access",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			assert.Equal(t, ErrorCategoryApplication, ClassifyError(tc))
		})
	}
}

func TestClassifyError_Unknown(t *testing.T) {
	testCases := []string{
		"some random error",
		"unexpected behavior",
		"",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			assert.Equal(t, ErrorCategoryUnknown, ClassifyError(tc))
		})
	}
}

func TestErrorClassifier_RecordAndRetrieve(t *testing.T) {
	classifier := NewErrorClassifier()

	// Record various errors
	classifier.RecordError("connection timeout")
	classifier.RecordError("connection timeout") // duplicate
	classifier.RecordError("connection refused")
	classifier.RecordError("500 internal server error")

	// Check total
	assert.Equal(t, 4, classifier.GetTotalErrors())

	// Check category counts
	counts := classifier.GetCategoryCounts()
	assert.Equal(t, 2, counts[ErrorCategoryTimeout])
	assert.Equal(t, 1, counts[ErrorCategoryConnection])
	assert.Equal(t, 1, counts[ErrorCategoryApplication])
}

func TestErrorClassifier_GetErrorsByCategory(t *testing.T) {
	classifier := NewErrorClassifier()

	classifier.RecordError("timeout error 1")
	classifier.RecordError("timeout error 2")
	classifier.RecordError("connection refused")

	timeoutErrors := classifier.GetErrorsByCategory(ErrorCategoryTimeout)
	assert.Len(t, timeoutErrors, 2)

	connectionErrors := classifier.GetErrorsByCategory(ErrorCategoryConnection)
	assert.Len(t, connectionErrors, 1)
}

func TestErrorClassifier_DuplicateTracking(t *testing.T) {
	classifier := NewErrorClassifier()

	// Record same error multiple times
	for i := 0; i < 5; i++ {
		classifier.RecordError("connection timeout")
	}

	errors := classifier.GetErrorsByCategory(ErrorCategoryTimeout)
	assert.Len(t, errors, 1) // Should be deduplicated
	assert.Equal(t, 5, errors[0].Count)
}

func TestErrorClassifier_Reset(t *testing.T) {
	classifier := NewErrorClassifier()

	classifier.RecordError("some error")
	assert.Equal(t, 1, classifier.GetTotalErrors())

	classifier.Reset()
	assert.Equal(t, 0, classifier.GetTotalErrors())
}
