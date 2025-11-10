# Changelog

All notable changes to this project will be documented in this file.



## [v1.8.0-beta.2] - 2025-05-20

### 🔧 Maintenance
- Update CHANGELOG to reflect recent changes.
- Bump `github.com/jackc/pgx/v5` from version 5.7.2 to 5.7.5 for improved compatibility and performance enhancements.

## [v1.8.0-beta.1] - 2025-05-19

### ✨ Features
- Add prefixes for error codes to improve error identification and handling.

### 🐛 Bug Fixes
- Correct error type mismatches to ensure proper error handling.
- Update PostgreSQL credentials to resolve connection issues.

### 🔄 Changes
- Restructure directory layout and rename files for clarity, enhancing project organization.
- Update internal dependencies and Redis consumer to use `lib-commons` for consistency and maintainability.
- Remove unused Grafana and OTEL-LGTM configurations for a cleaner setup.

### 🗑️ Removed
- Remove unused files to streamline the codebase.

### 📚 Documentation
- Enhance documentation auto-generation for better usability.
- Update project name references across documentation for accuracy.
- Remove outdated API endpoints section from README.

### 🔧 Maintenance
- Apply lint fixes to maintain code quality.
- Update project dependencies and migrate `golangci-lint` settings for improved performance and security.
- Add folders to `.gitignore` to prevent unnecessary files from being tracked.
- Update database replica credentials in the environment example for consistency.
- Remove `set-env` from the `make up` command to align with best practices.
- Add replica as a dependency to ensure proper setup.


## [v1.7.0] - 2025-05-09

### ✨ Features
- Remove monorepo test setup to streamline project configuration
- Remove CI monorepo configuration to enhance build efficiency
- Configure GPT for automated changelog generation, improving release documentation
- Set up CI checks to ensure code quality and consistency
- Configure monorepo CI settings for better integration and deployment processes
- Add path condition configurations to enhance build and deployment flexibility
- Test path condition configurations to ensure reliability and correctness

## [v1.7.0-beta.4] - 2025-05-09

### ✨ Features
- Remove CI monorepo configuration to streamline project setup
- Configure GPT changelog generation to automate and improve changelog creation

### 🔧 Maintenance
- Update CHANGELOG with recent changes to reflect the latest project updates

## [v1.7.0-beta.3] - 2025-05-07

### ✨ Features
- Configure code quality checks to enhance code validation processes

### 🔧 Maintenance
- Update CHANGELOG to reflect recent changes

## [v1.7.0-beta.2] - 2025-05-07

### ✨ Features
- Configure monorepo CI pipeline to streamline continuous integration processes.
- Add path conditions for CI configuration to enhance build efficiency and accuracy.
- Test path condition configuration in CI to ensure reliable deployment and testing workflows.
- Configure steps for Golang GitHub Actions pipeline to automate build and deployment processes.

### 🔧 Maintenance
- Refine CI configuration to improve the overall reliability and maintainability of the build system.

## [v1.7.0-beta.1] - 2025-05-07

### ✨ Features
- Configure path conditions for improved plugin flexibility

### 🔧 Maintenance
- Add tests for path condition configurations to ensure reliability
- Update CHANGELOG to reflect recent changes

## [v1.6.0] - 2025-05-06

### ✨ Features
- Configure steps for the Golang GitHub Actions pipeline to enhance CI/CD capabilities.

### 📚 Documentation
- Update CHANGELOG to reflect recent changes and improvements.

## [v1.6.0-beta.1] - 2025-05-06

### ✨ Features
- Configure steps for Golang GitHub Actions pipeline to streamline CI/CD processes

### 📚 Documentation
- Update CHANGELOG to reflect recent changes and ensure accurate project documentation

## [v1.5.0] - 2025-05-06

### ✨ Features
- Configure job names with a consistent structure for improved clarity and organization
- Normalize workflows to utilize `gptchangelog` and `golang-gh-actions` modules, enhancing automation and consistency
- Configure build processes for multiple platforms, expanding compatibility and deployment options

### 🔧 Maintenance
- Update CHANGELOG to reflect recent changes and improvements

## [v1.5.0-beta.3] - 2025-05-06

### 🔧 Maintenance
- Update CHANGELOG for recent changes
- Update `go.mod` to reflect latest dependencies

## [v1.5.0-beta.2] - 2025-05-05

### ✨ Features
- Configure job names consistently across workflows to improve clarity and maintainability.
- Normalize workflows to utilize `gptchangelog` and `golang-gh-actions` modules, enhancing integration and standardization across CI/CD processes.

### 🔧 Maintenance
- Update CHANGELOG with recent changes to ensure documentation reflects the latest project updates.

## [v1.5.0-beta.1] - 2025-04-25

### 🔧 Maintenance
- Configure build process for multiple platforms to enhance compatibility and deployment efficiency.
=======
## [v1.4.0] - 2025-04-25

### ✨ Features
- Configure test environment for plugin development, enhancing the development workflow and ensuring robust plugin testing.

### 🔧 Maintenance
- Update CHANGELOG with recent changes to reflect the latest project updates and improvements.


## [v1.4.0-beta.1] - 2025-04-25

### 🔧 Maintenance
- Configure testing framework for initial setup to ensure reliable test execution environment

## [v1.3.0] - 2025-04-25

### ✨ Features
- Configure release process to streamline deployment and versioning
- Set up file configuration to enhance customization options
- Configure release configuration file for better control over release parameters

### 🔧 Maintenance
- Update changelog to reflect recent changes and improvements

## [v1.3.0-beta.9] - 2025-04-25

### ✨ Features
- Configure release process to streamline deployment and ensure consistency across releases.

### 🔧 Maintenance
- Update CHANGELOG for recent changes to reflect the latest updates and improvements.

## [v1.3.0] - 2025-04-25

### ✨ Features
- Configure initial setup file for plugin functionality, establishing the foundation for future enhancements.

### 📚 Documentation
- Update CHANGELOG with recent changes, ensuring documentation reflects the latest project updates.

## [v1.3.0] - 2025-04-25

### ✨ Features
- Configure release process using a `.releaserc` file to streamline versioning and deployment.

### 📚 Documentation
- Update CHANGELOG to reflect recent changes and improvements in the project.

## [v1.3.0-beta.6] - 2025-04-25
=======
## [v1.2.0] - 2025-04-24

### ✨ Features
- Configure and test CHANGELOG generator to streamline release documentation
- Add comments to document release flow, enhancing clarity for developers

### 📚 Documentation
- Update CHANGELOG to reflect recent changes and improvements

## [v1.2.0] - 2025-04-24


### ✨ Features
- Configure HTTP headers for improved security

### 📚 Documentation
- Update CHANGELOG with recent changes

## [v1.3.0] - 2025-04-25

### ✨ Features
- Configure initial CHANGELOG generation to streamline documentation of project updates.

### 🔧 Maintenance
- Update CHANGELOG with recent changes to ensure all modifications are accurately reflected.

## [v1.3.0] - 2025-04-24

### ✨ Features
- Add configuration line handling to improve the flexibility of plugin configuration.
- Implement line checking functionality to enhance validation processes for configuration files.

### 📚 Documentation
- Update CHANGELOG with recent changes to reflect new features and improvements.

## [v1.3.0] - 2025-04-24

### ✨ Features
- Configure application flow for improved performance

### 📚 Documentation
- Update CHANGELOG to reflect recent changes

## [v1.3.0-beta.2] - 2025-04-24

### ✨ Features
- Configure automatic changelog generation to streamline release documentation

### 🔧 Maintenance
- Update CHANGELOG with recent changes to reflect the latest project updates
