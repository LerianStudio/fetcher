# Impact Summary

**Language:** go

## Warnings

- **Partial Results:** Some functions could not be fully analyzed.
- Truncated modified functions from 675 to 500

## Summary Metrics

| Metric | Value |
|--------|-------|
| Modified Functions | 500 |
| Direct Callers | 541 |
| Transitive Callers | 9467 |
| Affected Tests | 412 |
| Affected Packages | 41 |

### Affected Packages

- `components/manager/internal/adapters/http/in`
- `components/manager/internal/bootstrap`
- `components/manager/internal/services/command`
- `components/manager/internal/services/query`
- `components/worker/internal/adapters/rabbitmq`
- `components/worker/internal/bootstrap`
- `components/worker/internal/services`
- `pkg`
- `pkg/datasource`
- `pkg/itestkit/addons/e2ekit`
- `pkg/itestkit/addons/metricskit`
- `pkg/itestkit/addons/queuekit`
- `pkg/itestkit`
- `pkg/itestkit/infra/mongodb`
- `pkg/itestkit/infra/mssql`
- `pkg/itestkit/infra/mysql`
- `pkg/itestkit/infra/oracle`
- `pkg/itestkit/infra/postgres`
- `pkg/itestkit/infra/rabbitmq`
- `pkg/itestkit/infra/redis`
- `pkg/itestkit/infra/seaweedfs`
- `pkg/model/datasource/mongodb`
- `pkg/model/datasource/mysql`
- `pkg/model/datasource/oracle`
- `pkg/model/datasource/postgres`
- `pkg/model/datasource/sqlserver`
- `pkg/mongodb/connection`
- `pkg/mongodb`
- `pkg/mongodb/job`
- `pkg/mongodb/product`
- `pkg/mysql`
- `pkg/net/http`
- `pkg/oracle`
- `pkg/postgres`
- `pkg/rabbitmq`
- `pkg/redis`
- `pkg/seaweedfs/external`
- `pkg/sqlserver`
- `pkg/testutil`
- `tests/chaos`
- `tests/shared`

## High Impact Functions

Functions with 3 or more callers - changes here have wide-reaching effects.

### `setupConnectionTestApp`

**File:** `components/manager/internal/adapters/http/in/connection_test.go`
**Risk Level:** HIGH
**Callers:** 55

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestConnectionHandler_ListConnections_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:481)
- `TestConnectionHandler_ValidateSchema_InternalError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1696)
- `TestConnectionHandler_ValidateSchema_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1306)
- `TestConnectionHandler_ValidateSchema_Failure_DataSourceNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1485)
- `TestConnectionHandler_DeleteConnection_HandlerDirectly_InvalidUUID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1242)
- `TestConnectionHandler_ValidateSchema_Failure_TableNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1350)
- `TestConnectionHandler_ListConnections_EmptyList` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:542)
- `TestConnectionHandler_ValidateSchema_HandlerDirectly_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1967)
- `TestConnectionHandler_ValidateSchema_Failure_DataSourceDown` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1548)
- `TestConnectionHandler_CreateConnection_RealHandlerSuccess` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real_handler_test.go:45)
- `TestConnectionHandler_TestConnection_HandlerDirectly_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1951)
- `TestConnectionHandler_ListConnections_HandlerDirectly_InvalidSortOrder` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1886)
- `TestConnectionHandler_CreateConnection_InternalError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:287)
- `TestConnectionHandler_CreateConnection_HandlerDirectly_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1838)
- `TestConnectionHandler_ListConnections_RealHandlerSuccess` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real_handler_test.go:101)
- `TestConnectionHandler_TestConnection_RateLimited` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1142)
- `TestConnectionHandler_CreateConnection_Conflict` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:243)
- `TestConnectionHandler_ValidateSchema_Failure_FieldNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1417)
- `TestConnectionHandler_ValidateSchema_HandlerDirectly_InvalidJSON` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1796)
- `TestConnectionHandler_UpdateConnection_NotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:711)
- `TestConnectionHandler_GetConnection_InvalidID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:433)
- `TestConnectionHandler_TestConnection_InvalidID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1126)
- `TestConnectionHandler_GetConnection_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:335)
- `TestProductHandler_ListProducts_RealHandlerSuccess` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real_handler_test.go:293)
- `TestConnectionHandler_DeleteConnection_NotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:920)
- `TestConnectionHandler_ValidateSchema_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1646)
- `TestMigrationHandler_ListUnassignedConnections_RealHandlerSuccess` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real_handler_test.go:322)
- `TestConnectionHandler_GetConnection_NotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:391)
- `TestConnectionHandler_DeleteConnection_Conflict_ActiveJobs` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:962)
- `TestConnectionHandler_UpdateConnection_RealHandlerSuccess` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real_handler_test.go:140)
- `TestConnectionHandler_ListConnections_InvalidPaginationParams` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:590)
- `TestConnectionHandler_DeleteConnection_HandlerDirectly_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1935)
- `TestConnectionHandler_CreateConnection_InvalidJSON` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:150)
- `TestConnectionHandler_TestConnection_NotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1084)
- `TestConnectionHandler_TestConnection_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1026)
- `TestConnectionHandler_DeleteConnection_InvalidID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1006)
- `TestConnectionHandler_CreateConnection_HandlerDirectly_InvalidJSON` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1189)
- `TestConnectionHandler_UpdateConnection_HandlerDirectly_InvalidUUID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1225)
- `TestConnectionHandler_CreateConnection_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:90)
- `TestConnectionHandler_GetConnection_HandlerDirectly_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1902)
- `TestConnectionHandler_CreateConnection_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:193)
- `TestConnectionHandler_TestConnection_HandlerDirectly_InvalidUUID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1258)
- `TestConnectionHandler_UpdateConnection_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:642)
- `TestConnectionHandler_UpdateConnection_Conflict_ActiveJobs` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:805)
- `TestConnectionHandler_GetConnectionSchema_RealHandlerSuccess` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real_handler_test.go:222)
- `TestConnectionHandler_DeleteConnection_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:867)
- `TestConnectionHandler_GetConnection_HandlerDirectly_InvalidUUID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1209)
- `TestConnectionHandler_ValidateSchema_RealHandlerFailureResponse` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real_handler_test.go:183)
- `TestConnectionHandler_ValidateSchema_InvalidJSON` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1603)
- `TestProductHandler_CreateProduct_RealHandlerSuccess` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real_handler_test.go:260)
- `TestConnectionHandler_ValidateSchema_MultipleDataSources` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1740)
- `TestConnectionHandler_ListConnections_HandlerDirectly_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1274)
- `TestConnectionHandler_UpdateConnection_InvalidBody` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:767)
- `TestConnectionHandler_UpdateConnection_HandlerDirectly_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1918)
- `TestMigrationHandler_AssignConnectionToProduct_RealHandlerSuccess` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real_handler_test.go:351)

</details>

**Direct Callers:**

- `TestConnectionHandler_ListConnections_Success` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:481` (call site: connection_test.go:481)
- `TestConnectionHandler_ValidateSchema_InternalError` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1696` (call site: connection_test.go:1696)
- `TestConnectionHandler_ValidateSchema_Success` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1306` (call site: connection_test.go:1306)
- `TestConnectionHandler_ValidateSchema_Failure_DataSourceNotFound` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1485` (call site: connection_test.go:1485)
- `TestConnectionHandler_DeleteConnection_HandlerDirectly_InvalidUUID` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1242` (call site: connection_test.go:1242)
- `TestConnectionHandler_ValidateSchema_Failure_TableNotFound` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1350` (call site: connection_test.go:1350)
- `TestConnectionHandler_ListConnections_EmptyList` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:542` (call site: connection_test.go:542)
- `TestConnectionHandler_ValidateSchema_HandlerDirectly_MissingOrgHeader` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1967` (call site: connection_test.go:1967)
- `TestConnectionHandler_ValidateSchema_Failure_DataSourceDown` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection_test.go:1548` (call site: connection_test.go:1548)
- `TestConnectionHandler_CreateConnection_RealHandlerSuccess` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real_handler_test.go:45` (call site: real_handler_test.go:45)
- ... and 45 more

**Calls:**

- `New` at `/Users/fredamaral/go/pkg/mod/github.com/gofiber/fiber/v2@v2.52.12/app.go:497`
- `*github.com/gofiber/fiber/v2.App.Use` at `/Users/fredamaral/go/pkg/mod/github.com/gofiber/fiber/v2@v2.52.12/app.go:685`

---

### `setupTestApp`

**File:** `components/manager/internal/adapters/http/in/fetcher_test.go`
**Risk Level:** HIGH
**Callers:** 21

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestFetcherHandler_GetJob_NotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:596)
- `TestFetcherHandler_CreateJob_HandlerDirectly_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:1046)
- `TestFetcherHandler_GetJob_InvalidID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:638)
- `TestFetcherHandler_GetJob_FailedJob` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:976)
- `TestFetcherHandler_GetJob_HandlerDirectly_InvalidUUID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:744)
- `TestFetcherHandler_CreateJob_WithFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:853)
- `TestFetcherHandler_GetJob_InternalError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:696)
- `TestFetcherHandler_CreateJob_Conflict` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:448)
- `TestFetcherHandler_CreateJob_HandlerDirectly_InvalidJSON` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:760)
- `TestFetcherHandler_CreateJob_WithMetadata` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:784)
- `TestFetcherHandler_CreateJob_Success_DuplicateJob` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:390)
- `TestFetcherHandler_CreateJob_MetadataSourceValidation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:210)
- `TestFetcherHandler_GetJob_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:680)
- `TestFetcherHandler_GetJob_CompletedJob` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:924)
- `TestFetcherHandler_GetJob_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:540)
- `TestFetcherHandler_GetJob_HandlerDirectly_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:1063)
- `TestFetcherHandler_CreateJob_ContentTypeValidation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:163)
- `TestFetcherHandler_CreateJob_InvalidJSON` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:54)
- `TestFetcherHandler_CreateJob_MissingOrgHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:105)
- `TestFetcherHandler_CreateJob_Success_NewJob` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:326)
- `TestFetcherHandler_CreateJob_InternalError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:492)

</details>

**Direct Callers:**

- `TestFetcherHandler_GetJob_NotFound` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:596` (call site: fetcher_test.go:596)
- `TestFetcherHandler_CreateJob_HandlerDirectly_MissingOrgHeader` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:1046` (call site: fetcher_test.go:1046)
- `TestFetcherHandler_GetJob_InvalidID` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:638` (call site: fetcher_test.go:638)
- `TestFetcherHandler_GetJob_FailedJob` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:976` (call site: fetcher_test.go:976)
- `TestFetcherHandler_GetJob_HandlerDirectly_InvalidUUID` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:744` (call site: fetcher_test.go:744)
- `TestFetcherHandler_CreateJob_WithFilters` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:853` (call site: fetcher_test.go:853)
- `TestFetcherHandler_GetJob_InternalError` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:696` (call site: fetcher_test.go:696)
- `TestFetcherHandler_CreateJob_Conflict` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:448` (call site: fetcher_test.go:448)
- `TestFetcherHandler_CreateJob_HandlerDirectly_InvalidJSON` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:760` (call site: fetcher_test.go:760)
- `TestFetcherHandler_CreateJob_WithMetadata` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:784` (call site: fetcher_test.go:784)
- ... and 11 more

**Calls:**

- `New` at `/Users/fredamaral/go/pkg/mod/github.com/gofiber/fiber/v2@v2.52.12/app.go:497`
- `*github.com/gofiber/fiber/v2.App.Use` at `/Users/fredamaral/go/pkg/mod/github.com/gofiber/fiber/v2@v2.52.12/app.go:685`

---

### `TestFetcherHandler_CreateJob_MetadataSourceValidation`

**File:** `components/manager/internal/adapters/http/in/fetcher_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `setupTestApp` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher_test.go:28`
- `*github.com/gofiber/fiber/v2.App.Post` at `/Users/fredamaral/go/pkg/mod/github.com/gofiber/fiber/v2@v2.52.12/app.go:728`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `setupMiddlewareTestApp`

**File:** `components/manager/internal/adapters/http/in/middlewares_test.go`
**Risk Level:** HIGH
**Callers:** 14

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestParseHeaderParameters_MissingHeader` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:205)
- `TestParsePathParametersUUID_MultipleCalls` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:146)
- `TestParseHeaderParameters_CaseInsensitiveHeaderName` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:285)
- `TestParsePathParametersUUID_EmptyPathParameter` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:126)
- `TestParseHeaderParameters_ValidOrgID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:184)
- `TestParsePathParametersUUID_UUIDVersions$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:433)
- `TestMiddlewareChain_PathMiddlewareFails` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:386)
- `TestParseHeaderParameters_UUIDWithWhitespace` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:451)
- `TestMiddlewareChain_BothMiddlewares` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:341)
- `TestParsePathParametersUUID_InvalidUUID$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:108)
- `TestParseHeaderParameters_MultipleCalls` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:306)
- `TestParsePathParametersUUID_ValidUUID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:47)
- `TestMiddlewareChain_HeaderMiddlewareFails` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:367)
- `TestParseHeaderParameters_InvalidOrgID$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:266)

</details>

**Direct Callers:**

- `TestParseHeaderParameters_MissingHeader` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:205` (call site: middlewares_test.go:205)
- `TestParsePathParametersUUID_MultipleCalls` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:146` (call site: middlewares_test.go:146)
- `TestParseHeaderParameters_CaseInsensitiveHeaderName` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:285` (call site: middlewares_test.go:285)
- `TestParsePathParametersUUID_EmptyPathParameter` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:126` (call site: middlewares_test.go:126)
- `TestParseHeaderParameters_ValidOrgID` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:184` (call site: middlewares_test.go:184)
- `TestParsePathParametersUUID_UUIDVersions$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:433` (call site: middlewares_test.go:433)
- `TestMiddlewareChain_PathMiddlewareFails` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:386` (call site: middlewares_test.go:386)
- `TestParseHeaderParameters_UUIDWithWhitespace` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:451` (call site: middlewares_test.go:451)
- `TestMiddlewareChain_BothMiddlewares` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:341` (call site: middlewares_test.go:341)
- `TestParsePathParametersUUID_InvalidUUID$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares_test.go:108` (call site: middlewares_test.go:108)
- ... and 4 more

**Calls:**

- `New` at `/Users/fredamaral/go/pkg/mod/github.com/gofiber/fiber/v2@v2.52.12/app.go:497`
- `*github.com/gofiber/fiber/v2.App.Use` at `/Users/fredamaral/go/pkg/mod/github.com/gofiber/fiber/v2@v2.52.12/app.go:685`

---

### `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes`

**File:** `components/manager/internal/adapters/http/in/routes_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewNop` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/nil.go:9`
- `NewTelemetry` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/opentelemetry/otel.go:88`
- `NoError` at `/Users/fredamaral/go/pkg/mod/github.com/stretchr/testify@v1.11.1/require/require.go:1394`
- `NewAuthClient` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-auth/v2@v2.4.0/auth/middleware/middleware.go:52`
- `NewRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes.go:24`
- ... and 16 more

---

### `indexOfRoute`

**File:** `components/manager/internal/adapters/http/in/routes_test.go`
**Risk Level:** HIGH
**Callers:** 7

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:65)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:66)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:67)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:74)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:75)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:76)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:77)

</details>

**Direct Callers:**

- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:65` (call site: routes_test.go:65)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:66` (call site: routes_test.go:66)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:67` (call site: routes_test.go:67)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:74` (call site: routes_test.go:74)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:75` (call site: routes_test.go:75)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:76` (call site: routes_test.go:76)
- `TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes_test.go:77` (call site: routes_test.go:77)

**Calls:** None

---

### `must`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** HIGH
**Callers:** 11

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestMust$2` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:78)
- `TestMust$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:50)

</details>

**Direct Callers:**

- `ForEachPackage` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/allpackages.go:68` (call site: allpackages.go:68)
- `TestMust$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:78` (call site: config_test.go:78)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:146` (call site: config.go:146)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:149` (call site: config.go:149)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:155` (call site: config.go:155)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:159` (call site: config.go:159)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:163` (call site: config.go:163)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:188` (call site: config.go:188)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:194` (call site: config.go:194)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:197` (call site: config.go:197)
- ... and 1 more

**Calls:**

- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`

---

### `TestGetSchemaCacheTTL`

**File:** `components/manager/internal/bootstrap/config_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `getSchemaCacheTTL` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:351`
- `Equal` at `/Users/fredamaral/go/pkg/mod/github.com/stretchr/testify@v1.11.1/assert/assertions.go:495`
- `getSchemaCacheTTL` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:351`
- `Equal` at `/Users/fredamaral/go/pkg/mod/github.com/stretchr/testify@v1.11.1/assert/assertions.go:495`
- `getSchemaCacheTTL` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:351`
- ... and 1 more

---

### `TestGetRedisDB`

**File:** `components/manager/internal/bootstrap/config_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `getRedisDB` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:366`
- `Equal` at `/Users/fredamaral/go/pkg/mod/github.com/stretchr/testify@v1.11.1/assert/assertions.go:495`
- `getRedisDB` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:366`
- `Equal` at `/Users/fredamaral/go/pkg/mod/github.com/stretchr/testify@v1.11.1/assert/assertions.go:495`
- `getRedisDB` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:366`
- ... and 1 more

---

### `TestResolveZapEnvironment`

**File:** `components/manager/internal/bootstrap/config_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Parallel` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1758`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `NewCreateFetcherJobWithTester`

**File:** `components/manager/internal/services/command/create_fetcher_job.go`
**Risk Level:** HIGH
**Callers:** 8

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCreateFetcherJob_Execute_ProductNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:948)
- `TestCreateFetcherJob_Execute_ConnectionNotAssigned` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:1070)
- `TestCreateFetcherJob_Execute_ProductRepoError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:1010)
- `TestCreateFetcherJob_Execute_MultipleConnectionsSuccess_WithoutProductRepo` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:627)
- `TestCreateFetcherJob_Execute_FiltersWithMultipleDatasources` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:791)
- `TestCreateFetcherJob_Execute_PublishFailureMarksJobFailed` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:706)
- `TestCreateFetcherJob_Execute_ProductValidationSuccess` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:1208)
- `TestCreateFetcherJob_Execute_ProductMismatch` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:1139)

</details>

**Direct Callers:**

