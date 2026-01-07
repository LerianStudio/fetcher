package helpers

import (
	"strings"
	"sync"
	"time"
)

// ErrorCategory represents a category of errors during chaos testing.
type ErrorCategory string

const (
	// ErrorCategoryTimeout indicates a timeout error.
	ErrorCategoryTimeout ErrorCategory = "timeout"

	// ErrorCategoryConnection indicates a connection error (refused, reset, etc.).
	ErrorCategoryConnection ErrorCategory = "connection"

	// ErrorCategoryNetwork indicates a network-level error (DNS, routing, etc.).
	ErrorCategoryNetwork ErrorCategory = "network"

	// ErrorCategoryApplication indicates an application-level error (4xx, 5xx).
	ErrorCategoryApplication ErrorCategory = "application"

	// ErrorCategoryUnknown indicates an unclassified error.
	ErrorCategoryUnknown ErrorCategory = "unknown"
)

// ClassifiedError represents an error with its classification.
type ClassifiedError struct {
	Category  ErrorCategory
	Message   string
	Count     int
	FirstSeen time.Time
	LastSeen  time.Time
}

// ErrorClassifier categorizes errors during chaos testing.
type ErrorClassifier struct {
	mu     sync.Mutex
	errors map[ErrorCategory]map[string]*ClassifiedError // category -> message -> error
	total  int
}

// NewErrorClassifier creates a new ErrorClassifier.
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{
		errors: make(map[ErrorCategory]map[string]*ClassifiedError),
	}
}

// ClassifyError classifies an error message into a category.
// Pattern matching is ordered from most specific to least specific to ensure
// correct classification of ambiguous error messages.
func ClassifyError(errMsg string) ErrorCategory {
	msg := strings.ToLower(errMsg)

	// Network patterns (DNS/routing layer) - check FIRST as most specific
	// These indicate infrastructure-level failures before connection attempts.
	networkPatterns := []string{
		"dns", "lookup", "resolve", "routing",
		"network is down", "no route to host",
		"no such host", // DNS resolution failure
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(msg, pattern) {
			return ErrorCategoryNetwork
		}
	}

	// Connection patterns (transport layer) - check before generic timeouts
	// These indicate TCP/transport-level failures.
	connectionPatterns := []string{
		"connection timed out", // TCP connection establishment timeout (more specific)
		"connection refused", "connection reset",
		"connection closed", "broken pipe",
		"host unreachable", "network unreachable",
		"dial tcp", "connect:", "eof",
	}

	for _, pattern := range connectionPatterns {
		if strings.Contains(msg, pattern) {
			return ErrorCategoryConnection
		}
	}

	// Timeout patterns (request/application layer)
	// These indicate request-level or context timeouts, not TCP-level.
	timeoutPatterns := []string{
		"context deadline exceeded", "i/o timeout",
		"deadline exceeded", "timed out", "timeout",
	}

	for _, pattern := range timeoutPatterns {
		if strings.Contains(msg, pattern) {
			return ErrorCategoryTimeout
		}
	}

	// Application patterns (HTTP status codes and error messages)
	applicationPatterns := []string{
		"status 400", "status 401", "status 403", "status 404", "status 405",
		"status 500", "status 502", "status 503", "status 504",
		"http 400", "http 401", "http 403", "http 404", "http 405",
		"http 500", "http 502", "http 503", "http 504",
		"bad request", "unauthorized", "forbidden",
		"not found", "internal server error",
		"bad gateway", "service unavailable",
	}

	for _, pattern := range applicationPatterns {
		if strings.Contains(msg, pattern) {
			return ErrorCategoryApplication
		}
	}

	return ErrorCategoryUnknown
}

// RecordError records and classifies an error.
func (e *ErrorClassifier) RecordError(errMsg string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	category := ClassifyError(errMsg)
	now := time.Now()

	if e.errors[category] == nil {
		e.errors[category] = make(map[string]*ClassifiedError)
	}

	if existing, ok := e.errors[category][errMsg]; ok {
		existing.Count++
		existing.LastSeen = now
	} else {
		e.errors[category][errMsg] = &ClassifiedError{
			Category:  category,
			Message:   errMsg,
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
		}
	}

	e.total++
}

// GetErrorsByCategory returns all errors of a specific category.
func (e *ErrorClassifier) GetErrorsByCategory(category ErrorCategory) []*ClassifiedError {
	e.mu.Lock()
	defer e.mu.Unlock()

	var result []*ClassifiedError

	if categoryErrors, ok := e.errors[category]; ok {
		for _, err := range categoryErrors {
			errCopy := *err
			result = append(result, &errCopy)
		}
	}

	return result
}

// GetCategoryCounts returns the count of errors per category.
func (e *ErrorClassifier) GetCategoryCounts() map[ErrorCategory]int {
	e.mu.Lock()
	defer e.mu.Unlock()

	counts := make(map[ErrorCategory]int)

	for category, errors := range e.errors {
		for _, err := range errors {
			counts[category] += err.Count
		}
	}

	return counts
}

// GetTotalErrors returns the total number of errors recorded.
func (e *ErrorClassifier) GetTotalErrors() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.total
}

// GetAllErrors returns all classified errors.
func (e *ErrorClassifier) GetAllErrors() []*ClassifiedError {
	e.mu.Lock()
	defer e.mu.Unlock()

	var result []*ClassifiedError

	for _, categoryErrors := range e.errors {
		for _, err := range categoryErrors {
			errCopy := *err
			result = append(result, &errCopy)
		}
	}

	return result
}

// Reset clears all recorded errors.
func (e *ErrorClassifier) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.errors = make(map[ErrorCategory]map[string]*ClassifiedError)
	e.total = 0
}

// Clone creates a deep copy of the ErrorClassifier for thread-safe snapshots.
func (e *ErrorClassifier) Clone() *ErrorClassifier {
	e.mu.Lock()
	defer e.mu.Unlock()

	clone := &ErrorClassifier{
		errors: make(map[ErrorCategory]map[string]*ClassifiedError),
		total:  e.total,
	}

	for category, categoryErrors := range e.errors {
		clone.errors[category] = make(map[string]*ClassifiedError)

		for msg, err := range categoryErrors {
			errCopy := *err
			clone.errors[category][msg] = &errCopy
		}
	}

	return clone
}
