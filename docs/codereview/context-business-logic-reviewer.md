# Pre-Analysis Context: Business Logic

## Impact Analysis


### High Impact Changes


> Warning: call graph analysis is partial.



**Call Graph Warnings:**

- Truncated modified functions from 675 to 500




#### `setupConnectionTestApp`
**File:** `components/manager/internal/adapters/http/in/connection\_test.go`
**Risk Level:** HIGH (55 direct callers)

**Direct Callers (signature change affects these):**

1. `TestConnectionHandler\_ListConnections\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:481`
2. `TestConnectionHandler\_ValidateSchema\_InternalError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1696`
3. `TestConnectionHandler\_ValidateSchema\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1306`
4. `TestConnectionHandler\_ValidateSchema\_Failure\_DataSourceNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1485`
5. `TestConnectionHandler\_DeleteConnection\_HandlerDirectly\_InvalidUUID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1242`
6. `TestConnectionHandler\_ValidateSchema\_Failure\_TableNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1350`
7. `TestConnectionHandler\_ListConnections\_EmptyList` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:542`
8. `TestConnectionHandler\_ValidateSchema\_HandlerDirectly\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1967`
9. `TestConnectionHandler\_ValidateSchema\_Failure\_DataSourceDown` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1548`
10. `TestConnectionHandler\_CreateConnection\_RealHandlerSuccess` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real\_handler\_test.go:45`
11. `TestConnectionHandler\_TestConnection\_HandlerDirectly\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1951`
12. `TestConnectionHandler\_ListConnections\_HandlerDirectly\_InvalidSortOrder` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1886`
13. `TestConnectionHandler\_CreateConnection\_InternalError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:287`
14. `TestConnectionHandler\_CreateConnection\_HandlerDirectly\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1838`
15. `TestConnectionHandler\_ListConnections\_RealHandlerSuccess` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real\_handler\_test.go:101`
16. `TestConnectionHandler\_TestConnection\_RateLimited` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1142`
17. `TestConnectionHandler\_CreateConnection\_Conflict` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:243`
18. `TestConnectionHandler\_ValidateSchema\_Failure\_FieldNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1417`
19. `TestConnectionHandler\_ValidateSchema\_HandlerDirectly\_InvalidJSON` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1796`
20. `TestConnectionHandler\_UpdateConnection\_NotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:711`
21. `TestConnectionHandler\_GetConnection\_InvalidID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:433`
22. `TestConnectionHandler\_TestConnection\_InvalidID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1126`
23. `TestConnectionHandler\_GetConnection\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:335`
24. `TestProductHandler\_ListProducts\_RealHandlerSuccess` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real\_handler\_test.go:293`
25. `TestConnectionHandler\_DeleteConnection\_NotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:920`
26. `TestConnectionHandler\_ValidateSchema\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1646`
27. `TestMigrationHandler\_ListUnassignedConnections\_RealHandlerSuccess` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real\_handler\_test.go:322`
28. `TestConnectionHandler\_GetConnection\_NotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:391`
29. `TestConnectionHandler\_DeleteConnection\_Conflict\_ActiveJobs` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:962`
30. `TestConnectionHandler\_UpdateConnection\_RealHandlerSuccess` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real\_handler\_test.go:140`
31. `TestConnectionHandler\_ListConnections\_InvalidPaginationParams` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:590`
32. `TestConnectionHandler\_DeleteConnection\_HandlerDirectly\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1935`
33. `TestConnectionHandler\_CreateConnection\_InvalidJSON` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:150`
34. `TestConnectionHandler\_TestConnection\_NotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1084`
35. `TestConnectionHandler\_TestConnection\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1026`
36. `TestConnectionHandler\_DeleteConnection\_InvalidID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1006`
37. `TestConnectionHandler\_CreateConnection\_HandlerDirectly\_InvalidJSON` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1189`
38. `TestConnectionHandler\_UpdateConnection\_HandlerDirectly\_InvalidUUID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1225`
39. `TestConnectionHandler\_CreateConnection\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:90`
40. `TestConnectionHandler\_GetConnection\_HandlerDirectly\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1902`
41. `TestConnectionHandler\_CreateConnection\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:193`
42. `TestConnectionHandler\_TestConnection\_HandlerDirectly\_InvalidUUID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1258`
43. `TestConnectionHandler\_UpdateConnection\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:642`
44. `TestConnectionHandler\_UpdateConnection\_Conflict\_ActiveJobs` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:805`
45. `TestConnectionHandler\_GetConnectionSchema\_RealHandlerSuccess` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real\_handler\_test.go:222`
46. `TestConnectionHandler\_DeleteConnection\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:867`
47. `TestConnectionHandler\_GetConnection\_HandlerDirectly\_InvalidUUID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1209`
48. `TestConnectionHandler\_ValidateSchema\_RealHandlerFailureResponse` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real\_handler\_test.go:183`
49. `TestConnectionHandler\_ValidateSchema\_InvalidJSON` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1603`
50. `TestProductHandler\_CreateProduct\_RealHandlerSuccess` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real\_handler\_test.go:260`
51. `TestConnectionHandler\_ValidateSchema\_MultipleDataSources` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1740`
52. `TestConnectionHandler\_ListConnections\_HandlerDirectly\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1274`
53. `TestConnectionHandler\_UpdateConnection\_InvalidBody` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:767`
54. `TestConnectionHandler\_UpdateConnection\_HandlerDirectly\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/connection\_test.go:1918`
55. `TestMigrationHandler\_AssignConnectionToProduct\_RealHandlerSuccess` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/real\_handler\_test.go:351`

**Callees (this function depends on):**

1. `New`
2. `\*github.com/gofiber/fiber/v2.App.Use`

#### `setupTestApp`
**File:** `components/manager/internal/adapters/http/in/fetcher\_test.go`
**Risk Level:** HIGH (21 direct callers)

**Direct Callers (signature change affects these):**

1. `TestFetcherHandler\_GetJob\_NotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:596`
2. `TestFetcherHandler\_CreateJob\_HandlerDirectly\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:1046`
3. `TestFetcherHandler\_GetJob\_InvalidID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:638`
4. `TestFetcherHandler\_GetJob\_FailedJob` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:976`
5. `TestFetcherHandler\_GetJob\_HandlerDirectly\_InvalidUUID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:744`
6. `TestFetcherHandler\_CreateJob\_WithFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:853`
7. `TestFetcherHandler\_GetJob\_InternalError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:696`
8. `TestFetcherHandler\_CreateJob\_Conflict` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:448`
9. `TestFetcherHandler\_CreateJob\_HandlerDirectly\_InvalidJSON` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:760`
10. `TestFetcherHandler\_CreateJob\_WithMetadata` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:784`
11. `TestFetcherHandler\_CreateJob\_Success\_DuplicateJob` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:390`
12. `TestFetcherHandler\_CreateJob\_MetadataSourceValidation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:210`
13. `TestFetcherHandler\_GetJob\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:680`
14. `TestFetcherHandler\_GetJob\_CompletedJob` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:924`
15. `TestFetcherHandler\_GetJob\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:540`
16. `TestFetcherHandler\_GetJob\_HandlerDirectly\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:1063`
17. `TestFetcherHandler\_CreateJob\_ContentTypeValidation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:163`
18. `TestFetcherHandler\_CreateJob\_InvalidJSON` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:54`
19. `TestFetcherHandler\_CreateJob\_MissingOrgHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:105`
20. `TestFetcherHandler\_CreateJob\_Success\_NewJob` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:326`
21. `TestFetcherHandler\_CreateJob\_InternalError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/fetcher\_test.go:492`

**Callees (this function depends on):**

1. `New`
2. `\*github.com/gofiber/fiber/v2.App.Use`

#### `TestFetcherHandler\_CreateJob\_MetadataSourceValidation`
**File:** `components/manager/internal/adapters/http/in/fetcher\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `setupTestApp`
2. `\*github.com/gofiber/fiber/v2.App.Post`
3. `\*testing.T.Run`

#### `setupMiddlewareTestApp`
**File:** `components/manager/internal/adapters/http/in/middlewares\_test.go`
**Risk Level:** HIGH (14 direct callers)

**Direct Callers (signature change affects these):**

1. `TestParseHeaderParameters\_MissingHeader` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:205`
2. `TestParsePathParametersUUID\_MultipleCalls` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:146`
3. `TestParseHeaderParameters\_CaseInsensitiveHeaderName` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:285`
4. `TestParsePathParametersUUID\_EmptyPathParameter` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:126`
5. `TestParseHeaderParameters\_ValidOrgID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:184`
6. `TestParsePathParametersUUID\_UUIDVersions$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:433`
7. `TestMiddlewareChain\_PathMiddlewareFails` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:386`
8. `TestParseHeaderParameters\_UUIDWithWhitespace` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:451`
9. `TestMiddlewareChain\_BothMiddlewares` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:341`
10. `TestParsePathParametersUUID\_InvalidUUID$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:108`
11. `TestParseHeaderParameters\_MultipleCalls` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:306`
12. `TestParsePathParametersUUID\_ValidUUID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:47`
13. `TestMiddlewareChain\_HeaderMiddlewareFails` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:367`
14. `TestParseHeaderParameters\_InvalidOrgID$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/middlewares\_test.go:266`

**Callees (this function depends on):**

1. `New`
2. `\*github.com/gofiber/fiber/v2.App.Use`

#### `TestNewRoutes\_RegistersCriticalStaticRoutesBeforeIDRoutes`
**File:** `components/manager/internal/adapters/http/in/routes\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewNop`
2. `NewTelemetry`
3. `NoError`
4. `NewAuthClient`
5. `NewRoutes`
6. `\*github.com/gofiber/fiber/v2.App.GetRoutes`
7. `indexOfRoute`
8. `indexOfRoute`
9. `indexOfRoute`
10. `NotEqual`
11. `NotEqual`
12. `NotEqual`
13. `Less`
14. `indexOfRoute`
15. `NotEqual`
16. `indexOfRoute`
17. `NotEqual`
18. `indexOfRoute`
19. `NotEqual`
20. `indexOfRoute`
21. `NotEqual`

#### `indexOfRoute`
**File:** `components/manager/internal/adapters/http/in/routes\_test.go`
**Risk Level:** HIGH (7 direct callers)

**Direct Callers (signature change affects these):**

1. `TestNewRoutes\_RegistersCriticalStaticRoutesBeforeIDRoutes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes\_test.go:65`
2. `TestNewRoutes\_RegistersCriticalStaticRoutesBeforeIDRoutes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes\_test.go:66`
3. `TestNewRoutes\_RegistersCriticalStaticRoutesBeforeIDRoutes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes\_test.go:67`
4. `TestNewRoutes\_RegistersCriticalStaticRoutesBeforeIDRoutes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes\_test.go:74`
5. `TestNewRoutes\_RegistersCriticalStaticRoutesBeforeIDRoutes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes\_test.go:75`
6. `TestNewRoutes\_RegistersCriticalStaticRoutesBeforeIDRoutes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes\_test.go:76`
7. `TestNewRoutes\_RegistersCriticalStaticRoutesBeforeIDRoutes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/routes\_test.go:77`

**Callees (this function depends on):**


#### `must`
**File:** `components/manager/internal/bootstrap/config.go`
**Risk Level:** HIGH (11 direct callers)

**Direct Callers (signature change affects these):**

1. `ForEachPackage` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/allpackages.go:68`
2. `TestMust$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config\_test.go:78`
3. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:146`
4. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:149`
5. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:155`
6. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:159`
7. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:163`
8. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:188`
9. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:194`
10. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:197`
11. `TestMust$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config\_test.go:50`

**Callees (this function depends on):**

1. `Errorf`

#### `TestGetSchemaCacheTTL`
**File:** `components/manager/internal/bootstrap/config\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `getSchemaCacheTTL`
2. `Equal`
3. `getSchemaCacheTTL`
4. `Equal`
5. `getSchemaCacheTTL`
6. `Equal`

#### `TestGetRedisDB`
**File:** `components/manager/internal/bootstrap/config\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `getRedisDB`
2. `Equal`
3. `getRedisDB`
4. `Equal`
5. `getRedisDB`
6. `Equal`

#### `TestResolveZapEnvironment`
**File:** `components/manager/internal/bootstrap/config\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Parallel`
2. `\*testing.T.Run`

#### `NewCreateFetcherJobWithTester`
**File:** `components/manager/internal/services/command/create\_fetcher\_job.go`
**Risk Level:** HIGH (8 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCreateFetcherJob\_Execute\_ProductNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create\_fetcher\_job\_test.go:948`
2. `TestCreateFetcherJob\_Execute\_ConnectionNotAssigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create\_fetcher\_job\_test.go:1070`
3. `TestCreateFetcherJob\_Execute\_ProductRepoError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create\_fetcher\_job\_test.go:1010`
4. `TestCreateFetcherJob\_Execute\_MultipleConnectionsSuccess\_WithoutProductRepo` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create\_fetcher\_job\_test.go:627`
5. `TestCreateFetcherJob\_Execute\_FiltersWithMultipleDatasources` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create\_fetcher\_job\_test.go:791`
6. `TestCreateFetcherJob\_Execute\_PublishFailureMarksJobFailed` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create\_fetcher\_job\_test.go:706`
7. `TestCreateFetcherJob\_Execute\_ProductValidationSuccess` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create\_fetcher\_job\_test.go:1208`
8. `TestCreateFetcherJob\_Execute\_ProductMismatch` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create\_fetcher\_job\_test.go:1139`

**Callees (this function depends on):**


#### `TestCreateFetcherJob\_Execute\_PublishFailureMarksJobFailed`
**File:** `components/manager/internal/services/command/create\_fetcher\_job\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockRepository`
4. `NewMockRepository`
5. `NewMockConnectionTester`
6. `NewMockAdapter`
7. `NewCreateFetcherJobWithTester`
8. `testContext`
9. `New`
10. `New`
11. `New`
12. `\*github.com/LerianStudio/fetcher/pkg/mongodb/job.MockRepository.EXPECT`
13. `Any`
14. `Any`
15. `\*github.com/LerianStudio/fetcher/pkg/mongodb/job.MockRepositoryMockRecorder.FindByRequestHashWithinWindow`
16. `\*go.uber.org/mock/gomock.Call.Return`
17. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepository.EXPECT`
18. `Any`
19. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepositoryMockRecorder.FindByConfigNames`
20. `\*go.uber.org/mock/gomock.Call.Return`
21. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/command.MockConnectionTester.EXPECT`
22. `Any`
23. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/command.MockConnectionTesterMockRecorder.TestConnection`
24. `\*go.uber.org/mock/gomock.Call.Return`
25. `\*github.com/LerianStudio/fetcher/pkg/mongodb/job.MockRepository.EXPECT`
26. `Any`
27. `Any`
28. `\*github.com/LerianStudio/fetcher/pkg/mongodb/job.MockRepositoryMockRecorder.Create`
29. `\*go.uber.org/mock/gomock.Call.DoAndReturn`
30. `\*github.com/LerianStudio/fetcher/pkg/rabbitmq.MockAdapter.EXPECT`
31. `Any`
32. `Any`
33. `Any`
34. `\*github.com/LerianStudio/fetcher/pkg/rabbitmq.MockAdapterMockRecorder.ProducerDefault`
35. `\*go.uber.org/mock/gomock.Call.Return`
36. `\*github.com/LerianStudio/fetcher/pkg/mongodb/job.MockRepository.EXPECT`
37. `Any`
38. `Any`
39. `\*github.com/LerianStudio/fetcher/pkg/mongodb/job.MockRepositoryMockRecorder.Update`
40. `\*go.uber.org/mock/gomock.Call.DoAndReturn`
41. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/command.CreateFetcherJob.Execute`
42. `\*testing.common.Fatalf`
43. `\*testing.common.Fatal`
44. `As`
45. `\*testing.common.Fatalf`
46. `\*testing.common.Fatalf`
47. `\*github.com/pkg/errors.withMessage.Error`
48. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationError.Error`
49. `\*github.com/docker/docker/client.httpError.Error`
50. `\*runtime.boundsError.Error`
51. `\*gopkg.in/go-playground/validator.v9.InvalidValidationError.Error`
52. `go.mongodb.org/mongo-driver/mongo/options.MarshalError.Error`
53. `\*go.mongodb.org/mongo-driver/internal/codecutil.MarshalError.Error`
54. `github.com/docker/docker/api/types.ErrorResponse.Error`
55. `\*go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt.Error.Error`
56. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationError.Error`
57. `github.com/docker/docker/client.emptyIDError.Error`
58. `\*github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
59. `vendor/golang.org/x/net/http2/hpack.DecodingError.Error`
60. `\*net/http.unsupportedTEError.Error`
61. `\*net/http.http2headerFieldNameError.Error`
62. `\*github.com/docker/docker/api/types/network.joinError.Error`
63. `\*go.mongodb.org/mongo-driver/x/mongo/driver/auth.Error.Error`
64. `\*net.ParseError.Error`
65. `github.com/go-openapi/swag/yamlutils.yamlError.Error`
66. `runtime.errorAddressString.Error`
67. `\*net.InvalidAddrError.Error`
68. `\*crypto/x509.UnknownAuthorityError.Error`
69. `\*github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
70. `golang.org/x/crypto/ssh.ServerAuthError.Error`
71. `\*github.com/go-playground/validator/v10.ValidationErrors.Error`
72. `golang.org/x/net/http2.pseudoHeaderError.Error`
73. `golang.org/x/crypto/ocsp.ResponseError.Error`
74. `go.mongodb.org/mongo-driver/x/mongo/driver.QueryFailureError.Error`
75. `github.com/docker/docker/api/types/container.errInvalidParameter.Error`
76. `\*net/http.http2pseudoHeaderError.Error`
77. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.MalformedElementError.Error`
78. `\*crypto/tls.alert.Error`
79. `\*crypto/x509.InsecureAlgorithmError.Error`
80. `go.mongodb.org/mongo-driver/mongo.WriteErrors.Error`
81. `\*encoding/asn1.invalidUnmarshalError.Error`
82. `crypto/x509.UnhandledCriticalExtension.Error`
83. `go.mongodb.org/mongo-driver/bson/bsoncodec.ValueDecoderError.Error`
84. `github.com/containerd/errdefs.customMessage.Error`
85. `\*github.com/xdg-go/stringprep.Error.Error`
86. `\*crypto/tls.CertificateVerificationError.Error`
87. `\*fmt.wrapError.Error`
88. `\*github.com/yuin/gopher-lua/pm.Error.Error`
89. `github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
90. `\*net/http.http2headerFieldValueError.Error`
91. `golang.org/x/net/http2.duplicatePseudoHeaderError.Error`
92. `os.errSymlink.Error`
93. `vendor/golang.org/x/net/idna.labelError.Error`
94. `\*crypto/tls.RecordHeaderError.Error`
95. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoEncoder.Error`
96. `\*golang.org/x/mod/module.ModuleError.Error`
97. `\*encoding/json.UnmarshalTypeError.Error`
98. `github.com/containerd/errdefs.errInvalidArgument.Error`
99. `\*google.golang.org/protobuf/internal/errors.prefixError.Error`
100. `\*encoding/base64.CorruptInputError.Error`
101. `\*golang.org/x/mod/module.InvalidPathError.Error`
102. `\*go.mongodb.org/mongo-driver/x/mongo/driver.ResponseError.Error`
103. `net/netip.parseAddrError.Error`
104. `go/scanner.Error.Error`
105. `\*internal/poll.DeadlineExceededError.Error`
106. `\*golang.org/x/crypto/ssh.ExitMissingError.Error`
107. `time.fileSizeError.Error`
108. `\*github.com/redis/go-redis/v9/internal/proto.RedisError.Error`
109. `golang.org/x/net/http2.ConnectionError.Error`
110. `github.com/LerianStudio/lib-commons/v2/commons.Response.Error`
111. `go.mongodb.org/mongo-driver/internal/codecutil.MarshalError.Error`
112. `golang.org/x/net/http2/hpack.InvalidIndexError.Error`
113. `\*strconv.NumError.Error`
114. `\*go.opentelemetry.io/otel/sdk/trace.samplerArgParseError.Error`
115. `github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
116. `\*os/user.UnknownUserIdError.Error`
117. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteConcernError.Error`
118. `net/http.nothingWrittenError.Error`
119. `\*github.com/redis/go-redis/v9/internal/proto.MasterDownError.Error`
120. `\*go/scanner.ErrorList.Error`
121. `\*go.mongodb.org/mongo-driver/mongo.WriteException.Error`
122. `github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
123. `\*github.com/docker/docker/client.errConnectionFailed.Error`
124. `\*vendor/golang.org/x/net/dns/dnsmessage.nestedError.Error`
125. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InvalidDepthTraversalError.Error`
126. `net/http.requestBodyReadError.Error`
127. `net/http.tlsHandshakeTimeoutError.Error`
128. `\*github.com/pkg/errors.fundamental.Error`
129. `\*github.com/redis/go-redis/v9/internal/proto.MovedError.Error`
130. `\*golang.org/x/net/http2.goAwayFlowError.Error`
131. `\*github.com/containerd/errdefs.errPermissionDenied.Error`
132. `github.com/microsoft/go-mssqldb.RetryableError.Error`
133. `\*github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
134. `\*github.com/redis/go-redis/v9/internal/proto.ExecAbortError.Error`
135. `golang.org/x/text/encoding/internal.RepertoireError.Error`
136. `github.com/montanaflynn/stats.statsError.Error`
137. `\*gopkg.in/go-playground/validator.v9.fieldError.Error`
138. `\*github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
139. `\*net/http.http2GoAwayError.Error`
140. `github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
141. `github.com/docker/docker/client.objectNotFoundError.Error`
142. `\*net/http.nothingWrittenError.Error`
143. `\*github.com/montanaflynn/stats.statsError.Error`
144. `\*github.com/redis/go-redis/v9/push.ProcessorError.Error`
145. `\*go.mongodb.org/mongo-driver/x/mongo/driver/ocsp.Error.Error`
146. `golang.org/x/net/http2.noCachedConnError.Error`
147. `crypto/x509.ConstraintViolationError.Error`
148. `github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
149. `github.com/LerianStudio/lib-commons/v4/commons.Response.Error`
150. `net/http.http2StreamError.Error`
151. `github.com/LerianStudio/lib-license-go/v2/pkg.ResponseError.Error`
152. `\*os.SyscallError.Error`
153. `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc/internal.PartialSuccess.Error`
154. `crypto/tls.alert.Error`
155. `\*vendor/golang.org/x/net/idna.labelError.Error`
156. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteErrors.Error`
157. `\*net/netip.parseAddrError.Error`
158. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ElementTypeError.Error`
159. `vendor/golang.org/x/net/http2/hpack.InvalidIndexError.Error`
160. `\*github.com/containerd/errdefs.errNotModified.Error`
161. `\*net.temporaryError.Error`
162. `\*github.com/jackc/pgx/v5/pgconn.perDialConnectError.Error`
163. `\*go.mongodb.org/mongo-driver/mongo.BulkWriteError.Error`
164. `\*github.com/jackc/pgx/v5/pgconn.PgError.Error`
165. `github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
166. `go.mongodb.org/mongo-driver/bson/bsonrw.TransitionError.Error`
167. `\*go/types.ArgumentError.Error`
168. `github.com/containerd/errdefs.errNotImplemented.Error`
169. `\*gopkg.in/go-playground/validator.v9.ValidationErrors.Error`
170. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.DecodeError.Error`
171. `\*github.com/containerd/errdefs/pkg/internal/cause.ErrUnexpectedStatus.Error`
172. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ElementTypeError.Error`
173. `google.golang.org/grpc/internal/transport.ioError.Error`
174. `net/url.InvalidHostError.Error`
175. `golang.org/x/net/http2.StreamError.Error`
176. `\*github.com/jackc/pgx/v5/pgtype.nullAssignmentError.Error`
177. `\*github.com/redis/go-redis/v9/internal/proto.AuthError.Error`
178. `\*github.com/microsoft/go-mssqldb.RetryableError.Error`
179. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.ServerSelectionError.Error`
180. `internal/runtime/maps.unhashableTypeError.Error`
181. `\*google.golang.org/protobuf/internal/errors.SizeMismatchError.Error`
182. `go.mongodb.org/mongo-driver/x/mongo/driver.Error.Error`
183. `\*crypto/rc4.KeySizeError.Error`
184. `github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
185. `github.com/gofiber/fiber/v2/internal/schema.ConversionError.Error`
186. `\*github.com/valyala/fasthttp.ErrBrokenChunk.Error`
187. `google.golang.org/grpc/internal/transport.ConnectionError.Error`
188. `\*github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
189. `\*github.com/redis/go-redis/v9/internal/proto.OOMError.Error`
190. `\*golang.org/x/net/http2/hpack.DecodingError.Error`
191. `\*crypto/tls.ECHRejectionError.Error`
192. `crypto/aes.KeySizeError.Error`
193. `\*context.deadlineExceededError.Error`
194. `\*vendor/golang.org/x/net/http2/hpack.DecodingError.Error`
195. `github.com/xdg-go/stringprep.Error.Error`
196. `net/http.http2noCachedConnError.Error`
197. `\*vendor/golang.org/x/net/idna.runeError.Error`
198. `golang.org/x/net/http2/hpack.DecodingError.Error`
199. `\*github.com/jackc/pgx/v5/pgconn.contextAlreadyDoneError.Error`
200. `\*github.com/grpc-ecosystem/grpc-gateway/v2/runtime.HTTPStatusError.Error`
201. `\*debug/dwarf.DecodeError.Error`
202. `\*crypto/internal/fips140/hmac.errCloneUnsupported.Error`
203. `github.com/LerianStudio/lib-license-go/v2/pkg.HTTPError.Error`
204. `go.mongodb.org/mongo-driver/mongo.WriteException.Error`
205. `github.com/go-openapi/swag/jsonutils/adapters/stdlib/json.jsonError.Error`
206. `\*text/template.ExecError.Error`
207. `\*github.com/go-sql-driver/mysql.MySQLError.Error`
208. `github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
209. `\*encoding/json.InvalidUTF8Error.Error`
210. `\*golang.org/x/net/webdav/internal/xml.UnmarshalError.Error`
211. `\*github.com/go-playground/universal-translator.ErrExistingTranslator.Error`
212. `\*runtime.errorString.Error`
213. `\*time.parseDurationError.Error`
214. `encoding/hex.InvalidByteError.Error`
215. `\*crypto/tls.AlertError.Error`
216. `net/textproto.ProtocolError.Error`
217. `\*github.com/containerd/errdefs.errResourceExhausted.Error`
218. `\*crypto/des.KeySizeError.Error`
219. `github.com/moby/term.EscapeError.Error`
220. `\*github.com/LerianStudio/lib-commons/v4/commons/rabbitmq.sanitizedError.Error`
221. `compress/bzip2.StructuralError.Error`
222. `\*time.fileSizeError.Error`
223. `\*golang.org/x/net/http2.ConnectionError.Error`
224. `\*github.com/testcontainers/testcontainers-go.ParallelContainersError.Error`
225. `\*golang.org/x/net/http2.httpError.Error`
226. `\*github.com/gofiber/fiber/v2.Error.Error`
227. `go.mongodb.org/mongo-driver/mongo.EncryptionKeyVaultError.Error`
228. `github.com/go-openapi/jsonpointer.pointerError.Error`
229. `\*compress/flate.InternalError.Error`
230. `github.com/valyala/fasthttp.ErrNothingRead.Error`
231. `\*github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
232. `\*encoding/xml.UnmarshalError.Error`
233. `\*github.com/go-playground/validator/v10.InvalidValidationError.Error`
234. `github.com/containerd/errdefs.errConflict.Error`
235. `github.com/LerianStudio/lib-commons/v4/commons/net/http.ErrorResponse.Error`
236. `google.golang.org/protobuf/internal/impl.errInvalidUTF8.Error`
237. `github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
238. `\*internal/reflectlite.ValueError.Error`
239. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteCommandError.Error`
240. `github.com/go-openapi/swag/loading.loadingError.Error`
241. `\*github.com/gofiber/fiber/v2/internal/schema.MultiError.Error`
242. `\*google.golang.org/protobuf/internal/errors.wrapError.Error`
243. `\*runtime.TypeAssertionError.Error`
244. `\*crypto/x509.CertificateInvalidError.Error`
245. `\*github.com/LerianStudio/lib-license-go/v2/pkg.UnprocessableOperationError.Error`
246. `go.mongodb.org/mongo-driver/internal/aws/awserr.baseError.Error`
247. `golang.org/x/crypto/blowfish.KeySizeError.Error`
248. `github.com/valyala/fasthttp.EscapeError.Error`
249. `\*golang.org/x/crypto/ssh.cbcError.Error`
250. `\*github.com/go-playground/universal-translator.ErrCardinalTranslation.Error`
251. `\*github.com/valyala/fasthttp.EscapeError.Error`
252. `\*errors.joinError.Error`
253. `\*github.com/redis/go-redis/v9/internal/proto.ReadOnlyError.Error`
254. `\*github.com/valyala/fasthttp.ErrBodyStreamWritePanic.Error`
255. `\*crypto/tls.permanentError.Error`
256. `encoding/json.jsonError.Error`
257. `\*go.mongodb.org/mongo-driver/x/mongo/driver.InvalidOperationError.Error`
258. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.PoolError.Error`
259. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.PoolError.Error`
260. `golang.org/x/net/http2.headerFieldNameError.Error`
261. `\*github.com/go-logr/logr.notFoundError.Error`
262. `\*crypto/x509.ConstraintViolationError.Error`
263. `github.com/containerd/errdefs.errNotModified.Error`
264. `\*github.com/go-resty/resty/v2.noRetryErr.Error`
265. `\*github.com/redis/go-redis/v9/push.HandlerError.Error`
266. `\*golang.org/x/sync/singleflight.panicError.Error`
267. `\*github.com/containerd/errdefs.errUnavailable.Error`
268. `\*google.golang.org/grpc/internal/status.Error.Error`
269. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.WaitQueueTimeoutError.Error`
270. `\*github.com/containerd/errdefs.errAborted.Error`
271. `\*encoding/xml.TagPathError.Error`
272. `net/http.http2goAwayFlowError.Error`
273. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.poolClearedError.Error`
274. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoEncoder.Error`
275. `context.deadlineExceededError.Error`
276. `\*github.com/go-openapi/swag/jsonutils/adapters/stdlib/json.jsonError.Error`
277. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationUnknownFieldsError.Error`
278. `\*github.com/grpc-ecosystem/grpc-gateway/v2/internal/httprule.InvalidTemplateError.Error`
279. `crypto/x509.SystemRootsError.Error`
280. `\*github.com/containerd/errdefs.errDataLoss.Error`
281. `\*net/http.timeoutError.Error`
282. `\*github.com/jackc/pgx/v5.ScanArgError.Error`
283. `\*github.com/cenkalti/backoff/v4.PermanentError.Error`
284. `\*go.mongodb.org/mongo-driver/bson/bsonrw.TransitionError.Error`
285. `github.com/containerd/errdefs.errOutOfRange.Error`
286. `\*github.com/jackc/pgx/v5/pgproto3.ExceededMaxBodyLenErr.Error`
287. `\*go.uber.org/zap.errArrayElem.Error`
288. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.ConnectionError.Error`
289. `\*golang.org/x/crypto/ssh.ServerAuthError.Error`
290. `go.mongodb.org/mongo-driver/mongo.ErrMapForOrderedArgument.Error`
291. `golang.org/x/net/http2.headerFieldValueError.Error`
292. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.ConnectionError.Error`
293. `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.errMetric.Error`
294. `\*encoding/json.MarshalerError.Error`
295. `\*crypto/x509.UnhandledCriticalExtension.Error`
296. `\*github.com/jackc/pgx/v5/pgproto3.invalidMessageFormatErr.Error`
297. `net/url.EscapeError.Error`
298. `\*github.com/jackc/pgx/v5/pgconn.ConnectError.Error`
299. `\*github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
300. `github.com/containerd/errdefs/pkg/internal/cause.ErrUnexpectedStatus.Error`
301. `net.InvalidAddrError.Error`
302. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InsufficientBytesError.Error`
303. `\*archive/tar.headerError.Error`
304. `github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
305. `golang.org/x/net/http2.goAwayFlowError.Error`
306. `\*github.com/redis/go-redis/v9/internal/proto.LoadingError.Error`
307. `\*github.com/valyala/fasthttp.ErrSmallBuffer.Error`
308. `\*github.com/LerianStudio/lib-license-go/v2/pkg.UnauthorizedError.Error`
309. `go.mongodb.org/mongo-driver/x/mongo/driver.InvalidOperationError.Error`
310. `vendor/golang.org/x/net/idna.runeError.Error`
311. `\*github.com/Shopify/toxiproxy/v2/client.ApiError.Error`
312. `github.com/gofiber/fiber/v2/internal/schema.UnknownKeyError.Error`
313. `os/user.UnknownUserError.Error`
314. `\*os/exec.Error.Error`
315. `\*github.com/microsoft/go-mssqldb/aecmk.Error.Error`
316. `\*github.com/yuin/gopher-lua/parse.Error.Error`
317. `syscall.Errno.Error`
318. `net.canceledError.Error`
319. `\*net/http.http2noCachedConnError.Error`
320. `\*net/http.tlsHandshakeTimeoutError.Error`
321. `\*crypto/x509.HostnameError.Error`
322. `\*github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
323. `net/http.http2connError.Error`
324. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ValidationError.Error`
325. `\*golang.org/x/net/webdav/internal/xml.UnsupportedTypeError.Error`
326. `os/user.UnknownGroupIdError.Error`
327. `\*github.com/lib/pq.Error.Error`
328. `\*github.com/jackc/pgx/v5/pgconn.connLockError.Error`
329. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.poolClearedError.Error`
330. `\*go.mongodb.org/mongo-driver/mongo.WriteConcernError.Error`
331. `\*go/build/constraint.SyntaxError.Error`
332. `\*go.mongodb.org/mongo-driver/mongo.MongocryptdError.Error`
333. `\*github.com/jackc/pgx/v5.proxyError.Error`
334. `github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
335. `github.com/valyala/fasthttp.ErrBodyStreamWritePanic.Error`
336. `\*github.com/go-playground/universal-translator.ErrOrdinalTranslation.Error`
337. `\*github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
338. `net/http.http2duplicatePseudoHeaderError.Error`
339. `github.com/ebitengine/purego.Dlerror.Error`
340. `\*github.com/golang-jwt/jwt/v5.joinedError.Error`
341. `\*crypto/aes.KeySizeError.Error`
342. `\*net/http.http2StreamError.Error`
343. `golang.org/x/net/idna.runeError.Error`
344. `\*github.com/LerianStudio/lib-commons/v4/commons/net/http.ErrorResponse.Error`
345. `\*crypto/x509.SystemRootsError.Error`
346. `\*github.com/shirou/gopsutil/v4/internal/common.Warnings.Error`
347. `github.com/LerianStudio/lib-license-go/v2/pkg.UnauthorizedError.Error`
348. `\*errors.errorString.Error`
349. `github.com/LerianStudio/lib-commons/commons.Response.Error`
350. `\*golang.org/x/net/idna.runeError.Error`
351. `\*golang.org/x/net/idna.labelError.Error`
352. `\*golang.org/x/crypto/ssh.ExitError.Error`
353. `\*golang.org/x/net/http2.noCachedConnError.Error`
354. `\*reflect.ValueError.Error`
355. `net/http.http2headerFieldNameError.Error`
356. `internal/poll.errNetClosing.Error`
357. `github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
358. `\*github.com/ebitengine/purego.Dlerror.Error`
359. `\*compress/flate.WriteError.Error`
360. `github.com/microsoft/go-mssqldb.Error.Error`
361. `go/types.Error.Error`
362. `\*github.com/go-playground/universal-translator.ErrBadPluralDefinition.Error`
363. `github.com/docker/docker/client.errConnectionFailed.Error`
364. `\*github.com/LerianStudio/lib-commons/commons.Response.Error`
365. `\*net/netip.parsePrefixError.Error`
366. `go.mongodb.org/mongo-driver/mongo.BulkWriteError.Error`
367. `\*github.com/go-openapi/jsonpointer.pointerError.Error`
368. `\*net/http.http2duplicatePseudoHeaderError.Error`
369. `\*golang.org/x/net/http2.headerFieldValueError.Error`
370. `crypto/rc4.KeySizeError.Error`
371. `\*github.com/containerd/errdefs.errConflict.Error`
372. `\*github.com/gofiber/fiber/v2/internal/schema.UnknownKeyError.Error`
373. `\*github.com/containerd/errdefs.errFailedPrecondition.Error`
374. `\*golang.org/x/net/webdav/internal/xml.TagPathError.Error`
375. `github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
376. `\*golang.org/x/text/encoding/internal.RepertoireError.Error`
377. `\*go.mongodb.org/mongo-driver/mongo.WriteError.Error`
378. `github.com/rabbitmq/amqp091-go.Error.Error`
379. `compress/flate.CorruptInputError.Error`
380. `\*github.com/LerianStudio/lib-commons/v4/commons/mongo.SanitizedError.Error`
381. `\*go.mongodb.org/mongo-driver/x/mongo/driver.Error.Error`
382. `github.com/containerd/errdefs.errPermissionDenied.Error`
383. `crypto/x509.InsecureAlgorithmError.Error`
384. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.TransitionError.Error`
385. `\*net/http.http2ConnectionError.Error`
386. `\*github.com/LerianStudio/lib-license-go/v2/pkg.EntityConflictError.Error`
387. `\*github.com/containerd/errdefs.errInvalidArgument.Error`
388. `\*golang.org/x/crypto/ssh.PartialSuccessError.Error`
389. `\*github.com/go-openapi/swag/jsonutils/adapters.registryError.Error`
390. `\*github.com/jackc/pgx/v5/pgconn.pgconnError.Error`
391. `\*github.com/redis/go-redis/v9/internal/proto.ClusterDownError.Error`
392. `github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
393. `github.com/containerd/errdefs.errNotFound.Error`
394. `\*internal/chacha8rand.errUnmarshalChaCha8.Error`
395. `github.com/LerianStudio/lib-license-go/v2/pkg.EntityNotFoundError.Error`
396. `\*google.golang.org/grpc/internal/transport.NewStreamError.Error`
397. `\*net.timeoutError.Error`
398. `go.opentelemetry.io/otel/sdk/trace.errUnsupportedSampler.Error`
399. `\*internal/fuzz.MalformedCorpusError.Error`
400. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationUnknownFieldsError.Error`
401. `\*github.com/jackc/pgx/v5/pgconn.NotPreferredError.Error`
402. `net/netip.parsePrefixError.Error`
403. `\*google.golang.org/grpc.dropError.Error`
404. `github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
405. `\*github.com/LerianStudio/lib-license-go/v2/pkg.HTTPError.Error`
406. `crypto/des.KeySizeError.Error`
407. `net/http.transportReadFromServerError.Error`
408. `github.com/containerd/errdefs.errAborted.Error`
409. `\*github.com/gofiber/fiber/v2/internal/schema.EmptyFieldError.Error`
410. `github.com/containerd/errdefs.errDataLoss.Error`
411. `go.opentelemetry.io/otel/sdk/trace.samplerArgParseError.Error`
412. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.multiErr.Error`
413. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteErrors.Error`
414. `archive/tar.headerError.Error`
415. `github.com/go-logr/logr.notFoundError.Error`
416. `\*golang.org/x/net/http2.GoAwayError.Error`
417. `\*net/url.InvalidHostError.Error`
418. `\*os.LinkError.Error`
419. `\*go.mongodb.org/mongo-driver/x/mongo/driver.QueryFailureError.Error`
420. `go.mongodb.org/mongo-driver/mongo.WriteConcernError.Error`
421. `internal/strconv.Error.Error`
422. `\*net.addrinfoErrno.Error`
423. `\*net.OpError.Error`
424. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.WaitQueueTimeoutError.Error`
425. `text/template.ExecError.Error`
426. `\*go.uber.org/multierr.multiError.Error`
427. `\*os.errSymlink.Error`
428. `\*github.com/valyala/fasthttp.ErrNothingRead.Error`
429. `go.mongodb.org/mongo-driver/bson/bsoncodec.ValueEncoderError.Error`
430. `\*math/big.ErrNaN.Error`
431. `crypto/internal/fips140/hmac.errCloneUnsupported.Error`
432. `\*encoding/json.UnmarshalFieldError.Error`
433. `\*go.mongodb.org/mongo-driver/bson/bsonrw.errMaxDocumentSizeExceeded.Error`
434. `net/http.http2pseudoHeaderError.Error`
435. `\*golang.org/x/crypto/ocsp.ResponseError.Error`
436. `github.com/gofiber/fiber/v2/internal/schema.EmptyFieldError.Error`
437. `\*github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
438. `\*github.com/go-playground/universal-translator.ErrBadParamSyntax.Error`
439. `\*github.com/docker/docker/api/types/registry.errInvalidParameter.Error`
440. `\*golang.org/x/net/http2.duplicatePseudoHeaderError.Error`
441. `\*net.DNSConfigError.Error`
442. `\*os/user.UnknownUserError.Error`
443. `\*google.golang.org/grpc/internal/transport.ioError.Error`
444. `github.com/LerianStudio/lib-license-go/v2/pkg.InternalServerError.Error`
445. `\*golang.org/x/crypto/blowfish.KeySizeError.Error`
446. `\*runtime.errorAddressString.Error`
447. `go.uber.org/zap.errArrayElem.Error`
448. `runtime.boundsError.Error`
449. `net/http.http2GoAwayError.Error`
450. `github.com/pkg/errors.withStack.Error`
451. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InsufficientBytesError.Error`
452. `\*go.opentelemetry.io/otel/trace.errorConst.Error`
453. `\*golang.org/x/net/http2.pseudoHeaderError.Error`
454. `\*github.com/LerianStudio/lib-license-go/v2/pkg.EntityNotFoundError.Error`
455. `\*github.com/LerianStudio/lib-commons/v4/commons.Response.Error`
456. `crypto/internal/fips140/aes.KeySizeError.Error`
457. `\*go.mongodb.org/mongo-driver/mongo.CommandError.Error`
458. `github.com/containerd/errdefs.errAlreadyExists.Error`
459. `golang.org/x/net/http2.GoAwayError.Error`
460. `\*github.com/containerd/errdefs.errOutOfRange.Error`
461. `\*golang.org/x/net/webdav/internal/xml.SyntaxError.Error`
462. `\*go/build.NoGoError.Error`
463. `\*github.com/tryvium-travels/memongo/mongobin.UnsupportedSystemError.Error`
464. `\*os/signal.signalError.Error`
465. `\*os/user.UnknownGroupError.Error`
466. `\*os/exec.wrappedError.Error`
467. `go.mongodb.org/mongo-driver/bson/bsoncodec.TransitionError.Error`
468. `\*github.com/docker/docker/api/types/container.errInvalidParameter.Error`
469. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ValueEncoderError.Error`
470. `\*github.com/containerd/errdefs.errNotImplemented.Error`
471. `golang.org/x/net/idna.labelError.Error`
472. `\*go/types.Error.Error`
473. `github.com/microsoft/go-mssqldb.ServerError.Error`
474. `encoding/xml.UnmarshalError.Error`
475. `\*github.com/go-playground/validator/v10.fieldError.Error`
476. `\*github.com/microsoft/go-mssqldb.ServerError.Error`
477. `\*html/template.Error.Error`
478. `\*go.mongodb.org/mongo-driver/mongo/options.MarshalError.Error`
479. `\*compress/bzip2.StructuralError.Error`
480. `\*golang.org/x/crypto/ocsp.ParseError.Error`
481. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.MalformedElementError.Error`
482. `\*go.opentelemetry.io/otel/sdk/trace.errUnsupportedSampler.Error`
483. `\*go/scanner.Error.Error`
484. `github.com/LerianStudio/lib-license-go/v2/pkg.EntityConflictError.Error`
485. `\*golang.org/x/net/http2/hpack.InvalidIndexError.Error`
486. `\*github.com/jackc/pgx/v5/pgconn.errTimeout.Error`
487. `\*github.com/valyala/fasthttp/fasthttputil.timeoutError.Error`
488. `\*net/http.transportReadFromServerError.Error`
489. `\*github.com/cenkalti/backoff/v5.PermanentError.Error`
490. `github.com/gofiber/fiber/v2/internal/schema.MultiError.Error`
491. `\*golang.org/x/crypto/ssh.OpenChannelError.Error`
492. `go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt.Error.Error`
493. `\*time.ParseError.Error`
494. `github.com/valyala/fasthttp.ErrSmallBuffer.Error`
495. `\*github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
496. `net/mail.charsetError.Error`
497. `\*github.com/redis/go-redis/v9/internal/proto.NoReplicasError.Error`
498. `\*github.com/redis/go-redis/v9/internal/proto.PermissionError.Error`
499. `\*encoding/asn1.StructuralError.Error`
500. `\*go.mongodb.org/mongo-driver/internal/aws/awserr.errorList.Error`
501. `github.com/containerd/errdefs.errUnauthorized.Error`
502. `\*github.com/microsoft/go-mssqldb.StreamError.Error`
503. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InvalidDepthTraversalError.Error`
504. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteError.Error`
505. `\*internal/strconv.Error.Error`
506. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.ServerSelectionError.Error`
507. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ForbiddenError.Error`
508. `\*golang.org/x/crypto/ssh.BannerError.Error`
509. `\*net/http.http2connError.Error`
510. `go.mongodb.org/mongo-driver/mongo.MarshalError.Error`
511. `\*github.com/redis/go-redis/v9/internal/pool.BadConnError.Error`
512. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ValueDecoderError.Error`
513. `os/user.UnknownUserIdError.Error`
514. `\*net.UnknownNetworkError.Error`
515. `\*golang.org/x/crypto/ssh.disconnectMsg.Error`
516. `\*net/http.http2httpError.Error`
517. `\*github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
518. `github.com/docker/docker/api/types/registry.errInvalidParameter.Error`
519. `github.com/grpc-ecosystem/grpc-gateway/v2/runtime.MalformedSequenceError.Error`
520. `\*internal/bisect.parseError.Error`
521. `\*github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
522. `github.com/redis/go-redis/v9/internal/proto.RedisError.Error`
523. `go.mongodb.org/mongo-driver/mongo.WriteError.Error`
524. `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal.PartialSuccess.Error`
525. `\*github.com/go-resty/resty/v2.ResponseError.Error`
526. `\*github.com/valyala/fasthttp.InvalidHostError.Error`
527. `\*encoding/asn1.SyntaxError.Error`
528. `\*regexp/syntax.Error.Error`
529. `\*github.com/moby/term.EscapeError.Error`
530. `\*go/build.MultiplePackageError.Error`
531. `github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
532. `\*github.com/containerd/errdefs.errUnknown.Error`
533. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoDecoder.Error`
534. `\*go.uber.org/zap.errSinkNotFound.Error`
535. `\*go.mongodb.org/mongo-driver/mongo.MongocryptError.Error`
536. `github.com/valyala/fasthttp.InvalidHostError.Error`
537. `go.mongodb.org/mongo-driver/bson/bsonrw.errMaxDocumentSizeExceeded.Error`
538. `\*github.com/docker/docker/api/types/filters.invalidFilter.Error`
539. `\*github.com/jackc/pgx/v5/pgconn.ParseConfigError.Error`
540. `golang.org/x/crypto/ssh.cbcError.Error`
541. `google.golang.org/grpc.dropError.Error`
542. `\*github.com/valyala/fasthttp.ErrDialWithUpstream.Error`
543. `\*github.com/andybalholm/brotli.decodeError.Error`
544. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationKnownFieldsError.Error`
545. `\*github.com/LerianStudio/lib-commons/v2/commons.Response.Error`
546. `\*github.com/go-playground/universal-translator.ErrConflictingTranslation.Error`
547. `\*github.com/grpc-ecosystem/grpc-gateway/v2/runtime.MalformedSequenceError.Error`
548. `\*net/http.ProtocolError.Error`
549. `\*encoding/xml.UnsupportedTypeError.Error`
550. `\*github.com/containerd/errdefs.errUnauthorized.Error`
551. `\*github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
552. `os/signal.signalError.Error`
553. `github.com/testcontainers/testcontainers-go.ParallelContainersError.Error`
554. `go.mongodb.org/mongo-driver/internal/aws/awserr.errorList.Error`
555. `\*github.com/LerianStudio/lib-commons/v4/commons/runtime.panicError.Error`
556. `encoding/asn1.SyntaxError.Error`
557. `github.com/containerd/errdefs.errFailedPrecondition.Error`
558. `github.com/go-openapi/swag/jsonutils/adapters.registryError.Error`
559. `github.com/containerd/errdefs.errUnknown.Error`
560. `\*github.com/containerd/errdefs.errInternal.Error`
561. `\*google.golang.org/protobuf/internal/impl.errInvalidUTF8.Error`
562. `\*encoding/json.UnsupportedValueError.Error`
563. `\*github.com/microsoft/go-mssqldb.Error.Error`
564. `\*github.com/gofiber/fiber/v2/internal/schema.ConversionError.Error`
565. `github.com/grpc-ecosystem/grpc-gateway/v2/internal/httprule.InvalidTemplateError.Error`
566. `\*encoding/json.jsonError.Error`
567. `\*github.com/yuin/gopher-lua.ApiError.Error`
568. `\*github.com/go-resty/resty/v2.restyError.Error`
569. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal.PartialSuccess.Error`
570. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteCommandError.Error`
571. `\*gopkg.in/yaml.v3.TypeError.Error`
572. `\*encoding/hex.InvalidByteError.Error`
573. `\*github.com/go-playground/universal-translator.ErrMissingLocale.Error`
574. `\*go.mongodb.org/mongo-driver/internal/aws/awserr.baseError.Error`
575. `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal.PartialSuccess.Error`
576. `\*net.canceledError.Error`
577. `golang.org/x/net/webdav/internal/xml.UnmarshalError.Error`
578. `net/http.http2ConnectionError.Error`
579. `\*github.com/LerianStudio/lib-license-go/v2/pkg.FailedPreconditionError.Error`
580. `github.com/microsoft/go-mssqldb.StreamError.Error`
581. `github.com/LerianStudio/lib-license-go/v2/pkg.UnprocessableOperationError.Error`
582. `\*go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc/internal.PartialSuccess.Error`
583. `\*go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal.PartialSuccess.Error`
584. `\*go.mongodb.org/mongo-driver/mongo.BulkWriteException.Error`
585. `golang.org/x/crypto/ocsp.ParseError.Error`
586. `go.opentelemetry.io/otel/trace.errorConst.Error`
587. `\*github.com/yuin/gopher-lua.CompileError.Error`
588. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteConcernError.Error`
589. `math/big.ErrNaN.Error`
590. `\*os/user.UnknownGroupIdError.Error`
591. `\*encoding/csv.ParseError.Error`
592. `\*github.com/jackc/pgx/v5/pgconn.PrepareError.Error`
593. `github.com/golang-jwt/jwt/v5.joinedError.Error`
594. `gopkg.in/go-playground/validator.v9.ValidationErrors.Error`
595. `crypto/x509/internal/macos.OSStatus.Error`
596. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.decodeBinaryError.Error`
597. `\*github.com/docker/docker/client.objectNotFoundError.Error`
598. `go.mongodb.org/mongo-driver/bson/bsoncodec.decodeBinaryError.Error`
599. `github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
600. `net/http.http2headerFieldValueError.Error`
601. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteError.Error`
602. `\*go.yaml.in/yaml/v3.TypeError.Error`
603. `\*go.mongodb.org/mongo-driver/mongo.MarshalError.Error`
604. `\*golang.org/x/mod/module.InvalidVersionError.Error`
605. `\*encoding/json.InvalidUnmarshalError.Error`
606. `\*github.com/tryvium-travels/memongo/mongobin.UnsupportedMongoVersionError.Error`
607. `github.com/LerianStudio/lib-license-go/v2/pkg.FailedPreconditionError.Error`
608. `runtime.errorString.Error`
609. `\*net.DNSError.Error`
610. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ResponseError.Error`
611. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoDecoder.Error`
612. `\*github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
613. `github.com/klauspost/compress/flate.InternalError.Error`
614. `\*github.com/klauspost/compress/flate.InternalError.Error`
615. `\*github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
616. `\*runtime.synctestDeadlockError.Error`
617. `\*github.com/redis/go-redis/v9/internal/proto.MaxClientsError.Error`
618. `\*github.com/valyala/fasthttp.timeoutError.Error`
619. `\*io/fs.PathError.Error`
620. `\*github.com/docker/docker/client.emptyIDError.Error`
621. `\*internal/runtime/maps.unhashableTypeError.Error`
622. `github.com/docker/docker/api/types/filters.invalidFilter.Error`
623. `go/scanner.ErrorList.Error`
624. `\*net/http.http2goAwayFlowError.Error`
625. `net.UnknownNetworkError.Error`
626. `\*net/url.EscapeError.Error`
627. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoTypeMapEntry.Error`
628. `\*github.com/go-playground/universal-translator.ErrMissingPluralTranslation.Error`
629. `\*github.com/containerd/errdefs.customMessage.Error`
630. `net/http.statusError.Error`
631. `\*github.com/rabbitmq/amqp091-go.Error.Error`
632. `\*github.com/docker/docker/api/types.ErrorResponse.Error`
633. `github.com/containerd/errdefs.errInternal.Error`
634. `\*vendor/golang.org/x/net/http2/hpack.InvalidIndexError.Error`
635. `\*github.com/lib/pq.safeRetryError.Error`
636. `runtime.plainError.Error`
637. `\*github.com/alicebob/miniredis/v2/gopher-json.invalidTypeError.Error`
638. `\*go.mongodb.org/mongo-driver/mongo.EncryptionKeyVaultError.Error`
639. `\*net.AddrError.Error`
640. `github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
641. `\*net/url.Error.Error`
642. `github.com/containerd/errdefs.errUnavailable.Error`
643. `\*net/mail.charsetError.Error`
644. `\*crypto/x509/internal/macos.OSStatus.Error`
645. `\*github.com/sijms/go-ora/v2/network.OracleError.Error`
646. `\*internal/fuzz.crashError.Error`
647. `\*net/textproto.ProtocolError.Error`
648. `\*github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
649. `github.com/andybalholm/brotli.decodeError.Error`
650. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ValidationError.Error`
651. `\*github.com/containerd/errdefs.errNotFound.Error`
652. `\*github.com/redis/go-redis/v9/internal/proto.AskError.Error`
653. `go.mongodb.org/mongo-driver/mongo.CommandError.Error`
654. `os/exec.wrappedError.Error`
655. `\*net.notFoundError.Error`
656. `crypto/x509.UnknownAuthorityError.Error`
657. `crypto/x509.HostnameError.Error`
658. `github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
659. `\*github.com/go-playground/universal-translator.ErrRangeTranslation.Error`
660. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationKnownFieldsError.Error`
661. `\*runtime.plainError.Error`
662. `github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
663. `golang.org/x/net/http2.connError.Error`
664. `\*fmt.wrapErrors.Error`
665. `\*github.com/LerianStudio/lib-commons/v4/commons/assert.AssertionError.Error`
666. `go.mongodb.org/mongo-driver/mongo.MongocryptdError.Error`
667. `\*google.golang.org/grpc/internal/transport.ConnectionError.Error`
668. `\*net/http.requestBodyReadError.Error`
669. `\*compress/flate.ReadError.Error`
670. `\*syscall.Errno.Error`
671. `\*github.com/valyala/fasthttp/reuseport.ErrNoReusePort.Error`
672. `\*golang.org/x/net/http2.connError.Error`
673. `runtime.synctestDeadlockError.Error`
674. `\*compress/flate.CorruptInputError.Error`
675. `\*github.com/redis/go-redis/v9/internal/proto.TryAgainError.Error`
676. `github.com/redis/go-redis/v9/internal/pool.BadConnError.Error`
677. `github.com/jackc/pgx/v5.ScanArgError.Error`
678. `\*net/http.MaxBytesError.Error`
679. `\*golang.org/x/crypto/ssh.AlgorithmNegotiationError.Error`
680. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoTypeMapEntry.Error`
681. `github.com/google/uuid.invalidLengthError.Error`
682. `\*github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
683. `\*golang.org/x/net/http2.StreamError.Error`
684. `\*github.com/testcontainers/testcontainers-go/wait.PermanentError.Error`
685. `\*github.com/cenkalti/backoff/v5.RetryAfterError.Error`
686. `github.com/alicebob/miniredis/v2/gopher-json.invalidTypeError.Error`
687. `github.com/valyala/fasthttp.ErrBrokenChunk.Error`
688. `\*golang.org/x/crypto/ssh.PassphraseMissingError.Error`
689. `\*github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
690. `google.golang.org/grpc/internal/transport.NewStreamError.Error`
691. `\*github.com/containerd/errdefs.errAlreadyExists.Error`
692. `net.addrinfoErrno.Error`
693. `go.mongodb.org/mongo-driver/mongo.MongocryptError.Error`
694. `encoding/base64.CorruptInputError.Error`
695. `crypto/tls.AlertError.Error`
696. `\*encoding/json.SyntaxError.Error`
697. `\*net/textproto.Error.Error`
698. `\*github.com/jackc/pgx/v5/pgproto3.writeError.Error`
699. `\*github.com/go-openapi/swag/yamlutils.yamlError.Error`
700. `\*runtime.PanicNilError.Error`
701. `github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
702. `github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
703. `\*github.com/go-playground/universal-translator.ErrMissingBracket.Error`
704. `encoding/asn1.StructuralError.Error`
705. `compress/flate.InternalError.Error`
706. `\*crypto/tls.echConfigErr.Error`
707. `\*github.com/google/uuid.invalidLengthError.Error`
708. `\*github.com/LerianStudio/lib-license-go/v2/pkg.InternalServerError.Error`
709. `go.mongodb.org/mongo-driver/mongo.BulkWriteException.Error`
710. `github.com/go-playground/validator/v10.ValidationErrors.Error`
711. `\*go.mongodb.org/mongo-driver/mongo.ErrMapForOrderedArgument.Error`
712. `\*github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
713. `github.com/containerd/errdefs.errResourceExhausted.Error`
714. `go.mongodb.org/mongo-driver/x/mongo/driver.ResponseError.Error`
715. `\*github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
716. `\*encoding/json.UnsupportedTypeError.Error`
717. `\*encoding/xml.SyntaxError.Error`
718. `\*github.com/jackc/pgx/v5/pgproto3.invalidMessageLenErr.Error`
719. `\*crypto/internal/fips140/aes.KeySizeError.Error`
720. `debug/dwarf.DecodeError.Error`
721. `\*go.mongodb.org/mongo-driver/mongo.WriteErrors.Error`
722. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.errMetric.Error`
723. `os/user.UnknownGroupError.Error`
724. `\*os/exec.ExitError.Error`
725. `\*net/http.statusError.Error`
726. `\*golang.org/x/text/internal/language.ValueError.Error`
727. `golang.org/x/text/internal/language.ValueError.Error`
728. `crypto/x509.CertificateInvalidError.Error`
729. `crypto/tls.RecordHeaderError.Error`
730. `\*internal/poll.errNetClosing.Error`
731. `\*debug/macho.FormatError.Error`
732. `github.com/LerianStudio/lib-license-go/v2/pkg.ForbiddenError.Error`
733. `\*github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
734. `\*github.com/go-openapi/swag/loading.loadingError.Error`
735. `\*github.com/docker/docker/pkg/jsonmessage.JSONError.Error`
736. `\*github.com/pkg/errors.withStack.Error`
737. `\*golang.org/x/net/http2.headerFieldNameError.Error`
738. `Contains`

#### `testContext`
**File:** `components/manager/internal/services/command/helpers\_test.go`
**Risk Level:** HIGH (97 direct callers)

**Direct Callers (signature change affects these):**

1. `TestTestConnection\_Execute\_DecryptionKeyVersionMismatch` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:652`
2. `TestGetConnection\_Execute\_AllDatabaseTypes$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:363`
3. `TestListConnections\_Execute\_Pagination$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:753`
4. `TestValidateSchema\_LargeNumberOfTables` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:985`
5. `TestListConnections\_Execute\_WithDateFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:491`
6. `TestTestConnection\_Execute\_AllDatabaseTypes$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:415`
7. `TestTestConnection\_Execute\_WithSSLConfiguration` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:337`
8. `TestGetConnection\_Execute\_ConnectionWithSSL` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:291`
9. `TestTestConnection\_Execute\_EmptyUUIDs` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:522`
10. `TestGetConnectionSchema\_Execute\_NotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:115`
11. `TestTestConnection\_Execute\_DecryptionError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:141`
12. `TestListConnections\_Execute\_WithSortOrder$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:549`
13. `TestTestConnection\_Execute\_RateLimitError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:183`
14. `TestGetJob\_Execute$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_job\_test.go:108`
15. `TestValidateSchema\_EmptyConfigName` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:929`
16. `TestGetProduct\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:40`
17. `TestValidateSchema\_DataSourceNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:88`
18. `TestTestConnection\_Execute\_RateLimited` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:253`
19. `TestValidateSchema\_InvalidRequest\_EmptyMappedFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:280`
20. `TestListConnections\_Execute\_ConnectionWithSSL` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:796`
21. `TestTestConnection\_Execute\_ConnectionWithAllFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:693`
22. `TestGetProduct\_Execute\_TableDriven$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:254`
23. `TestListProducts\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:317`
24. `TestTestConnection\_Execute\_DifferentOrganizations` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:762`
25. `TestListUnassignedConnections\_Execute\_TableDriven$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:245`
26. `TestListUnassignedConnections\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:284`
27. `TestListConnections\_Execute\_EmptyOrganizationID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:933`
28. `TestListProducts\_Execute\_EmptyList` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:84`
29. `TestListConnections\_Execute\_EmptyList` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:103`
30. `TestListConnections\_Execute\_DifferentOrganizations` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:395`
31. `TestGetConnection\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:168`
32. `TestGetConnection\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:133`
33. `TestGetConnection\_Execute\_TableDriven$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:245`
34. `TestGetConnection\_Execute\_ConnectionWithAllFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:418`
35. `TestGetProduct\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:177`
36. `TestGetConnectionSchema\_Execute\_DataSourceFactoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:183`
37. `TestTestConnection\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:95`
38. `TestGetConnectionSchema\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:55`
39. `TestListConnections\_Execute\_EmptyFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:686`
40. `TestGetConnectionSchema\_Execute\_NilSchema` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:399`
41. `TestListConnections\_Execute\_WithProductFilter\_NotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:1024`
42. `TestTestConnection\_Execute\_MultipleRepositoryErrors$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:484`
43. `TestTestConnection\_Execute\_NotFoundError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:55`
44. `TestValidateSchema\_MultipleDatasources\_AllValid` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:522`
45. `TestValidateSchema\_InvalidRequest\_NilMappedFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:257`
46. `TestValidateSchema\_NilSchemaFromDatasource` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:902`
47. `TestListConnections\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:350`
48. `TestListConnections\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:55`
49. `TestListConnections\_Execute\_TableDriven$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:308`
50. `TestGetConnectionSchema\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:150`
51. `TestListProducts\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:39`
52. `TestListUnassignedConnections\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:103`
53. `TestGetProduct\_Execute\_EmptyUUIDs` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:429`
54. `TestValidateSchema\_TableNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:135`
55. `TestGetConnection\_Execute\_EmptyUUIDs` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:568`
56. `TestValidateSchema\_PartialConnectionsFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:411`
57. `TestNewLoggerFromContext$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:57`
58. `TestListUnassignedConnections\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:23`
59. `TestValidateSchema\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:55`
60. `TestGetConnectionSchema\_Execute\_GetSchemaInfoError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:219`
61. `TestValidateSchema\_FieldNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:181`
62. `TestValidateSchema\_CacheError\_ContinuesToFetch` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:367`
63. `TestGetConnectionSchema\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:483`
64. `TestValidateSchema\_PostgreSQLWithMixedQualifiedTables` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:718`
65. `TestListProducts\_Execute\_ErrorScenarios$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:395`
66. `TestGetProduct\_Execute\_ProductWithAllFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:300`
67. `TestValidateSchema\_PostgreSQLWithUnqualifiedTables` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:669`
68. `TestGetProduct\_Execute\_NotFoundError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:89`
69. `TestListUnassignedConnections\_Execute\_EmptyList` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:69`
70. `TestValidateSchema\_MultipleErrors` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:228`
71. `TestGetProduct\_Execute\_MultipleRepositoryErrors$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:394`
72. `TestValidateSchema\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:310`
73. `TestListConnections\_Execute\_WithCursor` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:650`
74. `TestListConnections\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:139`
75. `TestListProducts\_Execute\_TableDriven$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:277`
76. `TestGetConnectionSchema\_Execute\_FiltersSystemTables$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:338`
77. `TestGetProduct\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:124`
78. `TestListProducts\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:118`
79. `TestListUnassignedConnections\_Execute\_ErrorScenarios$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:381`
80. `TestValidateSchema\_NonPostgreSQLDoesNotAddPublicSchema$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:783`
81. `TestListConnections\_Execute\_ErrorScenarios$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:897`
82. `TestListConnections\_Execute\_WithMetadataFilter` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:451`
83. `TestTestConnection\_Execute\_RateLimitResetTime$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:612`
84. `TestContextWithTracer$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:221`
85. `TestTestConnection\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:227`
86. `TestListConnections\_Execute\_WithProductFilter\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:966`
87. `TestValidateSchema\_CacheSetError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:843`
88. `TestListConnections\_Execute\_WithProductFilter\_RepoError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:1066`
89. `TestContextWithLogger$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:167`
90. `TestNewTracerFromContext$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:112`
91. `TestListConnections\_Execute\_ConnectionTypes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:589`
92. `TestGetConnection\_Execute\_MultipleRepositoryErrors$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:533`
93. `TestValidateSchema\_NoConnections\_ReturnsSchemaEntityType` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:467`
94. `TestTestConnection\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:293`
95. `TestGetConnection\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:45`
96. `TestGetConnection\_Execute\_NotFoundError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:98`
97. `TestGetConnectionSchema\_Execute\_EmptySchema` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:441`

**Callees (this function depends on):**

1. `Tracer`
2. `Background`
3. `WithValue`

#### `testContext`
**File:** `components/manager/internal/services/query/helpers\_test.go`
**Risk Level:** HIGH (97 direct callers)

**Direct Callers (signature change affects these):**

1. `TestTestConnection\_Execute\_DecryptionKeyVersionMismatch` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:652`
2. `TestGetConnection\_Execute\_AllDatabaseTypes$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:363`
3. `TestListConnections\_Execute\_Pagination$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:753`
4. `TestValidateSchema\_LargeNumberOfTables` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:985`
5. `TestListConnections\_Execute\_WithDateFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:491`
6. `TestTestConnection\_Execute\_AllDatabaseTypes$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:415`
7. `TestTestConnection\_Execute\_WithSSLConfiguration` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:337`
8. `TestGetConnection\_Execute\_ConnectionWithSSL` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:291`
9. `TestTestConnection\_Execute\_EmptyUUIDs` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:522`
10. `TestGetConnectionSchema\_Execute\_NotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:115`
11. `TestTestConnection\_Execute\_DecryptionError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:141`
12. `TestListConnections\_Execute\_WithSortOrder$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:549`
13. `TestTestConnection\_Execute\_RateLimitError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:183`
14. `TestGetJob\_Execute$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_job\_test.go:108`
15. `TestValidateSchema\_EmptyConfigName` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:929`
16. `TestGetProduct\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:40`
17. `TestValidateSchema\_DataSourceNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:88`
18. `TestTestConnection\_Execute\_RateLimited` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:253`
19. `TestValidateSchema\_InvalidRequest\_EmptyMappedFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:280`
20. `TestListConnections\_Execute\_ConnectionWithSSL` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:796`
21. `TestTestConnection\_Execute\_ConnectionWithAllFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:693`
22. `TestGetProduct\_Execute\_TableDriven$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:254`
23. `TestListProducts\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:317`
24. `TestTestConnection\_Execute\_DifferentOrganizations` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:762`
25. `TestListUnassignedConnections\_Execute\_TableDriven$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:245`
26. `TestListUnassignedConnections\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:284`
27. `TestListConnections\_Execute\_EmptyOrganizationID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:933`
28. `TestListProducts\_Execute\_EmptyList` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:84`
29. `TestListConnections\_Execute\_EmptyList` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:103`
30. `TestListConnections\_Execute\_DifferentOrganizations` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:395`
31. `TestGetConnection\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:168`
32. `TestGetConnection\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:133`
33. `TestGetConnection\_Execute\_TableDriven$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:245`
34. `TestGetConnection\_Execute\_ConnectionWithAllFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:418`
35. `TestGetProduct\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:177`
36. `TestGetConnectionSchema\_Execute\_DataSourceFactoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:183`
37. `TestTestConnection\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:95`
38. `TestGetConnectionSchema\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:55`
39. `TestListConnections\_Execute\_EmptyFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:686`
40. `TestGetConnectionSchema\_Execute\_NilSchema` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:399`
41. `TestListConnections\_Execute\_WithProductFilter\_NotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:1024`
42. `TestTestConnection\_Execute\_MultipleRepositoryErrors$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:484`
43. `TestTestConnection\_Execute\_NotFoundError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:55`
44. `TestValidateSchema\_MultipleDatasources\_AllValid` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:522`
45. `TestValidateSchema\_InvalidRequest\_NilMappedFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:257`
46. `TestValidateSchema\_NilSchemaFromDatasource` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:902`
47. `TestListConnections\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:350`
48. `TestListConnections\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:55`
49. `TestListConnections\_Execute\_TableDriven$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:308`
50. `TestGetConnectionSchema\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:150`
51. `TestListProducts\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:39`
52. `TestListUnassignedConnections\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:103`
53. `TestGetProduct\_Execute\_EmptyUUIDs` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:429`
54. `TestValidateSchema\_TableNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:135`
55. `TestGetConnection\_Execute\_EmptyUUIDs` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:568`
56. `TestValidateSchema\_PartialConnectionsFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:411`
57. `TestNewLoggerFromContext$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:57`
58. `TestListUnassignedConnections\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:23`
59. `TestValidateSchema\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:55`
60. `TestGetConnectionSchema\_Execute\_GetSchemaInfoError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:219`
61. `TestValidateSchema\_FieldNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:181`
62. `TestValidateSchema\_CacheError\_ContinuesToFetch` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:367`
63. `TestGetConnectionSchema\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:483`
64. `TestValidateSchema\_PostgreSQLWithMixedQualifiedTables` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:718`
65. `TestListProducts\_Execute\_ErrorScenarios$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:395`
66. `TestGetProduct\_Execute\_ProductWithAllFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:300`
67. `TestValidateSchema\_PostgreSQLWithUnqualifiedTables` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:669`
68. `TestGetProduct\_Execute\_NotFoundError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:89`
69. `TestListUnassignedConnections\_Execute\_EmptyList` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:69`
70. `TestValidateSchema\_MultipleErrors` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:228`
71. `TestGetProduct\_Execute\_MultipleRepositoryErrors$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:394`
72. `TestValidateSchema\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:310`
73. `TestListConnections\_Execute\_WithCursor` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:650`
74. `TestListConnections\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:139`
75. `TestListProducts\_Execute\_TableDriven$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:277`
76. `TestGetConnectionSchema\_Execute\_FiltersSystemTables$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:338`
77. `TestGetProduct\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_product\_test.go:124`
78. `TestListProducts\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_products\_test.go:118`
79. `TestListUnassignedConnections\_Execute\_ErrorScenarios$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_unassigned\_connections\_test.go:381`
80. `TestValidateSchema\_NonPostgreSQLDoesNotAddPublicSchema$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:783`
81. `TestListConnections\_Execute\_ErrorScenarios$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:897`
82. `TestListConnections\_Execute\_WithMetadataFilter` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:451`
83. `TestTestConnection\_Execute\_RateLimitResetTime$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:612`
84. `TestContextWithTracer$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:221`
85. `TestTestConnection\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:227`
86. `TestListConnections\_Execute\_WithProductFilter\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:966`
87. `TestValidateSchema\_CacheSetError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:843`
88. `TestListConnections\_Execute\_WithProductFilter\_RepoError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:1066`
89. `TestContextWithLogger$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:167`
90. `TestNewTracerFromContext$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:112`
91. `TestListConnections\_Execute\_ConnectionTypes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/list\_connections\_test.go:589`
92. `TestGetConnection\_Execute\_MultipleRepositoryErrors$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:533`
93. `TestValidateSchema\_NoConnections\_ReturnsSchemaEntityType` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:467`
94. `TestTestConnection\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:293`
95. `TestGetConnection\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:45`
96. `TestGetConnection\_Execute\_NotFoundError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_test.go:98`
97. `TestGetConnectionSchema\_Execute\_EmptySchema` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/get\_connection\_schema\_test.go:441`

**Callees (this function depends on):**

1. `Tracer`
2. `Background`
3. `WithValue`

#### `NewTestConnection`
**File:** `components/manager/internal/services/query/test\_connection.go`
**Risk Level:** HIGH (16 direct callers)

**Direct Callers (signature change affects these):**

1. `TestTestConnection\_Execute\_DecryptionKeyVersionMismatch` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:650`
2. `TestTestConnection\_Execute\_AllDatabaseTypes$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:413`
3. `TestTestConnection\_Execute\_WithSSLConfiguration` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:335`
4. `TestTestConnection\_Execute\_EmptyUUIDs` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:520`
5. `TestTestConnection\_Execute\_DecryptionError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:139`
6. `TestTestConnection\_Execute\_RateLimitError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:181`
7. `TestTestConnection\_Execute\_RateLimited` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:251`
8. `TestNewTestConnection` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:557`
9. `TestTestConnection\_Execute\_ConnectionWithAllFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:691`
10. `TestTestConnection\_Execute\_DifferentOrganizations` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:760`
11. `TestTestConnection\_Execute\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:93`
12. `TestTestConnection\_Execute\_MultipleRepositoryErrors$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:482`
13. `TestTestConnection\_Execute\_NotFoundError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:53`
14. `TestTestConnection\_Execute\_RateLimitResetTime$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:610`
15. `TestTestConnection\_Execute\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:220`
16. `TestTestConnection\_Execute\_OrganizationIsolation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection\_test.go:291`

**Callees (this function depends on):**


#### `TestTestConnection\_Execute\_WithSSLConfiguration`
**File:** `components/manager/internal/services/query/test\_connection\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockRepository`
4. `NewMockCryptor`
5. `NewMockRateLimiterStore`
6. `NewMockDataSource`
7. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.MockRateLimiterStore.EXPECT`
8. `Any`
9. `Any`
10. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.MockRateLimiterStoreMockRecorder.Take`
11. `Now`
12. `time.Time.UTC`
13. `time.Time.Add`
14. `time.Time.UnixNano`
15. `\*go.uber.org/mock/gomock.Call.Return`
16. `NewTestConnection`
17. `testContext`
18. `New`
19. `New`
20. `newTestConnectionFixture`
21. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepository.EXPECT`
22. `Any`
23. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepositoryMockRecorder.FindByID`
24. `\*go.uber.org/mock/gomock.Call.Return`
25. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSource.EXPECT`
26. `Any`
27. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSourceMockRecorder.Close`
28. `\*go.uber.org/mock/gomock.Call.Return`
29. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.TestConnection.Execute`
30. `NoError`
31. `NotNil`
32. `Equal`

#### `TestTestConnection\_Execute\_AllDatabaseTypes`
**File:** `components/manager/internal/services/query/test\_connection\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`

#### `TestTestConnection\_Execute\_Success`
**File:** `components/manager/internal/services/query/test\_connection\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockRepository`
4. `NewMockCryptor`
5. `NewMockRateLimiterStore`
6. `NewMockDataSource`
7. `New`
8. `New`
9. `newTestConnectionFixture`
10. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.MockRateLimiterStore.EXPECT`
11. `Any`
12. `Any`
13. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.MockRateLimiterStoreMockRecorder.Take`
14. `Now`
15. `time.Time.UTC`
16. `time.Time.Add`
17. `time.Time.UnixNano`
18. `\*go.uber.org/mock/gomock.Call.Return`
19. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepository.EXPECT`
20. `Any`
21. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepositoryMockRecorder.FindByID`
22. `\*go.uber.org/mock/gomock.Call.Return`
23. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSource.EXPECT`
24. `Any`
25. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSourceMockRecorder.Close`
26. `\*go.uber.org/mock/gomock.Call.Return`
27. `NewTestConnection`
28. `testContext`
29. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.TestConnection.Execute`
30. `NoError`
31. `NotNil`
32. `Equal`
33. `Equal`
34. `GreaterOrEqual`

#### `NewValidateSchema`
**File:** `components/manager/internal/services/query/validate\_schema.go`
**Risk Level:** HIGH (20 direct callers)

**Direct Callers (signature change affects these):**

1. `TestValidateSchema\_LargeNumberOfTables` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:983`
2. `TestValidateSchema\_EmptyConfigName` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:927`
3. `TestValidateSchema\_DataSourceNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:86`
4. `TestValidateSchema\_InvalidRequest\_EmptyMappedFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:278`
5. `TestValidateSchema\_MultipleDatasources\_AllValid` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:520`
6. `TestValidateSchema\_InvalidRequest\_NilMappedFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:255`
7. `TestValidateSchema\_NilSchemaFromDatasource` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:897`
8. `TestValidateSchema\_TableNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:133`
9. `TestValidateSchema\_PartialConnectionsFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:409`
10. `TestNewValidateSchema` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:443`
11. `TestValidateSchema\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:53`
12. `TestValidateSchema\_FieldNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:179`
13. `TestValidateSchema\_CacheError\_ContinuesToFetch` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:360`
14. `TestValidateSchema\_PostgreSQLWithMixedQualifiedTables` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:716`
15. `TestValidateSchema\_PostgreSQLWithUnqualifiedTables` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:667`
16. `TestValidateSchema\_MultipleErrors` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:226`
17. `TestValidateSchema\_RepositoryError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:308`
18. `TestValidateSchema\_NonPostgreSQLDoesNotAddPublicSchema$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:781`
19. `TestValidateSchema\_CacheSetError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:836`
20. `TestValidateSchema\_NoConnections\_ReturnsSchemaEntityType` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema\_test.go:465`

**Callees (this function depends on):**


#### `TestValidateSchema\_CacheSetError`
**File:** `components/manager/internal/services/query/validate\_schema\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockRepository`
4. `NewMockCryptor`
5. `NewMockSchemaCacheRepository`
6. `NewMockDataSource`
7. `New`
8. `New`
9. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepository.EXPECT`
10. `Any`
11. `Any`
12. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepositoryMockRecorder.FindByConfigNames`
13. `\*go.uber.org/mock/gomock.Call.Return`
14. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepository.EXPECT`
15. `Any`
16. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepositoryMockRecorder.Get`
17. `\*go.uber.org/mock/gomock.Call.Return`
18. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSource.EXPECT`
19. `Any`
20. `Any`
21. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSourceMockRecorder.GetSchemaInfo`
22. `\*go.uber.org/mock/gomock.Call.Return`
23. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSource.EXPECT`
24. `Any`
25. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSourceMockRecorder.Close`
26. `\*go.uber.org/mock/gomock.Call.Return`
27. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepository.EXPECT`
28. `Any`
29. `Any`
30. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepositoryMockRecorder.Set`
31. `New`
32. `\*go.uber.org/mock/gomock.Call.Return`
33. `NewValidateSchema`
34. `testContext`
35. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.ValidateSchema.Execute`
36. `NoError`
37. `NotNil`
38. `Equal`
39. `Empty`

#### `TestValidateSchema\_CacheError\_ContinuesToFetch`
**File:** `components/manager/internal/services/query/validate\_schema\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockRepository`
4. `NewMockCryptor`
5. `NewMockSchemaCacheRepository`
6. `NewMockDataSource`
7. `New`
8. `New`
9. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepository.EXPECT`
10. `Any`
11. `Any`
12. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepositoryMockRecorder.FindByConfigNames`
13. `\*go.uber.org/mock/gomock.Call.Return`
14. `New`
15. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepository.EXPECT`
16. `Any`
17. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepositoryMockRecorder.Get`
18. `\*go.uber.org/mock/gomock.Call.Return`
19. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSource.EXPECT`
20. `Any`
21. `Any`
22. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSourceMockRecorder.GetSchemaInfo`
23. `\*go.uber.org/mock/gomock.Call.Return`
24. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSource.EXPECT`
25. `Any`
26. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSourceMockRecorder.Close`
27. `\*go.uber.org/mock/gomock.Call.Return`
28. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepository.EXPECT`
29. `Any`
30. `Any`
31. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepositoryMockRecorder.Set`
32. `\*go.uber.org/mock/gomock.Call.Return`
33. `NewValidateSchema`
34. `testContext`
35. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.ValidateSchema.Execute`
36. `NoError`
37. `NotNil`
38. `Equal`
39. `Empty`

#### `TestValidateSchema\_NilSchemaFromDatasource`
**File:** `components/manager/internal/services/query/validate\_schema\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockRepository`
4. `NewMockCryptor`
5. `NewMockSchemaCacheRepository`
6. `NewMockDataSource`
7. `New`
8. `New`
9. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepository.EXPECT`
10. `Any`
11. `Any`
12. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockRepositoryMockRecorder.FindByConfigNames`
13. `\*go.uber.org/mock/gomock.Call.Return`
14. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepository.EXPECT`
15. `Any`
16. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepositoryMockRecorder.Get`
17. `\*go.uber.org/mock/gomock.Call.Return`
18. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSource.EXPECT`
19. `Any`
20. `Any`
21. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSourceMockRecorder.GetSchemaInfo`
22. `\*go.uber.org/mock/gomock.Call.Return`
23. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSource.EXPECT`
24. `Any`
25. `\*github.com/LerianStudio/fetcher/pkg/model/datasource.MockDataSourceMockRecorder.Close`
26. `\*go.uber.org/mock/gomock.Call.Return`
27. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepository.EXPECT`
28. `Any`
29. `Any`
30. `\*github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache.MockSchemaCacheRepositoryMockRecorder.Set`
31. `\*go.uber.org/mock/gomock.Call.DoAndReturn`
32. `NewValidateSchema`
33. `testContext`
34. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.ValidateSchema.Execute`
35. `NoError`
36. `NotNil`
37. `Equal`
38. `Len`
39. `Equal`

#### `NewConsumerRoutesWithAdapter`
**File:** `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go`
**Risk Level:** MEDIUM (4 direct callers)

**Direct Callers (signature change affects these):**

1. `TestNewMultiQueueConsumerRegistersQueue` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer\_service\_test.go:45`
2. `TestMultiQueueConsumerRun$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer\_service\_test.go:159`
3. `TestMultiQueueConsumerRun$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer\_service\_test.go:138`
4. `NewConsumerRoutes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go:44`

**Callees (this function depends on):**


#### `TestNewConsumerRoutes`
**File:** `components/worker/internal/adapters/rabbitmq/consumer\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockAdapter`
4. `NewNop`
5. `\*testing.T.Run`
6. `\*testing.T.Run`

#### `TestConsumerRoutes\_Shutdown`
**File:** `components/worker/internal/adapters/rabbitmq/consumer\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockAdapter`
4. `\*github.com/LerianStudio/fetcher/pkg/rabbitmq.MockAdapter.EXPECT`
5. `Any`
6. `\*github.com/LerianStudio/fetcher/pkg/rabbitmq.MockAdapterMockRecorder.Shutdown`
7. `\*go.uber.org/mock/gomock.Call.Return`
8. `NewNop`
9. `\*sync.WaitGroup.Add`
10. `TestConsumerRoutes\_Shutdown$1`
11. `TestConsumerRoutes\_Shutdown$2`
12. `After`
13. `NoError`
14. `\*testing.common.Error`

#### `TestConsumerRoutes\_Shutdown\_Error`
**File:** `components/worker/internal/adapters/rabbitmq/consumer\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockAdapter`
4. `\*github.com/LerianStudio/fetcher/pkg/rabbitmq.MockAdapter.EXPECT`
5. `Any`
6. `\*github.com/LerianStudio/fetcher/pkg/rabbitmq.MockAdapterMockRecorder.Shutdown`
7. `\*go.uber.org/mock/gomock.Call.Return`
8. `NewNop`
9. `Background`
10. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Shutdown`
11. `Error`
12. `Equal`

#### `TestConsumerRoutes\_RunConsumers`
**File:** `components/worker/internal/adapters/rabbitmq/consumer\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`

#### `NewPublisherRoutesWithAdapter`
**File:** `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go`
**Risk Level:** HIGH (6 direct callers)

**Direct Callers (signature change affects these):**

1. `TestPublisherRoutes\_Publish$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher\_test.go:52`
2. `TestNewPublisherRoutes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher\_test.go:23`
3. `TestPublisherRoutes\_Shutdown$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher\_test.go:70`
4. `NewPublisherRoutes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go:37`
5. `TestPublisherRoutes\_Publish$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher\_test.go:42`
6. `TestPublisherRoutes\_Shutdown$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/adapters/rabbitmq/publisher\_test.go:78`

**Callees (this function depends on):**


#### `TestPublisherRoutes\_Shutdown`
**File:** `components/worker/internal/adapters/rabbitmq/publisher\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockAdapter`
4. `NewNop`
5. `Background`
6. `\*testing.T.Run`
7. `\*testing.T.Run`

#### `TestNewPublisherRoutes`
**File:** `components/worker/internal/adapters/rabbitmq/publisher\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockAdapter`
4. `NewNop`
5. `NewPublisherRoutesWithAdapter`
6. `NotNil`
7. `Equal`

#### `TestPublisherRoutes\_Publish`
**File:** `components/worker/internal/adapters/rabbitmq/publisher\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockAdapter`
4. `NewNop`
5. `Background`
6. `\*testing.T.Run`
7. `\*testing.T.Run`

#### `InitWorker`
**File:** `components/worker/internal/bootstrap/config.go`
**Risk Level:** MEDIUM (3 direct callers)

**Direct Callers (signature change affects these):**

1. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config\_test.go:181`
2. `TestInitWorker\_PanicsWhenLoggerInitFails` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config\_test.go:136`
3. `TestInitWorker\_PanicsWhenConfigLoadFails` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config\_test.go:103`

**Callees (this function depends on):**

1. `NullAssignTo`
2. `TestInitWorker\_PanicsWhenConfigLoadFails$2`
3. `UnmarshalJSON$1`
4. `SetConfigFromEnvVars`
5. `SetConfigFromEnvVars`
6. `TestInitWorker\_PanicsWhenLoggerInitFails$2`
7. `SetConfigFromEnvVars`
8. `ValidateStruct`
9. `NewDecoder$1`
10. `NewEncoder$1`
11. `ValidateStruct`
12. `SetConfigFromEnvVars`
13. `newPanicError`
14. `validateField`
15. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$2`
16. `ValidateStruct`
17. `unpackError`
18. `callObsoleteUnmarshaler$1`
19. `NewEncoder$1`
20. `callObsoleteUnmarshaler$1`
21. `SetConfigFromEnvVars`
22. `processUnaryRPC$3`
23. `resolveZapEnvironment`
24. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$3`
25. `init$1`
26. `TestInitWorker\_PanicsWhenLoggerInitFails$3`
27. `init$1`
28. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$4`
29. `NewTelemetry`
30. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$5`
31. `init$2`
32. `init$2`
33. `PathEscape`
34. `QueryEscape`
35. `Sprintf`
36. `DecodeMasterKey`
37. `must`
38. `NewHKDFKeyDeriver`
39. `must`
40. `Background`
41. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
42. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
43. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
44. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
45. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
46. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
47. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
48. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
49. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
50. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
51. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
52. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
53. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
54. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
55. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
56. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
57. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
58. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
59. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
60. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
61. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
62. `\*github.com/LerianStudio/fetcher/pkg/crypto.HKDFKeyDeriver.GetCredentialKey`
63. `NewAESGCMService`
64. `must`
65. `\*github.com/LerianStudio/fetcher/pkg/crypto.HKDFKeyDeriver.GetInternalHMACKey`
66. `NewHMACSigner`
67. `must`
68. `\*github.com/LerianStudio/fetcher/pkg/crypto.HKDFKeyDeriver.GetExternalHMACKey`
69. `NewHMACSigner`
70. `must`
71. `NewConsumerRoutes`
72. `NewPublisherRoutes`
73. `Sprintf`
74. `NewSeaweedFSClient`
75. `QueryEscape`
76. `Sprintf`
77. `Background`
78. `NewClient`
79. `must`
80. `NewSimpleRepository`
81. `NewJobMongoDBRepository`
82. `must`
83. `NewConnectionMongoDBRepository`
84. `must`
85. `Background`
86. `Sprintf`
87. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
88. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
89. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
90. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
91. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
92. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
93. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
94. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
95. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
96. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
97. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
98. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
99. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
100. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
101. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
102. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
103. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
104. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
105. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
106. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
107. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
108. `InitializeLogger`
109. `NewLicenseClient`
110. `NewMultiQueueConsumer`
111. `\*github.com/LerianStudio/lib-license-go/v2/middleware.LicenseClient.GetLicenseManagerShutdown`
112. `Background`
113. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
114. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
115. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
116. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
117. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
118. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
119. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
120. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
121. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
122. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
123. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
124. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
125. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
126. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
127. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
128. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
129. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
130. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
131. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
132. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
133. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`

#### `must`
**File:** `components/worker/internal/bootstrap/config.go`
**Risk Level:** HIGH (13 direct callers)

**Direct Callers (signature change affects these):**

1. `ForEachPackage` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/allpackages.go:68`
2. `initPlatformDependencies` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:245`
3. `initCrypto` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:197`
4. `initCrypto` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:200`
5. `initCrypto` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:204`
6. `initCrypto` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:207`
7. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:171`
8. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:174`
9. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:176`
10. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:179`
11. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:181`
12. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:184`
13. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:186`

**Callees (this function depends on):**

1. `Errorf`

#### `TestInitWorker\_PanicsWhenTelemetryGlobalsFail`
**File:** `components/worker/internal/bootstrap/config\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.common.Cleanup`
2. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$6`
3. `InitWorker`

#### `testBootstrapLogger`
**File:** `components/worker/internal/bootstrap/config\_test.go`
**Risk Level:** HIGH (7 direct callers)

**Direct Callers (signature change affects these):**

1. `TestNewMultiQueueConsumerRegistersQueue` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer\_service\_test.go:45`
2. `contextWithBootstrapTracking` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer\_service\_test.go:239`
3. `TestMultiQueueConsumerRun$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer\_service\_test.go:159`
4. `TestServiceRun$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer\_service\_test.go:218`
5. `TestMultiQueueConsumerRun$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer\_service\_test.go:138`
6. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config\_test.go:159`
7. `TestServiceRun$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/consumer\_service\_test.go:178`

**Callees (this function depends on):**


#### `TestResolveZapEnvironment`
**File:** `components/worker/internal/bootstrap/config\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Parallel`
2. `\*testing.T.Run`

#### `TestMust`
**File:** `components/worker/internal/bootstrap/config\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Parallel`
2. `\*testing.T.Run`
3. `\*testing.T.Run`

#### `TestInitWorker\_PanicsWhenConfigLoadFails`
**File:** `components/worker/internal/bootstrap/config\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.common.Cleanup`
2. `TestInitWorker\_PanicsWhenConfigLoadFails$3`
3. `InitWorker`

#### `TestInitWorker\_PanicsWhenLoggerInitFails`
**File:** `components/worker/internal/bootstrap/config\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.common.Cleanup`
2. `TestInitWorker\_PanicsWhenLoggerInitFails$4`
3. `InitWorker`

#### `TestQueryPluginCRMCollectionWithFilters\_NoFilters`
**File:** `components/worker/internal/services/extract\_crm\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `\*testing.common.Cleanup`
4. `newTestMocks`
5. `newTestUseCase`
6. `testContext`
7. `testLogger`
8. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.queryPluginCRMCollectionWithFilters`
9. `\*testing.common.Fatalf`
10. `\*testing.common.Fatalf`

#### `TestQueryPluginCRM\_WithOrganizationOnly`
**File:** `components/worker/internal/services/extract\_crm\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `\*testing.common.Cleanup`
4. `newTestMocks`
5. `newTestUseCase`
6. `testContext`
7. `testLogger`
8. `MustParse`
9. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.QueryPluginCRM`
10. `\*testing.common.Fatalf`
11. `\*testing.common.Fatal`

#### `TestQueryPluginCRM\_WithFilters`
**File:** `components/worker/internal/services/extract\_crm\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `\*testing.common.Cleanup`
4. `newTestMocks`
5. `newTestUseCase`
6. `testContext`
7. `testLogger`
8. `MustParse`
9. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.QueryPluginCRM`
10. `\*testing.common.Fatalf`
11. `\*testing.common.Fatal`

#### `TestProcessPluginCRMCollection\_WithValidOrganization`
**File:** `components/worker/internal/services/extract\_crm\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `\*testing.common.Cleanup`
4. `newTestMocks`
5. `newTestUseCase`
6. `testContext`
7. `testLogger`
8. `MustParse`
9. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.processPluginCRMCollection`
10. `\*testing.common.Fatalf`
11. `\*github.com/pkg/errors.withMessage.Error`
12. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationError.Error`
13. `\*github.com/docker/docker/client.httpError.Error`
14. `\*runtime.boundsError.Error`
15. `\*gopkg.in/go-playground/validator.v9.InvalidValidationError.Error`
16. `go.mongodb.org/mongo-driver/mongo/options.MarshalError.Error`
17. `\*go.mongodb.org/mongo-driver/internal/codecutil.MarshalError.Error`
18. `github.com/docker/docker/api/types.ErrorResponse.Error`
19. `\*go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt.Error.Error`
20. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationError.Error`
21. `github.com/docker/docker/client.emptyIDError.Error`
22. `\*github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
23. `vendor/golang.org/x/net/http2/hpack.DecodingError.Error`
24. `\*net/http.unsupportedTEError.Error`
25. `\*net/http.http2headerFieldNameError.Error`
26. `\*github.com/docker/docker/api/types/network.joinError.Error`
27. `\*go.mongodb.org/mongo-driver/x/mongo/driver/auth.Error.Error`
28. `\*net.ParseError.Error`
29. `github.com/go-openapi/swag/yamlutils.yamlError.Error`
30. `runtime.errorAddressString.Error`
31. `\*net.InvalidAddrError.Error`
32. `\*crypto/x509.UnknownAuthorityError.Error`
33. `\*github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
34. `golang.org/x/crypto/ssh.ServerAuthError.Error`
35. `\*github.com/go-playground/validator/v10.ValidationErrors.Error`
36. `golang.org/x/net/http2.pseudoHeaderError.Error`
37. `golang.org/x/crypto/ocsp.ResponseError.Error`
38. `go.mongodb.org/mongo-driver/x/mongo/driver.QueryFailureError.Error`
39. `github.com/docker/docker/api/types/container.errInvalidParameter.Error`
40. `\*net/http.http2pseudoHeaderError.Error`
41. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.MalformedElementError.Error`
42. `\*crypto/tls.alert.Error`
43. `\*crypto/x509.InsecureAlgorithmError.Error`
44. `go.mongodb.org/mongo-driver/mongo.WriteErrors.Error`
45. `\*encoding/asn1.invalidUnmarshalError.Error`
46. `crypto/x509.UnhandledCriticalExtension.Error`
47. `go.mongodb.org/mongo-driver/bson/bsoncodec.ValueDecoderError.Error`
48. `github.com/containerd/errdefs.customMessage.Error`
49. `\*github.com/xdg-go/stringprep.Error.Error`
50. `\*crypto/tls.CertificateVerificationError.Error`
51. `\*fmt.wrapError.Error`
52. `\*github.com/yuin/gopher-lua/pm.Error.Error`
53. `github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
54. `\*net/http.http2headerFieldValueError.Error`
55. `golang.org/x/net/http2.duplicatePseudoHeaderError.Error`
56. `os.errSymlink.Error`
57. `vendor/golang.org/x/net/idna.labelError.Error`
58. `\*crypto/tls.RecordHeaderError.Error`
59. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoEncoder.Error`
60. `\*golang.org/x/mod/module.ModuleError.Error`
61. `\*encoding/json.UnmarshalTypeError.Error`
62. `github.com/containerd/errdefs.errInvalidArgument.Error`
63. `\*google.golang.org/protobuf/internal/errors.prefixError.Error`
64. `\*encoding/base64.CorruptInputError.Error`
65. `\*golang.org/x/mod/module.InvalidPathError.Error`
66. `\*go.mongodb.org/mongo-driver/x/mongo/driver.ResponseError.Error`
67. `net/netip.parseAddrError.Error`
68. `go/scanner.Error.Error`
69. `\*internal/poll.DeadlineExceededError.Error`
70. `\*golang.org/x/crypto/ssh.ExitMissingError.Error`
71. `time.fileSizeError.Error`
72. `\*github.com/redis/go-redis/v9/internal/proto.RedisError.Error`
73. `golang.org/x/net/http2.ConnectionError.Error`
74. `github.com/LerianStudio/lib-commons/v2/commons.Response.Error`
75. `go.mongodb.org/mongo-driver/internal/codecutil.MarshalError.Error`
76. `golang.org/x/net/http2/hpack.InvalidIndexError.Error`
77. `\*strconv.NumError.Error`
78. `\*go.opentelemetry.io/otel/sdk/trace.samplerArgParseError.Error`
79. `github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
80. `\*os/user.UnknownUserIdError.Error`
81. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteConcernError.Error`
82. `net/http.nothingWrittenError.Error`
83. `\*github.com/redis/go-redis/v9/internal/proto.MasterDownError.Error`
84. `\*go/scanner.ErrorList.Error`
85. `\*go.mongodb.org/mongo-driver/mongo.WriteException.Error`
86. `github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
87. `\*github.com/docker/docker/client.errConnectionFailed.Error`
88. `\*vendor/golang.org/x/net/dns/dnsmessage.nestedError.Error`
89. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InvalidDepthTraversalError.Error`
90. `net/http.requestBodyReadError.Error`
91. `net/http.tlsHandshakeTimeoutError.Error`
92. `\*github.com/pkg/errors.fundamental.Error`
93. `\*github.com/redis/go-redis/v9/internal/proto.MovedError.Error`
94. `\*golang.org/x/net/http2.goAwayFlowError.Error`
95. `\*github.com/containerd/errdefs.errPermissionDenied.Error`
96. `github.com/microsoft/go-mssqldb.RetryableError.Error`
97. `\*github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
98. `\*github.com/redis/go-redis/v9/internal/proto.ExecAbortError.Error`
99. `golang.org/x/text/encoding/internal.RepertoireError.Error`
100. `github.com/montanaflynn/stats.statsError.Error`
101. `\*gopkg.in/go-playground/validator.v9.fieldError.Error`
102. `\*github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
103. `\*net/http.http2GoAwayError.Error`
104. `github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
105. `github.com/docker/docker/client.objectNotFoundError.Error`
106. `\*net/http.nothingWrittenError.Error`
107. `\*github.com/montanaflynn/stats.statsError.Error`
108. `\*github.com/redis/go-redis/v9/push.ProcessorError.Error`
109. `\*go.mongodb.org/mongo-driver/x/mongo/driver/ocsp.Error.Error`
110. `golang.org/x/net/http2.noCachedConnError.Error`
111. `crypto/x509.ConstraintViolationError.Error`
112. `github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
113. `github.com/LerianStudio/lib-commons/v4/commons.Response.Error`
114. `net/http.http2StreamError.Error`
115. `github.com/LerianStudio/lib-license-go/v2/pkg.ResponseError.Error`
116. `\*os.SyscallError.Error`
117. `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc/internal.PartialSuccess.Error`
118. `crypto/tls.alert.Error`
119. `\*vendor/golang.org/x/net/idna.labelError.Error`
120. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteErrors.Error`
121. `\*net/netip.parseAddrError.Error`
122. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ElementTypeError.Error`
123. `vendor/golang.org/x/net/http2/hpack.InvalidIndexError.Error`
124. `\*github.com/containerd/errdefs.errNotModified.Error`
125. `\*net.temporaryError.Error`
126. `\*github.com/jackc/pgx/v5/pgconn.perDialConnectError.Error`
127. `\*go.mongodb.org/mongo-driver/mongo.BulkWriteError.Error`
128. `\*github.com/jackc/pgx/v5/pgconn.PgError.Error`
129. `github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
130. `go.mongodb.org/mongo-driver/bson/bsonrw.TransitionError.Error`
131. `\*go/types.ArgumentError.Error`
132. `github.com/containerd/errdefs.errNotImplemented.Error`
133. `\*gopkg.in/go-playground/validator.v9.ValidationErrors.Error`
134. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.DecodeError.Error`
135. `\*github.com/containerd/errdefs/pkg/internal/cause.ErrUnexpectedStatus.Error`
136. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ElementTypeError.Error`
137. `google.golang.org/grpc/internal/transport.ioError.Error`
138. `net/url.InvalidHostError.Error`
139. `golang.org/x/net/http2.StreamError.Error`
140. `\*github.com/jackc/pgx/v5/pgtype.nullAssignmentError.Error`
141. `\*github.com/redis/go-redis/v9/internal/proto.AuthError.Error`
142. `\*github.com/microsoft/go-mssqldb.RetryableError.Error`
143. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.ServerSelectionError.Error`
144. `internal/runtime/maps.unhashableTypeError.Error`
145. `\*google.golang.org/protobuf/internal/errors.SizeMismatchError.Error`
146. `go.mongodb.org/mongo-driver/x/mongo/driver.Error.Error`
147. `\*crypto/rc4.KeySizeError.Error`
148. `github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
149. `github.com/gofiber/fiber/v2/internal/schema.ConversionError.Error`
150. `\*github.com/valyala/fasthttp.ErrBrokenChunk.Error`
151. `google.golang.org/grpc/internal/transport.ConnectionError.Error`
152. `\*github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
153. `\*github.com/redis/go-redis/v9/internal/proto.OOMError.Error`
154. `\*golang.org/x/net/http2/hpack.DecodingError.Error`
155. `\*crypto/tls.ECHRejectionError.Error`
156. `crypto/aes.KeySizeError.Error`
157. `\*context.deadlineExceededError.Error`
158. `\*vendor/golang.org/x/net/http2/hpack.DecodingError.Error`
159. `github.com/xdg-go/stringprep.Error.Error`
160. `net/http.http2noCachedConnError.Error`
161. `\*vendor/golang.org/x/net/idna.runeError.Error`
162. `golang.org/x/net/http2/hpack.DecodingError.Error`
163. `\*github.com/jackc/pgx/v5/pgconn.contextAlreadyDoneError.Error`
164. `\*github.com/grpc-ecosystem/grpc-gateway/v2/runtime.HTTPStatusError.Error`
165. `\*debug/dwarf.DecodeError.Error`
166. `\*crypto/internal/fips140/hmac.errCloneUnsupported.Error`
167. `github.com/LerianStudio/lib-license-go/v2/pkg.HTTPError.Error`
168. `go.mongodb.org/mongo-driver/mongo.WriteException.Error`
169. `github.com/go-openapi/swag/jsonutils/adapters/stdlib/json.jsonError.Error`
170. `\*text/template.ExecError.Error`
171. `\*github.com/go-sql-driver/mysql.MySQLError.Error`
172. `github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
173. `\*encoding/json.InvalidUTF8Error.Error`
174. `\*golang.org/x/net/webdav/internal/xml.UnmarshalError.Error`
175. `\*github.com/go-playground/universal-translator.ErrExistingTranslator.Error`
176. `\*runtime.errorString.Error`
177. `\*time.parseDurationError.Error`
178. `encoding/hex.InvalidByteError.Error`
179. `\*crypto/tls.AlertError.Error`
180. `net/textproto.ProtocolError.Error`
181. `\*github.com/containerd/errdefs.errResourceExhausted.Error`
182. `\*crypto/des.KeySizeError.Error`
183. `github.com/moby/term.EscapeError.Error`
184. `\*github.com/LerianStudio/lib-commons/v4/commons/rabbitmq.sanitizedError.Error`
185. `compress/bzip2.StructuralError.Error`
186. `\*time.fileSizeError.Error`
187. `\*golang.org/x/net/http2.ConnectionError.Error`
188. `\*github.com/testcontainers/testcontainers-go.ParallelContainersError.Error`
189. `\*golang.org/x/net/http2.httpError.Error`
190. `\*github.com/gofiber/fiber/v2.Error.Error`
191. `go.mongodb.org/mongo-driver/mongo.EncryptionKeyVaultError.Error`
192. `github.com/go-openapi/jsonpointer.pointerError.Error`
193. `\*compress/flate.InternalError.Error`
194. `github.com/valyala/fasthttp.ErrNothingRead.Error`
195. `\*github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
196. `\*encoding/xml.UnmarshalError.Error`
197. `\*github.com/go-playground/validator/v10.InvalidValidationError.Error`
198. `github.com/containerd/errdefs.errConflict.Error`
199. `github.com/LerianStudio/lib-commons/v4/commons/net/http.ErrorResponse.Error`
200. `google.golang.org/protobuf/internal/impl.errInvalidUTF8.Error`
201. `github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
202. `\*internal/reflectlite.ValueError.Error`
203. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteCommandError.Error`
204. `github.com/go-openapi/swag/loading.loadingError.Error`
205. `\*github.com/gofiber/fiber/v2/internal/schema.MultiError.Error`
206. `\*google.golang.org/protobuf/internal/errors.wrapError.Error`
207. `\*runtime.TypeAssertionError.Error`
208. `\*crypto/x509.CertificateInvalidError.Error`
209. `\*github.com/LerianStudio/lib-license-go/v2/pkg.UnprocessableOperationError.Error`
210. `go.mongodb.org/mongo-driver/internal/aws/awserr.baseError.Error`
211. `golang.org/x/crypto/blowfish.KeySizeError.Error`
212. `github.com/valyala/fasthttp.EscapeError.Error`
213. `\*golang.org/x/crypto/ssh.cbcError.Error`
214. `\*github.com/go-playground/universal-translator.ErrCardinalTranslation.Error`
215. `\*github.com/valyala/fasthttp.EscapeError.Error`
216. `\*errors.joinError.Error`
217. `\*github.com/redis/go-redis/v9/internal/proto.ReadOnlyError.Error`
218. `\*github.com/valyala/fasthttp.ErrBodyStreamWritePanic.Error`
219. `\*crypto/tls.permanentError.Error`
220. `encoding/json.jsonError.Error`
221. `\*go.mongodb.org/mongo-driver/x/mongo/driver.InvalidOperationError.Error`
222. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.PoolError.Error`
223. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.PoolError.Error`
224. `golang.org/x/net/http2.headerFieldNameError.Error`
225. `\*github.com/go-logr/logr.notFoundError.Error`
226. `\*crypto/x509.ConstraintViolationError.Error`
227. `github.com/containerd/errdefs.errNotModified.Error`
228. `\*github.com/go-resty/resty/v2.noRetryErr.Error`
229. `\*github.com/redis/go-redis/v9/push.HandlerError.Error`
230. `\*golang.org/x/sync/singleflight.panicError.Error`
231. `\*github.com/containerd/errdefs.errUnavailable.Error`
232. `\*google.golang.org/grpc/internal/status.Error.Error`
233. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.WaitQueueTimeoutError.Error`
234. `\*github.com/containerd/errdefs.errAborted.Error`
235. `\*encoding/xml.TagPathError.Error`
236. `net/http.http2goAwayFlowError.Error`
237. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.poolClearedError.Error`
238. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoEncoder.Error`
239. `context.deadlineExceededError.Error`
240. `\*github.com/go-openapi/swag/jsonutils/adapters/stdlib/json.jsonError.Error`
241. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationUnknownFieldsError.Error`
242. `\*github.com/grpc-ecosystem/grpc-gateway/v2/internal/httprule.InvalidTemplateError.Error`
243. `crypto/x509.SystemRootsError.Error`
244. `\*github.com/containerd/errdefs.errDataLoss.Error`
245. `\*net/http.timeoutError.Error`
246. `\*github.com/jackc/pgx/v5.ScanArgError.Error`
247. `\*github.com/cenkalti/backoff/v4.PermanentError.Error`
248. `\*go.mongodb.org/mongo-driver/bson/bsonrw.TransitionError.Error`
249. `github.com/containerd/errdefs.errOutOfRange.Error`
250. `\*github.com/jackc/pgx/v5/pgproto3.ExceededMaxBodyLenErr.Error`
251. `\*go.uber.org/zap.errArrayElem.Error`
252. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.ConnectionError.Error`
253. `\*golang.org/x/crypto/ssh.ServerAuthError.Error`
254. `go.mongodb.org/mongo-driver/mongo.ErrMapForOrderedArgument.Error`
255. `golang.org/x/net/http2.headerFieldValueError.Error`
256. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.ConnectionError.Error`
257. `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.errMetric.Error`
258. `\*encoding/json.MarshalerError.Error`
259. `\*crypto/x509.UnhandledCriticalExtension.Error`
260. `\*github.com/jackc/pgx/v5/pgproto3.invalidMessageFormatErr.Error`
261. `net/url.EscapeError.Error`
262. `\*github.com/jackc/pgx/v5/pgconn.ConnectError.Error`
263. `\*github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
264. `github.com/containerd/errdefs/pkg/internal/cause.ErrUnexpectedStatus.Error`
265. `net.InvalidAddrError.Error`
266. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InsufficientBytesError.Error`
267. `\*archive/tar.headerError.Error`
268. `github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
269. `golang.org/x/net/http2.goAwayFlowError.Error`
270. `\*github.com/redis/go-redis/v9/internal/proto.LoadingError.Error`
271. `\*github.com/valyala/fasthttp.ErrSmallBuffer.Error`
272. `\*github.com/LerianStudio/lib-license-go/v2/pkg.UnauthorizedError.Error`
273. `go.mongodb.org/mongo-driver/x/mongo/driver.InvalidOperationError.Error`
274. `vendor/golang.org/x/net/idna.runeError.Error`
275. `\*github.com/Shopify/toxiproxy/v2/client.ApiError.Error`
276. `github.com/gofiber/fiber/v2/internal/schema.UnknownKeyError.Error`
277. `os/user.UnknownUserError.Error`
278. `\*os/exec.Error.Error`
279. `\*github.com/microsoft/go-mssqldb/aecmk.Error.Error`
280. `\*github.com/yuin/gopher-lua/parse.Error.Error`
281. `syscall.Errno.Error`
282. `net.canceledError.Error`
283. `\*net/http.http2noCachedConnError.Error`
284. `\*net/http.tlsHandshakeTimeoutError.Error`
285. `\*crypto/x509.HostnameError.Error`
286. `\*github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
287. `net/http.http2connError.Error`
288. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ValidationError.Error`
289. `\*golang.org/x/net/webdav/internal/xml.UnsupportedTypeError.Error`
290. `os/user.UnknownGroupIdError.Error`
291. `\*github.com/lib/pq.Error.Error`
292. `\*github.com/jackc/pgx/v5/pgconn.connLockError.Error`
293. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.poolClearedError.Error`
294. `\*go.mongodb.org/mongo-driver/mongo.WriteConcernError.Error`
295. `\*go/build/constraint.SyntaxError.Error`
296. `\*go.mongodb.org/mongo-driver/mongo.MongocryptdError.Error`
297. `\*github.com/jackc/pgx/v5.proxyError.Error`
298. `github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
299. `github.com/valyala/fasthttp.ErrBodyStreamWritePanic.Error`
300. `\*github.com/go-playground/universal-translator.ErrOrdinalTranslation.Error`
301. `\*github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
302. `net/http.http2duplicatePseudoHeaderError.Error`
303. `github.com/ebitengine/purego.Dlerror.Error`
304. `\*github.com/golang-jwt/jwt/v5.joinedError.Error`
305. `\*crypto/aes.KeySizeError.Error`
306. `\*net/http.http2StreamError.Error`
307. `golang.org/x/net/idna.runeError.Error`
308. `\*github.com/LerianStudio/lib-commons/v4/commons/net/http.ErrorResponse.Error`
309. `\*crypto/x509.SystemRootsError.Error`
310. `\*github.com/shirou/gopsutil/v4/internal/common.Warnings.Error`
311. `github.com/LerianStudio/lib-license-go/v2/pkg.UnauthorizedError.Error`
312. `\*errors.errorString.Error`
313. `github.com/LerianStudio/lib-commons/commons.Response.Error`
314. `\*golang.org/x/net/idna.runeError.Error`
315. `\*golang.org/x/net/idna.labelError.Error`
316. `\*golang.org/x/crypto/ssh.ExitError.Error`
317. `\*golang.org/x/net/http2.noCachedConnError.Error`
318. `\*reflect.ValueError.Error`
319. `net/http.http2headerFieldNameError.Error`
320. `internal/poll.errNetClosing.Error`
321. `github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
322. `\*github.com/ebitengine/purego.Dlerror.Error`
323. `\*compress/flate.WriteError.Error`
324. `github.com/microsoft/go-mssqldb.Error.Error`
325. `go/types.Error.Error`
326. `\*github.com/go-playground/universal-translator.ErrBadPluralDefinition.Error`
327. `github.com/docker/docker/client.errConnectionFailed.Error`
328. `\*github.com/LerianStudio/lib-commons/commons.Response.Error`
329. `\*net/netip.parsePrefixError.Error`
330. `go.mongodb.org/mongo-driver/mongo.BulkWriteError.Error`
331. `\*github.com/go-openapi/jsonpointer.pointerError.Error`
332. `\*net/http.http2duplicatePseudoHeaderError.Error`
333. `\*golang.org/x/net/http2.headerFieldValueError.Error`
334. `crypto/rc4.KeySizeError.Error`
335. `\*github.com/containerd/errdefs.errConflict.Error`
336. `\*github.com/gofiber/fiber/v2/internal/schema.UnknownKeyError.Error`
337. `\*github.com/containerd/errdefs.errFailedPrecondition.Error`
338. `\*golang.org/x/net/webdav/internal/xml.TagPathError.Error`
339. `github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
340. `\*golang.org/x/text/encoding/internal.RepertoireError.Error`
341. `\*go.mongodb.org/mongo-driver/mongo.WriteError.Error`
342. `github.com/rabbitmq/amqp091-go.Error.Error`
343. `compress/flate.CorruptInputError.Error`
344. `\*github.com/LerianStudio/lib-commons/v4/commons/mongo.SanitizedError.Error`
345. `\*go.mongodb.org/mongo-driver/x/mongo/driver.Error.Error`
346. `github.com/containerd/errdefs.errPermissionDenied.Error`
347. `crypto/x509.InsecureAlgorithmError.Error`
348. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.TransitionError.Error`
349. `\*net/http.http2ConnectionError.Error`
350. `\*github.com/LerianStudio/lib-license-go/v2/pkg.EntityConflictError.Error`
351. `\*github.com/containerd/errdefs.errInvalidArgument.Error`
352. `\*golang.org/x/crypto/ssh.PartialSuccessError.Error`
353. `\*github.com/go-openapi/swag/jsonutils/adapters.registryError.Error`
354. `\*github.com/jackc/pgx/v5/pgconn.pgconnError.Error`
355. `\*github.com/redis/go-redis/v9/internal/proto.ClusterDownError.Error`
356. `github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
357. `github.com/containerd/errdefs.errNotFound.Error`
358. `\*internal/chacha8rand.errUnmarshalChaCha8.Error`
359. `github.com/LerianStudio/lib-license-go/v2/pkg.EntityNotFoundError.Error`
360. `\*google.golang.org/grpc/internal/transport.NewStreamError.Error`
361. `\*net.timeoutError.Error`
362. `go.opentelemetry.io/otel/sdk/trace.errUnsupportedSampler.Error`
363. `\*internal/fuzz.MalformedCorpusError.Error`
364. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationUnknownFieldsError.Error`
365. `\*github.com/jackc/pgx/v5/pgconn.NotPreferredError.Error`
366. `net/netip.parsePrefixError.Error`
367. `\*google.golang.org/grpc.dropError.Error`
368. `github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
369. `\*github.com/LerianStudio/lib-license-go/v2/pkg.HTTPError.Error`
370. `crypto/des.KeySizeError.Error`
371. `net/http.transportReadFromServerError.Error`
372. `github.com/containerd/errdefs.errAborted.Error`
373. `\*github.com/gofiber/fiber/v2/internal/schema.EmptyFieldError.Error`
374. `github.com/containerd/errdefs.errDataLoss.Error`
375. `go.opentelemetry.io/otel/sdk/trace.samplerArgParseError.Error`
376. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.multiErr.Error`
377. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteErrors.Error`
378. `archive/tar.headerError.Error`
379. `github.com/go-logr/logr.notFoundError.Error`
380. `\*golang.org/x/net/http2.GoAwayError.Error`
381. `\*net/url.InvalidHostError.Error`
382. `\*os.LinkError.Error`
383. `\*go.mongodb.org/mongo-driver/x/mongo/driver.QueryFailureError.Error`
384. `go.mongodb.org/mongo-driver/mongo.WriteConcernError.Error`
385. `internal/strconv.Error.Error`
386. `\*net.addrinfoErrno.Error`
387. `\*net.OpError.Error`
388. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.WaitQueueTimeoutError.Error`
389. `text/template.ExecError.Error`
390. `\*go.uber.org/multierr.multiError.Error`
391. `\*os.errSymlink.Error`
392. `\*github.com/valyala/fasthttp.ErrNothingRead.Error`
393. `go.mongodb.org/mongo-driver/bson/bsoncodec.ValueEncoderError.Error`
394. `\*math/big.ErrNaN.Error`
395. `crypto/internal/fips140/hmac.errCloneUnsupported.Error`
396. `\*encoding/json.UnmarshalFieldError.Error`
397. `\*go.mongodb.org/mongo-driver/bson/bsonrw.errMaxDocumentSizeExceeded.Error`
398. `net/http.http2pseudoHeaderError.Error`
399. `\*golang.org/x/crypto/ocsp.ResponseError.Error`
400. `github.com/gofiber/fiber/v2/internal/schema.EmptyFieldError.Error`
401. `\*github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
402. `\*github.com/go-playground/universal-translator.ErrBadParamSyntax.Error`
403. `\*github.com/docker/docker/api/types/registry.errInvalidParameter.Error`
404. `\*golang.org/x/net/http2.duplicatePseudoHeaderError.Error`
405. `\*net.DNSConfigError.Error`
406. `\*os/user.UnknownUserError.Error`
407. `\*google.golang.org/grpc/internal/transport.ioError.Error`
408. `github.com/LerianStudio/lib-license-go/v2/pkg.InternalServerError.Error`
409. `\*golang.org/x/crypto/blowfish.KeySizeError.Error`
410. `\*runtime.errorAddressString.Error`
411. `go.uber.org/zap.errArrayElem.Error`
412. `runtime.boundsError.Error`
413. `net/http.http2GoAwayError.Error`
414. `github.com/pkg/errors.withStack.Error`
415. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InsufficientBytesError.Error`
416. `\*go.opentelemetry.io/otel/trace.errorConst.Error`
417. `\*golang.org/x/net/http2.pseudoHeaderError.Error`
418. `\*github.com/LerianStudio/lib-license-go/v2/pkg.EntityNotFoundError.Error`
419. `\*github.com/LerianStudio/lib-commons/v4/commons.Response.Error`
420. `crypto/internal/fips140/aes.KeySizeError.Error`
421. `\*go.mongodb.org/mongo-driver/mongo.CommandError.Error`
422. `github.com/containerd/errdefs.errAlreadyExists.Error`
423. `golang.org/x/net/http2.GoAwayError.Error`
424. `\*github.com/containerd/errdefs.errOutOfRange.Error`
425. `\*golang.org/x/net/webdav/internal/xml.SyntaxError.Error`
426. `\*go/build.NoGoError.Error`
427. `\*github.com/tryvium-travels/memongo/mongobin.UnsupportedSystemError.Error`
428. `\*os/signal.signalError.Error`
429. `\*os/user.UnknownGroupError.Error`
430. `\*os/exec.wrappedError.Error`
431. `go.mongodb.org/mongo-driver/bson/bsoncodec.TransitionError.Error`
432. `\*github.com/docker/docker/api/types/container.errInvalidParameter.Error`
433. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ValueEncoderError.Error`
434. `\*github.com/containerd/errdefs.errNotImplemented.Error`
435. `golang.org/x/net/idna.labelError.Error`
436. `\*go/types.Error.Error`
437. `github.com/microsoft/go-mssqldb.ServerError.Error`
438. `encoding/xml.UnmarshalError.Error`
439. `\*github.com/go-playground/validator/v10.fieldError.Error`
440. `\*github.com/microsoft/go-mssqldb.ServerError.Error`
441. `\*html/template.Error.Error`
442. `\*go.mongodb.org/mongo-driver/mongo/options.MarshalError.Error`
443. `\*compress/bzip2.StructuralError.Error`
444. `\*golang.org/x/crypto/ocsp.ParseError.Error`
445. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.MalformedElementError.Error`
446. `\*go.opentelemetry.io/otel/sdk/trace.errUnsupportedSampler.Error`
447. `\*go/scanner.Error.Error`
448. `github.com/LerianStudio/lib-license-go/v2/pkg.EntityConflictError.Error`
449. `\*golang.org/x/net/http2/hpack.InvalidIndexError.Error`
450. `\*github.com/jackc/pgx/v5/pgconn.errTimeout.Error`
451. `\*github.com/valyala/fasthttp/fasthttputil.timeoutError.Error`
452. `\*net/http.transportReadFromServerError.Error`
453. `\*github.com/cenkalti/backoff/v5.PermanentError.Error`
454. `github.com/gofiber/fiber/v2/internal/schema.MultiError.Error`
455. `\*golang.org/x/crypto/ssh.OpenChannelError.Error`
456. `go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt.Error.Error`
457. `\*time.ParseError.Error`
458. `github.com/valyala/fasthttp.ErrSmallBuffer.Error`
459. `\*github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
460. `net/mail.charsetError.Error`
461. `\*github.com/redis/go-redis/v9/internal/proto.NoReplicasError.Error`
462. `\*github.com/redis/go-redis/v9/internal/proto.PermissionError.Error`
463. `\*encoding/asn1.StructuralError.Error`
464. `\*go.mongodb.org/mongo-driver/internal/aws/awserr.errorList.Error`
465. `github.com/containerd/errdefs.errUnauthorized.Error`
466. `\*github.com/microsoft/go-mssqldb.StreamError.Error`
467. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InvalidDepthTraversalError.Error`
468. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteError.Error`
469. `\*internal/strconv.Error.Error`
470. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.ServerSelectionError.Error`
471. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ForbiddenError.Error`
472. `\*golang.org/x/crypto/ssh.BannerError.Error`
473. `\*net/http.http2connError.Error`
474. `go.mongodb.org/mongo-driver/mongo.MarshalError.Error`
475. `\*github.com/redis/go-redis/v9/internal/pool.BadConnError.Error`
476. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ValueDecoderError.Error`
477. `os/user.UnknownUserIdError.Error`
478. `\*net.UnknownNetworkError.Error`
479. `\*golang.org/x/crypto/ssh.disconnectMsg.Error`
480. `\*net/http.http2httpError.Error`
481. `\*github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
482. `github.com/docker/docker/api/types/registry.errInvalidParameter.Error`
483. `github.com/grpc-ecosystem/grpc-gateway/v2/runtime.MalformedSequenceError.Error`
484. `\*internal/bisect.parseError.Error`
485. `\*github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
486. `github.com/redis/go-redis/v9/internal/proto.RedisError.Error`
487. `go.mongodb.org/mongo-driver/mongo.WriteError.Error`
488. `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal.PartialSuccess.Error`
489. `\*github.com/go-resty/resty/v2.ResponseError.Error`
490. `\*github.com/valyala/fasthttp.InvalidHostError.Error`
491. `\*encoding/asn1.SyntaxError.Error`
492. `\*regexp/syntax.Error.Error`
493. `\*github.com/moby/term.EscapeError.Error`
494. `\*go/build.MultiplePackageError.Error`
495. `github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
496. `\*github.com/containerd/errdefs.errUnknown.Error`
497. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoDecoder.Error`
498. `\*go.uber.org/zap.errSinkNotFound.Error`
499. `\*go.mongodb.org/mongo-driver/mongo.MongocryptError.Error`
500. `github.com/valyala/fasthttp.InvalidHostError.Error`
501. `go.mongodb.org/mongo-driver/bson/bsonrw.errMaxDocumentSizeExceeded.Error`
502. `\*github.com/docker/docker/api/types/filters.invalidFilter.Error`
503. `\*github.com/jackc/pgx/v5/pgconn.ParseConfigError.Error`
504. `golang.org/x/crypto/ssh.cbcError.Error`
505. `google.golang.org/grpc.dropError.Error`
506. `\*github.com/valyala/fasthttp.ErrDialWithUpstream.Error`
507. `\*github.com/andybalholm/brotli.decodeError.Error`
508. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationKnownFieldsError.Error`
509. `\*github.com/LerianStudio/lib-commons/v2/commons.Response.Error`
510. `\*github.com/go-playground/universal-translator.ErrConflictingTranslation.Error`
511. `\*github.com/grpc-ecosystem/grpc-gateway/v2/runtime.MalformedSequenceError.Error`
512. `\*net/http.ProtocolError.Error`
513. `\*encoding/xml.UnsupportedTypeError.Error`
514. `\*github.com/containerd/errdefs.errUnauthorized.Error`
515. `\*github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
516. `os/signal.signalError.Error`
517. `github.com/testcontainers/testcontainers-go.ParallelContainersError.Error`
518. `go.mongodb.org/mongo-driver/internal/aws/awserr.errorList.Error`
519. `\*github.com/LerianStudio/lib-commons/v4/commons/runtime.panicError.Error`
520. `encoding/asn1.SyntaxError.Error`
521. `github.com/containerd/errdefs.errFailedPrecondition.Error`
522. `github.com/go-openapi/swag/jsonutils/adapters.registryError.Error`
523. `github.com/containerd/errdefs.errUnknown.Error`
524. `\*github.com/containerd/errdefs.errInternal.Error`
525. `\*google.golang.org/protobuf/internal/impl.errInvalidUTF8.Error`
526. `\*encoding/json.UnsupportedValueError.Error`
527. `\*github.com/microsoft/go-mssqldb.Error.Error`
528. `\*github.com/gofiber/fiber/v2/internal/schema.ConversionError.Error`
529. `github.com/grpc-ecosystem/grpc-gateway/v2/internal/httprule.InvalidTemplateError.Error`
530. `\*encoding/json.jsonError.Error`
531. `\*github.com/yuin/gopher-lua.ApiError.Error`
532. `\*github.com/go-resty/resty/v2.restyError.Error`
533. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal.PartialSuccess.Error`
534. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteCommandError.Error`
535. `\*gopkg.in/yaml.v3.TypeError.Error`
536. `\*encoding/hex.InvalidByteError.Error`
537. `\*github.com/go-playground/universal-translator.ErrMissingLocale.Error`
538. `\*go.mongodb.org/mongo-driver/internal/aws/awserr.baseError.Error`
539. `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal.PartialSuccess.Error`
540. `\*net.canceledError.Error`
541. `golang.org/x/net/webdav/internal/xml.UnmarshalError.Error`
542. `net/http.http2ConnectionError.Error`
543. `\*github.com/LerianStudio/lib-license-go/v2/pkg.FailedPreconditionError.Error`
544. `github.com/microsoft/go-mssqldb.StreamError.Error`
545. `github.com/LerianStudio/lib-license-go/v2/pkg.UnprocessableOperationError.Error`
546. `\*go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc/internal.PartialSuccess.Error`
547. `\*go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal.PartialSuccess.Error`
548. `\*go.mongodb.org/mongo-driver/mongo.BulkWriteException.Error`
549. `golang.org/x/crypto/ocsp.ParseError.Error`
550. `go.opentelemetry.io/otel/trace.errorConst.Error`
551. `\*github.com/yuin/gopher-lua.CompileError.Error`
552. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteConcernError.Error`
553. `math/big.ErrNaN.Error`
554. `\*os/user.UnknownGroupIdError.Error`
555. `\*encoding/csv.ParseError.Error`
556. `\*github.com/jackc/pgx/v5/pgconn.PrepareError.Error`
557. `github.com/golang-jwt/jwt/v5.joinedError.Error`
558. `gopkg.in/go-playground/validator.v9.ValidationErrors.Error`
559. `crypto/x509/internal/macos.OSStatus.Error`
560. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.decodeBinaryError.Error`
561. `\*github.com/docker/docker/client.objectNotFoundError.Error`
562. `go.mongodb.org/mongo-driver/bson/bsoncodec.decodeBinaryError.Error`
563. `github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
564. `net/http.http2headerFieldValueError.Error`
565. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteError.Error`
566. `\*go.yaml.in/yaml/v3.TypeError.Error`
567. `\*go.mongodb.org/mongo-driver/mongo.MarshalError.Error`
568. `\*golang.org/x/mod/module.InvalidVersionError.Error`
569. `\*encoding/json.InvalidUnmarshalError.Error`
570. `\*github.com/tryvium-travels/memongo/mongobin.UnsupportedMongoVersionError.Error`
571. `github.com/LerianStudio/lib-license-go/v2/pkg.FailedPreconditionError.Error`
572. `runtime.errorString.Error`
573. `\*net.DNSError.Error`
574. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ResponseError.Error`
575. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoDecoder.Error`
576. `\*github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
577. `github.com/klauspost/compress/flate.InternalError.Error`
578. `\*github.com/klauspost/compress/flate.InternalError.Error`
579. `\*github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
580. `\*runtime.synctestDeadlockError.Error`
581. `\*github.com/redis/go-redis/v9/internal/proto.MaxClientsError.Error`
582. `\*github.com/valyala/fasthttp.timeoutError.Error`
583. `\*io/fs.PathError.Error`
584. `\*github.com/docker/docker/client.emptyIDError.Error`
585. `\*internal/runtime/maps.unhashableTypeError.Error`
586. `github.com/docker/docker/api/types/filters.invalidFilter.Error`
587. `go/scanner.ErrorList.Error`
588. `\*net/http.http2goAwayFlowError.Error`
589. `net.UnknownNetworkError.Error`
590. `\*net/url.EscapeError.Error`
591. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoTypeMapEntry.Error`
592. `\*github.com/go-playground/universal-translator.ErrMissingPluralTranslation.Error`
593. `\*github.com/containerd/errdefs.customMessage.Error`
594. `net/http.statusError.Error`
595. `\*github.com/rabbitmq/amqp091-go.Error.Error`
596. `\*github.com/docker/docker/api/types.ErrorResponse.Error`
597. `github.com/containerd/errdefs.errInternal.Error`
598. `\*vendor/golang.org/x/net/http2/hpack.InvalidIndexError.Error`
599. `\*github.com/lib/pq.safeRetryError.Error`
600. `runtime.plainError.Error`
601. `\*github.com/alicebob/miniredis/v2/gopher-json.invalidTypeError.Error`
602. `\*go.mongodb.org/mongo-driver/mongo.EncryptionKeyVaultError.Error`
603. `\*net.AddrError.Error`
604. `github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
605. `\*net/url.Error.Error`
606. `github.com/containerd/errdefs.errUnavailable.Error`
607. `\*net/mail.charsetError.Error`
608. `\*crypto/x509/internal/macos.OSStatus.Error`
609. `\*github.com/sijms/go-ora/v2/network.OracleError.Error`
610. `\*internal/fuzz.crashError.Error`
611. `\*net/textproto.ProtocolError.Error`
612. `\*github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
613. `github.com/andybalholm/brotli.decodeError.Error`
614. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ValidationError.Error`
615. `\*github.com/containerd/errdefs.errNotFound.Error`
616. `\*github.com/redis/go-redis/v9/internal/proto.AskError.Error`
617. `go.mongodb.org/mongo-driver/mongo.CommandError.Error`
618. `os/exec.wrappedError.Error`
619. `\*net.notFoundError.Error`
620. `crypto/x509.UnknownAuthorityError.Error`
621. `crypto/x509.HostnameError.Error`
622. `github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
623. `\*github.com/go-playground/universal-translator.ErrRangeTranslation.Error`
624. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationKnownFieldsError.Error`
625. `\*runtime.plainError.Error`
626. `github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
627. `golang.org/x/net/http2.connError.Error`
628. `\*fmt.wrapErrors.Error`
629. `\*github.com/LerianStudio/lib-commons/v4/commons/assert.AssertionError.Error`
630. `go.mongodb.org/mongo-driver/mongo.MongocryptdError.Error`
631. `\*google.golang.org/grpc/internal/transport.ConnectionError.Error`
632. `\*net/http.requestBodyReadError.Error`
633. `\*compress/flate.ReadError.Error`
634. `\*syscall.Errno.Error`
635. `\*github.com/valyala/fasthttp/reuseport.ErrNoReusePort.Error`
636. `\*golang.org/x/net/http2.connError.Error`
637. `runtime.synctestDeadlockError.Error`
638. `\*compress/flate.CorruptInputError.Error`
639. `\*github.com/redis/go-redis/v9/internal/proto.TryAgainError.Error`
640. `github.com/redis/go-redis/v9/internal/pool.BadConnError.Error`
641. `github.com/jackc/pgx/v5.ScanArgError.Error`
642. `\*net/http.MaxBytesError.Error`
643. `\*golang.org/x/crypto/ssh.AlgorithmNegotiationError.Error`
644. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoTypeMapEntry.Error`
645. `github.com/google/uuid.invalidLengthError.Error`
646. `\*github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
647. `\*golang.org/x/net/http2.StreamError.Error`
648. `\*github.com/testcontainers/testcontainers-go/wait.PermanentError.Error`
649. `\*github.com/cenkalti/backoff/v5.RetryAfterError.Error`
650. `github.com/alicebob/miniredis/v2/gopher-json.invalidTypeError.Error`
651. `github.com/valyala/fasthttp.ErrBrokenChunk.Error`
652. `\*golang.org/x/crypto/ssh.PassphraseMissingError.Error`
653. `\*github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
654. `google.golang.org/grpc/internal/transport.NewStreamError.Error`
655. `\*github.com/containerd/errdefs.errAlreadyExists.Error`
656. `net.addrinfoErrno.Error`
657. `go.mongodb.org/mongo-driver/mongo.MongocryptError.Error`
658. `encoding/base64.CorruptInputError.Error`
659. `crypto/tls.AlertError.Error`
660. `\*encoding/json.SyntaxError.Error`
661. `\*net/textproto.Error.Error`
662. `\*github.com/jackc/pgx/v5/pgproto3.writeError.Error`
663. `\*github.com/go-openapi/swag/yamlutils.yamlError.Error`
664. `\*runtime.PanicNilError.Error`
665. `github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
666. `github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
667. `\*github.com/go-playground/universal-translator.ErrMissingBracket.Error`
668. `encoding/asn1.StructuralError.Error`
669. `compress/flate.InternalError.Error`
670. `\*crypto/tls.echConfigErr.Error`
671. `\*github.com/google/uuid.invalidLengthError.Error`
672. `\*github.com/LerianStudio/lib-license-go/v2/pkg.InternalServerError.Error`
673. `go.mongodb.org/mongo-driver/mongo.BulkWriteException.Error`
674. `github.com/go-playground/validator/v10.ValidationErrors.Error`
675. `\*go.mongodb.org/mongo-driver/mongo.ErrMapForOrderedArgument.Error`
676. `\*github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
677. `github.com/containerd/errdefs.errResourceExhausted.Error`
678. `go.mongodb.org/mongo-driver/x/mongo/driver.ResponseError.Error`
679. `\*github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
680. `\*encoding/json.UnsupportedTypeError.Error`
681. `\*encoding/xml.SyntaxError.Error`
682. `\*github.com/jackc/pgx/v5/pgproto3.invalidMessageLenErr.Error`
683. `\*crypto/internal/fips140/aes.KeySizeError.Error`
684. `debug/dwarf.DecodeError.Error`
685. `\*go.mongodb.org/mongo-driver/mongo.WriteErrors.Error`
686. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.errMetric.Error`
687. `os/user.UnknownGroupError.Error`
688. `\*os/exec.ExitError.Error`
689. `\*net/http.statusError.Error`
690. `\*golang.org/x/text/internal/language.ValueError.Error`
691. `golang.org/x/text/internal/language.ValueError.Error`
692. `crypto/x509.CertificateInvalidError.Error`
693. `crypto/tls.RecordHeaderError.Error`
694. `\*internal/poll.errNetClosing.Error`
695. `\*debug/macho.FormatError.Error`
696. `github.com/LerianStudio/lib-license-go/v2/pkg.ForbiddenError.Error`
697. `\*github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
698. `\*github.com/go-openapi/swag/loading.loadingError.Error`
699. `\*github.com/docker/docker/pkg/jsonmessage.JSONError.Error`
700. `\*github.com/pkg/errors.withStack.Error`
701. `\*golang.org/x/net/http2.headerFieldNameError.Error`
702. `Contains`

#### `TestProcessPluginCRMCollection\_WithOrganizationID`
**File:** `components/worker/internal/services/extract\_crm\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `\*testing.common.Cleanup`
4. `newTestMocks`
5. `newTestUseCase`
6. `testContext`
7. `testLogger`
8. `MustParse`
9. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.processPluginCRMCollection`
10. `\*testing.common.Fatalf`
11. `\*testing.common.Fatalf`

#### `TestExtractJobIDFromMultipleSources\_EdgeCases`
**File:** `components/worker/internal/services/extract\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`

#### `TestExtractJobIDFromMultipleSources\_FromHeaders`
**File:** `components/worker/internal/services/extract\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `newTestMocks`
4. `newTestUseCase`
5. `testLogger`
6. `newTestJobID`
7. `newTestOrgID`
8. `github.com/google/uuid.UUID.String`
9. `github.com/google/uuid.UUID.String`
10. `Background`
11. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.extractJobIDFromMultipleSources`
12. `\*testing.common.Fatalf`
13. `\*testing.common.Fatalf`

#### `TestExtractJobIDFromPartialJSON`
**File:** `components/worker/internal/services/extract\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`

#### `TestExtractJobIDFromMultipleSources\_FromPartialJSON`
**File:** `components/worker/internal/services/extract\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `newTestMocks`
4. `newTestUseCase`
5. `testLogger`
6. `newTestJobID`
7. `newTestOrgID`
8. `github.com/google/uuid.UUID.String`
9. `github.com/google/uuid.UUID.String`
10. `Background`
11. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.extractJobIDFromMultipleSources`
12. `\*testing.common.Fatalf`
13. `\*testing.common.Fatalf`

#### `TestExtractJobIDFromMultipleSources\_NoIDs`
**File:** `components/worker/internal/services/extract\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `newTestMocks`
4. `newTestUseCase`
5. `testLogger`
6. `Background`
7. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.extractJobIDFromMultipleSources`
8. `\*testing.common.Fatalf`
9. `\*testing.common.Fatalf`

#### `TestExtractJobIDFromPartialJSON\_ValidJobIDInvalidOrgID`
**File:** `components/worker/internal/services/extract\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `newTestMocks`
4. `newTestUseCase`
5. `testLogger`
6. `Background`
7. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.extractJobIDFromPartialJSON`
8. `\*testing.common.Error`
9. `\*testing.common.Errorf`

#### `TestExtractJobIDFromPartialJSON\_RegexFallback`
**File:** `components/worker/internal/services/extract\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `newTestMocks`
4. `newTestUseCase`
5. `testLogger`
6. `Background`
7. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.extractJobIDFromPartialJSON`
8. `\*testing.common.Error`
9. `\*testing.common.Error`

#### `TestExtractJobIDFromMultipleSources\_HeaderPrecedence`
**File:** `components/worker/internal/services/extract\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `newTestMocks`
4. `newTestUseCase`
5. `testLogger`
6. `New`
7. `New`
8. `github.com/google/uuid.UUID.String`
9. `github.com/google/uuid.UUID.String`
10. `Background`
11. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.extractJobIDFromMultipleSources`
12. `\*testing.common.Errorf`

#### `TestParseMessage\_JSONNull`
**File:** `components/worker/internal/services/extract\_data\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `newTestMocks`
4. `newTestUseCase`
5. `testContext`
6. `testLogger`
7. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.parseMessage`
8. `\*testing.common.Fatal`
9. `\*testing.common.Fatalf`

#### `testLogger`
**File:** `components/worker/internal/services/test\_helpers\_test.go`
**Risk Level:** HIGH (80 direct callers)

**Direct Callers (signature change affects these):**

1. `TestTransformPluginCRMAdvancedFilters\_AllConditionTypes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1027`
2. `TestPublishJobNotification\_WithCompletedAt` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:451`
3. `TestDecryptRecord\_EmptyRecord` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:493`
4. `TestDecryptPluginCRMData\_WithNestedField` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:221`
5. `TestSaveExternalDataToSeaweedFS\_MarshalError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1502`
6. `TestDecryptFieldValue\_EdgeCases` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:406`
7. `TestShouldSkipProcessing$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:369`
8. `TestPublishJobNotification\_EmptyExchange` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:312`
9. `TestQueryDatabase\_ConnectionFoundButDifferentConfigName` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1452`
10. `TestDecryptPluginCRMData\_NoDecryptionNeeded` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:158`
11. `TestParseMessage\_WithNilError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:956`
12. `TestCheckReportStatus\_JobDataNil` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1256`
13. `TestParseMessage\_InvalidJSON` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:71`
14. `TestPublishJobNotification\_WithErrorMetadata` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:69`
15. `TestDecryptBankingDetailsFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:757`
16. `TestPublishJobNotification\_MetadataPreservation` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:336`
17. `TestTransformPluginCRMAdvancedFilters\_WithEncryptedFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1642`
18. `TestPublishJobNotification\_UnknownSource` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:183`
19. `TestProcessPluginCRMCollection\_WithValidOrganization` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1762`
20. `TestExtractJobIDFromMultipleSources\_EdgeCases$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:667`
21. `TestExtractJobIDFromMultipleSources\_FromPartialJSON` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:175`
22. `TestParseMessage\_InvalidJSONWithJobIDInHeaders` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:115`
23. `TestPublishJobNotification\_WithResultData` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:247`
24. `TestDecryptNaturalPersonFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:824`
25. `TestDecryptLegalPersonFields\_WithNilLegalPerson` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1462`
26. `TestHashFilterValues\_EdgeCases` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:342`
27. `TestQueryPluginCRM\_NilCollections` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1272`
28. `TestEncryptDataForSeaweedFS\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1878`
29. `TestTransformPluginCRMAdvancedFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:856`
30. `testContext` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/test\_helpers\_test.go:22`
31. `TestPublishJobNotification\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:23`
32. `TestQueryPluginCRMCollectionWithFilters\_NoFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1154`
33. `TestQueryPluginCRM\_EmptyCollections` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1241`
34. `TestSaveExternalDataToSeaweedFS\_SeaweedFSPutError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1573`
35. `TestParseMessage\_EmptyBody` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1307`
36. `TestDecryptRecord\_WithAllFieldTypes` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1346`
37. `TestDecryptContactFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:723`
38. `TestExtractJobIDFromMultipleSources\_FromHeaders` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:147`
39. `TestQueryPluginCRM\_WithFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1686`
40. `TestSaveExternalDataToSeaweedFS\_MissingEnvVars` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1537`
41. `TestCheckReportStatus$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:755`
42. `TestPublishJobNotification\_PublishError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:150`
43. `TestDecryptPluginCRMData\_MissingEncryptSecretKey` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1211`
44. `TestParseMessage\_JSONNull` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:94`
45. `TestQueryPluginCRM\_WithOrganizationOnly` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1308`
46. `TestDecryptPluginCRMData\_WithValidCrypto` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1610`
47. `TestDecryptNestedFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:615`
48. `TestProcessPluginCRMCollection\_WithOrganizationID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1100`
49. `TestHashFilterValues\_ConsistentHashing` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1583`
50. `TestPublishJobNotification\_RoutingKeyGeneration$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:422`
51. `TestExtractJobIDFromMultipleSources\_HeaderPrecedence` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1328`
52. `TestDecryptBankingDetailsFields\_WithNilBankingDetails` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1432`
53. `TestDecryptLegalPersonFields\_WithEmptyRepresentative` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1522`
54. `TestExtractJobIDFromMultipleSources\_NoIDs` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:201`
55. `TestDecryptPluginCRMData\_EmptyResult` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:245`
56. `TestDecryptPluginCRMData\_WithEncryptedFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:197`
57. `TestTransformPluginCRMAdvancedFilters\_FieldMappings` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:927`
58. `TestHandleErrorWithUpdate$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:925`
59. `TestPublishJobNotification\_PublisherNotConfigured` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:126`
60. `TestSaveExternalDataToSeaweedFS\_DocumentSignerError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_additional\_test.go:233`
61. `TestEncryptDataForSeaweedFS` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1083`
62. `TestQueryDatabase\_WithFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1995`
63. `TestDecryptNaturalPersonFields\_WithNilNaturalPerson` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1492`
64. `TestExtractJobIDFromPartialJSON$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:568`
65. `TestSaveExternalDataToSeaweedFS\_Success` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1616`
66. `TestPublishJobNotification\_WithAllOptions` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/job\_notification\_test.go:492`
67. `TestDecryptPluginCRMData\_MissingHashSecretKey` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1183`
68. `TestDecryptTopLevelFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:552`
69. `TestDecryptLegalPersonFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:789`
70. `TestDecryptContactFields\_WithNilContact` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1402`
71. `TestParseMessage\_UpdateStatusError` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1011`
72. `TestHashFilterValues` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:73`
73. `TestExtractJobIDFromPartialJSON\_RegexFallback` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:987`
74. `TestSaveExternalDataToSeaweedFS\_EmptyResult` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1948`
75. `TestExtractJobIDFromPartialJSON\_ValidJobIDInvalidOrgID` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1281`
76. `TestDecryptFieldValue\_WithValidEncryptedValue` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_crm\_data\_test.go:1725`
77. `TestParseMessage\_ValidMessage` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:43`
78. `TestQueryDatabase\_DataSourceFactoryAndLifecycleErrors$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_additional\_test.go:179`
79. `TestQueryDatabase\_ConnectionNotFound` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1411`
80. `TestEncryptDataForSeaweedFS\_InvalidCipherInitialization` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract\_data\_test.go:1857`

**Callees (this function depends on):**


#### `TestCustomContextKey\_Type`
**File:** `pkg/context\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`

#### `TestNewLoggerFromContext`
**File:** `pkg/context\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`

#### `TestContextWithLogger`
**File:** `pkg/context\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`

#### `TestContextWithTracer`
**File:** `pkg/context\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewTracerProvider`
2. `go.opentelemetry.io/otel/trace/noop.TracerProvider.Tracer`
3. `NewTracerProvider`
4. `go.opentelemetry.io/otel/trace/noop.TracerProvider.Tracer`
5. `NewTracerProvider`
6. `go.opentelemetry.io/otel/trace/noop.TracerProvider.Tracer`
7. `NewTracerProvider`
8. `go.opentelemetry.io/otel/trace/noop.TracerProvider.Tracer`
9. `\*testing.T.Run`

#### `TestContextWithLoggerAndTracer\_Combined`
**File:** `pkg/context\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`

#### `TestCustomContextKeyValue\_Integration`
**File:** `pkg/context\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`

#### `NewDataSourceFromConnection`
**File:** `pkg/datasource/datasource-factory.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.TestConnection.Execute` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection.go:124`
2. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.queryDatabase` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract-data.go:384`
3. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/command.CreateFetcherJob.TestConnection` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create\_fetcher\_job.go:366`
4. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.TestConnection.Execute` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/test\_connection.go:124`
5. `\*github.com/LerianStudio/fetcher/components/worker/internal/services.UseCase.queryDatabase` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/services/extract-data.go:384`
6. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.ValidateSchema.getOrFetchSchema` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema.go:257`
7. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/query.ValidateSchema.getOrFetchSchema` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/query/validate\_schema.go:257`
8. `\*github.com/LerianStudio/fetcher/components/manager/internal/services/command.CreateFetcherJob.TestConnection` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/services/command/create\_fetcher\_job.go:366`
9. `NewDataSourceFromConnectionWithLogger$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/datasource/datasource-factory.go:454`

**Callees (this function depends on):**

1. `newDataSourceFromConnection`

#### `generateImageTag`
**File:** `pkg/itestkit/addons/e2ekit/build\_secrets.go`
**Risk Level:** HIGH (21 direct callers)

**Direct Callers (signature change affects these):**

1. `\*github.com/xdg-go/scram.ServerConversation.firstMsg` - `/Users/fredamaral/go/pkg/mod/github.com/xdg-go/scram@v1.2.0/server\_conv.go:172`
2. `\*go.mongodb.org/mongo-driver/mongo/options.ClientOptions.ApplyURI` - `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/mongo/options/clientoptions.go:577`
3. `\*github.com/microsoft/go-mssqldb.tdsSession.Log` - `/Users/fredamaral/go/pkg/mod/github.com/microsoft/go-mssqldb@v1.9.6/session.go:92`
4. `go.uber.org/mock/gomock.StringerFunc.String` - `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/matchers.go:58`
5. `run$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/benchmark.go:282`
6. `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167`
7. `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167`
8. `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167`
9. `go.opentelemetry.io/otel/sdk/resource.processRuntimeNameDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:156`
10. `\*github.com/LerianStudio/fetcher/tests/fuzz/shared/mocks.MockCryptor.KeyVersion` - `/Users/fredamaral/repos/lerianstudio/fetcher/tests/fuzz/shared/mocks/mocks.go:39`
11. `Dlclose` - `/Users/fredamaral/go/pkg/mod/github.com/ebitengine/purego@v0.9.1/dlfcn.go:71`
12. `TestBuildConfigHelpers` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:88`
13. `TestBuildConfigHelpers` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:89`
14. `Dlsym` - `/Users/fredamaral/go/pkg/mod/github.com/ebitengine/purego@v0.9.1/dlfcn.go:58`
15. `go.opentelemetry.io/otel/sdk/resource.processRuntimeVersionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:161`
16. `buildImageWithSecrets` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets.go:52`
17. `Run$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/benchmark.go:854`
18. `\*github.com/xdg-go/scram.ClientConversation.firstMsg` - `/Users/fredamaral/go/pkg/mod/github.com/xdg-go/scram@v1.2.0/client\_conv.go:81`
19. `Dlopen` - `/Users/fredamaral/go/pkg/mod/github.com/ebitengine/purego@v0.9.1/dlfcn.go:43`
20. `go.opentelemetry.io/otel/sdk/resource.osTypeDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/os.go:36`
21. `OnceValue\[string\]$1$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/sync/oncefunc.go:66`

**Callees (this function depends on):**

1. `Read`
2. `EncodeToString`
3. `Sprintf`

#### `uniqueAppend`
**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** HIGH (6 direct callers)

**Direct Callers (signature change affects these):**

1. `github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.waitHTTP.Configure` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:298`
2. `github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.waitPort.Configure` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:340`
3. `findPath$1` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/loader/loader.go:917`
4. `\*golang.org/x/tools/go/loader.importer.findPath` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/loader/loader.go:924`
5. `TestBuilderHelpersAndProjectRoot$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:192`
6. `TestBuilderHelpersAndProjectRoot$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:193`

**Callees (this function depends on):**


#### `rewriteLocalhostForContainer`
**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** HIGH (15 direct callers)

**Direct Callers (signature change affects these):**

1. `\*go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.42.0/internal/envconfig/envconfig.go:46`
2. `ProcessFiles` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/internal/cgo/cgo.go:79`
3. `TestBuilderHelpersAndProjectRoot$4$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:179`
4. `WithServerAppName$1` - `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/x/mongo/driver/topology/server\_options.go:96`
5. `github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.localhostToHostGatewayRewriter.Rewrite` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:373`
6. `addStdlibCandidates$2` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/internal/imports/fix.go:1121`
7. `ParseFile` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/util.go:41`
8. `FakeContext$2` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:48`
9. `sanitiseName` - `/Users/fredamaral/go/pkg/mod/github.com/mdelapenya/tlscert@v0.2.0/tlscert.go:321`
10. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@v1.42.0/internal/envconfig/envconfig.go:46`
11. `FakeContext$4` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:72`
12. `Expand` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:35`
13. `FakeContext$3` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:55`
14. `parseFiles$2` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/loader/util.go:62`
15. `ReplaceAllStringFunc$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/regexp/regexp.go:578`

**Callees (this function depends on):**

1. `HostGatewayIP`
2. `Parse`
3. `\*net/url.URL.Hostname`
4. `\*net/url.URL.Port`
5. `\*net/url.URL.String`
6. `HasPrefix`
7. `Replace`
8. `HasPrefix`
9. `Replace`
10. `ReplaceAll`
11. `ReplaceAll`
12. `ReplaceAll`
13. `ReplaceAll`
14. `ReplaceAll`
15. `ReplaceAll`

#### `cloneMap`
**File:** `pkg/itestkit/addons/e2ekit/builder.go`
**Risk Level:** MEDIUM (4 direct callers)

**Direct Callers (signature change affects these):**

1. `github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.stubRewriter.Rewrite` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:17`
2. `\*github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.Builder.Run` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:188`
3. `github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit.localhostToHostGatewayRewriter.Rewrite` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/builder.go:371`
4. `TestBuilderHelpersAndProjectRoot$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:199`

**Callees (this function depends on):**


#### `ProjectRoot`
**File:** `pkg/itestkit/addons/e2ekit/helpers.go`
**Risk Level:** HIGH (19 direct callers)

**Direct Callers (signature change affects these):**

1. `\*github.com/xdg-go/scram.ServerConversation.firstMsg` - `/Users/fredamaral/go/pkg/mod/github.com/xdg-go/scram@v1.2.0/server\_conv.go:172`
2. `\*go.mongodb.org/mongo-driver/mongo/options.ClientOptions.ApplyURI` - `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/mongo/options/clientoptions.go:577`
3. `\*github.com/microsoft/go-mssqldb.tdsSession.Log` - `/Users/fredamaral/go/pkg/mod/github.com/microsoft/go-mssqldb@v1.9.6/session.go:92`
4. `go.uber.org/mock/gomock.StringerFunc.String` - `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/matchers.go:58`
5. `run$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/benchmark.go:282`
6. `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167`
7. `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167`
8. `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167`
9. `go.opentelemetry.io/otel/sdk/resource.processRuntimeNameDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:156`
10. `\*github.com/LerianStudio/fetcher/tests/fuzz/shared/mocks.MockCryptor.KeyVersion` - `/Users/fredamaral/repos/lerianstudio/fetcher/tests/fuzz/shared/mocks/mocks.go:39`
11. `Dlclose` - `/Users/fredamaral/go/pkg/mod/github.com/ebitengine/purego@v0.9.1/dlfcn.go:71`
12. `Dlsym` - `/Users/fredamaral/go/pkg/mod/github.com/ebitengine/purego@v0.9.1/dlfcn.go:58`
13. `go.opentelemetry.io/otel/sdk/resource.processRuntimeVersionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:161`
14. `Run$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/benchmark.go:854`
15. `\*github.com/xdg-go/scram.ClientConversation.firstMsg` - `/Users/fredamaral/go/pkg/mod/github.com/xdg-go/scram@v1.2.0/client\_conv.go:81`
16. `Dlopen` - `/Users/fredamaral/go/pkg/mod/github.com/ebitengine/purego@v0.9.1/dlfcn.go:43`
17. `go.opentelemetry.io/otel/sdk/resource.osTypeDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/os.go:36`
18. `OnceValue\[string\]$1$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/sync/oncefunc.go:66`
19. `TestBuilderHelpersAndProjectRoot$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:123`

**Callees (this function depends on):**

1. `\*sync.Once.Do`

#### `ProjectRootFrom`
**File:** `pkg/itestkit/addons/e2ekit/helpers.go`
**Risk Level:** HIGH (16 direct callers)

**Direct Callers (signature change affects these):**

1. `\*go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.42.0/internal/envconfig/envconfig.go:46`
2. `ProcessFiles` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/internal/cgo/cgo.go:79`
3. `WithServerAppName$1` - `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/x/mongo/driver/topology/server\_options.go:96`
4. `addStdlibCandidates$2` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/internal/imports/fix.go:1121`
5. `ParseFile` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/util.go:41`
6. `FakeContext$2` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:48`
7. `sanitiseName` - `/Users/fredamaral/go/pkg/mod/github.com/mdelapenya/tlscert@v0.2.0/tlscert.go:321`
8. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@v1.42.0/internal/envconfig/envconfig.go:46`
9. `FakeContext$4` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:72`
10. `Expand` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:35`
11. `FakeContext$3` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:55`
12. `parseFiles$2` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/loader/util.go:62`
13. `ReplaceAllStringFunc$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/regexp/regexp.go:578`
14. `TestBuilderHelpersAndProjectRoot$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:111`
15. `TestBuilderHelpersAndProjectRoot$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:115`
16. `TestBuilderHelpersAndProjectRoot$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:119`

**Callees (this function depends on):**

1. `Stat`
2. `Dir`
3. `\*archive/tar.headerFileInfo.IsDir`
4. `\*golang.org/x/tools/go/buildutil.fakeDirInfo.IsDir`
5. `golang.org/x/tools/go/buildutil.fakeDirInfo.IsDir`
6. `\*golang.org/x/net/webdav.memFileInfo.IsDir`
7. `\*github.com/spf13/afero/mem.FileInfo.IsDir`
8. `\*embed.file.IsDir`
9. `\*github.com/moby/go-archive/tarheader.nosysFileInfo.IsDir`
10. `archive/tar.headerFileInfo.IsDir`
11. `golang.org/x/tools/go/buildutil.fakeFileInfo.IsDir`
12. `github.com/moby/go-archive/tarheader.nosysFileInfo.IsDir`
13. `github.com/spf13/afero.dirEntry.IsDir`
14. `\*os.fileStat.IsDir`
15. `\*golang.org/x/tools/go/buildutil.fakeFileInfo.IsDir`
16. `\*github.com/spf13/afero.dirEntry.IsDir`
17. `Join`
18. `Stat`
19. `Dir`

#### `compareValues`
**File:** `pkg/itestkit/addons/queuekit/matcher.go`
**Risk Level:** MEDIUM (4 direct callers)

**Direct Callers (signature change affects these):**

1. `MatchJSONField$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher.go:159`
2. `TestCompareValues$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/matcher\_test.go:338`
3. `\*internal/sync.HashTrieMap\[any, any\].iter\[any any\]` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/internal/sync/hashtriemap.go:512`
4. `AssertJSONField` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions.go:284`

**Callees (this function depends on):**


#### `portKey`
**File:** `pkg/itestkit/container\_generic.go`
**Risk Level:** HIGH (14 direct callers)

**Direct Callers (signature change affects these):**

1. `\*go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.42.0/internal/envconfig/envconfig.go:46`
2. `ProcessFiles` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/internal/cgo/cgo.go:79`
3. `WithServerAppName$1` - `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/x/mongo/driver/topology/server\_options.go:96`
4. `addStdlibCandidates$2` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/internal/imports/fix.go:1121`
5. `ParseFile` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/util.go:41`
6. `FakeContext$2` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:48`
7. `sanitiseName` - `/Users/fredamaral/go/pkg/mod/github.com/mdelapenya/tlscert@v0.2.0/tlscert.go:321`
8. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/envconfig.EnvOptionsReader.GetEnvValue` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@v1.42.0/internal/envconfig/envconfig.go:46`
9. `FakeContext$4` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:72`
10. `Expand` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/os/env.go:35`
11. `\*github.com/LerianStudio/fetcher/pkg/itestkit.genericContainerInfra.Start` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/container\_generic.go:143`
12. `FakeContext$3` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/buildutil/fakecontext.go:55`
13. `parseFiles$2` - `/Users/fredamaral/go/pkg/mod/golang.org/x/tools@v0.41.0/go/loader/util.go:62`
14. `ReplaceAllStringFunc$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/regexp/regexp.go:578`

**Callees (this function depends on):**


#### `uniqueAppendMany`
**File:** `pkg/itestkit/customizer\_options.go`
**Risk Level:** HIGH (5 direct callers)

**Direct Callers (signature change affects these):**

1. `CHostDockerInternal$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options.go:63`
2. `CBindMount$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options.go:125`
3. `CExposedPorts$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options.go:31`
4. `CNetworks$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options.go:100`
5. `CNetworkAliases$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options.go:116`

**Callees (this function depends on):**


#### `CNetworkAliases`
**File:** `pkg/itestkit/customizer\_options.go`
**Risk Level:** HIGH (6 direct callers)

**Direct Callers (signature change affects these):**

1. `\*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mysql.MySQLInfra.Start` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mysql/mysql.go:100`
2. `\*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mongodb.MongoDBInfra.Start` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mongodb/mongodb.go:85`
3. `\*github.com/LerianStudio/fetcher/pkg/itestkit/infra/redis.RedisInfra.Start` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/redis/redis.go:75`
4. `\*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mssql.MSSQLInfra.Start` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mssql/mssql.go:85`
5. `\*github.com/LerianStudio/fetcher/pkg/itestkit/infra/rabbitmq.RabbitInfra.Start` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/rabbitmq/rabbitmq.go:89`
6. `\*github.com/LerianStudio/fetcher/pkg/itestkit/infra/postgres.PostgresInfra.Start` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/postgres/postgres.go:94`

**Callees (this function depends on):**


#### `HostGatewayIP`
**File:** `pkg/itestkit/hostport.go`
**Risk Level:** HIGH (19 direct callers)

**Direct Callers (signature change affects these):**

1. `\*github.com/xdg-go/scram.ServerConversation.firstMsg` - `/Users/fredamaral/go/pkg/mod/github.com/xdg-go/scram@v1.2.0/server\_conv.go:172`
2. `\*go.mongodb.org/mongo-driver/mongo/options.ClientOptions.ApplyURI` - `/Users/fredamaral/go/pkg/mod/go.mongodb.org/mongo-driver@v1.17.9/mongo/options/clientoptions.go:577`
3. `\*github.com/microsoft/go-mssqldb.tdsSession.Log` - `/Users/fredamaral/go/pkg/mod/github.com/microsoft/go-mssqldb@v1.9.6/session.go:92`
4. `go.uber.org/mock/gomock.StringerFunc.String` - `/Users/fredamaral/go/pkg/mod/go.uber.org/mock@v0.6.0/gomock/matchers.go:58`
5. `run$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/benchmark.go:282`
6. `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167`
7. `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167`
8. `go.opentelemetry.io/otel/sdk/resource.processRuntimeDescriptionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:167`
9. `go.opentelemetry.io/otel/sdk/resource.processRuntimeNameDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:156`
10. `\*github.com/LerianStudio/fetcher/tests/fuzz/shared/mocks.MockCryptor.KeyVersion` - `/Users/fredamaral/repos/lerianstudio/fetcher/tests/fuzz/shared/mocks/mocks.go:39`
11. `Dlclose` - `/Users/fredamaral/go/pkg/mod/github.com/ebitengine/purego@v0.9.1/dlfcn.go:71`
12. `Dlsym` - `/Users/fredamaral/go/pkg/mod/github.com/ebitengine/purego@v0.9.1/dlfcn.go:58`
13. `go.opentelemetry.io/otel/sdk/resource.processRuntimeVersionDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/process.go:161`
14. `Run$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/benchmark.go:854`
15. `\*github.com/xdg-go/scram.ClientConversation.firstMsg` - `/Users/fredamaral/go/pkg/mod/github.com/xdg-go/scram@v1.2.0/client\_conv.go:81`
16. `NormalizeHost` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:88`
17. `Dlopen` - `/Users/fredamaral/go/pkg/mod/github.com/ebitengine/purego@v0.9.1/dlfcn.go:43`
18. `go.opentelemetry.io/otel/sdk/resource.osTypeDetector.Detect` - `/Users/fredamaral/go/pkg/mod/go.opentelemetry.io/otel/sdk@v1.42.0/resource/os.go:36`
19. `OnceValue\[string\]$1$1` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/sync/oncefunc.go:66`

**Callees (this function depends on):**

1. `\*sync.Once.Do`

#### `ParseHostPort`
**File:** `pkg/itestkit/hostport.go`
**Risk Level:** MEDIUM (4 direct callers)

**Direct Callers (signature change affects these):**

1. `ResolveContainerHostPort` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:126`
2. `ResolveContainerHostPort` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:133`
3. `ResolveHostHostPort` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:113`
4. `ResolveHostHostPort` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/hostport.go:116`

**Callees (this function depends on):**

1. `SplitHostPort`
2. `Errorf`
3. `Atoi`
4. `Errorf`

#### `ResolveHostHostPort`
**File:** `pkg/itestkit/hostport.go`
**Risk Level:** MEDIUM (4 direct callers)

**Direct Callers (signature change affects these):**

1. `\*github.com/LerianStudio/fetcher/pkg/itestkit/infra/redis.RedisInfra.HostPort` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/redis/redis.go:178`
2. `\*github.com/LerianStudio/fetcher/pkg/itestkit/infra/postgres.PostgresInfra.HostPort` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/postgres/postgres.go:184`
3. `\*github.com/LerianStudio/fetcher/pkg/itestkit/infra/mongodb.MongoDBInfra.HostPort` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/mongodb/mongodb.go:182`
4. `\*github.com/LerianStudio/fetcher/pkg/itestkit/infra/rabbitmq.RabbitInfra.HostPort` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/rabbitmq/rabbitmq.go:174`

**Callees (this function depends on):**

1. `ParseHostPort`
2. `ParseHostPort`

#### `NewSeaweedFSInfra`
**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go`
**Risk Level:** MEDIUM (3 direct callers)

**Direct Callers (signature change affects these):**

1. `TestSeaweedFSHelpersWithoutDocker$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:46`
2. `TestSeaweedFSHelpersWithoutDocker$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:136`
3. `TestSeaweedFSHelpersWithoutDocker$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:23`

**Callees (this function depends on):**


#### `NewConnectionMongoDBRepository`
**File:** `pkg/mongodb/connection/connection.mongodb.go`
**Risk Level:** MEDIUM (4 direct callers)

**Direct Callers (signature change affects these):**

1. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:196`
2. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:196`
3. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:173`
4. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:173`

**Callees (this function depends on):**

1. `New`
2. `Background`
3. `WithTimeout`
4. `init$5`
5. `init$bound`
6. `DefPredeclaredTestFuncs`
7. `init$bound`
8. `Parse$1`
9. `OnceValues\[map\[string\]rune map\[string\]\[2\]rune\]$1$1$1`
10. `\_GC`
11. `init\#2`
12. `libc\_munlock\_trampoline`
13. `init\#3`
14. `init\#12`
15. `init\#1`
16. `logUnexpectedFailure$1`
17. `init\#1`
18. `queryDatabase$1`
19. `initP256`
20. `fuzz$1`
21. `encap$2`
22. `libc\_setpgid\_trampoline`
23. `acquireThread$1`
24. `libc\_setuid\_trampoline`
25. `fRunner$1`
26. `init\#2`
27. `libc\_kill\_trampoline`
28. `main`
29. `arc4random\_buf\_trampoline`
30. `setDefaultRuntimeProviders`
31. `makeSha256Reader$1`
32. `MockEnv$1`
33. `TestRabbitMQAdapter\_ProcessDelivery\_ExtractsHeaders$1$3`
34. `tRunner$1`
35. `cancellationListenerCallback$bound`
36. `addTLS$1`
37. `libc\_faccessat\_trampoline`
38. `SafeGoWithContextAndComponent$1`
39. `pthread\_kill\_trampoline`
40. `watchCancel$1`
41. `x509\_CFStringCreateExternalRepresentation\_trampoline`
42. `readRequest$1`
43. `setitimer\_trampoline`
44. `init\#3`
45. `handleRawConn$1`
46. `stubConnectionSpanAttributes$2`
47. `traceParameterStatus$1`
48. `traceBackendKeyData$1`
49. `onShutdownTimer$bound`
50. `open$1`
51. `TestScanRows$2$1`
52. `dispatchDeliveries$1`
53. `init$1`
54. `debugCallWrap2$1`
55. `TestRabbitMQAdapter\_ProducerDefault\_SkipsSigningWhenDisabled$1`
56. `entersyscallblock$3`
57. `fixedHuffmanDecoderInit`
58. `OnceValue\[\[\]string\]$1$1`
59. `setDefaultOSProviders`
60. `init\#1`
61. `typInternal$2`
62. `setMemoryLimit$1`
63. `snapshotMetricsRegistryForTesting$1`
64. `libc\_fpathconf\_trampoline`
65. `clientGetURLDeadline$1`
66. `startupMessage$5`
67. `init\#1`
68. `Clearenv`
69. `init\#1`
70. `init\#1`
71. `startHealthCheck$4`
72. `HostGatewayIP$1`
73. `newClientStreamWithParams$3`
74. `libc\_kevent\_trampoline`
75. `ReadFile$1`
76. `sharedMemTempFile$1`
77. `libc\_pread\_trampoline`
78. `init\#3`
79. `reentersyscall$1`
80. `init\#1`
81. `eventsTmpl$1`
82. `Compile$1`
83. `main`
84. `getGCMaskOnDemand$1`
85. `init\#9`
86. `newTypeObject$1`
87. `init\#1`
88. `TestRabbitMQAdapter\_ConsumerLoop\_ReturnsNilOnContextCancellation$1`
89. `LockOSThread`
90. `stop$bound`
91. `Gosched`
92. `libc\_getdtablesize\_trampoline`
93. `init\#2$4`
94. `legacyLoadMessageDesc$1$1`
95. `poll$1`
96. `sendBatchExtendedWithDescription$1`
97. `lazyInit$1`
98. `file\_opentelemetry\_proto\_trace\_v1\_trace\_proto\_init`
99. `start$1`
100. `syscall\_runtime\_AfterForkInChild`
101. `init\#1`
102. `TestInitWorker\_PanicsWhenLoggerInitFails$4`
103. `NewHTTP2Client$3`
104. `New\[\*crypto/internal/fips140/sha512.Digest\]$1`
105. `init\#1$1$1`
106. `SaveMultipartFile$1`
107. `CloseNotify$1`
108. `spillArgs`
109. `NotifyContext$1`
110. `Shutdown$1`
111. `init\#1`
112. `freemcache$1`
113. `SendMsg$2`
114. `Build$1`
115. `init\#2`
116. `collectTypeParams$1`
117. `Add$1`
118. `libc\_read\_trampoline`
119. `libresolv\_res\_9\_ninit\_trampoline`
120. `Less$1$1`
121. `Pull2$3`
122. `main`
123. `file\_google\_protobuf\_duration\_proto\_init`
124. `initMime`
125. `init\#1`
126. `sendFile$1`
127. `TestRabbitMQAdapter\_ConsumerLoop\_VerifiesSignatureSuccessfully$1`
128. `initMetrics`
129. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnInvalidSignature$1`
130. `cgoBindM`
131. `chansend$1`
132. `x509\_SecPolicyCreateSSL\_trampoline`
133. `runtime\_procUnpin`
134. `runCleanup$1`
135. `libc\_getnameinfo\_trampoline`
136. `file\_opentelemetry\_proto\_collector\_logs\_v1\_logs\_service\_proto\_init`
137. `getConn$1`
138. `init\#1`
139. `callContinuation$1`
140. `generatorTable$1`
141. `p224SqrtCandidate$1`
142. `resetspinning`
143. `acquireForkLock`
144. `sigpanic`
145. `CopyFrom$1`
146. `libc\_ioctl\_trampoline`
147. `readValue$1`
148. `instantiatedType$2`
149. `probe$bound`
150. `Ping$1`
151. `AddSet$1`
152. `runHandler$1`
153. `invokeMarshaler$1`
154. `newClusterState$1`
155. `dropg`
156. `onceSetNextProtoDefaults$bound`
157. `printunlock`
158. `sysctlbyname\_trampoline`
159. `dumpmemprof`
160. `kqueue\_trampoline`
161. `ParseBase$1`
162. `ReadTrace$1`
163. `TestRabbitMQAdapter\_ConsumerLoop\_ReturnsNilOnContextDeadlineExceeded$1`
164. `prepareDC$2`
165. `alloc$1`
166. `init\#1`
167. `runN$1`
168. `libc\_utimensat\_trampoline`
169. `SendMsg$1`
170. `TestValidateParametersNonMetadataKeys$1`
171. `init\#1`
172. `urandomRead$1`
173. `SendFile$1`
174. `traceStopReadCPU`
175. `main`
176. `file\_opentelemetry\_proto\_collector\_metrics\_v1\_metrics\_service\_proto\_rawDescGZIP$1`
177. `Notify$1$1`
178. `scheduleNextConnectionLocked$2`
179. `gcMarkDone$2`
180. `TestRedisCache\_NewRedisCache\_NilConnection\_Panics$1$1`
181. `init\#1`
182. `Close$1`
183. `asyncPreempt`
184. `init\#2`
185. `libc\_mkdirat\_trampoline`
186. `objDecl$1`
187. `TestRabbitMQAdapter\_ProcessDelivery\_ExtractsHeaders$1$1`
188. `Construct$1`
189. `put$1`
190. `traceParse$1`
191. `readType$1`
192. `freeSpan$1`
193. `handleForwards$bound`
194. `libc\_munlockall\_trampoline`
195. `libc\_getgrgid\_r\_trampoline`
196. `gcResetMarkState`
197. `retryLocked$1$1`
198. `runPiped$2`
199. `isParameterized$1`
200. `OnceValue\[map\[string\]reflect.Value\]$1$1`
201. `WithCallStackHelper$1`
202. `basepointNafTable$1`
203. `init\#1`
204. `trace$1`
205. `rawExpr$1`
206. `dumpms`
207. `collectExemplars\[int64\]$1`
208. `gcComputeStartingStackSize`
209. `withConn$1`
210. `ForceFlush$1`
211. `TestRabbitMQAdapter\_ProducerDefault\_SignsMessage$1`
212. `libc\_setsockopt\_trampoline`
213. `init\#2`
214. `ClearStringValidations$1`
215. `init\#1`
216. `libc\_getpgid\_trampoline`
217. `StopMetricsCollector`
218. `WriteOverlays$1`
219. `init\#2`
220. `Shutdown$1`
221. `createContext$8`
222. `libc\_symlink\_trampoline`
223. `stacklessWriterFunc$1`
224. `libpreinit`
225. `x509\_CFNumberGetValue\_trampoline`
226. `Peek$1`
227. `main$1`
228. `resolve$1`
229. `rlock$1`
230. `decIgnoreOpFor$1`
231. `Stop$1`
232. `captureStack$1`
233. `cgounimpl`
234. `libc\_freeaddrinfo\_trampoline`
235. `file\_google\_rpc\_error\_details\_proto\_rawDescGZIP$1`
236. `TestInitWorker\_PanicsWhenConfigLoadFails$1`
237. `stop$1`
238. `stacklessWriteBrotli$1`
239. `init\#1`
240. `run1$1`
241. `read\_trampoline`
242. `handlePing$1`
243. `maintain$4`
244. `noopRedeemer`
245. `goschedIfBusy`
246. `init\#1`
247. `connectOne$2`
248. `main$1`
249. `Test$1$1`
250. `init\#2`
251. `runtime\_AfterFork`
252. `FlushDNSCache`
253. `exitsyscall$2`
254. `EndTracingSpansInterceptor$1$1`
255. `fatalpanic$1`
256. `fRunner$1$1`
257. `worldStarted`
258. `handleSettings$2`
259. `init\#3`
260. `registerHTTPSProtocol$1`
261. `performHandoffInternal$1`
262. `initConfVal$1`
263. `WithCallStackHelper$2`
264. `itabsinit`
265. `checkFinalizersAndCleanups`
266. `validateNorm$1`
267. `main`
268. `prepareForRecovery$1`
269. `archInitIEEE`
270. `build$1`
271. `watchCancel$2`
272. `Shutdown$1`
273. `cancellationListenerCallback$bound`
274. `yaml\_parser\_fetch\_next\_token$1`
275. `Read$1`
276. `init\#1`
277. `Pull2$1$2`
278. `checkmcount`
279. `testAtomic64`
280. `x509\_CFDataCreate\_trampoline`
281. `Close$bound`
282. `traceSyncBatch$1`
283. `libc\_getsockopt\_trampoline`
284. `list$1`
285. `libc\_write\_trampoline`
286. `dispatchDeliveries$1$1`
287. `generatorTable$1`
288. `newUserArenaChunk$1`
289. `init\#1`
290. `checkMinIdleConns$1`
291. `init\#1`
292. `exposeHostPorts$2`
293. `libc\_socketpair\_trampoline`
294. `synctest\_inBubble$1`
295. `doRetryNotify\[github.com/docker/docker/api/types/build.ImageBuildResponse\]$1`
296. `OnceValues$1$1`
297. `internal\_sync\_runtime\_doSpin`
298. `Start$1`
299. `libc\_access\_trampoline`
300. `file\_google\_rpc\_error\_details\_proto\_init`
301. `init\#1`
302. `executeShutdown$1`
303. `OnceValue$1$1$1`
304. `init\#1`
305. `handleEnv`
306. `InitLocalEnvConfig$1`
307. `file\_google\_api\_httpbody\_proto\_init`
308. `TestServiceRun$1`
309. `NewServerTransport$2`
310. `libc\_pwrite\_trampoline`
311. `writeHeapProto$1`
312. `fatalthrow$1`
313. `forcegchelper`
314. `init\#1`
315. `walkRange$2$1`
316. `DisableDIT`
317. `lockRankMayTraceFlush`
318. `lazyInit$1`
319. `StandardCrypto`
320. `initResolutionCache`
321. `coverReport`
322. `initP224`
323. `init\#1`
324. `decPenalty$bound`
325. `envProxyFunc$1`
326. `handleForwards$bound`
327. `init\#1`
328. `p521B$1`
329. `getLen$1`
330. `init\#1`
331. `onWriteTimeout$bound`
332. `exec$1`
333. `exposeHostPort$1`
334. `init\#1`
335. `ChannelWithSubscriptions$1`
336. `operateHeaders$2`
337. `NewPeriodicReader$2`
338. `main`
339. `ServeHTTP$1`
340. `onceSetNextProtoDefaults$bound`
341. `tlsClientHandshake$1`
342. `main$2`
343. `mstartm0`
344. `TestingOnlyAbandon`
345. `TestRabbitMQAdapter\_IsHealthy\_ReturnsTrue\_WhenChannelOpen$1`
346. `goroutineProfileWithLabelsSync$3`
347. `write$2`
348. `unminit`
349. `WithRecover$1$1`
350. `init\#1`
351. `ForEachPackage$2`
352. `typeDecl$3`
353. `minitSignalStack`
354. `entersyscallWakeSysmon`
355. `healthCheck$bound`
356. `init\#1`
357. `ParseBuildInfo$1`
358. `mapv$1`
359. `init\#1`
360. `main`
361. `main`
362. `throw$1`
363. `init\#1`
364. `asminit`
365. `do$1`
366. `libc\_pathconf\_trampoline`
367. `Exec$1`
368. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$1`
369. `setDefaultUserProviders`
370. `TestRabbitMQAdapter\_RunConsumerCycle\_FailsOnQosError$1`
371. `NewTelemetry$1`
372. `badmorestackg0$1`
373. `DecodeFull$1`
374. `maybeRunStateHook$bound`
375. `initHealthCheck$1`
376. `panicwrap`
377. `libc\_undelete\_trampoline`
378. `init\#2`
379. `WithCancel$1`
380. `initLocal`
381. `SendMsg$2`
382. `ReplaceGlobals$1`
383. `flushallmcaches`
384. `unlockOSThread`
385. `mapinitnoop`
386. `mallocinit`
387. `Close$1`
388. `TestRabbitMQAdapter\_ProducerDefault\_RetriesOnFailure$3$1`
389. `mountStartupProcess$1`
390. `Start$1`
391. `init\#1`
392. `gcinit`
393. `initConnPool$bound`
394. `init\#1`
395. `serveStreams$1`
396. `Go$1$1`
397. `libc\_setgid\_trampoline`
398. `gcWriteBarrier6`
399. `init\#3`
400. `RegisterMetricsServiceHandlerFromEndpoint$1`
401. `lazyInit$1`
402. `getcp950$1`
403. `init\#1`
404. `gcenable`
405. `swap$1`
406. `healthCheck$bound`
407. `NewHTTP2Client$5`
408. `StartTimeStampUpdater`
409. `init\#1`
410. `panicBounds`
411. `GetOrDownloadMongod$1`
412. `Reset`
413. `libc\_sendto\_trampoline`
414. `libc\_sync\_trampoline`
415. `WithDeadlineCause$3`
416. `init\#1`
417. `Go$1`
418. `tRunner$2`
419. `WriteOverlays$2$1`
420. `cmdStream$1`
421. `CoordinateFuzzing$2`
422. `isImage$1`
423. `basepointTable$1`
424. `Record$1`
425. `walkRange$1`
426. `AddSet$1`
427. `init\#2`
428. `apply$1`
429. `goServe$1`
430. `runSafePointFn`
431. `gcControllerCommit`
432. `netpollBreak`
433. `Readdir$1`
434. `prepareNext$1`
435. `ServeConn$1`
436. `Put$1`
437. `libc\_utimes\_trampoline`
438. `startTheWorld$1`
439. `init\#1`
440. `init\#1`
441. `testSPWrite`
442. `Record$1`
443. `libc\_rename\_trampoline`
444. `triggerHealthCheck$1`
445. `TurnOn`
446. `netpollinit`
447. `mProf\_Malloc$1`
448. `gcRestoreSyncObjects`
449. `EventuallyWithT$1$1`
450. `pthread\_attr\_setdetachstate\_trampoline`
451. `minitSignals`
452. `init\#1`
453. `reentersyscall$4`
454. `worldStopped`
455. `init\#6`
456. `init\#4`
457. `Execute$1`
458. `SendMsg$1`
459. `NewBatchSpanProcessor$2`
460. `TryGo$1`
461. `TestRabbitMQAdapter\_ProcessDelivery\_RecoverFromPanic$3`
462. `init\#1`
463. `setupMiniredis$1`
464. `runCleanups`
465. `main`
466. `setupHijackConn$1`
467. `RebaseArchiveEntries$1`
468. `TestConcurrentEncryption$1`
469. `ArtifactDir$1`
470. `gcAssistAlloc$2`
471. `libc\_fstat\_trampoline`
472. `MakeTimeoutContext$1`
473. `blocking$2`
474. `libc\_getgid\_trampoline`
475. `libc\_mmap\_trampoline`
476. `getData$1`
477. `file\_grpc\_health\_v1\_health\_proto\_init`
478. `collectFileInfoForChanges$1`
479. `DialContext$1`
480. `init\#1`
481. `init\#1`
482. `connect$1`
483. `basepointTable$1`
484. `kevent\_trampoline`
485. `init\#2`
486. `startChannelWatcher$1`
487. `WaitN$1$1`
488. `Next$1`
489. `startCloseMonitor$1`
490. `searchInStaticDictionary$1`
491. `propagateCancel$2`
492. `writeRecordLocked$1`
493. `CreateLUT`
494. `reset$1`
495. `libc\_getsid\_trampoline`
496. `init\#2$2`
497. `init\#1`
498. `RangeEntries$1`
499. `EnableColorsStdout$1`
500. `Flush$bound`
501. `libc\_umask\_trampoline`
502. `libc\_exchangedata\_trampoline`
503. `addmoduledata`
504. `initBenchmarkFlags`
505. `outgoingGoAwayHandler$1`
506. `init\#1`
507. `Cleanup$1$1`
508. `markroot$1`
509. `main`
510. `libc\_setegid\_trampoline`
511. `TestRabbitMQAdapter\_ConsumerLoop\_SkipsVerificationWhenDisabled$1`
512. `writeTrace$1`
513. `libc\_open\_trampoline`
514. `freezetheworld`
515. `printArgs$3`
516. `sync\_atomic\_runtime\_procUnpin`
517. `sigaction\_trampoline`
518. `collectExemplars\[float64\]$1`
519. `doCall$2`
520. `setSyncObjectsUntraceable`
521. `resetProxyConfig`
522. `operateHeaders$1`
523. `startDialConnForLocked$1`
524. `abortStreamLocked$1`
525. `HostGatewayIP$1`
526. `dounlockOSThread`
527. `gcMarkTermination$1`
528. `UUID$1`
529. `SaveImagesWithOpts$1`
530. `secret\_eraseSecrets`
531. `libc\_getgrouplist\_trampoline`
532. `madvise\_trampoline`
533. `init\#2`
534. `OrderedUnmarshalJSON$1`
535. `newextram`
536. `badTimer`
537. `Go$1`
538. `setPinned$2`
539. `libc\_revoke\_trampoline`
540. `debugCallWrap1`
541. `onIdleTimeout$bound`
542. `delayedFlush$bound`
543. `file\_opentelemetry\_proto\_metrics\_v1\_metrics\_proto\_rawDescGZIP$1`
544. `unminitSignals`
545. `init$bound`
546. `Add$1`
547. `Confirm$1`
548. `getempty$1`
549. `init\#1`
550. `init\#13`
551. `main`
552. `init\#1`
553. `init\#1`
554. `funcLit$1`
555. `libc\_mkdir\_trampoline`
556. `signalMessage$1`
557. `cleanup$1`
558. `\_System`
559. `init\#1`
560. `bytes$1`
561. `handleStateChange$1$1`
562. `WithDeadlineCause$1`
563. `register$bound`
564. `validType0$1`
565. `runtime\_pollServerInit`
566. `doInRoot\[string\]$1`
567. `Close$1`
568. `Alignof$1`
569. `getcp936$1`
570. `traceStartReadCPU`
571. `ProjectRoot$1`
572. `load\_g`
573. `nanotime\_trampoline`
574. `init\#1`
575. `onIdleTimer$bound`
576. `init\#1$1`
577. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnVersionMismatch$1`
578. `cpuVariant$1`
579. `connect$1`
580. `lockRankMayQueueFinalizer`
581. `libc\_fork\_trampoline`
582. `Test$1`
583. `methodValueCall`
584. `Read$1`
585. `mstart0`
586. `TestInitWorker\_PanicsWhenConfigLoadFails$3`
587. `init\#1`
588. `libc\_getrusage\_trampoline`
589. `schedule`
590. `abort`
591. `RecordApproved`
592. `init\#2`
593. `TestHMACSigner\_ConcurrentSigning$1`
594. `UnreachableExceptTests`
595. `MustExtractDockerHost$1`
596. `TestRabbitMQAdapter\_ProcessDelivery\_HandlesNonStringHeaderID$1`
597. `run$2`
598. `initFuzzFlags`
599. `Consume$1`
600. `makeFuncStub`
601. `libc\_fcntl\_trampoline`
602. `mach\_vm\_region\_trampoline`
603. `gcMarkTinyAllocs`
604. `init\#1`
605. `x509\_SecCertificateCreateWithData\_trampoline`
606. `Import$1`
607. `panicdivide`
608. `init\#1`
609. `gcMarkDone`
610. `Add$1`
611. `NewServerTransport$3`
612. `convertAssignRows$2`
613. `libc\_setpriority\_trampoline`
614. `serveContent$1`
615. `init\#4`
616. `RunFuzzWorker$1$1`
617. `init\#1`
618. `sweep$1`
619. `Channel$1`
620. `init\#1`
621. `TestRabbitMQAdapter\_ProducerDefault\_AllRetriesFail$1`
622. `ContainerWait$1`
623. `NewServerTransport$1`
624. `watchCancel$1`
625. `SortSliceBetween$1`
626. `run$1`
627. `init\#4`
628. `init\#1`
629. `listenerBacklog$1`
630. `init\#1`
631. `goenvs`
632. `onceSetNextProtoDefaults$bound`
633. `init\#1`
634. `OnEmit$2`
635. `init\#1`
636. `init\#1`
637. `NewWithConfig$5`
638. `file\_opentelemetry\_proto\_common\_v1\_common\_proto\_init`
639. `libc\_readlink\_trampoline`
640. `readFrames$1`
641. `schedinit`
642. `xRegInitAlloc`
643. `Decode$1`
644. `libc\_geteuid\_trampoline`
645. `\_ExternalCode`
646. `file\_opentelemetry\_proto\_collector\_trace\_v1\_trace\_service\_proto\_rawDescGZIP$1`
647. `exportContext$1`
648. `init\#1`
649. `ResolverError$1`
650. `startGracefulShutdown$1`
651. `setRequestCancel$2`
652. `getcp1251$1`
653. `delayedFlush$bound`
654. `ignoreSIGSYS`
655. `initPostcodes`
656. `main`
657. `mProf\_Flush`
658. `privateLogw$1`
659. `didPanic$1`
660. `runtime\_AfterExec`
661. `init\#1`
662. `Read$1`
663. `libc\_setrlimit\_trampoline`
664. `InitLocalEnvConfig$1`
665. `init\#1`
666. `Events$1`
667. `traceExitedSyscall`
668. `Enable`
669. `TestInitWorker\_PanicsWhenLoggerInitFails$1`
670. `tunnel$2`
671. `init\#2`
672. `casgstatus$1`
673. `stacklessWriteZstd$1`
674. `InitLocalEnvConfig$1`
675. `gcMarkTermination$5`
676. `loadHTTPBytes$1$1`
677. `warnBlocked`
678. `stacklessWriteGzip$1`
679. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P256Point\]\]$1$1`
680. `OnceValues$1$1$1`
681. `writeBody$1`
682. `TestRabbitMQAdapter\_ProducerDefault\_SkipsSigningWhenNoSigner$1`
683. `gcWriteBarrier5`
684. `pthread\_mutex\_unlock\_trampoline`
685. `freeStackSpans`
686. `init\#1`
687. `x509\_CFArrayAppendValue\_trampoline`
688. `libc\_recvfrom\_trampoline`
689. `libc\_gai\_strerror\_trampoline`
690. `libc\_getpeername\_trampoline`
691. `Watch$1`
692. `init\#3`
693. `lock$1`
694. `Record$1`
695. `defaultGOMAXPROCSUpdateEnable`
696. `infer$1`
697. `morestackc`
698. `doRetryNotify\[\*github.com/docker/docker/api/types/container.Summary\]$1`
699. `readPreface$1`
700. `modulesinit`
701. `PCall$1`
702. `init$2`
703. `write$1`
704. `FreeOSMemory`
705. `libc\_chmod\_trampoline`
706. `Default$1`
707. `setDefaultOSDescriptionProvider`
708. `init\#1`
709. `cgoSigtramp`
710. `ResetCoverage`
711. `runCmdContext$4`
712. `init\#5`
713. `typInternal$1`
714. `libc\_setreuid\_trampoline`
715. `gcStart$1`
716. `logDropped\[go.opentelemetry.io/otel/sdk/trace.Link\]$1`
717. `file\_opentelemetry\_proto\_trace\_v1\_trace\_proto\_rawDescGZIP$1`
718. `instantiatedType$1`
719. `init\#1`
720. `removePerishedConns$1`
721. `main`
722. `maybeRunStateHook$1`
723. `newNonRetryClientStream$2`
724. `gcTestMoveStackOnNextCall`
725. `AddSet$1`
726. `os\_sigpipe`
727. `Run$1`
728. `unlockProfiles`
729. `TestQueryDatabase\_DataSourceFactoryAndLifecycleErrors$1$1`
730. `ExportChanges$1`
731. `runCmdContext$3$1`
732. `main`
733. `init\#6`
734. `printDebugLog`
735. `InitializeTelemetry$2`
736. `writeEarlyAbort$2`
737. `connect$1`
738. `doinit`
739. `init\#1`
740. `fixedHuffmanDecoderInit$1`
741. `main`
742. `OnPut$1`
743. `xRegRestore$1`
744. `startGracefulShutdown$bound`
745. `TestRabbitMQAdapter\_ProducerDefault\_ReturnsError\_WhenEnsureChannelFails$1`
746. `Unreachable`
747. `goroutineLeakProfileWithLabelsConcurrent$1$1`
748. `Apply$1`
749. `updateTargetResolverState$1`
750. `AddSet$1`
751. `secret\_inc`
752. `libc\_setgroups\_trampoline`
753. `sendResponse$1`
754. `Stop$1`
755. `NewHTTP2Client$1`
756. `poolCleanup`
757. `TestRabbitMQAdapter\_RunConsumerCycle\_FailsOnConsumeError$1`
758. `GetOrBuildProducer$1`
759. `libc\_fchown\_trampoline`
760. `archInit`
761. `closeReqBodyLocked$1`
762. `Pull$3`
763. `init$2`
764. `parseFiles$2$1`
765. `libc\_exit\_trampoline`
766. `TestRabbitMQAdapter\_ProcessDelivery\_NackError$3`
767. `parseUnion$1`
768. `file\_google\_protobuf\_wrappers\_proto\_rawDescGZIP$1`
769. `lazyInit$1`
770. `runtime\_debug\_freeOSMemory$1`
771. `StatelessDeflate$2`
772. `init\#1`
773. `readAll$1`
774. `Force`
775. `onWriteTimeout$bound`
776. `init\#1`
777. `file\_opentelemetry\_proto\_logs\_v1\_logs\_proto\_rawDescGZIP$1`
778. `reflectOffsLock`
779. `TestProcessPluginCRMCollection\_WithValidOrganization$1`
780. `queryDatabase$1`
781. `allocm$1`
782. `secure`
783. `CreateContainer$1`
784. `ResetWithOptions$1`
785. `runExample$1`
786. `libc\_issetugid\_trampoline`
787. `init\#1`
788. `init\#1`
789. `Do$1`
790. `ResetServiceIndicator`
791. `main`
792. `main`
793. `AddSet$1`
794. `Close$1`
795. `init\#1`
796. `libc\_readdir\_r\_trampoline`
797. `init\#6`
798. `walk$1$1`
799. `libc\_sendmsg\_trampoline`
800. `basepointNafTable$1`
801. `initResourceValue\[\*github.com/jackc/pgx/v5/pgxpool.connResource\]$1`
802. `invokeError$1`
803. `CheckPath$1`
804. `send$1`
805. `scavenge$1`
806. `x509\_CFArrayCreateMutable\_trampoline`
807. `init\#17`
808. `Stop$1`
809. `mayMoreStackPreempt`
810. `SetMx$1`
811. `startupValidation$1`
812. `goready$1`
813. `connect$2`
814. `onceSetNextProtoDefaults\_Serve$bound`
815. `init\#1`
816. `addOption$1`
817. `corostart`
818. `ResetPanicMetrics`
819. `Acquire$1`
820. `libc\_sysctl\_trampoline`
821. `startStreamDecoder$1`
822. `onReadTimeout$bound`
823. `Flush$bound`
824. `search$2$1$1`
825. `Clearenv`
826. `main`
827. `pageTmpl$1`
828. `processTxPipelineNode$1$1`
829. `TestScanColumns$2$1`
830. `libc\_stat\_trampoline`
831. `breakpoint`
832. `NextResultSet$1`
833. `init\#1`
834. `pthread\_self\_trampoline`
835. `main`
836. `unsetBypass`
837. `StatsPrint`
838. `malg$1`
839. `libc\_getrlimit\_trampoline`
840. `syscall\_runtime\_BeforeExec`
841. `init$1`
842. `traceNotificationResponse$1`
843. `TestValidateParameters$1`
844. `TestQueryPluginCRM\_WithOrganizationOnly$1`
845. `getcp874$1`
846. `Commit$1`
847. `x509\_CFArrayGetCount\_trampoline`
848. `Record$1`
849. `Read$1`
850. `init\#1`
851. `createIdleResources$1`
852. `init\#2`
853. `ServeHTTP$1$1`
854. `updateMaxProcsGoroutine`
855. `Add$1`
856. `needsInitCheckLocked$1`
857. `panicunsafeslicenilptr`
858. `x509\_CFArrayGetValueAtIndex\_trampoline`
859. `ServeFile$1`
860. `main$1`
861. `scan$3`
862. `redirectStdLogAt$1`
863. `mapv$1`
864. `saveFile$1`
865. `mountStartupProcess$2`
866. `parsePattern$1`
867. `OrderedMarshalJSON$1`
868. `init\#1`
869. `systemstack\_switch`
870. `UpdateClientConnState$1`
871. `init\#4`
872. `RegisterTraceServiceHandlerFromEndpoint$1`
873. `init\#1`
874. `writeContext$1`
875. `addrLookupOrder$1`
876. `save\_g`
877. `NewPrivateKey$1`
878. `runtime\_AfterForkInChild`
879. `init\#2`
880. `exit\_trampoline`
881. `init\#1`
882. `InitLocalEnvConfig$1`
883. `Close$1`
884. `doRetryNotify\[\*github.com/testcontainers/testcontainers-go.Reaper\]$1`
885. `cancel$1`
886. `init\#4`
887. `compute$1`
888. `TestRedisCache\_NewRedisCache\_NilConnection\_Panics$2$1`
889. `embeddedIfaceMethStub`
890. `setRequestCancel$3`
891. `init\#2`
892. `x509\_SecCertificateCopyData\_trampoline`
893. `SetFinalizer$1`
894. `appendJSONMarshal$1`
895. `dumpScanStats`
896. `buildCommonHeaderMaps`
897. `TestConsumerRoutes\_Shutdown$2`
898. `StmtContext$2`
899. `chanrecv$1`
900. `run$1`
901. `TestScanColumns$1$1`
902. `initCommonHeader`
903. `init\#3`
904. `CloseNotify$1`
905. `processHandoffRequest$1`
906. `makeTempDir$1`
907. `PCall$1$1`
908. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P521Point\]\]$1$1$1`
909. `libc\_getfsstat\_trampoline`
910. `init\#1`
911. `writeHeader$1`
912. `prefork$2`
913. `ResetTestLicenseBaseURL`
914. `pingDC$1`
915. `parsePrimaryExpr$1`
916. `OnceFunc$1$1$1`
917. `file\_grpc\_health\_v1\_health\_proto\_rawDescGZIP$1`
918. `runtime\_goroutineLeakGC`
919. `listPackages$1`
920. `onReadTimeout$bound`
921. `OnceValue\[encoding/json.encoderFunc\]$1$1$1`
922. `file\_opentelemetry\_proto\_collector\_trace\_v1\_trace\_service\_proto\_init`
923. `traceCopyFail$1`
924. `sysmonUpdateGOMAXPROCS`
925. `synctestRun$2`
926. `main`
927. `prepareNext$1`
928. `snapshotConnection$1`
929. `libc\_writev\_trampoline`
930. `\_VDSO`
931. `RecordSet$1`
932. `prepareDC$1`
933. `scan$4`
934. `gorecover$1`
935. `Write$1`
936. `ScanAndListen$2`
937. `Run$1`
938. `Shutdown$1`
939. `parse$1`
940. `processOptions`
941. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P384Point\]\]$1$1$1`
942. `entersyscall`
943. `TestConsumerRoutes\_Shutdown$1`
944. `next$1`
945. `newTdsBuffer$1`
946. `stubJobSpanAttributes$2`
947. `readRequest$1`
948. `logDropped\[go.opentelemetry.io/otel/sdk/trace.Event\]$1`
949. `Stack$1`
950. `libc\_getppid\_trampoline`
951. `traceAdvance$1$1`
952. `init\#1`
953. `startTemplateThread`
954. `init\#1`
955. `libc\_unmount\_trampoline`
956. `gfget$1`
957. `LazyReload$1`
958. `TestProcessPluginCRMCollection\_WithOrganizationID$1`
959. `getCh$1`
960. `Cleanup$1`
961. `containsElement$1`
962. `libc\_ftruncate\_trampoline`
963. `UnregisterSpanProcessor$1`
964. `validateNorm$1`
965. `stop$1`
966. `dispatch0$1`
967. `unpinConnectionFromCursor$bound`
968. `Serve$2`
969. `TestValidateParametersWithCursor$1`
970. `setPinned$1`
971. `init\#1`
972. `markrootFreeGStacks`
973. `queryDC$1`
974. `init\#1`
975. `reentersyscall$2`
976. `traceDescribe$1`
977. `gcBgMarkStartWorkers`
978. `nextMarkBitArenaEpoch`
979. `pollSRVRecords$1`
980. `onReadIdleTimer$bound`
981. `traceInitReadCPU`
982. `libc\_socket\_trampoline`
983. `libc\_error\_trampoline`
984. `gcBeginWork`
985. `minimize$1`
986. `sendOpen$1`
987. `init\#7`
988. `TestMultiQueueConsumerRun$1`
989. `lockVerifyMSize`
990. `InitLocalEnvConfig$1`
991. `libc\_renameat\_trampoline`
992. `runCleanup$2`
993. `AddSet$1`
994. `handleUpgradeResponse$1`
995. `DecodeAll$1`
996. `readForm$1`
997. `CopyFileWithTar$1`
998. `beginDC$1`
999. `procUnpin`
1000. `closeRead$1`
1001. `initConfVal`
1002. `maintain$3`
1003. `close$1`
1004. `doRetryNotify$1`
1005. `structType$3`
1006. `Remove$1`
1007. `UploadLogs$1`
1008. `RegisterLogsServiceHandlerFromEndpoint$1$1`
1009. `read$1`
1010. `TestRabbitMQAdapter\_ProducerDefault\_Success$1$1`
1011. `duffcopy`
1012. `write\_trampoline`
1013. `CreateNetwork$1`
1014. `lostProfileEvent`
1015. `libc\_recvmsg\_trampoline`
1016. `NewHTTP2Client$6`
1017. `handleIdleTimeout$bound`
1018. `doInRoot\[os.FileInfo\]$1`
1019. `init\#3`
1020. `RunConsumers$1`
1021. `archInitCastagnoli`
1022. `pinConnectionToTransaction$bound`
1023. `execDC$2`
1024. `invalidateChannel$1`
1025. `buildCommonHeaderMapsOnce`
1026. `init\#16`
1027. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnHandlerError$1`
1028. `ParseAcceptLanguage$1`
1029. `libc\_chown\_trampoline`
1030. `Subscribe$2`
1031. `fieldByIndexErr$1`
1032. `doInRoot\[struct{}\]$1`
1033. `shutdownWorkers$2`
1034. `buildOnce$bound`
1035. `updateTargetResolverState$2`
1036. `prefork$1`
1037. `minitSignalMask`
1038. `getcp1254$1`
1039. `init\#3`
1040. `init\#1`
1041. `Serve$1`
1042. `morestack`
1043. `newServer$1`
1044. `ParseScript$1`
1045. `readPreface$1`
1046. `contactResponders$1$1`
1047. `StopCPUProfile`
1048. `ParseExprFrom$1`
1049. `AddSet$1`
1050. `ParseVariant$1`
1051. `goenvs\_unix`
1052. `SendBatch$1`
1053. `stoplockedm`
1054. `onWriteTimeout$bound`
1055. `write$1`
1056. `parseMultiplexedLogs$1`
1057. `minimize$2`
1058. `NewFastHTTPHandler$1$1`
1059. `lazyRegexCompile$1$1`
1060. `Shutdown$1$1$1`
1061. `ExitIdle$bound`
1062. `handleMoving$1`
1063. `ParseRegion$1`
1064. `main`
1065. `processStreamingRPC$1`
1066. `dispatchClosed$1`
1067. `SendMsg$4`
1068. `doCall$1`
1069. `init\#1`
1070. `fatal$1`
1071. `commandLineUsage`
1072. `AddSet$1`
1073. `Shutdown$1`
1074. `emptyfunc`
1075. `Less$1`
1076. `http2registerHTTPSProtocol$1`
1077. `init\#1`
1078. `beginFuncExec$1`
1079. `encodeError$1`
1080. `libc\_dup\_trampoline`
1081. `stderrHandler$1`
1082. `StatelessDeflate$1`
1083. `NotifyConfirm$1`
1084. `connect$1`
1085. `gcWakeAllStrongFromWeak`
1086. `Start$2`
1087. `init\#1`
1088. `getcp1255$1`
1089. `startLocked$1`
1090. `x509\_CFDataGetLength\_trampoline`
1091. `dit\_setDisabled`
1092. `checkOut$1`
1093. `UploadTraces$1`
1094. `RecordSet$1`
1095. `TestMultiQueueConsumerRun$2$1`
1096. `TestWithSwaggerEnvConfig\_DefaultValues$1`
1097. `callers$1`
1098. `file\_opentelemetry\_proto\_common\_v1\_common\_proto\_rawDescGZIP$1`
1099. `selectgo$2`
1100. `buildOnce$bound`
1101. `traceRowDescription$1`
1102. `ClearObjectValidations$1`
1103. `after$1`
1104. `failWantMap`
1105. `dial$1`
1106. `TestHandlerGenerateReport\_DelegatesToUseCase$1`
1107. `checkdead`
1108. `lazyInit$1`
1109. `aberrantDeriveMessageName$1`
1110. `libc\_grantpt\_trampoline`
1111. `main`
1112. `Scan$1`
1113. `gcWriteBarrier3`
1114. `SaveImagesWithOpts$2`
1115. `Shutdown$1`
1116. `exportSync$1`
1117. `startHealthCheck$1`
1118. `equalServiceConfig$1`
1119. `traceCPUFlush$1`
1120. `RecordSet$1`
1121. `SetMeterProvider$1`
1122. `randinit`
1123. `Run$1`
1124. `gcstopm`
1125. `gcStart$3`
1126. `getcp850$1`
1127. `libc\_shutdown\_trampoline`
1128. `SelectServer$1`
1129. `startLoad$1`
1130. `StartTimeStampUpdater$1`
1131. `pipe\_trampoline`
1132. `gcBgMarkWorker$2`
1133. `init\#1`
1134. `sysctl\_trampoline`
1135. `secret\_dec`
1136. `init\#4$4`
1137. `init\#1`
1138. `Record$1`
1139. `init\#1`
1140. `AddSet$1`
1141. `updateClientConnState$2`
1142. `init\#1`
1143. `buildCommonHeaderMaps`
1144. `buildShutdownHandlers$1`
1145. `main`
1146. `lazyInit$1`
1147. `PrintStack`
1148. `panicunsafeslicelen`
1149. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnHandlerError$3`
1150. `tryDial$1`
1151. `Start$1`
1152. `write$1`
1153. `init\#2`
1154. `libc\_fchmod\_trampoline`
1155. `SetErrorHandler$1`
1156. `ReuseOrCreateContainer$1`
1157. `init\#2`
1158. `x509\_CFRelease\_trampoline`
1159. `propagateCancel$1`
1160. `sync\_runtime\_procUnpin`
1161. `reflectOffsUnlock`
1162. `mstart\_stub`
1163. `libc\_statfs\_trampoline`
1164. `ParseExtension$1`
1165. `buildRootHuffmanNode`
1166. `TestRateLimiter\_ConcurrentAccess$1`
1167. `lazyInit$1`
1168. `libc\_getcwd\_trampoline`
1169. `osinit\_hack\_trampoline`
1170. `addMultiCallback$1`
1171. `keepalive$1`
1172. `onShutdownTimer$bound`
1173. `main`
1174. `aberrantDeriveMessageName$1$1`
1175. `preprintpanics$1`
1176. `HandleStreams$2`
1177. `parseBinaryExpr$1`
1178. `servePeer$1`
1179. `init\#1`
1180. `defPredeclaredNil`
1181. `RunParallel$1`
1182. `encodeStringer$1`
1183. `init\#1`
1184. `FixedZone$1`
1185. `UnlockOSThread`
1186. `restoreSIGSYS`
1187. `setDefaultUnameProvider`
1188. `runExample$2`
1189. `init\#1`
1190. `RecvMsg$1`
1191. `libc\_fchownat\_trampoline`
1192. `panicunsafestringnilptr`
1193. `ReadMemStats$1`
1194. `copyTrailersToHandlerRequest$bound`
1195. `init\#1`
1196. `libc\_getuid\_trampoline`
1197. `libc\_futimes\_trampoline`
1198. `init\#7`
1199. `handshakeContext$1`
1200. `printDebugLogImpl`
1201. `TestMust$2$1`
1202. `Shutdown$1`
1203. `parseCpuList`
1204. `flush`
1205. `init\#1`
1206. `initMsgChan$1`
1207. `Connection$1`
1208. `SetDelegate$1`
1209. `runtime\_procUnpin`
1210. `mutateBytes$1`
1211. `libc\_chflags\_trampoline`
1212. `traceReadyForQuery$1`
1213. `getcp1257$1`
1214. `Getsockopt$1`
1215. `oneNewExtraM`
1216. `file\_google\_protobuf\_field\_mask\_proto\_rawDescGZIP$1`
1217. `serve$1`
1218. `issetugid\_trampoline`
1219. `panicmakeslicecap`
1220. `sigtramp`
1221. `libc\_close\_trampoline`
1222. `main`
1223. `GetValidator$1`
1224. `closeWrite$1`
1225. `SetDefaultGOMAXPROCS`
1226. `init\#4`
1227. `Add$1`
1228. `init$1`
1229. `main`
1230. `fuzz$2`
1231. `typeDecl$1`
1232. `init\#1`
1233. `serveStreams$2$1`
1234. `PluginInstall$1$1`
1235. `printlock`
1236. `init\#1`
1237. `file\_opentelemetry\_proto\_collector\_logs\_v1\_logs\_service\_proto\_rawDescGZIP$1`
1238. `readHosts`
1239. `RecordSet$1`
1240. `search$2`
1241. `file\_google\_protobuf\_timestamp\_proto\_init`
1242. `structv$1`
1243. `file\_opentelemetry\_proto\_resource\_v1\_resource\_proto\_init`
1244. `init\#1`
1245. `runFinalizers`
1246. `init\#11`
1247. `libc\_fdopendir\_trampoline`
1248. `Add$1`
1249. `AfterFunc$1$1`
1250. `appendValue$1`
1251. `exitsyscall$1`
1252. `defPredeclaredFuncs`
1253. `flushLine$1`
1254. `safeCall$1`
1255. `releaseThread`
1256. `libc\_msync\_trampoline`
1257. `main`
1258. `Never$1`
1259. `init\#2`
1260. `main`
1261. `destroy$1`
1262. `prepareFreeWorkbufs`
1263. `walk$1`
1264. `exitsyscall$4`
1265. `readFrames$1`
1266. `Run$1`
1267. `ReplaceFileTarWrapper$1`
1268. `\_LostContendedRuntimeLock`
1269. `panicSimdImm`
1270. `doRetryNotify\[struct{}\]$1`
1271. `refreshServerDate`
1272. `createContext$7`
1273. `init\#1`
1274. `checkfds`
1275. `ResetWithOptions$1`
1276. `OnPut$1$1`
1277. `mspinning`
1278. `alloc$1`
1279. `checkGenericIsExpected`
1280. `racegoend`
1281. `runCmdContext$2`
1282. `libc\_ptrace\_trampoline`
1283. `init\#1`
1284. `init\#1`
1285. `Write$1`
1286. `init\#1`
1287. `initCategoryAliases`
1288. `createfing`
1289. `lazyInit$1`
1290. `isZeroValue$1`
1291. `file\_opentelemetry\_proto\_resource\_v1\_resource\_proto\_rawDescGZIP$1`
1292. `pthread\_cond\_timedwait\_relative\_np\_trampoline`
1293. `doInRoot$1`
1294. `main`
1295. `gcWriteBarrier1`
1296. `connect$bound`
1297. `yaml\_parser\_fetch\_next\_token$1`
1298. `doRecordGoroutineProfile$1`
1299. `initDefaultMap`
1300. `init$13`
1301. `executeMultiSlot$2`
1302. `processHandoffRequest$2`
1303. `raiseproc\_trampoline`
1304. `ProjectRoot$1`
1305. `Prepare$1`
1306. `RegisterMetricsServiceHandlerFromEndpoint$1$1`
1307. `TestRabbitMQAdapter\_ConsumerLoop\_NormalizesConcurrency$1`
1308. `Shutdown$1`
1309. `readGCStats$1`
1310. `sweepone$1`
1311. `libc\_wait4\_trampoline`
1312. `build$bound`
1313. `init\#1`
1314. `Close$bound`
1315. `init\#8`
1316. `libc\_linkat\_trampoline`
1317. `handleRSTStream$1`
1318. `runtime\_BeforeExec`
1319. `defaultGOMAXPROCSUpdateGODEBUG`
1320. `DialContext$1`
1321. `unmarshalFull$1$1`
1322. `commitAttemptLocked$bound`
1323. `parseParameterList$1`
1324. `Do$1`
1325. `finish$bound`
1326. `pthread\_create\_trampoline`
1327. `initAlgAES`
1328. `debugCallWrap$1`
1329. `resolve$1`
1330. `Disable`
1331. `build$bound`
1332. `finishsweep\_m`
1333. `probe$bound`
1334. `fips140\_setBypass`
1335. `Resolve$1`
1336. `buildRootHuffmanNode`
1337. `getConn$1`
1338. `panicfloat`
1339. `init\#1`
1340. `maybeRunAsync$1`
1341. `libc\_mlock\_trampoline`
1342. `addOption$1`
1343. `validVarType$1`
1344. `main`
1345. `file\_google\_protobuf\_duration\_proto\_rawDescGZIP$1`
1346. `init\#1`
1347. `CloseSend$2`
1348. `init\#1`
1349. `gcMarkTermination$2`
1350. `init\#3`
1351. `main`
1352. `InitMongoDBExternal$1`
1353. `CopyFileWithTar$2`
1354. `lockOSThread`
1355. `racefini`
1356. `Parse$1`
1357. `refill$1`
1358. `wirep$1`
1359. `pthread\_attr\_init\_trampoline`
1360. `libc\_link\_trampoline`
1361. `checkInNoEvent$1`
1362. `init\#1`
1363. `processSingleResponse$1`
1364. `Record$1`
1365. `OnceValue\[string\]$1$1`
1366. `fRunner$2`
1367. `readDataFrame$1`
1368. `libc\_closedir\_trampoline`
1369. `connect$1`
1370. `init\#1`
1371. `ensureSigM`
1372. `threadRun$1`
1373. `printsp`
1374. `collectFileInfoForChanges$2`
1375. `registerBasics`
1376. `initServerWorkers$1`
1377. `processDelivery$1`
1378. `traceAdvance$5`
1379. `TestQueryPluginCRMCollectionWithFilters\_NoFilters$1`
1380. `init\#1`
1381. `init\#1`
1382. `connect$2`
1383. `defPredeclaredConsts`
1384. `startAlarm$1`
1385. `p384B$1`
1386. `CleanupContainer$1`
1387. `file\_grpc\_binlog\_v1\_binarylog\_proto\_init`
1388. `TestRabbitMQAdapter\_Shutdown\_ContextCanceledDuringWait$1`
1389. `Stop$1`
1390. `init\#2`
1391. `init\#1`
1392. `buildRecompMap`
1393. `init\#1`
1394. `init\#1`
1395. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P256Point\]\]$1$1$1`
1396. `init\#1`
1397. `goroutineProfileWithLabelsSync$4$1`
1398. `Shutdown$1$1`
1399. `initOptions`
1400. `synctestRun$1`
1401. `init\#1`
1402. `signature$1`
1403. `libc\_bind\_trampoline`
1404. `panicoverflow`
1405. `Finish$bound`
1406. `needAndBindM`
1407. `init\#1`
1408. `ParallelContainers$1`
1409. `updateProxyResolverState$1`
1410. `cancel$1`
1411. `SkipIfProviderIsNotHealthy$1`
1412. `dialCtx$1`
1413. `init\#1`
1414. `goschedguarded`
1415. `finish$bound`
1416. `disableInfinityTS`
1417. `fips140\_unsetBypass`
1418. `emit$1`
1419. `init\#1`
1420. `newPortForwarder$1`
1421. `newClientStream$1`
1422. `ExitIdle$1`
1423. `init\#1`
1424. `init\#1`
1425. `Read$1`
1426. `init\#2`
1427. `Peek$1`
1428. `init\#1`
1429. `init\#1`
1430. `freeSomeWbufs$1`
1431. `TestRabbitMQAdapter\_IsHealthy\_ReturnsFalse\_WhenChannelClosed$1`
1432. `getcp1258$1`
1433. `main`
1434. `find$1`
1435. `Add$1`
1436. `sysmon`
1437. `doBlockingWithCtx\[\[\]vendor/golang.org/x/net/dns/dnsmessage.Resource\]$1`
1438. `syscall\_runtime\_AfterExec`
1439. `x509\_SecTrustEvaluateWithError\_trampoline`
1440. `clientHandshake$1`
1441. `traceBind$1`
1442. `Parse$1`
1443. `Serve$3`
1444. `libc\_setregid\_trampoline`
1445. `Close$1`
1446. `read$1`
1447. `gcMarkTermination$3`
1448. `RecordSet$1`
1449. `Shutdown$1`
1450. `crash`
1451. `asyncIsExpired$1`
1452. `secureEnv`
1453. `init$1`
1454. `copyTrailers$bound`
1455. `StmtContext$1`
1456. `pthread\_setspecific\_trampoline`
1457. `keepalive$1`
1458. `minimize$1`
1459. `runtime\_BeforeFork`
1460. `FindLongestMatch$1`
1461. `TestExternalDataSource\_CloseConnection\_NilConnection$1$1`
1462. `x509\_CFDataGetBytePtr\_trampoline`
1463. `syscallN\_trampoline`
1464. `term$1`
1465. `Pull$1$2`
1466. `init\#1`
1467. `TestRabbitMQAdapter\_ProcessDelivery\_NackError$1`
1468. `libc\_chdir\_trampoline`
1469. `DisableRandPool`
1470. `onWriteTimeout$bound`
1471. `Has$1`
1472. `collectExemplars$1`
1473. `Current$1`
1474. `init\#2`
1475. `traceCommandComplete$1`
1476. `Parse$1`
1477. `libresolv\_res\_9\_nclose\_trampoline`
1478. `OnceValue\[map\[string\]reflect.Value\]$1$1$1`
1479. `libc\_lseek\_trampoline`
1480. `main`
1481. `libc\_ptsname\_r\_trampoline`
1482. `init\#2`
1483. `merge$1`
1484. `init\#1`
1485. `unifyShutdown$1$1`
1486. `init\#5`
1487. `newcoro$1`
1488. `init\#1`
1489. `RecvMsg$1`
1490. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnVersionMismatch$3`
1491. `main`
1492. `objDecl$2`
1493. `ensureSigM$1`
1494. `file\_opentelemetry\_proto\_logs\_v1\_logs\_proto\_init`
1495. `queuedNewConn$2$1`
1496. `TestRabbitMQAdapter\_ProducerDefault\_ReturnsError\_WhenCircuitOpen$1`
1497. `badmorestackgsignal`
1498. `New\[\*crypto/internal/fips140/sha512.Digest\]$1$1`
1499. `libc\_truncate\_trampoline`
1500. `entersyscallblock$2`
1501. `doBlockingWithCtx\[\[\]string\]$1`
1502. `EvaluateConstValue$1`
1503. `Consume$1`
1504. `initOnce$1`
1505. `watchCancel$2`
1506. `ParseFile$1`
1507. `interfaceEqual$1`
1508. `gcMarkTermination$4$1`
1509. `libc\_posix\_openpt\_trampoline`
1510. `init\#1`
1511. `badunlockosthread`
1512. `closeStream$1`
1513. `newClientStreamWithParams$1`
1514. `queryDC$2`
1515. `pthread\_attr\_getstacksize\_trampoline`
1516. `init\#1`
1517. `main`
1518. `distTmpl$1`
1519. `roundTrip$1`
1520. `Check`
1521. `init\#19`
1522. `IncNonDefault$bound`
1523. `RecordMetrics$1`
1524. `Shutdown$1`
1525. `main`
1526. `initAll`
1527. `goyield`
1528. `exitsyscall`
1529. `libc\_rmdir\_trampoline`
1530. `doHTTPConnectHandshake$1`
1531. `mstart`
1532. `init\#1`
1533. `init\#1`
1534. `atom$1`
1535. `init\#2`
1536. `BoringCrypto`
1537. `endCheckmarks`
1538. `RecordSet$1`
1539. `initSecureMode`
1540. `init$1`
1541. `file\_google\_protobuf\_struct\_proto\_init`
1542. `publicationBarrier`
1543. `badmorestackg0`
1544. `operateHeaders$5`
1545. `mayMoreStackMove`
1546. `init\#6`
1547. `ResetMaxTraceEntryToDefault`
1548. `append$1`
1549. `executeParallel$2`
1550. `WithRecover$1$1`
1551. `init\#1`
1552. `StopTrace`
1553. `runWithClient$1`
1554. `TestWithSwaggerEnvConfig\_EmptyEnvVars$1`
1555. `getcp949$1`
1556. `Ping$1`
1557. `onReadIdleTimer$bound`
1558. `finalClose$2`
1559. `getcp1256$1`
1560. `badctxt`
1561. `badsystemstack`
1562. `debugOptions`
1563. `libc\_chroot\_trampoline`
1564. `bootstrapRandReseed`
1565. `init\#4`
1566. `NewFunc$1`
1567. `DefaultEncoder$1`
1568. `traceAdvance$4`
1569. `init\#1`
1570. `TimeoutWithCodeHandler$1$1`
1571. `libc\_openat\_trampoline`
1572. `badreflectcall`
1573. `OnceFunc$1$1`
1574. `synctestWait`
1575. `New\[\*crypto/internal/fips140/sha256.Digest\]$1`
1576. `init\#1`
1577. `goexit1`
1578. `AddSet$1`
1579. `TestRabbitMQAdapter\_ProcessDelivery\_RecoverFromPanic$1`
1580. `stacklessWriteDeflate$1`
1581. `loop`
1582. `Record$1`
1583. `structv$1`
1584. `nextFrame$1`
1585. `Add$1`
1586. `set$1`
1587. `End$bound`
1588. `genericExprList$1`
1589. `Value$1`
1590. `operateHeaders$3`
1591. `OnceValue\[encoding/json.encoderFunc\]$1$1`
1592. `traceThreadDestroy$1`
1593. `casgstatus$2`
1594. `libc\_unlink\_trampoline`
1595. `init\#18`
1596. `libc\_fchmodat\_trampoline`
1597. `Run$1`
1598. `WriteOverlays$2`
1599. `sigpipe`
1600. `shutdown$1`
1601. `TestScanRows$3$1`
1602. `libresolv\_res\_9\_nsearch\_trampoline`
1603. `RoundTrip$1`
1604. `libc\_sysconf\_trampoline`
1605. `gcStart$4`
1606. `processSRVResults$1`
1607. `runConcurrent$1`
1608. `printCountProfile$2`
1609. `gcWriteBarrier7`
1610. `startBackgroundRead$bound`
1611. `readContent$1`
1612. `file\_google\_rpc\_status\_proto\_rawDescGZIP$1`
1613. `invokeStringer$1`
1614. `update$1`
1615. `sigprocmask\_trampoline`
1616. `Release$1`
1617. `pthread\_cond\_init\_trampoline`
1618. `init$bound`
1619. `Token$1`
1620. `main`
1621. `wbBufFlush`
1622. `init\#1`
1623. `file\_google\_protobuf\_wrappers\_proto\_init`
1624. `Flush$1`
1625. `main`
1626. `PrintDefaults`
1627. `main`
1628. `osyield`
1629. `Close$1`
1630. `DumpRequestOut$2`
1631. `Compose$1`
1632. `init\#1`
1633. `FiberWrapHandler$1$1`
1634. `run1$1$1`
1635. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnInvalidSignature$3`
1636. `flush$1`
1637. `onSettingsTimer$bound`
1638. `buildRecompMap`
1639. `initialize$bound`
1640. `NewFastHTTPHandler$1$1$1`
1641. `Close$1`
1642. `init\#1`
1643. `SignalNum$1`
1644. `ClearArrayValidations$1`
1645. `libc\_setsid\_trampoline`
1646. `queuedNewConn$1`
1647. `connect$1`
1648. `Close$1`
1649. `initPredefined$1`
1650. `entersyscallblock$4`
1651. `dispatchDeliveries$1$1`
1652. `fixedHuffmanDecoderInit`
1653. `init\#1`
1654. `queuedNewConn$2$2`
1655. `libc\_listen\_trampoline`
1656. `Add$1`
1657. `Find$1`
1658. `runCmdContext$3`
1659. `finalClose$1`
1660. `fprint$1`
1661. `defPredeclaredTypes`
1662. `trace$1`
1663. `init\#1`
1664. `libc\_connect\_trampoline`
1665. `checkTimeouts`
1666. `synctestRun$3`
1667. `pthread\_mutex\_init\_trampoline`
1668. `pinConnectionToCursor$bound`
1669. `ForEachPackage$1`
1670. `decap$2`
1671. `doCall$2$1`
1672. `checkMinIdleConns$1$1`
1673. `ConsumeWithContext$1`
1674. `getDockerAuthConfigs$3`
1675. `init\#1`
1676. `vgetrandomInit`
1677. `onceSetNextProtoDefaults$bound`
1678. `New\[hash.Hash\]$1`
1679. `runtime\_debug\_WriteHeapDump$1`
1680. `init\#1`
1681. `init$6`
1682. `panicunsafestringlen`
1683. `MarshalAppendWithContext$1`
1684. `libc\_readlinkat\_trampoline`
1685. `file\_google\_protobuf\_any\_proto\_rawDescGZIP$1`
1686. `libc\_settimeofday\_trampoline`
1687. `init\#1`
1688. `startGracefulShutdown$bound`
1689. `reader$1`
1690. `isAbstractSocketExists$1`
1691. `Run$1`
1692. `legacyContainerWait$1`
1693. `rollback$1`
1694. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnMissingSignature$3`
1695. `setDITDisabled`
1696. `init\#5`
1697. `ShouldPanic$1`
1698. `file\_google\_protobuf\_any\_proto\_init`
1699. `syscall\_runtime\_AfterFork`
1700. `panicExtend`
1701. `interfaceType$2`
1702. `init$1$1`
1703. `setGCPercent$1`
1704. `libc\_dup2\_trampoline`
1705. `getcp1253$1`
1706. `libc\_select\_trampoline`
1707. `getDefaultMetricsFactory$1`
1708. `sendBatchExtendedWithDescription$2$1`
1709. `hasVarSize$1`
1710. `stdin$1`
1711. `freeDeadSpanSPMCs`
1712. `decap$1`
1713. `init\#5`
1714. `init\#1`
1715. `RegisterTraceServiceHandlerFromEndpoint$1$1`
1716. `Parse$1`
1717. `GC`
1718. `resolveUnderlying$1`
1719. `collectMethods$1`
1720. `entersyscallblock$5`
1721. `init$1`
1722. `main`
1723. `nextBlock$1$2$1`
1724. `x509\_SecTrustSetVerifyDate\_trampoline`
1725. `newReaper$2`
1726. `close$bound`
1727. `printnl`
1728. `checkGenericIsExpected`
1729. `writeBodyStream$1`
1730. `entersyscallblock$1`
1731. `HandleStreams$2`
1732. `EnableRandPool`
1733. `Record$1`
1734. `libc\_getaddrinfo\_trampoline`
1735. `lazyInit$1`
1736. `commitAttemptLocked$bound`
1737. `readLoop$1`
1738. `panicshift`
1739. `MustExtractDockerSocket$1`
1740. `update$2`
1741. `NewPeriodicReader$2$1`
1742. `AddSet$1`
1743. `startStreamDecoder$2`
1744. `fcntl\_trampoline`
1745. `lazyInit$1`
1746. `run$1`
1747. `walltime\_trampoline`
1748. `panicmem`
1749. `Add$1`
1750. `ResetAssertionMetrics`
1751. `init\#1`
1752. `execDC$1`
1753. `TestScanRows$1$1`
1754. `closeReqBodyLocked$1`
1755. `newSession$1`
1756. `metricsUnlock`
1757. `init\#1`
1758. `initRequestHandler$bound`
1759. `moduledataverify`
1760. `init\#2`
1761. `watchMaster`
1762. `HandleStreams$1`
1763. `osinit`
1764. `gcStart$2`
1765. `processSubAppsRoutes$1`
1766. `RegisterLogsServiceHandlerFromEndpoint$1`
1767. `unreachableMethod`
1768. `writeStatus$1`
1769. `osInit`
1770. `syscall\_x509`
1771. `Add$1`
1772. `initMimeUnix`
1773. `start$1`
1774. `init\#1`
1775. `init\#2`
1776. `mProf\_PostSweep`
1777. `dialCtx$2`
1778. `dropm`
1779. `gcWriteBarrier4`
1780. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P521Point\]\]$1$1`
1781. `check`
1782. `init\#1`
1783. `AddSet$1`
1784. `unregisterAllDrivers`
1785. `wbBufFlush$1`
1786. `lazyInit$1`
1787. `init$13`
1788. `init\#2`
1789. `TestRabbitMQAdapter\_Shutdown\_WaitsForConsumers$1`
1790. `aggregate$1`
1791. `TestRabbitMQAdapter\_ProcessDelivery\_HandlesNonStringHeaderID$3`
1792. `init\#3`
1793. `run$1`
1794. `dumpparams`
1795. `metricsLock`
1796. `getcp1250$1`
1797. `setPossiblyUnhashableKey$1`
1798. `attemptArgMatch$1`
1799. `OnceValue\[\[\]string\]$1$1$1`
1800. `libc\_arc4random\_buf\_trampoline`
1801. `stkobjinit`
1802. `runHandler$1`
1803. `init\#1`
1804. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnMissingSignature$1`
1805. `init\#5`
1806. `runHandlers`
1807. `init\#1`
1808. `reentersyscall$3`
1809. `main`
1810. `init\#1`
1811. `operateHeaders$4`
1812. `onceSetNextProtoDefaults$bound`
1813. `wirep$2`
1814. `gcPrepareMarkRoots`
1815. `Record$1`
1816. `stackinit`
1817. `init\#1`
1818. `newClientStreamWithParams$4`
1819. `file\_opentelemetry\_proto\_metrics\_v1\_metrics\_proto\_init`
1820. `fuzz$1`
1821. `processUnaryRPC$1`
1822. `hostLookupOrder$1`
1823. `initConnPool$bound`
1824. `dumpobjs`
1825. `cleanup$1`
1826. `init\#1`
1827. `goroutineLeakGC`
1828. `nop`
1829. `StartWithGracefulShutdown$1`
1830. `parse$1`
1831. `templateThread`
1832. `dumproots`
1833. `launch$1`
1834. `libc\_kqueue\_trampoline`
1835. `New\[hash.Hash\]$1$1`
1836. `newproc$1`
1837. `scan$3$1`
1838. `init\#4`
1839. `file\_google\_protobuf\_struct\_proto\_rawDescGZIP$1`
1840. `logCloseHangDebugInfo$bound`
1841. `doRetryNotify\[\*github.com/testcontainers/testcontainers-go.DockerContainer\]$1`
1842. `libc\_munmap\_trampoline`
1843. `connect$2`
1844. `worker$2`
1845. `Alignof$1`
1846. `ExportPair$2`
1847. `main`
1848. `Execute$1`
1849. `TestScanColumns$3$1`
1850. `interrupt`
1851. `libc\_getpid\_trampoline`
1852. `init\#1`
1853. `Add$1`
1854. `Do$1`
1855. `init\#1`
1856. `minit`
1857. `handleSettings$1$1`
1858. `Breakpoint`
1859. `closeStream$1`
1860. `Free$bound`
1861. `SnapshotCoverage`
1862. `setCommandValueReflection$1`
1863. `log$1`
1864. `SafeGo$1`
1865. `NewClient$1`
1866. `lsandoleakcheck`
1867. `Setenv$2`
1868. `handleForwards$bound`
1869. `deferreturn`
1870. `x509\_CFStringCreateWithBytes\_trampoline`
1871. `TestRabbitMQAdapter\_ProcessDelivery\_AckError$1`
1872. `HandleCancel$1`
1873. `Collect$1`
1874. `getcp932$1`
1875. `runCmdContext$1`
1876. `open$1`
1877. `gcWriteBarrier2`
1878. `traceLockInit`
1879. `init\#2`
1880. `defaultUsage$bound`
1881. `ClearCache`
1882. `main`
1883. `keccakF1600Generic$1`
1884. `unpack$1`
1885. `init\#1`
1886. `initPredefined`
1887. `init\#1`
1888. `SendMsg$1`
1889. `libc\_sendfile\_trampoline`
1890. `mProf\_NextCycle`
1891. `init\#1`
1892. `init\#1`
1893. `osinit\_hack`
1894. `file\_google\_protobuf\_field\_mask\_proto\_init`
1895. `init\#2`
1896. `file\_grpc\_binlog\_v1\_binarylog\_proto\_rawDescGZIP$1`
1897. `exitsyscall$3`
1898. `SetTextMapPropagator$1`
1899. `doBlockingWithCtx$1`
1900. `libc\_getgrnam\_r\_trampoline`
1901. `InitializeTelemetry$1`
1902. `gcAssistAlloc$1`
1903. `forEachP$1`
1904. `traceAdvance$6`
1905. `WithDeadlineCause$2`
1906. `New$1$1`
1907. `doinit`
1908. `NewController$1`
1909. `dumpitabs`
1910. `getCaller$1`
1911. `panicmakeslicelen`
1912. `init\#2`
1913. `init$1`
1914. `startFlushGoroutine$1`
1915. `libc\_getpwnam\_r\_trampoline`
1916. `init\#1`
1917. `readARM64Registers`
1918. `init\#1`
1919. `processDelivery$1`
1920. `init\#1`
1921. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P384Point\]\]$1$1`
1922. `TestExtractExternalData\_SuccessWithNonFatalWarnings$1`
1923. `OnceValues\[map\[string\]rune map\[string\]\[2\]rune\]$1$1`
1924. `connect$1`
1925. `init\#1`
1926. `AddSet$1`
1927. `init$1`
1928. `libc\_setprivexec\_trampoline`
1929. `init\#15`
1930. `block`
1931. `sigpipe`
1932. `init\#2`
1933. `allPackages$1$1`
1934. `onSettingsTimer$bound`
1935. `TryGo$1$1`
1936. `asyncPreempt2`
1937. `poll\_runtime\_pollServerInit`
1938. `setMinimalFeatures`
1939. `Go$1$1`
1940. `libc\_getgroups\_trampoline`
1941. `consumeAddrSpec$1`
1942. `confirmLocks$1`
1943. `Goexit`
1944. `libc\_mknod\_trampoline`
1945. `log$1`
1946. `\_LostSIGPROFDuringAtomic64`
1947. `x509\_SecTrustCreateWithCertificates\_trampoline`
1948. `resolve$1`
1949. `x509\_SecTrustCopyCertificateChain\_trampoline`
1950. `pthread\_mutex\_lock\_trampoline`
1951. `goroutineProfileWithLabelsConcurrent$1`
1952. `newstack`
1953. `checkResponseErr$1`
1954. `subscribedState$1`
1955. `scan$4$1`
1956. `libc\_mlockall\_trampoline`
1957. `init$2`
1958. `RecordNonApproved`
1959. `New\[\*crypto/internal/fips140/sha256.Digest\]$1$1`
1960. `start$1`
1961. `SendMsg$2`
1962. `wakep`
1963. `init\#1`
1964. `entersyscallblock`
1965. `maybeRunChan$1`
1966. `encap$1`
1967. `traceExitingSyscall`
1968. `mstart1`
1969. `init\#1`
1970. `RegisterFunc$4$5`
1971. `tRunner$1$1`
1972. `New$1`
1973. `clearpools`
1974. `init\#1`
1975. `OnceValue$1$1`
1976. `three$1`
1977. `selectgo$3`
1978. `init\#1`
1979. `x509\_CFDictionaryGetValueIfPresent\_trampoline`
1980. `startGracefulShutdown$1`
1981. `bgRead$1`
1982. `goargs`
1983. `libc\_gettimeofday\_trampoline`
1984. `runPerThreadSyscall`
1985. `gcBgMarkPrepare`
1986. `merge$2`
1987. `init\#3`
1988. `log`
1989. `TestSetConfigFromEnvVars$1$1`
1990. `tunnel$1`
1991. `handshakeContext$2`
1992. `init$6`
1993. `gcWakeAllAssists`
1994. `startChannelWatcher$1`
1995. `typeDecl$2`
1996. `fatalpanic$2`
1997. `NewStreamReader$1`
1998. `ClientHandshake$1`
1999. `init\#1`
2000. `libc\_pipe\_trampoline`
2001. `CoordinateFuzzing$3`
2002. `Disable`
2003. `MasterAddr$1$1`
2004. `init\#8`
2005. `runtime\_doSpin`
2006. `Sync$1$1`
2007. `pthread\_cond\_signal\_trampoline`
2008. `main`
2009. `set\_crosscall2`
2010. `EndTracingSpans$1`
2011. `onDemandWorker$1`
2012. `doInRoot\[int\]$1`
2013. `PluginInstall$1`
2014. `DialContext$2`
2015. `runPiped$1`
2016. `libc\_unlockpt\_trampoline`
2017. `init\#1`
2018. `frameSkip$1`
2019. `lazyInit$1`
2020. `GetValue$1`
2021. `RegisterAsyncReporter$1`
2022. `initFeistelBox`
2023. `ClearNumberValidations$1`
2024. `assertWorldStopped`
2025. `shutdownWorkers$1`
2026. `libc\_getpgrp\_trampoline`
2027. `libc\_lstat\_trampoline`
2028. `DecodeFull$1`
2029. `main`
2030. `libc\_mprotect\_trampoline`
2031. `signalWaitUntilIdle`
2032. `close\_trampoline`
2033. `Execute$1`
2034. `run$4`
2035. `RecordSet$1`
2036. `init\#1`
2037. `abortStreamLocked$1`
2038. `TestQueryPluginCRM\_WithFilters$1`
2039. `initSystemRoots`
2040. `init\#5`
2041. `connStmt$1`
2042. `init\#2`
2043. `close$1`
2044. `init\#9`
2045. `handleLock$1`
2046. `runtime\_debug\_freeOSMemory`
2047. `nextBlock$1$2`
2048. `loadPackageNames$1`
2049. `SaveMultipartFile$2`
2050. `init\#1`
2051. `setupFallbackCache$1`
2052. `scheduleNextConnectionLocked$1`
2053. `startCheckmarks`
2054. `onReadTimeout$bound`
2055. `libc\_unlinkat\_trampoline`
2056. `AppendCertsFromPEM$1$1`
2057. `LoadLocation$1`
2058. `SetFallbackRoots$1`
2059. `handleSettings$1$1`
2060. `computeInterfaceTypeSet$2$1`
2061. `getcp1252$1`
2062. `shutdown$1`
2063. `init\#14`
2064. `search$1`
2065. `libc\_fsync\_trampoline`
2066. `TryAcquire\[\*github.com/jackc/pgx/v5/pgxpool.connResource\]$1`
2067. `\_LostExternalCode`
2068. `init$3`
2069. `probe$bound`
2070. `pthread\_cond\_wait\_trampoline`
2071. `mlock\_trampoline`
2072. `main`
2073. `noteUnusedDriverStatement$1`
2074. `syscall\_runtime\_BeforeFork`
2075. `Add$1`
2076. `main`
2077. `readServices`
2078. `getOrQueueForIdleConn$1`
2079. `libc\_symlinkat\_trampoline`
2080. `libc\_fstatfs\_trampoline`
2081. `SetFinalizer$2`
2082. `file\_google\_rpc\_status\_proto\_init`
2083. `init\#1`
2084. `main`
2085. `finishDebugVarsSetup`
2086. `setBypass`
2087. `execDC$3`
2088. `HandleStreams$1`
2089. `persistentalloc$1`
2090. `copyTrailersToHandlerRequest$bound`
2091. `Format$1`
2092. `clearSignalHandlers`
2093. `initAllChan$1`
2094. `Consume$1`
2095. `lazyInit$1`
2096. `initP521`
2097. `instantiateSignature$2`
2098. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$6`
2099. `duffzero`
2100. `ServeHTTP$1`
2101. `file\_google\_api\_httpbody\_proto\_rawDescGZIP$1`
2102. `initP384`
2103. `usleep\_trampoline`
2104. `init\#1`
2105. `gcMarkDone$3`
2106. `libc\_setlogin\_trampoline`
2107. `TraceQueryute$1`
2108. `dialSingle$1`
2109. `init\#1`
2110. `grabConn$1`
2111. `SetLoggerProvider$1`
2112. `computeInterfaceTypeSet$1`
2113. `NewPublicKey$1`
2114. `init\#1`
2115. `libc\_fchflags\_trampoline`
2116. `doBlockingWithCtx\[\[\]net.IPAddr\]$1`
2117. `stopm`
2118. `init$4`
2119. `getcp437$1`
2120. `modify$1`
2121. `processUnaryRPC$2`
2122. `traceQuery$1`
2123. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P224Point\]\]$1$1`
2124. `newClientStream$5`
2125. `Run$1`
2126. `getMacOSMajor$1`
2127. `SetTracerProvider$1`
2128. `nextBlock$1$1`
2129. `init\#1`
2130. `panicUnaligned`
2131. `init\#1`
2132. `Sync$1`
2133. `main`
2134. `growSlice$1`
2135. `init\#1`
2136. `Setenv$1`
2137. `TestWithSwaggerEnvConfig\_WithEnvVars$1`
2138. `newNonRetryClientStream$1`
2139. `gfget$2`
2140. `onIdleTimeout$bound`
2141. `forceCloseConn$bound`
2142. `parseFieldName$1`
2143. `failWantMap`
2144. `lookupIPAddr$3`
2145. `collectRecv$1`
2146. `closeConnIfStillIdle$bound`
2147. `munmap\_trampoline`
2148. `Init`
2149. `sigaltstack\_trampoline`
2150. `init\#1`
2151. `p224B$1`
2152. `EventuallyWithT$1`
2153. `Shutdown$1`
2154. `forceCloseConn$bound`
2155. `stopTheWorld$1`
2156. `cgoCheckPtrWrite$1`
2157. `init\#1`
2158. `init\#1`
2159. `stdoutHandler$1`
2160. `init\#2`
2161. `open\_trampoline`
2162. `handleCleanCache$1`
2163. `nify$1`
2164. `main`
2165. `TestEnsureConfigFromEnvVars$2$1`
2166. `collect$1`
2167. `gcWriteBarrier8`
2168. `proc\_regionfilename\_trampoline`
2169. `RangeFields$1`
2170. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P224Point\]\]$1$1$1`
2171. `setUserArenaChunkToFault$1`
2172. `TestRabbitMQAdapter\_ConsumerLoop\_AcksOnSuccess$1$1`
2173. `libc\_fstatat\_trampoline`
2174. `x509\_CFErrorGetCode\_trampoline`
2175. `typelinksinit`
2176. `Multiplexed$1$1`
2177. `marshal$1`
2178. `setRequestCancel$4`
2179. `addTLS$2`
2180. `libc\_getpwuid\_r\_trampoline`
2181. `ExportPair$1`
2182. `onReadTimeout$bound`
2183. `Eventually$1`
2184. `x509\_CFErrorCopyDescription\_trampoline`
2185. `awaitGoroutines$1`
2186. `servePeer$2`
2187. `netpollGenericInit`
2188. `Parse`
2189. `AcquireConn$1`
2190. `sigpanic0`
2191. `Stop`
2192. `x509\_CFEqual\_trampoline`
2193. `infer$2`
2194. `healthCheck$1`
2195. `RunConsumers$1`
2196. `main\_main`
2197. `getMetadata$1`
2198. `sync\_runtime\_doSpin`
2199. `validCycle$1`
2200. `funcDecl$1`
2201. `search$2$1`
2202. `init\#1`
2203. `raise\_trampoline`
2204. `handleMonitoredClose$1`
2205. `libc\_flock\_trampoline`
2206. `Close$bound`
2207. `finishStream$1`
2208. `traceStartReadCPU$1`
2209. `defaultCleanUp$1`
2210. `libc\_adjtime\_trampoline`
2211. `traceAdvance$3`
2212. `AddSet$1`
2213. `casgstatus$3`
2214. `readTrace0$1`
2215. `OnEmit$1`
2216. `getPossiblyUnhashableKey$1`
2217. `fixedHuffmanDecoderInit$1`
2218. `init$5`
2219. `ensureMetricsCollector$1`
2220. `defaultGOMAXPROCSInit`
2221. `buildImageWithSecrets$1`
2222. `unspillArgs`
2223. `init\#1`
2224. `osyield\_no\_g`
2225. `file\_opentelemetry\_proto\_collector\_metrics\_v1\_metrics\_service\_proto\_init`
2226. `TestWithSwaggerEnvConfig\_InvalidHost$1`
2227. `unpinConnectionFromTransaction$bound`
2228. `x509\_SecTrustEvaluate\_trampoline`
2229. `Chdir$1`
2230. `runHandler$1`
2231. `CleanupNetwork$1`
2232. `alginit`
2233. `libc\_getsockname\_trampoline`
2234. `libc\_accept\_trampoline`
2235. `overrideUmask$1`
2236. `libc\_mkfifo\_trampoline`
2237. `Clearenv`
2238. `ensureSensitiveFieldsMap$1`
2239. `init$1`
2240. `init\#1`
2241. `init\#1`
2242. `Read$1`
2243. `doBlockingWithCtx\[int\]$1`
2244. `init\#1`
2245. `Close$1`
2246. `RecordSet$1`
2247. `FIPSOnly`
2248. `EncodeAll$1`
2249. `init\#1`
2250. `secretEraseRegisters`
2251. `copyTrailers$bound`
2252. `run$3`
2253. `OrderedMarshal$1`
2254. `init\#10`
2255. `Execute$1`
2256. `file\_google\_protobuf\_timestamp\_proto\_rawDescGZIP$1`
2257. `libc\_seteuid\_trampoline`
2258. `Execute$2`
2259. `generatorTable$1`
2260. `dumpgs`
2261. `parseExpr$1`
2262. `mmap\_trampoline`
2263. `RecordSet$1`
2264. `expandRHS$1`
2265. `nilfunc`
2266. `processPipelineNode$1$1`
2267. `optionsUnmarshaler$1$1`
2268. `OnceValue\[string\]$1$1$1`
2269. `init\#1`
2270. `freeOSMemory`
2271. `decryptKey$1`
2272. `libc\_getegid\_trampoline`
2273. `worker$1`
2274. `libc\_lchown\_trampoline`
2275. `pthread\_key\_create\_trampoline`
2276. `debugCallV2`
2277. `main`
2278. `mergeRuneSets$1`
2279. `main`
2280. `dispatchDeliveries$1`
2281. `lazyInit$1`
2282. `init\#3`
2283. `init\#3`
2284. `mPark`
2285. `libc\_execve\_trampoline`
2286. `init\#1`
2287. `lockProfiles`
2288. `badcgocallback`
2289. `debugCallCheck$1`
2290. `dolockOSThread`
2291. `newHealthData$1`
2292. `startServers$3`
2293. `asyncClose$1`
2294. `libc\_fchdir\_trampoline`
2295. `startWatcher$1`
2296. `gcMarkRootCheck`
2297. `signalWaitUntilIdle`
2298. `updateServerDate$1`
2299. `init$1$1`
2300. `racefingo`
2301. `WithDataIndependentTiming$1`
2302. `instantiateSignature$1`
2303. `init\#1`
2304. `onIdleTimer$bound`
2305. `dialConn$3`
2306. `Add$1`
2307. `init\#1`
2308. `morestack\_noctxt`
2309. `enableWER`
2310. `initialize$1`
2311. `gcMarkDone$4`
2312. `libc\_getpriority\_trampoline`
2313. `updateServerDate`
2314. `rt0\_go`
2315. `x509\_CFDateCreate\_trampoline`
2316. `init$1`
2317. `init\#1`
2318. `init\#1`
2319. `Wait`
2320. `buildCommonHeaderMapsOnce`
2321. `allocmcache$1`
2322. `traceDataRow$1`
2323. `buildImageWithSecrets$1`
2324. `Raw$1`
2325. `OnceFunc$1`
2326. `Close$1`
2327. `newCompressedFSFileCache$1`
2328. `releaseForkLock`
2329. `\*github.com/LerianStudio/lib-commons/v4/commons/mongo.Client.Client`
2330. `\*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.Client`
2331. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockmongoDatabaseProvider.Client`
2332. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.stubMongoDatabaseProvider.Client`
2333. `github.com/LerianStudio/fetcher/pkg/mongodb/product.stubMongoDatabaseProvider.Client`
2334. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockmongoDatabaseProvider.Client`
2335. `\*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.Client`

#### `TestConnectionMongoDBRepository\_Create`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`
5. `\*testing.T.Run`

#### `newConnectionRepository`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (30 direct callers)

**Direct Callers (signature change affects these):**

1. `TestConnectionMongoDBRepository\_FindByOrganizationAndDatabaseName$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:522`
2. `TestConnectionMongoDBRepository\_EnsureIndexes$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:722`
3. `TestConnectionMongoDBRepository\_DropIndexes$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:760`
4. `TestConnectionMongoDBRepository\_FindByConfigNames$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:644`
5. `TestConnectionMongoDBRepository\_Delete$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:379`
6. `TestConnectionMongoDBRepository\_Update$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:272`
7. `TestConnectionMongoDBRepository\_FindByConfigNames$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:655`
8. `TestConnectionMongoDBRepository\_Update$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:259`
9. `TestConnectionMongoDBRepository\_Delete$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:357`
10. `TestConnectionMongoDBRepository\_Update$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:296`
11. `TestConnectionMongoDBRepository\_Create$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:167`
12. `TestConnectionMongoDBRepository\_Create$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:198`
13. `TestConnectionMongoDBRepository\_FindByOrganizationAndName$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:471`
14. `TestConnectionMongoDBRepository\_FindByOrganizationAndName$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:483`
15. `TestConnectionMongoDBRepository\_FindByOrganizationAndDatabaseName$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:546`
16. `TestConnectionMongoDBRepository\_Create$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:179`
17. `TestConnectionMongoDBRepository\_FindByConfigNames$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:614`
18. `TestConnectionMongoDBRepository\_EnsureIndexes$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:730`
19. `TestConnectionMongoDBRepository\_FindByConfigNames$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:625`
20. `TestConnectionMongoDBRepository\_FindByID$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:420`
21. `TestConnectionMongoDBRepository\_Create$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:155`
22. `TestConnectionMongoDBRepository\_FindByOrganizationAndDatabaseName$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:534`
23. `TestConnectionMongoDBRepository\_Update$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:316`
24. `TestConnectionMongoDBRepository\_FindByConfigNames$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:678`
25. `TestConnectionMongoDBRepository\_FindByConfigNames$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:585`
26. `TestConnectionMongoDBRepository\_Update$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:234`
27. `TestConnectionMongoDBRepository\_List$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:855`
28. `TestConnectionMongoDBRepository\_FindByID$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:432`
29. `TestConnectionMongoDBRepository\_List$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:822`
30. `TestConnectionMongoDBRepository\_Update$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:284`

**Callees (this function depends on):**

1. `\*testing.common.Helper`
2. `clearConnectionsCollection`
3. `NewConnectionMongoDBRepository`
4. `\*testing.common.Fatalf`
5. `Background`
6. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.EnsureIndexes`
7. `\*testing.common.Fatalf`

#### `clearConnectionsCollection`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (10 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `newConnectionRepository` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:64`
3. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
4. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
5. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
6. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
7. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
8. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
9. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
10. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.common.Helper`
2. `\*testing.common.Fatalf`
3. `Background`
4. `\*github.com/LerianStudio/lib-commons/v4/commons/mongo.Client.Client`
5. `\*testing.common.Fatalf`
6. `ToLower`
7. `\*go.mongodb.org/mongo-driver/mongo.Client.Database`
8. `ToLower`
9. `\*go.mongodb.org/mongo-driver/mongo.Database.Collection`
10. `Background`
11. `\*go.mongodb.org/mongo-driver/mongo.Collection.Drop`
12. `As`
13. `\*testing.common.Fatalf`

#### `TestConnectionMongoDBRepository\_Update`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`
5. `\*testing.T.Run`
6. `\*testing.T.Run`
7. `\*testing.T.Run`

#### `TestConnectionMongoDBRepository\_FindByID`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`

#### `TestConnectionMongoDBRepository\_EnsureIndexes`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`

#### `TestConnectionMongoDBRepository\_Delete`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`

#### `TestConnectionMongoDBRepository\_FindByOrganizationAndDatabaseName`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`

#### `TestConnectionMongoDBRepository\_FindByConfigNames`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`
5. `\*testing.T.Run`
6. `\*testing.T.Run`
7. `\*testing.T.Run`

#### `TestConnectionMongoDBRepository\_DropIndexes`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`

#### `TestConnectionMongoDBRepository\_List`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`

#### `stubConnectionSpanAttributes`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** MEDIUM (3 direct callers)

**Direct Callers (signature change affects these):**

1. `TestConnectionMongoDBRepository\_Create$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:199`
2. `TestProductMongoDBRepository\_Create\_ErrorPaths$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb\_test.go:181`
3. `TestConnectionMongoDBRepository\_Update$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb\_test.go:318`

**Callees (this function depends on):**

1. `\*testing.common.Helper`
2. `\*testing.common.Cleanup`

#### `TestConnectionMongoDBRepository\_FindByOrganizationAndName`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`

#### `TestNewConnectionMongoDBRepository\_NilClient`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewConnectionMongoDBRepository`
2. `\*testing.common.Fatalf`
3. `\*testing.common.Fatalf`

#### `NewJobMongoDBRepository`
**File:** `pkg/mongodb/job/job.mongodb.go`
**Risk Level:** MEDIUM (4 direct callers)

**Direct Callers (signature change affects these):**

1. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:193`
2. `InitWorker` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/worker/internal/bootstrap/config.go:193`
3. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:178`
4. `initMongoRepositories` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/bootstrap/config.go:178`

**Callees (this function depends on):**

1. `New`
2. `Background`
3. `WithTimeout`
4. `init$5`
5. `init$bound`
6. `DefPredeclaredTestFuncs`
7. `init$bound`
8. `Parse$1`
9. `OnceValues\[map\[string\]rune map\[string\]\[2\]rune\]$1$1$1`
10. `\_GC`
11. `init\#2`
12. `libc\_munlock\_trampoline`
13. `init\#3`
14. `init\#12`
15. `init\#1`
16. `logUnexpectedFailure$1`
17. `init\#1`
18. `queryDatabase$1`
19. `initP256`
20. `fuzz$1`
21. `encap$2`
22. `libc\_setpgid\_trampoline`
23. `acquireThread$1`
24. `libc\_setuid\_trampoline`
25. `fRunner$1`
26. `init\#2`
27. `libc\_kill\_trampoline`
28. `main`
29. `arc4random\_buf\_trampoline`
30. `setDefaultRuntimeProviders`
31. `makeSha256Reader$1`
32. `MockEnv$1`
33. `TestRabbitMQAdapter\_ProcessDelivery\_ExtractsHeaders$1$3`
34. `tRunner$1`
35. `cancellationListenerCallback$bound`
36. `addTLS$1`
37. `libc\_faccessat\_trampoline`
38. `SafeGoWithContextAndComponent$1`
39. `pthread\_kill\_trampoline`
40. `watchCancel$1`
41. `x509\_CFStringCreateExternalRepresentation\_trampoline`
42. `readRequest$1`
43. `setitimer\_trampoline`
44. `init\#3`
45. `handleRawConn$1`
46. `stubConnectionSpanAttributes$2`
47. `traceParameterStatus$1`
48. `traceBackendKeyData$1`
49. `onShutdownTimer$bound`
50. `open$1`
51. `TestScanRows$2$1`
52. `dispatchDeliveries$1`
53. `init$1`
54. `debugCallWrap2$1`
55. `TestRabbitMQAdapter\_ProducerDefault\_SkipsSigningWhenDisabled$1`
56. `entersyscallblock$3`
57. `fixedHuffmanDecoderInit`
58. `OnceValue\[\[\]string\]$1$1`
59. `setDefaultOSProviders`
60. `init\#1`
61. `typInternal$2`
62. `setMemoryLimit$1`
63. `snapshotMetricsRegistryForTesting$1`
64. `libc\_fpathconf\_trampoline`
65. `clientGetURLDeadline$1`
66. `startupMessage$5`
67. `init\#1`
68. `Clearenv`
69. `init\#1`
70. `init\#1`
71. `startHealthCheck$4`
72. `HostGatewayIP$1`
73. `newClientStreamWithParams$3`
74. `libc\_kevent\_trampoline`
75. `ReadFile$1`
76. `sharedMemTempFile$1`
77. `libc\_pread\_trampoline`
78. `init\#3`
79. `reentersyscall$1`
80. `init\#1`
81. `eventsTmpl$1`
82. `Compile$1`
83. `main`
84. `getGCMaskOnDemand$1`
85. `init\#9`
86. `newTypeObject$1`
87. `init\#1`
88. `TestRabbitMQAdapter\_ConsumerLoop\_ReturnsNilOnContextCancellation$1`
89. `LockOSThread`
90. `stop$bound`
91. `Gosched`
92. `libc\_getdtablesize\_trampoline`
93. `init\#2$4`
94. `legacyLoadMessageDesc$1$1`
95. `poll$1`
96. `sendBatchExtendedWithDescription$1`
97. `lazyInit$1`
98. `file\_opentelemetry\_proto\_trace\_v1\_trace\_proto\_init`
99. `start$1`
100. `syscall\_runtime\_AfterForkInChild`
101. `init\#1`
102. `TestInitWorker\_PanicsWhenLoggerInitFails$4`
103. `NewHTTP2Client$3`
104. `New\[\*crypto/internal/fips140/sha512.Digest\]$1`
105. `init\#1$1$1`
106. `SaveMultipartFile$1`
107. `CloseNotify$1`
108. `spillArgs`
109. `NotifyContext$1`
110. `Shutdown$1`
111. `init\#1`
112. `freemcache$1`
113. `SendMsg$2`
114. `Build$1`
115. `init\#2`
116. `collectTypeParams$1`
117. `Add$1`
118. `libc\_read\_trampoline`
119. `libresolv\_res\_9\_ninit\_trampoline`
120. `Less$1$1`
121. `Pull2$3`
122. `main`
123. `file\_google\_protobuf\_duration\_proto\_init`
124. `initMime`
125. `init\#1`
126. `sendFile$1`
127. `TestRabbitMQAdapter\_ConsumerLoop\_VerifiesSignatureSuccessfully$1`
128. `initMetrics`
129. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnInvalidSignature$1`
130. `cgoBindM`
131. `chansend$1`
132. `x509\_SecPolicyCreateSSL\_trampoline`
133. `runtime\_procUnpin`
134. `runCleanup$1`
135. `libc\_getnameinfo\_trampoline`
136. `file\_opentelemetry\_proto\_collector\_logs\_v1\_logs\_service\_proto\_init`
137. `getConn$1`
138. `init\#1`
139. `callContinuation$1`
140. `generatorTable$1`
141. `p224SqrtCandidate$1`
142. `resetspinning`
143. `acquireForkLock`
144. `sigpanic`
145. `CopyFrom$1`
146. `libc\_ioctl\_trampoline`
147. `readValue$1`
148. `instantiatedType$2`
149. `probe$bound`
150. `Ping$1`
151. `AddSet$1`
152. `runHandler$1`
153. `invokeMarshaler$1`
154. `newClusterState$1`
155. `dropg`
156. `onceSetNextProtoDefaults$bound`
157. `printunlock`
158. `sysctlbyname\_trampoline`
159. `dumpmemprof`
160. `kqueue\_trampoline`
161. `ParseBase$1`
162. `ReadTrace$1`
163. `TestRabbitMQAdapter\_ConsumerLoop\_ReturnsNilOnContextDeadlineExceeded$1`
164. `prepareDC$2`
165. `alloc$1`
166. `init\#1`
167. `runN$1`
168. `libc\_utimensat\_trampoline`
169. `SendMsg$1`
170. `TestValidateParametersNonMetadataKeys$1`
171. `init\#1`
172. `urandomRead$1`
173. `SendFile$1`
174. `traceStopReadCPU`
175. `main`
176. `file\_opentelemetry\_proto\_collector\_metrics\_v1\_metrics\_service\_proto\_rawDescGZIP$1`
177. `Notify$1$1`
178. `scheduleNextConnectionLocked$2`
179. `gcMarkDone$2`
180. `TestRedisCache\_NewRedisCache\_NilConnection\_Panics$1$1`
181. `init\#1`
182. `Close$1`
183. `asyncPreempt`
184. `init\#2`
185. `libc\_mkdirat\_trampoline`
186. `objDecl$1`
187. `TestRabbitMQAdapter\_ProcessDelivery\_ExtractsHeaders$1$1`
188. `Construct$1`
189. `put$1`
190. `traceParse$1`
191. `readType$1`
192. `freeSpan$1`
193. `handleForwards$bound`
194. `libc\_munlockall\_trampoline`
195. `libc\_getgrgid\_r\_trampoline`
196. `gcResetMarkState`
197. `retryLocked$1$1`
198. `runPiped$2`
199. `isParameterized$1`
200. `OnceValue\[map\[string\]reflect.Value\]$1$1`
201. `WithCallStackHelper$1`
202. `basepointNafTable$1`
203. `init\#1`
204. `trace$1`
205. `rawExpr$1`
206. `dumpms`
207. `collectExemplars\[int64\]$1`
208. `gcComputeStartingStackSize`
209. `withConn$1`
210. `ForceFlush$1`
211. `TestRabbitMQAdapter\_ProducerDefault\_SignsMessage$1`
212. `libc\_setsockopt\_trampoline`
213. `init\#2`
214. `ClearStringValidations$1`
215. `init\#1`
216. `libc\_getpgid\_trampoline`
217. `StopMetricsCollector`
218. `WriteOverlays$1`
219. `init\#2`
220. `Shutdown$1`
221. `createContext$8`
222. `libc\_symlink\_trampoline`
223. `stacklessWriterFunc$1`
224. `libpreinit`
225. `x509\_CFNumberGetValue\_trampoline`
226. `Peek$1`
227. `main$1`
228. `resolve$1`
229. `rlock$1`
230. `decIgnoreOpFor$1`
231. `Stop$1`
232. `captureStack$1`
233. `cgounimpl`
234. `libc\_freeaddrinfo\_trampoline`
235. `file\_google\_rpc\_error\_details\_proto\_rawDescGZIP$1`
236. `TestInitWorker\_PanicsWhenConfigLoadFails$1`
237. `stop$1`
238. `stacklessWriteBrotli$1`
239. `init\#1`
240. `run1$1`
241. `read\_trampoline`
242. `handlePing$1`
243. `maintain$4`
244. `noopRedeemer`
245. `goschedIfBusy`
246. `init\#1`
247. `connectOne$2`
248. `main$1`
249. `Test$1$1`
250. `init\#2`
251. `runtime\_AfterFork`
252. `FlushDNSCache`
253. `exitsyscall$2`
254. `EndTracingSpansInterceptor$1$1`
255. `fatalpanic$1`
256. `fRunner$1$1`
257. `worldStarted`
258. `handleSettings$2`
259. `init\#3`
260. `registerHTTPSProtocol$1`
261. `performHandoffInternal$1`
262. `initConfVal$1`
263. `WithCallStackHelper$2`
264. `itabsinit`
265. `checkFinalizersAndCleanups`
266. `validateNorm$1`
267. `main`
268. `prepareForRecovery$1`
269. `archInitIEEE`
270. `build$1`
271. `watchCancel$2`
272. `Shutdown$1`
273. `cancellationListenerCallback$bound`
274. `yaml\_parser\_fetch\_next\_token$1`
275. `Read$1`
276. `init\#1`
277. `Pull2$1$2`
278. `checkmcount`
279. `testAtomic64`
280. `x509\_CFDataCreate\_trampoline`
281. `Close$bound`
282. `traceSyncBatch$1`
283. `libc\_getsockopt\_trampoline`
284. `list$1`
285. `libc\_write\_trampoline`
286. `dispatchDeliveries$1$1`
287. `generatorTable$1`
288. `newUserArenaChunk$1`
289. `init\#1`
290. `checkMinIdleConns$1`
291. `init\#1`
292. `exposeHostPorts$2`
293. `libc\_socketpair\_trampoline`
294. `synctest\_inBubble$1`
295. `doRetryNotify\[github.com/docker/docker/api/types/build.ImageBuildResponse\]$1`
296. `OnceValues$1$1`
297. `internal\_sync\_runtime\_doSpin`
298. `Start$1`
299. `libc\_access\_trampoline`
300. `file\_google\_rpc\_error\_details\_proto\_init`
301. `init\#1`
302. `executeShutdown$1`
303. `OnceValue$1$1$1`
304. `init\#1`
305. `handleEnv`
306. `InitLocalEnvConfig$1`
307. `file\_google\_api\_httpbody\_proto\_init`
308. `TestServiceRun$1`
309. `NewServerTransport$2`
310. `libc\_pwrite\_trampoline`
311. `writeHeapProto$1`
312. `fatalthrow$1`
313. `forcegchelper`
314. `init\#1`
315. `walkRange$2$1`
316. `DisableDIT`
317. `lockRankMayTraceFlush`
318. `lazyInit$1`
319. `StandardCrypto`
320. `initResolutionCache`
321. `coverReport`
322. `initP224`
323. `init\#1`
324. `decPenalty$bound`
325. `envProxyFunc$1`
326. `handleForwards$bound`
327. `init\#1`
328. `p521B$1`
329. `getLen$1`
330. `init\#1`
331. `onWriteTimeout$bound`
332. `exec$1`
333. `exposeHostPort$1`
334. `init\#1`
335. `ChannelWithSubscriptions$1`
336. `operateHeaders$2`
337. `NewPeriodicReader$2`
338. `main`
339. `ServeHTTP$1`
340. `onceSetNextProtoDefaults$bound`
341. `tlsClientHandshake$1`
342. `main$2`
343. `mstartm0`
344. `TestingOnlyAbandon`
345. `TestRabbitMQAdapter\_IsHealthy\_ReturnsTrue\_WhenChannelOpen$1`
346. `goroutineProfileWithLabelsSync$3`
347. `write$2`
348. `unminit`
349. `WithRecover$1$1`
350. `init\#1`
351. `ForEachPackage$2`
352. `typeDecl$3`
353. `minitSignalStack`
354. `entersyscallWakeSysmon`
355. `healthCheck$bound`
356. `init\#1`
357. `ParseBuildInfo$1`
358. `mapv$1`
359. `init\#1`
360. `main`
361. `main`
362. `throw$1`
363. `init\#1`
364. `asminit`
365. `do$1`
366. `libc\_pathconf\_trampoline`
367. `Exec$1`
368. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$1`
369. `setDefaultUserProviders`
370. `TestRabbitMQAdapter\_RunConsumerCycle\_FailsOnQosError$1`
371. `NewTelemetry$1`
372. `badmorestackg0$1`
373. `DecodeFull$1`
374. `maybeRunStateHook$bound`
375. `initHealthCheck$1`
376. `panicwrap`
377. `libc\_undelete\_trampoline`
378. `init\#2`
379. `WithCancel$1`
380. `initLocal`
381. `SendMsg$2`
382. `ReplaceGlobals$1`
383. `flushallmcaches`
384. `unlockOSThread`
385. `mapinitnoop`
386. `mallocinit`
387. `Close$1`
388. `TestRabbitMQAdapter\_ProducerDefault\_RetriesOnFailure$3$1`
389. `mountStartupProcess$1`
390. `Start$1`
391. `init\#1`
392. `gcinit`
393. `initConnPool$bound`
394. `init\#1`
395. `serveStreams$1`
396. `Go$1$1`
397. `libc\_setgid\_trampoline`
398. `gcWriteBarrier6`
399. `init\#3`
400. `RegisterMetricsServiceHandlerFromEndpoint$1`
401. `lazyInit$1`
402. `getcp950$1`
403. `init\#1`
404. `gcenable`
405. `swap$1`
406. `healthCheck$bound`
407. `NewHTTP2Client$5`
408. `StartTimeStampUpdater`
409. `init\#1`
410. `panicBounds`
411. `GetOrDownloadMongod$1`
412. `Reset`
413. `libc\_sendto\_trampoline`
414. `libc\_sync\_trampoline`
415. `WithDeadlineCause$3`
416. `init\#1`
417. `Go$1`
418. `tRunner$2`
419. `WriteOverlays$2$1`
420. `cmdStream$1`
421. `CoordinateFuzzing$2`
422. `isImage$1`
423. `basepointTable$1`
424. `Record$1`
425. `walkRange$1`
426. `AddSet$1`
427. `init\#2`
428. `apply$1`
429. `goServe$1`
430. `runSafePointFn`
431. `gcControllerCommit`
432. `netpollBreak`
433. `Readdir$1`
434. `prepareNext$1`
435. `ServeConn$1`
436. `Put$1`
437. `libc\_utimes\_trampoline`
438. `startTheWorld$1`
439. `init\#1`
440. `init\#1`
441. `testSPWrite`
442. `Record$1`
443. `libc\_rename\_trampoline`
444. `triggerHealthCheck$1`
445. `TurnOn`
446. `netpollinit`
447. `mProf\_Malloc$1`
448. `gcRestoreSyncObjects`
449. `EventuallyWithT$1$1`
450. `pthread\_attr\_setdetachstate\_trampoline`
451. `minitSignals`
452. `init\#1`
453. `reentersyscall$4`
454. `worldStopped`
455. `init\#6`
456. `init\#4`
457. `Execute$1`
458. `SendMsg$1`
459. `NewBatchSpanProcessor$2`
460. `TryGo$1`
461. `TestRabbitMQAdapter\_ProcessDelivery\_RecoverFromPanic$3`
462. `init\#1`
463. `setupMiniredis$1`
464. `runCleanups`
465. `main`
466. `setupHijackConn$1`
467. `RebaseArchiveEntries$1`
468. `TestConcurrentEncryption$1`
469. `ArtifactDir$1`
470. `gcAssistAlloc$2`
471. `libc\_fstat\_trampoline`
472. `MakeTimeoutContext$1`
473. `blocking$2`
474. `libc\_getgid\_trampoline`
475. `libc\_mmap\_trampoline`
476. `getData$1`
477. `file\_grpc\_health\_v1\_health\_proto\_init`
478. `collectFileInfoForChanges$1`
479. `DialContext$1`
480. `init\#1`
481. `init\#1`
482. `connect$1`
483. `basepointTable$1`
484. `kevent\_trampoline`
485. `init\#2`
486. `startChannelWatcher$1`
487. `WaitN$1$1`
488. `Next$1`
489. `startCloseMonitor$1`
490. `searchInStaticDictionary$1`
491. `propagateCancel$2`
492. `writeRecordLocked$1`
493. `CreateLUT`
494. `reset$1`
495. `libc\_getsid\_trampoline`
496. `init\#2$2`
497. `init\#1`
498. `RangeEntries$1`
499. `EnableColorsStdout$1`
500. `Flush$bound`
501. `libc\_umask\_trampoline`
502. `libc\_exchangedata\_trampoline`
503. `addmoduledata`
504. `initBenchmarkFlags`
505. `outgoingGoAwayHandler$1`
506. `init\#1`
507. `Cleanup$1$1`
508. `markroot$1`
509. `main`
510. `libc\_setegid\_trampoline`
511. `TestRabbitMQAdapter\_ConsumerLoop\_SkipsVerificationWhenDisabled$1`
512. `writeTrace$1`
513. `libc\_open\_trampoline`
514. `freezetheworld`
515. `printArgs$3`
516. `sync\_atomic\_runtime\_procUnpin`
517. `sigaction\_trampoline`
518. `collectExemplars\[float64\]$1`
519. `doCall$2`
520. `setSyncObjectsUntraceable`
521. `resetProxyConfig`
522. `operateHeaders$1`
523. `startDialConnForLocked$1`
524. `abortStreamLocked$1`
525. `HostGatewayIP$1`
526. `dounlockOSThread`
527. `gcMarkTermination$1`
528. `UUID$1`
529. `SaveImagesWithOpts$1`
530. `secret\_eraseSecrets`
531. `libc\_getgrouplist\_trampoline`
532. `madvise\_trampoline`
533. `init\#2`
534. `OrderedUnmarshalJSON$1`
535. `newextram`
536. `badTimer`
537. `Go$1`
538. `setPinned$2`
539. `libc\_revoke\_trampoline`
540. `debugCallWrap1`
541. `onIdleTimeout$bound`
542. `delayedFlush$bound`
543. `file\_opentelemetry\_proto\_metrics\_v1\_metrics\_proto\_rawDescGZIP$1`
544. `unminitSignals`
545. `init$bound`
546. `Add$1`
547. `Confirm$1`
548. `getempty$1`
549. `init\#1`
550. `init\#13`
551. `main`
552. `init\#1`
553. `init\#1`
554. `funcLit$1`
555. `libc\_mkdir\_trampoline`
556. `signalMessage$1`
557. `cleanup$1`
558. `\_System`
559. `init\#1`
560. `bytes$1`
561. `handleStateChange$1$1`
562. `WithDeadlineCause$1`
563. `register$bound`
564. `validType0$1`
565. `runtime\_pollServerInit`
566. `doInRoot\[string\]$1`
567. `Close$1`
568. `Alignof$1`
569. `getcp936$1`
570. `traceStartReadCPU`
571. `ProjectRoot$1`
572. `load\_g`
573. `nanotime\_trampoline`
574. `init\#1`
575. `onIdleTimer$bound`
576. `init\#1$1`
577. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnVersionMismatch$1`
578. `cpuVariant$1`
579. `connect$1`
580. `lockRankMayQueueFinalizer`
581. `libc\_fork\_trampoline`
582. `Test$1`
583. `methodValueCall`
584. `Read$1`
585. `mstart0`
586. `TestInitWorker\_PanicsWhenConfigLoadFails$3`
587. `init\#1`
588. `libc\_getrusage\_trampoline`
589. `schedule`
590. `abort`
591. `RecordApproved`
592. `init\#2`
593. `TestHMACSigner\_ConcurrentSigning$1`
594. `UnreachableExceptTests`
595. `MustExtractDockerHost$1`
596. `TestRabbitMQAdapter\_ProcessDelivery\_HandlesNonStringHeaderID$1`
597. `run$2`
598. `initFuzzFlags`
599. `Consume$1`
600. `makeFuncStub`
601. `libc\_fcntl\_trampoline`
602. `mach\_vm\_region\_trampoline`
603. `gcMarkTinyAllocs`
604. `init\#1`
605. `x509\_SecCertificateCreateWithData\_trampoline`
606. `Import$1`
607. `panicdivide`
608. `init\#1`
609. `gcMarkDone`
610. `Add$1`
611. `NewServerTransport$3`
612. `convertAssignRows$2`
613. `libc\_setpriority\_trampoline`
614. `serveContent$1`
615. `init\#4`
616. `RunFuzzWorker$1$1`
617. `init\#1`
618. `sweep$1`
619. `Channel$1`
620. `init\#1`
621. `TestRabbitMQAdapter\_ProducerDefault\_AllRetriesFail$1`
622. `ContainerWait$1`
623. `NewServerTransport$1`
624. `watchCancel$1`
625. `SortSliceBetween$1`
626. `run$1`
627. `init\#4`
628. `init\#1`
629. `listenerBacklog$1`
630. `init\#1`
631. `goenvs`
632. `onceSetNextProtoDefaults$bound`
633. `init\#1`
634. `OnEmit$2`
635. `init\#1`
636. `init\#1`
637. `NewWithConfig$5`
638. `file\_opentelemetry\_proto\_common\_v1\_common\_proto\_init`
639. `libc\_readlink\_trampoline`
640. `readFrames$1`
641. `schedinit`
642. `xRegInitAlloc`
643. `Decode$1`
644. `libc\_geteuid\_trampoline`
645. `\_ExternalCode`
646. `file\_opentelemetry\_proto\_collector\_trace\_v1\_trace\_service\_proto\_rawDescGZIP$1`
647. `exportContext$1`
648. `init\#1`
649. `ResolverError$1`
650. `startGracefulShutdown$1`
651. `setRequestCancel$2`
652. `getcp1251$1`
653. `delayedFlush$bound`
654. `ignoreSIGSYS`
655. `initPostcodes`
656. `main`
657. `mProf\_Flush`
658. `privateLogw$1`
659. `didPanic$1`
660. `runtime\_AfterExec`
661. `init\#1`
662. `Read$1`
663. `libc\_setrlimit\_trampoline`
664. `InitLocalEnvConfig$1`
665. `init\#1`
666. `Events$1`
667. `traceExitedSyscall`
668. `Enable`
669. `TestInitWorker\_PanicsWhenLoggerInitFails$1`
670. `tunnel$2`
671. `init\#2`
672. `casgstatus$1`
673. `stacklessWriteZstd$1`
674. `InitLocalEnvConfig$1`
675. `gcMarkTermination$5`
676. `loadHTTPBytes$1$1`
677. `warnBlocked`
678. `stacklessWriteGzip$1`
679. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P256Point\]\]$1$1`
680. `OnceValues$1$1$1`
681. `writeBody$1`
682. `TestRabbitMQAdapter\_ProducerDefault\_SkipsSigningWhenNoSigner$1`
683. `gcWriteBarrier5`
684. `pthread\_mutex\_unlock\_trampoline`
685. `freeStackSpans`
686. `init\#1`
687. `x509\_CFArrayAppendValue\_trampoline`
688. `libc\_recvfrom\_trampoline`
689. `libc\_gai\_strerror\_trampoline`
690. `libc\_getpeername\_trampoline`
691. `Watch$1`
692. `init\#3`
693. `lock$1`
694. `Record$1`
695. `defaultGOMAXPROCSUpdateEnable`
696. `infer$1`
697. `morestackc`
698. `doRetryNotify\[\*github.com/docker/docker/api/types/container.Summary\]$1`
699. `readPreface$1`
700. `modulesinit`
701. `PCall$1`
702. `init$2`
703. `write$1`
704. `FreeOSMemory`
705. `libc\_chmod\_trampoline`
706. `Default$1`
707. `setDefaultOSDescriptionProvider`
708. `init\#1`
709. `cgoSigtramp`
710. `ResetCoverage`
711. `runCmdContext$4`
712. `init\#5`
713. `typInternal$1`
714. `libc\_setreuid\_trampoline`
715. `gcStart$1`
716. `logDropped\[go.opentelemetry.io/otel/sdk/trace.Link\]$1`
717. `file\_opentelemetry\_proto\_trace\_v1\_trace\_proto\_rawDescGZIP$1`
718. `instantiatedType$1`
719. `init\#1`
720. `removePerishedConns$1`
721. `main`
722. `maybeRunStateHook$1`
723. `newNonRetryClientStream$2`
724. `gcTestMoveStackOnNextCall`
725. `AddSet$1`
726. `os\_sigpipe`
727. `Run$1`
728. `unlockProfiles`
729. `TestQueryDatabase\_DataSourceFactoryAndLifecycleErrors$1$1`
730. `ExportChanges$1`
731. `runCmdContext$3$1`
732. `main`
733. `init\#6`
734. `printDebugLog`
735. `InitializeTelemetry$2`
736. `writeEarlyAbort$2`
737. `connect$1`
738. `doinit`
739. `init\#1`
740. `fixedHuffmanDecoderInit$1`
741. `main`
742. `OnPut$1`
743. `xRegRestore$1`
744. `startGracefulShutdown$bound`
745. `TestRabbitMQAdapter\_ProducerDefault\_ReturnsError\_WhenEnsureChannelFails$1`
746. `Unreachable`
747. `goroutineLeakProfileWithLabelsConcurrent$1$1`
748. `Apply$1`
749. `updateTargetResolverState$1`
750. `AddSet$1`
751. `secret\_inc`
752. `libc\_setgroups\_trampoline`
753. `sendResponse$1`
754. `Stop$1`
755. `NewHTTP2Client$1`
756. `poolCleanup`
757. `TestRabbitMQAdapter\_RunConsumerCycle\_FailsOnConsumeError$1`
758. `GetOrBuildProducer$1`
759. `libc\_fchown\_trampoline`
760. `archInit`
761. `closeReqBodyLocked$1`
762. `Pull$3`
763. `init$2`
764. `parseFiles$2$1`
765. `libc\_exit\_trampoline`
766. `TestRabbitMQAdapter\_ProcessDelivery\_NackError$3`
767. `parseUnion$1`
768. `file\_google\_protobuf\_wrappers\_proto\_rawDescGZIP$1`
769. `lazyInit$1`
770. `runtime\_debug\_freeOSMemory$1`
771. `StatelessDeflate$2`
772. `init\#1`
773. `readAll$1`
774. `Force`
775. `onWriteTimeout$bound`
776. `init\#1`
777. `file\_opentelemetry\_proto\_logs\_v1\_logs\_proto\_rawDescGZIP$1`
778. `reflectOffsLock`
779. `TestProcessPluginCRMCollection\_WithValidOrganization$1`
780. `queryDatabase$1`
781. `allocm$1`
782. `secure`
783. `CreateContainer$1`
784. `ResetWithOptions$1`
785. `runExample$1`
786. `libc\_issetugid\_trampoline`
787. `init\#1`
788. `init\#1`
789. `Do$1`
790. `ResetServiceIndicator`
791. `main`
792. `main`
793. `AddSet$1`
794. `Close$1`
795. `init\#1`
796. `libc\_readdir\_r\_trampoline`
797. `init\#6`
798. `walk$1$1`
799. `libc\_sendmsg\_trampoline`
800. `basepointNafTable$1`
801. `initResourceValue\[\*github.com/jackc/pgx/v5/pgxpool.connResource\]$1`
802. `invokeError$1`
803. `CheckPath$1`
804. `send$1`
805. `scavenge$1`
806. `x509\_CFArrayCreateMutable\_trampoline`
807. `init\#17`
808. `Stop$1`
809. `mayMoreStackPreempt`
810. `SetMx$1`
811. `startupValidation$1`
812. `goready$1`
813. `connect$2`
814. `onceSetNextProtoDefaults\_Serve$bound`
815. `init\#1`
816. `addOption$1`
817. `corostart`
818. `ResetPanicMetrics`
819. `Acquire$1`
820. `libc\_sysctl\_trampoline`
821. `startStreamDecoder$1`
822. `onReadTimeout$bound`
823. `Flush$bound`
824. `search$2$1$1`
825. `Clearenv`
826. `main`
827. `pageTmpl$1`
828. `processTxPipelineNode$1$1`
829. `TestScanColumns$2$1`
830. `libc\_stat\_trampoline`
831. `breakpoint`
832. `NextResultSet$1`
833. `init\#1`
834. `pthread\_self\_trampoline`
835. `main`
836. `unsetBypass`
837. `StatsPrint`
838. `malg$1`
839. `libc\_getrlimit\_trampoline`
840. `syscall\_runtime\_BeforeExec`
841. `init$1`
842. `traceNotificationResponse$1`
843. `TestValidateParameters$1`
844. `TestQueryPluginCRM\_WithOrganizationOnly$1`
845. `getcp874$1`
846. `Commit$1`
847. `x509\_CFArrayGetCount\_trampoline`
848. `Record$1`
849. `Read$1`
850. `init\#1`
851. `createIdleResources$1`
852. `init\#2`
853. `ServeHTTP$1$1`
854. `updateMaxProcsGoroutine`
855. `Add$1`
856. `needsInitCheckLocked$1`
857. `panicunsafeslicenilptr`
858. `x509\_CFArrayGetValueAtIndex\_trampoline`
859. `ServeFile$1`
860. `main$1`
861. `scan$3`
862. `redirectStdLogAt$1`
863. `mapv$1`
864. `saveFile$1`
865. `mountStartupProcess$2`
866. `parsePattern$1`
867. `OrderedMarshalJSON$1`
868. `init\#1`
869. `systemstack\_switch`
870. `UpdateClientConnState$1`
871. `init\#4`
872. `RegisterTraceServiceHandlerFromEndpoint$1`
873. `init\#1`
874. `writeContext$1`
875. `addrLookupOrder$1`
876. `save\_g`
877. `NewPrivateKey$1`
878. `runtime\_AfterForkInChild`
879. `init\#2`
880. `exit\_trampoline`
881. `init\#1`
882. `InitLocalEnvConfig$1`
883. `Close$1`
884. `doRetryNotify\[\*github.com/testcontainers/testcontainers-go.Reaper\]$1`
885. `cancel$1`
886. `init\#4`
887. `compute$1`
888. `TestRedisCache\_NewRedisCache\_NilConnection\_Panics$2$1`
889. `embeddedIfaceMethStub`
890. `setRequestCancel$3`
891. `init\#2`
892. `x509\_SecCertificateCopyData\_trampoline`
893. `SetFinalizer$1`
894. `appendJSONMarshal$1`
895. `dumpScanStats`
896. `buildCommonHeaderMaps`
897. `TestConsumerRoutes\_Shutdown$2`
898. `StmtContext$2`
899. `chanrecv$1`
900. `run$1`
901. `TestScanColumns$1$1`
902. `initCommonHeader`
903. `init\#3`
904. `CloseNotify$1`
905. `processHandoffRequest$1`
906. `makeTempDir$1`
907. `PCall$1$1`
908. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P521Point\]\]$1$1$1`
909. `libc\_getfsstat\_trampoline`
910. `init\#1`
911. `writeHeader$1`
912. `prefork$2`
913. `ResetTestLicenseBaseURL`
914. `pingDC$1`
915. `parsePrimaryExpr$1`
916. `OnceFunc$1$1$1`
917. `file\_grpc\_health\_v1\_health\_proto\_rawDescGZIP$1`
918. `runtime\_goroutineLeakGC`
919. `listPackages$1`
920. `onReadTimeout$bound`
921. `OnceValue\[encoding/json.encoderFunc\]$1$1$1`
922. `file\_opentelemetry\_proto\_collector\_trace\_v1\_trace\_service\_proto\_init`
923. `traceCopyFail$1`
924. `sysmonUpdateGOMAXPROCS`
925. `synctestRun$2`
926. `main`
927. `prepareNext$1`
928. `snapshotConnection$1`
929. `libc\_writev\_trampoline`
930. `\_VDSO`
931. `RecordSet$1`
932. `prepareDC$1`
933. `scan$4`
934. `gorecover$1`
935. `Write$1`
936. `ScanAndListen$2`
937. `Run$1`
938. `Shutdown$1`
939. `parse$1`
940. `processOptions`
941. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P384Point\]\]$1$1$1`
942. `entersyscall`
943. `TestConsumerRoutes\_Shutdown$1`
944. `next$1`
945. `newTdsBuffer$1`
946. `stubJobSpanAttributes$2`
947. `readRequest$1`
948. `logDropped\[go.opentelemetry.io/otel/sdk/trace.Event\]$1`
949. `Stack$1`
950. `libc\_getppid\_trampoline`
951. `traceAdvance$1$1`
952. `init\#1`
953. `startTemplateThread`
954. `init\#1`
955. `libc\_unmount\_trampoline`
956. `gfget$1`
957. `LazyReload$1`
958. `TestProcessPluginCRMCollection\_WithOrganizationID$1`
959. `getCh$1`
960. `Cleanup$1`
961. `containsElement$1`
962. `libc\_ftruncate\_trampoline`
963. `UnregisterSpanProcessor$1`
964. `validateNorm$1`
965. `stop$1`
966. `dispatch0$1`
967. `unpinConnectionFromCursor$bound`
968. `Serve$2`
969. `TestValidateParametersWithCursor$1`
970. `setPinned$1`
971. `init\#1`
972. `markrootFreeGStacks`
973. `queryDC$1`
974. `init\#1`
975. `reentersyscall$2`
976. `traceDescribe$1`
977. `gcBgMarkStartWorkers`
978. `nextMarkBitArenaEpoch`
979. `pollSRVRecords$1`
980. `onReadIdleTimer$bound`
981. `traceInitReadCPU`
982. `libc\_socket\_trampoline`
983. `libc\_error\_trampoline`
984. `gcBeginWork`
985. `minimize$1`
986. `sendOpen$1`
987. `init\#7`
988. `TestMultiQueueConsumerRun$1`
989. `lockVerifyMSize`
990. `InitLocalEnvConfig$1`
991. `libc\_renameat\_trampoline`
992. `runCleanup$2`
993. `AddSet$1`
994. `handleUpgradeResponse$1`
995. `DecodeAll$1`
996. `readForm$1`
997. `CopyFileWithTar$1`
998. `beginDC$1`
999. `procUnpin`
1000. `closeRead$1`
1001. `initConfVal`
1002. `maintain$3`
1003. `close$1`
1004. `doRetryNotify$1`
1005. `structType$3`
1006. `Remove$1`
1007. `UploadLogs$1`
1008. `RegisterLogsServiceHandlerFromEndpoint$1$1`
1009. `read$1`
1010. `TestRabbitMQAdapter\_ProducerDefault\_Success$1$1`
1011. `duffcopy`
1012. `write\_trampoline`
1013. `CreateNetwork$1`
1014. `lostProfileEvent`
1015. `libc\_recvmsg\_trampoline`
1016. `NewHTTP2Client$6`
1017. `handleIdleTimeout$bound`
1018. `doInRoot\[os.FileInfo\]$1`
1019. `init\#3`
1020. `RunConsumers$1`
1021. `archInitCastagnoli`
1022. `pinConnectionToTransaction$bound`
1023. `execDC$2`
1024. `invalidateChannel$1`
1025. `buildCommonHeaderMapsOnce`
1026. `init\#16`
1027. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnHandlerError$1`
1028. `ParseAcceptLanguage$1`
1029. `libc\_chown\_trampoline`
1030. `Subscribe$2`
1031. `fieldByIndexErr$1`
1032. `doInRoot\[struct{}\]$1`
1033. `shutdownWorkers$2`
1034. `buildOnce$bound`
1035. `updateTargetResolverState$2`
1036. `prefork$1`
1037. `minitSignalMask`
1038. `getcp1254$1`
1039. `init\#3`
1040. `init\#1`
1041. `Serve$1`
1042. `morestack`
1043. `newServer$1`
1044. `ParseScript$1`
1045. `readPreface$1`
1046. `contactResponders$1$1`
1047. `StopCPUProfile`
1048. `ParseExprFrom$1`
1049. `AddSet$1`
1050. `ParseVariant$1`
1051. `goenvs\_unix`
1052. `SendBatch$1`
1053. `stoplockedm`
1054. `onWriteTimeout$bound`
1055. `write$1`
1056. `parseMultiplexedLogs$1`
1057. `minimize$2`
1058. `NewFastHTTPHandler$1$1`
1059. `lazyRegexCompile$1$1`
1060. `Shutdown$1$1$1`
1061. `ExitIdle$bound`
1062. `handleMoving$1`
1063. `ParseRegion$1`
1064. `main`
1065. `processStreamingRPC$1`
1066. `dispatchClosed$1`
1067. `SendMsg$4`
1068. `doCall$1`
1069. `init\#1`
1070. `fatal$1`
1071. `commandLineUsage`
1072. `AddSet$1`
1073. `Shutdown$1`
1074. `emptyfunc`
1075. `Less$1`
1076. `http2registerHTTPSProtocol$1`
1077. `init\#1`
1078. `beginFuncExec$1`
1079. `encodeError$1`
1080. `libc\_dup\_trampoline`
1081. `stderrHandler$1`
1082. `StatelessDeflate$1`
1083. `NotifyConfirm$1`
1084. `connect$1`
1085. `gcWakeAllStrongFromWeak`
1086. `Start$2`
1087. `init\#1`
1088. `getcp1255$1`
1089. `startLocked$1`
1090. `x509\_CFDataGetLength\_trampoline`
1091. `dit\_setDisabled`
1092. `checkOut$1`
1093. `UploadTraces$1`
1094. `RecordSet$1`
1095. `TestMultiQueueConsumerRun$2$1`
1096. `TestWithSwaggerEnvConfig\_DefaultValues$1`
1097. `callers$1`
1098. `file\_opentelemetry\_proto\_common\_v1\_common\_proto\_rawDescGZIP$1`
1099. `selectgo$2`
1100. `buildOnce$bound`
1101. `traceRowDescription$1`
1102. `ClearObjectValidations$1`
1103. `after$1`
1104. `failWantMap`
1105. `dial$1`
1106. `TestHandlerGenerateReport\_DelegatesToUseCase$1`
1107. `checkdead`
1108. `lazyInit$1`
1109. `aberrantDeriveMessageName$1`
1110. `libc\_grantpt\_trampoline`
1111. `main`
1112. `Scan$1`
1113. `gcWriteBarrier3`
1114. `SaveImagesWithOpts$2`
1115. `Shutdown$1`
1116. `exportSync$1`
1117. `startHealthCheck$1`
1118. `equalServiceConfig$1`
1119. `traceCPUFlush$1`
1120. `RecordSet$1`
1121. `SetMeterProvider$1`
1122. `randinit`
1123. `Run$1`
1124. `gcstopm`
1125. `gcStart$3`
1126. `getcp850$1`
1127. `libc\_shutdown\_trampoline`
1128. `SelectServer$1`
1129. `startLoad$1`
1130. `StartTimeStampUpdater$1`
1131. `pipe\_trampoline`
1132. `gcBgMarkWorker$2`
1133. `init\#1`
1134. `sysctl\_trampoline`
1135. `secret\_dec`
1136. `init\#4$4`
1137. `init\#1`
1138. `Record$1`
1139. `init\#1`
1140. `AddSet$1`
1141. `updateClientConnState$2`
1142. `init\#1`
1143. `buildCommonHeaderMaps`
1144. `buildShutdownHandlers$1`
1145. `main`
1146. `lazyInit$1`
1147. `PrintStack`
1148. `panicunsafeslicelen`
1149. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnHandlerError$3`
1150. `tryDial$1`
1151. `Start$1`
1152. `write$1`
1153. `init\#2`
1154. `libc\_fchmod\_trampoline`
1155. `SetErrorHandler$1`
1156. `ReuseOrCreateContainer$1`
1157. `init\#2`
1158. `x509\_CFRelease\_trampoline`
1159. `propagateCancel$1`
1160. `sync\_runtime\_procUnpin`
1161. `reflectOffsUnlock`
1162. `mstart\_stub`
1163. `libc\_statfs\_trampoline`
1164. `ParseExtension$1`
1165. `buildRootHuffmanNode`
1166. `TestRateLimiter\_ConcurrentAccess$1`
1167. `lazyInit$1`
1168. `libc\_getcwd\_trampoline`
1169. `osinit\_hack\_trampoline`
1170. `addMultiCallback$1`
1171. `keepalive$1`
1172. `onShutdownTimer$bound`
1173. `main`
1174. `aberrantDeriveMessageName$1$1`
1175. `preprintpanics$1`
1176. `HandleStreams$2`
1177. `parseBinaryExpr$1`
1178. `servePeer$1`
1179. `init\#1`
1180. `defPredeclaredNil`
1181. `RunParallel$1`
1182. `encodeStringer$1`
1183. `init\#1`
1184. `FixedZone$1`
1185. `UnlockOSThread`
1186. `restoreSIGSYS`
1187. `setDefaultUnameProvider`
1188. `runExample$2`
1189. `init\#1`
1190. `RecvMsg$1`
1191. `libc\_fchownat\_trampoline`
1192. `panicunsafestringnilptr`
1193. `ReadMemStats$1`
1194. `copyTrailersToHandlerRequest$bound`
1195. `init\#1`
1196. `libc\_getuid\_trampoline`
1197. `libc\_futimes\_trampoline`
1198. `init\#7`
1199. `handshakeContext$1`
1200. `printDebugLogImpl`
1201. `TestMust$2$1`
1202. `Shutdown$1`
1203. `parseCpuList`
1204. `flush`
1205. `init\#1`
1206. `initMsgChan$1`
1207. `Connection$1`
1208. `SetDelegate$1`
1209. `runtime\_procUnpin`
1210. `mutateBytes$1`
1211. `libc\_chflags\_trampoline`
1212. `traceReadyForQuery$1`
1213. `getcp1257$1`
1214. `Getsockopt$1`
1215. `oneNewExtraM`
1216. `file\_google\_protobuf\_field\_mask\_proto\_rawDescGZIP$1`
1217. `serve$1`
1218. `issetugid\_trampoline`
1219. `panicmakeslicecap`
1220. `sigtramp`
1221. `libc\_close\_trampoline`
1222. `main`
1223. `GetValidator$1`
1224. `closeWrite$1`
1225. `SetDefaultGOMAXPROCS`
1226. `init\#4`
1227. `Add$1`
1228. `init$1`
1229. `main`
1230. `fuzz$2`
1231. `typeDecl$1`
1232. `init\#1`
1233. `serveStreams$2$1`
1234. `PluginInstall$1$1`
1235. `printlock`
1236. `init\#1`
1237. `file\_opentelemetry\_proto\_collector\_logs\_v1\_logs\_service\_proto\_rawDescGZIP$1`
1238. `readHosts`
1239. `RecordSet$1`
1240. `search$2`
1241. `file\_google\_protobuf\_timestamp\_proto\_init`
1242. `structv$1`
1243. `file\_opentelemetry\_proto\_resource\_v1\_resource\_proto\_init`
1244. `init\#1`
1245. `runFinalizers`
1246. `init\#11`
1247. `libc\_fdopendir\_trampoline`
1248. `Add$1`
1249. `AfterFunc$1$1`
1250. `appendValue$1`
1251. `exitsyscall$1`
1252. `defPredeclaredFuncs`
1253. `flushLine$1`
1254. `safeCall$1`
1255. `releaseThread`
1256. `libc\_msync\_trampoline`
1257. `main`
1258. `Never$1`
1259. `init\#2`
1260. `main`
1261. `destroy$1`
1262. `prepareFreeWorkbufs`
1263. `walk$1`
1264. `exitsyscall$4`
1265. `readFrames$1`
1266. `Run$1`
1267. `ReplaceFileTarWrapper$1`
1268. `\_LostContendedRuntimeLock`
1269. `panicSimdImm`
1270. `doRetryNotify\[struct{}\]$1`
1271. `refreshServerDate`
1272. `createContext$7`
1273. `init\#1`
1274. `checkfds`
1275. `ResetWithOptions$1`
1276. `OnPut$1$1`
1277. `mspinning`
1278. `alloc$1`
1279. `checkGenericIsExpected`
1280. `racegoend`
1281. `runCmdContext$2`
1282. `libc\_ptrace\_trampoline`
1283. `init\#1`
1284. `init\#1`
1285. `Write$1`
1286. `init\#1`
1287. `initCategoryAliases`
1288. `createfing`
1289. `lazyInit$1`
1290. `isZeroValue$1`
1291. `file\_opentelemetry\_proto\_resource\_v1\_resource\_proto\_rawDescGZIP$1`
1292. `pthread\_cond\_timedwait\_relative\_np\_trampoline`
1293. `doInRoot$1`
1294. `main`
1295. `gcWriteBarrier1`
1296. `connect$bound`
1297. `yaml\_parser\_fetch\_next\_token$1`
1298. `doRecordGoroutineProfile$1`
1299. `initDefaultMap`
1300. `init$13`
1301. `executeMultiSlot$2`
1302. `processHandoffRequest$2`
1303. `raiseproc\_trampoline`
1304. `ProjectRoot$1`
1305. `Prepare$1`
1306. `RegisterMetricsServiceHandlerFromEndpoint$1$1`
1307. `TestRabbitMQAdapter\_ConsumerLoop\_NormalizesConcurrency$1`
1308. `Shutdown$1`
1309. `readGCStats$1`
1310. `sweepone$1`
1311. `libc\_wait4\_trampoline`
1312. `build$bound`
1313. `init\#1`
1314. `Close$bound`
1315. `init\#8`
1316. `libc\_linkat\_trampoline`
1317. `handleRSTStream$1`
1318. `runtime\_BeforeExec`
1319. `defaultGOMAXPROCSUpdateGODEBUG`
1320. `DialContext$1`
1321. `unmarshalFull$1$1`
1322. `commitAttemptLocked$bound`
1323. `parseParameterList$1`
1324. `Do$1`
1325. `finish$bound`
1326. `pthread\_create\_trampoline`
1327. `initAlgAES`
1328. `debugCallWrap$1`
1329. `resolve$1`
1330. `Disable`
1331. `build$bound`
1332. `finishsweep\_m`
1333. `probe$bound`
1334. `fips140\_setBypass`
1335. `Resolve$1`
1336. `buildRootHuffmanNode`
1337. `getConn$1`
1338. `panicfloat`
1339. `init\#1`
1340. `maybeRunAsync$1`
1341. `libc\_mlock\_trampoline`
1342. `addOption$1`
1343. `validVarType$1`
1344. `main`
1345. `file\_google\_protobuf\_duration\_proto\_rawDescGZIP$1`
1346. `init\#1`
1347. `CloseSend$2`
1348. `init\#1`
1349. `gcMarkTermination$2`
1350. `init\#3`
1351. `main`
1352. `InitMongoDBExternal$1`
1353. `CopyFileWithTar$2`
1354. `lockOSThread`
1355. `racefini`
1356. `Parse$1`
1357. `refill$1`
1358. `wirep$1`
1359. `pthread\_attr\_init\_trampoline`
1360. `libc\_link\_trampoline`
1361. `checkInNoEvent$1`
1362. `init\#1`
1363. `processSingleResponse$1`
1364. `Record$1`
1365. `OnceValue\[string\]$1$1`
1366. `fRunner$2`
1367. `readDataFrame$1`
1368. `libc\_closedir\_trampoline`
1369. `connect$1`
1370. `init\#1`
1371. `ensureSigM`
1372. `threadRun$1`
1373. `printsp`
1374. `collectFileInfoForChanges$2`
1375. `registerBasics`
1376. `initServerWorkers$1`
1377. `processDelivery$1`
1378. `traceAdvance$5`
1379. `TestQueryPluginCRMCollectionWithFilters\_NoFilters$1`
1380. `init\#1`
1381. `init\#1`
1382. `connect$2`
1383. `defPredeclaredConsts`
1384. `startAlarm$1`
1385. `p384B$1`
1386. `CleanupContainer$1`
1387. `file\_grpc\_binlog\_v1\_binarylog\_proto\_init`
1388. `TestRabbitMQAdapter\_Shutdown\_ContextCanceledDuringWait$1`
1389. `Stop$1`
1390. `init\#2`
1391. `init\#1`
1392. `buildRecompMap`
1393. `init\#1`
1394. `init\#1`
1395. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P256Point\]\]$1$1$1`
1396. `init\#1`
1397. `goroutineProfileWithLabelsSync$4$1`
1398. `Shutdown$1$1`
1399. `initOptions`
1400. `synctestRun$1`
1401. `init\#1`
1402. `signature$1`
1403. `libc\_bind\_trampoline`
1404. `panicoverflow`
1405. `Finish$bound`
1406. `needAndBindM`
1407. `init\#1`
1408. `ParallelContainers$1`
1409. `updateProxyResolverState$1`
1410. `cancel$1`
1411. `SkipIfProviderIsNotHealthy$1`
1412. `dialCtx$1`
1413. `init\#1`
1414. `goschedguarded`
1415. `finish$bound`
1416. `disableInfinityTS`
1417. `fips140\_unsetBypass`
1418. `emit$1`
1419. `init\#1`
1420. `newPortForwarder$1`
1421. `newClientStream$1`
1422. `ExitIdle$1`
1423. `init\#1`
1424. `init\#1`
1425. `Read$1`
1426. `init\#2`
1427. `Peek$1`
1428. `init\#1`
1429. `init\#1`
1430. `freeSomeWbufs$1`
1431. `TestRabbitMQAdapter\_IsHealthy\_ReturnsFalse\_WhenChannelClosed$1`
1432. `getcp1258$1`
1433. `main`
1434. `find$1`
1435. `Add$1`
1436. `sysmon`
1437. `doBlockingWithCtx\[\[\]vendor/golang.org/x/net/dns/dnsmessage.Resource\]$1`
1438. `syscall\_runtime\_AfterExec`
1439. `x509\_SecTrustEvaluateWithError\_trampoline`
1440. `clientHandshake$1`
1441. `traceBind$1`
1442. `Parse$1`
1443. `Serve$3`
1444. `libc\_setregid\_trampoline`
1445. `Close$1`
1446. `read$1`
1447. `gcMarkTermination$3`
1448. `RecordSet$1`
1449. `Shutdown$1`
1450. `crash`
1451. `asyncIsExpired$1`
1452. `secureEnv`
1453. `init$1`
1454. `copyTrailers$bound`
1455. `StmtContext$1`
1456. `pthread\_setspecific\_trampoline`
1457. `keepalive$1`
1458. `minimize$1`
1459. `runtime\_BeforeFork`
1460. `FindLongestMatch$1`
1461. `TestExternalDataSource\_CloseConnection\_NilConnection$1$1`
1462. `x509\_CFDataGetBytePtr\_trampoline`
1463. `syscallN\_trampoline`
1464. `term$1`
1465. `Pull$1$2`
1466. `init\#1`
1467. `TestRabbitMQAdapter\_ProcessDelivery\_NackError$1`
1468. `libc\_chdir\_trampoline`
1469. `DisableRandPool`
1470. `onWriteTimeout$bound`
1471. `Has$1`
1472. `collectExemplars$1`
1473. `Current$1`
1474. `init\#2`
1475. `traceCommandComplete$1`
1476. `Parse$1`
1477. `libresolv\_res\_9\_nclose\_trampoline`
1478. `OnceValue\[map\[string\]reflect.Value\]$1$1$1`
1479. `libc\_lseek\_trampoline`
1480. `main`
1481. `libc\_ptsname\_r\_trampoline`
1482. `init\#2`
1483. `merge$1`
1484. `init\#1`
1485. `unifyShutdown$1$1`
1486. `init\#5`
1487. `newcoro$1`
1488. `init\#1`
1489. `RecvMsg$1`
1490. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnVersionMismatch$3`
1491. `main`
1492. `objDecl$2`
1493. `ensureSigM$1`
1494. `file\_opentelemetry\_proto\_logs\_v1\_logs\_proto\_init`
1495. `queuedNewConn$2$1`
1496. `TestRabbitMQAdapter\_ProducerDefault\_ReturnsError\_WhenCircuitOpen$1`
1497. `badmorestackgsignal`
1498. `New\[\*crypto/internal/fips140/sha512.Digest\]$1$1`
1499. `libc\_truncate\_trampoline`
1500. `entersyscallblock$2`
1501. `doBlockingWithCtx\[\[\]string\]$1`
1502. `EvaluateConstValue$1`
1503. `Consume$1`
1504. `initOnce$1`
1505. `watchCancel$2`
1506. `ParseFile$1`
1507. `interfaceEqual$1`
1508. `gcMarkTermination$4$1`
1509. `libc\_posix\_openpt\_trampoline`
1510. `init\#1`
1511. `badunlockosthread`
1512. `closeStream$1`
1513. `newClientStreamWithParams$1`
1514. `queryDC$2`
1515. `pthread\_attr\_getstacksize\_trampoline`
1516. `init\#1`
1517. `main`
1518. `distTmpl$1`
1519. `roundTrip$1`
1520. `Check`
1521. `init\#19`
1522. `IncNonDefault$bound`
1523. `RecordMetrics$1`
1524. `Shutdown$1`
1525. `main`
1526. `initAll`
1527. `goyield`
1528. `exitsyscall`
1529. `libc\_rmdir\_trampoline`
1530. `doHTTPConnectHandshake$1`
1531. `mstart`
1532. `init\#1`
1533. `init\#1`
1534. `atom$1`
1535. `init\#2`
1536. `BoringCrypto`
1537. `endCheckmarks`
1538. `RecordSet$1`
1539. `initSecureMode`
1540. `init$1`
1541. `file\_google\_protobuf\_struct\_proto\_init`
1542. `publicationBarrier`
1543. `badmorestackg0`
1544. `operateHeaders$5`
1545. `mayMoreStackMove`
1546. `init\#6`
1547. `ResetMaxTraceEntryToDefault`
1548. `append$1`
1549. `executeParallel$2`
1550. `WithRecover$1$1`
1551. `init\#1`
1552. `StopTrace`
1553. `runWithClient$1`
1554. `TestWithSwaggerEnvConfig\_EmptyEnvVars$1`
1555. `getcp949$1`
1556. `Ping$1`
1557. `onReadIdleTimer$bound`
1558. `finalClose$2`
1559. `getcp1256$1`
1560. `badctxt`
1561. `badsystemstack`
1562. `debugOptions`
1563. `libc\_chroot\_trampoline`
1564. `bootstrapRandReseed`
1565. `init\#4`
1566. `NewFunc$1`
1567. `DefaultEncoder$1`
1568. `traceAdvance$4`
1569. `init\#1`
1570. `TimeoutWithCodeHandler$1$1`
1571. `libc\_openat\_trampoline`
1572. `badreflectcall`
1573. `OnceFunc$1$1`
1574. `synctestWait`
1575. `New\[\*crypto/internal/fips140/sha256.Digest\]$1`
1576. `init\#1`
1577. `goexit1`
1578. `AddSet$1`
1579. `TestRabbitMQAdapter\_ProcessDelivery\_RecoverFromPanic$1`
1580. `stacklessWriteDeflate$1`
1581. `loop`
1582. `Record$1`
1583. `structv$1`
1584. `nextFrame$1`
1585. `Add$1`
1586. `set$1`
1587. `End$bound`
1588. `genericExprList$1`
1589. `Value$1`
1590. `operateHeaders$3`
1591. `OnceValue\[encoding/json.encoderFunc\]$1$1`
1592. `traceThreadDestroy$1`
1593. `casgstatus$2`
1594. `libc\_unlink\_trampoline`
1595. `init\#18`
1596. `libc\_fchmodat\_trampoline`
1597. `Run$1`
1598. `WriteOverlays$2`
1599. `sigpipe`
1600. `shutdown$1`
1601. `TestScanRows$3$1`
1602. `libresolv\_res\_9\_nsearch\_trampoline`
1603. `RoundTrip$1`
1604. `libc\_sysconf\_trampoline`
1605. `gcStart$4`
1606. `processSRVResults$1`
1607. `runConcurrent$1`
1608. `printCountProfile$2`
1609. `gcWriteBarrier7`
1610. `startBackgroundRead$bound`
1611. `readContent$1`
1612. `file\_google\_rpc\_status\_proto\_rawDescGZIP$1`
1613. `invokeStringer$1`
1614. `update$1`
1615. `sigprocmask\_trampoline`
1616. `Release$1`
1617. `pthread\_cond\_init\_trampoline`
1618. `init$bound`
1619. `Token$1`
1620. `main`
1621. `wbBufFlush`
1622. `init\#1`
1623. `file\_google\_protobuf\_wrappers\_proto\_init`
1624. `Flush$1`
1625. `main`
1626. `PrintDefaults`
1627. `main`
1628. `osyield`
1629. `Close$1`
1630. `DumpRequestOut$2`
1631. `Compose$1`
1632. `init\#1`
1633. `FiberWrapHandler$1$1`
1634. `run1$1$1`
1635. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnInvalidSignature$3`
1636. `flush$1`
1637. `onSettingsTimer$bound`
1638. `buildRecompMap`
1639. `initialize$bound`
1640. `NewFastHTTPHandler$1$1$1`
1641. `Close$1`
1642. `init\#1`
1643. `SignalNum$1`
1644. `ClearArrayValidations$1`
1645. `libc\_setsid\_trampoline`
1646. `queuedNewConn$1`
1647. `connect$1`
1648. `Close$1`
1649. `initPredefined$1`
1650. `entersyscallblock$4`
1651. `dispatchDeliveries$1$1`
1652. `fixedHuffmanDecoderInit`
1653. `init\#1`
1654. `queuedNewConn$2$2`
1655. `libc\_listen\_trampoline`
1656. `Add$1`
1657. `Find$1`
1658. `runCmdContext$3`
1659. `finalClose$1`
1660. `fprint$1`
1661. `defPredeclaredTypes`
1662. `trace$1`
1663. `init\#1`
1664. `libc\_connect\_trampoline`
1665. `checkTimeouts`
1666. `synctestRun$3`
1667. `pthread\_mutex\_init\_trampoline`
1668. `pinConnectionToCursor$bound`
1669. `ForEachPackage$1`
1670. `decap$2`
1671. `doCall$2$1`
1672. `checkMinIdleConns$1$1`
1673. `ConsumeWithContext$1`
1674. `getDockerAuthConfigs$3`
1675. `init\#1`
1676. `vgetrandomInit`
1677. `onceSetNextProtoDefaults$bound`
1678. `New\[hash.Hash\]$1`
1679. `runtime\_debug\_WriteHeapDump$1`
1680. `init\#1`
1681. `init$6`
1682. `panicunsafestringlen`
1683. `MarshalAppendWithContext$1`
1684. `libc\_readlinkat\_trampoline`
1685. `file\_google\_protobuf\_any\_proto\_rawDescGZIP$1`
1686. `libc\_settimeofday\_trampoline`
1687. `init\#1`
1688. `startGracefulShutdown$bound`
1689. `reader$1`
1690. `isAbstractSocketExists$1`
1691. `Run$1`
1692. `legacyContainerWait$1`
1693. `rollback$1`
1694. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnMissingSignature$3`
1695. `setDITDisabled`
1696. `init\#5`
1697. `ShouldPanic$1`
1698. `file\_google\_protobuf\_any\_proto\_init`
1699. `syscall\_runtime\_AfterFork`
1700. `panicExtend`
1701. `interfaceType$2`
1702. `init$1$1`
1703. `setGCPercent$1`
1704. `libc\_dup2\_trampoline`
1705. `getcp1253$1`
1706. `libc\_select\_trampoline`
1707. `getDefaultMetricsFactory$1`
1708. `sendBatchExtendedWithDescription$2$1`
1709. `hasVarSize$1`
1710. `stdin$1`
1711. `freeDeadSpanSPMCs`
1712. `decap$1`
1713. `init\#5`
1714. `init\#1`
1715. `RegisterTraceServiceHandlerFromEndpoint$1$1`
1716. `Parse$1`
1717. `GC`
1718. `resolveUnderlying$1`
1719. `collectMethods$1`
1720. `entersyscallblock$5`
1721. `init$1`
1722. `main`
1723. `nextBlock$1$2$1`
1724. `x509\_SecTrustSetVerifyDate\_trampoline`
1725. `newReaper$2`
1726. `close$bound`
1727. `printnl`
1728. `checkGenericIsExpected`
1729. `writeBodyStream$1`
1730. `entersyscallblock$1`
1731. `HandleStreams$2`
1732. `EnableRandPool`
1733. `Record$1`
1734. `libc\_getaddrinfo\_trampoline`
1735. `lazyInit$1`
1736. `commitAttemptLocked$bound`
1737. `readLoop$1`
1738. `panicshift`
1739. `MustExtractDockerSocket$1`
1740. `update$2`
1741. `NewPeriodicReader$2$1`
1742. `AddSet$1`
1743. `startStreamDecoder$2`
1744. `fcntl\_trampoline`
1745. `lazyInit$1`
1746. `run$1`
1747. `walltime\_trampoline`
1748. `panicmem`
1749. `Add$1`
1750. `ResetAssertionMetrics`
1751. `init\#1`
1752. `execDC$1`
1753. `TestScanRows$1$1`
1754. `closeReqBodyLocked$1`
1755. `newSession$1`
1756. `metricsUnlock`
1757. `init\#1`
1758. `initRequestHandler$bound`
1759. `moduledataverify`
1760. `init\#2`
1761. `watchMaster`
1762. `HandleStreams$1`
1763. `osinit`
1764. `gcStart$2`
1765. `processSubAppsRoutes$1`
1766. `RegisterLogsServiceHandlerFromEndpoint$1`
1767. `unreachableMethod`
1768. `writeStatus$1`
1769. `osInit`
1770. `syscall\_x509`
1771. `Add$1`
1772. `initMimeUnix`
1773. `start$1`
1774. `init\#1`
1775. `init\#2`
1776. `mProf\_PostSweep`
1777. `dialCtx$2`
1778. `dropm`
1779. `gcWriteBarrier4`
1780. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P521Point\]\]$1$1`
1781. `check`
1782. `init\#1`
1783. `AddSet$1`
1784. `unregisterAllDrivers`
1785. `wbBufFlush$1`
1786. `lazyInit$1`
1787. `init$13`
1788. `init\#2`
1789. `TestRabbitMQAdapter\_Shutdown\_WaitsForConsumers$1`
1790. `aggregate$1`
1791. `TestRabbitMQAdapter\_ProcessDelivery\_HandlesNonStringHeaderID$3`
1792. `init\#3`
1793. `run$1`
1794. `dumpparams`
1795. `metricsLock`
1796. `getcp1250$1`
1797. `setPossiblyUnhashableKey$1`
1798. `attemptArgMatch$1`
1799. `OnceValue\[\[\]string\]$1$1$1`
1800. `libc\_arc4random\_buf\_trampoline`
1801. `stkobjinit`
1802. `runHandler$1`
1803. `init\#1`
1804. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnMissingSignature$1`
1805. `init\#5`
1806. `runHandlers`
1807. `init\#1`
1808. `reentersyscall$3`
1809. `main`
1810. `init\#1`
1811. `operateHeaders$4`
1812. `onceSetNextProtoDefaults$bound`
1813. `wirep$2`
1814. `gcPrepareMarkRoots`
1815. `Record$1`
1816. `stackinit`
1817. `init\#1`
1818. `newClientStreamWithParams$4`
1819. `file\_opentelemetry\_proto\_metrics\_v1\_metrics\_proto\_init`
1820. `fuzz$1`
1821. `processUnaryRPC$1`
1822. `hostLookupOrder$1`
1823. `initConnPool$bound`
1824. `dumpobjs`
1825. `cleanup$1`
1826. `init\#1`
1827. `goroutineLeakGC`
1828. `nop`
1829. `StartWithGracefulShutdown$1`
1830. `parse$1`
1831. `templateThread`
1832. `dumproots`
1833. `launch$1`
1834. `libc\_kqueue\_trampoline`
1835. `New\[hash.Hash\]$1$1`
1836. `newproc$1`
1837. `scan$3$1`
1838. `init\#4`
1839. `file\_google\_protobuf\_struct\_proto\_rawDescGZIP$1`
1840. `logCloseHangDebugInfo$bound`
1841. `doRetryNotify\[\*github.com/testcontainers/testcontainers-go.DockerContainer\]$1`
1842. `libc\_munmap\_trampoline`
1843. `connect$2`
1844. `worker$2`
1845. `Alignof$1`
1846. `ExportPair$2`
1847. `main`
1848. `Execute$1`
1849. `TestScanColumns$3$1`
1850. `interrupt`
1851. `libc\_getpid\_trampoline`
1852. `init\#1`
1853. `Add$1`
1854. `Do$1`
1855. `init\#1`
1856. `minit`
1857. `handleSettings$1$1`
1858. `Breakpoint`
1859. `closeStream$1`
1860. `Free$bound`
1861. `SnapshotCoverage`
1862. `setCommandValueReflection$1`
1863. `log$1`
1864. `SafeGo$1`
1865. `NewClient$1`
1866. `lsandoleakcheck`
1867. `Setenv$2`
1868. `handleForwards$bound`
1869. `deferreturn`
1870. `x509\_CFStringCreateWithBytes\_trampoline`
1871. `TestRabbitMQAdapter\_ProcessDelivery\_AckError$1`
1872. `HandleCancel$1`
1873. `Collect$1`
1874. `getcp932$1`
1875. `runCmdContext$1`
1876. `open$1`
1877. `gcWriteBarrier2`
1878. `traceLockInit`
1879. `init\#2`
1880. `defaultUsage$bound`
1881. `ClearCache`
1882. `main`
1883. `keccakF1600Generic$1`
1884. `unpack$1`
1885. `init\#1`
1886. `initPredefined`
1887. `init\#1`
1888. `SendMsg$1`
1889. `libc\_sendfile\_trampoline`
1890. `mProf\_NextCycle`
1891. `init\#1`
1892. `init\#1`
1893. `osinit\_hack`
1894. `file\_google\_protobuf\_field\_mask\_proto\_init`
1895. `init\#2`
1896. `file\_grpc\_binlog\_v1\_binarylog\_proto\_rawDescGZIP$1`
1897. `exitsyscall$3`
1898. `SetTextMapPropagator$1`
1899. `doBlockingWithCtx$1`
1900. `libc\_getgrnam\_r\_trampoline`
1901. `InitializeTelemetry$1`
1902. `gcAssistAlloc$1`
1903. `forEachP$1`
1904. `traceAdvance$6`
1905. `WithDeadlineCause$2`
1906. `New$1$1`
1907. `doinit`
1908. `NewController$1`
1909. `dumpitabs`
1910. `getCaller$1`
1911. `panicmakeslicelen`
1912. `init\#2`
1913. `init$1`
1914. `startFlushGoroutine$1`
1915. `libc\_getpwnam\_r\_trampoline`
1916. `init\#1`
1917. `readARM64Registers`
1918. `init\#1`
1919. `processDelivery$1`
1920. `init\#1`
1921. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P384Point\]\]$1$1`
1922. `TestExtractExternalData\_SuccessWithNonFatalWarnings$1`
1923. `OnceValues\[map\[string\]rune map\[string\]\[2\]rune\]$1$1`
1924. `connect$1`
1925. `init\#1`
1926. `AddSet$1`
1927. `init$1`
1928. `libc\_setprivexec\_trampoline`
1929. `init\#15`
1930. `block`
1931. `sigpipe`
1932. `init\#2`
1933. `allPackages$1$1`
1934. `onSettingsTimer$bound`
1935. `TryGo$1$1`
1936. `asyncPreempt2`
1937. `poll\_runtime\_pollServerInit`
1938. `setMinimalFeatures`
1939. `Go$1$1`
1940. `libc\_getgroups\_trampoline`
1941. `consumeAddrSpec$1`
1942. `confirmLocks$1`
1943. `Goexit`
1944. `libc\_mknod\_trampoline`
1945. `log$1`
1946. `\_LostSIGPROFDuringAtomic64`
1947. `x509\_SecTrustCreateWithCertificates\_trampoline`
1948. `resolve$1`
1949. `x509\_SecTrustCopyCertificateChain\_trampoline`
1950. `pthread\_mutex\_lock\_trampoline`
1951. `goroutineProfileWithLabelsConcurrent$1`
1952. `newstack`
1953. `checkResponseErr$1`
1954. `subscribedState$1`
1955. `scan$4$1`
1956. `libc\_mlockall\_trampoline`
1957. `init$2`
1958. `RecordNonApproved`
1959. `New\[\*crypto/internal/fips140/sha256.Digest\]$1$1`
1960. `start$1`
1961. `SendMsg$2`
1962. `wakep`
1963. `init\#1`
1964. `entersyscallblock`
1965. `maybeRunChan$1`
1966. `encap$1`
1967. `traceExitingSyscall`
1968. `mstart1`
1969. `init\#1`
1970. `RegisterFunc$4$5`
1971. `tRunner$1$1`
1972. `New$1`
1973. `clearpools`
1974. `init\#1`
1975. `OnceValue$1$1`
1976. `three$1`
1977. `selectgo$3`
1978. `init\#1`
1979. `x509\_CFDictionaryGetValueIfPresent\_trampoline`
1980. `startGracefulShutdown$1`
1981. `bgRead$1`
1982. `goargs`
1983. `libc\_gettimeofday\_trampoline`
1984. `runPerThreadSyscall`
1985. `gcBgMarkPrepare`
1986. `merge$2`
1987. `init\#3`
1988. `log`
1989. `TestSetConfigFromEnvVars$1$1`
1990. `tunnel$1`
1991. `handshakeContext$2`
1992. `init$6`
1993. `gcWakeAllAssists`
1994. `startChannelWatcher$1`
1995. `typeDecl$2`
1996. `fatalpanic$2`
1997. `NewStreamReader$1`
1998. `ClientHandshake$1`
1999. `init\#1`
2000. `libc\_pipe\_trampoline`
2001. `CoordinateFuzzing$3`
2002. `Disable`
2003. `MasterAddr$1$1`
2004. `init\#8`
2005. `runtime\_doSpin`
2006. `Sync$1$1`
2007. `pthread\_cond\_signal\_trampoline`
2008. `main`
2009. `set\_crosscall2`
2010. `EndTracingSpans$1`
2011. `onDemandWorker$1`
2012. `doInRoot\[int\]$1`
2013. `PluginInstall$1`
2014. `DialContext$2`
2015. `runPiped$1`
2016. `libc\_unlockpt\_trampoline`
2017. `init\#1`
2018. `frameSkip$1`
2019. `lazyInit$1`
2020. `GetValue$1`
2021. `RegisterAsyncReporter$1`
2022. `initFeistelBox`
2023. `ClearNumberValidations$1`
2024. `assertWorldStopped`
2025. `shutdownWorkers$1`
2026. `libc\_getpgrp\_trampoline`
2027. `libc\_lstat\_trampoline`
2028. `DecodeFull$1`
2029. `main`
2030. `libc\_mprotect\_trampoline`
2031. `signalWaitUntilIdle`
2032. `close\_trampoline`
2033. `Execute$1`
2034. `run$4`
2035. `RecordSet$1`
2036. `init\#1`
2037. `abortStreamLocked$1`
2038. `TestQueryPluginCRM\_WithFilters$1`
2039. `initSystemRoots`
2040. `init\#5`
2041. `connStmt$1`
2042. `init\#2`
2043. `close$1`
2044. `init\#9`
2045. `handleLock$1`
2046. `runtime\_debug\_freeOSMemory`
2047. `nextBlock$1$2`
2048. `loadPackageNames$1`
2049. `SaveMultipartFile$2`
2050. `init\#1`
2051. `setupFallbackCache$1`
2052. `scheduleNextConnectionLocked$1`
2053. `startCheckmarks`
2054. `onReadTimeout$bound`
2055. `libc\_unlinkat\_trampoline`
2056. `AppendCertsFromPEM$1$1`
2057. `LoadLocation$1`
2058. `SetFallbackRoots$1`
2059. `handleSettings$1$1`
2060. `computeInterfaceTypeSet$2$1`
2061. `getcp1252$1`
2062. `shutdown$1`
2063. `init\#14`
2064. `search$1`
2065. `libc\_fsync\_trampoline`
2066. `TryAcquire\[\*github.com/jackc/pgx/v5/pgxpool.connResource\]$1`
2067. `\_LostExternalCode`
2068. `init$3`
2069. `probe$bound`
2070. `pthread\_cond\_wait\_trampoline`
2071. `mlock\_trampoline`
2072. `main`
2073. `noteUnusedDriverStatement$1`
2074. `syscall\_runtime\_BeforeFork`
2075. `Add$1`
2076. `main`
2077. `readServices`
2078. `getOrQueueForIdleConn$1`
2079. `libc\_symlinkat\_trampoline`
2080. `libc\_fstatfs\_trampoline`
2081. `SetFinalizer$2`
2082. `file\_google\_rpc\_status\_proto\_init`
2083. `init\#1`
2084. `main`
2085. `finishDebugVarsSetup`
2086. `setBypass`
2087. `execDC$3`
2088. `HandleStreams$1`
2089. `persistentalloc$1`
2090. `copyTrailersToHandlerRequest$bound`
2091. `Format$1`
2092. `clearSignalHandlers`
2093. `initAllChan$1`
2094. `Consume$1`
2095. `lazyInit$1`
2096. `initP521`
2097. `instantiateSignature$2`
2098. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$6`
2099. `duffzero`
2100. `ServeHTTP$1`
2101. `file\_google\_api\_httpbody\_proto\_rawDescGZIP$1`
2102. `initP384`
2103. `usleep\_trampoline`
2104. `init\#1`
2105. `gcMarkDone$3`
2106. `libc\_setlogin\_trampoline`
2107. `TraceQueryute$1`
2108. `dialSingle$1`
2109. `init\#1`
2110. `grabConn$1`
2111. `SetLoggerProvider$1`
2112. `computeInterfaceTypeSet$1`
2113. `NewPublicKey$1`
2114. `init\#1`
2115. `libc\_fchflags\_trampoline`
2116. `doBlockingWithCtx\[\[\]net.IPAddr\]$1`
2117. `stopm`
2118. `init$4`
2119. `getcp437$1`
2120. `modify$1`
2121. `processUnaryRPC$2`
2122. `traceQuery$1`
2123. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P224Point\]\]$1$1`
2124. `newClientStream$5`
2125. `Run$1`
2126. `getMacOSMajor$1`
2127. `SetTracerProvider$1`
2128. `nextBlock$1$1`
2129. `init\#1`
2130. `panicUnaligned`
2131. `init\#1`
2132. `Sync$1`
2133. `main`
2134. `growSlice$1`
2135. `init\#1`
2136. `Setenv$1`
2137. `TestWithSwaggerEnvConfig\_WithEnvVars$1`
2138. `newNonRetryClientStream$1`
2139. `gfget$2`
2140. `onIdleTimeout$bound`
2141. `forceCloseConn$bound`
2142. `parseFieldName$1`
2143. `failWantMap`
2144. `lookupIPAddr$3`
2145. `collectRecv$1`
2146. `closeConnIfStillIdle$bound`
2147. `munmap\_trampoline`
2148. `Init`
2149. `sigaltstack\_trampoline`
2150. `init\#1`
2151. `p224B$1`
2152. `EventuallyWithT$1`
2153. `Shutdown$1`
2154. `forceCloseConn$bound`
2155. `stopTheWorld$1`
2156. `cgoCheckPtrWrite$1`
2157. `init\#1`
2158. `init\#1`
2159. `stdoutHandler$1`
2160. `init\#2`
2161. `open\_trampoline`
2162. `handleCleanCache$1`
2163. `nify$1`
2164. `main`
2165. `TestEnsureConfigFromEnvVars$2$1`
2166. `collect$1`
2167. `gcWriteBarrier8`
2168. `proc\_regionfilename\_trampoline`
2169. `RangeFields$1`
2170. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P224Point\]\]$1$1$1`
2171. `setUserArenaChunkToFault$1`
2172. `TestRabbitMQAdapter\_ConsumerLoop\_AcksOnSuccess$1$1`
2173. `libc\_fstatat\_trampoline`
2174. `x509\_CFErrorGetCode\_trampoline`
2175. `typelinksinit`
2176. `Multiplexed$1$1`
2177. `marshal$1`
2178. `setRequestCancel$4`
2179. `addTLS$2`
2180. `libc\_getpwuid\_r\_trampoline`
2181. `ExportPair$1`
2182. `onReadTimeout$bound`
2183. `Eventually$1`
2184. `x509\_CFErrorCopyDescription\_trampoline`
2185. `awaitGoroutines$1`
2186. `servePeer$2`
2187. `netpollGenericInit`
2188. `Parse`
2189. `AcquireConn$1`
2190. `sigpanic0`
2191. `Stop`
2192. `x509\_CFEqual\_trampoline`
2193. `infer$2`
2194. `healthCheck$1`
2195. `RunConsumers$1`
2196. `main\_main`
2197. `getMetadata$1`
2198. `sync\_runtime\_doSpin`
2199. `validCycle$1`
2200. `funcDecl$1`
2201. `search$2$1`
2202. `init\#1`
2203. `raise\_trampoline`
2204. `handleMonitoredClose$1`
2205. `libc\_flock\_trampoline`
2206. `Close$bound`
2207. `finishStream$1`
2208. `traceStartReadCPU$1`
2209. `defaultCleanUp$1`
2210. `libc\_adjtime\_trampoline`
2211. `traceAdvance$3`
2212. `AddSet$1`
2213. `casgstatus$3`
2214. `readTrace0$1`
2215. `OnEmit$1`
2216. `getPossiblyUnhashableKey$1`
2217. `fixedHuffmanDecoderInit$1`
2218. `init$5`
2219. `ensureMetricsCollector$1`
2220. `defaultGOMAXPROCSInit`
2221. `buildImageWithSecrets$1`
2222. `unspillArgs`
2223. `init\#1`
2224. `osyield\_no\_g`
2225. `file\_opentelemetry\_proto\_collector\_metrics\_v1\_metrics\_service\_proto\_init`
2226. `TestWithSwaggerEnvConfig\_InvalidHost$1`
2227. `unpinConnectionFromTransaction$bound`
2228. `x509\_SecTrustEvaluate\_trampoline`
2229. `Chdir$1`
2230. `runHandler$1`
2231. `CleanupNetwork$1`
2232. `alginit`
2233. `libc\_getsockname\_trampoline`
2234. `libc\_accept\_trampoline`
2235. `overrideUmask$1`
2236. `libc\_mkfifo\_trampoline`
2237. `Clearenv`
2238. `ensureSensitiveFieldsMap$1`
2239. `init$1`
2240. `init\#1`
2241. `init\#1`
2242. `Read$1`
2243. `doBlockingWithCtx\[int\]$1`
2244. `init\#1`
2245. `Close$1`
2246. `RecordSet$1`
2247. `FIPSOnly`
2248. `EncodeAll$1`
2249. `init\#1`
2250. `secretEraseRegisters`
2251. `copyTrailers$bound`
2252. `run$3`
2253. `OrderedMarshal$1`
2254. `init\#10`
2255. `Execute$1`
2256. `file\_google\_protobuf\_timestamp\_proto\_rawDescGZIP$1`
2257. `libc\_seteuid\_trampoline`
2258. `Execute$2`
2259. `generatorTable$1`
2260. `dumpgs`
2261. `parseExpr$1`
2262. `mmap\_trampoline`
2263. `RecordSet$1`
2264. `expandRHS$1`
2265. `nilfunc`
2266. `processPipelineNode$1$1`
2267. `optionsUnmarshaler$1$1`
2268. `OnceValue\[string\]$1$1$1`
2269. `init\#1`
2270. `freeOSMemory`
2271. `decryptKey$1`
2272. `libc\_getegid\_trampoline`
2273. `worker$1`
2274. `libc\_lchown\_trampoline`
2275. `pthread\_key\_create\_trampoline`
2276. `debugCallV2`
2277. `main`
2278. `mergeRuneSets$1`
2279. `main`
2280. `dispatchDeliveries$1`
2281. `lazyInit$1`
2282. `init\#3`
2283. `init\#3`
2284. `mPark`
2285. `libc\_execve\_trampoline`
2286. `init\#1`
2287. `lockProfiles`
2288. `badcgocallback`
2289. `debugCallCheck$1`
2290. `dolockOSThread`
2291. `newHealthData$1`
2292. `startServers$3`
2293. `asyncClose$1`
2294. `libc\_fchdir\_trampoline`
2295. `startWatcher$1`
2296. `gcMarkRootCheck`
2297. `signalWaitUntilIdle`
2298. `updateServerDate$1`
2299. `init$1$1`
2300. `racefingo`
2301. `WithDataIndependentTiming$1`
2302. `instantiateSignature$1`
2303. `init\#1`
2304. `onIdleTimer$bound`
2305. `dialConn$3`
2306. `Add$1`
2307. `init\#1`
2308. `morestack\_noctxt`
2309. `enableWER`
2310. `initialize$1`
2311. `gcMarkDone$4`
2312. `libc\_getpriority\_trampoline`
2313. `updateServerDate`
2314. `rt0\_go`
2315. `x509\_CFDateCreate\_trampoline`
2316. `init$1`
2317. `init\#1`
2318. `init\#1`
2319. `Wait`
2320. `buildCommonHeaderMapsOnce`
2321. `allocmcache$1`
2322. `traceDataRow$1`
2323. `buildImageWithSecrets$1`
2324. `Raw$1`
2325. `OnceFunc$1`
2326. `Close$1`
2327. `newCompressedFSFileCache$1`
2328. `releaseForkLock`
2329. `\*github.com/LerianStudio/lib-commons/v4/commons/mongo.Client.Client`
2330. `\*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.Client`
2331. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockmongoDatabaseProvider.Client`
2332. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.stubMongoDatabaseProvider.Client`
2333. `github.com/LerianStudio/fetcher/pkg/mongodb/product.stubMongoDatabaseProvider.Client`
2334. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockmongoDatabaseProvider.Client`
2335. `\*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.Client`

#### `TestJobMongoDBRepository\_List`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`
5. `\*testing.T.Run`

#### `TestJobMongoDBRepository\_ExistsRunningByMappedFieldKey`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`
5. `\*testing.T.Run`
6. `\*testing.T.Run`
7. `\*testing.T.Run`
8. `\*testing.T.Run`

#### `TestJobMongoDBRepository\_Update`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`
5. `\*testing.T.Run`
6. `\*testing.T.Run`
7. `\*testing.T.Run`

#### `TestJobMongoDBRepository\_FindByID`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`

#### `TestJobMongoDBRepository\_UpdateStatus`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`
5. `\*testing.T.Run`
6. `\*testing.T.Run`
7. `\*testing.T.Run`
8. `\*testing.T.Run`

#### `TestDropIndexesDatabaseError`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockMongoClientProvider`
4. `\*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.EXPECT`
5. `Any`
6. `\*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProviderMockRecorder.Client`
7. `New`
8. `\*go.uber.org/mock/gomock.Call.Return`
9. `Background`
10. `\*github.com/LerianStudio/fetcher/pkg/mongodb/job.JobMongoDBRepository.DropIndexes`
11. `\*testing.common.Fatalf`
12. `\*github.com/pkg/errors.withMessage.Error`
13. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationError.Error`
14. `\*github.com/docker/docker/client.httpError.Error`
15. `\*runtime.boundsError.Error`
16. `\*gopkg.in/go-playground/validator.v9.InvalidValidationError.Error`
17. `go.mongodb.org/mongo-driver/mongo/options.MarshalError.Error`
18. `\*go.mongodb.org/mongo-driver/internal/codecutil.MarshalError.Error`
19. `github.com/docker/docker/api/types.ErrorResponse.Error`
20. `\*go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt.Error.Error`
21. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationError.Error`
22. `github.com/docker/docker/client.emptyIDError.Error`
23. `\*github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
24. `vendor/golang.org/x/net/http2/hpack.DecodingError.Error`
25. `\*net/http.unsupportedTEError.Error`
26. `\*net/http.http2headerFieldNameError.Error`
27. `\*github.com/docker/docker/api/types/network.joinError.Error`
28. `\*go.mongodb.org/mongo-driver/x/mongo/driver/auth.Error.Error`
29. `\*net.ParseError.Error`
30. `github.com/go-openapi/swag/yamlutils.yamlError.Error`
31. `runtime.errorAddressString.Error`
32. `\*net.InvalidAddrError.Error`
33. `\*crypto/x509.UnknownAuthorityError.Error`
34. `\*github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
35. `golang.org/x/crypto/ssh.ServerAuthError.Error`
36. `\*github.com/go-playground/validator/v10.ValidationErrors.Error`
37. `golang.org/x/net/http2.pseudoHeaderError.Error`
38. `golang.org/x/crypto/ocsp.ResponseError.Error`
39. `go.mongodb.org/mongo-driver/x/mongo/driver.QueryFailureError.Error`
40. `github.com/docker/docker/api/types/container.errInvalidParameter.Error`
41. `\*net/http.http2pseudoHeaderError.Error`
42. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.MalformedElementError.Error`
43. `\*crypto/tls.alert.Error`
44. `\*crypto/x509.InsecureAlgorithmError.Error`
45. `go.mongodb.org/mongo-driver/mongo.WriteErrors.Error`
46. `\*encoding/asn1.invalidUnmarshalError.Error`
47. `crypto/x509.UnhandledCriticalExtension.Error`
48. `go.mongodb.org/mongo-driver/bson/bsoncodec.ValueDecoderError.Error`
49. `github.com/containerd/errdefs.customMessage.Error`
50. `\*github.com/xdg-go/stringprep.Error.Error`
51. `\*crypto/tls.CertificateVerificationError.Error`
52. `\*fmt.wrapError.Error`
53. `\*github.com/yuin/gopher-lua/pm.Error.Error`
54. `github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
55. `\*net/http.http2headerFieldValueError.Error`
56. `golang.org/x/net/http2.duplicatePseudoHeaderError.Error`
57. `os.errSymlink.Error`
58. `vendor/golang.org/x/net/idna.labelError.Error`
59. `\*crypto/tls.RecordHeaderError.Error`
60. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoEncoder.Error`
61. `\*golang.org/x/mod/module.ModuleError.Error`
62. `\*encoding/json.UnmarshalTypeError.Error`
63. `github.com/containerd/errdefs.errInvalidArgument.Error`
64. `\*google.golang.org/protobuf/internal/errors.prefixError.Error`
65. `\*encoding/base64.CorruptInputError.Error`
66. `\*golang.org/x/mod/module.InvalidPathError.Error`
67. `\*go.mongodb.org/mongo-driver/x/mongo/driver.ResponseError.Error`
68. `net/netip.parseAddrError.Error`
69. `go/scanner.Error.Error`
70. `\*internal/poll.DeadlineExceededError.Error`
71. `\*golang.org/x/crypto/ssh.ExitMissingError.Error`
72. `time.fileSizeError.Error`
73. `\*github.com/redis/go-redis/v9/internal/proto.RedisError.Error`
74. `golang.org/x/net/http2.ConnectionError.Error`
75. `github.com/LerianStudio/lib-commons/v2/commons.Response.Error`
76. `go.mongodb.org/mongo-driver/internal/codecutil.MarshalError.Error`
77. `golang.org/x/net/http2/hpack.InvalidIndexError.Error`
78. `\*strconv.NumError.Error`
79. `\*go.opentelemetry.io/otel/sdk/trace.samplerArgParseError.Error`
80. `github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
81. `\*os/user.UnknownUserIdError.Error`
82. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteConcernError.Error`
83. `net/http.nothingWrittenError.Error`
84. `\*github.com/redis/go-redis/v9/internal/proto.MasterDownError.Error`
85. `\*go/scanner.ErrorList.Error`
86. `\*go.mongodb.org/mongo-driver/mongo.WriteException.Error`
87. `github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
88. `\*github.com/docker/docker/client.errConnectionFailed.Error`
89. `\*vendor/golang.org/x/net/dns/dnsmessage.nestedError.Error`
90. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InvalidDepthTraversalError.Error`
91. `net/http.requestBodyReadError.Error`
92. `net/http.tlsHandshakeTimeoutError.Error`
93. `\*github.com/pkg/errors.fundamental.Error`
94. `\*github.com/redis/go-redis/v9/internal/proto.MovedError.Error`
95. `\*golang.org/x/net/http2.goAwayFlowError.Error`
96. `\*github.com/containerd/errdefs.errPermissionDenied.Error`
97. `github.com/microsoft/go-mssqldb.RetryableError.Error`
98. `\*github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
99. `\*github.com/redis/go-redis/v9/internal/proto.ExecAbortError.Error`
100. `golang.org/x/text/encoding/internal.RepertoireError.Error`
101. `github.com/montanaflynn/stats.statsError.Error`
102. `\*gopkg.in/go-playground/validator.v9.fieldError.Error`
103. `\*github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
104. `\*net/http.http2GoAwayError.Error`
105. `github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
106. `github.com/docker/docker/client.objectNotFoundError.Error`
107. `\*net/http.nothingWrittenError.Error`
108. `\*github.com/montanaflynn/stats.statsError.Error`
109. `\*github.com/redis/go-redis/v9/push.ProcessorError.Error`
110. `\*go.mongodb.org/mongo-driver/x/mongo/driver/ocsp.Error.Error`
111. `golang.org/x/net/http2.noCachedConnError.Error`
112. `crypto/x509.ConstraintViolationError.Error`
113. `github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
114. `github.com/LerianStudio/lib-commons/v4/commons.Response.Error`
115. `net/http.http2StreamError.Error`
116. `github.com/LerianStudio/lib-license-go/v2/pkg.ResponseError.Error`
117. `\*os.SyscallError.Error`
118. `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc/internal.PartialSuccess.Error`
119. `crypto/tls.alert.Error`
120. `\*vendor/golang.org/x/net/idna.labelError.Error`
121. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteErrors.Error`
122. `\*net/netip.parseAddrError.Error`
123. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ElementTypeError.Error`
124. `vendor/golang.org/x/net/http2/hpack.InvalidIndexError.Error`
125. `\*github.com/containerd/errdefs.errNotModified.Error`
126. `\*net.temporaryError.Error`
127. `\*github.com/jackc/pgx/v5/pgconn.perDialConnectError.Error`
128. `\*go.mongodb.org/mongo-driver/mongo.BulkWriteError.Error`
129. `\*github.com/jackc/pgx/v5/pgconn.PgError.Error`
130. `github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
131. `go.mongodb.org/mongo-driver/bson/bsonrw.TransitionError.Error`
132. `\*go/types.ArgumentError.Error`
133. `github.com/containerd/errdefs.errNotImplemented.Error`
134. `\*gopkg.in/go-playground/validator.v9.ValidationErrors.Error`
135. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.DecodeError.Error`
136. `\*github.com/containerd/errdefs/pkg/internal/cause.ErrUnexpectedStatus.Error`
137. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ElementTypeError.Error`
138. `google.golang.org/grpc/internal/transport.ioError.Error`
139. `net/url.InvalidHostError.Error`
140. `golang.org/x/net/http2.StreamError.Error`
141. `\*github.com/jackc/pgx/v5/pgtype.nullAssignmentError.Error`
142. `\*github.com/redis/go-redis/v9/internal/proto.AuthError.Error`
143. `\*github.com/microsoft/go-mssqldb.RetryableError.Error`
144. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.ServerSelectionError.Error`
145. `internal/runtime/maps.unhashableTypeError.Error`
146. `\*google.golang.org/protobuf/internal/errors.SizeMismatchError.Error`
147. `go.mongodb.org/mongo-driver/x/mongo/driver.Error.Error`
148. `\*crypto/rc4.KeySizeError.Error`
149. `github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
150. `github.com/gofiber/fiber/v2/internal/schema.ConversionError.Error`
151. `\*github.com/valyala/fasthttp.ErrBrokenChunk.Error`
152. `google.golang.org/grpc/internal/transport.ConnectionError.Error`
153. `\*github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
154. `\*github.com/redis/go-redis/v9/internal/proto.OOMError.Error`
155. `\*golang.org/x/net/http2/hpack.DecodingError.Error`
156. `\*crypto/tls.ECHRejectionError.Error`
157. `crypto/aes.KeySizeError.Error`
158. `\*context.deadlineExceededError.Error`
159. `\*vendor/golang.org/x/net/http2/hpack.DecodingError.Error`
160. `github.com/xdg-go/stringprep.Error.Error`
161. `net/http.http2noCachedConnError.Error`
162. `\*vendor/golang.org/x/net/idna.runeError.Error`
163. `golang.org/x/net/http2/hpack.DecodingError.Error`
164. `\*github.com/jackc/pgx/v5/pgconn.contextAlreadyDoneError.Error`
165. `\*github.com/grpc-ecosystem/grpc-gateway/v2/runtime.HTTPStatusError.Error`
166. `\*debug/dwarf.DecodeError.Error`
167. `\*crypto/internal/fips140/hmac.errCloneUnsupported.Error`
168. `github.com/LerianStudio/lib-license-go/v2/pkg.HTTPError.Error`
169. `go.mongodb.org/mongo-driver/mongo.WriteException.Error`
170. `github.com/go-openapi/swag/jsonutils/adapters/stdlib/json.jsonError.Error`
171. `\*text/template.ExecError.Error`
172. `\*github.com/go-sql-driver/mysql.MySQLError.Error`
173. `github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
174. `\*encoding/json.InvalidUTF8Error.Error`
175. `\*golang.org/x/net/webdav/internal/xml.UnmarshalError.Error`
176. `\*github.com/go-playground/universal-translator.ErrExistingTranslator.Error`
177. `\*runtime.errorString.Error`
178. `\*time.parseDurationError.Error`
179. `encoding/hex.InvalidByteError.Error`
180. `\*crypto/tls.AlertError.Error`
181. `net/textproto.ProtocolError.Error`
182. `\*github.com/containerd/errdefs.errResourceExhausted.Error`
183. `\*crypto/des.KeySizeError.Error`
184. `github.com/moby/term.EscapeError.Error`
185. `\*github.com/LerianStudio/lib-commons/v4/commons/rabbitmq.sanitizedError.Error`
186. `compress/bzip2.StructuralError.Error`
187. `\*time.fileSizeError.Error`
188. `\*golang.org/x/net/http2.ConnectionError.Error`
189. `\*github.com/testcontainers/testcontainers-go.ParallelContainersError.Error`
190. `\*golang.org/x/net/http2.httpError.Error`
191. `\*github.com/gofiber/fiber/v2.Error.Error`
192. `go.mongodb.org/mongo-driver/mongo.EncryptionKeyVaultError.Error`
193. `github.com/go-openapi/jsonpointer.pointerError.Error`
194. `\*compress/flate.InternalError.Error`
195. `github.com/valyala/fasthttp.ErrNothingRead.Error`
196. `\*github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
197. `\*encoding/xml.UnmarshalError.Error`
198. `\*github.com/go-playground/validator/v10.InvalidValidationError.Error`
199. `github.com/containerd/errdefs.errConflict.Error`
200. `github.com/LerianStudio/lib-commons/v4/commons/net/http.ErrorResponse.Error`
201. `google.golang.org/protobuf/internal/impl.errInvalidUTF8.Error`
202. `github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
203. `\*internal/reflectlite.ValueError.Error`
204. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteCommandError.Error`
205. `github.com/go-openapi/swag/loading.loadingError.Error`
206. `\*github.com/gofiber/fiber/v2/internal/schema.MultiError.Error`
207. `\*google.golang.org/protobuf/internal/errors.wrapError.Error`
208. `\*runtime.TypeAssertionError.Error`
209. `\*crypto/x509.CertificateInvalidError.Error`
210. `\*github.com/LerianStudio/lib-license-go/v2/pkg.UnprocessableOperationError.Error`
211. `go.mongodb.org/mongo-driver/internal/aws/awserr.baseError.Error`
212. `golang.org/x/crypto/blowfish.KeySizeError.Error`
213. `github.com/valyala/fasthttp.EscapeError.Error`
214. `\*golang.org/x/crypto/ssh.cbcError.Error`
215. `\*github.com/go-playground/universal-translator.ErrCardinalTranslation.Error`
216. `\*github.com/valyala/fasthttp.EscapeError.Error`
217. `\*errors.joinError.Error`
218. `\*github.com/redis/go-redis/v9/internal/proto.ReadOnlyError.Error`
219. `\*github.com/valyala/fasthttp.ErrBodyStreamWritePanic.Error`
220. `\*crypto/tls.permanentError.Error`
221. `encoding/json.jsonError.Error`
222. `\*go.mongodb.org/mongo-driver/x/mongo/driver.InvalidOperationError.Error`
223. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.PoolError.Error`
224. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.PoolError.Error`
225. `golang.org/x/net/http2.headerFieldNameError.Error`
226. `\*github.com/go-logr/logr.notFoundError.Error`
227. `\*crypto/x509.ConstraintViolationError.Error`
228. `github.com/containerd/errdefs.errNotModified.Error`
229. `\*github.com/go-resty/resty/v2.noRetryErr.Error`
230. `\*github.com/redis/go-redis/v9/push.HandlerError.Error`
231. `\*golang.org/x/sync/singleflight.panicError.Error`
232. `\*github.com/containerd/errdefs.errUnavailable.Error`
233. `\*google.golang.org/grpc/internal/status.Error.Error`
234. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.WaitQueueTimeoutError.Error`
235. `\*github.com/containerd/errdefs.errAborted.Error`
236. `\*encoding/xml.TagPathError.Error`
237. `net/http.http2goAwayFlowError.Error`
238. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.poolClearedError.Error`
239. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoEncoder.Error`
240. `context.deadlineExceededError.Error`
241. `\*github.com/go-openapi/swag/jsonutils/adapters/stdlib/json.jsonError.Error`
242. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationUnknownFieldsError.Error`
243. `\*github.com/grpc-ecosystem/grpc-gateway/v2/internal/httprule.InvalidTemplateError.Error`
244. `crypto/x509.SystemRootsError.Error`
245. `\*github.com/containerd/errdefs.errDataLoss.Error`
246. `\*net/http.timeoutError.Error`
247. `\*github.com/jackc/pgx/v5.ScanArgError.Error`
248. `\*github.com/cenkalti/backoff/v4.PermanentError.Error`
249. `\*go.mongodb.org/mongo-driver/bson/bsonrw.TransitionError.Error`
250. `github.com/containerd/errdefs.errOutOfRange.Error`
251. `\*github.com/jackc/pgx/v5/pgproto3.ExceededMaxBodyLenErr.Error`
252. `\*go.uber.org/zap.errArrayElem.Error`
253. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.ConnectionError.Error`
254. `\*golang.org/x/crypto/ssh.ServerAuthError.Error`
255. `go.mongodb.org/mongo-driver/mongo.ErrMapForOrderedArgument.Error`
256. `golang.org/x/net/http2.headerFieldValueError.Error`
257. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.ConnectionError.Error`
258. `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.errMetric.Error`
259. `\*encoding/json.MarshalerError.Error`
260. `\*crypto/x509.UnhandledCriticalExtension.Error`
261. `\*github.com/jackc/pgx/v5/pgproto3.invalidMessageFormatErr.Error`
262. `net/url.EscapeError.Error`
263. `\*github.com/jackc/pgx/v5/pgconn.ConnectError.Error`
264. `\*github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
265. `github.com/containerd/errdefs/pkg/internal/cause.ErrUnexpectedStatus.Error`
266. `net.InvalidAddrError.Error`
267. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InsufficientBytesError.Error`
268. `\*archive/tar.headerError.Error`
269. `github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
270. `golang.org/x/net/http2.goAwayFlowError.Error`
271. `\*github.com/redis/go-redis/v9/internal/proto.LoadingError.Error`
272. `\*github.com/valyala/fasthttp.ErrSmallBuffer.Error`
273. `\*github.com/LerianStudio/lib-license-go/v2/pkg.UnauthorizedError.Error`
274. `go.mongodb.org/mongo-driver/x/mongo/driver.InvalidOperationError.Error`
275. `vendor/golang.org/x/net/idna.runeError.Error`
276. `\*github.com/Shopify/toxiproxy/v2/client.ApiError.Error`
277. `github.com/gofiber/fiber/v2/internal/schema.UnknownKeyError.Error`
278. `os/user.UnknownUserError.Error`
279. `\*os/exec.Error.Error`
280. `\*github.com/microsoft/go-mssqldb/aecmk.Error.Error`
281. `\*github.com/yuin/gopher-lua/parse.Error.Error`
282. `syscall.Errno.Error`
283. `net.canceledError.Error`
284. `\*net/http.http2noCachedConnError.Error`
285. `\*net/http.tlsHandshakeTimeoutError.Error`
286. `\*crypto/x509.HostnameError.Error`
287. `\*github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
288. `net/http.http2connError.Error`
289. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ValidationError.Error`
290. `\*golang.org/x/net/webdav/internal/xml.UnsupportedTypeError.Error`
291. `os/user.UnknownGroupIdError.Error`
292. `\*github.com/lib/pq.Error.Error`
293. `\*github.com/jackc/pgx/v5/pgconn.connLockError.Error`
294. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.poolClearedError.Error`
295. `\*go.mongodb.org/mongo-driver/mongo.WriteConcernError.Error`
296. `\*go/build/constraint.SyntaxError.Error`
297. `\*go.mongodb.org/mongo-driver/mongo.MongocryptdError.Error`
298. `\*github.com/jackc/pgx/v5.proxyError.Error`
299. `github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
300. `github.com/valyala/fasthttp.ErrBodyStreamWritePanic.Error`
301. `\*github.com/go-playground/universal-translator.ErrOrdinalTranslation.Error`
302. `\*github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
303. `net/http.http2duplicatePseudoHeaderError.Error`
304. `github.com/ebitengine/purego.Dlerror.Error`
305. `\*github.com/golang-jwt/jwt/v5.joinedError.Error`
306. `\*crypto/aes.KeySizeError.Error`
307. `\*net/http.http2StreamError.Error`
308. `golang.org/x/net/idna.runeError.Error`
309. `\*github.com/LerianStudio/lib-commons/v4/commons/net/http.ErrorResponse.Error`
310. `\*crypto/x509.SystemRootsError.Error`
311. `\*github.com/shirou/gopsutil/v4/internal/common.Warnings.Error`
312. `github.com/LerianStudio/lib-license-go/v2/pkg.UnauthorizedError.Error`
313. `\*errors.errorString.Error`
314. `github.com/LerianStudio/lib-commons/commons.Response.Error`
315. `\*golang.org/x/net/idna.runeError.Error`
316. `\*golang.org/x/net/idna.labelError.Error`
317. `\*golang.org/x/crypto/ssh.ExitError.Error`
318. `\*golang.org/x/net/http2.noCachedConnError.Error`
319. `\*reflect.ValueError.Error`
320. `net/http.http2headerFieldNameError.Error`
321. `internal/poll.errNetClosing.Error`
322. `github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
323. `\*github.com/ebitengine/purego.Dlerror.Error`
324. `\*compress/flate.WriteError.Error`
325. `github.com/microsoft/go-mssqldb.Error.Error`
326. `go/types.Error.Error`
327. `\*github.com/go-playground/universal-translator.ErrBadPluralDefinition.Error`
328. `github.com/docker/docker/client.errConnectionFailed.Error`
329. `\*github.com/LerianStudio/lib-commons/commons.Response.Error`
330. `\*net/netip.parsePrefixError.Error`
331. `go.mongodb.org/mongo-driver/mongo.BulkWriteError.Error`
332. `\*github.com/go-openapi/jsonpointer.pointerError.Error`
333. `\*net/http.http2duplicatePseudoHeaderError.Error`
334. `\*golang.org/x/net/http2.headerFieldValueError.Error`
335. `crypto/rc4.KeySizeError.Error`
336. `\*github.com/containerd/errdefs.errConflict.Error`
337. `\*github.com/gofiber/fiber/v2/internal/schema.UnknownKeyError.Error`
338. `\*github.com/containerd/errdefs.errFailedPrecondition.Error`
339. `\*golang.org/x/net/webdav/internal/xml.TagPathError.Error`
340. `github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
341. `\*golang.org/x/text/encoding/internal.RepertoireError.Error`
342. `\*go.mongodb.org/mongo-driver/mongo.WriteError.Error`
343. `github.com/rabbitmq/amqp091-go.Error.Error`
344. `compress/flate.CorruptInputError.Error`
345. `\*github.com/LerianStudio/lib-commons/v4/commons/mongo.SanitizedError.Error`
346. `\*go.mongodb.org/mongo-driver/x/mongo/driver.Error.Error`
347. `github.com/containerd/errdefs.errPermissionDenied.Error`
348. `crypto/x509.InsecureAlgorithmError.Error`
349. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.TransitionError.Error`
350. `\*net/http.http2ConnectionError.Error`
351. `\*github.com/LerianStudio/lib-license-go/v2/pkg.EntityConflictError.Error`
352. `\*github.com/containerd/errdefs.errInvalidArgument.Error`
353. `\*golang.org/x/crypto/ssh.PartialSuccessError.Error`
354. `\*github.com/go-openapi/swag/jsonutils/adapters.registryError.Error`
355. `\*github.com/jackc/pgx/v5/pgconn.pgconnError.Error`
356. `\*github.com/redis/go-redis/v9/internal/proto.ClusterDownError.Error`
357. `github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
358. `github.com/containerd/errdefs.errNotFound.Error`
359. `\*internal/chacha8rand.errUnmarshalChaCha8.Error`
360. `github.com/LerianStudio/lib-license-go/v2/pkg.EntityNotFoundError.Error`
361. `\*google.golang.org/grpc/internal/transport.NewStreamError.Error`
362. `\*net.timeoutError.Error`
363. `go.opentelemetry.io/otel/sdk/trace.errUnsupportedSampler.Error`
364. `\*internal/fuzz.MalformedCorpusError.Error`
365. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationUnknownFieldsError.Error`
366. `\*github.com/jackc/pgx/v5/pgconn.NotPreferredError.Error`
367. `net/netip.parsePrefixError.Error`
368. `\*google.golang.org/grpc.dropError.Error`
369. `github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
370. `\*github.com/LerianStudio/lib-license-go/v2/pkg.HTTPError.Error`
371. `crypto/des.KeySizeError.Error`
372. `net/http.transportReadFromServerError.Error`
373. `github.com/containerd/errdefs.errAborted.Error`
374. `\*github.com/gofiber/fiber/v2/internal/schema.EmptyFieldError.Error`
375. `github.com/containerd/errdefs.errDataLoss.Error`
376. `go.opentelemetry.io/otel/sdk/trace.samplerArgParseError.Error`
377. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.multiErr.Error`
378. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteErrors.Error`
379. `archive/tar.headerError.Error`
380. `github.com/go-logr/logr.notFoundError.Error`
381. `\*golang.org/x/net/http2.GoAwayError.Error`
382. `\*net/url.InvalidHostError.Error`
383. `\*os.LinkError.Error`
384. `\*go.mongodb.org/mongo-driver/x/mongo/driver.QueryFailureError.Error`
385. `go.mongodb.org/mongo-driver/mongo.WriteConcernError.Error`
386. `internal/strconv.Error.Error`
387. `\*net.addrinfoErrno.Error`
388. `\*net.OpError.Error`
389. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.WaitQueueTimeoutError.Error`
390. `text/template.ExecError.Error`
391. `\*go.uber.org/multierr.multiError.Error`
392. `\*os.errSymlink.Error`
393. `\*github.com/valyala/fasthttp.ErrNothingRead.Error`
394. `go.mongodb.org/mongo-driver/bson/bsoncodec.ValueEncoderError.Error`
395. `\*math/big.ErrNaN.Error`
396. `crypto/internal/fips140/hmac.errCloneUnsupported.Error`
397. `\*encoding/json.UnmarshalFieldError.Error`
398. `\*go.mongodb.org/mongo-driver/bson/bsonrw.errMaxDocumentSizeExceeded.Error`
399. `net/http.http2pseudoHeaderError.Error`
400. `\*golang.org/x/crypto/ocsp.ResponseError.Error`
401. `github.com/gofiber/fiber/v2/internal/schema.EmptyFieldError.Error`
402. `\*github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
403. `\*github.com/go-playground/universal-translator.ErrBadParamSyntax.Error`
404. `\*github.com/docker/docker/api/types/registry.errInvalidParameter.Error`
405. `\*golang.org/x/net/http2.duplicatePseudoHeaderError.Error`
406. `\*net.DNSConfigError.Error`
407. `\*os/user.UnknownUserError.Error`
408. `\*google.golang.org/grpc/internal/transport.ioError.Error`
409. `github.com/LerianStudio/lib-license-go/v2/pkg.InternalServerError.Error`
410. `\*golang.org/x/crypto/blowfish.KeySizeError.Error`
411. `\*runtime.errorAddressString.Error`
412. `go.uber.org/zap.errArrayElem.Error`
413. `runtime.boundsError.Error`
414. `net/http.http2GoAwayError.Error`
415. `github.com/pkg/errors.withStack.Error`
416. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InsufficientBytesError.Error`
417. `\*go.opentelemetry.io/otel/trace.errorConst.Error`
418. `\*golang.org/x/net/http2.pseudoHeaderError.Error`
419. `\*github.com/LerianStudio/lib-license-go/v2/pkg.EntityNotFoundError.Error`
420. `\*github.com/LerianStudio/lib-commons/v4/commons.Response.Error`
421. `crypto/internal/fips140/aes.KeySizeError.Error`
422. `\*go.mongodb.org/mongo-driver/mongo.CommandError.Error`
423. `github.com/containerd/errdefs.errAlreadyExists.Error`
424. `golang.org/x/net/http2.GoAwayError.Error`
425. `\*github.com/containerd/errdefs.errOutOfRange.Error`
426. `\*golang.org/x/net/webdav/internal/xml.SyntaxError.Error`
427. `\*go/build.NoGoError.Error`
428. `\*github.com/tryvium-travels/memongo/mongobin.UnsupportedSystemError.Error`
429. `\*os/signal.signalError.Error`
430. `\*os/user.UnknownGroupError.Error`
431. `\*os/exec.wrappedError.Error`
432. `go.mongodb.org/mongo-driver/bson/bsoncodec.TransitionError.Error`
433. `\*github.com/docker/docker/api/types/container.errInvalidParameter.Error`
434. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ValueEncoderError.Error`
435. `\*github.com/containerd/errdefs.errNotImplemented.Error`
436. `golang.org/x/net/idna.labelError.Error`
437. `\*go/types.Error.Error`
438. `github.com/microsoft/go-mssqldb.ServerError.Error`
439. `encoding/xml.UnmarshalError.Error`
440. `\*github.com/go-playground/validator/v10.fieldError.Error`
441. `\*github.com/microsoft/go-mssqldb.ServerError.Error`
442. `\*html/template.Error.Error`
443. `\*go.mongodb.org/mongo-driver/mongo/options.MarshalError.Error`
444. `\*compress/bzip2.StructuralError.Error`
445. `\*golang.org/x/crypto/ocsp.ParseError.Error`
446. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.MalformedElementError.Error`
447. `\*go.opentelemetry.io/otel/sdk/trace.errUnsupportedSampler.Error`
448. `\*go/scanner.Error.Error`
449. `github.com/LerianStudio/lib-license-go/v2/pkg.EntityConflictError.Error`
450. `\*golang.org/x/net/http2/hpack.InvalidIndexError.Error`
451. `\*github.com/jackc/pgx/v5/pgconn.errTimeout.Error`
452. `\*github.com/valyala/fasthttp/fasthttputil.timeoutError.Error`
453. `\*net/http.transportReadFromServerError.Error`
454. `\*github.com/cenkalti/backoff/v5.PermanentError.Error`
455. `github.com/gofiber/fiber/v2/internal/schema.MultiError.Error`
456. `\*golang.org/x/crypto/ssh.OpenChannelError.Error`
457. `go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt.Error.Error`
458. `\*time.ParseError.Error`
459. `github.com/valyala/fasthttp.ErrSmallBuffer.Error`
460. `\*github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
461. `net/mail.charsetError.Error`
462. `\*github.com/redis/go-redis/v9/internal/proto.NoReplicasError.Error`
463. `\*github.com/redis/go-redis/v9/internal/proto.PermissionError.Error`
464. `\*encoding/asn1.StructuralError.Error`
465. `\*go.mongodb.org/mongo-driver/internal/aws/awserr.errorList.Error`
466. `github.com/containerd/errdefs.errUnauthorized.Error`
467. `\*github.com/microsoft/go-mssqldb.StreamError.Error`
468. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InvalidDepthTraversalError.Error`
469. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteError.Error`
470. `\*internal/strconv.Error.Error`
471. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.ServerSelectionError.Error`
472. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ForbiddenError.Error`
473. `\*golang.org/x/crypto/ssh.BannerError.Error`
474. `\*net/http.http2connError.Error`
475. `go.mongodb.org/mongo-driver/mongo.MarshalError.Error`
476. `\*github.com/redis/go-redis/v9/internal/pool.BadConnError.Error`
477. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ValueDecoderError.Error`
478. `os/user.UnknownUserIdError.Error`
479. `\*net.UnknownNetworkError.Error`
480. `\*golang.org/x/crypto/ssh.disconnectMsg.Error`
481. `\*net/http.http2httpError.Error`
482. `\*github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
483. `github.com/docker/docker/api/types/registry.errInvalidParameter.Error`
484. `github.com/grpc-ecosystem/grpc-gateway/v2/runtime.MalformedSequenceError.Error`
485. `\*internal/bisect.parseError.Error`
486. `\*github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
487. `github.com/redis/go-redis/v9/internal/proto.RedisError.Error`
488. `go.mongodb.org/mongo-driver/mongo.WriteError.Error`
489. `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal.PartialSuccess.Error`
490. `\*github.com/go-resty/resty/v2.ResponseError.Error`
491. `\*github.com/valyala/fasthttp.InvalidHostError.Error`
492. `\*encoding/asn1.SyntaxError.Error`
493. `\*regexp/syntax.Error.Error`
494. `\*github.com/moby/term.EscapeError.Error`
495. `\*go/build.MultiplePackageError.Error`
496. `github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
497. `\*github.com/containerd/errdefs.errUnknown.Error`
498. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoDecoder.Error`
499. `\*go.uber.org/zap.errSinkNotFound.Error`
500. `\*go.mongodb.org/mongo-driver/mongo.MongocryptError.Error`
501. `github.com/valyala/fasthttp.InvalidHostError.Error`
502. `go.mongodb.org/mongo-driver/bson/bsonrw.errMaxDocumentSizeExceeded.Error`
503. `\*github.com/docker/docker/api/types/filters.invalidFilter.Error`
504. `\*github.com/jackc/pgx/v5/pgconn.ParseConfigError.Error`
505. `golang.org/x/crypto/ssh.cbcError.Error`
506. `google.golang.org/grpc.dropError.Error`
507. `\*github.com/valyala/fasthttp.ErrDialWithUpstream.Error`
508. `\*github.com/andybalholm/brotli.decodeError.Error`
509. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationKnownFieldsError.Error`
510. `\*github.com/LerianStudio/lib-commons/v2/commons.Response.Error`
511. `\*github.com/go-playground/universal-translator.ErrConflictingTranslation.Error`
512. `\*github.com/grpc-ecosystem/grpc-gateway/v2/runtime.MalformedSequenceError.Error`
513. `\*net/http.ProtocolError.Error`
514. `\*encoding/xml.UnsupportedTypeError.Error`
515. `\*github.com/containerd/errdefs.errUnauthorized.Error`
516. `\*github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
517. `os/signal.signalError.Error`
518. `github.com/testcontainers/testcontainers-go.ParallelContainersError.Error`
519. `go.mongodb.org/mongo-driver/internal/aws/awserr.errorList.Error`
520. `\*github.com/LerianStudio/lib-commons/v4/commons/runtime.panicError.Error`
521. `encoding/asn1.SyntaxError.Error`
522. `github.com/containerd/errdefs.errFailedPrecondition.Error`
523. `github.com/go-openapi/swag/jsonutils/adapters.registryError.Error`
524. `github.com/containerd/errdefs.errUnknown.Error`
525. `\*github.com/containerd/errdefs.errInternal.Error`
526. `\*google.golang.org/protobuf/internal/impl.errInvalidUTF8.Error`
527. `\*encoding/json.UnsupportedValueError.Error`
528. `\*github.com/microsoft/go-mssqldb.Error.Error`
529. `\*github.com/gofiber/fiber/v2/internal/schema.ConversionError.Error`
530. `github.com/grpc-ecosystem/grpc-gateway/v2/internal/httprule.InvalidTemplateError.Error`
531. `\*encoding/json.jsonError.Error`
532. `\*github.com/yuin/gopher-lua.ApiError.Error`
533. `\*github.com/go-resty/resty/v2.restyError.Error`
534. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal.PartialSuccess.Error`
535. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteCommandError.Error`
536. `\*gopkg.in/yaml.v3.TypeError.Error`
537. `\*encoding/hex.InvalidByteError.Error`
538. `\*github.com/go-playground/universal-translator.ErrMissingLocale.Error`
539. `\*go.mongodb.org/mongo-driver/internal/aws/awserr.baseError.Error`
540. `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal.PartialSuccess.Error`
541. `\*net.canceledError.Error`
542. `golang.org/x/net/webdav/internal/xml.UnmarshalError.Error`
543. `net/http.http2ConnectionError.Error`
544. `\*github.com/LerianStudio/lib-license-go/v2/pkg.FailedPreconditionError.Error`
545. `github.com/microsoft/go-mssqldb.StreamError.Error`
546. `github.com/LerianStudio/lib-license-go/v2/pkg.UnprocessableOperationError.Error`
547. `\*go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc/internal.PartialSuccess.Error`
548. `\*go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal.PartialSuccess.Error`
549. `\*go.mongodb.org/mongo-driver/mongo.BulkWriteException.Error`
550. `golang.org/x/crypto/ocsp.ParseError.Error`
551. `go.opentelemetry.io/otel/trace.errorConst.Error`
552. `\*github.com/yuin/gopher-lua.CompileError.Error`
553. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteConcernError.Error`
554. `math/big.ErrNaN.Error`
555. `\*os/user.UnknownGroupIdError.Error`
556. `\*encoding/csv.ParseError.Error`
557. `\*github.com/jackc/pgx/v5/pgconn.PrepareError.Error`
558. `github.com/golang-jwt/jwt/v5.joinedError.Error`
559. `gopkg.in/go-playground/validator.v9.ValidationErrors.Error`
560. `crypto/x509/internal/macos.OSStatus.Error`
561. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.decodeBinaryError.Error`
562. `\*github.com/docker/docker/client.objectNotFoundError.Error`
563. `go.mongodb.org/mongo-driver/bson/bsoncodec.decodeBinaryError.Error`
564. `github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
565. `net/http.http2headerFieldValueError.Error`
566. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteError.Error`
567. `\*go.yaml.in/yaml/v3.TypeError.Error`
568. `\*go.mongodb.org/mongo-driver/mongo.MarshalError.Error`
569. `\*golang.org/x/mod/module.InvalidVersionError.Error`
570. `\*encoding/json.InvalidUnmarshalError.Error`
571. `\*github.com/tryvium-travels/memongo/mongobin.UnsupportedMongoVersionError.Error`
572. `github.com/LerianStudio/lib-license-go/v2/pkg.FailedPreconditionError.Error`
573. `runtime.errorString.Error`
574. `\*net.DNSError.Error`
575. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ResponseError.Error`
576. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoDecoder.Error`
577. `\*github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
578. `github.com/klauspost/compress/flate.InternalError.Error`
579. `\*github.com/klauspost/compress/flate.InternalError.Error`
580. `\*github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
581. `\*runtime.synctestDeadlockError.Error`
582. `\*github.com/redis/go-redis/v9/internal/proto.MaxClientsError.Error`
583. `\*github.com/valyala/fasthttp.timeoutError.Error`
584. `\*io/fs.PathError.Error`
585. `\*github.com/docker/docker/client.emptyIDError.Error`
586. `\*internal/runtime/maps.unhashableTypeError.Error`
587. `github.com/docker/docker/api/types/filters.invalidFilter.Error`
588. `go/scanner.ErrorList.Error`
589. `\*net/http.http2goAwayFlowError.Error`
590. `net.UnknownNetworkError.Error`
591. `\*net/url.EscapeError.Error`
592. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoTypeMapEntry.Error`
593. `\*github.com/go-playground/universal-translator.ErrMissingPluralTranslation.Error`
594. `\*github.com/containerd/errdefs.customMessage.Error`
595. `net/http.statusError.Error`
596. `\*github.com/rabbitmq/amqp091-go.Error.Error`
597. `\*github.com/docker/docker/api/types.ErrorResponse.Error`
598. `github.com/containerd/errdefs.errInternal.Error`
599. `\*vendor/golang.org/x/net/http2/hpack.InvalidIndexError.Error`
600. `\*github.com/lib/pq.safeRetryError.Error`
601. `runtime.plainError.Error`
602. `\*github.com/alicebob/miniredis/v2/gopher-json.invalidTypeError.Error`
603. `\*go.mongodb.org/mongo-driver/mongo.EncryptionKeyVaultError.Error`
604. `\*net.AddrError.Error`
605. `github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
606. `\*net/url.Error.Error`
607. `github.com/containerd/errdefs.errUnavailable.Error`
608. `\*net/mail.charsetError.Error`
609. `\*crypto/x509/internal/macos.OSStatus.Error`
610. `\*github.com/sijms/go-ora/v2/network.OracleError.Error`
611. `\*internal/fuzz.crashError.Error`
612. `\*net/textproto.ProtocolError.Error`
613. `\*github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
614. `github.com/andybalholm/brotli.decodeError.Error`
615. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ValidationError.Error`
616. `\*github.com/containerd/errdefs.errNotFound.Error`
617. `\*github.com/redis/go-redis/v9/internal/proto.AskError.Error`
618. `go.mongodb.org/mongo-driver/mongo.CommandError.Error`
619. `os/exec.wrappedError.Error`
620. `\*net.notFoundError.Error`
621. `crypto/x509.UnknownAuthorityError.Error`
622. `crypto/x509.HostnameError.Error`
623. `github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
624. `\*github.com/go-playground/universal-translator.ErrRangeTranslation.Error`
625. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationKnownFieldsError.Error`
626. `\*runtime.plainError.Error`
627. `github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
628. `golang.org/x/net/http2.connError.Error`
629. `\*fmt.wrapErrors.Error`
630. `\*github.com/LerianStudio/lib-commons/v4/commons/assert.AssertionError.Error`
631. `go.mongodb.org/mongo-driver/mongo.MongocryptdError.Error`
632. `\*google.golang.org/grpc/internal/transport.ConnectionError.Error`
633. `\*net/http.requestBodyReadError.Error`
634. `\*compress/flate.ReadError.Error`
635. `\*syscall.Errno.Error`
636. `\*github.com/valyala/fasthttp/reuseport.ErrNoReusePort.Error`
637. `\*golang.org/x/net/http2.connError.Error`
638. `runtime.synctestDeadlockError.Error`
639. `\*compress/flate.CorruptInputError.Error`
640. `\*github.com/redis/go-redis/v9/internal/proto.TryAgainError.Error`
641. `github.com/redis/go-redis/v9/internal/pool.BadConnError.Error`
642. `github.com/jackc/pgx/v5.ScanArgError.Error`
643. `\*net/http.MaxBytesError.Error`
644. `\*golang.org/x/crypto/ssh.AlgorithmNegotiationError.Error`
645. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoTypeMapEntry.Error`
646. `github.com/google/uuid.invalidLengthError.Error`
647. `\*github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
648. `\*golang.org/x/net/http2.StreamError.Error`
649. `\*github.com/testcontainers/testcontainers-go/wait.PermanentError.Error`
650. `\*github.com/cenkalti/backoff/v5.RetryAfterError.Error`
651. `github.com/alicebob/miniredis/v2/gopher-json.invalidTypeError.Error`
652. `github.com/valyala/fasthttp.ErrBrokenChunk.Error`
653. `\*golang.org/x/crypto/ssh.PassphraseMissingError.Error`
654. `\*github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
655. `google.golang.org/grpc/internal/transport.NewStreamError.Error`
656. `\*github.com/containerd/errdefs.errAlreadyExists.Error`
657. `net.addrinfoErrno.Error`
658. `go.mongodb.org/mongo-driver/mongo.MongocryptError.Error`
659. `encoding/base64.CorruptInputError.Error`
660. `crypto/tls.AlertError.Error`
661. `\*encoding/json.SyntaxError.Error`
662. `\*net/textproto.Error.Error`
663. `\*github.com/jackc/pgx/v5/pgproto3.writeError.Error`
664. `\*github.com/go-openapi/swag/yamlutils.yamlError.Error`
665. `\*runtime.PanicNilError.Error`
666. `github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
667. `github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
668. `\*github.com/go-playground/universal-translator.ErrMissingBracket.Error`
669. `encoding/asn1.StructuralError.Error`
670. `compress/flate.InternalError.Error`
671. `\*crypto/tls.echConfigErr.Error`
672. `\*github.com/google/uuid.invalidLengthError.Error`
673. `\*github.com/LerianStudio/lib-license-go/v2/pkg.InternalServerError.Error`
674. `go.mongodb.org/mongo-driver/mongo.BulkWriteException.Error`
675. `github.com/go-playground/validator/v10.ValidationErrors.Error`
676. `\*go.mongodb.org/mongo-driver/mongo.ErrMapForOrderedArgument.Error`
677. `\*github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
678. `github.com/containerd/errdefs.errResourceExhausted.Error`
679. `go.mongodb.org/mongo-driver/x/mongo/driver.ResponseError.Error`
680. `\*github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
681. `\*encoding/json.UnsupportedTypeError.Error`
682. `\*encoding/xml.SyntaxError.Error`
683. `\*github.com/jackc/pgx/v5/pgproto3.invalidMessageLenErr.Error`
684. `\*crypto/internal/fips140/aes.KeySizeError.Error`
685. `debug/dwarf.DecodeError.Error`
686. `\*go.mongodb.org/mongo-driver/mongo.WriteErrors.Error`
687. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.errMetric.Error`
688. `os/user.UnknownGroupError.Error`
689. `\*os/exec.ExitError.Error`
690. `\*net/http.statusError.Error`
691. `\*golang.org/x/text/internal/language.ValueError.Error`
692. `golang.org/x/text/internal/language.ValueError.Error`
693. `crypto/x509.CertificateInvalidError.Error`
694. `crypto/tls.RecordHeaderError.Error`
695. `\*internal/poll.errNetClosing.Error`
696. `\*debug/macho.FormatError.Error`
697. `github.com/LerianStudio/lib-license-go/v2/pkg.ForbiddenError.Error`
698. `\*github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
699. `\*github.com/go-openapi/swag/loading.loadingError.Error`
700. `\*github.com/docker/docker/pkg/jsonmessage.JSONError.Error`
701. `\*github.com/pkg/errors.withStack.Error`
702. `\*golang.org/x/net/http2.headerFieldNameError.Error`
703. `Contains`

#### `stubJobSpanAttributes`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** MEDIUM (4 direct callers)

**Direct Callers (signature change affects these):**

1. `TestJobMongoDBRepository\_List$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:407`
2. `TestJobMongoDBRepository\_Update$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:279`
3. `TestProductMongoDBRepository\_Create\_ErrorPaths$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb\_test.go:181`
4. `TestJobMongoDBRepository\_Create$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:179`

**Callees (this function depends on):**

1. `\*testing.common.Helper`
2. `\*testing.common.Cleanup`

#### `TestJobMongoDBRepository\_FindByRequestHashWithinWindow`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`
5. `\*testing.T.Run`
6. `\*testing.T.Run`

#### `TestRepositoryConstructorValidatesDB`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewJobMongoDBRepository`
2. `\*testing.common.Fatalf`
3. `\*testing.common.Fatalf`

#### `newJobRepository`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (45 direct callers)

**Direct Callers (signature change affects these):**

1. `TestJobMongoDBRepository\_Update$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:267`
2. `TestListUsesDescendingByDefault` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:1006`
3. `TestJobMongoDBRepository\_Update$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:251`
4. `TestJobMongoDBRepository\_UpdateStatus$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:514`
5. `TestJobMongoDBRepository\_ExistsRunningByMappedFieldKey$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:720`
6. `TestJobMongoDBRepository\_FindByRequestHashWithinWindow$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:625`
7. `TestJobMongoDBRepository\_List$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:352`
8. `TestJobMongoDBRepository\_FindByRequestHashWithinWindow$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:613`
9. `TestJobMongoDBRepository\_UpdateStatus$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:443`
10. `TestJobMongoDBRepository\_Create$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:156`
11. `TestListWithPaginationSecondPageEmpty` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:1088`
12. `TestJobMongoDBRepository\_Update$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:258`
13. `TestJobMongoDBRepository\_FindByID$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:321`
14. `TestJobMongoDBRepository\_ExistsRunningByMappedFieldKey$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:791`
15. `TestJobMongoDBRepository\_Update$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:236`
16. `TestJobMongoDBRepository\_Create$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:143`
17. `TestJobMongoDBRepository\_Update$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:220`
18. `TestJobMongoDBRepository\_UpdateStatus$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:546`
19. `TestListPartialFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:1031`
20. `TestJobMongoDBRepository\_FindByRequestHashWithinWindow$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:647`
21. `TestJobMongoDBRepository\_EnsureIndexes` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:910`
22. `TestJobMongoDBRepository\_Create$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:186`
23. `TestJobMongoDBRepository\_List$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:405`
24. `TestJobMongoDBRepository\_ExistsRunningByMappedFieldKey$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:814`
25. `TestEnsureIndexesHandlesConflicts` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:927`
26. `TestJobMongoDBRepository\_UpdateStatus$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:536`
27. `TestJobMongoDBRepository\_FindByID$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:309`
28. `TestJobMongoDBRepository\_Update$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:277`
29. `TestJobMongoDBRepository\_ExistsRunningByMappedFieldKey$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:838`
30. `TestJobMongoDBRepository\_UpdateStatus$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:489`
31. `TestJobMongoDBRepository\_FindByRequestHashWithinWindow$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:589`
32. `TestListCompletedRangeFilter` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:983`
33. `TestJobMongoDBRepository\_Create$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:178`
34. `TestJobMongoDBRepository\_UpdateStatus$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:468`
35. `TestCreateSetsDefaults` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:1050`
36. `TestUpdateWithoutCompletedAtWhenFailed` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:1070`
37. `TestJobMongoDBRepository\_DropIndexes` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:917`
38. `TestJobMongoDBRepository\_FindByRequestHashWithinWindow$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:679`
39. `TestJobMongoDBRepository\_List$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:432`
40. `TestJobMongoDBRepository\_ExistsRunningByMappedFieldKey$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:847`
41. `TestJobMongoDBRepository\_Create$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:171`
42. `TestJobMongoDBRepository\_ExistsRunningByMappedFieldKey$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:743`
43. `TestJobMongoDBRepository\_UpdateStatus$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:574`
44. `TestJobMongoDBRepository\_List$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:391`
45. `TestJobMongoDBRepository\_ExistsRunningByMappedFieldKey$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:766`

**Callees (this function depends on):**

1. `\*testing.common.Helper`
2. `clearJobsCollection`
3. `NewJobMongoDBRepository`
4. `\*testing.common.Fatalf`

#### `TestEnsureIndexesDatabaseError`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewController`
2. `\*go.uber.org/mock/gomock.Controller.Finish`
3. `NewMockMongoClientProvider`
4. `\*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.EXPECT`
5. `Any`
6. `\*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProviderMockRecorder.Client`
7. `New`
8. `\*go.uber.org/mock/gomock.Call.Return`
9. `Background`
10. `\*github.com/LerianStudio/fetcher/pkg/mongodb/job.JobMongoDBRepository.EnsureIndexes`
11. `\*testing.common.Fatalf`
12. `\*github.com/pkg/errors.withMessage.Error`
13. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationError.Error`
14. `\*github.com/docker/docker/client.httpError.Error`
15. `\*runtime.boundsError.Error`
16. `\*gopkg.in/go-playground/validator.v9.InvalidValidationError.Error`
17. `go.mongodb.org/mongo-driver/mongo/options.MarshalError.Error`
18. `\*go.mongodb.org/mongo-driver/internal/codecutil.MarshalError.Error`
19. `github.com/docker/docker/api/types.ErrorResponse.Error`
20. `\*go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt.Error.Error`
21. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationError.Error`
22. `github.com/docker/docker/client.emptyIDError.Error`
23. `\*github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
24. `vendor/golang.org/x/net/http2/hpack.DecodingError.Error`
25. `\*net/http.unsupportedTEError.Error`
26. `\*net/http.http2headerFieldNameError.Error`
27. `\*github.com/docker/docker/api/types/network.joinError.Error`
28. `\*go.mongodb.org/mongo-driver/x/mongo/driver/auth.Error.Error`
29. `\*net.ParseError.Error`
30. `github.com/go-openapi/swag/yamlutils.yamlError.Error`
31. `runtime.errorAddressString.Error`
32. `\*net.InvalidAddrError.Error`
33. `\*crypto/x509.UnknownAuthorityError.Error`
34. `\*github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
35. `golang.org/x/crypto/ssh.ServerAuthError.Error`
36. `\*github.com/go-playground/validator/v10.ValidationErrors.Error`
37. `golang.org/x/net/http2.pseudoHeaderError.Error`
38. `golang.org/x/crypto/ocsp.ResponseError.Error`
39. `go.mongodb.org/mongo-driver/x/mongo/driver.QueryFailureError.Error`
40. `github.com/docker/docker/api/types/container.errInvalidParameter.Error`
41. `\*net/http.http2pseudoHeaderError.Error`
42. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.MalformedElementError.Error`
43. `\*crypto/tls.alert.Error`
44. `\*crypto/x509.InsecureAlgorithmError.Error`
45. `go.mongodb.org/mongo-driver/mongo.WriteErrors.Error`
46. `\*encoding/asn1.invalidUnmarshalError.Error`
47. `crypto/x509.UnhandledCriticalExtension.Error`
48. `go.mongodb.org/mongo-driver/bson/bsoncodec.ValueDecoderError.Error`
49. `github.com/containerd/errdefs.customMessage.Error`
50. `\*github.com/xdg-go/stringprep.Error.Error`
51. `\*crypto/tls.CertificateVerificationError.Error`
52. `\*fmt.wrapError.Error`
53. `\*github.com/yuin/gopher-lua/pm.Error.Error`
54. `github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
55. `\*net/http.http2headerFieldValueError.Error`
56. `golang.org/x/net/http2.duplicatePseudoHeaderError.Error`
57. `os.errSymlink.Error`
58. `vendor/golang.org/x/net/idna.labelError.Error`
59. `\*crypto/tls.RecordHeaderError.Error`
60. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoEncoder.Error`
61. `\*golang.org/x/mod/module.ModuleError.Error`
62. `\*encoding/json.UnmarshalTypeError.Error`
63. `github.com/containerd/errdefs.errInvalidArgument.Error`
64. `\*google.golang.org/protobuf/internal/errors.prefixError.Error`
65. `\*encoding/base64.CorruptInputError.Error`
66. `\*golang.org/x/mod/module.InvalidPathError.Error`
67. `\*go.mongodb.org/mongo-driver/x/mongo/driver.ResponseError.Error`
68. `net/netip.parseAddrError.Error`
69. `go/scanner.Error.Error`
70. `\*internal/poll.DeadlineExceededError.Error`
71. `\*golang.org/x/crypto/ssh.ExitMissingError.Error`
72. `time.fileSizeError.Error`
73. `\*github.com/redis/go-redis/v9/internal/proto.RedisError.Error`
74. `golang.org/x/net/http2.ConnectionError.Error`
75. `github.com/LerianStudio/lib-commons/v2/commons.Response.Error`
76. `go.mongodb.org/mongo-driver/internal/codecutil.MarshalError.Error`
77. `golang.org/x/net/http2/hpack.InvalidIndexError.Error`
78. `\*strconv.NumError.Error`
79. `\*go.opentelemetry.io/otel/sdk/trace.samplerArgParseError.Error`
80. `github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
81. `\*os/user.UnknownUserIdError.Error`
82. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteConcernError.Error`
83. `net/http.nothingWrittenError.Error`
84. `\*github.com/redis/go-redis/v9/internal/proto.MasterDownError.Error`
85. `\*go/scanner.ErrorList.Error`
86. `\*go.mongodb.org/mongo-driver/mongo.WriteException.Error`
87. `github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
88. `\*github.com/docker/docker/client.errConnectionFailed.Error`
89. `\*vendor/golang.org/x/net/dns/dnsmessage.nestedError.Error`
90. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InvalidDepthTraversalError.Error`
91. `net/http.requestBodyReadError.Error`
92. `net/http.tlsHandshakeTimeoutError.Error`
93. `\*github.com/pkg/errors.fundamental.Error`
94. `\*github.com/redis/go-redis/v9/internal/proto.MovedError.Error`
95. `\*golang.org/x/net/http2.goAwayFlowError.Error`
96. `\*github.com/containerd/errdefs.errPermissionDenied.Error`
97. `github.com/microsoft/go-mssqldb.RetryableError.Error`
98. `\*github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
99. `\*github.com/redis/go-redis/v9/internal/proto.ExecAbortError.Error`
100. `golang.org/x/text/encoding/internal.RepertoireError.Error`
101. `github.com/montanaflynn/stats.statsError.Error`
102. `\*gopkg.in/go-playground/validator.v9.fieldError.Error`
103. `\*github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
104. `\*net/http.http2GoAwayError.Error`
105. `github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
106. `github.com/docker/docker/client.objectNotFoundError.Error`
107. `\*net/http.nothingWrittenError.Error`
108. `\*github.com/montanaflynn/stats.statsError.Error`
109. `\*github.com/redis/go-redis/v9/push.ProcessorError.Error`
110. `\*go.mongodb.org/mongo-driver/x/mongo/driver/ocsp.Error.Error`
111. `golang.org/x/net/http2.noCachedConnError.Error`
112. `crypto/x509.ConstraintViolationError.Error`
113. `github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
114. `github.com/LerianStudio/lib-commons/v4/commons.Response.Error`
115. `net/http.http2StreamError.Error`
116. `github.com/LerianStudio/lib-license-go/v2/pkg.ResponseError.Error`
117. `\*os.SyscallError.Error`
118. `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc/internal.PartialSuccess.Error`
119. `crypto/tls.alert.Error`
120. `\*vendor/golang.org/x/net/idna.labelError.Error`
121. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteErrors.Error`
122. `\*net/netip.parseAddrError.Error`
123. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ElementTypeError.Error`
124. `vendor/golang.org/x/net/http2/hpack.InvalidIndexError.Error`
125. `\*github.com/containerd/errdefs.errNotModified.Error`
126. `\*net.temporaryError.Error`
127. `\*github.com/jackc/pgx/v5/pgconn.perDialConnectError.Error`
128. `\*go.mongodb.org/mongo-driver/mongo.BulkWriteError.Error`
129. `\*github.com/jackc/pgx/v5/pgconn.PgError.Error`
130. `github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
131. `go.mongodb.org/mongo-driver/bson/bsonrw.TransitionError.Error`
132. `\*go/types.ArgumentError.Error`
133. `github.com/containerd/errdefs.errNotImplemented.Error`
134. `\*gopkg.in/go-playground/validator.v9.ValidationErrors.Error`
135. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.DecodeError.Error`
136. `\*github.com/containerd/errdefs/pkg/internal/cause.ErrUnexpectedStatus.Error`
137. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ElementTypeError.Error`
138. `google.golang.org/grpc/internal/transport.ioError.Error`
139. `net/url.InvalidHostError.Error`
140. `golang.org/x/net/http2.StreamError.Error`
141. `\*github.com/jackc/pgx/v5/pgtype.nullAssignmentError.Error`
142. `\*github.com/redis/go-redis/v9/internal/proto.AuthError.Error`
143. `\*github.com/microsoft/go-mssqldb.RetryableError.Error`
144. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.ServerSelectionError.Error`
145. `internal/runtime/maps.unhashableTypeError.Error`
146. `\*google.golang.org/protobuf/internal/errors.SizeMismatchError.Error`
147. `go.mongodb.org/mongo-driver/x/mongo/driver.Error.Error`
148. `\*crypto/rc4.KeySizeError.Error`
149. `github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
150. `github.com/gofiber/fiber/v2/internal/schema.ConversionError.Error`
151. `\*github.com/valyala/fasthttp.ErrBrokenChunk.Error`
152. `google.golang.org/grpc/internal/transport.ConnectionError.Error`
153. `\*github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
154. `\*github.com/redis/go-redis/v9/internal/proto.OOMError.Error`
155. `\*golang.org/x/net/http2/hpack.DecodingError.Error`
156. `\*crypto/tls.ECHRejectionError.Error`
157. `crypto/aes.KeySizeError.Error`
158. `\*context.deadlineExceededError.Error`
159. `\*vendor/golang.org/x/net/http2/hpack.DecodingError.Error`
160. `github.com/xdg-go/stringprep.Error.Error`
161. `net/http.http2noCachedConnError.Error`
162. `\*vendor/golang.org/x/net/idna.runeError.Error`
163. `golang.org/x/net/http2/hpack.DecodingError.Error`
164. `\*github.com/jackc/pgx/v5/pgconn.contextAlreadyDoneError.Error`
165. `\*github.com/grpc-ecosystem/grpc-gateway/v2/runtime.HTTPStatusError.Error`
166. `\*debug/dwarf.DecodeError.Error`
167. `\*crypto/internal/fips140/hmac.errCloneUnsupported.Error`
168. `github.com/LerianStudio/lib-license-go/v2/pkg.HTTPError.Error`
169. `go.mongodb.org/mongo-driver/mongo.WriteException.Error`
170. `github.com/go-openapi/swag/jsonutils/adapters/stdlib/json.jsonError.Error`
171. `\*text/template.ExecError.Error`
172. `\*github.com/go-sql-driver/mysql.MySQLError.Error`
173. `github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
174. `\*encoding/json.InvalidUTF8Error.Error`
175. `\*golang.org/x/net/webdav/internal/xml.UnmarshalError.Error`
176. `\*github.com/go-playground/universal-translator.ErrExistingTranslator.Error`
177. `\*runtime.errorString.Error`
178. `\*time.parseDurationError.Error`
179. `encoding/hex.InvalidByteError.Error`
180. `\*crypto/tls.AlertError.Error`
181. `net/textproto.ProtocolError.Error`
182. `\*github.com/containerd/errdefs.errResourceExhausted.Error`
183. `\*crypto/des.KeySizeError.Error`
184. `github.com/moby/term.EscapeError.Error`
185. `\*github.com/LerianStudio/lib-commons/v4/commons/rabbitmq.sanitizedError.Error`
186. `compress/bzip2.StructuralError.Error`
187. `\*time.fileSizeError.Error`
188. `\*golang.org/x/net/http2.ConnectionError.Error`
189. `\*github.com/testcontainers/testcontainers-go.ParallelContainersError.Error`
190. `\*golang.org/x/net/http2.httpError.Error`
191. `\*github.com/gofiber/fiber/v2.Error.Error`
192. `go.mongodb.org/mongo-driver/mongo.EncryptionKeyVaultError.Error`
193. `github.com/go-openapi/jsonpointer.pointerError.Error`
194. `\*compress/flate.InternalError.Error`
195. `github.com/valyala/fasthttp.ErrNothingRead.Error`
196. `\*github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
197. `\*encoding/xml.UnmarshalError.Error`
198. `\*github.com/go-playground/validator/v10.InvalidValidationError.Error`
199. `github.com/containerd/errdefs.errConflict.Error`
200. `github.com/LerianStudio/lib-commons/v4/commons/net/http.ErrorResponse.Error`
201. `google.golang.org/protobuf/internal/impl.errInvalidUTF8.Error`
202. `github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
203. `\*internal/reflectlite.ValueError.Error`
204. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteCommandError.Error`
205. `github.com/go-openapi/swag/loading.loadingError.Error`
206. `\*github.com/gofiber/fiber/v2/internal/schema.MultiError.Error`
207. `\*google.golang.org/protobuf/internal/errors.wrapError.Error`
208. `\*runtime.TypeAssertionError.Error`
209. `\*crypto/x509.CertificateInvalidError.Error`
210. `\*github.com/LerianStudio/lib-license-go/v2/pkg.UnprocessableOperationError.Error`
211. `go.mongodb.org/mongo-driver/internal/aws/awserr.baseError.Error`
212. `golang.org/x/crypto/blowfish.KeySizeError.Error`
213. `github.com/valyala/fasthttp.EscapeError.Error`
214. `\*golang.org/x/crypto/ssh.cbcError.Error`
215. `\*github.com/go-playground/universal-translator.ErrCardinalTranslation.Error`
216. `\*github.com/valyala/fasthttp.EscapeError.Error`
217. `\*errors.joinError.Error`
218. `\*github.com/redis/go-redis/v9/internal/proto.ReadOnlyError.Error`
219. `\*github.com/valyala/fasthttp.ErrBodyStreamWritePanic.Error`
220. `\*crypto/tls.permanentError.Error`
221. `encoding/json.jsonError.Error`
222. `\*go.mongodb.org/mongo-driver/x/mongo/driver.InvalidOperationError.Error`
223. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.PoolError.Error`
224. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.PoolError.Error`
225. `golang.org/x/net/http2.headerFieldNameError.Error`
226. `\*github.com/go-logr/logr.notFoundError.Error`
227. `\*crypto/x509.ConstraintViolationError.Error`
228. `github.com/containerd/errdefs.errNotModified.Error`
229. `\*github.com/go-resty/resty/v2.noRetryErr.Error`
230. `\*github.com/redis/go-redis/v9/push.HandlerError.Error`
231. `\*golang.org/x/sync/singleflight.panicError.Error`
232. `\*github.com/containerd/errdefs.errUnavailable.Error`
233. `\*google.golang.org/grpc/internal/status.Error.Error`
234. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.WaitQueueTimeoutError.Error`
235. `\*github.com/containerd/errdefs.errAborted.Error`
236. `\*encoding/xml.TagPathError.Error`
237. `net/http.http2goAwayFlowError.Error`
238. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.poolClearedError.Error`
239. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoEncoder.Error`
240. `context.deadlineExceededError.Error`
241. `\*github.com/go-openapi/swag/jsonutils/adapters/stdlib/json.jsonError.Error`
242. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationUnknownFieldsError.Error`
243. `\*github.com/grpc-ecosystem/grpc-gateway/v2/internal/httprule.InvalidTemplateError.Error`
244. `crypto/x509.SystemRootsError.Error`
245. `\*github.com/containerd/errdefs.errDataLoss.Error`
246. `\*net/http.timeoutError.Error`
247. `\*github.com/jackc/pgx/v5.ScanArgError.Error`
248. `\*github.com/cenkalti/backoff/v4.PermanentError.Error`
249. `\*go.mongodb.org/mongo-driver/bson/bsonrw.TransitionError.Error`
250. `github.com/containerd/errdefs.errOutOfRange.Error`
251. `\*github.com/jackc/pgx/v5/pgproto3.ExceededMaxBodyLenErr.Error`
252. `\*go.uber.org/zap.errArrayElem.Error`
253. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.ConnectionError.Error`
254. `\*golang.org/x/crypto/ssh.ServerAuthError.Error`
255. `go.mongodb.org/mongo-driver/mongo.ErrMapForOrderedArgument.Error`
256. `golang.org/x/net/http2.headerFieldValueError.Error`
257. `go.mongodb.org/mongo-driver/x/mongo/driver/topology.ConnectionError.Error`
258. `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.errMetric.Error`
259. `\*encoding/json.MarshalerError.Error`
260. `\*crypto/x509.UnhandledCriticalExtension.Error`
261. `\*github.com/jackc/pgx/v5/pgproto3.invalidMessageFormatErr.Error`
262. `net/url.EscapeError.Error`
263. `\*github.com/jackc/pgx/v5/pgconn.ConnectError.Error`
264. `\*github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
265. `github.com/containerd/errdefs/pkg/internal/cause.ErrUnexpectedStatus.Error`
266. `net.InvalidAddrError.Error`
267. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InsufficientBytesError.Error`
268. `\*archive/tar.headerError.Error`
269. `github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
270. `golang.org/x/net/http2.goAwayFlowError.Error`
271. `\*github.com/redis/go-redis/v9/internal/proto.LoadingError.Error`
272. `\*github.com/valyala/fasthttp.ErrSmallBuffer.Error`
273. `\*github.com/LerianStudio/lib-license-go/v2/pkg.UnauthorizedError.Error`
274. `go.mongodb.org/mongo-driver/x/mongo/driver.InvalidOperationError.Error`
275. `vendor/golang.org/x/net/idna.runeError.Error`
276. `\*github.com/Shopify/toxiproxy/v2/client.ApiError.Error`
277. `github.com/gofiber/fiber/v2/internal/schema.UnknownKeyError.Error`
278. `os/user.UnknownUserError.Error`
279. `\*os/exec.Error.Error`
280. `\*github.com/microsoft/go-mssqldb/aecmk.Error.Error`
281. `\*github.com/yuin/gopher-lua/parse.Error.Error`
282. `syscall.Errno.Error`
283. `net.canceledError.Error`
284. `\*net/http.http2noCachedConnError.Error`
285. `\*net/http.tlsHandshakeTimeoutError.Error`
286. `\*crypto/x509.HostnameError.Error`
287. `\*github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
288. `net/http.http2connError.Error`
289. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ValidationError.Error`
290. `\*golang.org/x/net/webdav/internal/xml.UnsupportedTypeError.Error`
291. `os/user.UnknownGroupIdError.Error`
292. `\*github.com/lib/pq.Error.Error`
293. `\*github.com/jackc/pgx/v5/pgconn.connLockError.Error`
294. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.poolClearedError.Error`
295. `\*go.mongodb.org/mongo-driver/mongo.WriteConcernError.Error`
296. `\*go/build/constraint.SyntaxError.Error`
297. `\*go.mongodb.org/mongo-driver/mongo.MongocryptdError.Error`
298. `\*github.com/jackc/pgx/v5.proxyError.Error`
299. `github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
300. `github.com/valyala/fasthttp.ErrBodyStreamWritePanic.Error`
301. `\*github.com/go-playground/universal-translator.ErrOrdinalTranslation.Error`
302. `\*github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
303. `net/http.http2duplicatePseudoHeaderError.Error`
304. `github.com/ebitengine/purego.Dlerror.Error`
305. `\*github.com/golang-jwt/jwt/v5.joinedError.Error`
306. `\*crypto/aes.KeySizeError.Error`
307. `\*net/http.http2StreamError.Error`
308. `golang.org/x/net/idna.runeError.Error`
309. `\*github.com/LerianStudio/lib-commons/v4/commons/net/http.ErrorResponse.Error`
310. `\*crypto/x509.SystemRootsError.Error`
311. `\*github.com/shirou/gopsutil/v4/internal/common.Warnings.Error`
312. `github.com/LerianStudio/lib-license-go/v2/pkg.UnauthorizedError.Error`
313. `\*errors.errorString.Error`
314. `github.com/LerianStudio/lib-commons/commons.Response.Error`
315. `\*golang.org/x/net/idna.runeError.Error`
316. `\*golang.org/x/net/idna.labelError.Error`
317. `\*golang.org/x/crypto/ssh.ExitError.Error`
318. `\*golang.org/x/net/http2.noCachedConnError.Error`
319. `\*reflect.ValueError.Error`
320. `net/http.http2headerFieldNameError.Error`
321. `internal/poll.errNetClosing.Error`
322. `github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
323. `\*github.com/ebitengine/purego.Dlerror.Error`
324. `\*compress/flate.WriteError.Error`
325. `github.com/microsoft/go-mssqldb.Error.Error`
326. `go/types.Error.Error`
327. `\*github.com/go-playground/universal-translator.ErrBadPluralDefinition.Error`
328. `github.com/docker/docker/client.errConnectionFailed.Error`
329. `\*github.com/LerianStudio/lib-commons/commons.Response.Error`
330. `\*net/netip.parsePrefixError.Error`
331. `go.mongodb.org/mongo-driver/mongo.BulkWriteError.Error`
332. `\*github.com/go-openapi/jsonpointer.pointerError.Error`
333. `\*net/http.http2duplicatePseudoHeaderError.Error`
334. `\*golang.org/x/net/http2.headerFieldValueError.Error`
335. `crypto/rc4.KeySizeError.Error`
336. `\*github.com/containerd/errdefs.errConflict.Error`
337. `\*github.com/gofiber/fiber/v2/internal/schema.UnknownKeyError.Error`
338. `\*github.com/containerd/errdefs.errFailedPrecondition.Error`
339. `\*golang.org/x/net/webdav/internal/xml.TagPathError.Error`
340. `github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
341. `\*golang.org/x/text/encoding/internal.RepertoireError.Error`
342. `\*go.mongodb.org/mongo-driver/mongo.WriteError.Error`
343. `github.com/rabbitmq/amqp091-go.Error.Error`
344. `compress/flate.CorruptInputError.Error`
345. `\*github.com/LerianStudio/lib-commons/v4/commons/mongo.SanitizedError.Error`
346. `\*go.mongodb.org/mongo-driver/x/mongo/driver.Error.Error`
347. `github.com/containerd/errdefs.errPermissionDenied.Error`
348. `crypto/x509.InsecureAlgorithmError.Error`
349. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.TransitionError.Error`
350. `\*net/http.http2ConnectionError.Error`
351. `\*github.com/LerianStudio/lib-license-go/v2/pkg.EntityConflictError.Error`
352. `\*github.com/containerd/errdefs.errInvalidArgument.Error`
353. `\*golang.org/x/crypto/ssh.PartialSuccessError.Error`
354. `\*github.com/go-openapi/swag/jsonutils/adapters.registryError.Error`
355. `\*github.com/jackc/pgx/v5/pgconn.pgconnError.Error`
356. `\*github.com/redis/go-redis/v9/internal/proto.ClusterDownError.Error`
357. `github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
358. `github.com/containerd/errdefs.errNotFound.Error`
359. `\*internal/chacha8rand.errUnmarshalChaCha8.Error`
360. `github.com/LerianStudio/lib-license-go/v2/pkg.EntityNotFoundError.Error`
361. `\*google.golang.org/grpc/internal/transport.NewStreamError.Error`
362. `\*net.timeoutError.Error`
363. `go.opentelemetry.io/otel/sdk/trace.errUnsupportedSampler.Error`
364. `\*internal/fuzz.MalformedCorpusError.Error`
365. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationUnknownFieldsError.Error`
366. `\*github.com/jackc/pgx/v5/pgconn.NotPreferredError.Error`
367. `net/netip.parsePrefixError.Error`
368. `\*google.golang.org/grpc.dropError.Error`
369. `github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
370. `\*github.com/LerianStudio/lib-license-go/v2/pkg.HTTPError.Error`
371. `crypto/des.KeySizeError.Error`
372. `net/http.transportReadFromServerError.Error`
373. `github.com/containerd/errdefs.errAborted.Error`
374. `\*github.com/gofiber/fiber/v2/internal/schema.EmptyFieldError.Error`
375. `github.com/containerd/errdefs.errDataLoss.Error`
376. `go.opentelemetry.io/otel/sdk/trace.samplerArgParseError.Error`
377. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.multiErr.Error`
378. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteErrors.Error`
379. `archive/tar.headerError.Error`
380. `github.com/go-logr/logr.notFoundError.Error`
381. `\*golang.org/x/net/http2.GoAwayError.Error`
382. `\*net/url.InvalidHostError.Error`
383. `\*os.LinkError.Error`
384. `\*go.mongodb.org/mongo-driver/x/mongo/driver.QueryFailureError.Error`
385. `go.mongodb.org/mongo-driver/mongo.WriteConcernError.Error`
386. `internal/strconv.Error.Error`
387. `\*net.addrinfoErrno.Error`
388. `\*net.OpError.Error`
389. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.WaitQueueTimeoutError.Error`
390. `text/template.ExecError.Error`
391. `\*go.uber.org/multierr.multiError.Error`
392. `\*os.errSymlink.Error`
393. `\*github.com/valyala/fasthttp.ErrNothingRead.Error`
394. `go.mongodb.org/mongo-driver/bson/bsoncodec.ValueEncoderError.Error`
395. `\*math/big.ErrNaN.Error`
396. `crypto/internal/fips140/hmac.errCloneUnsupported.Error`
397. `\*encoding/json.UnmarshalFieldError.Error`
398. `\*go.mongodb.org/mongo-driver/bson/bsonrw.errMaxDocumentSizeExceeded.Error`
399. `net/http.http2pseudoHeaderError.Error`
400. `\*golang.org/x/crypto/ocsp.ResponseError.Error`
401. `github.com/gofiber/fiber/v2/internal/schema.EmptyFieldError.Error`
402. `\*github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
403. `\*github.com/go-playground/universal-translator.ErrBadParamSyntax.Error`
404. `\*github.com/docker/docker/api/types/registry.errInvalidParameter.Error`
405. `\*golang.org/x/net/http2.duplicatePseudoHeaderError.Error`
406. `\*net.DNSConfigError.Error`
407. `\*os/user.UnknownUserError.Error`
408. `\*google.golang.org/grpc/internal/transport.ioError.Error`
409. `github.com/LerianStudio/lib-license-go/v2/pkg.InternalServerError.Error`
410. `\*golang.org/x/crypto/blowfish.KeySizeError.Error`
411. `\*runtime.errorAddressString.Error`
412. `go.uber.org/zap.errArrayElem.Error`
413. `runtime.boundsError.Error`
414. `net/http.http2GoAwayError.Error`
415. `github.com/pkg/errors.withStack.Error`
416. `go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InsufficientBytesError.Error`
417. `\*go.opentelemetry.io/otel/trace.errorConst.Error`
418. `\*golang.org/x/net/http2.pseudoHeaderError.Error`
419. `\*github.com/LerianStudio/lib-license-go/v2/pkg.EntityNotFoundError.Error`
420. `\*github.com/LerianStudio/lib-commons/v4/commons.Response.Error`
421. `crypto/internal/fips140/aes.KeySizeError.Error`
422. `\*go.mongodb.org/mongo-driver/mongo.CommandError.Error`
423. `github.com/containerd/errdefs.errAlreadyExists.Error`
424. `golang.org/x/net/http2.GoAwayError.Error`
425. `\*github.com/containerd/errdefs.errOutOfRange.Error`
426. `\*golang.org/x/net/webdav/internal/xml.SyntaxError.Error`
427. `\*go/build.NoGoError.Error`
428. `\*github.com/tryvium-travels/memongo/mongobin.UnsupportedSystemError.Error`
429. `\*os/signal.signalError.Error`
430. `\*os/user.UnknownGroupError.Error`
431. `\*os/exec.wrappedError.Error`
432. `go.mongodb.org/mongo-driver/bson/bsoncodec.TransitionError.Error`
433. `\*github.com/docker/docker/api/types/container.errInvalidParameter.Error`
434. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ValueEncoderError.Error`
435. `\*github.com/containerd/errdefs.errNotImplemented.Error`
436. `golang.org/x/net/idna.labelError.Error`
437. `\*go/types.Error.Error`
438. `github.com/microsoft/go-mssqldb.ServerError.Error`
439. `encoding/xml.UnmarshalError.Error`
440. `\*github.com/go-playground/validator/v10.fieldError.Error`
441. `\*github.com/microsoft/go-mssqldb.ServerError.Error`
442. `\*html/template.Error.Error`
443. `\*go.mongodb.org/mongo-driver/mongo/options.MarshalError.Error`
444. `\*compress/bzip2.StructuralError.Error`
445. `\*golang.org/x/crypto/ocsp.ParseError.Error`
446. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.MalformedElementError.Error`
447. `\*go.opentelemetry.io/otel/sdk/trace.errUnsupportedSampler.Error`
448. `\*go/scanner.Error.Error`
449. `github.com/LerianStudio/lib-license-go/v2/pkg.EntityConflictError.Error`
450. `\*golang.org/x/net/http2/hpack.InvalidIndexError.Error`
451. `\*github.com/jackc/pgx/v5/pgconn.errTimeout.Error`
452. `\*github.com/valyala/fasthttp/fasthttputil.timeoutError.Error`
453. `\*net/http.transportReadFromServerError.Error`
454. `\*github.com/cenkalti/backoff/v5.PermanentError.Error`
455. `github.com/gofiber/fiber/v2/internal/schema.MultiError.Error`
456. `\*golang.org/x/crypto/ssh.OpenChannelError.Error`
457. `go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt.Error.Error`
458. `\*time.ParseError.Error`
459. `github.com/valyala/fasthttp.ErrSmallBuffer.Error`
460. `\*github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
461. `net/mail.charsetError.Error`
462. `\*github.com/redis/go-redis/v9/internal/proto.NoReplicasError.Error`
463. `\*github.com/redis/go-redis/v9/internal/proto.PermissionError.Error`
464. `\*encoding/asn1.StructuralError.Error`
465. `\*go.mongodb.org/mongo-driver/internal/aws/awserr.errorList.Error`
466. `github.com/containerd/errdefs.errUnauthorized.Error`
467. `\*github.com/microsoft/go-mssqldb.StreamError.Error`
468. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.InvalidDepthTraversalError.Error`
469. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteError.Error`
470. `\*internal/strconv.Error.Error`
471. `\*go.mongodb.org/mongo-driver/x/mongo/driver/topology.ServerSelectionError.Error`
472. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ForbiddenError.Error`
473. `\*golang.org/x/crypto/ssh.BannerError.Error`
474. `\*net/http.http2connError.Error`
475. `go.mongodb.org/mongo-driver/mongo.MarshalError.Error`
476. `\*github.com/redis/go-redis/v9/internal/pool.BadConnError.Error`
477. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ValueDecoderError.Error`
478. `os/user.UnknownUserIdError.Error`
479. `\*net.UnknownNetworkError.Error`
480. `\*golang.org/x/crypto/ssh.disconnectMsg.Error`
481. `\*net/http.http2httpError.Error`
482. `\*github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
483. `github.com/docker/docker/api/types/registry.errInvalidParameter.Error`
484. `github.com/grpc-ecosystem/grpc-gateway/v2/runtime.MalformedSequenceError.Error`
485. `\*internal/bisect.parseError.Error`
486. `\*github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
487. `github.com/redis/go-redis/v9/internal/proto.RedisError.Error`
488. `go.mongodb.org/mongo-driver/mongo.WriteError.Error`
489. `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal.PartialSuccess.Error`
490. `\*github.com/go-resty/resty/v2.ResponseError.Error`
491. `\*github.com/valyala/fasthttp.InvalidHostError.Error`
492. `\*encoding/asn1.SyntaxError.Error`
493. `\*regexp/syntax.Error.Error`
494. `\*github.com/moby/term.EscapeError.Error`
495. `\*go/build.MultiplePackageError.Error`
496. `github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
497. `\*github.com/containerd/errdefs.errUnknown.Error`
498. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoDecoder.Error`
499. `\*go.uber.org/zap.errSinkNotFound.Error`
500. `\*go.mongodb.org/mongo-driver/mongo.MongocryptError.Error`
501. `github.com/valyala/fasthttp.InvalidHostError.Error`
502. `go.mongodb.org/mongo-driver/bson/bsonrw.errMaxDocumentSizeExceeded.Error`
503. `\*github.com/docker/docker/api/types/filters.invalidFilter.Error`
504. `\*github.com/jackc/pgx/v5/pgconn.ParseConfigError.Error`
505. `golang.org/x/crypto/ssh.cbcError.Error`
506. `google.golang.org/grpc.dropError.Error`
507. `\*github.com/valyala/fasthttp.ErrDialWithUpstream.Error`
508. `\*github.com/andybalholm/brotli.decodeError.Error`
509. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ValidationKnownFieldsError.Error`
510. `\*github.com/LerianStudio/lib-commons/v2/commons.Response.Error`
511. `\*github.com/go-playground/universal-translator.ErrConflictingTranslation.Error`
512. `\*github.com/grpc-ecosystem/grpc-gateway/v2/runtime.MalformedSequenceError.Error`
513. `\*net/http.ProtocolError.Error`
514. `\*encoding/xml.UnsupportedTypeError.Error`
515. `\*github.com/containerd/errdefs.errUnauthorized.Error`
516. `\*github.com/LerianStudio/fetcher/pkg.UnprocessableOperationError.Error`
517. `os/signal.signalError.Error`
518. `github.com/testcontainers/testcontainers-go.ParallelContainersError.Error`
519. `go.mongodb.org/mongo-driver/internal/aws/awserr.errorList.Error`
520. `\*github.com/LerianStudio/lib-commons/v4/commons/runtime.panicError.Error`
521. `encoding/asn1.SyntaxError.Error`
522. `github.com/containerd/errdefs.errFailedPrecondition.Error`
523. `github.com/go-openapi/swag/jsonutils/adapters.registryError.Error`
524. `github.com/containerd/errdefs.errUnknown.Error`
525. `\*github.com/containerd/errdefs.errInternal.Error`
526. `\*google.golang.org/protobuf/internal/impl.errInvalidUTF8.Error`
527. `\*encoding/json.UnsupportedValueError.Error`
528. `\*github.com/microsoft/go-mssqldb.Error.Error`
529. `\*github.com/gofiber/fiber/v2/internal/schema.ConversionError.Error`
530. `github.com/grpc-ecosystem/grpc-gateway/v2/internal/httprule.InvalidTemplateError.Error`
531. `\*encoding/json.jsonError.Error`
532. `\*github.com/yuin/gopher-lua.ApiError.Error`
533. `\*github.com/go-resty/resty/v2.restyError.Error`
534. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal.PartialSuccess.Error`
535. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteCommandError.Error`
536. `\*gopkg.in/yaml.v3.TypeError.Error`
537. `\*encoding/hex.InvalidByteError.Error`
538. `\*github.com/go-playground/universal-translator.ErrMissingLocale.Error`
539. `\*go.mongodb.org/mongo-driver/internal/aws/awserr.baseError.Error`
540. `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal.PartialSuccess.Error`
541. `\*net.canceledError.Error`
542. `golang.org/x/net/webdav/internal/xml.UnmarshalError.Error`
543. `net/http.http2ConnectionError.Error`
544. `\*github.com/LerianStudio/lib-license-go/v2/pkg.FailedPreconditionError.Error`
545. `github.com/microsoft/go-mssqldb.StreamError.Error`
546. `github.com/LerianStudio/lib-license-go/v2/pkg.UnprocessableOperationError.Error`
547. `\*go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc/internal.PartialSuccess.Error`
548. `\*go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc/internal.PartialSuccess.Error`
549. `\*go.mongodb.org/mongo-driver/mongo.BulkWriteException.Error`
550. `golang.org/x/crypto/ocsp.ParseError.Error`
551. `go.opentelemetry.io/otel/trace.errorConst.Error`
552. `\*github.com/yuin/gopher-lua.CompileError.Error`
553. `\*go.mongodb.org/mongo-driver/x/mongo/driver.WriteConcernError.Error`
554. `math/big.ErrNaN.Error`
555. `\*os/user.UnknownGroupIdError.Error`
556. `\*encoding/csv.ParseError.Error`
557. `\*github.com/jackc/pgx/v5/pgconn.PrepareError.Error`
558. `github.com/golang-jwt/jwt/v5.joinedError.Error`
559. `gopkg.in/go-playground/validator.v9.ValidationErrors.Error`
560. `crypto/x509/internal/macos.OSStatus.Error`
561. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.decodeBinaryError.Error`
562. `\*github.com/docker/docker/client.objectNotFoundError.Error`
563. `go.mongodb.org/mongo-driver/bson/bsoncodec.decodeBinaryError.Error`
564. `github.com/LerianStudio/fetcher/pkg.HTTPError.Error`
565. `net/http.http2headerFieldValueError.Error`
566. `go.mongodb.org/mongo-driver/x/mongo/driver.WriteError.Error`
567. `\*go.yaml.in/yaml/v3.TypeError.Error`
568. `\*go.mongodb.org/mongo-driver/mongo.MarshalError.Error`
569. `\*golang.org/x/mod/module.InvalidVersionError.Error`
570. `\*encoding/json.InvalidUnmarshalError.Error`
571. `\*github.com/tryvium-travels/memongo/mongobin.UnsupportedMongoVersionError.Error`
572. `github.com/LerianStudio/lib-license-go/v2/pkg.FailedPreconditionError.Error`
573. `runtime.errorString.Error`
574. `\*net.DNSError.Error`
575. `\*github.com/LerianStudio/lib-license-go/v2/pkg.ResponseError.Error`
576. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoDecoder.Error`
577. `\*github.com/LerianStudio/fetcher/pkg.ResponseError.Error`
578. `github.com/klauspost/compress/flate.InternalError.Error`
579. `\*github.com/klauspost/compress/flate.InternalError.Error`
580. `\*github.com/LerianStudio/fetcher/pkg.InternalServerError.Error`
581. `\*runtime.synctestDeadlockError.Error`
582. `\*github.com/redis/go-redis/v9/internal/proto.MaxClientsError.Error`
583. `\*github.com/valyala/fasthttp.timeoutError.Error`
584. `\*io/fs.PathError.Error`
585. `\*github.com/docker/docker/client.emptyIDError.Error`
586. `\*internal/runtime/maps.unhashableTypeError.Error`
587. `github.com/docker/docker/api/types/filters.invalidFilter.Error`
588. `go/scanner.ErrorList.Error`
589. `\*net/http.http2goAwayFlowError.Error`
590. `net.UnknownNetworkError.Error`
591. `\*net/url.EscapeError.Error`
592. `go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoTypeMapEntry.Error`
593. `\*github.com/go-playground/universal-translator.ErrMissingPluralTranslation.Error`
594. `\*github.com/containerd/errdefs.customMessage.Error`
595. `net/http.statusError.Error`
596. `\*github.com/rabbitmq/amqp091-go.Error.Error`
597. `\*github.com/docker/docker/api/types.ErrorResponse.Error`
598. `github.com/containerd/errdefs.errInternal.Error`
599. `\*vendor/golang.org/x/net/http2/hpack.InvalidIndexError.Error`
600. `\*github.com/lib/pq.safeRetryError.Error`
601. `runtime.plainError.Error`
602. `\*github.com/alicebob/miniredis/v2/gopher-json.invalidTypeError.Error`
603. `\*go.mongodb.org/mongo-driver/mongo.EncryptionKeyVaultError.Error`
604. `\*net.AddrError.Error`
605. `github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
606. `\*net/url.Error.Error`
607. `github.com/containerd/errdefs.errUnavailable.Error`
608. `\*net/mail.charsetError.Error`
609. `\*crypto/x509/internal/macos.OSStatus.Error`
610. `\*github.com/sijms/go-ora/v2/network.OracleError.Error`
611. `\*internal/fuzz.crashError.Error`
612. `\*net/textproto.ProtocolError.Error`
613. `\*github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
614. `github.com/andybalholm/brotli.decodeError.Error`
615. `\*go.mongodb.org/mongo-driver/x/bsonx/bsoncore.ValidationError.Error`
616. `\*github.com/containerd/errdefs.errNotFound.Error`
617. `\*github.com/redis/go-redis/v9/internal/proto.AskError.Error`
618. `go.mongodb.org/mongo-driver/mongo.CommandError.Error`
619. `os/exec.wrappedError.Error`
620. `\*net.notFoundError.Error`
621. `crypto/x509.UnknownAuthorityError.Error`
622. `crypto/x509.HostnameError.Error`
623. `github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
624. `\*github.com/go-playground/universal-translator.ErrRangeTranslation.Error`
625. `github.com/LerianStudio/lib-license-go/v2/pkg.ValidationKnownFieldsError.Error`
626. `\*runtime.plainError.Error`
627. `github.com/LerianStudio/fetcher/pkg.ResponseErrorWithStatusCode.Error`
628. `golang.org/x/net/http2.connError.Error`
629. `\*fmt.wrapErrors.Error`
630. `\*github.com/LerianStudio/lib-commons/v4/commons/assert.AssertionError.Error`
631. `go.mongodb.org/mongo-driver/mongo.MongocryptdError.Error`
632. `\*google.golang.org/grpc/internal/transport.ConnectionError.Error`
633. `\*net/http.requestBodyReadError.Error`
634. `\*compress/flate.ReadError.Error`
635. `\*syscall.Errno.Error`
636. `\*github.com/valyala/fasthttp/reuseport.ErrNoReusePort.Error`
637. `\*golang.org/x/net/http2.connError.Error`
638. `runtime.synctestDeadlockError.Error`
639. `\*compress/flate.CorruptInputError.Error`
640. `\*github.com/redis/go-redis/v9/internal/proto.TryAgainError.Error`
641. `github.com/redis/go-redis/v9/internal/pool.BadConnError.Error`
642. `github.com/jackc/pgx/v5.ScanArgError.Error`
643. `\*net/http.MaxBytesError.Error`
644. `\*golang.org/x/crypto/ssh.AlgorithmNegotiationError.Error`
645. `\*go.mongodb.org/mongo-driver/bson/bsoncodec.ErrNoTypeMapEntry.Error`
646. `github.com/google/uuid.invalidLengthError.Error`
647. `\*github.com/LerianStudio/fetcher/pkg.ValidationKnownFieldsError.Error`
648. `\*golang.org/x/net/http2.StreamError.Error`
649. `\*github.com/testcontainers/testcontainers-go/wait.PermanentError.Error`
650. `\*github.com/cenkalti/backoff/v5.RetryAfterError.Error`
651. `github.com/alicebob/miniredis/v2/gopher-json.invalidTypeError.Error`
652. `github.com/valyala/fasthttp.ErrBrokenChunk.Error`
653. `\*golang.org/x/crypto/ssh.PassphraseMissingError.Error`
654. `\*github.com/LerianStudio/fetcher/pkg.FailedPreconditionError.Error`
655. `google.golang.org/grpc/internal/transport.NewStreamError.Error`
656. `\*github.com/containerd/errdefs.errAlreadyExists.Error`
657. `net.addrinfoErrno.Error`
658. `go.mongodb.org/mongo-driver/mongo.MongocryptError.Error`
659. `encoding/base64.CorruptInputError.Error`
660. `crypto/tls.AlertError.Error`
661. `\*encoding/json.SyntaxError.Error`
662. `\*net/textproto.Error.Error`
663. `\*github.com/jackc/pgx/v5/pgproto3.writeError.Error`
664. `\*github.com/go-openapi/swag/yamlutils.yamlError.Error`
665. `\*runtime.PanicNilError.Error`
666. `github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
667. `github.com/LerianStudio/fetcher/pkg.ForbiddenError.Error`
668. `\*github.com/go-playground/universal-translator.ErrMissingBracket.Error`
669. `encoding/asn1.StructuralError.Error`
670. `compress/flate.InternalError.Error`
671. `\*crypto/tls.echConfigErr.Error`
672. `\*github.com/google/uuid.invalidLengthError.Error`
673. `\*github.com/LerianStudio/lib-license-go/v2/pkg.InternalServerError.Error`
674. `go.mongodb.org/mongo-driver/mongo.BulkWriteException.Error`
675. `github.com/go-playground/validator/v10.ValidationErrors.Error`
676. `\*go.mongodb.org/mongo-driver/mongo.ErrMapForOrderedArgument.Error`
677. `\*github.com/LerianStudio/fetcher/pkg.ValidationUnknownFieldsError.Error`
678. `github.com/containerd/errdefs.errResourceExhausted.Error`
679. `go.mongodb.org/mongo-driver/x/mongo/driver.ResponseError.Error`
680. `\*github.com/LerianStudio/fetcher/pkg.ValidationError.Error`
681. `\*encoding/json.UnsupportedTypeError.Error`
682. `\*encoding/xml.SyntaxError.Error`
683. `\*github.com/jackc/pgx/v5/pgproto3.invalidMessageLenErr.Error`
684. `\*crypto/internal/fips140/aes.KeySizeError.Error`
685. `debug/dwarf.DecodeError.Error`
686. `\*go.mongodb.org/mongo-driver/mongo.WriteErrors.Error`
687. `\*go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/transform.errMetric.Error`
688. `os/user.UnknownGroupError.Error`
689. `\*os/exec.ExitError.Error`
690. `\*net/http.statusError.Error`
691. `\*golang.org/x/text/internal/language.ValueError.Error`
692. `golang.org/x/text/internal/language.ValueError.Error`
693. `crypto/x509.CertificateInvalidError.Error`
694. `crypto/tls.RecordHeaderError.Error`
695. `\*internal/poll.errNetClosing.Error`
696. `\*debug/macho.FormatError.Error`
697. `github.com/LerianStudio/lib-license-go/v2/pkg.ForbiddenError.Error`
698. `\*github.com/LerianStudio/fetcher/pkg.UnauthorizedError.Error`
699. `\*github.com/go-openapi/swag/loading.loadingError.Error`
700. `\*github.com/docker/docker/pkg/jsonmessage.JSONError.Error`
701. `\*github.com/pkg/errors.withStack.Error`
702. `\*golang.org/x/net/http2.headerFieldNameError.Error`
703. `Contains`

#### `clearJobsCollection`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (10 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `newJobRepository` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/job/job.mongodb\_test.go:63`
5. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
6. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
7. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
8. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
9. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
10. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.common.Helper`
2. `\*testing.common.Fatalf`
3. `Background`
4. `\*github.com/LerianStudio/lib-commons/v4/commons/mongo.Client.Client`
5. `\*testing.common.Fatalf`
6. `ToLower`
7. `\*go.mongodb.org/mongo-driver/mongo.Client.Database`
8. `ToLower`
9. `\*go.mongodb.org/mongo-driver/mongo.Database.Collection`
10. `Background`
11. `\*go.mongodb.org/mongo-driver/mongo.Collection.Drop`
12. `As`
13. `\*testing.common.Fatalf`

#### `TestJobMongoDBRepository\_Create`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`
5. `\*testing.T.Run`
6. `\*testing.T.Run`

#### `TestNewJobMongoDBRepository\_NilClient`
**File:** `pkg/mongodb/job/job.mongodb\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `NewJobMongoDBRepository`
2. `\*testing.common.Fatalf`
3. `\*testing.common.Fatalf`

#### `MapMongoErrorToResponse`
**File:** `pkg/mongodb/mongo.go`
**Risk Level:** HIGH (90 direct callers)

**Direct Callers (signature change affects these):**

1. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Create` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:128`
2. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Create` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:155`
3. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Update` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:191`
4. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Update` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:226`
5. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.AssignProduct` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:723`
6. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.AssignProduct` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:753`
7. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:565`
8. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:581`
9. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:587`
10. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:602`
11. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:616`
12. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByOrganizationAndName` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:374`
13. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByOrganizationAndName` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:393`
14. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Delete` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:274`
15. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Delete` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:294`
16. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Delete` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:274`
17. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Delete` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:294`
18. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Create` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:128`
19. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Create` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:155`
20. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByOrganizationAndName` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:374`
21. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByOrganizationAndName` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:393`
22. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Update` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:187`
23. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Update` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:245`
24. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Delete` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:255`
25. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Delete` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:275`
26. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByOrganizationAndDatabaseName` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:433`
27. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByOrganizationAndDatabaseName` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:453`
28. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:403`
29. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:419`
30. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:425`
31. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:440`
32. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:454`
33. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByID` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:326`
34. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByID` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:345`
35. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByID` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:326`
36. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByID` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:345`
37. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Create` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:129`
38. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Create` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:151`
39. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Update` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:187`
40. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Update` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:245`
41. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.AssignProduct` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:723`
42. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.AssignProduct` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:753`
43. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.FindByCode` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:355`
44. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.FindByCode` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:374`
45. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Delete` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:255`
46. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Delete` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:275`
47. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.FindByID` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:307`
48. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.FindByID` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:326`
49. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:403`
50. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:419`
51. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:425`
52. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:440`
53. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:454`
54. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.FindByCode` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:355`
55. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.FindByCode` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:374`
56. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:565`
57. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:581`
58. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:587`
59. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:602`
60. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.List` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:616`
61. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Update` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:191`
62. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.Update` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:226`
63. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.ListUnassigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:641`
64. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.ListUnassigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:663`
65. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.ListUnassigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:669`
66. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.ListUnassigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:684`
67. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.ListUnassigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:698`
68. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByConfigNames` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:498`
69. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByConfigNames` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:516`
70. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByConfigNames` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:526`
71. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByConfigNames` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:540`
72. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Create` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:129`
73. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.Create` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:151`
74. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.ListUnassigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:641`
75. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.ListUnassigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:663`
76. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.ListUnassigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:669`
77. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.ListUnassigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:684`
78. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.ListUnassigned` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:698`
79. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByConfigNames` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:498`
80. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByConfigNames` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:516`
81. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByConfigNames` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:526`
82. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByConfigNames` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:540`
83. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByOrganizationAndDatabaseName` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:433`
84. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.FindByOrganizationAndDatabaseName` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:453`
85. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.CountByProduct` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:781`
86. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.CountByProduct` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:795`
87. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.CountByProduct` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:781`
88. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.ConnectionMongoDBRepository.CountByProduct` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/connection/connection.mongodb.go:795`
89. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.FindByID` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:307`
90. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.ProductMongoDBRepository.FindByID` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/product/product.mongodb.go:326`

**Callees (this function depends on):**

1. `NewLoggerFromContext`
2. `Is`
3. `Sprintf`
4. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
5. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
6. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
7. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
8. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
9. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
10. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
11. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
12. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
13. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
14. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
15. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
16. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
17. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
18. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
19. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
20. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
21. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
22. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
23. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
24. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
25. `ValidateInternalError`
26. `Is`
27. `Sprintf`
28. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
29. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
30. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
31. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
32. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
33. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
34. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
35. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
36. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
37. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
38. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
39. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
40. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
41. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
42. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
43. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
44. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
45. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
46. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
47. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
48. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
49. `ValidateInternalError`
50. `Is`
51. `IsTimeout`
52. `Sprintf`
53. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
54. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
55. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
56. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
57. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
58. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
59. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
60. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
61. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
62. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
63. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
64. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
65. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
66. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
67. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
68. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
69. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
70. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
71. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
72. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
73. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
74. `ValidateInternalError`
75. `As`
76. `Sprintf`
77. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
78. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
79. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
80. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
81. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
82. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
83. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
84. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
85. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
86. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
87. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
88. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
89. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
90. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
91. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
92. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
93. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
94. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
95. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
96. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
97. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
98. `ValidateInternalError`
99. `IsNetworkError`
100. `Sprintf`
101. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
102. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
103. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
104. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
105. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
106. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
107. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
108. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
109. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
110. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
111. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
112. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
113. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
114. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
115. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
116. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
117. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
118. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
119. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
120. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
121. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
122. `ValidateInternalError`
123. `Is`
124. `Sprintf`
125. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
126. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
127. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
128. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
129. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
130. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
131. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
132. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
133. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
134. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
135. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
136. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
137. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
138. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
139. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
140. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
141. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
142. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
143. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
144. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
145. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
146. `ValidateInternalError`
147. `IsDuplicateKeyError`
148. `Sprintf`
149. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
150. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
151. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
152. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
153. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
154. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
155. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
156. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
157. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
158. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
159. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
160. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
161. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
162. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
163. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
164. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
165. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
166. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
167. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
168. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
169. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
170. `ValidateInternalError`
171. `As`
172. `As`
173. `Sprintf`
174. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
175. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
176. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
177. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
178. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
179. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
180. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
181. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
182. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
183. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
184. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
185. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
186. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
187. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
188. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
189. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
190. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
191. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
192. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
193. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
194. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
195. `ValidateInternalError`
196. `Sprintf`
197. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
198. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
199. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
200. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
201. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
202. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
203. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
204. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
205. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
206. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
207. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
208. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
209. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
210. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
211. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
212. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
213. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
214. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
215. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
216. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
217. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
218. `ValidateInternalError`
219. `Sprintf`
220. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
221. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
222. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
223. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
224. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
225. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
226. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
227. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
228. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
229. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
230. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
231. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
232. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
233. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
234. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
235. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
236. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
237. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
238. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
239. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
240. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
241. `ValidateInternalError`
242. `Sprintf`
243. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
244. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
245. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
246. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
247. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
248. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
249. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
250. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
251. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
252. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
253. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
254. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
255. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
256. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
257. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
258. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
259. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
260. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
261. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
262. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
263. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
264. `ValidateInternalError`
265. `Sprintf`
266. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
267. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
268. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
269. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
270. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
271. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
272. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
273. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
274. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
275. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
276. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
277. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
278. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
279. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
280. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
281. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
282. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
283. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
284. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
285. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
286. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
287. `ValidateInternalError`
288. `As`
289. `Sprintf`
290. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
291. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
292. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
293. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
294. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
295. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
296. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
297. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
298. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
299. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
300. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
301. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
302. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
303. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
304. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
305. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
306. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
307. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
308. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
309. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
310. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
311. `ValidateInternalError`
312. `Sprintf`
313. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
314. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
315. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
316. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
317. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
318. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
319. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
320. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
321. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
322. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
323. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
324. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
325. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
326. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
327. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
328. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
329. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
330. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
331. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
332. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
333. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
334. `ValidateInternalError`
335. `Sprintf`
336. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
337. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
338. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
339. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
340. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
341. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
342. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
343. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
344. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
345. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
346. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
347. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
348. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
349. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
350. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
351. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
352. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
353. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
354. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
355. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
356. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
357. `ValidateInternalError`
358. `Sprintf`
359. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
360. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
361. `github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
362. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
363. `\*github.com/LerianStudio/lib-commons/v4/commons/log.MockLogger.Log`
364. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
365. `\*github.com/LerianStudio/fetcher/pkg/testutil.MockLogger.Log`
366. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
367. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
368. `\*github.com/LerianStudio/lib-commons/v4/commons/log.GoLogger.Log`
369. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
370. `github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
371. `\*github.com/LerianStudio/lib-commons/v4/commons/log.NopLogger.Log`
372. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
373. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
374. `\*github.com/LerianStudio/fetcher/components/manager/internal/bootstrap.Service.Log`
375. `\*github.com/LerianStudio/fetcher/components/worker/internal/bootstrap.Service.Log`
376. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
377. `\*github.com/LerianStudio/lib-commons/v4/commons/zap.Logger.Log`
378. `github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.ConsumerRoutes.Log`
379. `\*github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq.PublisherRoutes.Log`
380. `ValidateInternalError`

#### `PingMongo`
**File:** `pkg/mongodb/mongo.go`
**Risk Level:** MEDIUM (4 direct callers)

**Direct Callers (signature change affects these):**

1. `TestPingMongo$1` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo\_test.go:281`
2. `TestPingMongo$3` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo\_test.go:311`
3. `TestPingMongo$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo\_test.go:295`
4. `TestPingMongo$4` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mongodb/mongo\_test.go:324`

**Callees (this function depends on):**

1. `New`
2. `\*github.com/LerianStudio/lib-commons/v4/commons/mongo.Client.Client`
3. `\*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.Client`
4. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockmongoDatabaseProvider.Client`
5. `\*github.com/LerianStudio/fetcher/pkg/mongodb/product.stubMongoDatabaseProvider.Client`
6. `github.com/LerianStudio/fetcher/pkg/mongodb/product.stubMongoDatabaseProvider.Client`
7. `\*github.com/LerianStudio/fetcher/pkg/mongodb/connection.MockmongoDatabaseProvider.Client`
8. `\*github.com/LerianStudio/fetcher/pkg/mongodb.MockMongoClientProvider.Client`
9. `Errorf`
10. `WithTimeout`
11. `init$5`
12. `init$bound`
13. `DefPredeclaredTestFuncs`
14. `init$bound`
15. `Parse$1`
16. `OnceValues\[map\[string\]rune map\[string\]\[2\]rune\]$1$1$1`
17. `\_GC`
18. `init\#2`
19. `libc\_munlock\_trampoline`
20. `init\#3`
21. `init\#12`
22. `init\#1`
23. `logUnexpectedFailure$1`
24. `init\#1`
25. `queryDatabase$1`
26. `initP256`
27. `fuzz$1`
28. `encap$2`
29. `libc\_setpgid\_trampoline`
30. `acquireThread$1`
31. `libc\_setuid\_trampoline`
32. `fRunner$1`
33. `init\#2`
34. `libc\_kill\_trampoline`
35. `main`
36. `arc4random\_buf\_trampoline`
37. `setDefaultRuntimeProviders`
38. `makeSha256Reader$1`
39. `MockEnv$1`
40. `TestRabbitMQAdapter\_ProcessDelivery\_ExtractsHeaders$1$3`
41. `tRunner$1`
42. `cancellationListenerCallback$bound`
43. `addTLS$1`
44. `libc\_faccessat\_trampoline`
45. `SafeGoWithContextAndComponent$1`
46. `pthread\_kill\_trampoline`
47. `watchCancel$1`
48. `x509\_CFStringCreateExternalRepresentation\_trampoline`
49. `readRequest$1`
50. `setitimer\_trampoline`
51. `init\#3`
52. `handleRawConn$1`
53. `stubConnectionSpanAttributes$2`
54. `traceParameterStatus$1`
55. `traceBackendKeyData$1`
56. `onShutdownTimer$bound`
57. `open$1`
58. `TestScanRows$2$1`
59. `dispatchDeliveries$1`
60. `init$1`
61. `debugCallWrap2$1`
62. `TestRabbitMQAdapter\_ProducerDefault\_SkipsSigningWhenDisabled$1`
63. `entersyscallblock$3`
64. `fixedHuffmanDecoderInit`
65. `OnceValue\[\[\]string\]$1$1`
66. `setDefaultOSProviders`
67. `init\#1`
68. `typInternal$2`
69. `setMemoryLimit$1`
70. `snapshotMetricsRegistryForTesting$1`
71. `libc\_fpathconf\_trampoline`
72. `clientGetURLDeadline$1`
73. `startupMessage$5`
74. `init\#1`
75. `Clearenv`
76. `init\#1`
77. `init\#1`
78. `startHealthCheck$4`
79. `HostGatewayIP$1`
80. `newClientStreamWithParams$3`
81. `libc\_kevent\_trampoline`
82. `ReadFile$1`
83. `sharedMemTempFile$1`
84. `libc\_pread\_trampoline`
85. `init\#3`
86. `reentersyscall$1`
87. `init\#1`
88. `eventsTmpl$1`
89. `Compile$1`
90. `main`
91. `getGCMaskOnDemand$1`
92. `init\#9`
93. `newTypeObject$1`
94. `init\#1`
95. `TestRabbitMQAdapter\_ConsumerLoop\_ReturnsNilOnContextCancellation$1`
96. `LockOSThread`
97. `stop$bound`
98. `Gosched`
99. `libc\_getdtablesize\_trampoline`
100. `init\#2$4`
101. `legacyLoadMessageDesc$1$1`
102. `poll$1`
103. `sendBatchExtendedWithDescription$1`
104. `lazyInit$1`
105. `file\_opentelemetry\_proto\_trace\_v1\_trace\_proto\_init`
106. `start$1`
107. `syscall\_runtime\_AfterForkInChild`
108. `init\#1`
109. `TestInitWorker\_PanicsWhenLoggerInitFails$4`
110. `NewHTTP2Client$3`
111. `New\[\*crypto/internal/fips140/sha512.Digest\]$1`
112. `init\#1$1$1`
113. `SaveMultipartFile$1`
114. `CloseNotify$1`
115. `spillArgs`
116. `NotifyContext$1`
117. `Shutdown$1`
118. `init\#1`
119. `freemcache$1`
120. `SendMsg$2`
121. `Build$1`
122. `init\#2`
123. `collectTypeParams$1`
124. `Add$1`
125. `libc\_read\_trampoline`
126. `libresolv\_res\_9\_ninit\_trampoline`
127. `Less$1$1`
128. `Pull2$3`
129. `main`
130. `file\_google\_protobuf\_duration\_proto\_init`
131. `initMime`
132. `init\#1`
133. `sendFile$1`
134. `TestRabbitMQAdapter\_ConsumerLoop\_VerifiesSignatureSuccessfully$1`
135. `initMetrics`
136. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnInvalidSignature$1`
137. `cgoBindM`
138. `chansend$1`
139. `x509\_SecPolicyCreateSSL\_trampoline`
140. `runtime\_procUnpin`
141. `runCleanup$1`
142. `libc\_getnameinfo\_trampoline`
143. `file\_opentelemetry\_proto\_collector\_logs\_v1\_logs\_service\_proto\_init`
144. `getConn$1`
145. `init\#1`
146. `callContinuation$1`
147. `generatorTable$1`
148. `p224SqrtCandidate$1`
149. `resetspinning`
150. `acquireForkLock`
151. `sigpanic`
152. `CopyFrom$1`
153. `libc\_ioctl\_trampoline`
154. `readValue$1`
155. `instantiatedType$2`
156. `probe$bound`
157. `Ping$1`
158. `AddSet$1`
159. `runHandler$1`
160. `invokeMarshaler$1`
161. `newClusterState$1`
162. `dropg`
163. `onceSetNextProtoDefaults$bound`
164. `printunlock`
165. `sysctlbyname\_trampoline`
166. `dumpmemprof`
167. `kqueue\_trampoline`
168. `ParseBase$1`
169. `ReadTrace$1`
170. `TestRabbitMQAdapter\_ConsumerLoop\_ReturnsNilOnContextDeadlineExceeded$1`
171. `prepareDC$2`
172. `alloc$1`
173. `init\#1`
174. `runN$1`
175. `libc\_utimensat\_trampoline`
176. `SendMsg$1`
177. `TestValidateParametersNonMetadataKeys$1`
178. `init\#1`
179. `urandomRead$1`
180. `SendFile$1`
181. `traceStopReadCPU`
182. `main`
183. `file\_opentelemetry\_proto\_collector\_metrics\_v1\_metrics\_service\_proto\_rawDescGZIP$1`
184. `Notify$1$1`
185. `scheduleNextConnectionLocked$2`
186. `gcMarkDone$2`
187. `TestRedisCache\_NewRedisCache\_NilConnection\_Panics$1$1`
188. `init\#1`
189. `Close$1`
190. `asyncPreempt`
191. `init\#2`
192. `libc\_mkdirat\_trampoline`
193. `objDecl$1`
194. `TestRabbitMQAdapter\_ProcessDelivery\_ExtractsHeaders$1$1`
195. `Construct$1`
196. `put$1`
197. `traceParse$1`
198. `readType$1`
199. `freeSpan$1`
200. `handleForwards$bound`
201. `libc\_munlockall\_trampoline`
202. `libc\_getgrgid\_r\_trampoline`
203. `gcResetMarkState`
204. `retryLocked$1$1`
205. `runPiped$2`
206. `isParameterized$1`
207. `OnceValue\[map\[string\]reflect.Value\]$1$1`
208. `WithCallStackHelper$1`
209. `basepointNafTable$1`
210. `init\#1`
211. `trace$1`
212. `rawExpr$1`
213. `dumpms`
214. `collectExemplars\[int64\]$1`
215. `gcComputeStartingStackSize`
216. `withConn$1`
217. `ForceFlush$1`
218. `TestRabbitMQAdapter\_ProducerDefault\_SignsMessage$1`
219. `libc\_setsockopt\_trampoline`
220. `init\#2`
221. `ClearStringValidations$1`
222. `init\#1`
223. `libc\_getpgid\_trampoline`
224. `StopMetricsCollector`
225. `WriteOverlays$1`
226. `init\#2`
227. `Shutdown$1`
228. `createContext$8`
229. `libc\_symlink\_trampoline`
230. `stacklessWriterFunc$1`
231. `libpreinit`
232. `x509\_CFNumberGetValue\_trampoline`
233. `Peek$1`
234. `main$1`
235. `resolve$1`
236. `rlock$1`
237. `decIgnoreOpFor$1`
238. `Stop$1`
239. `captureStack$1`
240. `cgounimpl`
241. `libc\_freeaddrinfo\_trampoline`
242. `file\_google\_rpc\_error\_details\_proto\_rawDescGZIP$1`
243. `TestInitWorker\_PanicsWhenConfigLoadFails$1`
244. `stop$1`
245. `stacklessWriteBrotli$1`
246. `init\#1`
247. `run1$1`
248. `read\_trampoline`
249. `handlePing$1`
250. `maintain$4`
251. `noopRedeemer`
252. `goschedIfBusy`
253. `init\#1`
254. `connectOne$2`
255. `main$1`
256. `Test$1$1`
257. `init\#2`
258. `runtime\_AfterFork`
259. `FlushDNSCache`
260. `exitsyscall$2`
261. `EndTracingSpansInterceptor$1$1`
262. `fatalpanic$1`
263. `fRunner$1$1`
264. `worldStarted`
265. `handleSettings$2`
266. `init\#3`
267. `registerHTTPSProtocol$1`
268. `performHandoffInternal$1`
269. `initConfVal$1`
270. `WithCallStackHelper$2`
271. `itabsinit`
272. `checkFinalizersAndCleanups`
273. `validateNorm$1`
274. `main`
275. `prepareForRecovery$1`
276. `archInitIEEE`
277. `build$1`
278. `watchCancel$2`
279. `Shutdown$1`
280. `cancellationListenerCallback$bound`
281. `yaml\_parser\_fetch\_next\_token$1`
282. `Read$1`
283. `init\#1`
284. `Pull2$1$2`
285. `checkmcount`
286. `testAtomic64`
287. `x509\_CFDataCreate\_trampoline`
288. `Close$bound`
289. `traceSyncBatch$1`
290. `libc\_getsockopt\_trampoline`
291. `list$1`
292. `libc\_write\_trampoline`
293. `dispatchDeliveries$1$1`
294. `generatorTable$1`
295. `newUserArenaChunk$1`
296. `init\#1`
297. `checkMinIdleConns$1`
298. `init\#1`
299. `exposeHostPorts$2`
300. `libc\_socketpair\_trampoline`
301. `synctest\_inBubble$1`
302. `doRetryNotify\[github.com/docker/docker/api/types/build.ImageBuildResponse\]$1`
303. `OnceValues$1$1`
304. `internal\_sync\_runtime\_doSpin`
305. `Start$1`
306. `libc\_access\_trampoline`
307. `file\_google\_rpc\_error\_details\_proto\_init`
308. `init\#1`
309. `executeShutdown$1`
310. `OnceValue$1$1$1`
311. `init\#1`
312. `handleEnv`
313. `InitLocalEnvConfig$1`
314. `file\_google\_api\_httpbody\_proto\_init`
315. `TestServiceRun$1`
316. `NewServerTransport$2`
317. `libc\_pwrite\_trampoline`
318. `writeHeapProto$1`
319. `fatalthrow$1`
320. `forcegchelper`
321. `init\#1`
322. `walkRange$2$1`
323. `DisableDIT`
324. `lockRankMayTraceFlush`
325. `lazyInit$1`
326. `StandardCrypto`
327. `initResolutionCache`
328. `coverReport`
329. `initP224`
330. `init\#1`
331. `decPenalty$bound`
332. `envProxyFunc$1`
333. `handleForwards$bound`
334. `init\#1`
335. `p521B$1`
336. `getLen$1`
337. `init\#1`
338. `onWriteTimeout$bound`
339. `exec$1`
340. `exposeHostPort$1`
341. `init\#1`
342. `ChannelWithSubscriptions$1`
343. `operateHeaders$2`
344. `NewPeriodicReader$2`
345. `main`
346. `ServeHTTP$1`
347. `onceSetNextProtoDefaults$bound`
348. `tlsClientHandshake$1`
349. `main$2`
350. `mstartm0`
351. `TestingOnlyAbandon`
352. `TestRabbitMQAdapter\_IsHealthy\_ReturnsTrue\_WhenChannelOpen$1`
353. `goroutineProfileWithLabelsSync$3`
354. `write$2`
355. `unminit`
356. `WithRecover$1$1`
357. `init\#1`
358. `ForEachPackage$2`
359. `typeDecl$3`
360. `minitSignalStack`
361. `entersyscallWakeSysmon`
362. `healthCheck$bound`
363. `init\#1`
364. `ParseBuildInfo$1`
365. `mapv$1`
366. `init\#1`
367. `main`
368. `main`
369. `throw$1`
370. `init\#1`
371. `asminit`
372. `do$1`
373. `libc\_pathconf\_trampoline`
374. `Exec$1`
375. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$1`
376. `setDefaultUserProviders`
377. `TestRabbitMQAdapter\_RunConsumerCycle\_FailsOnQosError$1`
378. `NewTelemetry$1`
379. `badmorestackg0$1`
380. `DecodeFull$1`
381. `maybeRunStateHook$bound`
382. `initHealthCheck$1`
383. `panicwrap`
384. `libc\_undelete\_trampoline`
385. `init\#2`
386. `WithCancel$1`
387. `initLocal`
388. `SendMsg$2`
389. `ReplaceGlobals$1`
390. `flushallmcaches`
391. `unlockOSThread`
392. `mapinitnoop`
393. `mallocinit`
394. `Close$1`
395. `TestRabbitMQAdapter\_ProducerDefault\_RetriesOnFailure$3$1`
396. `mountStartupProcess$1`
397. `Start$1`
398. `init\#1`
399. `gcinit`
400. `initConnPool$bound`
401. `init\#1`
402. `serveStreams$1`
403. `Go$1$1`
404. `libc\_setgid\_trampoline`
405. `gcWriteBarrier6`
406. `init\#3`
407. `RegisterMetricsServiceHandlerFromEndpoint$1`
408. `lazyInit$1`
409. `getcp950$1`
410. `init\#1`
411. `gcenable`
412. `swap$1`
413. `healthCheck$bound`
414. `NewHTTP2Client$5`
415. `StartTimeStampUpdater`
416. `init\#1`
417. `panicBounds`
418. `GetOrDownloadMongod$1`
419. `Reset`
420. `libc\_sendto\_trampoline`
421. `libc\_sync\_trampoline`
422. `WithDeadlineCause$3`
423. `init\#1`
424. `Go$1`
425. `tRunner$2`
426. `WriteOverlays$2$1`
427. `cmdStream$1`
428. `CoordinateFuzzing$2`
429. `isImage$1`
430. `basepointTable$1`
431. `Record$1`
432. `walkRange$1`
433. `AddSet$1`
434. `init\#2`
435. `apply$1`
436. `goServe$1`
437. `runSafePointFn`
438. `gcControllerCommit`
439. `netpollBreak`
440. `Readdir$1`
441. `prepareNext$1`
442. `ServeConn$1`
443. `Put$1`
444. `libc\_utimes\_trampoline`
445. `startTheWorld$1`
446. `init\#1`
447. `init\#1`
448. `testSPWrite`
449. `Record$1`
450. `libc\_rename\_trampoline`
451. `triggerHealthCheck$1`
452. `TurnOn`
453. `netpollinit`
454. `mProf\_Malloc$1`
455. `gcRestoreSyncObjects`
456. `EventuallyWithT$1$1`
457. `pthread\_attr\_setdetachstate\_trampoline`
458. `minitSignals`
459. `init\#1`
460. `reentersyscall$4`
461. `worldStopped`
462. `init\#6`
463. `init\#4`
464. `Execute$1`
465. `SendMsg$1`
466. `NewBatchSpanProcessor$2`
467. `TryGo$1`
468. `TestRabbitMQAdapter\_ProcessDelivery\_RecoverFromPanic$3`
469. `init\#1`
470. `setupMiniredis$1`
471. `runCleanups`
472. `main`
473. `setupHijackConn$1`
474. `RebaseArchiveEntries$1`
475. `TestConcurrentEncryption$1`
476. `ArtifactDir$1`
477. `gcAssistAlloc$2`
478. `libc\_fstat\_trampoline`
479. `MakeTimeoutContext$1`
480. `blocking$2`
481. `libc\_getgid\_trampoline`
482. `libc\_mmap\_trampoline`
483. `getData$1`
484. `file\_grpc\_health\_v1\_health\_proto\_init`
485. `collectFileInfoForChanges$1`
486. `DialContext$1`
487. `init\#1`
488. `init\#1`
489. `connect$1`
490. `basepointTable$1`
491. `kevent\_trampoline`
492. `init\#2`
493. `startChannelWatcher$1`
494. `WaitN$1$1`
495. `Next$1`
496. `startCloseMonitor$1`
497. `searchInStaticDictionary$1`
498. `propagateCancel$2`
499. `writeRecordLocked$1`
500. `CreateLUT`
501. `reset$1`
502. `libc\_getsid\_trampoline`
503. `init\#2$2`
504. `init\#1`
505. `RangeEntries$1`
506. `EnableColorsStdout$1`
507. `Flush$bound`
508. `libc\_umask\_trampoline`
509. `libc\_exchangedata\_trampoline`
510. `addmoduledata`
511. `initBenchmarkFlags`
512. `outgoingGoAwayHandler$1`
513. `init\#1`
514. `Cleanup$1$1`
515. `markroot$1`
516. `main`
517. `libc\_setegid\_trampoline`
518. `TestRabbitMQAdapter\_ConsumerLoop\_SkipsVerificationWhenDisabled$1`
519. `writeTrace$1`
520. `libc\_open\_trampoline`
521. `freezetheworld`
522. `printArgs$3`
523. `sync\_atomic\_runtime\_procUnpin`
524. `sigaction\_trampoline`
525. `collectExemplars\[float64\]$1`
526. `doCall$2`
527. `setSyncObjectsUntraceable`
528. `resetProxyConfig`
529. `operateHeaders$1`
530. `startDialConnForLocked$1`
531. `abortStreamLocked$1`
532. `HostGatewayIP$1`
533. `dounlockOSThread`
534. `gcMarkTermination$1`
535. `UUID$1`
536. `SaveImagesWithOpts$1`
537. `secret\_eraseSecrets`
538. `libc\_getgrouplist\_trampoline`
539. `madvise\_trampoline`
540. `init\#2`
541. `OrderedUnmarshalJSON$1`
542. `newextram`
543. `badTimer`
544. `Go$1`
545. `setPinned$2`
546. `libc\_revoke\_trampoline`
547. `debugCallWrap1`
548. `onIdleTimeout$bound`
549. `delayedFlush$bound`
550. `file\_opentelemetry\_proto\_metrics\_v1\_metrics\_proto\_rawDescGZIP$1`
551. `unminitSignals`
552. `init$bound`
553. `Add$1`
554. `Confirm$1`
555. `getempty$1`
556. `init\#1`
557. `init\#13`
558. `main`
559. `init\#1`
560. `init\#1`
561. `funcLit$1`
562. `libc\_mkdir\_trampoline`
563. `signalMessage$1`
564. `cleanup$1`
565. `\_System`
566. `init\#1`
567. `bytes$1`
568. `handleStateChange$1$1`
569. `WithDeadlineCause$1`
570. `register$bound`
571. `validType0$1`
572. `runtime\_pollServerInit`
573. `doInRoot\[string\]$1`
574. `Close$1`
575. `Alignof$1`
576. `getcp936$1`
577. `traceStartReadCPU`
578. `ProjectRoot$1`
579. `load\_g`
580. `nanotime\_trampoline`
581. `init\#1`
582. `onIdleTimer$bound`
583. `init\#1$1`
584. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnVersionMismatch$1`
585. `cpuVariant$1`
586. `connect$1`
587. `lockRankMayQueueFinalizer`
588. `libc\_fork\_trampoline`
589. `Test$1`
590. `methodValueCall`
591. `Read$1`
592. `mstart0`
593. `TestInitWorker\_PanicsWhenConfigLoadFails$3`
594. `init\#1`
595. `libc\_getrusage\_trampoline`
596. `schedule`
597. `abort`
598. `RecordApproved`
599. `init\#2`
600. `TestHMACSigner\_ConcurrentSigning$1`
601. `UnreachableExceptTests`
602. `MustExtractDockerHost$1`
603. `TestRabbitMQAdapter\_ProcessDelivery\_HandlesNonStringHeaderID$1`
604. `run$2`
605. `initFuzzFlags`
606. `Consume$1`
607. `makeFuncStub`
608. `libc\_fcntl\_trampoline`
609. `mach\_vm\_region\_trampoline`
610. `gcMarkTinyAllocs`
611. `init\#1`
612. `x509\_SecCertificateCreateWithData\_trampoline`
613. `Import$1`
614. `panicdivide`
615. `init\#1`
616. `gcMarkDone`
617. `Add$1`
618. `NewServerTransport$3`
619. `convertAssignRows$2`
620. `libc\_setpriority\_trampoline`
621. `serveContent$1`
622. `init\#4`
623. `RunFuzzWorker$1$1`
624. `init\#1`
625. `sweep$1`
626. `Channel$1`
627. `init\#1`
628. `TestRabbitMQAdapter\_ProducerDefault\_AllRetriesFail$1`
629. `ContainerWait$1`
630. `NewServerTransport$1`
631. `watchCancel$1`
632. `SortSliceBetween$1`
633. `run$1`
634. `init\#4`
635. `init\#1`
636. `listenerBacklog$1`
637. `init\#1`
638. `goenvs`
639. `onceSetNextProtoDefaults$bound`
640. `init\#1`
641. `OnEmit$2`
642. `init\#1`
643. `init\#1`
644. `NewWithConfig$5`
645. `file\_opentelemetry\_proto\_common\_v1\_common\_proto\_init`
646. `libc\_readlink\_trampoline`
647. `readFrames$1`
648. `schedinit`
649. `xRegInitAlloc`
650. `Decode$1`
651. `libc\_geteuid\_trampoline`
652. `\_ExternalCode`
653. `file\_opentelemetry\_proto\_collector\_trace\_v1\_trace\_service\_proto\_rawDescGZIP$1`
654. `exportContext$1`
655. `init\#1`
656. `ResolverError$1`
657. `startGracefulShutdown$1`
658. `setRequestCancel$2`
659. `getcp1251$1`
660. `delayedFlush$bound`
661. `ignoreSIGSYS`
662. `initPostcodes`
663. `main`
664. `mProf\_Flush`
665. `privateLogw$1`
666. `didPanic$1`
667. `runtime\_AfterExec`
668. `init\#1`
669. `Read$1`
670. `libc\_setrlimit\_trampoline`
671. `InitLocalEnvConfig$1`
672. `init\#1`
673. `Events$1`
674. `traceExitedSyscall`
675. `Enable`
676. `TestInitWorker\_PanicsWhenLoggerInitFails$1`
677. `tunnel$2`
678. `init\#2`
679. `casgstatus$1`
680. `stacklessWriteZstd$1`
681. `InitLocalEnvConfig$1`
682. `gcMarkTermination$5`
683. `loadHTTPBytes$1$1`
684. `warnBlocked`
685. `stacklessWriteGzip$1`
686. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P256Point\]\]$1$1`
687. `OnceValues$1$1$1`
688. `writeBody$1`
689. `TestRabbitMQAdapter\_ProducerDefault\_SkipsSigningWhenNoSigner$1`
690. `gcWriteBarrier5`
691. `pthread\_mutex\_unlock\_trampoline`
692. `freeStackSpans`
693. `init\#1`
694. `x509\_CFArrayAppendValue\_trampoline`
695. `libc\_recvfrom\_trampoline`
696. `libc\_gai\_strerror\_trampoline`
697. `libc\_getpeername\_trampoline`
698. `Watch$1`
699. `init\#3`
700. `lock$1`
701. `Record$1`
702. `defaultGOMAXPROCSUpdateEnable`
703. `infer$1`
704. `morestackc`
705. `doRetryNotify\[\*github.com/docker/docker/api/types/container.Summary\]$1`
706. `readPreface$1`
707. `modulesinit`
708. `PCall$1`
709. `init$2`
710. `write$1`
711. `FreeOSMemory`
712. `libc\_chmod\_trampoline`
713. `Default$1`
714. `setDefaultOSDescriptionProvider`
715. `init\#1`
716. `cgoSigtramp`
717. `ResetCoverage`
718. `runCmdContext$4`
719. `init\#5`
720. `typInternal$1`
721. `libc\_setreuid\_trampoline`
722. `gcStart$1`
723. `logDropped\[go.opentelemetry.io/otel/sdk/trace.Link\]$1`
724. `file\_opentelemetry\_proto\_trace\_v1\_trace\_proto\_rawDescGZIP$1`
725. `instantiatedType$1`
726. `init\#1`
727. `removePerishedConns$1`
728. `main`
729. `maybeRunStateHook$1`
730. `newNonRetryClientStream$2`
731. `gcTestMoveStackOnNextCall`
732. `AddSet$1`
733. `os\_sigpipe`
734. `Run$1`
735. `unlockProfiles`
736. `TestQueryDatabase\_DataSourceFactoryAndLifecycleErrors$1$1`
737. `ExportChanges$1`
738. `runCmdContext$3$1`
739. `main`
740. `init\#6`
741. `printDebugLog`
742. `InitializeTelemetry$2`
743. `writeEarlyAbort$2`
744. `connect$1`
745. `doinit`
746. `init\#1`
747. `fixedHuffmanDecoderInit$1`
748. `main`
749. `OnPut$1`
750. `xRegRestore$1`
751. `startGracefulShutdown$bound`
752. `TestRabbitMQAdapter\_ProducerDefault\_ReturnsError\_WhenEnsureChannelFails$1`
753. `Unreachable`
754. `goroutineLeakProfileWithLabelsConcurrent$1$1`
755. `Apply$1`
756. `updateTargetResolverState$1`
757. `AddSet$1`
758. `secret\_inc`
759. `libc\_setgroups\_trampoline`
760. `sendResponse$1`
761. `Stop$1`
762. `NewHTTP2Client$1`
763. `poolCleanup`
764. `TestRabbitMQAdapter\_RunConsumerCycle\_FailsOnConsumeError$1`
765. `GetOrBuildProducer$1`
766. `libc\_fchown\_trampoline`
767. `archInit`
768. `closeReqBodyLocked$1`
769. `Pull$3`
770. `init$2`
771. `parseFiles$2$1`
772. `libc\_exit\_trampoline`
773. `TestRabbitMQAdapter\_ProcessDelivery\_NackError$3`
774. `parseUnion$1`
775. `file\_google\_protobuf\_wrappers\_proto\_rawDescGZIP$1`
776. `lazyInit$1`
777. `runtime\_debug\_freeOSMemory$1`
778. `StatelessDeflate$2`
779. `init\#1`
780. `readAll$1`
781. `Force`
782. `onWriteTimeout$bound`
783. `init\#1`
784. `file\_opentelemetry\_proto\_logs\_v1\_logs\_proto\_rawDescGZIP$1`
785. `reflectOffsLock`
786. `TestProcessPluginCRMCollection\_WithValidOrganization$1`
787. `queryDatabase$1`
788. `allocm$1`
789. `secure`
790. `CreateContainer$1`
791. `ResetWithOptions$1`
792. `runExample$1`
793. `libc\_issetugid\_trampoline`
794. `init\#1`
795. `init\#1`
796. `Do$1`
797. `ResetServiceIndicator`
798. `main`
799. `main`
800. `AddSet$1`
801. `Close$1`
802. `init\#1`
803. `libc\_readdir\_r\_trampoline`
804. `init\#6`
805. `walk$1$1`
806. `libc\_sendmsg\_trampoline`
807. `basepointNafTable$1`
808. `initResourceValue\[\*github.com/jackc/pgx/v5/pgxpool.connResource\]$1`
809. `invokeError$1`
810. `CheckPath$1`
811. `send$1`
812. `scavenge$1`
813. `x509\_CFArrayCreateMutable\_trampoline`
814. `init\#17`
815. `Stop$1`
816. `mayMoreStackPreempt`
817. `SetMx$1`
818. `startupValidation$1`
819. `goready$1`
820. `connect$2`
821. `onceSetNextProtoDefaults\_Serve$bound`
822. `init\#1`
823. `addOption$1`
824. `corostart`
825. `ResetPanicMetrics`
826. `Acquire$1`
827. `libc\_sysctl\_trampoline`
828. `startStreamDecoder$1`
829. `onReadTimeout$bound`
830. `Flush$bound`
831. `search$2$1$1`
832. `Clearenv`
833. `main`
834. `pageTmpl$1`
835. `processTxPipelineNode$1$1`
836. `TestScanColumns$2$1`
837. `libc\_stat\_trampoline`
838. `breakpoint`
839. `NextResultSet$1`
840. `init\#1`
841. `pthread\_self\_trampoline`
842. `main`
843. `unsetBypass`
844. `StatsPrint`
845. `malg$1`
846. `libc\_getrlimit\_trampoline`
847. `syscall\_runtime\_BeforeExec`
848. `init$1`
849. `traceNotificationResponse$1`
850. `TestValidateParameters$1`
851. `TestQueryPluginCRM\_WithOrganizationOnly$1`
852. `getcp874$1`
853. `Commit$1`
854. `x509\_CFArrayGetCount\_trampoline`
855. `Record$1`
856. `Read$1`
857. `init\#1`
858. `createIdleResources$1`
859. `init\#2`
860. `ServeHTTP$1$1`
861. `updateMaxProcsGoroutine`
862. `Add$1`
863. `needsInitCheckLocked$1`
864. `panicunsafeslicenilptr`
865. `x509\_CFArrayGetValueAtIndex\_trampoline`
866. `ServeFile$1`
867. `main$1`
868. `scan$3`
869. `redirectStdLogAt$1`
870. `mapv$1`
871. `saveFile$1`
872. `mountStartupProcess$2`
873. `parsePattern$1`
874. `OrderedMarshalJSON$1`
875. `init\#1`
876. `systemstack\_switch`
877. `UpdateClientConnState$1`
878. `init\#4`
879. `RegisterTraceServiceHandlerFromEndpoint$1`
880. `init\#1`
881. `writeContext$1`
882. `addrLookupOrder$1`
883. `save\_g`
884. `NewPrivateKey$1`
885. `runtime\_AfterForkInChild`
886. `init\#2`
887. `exit\_trampoline`
888. `init\#1`
889. `InitLocalEnvConfig$1`
890. `Close$1`
891. `doRetryNotify\[\*github.com/testcontainers/testcontainers-go.Reaper\]$1`
892. `cancel$1`
893. `init\#4`
894. `compute$1`
895. `TestRedisCache\_NewRedisCache\_NilConnection\_Panics$2$1`
896. `embeddedIfaceMethStub`
897. `setRequestCancel$3`
898. `init\#2`
899. `x509\_SecCertificateCopyData\_trampoline`
900. `SetFinalizer$1`
901. `appendJSONMarshal$1`
902. `dumpScanStats`
903. `buildCommonHeaderMaps`
904. `TestConsumerRoutes\_Shutdown$2`
905. `StmtContext$2`
906. `chanrecv$1`
907. `run$1`
908. `TestScanColumns$1$1`
909. `initCommonHeader`
910. `init\#3`
911. `CloseNotify$1`
912. `processHandoffRequest$1`
913. `makeTempDir$1`
914. `PCall$1$1`
915. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P521Point\]\]$1$1$1`
916. `libc\_getfsstat\_trampoline`
917. `init\#1`
918. `writeHeader$1`
919. `prefork$2`
920. `ResetTestLicenseBaseURL`
921. `pingDC$1`
922. `parsePrimaryExpr$1`
923. `OnceFunc$1$1$1`
924. `file\_grpc\_health\_v1\_health\_proto\_rawDescGZIP$1`
925. `runtime\_goroutineLeakGC`
926. `listPackages$1`
927. `onReadTimeout$bound`
928. `OnceValue\[encoding/json.encoderFunc\]$1$1$1`
929. `file\_opentelemetry\_proto\_collector\_trace\_v1\_trace\_service\_proto\_init`
930. `traceCopyFail$1`
931. `sysmonUpdateGOMAXPROCS`
932. `synctestRun$2`
933. `main`
934. `prepareNext$1`
935. `snapshotConnection$1`
936. `libc\_writev\_trampoline`
937. `\_VDSO`
938. `RecordSet$1`
939. `prepareDC$1`
940. `scan$4`
941. `gorecover$1`
942. `Write$1`
943. `ScanAndListen$2`
944. `Run$1`
945. `Shutdown$1`
946. `parse$1`
947. `processOptions`
948. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P384Point\]\]$1$1$1`
949. `entersyscall`
950. `TestConsumerRoutes\_Shutdown$1`
951. `next$1`
952. `newTdsBuffer$1`
953. `stubJobSpanAttributes$2`
954. `readRequest$1`
955. `logDropped\[go.opentelemetry.io/otel/sdk/trace.Event\]$1`
956. `Stack$1`
957. `libc\_getppid\_trampoline`
958. `traceAdvance$1$1`
959. `init\#1`
960. `startTemplateThread`
961. `init\#1`
962. `libc\_unmount\_trampoline`
963. `gfget$1`
964. `LazyReload$1`
965. `TestProcessPluginCRMCollection\_WithOrganizationID$1`
966. `getCh$1`
967. `Cleanup$1`
968. `containsElement$1`
969. `libc\_ftruncate\_trampoline`
970. `UnregisterSpanProcessor$1`
971. `validateNorm$1`
972. `stop$1`
973. `dispatch0$1`
974. `unpinConnectionFromCursor$bound`
975. `Serve$2`
976. `TestValidateParametersWithCursor$1`
977. `setPinned$1`
978. `init\#1`
979. `markrootFreeGStacks`
980. `queryDC$1`
981. `init\#1`
982. `reentersyscall$2`
983. `traceDescribe$1`
984. `gcBgMarkStartWorkers`
985. `nextMarkBitArenaEpoch`
986. `pollSRVRecords$1`
987. `onReadIdleTimer$bound`
988. `traceInitReadCPU`
989. `libc\_socket\_trampoline`
990. `libc\_error\_trampoline`
991. `gcBeginWork`
992. `minimize$1`
993. `sendOpen$1`
994. `init\#7`
995. `TestMultiQueueConsumerRun$1`
996. `lockVerifyMSize`
997. `InitLocalEnvConfig$1`
998. `libc\_renameat\_trampoline`
999. `runCleanup$2`
1000. `AddSet$1`
1001. `handleUpgradeResponse$1`
1002. `DecodeAll$1`
1003. `readForm$1`
1004. `CopyFileWithTar$1`
1005. `beginDC$1`
1006. `procUnpin`
1007. `closeRead$1`
1008. `initConfVal`
1009. `maintain$3`
1010. `close$1`
1011. `doRetryNotify$1`
1012. `structType$3`
1013. `Remove$1`
1014. `UploadLogs$1`
1015. `RegisterLogsServiceHandlerFromEndpoint$1$1`
1016. `read$1`
1017. `TestRabbitMQAdapter\_ProducerDefault\_Success$1$1`
1018. `duffcopy`
1019. `write\_trampoline`
1020. `CreateNetwork$1`
1021. `lostProfileEvent`
1022. `libc\_recvmsg\_trampoline`
1023. `NewHTTP2Client$6`
1024. `handleIdleTimeout$bound`
1025. `doInRoot\[os.FileInfo\]$1`
1026. `init\#3`
1027. `RunConsumers$1`
1028. `archInitCastagnoli`
1029. `pinConnectionToTransaction$bound`
1030. `execDC$2`
1031. `invalidateChannel$1`
1032. `buildCommonHeaderMapsOnce`
1033. `init\#16`
1034. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnHandlerError$1`
1035. `ParseAcceptLanguage$1`
1036. `libc\_chown\_trampoline`
1037. `Subscribe$2`
1038. `fieldByIndexErr$1`
1039. `doInRoot\[struct{}\]$1`
1040. `shutdownWorkers$2`
1041. `buildOnce$bound`
1042. `updateTargetResolverState$2`
1043. `prefork$1`
1044. `minitSignalMask`
1045. `getcp1254$1`
1046. `init\#3`
1047. `init\#1`
1048. `Serve$1`
1049. `morestack`
1050. `newServer$1`
1051. `ParseScript$1`
1052. `readPreface$1`
1053. `contactResponders$1$1`
1054. `StopCPUProfile`
1055. `ParseExprFrom$1`
1056. `AddSet$1`
1057. `ParseVariant$1`
1058. `goenvs\_unix`
1059. `SendBatch$1`
1060. `stoplockedm`
1061. `onWriteTimeout$bound`
1062. `write$1`
1063. `parseMultiplexedLogs$1`
1064. `minimize$2`
1065. `NewFastHTTPHandler$1$1`
1066. `lazyRegexCompile$1$1`
1067. `Shutdown$1$1$1`
1068. `ExitIdle$bound`
1069. `handleMoving$1`
1070. `ParseRegion$1`
1071. `main`
1072. `processStreamingRPC$1`
1073. `dispatchClosed$1`
1074. `SendMsg$4`
1075. `doCall$1`
1076. `init\#1`
1077. `fatal$1`
1078. `commandLineUsage`
1079. `AddSet$1`
1080. `Shutdown$1`
1081. `emptyfunc`
1082. `Less$1`
1083. `http2registerHTTPSProtocol$1`
1084. `init\#1`
1085. `beginFuncExec$1`
1086. `encodeError$1`
1087. `libc\_dup\_trampoline`
1088. `stderrHandler$1`
1089. `StatelessDeflate$1`
1090. `NotifyConfirm$1`
1091. `connect$1`
1092. `gcWakeAllStrongFromWeak`
1093. `Start$2`
1094. `init\#1`
1095. `getcp1255$1`
1096. `startLocked$1`
1097. `x509\_CFDataGetLength\_trampoline`
1098. `dit\_setDisabled`
1099. `checkOut$1`
1100. `UploadTraces$1`
1101. `RecordSet$1`
1102. `TestMultiQueueConsumerRun$2$1`
1103. `TestWithSwaggerEnvConfig\_DefaultValues$1`
1104. `callers$1`
1105. `file\_opentelemetry\_proto\_common\_v1\_common\_proto\_rawDescGZIP$1`
1106. `selectgo$2`
1107. `buildOnce$bound`
1108. `traceRowDescription$1`
1109. `ClearObjectValidations$1`
1110. `after$1`
1111. `failWantMap`
1112. `dial$1`
1113. `TestHandlerGenerateReport\_DelegatesToUseCase$1`
1114. `checkdead`
1115. `lazyInit$1`
1116. `aberrantDeriveMessageName$1`
1117. `libc\_grantpt\_trampoline`
1118. `main`
1119. `Scan$1`
1120. `gcWriteBarrier3`
1121. `SaveImagesWithOpts$2`
1122. `Shutdown$1`
1123. `exportSync$1`
1124. `startHealthCheck$1`
1125. `equalServiceConfig$1`
1126. `traceCPUFlush$1`
1127. `RecordSet$1`
1128. `SetMeterProvider$1`
1129. `randinit`
1130. `Run$1`
1131. `gcstopm`
1132. `gcStart$3`
1133. `getcp850$1`
1134. `libc\_shutdown\_trampoline`
1135. `SelectServer$1`
1136. `startLoad$1`
1137. `StartTimeStampUpdater$1`
1138. `pipe\_trampoline`
1139. `gcBgMarkWorker$2`
1140. `init\#1`
1141. `sysctl\_trampoline`
1142. `secret\_dec`
1143. `init\#4$4`
1144. `init\#1`
1145. `Record$1`
1146. `init\#1`
1147. `AddSet$1`
1148. `updateClientConnState$2`
1149. `init\#1`
1150. `buildCommonHeaderMaps`
1151. `buildShutdownHandlers$1`
1152. `main`
1153. `lazyInit$1`
1154. `PrintStack`
1155. `panicunsafeslicelen`
1156. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnHandlerError$3`
1157. `tryDial$1`
1158. `Start$1`
1159. `write$1`
1160. `init\#2`
1161. `libc\_fchmod\_trampoline`
1162. `SetErrorHandler$1`
1163. `ReuseOrCreateContainer$1`
1164. `init\#2`
1165. `x509\_CFRelease\_trampoline`
1166. `propagateCancel$1`
1167. `sync\_runtime\_procUnpin`
1168. `reflectOffsUnlock`
1169. `mstart\_stub`
1170. `libc\_statfs\_trampoline`
1171. `ParseExtension$1`
1172. `buildRootHuffmanNode`
1173. `TestRateLimiter\_ConcurrentAccess$1`
1174. `lazyInit$1`
1175. `libc\_getcwd\_trampoline`
1176. `osinit\_hack\_trampoline`
1177. `addMultiCallback$1`
1178. `keepalive$1`
1179. `onShutdownTimer$bound`
1180. `main`
1181. `aberrantDeriveMessageName$1$1`
1182. `preprintpanics$1`
1183. `HandleStreams$2`
1184. `parseBinaryExpr$1`
1185. `servePeer$1`
1186. `init\#1`
1187. `defPredeclaredNil`
1188. `RunParallel$1`
1189. `encodeStringer$1`
1190. `init\#1`
1191. `FixedZone$1`
1192. `UnlockOSThread`
1193. `restoreSIGSYS`
1194. `setDefaultUnameProvider`
1195. `runExample$2`
1196. `init\#1`
1197. `RecvMsg$1`
1198. `libc\_fchownat\_trampoline`
1199. `panicunsafestringnilptr`
1200. `ReadMemStats$1`
1201. `copyTrailersToHandlerRequest$bound`
1202. `init\#1`
1203. `libc\_getuid\_trampoline`
1204. `libc\_futimes\_trampoline`
1205. `init\#7`
1206. `handshakeContext$1`
1207. `printDebugLogImpl`
1208. `TestMust$2$1`
1209. `Shutdown$1`
1210. `parseCpuList`
1211. `flush`
1212. `init\#1`
1213. `initMsgChan$1`
1214. `Connection$1`
1215. `SetDelegate$1`
1216. `runtime\_procUnpin`
1217. `mutateBytes$1`
1218. `libc\_chflags\_trampoline`
1219. `traceReadyForQuery$1`
1220. `getcp1257$1`
1221. `Getsockopt$1`
1222. `oneNewExtraM`
1223. `file\_google\_protobuf\_field\_mask\_proto\_rawDescGZIP$1`
1224. `serve$1`
1225. `issetugid\_trampoline`
1226. `panicmakeslicecap`
1227. `sigtramp`
1228. `libc\_close\_trampoline`
1229. `main`
1230. `GetValidator$1`
1231. `closeWrite$1`
1232. `SetDefaultGOMAXPROCS`
1233. `init\#4`
1234. `Add$1`
1235. `init$1`
1236. `main`
1237. `fuzz$2`
1238. `typeDecl$1`
1239. `init\#1`
1240. `serveStreams$2$1`
1241. `PluginInstall$1$1`
1242. `printlock`
1243. `init\#1`
1244. `file\_opentelemetry\_proto\_collector\_logs\_v1\_logs\_service\_proto\_rawDescGZIP$1`
1245. `readHosts`
1246. `RecordSet$1`
1247. `search$2`
1248. `file\_google\_protobuf\_timestamp\_proto\_init`
1249. `structv$1`
1250. `file\_opentelemetry\_proto\_resource\_v1\_resource\_proto\_init`
1251. `init\#1`
1252. `runFinalizers`
1253. `init\#11`
1254. `libc\_fdopendir\_trampoline`
1255. `Add$1`
1256. `AfterFunc$1$1`
1257. `appendValue$1`
1258. `exitsyscall$1`
1259. `defPredeclaredFuncs`
1260. `flushLine$1`
1261. `safeCall$1`
1262. `releaseThread`
1263. `libc\_msync\_trampoline`
1264. `main`
1265. `Never$1`
1266. `init\#2`
1267. `main`
1268. `destroy$1`
1269. `prepareFreeWorkbufs`
1270. `walk$1`
1271. `exitsyscall$4`
1272. `readFrames$1`
1273. `Run$1`
1274. `ReplaceFileTarWrapper$1`
1275. `\_LostContendedRuntimeLock`
1276. `panicSimdImm`
1277. `doRetryNotify\[struct{}\]$1`
1278. `refreshServerDate`
1279. `createContext$7`
1280. `init\#1`
1281. `checkfds`
1282. `ResetWithOptions$1`
1283. `OnPut$1$1`
1284. `mspinning`
1285. `alloc$1`
1286. `checkGenericIsExpected`
1287. `racegoend`
1288. `runCmdContext$2`
1289. `libc\_ptrace\_trampoline`
1290. `init\#1`
1291. `init\#1`
1292. `Write$1`
1293. `init\#1`
1294. `initCategoryAliases`
1295. `createfing`
1296. `lazyInit$1`
1297. `isZeroValue$1`
1298. `file\_opentelemetry\_proto\_resource\_v1\_resource\_proto\_rawDescGZIP$1`
1299. `pthread\_cond\_timedwait\_relative\_np\_trampoline`
1300. `doInRoot$1`
1301. `main`
1302. `gcWriteBarrier1`
1303. `connect$bound`
1304. `yaml\_parser\_fetch\_next\_token$1`
1305. `doRecordGoroutineProfile$1`
1306. `initDefaultMap`
1307. `init$13`
1308. `executeMultiSlot$2`
1309. `processHandoffRequest$2`
1310. `raiseproc\_trampoline`
1311. `ProjectRoot$1`
1312. `Prepare$1`
1313. `RegisterMetricsServiceHandlerFromEndpoint$1$1`
1314. `TestRabbitMQAdapter\_ConsumerLoop\_NormalizesConcurrency$1`
1315. `Shutdown$1`
1316. `readGCStats$1`
1317. `sweepone$1`
1318. `libc\_wait4\_trampoline`
1319. `build$bound`
1320. `init\#1`
1321. `Close$bound`
1322. `init\#8`
1323. `libc\_linkat\_trampoline`
1324. `handleRSTStream$1`
1325. `runtime\_BeforeExec`
1326. `defaultGOMAXPROCSUpdateGODEBUG`
1327. `DialContext$1`
1328. `unmarshalFull$1$1`
1329. `commitAttemptLocked$bound`
1330. `parseParameterList$1`
1331. `Do$1`
1332. `finish$bound`
1333. `pthread\_create\_trampoline`
1334. `initAlgAES`
1335. `debugCallWrap$1`
1336. `resolve$1`
1337. `Disable`
1338. `build$bound`
1339. `finishsweep\_m`
1340. `probe$bound`
1341. `fips140\_setBypass`
1342. `Resolve$1`
1343. `buildRootHuffmanNode`
1344. `getConn$1`
1345. `panicfloat`
1346. `init\#1`
1347. `maybeRunAsync$1`
1348. `libc\_mlock\_trampoline`
1349. `addOption$1`
1350. `validVarType$1`
1351. `main`
1352. `file\_google\_protobuf\_duration\_proto\_rawDescGZIP$1`
1353. `init\#1`
1354. `CloseSend$2`
1355. `init\#1`
1356. `gcMarkTermination$2`
1357. `init\#3`
1358. `main`
1359. `InitMongoDBExternal$1`
1360. `CopyFileWithTar$2`
1361. `lockOSThread`
1362. `racefini`
1363. `Parse$1`
1364. `refill$1`
1365. `wirep$1`
1366. `pthread\_attr\_init\_trampoline`
1367. `libc\_link\_trampoline`
1368. `checkInNoEvent$1`
1369. `init\#1`
1370. `processSingleResponse$1`
1371. `Record$1`
1372. `OnceValue\[string\]$1$1`
1373. `fRunner$2`
1374. `readDataFrame$1`
1375. `libc\_closedir\_trampoline`
1376. `connect$1`
1377. `init\#1`
1378. `ensureSigM`
1379. `threadRun$1`
1380. `printsp`
1381. `collectFileInfoForChanges$2`
1382. `registerBasics`
1383. `initServerWorkers$1`
1384. `processDelivery$1`
1385. `traceAdvance$5`
1386. `TestQueryPluginCRMCollectionWithFilters\_NoFilters$1`
1387. `init\#1`
1388. `init\#1`
1389. `connect$2`
1390. `defPredeclaredConsts`
1391. `startAlarm$1`
1392. `p384B$1`
1393. `CleanupContainer$1`
1394. `file\_grpc\_binlog\_v1\_binarylog\_proto\_init`
1395. `TestRabbitMQAdapter\_Shutdown\_ContextCanceledDuringWait$1`
1396. `Stop$1`
1397. `init\#2`
1398. `init\#1`
1399. `buildRecompMap`
1400. `init\#1`
1401. `init\#1`
1402. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P256Point\]\]$1$1$1`
1403. `init\#1`
1404. `goroutineProfileWithLabelsSync$4$1`
1405. `Shutdown$1$1`
1406. `initOptions`
1407. `synctestRun$1`
1408. `init\#1`
1409. `signature$1`
1410. `libc\_bind\_trampoline`
1411. `panicoverflow`
1412. `Finish$bound`
1413. `needAndBindM`
1414. `init\#1`
1415. `ParallelContainers$1`
1416. `updateProxyResolverState$1`
1417. `cancel$1`
1418. `SkipIfProviderIsNotHealthy$1`
1419. `dialCtx$1`
1420. `init\#1`
1421. `goschedguarded`
1422. `finish$bound`
1423. `disableInfinityTS`
1424. `fips140\_unsetBypass`
1425. `emit$1`
1426. `init\#1`
1427. `newPortForwarder$1`
1428. `newClientStream$1`
1429. `ExitIdle$1`
1430. `init\#1`
1431. `init\#1`
1432. `Read$1`
1433. `init\#2`
1434. `Peek$1`
1435. `init\#1`
1436. `init\#1`
1437. `freeSomeWbufs$1`
1438. `TestRabbitMQAdapter\_IsHealthy\_ReturnsFalse\_WhenChannelClosed$1`
1439. `getcp1258$1`
1440. `main`
1441. `find$1`
1442. `Add$1`
1443. `sysmon`
1444. `doBlockingWithCtx\[\[\]vendor/golang.org/x/net/dns/dnsmessage.Resource\]$1`
1445. `syscall\_runtime\_AfterExec`
1446. `x509\_SecTrustEvaluateWithError\_trampoline`
1447. `clientHandshake$1`
1448. `traceBind$1`
1449. `Parse$1`
1450. `Serve$3`
1451. `libc\_setregid\_trampoline`
1452. `Close$1`
1453. `read$1`
1454. `gcMarkTermination$3`
1455. `RecordSet$1`
1456. `Shutdown$1`
1457. `crash`
1458. `asyncIsExpired$1`
1459. `secureEnv`
1460. `init$1`
1461. `copyTrailers$bound`
1462. `StmtContext$1`
1463. `pthread\_setspecific\_trampoline`
1464. `keepalive$1`
1465. `minimize$1`
1466. `runtime\_BeforeFork`
1467. `FindLongestMatch$1`
1468. `TestExternalDataSource\_CloseConnection\_NilConnection$1$1`
1469. `x509\_CFDataGetBytePtr\_trampoline`
1470. `syscallN\_trampoline`
1471. `term$1`
1472. `Pull$1$2`
1473. `init\#1`
1474. `TestRabbitMQAdapter\_ProcessDelivery\_NackError$1`
1475. `libc\_chdir\_trampoline`
1476. `DisableRandPool`
1477. `onWriteTimeout$bound`
1478. `Has$1`
1479. `collectExemplars$1`
1480. `Current$1`
1481. `init\#2`
1482. `traceCommandComplete$1`
1483. `Parse$1`
1484. `libresolv\_res\_9\_nclose\_trampoline`
1485. `OnceValue\[map\[string\]reflect.Value\]$1$1$1`
1486. `libc\_lseek\_trampoline`
1487. `main`
1488. `libc\_ptsname\_r\_trampoline`
1489. `init\#2`
1490. `merge$1`
1491. `init\#1`
1492. `unifyShutdown$1$1`
1493. `init\#5`
1494. `newcoro$1`
1495. `init\#1`
1496. `RecvMsg$1`
1497. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnVersionMismatch$3`
1498. `main`
1499. `objDecl$2`
1500. `ensureSigM$1`
1501. `file\_opentelemetry\_proto\_logs\_v1\_logs\_proto\_init`
1502. `queuedNewConn$2$1`
1503. `TestRabbitMQAdapter\_ProducerDefault\_ReturnsError\_WhenCircuitOpen$1`
1504. `badmorestackgsignal`
1505. `New\[\*crypto/internal/fips140/sha512.Digest\]$1$1`
1506. `libc\_truncate\_trampoline`
1507. `entersyscallblock$2`
1508. `doBlockingWithCtx\[\[\]string\]$1`
1509. `EvaluateConstValue$1`
1510. `Consume$1`
1511. `initOnce$1`
1512. `watchCancel$2`
1513. `ParseFile$1`
1514. `interfaceEqual$1`
1515. `gcMarkTermination$4$1`
1516. `libc\_posix\_openpt\_trampoline`
1517. `init\#1`
1518. `badunlockosthread`
1519. `closeStream$1`
1520. `newClientStreamWithParams$1`
1521. `queryDC$2`
1522. `pthread\_attr\_getstacksize\_trampoline`
1523. `init\#1`
1524. `main`
1525. `distTmpl$1`
1526. `roundTrip$1`
1527. `Check`
1528. `init\#19`
1529. `IncNonDefault$bound`
1530. `RecordMetrics$1`
1531. `Shutdown$1`
1532. `main`
1533. `initAll`
1534. `goyield`
1535. `exitsyscall`
1536. `libc\_rmdir\_trampoline`
1537. `doHTTPConnectHandshake$1`
1538. `mstart`
1539. `init\#1`
1540. `init\#1`
1541. `atom$1`
1542. `init\#2`
1543. `BoringCrypto`
1544. `endCheckmarks`
1545. `RecordSet$1`
1546. `initSecureMode`
1547. `init$1`
1548. `file\_google\_protobuf\_struct\_proto\_init`
1549. `publicationBarrier`
1550. `badmorestackg0`
1551. `operateHeaders$5`
1552. `mayMoreStackMove`
1553. `init\#6`
1554. `ResetMaxTraceEntryToDefault`
1555. `append$1`
1556. `executeParallel$2`
1557. `WithRecover$1$1`
1558. `init\#1`
1559. `StopTrace`
1560. `runWithClient$1`
1561. `TestWithSwaggerEnvConfig\_EmptyEnvVars$1`
1562. `getcp949$1`
1563. `Ping$1`
1564. `onReadIdleTimer$bound`
1565. `finalClose$2`
1566. `getcp1256$1`
1567. `badctxt`
1568. `badsystemstack`
1569. `debugOptions`
1570. `libc\_chroot\_trampoline`
1571. `bootstrapRandReseed`
1572. `init\#4`
1573. `NewFunc$1`
1574. `DefaultEncoder$1`
1575. `traceAdvance$4`
1576. `init\#1`
1577. `TimeoutWithCodeHandler$1$1`
1578. `libc\_openat\_trampoline`
1579. `badreflectcall`
1580. `OnceFunc$1$1`
1581. `synctestWait`
1582. `New\[\*crypto/internal/fips140/sha256.Digest\]$1`
1583. `init\#1`
1584. `goexit1`
1585. `AddSet$1`
1586. `TestRabbitMQAdapter\_ProcessDelivery\_RecoverFromPanic$1`
1587. `stacklessWriteDeflate$1`
1588. `loop`
1589. `Record$1`
1590. `structv$1`
1591. `nextFrame$1`
1592. `Add$1`
1593. `set$1`
1594. `End$bound`
1595. `genericExprList$1`
1596. `Value$1`
1597. `operateHeaders$3`
1598. `OnceValue\[encoding/json.encoderFunc\]$1$1`
1599. `traceThreadDestroy$1`
1600. `casgstatus$2`
1601. `libc\_unlink\_trampoline`
1602. `init\#18`
1603. `libc\_fchmodat\_trampoline`
1604. `Run$1`
1605. `WriteOverlays$2`
1606. `sigpipe`
1607. `shutdown$1`
1608. `TestScanRows$3$1`
1609. `libresolv\_res\_9\_nsearch\_trampoline`
1610. `RoundTrip$1`
1611. `libc\_sysconf\_trampoline`
1612. `gcStart$4`
1613. `processSRVResults$1`
1614. `runConcurrent$1`
1615. `printCountProfile$2`
1616. `gcWriteBarrier7`
1617. `startBackgroundRead$bound`
1618. `readContent$1`
1619. `file\_google\_rpc\_status\_proto\_rawDescGZIP$1`
1620. `invokeStringer$1`
1621. `update$1`
1622. `sigprocmask\_trampoline`
1623. `Release$1`
1624. `pthread\_cond\_init\_trampoline`
1625. `init$bound`
1626. `Token$1`
1627. `main`
1628. `wbBufFlush`
1629. `init\#1`
1630. `file\_google\_protobuf\_wrappers\_proto\_init`
1631. `Flush$1`
1632. `main`
1633. `PrintDefaults`
1634. `main`
1635. `osyield`
1636. `Close$1`
1637. `DumpRequestOut$2`
1638. `Compose$1`
1639. `init\#1`
1640. `FiberWrapHandler$1$1`
1641. `run1$1$1`
1642. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnInvalidSignature$3`
1643. `flush$1`
1644. `onSettingsTimer$bound`
1645. `buildRecompMap`
1646. `initialize$bound`
1647. `NewFastHTTPHandler$1$1$1`
1648. `Close$1`
1649. `init\#1`
1650. `SignalNum$1`
1651. `ClearArrayValidations$1`
1652. `libc\_setsid\_trampoline`
1653. `queuedNewConn$1`
1654. `connect$1`
1655. `Close$1`
1656. `initPredefined$1`
1657. `entersyscallblock$4`
1658. `dispatchDeliveries$1$1`
1659. `fixedHuffmanDecoderInit`
1660. `init\#1`
1661. `queuedNewConn$2$2`
1662. `libc\_listen\_trampoline`
1663. `Add$1`
1664. `Find$1`
1665. `runCmdContext$3`
1666. `finalClose$1`
1667. `fprint$1`
1668. `defPredeclaredTypes`
1669. `trace$1`
1670. `init\#1`
1671. `libc\_connect\_trampoline`
1672. `checkTimeouts`
1673. `synctestRun$3`
1674. `pthread\_mutex\_init\_trampoline`
1675. `pinConnectionToCursor$bound`
1676. `ForEachPackage$1`
1677. `decap$2`
1678. `doCall$2$1`
1679. `checkMinIdleConns$1$1`
1680. `ConsumeWithContext$1`
1681. `getDockerAuthConfigs$3`
1682. `init\#1`
1683. `vgetrandomInit`
1684. `onceSetNextProtoDefaults$bound`
1685. `New\[hash.Hash\]$1`
1686. `runtime\_debug\_WriteHeapDump$1`
1687. `init\#1`
1688. `init$6`
1689. `panicunsafestringlen`
1690. `MarshalAppendWithContext$1`
1691. `libc\_readlinkat\_trampoline`
1692. `file\_google\_protobuf\_any\_proto\_rawDescGZIP$1`
1693. `libc\_settimeofday\_trampoline`
1694. `init\#1`
1695. `startGracefulShutdown$bound`
1696. `reader$1`
1697. `isAbstractSocketExists$1`
1698. `Run$1`
1699. `legacyContainerWait$1`
1700. `rollback$1`
1701. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnMissingSignature$3`
1702. `setDITDisabled`
1703. `init\#5`
1704. `ShouldPanic$1`
1705. `file\_google\_protobuf\_any\_proto\_init`
1706. `syscall\_runtime\_AfterFork`
1707. `panicExtend`
1708. `interfaceType$2`
1709. `init$1$1`
1710. `setGCPercent$1`
1711. `libc\_dup2\_trampoline`
1712. `getcp1253$1`
1713. `libc\_select\_trampoline`
1714. `getDefaultMetricsFactory$1`
1715. `sendBatchExtendedWithDescription$2$1`
1716. `hasVarSize$1`
1717. `stdin$1`
1718. `freeDeadSpanSPMCs`
1719. `decap$1`
1720. `init\#5`
1721. `init\#1`
1722. `RegisterTraceServiceHandlerFromEndpoint$1$1`
1723. `Parse$1`
1724. `GC`
1725. `resolveUnderlying$1`
1726. `collectMethods$1`
1727. `entersyscallblock$5`
1728. `init$1`
1729. `main`
1730. `nextBlock$1$2$1`
1731. `x509\_SecTrustSetVerifyDate\_trampoline`
1732. `newReaper$2`
1733. `close$bound`
1734. `printnl`
1735. `checkGenericIsExpected`
1736. `writeBodyStream$1`
1737. `entersyscallblock$1`
1738. `HandleStreams$2`
1739. `EnableRandPool`
1740. `Record$1`
1741. `libc\_getaddrinfo\_trampoline`
1742. `lazyInit$1`
1743. `commitAttemptLocked$bound`
1744. `readLoop$1`
1745. `panicshift`
1746. `MustExtractDockerSocket$1`
1747. `update$2`
1748. `NewPeriodicReader$2$1`
1749. `AddSet$1`
1750. `startStreamDecoder$2`
1751. `fcntl\_trampoline`
1752. `lazyInit$1`
1753. `run$1`
1754. `walltime\_trampoline`
1755. `panicmem`
1756. `Add$1`
1757. `ResetAssertionMetrics`
1758. `init\#1`
1759. `execDC$1`
1760. `TestScanRows$1$1`
1761. `closeReqBodyLocked$1`
1762. `newSession$1`
1763. `metricsUnlock`
1764. `init\#1`
1765. `initRequestHandler$bound`
1766. `moduledataverify`
1767. `init\#2`
1768. `watchMaster`
1769. `HandleStreams$1`
1770. `osinit`
1771. `gcStart$2`
1772. `processSubAppsRoutes$1`
1773. `RegisterLogsServiceHandlerFromEndpoint$1`
1774. `unreachableMethod`
1775. `writeStatus$1`
1776. `osInit`
1777. `syscall\_x509`
1778. `Add$1`
1779. `initMimeUnix`
1780. `start$1`
1781. `init\#1`
1782. `init\#2`
1783. `mProf\_PostSweep`
1784. `dialCtx$2`
1785. `dropm`
1786. `gcWriteBarrier4`
1787. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P521Point\]\]$1$1`
1788. `check`
1789. `init\#1`
1790. `AddSet$1`
1791. `unregisterAllDrivers`
1792. `wbBufFlush$1`
1793. `lazyInit$1`
1794. `init$13`
1795. `init\#2`
1796. `TestRabbitMQAdapter\_Shutdown\_WaitsForConsumers$1`
1797. `aggregate$1`
1798. `TestRabbitMQAdapter\_ProcessDelivery\_HandlesNonStringHeaderID$3`
1799. `init\#3`
1800. `run$1`
1801. `dumpparams`
1802. `metricsLock`
1803. `getcp1250$1`
1804. `setPossiblyUnhashableKey$1`
1805. `attemptArgMatch$1`
1806. `OnceValue\[\[\]string\]$1$1$1`
1807. `libc\_arc4random\_buf\_trampoline`
1808. `stkobjinit`
1809. `runHandler$1`
1810. `init\#1`
1811. `TestRabbitMQAdapter\_ConsumerLoop\_NacksOnMissingSignature$1`
1812. `init\#5`
1813. `runHandlers`
1814. `init\#1`
1815. `reentersyscall$3`
1816. `main`
1817. `init\#1`
1818. `operateHeaders$4`
1819. `onceSetNextProtoDefaults$bound`
1820. `wirep$2`
1821. `gcPrepareMarkRoots`
1822. `Record$1`
1823. `stackinit`
1824. `init\#1`
1825. `newClientStreamWithParams$4`
1826. `file\_opentelemetry\_proto\_metrics\_v1\_metrics\_proto\_init`
1827. `fuzz$1`
1828. `processUnaryRPC$1`
1829. `hostLookupOrder$1`
1830. `initConnPool$bound`
1831. `dumpobjs`
1832. `cleanup$1`
1833. `init\#1`
1834. `goroutineLeakGC`
1835. `nop`
1836. `StartWithGracefulShutdown$1`
1837. `parse$1`
1838. `templateThread`
1839. `dumproots`
1840. `launch$1`
1841. `libc\_kqueue\_trampoline`
1842. `New\[hash.Hash\]$1$1`
1843. `newproc$1`
1844. `scan$3$1`
1845. `init\#4`
1846. `file\_google\_protobuf\_struct\_proto\_rawDescGZIP$1`
1847. `logCloseHangDebugInfo$bound`
1848. `doRetryNotify\[\*github.com/testcontainers/testcontainers-go.DockerContainer\]$1`
1849. `libc\_munmap\_trampoline`
1850. `connect$2`
1851. `worker$2`
1852. `Alignof$1`
1853. `ExportPair$2`
1854. `main`
1855. `Execute$1`
1856. `TestScanColumns$3$1`
1857. `interrupt`
1858. `libc\_getpid\_trampoline`
1859. `init\#1`
1860. `Add$1`
1861. `Do$1`
1862. `init\#1`
1863. `minit`
1864. `handleSettings$1$1`
1865. `Breakpoint`
1866. `closeStream$1`
1867. `Free$bound`
1868. `SnapshotCoverage`
1869. `setCommandValueReflection$1`
1870. `log$1`
1871. `SafeGo$1`
1872. `NewClient$1`
1873. `lsandoleakcheck`
1874. `Setenv$2`
1875. `handleForwards$bound`
1876. `deferreturn`
1877. `x509\_CFStringCreateWithBytes\_trampoline`
1878. `TestRabbitMQAdapter\_ProcessDelivery\_AckError$1`
1879. `HandleCancel$1`
1880. `Collect$1`
1881. `getcp932$1`
1882. `runCmdContext$1`
1883. `open$1`
1884. `gcWriteBarrier2`
1885. `traceLockInit`
1886. `init\#2`
1887. `defaultUsage$bound`
1888. `ClearCache`
1889. `main`
1890. `keccakF1600Generic$1`
1891. `unpack$1`
1892. `init\#1`
1893. `initPredefined`
1894. `init\#1`
1895. `SendMsg$1`
1896. `libc\_sendfile\_trampoline`
1897. `mProf\_NextCycle`
1898. `init\#1`
1899. `init\#1`
1900. `osinit\_hack`
1901. `file\_google\_protobuf\_field\_mask\_proto\_init`
1902. `init\#2`
1903. `file\_grpc\_binlog\_v1\_binarylog\_proto\_rawDescGZIP$1`
1904. `exitsyscall$3`
1905. `SetTextMapPropagator$1`
1906. `doBlockingWithCtx$1`
1907. `libc\_getgrnam\_r\_trampoline`
1908. `InitializeTelemetry$1`
1909. `gcAssistAlloc$1`
1910. `forEachP$1`
1911. `traceAdvance$6`
1912. `WithDeadlineCause$2`
1913. `New$1$1`
1914. `doinit`
1915. `NewController$1`
1916. `dumpitabs`
1917. `getCaller$1`
1918. `panicmakeslicelen`
1919. `init\#2`
1920. `init$1`
1921. `startFlushGoroutine$1`
1922. `libc\_getpwnam\_r\_trampoline`
1923. `init\#1`
1924. `readARM64Registers`
1925. `init\#1`
1926. `processDelivery$1`
1927. `init\#1`
1928. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P384Point\]\]$1$1`
1929. `TestExtractExternalData\_SuccessWithNonFatalWarnings$1`
1930. `OnceValues\[map\[string\]rune map\[string\]\[2\]rune\]$1$1`
1931. `connect$1`
1932. `init\#1`
1933. `AddSet$1`
1934. `init$1`
1935. `libc\_setprivexec\_trampoline`
1936. `init\#15`
1937. `block`
1938. `sigpipe`
1939. `init\#2`
1940. `allPackages$1$1`
1941. `onSettingsTimer$bound`
1942. `TryGo$1$1`
1943. `asyncPreempt2`
1944. `poll\_runtime\_pollServerInit`
1945. `setMinimalFeatures`
1946. `Go$1$1`
1947. `libc\_getgroups\_trampoline`
1948. `consumeAddrSpec$1`
1949. `confirmLocks$1`
1950. `Goexit`
1951. `libc\_mknod\_trampoline`
1952. `log$1`
1953. `\_LostSIGPROFDuringAtomic64`
1954. `x509\_SecTrustCreateWithCertificates\_trampoline`
1955. `resolve$1`
1956. `x509\_SecTrustCopyCertificateChain\_trampoline`
1957. `pthread\_mutex\_lock\_trampoline`
1958. `goroutineProfileWithLabelsConcurrent$1`
1959. `newstack`
1960. `checkResponseErr$1`
1961. `subscribedState$1`
1962. `scan$4$1`
1963. `libc\_mlockall\_trampoline`
1964. `init$2`
1965. `RecordNonApproved`
1966. `New\[\*crypto/internal/fips140/sha256.Digest\]$1$1`
1967. `start$1`
1968. `SendMsg$2`
1969. `wakep`
1970. `init\#1`
1971. `entersyscallblock`
1972. `maybeRunChan$1`
1973. `encap$1`
1974. `traceExitingSyscall`
1975. `mstart1`
1976. `init\#1`
1977. `RegisterFunc$4$5`
1978. `tRunner$1$1`
1979. `New$1`
1980. `clearpools`
1981. `init\#1`
1982. `OnceValue$1$1`
1983. `three$1`
1984. `selectgo$3`
1985. `init\#1`
1986. `x509\_CFDictionaryGetValueIfPresent\_trampoline`
1987. `startGracefulShutdown$1`
1988. `bgRead$1`
1989. `goargs`
1990. `libc\_gettimeofday\_trampoline`
1991. `runPerThreadSyscall`
1992. `gcBgMarkPrepare`
1993. `merge$2`
1994. `init\#3`
1995. `log`
1996. `TestSetConfigFromEnvVars$1$1`
1997. `tunnel$1`
1998. `handshakeContext$2`
1999. `init$6`
2000. `gcWakeAllAssists`
2001. `startChannelWatcher$1`
2002. `typeDecl$2`
2003. `fatalpanic$2`
2004. `NewStreamReader$1`
2005. `ClientHandshake$1`
2006. `init\#1`
2007. `libc\_pipe\_trampoline`
2008. `CoordinateFuzzing$3`
2009. `Disable`
2010. `MasterAddr$1$1`
2011. `init\#8`
2012. `runtime\_doSpin`
2013. `Sync$1$1`
2014. `pthread\_cond\_signal\_trampoline`
2015. `main`
2016. `set\_crosscall2`
2017. `EndTracingSpans$1`
2018. `onDemandWorker$1`
2019. `doInRoot\[int\]$1`
2020. `PluginInstall$1`
2021. `DialContext$2`
2022. `runPiped$1`
2023. `libc\_unlockpt\_trampoline`
2024. `init\#1`
2025. `frameSkip$1`
2026. `lazyInit$1`
2027. `GetValue$1`
2028. `RegisterAsyncReporter$1`
2029. `initFeistelBox`
2030. `ClearNumberValidations$1`
2031. `assertWorldStopped`
2032. `shutdownWorkers$1`
2033. `libc\_getpgrp\_trampoline`
2034. `libc\_lstat\_trampoline`
2035. `DecodeFull$1`
2036. `main`
2037. `libc\_mprotect\_trampoline`
2038. `signalWaitUntilIdle`
2039. `close\_trampoline`
2040. `Execute$1`
2041. `run$4`
2042. `RecordSet$1`
2043. `init\#1`
2044. `abortStreamLocked$1`
2045. `TestQueryPluginCRM\_WithFilters$1`
2046. `initSystemRoots`
2047. `init\#5`
2048. `connStmt$1`
2049. `init\#2`
2050. `close$1`
2051. `init\#9`
2052. `handleLock$1`
2053. `runtime\_debug\_freeOSMemory`
2054. `nextBlock$1$2`
2055. `loadPackageNames$1`
2056. `SaveMultipartFile$2`
2057. `init\#1`
2058. `setupFallbackCache$1`
2059. `scheduleNextConnectionLocked$1`
2060. `startCheckmarks`
2061. `onReadTimeout$bound`
2062. `libc\_unlinkat\_trampoline`
2063. `AppendCertsFromPEM$1$1`
2064. `LoadLocation$1`
2065. `SetFallbackRoots$1`
2066. `handleSettings$1$1`
2067. `computeInterfaceTypeSet$2$1`
2068. `getcp1252$1`
2069. `shutdown$1`
2070. `init\#14`
2071. `search$1`
2072. `libc\_fsync\_trampoline`
2073. `TryAcquire\[\*github.com/jackc/pgx/v5/pgxpool.connResource\]$1`
2074. `\_LostExternalCode`
2075. `init$3`
2076. `probe$bound`
2077. `pthread\_cond\_wait\_trampoline`
2078. `mlock\_trampoline`
2079. `main`
2080. `noteUnusedDriverStatement$1`
2081. `syscall\_runtime\_BeforeFork`
2082. `Add$1`
2083. `main`
2084. `readServices`
2085. `getOrQueueForIdleConn$1`
2086. `libc\_symlinkat\_trampoline`
2087. `libc\_fstatfs\_trampoline`
2088. `SetFinalizer$2`
2089. `file\_google\_rpc\_status\_proto\_init`
2090. `init\#1`
2091. `main`
2092. `finishDebugVarsSetup`
2093. `setBypass`
2094. `execDC$3`
2095. `HandleStreams$1`
2096. `persistentalloc$1`
2097. `copyTrailersToHandlerRequest$bound`
2098. `Format$1`
2099. `clearSignalHandlers`
2100. `initAllChan$1`
2101. `Consume$1`
2102. `lazyInit$1`
2103. `initP521`
2104. `instantiateSignature$2`
2105. `TestInitWorker\_PanicsWhenTelemetryGlobalsFail$6`
2106. `duffzero`
2107. `ServeHTTP$1`
2108. `file\_google\_api\_httpbody\_proto\_rawDescGZIP$1`
2109. `initP384`
2110. `usleep\_trampoline`
2111. `init\#1`
2112. `gcMarkDone$3`
2113. `libc\_setlogin\_trampoline`
2114. `TraceQueryute$1`
2115. `dialSingle$1`
2116. `init\#1`
2117. `grabConn$1`
2118. `SetLoggerProvider$1`
2119. `computeInterfaceTypeSet$1`
2120. `NewPublicKey$1`
2121. `init\#1`
2122. `libc\_fchflags\_trampoline`
2123. `doBlockingWithCtx\[\[\]net.IPAddr\]$1`
2124. `stopm`
2125. `init$4`
2126. `getcp437$1`
2127. `modify$1`
2128. `processUnaryRPC$2`
2129. `traceQuery$1`
2130. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P224Point\]\]$1$1`
2131. `newClientStream$5`
2132. `Run$1`
2133. `getMacOSMajor$1`
2134. `SetTracerProvider$1`
2135. `nextBlock$1$1`
2136. `init\#1`
2137. `panicUnaligned`
2138. `init\#1`
2139. `Sync$1`
2140. `main`
2141. `growSlice$1`
2142. `init\#1`
2143. `Setenv$1`
2144. `TestWithSwaggerEnvConfig\_WithEnvVars$1`
2145. `newNonRetryClientStream$1`
2146. `gfget$2`
2147. `onIdleTimeout$bound`
2148. `forceCloseConn$bound`
2149. `parseFieldName$1`
2150. `failWantMap`
2151. `lookupIPAddr$3`
2152. `collectRecv$1`
2153. `closeConnIfStillIdle$bound`
2154. `munmap\_trampoline`
2155. `Init`
2156. `sigaltstack\_trampoline`
2157. `init\#1`
2158. `p224B$1`
2159. `EventuallyWithT$1`
2160. `Shutdown$1`
2161. `forceCloseConn$bound`
2162. `stopTheWorld$1`
2163. `cgoCheckPtrWrite$1`
2164. `init\#1`
2165. `init\#1`
2166. `stdoutHandler$1`
2167. `init\#2`
2168. `open\_trampoline`
2169. `handleCleanCache$1`
2170. `nify$1`
2171. `main`
2172. `TestEnsureConfigFromEnvVars$2$1`
2173. `collect$1`
2174. `gcWriteBarrier8`
2175. `proc\_regionfilename\_trampoline`
2176. `RangeFields$1`
2177. `OnceValue\[\*crypto/internal/fips140/ecdsa.Curve\[\*crypto/internal/fips140/nistec.P224Point\]\]$1$1$1`
2178. `setUserArenaChunkToFault$1`
2179. `TestRabbitMQAdapter\_ConsumerLoop\_AcksOnSuccess$1$1`
2180. `libc\_fstatat\_trampoline`
2181. `x509\_CFErrorGetCode\_trampoline`
2182. `typelinksinit`
2183. `Multiplexed$1$1`
2184. `marshal$1`
2185. `setRequestCancel$4`
2186. `addTLS$2`
2187. `libc\_getpwuid\_r\_trampoline`
2188. `ExportPair$1`
2189. `onReadTimeout$bound`
2190. `Eventually$1`
2191. `x509\_CFErrorCopyDescription\_trampoline`
2192. `awaitGoroutines$1`
2193. `servePeer$2`
2194. `netpollGenericInit`
2195. `Parse`
2196. `AcquireConn$1`
2197. `sigpanic0`
2198. `Stop`
2199. `x509\_CFEqual\_trampoline`
2200. `infer$2`
2201. `healthCheck$1`
2202. `RunConsumers$1`
2203. `main\_main`
2204. `getMetadata$1`
2205. `sync\_runtime\_doSpin`
2206. `validCycle$1`
2207. `funcDecl$1`
2208. `search$2$1`
2209. `init\#1`
2210. `raise\_trampoline`
2211. `handleMonitoredClose$1`
2212. `libc\_flock\_trampoline`
2213. `Close$bound`
2214. `finishStream$1`
2215. `traceStartReadCPU$1`
2216. `defaultCleanUp$1`
2217. `libc\_adjtime\_trampoline`
2218. `traceAdvance$3`
2219. `AddSet$1`
2220. `casgstatus$3`
2221. `readTrace0$1`
2222. `OnEmit$1`
2223. `getPossiblyUnhashableKey$1`
2224. `fixedHuffmanDecoderInit$1`
2225. `init$5`
2226. `ensureMetricsCollector$1`
2227. `defaultGOMAXPROCSInit`
2228. `buildImageWithSecrets$1`
2229. `unspillArgs`
2230. `init\#1`
2231. `osyield\_no\_g`
2232. `file\_opentelemetry\_proto\_collector\_metrics\_v1\_metrics\_service\_proto\_init`
2233. `TestWithSwaggerEnvConfig\_InvalidHost$1`
2234. `unpinConnectionFromTransaction$bound`
2235. `x509\_SecTrustEvaluate\_trampoline`
2236. `Chdir$1`
2237. `runHandler$1`
2238. `CleanupNetwork$1`
2239. `alginit`
2240. `libc\_getsockname\_trampoline`
2241. `libc\_accept\_trampoline`
2242. `overrideUmask$1`
2243. `libc\_mkfifo\_trampoline`
2244. `Clearenv`
2245. `ensureSensitiveFieldsMap$1`
2246. `init$1`
2247. `init\#1`
2248. `init\#1`
2249. `Read$1`
2250. `doBlockingWithCtx\[int\]$1`
2251. `init\#1`
2252. `Close$1`
2253. `RecordSet$1`
2254. `FIPSOnly`
2255. `EncodeAll$1`
2256. `init\#1`
2257. `secretEraseRegisters`
2258. `copyTrailers$bound`
2259. `run$3`
2260. `OrderedMarshal$1`
2261. `init\#10`
2262. `Execute$1`
2263. `file\_google\_protobuf\_timestamp\_proto\_rawDescGZIP$1`
2264. `libc\_seteuid\_trampoline`
2265. `Execute$2`
2266. `generatorTable$1`
2267. `dumpgs`
2268. `parseExpr$1`
2269. `mmap\_trampoline`
2270. `RecordSet$1`
2271. `expandRHS$1`
2272. `nilfunc`
2273. `processPipelineNode$1$1`
2274. `optionsUnmarshaler$1$1`
2275. `OnceValue\[string\]$1$1$1`
2276. `init\#1`
2277. `freeOSMemory`
2278. `decryptKey$1`
2279. `libc\_getegid\_trampoline`
2280. `worker$1`
2281. `libc\_lchown\_trampoline`
2282. `pthread\_key\_create\_trampoline`
2283. `debugCallV2`
2284. `main`
2285. `mergeRuneSets$1`
2286. `main`
2287. `dispatchDeliveries$1`
2288. `lazyInit$1`
2289. `init\#3`
2290. `init\#3`
2291. `mPark`
2292. `libc\_execve\_trampoline`
2293. `init\#1`
2294. `lockProfiles`
2295. `badcgocallback`
2296. `debugCallCheck$1`
2297. `dolockOSThread`
2298. `newHealthData$1`
2299. `startServers$3`
2300. `asyncClose$1`
2301. `libc\_fchdir\_trampoline`
2302. `startWatcher$1`
2303. `gcMarkRootCheck`
2304. `signalWaitUntilIdle`
2305. `updateServerDate$1`
2306. `init$1$1`
2307. `racefingo`
2308. `WithDataIndependentTiming$1`
2309. `instantiateSignature$1`
2310. `init\#1`
2311. `onIdleTimer$bound`
2312. `dialConn$3`
2313. `Add$1`
2314. `init\#1`
2315. `morestack\_noctxt`
2316. `enableWER`
2317. `initialize$1`
2318. `gcMarkDone$4`
2319. `libc\_getpriority\_trampoline`
2320. `updateServerDate`
2321. `rt0\_go`
2322. `x509\_CFDateCreate\_trampoline`
2323. `init$1`
2324. `init\#1`
2325. `init\#1`
2326. `Wait`
2327. `buildCommonHeaderMapsOnce`
2328. `allocmcache$1`
2329. `traceDataRow$1`
2330. `buildImageWithSecrets$1`
2331. `Raw$1`
2332. `OnceFunc$1`
2333. `Close$1`
2334. `newCompressedFSFileCache$1`
2335. `releaseForkLock`
2336. `Primary`
2337. `\*go.mongodb.org/mongo-driver/mongo.Client.Ping`
2338. `Errorf`

#### `TestPingMongo`
**File:** `pkg/mongodb/mongo\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`

#### `testContext`
**File:** `pkg/mysql/datasource.mysql\_test.go`
**Risk Level:** HIGH (8 direct callers)

**Direct Callers (signature change affects these):**

1. `TestExternalDataSource\_MySQL\_GetDatabaseSchema` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql\_test.go:703`
2. `TestExternalDataSource\_MySQL\_Query` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql\_test.go:124`
3. `TestExternalDataSource\_MySQL\_ValidateTableAndFields` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql\_test.go:574`
4. `TestNewLoggerFromContext$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:57`
5. `TestContextWithTracer$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:221`
6. `TestContextWithLogger$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:167`
7. `TestExternalDataSource\_MySQL\_QueryWithAdvancedFilters` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/mysql/datasource.mysql\_test.go:363`
8. `TestNewTracerFromContext$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/context\_test.go:112`

**Callees (this function depends on):**

1. `Tracer`
2. `Background`
3. `WithValue`

#### `TestQueryHeaderMetadata`
**File:** `pkg/net/http/http-utils\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `\*testing.T.Run`
2. `\*testing.T.Run`
3. `\*testing.T.Run`
4. `\*testing.T.Run`

#### `TestValidateParameters`
**File:** `pkg/net/http/http-utils\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `Setenv`
2. `Setenv`
3. `TestValidateParameters$1`
4. `\*testing.T.Run`

#### `TestValidateParametersNonMetadataKeys`
**File:** `pkg/net/http/http-utils\_test.go`
**Risk Level:** HIGH (9 direct callers)

**Direct Callers (signature change affects these):**

1. `TestCustomizerOptions\_MutateContainerRequest$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/customizer\_options\_test.go:182`
2. `TestQueueAssertionsHelpers$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/queuekit/assertions\_test.go:198`
3. `TestBuilderConfigurationAndSuiteLifecycle$6` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/suite\_test.go:193`
4. `TestSeaweedFSHelpersWithoutDocker$8` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:157`
5. `TestWithSwaggerEnvConfig\_WithEnvVars$10` - `/Users/fredamaral/repos/lerianstudio/fetcher/components/manager/internal/adapters/http/in/swagger\_test.go:187`
6. `tRunner` - `/opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036`
7. `TestBuildImageWithSecretsValidation$2` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:62`
8. `TestBuilderHelpersAndProjectRoot$7` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/e2ekit/helpers\_test.go:228`
9. `TestChaosMetricsAndReporting$5` - `/Users/fredamaral/repos/lerianstudio/fetcher/pkg/itestkit/addons/metricskit/metrics\_test.go:238`

**Callees (this function depends on):**

1. `Setenv`
2. `Setenv`
3. `TestValidateParametersNonMetadataKeys$1`
4. `\*testing.T.Run`



## Semantic Changes


### Functions with Logic Changes

1. **`.\\*ConnectionHandler.ListConnections`** - implementation changed
2. **`.\\*ConnectionHandler.GetConnectionSchema`** - implementation changed
3. **`.\\*ConnectionHandler.CreateConnection`** - implementation changed
4. **`.\\*ConnectionHandler.GetConnection`** - implementation changed
5. **`.\\*ConnectionHandler.TestConnection`** - implementation changed
6. **`.\\*ConnectionHandler.UpdateConnection`** - implementation changed
7. **`.\\*ConnectionHandler.DeleteConnection`** - implementation changed
8. **`.\\*ConnectionHandler.ValidateSchema`** - implementation changed
9. **`.setupConnectionTestApp`** - implementation changed
10. **`.\\*FetcherHandler.GetJob`** - implementation changed
11. **`.\\*FetcherHandler.CreateJob`** - implementation changed
12. **`.setupTestApp`** - implementation changed
13. **`.setupMiddlewareTestApp`** - implementation changed
14. **`.\\*MigrationHandler.AssignConnectionToProduct`** - implementation changed
15. **`.\\*MigrationHandler.ListUnassignedConnections`** - implementation changed
16. **`.\\*ProductHandler.CreateProduct`** - implementation changed
17. **`.\\*ProductHandler.ListProducts`** - implementation changed
18. **`.\\*ProductHandler.GetProduct`** - implementation changed
19. **`.\\*ProductHandler.UpdateProduct`** - implementation changed
20. **`.\\*ProductHandler.DeleteProduct`** - implementation changed
21. **`.NewRoutes`** - implementation changed
22. **`.InitServers`** - implementation changed
23. **`.\\*Server.Run`** - implementation changed
24. **`.\\*AssignConnection.Execute`** - implementation changed
25. **`.\\*CreateConnection.Execute`** - implementation changed
26. **`.NewCreateFetcherJob`** - parameters changed, signature\\_changed
27. **`.NewCreateFetcherJobWithTester`** - parameters changed, signature\\_changed
28. **`.\\*CreateFetcherJob.Execute`** - implementation changed
29. **`.\\*CreateFetcherJob.validateProductOwnership`** - parameters changed, implementation changed, signature\\_changed
30. **`.\\*CreateFetcherJob.TestConnection`** - implementation changed
31. **`.\\*CreateProduct.Execute`** - implementation changed
32. **`.\\*DeleteProduct.Execute`** - implementation changed
33. **`.testContext`** - implementation changed
34. **`.\\*UpdateConnection.Execute`** - implementation changed
35. **`.\\*UpdateProduct.Execute`** - implementation changed
36. **`.\\*GetConnectionSchema.Execute`** - implementation changed
37. **`.testContext`** - implementation changed
38. **`.\\*ListConnections.Execute`** - implementation changed
39. **`.\\*ListProducts.Execute`** - implementation changed
40. **`.\\*ListUnassignedConnections.Execute`** - implementation changed
41. **`.\\*TestConnection.Execute`** - implementation changed
42. **`.NewTestConnection`** - implementation changed
43. **`.TestTestConnection\\_Execute\\_WithSSLConfiguration`** - implementation changed
44. **`.TestTestConnection\\_Execute\\_AllDatabaseTypes`** - implementation changed
45. **`.NewValidateSchema`** - implementation changed
46. **`.\\*ValidateSchema.Execute`** - implementation changed
47. **`.\\*ValidateSchema.getOrFetchSchema`** - implementation changed
48. **`.TestValidateSchema\\_CacheSetError`** - implementation changed
49. **`.TestValidateSchema\\_CacheError\\_ContinuesToFetch`** - implementation changed
50. **`.\\*ConsumerRoutes.Shutdown`** - implementation changed
51. **`.NewConsumerRoutes`** - parameters changed, signature\\_changed
52. **`.NewConsumerRoutesWithAdapter`** - parameters changed, signature\\_changed
53. **`.\\*ConsumerRoutes.RunConsumers`** - implementation changed
54. **`.TestNewConsumerRoutes`** - implementation changed
55. **`.TestConsumerRoutes\\_Shutdown`** - implementation changed
56. **`.TestConsumerRoutes\\_Shutdown\\_Error`** - implementation changed
57. **`.TestConsumerRoutes\\_RunConsumers`** - implementation changed
58. **`.NewPublisherRoutes`** - parameters changed, signature\\_changed
59. **`.NewPublisherRoutesWithAdapter`** - parameters changed, signature\\_changed
60. **`.\\*PublisherRoutes.Publish`** - implementation changed
61. **`.\\*PublisherRoutes.Shutdown`** - implementation changed
62. **`.TestPublisherRoutes\\_Shutdown`** - implementation changed
63. **`.TestNewPublisherRoutes`** - implementation changed
64. **`.TestPublisherRoutes\\_Publish`** - implementation changed
65. **`.InitWorker`** - implementation changed
66. **`.\\*MultiQueueConsumer.Run`** - implementation changed
67. **`.\\*MultiQueueConsumer.handlerGenerateReport`** - implementation changed
68. **`.\\*Service.Run`** - implementation changed
69. **`.\\*UseCase.ExtractExternalData`** - implementation changed
70. **`.\\*UseCase.saveExternalDataToSeaweedFS`** - parameters changed, implementation changed, signature\\_changed
71. **`.\\*UseCase.shouldSkipProcessing`** - parameters changed, implementation changed, signature\\_changed
72. **`.\\*UseCase.parseMessage`** - parameters changed, implementation changed, signature\\_changed
73. **`.\\*UseCase.extractJobIDFromMultipleSources`** - parameters changed, implementation changed, signature\\_changed
74. **`.\\*UseCase.extractJobIDFromPartialJSON`** - parameters changed, implementation changed, signature\\_changed
75. **`.\\*UseCase.queryDatabase`** - parameters changed, implementation changed, signature\\_changed
76. **`.\\*UseCase.handleErrorWithUpdate`** - parameters changed, implementation changed, signature\\_changed
77. **`.\\*UseCase.encryptDataForSeaweedFS`** - parameters changed, signature\\_changed
78. **`.\\*UseCase.checkReportStatus`** - parameters changed, implementation changed, signature\\_changed
79. **`.\\*UseCase.transformPluginCRMAdvancedFilters`** - parameters changed, implementation changed, signature\\_changed
80. **`.\\*UseCase.decryptPluginCRMData`** - parameters changed, signature\\_changed
81. **`.\\*UseCase.QueryPluginCRM`** - parameters changed, implementation changed, signature\\_changed
82. **`.\\*UseCase.queryPluginCRMCollectionWithFilters`** - parameters changed, implementation changed, signature\\_changed
83. **`.\\*UseCase.processPluginCRMCollection`** - parameters changed, implementation changed, signature\\_changed
84. **`.TestQueryPluginCRMCollectionWithFilters\\_NoFilters`** - implementation changed
85. **`.TestQueryPluginCRM\\_WithOrganizationOnly`** - implementation changed
86. **`.TestQueryPluginCRM\\_WithFilters`** - implementation changed
87. **`.TestProcessPluginCRMCollection\\_WithValidOrganization`** - implementation changed
88. **`.TestProcessPluginCRMCollection\\_WithOrganizationID`** - implementation changed
89. **`.TestExtractJobIDFromMultipleSources\\_EdgeCases`** - implementation changed
90. **`.TestExtractJobIDFromMultipleSources\\_FromHeaders`** - implementation changed
91. **`.TestExtractJobIDFromPartialJSON`** - implementation changed
92. **`.TestExtractJobIDFromMultipleSources\\_FromPartialJSON`** - implementation changed
93. **`.TestExtractJobIDFromMultipleSources\\_NoIDs`** - implementation changed
94. **`.TestExtractJobIDFromPartialJSON\\_ValidJobIDInvalidOrgID`** - implementation changed
95. **`.TestExtractJobIDFromPartialJSON\\_RegexFallback`** - implementation changed
96. **`.TestExtractJobIDFromMultipleSources\\_HeaderPrecedence`** - implementation changed
97. **`.\\*UseCase.publishJobNotification`** - parameters changed, implementation changed, signature\\_changed
98. **`.testLogger`** - implementation changed
99. **`.NewLoggerFromContext`** - implementation changed
100. **`.TestCustomContextKey\\_Type`** - implementation changed
101. **`.TestNewLoggerFromContext`** - implementation changed
102. **`.TestContextWithLogger`** - implementation changed
103. **`.TestContextWithTracer`** - implementation changed
104. **`.TestContextWithLoggerAndTracer\\_Combined`** - implementation changed
105. **`.TestCustomContextKeyValue\\_Integration`** - implementation changed
106. **`.newDataSourceConfigMongoDB`** - parameters changed, implementation changed, signature\\_changed
107. **`.newDataSourceConfigPostgres`** - parameters changed, implementation changed, signature\\_changed
108. **`.newDataSourceConfigOracle`** - parameters changed, implementation changed, signature\\_changed
109. **`.newDataSourceConfigMySQL`** - parameters changed, implementation changed, signature\\_changed
110. **`.newDataSourceConfigSQLServer`** - parameters changed, implementation changed, signature\\_changed
111. **`.NewDataSourceFromConnectionWithLogger`** - parameters changed, signature\\_changed
112. **`.NewDataSourceFromConnection`** - parameters changed, implementation changed, signature\\_changed
113. **`.buildImageWithSecrets`** - implementation changed
114. **`.generateImageTag`** - implementation changed
115. **`.New`** - implementation changed
116. **`.WaitLog`** - implementation changed
117. **`.WaitPort`** - implementation changed
118. **`.\\*Builder.WithRewriter`** - implementation changed
119. **`.WaitHTTP`** - implementation changed
120. **`.localhostToHostGatewayRewriter.Rewrite`** - implementation changed
121. **`.\\*Builder.WithImage`** - implementation changed
122. **`.\\*Builder.Run`** - implementation changed
123. **`.uniqueAppend`** - implementation changed
124. **`.rewriteLocalhostForContainer`** - implementation changed
125. **`.dumpRecentLogs`** - implementation changed
126. **`.\\*Builder.WithEnv`** - implementation changed
127. **`.\\*Builder.ExposePort`** - implementation changed
128. **`.\\*Builder.DisableDefaultLocalhostRewrite`** - implementation changed
129. **`.\\*Builder.WithLogsOnFailureMaxBytes`** - implementation changed
130. **`.\\*Builder.WithWait`** - implementation changed
131. **`.cloneMap`** - implementation changed
132. **`.WaitRunning`** - implementation changed
133. **`.waitHTTP.Configure`** - implementation changed
134. **`.ProjectRoot`** - implementation changed
135. **`.ProjectRootFrom`** - implementation changed
136. **`.\\*ChaosAssertions.TimeoutsBelow`** - implementation changed
137. **`.\\*ChaosAssertions.ThroughputAbove`** - implementation changed
138. **`.\\*ChaosAssertions.FailedResults`** - implementation changed
139. **`.\\*ChaosAssertions.P50Below`** - implementation changed
140. **`.\\*ChaosAssertions.MinRequestsReached`** - implementation changed
141. **`.\\*ChaosAssertions.Summary`** - implementation changed
142. **`.\\*ChaosAssertions.AverageLatencyBelow`** - implementation changed
143. **`.\\*ChaosAssertions.FailuresBelow`** - implementation changed
144. **`.\\*ChaosAssertions.SuccessRateAbove`** - implementation changed
145. **`.\\*ChaosAssertions.P99Below`** - implementation changed
146. **`.\\*ChaosAssertions.P95Below`** - implementation changed
147. **`.\\*ErrorClassifier.GetCategoryCounts`** - implementation changed
148. **`.\\*ChaosMetrics.GetTotalRequests`** - implementation changed
149. **`.\\*ChaosMetrics.SuccessRate`** - implementation changed
150. **`.\\*ChaosMetrics.StartTest`** - implementation changed
151. **`.\\*ChaosMetrics.StartChaos`** - implementation changed
152. **`.\\*ChaosMetrics.GetErrorCounts`** - implementation changed
153. **`.\\*ChaosMetrics.ChaosThroughputRPS`** - implementation changed
154. **`.\\*ChaosMetrics.Percentile`** - implementation changed
155. **`.\\*ChaosMetrics.EndChaos`** - implementation changed
156. **`.\\*ChaosMetrics.RecordRequest`** - implementation changed
157. **`.\\*ChaosMetrics.GetTimeoutRequests`** - implementation changed
158. **`.\\*ChaosMetrics.SuccessfulThroughputRPS`** - implementation changed
159. **`.\\*ChaosMetrics.ChaosDuration`** - implementation changed
160. **`.\\*ChaosMetrics.TestDuration`** - implementation changed
161. **`.\\*ChaosMetrics.GetFailedRequests`** - implementation changed
162. **`.\\*ChaosMetrics.GetMinLatency`** - implementation changed
163. **`.\\*ChaosMetrics.AverageLatency`** - implementation changed
164. **`.\\*ChaosMetrics.ThroughputRPS`** - implementation changed
165. **`.\\*ChaosMetrics.EndTest`** - implementation changed
166. **`.\\*Reporter.WriteReport`** - implementation changed
167. **`.\\*Reporter.String`** - implementation changed
168. **`.\\*Reporter.CompactSummary`** - implementation changed
169. **`.\\*AMQPConsumerBuilder.WithPrefetch`** - implementation changed
170. **`.\\*AMQPConsumer.Close`** - implementation changed
171. **`.\\*AMQPPublisher.connect`** - implementation changed
172. **`.\\*AMQPPublisher.Close`** - implementation changed
173. **`.\\*AMQPConsumerBuilder.BindTo`** - implementation changed
174. **`.\\*AMQPConsumerBuilder.WithQueueDeclare`** - implementation changed
175. **`.NewAMQPConsumer`** - implementation changed
176. **`.\\*AMQPConsumer.connect`** - implementation changed
177. **`.\\*Assertions\\[T\\].HasHeaderKey`** - implementation changed
178. **`.\\*ResultAssertions\\[T\\].At`** - implementation changed
179. **`.\\*MessageSequence\\[T\\].FilterBy`** - implementation changed
180. **`.\\*ExpectMessagesHelper\\[T\\].OrFatal`** - implementation changed
181. **`.\\*Assertions\\[T\\].PayloadSatisfies`** - implementation changed
182. **`.\\*ResultAssertions\\[T\\].UnmatchedCount`** - implementation changed
183. **`.\\*Assertions\\[T\\].HasRoutingKey`** - implementation changed
184. **`.\\*Assertions\\[T\\].HasMessageID`** - implementation changed
185. **`.\\*Assertions\\[T\\].HasContentType`** - implementation changed
186. **`.\\*ResultAssertions\\[T\\].First`** - implementation changed
187. **`.AssertResult`** - implementation changed
188. **`.AssertJSONEqual`** - implementation changed
189. **`.\\*MessageSequence\\[T\\].GroupBy`** - implementation changed
190. **`.\\*ExpectMessagesHelper\\[T\\].ToContainWhere`** - implementation changed
191. **`.AssertMessage`** - implementation changed
192. **`.\\*Assertions\\[T\\].HasHeader`** - implementation changed
193. **`.\\*ResultAssertions\\[T\\].HasAtLeast`** - implementation changed
194. **`.JSONEqual`** - implementation changed
195. **`.\\*ExpectMessagesHelper\\[T\\].ToHaveCount`** - implementation changed
196. **`.\\*Assertions\\[T\\].HasCorrelationID`** - implementation changed
197. **`.\\*Assertions\\[T\\].PayloadEquals`** - implementation changed
198. **`.\\*ResultAssertions\\[T\\].HasCount`** - implementation changed
199. **`.\\*ResultAssertions\\[T\\].DidNotTimeout`** - implementation changed
200. **`.\\*ResultAssertions\\[T\\].HasNoErrors`** - implementation changed
201. **`.\\*ResultAssertions\\[T\\].All`** - implementation changed
202. **`.\\*MessageSequence\\[T\\].RoutingKeysInOrder`** - implementation changed
203. **`.\\*ExpectMessagesHelper\\[T\\].ToSucceed`** - implementation changed
204. **`.\\*Consumer\\[T\\].CaptureAll`** - implementation changed
205. **`.\\*ConsumerBuilder\\[T\\].WithMatcher`** - implementation changed
206. **`.\\*ConsumerBuilder\\[T\\].WithTimeout`** - implementation changed
207. **`.\\*Consumer\\[T\\].DrainQueue`** - implementation changed
208. **`.\\*Consumer\\[T\\].captureMessage`** - implementation changed
209. **`.NewConsumer`** - implementation changed
210. **`.truncateBody`** - implementation changed
211. **`.\\*ConsumerBuilder\\[T\\].WithUnmarshaler`** - implementation changed
212. **`.\\*ConsumerBuilder\\[T\\].Build`** - implementation changed
213. **`.\\*Consumer\\[T\\].WaitForMessages`** - implementation changed
214. **`.\\*Consumer\\[T\\].GetCaptured`** - implementation changed
215. **`.\\*Consumer\\[T\\].ClearCaptured`** - implementation changed
216. **`.compareValues`** - implementation changed
217. **`.MatchHeader`** - implementation changed
218. **`.MatchHeaderExists`** - implementation changed
219. **`.MatchJSONField`** - implementation changed
220. **`.MatchJSONFieldPattern`** - implementation changed
221. **`.MatchBodyPattern`** - implementation changed
222. **`.hasNestedValue`** - implementation changed
223. **`.MatchAll`** - implementation changed
224. **`.MatchAny`** - implementation changed
225. **`.MatchRoutingKeyPattern`** - implementation changed
226. **`.applyPublishOptions`** - implementation changed
227. **`.WaitResult\\[T\\].First`** - implementation changed
228. **`.\\*toxiproxyChaos.RemoveToxic`** - implementation changed
229. **`.\\*toxiproxyChaos.RemoveAllToxics`** - implementation changed
230. **`.\\*toxiproxyChaos.Close`** - implementation changed
231. **`.NewToxiproxyChaos`** - implementation changed
232. **`.\\*toxiproxyChaos.CreateProxy`** - implementation changed
233. **`.\\*toxiproxyChaos.AddBandwidth`** - implementation changed
234. **`.\\*toxiproxyChaos.AddLatency`** - implementation changed
235. **`.\\*toxiproxyChaos.CutConnection`** - implementation changed
236. **`.\\*toxiproxyChaos.AddTimeout`** - implementation changed
237. **`.WaitListeningPort.Apply`** - implementation changed
238. **`.\\*Builder.WithContainerCustomize`** - implementation changed
239. **`.\\*genericContainerInfra.Start`** - implementation changed
240. **`.\\*genericContainerInfra.Terminate`** - implementation changed
241. **`.portKey`** - implementation changed
242. **`.CustomizerFunc.Customize`** - implementation changed
243. **`.MergeCustomizers`** - implementation changed
244. **`.CExposedPorts`** - implementation changed
245. **`.CEnvFromOS`** - implementation changed
246. **`.uniqueAppendMany`** - implementation changed
247. **`.CBindMount`** - implementation changed
248. **`.CEnvs`** - implementation changed
249. **`.CHostDockerInternal`** - implementation changed
250. **`.CNetworks`** - implementation changed
251. **`.CAll`** - implementation changed
252. **`.CNetworkAliases`** - implementation changed
253. **`.HostGatewayIP`** - implementation changed
254. **`.validateUniqueInfraNames`** - implementation changed
255. **`.NewMongoDBInfra`** - implementation changed
256. **`.\\*MongoDBInfra.HostPort`** - implementation changed
257. **`.NewMongoDBInfraStub`** - implementation changed
258. **`.\\*MongoDBInfra.Start`** - implementation changed
259. **`.\\*MongoDBInfra.Endpoint`** - implementation changed
260. **`.\\*MongoDBInfra.URI`** - implementation changed
261. **`.\\*MongoDBInfra.Terminate`** - implementation changed
262. **`.WithMongoDBFixedPort`** - implementation changed
263. **`.\\*MSSQLInfra.Endpoint`** - implementation changed
264. **`.NewMSSQLInfraStub`** - implementation changed
265. **`.NewMSSQLInfra`** - implementation changed
266. **`.\\*MSSQLInfra.Start`** - implementation changed
267. **`.\\*MSSQLInfra.DSN`** - implementation changed
268. **`.\\*MSSQLInfra.HostPort`** - implementation changed
269. **`.\\*MSSQLInfra.Terminate`** - implementation changed
270. **`.WithMSSQLFixedPort`** - implementation changed
271. **`.NewMySQLInfra`** - implementation changed
272. **`.\\*MySQLInfra.Endpoint`** - implementation changed
273. **`.\\*MySQLInfra.Terminate`** - implementation changed
274. **`.\\*MySQLInfra.Start`** - implementation changed
275. **`.\\*MySQLInfra.DSN`** - implementation changed
276. **`.\\*MySQLInfra.HostPort`** - implementation changed
277. **`.NewMySQLInfraStub`** - implementation changed
278. **`.WithMySQLInitScript`** - implementation changed
279. **`.WithMySQLFixedPort`** - implementation changed
280. **`.\\*OracleInfra.Endpoint`** - implementation changed
281. **`.NewOracleInfraStub`** - implementation changed
282. **`.\\*OracleInfra.DSN`** - implementation changed
283. **`.\\*OracleInfra.GoDRORDSN`** - implementation changed
284. **`.\\*OracleInfra.HostPort`** - implementation changed
285. **`.\\*OracleInfra.Terminate`** - implementation changed
286. **`.NewOracleInfra`** - implementation changed
287. **`.\\*OracleInfra.Start`** - implementation changed
288. **`.WithOracleInitScript`** - implementation changed
289. **`.WithOracleFixedPort`** - implementation changed
290. **`.\\*PostgresInfra.Start`** - implementation changed
291. **`.\\*PostgresInfra.DSN`** - implementation changed
292. **`.\\*PostgresInfra.HostPort`** - implementation changed
293. **`.NewPostgresInfra`** - implementation changed
294. **`.\\*PostgresInfra.Endpoint`** - implementation changed
295. **`.\\*PostgresInfra.Terminate`** - implementation changed
296. **`.NewPostgresInfraStub`** - implementation changed
297. **`.WithPGInitFile`** - implementation changed
298. **`.WithPGFixedPort`** - implementation changed
299. **`.WithRabbitFixedPort`** - implementation changed
300. **`.\\*configReader.Read`** - implementation changed
301. **`.NewRabbitInfra`** - implementation changed
302. **`.\\*RabbitInfra.Start`** - implementation changed
303. **`.\\*RabbitInfra.Endpoint`** - implementation changed
304. **`.\\*RabbitInfra.AMQPURL`** - implementation changed
305. **`.\\*RabbitInfra.HostPort`** - implementation changed
306. **`.\\*RabbitInfra.Terminate`** - implementation changed
307. **`.NewRedisInfra`** - implementation changed
308. **`.\\*RedisInfra.Start`** - implementation changed
309. **`.\\*RedisInfra.HostPort`** - implementation changed
310. **`.\\*RedisInfra.Terminate`** - implementation changed
311. **`.\\*RedisInfra.Endpoint`** - implementation changed
312. **`.\\*RedisInfra.URL`** - implementation changed
313. **`.\\*RedisInfra.Addr`** - implementation changed
314. **`.WithRedisFixedPort`** - implementation changed
315. **`.NewSeaweedFSInfra`** - implementation changed
316. **`.\\*SeaweedFSInfra.Start`** - implementation changed
317. **`.\\*SeaweedFSInfra.Endpoint`** - implementation changed
318. **`.\\*SeaweedFSInfra.URL`** - implementation changed
319. **`.\\*SeaweedFSInfra.HostPort`** - implementation changed
320. **`.\\*SeaweedFSInfra.Terminate`** - implementation changed
321. **`.WithSeaweedFSFixedPort`** - implementation changed
322. **`.\\*Builder.Build`** - implementation changed
323. **`.\\*Suite.Terminate`** - implementation changed
324. **`.New`** - implementation changed
325. **`.\\*Builder.WithInfra`** - implementation changed
326. **`.\\*Builder.WithInfras`** - implementation changed
327. **`.\\*Suite.Network`** - implementation changed
328. **`.\\*DataSourceConfigMongoDB.GetSchemaInfo`** - implementation changed
329. **`.\\*DataSourceConfigMongoDB.Connect`** - parameters changed, implementation changed, signature\\_changed
330. **`.\\*DataSourceConfigMongoDB.Query`** - parameters changed, implementation changed, signature\\_changed
331. **`.\\*DataSourceConfigMySQL.Connect`** - parameters changed, implementation changed, signature\\_changed
332. **`.\\*DataSourceConfigMySQL.Query`** - parameters changed, implementation changed, signature\\_changed
333. **`.\\*DataSourceConfigMySQL.GetSchemaInfo`** - implementation changed
334. **`.\\*DataSourceConfigOracle.Connect`** - parameters changed, implementation changed, signature\\_changed
335. **`.\\*DataSourceConfigOracle.Query`** - parameters changed, implementation changed, signature\\_changed
336. **`.\\*DataSourceConfigOracle.GetSchemaInfo`** - implementation changed
337. **`.\\*DataSourceConfigPostgres.Connect`** - parameters changed, implementation changed, signature\\_changed
338. **`.\\*DataSourceConfigPostgres.GetSchemaInfo`** - implementation changed
339. **`.\\*DataSourceConfigPostgres.Query`** - parameters changed, implementation changed, signature\\_changed
340. **`.\\*DataSourceConfigSQLServer.Query`** - parameters changed, implementation changed, signature\\_changed
341. **`.\\*DataSourceConfigSQLServer.GetSchemaInfo`** - implementation changed
342. **`.\\*DataSourceConfigSQLServer.Connect`** - parameters changed, implementation changed, signature\\_changed
343. **`.\\*ConnectionMongoDBRepository.ListUnassigned`** - implementation changed
344. **`.\\*ConnectionMongoDBRepository.AssignProduct`** - implementation changed
345. **`.\\*ConnectionMongoDBRepository.FindByID`** - implementation changed
346. **`.\\*ConnectionMongoDBRepository.FindByOrganizationAndName`** - implementation changed
347. **`.\\*ConnectionMongoDBRepository.FindByOrganizationAndDatabaseName`** - implementation changed
348. **`.\\*ConnectionMongoDBRepository.CountByProduct`** - implementation changed
349. **`.\\*ConnectionMongoDBRepository.buildQueryFilter`** - implementation changed
350. **`.NewConnectionMongoDBRepository`** - parameters changed, implementation changed, signature\\_changed
351. **`.\\*ConnectionMongoDBRepository.Create`** - implementation changed
352. **`.\\*ConnectionMongoDBRepository.Update`** - implementation changed
353. **`.\\*ConnectionMongoDBRepository.Delete`** - implementation changed
354. **`.\\*ConnectionMongoDBRepository.FindByConfigNames`** - implementation changed
355. **`.\\*ConnectionMongoDBRepository.List`** - implementation changed
356. **`.TestConnectionMongoDBRepository\\_Create`** - implementation changed
357. **`.newConnectionRepository`** - implementation changed
358. **`.clearConnectionsCollection`** - implementation changed
359. **`.TestConnectionMongoDBRepository\\_Update`** - implementation changed
360. **`.TestConnectionMongoDBRepository\\_FindByID`** - implementation changed
361. **`.TestConnectionMongoDBRepository\\_EnsureIndexes`** - implementation changed
362. **`.TestConnectionMongoDBRepository\\_Delete`** - implementation changed
363. **`.TestConnectionMongoDBRepository\\_FindByOrganizationAndDatabaseName`** - implementation changed
364. **`.TestConnectionMongoDBRepository\\_FindByConfigNames`** - implementation changed
365. **`.TestConnectionMongoDBRepository\\_DropIndexes`** - implementation changed
366. **`.TestConnectionMongoDBRepository\\_List`** - implementation changed
367. **`.TestMain`** - implementation changed
368. **`.stubConnectionSpanAttributes`** - implementation changed
369. **`.TestConnectionMongoDBRepository\\_FindByOrganizationAndName`** - implementation changed
370. **`.\\*ConnectionMongoDBRepository.EnsureIndexes`** - implementation changed
371. **`.\\*ConnectionMongoDBRepository.DropIndexes`** - implementation changed
372. **`.NewDataSourceRepository`** - parameters changed, implementation changed, signature\\_changed
373. **`.\\*ExternalDataSource.Query`** - implementation changed
374. **`.\\*ExternalDataSource.QueryWithAdvancedFilters`** - implementation changed
375. **`.\\*ExternalDataSource.GetDatabaseSchema`** - implementation changed
376. **`.\\*ExternalDataSource.processQueryResults`** - parameters changed, implementation changed, signature\\_changed
377. **`.\\*ExternalDataSource.CloseConnection`** - implementation changed
378. **`.testLogger`** - implementation changed
379. **`.\\*JobMongoDBRepository.EnsureIndexes`** - implementation changed
380. **`.\\*JobMongoDBRepository.DropIndexes`** - implementation changed
381. **`.\\*JobMongoDBRepository.Create`** - implementation changed
382. **`.\\*JobMongoDBRepository.UpdateStatus`** - implementation changed
383. **`.\\*JobMongoDBRepository.FindByRequestHashWithinWindow`** - implementation changed
384. **`.NewJobMongoDBRepository`** - parameters changed, implementation changed, signature\\_changed
385. **`.\\*JobMongoDBRepository.Update`** - implementation changed
386. **`.\\*JobMongoDBRepository.ExistsRunningByMappedFieldKey`** - implementation changed
387. **`.\\*JobMongoDBRepository.scanJobs`** - parameters changed, signature\\_changed
388. **`.\\*JobMongoDBRepository.FindByID`** - implementation changed
389. **`.\\*JobMongoDBRepository.List`** - implementation changed
390. **`.TestJobMongoDBRepository\\_List`** - implementation changed
391. **`.TestJobMongoDBRepository\\_ExistsRunningByMappedFieldKey`** - implementation changed
392. **`.TestJobMongoDBRepository\\_Update`** - implementation changed
393. **`.TestJobMongoDBRepository\\_FindByID`** - implementation changed
394. **`.TestJobMongoDBRepository\\_UpdateStatus`** - implementation changed
395. **`.TestDropIndexesDatabaseError`** - implementation changed
396. **`.stubJobSpanAttributes`** - implementation changed
397. **`.TestJobMongoDBRepository\\_FindByRequestHashWithinWindow`** - implementation changed
398. **`.TestRepositoryConstructorValidatesDB`** - implementation changed
399. **`.TestMain`** - implementation changed
400. **`.newJobRepository`** - implementation changed
401. **`.TestEnsureIndexesDatabaseError`** - implementation changed
402. **`.clearJobsCollection`** - implementation changed
403. **`.TestJobMongoDBRepository\\_Create`** - implementation changed
404. **`.MapMongoErrorToResponse`** - implementation changed
405. **`.PingMongo`** - implementation changed
406. **`.TestPingMongo`** - implementation changed
407. **`.\\*ProductMongoDBRepository.EnsureIndexes`** - implementation changed
408. **`.\\*ProductMongoDBRepository.DropIndexes`** - implementation changed
409. **`.\\*ProductMongoDBRepository.List`** - implementation changed
410. **`.\\*ProductMongoDBRepository.buildQueryFilter`** - implementation changed
411. **`.NewProductMongoDBRepository`** - parameters changed, implementation changed, signature\\_changed
412. **`.\\*ProductMongoDBRepository.Create`** - implementation changed
413. **`.\\*ProductMongoDBRepository.Update`** - implementation changed
414. **`.\\*ProductMongoDBRepository.Delete`** - implementation changed
415. **`.\\*ProductMongoDBRepository.FindByID`** - implementation changed
416. **`.\\*ProductMongoDBRepository.FindByCode`** - implementation changed
417. **`.parseJSONField`** - parameters changed, implementation changed, signature\\_changed
418. **`.\\*ExternalDataSource.GetDatabaseSchema`** - implementation changed
419. **`.\\*ExternalDataSource.buildSchema`** - parameters changed, signature\\_changed
420. **`.\\*ExternalDataSource.CloseConnection`** - implementation changed
421. **`.\\*ExternalDataSource.scanColumns`** - parameters changed, implementation changed, signature\\_changed
422. **`.NewDataSourceRepository`** - implementation changed
423. **`.\\*ExternalDataSource.Query`** - implementation changed
424. **`.\\*ExternalDataSource.buildTableSchema`** - parameters changed, signature\\_changed
425. **`.scanRows`** - parameters changed, signature\\_changed
426. **`.createRowMap`** - parameters changed, signature\\_changed
427. **`.\\*ExternalDataSource.ValidateTableAndFields`** - implementation changed
428. **`.\\*ExternalDataSource.QueryWithAdvancedFilters`** - implementation changed
429. **`.testContext`** - implementation changed
430. **`.\\*Connection.GetDB`** - implementation changed
431. **`.\\*Connection.Connect`** - implementation changed
432. **`.TestQueryHeaderMetadata`** - implementation changed
433. **`.TestValidateParameters`** - implementation changed
434. **`.TestValidateParametersNonMetadataKeys`** - implementation changed
435. **`.WithRecover`** - implementation changed
436. **`.\\*ExternalDataSource.CloseConnection`** - implementation changed
437. **`.\\*ExternalDataSource.GetDatabaseSchema`** - implementation changed
438. **`.\\*ExternalDataSource.buildSchema`** - parameters changed, signature\\_changed
439. **`.\\*ExternalDataSource.ValidateTableAndFields`** - implementation changed
440. **`.createRowMap`** - parameters changed, signature\\_changed
441. **`.\\*ExternalDataSource.Query`** - implementation changed
442. **`.\\*ExternalDataSource.scanColumns`** - parameters changed, implementation changed, signature\\_changed
443. **`.\\*ExternalDataSource.QueryWithAdvancedFilters`** - implementation changed
444. **`.NewDataSourceRepository`** - implementation changed
445. **`.scanRows`** - parameters changed, signature\\_changed
446. **`.parseJSONField`** - parameters changed, implementation changed, signature\\_changed
447. **`.\\*ExternalDataSource.buildTableSchema`** - parameters changed, signature\\_changed
448. **`.\\*Connection.Connect`** - implementation changed
449. **`.\\*Connection.GetDB`** - implementation changed
450. **`.\\*ExternalDataSource.Query`** - implementation changed
451. **`.parseJSONBField`** - parameters changed, implementation changed, signature\\_changed
452. **`.\\*ExternalDataSource.buildSchema`** - parameters changed, signature\\_changed
453. **`.\\*ExternalDataSource.scanColumns`** - parameters changed, implementation changed, signature\\_changed
454. **`.createRowMap`** - parameters changed, signature\\_changed
455. **`.\\*ExternalDataSource.GetDatabaseSchema`** - implementation changed
456. **`.\\*ExternalDataSource.CloseConnection`** - implementation changed
457. **`.\\*ExternalDataSource.buildTableSchema`** - parameters changed, signature\\_changed
458. **`.scanRows`** - parameters changed, signature\\_changed
459. **`.\\*ExternalDataSource.ValidateTableAndFields`** - implementation changed
460. **`.\\*ExternalDataSource.QueryWithAdvancedFilters`** - implementation changed
461. **`.NewDataSourceRepository`** - implementation changed
462. **`.testContext`** - implementation changed
463. **`.testLogger`** - implementation changed
464. **`.\\*Connection.Connect`** - implementation changed
465. **`.\\*Connection.GetDB`** - implementation changed
466. **`.\\*RabbitMQAdapter.verifyMessageSignature`** - parameters changed, implementation changed, signature\\_changed
467. **`.\\*RabbitMQAdapter.ensureChannel`** - parameters changed, implementation changed, signature\\_changed
468. **`.\\*RabbitMQAdapter.processDelivery`** - implementation changed
469. **`.\\*RabbitMQAdapter.startChannelWatcher`** - implementation changed
470. **`.\\*RabbitMQAdapter.initMetrics`** - implementation changed
471. **`.\\*RabbitMQAdapter.ConsumerLoop`** - implementation changed
472. **`.\\*RabbitMQAdapter.Shutdown`** - implementation changed
473. **`.\\*RabbitMQAdapter.ProducerDefault`** - implementation changed
474. **`.\\*RabbitMQAdapter.dispatchDeliveries`** - implementation changed
475. **`.NewRabbitMQAdapterWithOptions`** - implementation changed
476. **`.\\*RabbitMQAdapter.invalidateChannel`** - implementation changed
477. **`.\\*RabbitMQAdapter.runConsumerCycle`** - implementation changed
478. **`.TestRabbitMQStressProducerAndConsumer`** - implementation changed
479. **`.TestRabbitMQAdapter\\_ConsumerLoop\\_NacksOnVersionMismatch`** - implementation changed
480. **`.TestVerifyMessageSignature\\_TimestampAsInt64`** - implementation changed
481. **`.TestRabbitMQAdapter\\_InvalidateChannel\\_WithNilChannel`** - implementation changed
482. **`.TestRabbitMQAdapter\\_EnsureChannel\\_ReturnsExistingChannel`** - implementation changed
483. **`.TestVerifyMessageSignature\\_MissingSignatureHeader`** - implementation changed
484. **`.TestVerifyMessageSignature\\_NonStringVersion`** - implementation changed
485. **`.TestRabbitMQAdapter\\_StartChannelWatcher\\_HandlesNilChannel`** - implementation changed
486. **`.TestVerifyMessageSignature\\_MissingVersionHeader`** - implementation changed
487. **`.TestVerifyMessageSignature\\_UnsupportedTimestampType`** - implementation changed
488. **`.TestRabbitMQAdapter\\_InvalidateChannel\\_ClosesAndNullifiesChannel`** - implementation changed
489. **`.TestRabbitMQAdapter\\_StartChannelWatcher\\_NullifiesChannelOnClose`** - implementation changed
490. **`.TestRabbitMQAdapter\\_StartChannelWatcher\\_HandlesNilError`** - implementation changed
491. **`.TestVerifyMessageSignature\\_MissingTimestampHeader`** - implementation changed
492. **`.testContextWithHeader`** - implementation changed
493. **`.TestRabbitMQAdapter\\_EnsureChannel\\_ReconnectsWhenChannelClosed`** - implementation changed
494. **`.TestRabbitMQAdapter\\_ConsumerLoop\\_VerifiesSignatureSuccessfully`** - implementation changed
495. **`.TestVerifyMessageSignature\\_InvalidTimestampFormat`** - implementation changed
496. **`.TestRabbitMQAdapter\\_ConsumerLoop\\_NacksOnInvalidSignature`** - implementation changed
497. **`.TestRabbitMQAdapter\\_EnsureChannel\\_RetriesOnFailure`** - implementation changed
498. **`.TestVerifyMessageSignature\\_NonStringSignature`** - implementation changed
499. **`.TestVerifyMessageSignature\\_TimestampAsInt`** - implementation changed
500. **`.NewCacheWithFallback`** - parameters changed, implementation changed, signature\\_changed
501. **`.MustNewCacheWithFallback`** - parameters changed, signature\\_changed
502. **`.TestNewCacheWithFallback\\_ZeroTTL\\_UsesDefault`** - implementation changed
503. **`.TestNewCacheWithFallback\\_RedisAvailable\\_ReturnsFallbackCache`** - implementation changed
504. **`.TestMustNewCacheWithFallback\\_RedisUnavailable\\_ReturnsMemoryOnlyCache`** - implementation changed
505. **`.TestMustNewCacheWithFallback\\_ZeroTTL\\_UsesDefault`** - implementation changed
506. **`.TestNewCacheWithFallback\\_NegativeTTL\\_UsesDefault`** - implementation changed
507. **`.TestNewCacheWithFallback\\_RedisUnavailable\\_ReturnsMemoryOnlyCache`** - implementation changed
508. **`.\\*FallbackCache\\[T\\].Set`** - implementation changed
509. **`.\\*FallbackCache\\[T\\].Delete`** - implementation changed
510. **`.\\*FallbackCache\\[T\\].Clear`** - implementation changed
511. **`.\\*FallbackCache\\[T\\].Close`** - implementation changed
512. **`.\\*FallbackCache\\[T\\].monitorRedisHealth`** - implementation changed
513. **`.NewFallbackCache`** - parameters changed, implementation changed, signature\\_changed
514. **`.\\*FallbackCache\\[T\\].Get`** - implementation changed
515. **`.\\*InMemoryCache\\[T\\].cleanupExpired`** - implementation changed
516. **`.NewInMemoryCache`** - parameters changed, signature\\_changed
517. **`.\\*InMemoryCache\\[T\\].Get`** - implementation changed
518. **`.\\*InMemoryCache\\[T\\].Set`** - implementation changed
519. **`.NewRedisConnection`** - parameters changed, implementation changed, signature\\_changed
520. **`.\\*RedisConnection.Close`** - implementation changed
521. **`.\\*RedisCache\\[T\\].Set`** - implementation changed
522. **`.\\*RedisCache\\[T\\].Delete`** - implementation changed
523. **`.\\*RedisCache\\[T\\].Clear`** - implementation changed
524. **`.\\*RedisCache\\[T\\].Get`** - implementation changed
525. **`.\\*SimpleRepository.Put`** - implementation changed
526. **`.\\*ExternalDataSource.buildSchema`** - parameters changed, signature\\_changed
527. **`.\\*ExternalDataSource.QueryWithAdvancedFilters`** - implementation changed
528. **`.\\*ExternalDataSource.ValidateTableAndFields`** - implementation changed
529. **`.\\*ExternalDataSource.Query`** - implementation changed
530. **`.\\*ExternalDataSource.scanColumns`** - parameters changed, implementation changed, signature\\_changed
531. **`.\\*ExternalDataSource.buildTableSchema`** - parameters changed, signature\\_changed
532. **`.scanRows`** - parameters changed, signature\\_changed
533. **`.NewDataSourceRepository`** - implementation changed
534. **`.createRowMap`** - parameters changed, signature\\_changed
535. **`.\\*ExternalDataSource.CloseConnection`** - implementation changed
536. **`.\\*ExternalDataSource.GetDatabaseSchema`** - implementation changed
537. **`.parseJSONField`** - parameters changed, implementation changed, signature\\_changed
538. **`.testContext`** - implementation changed
539. **`.\\*Connection.Connect`** - implementation changed
540. **`.\\*Connection.GetDB`** - implementation changed
541. **`.\\*MockLogger.Sync`** - parameters changed, signature\\_changed
542. **`.buildProxiedAppEnv`** - implementation changed
543. **`.BuildAppEnv`** - implementation changed
544. **`.\\*AppEnv.ManagerEnv`** - implementation changed
545. **`.RequireNoError`** - parameters changed, signature\\_changed
546. **`.AssertJobCompleted`** - implementation changed
547. **`.checkStatus`** - implementation changed
548. **`.ListConnectionsParams.toQueryString`** - implementation changed
549. **`.isFixedPortEnabled`** - implementation changed
550. **`.definitionsPath`** - implementation changed


## Focus Areas

Based on analysis, pay special attention to:

1. **High-Impact Functions** - 112 functions with 3+ callers modified
2. **New Functions** - 125 new functions added - verify business requirements