- `TestCreateFetcherJob_Execute_ProductNotFound` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:948` (call site: create_fetcher_job_test.go:948)
- `TestCreateFetcherJob_Execute_ConnectionNotAssigned` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:1070` (call site: create_fetcher_job_test.go:1070)
- `TestCreateFetcherJob_Execute_ProductRepoError` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:1010` (call site: create_fetcher_job_test.go:1010)
- `TestCreateFetcherJob_Execute_MultipleConnectionsSuccess_WithoutProductRepo` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:627` (call site: create_fetcher_job_test.go:627)
- `TestCreateFetcherJob_Execute_FiltersWithMultipleDatasources` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:791` (call site: create_fetcher_job_test.go:791)
- `TestCreateFetcherJob_Execute_PublishFailureMarksJobFailed` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:706` (call site: create_fetcher_job_test.go:706)
- `TestCreateFetcherJob_Execute_ProductValidationSuccess` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:1208` (call site: create_fetcher_job_test.go:1208)
- `TestCreateFetcherJob_Execute_ProductMismatch` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job_test.go:1139` (call site: create_fetcher_job_test.go:1139)

**Calls:** None

---

### `TestCreateFetcherJob_Execute_PublishFailureMarksJobFailed`

**File:** `components/manager/internal/services/command/create_fetcher_job_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.mock.go:37`
- `NewMockRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb.mock.go:34`
- `NewMockConnectionTester` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/connection_tester.mock.go:27`
- ... and 733 more

---

### `testContext`

**File:** `components/manager/internal/services/command/helpers_test.go`
**Risk Level:** HIGH
**Callers:** 97

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestTestConnection_Execute_DecryptionKeyVersionMismatch` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:652)
- `TestGetConnection_Execute_AllDatabaseTypes$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:363)
- `TestListConnections_Execute_Pagination$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:753)
- `TestValidateSchema_LargeNumberOfTables` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:985)
- `TestListConnections_Execute_WithDateFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:491)
- `TestTestConnection_Execute_AllDatabaseTypes$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:415)
- `TestTestConnection_Execute_WithSSLConfiguration` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:337)
- `TestGetConnection_Execute_ConnectionWithSSL` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:291)
- `TestTestConnection_Execute_EmptyUUIDs` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:522)
- `TestGetConnectionSchema_Execute_NotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:115)
- `TestTestConnection_Execute_DecryptionError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:141)
- `TestListConnections_Execute_WithSortOrder$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:549)
- `TestTestConnection_Execute_RateLimitError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:183)
- `TestGetJob_Execute$5` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_job_test.go:108)
- `TestValidateSchema_EmptyConfigName` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:929)
- `TestGetProduct_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:40)
- `TestValidateSchema_DataSourceNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:88)
- `TestTestConnection_Execute_RateLimited` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:253)
- `TestValidateSchema_InvalidRequest_EmptyMappedFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:280)
- `TestListConnections_Execute_ConnectionWithSSL` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:796)
- `TestTestConnection_Execute_ConnectionWithAllFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:693)
- `TestGetProduct_Execute_TableDriven$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:254)
- `TestListProducts_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:317)
- `TestTestConnection_Execute_DifferentOrganizations` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:762)
- `TestListUnassignedConnections_Execute_TableDriven$6` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:245)
- `TestListUnassignedConnections_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:284)
- `TestListConnections_Execute_EmptyOrganizationID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:933)
- `TestListProducts_Execute_EmptyList` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:84)
- `TestListConnections_Execute_EmptyList` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:103)
- `TestListConnections_Execute_DifferentOrganizations` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:395)
- `TestGetConnection_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:168)
- `TestGetConnection_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:133)
- `TestGetConnection_Execute_TableDriven$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:245)
- `TestGetConnection_Execute_ConnectionWithAllFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:418)
- `TestGetProduct_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:177)
- `TestGetConnectionSchema_Execute_DataSourceFactoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:183)
- `TestTestConnection_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:95)
- `TestGetConnectionSchema_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:55)
- `TestListConnections_Execute_EmptyFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:686)
- `TestGetConnectionSchema_Execute_NilSchema` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:399)
- `TestListConnections_Execute_WithProductFilter_NotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:1024)
- `TestTestConnection_Execute_MultipleRepositoryErrors$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:484)
- `TestTestConnection_Execute_NotFoundError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:55)
- `TestValidateSchema_MultipleDatasources_AllValid` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:522)
- `TestValidateSchema_InvalidRequest_NilMappedFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:257)
- `TestValidateSchema_NilSchemaFromDatasource` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:902)
- `TestListConnections_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:350)
- `TestListConnections_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:55)
- `TestListConnections_Execute_TableDriven$7` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:308)
- `TestGetConnectionSchema_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:150)
- `TestListProducts_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:39)
- `TestListUnassignedConnections_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:103)
- `TestGetProduct_Execute_EmptyUUIDs` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:429)
- `TestValidateSchema_TableNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:135)
- `TestGetConnection_Execute_EmptyUUIDs` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:568)
- `TestValidateSchema_PartialConnectionsFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:411)
- `TestNewLoggerFromContext$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:57)
- `TestListUnassignedConnections_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:23)
- `TestValidateSchema_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:55)
- `TestGetConnectionSchema_Execute_GetSchemaInfoError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:219)
- `TestValidateSchema_FieldNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:181)
- `TestValidateSchema_CacheError_ContinuesToFetch` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:367)
- `TestGetConnectionSchema_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:483)
- `TestValidateSchema_PostgreSQLWithMixedQualifiedTables` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:718)
- `TestListProducts_Execute_ErrorScenarios$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:395)
- `TestGetProduct_Execute_ProductWithAllFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:300)
- `TestValidateSchema_PostgreSQLWithUnqualifiedTables` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:669)
- `TestGetProduct_Execute_NotFoundError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:89)
- `TestListUnassignedConnections_Execute_EmptyList` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:69)
- `TestValidateSchema_MultipleErrors` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:228)
- `TestGetProduct_Execute_MultipleRepositoryErrors$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:394)
- `TestValidateSchema_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:310)
- `TestListConnections_Execute_WithCursor` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:650)
- `TestListConnections_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:139)
- `TestListProducts_Execute_TableDriven$7` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:277)
- `TestGetConnectionSchema_Execute_FiltersSystemTables$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:338)
- `TestGetProduct_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:124)
- `TestListProducts_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:118)
- `TestListUnassignedConnections_Execute_ErrorScenarios$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:381)
- `TestValidateSchema_NonPostgreSQLDoesNotAddPublicSchema$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:783)
- `TestListConnections_Execute_ErrorScenarios$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:897)
- `TestListConnections_Execute_WithMetadataFilter` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:451)
- `TestTestConnection_Execute_RateLimitResetTime$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:612)
- `TestContextWithTracer$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:221)
- `TestTestConnection_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:227)
- `TestListConnections_Execute_WithProductFilter_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:966)
- `TestValidateSchema_CacheSetError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:843)
- `TestListConnections_Execute_WithProductFilter_RepoError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:1066)
- `TestContextWithLogger$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:167)
- `TestNewTracerFromContext$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:112)
- `TestListConnections_Execute_ConnectionTypes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:589)
- `TestGetConnection_Execute_MultipleRepositoryErrors$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:533)
- `TestValidateSchema_NoConnections_ReturnsSchemaEntityType` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:467)
- `TestTestConnection_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:293)
- `TestGetConnection_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:45)
- `TestGetConnection_Execute_NotFoundError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:98)
- `TestGetConnectionSchema_Execute_EmptySchema` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:441)

</details>

**Direct Callers:**

- `TestTestConnection_Execute_DecryptionKeyVersionMismatch` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:652` (call site: test_connection_test.go:652)
- `TestGetConnection_Execute_AllDatabaseTypes$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:363` (call site: get_connection_test.go:363)
- `TestListConnections_Execute_Pagination$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:753` (call site: list_connections_test.go:753)
- `TestValidateSchema_LargeNumberOfTables` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:985` (call site: validate_schema_test.go:985)
- `TestListConnections_Execute_WithDateFilters` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:491` (call site: list_connections_test.go:491)
- `TestTestConnection_Execute_AllDatabaseTypes$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:415` (call site: test_connection_test.go:415)
- `TestTestConnection_Execute_WithSSLConfiguration` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:337` (call site: test_connection_test.go:337)
- `TestGetConnection_Execute_ConnectionWithSSL` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:291` (call site: get_connection_test.go:291)
- `TestTestConnection_Execute_EmptyUUIDs` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:522` (call site: test_connection_test.go:522)
- `TestGetConnectionSchema_Execute_NotFound` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:115` (call site: get_connection_schema_test.go:115)
- ... and 87 more

**Calls:**

- `Tracer` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel@v1.42.0/trace.go:15`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `WithValue` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:727`

---

### `testContext`

**File:** `components/manager/internal/services/query/helpers_test.go`
**Risk Level:** HIGH
**Callers:** 97

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestTestConnection_Execute_DecryptionKeyVersionMismatch` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:652)
- `TestGetConnection_Execute_AllDatabaseTypes$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:363)
- `TestListConnections_Execute_Pagination$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:753)
- `TestValidateSchema_LargeNumberOfTables` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:985)
- `TestListConnections_Execute_WithDateFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:491)
- `TestTestConnection_Execute_AllDatabaseTypes$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:415)
- `TestTestConnection_Execute_WithSSLConfiguration` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:337)
- `TestGetConnection_Execute_ConnectionWithSSL` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:291)
- `TestTestConnection_Execute_EmptyUUIDs` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:522)
- `TestGetConnectionSchema_Execute_NotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:115)
- `TestTestConnection_Execute_DecryptionError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:141)
- `TestListConnections_Execute_WithSortOrder$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:549)
- `TestTestConnection_Execute_RateLimitError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:183)
- `TestGetJob_Execute$5` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_job_test.go:108)
- `TestValidateSchema_EmptyConfigName` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:929)
- `TestGetProduct_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:40)
- `TestValidateSchema_DataSourceNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:88)
- `TestTestConnection_Execute_RateLimited` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:253)
- `TestValidateSchema_InvalidRequest_EmptyMappedFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:280)
- `TestListConnections_Execute_ConnectionWithSSL` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:796)
- `TestTestConnection_Execute_ConnectionWithAllFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:693)
- `TestGetProduct_Execute_TableDriven$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:254)
- `TestListProducts_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:317)
- `TestTestConnection_Execute_DifferentOrganizations` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:762)
- `TestListUnassignedConnections_Execute_TableDriven$6` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:245)
- `TestListUnassignedConnections_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:284)
- `TestListConnections_Execute_EmptyOrganizationID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:933)
- `TestListProducts_Execute_EmptyList` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:84)
- `TestListConnections_Execute_EmptyList` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:103)
- `TestListConnections_Execute_DifferentOrganizations` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:395)
- `TestGetConnection_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:168)
- `TestGetConnection_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:133)
- `TestGetConnection_Execute_TableDriven$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:245)
- `TestGetConnection_Execute_ConnectionWithAllFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:418)
- `TestGetProduct_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:177)
- `TestGetConnectionSchema_Execute_DataSourceFactoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:183)
- `TestTestConnection_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:95)
- `TestGetConnectionSchema_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:55)
- `TestListConnections_Execute_EmptyFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:686)
- `TestGetConnectionSchema_Execute_NilSchema` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:399)
- `TestListConnections_Execute_WithProductFilter_NotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:1024)
- `TestTestConnection_Execute_MultipleRepositoryErrors$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:484)
- `TestTestConnection_Execute_NotFoundError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:55)
- `TestValidateSchema_MultipleDatasources_AllValid` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:522)
- `TestValidateSchema_InvalidRequest_NilMappedFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:257)
- `TestValidateSchema_NilSchemaFromDatasource` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:902)
- `TestListConnections_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:350)
- `TestListConnections_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:55)
- `TestListConnections_Execute_TableDriven$7` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:308)
- `TestGetConnectionSchema_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:150)
- `TestListProducts_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:39)
- `TestListUnassignedConnections_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:103)
- `TestGetProduct_Execute_EmptyUUIDs` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:429)
- `TestValidateSchema_TableNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:135)
- `TestGetConnection_Execute_EmptyUUIDs` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:568)
- `TestValidateSchema_PartialConnectionsFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:411)
- `TestNewLoggerFromContext$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:57)
- `TestListUnassignedConnections_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:23)
- `TestValidateSchema_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:55)
- `TestGetConnectionSchema_Execute_GetSchemaInfoError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:219)
- `TestValidateSchema_FieldNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:181)
- `TestValidateSchema_CacheError_ContinuesToFetch` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:367)
- `TestGetConnectionSchema_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:483)
- `TestValidateSchema_PostgreSQLWithMixedQualifiedTables` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:718)
- `TestListProducts_Execute_ErrorScenarios$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:395)
- `TestGetProduct_Execute_ProductWithAllFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:300)
- `TestValidateSchema_PostgreSQLWithUnqualifiedTables` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:669)
- `TestGetProduct_Execute_NotFoundError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:89)
- `TestListUnassignedConnections_Execute_EmptyList` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:69)
- `TestValidateSchema_MultipleErrors` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:228)
- `TestGetProduct_Execute_MultipleRepositoryErrors$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:394)
- `TestValidateSchema_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:310)
- `TestListConnections_Execute_WithCursor` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:650)
- `TestListConnections_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:139)
- `TestListProducts_Execute_TableDriven$7` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:277)
- `TestGetConnectionSchema_Execute_FiltersSystemTables$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:338)
- `TestGetProduct_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_product_test.go:124)
- `TestListProducts_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_products_test.go:118)
- `TestListUnassignedConnections_Execute_ErrorScenarios$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_unassigned_connections_test.go:381)
- `TestValidateSchema_NonPostgreSQLDoesNotAddPublicSchema$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:783)
- `TestListConnections_Execute_ErrorScenarios$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:897)
- `TestListConnections_Execute_WithMetadataFilter` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:451)
- `TestTestConnection_Execute_RateLimitResetTime$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:612)
- `TestContextWithTracer$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:221)
- `TestTestConnection_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:227)
- `TestListConnections_Execute_WithProductFilter_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:966)
- `TestValidateSchema_CacheSetError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:843)
- `TestListConnections_Execute_WithProductFilter_RepoError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:1066)
- `TestContextWithLogger$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:167)
- `TestNewTracerFromContext$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:112)
- `TestListConnections_Execute_ConnectionTypes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:589)
- `TestGetConnection_Execute_MultipleRepositoryErrors$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:533)
- `TestValidateSchema_NoConnections_ReturnsSchemaEntityType` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:467)
- `TestTestConnection_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:293)
- `TestGetConnection_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:45)
- `TestGetConnection_Execute_NotFoundError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:98)
- `TestGetConnectionSchema_Execute_EmptySchema` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:441)

</details>

**Direct Callers:**

- `TestTestConnection_Execute_DecryptionKeyVersionMismatch` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:652` (call site: test_connection_test.go:652)
- `TestGetConnection_Execute_AllDatabaseTypes$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:363` (call site: get_connection_test.go:363)
- `TestListConnections_Execute_Pagination$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:753` (call site: list_connections_test.go:753)
- `TestValidateSchema_LargeNumberOfTables` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:985` (call site: validate_schema_test.go:985)
- `TestListConnections_Execute_WithDateFilters` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections_test.go:491` (call site: list_connections_test.go:491)
- `TestTestConnection_Execute_AllDatabaseTypes$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:415` (call site: test_connection_test.go:415)
- `TestTestConnection_Execute_WithSSLConfiguration` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:337` (call site: test_connection_test.go:337)
- `TestGetConnection_Execute_ConnectionWithSSL` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_test.go:291` (call site: get_connection_test.go:291)
- `TestTestConnection_Execute_EmptyUUIDs` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:522` (call site: test_connection_test.go:522)
- `TestGetConnectionSchema_Execute_NotFound` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection_schema_test.go:115` (call site: get_connection_schema_test.go:115)
- ... and 87 more

**Calls:**

- `Tracer` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel@v1.42.0/trace.go:15`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `WithValue` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:727`

---

### `NewTestConnection`

**File:** `components/manager/internal/services/query/test_connection.go`
**Risk Level:** HIGH
**Callers:** 16

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestTestConnection_Execute_DecryptionKeyVersionMismatch` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:650)
- `TestTestConnection_Execute_AllDatabaseTypes$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:413)
- `TestTestConnection_Execute_WithSSLConfiguration` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:335)
- `TestTestConnection_Execute_EmptyUUIDs` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:520)
- `TestTestConnection_Execute_DecryptionError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:139)
- `TestTestConnection_Execute_RateLimitError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:181)
- `TestTestConnection_Execute_RateLimited` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:251)
- `TestNewTestConnection` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:557)
- `TestTestConnection_Execute_ConnectionWithAllFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:691)
- `TestTestConnection_Execute_DifferentOrganizations` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:760)
- `TestTestConnection_Execute_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:93)
- `TestTestConnection_Execute_MultipleRepositoryErrors$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:482)
- `TestTestConnection_Execute_NotFoundError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:53)
- `TestTestConnection_Execute_RateLimitResetTime$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:610)
- `TestTestConnection_Execute_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:220)
- `TestTestConnection_Execute_OrganizationIsolation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:291)

</details>

**Direct Callers:**

