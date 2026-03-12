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

# Coverage configuration
# PKG: specific package to test (e.g., PKG=./components/manager/...)
PKG ?=
COVERAGE_DIR ?= $(ARTIFACTS_DIR)

# Test configuration
# RETRY_ON_FAIL: retry tests once on failure (useful for CI)
RETRY_ON_FAIL ?= 0
GOTESTSUM := $(shell command -v gotestsum 2>/dev/null)

# macOS linker compatibility (fixes ld warnings on macOS)
ifeq ($(shell uname),Darwin)
    GO_TEST_LDFLAGS := -ldflags="-linkmode=external -extldflags=-ld_classic"
else
    GO_TEST_LDFLAGS :=
endif

# Benchmark configuration
# BENCH: specific benchmark pattern (e.g., BENCH=BenchmarkCache)
# BENCH_PKG: specific package to benchmark (e.g., BENCH_PKG=./pkg/redis/...)
BENCH ?= .
BENCH_PKG ?= ./...

# E2E test configuration
# GITHUB_TOKEN: GitHub token for building images with private dependencies
# E2E_SKIP_BUILD: Skip Docker build, use pre-built images (default: true)
# MANAGER_IMAGE: Docker image for Manager container (default: fetcher-manager:latest)
# WORKER_IMAGE: Docker image for Worker container (default: fetcher-worker:latest)
GITHUB_TOKEN ?=
E2E_SKIP_BUILD ?= true
MANAGER_IMAGE ?= fetcher-manager:latest
WORKER_IMAGE ?= fetcher-worker:latest
export GITHUB_TOKEN
export E2E_SKIP_BUILD
export MANAGER_IMAGE
export WORKER_IMAGE

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
	@echo "  make help                              - Display this help message"
	@echo "  make test-unit                         - Run unit tests on all components"
	@echo "  make build                             - Build all components"
	@echo "  make clean                             - Clean all build artifacts"
	@echo "  make coverage-unit                     - Run unit tests with coverage (PKG=./path, RETRY_ON_FAIL=1)"
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
	@echo "Test Suite Aliases:"
	@echo "  make test-all                         - Run all tests sequentially (unit, fuzzy, e2e, chaos) (~25m duration)"
	@echo "  make test-unit                        - Run unit tests on all components (~1m duration)"
	@echo "  make test-fuzzy                       - Run fuzz tests on all components (~3m duration)"
	@echo "  make test-e2e                         - Run E2E tests (~2m duration)"
	@echo "  make test-chaos                       - Run chaos tests (~20m duration)"
	@echo "  make test-bench                       - Run benchmark tests (BENCH=pattern, BENCH_PKG=./path) (~1m duration)"
	@echo ""
	@echo ""
	@echo "Coverage Commands:"
	@echo "  make coverage-unit                    - Run unit tests with coverage (PKG=./path, RETRY_ON_FAIL=1) (~1m duration)"
	@echo ""
	@echo ""
	@echo "Cryptographic Utility Commands:"
	@echo "  make derive-key KEY=...          - Derive external HMAC key from master key"
	@echo "  make generate-master-key         - Generate a new cryptographically secure master key"
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

.PHONY: test-unit
test-unit: ## Run unit tests on all components
	$(call print_title,Running unit tests)
	@start_time=$$(date +%s); \
	PACKAGES=$$(go list ./... | grep -v -f ./scripts/coverage_ignore.txt 2>/dev/null || go list ./...); \
	go test -v $$PACKAGES; \
	test_exit_code=$$?; \
	end_time=$$(date +%s); \
	elapsed=$$((end_time - start_time)); \
	minutes=$$((elapsed / 60)); \
	seconds=$$((elapsed % 60)); \
	echo ""; \
	echo "=========================================="; \
	echo "  UNIT TEST REPORT"; \
	echo "=========================================="; \
	echo "  Duration: $${minutes}m $${seconds}s"; \
	if [ $$test_exit_code -eq 0 ]; then \
	  echo "  Status:   PASS"; \
	  echo "=========================================="; \
	else \
	  echo "  Status:   FAIL"; \
	  echo "=========================================="; \
	  exit $$test_exit_code; \
	fi

