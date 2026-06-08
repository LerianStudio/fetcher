# Gate 4 API design: Fetcher embedded runtime

## Scope

This document defines embedded/component contracts for an importable Fetcher Engine. It does not define new HTTP routes. Existing HTTP routes are compatibility adapter concerns and should map to these operation contracts where relevant.

Contract names are conceptual. Gate 7 has fixed `pkg/engine` as the Engine core package and optional compatibility wrappers under `pkg/enginecompat/*`. Exact method names remain implementation details.

## Contract principles

- Embedded contracts use camelCase JSON-style field names where serialized.
- Contracts describe operations and schemas, not Fiber handlers.
- Host apps pass authorized context; Engine does not perform auth/license checks.
- Host apps provide stores, sinks, caches, events, and observability hooks through ports.
- Engine errors are stable and mappable to HTTP, queue, CLI, or in-process workflows.

## Engine construction contract

Purpose: Create an Engine instance from host-provided capabilities.

Inputs:

- `connectionStore`: required for persisted connection lifecycle unless host uses resolver-only mode.
- `credentialProtector`: required when persisted encrypted credentials are used.
- `connectorRegistry`: required; includes supported datasource connectors.
- `resultSink`: optional; required for store-and-reference execution.
- `executionStore`: optional; required for durable async/job compatibility.
- `schemaCache`: optional.
- `eventSink`: optional.
- `limits`: optional; default safe limits apply.
- `observabilityHooks`: optional.
- `tenantResolver`: optional but required for internal datasource resolution or tenant-scoped host integrations.

Output:

- `Engine` with connection, schema, validation, planning, and execution operations.

## Operation contracts

| Operation | Purpose | Required host capabilities |
| --- | --- | --- |
| `createConnection` | Validate and persist a connection definition | connectionStore, credentialProtector |
| `updateConnection` | Patch a connection definition safely | connectionStore, credentialProtector |
| `deleteConnection` | Delete or mark connection unavailable | connectionStore |
| `getConnection` | Retrieve a connection definition | connectionStore or resolver |
| `listConnections` | List scoped connections | connectionStore or resolver |
| `testConnection` | Verify datasource connectivity | connectionStore/resolver or inline connection, credentialProtector, connectorRegistry |
| `discoverSchema` | Return canonical schema for a connection | connectionStore/resolver, credentialProtector, connectorRegistry, optional schemaCache |
| `validateSchema` | Validate mapped fields against datasource schemas | connectionStore/resolver, credentialProtector when persisted credentials are used, connectorRegistry, optional schemaCache |
| `planExtraction` | Build executable extraction plan | connectionStore/resolver, credentialProtector when persisted credentials are used, connectorRegistry, optional schemaCache, limits |
| `executeExtraction` | Run extraction and return canonical result | connectorRegistry, credentialProtector when persisted credentials are used, optional resultSink, optional executionStore |
| `getExecution` | Retrieve execution status/result reference | executionStore |
| `cancelExecution` | Request cancellation where host supports it | executionStore or host scheduler |

## Shared schemas

### Execution context

Purpose: Carry host-owned identity, tenant, product, correlation, and policy context into Engine operations.

Fields:

- `tenantId`: optional unless host is multi-tenant or uses tenant-scoped internal datasource resolution.
- `organizationId`: optional compatibility field for current Manager API behavior.
- `productName`: required for external connection ownership where persisted connections are used.
- `requestId`: optional correlation identifier.
- `actor`: optional host-authenticated subject metadata.
- `deadline`: host-provided through context, not serialized by default.

### Connection definition

Purpose: Canonical datasource connection input and output.

Fields:

- `id`: stable connection identifier.
- `productName`: product ownership boundary.
- `configName`: datasource alias used in mapped fields.
- `type`: `MONGODB`, `POSTGRESQL`, `MYSQL`, `ORACLE`, or `SQL_SERVER`.
- `host`: datasource host.
- `port`: datasource port.
- `databaseName`: logical database name.
- `schema`: optional default schema.
- `userName`: datasource user.
- `password`: accepted on create/update input only; never returned.
- `passwordEncrypted`: internal/store-facing encrypted value; not exposed to normal host API responses.
- `encryptionKeyVersion`: key version metadata.
- `ssl`: optional SSL settings.
- `metadata`: optional connector-specific metadata.
- `createdAt`, `updatedAt`, `deletedAt`: lifecycle timestamps.

### Mapped fields

Purpose: Datasource/table/field selection for validation and extraction.