- `TestTestConnection_Execute_DecryptionKeyVersionMismatch` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:650` (call site: test_connection_test.go:650)
- `TestTestConnection_Execute_AllDatabaseTypes$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:413` (call site: test_connection_test.go:413)
- `TestTestConnection_Execute_WithSSLConfiguration` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:335` (call site: test_connection_test.go:335)
- `TestTestConnection_Execute_EmptyUUIDs` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:520` (call site: test_connection_test.go:520)
- `TestTestConnection_Execute_DecryptionError` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:139` (call site: test_connection_test.go:139)
- `TestTestConnection_Execute_RateLimitError` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:181` (call site: test_connection_test.go:181)
- `TestTestConnection_Execute_RateLimited` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:251` (call site: test_connection_test.go:251)
- `TestNewTestConnection` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:557` (call site: test_connection_test.go:557)
- `TestTestConnection_Execute_ConnectionWithAllFields` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:691` (call site: test_connection_test.go:691)
- `TestTestConnection_Execute_DifferentOrganizations` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection_test.go:760` (call site: test_connection_test.go:760)
- ... and 6 more

**Calls:** None

---

### `TestTestConnection_Execute_WithSSLConfiguration`

**File:** `components/manager/internal/services/query/test_connection_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.mock.go:37`
- `NewMockCryptor` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/crypto/crypto.mock.go:32`
- `NewMockRateLimiterStore` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/rate_limiter_store.mock.go:26`
- ... and 27 more

---

### `TestTestConnection_Execute_AllDatabaseTypes`

**File:** `components/manager/internal/services/query/test_connection_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestTestConnection_Execute_Success`

**File:** `components/manager/internal/services/query/test_connection_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.mock.go:37`
- `NewMockCryptor` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/crypto/crypto.mock.go:32`
- `NewMockRateLimiterStore` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/rate_limiter_store.mock.go:26`
- ... and 29 more

---

### `NewValidateSchema`

**File:** `components/manager/internal/services/query/validate_schema.go`
**Risk Level:** HIGH
**Callers:** 20

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestValidateSchema_LargeNumberOfTables` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:983)
- `TestValidateSchema_EmptyConfigName` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:927)
- `TestValidateSchema_DataSourceNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:86)
- `TestValidateSchema_InvalidRequest_EmptyMappedFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:278)
- `TestValidateSchema_MultipleDatasources_AllValid` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:520)
- `TestValidateSchema_InvalidRequest_NilMappedFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:255)
- `TestValidateSchema_NilSchemaFromDatasource` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:897)
- `TestValidateSchema_TableNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:133)
- `TestValidateSchema_PartialConnectionsFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:409)
- `TestNewValidateSchema` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:443)
- `TestValidateSchema_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:53)
- `TestValidateSchema_FieldNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:179)
- `TestValidateSchema_CacheError_ContinuesToFetch` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:360)
- `TestValidateSchema_PostgreSQLWithMixedQualifiedTables` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:716)
- `TestValidateSchema_PostgreSQLWithUnqualifiedTables` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:667)
- `TestValidateSchema_MultipleErrors` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:226)
- `TestValidateSchema_RepositoryError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:308)
- `TestValidateSchema_NonPostgreSQLDoesNotAddPublicSchema$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:781)
- `TestValidateSchema_CacheSetError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:836)
- `TestValidateSchema_NoConnections_ReturnsSchemaEntityType` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:465)

</details>

**Direct Callers:**

- `TestValidateSchema_LargeNumberOfTables` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:983` (call site: validate_schema_test.go:983)
- `TestValidateSchema_EmptyConfigName` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:927` (call site: validate_schema_test.go:927)
- `TestValidateSchema_DataSourceNotFound` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:86` (call site: validate_schema_test.go:86)
- `TestValidateSchema_InvalidRequest_EmptyMappedFields` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:278` (call site: validate_schema_test.go:278)
- `TestValidateSchema_MultipleDatasources_AllValid` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:520` (call site: validate_schema_test.go:520)
- `TestValidateSchema_InvalidRequest_NilMappedFields` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:255` (call site: validate_schema_test.go:255)
- `TestValidateSchema_NilSchemaFromDatasource` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:897` (call site: validate_schema_test.go:897)
- `TestValidateSchema_TableNotFound` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:133` (call site: validate_schema_test.go:133)
- `TestValidateSchema_PartialConnectionsFound` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:409` (call site: validate_schema_test.go:409)
- `TestNewValidateSchema` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema_test.go:443` (call site: validate_schema_test.go:443)
- ... and 10 more

**Calls:** None

---

### `TestValidateSchema_CacheSetError`

**File:** `components/manager/internal/services/query/validate_schema_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.mock.go:37`
- `NewMockCryptor` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/crypto/crypto.mock.go:32`
- `NewMockSchemaCacheRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/cache/schema_cache.mock.go:28`
- ... and 34 more

---

### `TestValidateSchema_CacheError_ContinuesToFetch`

**File:** `components/manager/internal/services/query/validate_schema_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.mock.go:37`
- `NewMockCryptor` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/crypto/crypto.mock.go:32`
- `NewMockSchemaCacheRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/cache/schema_cache.mock.go:28`
- ... and 34 more

---

### `TestValidateSchema_NilSchemaFromDatasource`

**File:** `components/manager/internal/services/query/validate_schema_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.mock.go:37`
- `NewMockCryptor` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/crypto/crypto.mock.go:32`
- `NewMockSchemaCacheRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/cache/schema_cache.mock.go:28`
- ... and 34 more

---

### `NewConsumerRoutesWithAdapter`

**File:** `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go`
**Risk Level:** HIGH
**Callers:** 4

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestNewMultiQueueConsumerRegistersQueue` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:45)
- `TestMultiQueueConsumerRun$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:159)
- `TestMultiQueueConsumerRun$3` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:138)

</details>

**Direct Callers:**

- `TestNewMultiQueueConsumerRegistersQueue` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:45` (call site: consumer_service_test.go:45)
- `TestMultiQueueConsumerRun$4` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:159` (call site: consumer_service_test.go:159)
- `TestMultiQueueConsumerRun$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:138` (call site: consumer_service_test.go:138)
- `NewConsumerRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go:44` (call site: consumer.rabbitmq.go:44)

**Calls:** None

---

### `TestNewConsumerRoutes`

**File:** `components/worker/internal/adapters/rabbitmq/consumer_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockAdapter` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.mock.go:32`
- `NewNop` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/nil.go:9`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- ... and 1 more

---

### `TestConsumerRoutes_Shutdown`

**File:** `components/worker/internal/adapters/rabbitmq/consumer_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockAdapter` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.mock.go:32`
- `*github.com/LerianStudio/fetcher/pkg/rabbitmq.MockAdapter.EXPECT` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.mock.go:39`
- `Any` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/matchers.go:338`
- ... and 9 more

---

### `TestConsumerRoutes_Shutdown_Error`

**File:** `components/worker/internal/adapters/rabbitmq/consumer_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockAdapter` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.mock.go:32`
- `*github.com/LerianStudio/fetcher/pkg/rabbitmq.MockAdapter.EXPECT` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.mock.go:39`
- `Any` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/matchers.go:338`
- ... and 7 more

---

### `TestConsumerRoutes_RunConsumers`

**File:** `components/worker/internal/adapters/rabbitmq/consumer_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `NewPublisherRoutesWithAdapter`

**File:** `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go`
**Risk Level:** HIGH
**Callers:** 6

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestPublisherRoutes_Publish$2` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher_test.go:52)
- `TestNewPublisherRoutes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher_test.go:23)
- `TestPublisherRoutes_Shutdown$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher_test.go:70)
- `TestPublisherRoutes_Publish$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher_test.go:42)
- `TestPublisherRoutes_Shutdown$2` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher_test.go:78)

</details>

**Direct Callers:**

- `TestPublisherRoutes_Publish$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher_test.go:52` (call site: publisher_test.go:52)
- `TestNewPublisherRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher_test.go:23` (call site: publisher_test.go:23)
- `TestPublisherRoutes_Shutdown$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher_test.go:70` (call site: publisher_test.go:70)
- `NewPublisherRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go:37` (call site: publisher.rabbitmq.go:37)
- `TestPublisherRoutes_Publish$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher_test.go:42` (call site: publisher_test.go:42)
- `TestPublisherRoutes_Shutdown$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher_test.go:78` (call site: publisher_test.go:78)

**Calls:** None

---

### `TestPublisherRoutes_Shutdown`

**File:** `components/worker/internal/adapters/rabbitmq/publisher_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockAdapter` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.mock.go:32`
- `NewNop` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/nil.go:9`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- ... and 2 more

---

### `TestNewPublisherRoutes`

**File:** `components/worker/internal/adapters/rabbitmq/publisher_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockAdapter` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.mock.go:32`
- `NewNop` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/nil.go:9`
- `NewPublisherRoutesWithAdapter` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go:41`
- ... and 2 more

---

### `TestPublisherRoutes_Publish`

**File:** `components/worker/internal/adapters/rabbitmq/publisher_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockAdapter` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.mock.go:32`
- `NewNop` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/nil.go:9`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- ... and 2 more

---

### `InitWorker`

**File:** `components/worker/internal/bootstrap/config.go`
**Risk Level:** HIGH
**Callers:** 3

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestInitWorker_PanicsWhenTelemetryGlobalsFail` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:181)
- `TestInitWorker_PanicsWhenLoggerInitFails` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:136)
- `TestInitWorker_PanicsWhenConfigLoadFails` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:103)

</details>

**Direct Callers:**

- `TestInitWorker_PanicsWhenTelemetryGlobalsFail` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:181` (call site: config_test.go:181)
- `TestInitWorker_PanicsWhenLoggerInitFails` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:136` (call site: config_test.go:136)
- `TestInitWorker_PanicsWhenConfigLoadFails` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:103` (call site: config_test.go:103)

**Calls:**

- `NullAssignTo` at `/Users/fredamaral/go/pkg/mod/github.com/jackc/pgx/v5@v5.8.0/pgtype/convert.go:7`
- `TestInitWorker_PanicsWhenConfigLoadFails$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:88`
- `UnmarshalJSON$1` at `/Users/fredamaral/go/pkg/mod/go.uber.org/zap@v1.27.1/zapcore/encoder.go:222`
- `SetConfigFromEnvVars` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons@v1.18.0/commons/os.go:99`
- `SetConfigFromEnvVars` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v2@v2.6.2/commons/os.go:99`
- ... and 128 more

---

### `must`

**File:** `components/worker/internal/bootstrap/config.go`
**Risk Level:** HIGH
**Callers:** 13

**Test Coverage:** No tests found

**Direct Callers:**

- `ForEachPackage` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/allpackages.go:68` (call site: allpackages.go:68)
- `initPlatformDependencies` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:245` (call site: config.go:245)
- `initCrypto` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:197` (call site: config.go:197)
- `initCrypto` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:200` (call site: config.go:200)
- `initCrypto` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:204` (call site: config.go:204)
- `initCrypto` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:207` (call site: config.go:207)
- `initMongoRepositories` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:171` (call site: config.go:171)
- `initMongoRepositories` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:174` (call site: config.go:174)
- `initMongoRepositories` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:176` (call site: config.go:176)
- `initMongoRepositories` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:179` (call site: config.go:179)
- ... and 3 more

**Calls:**

- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`

---

### `TestInitWorker_PanicsWhenTelemetryGlobalsFail`

**File:** `components/worker/internal/bootstrap/config_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.common.Cleanup` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1296`
- `TestInitWorker_PanicsWhenTelemetryGlobalsFail$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:170`
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:81`

---

### `testBootstrapLogger`

**File:** `components/worker/internal/bootstrap/config_test.go`
**Risk Level:** HIGH
**Callers:** 7

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestNewMultiQueueConsumerRegistersQueue` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:45)
- `TestMultiQueueConsumerRun$4` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:159)
- `TestServiceRun$3` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:218)
- `TestMultiQueueConsumerRun$3` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:138)
- `TestInitWorker_PanicsWhenTelemetryGlobalsFail$3` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:159)
- `TestServiceRun$2` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:178)

</details>

**Direct Callers:**

- `TestNewMultiQueueConsumerRegistersQueue` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:45` (call site: consumer_service_test.go:45)
- `contextWithBootstrapTracking` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:239` (call site: consumer_service_test.go:239)
- `TestMultiQueueConsumerRun$4` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:159` (call site: consumer_service_test.go:159)
- `TestServiceRun$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:218` (call site: consumer_service_test.go:218)
- `TestMultiQueueConsumerRun$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:138` (call site: consumer_service_test.go:138)
- `TestInitWorker_PanicsWhenTelemetryGlobalsFail$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:159` (call site: config_test.go:159)
- `TestServiceRun$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer_service_test.go:178` (call site: consumer_service_test.go:178)

**Calls:** None

---

### `TestResolveZapEnvironment`

**File:** `components/worker/internal/bootstrap/config_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Parallel` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1758`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestMust`

**File:** `components/worker/internal/bootstrap/config_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Parallel` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1758`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestInitWorker_PanicsWhenConfigLoadFails`

**File:** `components/worker/internal/bootstrap/config_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.common.Cleanup` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1296`
- `TestInitWorker_PanicsWhenConfigLoadFails$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:92`
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:81`

---

### `TestInitWorker_PanicsWhenLoggerInitFails`

**File:** `components/worker/internal/bootstrap/config_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.common.Cleanup` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1296`
- `TestInitWorker_PanicsWhenLoggerInitFails$4` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config_test.go:125`
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:81`

---

### `TestQueryPluginCRMCollectionWithFilters_NoFilters`

**File:** `components/worker/internal/services/extract_crm_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `*testing.common.Cleanup` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1296`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- ... and 5 more

---

### `TestQueryPluginCRM_WithOrganizationOnly`

**File:** `components/worker/internal/services/extract_crm_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `*testing.common.Cleanup` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1296`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- ... and 6 more

---

### `TestQueryPluginCRM_WithFilters`

**File:** `components/worker/internal/services/extract_crm_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `*testing.common.Cleanup` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1296`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- ... and 6 more

---

### `TestProcessPluginCRMCollection_WithValidOrganization`

**File:** `components/worker/internal/services/extract_crm_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `*testing.common.Cleanup` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1296`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- ... and 697 more

---

### `TestProcessPluginCRMCollection_WithOrganizationID`

**File:** `components/worker/internal/services/extract_crm_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `*testing.common.Cleanup` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1296`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- ... and 6 more

---

### `TestExtractJobIDFromMultipleSources_EdgeCases`

**File:** `components/worker/internal/services/extract_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestExtractJobIDFromMultipleSources_FromHeaders`

**File:** `components/worker/internal/services/extract_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- `testLogger` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:33`
- ... and 8 more

---

### `TestExtractJobIDFromPartialJSON`

**File:** `components/worker/internal/services/extract_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestExtractJobIDFromMultipleSources_FromPartialJSON`

**File:** `components/worker/internal/services/extract_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- `testLogger` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:33`
- ... and 8 more

---

### `TestExtractJobIDFromMultipleSources_NoIDs`

**File:** `components/worker/internal/services/extract_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- `testLogger` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:33`
- ... and 4 more

---

### `TestExtractJobIDFromPartialJSON_ValidJobIDInvalidOrgID`

**File:** `components/worker/internal/services/extract_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- `testLogger` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:33`
- ... and 4 more

---

### `TestExtractJobIDFromPartialJSON_RegexFallback`

**File:** `components/worker/internal/services/extract_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- `testLogger` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:33`
- ... and 4 more

---

### `TestExtractJobIDFromMultipleSources_HeaderPrecedence`

**File:** `components/worker/internal/services/extract_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- `testLogger` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:33`
- ... and 7 more

---

### `TestParseMessage_JSONNull`

**File:** `components/worker/internal/services/extract_data_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `newTestMocks` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:48`
- `newTestUseCase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:61`
- `testContext` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test_helpers_test.go:21`
- ... and 4 more

---

### `testLogger`

**File:** `components/worker/internal/services/test_helpers_test.go`
**Risk Level:** HIGH
**Callers:** 80

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestTransformPluginCRMAdvancedFilters_AllConditionTypes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1027)
- `TestPublishJobNotification_WithCompletedAt` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:451)
- `TestDecryptRecord_EmptyRecord` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:493)
- `TestDecryptPluginCRMData_WithNestedField` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:221)
- `TestSaveExternalDataToSeaweedFS_MarshalError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1502)
- `TestDecryptFieldValue_EdgeCases` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:406)
- `TestShouldSkipProcessing$7` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:369)
- `TestPublishJobNotification_EmptyExchange` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:312)
- `TestQueryDatabase_ConnectionFoundButDifferentConfigName` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1452)
- `TestDecryptPluginCRMData_NoDecryptionNeeded` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:158)
- `TestParseMessage_WithNilError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:956)
- `TestCheckReportStatus_JobDataNil` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1256)
- `TestParseMessage_InvalidJSON` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:71)
- `TestPublishJobNotification_WithErrorMetadata` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:69)
- `TestDecryptBankingDetailsFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:757)
- `TestPublishJobNotification_MetadataPreservation` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:336)
- `TestTransformPluginCRMAdvancedFilters_WithEncryptedFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1642)
- `TestPublishJobNotification_UnknownSource` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:183)
- `TestProcessPluginCRMCollection_WithValidOrganization` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1762)
- `TestExtractJobIDFromMultipleSources_EdgeCases$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:667)
- `TestExtractJobIDFromMultipleSources_FromPartialJSON` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:175)
- `TestParseMessage_InvalidJSONWithJobIDInHeaders` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:115)
- `TestPublishJobNotification_WithResultData` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:247)
- `TestDecryptNaturalPersonFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:824)
- `TestDecryptLegalPersonFields_WithNilLegalPerson` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1462)
- `TestHashFilterValues_EdgeCases` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:342)
- `TestQueryPluginCRM_NilCollections` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1272)
- `TestEncryptDataForSeaweedFS_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1878)
- `TestTransformPluginCRMAdvancedFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:856)
- `TestPublishJobNotification_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:23)
- `TestQueryPluginCRMCollectionWithFilters_NoFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1154)
- `TestQueryPluginCRM_EmptyCollections` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1241)
- `TestSaveExternalDataToSeaweedFS_SeaweedFSPutError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1573)
- `TestParseMessage_EmptyBody` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1307)
- `TestDecryptRecord_WithAllFieldTypes` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1346)
- `TestDecryptContactFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:723)
- `TestExtractJobIDFromMultipleSources_FromHeaders` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:147)
- `TestQueryPluginCRM_WithFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1686)
- `TestSaveExternalDataToSeaweedFS_MissingEnvVars` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1537)
- `TestCheckReportStatus$5` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:755)
- `TestPublishJobNotification_PublishError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:150)
- `TestDecryptPluginCRMData_MissingEncryptSecretKey` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1211)
- `TestParseMessage_JSONNull` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:94)
- `TestQueryPluginCRM_WithOrganizationOnly` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1308)
- `TestDecryptPluginCRMData_WithValidCrypto` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1610)
- `TestDecryptNestedFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:615)
- `TestProcessPluginCRMCollection_WithOrganizationID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1100)
- `TestHashFilterValues_ConsistentHashing` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1583)
- `TestPublishJobNotification_RoutingKeyGeneration$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:422)
- `TestExtractJobIDFromMultipleSources_HeaderPrecedence` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1328)
- `TestDecryptBankingDetailsFields_WithNilBankingDetails` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1432)
- `TestDecryptLegalPersonFields_WithEmptyRepresentative` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1522)
- `TestExtractJobIDFromMultipleSources_NoIDs` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:201)
- `TestDecryptPluginCRMData_EmptyResult` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:245)
- `TestDecryptPluginCRMData_WithEncryptedFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:197)
- `TestTransformPluginCRMAdvancedFilters_FieldMappings` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:927)
- `TestHandleErrorWithUpdate$3` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:925)
- `TestPublishJobNotification_PublisherNotConfigured` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:126)
- `TestSaveExternalDataToSeaweedFS_DocumentSignerError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_additional_test.go:233)
- `TestEncryptDataForSeaweedFS` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1083)
- `TestQueryDatabase_WithFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1995)
- `TestDecryptNaturalPersonFields_WithNilNaturalPerson` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1492)
- `TestExtractJobIDFromPartialJSON$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:568)
- `TestSaveExternalDataToSeaweedFS_Success` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1616)
- `TestPublishJobNotification_WithAllOptions` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:492)
- `TestDecryptPluginCRMData_MissingHashSecretKey` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1183)
- `TestDecryptTopLevelFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:552)
- `TestDecryptLegalPersonFields` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:789)
- `TestDecryptContactFields_WithNilContact` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1402)
- `TestParseMessage_UpdateStatusError` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1011)
- `TestHashFilterValues` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:73)
- `TestExtractJobIDFromPartialJSON_RegexFallback` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:987)
- `TestSaveExternalDataToSeaweedFS_EmptyResult` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1948)
- `TestExtractJobIDFromPartialJSON_ValidJobIDInvalidOrgID` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1281)
- `TestDecryptFieldValue_WithValidEncryptedValue` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1725)
- `TestParseMessage_ValidMessage` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:43)
- `TestQueryDatabase_DataSourceFactoryAndLifecycleErrors$1` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_additional_test.go:179)
- `TestQueryDatabase_ConnectionNotFound` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1411)
- `TestEncryptDataForSeaweedFS_InvalidCipherInitialization` (/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1857)

