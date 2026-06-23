# Gate 6 dependency map: Fetcher embedded runtime

## Scope

This dependency map distinguishes Engine core dependencies from optional adapters and current service compatibility dependencies. It references the existing project architecture source of truth at `docs/PROJECT_RULES.md` and does not overwrite it.

## Version source notes

`go.mod` declares module `github.com/LerianStudio/fetcher` and Go `1.25.9` (`go.mod:1-3`). `docs/PROJECT_RULES.md` currently lists Go `1.25.6` (`docs/PROJECT_RULES.md:11`). Gate 7 resolves the decision by treating `go.mod` as the source of truth and adding a T-001 dev-cycle subtask to reconcile documentation or tooling before Engine code implementation starts. This pre-dev pass does not modify `docs/PROJECT_RULES.md`.

## Dependency classification

| Dependency | Current version | Engine core | Optional adapter | Current compatibility shell | Notes |
| --- | --- | --- | --- | --- | --- |
| Go standard library | Go 1.25.x | Yes | Yes | Yes | Core should stay mostly standard-library plus explicit ports. |
| `github.com/LerianStudio/lib-commons/v5` | v5.2.0 | Limited | Yes | Yes | Mandatory Lerian library, but avoid dragging service shell concerns into core. Current code uses logging, telemetry, tenant manager, RabbitMQ helpers. |
| `github.com/LerianStudio/lib-auth/v2` | v2.7.0 | No | No | Yes | Auth stays host/Manager adapter concern. |
| `github.com/LerianStudio/lib-license-go/v2` | v2.3.4 | No | No | Yes | License stays host/Manager adapter concern. |
| `github.com/google/uuid` | v1.6.0 | Yes | Yes | Yes | Safe for logical IDs. |
| OpenTelemetry packages | v1.43.0 | Hook/interface only | Yes | Yes | Core should expose hooks and avoid owning exporters/endpoints. |
| `github.com/gofiber/fiber/v2` | v2.52.13 | No | Manager adapter | Yes | HTTP route shape remains outside Engine. |
| `github.com/swaggo/swag` | v1.16.6 | No | Manager adapter | Yes | Swagger generation is compatibility/API shell. |
| `github.com/swaggo/fiber-swagger` | v1.3.0 | No | Manager adapter | Yes | Swagger UI is shell-only. |
| `github.com/rabbitmq/amqp091-go` | v1.10.0 | No | Queue/event adapter | Yes | RabbitMQ must not be mandatory Engine core. |
| `go.mongodb.org/mongo-driver` | v1.17.9 | No — connector adapter only | Mongo adapter | Yes | Current repositories and MongoDB datasource adapter. Core depends on connector contracts, not this driver. |
| `go.mongodb.org/mongo-driver/v2` | v2.5.0 | No — connector adapter only | Mongo adapter | Yes | Existing dependency; Gate 7 should avoid increasing dual-driver coupling in core. Core depends on connector contracts, not this driver. |
| `github.com/redis/go-redis/v9` | v9.18.0 | No | Cache/tenant adapter | Yes | Redis schema cache and tenant event discovery remain optional. |
| `github.com/aws/aws-sdk-go-v2` | v1.41.5 | No | Result sink adapter | Yes | S3-compatible storage adapter support only. |
| `github.com/aws/aws-sdk-go-v2/config` | v1.32.11 | No | Result sink adapter | Yes | S3-compatible storage adapter configuration only. |
| `github.com/aws/aws-sdk-go-v2/credentials` | v1.19.11 | No | Result sink adapter | Yes | S3-compatible storage adapter credentials only. |
| `github.com/aws/aws-sdk-go-v2/service/s3` | v1.96.3 | No | Result sink adapter | Yes | S3-compatible storage adapter client only. |
| SeaweedFS local client | local package | No | Result sink adapter | Yes | SeaweedFS is current default storage adapter, not core. |
| PostgreSQL drivers `github.com/jackc/pgx/v5`, `github.com/lib/pq` | v5.9.2 / v1.12.0 | No — connector adapter only | Connector adapter | Yes | Datasource connector dependency. Core depends on connector contracts, not concrete drivers. |
| MySQL driver `github.com/go-sql-driver/mysql` | v1.9.3 | No — connector adapter only | Connector adapter | Yes | Datasource connector dependency. Core depends on connector contracts, not this driver. |
| Oracle driver `github.com/sijms/go-ora/v2` | v2.9.0 | No — connector adapter only | Connector adapter | Yes | Datasource connector dependency. Core depends on connector contracts, not this driver. |
| SQL Server driver `github.com/microsoft/go-mssqldb` | v1.9.6 | No — connector adapter only | Connector adapter | Yes | Datasource connector dependency. Core depends on connector contracts, not this driver. |
| `github.com/Masterminds/squirrel` | v1.5.4 | No — planner adapter only; core contract requires ADR before adoption | Connector/planner adapter | Yes | Query construction dependency should stay near SQL connectors or an explicit planner adapter. |
| `github.com/DATA-DOG/go-sqlmock` | v1.5.2 | No | Test only | Test only | SQL connector tests only. |
| `github.com/Shopify/toxiproxy/v2` | v2.12.0 | No | Test only | Test only | Chaos/network tests only. |
| `github.com/alicebob/miniredis/v2` | v2.37.0 | No | Test only | Test only | Redis/cache tests only. |
| `github.com/stretchr/testify` | v1.11.1 | No | Test only | Test only | Test assertions only. |
| `github.com/testcontainers/testcontainers-go` | v0.41.0 | No | Test only | Test only | Containerized integration tests only. |
| `github.com/testcontainers/testcontainers-go/modules/mongodb` | v0.41.0 | No | Test only | Test only | MongoDB container tests only. |
| `github.com/testcontainers/testcontainers-go/modules/mssql` | v0.41.0 | No | Test only | Test only | SQL Server container tests only. |
| `github.com/testcontainers/testcontainers-go/modules/mysql` | v0.41.0 | No | Test only | Test only | MySQL container tests only. |
| `github.com/testcontainers/testcontainers-go/modules/postgres` | v0.41.0 | No | Test only | Test only | PostgreSQL container tests only. |
| `github.com/testcontainers/testcontainers-go/modules/rabbitmq` | v0.41.0 | No | Test only | Test only | RabbitMQ container tests only. |
| `github.com/testcontainers/testcontainers-go/modules/redis` | v0.41.0 | No | Test only | Test only | Redis container tests only. |
| `github.com/tryvium-travels/memongo` | v0.12.0 | No | Test only | Test only | MongoDB test harness only. |
| `github.com/ory/dockertest/v3` | Not declared in go.mod | No | Test only if introduced | No | Do not introduce this module without a concrete need and ADR. |

