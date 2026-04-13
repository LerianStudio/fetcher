# Fetcher Changelog

## [1.1.0](https://github.com/LerianStudio/fetcher/releases/tag/v1.1.0)

Features:
- Added environment-aware signature verification and nil-safe telemetry to improve security and reliability.
- Introduced a unique active-job index with auto-remediation and deduplication lookup for MongoDB.
- Implemented error sanitization utility for the bootstrap phase to enhance error handling.
- Added a derive-key CLI tool for external HMAC verification to support cryptographic operations.

Fixes:
- Added panic recovery to the main entrypoint of the worker to prevent unexpected crashes.
- Hardened validation and expanded edge-case test coverage in the worker to improve robustness.
- Replaced panics with error propagation in the worker bootstrap to ensure graceful error handling.
- Prevented nil pointer dereference in the worker's error handling to avoid runtime errors.
- Corrected variadic expansion and refactored the HTTP error handler for better error management.

Improvements:
- Removed the file-based health check mechanism in the worker to streamline health monitoring.
- Standardized blank lines before error returns in the manager to improve code readability.
- Migrated from deprecated golang/mock to go.uber.org/mock for better mock management.
- Updated testcontainers API and extracted secret helpers in itestkit for enhanced testing capabilities.
- Enhanced build workflow with a multi-arch strategy and E2E test placeholder for improved CI/CD processes.

Contributors: @bruno.souza, @brunoblsouza, @dependabot[bot], @ferr3ira-gabriel, @ferr3ira.gabriel, @fred, @gui.rodrigues, @guimoreirar, @lerian-studio-midaz-push-bot[bot], @maciell1

[Compare changes](https://github.com/LerianStudio/fetcher/compare/v1.0.1...v1.1.0)

---

## [v1.0.1-beta.1] - 2026-01-14

### 🔧 Maintenance
- update shared-workflows to v1.7.0 and add GPT changelog (#39) (#39)


## [v1.0.0]

