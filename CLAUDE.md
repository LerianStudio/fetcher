# CLAUDE.md

Lerian Fetcher is an enterprise data extraction microservice that unifies access to PostgreSQL, MySQL, Oracle, SQL Server, and MongoDB.

- **Module:** `github.com/LerianStudio/fetcher`
- **Go version:** Source of truth is `go.mod`; do not rely on stale toolchain guidance elsewhere.
- **Architecture:** Hexagonal Architecture + CQRS
- **Services:** Manager (HTTP API, Fiber, port 4006) and Worker (RabbitMQ consumer)

## Commands

### Setup

```bash
make set-env              # Copy .env.example to .env for all components
make generate-master-key  # Generate APP_ENC_KEY (REQUIRED before services start)
make dev-setup            # Install tools (golangci-lint, swag, mockgen, gosec) + tidy
```

### Build & Run

```bash
make build       # Build all components
make up          # Start all services with Docker Compose
make down        # Stop all services
make rebuild-up  # Rebuild and restart all services
```

### Test

```bash
make test-unit   # Unit tests
make test-e2e    # E2E tests (requires Docker, uses -tags=e2e)
make test-fuzzy  # Fuzz tests
make test-chaos  # Chaos tests with Toxiproxy (uses -tags=chaos)
make test-bench  # Benchmarks (BENCH=pattern BENCH_PKG=./path)
make test-all    # All suites sequentially
```

### Code Quality

```bash
make lint           # golangci-lint with --fix (28 linters, max cyclomatic complexity 18)
make format         # go fmt
make tidy           # go mod tidy
make sec            # gosec security analysis
make generate-docs  # Regenerate Swagger docs (swag init)
```

## Architecture

### Manager (`components/manager/`)

HTTP API server on Fiber framework, port 4006.

- Entry point: `components/manager/cmd/app/main.go`
- HTTP handlers: `internal/adapters/http/in/`
- CQRS commands: `internal/services/command/` (Create, Update, Delete)
- CQRS queries: `internal/services/query/` (Get, List, Test, Validate)
- Schema cache: `internal/adapters/cache/` (Redis)
- DI and config: `internal/bootstrap/`

### Worker (`components/worker/`)

Async RabbitMQ consumer. Does NOT follow CQRS - uses a single `UseCase` struct in `internal/services/`.

- Entry point: `components/worker/cmd/app/main.go`
- Consumer: `internal/adapters/rabbitmq/`
- Extraction logic: `internal/services/`

### Shared Packages (`pkg/`)

- `pkg/model/` - domain entities (Connection, Job, Schema)
- `pkg/ports/` - repository interfaces in subdirectories: `connection/`, `job/`, `cache/`, `publisher/`, `storage/`, `messaging/`, `datasource/`
- `pkg/datasource/datasource_factory.go` - factory for multi-DB support (switch on connection type)
- `pkg/crypto/` - AES-256-GCM encryption, HMAC-SHA256 signing
- `pkg/errors.go` - custom error types (ValidationError, UnauthorizedError, ForbiddenError, etc.)
- `pkg/context.go` - Logger/Tracer injection via context
- DB adapters: `pkg/postgres/`, `pkg/mysql/`, `pkg/oracle/`, `pkg/sqlserver/`, `pkg/mongodb/`
- Infrastructure: `pkg/rabbitmq/`, `pkg/redis/`, `pkg/seaweedfs/`, `pkg/ratelimit/`

## Code Conventions

**CQRS services (Manager only):** Struct with `Execute()` method. Constructor `NewXxx(deps...)`. Dependencies are port interfaces, never concrete implementations.

**HTTP handlers:** Thin handlers in `internal/adapters/http/in/` with Swagger annotations (swaggo). Extract org ID via `httpUtils.GetOrganizationID(c)`, product name from `X-Product-Name` header.

**Error handling:** Use `pkg.ValidateBusinessError(constant.ErrXxx, "entityType", args...)` for business errors. Custom types in `pkg/errors.go` map to HTTP status codes.

**Context propagation:** Use `commons.NewTrackingFromContext(ctx)` for logger, tracer, requestID. Start OTel spans: `ctx, span := tracer.Start(ctx, "service.op"); defer span.End()`.