</details>

**Direct Callers:**

- `TestTransformPluginCRMAdvancedFilters_AllConditionTypes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:1027` (call site: extract_crm_data_test.go:1027)
- `TestPublishJobNotification_WithCompletedAt` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:451` (call site: job_notification_test.go:451)
- `TestDecryptRecord_EmptyRecord` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:493` (call site: extract_crm_data_test.go:493)
- `TestDecryptPluginCRMData_WithNestedField` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:221` (call site: extract_crm_data_test.go:221)
- `TestSaveExternalDataToSeaweedFS_MarshalError` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1502` (call site: extract_data_test.go:1502)
- `TestDecryptFieldValue_EdgeCases` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:406` (call site: extract_crm_data_test.go:406)
- `TestShouldSkipProcessing$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:369` (call site: extract_data_test.go:369)
- `TestPublishJobNotification_EmptyExchange` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job_notification_test.go:312` (call site: job_notification_test.go:312)
- `TestQueryDatabase_ConnectionFoundButDifferentConfigName` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_data_test.go:1452` (call site: extract_data_test.go:1452)
- `TestDecryptPluginCRMData_NoDecryptionNeeded` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract_crm_data_test.go:158` (call site: extract_crm_data_test.go:158)
- ... and 70 more

**Calls:** None

---

### `TestCustomContextKey_Type`

**File:** `pkg/context_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestNewLoggerFromContext`

**File:** `pkg/context_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestContextWithLogger`

**File:** `pkg/context_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestContextWithTracer`

**File:** `pkg/context_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewTracerProvider` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/trace@v1.42.0/noop/noop.go:36`
- `go.opentelemetry.io/otel/trace/noop.TracerProvider.Tracer` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/trace@v1.42.0/noop/noop.go:41`
- `NewTracerProvider` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/trace@v1.42.0/noop/noop.go:36`
- `go.opentelemetry.io/otel/trace/noop.TracerProvider.Tracer` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/trace@v1.42.0/noop/noop.go:41`
- `NewTracerProvider` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/trace@v1.42.0/noop/noop.go:36`
- ... and 4 more

---

### `TestContextWithLoggerAndTracer_Combined`

**File:** `pkg/context_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestCustomContextKeyValue_Integration`

**File:** `pkg/context_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `NewDataSourceFromConnection`

**File:** `pkg/datasource/datasource-factory.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `*github.com/LerianStudio/fetcher/components/manager/internal/services/command.CreateFetcherJob.TestConnection` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job.go:366)
- `*github.com/LerianStudio/fetcher/components/manager/internal/services/command.CreateFetcherJob.TestConnection` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job.go:366)

</details>

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/components/manager/internal/services/query.TestConnection.Execute` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection.go:124` (call site: test_connection.go:124)
- `*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.queryDatabase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract-data.go:384` (call site: extract-data.go:384)
- `*github.com/LerianStudio/fetcher/components/manager/internal/services/command.CreateFetcherJob.TestConnection` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job.go:366` (call site: create_fetcher_job.go:366)
- `*github.com/LerianStudio/fetcher/components/manager/internal/services/query.TestConnection.Execute` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test_connection.go:124` (call site: test_connection.go:124)
- `*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.queryDatabase` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract-data.go:384` (call site: extract-data.go:384)
- `*github.com/LerianStudio/fetcher/components/manager/internal/services/query.ValidateSchema.getOrFetchSchema` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema.go:257` (call site: validate_schema.go:257)
- `*github.com/LerianStudio/fetcher/components/manager/internal/services/query.ValidateSchema.getOrFetchSchema` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate_schema.go:257` (call site: validate_schema.go:257)
- `*github.com/LerianStudio/fetcher/components/manager/internal/services/command.CreateFetcherJob.TestConnection` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_fetcher_job.go:366` (call site: create_fetcher_job.go:366)
- `NewDataSourceFromConnectionWithLogger$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:454` (call site: datasource-factory.go:454)

**Calls:**

- `newDataSourceFromConnection` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:59`

---

### `generateImageTag`

**File:** `pkg/itestkit/addons/e2ekit/build_secrets.go`
**Risk Level:** HIGH
**Callers:** 21

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuildConfigHelpers` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:88)
- `TestBuildConfigHelpers` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:89)

</details>

**Direct Callers:**

- `*github.com/xdg-go/scram.ServerConversation.firstMsg` at `/Users/fredamaral/go/pkg/mod/github.com/xdg-go/scram@v1.2.0/server_conv.go:172` (call site: server_conv.go:172)
- `*go.mongodb.org/mongo-driver/mongo/options.ClientOptions.ApplyURI` at `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/mongo/options/clientoptions.go:577` (call site: clientoptions.go:577)
- `*github.com/microsoft/go-mssqldb.tdsSession.Log` at `/Users/fredamaral/go/pkg/mod/github.com/microsoft/go-mssqldb@v1.9.6/session.go:92` (call site: session.go:92)
- `go.uber.org/mock/gomock.StringerFunc.String` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/matchers.go:58` (call site: matchers.go:58)
- `run$1` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/benchmark.go:282` (call site: benchmark.go:282)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167` (call site: process.go:167)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167` (call site: process.go:167)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167` (call site: process.go:167)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeNameDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:156` (call site: process.go:156)
- `*github.com/LerianStudio/fetcher/tests/fuzz/shared/mocks.MockCryptor.KeyVersion` at `/Users/fredamaral/repos/lerianstudio/fetcher/tests/fuzz/shared/mocks/mocks.go:39` (call site: mocks.go:39)
- ... and 11 more

**Calls:**

- `Read` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/crypto/rand/rand.go:47`
- `EncodeToString` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/encoding/hex/hex.go:126`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`

---

### `uniqueAppend`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** HIGH
**Callers:** 6

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuilderHelpersAndProjectRoot$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:192)
- `TestBuilderHelpersAndProjectRoot$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:193)

</details>

**Direct Callers:**

- `github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.waitHTTP.Configure` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:298` (call site: builder.go:298)
- `github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.waitPort.Configure` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:340` (call site: builder.go:340)
- `findPath$1` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/loader/loader.go:917` (call site: loader.go:917)
- `*golang.org/x/tools/go/loader.importer.findPath` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/loader/loader.go:924` (call site: loader.go:924)
- `TestBuilderHelpersAndProjectRoot$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:192` (call site: helpers_test.go:192)
- `TestBuilderHelpersAndProjectRoot$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:193` (call site: helpers_test.go:193)

**Calls:** None

---

### `rewriteLocalhostForContainer`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** HIGH
**Callers:** 15

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuilderHelpersAndProjectRoot$4$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:179)

</details>

**Direct Callers:**

- `*go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.42.0/internal/envconfig/envconfig.go:46` (call site: envconfig.go:46)
- `ProcessFiles` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/internal/cgo/cgo.go:79` (call site: cgo.go:79)
- `TestBuilderHelpersAndProjectRoot$4$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:179` (call site: helpers_test.go:179)
- `WithServerAppName$1` at `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/x/mongo/driver/topology/server_options.go:96` (call site: server_options.go:96)
- `github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.localhostToHostGatewayRewriter.Rewrite` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:373` (call site: builder.go:373)
- `addStdlibCandidates$2` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/internal/imports/fix.go:1121` (call site: fix.go:1121)
- `ParseFile` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/util.go:41` (call site: util.go:41)
- `FakeContext$2` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:48` (call site: fakecontext.go:48)
- `sanitiseName` at `/Users/fredamaral/go/pkg/mod/github.com/mdelapenya/tlscert@v0.2.0/tlscert.go:321` (call site: tlscert.go:321)
- `*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@v1.42.0/internal/envconfig/envconfig.go:46` (call site: envconfig.go:46)
- ... and 5 more

**Calls:**

- `HostGatewayIP` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers.go:92`
- `Parse` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:399`
- `*net/url.URL.Hostname` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:1179`
- `*net/url.URL.Port` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:1187`
- `*net/url.URL.String` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:794`
- ... and 10 more

---

### `cloneMap`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** HIGH
**Callers:** 4

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuilderHelpersAndProjectRoot$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:199)

</details>

**Direct Callers:**

- `github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.stubRewriter.Rewrite` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:17` (call site: helpers_test.go:17)
- `*github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.Builder.Run` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:188` (call site: builder.go:188)
- `github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.localhostToHostGatewayRewriter.Rewrite` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:371` (call site: builder.go:371)
- `TestBuilderHelpersAndProjectRoot$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:199` (call site: helpers_test.go:199)

**Calls:** None

---

### `ProjectRoot`

**File:** `pkg/itestkit/addons/e2ekit/helpers.go`
**Risk Level:** HIGH
**Callers:** 19

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuilderHelpersAndProjectRoot$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:123)

</details>

**Direct Callers:**

- `*github.com/xdg-go/scram.ServerConversation.firstMsg` at `/Users/fredamaral/go/pkg/mod/github.com/xdg-go/scram@v1.2.0/server_conv.go:172` (call site: server_conv.go:172)
- `*go.mongodb.org/mongo-driver/mongo/options.ClientOptions.ApplyURI` at `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/mongo/options/clientoptions.go:577` (call site: clientoptions.go:577)
- `*github.com/microsoft/go-mssqldb.tdsSession.Log` at `/Users/fredamaral/go/pkg/mod/github.com/microsoft/go-mssqldb@v1.9.6/session.go:92` (call site: session.go:92)
- `go.uber.org/mock/gomock.StringerFunc.String` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/matchers.go:58` (call site: matchers.go:58)
- `run$1` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/benchmark.go:282` (call site: benchmark.go:282)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167` (call site: process.go:167)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167` (call site: process.go:167)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167` (call site: process.go:167)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeNameDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:156` (call site: process.go:156)
- `*github.com/LerianStudio/fetcher/tests/fuzz/shared/mocks.MockCryptor.KeyVersion` at `/Users/fredamaral/repos/lerianstudio/fetcher/tests/fuzz/shared/mocks/mocks.go:39` (call site: mocks.go:39)
- ... and 9 more

**Calls:**

- `*sync.Once.Do` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/sync/once.go:52`

---

### `ProjectRootFrom`

**File:** `pkg/itestkit/addons/e2ekit/helpers.go`
**Risk Level:** HIGH
**Callers:** 16

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuilderHelpersAndProjectRoot$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:111)
- `TestBuilderHelpersAndProjectRoot$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:115)
- `TestBuilderHelpersAndProjectRoot$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:119)

</details>

**Direct Callers:**

- `*go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.42.0/internal/envconfig/envconfig.go:46` (call site: envconfig.go:46)
- `ProcessFiles` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/internal/cgo/cgo.go:79` (call site: cgo.go:79)
- `WithServerAppName$1` at `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/x/mongo/driver/topology/server_options.go:96` (call site: server_options.go:96)
- `addStdlibCandidates$2` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/internal/imports/fix.go:1121` (call site: fix.go:1121)
- `ParseFile` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/util.go:41` (call site: util.go:41)
- `FakeContext$2` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:48` (call site: fakecontext.go:48)
- `sanitiseName` at `/Users/fredamaral/go/pkg/mod/github.com/mdelapenya/tlscert@v0.2.0/tlscert.go:321` (call site: tlscert.go:321)
- `*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@v1.42.0/internal/envconfig/envconfig.go:46` (call site: envconfig.go:46)
- `FakeContext$4` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:72` (call site: fakecontext.go:72)
- `Expand` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:35` (call site: env.go:35)
- ... and 6 more

**Calls:**

- `Stat` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/stat.go:11`
- `Dir` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/path/filepath/path.go:466`
- `*archive/tar.headerFileInfo.IsDir` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/archive/tar/common.go:550`
- `*golang.org/x/tools/go/buildutil.fakeDirInfo.IsDir` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:109`
- `golang.org/x/tools/go/buildutil.fakeDirInfo.IsDir` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:109`
- ... and 14 more

---

### `compareValues`

**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** HIGH
**Callers:** 4

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCompareValues$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher_test.go:338)

</details>

**Direct Callers:**

- `MatchJSONField$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher.go:159` (call site: matcher.go:159)
- `TestCompareValues$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher_test.go:338` (call site: matcher_test.go:338)
- `*internal/sync.HashTrieMap[any, any].iter[any any]` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/internal/sync/hashtriemap.go:512` (call site: hashtriemap.go:512)
- `AssertJSONField` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions.go:284` (call site: assertions.go:284)

**Calls:** None

---

### `portKey`

**File:** `pkg/itestkit/container_generic.go`
**Risk Level:** HIGH
**Callers:** 14

**Test Coverage:** No tests found

**Direct Callers:**

- `*go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.42.0/internal/envconfig/envconfig.go:46` (call site: envconfig.go:46)
- `ProcessFiles` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/internal/cgo/cgo.go:79` (call site: cgo.go:79)
- `WithServerAppName$1` at `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/x/mongo/driver/topology/server_options.go:96` (call site: server_options.go:96)
- `addStdlibCandidates$2` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/internal/imports/fix.go:1121` (call site: fix.go:1121)
- `ParseFile` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/util.go:41` (call site: util.go:41)
- `FakeContext$2` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:48` (call site: fakecontext.go:48)
- `sanitiseName` at `/Users/fredamaral/go/pkg/mod/github.com/mdelapenya/tlscert@v0.2.0/tlscert.go:321` (call site: tlscert.go:321)
- `*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@v1.42.0/internal/envconfig/envconfig.go:46` (call site: envconfig.go:46)
- `FakeContext$4` at `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:72` (call site: fakecontext.go:72)
- `Expand` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:35` (call site: env.go:35)
- ... and 4 more

**Calls:** None

---

### `uniqueAppendMany`

**File:** `pkg/itestkit/customizer_options.go`
**Risk Level:** HIGH
**Callers:** 5

**Test Coverage:** No tests found

**Direct Callers:**

- `CHostDockerInternal$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options.go:63` (call site: customizer_options.go:63)
- `CBindMount$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options.go:125` (call site: customizer_options.go:125)
- `CExposedPorts$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options.go:31` (call site: customizer_options.go:31)
- `CNetworks$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options.go:100` (call site: customizer_options.go:100)
- `CNetworkAliases$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options.go:116` (call site: customizer_options.go:116)

**Calls:** None

---

### `CNetworkAliases`

**File:** `pkg/itestkit/customizer_options.go`
**Risk Level:** HIGH
**Callers:** 6

**Test Coverage:** No tests found

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mysql.MySQLInfra.Start` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mysql/mysql.go:100` (call site: mysql.go:100)
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mongodb.MongoDBInfra.Start` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mongodb/mongodb.go:85` (call site: mongodb.go:85)
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/redis.RedisInfra.Start` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/redis/redis.go:75` (call site: redis.go:75)
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mssql.MSSQLInfra.Start` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mssql/mssql.go:85` (call site: mssql.go:85)
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/rabbitmq.RabbitInfra.Start` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/rabbitmq/rabbitmq.go:89` (call site: rabbitmq.go:89)
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/postgres.PostgresInfra.Start` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/postgres/postgres.go:94` (call site: postgres.go:94)

**Calls:** None

---

### `HostGatewayIP`

**File:** `pkg/itestkit/hostport.go`
**Risk Level:** HIGH
**Callers:** 19

**Test Coverage:** No tests found

**Direct Callers:**

- `*github.com/xdg-go/scram.ServerConversation.firstMsg` at `/Users/fredamaral/go/pkg/mod/github.com/xdg-go/scram@v1.2.0/server_conv.go:172` (call site: server_conv.go:172)
- `*go.mongodb.org/mongo-driver/mongo/options.ClientOptions.ApplyURI` at `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/mongo/options/clientoptions.go:577` (call site: clientoptions.go:577)
- `*github.com/microsoft/go-mssqldb.tdsSession.Log` at `/Users/fredamaral/go/pkg/mod/github.com/microsoft/go-mssqldb@v1.9.6/session.go:92` (call site: session.go:92)
- `go.uber.org/mock/gomock.StringerFunc.String` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/matchers.go:58` (call site: matchers.go:58)
- `run$1` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/benchmark.go:282` (call site: benchmark.go:282)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167` (call site: process.go:167)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167` (call site: process.go:167)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167` (call site: process.go:167)
- `go.opentelemetry.io/otel/sdk/resource.processRuntimeNameDetector.Detect` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:156` (call site: process.go:156)
- `*github.com/LerianStudio/fetcher/tests/fuzz/shared/mocks.MockCryptor.KeyVersion` at `/Users/fredamaral/repos/lerianstudio/fetcher/tests/fuzz/shared/mocks/mocks.go:39` (call site: mocks.go:39)
- ... and 9 more

**Calls:**

- `*sync.Once.Do` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/sync/once.go:52`

