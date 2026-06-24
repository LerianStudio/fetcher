## [1.4.5](https://github.com/LerianStudio/fetcher/compare/v1.4.4...v1.4.5) (2026-06-24)


### Bug Fixes

* **ci:** add release-notes-generator and changelog plugins for maintenance ([#297](https://github.com/LerianStudio/fetcher/issues/297)) ([b56f745](https://github.com/LerianStudio/fetcher/commit/b56f745044515b983ac0ed10af28443a81589658))
* **ci:** add release-notes-generator and changelog plugins to releaserc ([3d45a93](https://github.com/LerianStudio/fetcher/commit/3d45a93f28bfcf8a3532db44dfe767c41e8c3d29))
* **ci:** disable gitops artifacts on maintenance release pipeline ([365d3a0](https://github.com/LerianStudio/fetcher/commit/365d3a0a570fa1d0d49409c781a4896f7d5f08fc))
* **ci:** trigger gptchangelog on maintenance branch releases ([eaf14ac](https://github.com/LerianStudio/fetcher/commit/eaf14ac4d24573dd81bf2aafe189602f75af3fa0))
* **ci:** use semantic-release plugins for changelog on maintenance branch ([e511747](https://github.com/LerianStudio/fetcher/commit/e511747d19cb44e0b64ed8427c35251207059e90))

# Fetcher Changelog

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