## Engine core dependency boundary

Engine core may depend on:

- Go standard library.
- Stable domain contracts and logical models extracted from current `pkg/model` where appropriate.
- Minimal Lerian shared packages required by standards, with care not to pull host operational concerns into core.
- UUID generation/parsing.
- Connector interfaces and registry contracts.
- Error, limit, tenant context, and observability hook contracts.

Engine core must not depend on:

- Fiber or Swagger.
- lib-auth or lib-license.
- RabbitMQ/amqp091-go.
- MongoDB drivers as mandatory state store dependencies.
- Redis as mandatory cache/event dependency.
- S3/SeaweedFS as mandatory storage dependencies.
- Docker Compose, KEDA, or deployment artifacts.
- Manager or Worker `internal` packages.

## Optional adapter boundary

Optional adapters may depend on:

- MongoDB repositories for connection/execution compatibility.
- Redis schema cache compatibility.
- RabbitMQ queue and event compatibility.
- S3 and SeaweedFS result sink compatibility.
- Fiber, lib-auth, lib-license, Swagger, health/readiness/metrics, and telemetry exporters for Manager compatibility.
- Current SQL/Mongo datasource drivers for connector implementations.

Adapters must depend inward on Engine contracts. Engine must not depend outward on adapters. If this rule is violated, embedded mode is fake. A library that imports RabbitMQ to avoid deploying RabbitMQ is comedy, just not the useful kind.