---

### `ParseHostPort`

**File:** `pkg/itestkit/hostport.go`
**Risk Level:** HIGH
**Callers:** 4

**Test Coverage:** No tests found

**Direct Callers:**

- `ResolveContainerHostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:126` (call site: hostport.go:126)
- `ResolveContainerHostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:133` (call site: hostport.go:133)
- `ResolveHostHostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:113` (call site: hostport.go:113)
- `ResolveHostHostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:116` (call site: hostport.go:116)

**Calls:**

- `SplitHostPort` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/ipsock.go:165`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `Atoi` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/strconv/number.go:146`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`

---

### `ResolveHostHostPort`

**File:** `pkg/itestkit/hostport.go`
**Risk Level:** HIGH
**Callers:** 4

**Test Coverage:** No tests found

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/redis.RedisInfra.HostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/redis/redis.go:178` (call site: redis.go:178)
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/postgres.PostgresInfra.HostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/postgres/postgres.go:184` (call site: postgres.go:184)
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mongodb.MongoDBInfra.HostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mongodb/mongodb.go:182` (call site: mongodb.go:182)
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/rabbitmq.RabbitInfra.HostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/rabbitmq/rabbitmq.go:174` (call site: rabbitmq.go:174)

**Calls:**

- `ParseHostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:95`
- `ParseHostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:95`

---

### `NewSeaweedFSInfra`

**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go`
**Risk Level:** HIGH
**Callers:** 3

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestSeaweedFSHelpersWithoutDocker$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:46)
- `TestSeaweedFSHelpersWithoutDocker$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:136)
- `TestSeaweedFSHelpersWithoutDocker$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:23)

</details>

**Direct Callers:**

- `TestSeaweedFSHelpersWithoutDocker$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:46` (call site: seaweedfs_test.go:46)
- `TestSeaweedFSHelpersWithoutDocker$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:136` (call site: seaweedfs_test.go:136)
- `TestSeaweedFSHelpersWithoutDocker$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:23` (call site: seaweedfs_test.go:23)

**Calls:** None

---

### `NewConnectionMongoDBRepository`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** HIGH
**Callers:** 4

**Test Coverage:** No tests found

**Direct Callers:**

- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:196` (call site: config.go:196)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:196` (call site: config.go:196)
- `initMongoRepositories` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:173` (call site: config.go:173)
- `initMongoRepositories` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:173` (call site: config.go:173)

**Calls:**

- `New` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/errors/errors.go:64`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `WithTimeout` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:703`
- `init$5` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/crypto/internal/fips140/ecdsa/cast.go:66`
- `init$bound` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/dnsclient_unix.go:374`
- ... and 2330 more

---

### `TestConnectionMongoDBRepository_Create`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `newConnectionRepository`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 30

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestConnectionMongoDBRepository_FindByOrganizationAndDatabaseName$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:522)
- `TestConnectionMongoDBRepository_EnsureIndexes$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:722)
- `TestConnectionMongoDBRepository_DropIndexes$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:760)
- `TestConnectionMongoDBRepository_FindByConfigNames$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:644)
- `TestConnectionMongoDBRepository_Delete$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:379)
- `TestConnectionMongoDBRepository_Update$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:272)
- `TestConnectionMongoDBRepository_FindByConfigNames$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:655)
- `TestConnectionMongoDBRepository_Update$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:259)
- `TestConnectionMongoDBRepository_Delete$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:357)
- `TestConnectionMongoDBRepository_Update$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:296)
- `TestConnectionMongoDBRepository_Create$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:167)
- `TestConnectionMongoDBRepository_Create$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:198)
- `TestConnectionMongoDBRepository_FindByOrganizationAndName$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:471)
- `TestConnectionMongoDBRepository_FindByOrganizationAndName$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:483)
- `TestConnectionMongoDBRepository_FindByOrganizationAndDatabaseName$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:546)
- `TestConnectionMongoDBRepository_Create$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:179)
- `TestConnectionMongoDBRepository_FindByConfigNames$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:614)
- `TestConnectionMongoDBRepository_EnsureIndexes$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:730)
- `TestConnectionMongoDBRepository_FindByConfigNames$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:625)
- `TestConnectionMongoDBRepository_FindByID$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:420)
- `TestConnectionMongoDBRepository_Create$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:155)
- `TestConnectionMongoDBRepository_FindByOrganizationAndDatabaseName$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:534)
- `TestConnectionMongoDBRepository_Update$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:316)
- `TestConnectionMongoDBRepository_FindByConfigNames$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:678)
- `TestConnectionMongoDBRepository_FindByConfigNames$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:585)
- `TestConnectionMongoDBRepository_Update$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:234)
- `TestConnectionMongoDBRepository_List$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:855)
- `TestConnectionMongoDBRepository_FindByID$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:432)
- `TestConnectionMongoDBRepository_List$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:822)
- `TestConnectionMongoDBRepository_Update$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:284)

</details>

**Direct Callers:**

- `TestConnectionMongoDBRepository_FindByOrganizationAndDatabaseName$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:522` (call site: connection.mongodb_test.go:522)
- `TestConnectionMongoDBRepository_EnsureIndexes$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:722` (call site: connection.mongodb_test.go:722)
- `TestConnectionMongoDBRepository_DropIndexes$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:760` (call site: connection.mongodb_test.go:760)
- `TestConnectionMongoDBRepository_FindByConfigNames$4` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:644` (call site: connection.mongodb_test.go:644)
- `TestConnectionMongoDBRepository_Delete$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:379` (call site: connection.mongodb_test.go:379)
- `TestConnectionMongoDBRepository_Update$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:272` (call site: connection.mongodb_test.go:272)
- `TestConnectionMongoDBRepository_FindByConfigNames$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:655` (call site: connection.mongodb_test.go:655)
- `TestConnectionMongoDBRepository_Update$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:259` (call site: connection.mongodb_test.go:259)
- `TestConnectionMongoDBRepository_Delete$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:357` (call site: connection.mongodb_test.go:357)
- `TestConnectionMongoDBRepository_Update$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:296` (call site: connection.mongodb_test.go:296)
- ... and 20 more

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `clearConnectionsCollection` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:75`
- `NewConnectionMongoDBRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:74`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- ... and 2 more

---

### `clearConnectionsCollection`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 10

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `newConnectionRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:64` (call site: connection.mongodb_test.go:64)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `*github.com/LerianStudio/lib-commons/v4/commons/mongo.Client.Client` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/mongo/mongo.go:315`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`
- ... and 8 more

---

### `TestConnectionMongoDBRepository_Update`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- ... and 2 more

---

### `TestConnectionMongoDBRepository_FindByID`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestConnectionMongoDBRepository_EnsureIndexes`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestConnectionMongoDBRepository_Delete`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestConnectionMongoDBRepository_FindByOrganizationAndDatabaseName`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestConnectionMongoDBRepository_FindByConfigNames`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- ... and 2 more

---

### `TestConnectionMongoDBRepository_DropIndexes`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestConnectionMongoDBRepository_List`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `stubConnectionSpanAttributes`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 3

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestConnectionMongoDBRepository_Create$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:199)
- `TestProductMongoDBRepository_Create_ErrorPaths$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb_test.go:181)
- `TestConnectionMongoDBRepository_Update$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:318)

</details>

**Direct Callers:**

- `TestConnectionMongoDBRepository_Create$4` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:199` (call site: connection.mongodb_test.go:199)
- `TestProductMongoDBRepository_Create_ErrorPaths$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb_test.go:181` (call site: product.mongodb_test.go:181)
- `TestConnectionMongoDBRepository_Update$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb_test.go:318` (call site: connection.mongodb_test.go:318)

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `*testing.common.Cleanup` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1296`

---

### `TestConnectionMongoDBRepository_FindByOrganizationAndName`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestNewConnectionMongoDBRepository_NilClient`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewConnectionMongoDBRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:74`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`

---

### `NewJobMongoDBRepository`

**File:** `pkg/mongodb/job/job.mongodb.go`
**Risk Level:** HIGH
**Callers:** 4

**Test Coverage:** No tests found

**Direct Callers:**

- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:193` (call site: config.go:193)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:193` (call site: config.go:193)
- `initMongoRepositories` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:178` (call site: config.go:178)
- `initMongoRepositories` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:178` (call site: config.go:178)

**Calls:**

- `New` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/errors/errors.go:64`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `WithTimeout` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:703`
- `init$5` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/crypto/internal/fips140/ecdsa/cast.go:66`
- `init$bound` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/dnsclient_unix.go:374`
- ... and 2330 more

---

### `TestJobMongoDBRepository_List`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestJobMongoDBRepository_ExistsRunningByMappedFieldKey`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- ... and 3 more

---

### `TestJobMongoDBRepository_Update`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- ... and 2 more

---

### `TestJobMongoDBRepository_FindByID`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestJobMongoDBRepository_UpdateStatus`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- ... and 3 more

---

### `TestDropIndexesDatabaseError`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockMongoClientProvider` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_client_provider.mock.go:33`
- `*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.EXPECT` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_client_provider.mock.go:40`
- `Any` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/matchers.go:338`
- ... and 698 more

---

### `stubJobSpanAttributes`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 4

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestJobMongoDBRepository_List$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:407)
- `TestJobMongoDBRepository_Update$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:279)
- `TestProductMongoDBRepository_Create_ErrorPaths$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb_test.go:181)
- `TestJobMongoDBRepository_Create$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:179)

</details>

**Direct Callers:**

- `TestJobMongoDBRepository_List$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:407` (call site: job.mongodb_test.go:407)
- `TestJobMongoDBRepository_Update$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:279` (call site: job.mongodb_test.go:279)
- `TestProductMongoDBRepository_Create_ErrorPaths$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb_test.go:181` (call site: product.mongodb_test.go:181)
- `TestJobMongoDBRepository_Create$4` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:179` (call site: job.mongodb_test.go:179)

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `*testing.common.Cleanup` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1296`

---

### `TestJobMongoDBRepository_FindByRequestHashWithinWindow`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- ... and 1 more

---

### `TestRepositoryConstructorValidatesDB`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewJobMongoDBRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb.go:71`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`

---

### `newJobRepository`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 45

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestJobMongoDBRepository_Update$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:267)
- `TestListUsesDescendingByDefault` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:1006)
- `TestJobMongoDBRepository_Update$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:251)
- `TestJobMongoDBRepository_UpdateStatus$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:514)
- `TestJobMongoDBRepository_ExistsRunningByMappedFieldKey$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:720)
- `TestJobMongoDBRepository_FindByRequestHashWithinWindow$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:625)
- `TestJobMongoDBRepository_List$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:352)
- `TestJobMongoDBRepository_FindByRequestHashWithinWindow$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:613)
- `TestJobMongoDBRepository_UpdateStatus$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:443)
- `TestJobMongoDBRepository_Create$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:156)
- `TestListWithPaginationSecondPageEmpty` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:1088)
- `TestJobMongoDBRepository_Update$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:258)
- `TestJobMongoDBRepository_FindByID$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:321)
- `TestJobMongoDBRepository_ExistsRunningByMappedFieldKey$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:791)
- `TestJobMongoDBRepository_Update$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:236)
- `TestJobMongoDBRepository_Create$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:143)
- `TestJobMongoDBRepository_Update$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:220)
- `TestJobMongoDBRepository_UpdateStatus$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:546)
- `TestListPartialFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:1031)
- `TestJobMongoDBRepository_FindByRequestHashWithinWindow$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:647)
- `TestJobMongoDBRepository_EnsureIndexes` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:910)
- `TestJobMongoDBRepository_Create$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:186)
- `TestJobMongoDBRepository_List$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:405)
- `TestJobMongoDBRepository_ExistsRunningByMappedFieldKey$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:814)
- `TestEnsureIndexesHandlesConflicts` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:927)
- `TestJobMongoDBRepository_UpdateStatus$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:536)
- `TestJobMongoDBRepository_FindByID$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:309)
- `TestJobMongoDBRepository_Update$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:277)
- `TestJobMongoDBRepository_ExistsRunningByMappedFieldKey$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:838)
- `TestJobMongoDBRepository_UpdateStatus$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:489)
- `TestJobMongoDBRepository_FindByRequestHashWithinWindow$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:589)
- `TestListCompletedRangeFilter` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:983)
- `TestJobMongoDBRepository_Create$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:178)
- `TestJobMongoDBRepository_UpdateStatus$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:468)
- `TestCreateSetsDefaults` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:1050)
- `TestUpdateWithoutCompletedAtWhenFailed` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:1070)
- `TestJobMongoDBRepository_DropIndexes` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:917)
- `TestJobMongoDBRepository_FindByRequestHashWithinWindow$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:679)
- `TestJobMongoDBRepository_List$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:432)
- `TestJobMongoDBRepository_ExistsRunningByMappedFieldKey$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:847)
- `TestJobMongoDBRepository_Create$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:171)
- `TestJobMongoDBRepository_ExistsRunningByMappedFieldKey$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:743)
- `TestJobMongoDBRepository_UpdateStatus$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:574)
- `TestJobMongoDBRepository_List$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:391)
- `TestJobMongoDBRepository_ExistsRunningByMappedFieldKey$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:766)

</details>

**Direct Callers:**

- `TestJobMongoDBRepository_Update$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:267` (call site: job.mongodb_test.go:267)
- `TestListUsesDescendingByDefault` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:1006` (call site: job.mongodb_test.go:1006)
- `TestJobMongoDBRepository_Update$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:251` (call site: job.mongodb_test.go:251)
- `TestJobMongoDBRepository_UpdateStatus$4` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:514` (call site: job.mongodb_test.go:514)
- `TestJobMongoDBRepository_ExistsRunningByMappedFieldKey$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:720` (call site: job.mongodb_test.go:720)
- `TestJobMongoDBRepository_FindByRequestHashWithinWindow$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:625` (call site: job.mongodb_test.go:625)
- `TestJobMongoDBRepository_List$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:352` (call site: job.mongodb_test.go:352)
- `TestJobMongoDBRepository_FindByRequestHashWithinWindow$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:613` (call site: job.mongodb_test.go:613)
- `TestJobMongoDBRepository_UpdateStatus$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:443` (call site: job.mongodb_test.go:443)
- `TestJobMongoDBRepository_Create$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:156` (call site: job.mongodb_test.go:156)
- ... and 35 more

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `clearJobsCollection` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:71`
- `NewJobMongoDBRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb.go:71`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`

---

### `TestEnsureIndexesDatabaseError`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewController` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:84`
- `*go.uber.org/mock/gomock.Controller.Finish` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/controller.go:247`
- `NewMockMongoClientProvider` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_client_provider.mock.go:33`
- `*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.EXPECT` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_client_provider.mock.go:40`
- `Any` at `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/matchers.go:338`
- ... and 698 more

---

### `clearJobsCollection`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 10

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `newJobRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb_test.go:63` (call site: job.mongodb_test.go:63)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `*github.com/LerianStudio/lib-commons/v4/commons/mongo.Client.Client` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/mongo/mongo.go:315`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`
- ... and 8 more

---

### `TestJobMongoDBRepository_Create`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- ... and 1 more

---

### `TestNewJobMongoDBRepository_NilClient`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `NewJobMongoDBRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb.go:71`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`
- `*testing.common.Fatalf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1225`

---

### `MapMongoErrorToResponse`

**File:** `pkg/mongodb/mongo.go`
**Risk Level:** HIGH
**Callers:** 90

**Test Coverage:** No tests found

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Create` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:128` (call site: product.mongodb.go:128)
- `*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Create` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:155` (call site: product.mongodb.go:155)
- `*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Update` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:191` (call site: product.mongodb.go:191)
- `*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Update` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:226` (call site: product.mongodb.go:226)
- `*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.AssignProduct` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:723` (call site: connection.mongodb.go:723)
- `*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.AssignProduct` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:753` (call site: connection.mongodb.go:753)
- `*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:565` (call site: connection.mongodb.go:565)
- `*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:581` (call site: connection.mongodb.go:581)
- `*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:587` (call site: connection.mongodb.go:587)
- `*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:602` (call site: connection.mongodb.go:602)
- ... and 80 more

**Calls:**

- `NewLoggerFromContext` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/context.go:46`
- `Is` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/errors/wrap.go:45`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/log.go:13`
- `*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/log.go:13`
- ... and 375 more

---

### `PingMongo`

**File:** `pkg/mongodb/mongo.go`
**Risk Level:** HIGH
**Callers:** 4

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestPingMongo$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_test.go:281)
- `TestPingMongo$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_test.go:311)
- `TestPingMongo$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_test.go:295)
- `TestPingMongo$4` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_test.go:324)

</details>

**Direct Callers:**

- `TestPingMongo$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_test.go:281` (call site: mongo_test.go:281)
- `TestPingMongo$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_test.go:311` (call site: mongo_test.go:311)
- `TestPingMongo$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_test.go:295` (call site: mongo_test.go:295)
- `TestPingMongo$4` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_test.go:324` (call site: mongo_test.go:324)

**Calls:**

- `New` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/errors/errors.go:64`
- `*github.com/LerianStudio/lib-commons/v4/commons/mongo.Client.Client` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/mongo/mongo.go:315`
- `*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.Client` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo_client_provider.mock.go:45`
- `*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockmongoDatabaseProvider.Client` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.mock.go:239`
- `*github.com/LerianStudio/fetcher/pkg/mongodb/product.stubMongoDatabaseProvider.Client` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb_test.go:24`
- ... and 2333 more

---

### `TestPingMongo`

**File:** `pkg/mongodb/mongo_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `testContext`

**File:** `pkg/mysql/datasource.mysql_test.go`
**Risk Level:** HIGH
**Callers:** 8

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestExternalDataSource_MySQL_GetDatabaseSchema` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:703)
- `TestExternalDataSource_MySQL_Query` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:124)
- `TestExternalDataSource_MySQL_ValidateTableAndFields` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:574)
- `TestNewLoggerFromContext$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:57)
- `TestContextWithTracer$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:221)
- `TestContextWithLogger$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:167)
- `TestExternalDataSource_MySQL_QueryWithAdvancedFilters` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:363)
- `TestNewTracerFromContext$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:112)