.PHONY: test-all
test-all: ## Run all tests sequentially (unit, fuzzy, e2e, chaos)
	$(call print_title,Running all tests sequentially)
	@mkdir -p $(ARTIFACTS_DIR); \
	total_start=$$(date +%s); \
	unit_status="SKIP"; unit_time=0; \
	fuzzy_status="SKIP"; fuzzy_time=0; \
	e2e_status="SKIP"; e2e_time=0; \
	chaos_status="SKIP"; chaos_time=0; \
	failed=""; \
	echo ""; \
	echo "=== [1/4] Running Unit Tests ==="; \
	start=$$(date +%s); \
	if $(MAKE) test-unit 2>&1 | tee $(ARTIFACTS_DIR)/test-unit.log; then \
	  unit_status="PASS"; \
	else \
	  unit_status="FAIL"; \
	  failed="$$failed unit"; \
	fi; \
	unit_time=$$(($$(date +%s) - start)); \
	echo ""; \
	echo "=== [2/4] Running Fuzz Tests ==="; \
	start=$$(date +%s); \
	if $(MAKE) test-fuzzy 2>&1 | tee $(ARTIFACTS_DIR)/test-fuzzy.log; then \
	  fuzzy_status="PASS"; \
	else \
	  fuzzy_status="FAIL"; \
	  failed="$$failed fuzzy"; \
	fi; \
	fuzzy_time=$$(($$(date +%s) - start)); \
	echo ""; \
	echo "=== [3/4] Running E2E Tests ==="; \
	start=$$(date +%s); \
	if $(MAKE) test-e2e 2>&1 | tee $(ARTIFACTS_DIR)/test-e2e.log; then \
	  e2e_status="PASS"; \
	else \
	  e2e_status="FAIL"; \
	  failed="$$failed e2e"; \
	fi; \
	e2e_time=$$(($$(date +%s) - start)); \
	echo ""; \
	echo "=== [4/4] Running Chaos Tests ==="; \
	start=$$(date +%s); \
	if $(MAKE) test-chaos 2>&1 | tee $(ARTIFACTS_DIR)/test-chaos.log; then \
	  chaos_status="PASS"; \
	else \
	  chaos_status="FAIL"; \
	  failed="$$failed chaos"; \
	fi; \
	chaos_time=$$(($$(date +%s) - start)); \
	total_time=$$(($$(date +%s) - total_start)); \
	total_min=$$((total_time / 60)); \
	total_sec=$$((total_time % 60)); \
	echo ""; \
	echo ""; \
	echo "=========================================================="; \
	echo "                   FULL TEST REPORT                       "; \
	echo "=========================================================="; \
	echo ""; \
	echo "  Test Suite        Status    Duration"; \
	echo "  ----------------  --------  --------"; \
	printf "  %-16s  %-8s  %4ds\n" "Unit" "$$unit_status" "$$unit_time"; \
	printf "  %-16s  %-8s  %4ds\n" "Fuzzy" "$$fuzzy_status" "$$fuzzy_time"; \
	printf "  %-16s  %-8s  %4ds\n" "E2E" "$$e2e_status" "$$e2e_time"; \
	printf "  %-16s  %-8s  %4ds\n" "Chaos" "$$chaos_status" "$$chaos_time"; \
	echo "  ----------------  --------  --------"; \
	printf "  %-16s  %8s  %dm %ds\n" "TOTAL" "" "$$total_min" "$$total_sec"; \
	echo ""; \
	if [ -n "$$failed" ]; then \
	  echo "  FAILED SUITES:$$failed"; \
	  echo ""; \
	  echo "  Failure Details:"; \
	  echo "  ----------------"; \
	  for suite in $$failed; do \
	    echo ""; \
	    echo "  [$${suite}] Last 20 lines:"; \
	    tail -20 $(ARTIFACTS_DIR)/test-$${suite}.log 2>/dev/null | sed 's/^/    /' || echo "    (log not available)"; \
	  done; \
	  echo ""; \
	  echo "  Full logs: $(ARTIFACTS_DIR)/test-*.log"; \
	  echo ""; \
	  echo "=========================================================="; \
	  echo "  RESULT: FAILED                                          "; \
	  echo "=========================================================="; \
	  exit 1; \
	else \
	  echo "=========================================================="; \
	  echo "  RESULT: ALL TESTS PASSED                                "; \
	  echo "=========================================================="; \
	fi