Example:

```json
{
  "mappedFields": {
    "midaz_onboarding": {
      "organization": ["id", "legalName"]
    }
  }
}
```

### Filters

Purpose: Optional field-level query predicates. Exact operator support remains connector-governed but must validate against Engine limits and datasource schema.

Example:

```json
{
  "filters": {
    "midaz_onboarding": {
      "organization": {
        "createdAt": { "gte": ["2026-01-01"], "lte": ["2026-01-31"] }
      }
    }
  }
}
```

### Schema

Purpose: Canonical schema discovery and validation model.

Fields:

- `configName`: datasource alias.
- `tables`: map of table/collection names to table schema.
- `tableName`: table or collection name.
- `columns`: map or list of available fields depending on operation output.
- `cachedAt`, `expiresAt`: present only when cache metadata is relevant.

### Extraction request

Purpose: Ask Engine to plan or execute extraction.

Fields:

- `mappedFields`: required.
- `filters`: optional.
- `metadata`: optional host metadata; current compatibility requires `source` for existing job flows.
- `mode`: optional host hint such as `direct`, `store`, or `planOnly`.
- `limits`: optional per-request overrides within Engine maximums.

### Extraction result

Purpose: Canonical result contract independent of storage backend.

Fields:

- `executionId`: optional for durable execution.
- `status`: `pending`, `processing`, `completed`, `failed`, or `cancelled`.
- `data`: optional direct payload for in-process direct return.
- `resultRef`: optional reference when written to a result sink.
- `format`: e.g. `json`.
- `sizeBytes`: plaintext size before optional encryption.
- `rowCount`: total extracted rows/records.
- `integrity`: optional result integrity metadata.
- `protection`: result protection metadata, separate from credential protection.
- `startedAt`, `completedAt`: execution timestamps.
- `errors`: optional canonical errors.

Integrity fields:

- `algorithm`: integrity algorithm, such as HMAC-SHA256 or SHA-256.
- `digest`: optional digest for checksum-style integrity.
- `signature`: optional signature for keyed integrity.

Protection fields:

- `encrypted`: whether result bytes are encrypted.
- `keyVersion`: optional result-encryption key version.
- `mode`: optional protection mode, such as envelope encryption or adapter-managed encryption.
- `appliedBy`: `engine`, `adapter`, or `host`.

Engine owns the canonical meaning of result `integrity` and `protection` metadata. This metadata describes extracted result bytes only; it does not describe credential encryption. Compatibility adapters preserve the current encrypted storage payload and HMAC behavior while mapping it into the canonical metadata shape.

## Port contracts

| Port | Responsibility | Mandatory core dependency |
| --- | --- | --- |
| `ConnectionStore` | Persist and retrieve external connection definitions | Interface only |
| `ExecutionStore` | Persist durable execution/job state | Interface only, optional |
| `ResultSink` | Store or stream extracted results | Interface only, optional by mode |
| `SchemaCache` | Cache discovered schemas | Interface only, optional |
| `EventSink` | Emit execution events | Interface only, optional |
| `CredentialProtector` | Encrypt/decrypt connection credentials | Interface required when persisted encrypted credentials are used |
| `Connector` | Connect, query, discover schema, close | Interface required |
| `ConnectorRegistry` | Resolve connectors by datasource type | Interface required |
| `TenantResolver` | Resolve internal/tenant-scoped datasources | Interface optional by deployment |
| `ObservabilityHooks` | Emit logs, metrics, traces, events without owning exporters | Interface optional |

## Error Contract

Engine errors should include:

- `code`: stable machine-readable code.
- `category`: `validation`, `notFound`, `conflict`, `unauthorizedContext`, `connection`, `schema`, `planning`, `extraction`, `timeout`, `cancelled`, `storage`, `internal`.
- `title`: short human-readable title.
- `message`: safe detail with no credentials or sensitive extracted data.
- `field`: optional field path for validation errors.
- `cause`: optional wrapped error for internal logs only.

Compatibility adapters map Engine errors to existing HTTP status codes, RabbitMQ handling, job metadata, and notification payloads.

## Compatibility Adapter Contracts

Manager HTTP adapter keeps current routes and middleware. It maps HTTP inputs to Engine operation contracts and Engine outputs/errors back to existing responses.

Worker queue adapter keeps current message payloads and headers. It maps queue messages to Engine execution requests and maps Engine results/errors to current job updates and notifications.

No new HTTP-route contract is required for Engine core.