</details>

**Direct Callers:**

- `TestExternalDataSource_MySQL_GetDatabaseSchema` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:703` (call site: datasource.mysql_test.go:703)
- `TestExternalDataSource_MySQL_Query` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:124` (call site: datasource.mysql_test.go:124)
- `TestExternalDataSource_MySQL_ValidateTableAndFields` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:574` (call site: datasource.mysql_test.go:574)
- `TestNewLoggerFromContext$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:57` (call site: context_test.go:57)
- `TestContextWithTracer$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:221` (call site: context_test.go:221)
- `TestContextWithLogger$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:167` (call site: context_test.go:167)
- `TestExternalDataSource_MySQL_QueryWithAdvancedFilters` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:363` (call site: datasource.mysql_test.go:363)
- `TestNewTracerFromContext$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context_test.go:112` (call site: context_test.go:112)

**Calls:**

- `Tracer` at `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel@v1.42.0/trace.go:15`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `WithValue` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:727`

---

### `TestQueryHeaderMetadata`

**File:** `pkg/net/http/http-utils_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestValidateParameters`

**File:** `pkg/net/http/http-utils_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `Setenv` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:119`
- `Setenv` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:119`
- `TestValidateParameters$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/net/http/http-utils_test.go:64`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

### `TestValidateParametersNonMetadataKeys`

**File:** `pkg/net/http/http-utils_test.go`
**Risk Level:** HIGH
**Callers:** 9

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` (/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187)
- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:182` (call site: customizer_options_test.go:182)
- `TestQueueAssertionsHelpers$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions_test.go:198` (call site: assertions_test.go:198)
- `TestBuilderConfigurationAndSuiteLifecycle$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite_test.go:193` (call site: suite_test.go:193)
- `TestSeaweedFSHelpersWithoutDocker$8` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs_test.go:157` (call site: seaweedfs_test.go:157)
- `TestWithSwaggerEnvConfig_WithEnvVars$10` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger_test.go:187` (call site: swagger_test.go:187)
- `tRunner` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036` (call site: testing.go:2036)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:62` (call site: build_secrets_test.go:62)
- `TestBuilderHelpersAndProjectRoot$7` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:228` (call site: helpers_test.go:228)
- `TestChaosMetricsAndReporting$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics_test.go:238` (call site: metrics_test.go:238)

**Calls:**

- `Setenv` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:119`
- `Setenv` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:119`
- `TestValidateParametersNonMetadataKeys$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/net/http/http-utils_test.go:816`
- `*testing.T.Run` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2050`

---

## Medium Impact Functions

Functions with 1-2 callers - changes affect a limited scope.

### `NewRoutes`

**File:** `components/manager/internal/adapters/http/in/routes.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** No tests found

**Direct Callers:**

- `assembleService` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:318` (call site: config.go:318)
- `assembleService` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:318` (call site: config.go:318)

**Calls:**

- `New` at `/Users/fredamaral/go/pkg/mod/github.com/gofiber/fiber/v2@v2.52.12/app.go:497`
- `NewTelemetryMiddleware` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/net/http/withTelemetry.go:55`
- `WithRecoverLogger` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/net/http/with_recover.go:26`
- `WithRecover` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/net/http/with_recover.go:43`
- `*github.com/gofiber/fiber/v2.App.Use` at `/Users/fredamaral/go/pkg/mod/github.com/gofiber/fiber/v2@v2.52.12/app.go:685`
- ... and 46 more

---

### `initLoggerAndTelemetry`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `InitServers` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:111` (call site: config.go:111)

**Calls:**

- `resolveZapEnvironment` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:379`
- `New` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/zap/injector.go:59`
- `NewTelemetry` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/opentelemetry/otel.go:88`
- `*github.com/LerianStudio/lib-commons/v4/commons/opentelemetry.Telemetry.ApplyGlobals` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/opentelemetry/otel.go:210`

---

### `initPlatformDependencies`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `InitServers` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:114` (call site: config.go:114)

**Calls:**

- `buildRabbitMQSource` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:342`
- `DefaultOptions` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.go:148`
- `InitializeLogger` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v2@v2.6.2/commons/zap/injector.go:15`
- `InitializeLogger` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v2@v2.6.2/commons/zap/injector.go:15`
- `getSchemaCacheTTL` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:351`
- ... and 8 more

---

### `assembleService`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `InitServers` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:116` (call site: config.go:116)

**Calls:**

- `NewCreateConnection` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create_connection.go:26`
- `NewUpdateConnection` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/update_connection.go:26`
- `NewDeleteConnection` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/delete_connection.go:23`
- `NewGetConnection` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get_connection.go:21`
- `NewListConnections` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list_connections.go:25`
- ... and 19 more

---

### `buildMongoSource`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `initMongoRepositories` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:167` (call site: config.go:167)

**Calls:**

- `PathEscape` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:192`
- `QueryEscape` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:186`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`

---

### `buildRabbitMQSource`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `initPlatformDependencies` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:217` (call site: config.go:217)

**Calls:**

- `PathEscape` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:192`
- `PathEscape` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:192`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`

---

### `resolveZapEnvironment`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `initLoggerAndTelemetry` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:137` (call site: config.go:137)

**Calls:**

- `TrimSpace` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/strings/strings.go:1091`
- `ToLower` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/strings/strings.go:727`

---

### `loadConfig`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `InitServers` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:109` (call site: config.go:109)

**Calls:**

- `SetConfigFromEnvVars` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/os.go:95`

---

### `initMongoRepositories`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `InitServers` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:112` (call site: config.go:112)

**Calls:**

- `buildMongoSource` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:335`
- `NewClient` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/mongo/mongo.go:162`
- `must` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:396`
- `NewConnectionMongoDBRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:74`
- `must` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:396`
- ... and 73 more

---

### `initCrypto`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `InitServers` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:113` (call site: config.go:113)

**Calls:**

- `DecodeMasterKey` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/crypto/key_deriver.go:145`
- `must` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:396`
- `NewHKDFKeyDeriver` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/crypto/key_deriver.go:62`
- `must` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:396`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- ... and 27 more

---

### `NewCreateFetcherJob`

**File:** `components/manager/internal/services/command/create_fetcher_job.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** No tests found

**Direct Callers:**

- `assembleService` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:307` (call site: config.go:307)
- `assembleService` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:307` (call site: config.go:307)

**Calls:** None

---

### `NewPublisherRoutes`

**File:** `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** No tests found

**Direct Callers:**

- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:167` (call site: config.go:167)
- `InitWorker` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:167` (call site: config.go:167)

**Calls:**

- `DefaultOptions` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.go:148`
- `NewRabbitMQAdapterWithOptions` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.go:356`
- `NewPublisherRoutesWithAdapter` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go:41`

---

### `resolveZapEnvironment`

**File:** `components/worker/internal/bootstrap/config.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `initLoggerAndTelemetry` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:137` (call site: config.go:137)

**Calls:**

- `TrimSpace` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/strings/strings.go:1091`
- `ToLower` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/strings/strings.go:727`

---

### `NewLoggerFromContext`

**File:** `pkg/context.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** No tests found

**Direct Callers:**

- `GetMemUsage` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons@v1.18.0/commons/utils.go:232` (call site: utils.go:232)
- `GetCPUUsage` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons@v1.18.0/commons/utils.go:215` (call site: utils.go:215)

**Calls:**

- `*go.mongodb.org/mongo-driver/mongo.sessionContext.Value` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:162`
- `*context.stopCtx.Value` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:162`
- `context.withoutCancelCtx.Value` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:608`
- `*context.emptyCtx.Value` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:195`
- `*github.com/valyala/fasthttp.RequestCtx.Value` at `/Users/fredamaral/go/pkg/mod/github.com/valyala/fasthttp@v1.69.0/server.go:2868`
- ... and 18 more

---

### `newDataSourceConfigMongoDB`

**File:** `pkg/datasource/datasource-factory.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `init$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:35` (call site: datasource-factory.go:35)

**Calls:**

- `*github.com/LerianStudio/fetcher/pkg/model.Connection.GetPasswordDecrypted` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/model/connection.go:382`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `ToLower` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/strings/strings.go:727`
- `QueryEscape` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:186`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- ... and 2389 more

---

### `newDataSourceConfigPostgres`

**File:** `pkg/datasource/datasource-factory.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `init$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:38` (call site: datasource-factory.go:38)

**Calls:**

- `*github.com/LerianStudio/fetcher/pkg/model.Connection.GetPasswordDecrypted` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/model/connection.go:382`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `ValidatePostgreSQLMode` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/sslmode/postgresql.go:43`
- `ToLower` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/strings/strings.go:727`
- `QueryEscape` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:186`
- ... and 28 more

---

### `newDataSourceConfigOracle`

**File:** `pkg/datasource/datasource-factory.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `init$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:41` (call site: datasource-factory.go:41)

**Calls:**

- `*github.com/LerianStudio/fetcher/pkg/model.Connection.GetPasswordDecrypted` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/model/connection.go:382`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `ValidateOracleMode` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/sslmode/oracle.go:47`
- `QueryEscape` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:186`
- ... and 73 more

---

### `newDataSourceConfigMySQL`

**File:** `pkg/datasource/datasource-factory.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `init$4` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:44` (call site: datasource-factory.go:44)

**Calls:**

- `*github.com/LerianStudio/fetcher/pkg/model.Connection.GetPasswordDecrypted` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/model/connection.go:382`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `ValidateMySQLMode` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/sslmode/mysql.go:42`
- `QueryEscape` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:186`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- ... and 27 more

---

### `newDataSourceConfigSQLServer`

**File:** `pkg/datasource/datasource-factory.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `init$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:47` (call site: datasource-factory.go:47)

**Calls:**

- `*github.com/LerianStudio/fetcher/pkg/model.Connection.GetPasswordDecrypted` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/model/connection.go:382`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `ValidateSQLServerMode` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/sslmode/sqlserver.go:42`
- `QueryEscape` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/url/url.go:186`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- ... and 27 more

---

### `NewDataSourceFromConnectionWithLogger`

**File:** `pkg/datasource/datasource-factory.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** No tests found

**Direct Callers:**

- `assembleService` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:279` (call site: config.go:279)
- `assembleService` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:279` (call site: config.go:279)

**Calls:** None

---

### `newDataSourceFromConnection`

**File:** `pkg/datasource/datasource-factory.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestNewDataSourceFromConnection$13` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory_test.go:140)

</details>

**Direct Callers:**

- `NewDataSourceFromConnection` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:56` (call site: datasource-factory.go:56)
- `TestNewDataSourceFromConnection$13` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory_test.go:140` (call site: datasource-factory_test.go:140)

**Calls:**

- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `newDataSourceConfigFromConnection` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:79`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `init$5` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:46`
- ... and 11 more

---

### `buildImageWithSecrets`

**File:** `pkg/itestkit/addons/e2ekit/build_secrets.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuildImageWithSecretsValidation$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:65)

</details>

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.Builder.Run` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:179` (call site: builder.go:179)
- `TestBuildImageWithSecretsValidation$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets_test.go:65` (call site: build_secrets_test.go:65)

**Calls:**

- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `generateImageTag` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets.go:166`
- `Join` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/path/filepath/path.go:130`
- `buildImageWithSecrets$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets.go:74`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- ... and 6 more

---

### `resolveBuildSecretSource`

**File:** `pkg/itestkit/addons/e2ekit/build_secrets.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `buildImageWithSecrets` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets.go:87` (call site: build_secrets.go:87)

**Calls:**

- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `createBuildSecretTempFile` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets.go:133`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`

---

### `createBuildSecretTempFile`

**File:** `pkg/itestkit/addons/e2ekit/build_secrets.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `resolveBuildSecretSource` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build_secrets.go:127` (call site: build_secrets.go:127)

**Calls:**

- `Getenv` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:101`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `CreateTemp` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/tempfile.go:35`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- ... and 14 more

---

### `New`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuilderHelpersAndProjectRoot$6` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:217)
- `TestBuilderHelpersAndProjectRoot$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:36)

</details>

**Direct Callers:**

- `TestBuilderHelpersAndProjectRoot$6` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:217` (call site: helpers_test.go:217)
- `TestBuilderHelpersAndProjectRoot$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:36` (call site: helpers_test.go:36)

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `WaitRunning` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:348`
- `RewriteLocalhostToHostGateway` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:366`

---

### `WaitLog`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuilderHelpersAndProjectRoot$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:146)

</details>

**Direct Callers:**

- `TestBuilderHelpersAndProjectRoot$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:146` (call site: helpers_test.go:146)

**Calls:** None

---

### `WaitPort`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuilderHelpersAndProjectRoot$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:140)
- `TestBuilderHelpersAndProjectRoot$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:48)

</details>

**Direct Callers:**

- `TestBuilderHelpersAndProjectRoot$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:140` (call site: helpers_test.go:140)
- `TestBuilderHelpersAndProjectRoot$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:48` (call site: helpers_test.go:48)

**Calls:** None

---

### `WaitHTTP`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuilderHelpersAndProjectRoot$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:134)

</details>

**Direct Callers:**

- `TestBuilderHelpersAndProjectRoot$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:134` (call site: helpers_test.go:134)

**Calls:** None

---

### `dumpRecentLogs`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `Run$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:243` (call site: builder.go:243)

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `*github.com/testcontainers/testcontainers-go/modules/mysql.MySQLContainer.Logs` at `/Users/fredamaral/go/pkg/mod/github.com/testcontainers/testcontainers-go@v0.40.0/container.go:57`
- `*github.com/testcontainers/testcontainers-go/modules/mssql.MSSQLServerContainer.Logs` at `/Users/fredamaral/go/pkg/mod/github.com/testcontainers/testcontainers-go@v0.40.0/container.go:57`
- `*github.com/testcontainers/testcontainers-go/modules/mongodb.MongoDBContainer.Logs` at `/Users/fredamaral/go/pkg/mod/github.com/testcontainers/testcontainers-go@v0.40.0/container.go:57`
- `*github.com/testcontainers/testcontainers-go/modules/redis.RedisContainer.Logs` at `/Users/fredamaral/go/pkg/mod/github.com/testcontainers/testcontainers-go@v0.40.0/container.go:57`
- ... and 248 more

---

### `WaitRunning`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestBuilderHelpersAndProjectRoot$3` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:152)

</details>

**Direct Callers:**

- `TestBuilderHelpersAndProjectRoot$3` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers_test.go:152` (call site: helpers_test.go:152)
- `New` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:52` (call site: builder.go:52)

**Calls:** None

---

### `NewAMQPConsumer`

**File:** `pkg/itestkit/addons/queuekit/amqp.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit.AMQPConsumerBuilder.Build` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/amqp.go:439` (call site: amqp.go:439)

**Calls:**

- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`
- `Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/errors.go:23`

---

### `JSONEqual`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `AssertJSONEqual` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions.go:268` (call site: assertions.go:268)

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `Unmarshal` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/encoding/json/decode.go:102`
- `*testing.common.Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1211`
- `Unmarshal` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/encoding/json/decode.go:102`
- `*testing.common.Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1211`
- ... and 1 more

---

### `NewConsumer`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** No tests found

**Direct Callers:**

- `NewConsumer[T]` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/consumer.go:36`
- `NewConsumer[T]` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/consumer.go:36`

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `MatchAlways` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher.go:52`
- `DefaultUnmarshaler` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/queuekit.go:48`

---

### `truncateBody`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `*github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit.Consumer[github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit.TestPayload].logMessage[github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit.TestPayload]` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/consumer.go:348)

</details>

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit.Consumer[github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit.TestPayload].logMessage[github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit.TestPayload]` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/consumer.go:348` (call site: consumer.go:348)
- `*github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit.Consumer[T].logMessage` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/consumer.go:348` (call site: consumer.go:348)

**Calls:** None

---

### `MatchHeader`

**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestMatchHeader$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher_test.go:73)

</details>

**Direct Callers:**

- `TestMatchHeader$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher_test.go:73` (call site: matcher_test.go:73)

**Calls:** None

---

### `MatchBodyPattern`

**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestMatchBodyPattern$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher_test.go:289)

</details>

**Direct Callers:**

- `TestMatchBodyPattern$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher_test.go:289` (call site: matcher_test.go:289)

**Calls:**

- `MustCompile` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/regexp/regexp.go:310`

---

### `hasNestedValue`

**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `MatchJSONFieldExists$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher.go:171` (call site: matcher.go:171)

**Calls:**

- `Split` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/strings/strings.go:361`

---

### `applyPublishOptions`

**File:** `pkg/itestkit/addons/queuekit/queuekit.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit.AMQPPublisher.Publish` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/amqp.go:305` (call site: amqp.go:305)

**Calls:**

- `WithCorrelationID$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/queuekit.go:108`
- `WithHeaders$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/queuekit.go:87`
- `WithRoutingKey$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/queuekit.go:94`
- `WithContentType$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/queuekit.go:101`
- `WithPersistent$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/queuekit.go:122`
- ... and 1 more

---

### `NewToxiproxyChaos`

**File:** `pkg/itestkit/chaos_toxiproxy.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/pkg/itestkit.Builder.Build` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite.go:88` (call site: suite.go:88)

**Calls:**

- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `ForListeningPort` at `/Users/fredamaral/go/pkg/mod/github.com/testcontainers/testcontainers-go@v0.40.0/wait/host_port.go:67`
- `*github.com/testcontainers/testcontainers-go/wait.HostPortStrategy.WithStartupTimeout` at `/Users/fredamaral/go/pkg/mod/github.com/testcontainers/testcontainers-go@v0.40.0/wait/host_port.go:102`
- `GenericContainer` at `/Users/fredamaral/go/pkg/mod/github.com/testcontainers/testcontainers-go@v0.40.0/generic.go:52`
- `github.com/testcontainers/testcontainers-go/modules/mongodb.MongoDBContainer.Host` at `/Users/fredamaral/go/pkg/mod/github.com/testcontainers/testcontainers-go@v0.40.0/container.go:45`
- ... and 59 more

---

### `CExposedPorts`

**File:** `pkg/itestkit/customizer_options.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:71)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:71` (call site: customizer_options_test.go:71)

**Calls:** None

---

### `CBindMount`

**File:** `pkg/itestkit/customizer_options.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:75)
- `TestCustomizerOptions_MutateContainerRequest$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:76)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:75` (call site: customizer_options_test.go:75)
- `TestCustomizerOptions_MutateContainerRequest$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:76` (call site: customizer_options_test.go:76)

