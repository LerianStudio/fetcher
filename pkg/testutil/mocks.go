// Package testutil provides shared test utilities and mocks for the fetcher project.
package testutil

import (
	"github.com/LerianStudio/lib-commons/v3/commons/log"
)

// Compile-time interface compliance verification.
var _ log.Logger = (*MockLogger)(nil)

// MockLogger implements log.Logger for testing.
//
// NOTE: This manual mock is intentionally retained because log.Logger is an external
// interface from github.com/LerianStudio/lib-commons/v3/commons/log. Generating mockgen
// mocks for external interfaces requires either:
// 1. A local wrapper interface (adds unnecessary indirection)
// 2. Reflect mode with full package path (fragile to library changes)
// For simple logging interfaces used only in tests, a manual mock is more maintainable.
type MockLogger struct{}

func (m *MockLogger) Info(args ...any)                                     {}
func (m *MockLogger) Infof(format string, args ...any)                     {}
func (m *MockLogger) Infoln(args ...any)                                   {}
func (m *MockLogger) Error(args ...any)                                    {}
func (m *MockLogger) Errorf(format string, args ...any)                    {}
func (m *MockLogger) Errorln(args ...any)                                  {}
func (m *MockLogger) Warn(args ...any)                                     {}
func (m *MockLogger) Warnf(format string, args ...any)                     {}
func (m *MockLogger) Warnln(args ...any)                                   {}
func (m *MockLogger) Debug(args ...any)                                    {}
func (m *MockLogger) Debugf(format string, args ...any)                    {}
func (m *MockLogger) Debugln(args ...any)                                  {}
func (m *MockLogger) Fatal(args ...any)                                    {}
func (m *MockLogger) Fatalf(format string, args ...any)                    {}
func (m *MockLogger) Fatalln(args ...any)                                  {}
func (m *MockLogger) WithFields(fields ...any) log.Logger                  { return m }
func (m *MockLogger) WithDefaultMessageTemplate(message string) log.Logger { return m }
func (m *MockLogger) Sync() error                                          { return nil }
