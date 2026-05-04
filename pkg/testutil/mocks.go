// Package testutil provides shared test utilities and mocks for the fetcher project.
package testutil

import (
	"context"

	"github.com/LerianStudio/lib-commons/v5/commons/log"
)

// Compile-time interface compliance verification.
var _ log.Logger = (*MockLogger)(nil)

// MockLogger implements log.Logger for testing.
//
// NOTE: This manual mock is intentionally retained because log.Logger is an external
// interface from github.com/LerianStudio/lib-commons/v5/commons/log. Generating mockgen
// mocks for external interfaces requires either:
// 1. A local wrapper interface (adds unnecessary indirection)
// 2. Reflect mode with full package path (fragile to library changes)
// For simple logging interfaces used only in tests, a manual mock is more maintainable.
type MockLogger struct{}

func (m *MockLogger) Log(_ context.Context, _ log.Level, _ string, _ ...log.Field) {}
func (m *MockLogger) With(_ ...log.Field) log.Logger                               { return m }
func (m *MockLogger) WithGroup(_ string) log.Logger                                { return m }
func (m *MockLogger) Enabled(_ log.Level) bool                                     { return true }
func (m *MockLogger) Sync(_ context.Context) error                                 { return nil }