**Calls:** None

---

### `CHostDockerInternal`

**File:** `pkg/itestkit/customizer_options.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:73)
- `TestCustomizerOptions_MutateContainerRequest$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:74)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:73` (call site: customizer_options_test.go:73)
- `TestCustomizerOptions_MutateContainerRequest$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:74` (call site: customizer_options_test.go:74)

**Calls:** None

---

### `CNetworks`

**File:** `pkg/itestkit/customizer_options.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCustomizerOptions_MutateContainerRequest$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:72)

</details>

**Direct Callers:**

- `TestCustomizerOptions_MutateContainerRequest$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options_test.go:72` (call site: customizer_options_test.go:72)

**Calls:** None

---

### `ResolveContainerHostPort`

**File:** `pkg/itestkit/hostport.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestResolveContainerHostPort$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport_test.go:46)
- `TestResolveContainerHostPort$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport_test.go:35)

</details>

**Direct Callers:**

- `TestResolveContainerHostPort$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport_test.go:46` (call site: hostport_test.go:46)
- `TestResolveContainerHostPort$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport_test.go:35` (call site: hostport_test.go:35)

**Calls:**

- `ParseHostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:95`
- `ParseHostPort` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:95`
- `NormalizeHost` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:85`

---

### `validateUniqueInfraNames`

**File:** `pkg/itestkit/infra.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/pkg/itestkit.Builder.Build` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite.go:109` (call site: suite.go:109)

**Calls:**

- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/postgres.PostgresInfra.InfraKind` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/postgres/postgres.go:205`
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mysql.MySQLInfra.InfraKind` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mysql/mysql.go:219`
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mssql.MSSQLInfra.InfraKind` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mssql/mssql.go:218`
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mongodb.MongoDBInfra.InfraKind` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mongodb/mongodb.go:203`
- `*github.com/LerianStudio/fetcher/pkg/itestkit/infra/seaweedfs.SeaweedFSInfra.InfraKind` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs.go:306`
- ... and 14 more

---

### `NewMongoDBInfra`

**File:** `pkg/itestkit/infra/mongodb/mongodb.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** No tests found

**Direct Callers:**

- `NewMongoDBInfraStub` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mongodb/mongodb.go:210` (call site: mongodb.go:210)
- `NewCoreInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/tests/shared/infra.go:60` (call site: infra.go:60)

**Calls:** None

---

### `WithMongoDBFixedPort`

**File:** `pkg/itestkit/infra/mongodb/mongodb_options.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `NewCoreInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/tests/shared/infra.go:53` (call site: infra.go:53)

**Calls:** None

---

### `NewMSSQLInfra`

**File:** `pkg/itestkit/infra/mssql/mssql.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `NewMSSQLInfraStub` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mssql/mssql.go:225` (call site: mssql.go:225)

**Calls:** None

---

### `NewMySQLInfra`

**File:** `pkg/itestkit/infra/mysql/mysql.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `NewMySQLInfraStub` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mysql/mysql.go:226` (call site: mysql.go:226)

**Calls:** None

---

### `NewOracleInfra`

**File:** `pkg/itestkit/infra/oracle/oracle.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `NewOracleInfraStub` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/oracle/oracle.go:251` (call site: oracle.go:251)

**Calls:** None

---

### `NewPostgresInfra`

**File:** `pkg/itestkit/infra/postgres/postgres.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `NewPostgresInfraStub` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/postgres/postgres.go:212` (call site: postgres.go:212)

**Calls:** None

---

### `WithRabbitFixedPort`

**File:** `pkg/itestkit/infra/rabbitmq/rabbit_options.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `NewCoreInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/tests/shared/infra.go:54` (call site: infra.go:54)

**Calls:** None

---

### `NewRabbitInfra`

**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `NewCoreInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/tests/shared/infra.go:66` (call site: infra.go:66)

**Calls:** None

---

### `NewRedisInfra`

**File:** `pkg/itestkit/infra/redis/redis.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `NewCoreInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/tests/shared/infra.go:72` (call site: infra.go:72)

**Calls:** None

---

### `WithRedisFixedPort`

**File:** `pkg/itestkit/infra/redis/redis_options.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `NewCoreInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/tests/shared/infra.go:55` (call site: infra.go:55)

**Calls:** None

---

### `WithSeaweedFSFixedPort`

**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs_options.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `NewCoreInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/tests/shared/infra.go:56` (call site: infra.go:56)

**Calls:** None

---

### `TestMain`

**File:** `pkg/mongodb/connection/connection.mongodb_test.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `main` at `/Users/fredamaral/Library/Caches/go-build/ad/ad5e305532c2464a64464e9f33123bb1658f8f525cb514cf523bc03ff1b7a0b7-d:86` (call site: ad5e305532c2464a64464e9f33123bb1658f8f525cb514cf523bc03ff1b7a0b7-d:86)

**Calls:**

- `Start` at `/Users/fredamaral/go/pkg/mod/github.com/tryvium-travels/memongo@v0.12.0/memongo.go:36`
- `Printf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/log/log.go:407`
- `Exit` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/proc.go:62`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `*github.com/tryvium-travels/memongo.Server.URI` at `/Users/fredamaral/go/pkg/mod/github.com/tryvium-travels/memongo@v0.12.0/memongo.go:224`
- ... and 6 more

---

### `NewDataSourceRepository`

**File:** `pkg/mongodb/datasource.mongodb.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestNewDataSourceRepository_UnitTests$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/datasource.mongodb_test.go:1042)
- `TestNewDataSourceRepository_UnitTests$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/datasource.mongodb_test.go:1051)

</details>

**Direct Callers:**

- `TestNewDataSourceRepository_UnitTests$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/datasource.mongodb_test.go:1042` (call site: datasource.mongodb_test.go:1042)
- `TestNewDataSourceRepository_UnitTests$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/datasource.mongodb_test.go:1051` (call site: datasource.mongodb_test.go:1051)

**Calls:**

- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `NewClient` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/mongo/mongo.go:162`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/log.go:13`
- ... and 21 more

---

### `testLogger`

**File:** `pkg/mongodb/datasource.mongodb_test.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestNewDataSourceRepository_UnitTests$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/datasource.mongodb_test.go:1040)
- `TestNewDataSourceRepository_UnitTests$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/datasource.mongodb_test.go:1049)

</details>

**Direct Callers:**

- `TestNewDataSourceRepository_UnitTests$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/datasource.mongodb_test.go:1040` (call site: datasource.mongodb_test.go:1040)
- `TestNewDataSourceRepository_UnitTests$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/datasource.mongodb_test.go:1049` (call site: datasource.mongodb_test.go:1049)

**Calls:** None

---

### `TestMain`

**File:** `pkg/mongodb/job/job.mongodb_test.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `main` at `/Users/fredamaral/Library/Caches/go-build/6b/6b7f2468d35d62cfb71583d64c697574fbcc3d0296178de33807dbcf39513755-d:88` (call site: 6b7f2468d35d62cfb71583d64c697574fbcc3d0296178de33807dbcf39513755-d:88)

**Calls:**

- `Start` at `/Users/fredamaral/go/pkg/mod/github.com/tryvium-travels/memongo@v0.12.0/memongo.go:36`
- `Printf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/log/log.go:407`
- `Exit` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/proc.go:62`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `*github.com/tryvium-travels/memongo.Server.URI` at `/Users/fredamaral/go/pkg/mod/github.com/tryvium-travels/memongo@v0.12.0/memongo.go:224`
- ... and 6 more

---

### `NewProductMongoDBRepository`

**File:** `pkg/mongodb/product/product.mongodb.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestNewProductMongoDBRepository_NilClient` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb_test.go:35)

</details>

**Direct Callers:**

- `TestNewProductMongoDBRepository_NilClient` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb_test.go:35` (call site: product.mongodb_test.go:35)

**Calls:**

- `New` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/errors/errors.go:64`
- `newProductMongoDBRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:73`

---

### `newProductMongoDBRepository`

**File:** `pkg/mongodb/product/product.mongodb.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestNewProductMongoDBRepository_ConfigAndInitialization$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb_test.go:85)

</details>

**Direct Callers:**

- `NewProductMongoDBRepository` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:70` (call site: product.mongodb.go:70)
- `TestNewProductMongoDBRepository_ConfigAndInitialization$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb_test.go:85` (call site: product.mongodb_test.go:85)

**Calls:**

- `New` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/errors/errors.go:64`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `WithTimeout` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:703`
- `init$5` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/crypto/internal/fips140/ecdsa/cast.go:66`
- `init$bound` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/net/dnsclient_unix.go:374`
- ... and 2330 more

---

### `parseJSONField`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestParseJSONField_MySQL$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:1823)

</details>

**Direct Callers:**

- `TestParseJSONField_MySQL$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:1823` (call site: datasource.mysql_test.go:1823)
- `createRowMap` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql.go:358` (call site: datasource.mysql.go:358)

**Calls:**

- `Unmarshal` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/encoding/json/decode.go:102`
- `Unmarshal` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/encoding/json/decode.go:102`
- `Unmarshal` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/encoding/json/decode.go:102`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- ... and 21 more

---

### `NewDataSourceRepository`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** No tests found

**Direct Callers:**

- `newDataSourceConfigMySQL` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:354` (call site: datasource-factory.go:354)
- `newDataSourceConfigMySQL` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:354` (call site: datasource-factory.go:354)

**Calls:**

- `*github.com/LerianStudio/fetcher/pkg/mysql.Connection.GetDB` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/mysql.go:61`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/log.go:13`
- `*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/log.go:13`
- ... and 20 more

---

### `scanRows`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** No tests found

**Direct Callers:**

- `*github.com/LerianStudio/fetcher/pkg/mysql.ExternalDataSource.QueryWithAdvancedFilters` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql.go:580` (call site: datasource.mysql.go:580)
- `*github.com/LerianStudio/fetcher/pkg/mysql.ExternalDataSource.Query` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql.go:141` (call site: datasource.mysql.go:141)

**Calls:**

- `*database/sql.Rows.Columns` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/database/sql/sql.go:3183`
- `*database/sql.Rows.Scan` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/database/sql/sql.go:3365`
- `*database/sql.Rows.Next` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/database/sql/sql.go:3029`
- `createRowMap` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql.go:353`

---

### `createRowMap`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestCreateRowMap_MySQL$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:1908)

</details>

**Direct Callers:**

- `scanRows` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql.go:345` (call site: datasource.mysql.go:345)
- `TestCreateRowMap_MySQL$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql_test.go:1908` (call site: datasource.mysql_test.go:1908)

**Calls:**

- `parseJSONField` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql.go:365`

---

### `WithRecover`

**File:** `pkg/net/http/with_recover.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** No tests found

**Direct Callers:**

- `NewRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes.go:42` (call site: routes.go:42)
- `NewRoutes` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes.go:42` (call site: routes.go:42)

**Calls:**

- `buildRecoverOpts` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/net/http/with_recover.go:32`

---

### `createRowMap`

**File:** `pkg/oracle/datasource.oracle.go`
**Risk Level:** MEDIUM
**Callers:** 1

**Test Coverage:** No tests found

**Direct Callers:**

- `scanRows` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/oracle/datasource.oracle.go:600` (call site: datasource.oracle.go:600)

**Calls:**

- `parseJSONField` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/oracle/datasource.oracle.go:620`

---

### `NewDataSourceRepository`

**File:** `pkg/oracle/datasource.oracle.go`
**Risk Level:** MEDIUM
**Callers:** 2

**Test Coverage:** Has tests

<details>
<summary>Tests covering this function</summary>

- `TestNewDataSourceRepository$1` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/oracle/datasource.oracle_test.go:55)
- `TestNewDataSourceRepository$2` (/Users/fredamaral/repos/lerianstudio/fetcher/pkg/oracle/datasource.oracle_test.go:69)

</details>

**Direct Callers:**

- `TestNewDataSourceRepository$1` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/oracle/datasource.oracle_test.go:55` (call site: datasource.oracle_test.go:55)
- `TestNewDataSourceRepository$2` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/oracle/datasource.oracle_test.go:69` (call site: datasource.oracle_test.go:69)

**Calls:**

- `*github.com/LerianStudio/fetcher/pkg/oracle.Connection.GetDB` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/oracle/oracle.go:69`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/log.go:13`
- `*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log` at `/Users/fredamaral/go/pkg/mod/github.com/!lerian!studio/lib-commons/v4@v4.0.0/commons/log/log.go:13`
- ... and 20 more

---

## Low Impact Functions

Functions with no callers - may be entry points, tests, or dead code.

### `*ConnectionHandler.ListConnections`

**File:** `components/manager/internal/adapters/http/in/connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionHandler.GetConnectionSchema`

**File:** `components/manager/internal/adapters/http/in/connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionHandler.CreateConnection`

**File:** `components/manager/internal/adapters/http/in/connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionHandler.GetConnection`

**File:** `components/manager/internal/adapters/http/in/connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionHandler.TestConnection`

**File:** `components/manager/internal/adapters/http/in/connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionHandler.UpdateConnection`

**File:** `components/manager/internal/adapters/http/in/connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionHandler.DeleteConnection`

**File:** `components/manager/internal/adapters/http/in/connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionHandler.ValidateSchema`

**File:** `components/manager/internal/adapters/http/in/connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*FetcherHandler.GetJob`

**File:** `components/manager/internal/adapters/http/in/fetcher.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*FetcherHandler.CreateJob`

**File:** `components/manager/internal/adapters/http/in/fetcher.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MigrationHandler.AssignConnectionToProduct`

**File:** `components/manager/internal/adapters/http/in/migration.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MigrationHandler.ListUnassignedConnections`

**File:** `components/manager/internal/adapters/http/in/migration.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductHandler.CreateProduct`

**File:** `components/manager/internal/adapters/http/in/product.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductHandler.ListProducts`

**File:** `components/manager/internal/adapters/http/in/product.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductHandler.GetProduct`

**File:** `components/manager/internal/adapters/http/in/product.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductHandler.UpdateProduct`

**File:** `components/manager/internal/adapters/http/in/product.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductHandler.DeleteProduct`

**File:** `components/manager/internal/adapters/http/in/product.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `InitServers`

**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `loadConfig` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:126`
- `Background` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/context/context.go:215`
- `initLoggerAndTelemetry` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:135`
- `initMongoRepositories` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:165`
- `initCrypto` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:195`
- ... and 2 more

---

### `*Server.Run`

**File:** `components/manager/internal/bootstrap/server.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*AssignConnection.Execute`

**File:** `components/manager/internal/services/command/assign_connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*CreateConnection.Execute`

**File:** `components/manager/internal/services/command/create_connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*CreateFetcherJob.Execute`

**File:** `components/manager/internal/services/command/create_fetcher_job.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*CreateFetcherJob.validateProductOwnership`

**File:** `components/manager/internal/services/command/create_fetcher_job.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*CreateFetcherJob.TestConnection`

**File:** `components/manager/internal/services/command/create_fetcher_job.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*CreateProduct.Execute`

**File:** `components/manager/internal/services/command/create_product.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DeleteProduct.Execute`

**File:** `components/manager/internal/services/command/delete_product.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UpdateConnection.Execute`

**File:** `components/manager/internal/services/command/update_connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UpdateProduct.Execute`

**File:** `components/manager/internal/services/command/update_product.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*GetConnectionSchema.Execute`

**File:** `components/manager/internal/services/query/get_connection_schema.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ListConnections.Execute`

**File:** `components/manager/internal/services/query/list_connections.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ListProducts.Execute`

**File:** `components/manager/internal/services/query/list_products.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ListUnassignedConnections.Execute`

**File:** `components/manager/internal/services/query/list_unassigned_connections.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*TestConnection.Execute`

**File:** `components/manager/internal/services/query/test_connection.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ValidateSchema.Execute`

**File:** `components/manager/internal/services/query/validate_schema.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ValidateSchema.getOrFetchSchema`

**File:** `components/manager/internal/services/query/validate_schema.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConsumerRoutes.Shutdown`

**File:** `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `NewConsumerRoutes`

**File:** `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `DefaultOptions` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.go:148`
- `NewRabbitMQAdapterWithOptions` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/rabbitmq/rabbitmq.go:356`
- `NewConsumerRoutesWithAdapter` at `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go:48`

---

### `*ConsumerRoutes.RunConsumers`

**File:** `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*PublisherRoutes.Publish`

**File:** `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*PublisherRoutes.Shutdown`

**File:** `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MultiQueueConsumer.Run`

**File:** `components/worker/internal/bootstrap/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MultiQueueConsumer.handlerGenerateReport`

**File:** `components/worker/internal/bootstrap/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Service.Run`

**File:** `components/worker/internal/bootstrap/service.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.ExtractExternalData`

**File:** `components/worker/internal/services/extract-data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.saveExternalDataToSeaweedFS`

**File:** `components/worker/internal/services/extract-data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.shouldSkipProcessing`

**File:** `components/worker/internal/services/extract-data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.parseMessage`

**File:** `components/worker/internal/services/extract-data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.extractJobIDFromMultipleSources`

**File:** `components/worker/internal/services/extract-data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.extractJobIDFromPartialJSON`

**File:** `components/worker/internal/services/extract-data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.queryDatabase`

**File:** `components/worker/internal/services/extract-data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.handleErrorWithUpdate`

**File:** `components/worker/internal/services/extract-data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.encryptDataForSeaweedFS`

**File:** `components/worker/internal/services/extract-data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.checkReportStatus`

**File:** `components/worker/internal/services/extract-data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.transformPluginCRMAdvancedFilters`

**File:** `components/worker/internal/services/extract_crm_data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.decryptPluginCRMData`

**File:** `components/worker/internal/services/extract_crm_data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.QueryPluginCRM`

**File:** `components/worker/internal/services/extract_crm_data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.queryPluginCRMCollectionWithFilters`

**File:** `components/worker/internal/services/extract_crm_data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.processPluginCRMCollection`

**File:** `components/worker/internal/services/extract_crm_data.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*UseCase.publishJobNotification`

**File:** `components/worker/internal/services/job_notification.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.WithRewriter`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `localhostToHostGatewayRewriter.Rewrite`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.WithImage`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.Run`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.WithEnv`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.ExposePort`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.DisableDefaultLocalhostRewrite`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.WithLogsOnFailureMaxBytes`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.WithWait`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `waitHTTP.Configure`

