# Gate 5 data model: Fetcher embedded runtime

## Scope

This document defines logical entities, relationships, ownership, lifecycle, and naming expectations. It intentionally avoids SQL, MongoDB document syntax, ORM tags, and DB-specific naming rules.

Logical field names use camelCase where serialized. Persistence adapters may follow their backing-store conventions.

## Entity ownership summary

| Entity | Owner | Notes |
| --- | --- | --- |
| Connection | Engine semantics, host persistence | Engine validates and protects credentials; host chooses store. |
| ConnectorDefinition | Engine | Registry-owned description of supported datasource capabilities. |
| SchemaSnapshot | Engine semantics, optional host cache | Engine defines shape; host chooses cache/storage. |
| ValidationReport | Engine | Returned to host; may be stored by host if useful. |
| ExtractionRequest | Host input, Engine validation | Host creates request; Engine validates and plans. |
| ExtractionPlan | Engine | Internal or host-visible plan depending operation. |
| Execution | Engine semantics, optional host durability | Engine defines states; host chooses durable store. |
| ExtractionResult | Engine | Canonical output or result reference. |
| ResultReference | Host sink plus Engine contract | Host owns physical storage; Engine returns logical metadata. |
| EngineError | Engine | Stable error taxonomy for adapters and host apps. |
| TenantContext | Host input, Engine safety primitive | Host authenticates; Engine scopes behavior. |

## Connection

Purpose: Represents a configured datasource that can be discovered, validated, tested, and queried.

Fields:

- `id`: stable identifier.
- `productName`: product ownership boundary.
- `configName`: datasource alias used by extraction requests.
- `type`: datasource type.
- `host`: datasource network host.
- `port`: datasource network port.
- `databaseName`: logical database name.
- `schema`: optional default schema.
- `userName`: datasource username.
- `passwordEncrypted`: protected credential for persisted connections.
- `encryptionKeyVersion`: credential key version metadata.
- `ssl`: optional SSL configuration.
- `metadata`: optional connector-specific metadata.
- `createdAt`, `updatedAt`, `deletedAt`: lifecycle timestamps.

Ownership:

- Engine owns validation semantics and credential protection semantics.
- Host owns persistence and authorization policy.
- Compatibility adapter maps current `pkg/model.Connection` behavior (`pkg/model/connection.go:18-36`).

Relationships:

- Connection is referenced by `configName` in extraction requests.
- Connection can produce schema snapshots.
- Connection participates in extraction plans.

## ConnectorDefinition

Purpose: Describes a datasource type supported by the Engine.

Fields:

- `type`: datasource type.
- `capabilities`: query, schema discovery, filter support, SSL support, streaming support if any.
- `defaultLimits`: connector-specific safe defaults.
- `metadataSchema`: optional connector-specific metadata contract.

Ownership:

- Engine owns registry semantics.
- Host may register connectors through the explicit connector registry contract.
- The first release does not add a generic product-specific extension API.
- `plugin_crm` remains adapter-level compatibility behavior, not an Engine core extension model.

Relationships:

- Connection selects one connector definition by `type`.
- Extraction plan uses connector capabilities.

## SchemaSnapshot

Purpose: Canonical representation of discovered datasource schema.

Fields:

- `configName`: datasource alias.
- `tables`: collection of table schemas.
- `cachedAt`: optional cache timestamp.
- `expiresAt`: optional cache expiry.

Table schema fields:

- `tableName`: table or collection name.
- `columns`: available fields and nested field markers where applicable.

Ownership:

- Engine owns shape and validation semantics.
- Host owns cache backend if caching is enabled.
- Compatibility adapter maps current `DataSourceSchema` and `TableSchema` (`pkg/model/schema.go:77-89`).

Relationships:

- Created from Connection plus Connector.
- Used by ValidationReport and ExtractionPlan.

## ValidationReport

Purpose: Result of validating requested tables, fields, and filters.

Fields:

- `status`: success or failure.
- `message`: safe summary.
- `errors`: list of validation errors.

Validation error fields:

- `type`: data source not found, table not found, field not found, source down, limit exceeded, invalid filter.
- `configName`: datasource alias.
- `table`: optional table name.
- `field`: optional field name.
- `message`: optional safe detail.

Ownership:

- Engine owns error semantics.
- Host owns display/API mapping.

## ExtractionRequest

Purpose: Host-provided request to plan or execute extraction.

Fields:

- `mappedFields`: required datasource/table/fields map.
- `filters`: optional nested filter map.
- `metadata`: optional host metadata.
- `limits`: optional request-specific limits.
- `mode`: optional direct/store/plan-only execution hint.

Ownership:

- Host owns creation and authorization.
- Engine owns validation and normalization.
- Compatibility adapter maps current `FetcherRequest` and `DataRequest` (`pkg/model/job.go:255-273`).

Relationships:

- References connections by `configName`.
- Produces ExtractionPlan.

## ExtractionPlan

Purpose: Normalized executable work derived from an extraction request.

Fields:

