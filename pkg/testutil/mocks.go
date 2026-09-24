// Package testutil provides shared test utilities and mocks for the fetcher project.
package testutil

import (
	"context"

	"github.com/LerianStudio/lib-observability/v4/log"
)

// Compile-time interface compliance verification.
var _ log.Logger = (*MockLogger)(nil)

// MockLogger implements log.Logger for testing.
//
// Hand-written rather than generated: mockgen on an external interface needs
// either a local wrapper interface or reflect mode against the library path.
type MockLogger struct{}

func (m *MockLogger) Log(_ context.Context, _ int, _ string, _ ...any) {}
func (m *MockLogger) With(_ ...any) log.Logger                         { return m }
func (m *MockLogger) WithGroup(_ string) log.Logger                    { return m }
func (m *MockLogger) Enabled(_ int) bool                               { return true }
func (m *MockLogger) Sync(_ context.Context) error                     { return nil }