# =============================================================================
# E2E Testing Commands
# =============================================================================

# E2E tests
# Run end-to-end tests with the full application stack.
# All parameters are optional. Default images: fetcher-manager:latest, fetcher-worker:latest
# Usage:
#   make test-e2e                                         							  # Use default pre-built images
#   make test-e2e E2E_SKIP_BUILD=false GITHUB_TOKEN=`cat .secrets/github_token.txt`   # Build images with private deps
#   make test-e2e MANAGER_IMAGE=xxx WORKER_IMAGE=yyy      							  # Use custom images
.PHONY: test-e2e
test-e2e: ## Run E2E tests
	$(call print_title,Running E2E tests)
	$(call check_command,docker,Install Docker from https://docs.docker.com/get-docker/)
	@start_time=$$(date +%s); \
	go test -v -tags=e2e -timeout 30m -count=1 ./tests/e2e/...; \
	test_exit_code=$$?; \
	end_time=$$(date +%s); \
	elapsed=$$((end_time - start_time)); \
	minutes=$$((elapsed / 60)); \
	seconds=$$((elapsed % 60)); \
	echo ""; \
	echo "=========================================="; \
	echo "  E2E TEST REPORT"; \
	echo "=========================================="; \
	echo "  Duration: $${minutes}m $${seconds}s"; \
	if [ $$test_exit_code -eq 0 ]; then \
	  echo "  Status:   PASS"; \
	  echo "=========================================="; \
	else \
	  echo "  Status:   FAIL"; \
	  echo "=========================================="; \
	  exit $$test_exit_code; \
	fi

# Unit tests with coverage (uses covermode=atomic)
# Supports PKG parameter to filter packages (e.g., PKG=./components/manager/...)
# Supports .ignorecoverunit file to exclude patterns from coverage stats
# Supports GOTESTSUM for better test output (auto-detected)
# Supports RETRY_ON_FAIL=1 for retry on failure
.PHONY: coverage-unit
coverage-unit: ## Run unit tests with coverage report
	$(call print_title,Running unit tests with coverage)
	$(call check_command,go,Install Go from https://golang.org/doc/install)
	@set -e; mkdir -p $(COVERAGE_DIR); \
	if [ -n "$(PKG)" ]; then \
	  echo "Using specified package: $(PKG)"; \
	  pkgs=$$(go list $(PKG) 2>/dev/null | grep -v -f ./scripts/coverage_ignore.txt 2>/dev/null | tr '\n' ' ' || go list $(PKG)); \
	else \
	  pkgs=$$(go list ./... | grep -v -f ./scripts/coverage_ignore.txt 2>/dev/null || go list ./...); \
	fi; \
	if [ -z "$$pkgs" ]; then \
	  echo "No packages found"; \
	else \
	  echo "Packages: $$pkgs"; \
	  if [ -n "$(GOTESTSUM)" ]; then \
	    echo "Running unit tests with gotestsum (coverage enabled)"; \
	    gotestsum --format testname -- -v -race -count=1 $(GO_TEST_LDFLAGS) -covermode=atomic -coverprofile=$(COVERAGE_DIR)/coverage.out $$pkgs || { \
	      if [ "$(RETRY_ON_FAIL)" = "1" ]; then \
	        echo "Retrying unit tests once..."; \
	        gotestsum --format testname -- -v -race -count=1 $(GO_TEST_LDFLAGS) -covermode=atomic -coverprofile=$(COVERAGE_DIR)/coverage.out $$pkgs; \
	      else \
	        exit 1; \
	      fi; \
	    }; \
	  else \
	    go test -v -race -count=1 $(GO_TEST_LDFLAGS) -covermode=atomic -coverprofile=$(COVERAGE_DIR)/coverage.out $$pkgs || { \
	      if [ "$(RETRY_ON_FAIL)" = "1" ]; then \
	        echo "Retrying unit tests once..."; \
	        go test -v -race -count=1 $(GO_TEST_LDFLAGS) -covermode=atomic -coverprofile=$(COVERAGE_DIR)/coverage.out $$pkgs; \
	      else \
	        exit 1; \
	      fi; \
	    }; \
	  fi; \
	  if [ -f .ignorecoverunit ]; then \
	    echo "Filtering coverage with .ignorecoverunit patterns..."; \
	    patterns=$$(grep -v '^#' .ignorecoverunit | grep -v '^$$' | tr '\n' '|' | sed 's/|$$//'); \
	    if [ -n "$$patterns" ]; then \
	      regex_patterns=$$(echo "$$patterns" | sed 's/\./\\./g' | sed 's/\*/.*/g'); \
	      head -1 $(COVERAGE_DIR)/coverage.out > $(COVERAGE_DIR)/coverage_filtered.out; \
	      tail -n +2 $(COVERAGE_DIR)/coverage.out | grep -vE "$$regex_patterns" >> $(COVERAGE_DIR)/coverage_filtered.out || true; \
	      mv $(COVERAGE_DIR)/coverage_filtered.out $(COVERAGE_DIR)/coverage.out; \
	      echo "Excluded patterns: $$patterns"; \
	    fi; \
	  fi; \
	  go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html; \
	  echo ""; \
	  echo "Coverage Summary:"; \
	  echo "----------------------------------------"; \
	  go tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total | awk '{print "Total coverage: " $$3}'; \
	  echo "----------------------------------------"; \
	  echo "Open $(COVERAGE_DIR)/coverage.html in your browser to view detailed coverage report"; \
	fi
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
			grep -v ".mock.go" coverage.tmp > coverage_filtered.tmp; \
			coverage=$$(go tool cover -func=coverage_filtered.tmp | grep total | awk '{print $$3}'); \
			echo "Test coverage (excluding .mock.go files): $$coverage"; \
			rm coverage.tmp coverage_filtered.tmp; \
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
		go install go.uber.org/mock/mockgen@latest; \
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

.PHONY: test-fuzzy fuzz-manager fuzz-worker fuzz-connection fuzz-fetcher fuzz-schema fuzz-message

test-fuzzy: ## Run fuzz tests on all components
	$(call print_title,Running fuzz tests)
	@start_time=$$(date +%s); \
	$(MAKE) fuzz-manager fuzz-worker; \
	test_exit_code=$$?; \
	end_time=$$(date +%s); \
	elapsed=$$((end_time - start_time)); \
	minutes=$$((elapsed / 60)); \
	seconds=$$((elapsed % 60)); \
	echo ""; \
	echo "=========================================="; \
	echo "  FUZZ TEST REPORT"; \
	echo "=========================================="; \
	echo "  Duration: $${minutes}m $${seconds}s"; \
	if [ $$test_exit_code -eq 0 ]; then \
	  echo "  Status:   PASS"; \
	  echo "=========================================="; \
	else \
	  echo "  Status:   FAIL"; \
	  echo "=========================================="; \
	  exit $$test_exit_code; \
	fi

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
	@FUZZ_TIME=60s $(MAKE) test-fuzzy

#-------------------------------------------------------
# Chaos Testing Commands
#-------------------------------------------------------

# Chaos tests
# Run chaos engineering tests with fault injection via Toxiproxy.
# All parameters are optional. Default images: fetcher-manager:latest, fetcher-worker:latest
# Usage:
#   make test-chaos                                         						    # Use default pre-built images
#   make test-chaos E2E_SKIP_BUILD=false GITHUB_TOKEN=`cat .secrets/github_token.txt`   # Build images with private deps
#   make test-chaos MANAGER_IMAGE=xxx WORKER_IMAGE=yyy       						    # Use custom images
.PHONY: test-chaos
test-chaos: ## Run chaos tests
	$(call print_title,Running chaos tests)
	$(call check_command,docker,Install Docker from https://docs.docker.com/get-docker/)
	@start_time=$$(date +%s); \
	go test -v -tags=chaos -timeout 45m -count=1 ./tests/chaos/...; \
	test_exit_code=$$?; \
	end_time=$$(date +%s); \
	elapsed=$$((end_time - start_time)); \
	minutes=$$((elapsed / 60)); \
	seconds=$$((elapsed % 60)); \
	echo ""; \
	echo "=========================================="; \
	echo "  CHAOS TEST REPORT"; \
	echo "=========================================="; \
	echo "  Duration: $${minutes}m $${seconds}s"; \
	if [ $$test_exit_code -eq 0 ]; then \
	  echo "  Status:   PASS"; \
	  echo "=========================================="; \
	else \
	  echo "  Status:   FAIL"; \
	  echo "=========================================="; \
	  exit $$test_exit_code; \
	fi

#-------------------------------------------------------
# Benchmark Testing Commands
#-------------------------------------------------------

# Benchmark tests
# Run performance benchmarks for critical code paths.
# Usage:
#   make test-bench                          # Run all benchmarks
#   make test-bench BENCH=BenchmarkCache     # Run specific benchmark pattern
#   make test-bench BENCH_PKG=./pkg/redis/...  # Run benchmarks in specific package
.PHONY: test-bench
test-bench: ## Run benchmark tests
	$(call print_title,Running benchmark tests)
	$(call check_command,go,Install Go from https://golang.org/doc/install)
	@echo "Benchmark pattern: $(BENCH)"
	@echo "Package: $(BENCH_PKG)"
	@go test -bench=$(BENCH) -benchmem -run=^$$ $(BENCH_PKG)
	@echo "[ok] Benchmark tests completed"

.PHONY: test-chaos-verbose
test-chaos-verbose: ## Run chaos tests with verbose output
	$(call print_title,Running chaos tests with verbose output)
	$(call check_command,docker,Install Docker from https://docs.docker.com/get-docker/)
	@echo "Running chaos tests with verbose output..."
	@go test -v -tags=chaos -timeout 45m -count=1 ./tests/chaos/...
	@echo "[ok] Chaos tests completed successfully"

#-------------------------------------------------------
# Cryptographic Utility Commands
#-------------------------------------------------------

.PHONY: derive-key
derive-key: ## Derive external HMAC key from master key
	$(call print_title,Deriving external HMAC key)
ifndef KEY
ifndef APP_ENC_KEY
	@echo "Error: KEY or APP_ENC_KEY is required"
	@echo ""
	@echo "Usage:"
	@echo "  make derive-key KEY=\"YOUR_BASE64_MASTER_KEY\""
	@echo "  APP_ENC_KEY=\"YOUR_BASE64_MASTER_KEY\" make derive-key"
	@echo ""
	@echo "Example:"
	@echo "  make derive-key KEY=\"dGhpcy1pcy1hLTMyLWJ5dGUtbWFzdGVyLWtleTEyMzQ=\""
	@echo ""
	@echo "See docs/security/verification-guide.md for more information."
	@exit 1
endif
	@go run ./scripts/crypto/derive-key/main.go
else
	@APP_ENC_KEY="$(KEY)" go run ./scripts/crypto/derive-key/main.go
endif

.PHONY: generate-master-key
generate-master-key: ## Generate a new cryptographically secure master key
	$(call print_title,Generating new master key)
	@echo "Generating a cryptographically secure 32-byte master key..."
	@KEY=$$(head -c 32 /dev/urandom | base64) && \
	echo "" && \
	echo "New Master Key (base64):" && \
	echo "$$KEY" && \
	echo "" && \
	echo "IMPORTANT: Store this key securely. It cannot be recovered if lost." && \
	echo "Add to your environment as APP_ENC_KEY."