- `planId`: optional stable identifier.
- `sources`: planned datasource work items.
- `limits`: resolved limits after defaults and overrides.
- `schemaPolicy`: discovered, cached, or provided schema usage.
- `resultPolicy`: direct return or result sink.
- `tenantContext`: required scope metadata where applicable.

Source work item fields:

- `configName`: datasource alias.
- `connectionId`: resolved connection.
- `type`: datasource type.
- `tables`: selected tables and fields.
- `filters`: selected filters.

Ownership:

- Engine owns planning.
- Host may inspect plans when useful but should not mutate executable internals.

## Execution

Purpose: Represents a run of an extraction plan.

Fields:

- `executionId`: stable identifier for durable execution.
- `status`: pending, processing, completed, failed, cancelled.
- `requestHash`: optional idempotency key.
- `metadata`: host metadata.
- `startedAt`, `completedAt`: timestamps.
- `result`: optional ExtractionResult.
- `errors`: optional EngineError list.

Ownership:

- Engine owns status semantics.
- Host owns durable store and scheduling if execution is async.
- Compatibility adapter maps current `Job` semantics (`pkg/model/job.go:35-77`).

Relationships:

- Execution runs one ExtractionPlan.
- Execution produces one ExtractionResult or failure errors.

## ExtractionResult

Purpose: Canonical output of extraction.

Fields:

- `data`: optional direct extracted payload.
- `resultRef`: optional result reference when stored.
- `format`: output format.
- `sizeBytes`: plaintext payload size before optional encryption.
- `rowCount`: total records extracted.
- `integrity`: optional result integrity metadata.
- `protection`: result protection metadata, separate from credential protection.
- `completedAt`: completion timestamp.

Integrity fields:

- `algorithm`: integrity algorithm, such as HMAC-SHA256 or SHA-256.
- `digest`: optional digest for checksum-style integrity.
- `signature`: optional signature for keyed integrity.

Protection fields:

- `encrypted`: whether result bytes are encrypted.
- `keyVersion`: optional result-encryption key version.
- `mode`: optional protection mode, such as envelope encryption or adapter-managed encryption.
- `appliedBy`: `engine`, `adapter`, or `host`.

Ownership:

- Engine owns result shape.
- Engine owns canonical result protection and integrity metadata semantics.
- Host owns storage backend and retention.
- Compatibility adapter maps current `JobResultData` (`components/worker/internal/services/job_notification.go:23-41`).

## ResultReference

Purpose: Logical reference to result data held by a host-selected sink.

Fields:

- `uri`: logical or storage-specific result path.
- `sink`: host-provided sink name/type.
- `bucket`: optional when meaningful.
- `key`: optional when meaningful.
- `integrity`: optional result integrity metadata when the sink returns or stores it with the reference.
- `protection`: optional result protection metadata when the sink returns or stores it with the reference.
- `expiresAt`: optional retention metadata.

Ownership:

- Host owns physical storage.
- Engine owns contract metadata, integrity semantics, and result protection semantics.

## EngineError

Purpose: Stable error model for host mapping.

Fields:

- `code`: stable machine-readable code.
- `category`: error category.
- `title`: short title.
- `message`: safe detail.
- `field`: optional logical field path.
- `retryable`: optional retry hint.

Ownership:

- Engine owns taxonomy.
- Host owns API/status/notification mapping.

## TenantContext

Purpose: Scopes Engine operations to tenant/product ownership boundaries.

Fields:

- `tenantId`: tenant boundary.
- `organizationId`: compatibility boundary where needed.
- `productName`: product ownership boundary.
- `requestId`: trace/correlation boundary.
- `actor`: host-authenticated subject metadata.

Ownership:

- Host authenticates and authorizes.
- Engine consumes context to enforce scoping, resolve internal datasources, decorate observability, and avoid cross-tenant result paths.

## Relationship Summary

TenantContext scopes Connection, SchemaSnapshot, ExtractionPlan, Execution, and ResultReference.

Connection is selected by ExtractionRequest through `configName`.

ConnectorDefinition determines how Connection is tested, discovered, queried, and closed.

SchemaSnapshot validates ExtractionRequest and informs ExtractionPlan.

ExtractionPlan produces Execution.

Execution produces ExtractionResult.

ExtractionResult may include ResultReference.

EngineError may attach to ValidationReport, Execution, or direct operation responses.

## Lifecycle Rules

Connection lifecycle: create, update, delete, list, get, test. Deletion may be soft in compatibility adapters; Engine only requires that deleted connections are not selected for new extraction.

Schema lifecycle: discovered on demand, optionally cached, invalidated by host or TTL.

Execution lifecycle: pending, processing, completed, failed, cancelled. Synchronous execution may skip durable pending state but must preserve equivalent result/error semantics.

Result lifecycle: produced, optionally assigned integrity metadata, optionally protected, optionally stored, returned to host, retained or deleted by host policy.

Credential lifecycle: accepted only on create/update or in-memory host configuration, protected before persistence, decrypted only for connector runtime, never logged.
