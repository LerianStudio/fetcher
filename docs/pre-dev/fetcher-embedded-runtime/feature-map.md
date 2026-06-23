# Gate 2 Feature Map: Fetcher Embedded Runtime

## Map Purpose

This map describes business capabilities, domains, journeys, and integration points for the embedded runtime. It separates business capabilities from technical integration constraints and intentionally avoids package layout except where needed to identify current compatibility boundaries.

## Business Capability Domains

| Domain | Capability | Engine Ownership | Host Ownership |
| --- | --- | --- | --- |
| Datasource setup | Define and manage external datasource connections | Connection semantics, validation, credential protection, datasource type rules | Persistence choice, product policy, access control, API route shape |
| Source understanding | Discover database schemas | Connector-owned schema discovery and canonical schema model | Scheduling, caching backend, API presentation |
| Request validation | Validate requested tables, fields, and filters | Validation rules, limits, datasource conventions, error taxonomy | User-facing error mapping and authorization |
| Extraction planning | Convert requested fields/filters into executable work | Plan model, datasource grouping, limits, tenant-safe scope | Execution mode decision and operational scheduling |
| Extraction execution | Run datasource queries | Connector behavior, execution runner, result normalization | Runtime lifecycle, cancellation policy, host observability sinks |
| Result handling | Return or store extracted data | Canonical result model, result sink contract, integrity metadata | Storage backend choice, object path policy, retention policy |
| Execution tracking | Track extraction state | Lightweight execution model and status transitions | Durable job store, async queue, polling APIs, event subscriptions |
| Safety and governance | Protect credentials, tenants, and client data | Credential semantics, no-secret logging discipline, tenant/product primitives | Auth, license, deployment boundary, policy enforcement |
| Compatibility | Keep existing Fetcher service behavior | Shared Engine behavior | Manager HTTP adapter, Worker queue adapter, existing infra topology |

## Core User Journeys

| Journey | Primary Actor | Outcome |
| --- | --- | --- |
| Embed Engine | Host app engineering team | Host app imports Fetcher capabilities and configures required ports/adapters in-process. |
| Register connection | Product backend | Connection is validated, credentials are protected, and the host-selected store records it. |
| Discover schema | Product backend | Host app requests schema for a connection and receives canonical table/field data. |
| Validate extraction request | Product backend | Requested mapped fields and filters are checked before execution. |
| Run synchronous extraction | Product backend | Host app calls Engine and receives canonical result data or a result sink reference. |
| Run asynchronous extraction | Product backend or adapter | Host app schedules execution through its own queue/state model while reusing Engine execution semantics. |
| Keep standalone compatibility | Fetcher operators | Existing Manager and Worker behavior continues through adapters over the Engine. |

## Capability Relationships

Connection management feeds schema discovery, schema validation, extraction planning, and extraction execution.

Schema discovery feeds validation and can feed host caches, but cache ownership belongs outside Engine core.

Validation gates planning. Planning gates execution. Execution produces canonical results. Result handling may be direct return, host storage, or evented workflow depending on host shell.

Tenant/product context crosses all capabilities. It is not an afterthought bolted to storage paths; it scopes connection resolution, internal datasource resolution, execution, result sinks, and observability attributes.

Credential protection crosses connection lifecycle and connector construction. The Engine owns the semantics; the host owns key provisioning and rotation policy.

## Technical Integration Constraints

| Integration Point | Direction | Required In Core | Notes |
| --- | --- | --- | --- |
| Host application | Calls Engine | Yes | Embedded operation contracts are the main product surface. |
| Connection store | Engine to host adapter | Port only | MongoDB remains current compatibility adapter, not mandatory core. |
| Execution store | Engine to host adapter | Optional port | Needed for durable async behavior, not for all synchronous calls. |
| Result sink | Engine to host adapter | Port only | S3/SeaweedFS are optional adapters. In-memory or host-owned sinks must be possible. |
| Schema cache | Engine to host adapter | Optional port | Redis remains current adapter, not mandatory core. |
| Event sink | Engine to host adapter | Optional port | RabbitMQ remains current adapter, not mandatory core. |
| Observability | Engine to host hooks | Hook contracts | Host owns exporters and endpoints. |
| Auth/license | Host to compatibility adapter | No | Current Manager keeps lib-auth/lib-license. Engine receives already-authorized context. |
| Internal datasource resolver | Engine/host boundary | Capability required | Existing resolver behavior should be formalized without forcing tenant-manager into core. |

## Compatibility Adapter Map

Current Manager adapter:

- Keeps Fiber routes, Swagger, health/readiness/metrics, lib-auth, lib-license, tenant middleware, env loading, and process lifecycle.
- Delegates connection, schema, validation, job/execution operations to Engine.
- Maps Engine errors/results to existing HTTP responses.

Current Worker adapter:

- Keeps RabbitMQ consumption, message headers, ack/nack behavior, queue topology, and process lifecycle.
- Delegates extraction execution and result production to Engine.
- Maps Engine execution events to existing job updates and RabbitMQ notifications.
- Keeps `plugin_crm` as explicit Manager/Worker compatibility behavior for the first release. It must not become generic Engine core behavior.

Current infrastructure:

- MongoDB, Redis, RabbitMQ, SeaweedFS/S3, Docker Compose, and KEDA remain compatibility infrastructure.
- Embedded hosts may reuse those adapters or provide their own equivalents.

## Business Risks

The most likely failure mode is extracting too much infrastructure into Engine core. That would reproduce the standalone service boundary inside a library, which is the exact thing this feature exists to remove.

The second failure mode is extracting too little. A thin helper library would push planning, validation, tenant-safety, result semantics, and connector behavior into host apps, causing fragmentation.

The right line is: Engine owns what makes Fetcher Fetcher; host apps own how Fetcher runs.