## Current service dependency anchors

Manager shell dependencies:

- Fiber/lib-auth/lib-license/telemetry middleware are mounted in current routes (`components/manager/internal/adapters/http/in/routes.go:23-51`, `components/manager/internal/adapters/http/in/routes.go:76-92`).
- Manager bootstrap wires MongoDB, RabbitMQ, Redis/schema cache, auth, license, readyz, tenant manager, and service commands/queries (`components/manager/internal/bootstrap/config.go:529-547`, `components/manager/internal/bootstrap/config.go:550-706`).

Worker shell dependencies:

- Worker `UseCase` carries storage, job repository, connection repository, cryptor, document signer, RabbitMQ publisher, datasource factory, and resolver dependencies (`components/worker/internal/services/service.go:16-59`).
- Worker extraction currently handles job repository status, storage encryption/write, and optional notifications (`components/worker/internal/services/extract_data.go:51-137`, `components/worker/internal/services/extract_data.go:495-575`, `components/worker/internal/services/job_notification.go:78-95`).

Datasource dependencies:

- Connector interface currently includes `Connect`, `Close`, `Query`, and `GetSchemaInfo` (`pkg/model/datasource/datasource-config.go:28-50`).
- Factory currently switches over MongoDB, PostgreSQL, Oracle, MySQL, SQL Server and resolves passwords (`pkg/datasource/datasource_factory.go:35-116`).
- Factory can perform early connection side effects and concrete repository construction (`pkg/datasource/datasource_factory.go:139-185`, `pkg/datasource/datasource_factory.go:238-260`). Gate 7 should separate construction from explicit test/connect operations.

Storage dependencies:

- Storage port is minimal `Get`/`Put` (`pkg/ports/storage/repository.go:8-14`).
- Provider factory selects SeaweedFS or S3-compatible storage (`pkg/storage/factory.go:20-91`).
- SeaweedFS and S3 adapters apply tenant-scoped object keys through lib-commons tenant-manager S3 context (`pkg/seaweedfs/external/external_data.go:42-55`, `pkg/storage/s3.go:137-156`).

Resolver dependencies:

- Resolver separates internal datasource resolution from external connection repository lookup (`pkg/resolver/resolver.go:9-37`).
- Internal datasource registry is currently finite and hardcoded (`pkg/resolver/registry.go:30-45`).
- Single-tenant resolver resolves internal datasource env connections and external repository connections (`pkg/resolver/single_tenant.go:11-69`).

## Compatibility requirements

Current Manager and Worker must remain optional compatibility adapters over the Engine.

Current HTTP API behavior should be preserved unless Gate 7 explicitly marks a behavior for migration.

Current queue payloads, job status values, result path/HMAC semantics, notification routing, and storage encryption behavior should be preserved for standalone service mode.

Current tests in manager/worker services plus E2E/fuzz suites are regression assets. Gate 7 should add Engine-level tests rather than replacing these suites.

## Gate 7 dependency requirements

- Add a dependency rule proving Engine core does not import `components/*/internal`.
- Add a dependency rule proving Engine core does not import Fiber, RabbitMQ, MongoDB, Redis, S3, SeaweedFS, lib-auth, or lib-license packages.
- Define exact package names for Engine contracts and adapters.
- Reconcile Go version documentation or tooling before code implementation starts, using `go.mod` Go `1.25.9` as the source of truth.
- Decide whether current `pkg/model` types are moved, aliased, or wrapped for Engine contracts.
- Decide whether connector implementations live in Engine-adjacent adapter packages or remain under current datasource packages with compatibility shims.
