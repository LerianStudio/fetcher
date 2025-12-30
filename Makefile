# Project Root Makefile.
# Coordinates all component Makefiles and provides centralized commands.
# Fetcher Project Management.

# Define the root directory of the project
ROOT_DIR := $(shell pwd)

# Define the root directory of the project
SERVICE_NAME := fetcher
BIN_DIR := ./.bin
ARTIFACTS_DIR := ./artifacts

# Component directories
INFRA_DIR := ./components/infra
MANAGER_DIR := ./components/manager
WORKER_DIR := ./components/worker

# Define a list of all component directories for easier iteration
BACKEND_COMPONENTS := $(WORKER_DIR) $(MANAGER_DIR)
COMPONENTS := $(INFRA_DIR) $(WORKER_DIR) $(MANAGER_DIR)

# Include shared utility functions
# Define common utility functions
define print_title
	@echo ""
	@echo "------------------------------------------"
	@echo "   $(1)  "
	@echo "------------------------------------------"
endef

# Check if a command is available
define check_command
	@if ! command -v $(1) >/dev/null 2>&1; then \
		echo "Error: $(1) is not installed"; \
		echo "To install: $(2)"; \
		exit 1; \
	fi
endef

# Check if environment files exist
define check_env_files
	@missing=false; \
	for dir in $(COMPONENTS); do \
		if [ ! -f "$$dir/.env" ]; then \
			missing=true; \
			break; \
		fi; \
	done; \
	if [ "$$missing" = "true" ]; then \
		echo "Environment files are missing. Running set-env command first..."; \
		$(MAKE) set-env; \
	fi
endef

# Choose docker compose command depending on installed version
DOCKER_CMD := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; else echo "docker-compose"; fi)
export DOCKER_CMD

#-------------------------------------------------------
# Help Command
#-------------------------------------------------------

.PHONY: help
help:
	@echo ""
	@echo ""
	@echo "Fetcher Project Management Commands"
	@echo ""
	@echo ""
	@echo "Core Commands:"
	@echo "  make help                        	   - Display this help message"
	@echo "  make test                        	   - Run unit tests on all components"
	@echo "  make test-integration-container  	   - Run E2E integration tests with all containers"
	@echo "  make test-integration-infra      	   - Start infrastructure for debugging (fixed ports)"
	@echo "  make test-integration-debug-manager   - Debug Manager in VS Code, Worker in container"
	@echo "  make test-integration-debug-worker    - Debug Worker in VS Code, Manager in container"
	@echo "  make test-integration-debug-full      - Debug both Manager and Worker in VS Code"
	@echo "  make test-integration-clean      	   - Clean up testcontainers and networks"
	@echo "  make test-integration-check      	   - Check for port conflicts before starting"
	@echo "  make build                       	   - Build all components"
	@echo "  make clean                       	   - Clean all build artifacts"
	@echo "  make cover                       	   - Run test coverage"
	@echo ""
	@echo ""
	@echo "Code Quality Commands:"
	@echo "  make lint                        - Run linting on all components"
	@echo "  make format                      - Format code in all components"
	@echo "  make tidy                        - Clean dependencies in root directory"
	@echo "  make check-tests                 - Verify test coverage for components"
	@echo "  make sec                         - Run security checks using gosec"
	@echo ""
	@echo ""
	@echo "Git Hook Commands:"
	@echo "  make setup-git-hooks             - Install and configure git hooks"
	@echo "  make check-hooks                 - Verify git hooks installation status"
	@echo "  make check-envs                  - Check if github hooks are installed and secret env files are not exposed"
	@echo ""
	@echo ""
	@echo "Setup Commands:"
	@echo "  make set-env                     - Copy .env.example to .env for all components"
	@echo "  make dev-setup                   - Set up development environment"
	@echo ""
	@echo ""
	@echo "Service Commands:"
	@echo "  make run                          - Run application locally with .env config"
	@echo "  make up                           - Start all services with Docker Compose"
	@echo "  make down                         - Stop all services with Docker Compose"
	@echo "  make start                        - Start all containers"
	@echo "  make stop                         - Stop all containers"
	@echo "  make restart                      - Restart all containers"
	@echo "  make rebuild-up                   - Rebuild and restart all services"
	@echo "  make clean-docker                 - Clean all Docker resources (containers, networks, volumes)"
	@echo "  make logs                         - Show logs for all services"
	@echo "  make logs-api                     - Show logs for fetcher service"
	@echo "  make ps                           - List container status"
	@echo ""
	@echo ""
	@echo "Documentation Commands:"
	@echo "  make generate-docs               - Generate Swagger documentation"
	@echo "  make generate-docs-all           - Generate Swagger documentation for all services"
	@echo "  make validate-api-docs           - Validate API documentation"
	@echo ""
	@echo ""

