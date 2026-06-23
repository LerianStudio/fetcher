# Fetcher Changelog

## [2.0.2](https://github.com/LerianStudio/fetcher/releases/tag/v2.0.2)

- Changelog for `fetcher` `v2.0.2`:

Fixes:
- Promote engine release channel fix to stable.
- Ensure engine release channel follows the parent prerelease.

Contributors: @fredcamaral, @lerian-studio

[Compare changes](https://github.com/LerianStudio/fetcher/compare/v2.0.1...v2.0.2)

---

## [2.0.1](https://github.com/LerianStudio/fetcher/releases/tag/v2.0.1)

- Updated `fetcher` to `v2.0.1`.
- Updated `fetcher` to `v2.0.1-beta.1`.
- Updated CHANGELOGs for `fetcher:v2.0.0`.

Contributors: @fredcamaral, @lerian-studio.

[Compare changes](https://github.com/LerianStudio/fetcher/compare/v2.0.0...v2.0.1)

---

## [2.0.0](https://github.com/LerianStudio/fetcher/releases/tag/v2.0.0)

- Features:
  - Migrated fetcher worker to use `lib-streaming` `v1.6` with empty tenant defaulting to single-tenant mode.
  - Introduced `ENGINE_MAX_RESULT_BYTES` override and streaming broker health probe for enhanced worker performance.
  - Implemented fetcher as a runtime engine, enabling execution via engine with host-side schema-name normalization.
  - Added canonical result model and in-memory result sink to the engine.
  - Enforced cancellation, timeout, and result-size limits in the engine runner.

- Fixes:
  - Made missing-env secret test deterministic in CI environments.
  - Restored Database Connection Error fidelity on `/schema` endpoint.
  - Hardened multi-tenant consumer and stabilized terminal event remediation in the worker.
  - Remediated terminal repair and publish confirms for streaming publisher.
  - Addressed CodeRabbit findings on fetcher as runtime engine PR.

- Improvements:
  - Removed `lib-license-go` enforcement as fetcher is public OSS.
  - Bumped Lerian libs to latest stable versions, excluding `lib-streaming`.
  - Added Elastic License 2.0 to the project.
  - Improved documentation for fetcher embedded runtime and updated architecture documentation.
  - Enhanced CI workflows with upgraded shared workflows and added Docker dependabot entries.

Contributors: @bedatty, @brunobls, @fredcamaral, @jeffersonrodrigues92, @lerian-studio.

[Compare changes](https://github.com/LerianStudio/fetcher/compare/v1.4.2...v2.0.0)

---

## [1.4.2](https://github.com/LerianStudio/fetcher/releases/tag/v1.4.2)

- Fixes:
  - Bumped lib-commons to v5.1.3 to address issues with HTTP/1.1 tmclient transport.

Contributors: @jeffersonrodrigues92, @lerian-studio

[Compare changes](https://github.com/LerianStudio/fetcher/compare/v1.4.1...v1.4.2)

---

## [1.4.1](https://github.com/LerianStudio/fetcher/releases/tag/v1.4.1)

- Fixes:
  - Fixed issue with subscribing to environment-scoped tenant-events channel.

Contributors: @jeffersonrodrigues92

[Compare changes](https://github.com/LerianStudio/fetcher/compare/v1.4.0...v1.4.1)

---

## [1.3.0](https://github.com/LerianStudio/fetcher/releases/tag/v1.3.0)

Features:
- Added `MONGO_PARAMETERS` environment variable for MongoDB connection query parameters.
- Introduced `MULTI_TENANT_ALLOW_INSECURE_HTTP` environment variable for explicit HTTP control.
- Added `RABBITMQ_TLS` environment variable for multi-tenant RabbitMQ TLS connections.
- Implemented TLS environment variable support for MongoDB, Redis, and multi-tenant Redis.

Fixes:
- Resolved parsing of TLS parameters from raw MongoDB URI when boolean fields are unset.
- Propagated MongoDB TLS, DirectConnection, and AuthSource from tenant-manager.
- Addressed CodeRabbit review findings.
- Fixed security issues by ignoring certain CVEs pending safe upgrade paths.

Improvements:
- Updated lib-commons to the latest version and added new functions.
- Improved struct field formatting and simplified `AllowInsecureHTTP` assignment.
- Enhanced test assertions with testify and removed redundant handler validation.
- Aligned module and CI toolchain to version 1.25.7.

Contributors: @arthur.ribeiro, @arthurkz, @bedatty, @bruno.souza, @brunoblsouza, @dependabot[bot], @fred, @gandalf, @jefferson.comff, @lucas.bedatty

[Compare changes](https://github.com/LerianStudio/fetcher/compare/v1.2.0...v1.3.0)

---

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