**Mocking:** `go.uber.org/mock` with `mockgen`. Mock files are `*.mock.go` alongside interfaces. Interfaces have `//go:generate mockgen` directives.

**Unit tests:** File alongside source (`xxx_test.go`). Use `testutil.TestContext()` for context, `gomock.NewController(t)` for mocks, `testify/assert` or `testify/require` for assertions.

**E2E tests:** Build tag `//go:build e2e`, in `tests/e2e/`. Each test must generate a unique product name via `e2eshared.GenerateProductName()` for isolation. Use `e2eshared.CreateTestConnection()` and `e2eshared.AssertJobCompleted()`.

**Fuzz tests:** In `tests/fuzz/`. **Chaos tests:** Build tag `//go:build chaos`, in `tests/chaos/`.

## Git Conventions

- Conventional commits: `type(scope): description`
- Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`
- Emojis are added automatically by `.githooks/commit-msg` - do NOT add them manually
- Branches: `feature/`, `fix/`, `hotfix/`, `docs/`, `refactor/`, `build/`, `test/`
- Protected branches: `main`, `develop`, `release-candidate`

## Gotchas

1. **Master key required:** Services will NOT start without `APP_ENC_KEY` in both `components/manager/.env` and `components/worker/.env`. Run `make generate-master-key` first.
2. **Build tags for tests:** E2E requires `-tags=e2e`, Chaos requires `-tags=chaos`. Without the tag, zero tests are found.
3. **Product name isolation in E2E:** Each test must use `e2eshared.GenerateProductName()` and pass it as `X-Product-Name` header for connections and `metadata.source` for jobs.
4. **Worker is NOT CQRS:** Uses a single `UseCase` struct, unlike the Manager.
5. **Ports are subdirectories:** Interfaces live in `pkg/ports/connection/`, `pkg/ports/job/`, etc. - not in a single file.
6. **Strict linting:** Max cyclomatic complexity 18, `io/ioutil` banned, strict shadow detection.
7. **`.env` files are gitignored:** Only `.env.example` files are committed.
8. **Distributed Docker Compose:** Infrastructure in `components/infra/docker-compose.yml`, Manager and Worker have their own.
9. **E2E requires GITHUB_TOKEN** for building Docker images with private dependencies. Use `E2E_SKIP_BUILD=true` to skip build.

## Documentation Index

| File | Content |
|------|---------|
| `docs/PROJECT_RULES.md` | Architecture rules, patterns, project structure (~1200 lines) |
| `README.md` | Project overview, quick start, API endpoints, security model |
| `tests/e2e/README.md` | E2E test conventions, patterns, test catalog |
| `components/manager/api/swagger.yaml` | OpenAPI specification |
| `components/manager/api/requests.http` | API request examples for manual testing |
| `.golangci.yml` | Full linter configuration (28 linters) |

## Common Workflows

### Adding a new API endpoint

1. Create handler in `components/manager/internal/adapters/http/in/` with Swagger annotations
2. Register route in the routes file
3. Create command or query service in `internal/services/command/` or `internal/services/query/`
4. Wire dependencies in `internal/bootstrap/`
5. Run `make generate-docs`

### Adding a new database adapter

1. Create package in `pkg/<dbtype>/`
2. Create model in `pkg/model/datasource/<dbtype>/`
3. Add case to the switch in `pkg/datasource/datasource_factory.go`
4. Add type constant in `pkg/constant/`

### Writing a unit test

1. Create `xxx_test.go` alongside the source file
2. Use `testutil.TestContext()` for context with logger/tracer
3. Use `gomock.NewController(t)` + mock constructors from `*.mock.go`
4. Assert with `testify/assert` or `testify/require`

### Writing an E2E test

1. Create file in `tests/e2e/` with `//go:build e2e` tag
2. Call `t.Parallel()` first
3. Generate product name: `productName := e2eshared.GenerateProductName()`
4. Use `e2eshared.CreateTestConnection(t, ...)` for auto-cleanup
5. Use `e2eshared.AssertJobCompleted(t, ...)` for job polling