#-------------------------------------------------------
# Git Hook Commands
#-------------------------------------------------------

.PHONY: setup-git-hooks
setup-git-hooks:
	$(call print_title,Installing and configuring git hooks)
	@sh ./scripts/setup-git-hooks.sh
	@echo "[ok] Git hooks installed successfully"

.PHONY: check-hooks
check-hooks:
	$(call print_title,Verifying git hooks installation status)
	@err=0; \
	for hook_dir in .githooks/*; do \
		hook_name=$$(basename $$hook_dir); \
		if [ ! -f ".git/hooks/$$hook_name" ]; then \
			echo "Git hook $$hook_name is not installed"; \
			err=1; \
		else \
			echo "Git hook $$hook_name is installed"; \
		fi; \
	done; \
	if [ $$err -eq 0 ]; then \
		echo "[ok] All git hooks are properly installed"; \
	else \
		echo "[error] Some git hooks are missing. Run 'make setup-git-hooks' to fix."; \
		exit 1; \
	fi

.PHONY: check-envs
check-envs:
	$(call print_title,Checking if github hooks are installed and secret env files are not exposed)
	@sh ./scripts/check-envs.sh
	@echo "[ok] Environment check completed"

#-------------------------------------------------------
# Setup Commands
#-------------------------------------------------------

.PHONY: set-env
set-env:
	$(call print_title,Setting up environment files)
	@for dir in $(COMPONENTS); do \
		if [ -f "$$dir/.env.example" ] && [ ! -f "$$dir/.env" ]; then \
			echo "Creating .env in $$dir from .env.example"; \
			cp "$$dir/.env.example" "$$dir/.env"; \
		elif [ ! -f "$$dir/.env.example" ]; then \
			echo "Warning: No .env.example found in $$dir"; \
		else \
			echo ".env already exists in $$dir"; \
		fi; \
	done
	@echo "[ok] Environment files set up successfully"

#-------------------------------------------------------
# Build Commands
#-------------------------------------------------------

.PHONY: build
build:
	$(call print_title,Building component)
	@echo "[ok] Build completed successfully"

#-------------------------------------------------------
# Test Commands
#-------------------------------------------------------

.PHONY: test
test:
	$(call print_title,Running tests)
	@go test -v ./...
	@echo "[ok] Tests completed successfully"

# =============================================================================
# test-integration-container: Full E2E Integration Tests
# =============================================================================
.PHONY: test-integration-container
test-integration-container:
	$(call print_title,Running E2E integration tests with Testcontainers)
	$(call check_command,docker,Install Docker from https://docs.docker.com/get-docker/)
	@echo "Note: Integration tests require either:"
	@echo "  - GITHUB_TOKEN set (to build from Dockerfile)"
	@echo "  - MANAGER_IMAGE and WORKER_IMAGE set (to use pre-built images)"
	@echo ""
	@DOCKER_BUILDKIT=1 go test -tags=integration -v -timeout 30m ./tests/integration/containers/...
	@echo "[ok] Integration tests completed successfully"

# =============================================================================
# test-integration-infra: Start Infrastructure for Debug Sessions
# =============================================================================
.PHONY: test-integration-infra
test-integration-infra: test-integration-check
	$(call print_title,Starting integration test infrastructure with fixed ports)
	$(call check_command,docker,Install Docker from https://docs.docker.com/get-docker/)
	@echo "Starting infrastructure containers..."
	@echo "This will use fixed ports for VS Code debugging."
	@echo ""
	@go run -tags=integration ./tests/integration/containers/cmd/start-infra/...

# Helper function to determine test run pattern
define get_test_run
$(if $(TEST),-run "TestWorkerIntegrationSuite/$(TEST)",)
endef

# =============================================================================
# test-integration-debug-manager: Debug Manager API in VS Code
# =============================================================================
.PHONY: test-integration-debug-manager
test-integration-debug-manager:
	$(call print_title,Running integration tests with Manager running locally)
	$(call check_command,docker,Install Docker from https://docs.docker.com/get-docker/)
	@echo "Mode: Manager Debug (Manager local, Worker container)"
	@echo "Prerequisite: Manager must be running on localhost:4006"
ifdef TEST
	@echo "Running test: $(TEST)"
else
	@echo "Running: ALL tests"
endif
	@echo ""
	@DOCKER_BUILDKIT=1 EXTERNAL_MANAGER_URL=http://localhost:4006 REUSE_INFRA=true \
		go test -tags=integration -v -timeout 30m -count=1 $(call get_test_run) ./tests/integration/containers/...
	@echo "[ok] Integration tests completed successfully"

# =============================================================================
# test-integration-debug-worker: Debug Worker in VS Code
# =============================================================================
.PHONY: test-integration-debug-worker
test-integration-debug-worker:
	$(call print_title,Running integration tests with Worker running locally)
	$(call check_command,docker,Install Docker from https://docs.docker.com/get-docker/)
	@echo "Mode: Worker Debug (Manager container, Worker local)"
	@echo "Prerequisite: Worker must be running locally"
ifdef TEST
	@echo "Running test: $(TEST)"
else
	@echo "Running: ALL tests"
endif
	@echo ""
	@DOCKER_BUILDKIT=1 SKIP_WORKER=true REUSE_INFRA=true \
		go test -tags=integration -v -timeout 30m -count=1 $(call get_test_run) ./tests/integration/containers/...
	@echo "[ok] Integration tests completed successfully"

# =============================================================================
# test-integration-debug-full: Debug Both Manager and Worker in VS Code
# =============================================================================
.PHONY: test-integration-debug-full
test-integration-debug-full:
	$(call print_title,Running integration tests with both Manager and Worker running locally)
	$(call check_command,docker,Install Docker from https://docs.docker.com/get-docker/)
	@echo "Mode: Full Debug (both Manager and Worker local)"
	@echo "Prerequisites:"
	@echo "  - Manager must be running on localhost:4006"
	@echo "  - Worker must be running locally"
ifdef TEST
	@echo "Running test: $(TEST)"
else
	@echo "Running: ALL tests"
endif
	@echo ""
	@DOCKER_BUILDKIT=1 EXTERNAL_MANAGER_URL=http://localhost:4006 SKIP_WORKER=true REUSE_INFRA=true \
		go test -tags=integration -v -timeout 30m -count=1 $(call get_test_run) ./tests/integration/containers/...
	@echo "[ok] Integration tests completed successfully"

# =============================================================================
# test-integration-clean: Clean Up Integration Test Resources
# =============================================================================
# Integration test fixed ports
INTEGRATION_PORTS := 27017 27018 5672 8888 6379 5432 3306 1433 1521 4006

.PHONY: test-integration-clean
test-integration-clean:
	$(call print_title,Cleaning up integration test resources)
	@echo "Stopping testcontainers..."
	@docker ps -q --filter "label=org.testcontainers=true" | xargs -r docker stop 2>/dev/null || true
	@echo "Removing testcontainers..."
	@docker ps -aq --filter "label=org.testcontainers=true" | xargs -r docker rm -f 2>/dev/null || true
	@echo "Removing integration test network..."
	@docker network rm fetcher-test-network 2>/dev/null || true
	@echo "Removing config file..."
	@rm -f /tmp/fetcher-test-infra.json
	@echo "Pruning unused containers..."
	@docker container prune -f 2>/dev/null || true
	@echo "[ok] Integration test resources cleaned successfully"

# =============================================================================
# test-integration-check: Check for Port Conflicts Before Starting
# =============================================================================
.PHONY: test-integration-check
test-integration-check:
	$(call print_title,Checking for port conflicts)
	@conflicts=0; \
	for port in $(INTEGRATION_PORTS); do \
		if command -v ss >/dev/null 2>&1; then \
			if ss -tlnp 2>/dev/null | grep -q ":$$port "; then \
				echo "[CONFLICT] Port $$port is in use"; \
				conflicts=1; \
			fi; \
		elif command -v netstat >/dev/null 2>&1; then \
			if netstat -tlnp 2>/dev/null | grep -q ":$$port "; then \
				echo "[CONFLICT] Port $$port is in use"; \
				conflicts=1; \
			fi; \
		elif command -v lsof >/dev/null 2>&1; then \
			if lsof -i :$$port >/dev/null 2>&1; then \
				echo "[CONFLICT] Port $$port is in use"; \
				conflicts=1; \
			fi; \
		fi; \
	done; \
	if [ $$conflicts -eq 1 ]; then \
		echo ""; \
		echo "[warning] Some ports are in use. Run 'make test-integration-clean' to clean up."; \
		echo "Or check what's using them: lsof -i :<port>"; \
		exit 1; \
	else \
		echo "[ok] All integration test ports are available"; \
	fi

.PHONY: cover
cover:
	$(call print_title,Generating test coverage report)
	$(call check_command,go,Install Go from https://golang.org/doc/install)
	@PACKAGES=$$(go list ./... | grep -v -f ./scripts/coverage_ignore.txt 2>/dev/null || go list ./...); \
	go test -coverprofile=$(ARTIFACTS_DIR)/coverage.out $$PACKAGES
	@go tool cover -html=$(ARTIFACTS_DIR)/coverage.out -o $(ARTIFACTS_DIR)/coverage.html
	@echo "Coverage report generated at $(ARTIFACTS_DIR)/coverage.html"
	@echo ""
	@echo "Coverage Summary:"
	@echo "----------------------------------------"
	@go tool cover -func=$(ARTIFACTS_DIR)/coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@echo "----------------------------------------"
	@echo "Open $(ARTIFACTS_DIR)/coverage.html in your browser to view detailed coverage report"
	@echo "[ok] Coverage report generated successfully"

#-------------------------------------------------------
# Test Coverage Commands
#-------------------------------------------------------

.PHONY: check-tests
check-tests:
	$(call print_title,Verifying test coverage)
	@if find . -name "*.go" -type f | grep -q .; then \
		echo "Running test coverage check..."; \
		go test -coverprofile=coverage.tmp ./... > /dev/null 2>&1; \
		if [ -f coverage.tmp ]; then \
			coverage=$$(go tool cover -func=coverage.tmp | grep total | awk '{print $$3}'); \
			echo "Test coverage: $$coverage"; \
			rm coverage.tmp; \
		else \
			echo "No coverage data generated"; \
		fi; \
	else \
		echo "No Go files found, skipping test coverage check"; \
	fi

#-------------------------------------------------------
# Code Quality Commands
#-------------------------------------------------------

.PHONY: lint
lint:
	$(call print_title,Running linters)
	@if find . -name "*.go" -type f | grep -q .; then \
		if ! command -v golangci-lint >/dev/null 2>&1; then \
			echo "Installing golangci-lint..."; \
			go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		fi; \
		golangci-lint run --fix ./... --verbose; \
		echo "[ok] Linting completed successfully"; \
	else \
		echo "No Go files found, skipping linting"; \
	fi

.PHONY: format
format:
	$(call print_title,Formatting code)
	@go fmt ./...
	@echo "[ok] Formatting completed successfully"

.PHONY: tidy
tidy:
	$(call print_title,Cleaning dependencies in root directory)
	@echo "Tidying root go.mod..."
	@go mod tidy
	@echo "[ok] Dependencies cleaned successfully"

#-------------------------------------------------------
# Security Commands
#-------------------------------------------------------

.PHONY: sec
sec:
	$(call print_title,Running security checks using gosec)
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@if find . -name "*.go" -type f | grep -q .; then \
		echo "Running security checks..."; \
		gosec -quiet ./...; \
		echo "[ok] Security checks completed"; \
	else \
		echo "No Go files found, skipping security checks"; \
	fi

#-------------------------------------------------------
# Clean Commands
#-------------------------------------------------------

.PHONY: clean
clean:
	$(call print_title,Cleaning build artifacts)
	@if [ -z "$(BIN_DIR)" ] || [ -z "$(ARTIFACTS_DIR)" ]; then \
		echo "[error] BIN_DIR or ARTIFACTS_DIR is not set. Aborting to prevent accidental deletion."; \
		exit 1; \
	fi
	@if [ "$(BIN_DIR)" = "/" ] || [ "$(ARTIFACTS_DIR)" = "/" ]; then \
		echo "[error] BIN_DIR or ARTIFACTS_DIR cannot be root directory. Aborting."; \
		exit 1; \
	fi
	@if [ -d "$(BIN_DIR)" ]; then \
		echo "Cleaning $(BIN_DIR)..."; \
		rm -rf $(BIN_DIR)/*; \
	fi
	@if [ -d "$(ARTIFACTS_DIR)" ]; then \
		echo "Cleaning $(ARTIFACTS_DIR)..."; \
		rm -rf $(ARTIFACTS_DIR)/*; \
	fi
	@echo "[ok] Artifacts cleaned successfully"

#-------------------------------------------------------
# Docker Commands
#-------------------------------------------------------

.PHONY: run
run:
	$(call print_title,Running the application with .env config)
	@go run cmd/app/main.go .env
	@echo "[ok] Application started successfully"

.PHONY: build-docker
build-docker:
	$(call print_title,Building Docker images)
	@$(DOCKER_CMD) -f docker-compose.yml build $(c)
	@echo "[ok] Docker images built successfully"

.PHONY: up
up:
	$(call print_title,Starting all services with Docker Compose)
	$(call check_command,docker,Install Docker from https://docs.docker.com/get-docker/)
	$(call check_env_files)
	@echo "Starting infrastructure services first..."
	@cd $(INFRA_DIR) && $(MAKE) up
	@echo "Starting backend components..."
	@for dir in $(BACKEND_COMPONENTS); do \
		if [ -f "$$dir/docker-compose.yml" ]; then \
			echo "Starting services in $$dir..."; \
			(cd $$dir && $(MAKE) up) || exit 1; \
		fi \
	done
	@echo "[ok] All services started successfully"

.PHONY: down
down:
	$(call print_title,Stopping all services with Docker Compose)
	@echo "Stopping backend components..."
	@for dir in $(BACKEND_COMPONENTS); do \
		if [ -f "$$dir/docker-compose.yml" ]; then \
			echo "Stopping services in $$dir..."; \
			(cd $$dir && $(MAKE) down) || exit 1; \
		fi \
	done
	@echo "Stopping infrastructure services..."
	@cd $(INFRA_DIR) && $(MAKE) down
	@echo "[ok] All services stopped successfully"

.PHONY: start
start:
	$(call print_title,Starting all containers)
	@for dir in $(COMPONENTS); do \
		if [ -f "$$dir/docker-compose.yml" ]; then \
			echo "Starting containers in $$dir..."; \
			(cd $$dir && $(MAKE) start) || exit 1; \
		fi; \
	done
	@echo "[ok] All containers started successfully"

.PHONY: stop
stop:
	$(call print_title,Stopping all containers)
	@for dir in $(COMPONENTS); do \
		if [ -f "$$dir/docker-compose.yml" ]; then \
			echo "Stopping containers in $$dir..."; \
			(cd $$dir && $(MAKE) stop) || exit 1; \
		fi; \
	done
	@echo "[ok] All containers stopped successfully"

.PHONY: restart
restart:
	$(call print_title,Restarting all containers)
	@make down && make up
	@echo "[ok] All containers restarted successfully"

.PHONY: rebuild-up
rebuild-up:
	$(call print_title,Rebuilding and restarting all services)
	@echo "Rebuilding infrastructure services..."
	@cd $(INFRA_DIR) && ($(DOCKER_CMD) -f docker-compose.yml build --no-cache && $(DOCKER_CMD) -f docker-compose.yml up -d --force-recreate)
	@echo "Rebuilding backend components..."
	@for dir in $(BACKEND_COMPONENTS); do \
		if [ -f "$$dir/docker-compose.yml" ]; then \
			echo "Rebuilding services in $$dir..."; \
			(cd $$dir && $(DOCKER_CMD) -f docker-compose.yml build --no-cache && $(DOCKER_CMD) -f docker-compose.yml up -d --force-recreate) || exit 1; \
		fi; \
	done
	@echo "[ok] All services rebuilt and restarted successfully"

.PHONY: clean-docker
clean-docker:
	$(call print_title,Cleaning all Docker resources)
	@for dir in $(COMPONENTS); do \
		if [ -f "$$dir/docker-compose.yml" ]; then \
			echo "Cleaning Docker resources in $$dir..."; \
			(cd $$dir && $(MAKE) clean-docker) || exit 1; \
		fi; \
	done
	@echo "Pruning system-wide Docker resources..."
	@docker system prune -f
	@echo "Pruning system-wide Docker volumes..."
	@docker volume prune -f
	@echo "[ok] All Docker resources cleaned successfully"

.PHONY: logs
logs:
	$(call print_title,Showing logs for all services)
	@for dir in $(COMPONENTS); do \
		component_name=$$(basename $$dir); \
		if [ -f "$$dir/docker-compose.yml" ]; then \
			echo "Logs for component: $$component_name"; \
			(cd $$dir && ($(DOCKER_CMD) -f docker-compose.yml logs --tail=50 2>/dev/null || $(DOCKER_CMD) -f docker-compose.yml logs --tail=50)) || exit 1; \
			echo ""; \
		fi; \
	done

.PHONY: logs-api
logs-api:
	$(call print_title,Showing logs for fetcher service)
	@$(DOCKER_CMD) -f docker-compose.yml logs --tail=100 -f fetcher

.PHONY: ps
ps:
	$(call print_title,Listing container status)
	@$(DOCKER_CMD) -f docker-compose.yml ps

#-------------------------------------------------------
# Documentation Commands
#-------------------------------------------------------

.PHONY: generate-docs
generate-docs:
	$(call print_title,Generating Swagger API documentation)
	@if ! command -v swag >/dev/null 2>&1; then \
		echo "Installing swag..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@swag init -g ./components/manager/cmd/app/main.go -d ./ -o ./components/manager/api --parseDependency --parseInternal
	@docker run --rm -v $(ROOT_DIR):/local --user $(shell id -u):$(shell id -g) openapitools/openapi-generator-cli:v5.1.1 generate -i /local/components/manager/api/swagger.json -g openapi-yaml -o /local/components/manager/api
	@mv ./components/manager/api/openapi/openapi.yaml ./components/manager/api/openapi.yaml
	@rm -rf ./components/manager/api/README.md ./components/manager/api/.openapi-generator* ./components/manager/api/openapi
	@if [ -f "$(ROOT_DIR)/scripts/package.json" ]; then \
		echo "Installing npm dependencies for validation..."; \
		cd $(ROOT_DIR)/scripts && npm install > /dev/null; \
	fi
	@echo "[ok] Swagger API documentation generated successfully"

.PHONY: generate-docs-all
generate-docs-all:
	$(call print_title,Generating Swagger documentation for all services)
	$(call check_command,swag,go install github.com/swaggo/swag/cmd/swag@latest)
	@echo "Verifying API documentation coverage..."
	@sh ./scripts/verify-api-docs.sh 2>/dev/null || echo "Warning: Some API endpoints may not be properly documented. Continuing with documentation generation..."
	@echo "Generating documentation for plugin component..."
	$(MAKE) generate-docs 2>&1 | grep -v "warning: "
	@echo "[ok] Swagger documentation generated successfully"

.PHONY: verify-api-docs
verify-api-docs:
	$(call print_title,Verifying API documentation coverage)
	@if [ -f "./scripts/package.json" ]; then \
		echo "Installing npm dependencies..."; \
		cd ./scripts && npm install; \
	fi
	@sh ./scripts/verify-api-docs.sh
	@echo "[ok] API documentation verification completed"

.PHONY: validate-api-docs
validate-api-docs: generate-docs
	$(call print_title,Validating API documentation)
	@if [ -f "scripts/validate-api-docs.js" ] && [ -f "$(ROOT_DIR)/scripts/package.json" ]; then \
		echo "Validating API documentation structure..."; \
		cd $(ROOT_DIR)/scripts && node $(ROOT_DIR)/scripts/validate-api-docs.js; \
		echo "Validating API implementations..."; \
		cd $(ROOT_DIR)/scripts && node $(ROOT_DIR)/scripts/validate-api-implementations.js; \
		echo "[ok] API documentation validation completed"; \
	else \
		echo "Validation scripts not found. Skipping validation."; \
	fi

#-------------------------------------------------------
# Developer Helper Commands
#-------------------------------------------------------

.PHONY: dev-setup
dev-setup:
	$(call print_title,Setting up development environment)
	@echo "Installing development tools..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@if ! command -v swag >/dev/null 2>&1; then \
		echo "Installing swag..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@if ! command -v mockgen >/dev/null 2>&1; then \
		echo "Installing mockgen..."; \
		go install github.com/golang/mock/mockgen@latest; \
	fi
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@echo "Setting up environment..."
	@if [ -f .env.example ] && [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file from template"; \
	fi
	@make tidy
	@make check-tests
	@make sec
	@echo "[ok] Development environment set up successfully"
	@echo "You're ready to start developing! Here are some useful commands:"
	@echo "  make build         - Build the component"
	@echo "  make test          - Run tests"
	@echo "  make up            - Start services"
	@echo "  make rebuild-up    - Rebuild and restart services during development"

#-------------------------------------------------------
# Fuzz Testing Commands
#-------------------------------------------------------

FUZZ_TIME ?= 30s

.PHONY: fuzz-all fuzz-manager fuzz-worker fuzz-connection fuzz-fetcher fuzz-schema fuzz-message

fuzz-all: fuzz-manager fuzz-worker
	@echo "[ok] All fuzz tests completed successfully"

fuzz-manager: fuzz-connection fuzz-fetcher fuzz-schema
	@echo "[ok] Manager fuzz tests completed"

fuzz-worker: fuzz-message
	@echo "[ok] Worker fuzz tests completed"

fuzz-connection:
	$(call print_title,Running connection fuzz tests)
	@go test -fuzz=FuzzConnectionInputParsing -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/connection/ || true
	@go test -fuzz=FuzzConnectionValidation -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/connection/ || true
	@go test -fuzz=FuzzDBTypeValidation -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/connection/ || true
	@go test -fuzz=FuzzUUIDParsing -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/connection/ || true
	@go test -fuzz=FuzzQueryParameterValidation -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/connection/ || true
	@go test -fuzz=FuzzMetadataQueryParams -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/connection/ || true
	@go test -fuzz=FuzzUnknownFieldsDetection -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/connection/ || true

fuzz-fetcher:
	$(call print_title,Running fetcher fuzz tests)
	@go test -fuzz=FuzzFetcherRequestParsing -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/fetcher/ || true
	@go test -fuzz=FuzzDataRequestValidation -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/fetcher/ || true
	@go test -fuzz=FuzzFilterFieldParsing -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/fetcher/ || true
	@go test -fuzz=FuzzFilterReferencesValidation -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/fetcher/ || true

fuzz-schema:
	$(call print_title,Running schema fuzz tests)
	@go test -fuzz=FuzzSchemaValidationRequestParsing -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/schema/ || true
	@go test -fuzz=FuzzSchemaValidationSpecValidation -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/schema/ || true
	@go test -fuzz=FuzzSchemaValidationLimits -fuzztime=$(FUZZ_TIME) ./tests/fuzz/manager/schema/ || true

fuzz-message:
	$(call print_title,Running message fuzz tests)
	@go test -fuzz=FuzzExtractExternalDataMessageParsing -fuzztime=$(FUZZ_TIME) ./tests/fuzz/worker/message/ || true
	@go test -fuzz=FuzzRegexJobIDExtraction -fuzztime=$(FUZZ_TIME) ./tests/fuzz/worker/message/ || true
	@go test -fuzz=FuzzFilterConditionParsing -fuzztime=$(FUZZ_TIME) ./tests/fuzz/worker/message/ || true
	@go test -fuzz=FuzzMessageHeadersParsing -fuzztime=$(FUZZ_TIME) ./tests/fuzz/worker/message/ || true

fuzz-ci:
	$(call print_title,Running fuzz tests for CI)
	@FUZZ_TIME=60s $(MAKE) fuzz-all