**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.TimeoutsBelow`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.ThroughputAbove`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.FailedResults`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.P50Below`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.MinRequestsReached`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.Summary`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.AverageLatencyBelow`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.FailuresBelow`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.SuccessRateAbove`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.P99Below`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosAssertions.P95Below`

**File:** `pkg/itestkit/addons/metricskit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ErrorClassifier.GetCategoryCounts`

**File:** `pkg/itestkit/addons/metricskit/error_classifier.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.GetTotalRequests`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.SuccessRate`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.StartTest`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.StartChaos`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.GetErrorCounts`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.ChaosThroughputRPS`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.Percentile`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.EndChaos`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.RecordRequest`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.GetTimeoutRequests`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.SuccessfulThroughputRPS`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.ChaosDuration`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.TestDuration`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.GetFailedRequests`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.GetMinLatency`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.AverageLatency`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.ThroughputRPS`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ChaosMetrics.EndTest`

**File:** `pkg/itestkit/addons/metricskit/metrics.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Reporter.WriteReport`

**File:** `pkg/itestkit/addons/metricskit/reporter.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Reporter.String`

**File:** `pkg/itestkit/addons/metricskit/reporter.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Reporter.CompactSummary`

**File:** `pkg/itestkit/addons/metricskit/reporter.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*AMQPConsumerBuilder.WithPrefetch`

**File:** `pkg/itestkit/addons/queuekit/amqp.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*AMQPConsumer.Close`

**File:** `pkg/itestkit/addons/queuekit/amqp.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*AMQPPublisher.connect`

**File:** `pkg/itestkit/addons/queuekit/amqp.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*AMQPPublisher.Close`

**File:** `pkg/itestkit/addons/queuekit/amqp.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*AMQPConsumerBuilder.BindTo`

**File:** `pkg/itestkit/addons/queuekit/amqp.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*AMQPConsumerBuilder.WithQueueDeclare`

**File:** `pkg/itestkit/addons/queuekit/amqp.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*AMQPConsumer.connect`

**File:** `pkg/itestkit/addons/queuekit/amqp.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Assertions[T].HasHeaderKey`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ResultAssertions[T].At`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MessageSequence[T].FilterBy`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExpectMessagesHelper[T].OrFatal`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Assertions[T].PayloadSatisfies`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ResultAssertions[T].UnmatchedCount`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Assertions[T].HasRoutingKey`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Assertions[T].HasMessageID`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Assertions[T].HasContentType`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ResultAssertions[T].First`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `AssertResult`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`

---

### `AssertJSONEqual`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`
- `JSONEqual` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions.go:247`
- `*testing.common.Errorf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1211`

---

### `*MessageSequence[T].GroupBy`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExpectMessagesHelper[T].ToContainWhere`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `AssertMessage`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`

---

### `*Assertions[T].HasHeader`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ResultAssertions[T].HasAtLeast`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExpectMessagesHelper[T].ToHaveCount`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Assertions[T].HasCorrelationID`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Assertions[T].PayloadEquals`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ResultAssertions[T].HasCount`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ResultAssertions[T].DidNotTimeout`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ResultAssertions[T].HasNoErrors`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ResultAssertions[T].All`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MessageSequence[T].RoutingKeysInOrder`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExpectMessagesHelper[T].ToSucceed`

**File:** `pkg/itestkit/addons/queuekit/assertions.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Consumer[T].CaptureAll`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConsumerBuilder[T].WithMatcher`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConsumerBuilder[T].WithTimeout`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Consumer[T].DrainQueue`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Consumer[T].captureMessage`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConsumerBuilder[T].WithUnmarshaler`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConsumerBuilder[T].Build`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Consumer[T].WaitForMessages`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Consumer[T].GetCaptured`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Consumer[T].ClearCaptured`

**File:** `pkg/itestkit/addons/queuekit/consumer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `MatchHeaderExists`

**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `MatchJSONField`

**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `MatchJSONFieldPattern`

**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `MustCompile` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/regexp/regexp.go:310`

---

### `MatchAll`

**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `MatchAny`

**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `MatchRoutingKeyPattern`

**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `MustCompile` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/regexp/regexp.go:310`

---

### `WaitResult[T].First`

**File:** `pkg/itestkit/addons/queuekit/queuekit.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*toxiproxyChaos.RemoveToxic`

**File:** `pkg/itestkit/chaos_toxiproxy.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*toxiproxyChaos.RemoveAllToxics`

**File:** `pkg/itestkit/chaos_toxiproxy.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*toxiproxyChaos.Close`

**File:** `pkg/itestkit/chaos_toxiproxy.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*toxiproxyChaos.CreateProxy`

**File:** `pkg/itestkit/chaos_toxiproxy.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*toxiproxyChaos.AddBandwidth`

**File:** `pkg/itestkit/chaos_toxiproxy.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*toxiproxyChaos.AddLatency`

**File:** `pkg/itestkit/chaos_toxiproxy.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*toxiproxyChaos.CutConnection`

**File:** `pkg/itestkit/chaos_toxiproxy.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*toxiproxyChaos.AddTimeout`

**File:** `pkg/itestkit/chaos_toxiproxy.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `WaitListeningPort.Apply`

**File:** `pkg/itestkit/container_generic.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.WithContainerCustomize`

**File:** `pkg/itestkit/container_generic.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*genericContainerInfra.Start`

**File:** `pkg/itestkit/container_generic.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*genericContainerInfra.Terminate`

**File:** `pkg/itestkit/container_generic.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `CustomizerFunc.Customize`

**File:** `pkg/itestkit/customizer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `MergeCustomizers`

**File:** `pkg/itestkit/customizer.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `CEnvFromOS`

**File:** `pkg/itestkit/customizer_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `LookupEnv` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:112`
- `CEnv` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer_options.go:14`

---

### `CEnvs`

**File:** `pkg/itestkit/customizer_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `WithEnv` at `/Users/fredamaral/go/pkg/mod/github.com/testcontainers/testcontainers-go@v0.40.0/options.go:75`

---

### `CAll`

**File:** `pkg/itestkit/customizer_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MongoDBInfra.HostPort`

**File:** `pkg/itestkit/infra/mongodb/mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `NewMongoDBInfraStub`

**File:** `pkg/itestkit/infra/mongodb/mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `NewMongoDBInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mongodb/mongodb.go:42`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`

---

### `*MongoDBInfra.Start`

**File:** `pkg/itestkit/infra/mongodb/mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MongoDBInfra.Endpoint`

**File:** `pkg/itestkit/infra/mongodb/mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MongoDBInfra.URI`

**File:** `pkg/itestkit/infra/mongodb/mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MongoDBInfra.Terminate`

**File:** `pkg/itestkit/infra/mongodb/mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MongoDBInfra.ContainerHostPort`

**File:** `pkg/itestkit/infra/mongodb/mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MSSQLInfra.Endpoint`

**File:** `pkg/itestkit/infra/mssql/mssql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `NewMSSQLInfraStub`

**File:** `pkg/itestkit/infra/mssql/mssql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `NewMSSQLInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mssql/mssql.go:43`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`

---

### `*MSSQLInfra.Start`

**File:** `pkg/itestkit/infra/mssql/mssql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MSSQLInfra.DSN`

**File:** `pkg/itestkit/infra/mssql/mssql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MSSQLInfra.HostPort`

**File:** `pkg/itestkit/infra/mssql/mssql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MSSQLInfra.Terminate`

**File:** `pkg/itestkit/infra/mssql/mssql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `WithMSSQLFixedPort`

**File:** `pkg/itestkit/infra/mssql/mssql_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MySQLInfra.Endpoint`

**File:** `pkg/itestkit/infra/mysql/mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MySQLInfra.Terminate`

**File:** `pkg/itestkit/infra/mysql/mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MySQLInfra.Start`

**File:** `pkg/itestkit/infra/mysql/mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MySQLInfra.DSN`

**File:** `pkg/itestkit/infra/mysql/mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MySQLInfra.HostPort`

**File:** `pkg/itestkit/infra/mysql/mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `NewMySQLInfraStub`

**File:** `pkg/itestkit/infra/mysql/mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `NewMySQLInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mysql/mysql.go:45`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`

---

### `WithMySQLInitScript`

**File:** `pkg/itestkit/infra/mysql/mysql_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `WithMySQLFixedPort`

**File:** `pkg/itestkit/infra/mysql/mysql_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*OracleInfra.Endpoint`

**File:** `pkg/itestkit/infra/oracle/oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `NewOracleInfraStub`

**File:** `pkg/itestkit/infra/oracle/oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `NewOracleInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/oracle/oracle.go:44`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`

---

### `*OracleInfra.DSN`

**File:** `pkg/itestkit/infra/oracle/oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*OracleInfra.GoDRORDSN`

**File:** `pkg/itestkit/infra/oracle/oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*OracleInfra.HostPort`

**File:** `pkg/itestkit/infra/oracle/oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*OracleInfra.Terminate`

**File:** `pkg/itestkit/infra/oracle/oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*OracleInfra.Start`

**File:** `pkg/itestkit/infra/oracle/oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `WithOracleInitScript`

**File:** `pkg/itestkit/infra/oracle/oracle_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `WithOracleFixedPort`

**File:** `pkg/itestkit/infra/oracle/oracle_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*PostgresInfra.Start`

**File:** `pkg/itestkit/infra/postgres/postgres.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*PostgresInfra.DSN`

**File:** `pkg/itestkit/infra/postgres/postgres.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*PostgresInfra.HostPort`

**File:** `pkg/itestkit/infra/postgres/postgres.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*PostgresInfra.Endpoint`

**File:** `pkg/itestkit/infra/postgres/postgres.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*PostgresInfra.Terminate`

**File:** `pkg/itestkit/infra/postgres/postgres.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `NewPostgresInfraStub`

**File:** `pkg/itestkit/infra/postgres/postgres.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `NewPostgresInfra` at `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/postgres/postgres.go:43`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`
- `Sprintf` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/fmt/print.go:237`

---

### `*PostgresInfra.ContainerHostPort`

**File:** `pkg/itestkit/infra/postgres/postgres.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `WithPGInitFile`

**File:** `pkg/itestkit/infra/postgres/postgres_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `WithPGFixedPort`

**File:** `pkg/itestkit/infra/postgres/postgres_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*configReader.Read`

**File:** `pkg/itestkit/infra/rabbitmq/rabbit_options.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RabbitInfra.Start`

**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RabbitInfra.Endpoint`

**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RabbitInfra.AMQPURL`

**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RabbitInfra.HostPort`

**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RabbitInfra.Terminate`

**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RabbitInfra.ContainerHostPort`

**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RedisInfra.Start`

**File:** `pkg/itestkit/infra/redis/redis.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RedisInfra.HostPort`

**File:** `pkg/itestkit/infra/redis/redis.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RedisInfra.Terminate`

**File:** `pkg/itestkit/infra/redis/redis.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RedisInfra.Endpoint`

**File:** `pkg/itestkit/infra/redis/redis.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RedisInfra.URL`

**File:** `pkg/itestkit/infra/redis/redis.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RedisInfra.Addr`

**File:** `pkg/itestkit/infra/redis/redis.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*RedisInfra.ContainerHostPort`

**File:** `pkg/itestkit/infra/redis/redis.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*SeaweedFSInfra.Start`

**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*SeaweedFSInfra.Endpoint`

**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*SeaweedFSInfra.URL`

**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*SeaweedFSInfra.HostPort`

**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*SeaweedFSInfra.Terminate`

**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*SeaweedFSInfra.ContainerHostPort`

**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.Build`

**File:** `pkg/itestkit/suite.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Suite.Terminate`

**File:** `pkg/itestkit/suite.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `New`

**File:** `pkg/itestkit/suite.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:**

- `*testing.common.Helper` at `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:1272`

---

### `*Builder.WithInfra`

**File:** `pkg/itestkit/suite.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Builder.WithInfras`

**File:** `pkg/itestkit/suite.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Suite.Network`

**File:** `pkg/itestkit/suite.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigMongoDB.GetSchemaInfo`

**File:** `pkg/model/datasource/mongodb/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigMongoDB.Connect`

**File:** `pkg/model/datasource/mongodb/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigMongoDB.Query`

**File:** `pkg/model/datasource/mongodb/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigMySQL.Connect`

**File:** `pkg/model/datasource/mysql/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigMySQL.Query`

**File:** `pkg/model/datasource/mysql/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigMySQL.GetSchemaInfo`

**File:** `pkg/model/datasource/mysql/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigOracle.Connect`

**File:** `pkg/model/datasource/oracle/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigOracle.Query`

**File:** `pkg/model/datasource/oracle/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigOracle.GetSchemaInfo`

**File:** `pkg/model/datasource/oracle/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigPostgres.Connect`

**File:** `pkg/model/datasource/postgres/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigPostgres.GetSchemaInfo`

**File:** `pkg/model/datasource/postgres/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigPostgres.Query`

**File:** `pkg/model/datasource/postgres/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigSQLServer.Query`

**File:** `pkg/model/datasource/sqlserver/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigSQLServer.GetSchemaInfo`

**File:** `pkg/model/datasource/sqlserver/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*DataSourceConfigSQLServer.Connect`

**File:** `pkg/model/datasource/sqlserver/datasource-config.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.ListUnassigned`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.AssignProduct`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.FindByID`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.FindByOrganizationAndName`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.FindByOrganizationAndDatabaseName`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.CountByProduct`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.buildQueryFilter`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.Create`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.Update`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.Delete`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.FindByConfigNames`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.List`

**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockmongoDatabaseProvider.Client`

**File:** `pkg/mongodb/connection/connection.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockmongoDatabaseProviderMockRecorder.Client`

**File:** `pkg/mongodb/connection/connection.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.EnsureIndexes`

**File:** `pkg/mongodb/connection/indexes.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ConnectionMongoDBRepository.DropIndexes`

**File:** `pkg/mongodb/connection/indexes.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.Query`

**File:** `pkg/mongodb/datasource.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.QueryWithAdvancedFilters`

**File:** `pkg/mongodb/datasource.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.GetDatabaseSchema`

**File:** `pkg/mongodb/datasource.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.processQueryResults`

**File:** `pkg/mongodb/datasource.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.CloseConnection`

**File:** `pkg/mongodb/datasource.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `NewMockDatasource`

**File:** `pkg/mongodb/datasource.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockDatasourceMockRecorder.CloseConnection`

**File:** `pkg/mongodb/datasource.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockDatasource.GetDatabaseSchema`

**File:** `pkg/mongodb/datasource.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockDatasource.Query`

**File:** `pkg/mongodb/datasource.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockDatasource.EXPECT`

**File:** `pkg/mongodb/datasource.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockDatasource.CloseConnection`

**File:** `pkg/mongodb/datasource.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockDatasourceMockRecorder.GetDatabaseSchema`

**File:** `pkg/mongodb/datasource.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockDatasourceMockRecorder.Query`

**File:** `pkg/mongodb/datasource.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockDatasource.QueryWithAdvancedFilters`

**File:** `pkg/mongodb/datasource.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockDatasourceMockRecorder.QueryWithAdvancedFilters`

**File:** `pkg/mongodb/datasource.mongodb.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*JobMongoDBRepository.EnsureIndexes`

**File:** `pkg/mongodb/job/indexes.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*JobMongoDBRepository.DropIndexes`

**File:** `pkg/mongodb/job/indexes.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*JobMongoDBRepository.Create`

**File:** `pkg/mongodb/job/job.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*JobMongoDBRepository.UpdateStatus`

**File:** `pkg/mongodb/job/job.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*JobMongoDBRepository.FindByRequestHashWithinWindow`

**File:** `pkg/mongodb/job/job.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*JobMongoDBRepository.Update`

**File:** `pkg/mongodb/job/job.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*JobMongoDBRepository.ExistsRunningByMappedFieldKey`

**File:** `pkg/mongodb/job/job.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*JobMongoDBRepository.scanJobs`

**File:** `pkg/mongodb/job/job.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*JobMongoDBRepository.FindByID`

**File:** `pkg/mongodb/job/job.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*JobMongoDBRepository.List`

**File:** `pkg/mongodb/job/job.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockMongoClientProvider.Client`

**File:** `pkg/mongodb/mongo_client_provider.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*MockMongoClientProviderMockRecorder.Client`

**File:** `pkg/mongodb/mongo_client_provider.mock.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductMongoDBRepository.EnsureIndexes`

**File:** `pkg/mongodb/product/indexes.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductMongoDBRepository.DropIndexes`

**File:** `pkg/mongodb/product/indexes.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductMongoDBRepository.List`

**File:** `pkg/mongodb/product/product.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductMongoDBRepository.buildQueryFilter`

**File:** `pkg/mongodb/product/product.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductMongoDBRepository.Create`

**File:** `pkg/mongodb/product/product.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductMongoDBRepository.Update`

**File:** `pkg/mongodb/product/product.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductMongoDBRepository.Delete`

**File:** `pkg/mongodb/product/product.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductMongoDBRepository.FindByID`

**File:** `pkg/mongodb/product/product.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ProductMongoDBRepository.FindByCode`

**File:** `pkg/mongodb/product/product.mongodb.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.GetDatabaseSchema`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.buildSchema`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.CloseConnection`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.scanColumns`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.Query`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.buildTableSchema`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.ValidateTableAndFields`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.QueryWithAdvancedFilters`

**File:** `pkg/mysql/datasource.mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Connection.GetDB`

**File:** `pkg/mysql/mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*Connection.Connect`

**File:** `pkg/mysql/mysql.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.CloseConnection`

**File:** `pkg/oracle/datasource.oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.GetDatabaseSchema`

**File:** `pkg/oracle/datasource.oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.buildSchema`

**File:** `pkg/oracle/datasource.oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.ValidateTableAndFields`

**File:** `pkg/oracle/datasource.oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.Query`

**File:** `pkg/oracle/datasource.oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.scanColumns`

**File:** `pkg/oracle/datasource.oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

### `*ExternalDataSource.QueryWithAdvancedFilters`

**File:** `pkg/oracle/datasource.oracle.go`
**Risk Level:** LOW
**Callers:** 0

**Test Coverage:** No tests found

**Direct Callers:** None

**Calls:** None

---

