# ST-T005-02 Implement validate schema operation

## Goal

Add Engine schema validation for mapped fields and filters using discovered or cached schema snapshots.

## Prerequisites

- ST-T005-01 complete.

## Files

- Update `pkg/engine/schema_ops.go`.
- Update `pkg/engine/schema_ops_test.go`.
- Read `components/manager/internal/services/query/validate_schema.go`.
- Read `components/manager/internal/services/query/validate_schema_test.go`.

## Implementation brief

- Resolve each persisted connection by config name through the configured `connectionStore` or resolver before validation.
- Use `credentialProtector` when validation needs persisted encrypted credentials to discover schema from the connector.
- Resolve datasource support through `connectorRegistry`; do not instantiate concrete datasource drivers directly in Engine core.
- Use `schemaCache` when configured. A cache hit must validate without connector schema discovery; a cache miss must use connector discovery and then validate from the discovered snapshot.
- Apply effective Engine limits while validating datasource count, table count, field count, filter count, and connector-discovery timeouts.
- Validate mapped datasources, tables, fields, and filter field references against schema snapshots.
- Build a canonical validation report with stable error types.
- Treat source-down connector failures separately from malformed request failures.
- Keep Manager HTTP response formatting outside Engine.

## Test plan

1. RED: Add tests for successful validation, cache hit, cache miss, credential/protector failure, connector source-down error, missing datasource, missing table, missing field, invalid filter field, limit violation, safe error output, and validation report shape.
2. GREEN: Implement `ValidateSchema` with safe `ValidationReport` and Engine error mapping.
3. REFACTOR: Share schema traversal helpers only if they make the code smaller and clearer.

## Acceptance assertions

- Valid mapped fields return a success report.
- Cache hits avoid connector schema discovery.
- Cache misses use connector discovery through `connectorRegistry`.
- Credential/protector failures return safe Engine errors without raw credential material.
- Connector/source-down cases return connection or schema errors separately from malformed request errors.
- Missing datasource, table, field, invalid filter, and limit violations return distinct validation errors.
- Error messages do not include credentials or extracted data.

## Commands

- Run `go test ./pkg/engine/...`.
- Run `go test ./components/manager/internal/services/query`.
- Run `go test ./...`.
- Expected pass pattern: validation behavior is covered in Engine and Manager tests still pass.

## Expected failure/pass patterns

- Failure: missing table returns a validation report failure, not a panic or raw connector error.
- Pass: Manager validation tests remain green before compatibility migration.

## Rollback

- Revert validation changes in `schema_ops.go` and tests.
