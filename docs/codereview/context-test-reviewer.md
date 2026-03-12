# Pre-Analysis Context: Testing

## Test Coverage for Modified Code



> Warning: call graph analysis is partial.



**Call Graph Warnings:**

- Truncated modified functions from 675 to 500



| Function | File | Tests | Status |
|----------|------|-------|--------|
| `\*ConnectionHandler.ListConnections` | components/manager/internal/adapters/http/in/connection.go | 0 tests | No tests |
| `\*ConnectionHandler.GetConnectionSchema` | components/manager/internal/adapters/http/in/connection.go | 0 tests | No tests |
| `\*ConnectionHandler.CreateConnection` | components/manager/internal/adapters/http/in/connection.go | 0 tests | No tests |
| `\*ConnectionHandler.GetConnection` | components/manager/internal/adapters/http/in/connection.go | 0 tests | No tests |
| `\*ConnectionHandler.TestConnection` | components/manager/internal/adapters/http/in/connection.go | 0 tests | No tests |
| `\*ConnectionHandler.UpdateConnection` | components/manager/internal/adapters/http/in/connection.go | 0 tests | No tests |
| `\*ConnectionHandler.DeleteConnection` | components/manager/internal/adapters/http/in/connection.go | 0 tests | No tests |
| `\*ConnectionHandler.ValidateSchema` | components/manager/internal/adapters/http/in/connection.go | 0 tests | No tests |
| `setupConnectionTestApp` | components/manager/internal/adapters/http/in/connection\_test.go | 55 tests | 55 tests |
| `\*FetcherHandler.GetJob` | components/manager/internal/adapters/http/in/fetcher.go | 0 tests | No tests |
| `\*FetcherHandler.CreateJob` | components/manager/internal/adapters/http/in/fetcher.go | 0 tests | No tests |
| `setupTestApp` | components/manager/internal/adapters/http/in/fetcher\_test.go | 21 tests | 21 tests |
| `TestFetcherHandler\_CreateJob\_MetadataSourceValidation` | components/manager/internal/adapters/http/in/fetcher\_test.go | 8 tests | 8 tests |
| `setupMiddlewareTestApp` | components/manager/internal/adapters/http/in/middlewares\_test.go | 14 tests | 14 tests |
| `\*MigrationHandler.AssignConnectionToProduct` | components/manager/internal/adapters/http/in/migration.go | 0 tests | No tests |
| `\*MigrationHandler.ListUnassignedConnections` | components/manager/internal/adapters/http/in/migration.go | 0 tests | No tests |
| `\*ProductHandler.CreateProduct` | components/manager/internal/adapters/http/in/product.go | 0 tests | No tests |
| `\*ProductHandler.ListProducts` | components/manager/internal/adapters/http/in/product.go | 0 tests | No tests |
| `\*ProductHandler.GetProduct` | components/manager/internal/adapters/http/in/product.go | 0 tests | No tests |
| `\*ProductHandler.UpdateProduct` | components/manager/internal/adapters/http/in/product.go | 0 tests | No tests |
| `\*ProductHandler.DeleteProduct` | components/manager/internal/adapters/http/in/product.go | 0 tests | No tests |
| `NewRoutes` | components/manager/internal/adapters/http/in/routes.go | 0 tests | No tests |
| `TestNewRoutes\_RegistersCriticalStaticRoutesBeforeIDRoutes` | components/manager/internal/adapters/http/in/routes\_test.go | 8 tests | 8 tests |
| `indexOfRoute` | components/manager/internal/adapters/http/in/routes\_test.go | 7 tests | 7 tests |
| `InitServers` | components/manager/internal/bootstrap/config.go | 0 tests | No tests |
| `initLoggerAndTelemetry` | components/manager/internal/bootstrap/config.go | 0 tests | No tests |
| `initPlatformDependencies` | components/manager/internal/bootstrap/config.go | 0 tests | No tests |
| `assembleService` | components/manager/internal/bootstrap/config.go | 0 tests | No tests |
| `buildMongoSource` | components/manager/internal/bootstrap/config.go | 0 tests | No tests |
| `buildRabbitMQSource` | components/manager/internal/bootstrap/config.go | 0 tests | No tests |
| `resolveZapEnvironment` | components/manager/internal/bootstrap/config.go | 0 tests | No tests |
| `loadConfig` | components/manager/internal/bootstrap/config.go | 0 tests | No tests |
| `initMongoRepositories` | components/manager/internal/bootstrap/config.go | 0 tests | No tests |
| `initCrypto` | components/manager/internal/bootstrap/config.go | 0 tests | No tests |
| `must` | components/manager/internal/bootstrap/config.go | 2 tests | 2 tests |
| `TestGetSchemaCacheTTL` | components/manager/internal/bootstrap/config\_test.go | 8 tests | 8 tests |
| `TestGetRedisDB` | components/manager/internal/bootstrap/config\_test.go | 8 tests | 8 tests |
| `TestResolveZapEnvironment` | components/manager/internal/bootstrap/config\_test.go | 8 tests | 8 tests |
| `\*Server.Run` | components/manager/internal/bootstrap/server.go | 0 tests | No tests |
| `\*AssignConnection.Execute` | components/manager/internal/services/command/assign\_connection.go | 0 tests | No tests |
| `\*CreateConnection.Execute` | components/manager/internal/services/command/create\_connection.go | 0 tests | No tests |
| `NewCreateFetcherJob` | components/manager/internal/services/command/create\_fetcher\_job.go | 0 tests | No tests |
| `NewCreateFetcherJobWithTester` | components/manager/internal/services/command/create\_fetcher\_job.go | 8 tests | 8 tests |
| `\*CreateFetcherJob.Execute` | components/manager/internal/services/command/create\_fetcher\_job.go | 0 tests | No tests |
| `\*CreateFetcherJob.validateProductOwnership` | components/manager/internal/services/command/create\_fetcher\_job.go | 0 tests | No tests |
| `\*CreateFetcherJob.TestConnection` | components/manager/internal/services/command/create\_fetcher\_job.go | 0 tests | No tests |
| `TestCreateFetcherJob\_Execute\_PublishFailureMarksJobFailed` | components/manager/internal/services/command/create\_fetcher\_job\_test.go | 8 tests | 8 tests |
| `\*CreateProduct.Execute` | components/manager/internal/services/command/create\_product.go | 0 tests | No tests |
| `\*DeleteProduct.Execute` | components/manager/internal/services/command/delete\_product.go | 0 tests | No tests |
| `testContext` | components/manager/internal/services/command/helpers\_test.go | 97 tests | 97 tests |
| `\*UpdateConnection.Execute` | components/manager/internal/services/command/update\_connection.go | 0 tests | No tests |
| `\*UpdateProduct.Execute` | components/manager/internal/services/command/update\_product.go | 0 tests | No tests |
| `\*GetConnectionSchema.Execute` | components/manager/internal/services/query/get\_connection\_schema.go | 0 tests | No tests |
| `testContext` | components/manager/internal/services/query/helpers\_test.go | 97 tests | 97 tests |
| `\*ListConnections.Execute` | components/manager/internal/services/query/list\_connections.go | 0 tests | No tests |
| `\*ListProducts.Execute` | components/manager/internal/services/query/list\_products.go | 0 tests | No tests |
| `\*ListUnassignedConnections.Execute` | components/manager/internal/services/query/list\_unassigned\_connections.go | 0 tests | No tests |
| `\*TestConnection.Execute` | components/manager/internal/services/query/test\_connection.go | 0 tests | No tests |
| `NewTestConnection` | components/manager/internal/services/query/test\_connection.go | 16 tests | 16 tests |
| `TestTestConnection\_Execute\_WithSSLConfiguration` | components/manager/internal/services/query/test\_connection\_test.go | 8 tests | 8 tests |
| `TestTestConnection\_Execute\_AllDatabaseTypes` | components/manager/internal/services/query/test\_connection\_test.go | 8 tests | 8 tests |
| `TestTestConnection\_Execute\_Success` | components/manager/internal/services/query/test\_connection\_test.go | 8 tests | 8 tests |
| `NewValidateSchema` | components/manager/internal/services/query/validate\_schema.go | 20 tests | 20 tests |
| `\*ValidateSchema.Execute` | components/manager/internal/services/query/validate\_schema.go | 0 tests | No tests |
| `\*ValidateSchema.getOrFetchSchema` | components/manager/internal/services/query/validate\_schema.go | 0 tests | No tests |
| `TestValidateSchema\_CacheSetError` | components/manager/internal/services/query/validate\_schema\_test.go | 8 tests | 8 tests |
| `TestValidateSchema\_CacheError\_ContinuesToFetch` | components/manager/internal/services/query/validate\_schema\_test.go | 8 tests | 8 tests |
| `TestValidateSchema\_NilSchemaFromDatasource` | components/manager/internal/services/query/validate\_schema\_test.go | 8 tests | 8 tests |
| `\*ConsumerRoutes.Shutdown` | components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go | 0 tests | No tests |
| `NewConsumerRoutes` | components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go | 0 tests | No tests |
| `NewConsumerRoutesWithAdapter` | components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go | 3 tests | 3 tests |
| `\*ConsumerRoutes.RunConsumers` | components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go | 0 tests | No tests |
| `TestNewConsumerRoutes` | components/worker/internal/adapters/rabbitmq/consumer\_test.go | 8 tests | 8 tests |
| `TestConsumerRoutes\_Shutdown` | components/worker/internal/adapters/rabbitmq/consumer\_test.go | 8 tests | 8 tests |
| `TestConsumerRoutes\_Shutdown\_Error` | components/worker/internal/adapters/rabbitmq/consumer\_test.go | 8 tests | 8 tests |
| `TestConsumerRoutes\_RunConsumers` | components/worker/internal/adapters/rabbitmq/consumer\_test.go | 8 tests | 8 tests |
| `NewPublisherRoutes` | components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go | 0 tests | No tests |
| `NewPublisherRoutesWithAdapter` | components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go | 5 tests | 5 tests |
| `\*PublisherRoutes.Publish` | components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go | 0 tests | No tests |
| `\*PublisherRoutes.Shutdown` | components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go | 0 tests | No tests |
| `TestPublisherRoutes\_Shutdown` | components/worker/internal/adapters/rabbitmq/publisher\_test.go | 8 tests | 8 tests |
| `TestNewPublisherRoutes` | components/worker/internal/adapters/rabbitmq/publisher\_test.go | 8 tests | 8 tests |
| `TestPublisherRoutes\_Publish` | components/worker/internal/adapters/rabbitmq/publisher\_test.go | 8 tests | 8 tests |
| `InitWorker` | components/worker/internal/bootstrap/config.go | 3 tests | 3 tests |
| `resolveZapEnvironment` | components/worker/internal/bootstrap/config.go | 0 tests | No tests |
| `must` | components/worker/internal/bootstrap/config.go | 0 tests | No tests |
| `TestInitWorker\_PanicsWhenTelemetryGlobalsFail` | components/worker/internal/bootstrap/config\_test.go | 8 tests | 8 tests |
| `testBootstrapLogger` | components/worker/internal/bootstrap/config\_test.go | 6 tests | 6 tests |
| `TestResolveZapEnvironment` | components/worker/internal/bootstrap/config\_test.go | 8 tests | 8 tests |
| `TestMust` | components/worker/internal/bootstrap/config\_test.go | 8 tests | 8 tests |
| `TestInitWorker\_PanicsWhenConfigLoadFails` | components/worker/internal/bootstrap/config\_test.go | 8 tests | 8 tests |
| `TestInitWorker\_PanicsWhenLoggerInitFails` | components/worker/internal/bootstrap/config\_test.go | 8 tests | 8 tests |
| `\*MultiQueueConsumer.Run` | components/worker/internal/bootstrap/consumer.go | 0 tests | No tests |
| `\*MultiQueueConsumer.handlerGenerateReport` | components/worker/internal/bootstrap/consumer.go | 0 tests | No tests |
| `\*Service.Run` | components/worker/internal/bootstrap/service.go | 0 tests | No tests |
| `\*UseCase.ExtractExternalData` | components/worker/internal/services/extract-data.go | 0 tests | No tests |
| `\*UseCase.saveExternalDataToSeaweedFS` | components/worker/internal/services/extract-data.go | 0 tests | No tests |
| `\*UseCase.shouldSkipProcessing` | components/worker/internal/services/extract-data.go | 0 tests | No tests |
| `\*UseCase.parseMessage` | components/worker/internal/services/extract-data.go | 0 tests | No tests |
| `\*UseCase.extractJobIDFromMultipleSources` | components/worker/internal/services/extract-data.go | 0 tests | No tests |
| `\*UseCase.extractJobIDFromPartialJSON` | components/worker/internal/services/extract-data.go | 0 tests | No tests |
| `\*UseCase.queryDatabase` | components/worker/internal/services/extract-data.go | 0 tests | No tests |
| `\*UseCase.handleErrorWithUpdate` | components/worker/internal/services/extract-data.go | 0 tests | No tests |
| `\*UseCase.encryptDataForSeaweedFS` | components/worker/internal/services/extract-data.go | 0 tests | No tests |
| `\*UseCase.checkReportStatus` | components/worker/internal/services/extract-data.go | 0 tests | No tests |
| `\*UseCase.transformPluginCRMAdvancedFilters` | components/worker/internal/services/extract\_crm\_data.go | 0 tests | No tests |
| `\*UseCase.decryptPluginCRMData` | components/worker/internal/services/extract\_crm\_data.go | 0 tests | No tests |
| `\*UseCase.QueryPluginCRM` | components/worker/internal/services/extract\_crm\_data.go | 0 tests | No tests |
| `\*UseCase.queryPluginCRMCollectionWithFilters` | components/worker/internal/services/extract\_crm\_data.go | 0 tests | No tests |
| `\*UseCase.processPluginCRMCollection` | components/worker/internal/services/extract\_crm\_data.go | 0 tests | No tests |
| `TestQueryPluginCRMCollectionWithFilters\_NoFilters` | components/worker/internal/services/extract\_crm\_data\_test.go | 8 tests | 8 tests |
| `TestQueryPluginCRM\_WithOrganizationOnly` | components/worker/internal/services/extract\_crm\_data\_test.go | 8 tests | 8 tests |
| `TestQueryPluginCRM\_WithFilters` | components/worker/internal/services/extract\_crm\_data\_test.go | 8 tests | 8 tests |
| `TestProcessPluginCRMCollection\_WithValidOrganization` | components/worker/internal/services/extract\_crm\_data\_test.go | 8 tests | 8 tests |
| `TestProcessPluginCRMCollection\_WithOrganizationID` | components/worker/internal/services/extract\_crm\_data\_test.go | 8 tests | 8 tests |
| `TestExtractJobIDFromMultipleSources\_EdgeCases` | components/worker/internal/services/extract\_data\_test.go | 8 tests | 8 tests |
| `TestExtractJobIDFromMultipleSources\_FromHeaders` | components/worker/internal/services/extract\_data\_test.go | 8 tests | 8 tests |
| `TestExtractJobIDFromPartialJSON` | components/worker/internal/services/extract\_data\_test.go | 8 tests | 8 tests |
| `TestExtractJobIDFromMultipleSources\_FromPartialJSON` | components/worker/internal/services/extract\_data\_test.go | 8 tests | 8 tests |
| `TestExtractJobIDFromMultipleSources\_NoIDs` | components/worker/internal/services/extract\_data\_test.go | 8 tests | 8 tests |
| `TestExtractJobIDFromPartialJSON\_ValidJobIDInvalidOrgID` | components/worker/internal/services/extract\_data\_test.go | 8 tests | 8 tests |
| `TestExtractJobIDFromPartialJSON\_RegexFallback` | components/worker/internal/services/extract\_data\_test.go | 8 tests | 8 tests |
| `TestExtractJobIDFromMultipleSources\_HeaderPrecedence` | components/worker/internal/services/extract\_data\_test.go | 8 tests | 8 tests |
| `TestParseMessage\_JSONNull` | components/worker/internal/services/extract\_data\_test.go | 8 tests | 8 tests |
| `\*UseCase.publishJobNotification` | components/worker/internal/services/job\_notification.go | 0 tests | No tests |
| `testLogger` | components/worker/internal/services/test\_helpers\_test.go | 79 tests | 79 tests |
| `NewLoggerFromContext` | pkg/context.go | 0 tests | No tests |
| `TestCustomContextKey\_Type` | pkg/context\_test.go | 8 tests | 8 tests |
| `TestNewLoggerFromContext` | pkg/context\_test.go | 8 tests | 8 tests |
| `TestContextWithLogger` | pkg/context\_test.go | 8 tests | 8 tests |
| `TestContextWithTracer` | pkg/context\_test.go | 8 tests | 8 tests |
| `TestContextWithLoggerAndTracer\_Combined` | pkg/context\_test.go | 8 tests | 8 tests |
| `TestCustomContextKeyValue\_Integration` | pkg/context\_test.go | 8 tests | 8 tests |
| `newDataSourceConfigMongoDB` | pkg/datasource/datasource-factory.go | 0 tests | No tests |
| `newDataSourceConfigPostgres` | pkg/datasource/datasource-factory.go | 0 tests | No tests |
| `newDataSourceConfigOracle` | pkg/datasource/datasource-factory.go | 0 tests | No tests |
| `newDataSourceConfigMySQL` | pkg/datasource/datasource-factory.go | 0 tests | No tests |
| `newDataSourceConfigSQLServer` | pkg/datasource/datasource-factory.go | 0 tests | No tests |
| `NewDataSourceFromConnectionWithLogger` | pkg/datasource/datasource-factory.go | 0 tests | No tests |
| `NewDataSourceFromConnection` | pkg/datasource/datasource-factory.go | 2 tests | 2 tests |
| `newDataSourceFromConnection` | pkg/datasource/datasource-factory.go | 1 tests | 1 tests |
| `buildImageWithSecrets` | pkg/itestkit/addons/e2ekit/build\_secrets.go | 1 tests | 1 tests |
| `generateImageTag` | pkg/itestkit/addons/e2ekit/build\_secrets.go | 2 tests | 2 tests |
| `resolveBuildSecretSource` | pkg/itestkit/addons/e2ekit/build\_secrets.go | 0 tests | No tests |
| `createBuildSecretTempFile` | pkg/itestkit/addons/e2ekit/build\_secrets.go | 0 tests | No tests |
| `New` | pkg/itestkit/addons/e2ekit/builder.go | 2 tests | 2 tests |
| `WaitLog` | pkg/itestkit/addons/e2ekit/builder.go | 1 tests | 1 tests |
| `WaitPort` | pkg/itestkit/addons/e2ekit/builder.go | 2 tests | 2 tests |
| `\*Builder.WithRewriter` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `WaitHTTP` | pkg/itestkit/addons/e2ekit/builder.go | 1 tests | 1 tests |
| `localhostToHostGatewayRewriter.Rewrite` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `\*Builder.WithImage` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `\*Builder.Run` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `uniqueAppend` | pkg/itestkit/addons/e2ekit/builder.go | 2 tests | 2 tests |
| `rewriteLocalhostForContainer` | pkg/itestkit/addons/e2ekit/builder.go | 1 tests | 1 tests |
| `dumpRecentLogs` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `\*Builder.WithEnv` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `\*Builder.ExposePort` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `\*Builder.DisableDefaultLocalhostRewrite` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `\*Builder.WithLogsOnFailureMaxBytes` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `\*Builder.WithWait` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `cloneMap` | pkg/itestkit/addons/e2ekit/builder.go | 1 tests | 1 tests |
| `WaitRunning` | pkg/itestkit/addons/e2ekit/builder.go | 1 tests | 1 tests |
| `waitHTTP.Configure` | pkg/itestkit/addons/e2ekit/builder.go | 0 tests | No tests |
| `ProjectRoot` | pkg/itestkit/addons/e2ekit/helpers.go | 1 tests | 1 tests |
| `ProjectRootFrom` | pkg/itestkit/addons/e2ekit/helpers.go | 3 tests | 3 tests |
| `\*ChaosAssertions.TimeoutsBelow` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ChaosAssertions.ThroughputAbove` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ChaosAssertions.FailedResults` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ChaosAssertions.P50Below` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ChaosAssertions.MinRequestsReached` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ChaosAssertions.Summary` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ChaosAssertions.AverageLatencyBelow` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ChaosAssertions.FailuresBelow` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ChaosAssertions.SuccessRateAbove` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ChaosAssertions.P99Below` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ChaosAssertions.P95Below` | pkg/itestkit/addons/metricskit/assertions.go | 0 tests | No tests |
| `\*ErrorClassifier.GetCategoryCounts` | pkg/itestkit/addons/metricskit/error\_classifier.go | 0 tests | No tests |
| `\*ChaosMetrics.GetTotalRequests` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.SuccessRate` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.StartTest` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.StartChaos` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.GetErrorCounts` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.ChaosThroughputRPS` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.Percentile` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.EndChaos` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.RecordRequest` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.GetTimeoutRequests` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.SuccessfulThroughputRPS` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.ChaosDuration` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.TestDuration` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.GetFailedRequests` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.GetMinLatency` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.AverageLatency` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.ThroughputRPS` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*ChaosMetrics.EndTest` | pkg/itestkit/addons/metricskit/metrics.go | 0 tests | No tests |
| `\*Reporter.WriteReport` | pkg/itestkit/addons/metricskit/reporter.go | 0 tests | No tests |
| `\*Reporter.String` | pkg/itestkit/addons/metricskit/reporter.go | 0 tests | No tests |
| `\*Reporter.CompactSummary` | pkg/itestkit/addons/metricskit/reporter.go | 0 tests | No tests |
| `\*AMQPConsumerBuilder.WithPrefetch` | pkg/itestkit/addons/queuekit/amqp.go | 0 tests | No tests |
| `\*AMQPConsumer.Close` | pkg/itestkit/addons/queuekit/amqp.go | 0 tests | No tests |
| `\*AMQPPublisher.connect` | pkg/itestkit/addons/queuekit/amqp.go | 0 tests | No tests |
| `\*AMQPPublisher.Close` | pkg/itestkit/addons/queuekit/amqp.go | 0 tests | No tests |
| `\*AMQPConsumerBuilder.BindTo` | pkg/itestkit/addons/queuekit/amqp.go | 0 tests | No tests |
| `\*AMQPConsumerBuilder.WithQueueDeclare` | pkg/itestkit/addons/queuekit/amqp.go | 0 tests | No tests |
| `NewAMQPConsumer` | pkg/itestkit/addons/queuekit/amqp.go | 0 tests | No tests |
| `\*AMQPConsumer.connect` | pkg/itestkit/addons/queuekit/amqp.go | 0 tests | No tests |
| `\*Assertions\[T\].HasHeaderKey` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ResultAssertions\[T\].At` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*MessageSequence\[T\].FilterBy` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ExpectMessagesHelper\[T\].OrFatal` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*Assertions\[T\].PayloadSatisfies` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ResultAssertions\[T\].UnmatchedCount` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*Assertions\[T\].HasRoutingKey` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*Assertions\[T\].HasMessageID` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*Assertions\[T\].HasContentType` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ResultAssertions\[T\].First` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `AssertResult` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `AssertJSONEqual` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*MessageSequence\[T\].GroupBy` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ExpectMessagesHelper\[T\].ToContainWhere` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `AssertMessage` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*Assertions\[T\].HasHeader` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ResultAssertions\[T\].HasAtLeast` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `JSONEqual` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ExpectMessagesHelper\[T\].ToHaveCount` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*Assertions\[T\].HasCorrelationID` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*Assertions\[T\].PayloadEquals` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ResultAssertions\[T\].HasCount` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ResultAssertions\[T\].DidNotTimeout` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ResultAssertions\[T\].HasNoErrors` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ResultAssertions\[T\].All` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*MessageSequence\[T\].RoutingKeysInOrder` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*ExpectMessagesHelper\[T\].ToSucceed` | pkg/itestkit/addons/queuekit/assertions.go | 0 tests | No tests |
| `\*Consumer\[T\].CaptureAll` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `\*ConsumerBuilder\[T\].WithMatcher` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `\*ConsumerBuilder\[T\].WithTimeout` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `\*Consumer\[T\].DrainQueue` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `\*Consumer\[T\].captureMessage` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `NewConsumer` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `truncateBody` | pkg/itestkit/addons/queuekit/consumer.go | 1 tests | 1 tests |
| `\*ConsumerBuilder\[T\].WithUnmarshaler` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `\*ConsumerBuilder\[T\].Build` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `\*Consumer\[T\].WaitForMessages` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `\*Consumer\[T\].GetCaptured` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `\*Consumer\[T\].ClearCaptured` | pkg/itestkit/addons/queuekit/consumer.go | 0 tests | No tests |
| `compareValues` | pkg/itestkit/addons/queuekit/matcher.go | 1 tests | 1 tests |
| `MatchHeader` | pkg/itestkit/addons/queuekit/matcher.go | 1 tests | 1 tests |
| `MatchHeaderExists` | pkg/itestkit/addons/queuekit/matcher.go | 0 tests | No tests |
| `MatchJSONField` | pkg/itestkit/addons/queuekit/matcher.go | 0 tests | No tests |
| `MatchJSONFieldPattern` | pkg/itestkit/addons/queuekit/matcher.go | 0 tests | No tests |
| `MatchBodyPattern` | pkg/itestkit/addons/queuekit/matcher.go | 1 tests | 1 tests |
| `hasNestedValue` | pkg/itestkit/addons/queuekit/matcher.go | 0 tests | No tests |
| `MatchAll` | pkg/itestkit/addons/queuekit/matcher.go | 0 tests | No tests |
| `MatchAny` | pkg/itestkit/addons/queuekit/matcher.go | 0 tests | No tests |
| `MatchRoutingKeyPattern` | pkg/itestkit/addons/queuekit/matcher.go | 0 tests | No tests |
| `applyPublishOptions` | pkg/itestkit/addons/queuekit/queuekit.go | 0 tests | No tests |
| `WaitResult\[T\].First` | pkg/itestkit/addons/queuekit/queuekit.go | 0 tests | No tests |
| `\*toxiproxyChaos.RemoveToxic` | pkg/itestkit/chaos\_toxiproxy.go | 0 tests | No tests |
| `\*toxiproxyChaos.RemoveAllToxics` | pkg/itestkit/chaos\_toxiproxy.go | 0 tests | No tests |
| `\*toxiproxyChaos.Close` | pkg/itestkit/chaos\_toxiproxy.go | 0 tests | No tests |
| `NewToxiproxyChaos` | pkg/itestkit/chaos\_toxiproxy.go | 0 tests | No tests |
| `\*toxiproxyChaos.CreateProxy` | pkg/itestkit/chaos\_toxiproxy.go | 0 tests | No tests |
| `\*toxiproxyChaos.AddBandwidth` | pkg/itestkit/chaos\_toxiproxy.go | 0 tests | No tests |
| `\*toxiproxyChaos.AddLatency` | pkg/itestkit/chaos\_toxiproxy.go | 0 tests | No tests |
| `\*toxiproxyChaos.CutConnection` | pkg/itestkit/chaos\_toxiproxy.go | 0 tests | No tests |
| `\*toxiproxyChaos.AddTimeout` | pkg/itestkit/chaos\_toxiproxy.go | 0 tests | No tests |
| `WaitListeningPort.Apply` | pkg/itestkit/container\_generic.go | 0 tests | No tests |
| `\*Builder.WithContainerCustomize` | pkg/itestkit/container\_generic.go | 0 tests | No tests |
| `\*genericContainerInfra.Start` | pkg/itestkit/container\_generic.go | 0 tests | No tests |
| `\*genericContainerInfra.Terminate` | pkg/itestkit/container\_generic.go | 0 tests | No tests |
| `portKey` | pkg/itestkit/container\_generic.go | 0 tests | No tests |
| `CustomizerFunc.Customize` | pkg/itestkit/customizer.go | 0 tests | No tests |
| `MergeCustomizers` | pkg/itestkit/customizer.go | 0 tests | No tests |
| `CExposedPorts` | pkg/itestkit/customizer\_options.go | 1 tests | 1 tests |
| `CEnvFromOS` | pkg/itestkit/customizer\_options.go | 0 tests | No tests |
| `uniqueAppendMany` | pkg/itestkit/customizer\_options.go | 0 tests | No tests |
| `CBindMount` | pkg/itestkit/customizer\_options.go | 2 tests | 2 tests |
| `CEnvs` | pkg/itestkit/customizer\_options.go | 0 tests | No tests |
| `CHostDockerInternal` | pkg/itestkit/customizer\_options.go | 2 tests | 2 tests |
| `CNetworks` | pkg/itestkit/customizer\_options.go | 1 tests | 1 tests |
| `CAll` | pkg/itestkit/customizer\_options.go | 0 tests | No tests |
| `CNetworkAliases` | pkg/itestkit/customizer\_options.go | 0 tests | No tests |
| `HostGatewayIP` | pkg/itestkit/hostport.go | 0 tests | No tests |
| `ParseHostPort` | pkg/itestkit/hostport.go | 0 tests | No tests |
| `ResolveHostHostPort` | pkg/itestkit/hostport.go | 0 tests | No tests |
| `ResolveContainerHostPort` | pkg/itestkit/hostport.go | 2 tests | 2 tests |
| `validateUniqueInfraNames` | pkg/itestkit/infra.go | 0 tests | No tests |
| `NewMongoDBInfra` | pkg/itestkit/infra/mongodb/mongodb.go | 0 tests | No tests |
| `\*MongoDBInfra.HostPort` | pkg/itestkit/infra/mongodb/mongodb.go | 0 tests | No tests |
| `NewMongoDBInfraStub` | pkg/itestkit/infra/mongodb/mongodb.go | 0 tests | No tests |
| `\*MongoDBInfra.Start` | pkg/itestkit/infra/mongodb/mongodb.go | 0 tests | No tests |
| `\*MongoDBInfra.Endpoint` | pkg/itestkit/infra/mongodb/mongodb.go | 0 tests | No tests |
| `\*MongoDBInfra.URI` | pkg/itestkit/infra/mongodb/mongodb.go | 0 tests | No tests |
| `\*MongoDBInfra.Terminate` | pkg/itestkit/infra/mongodb/mongodb.go | 0 tests | No tests |
| `\*MongoDBInfra.ContainerHostPort` | pkg/itestkit/infra/mongodb/mongodb.go | 0 tests | No tests |
| `WithMongoDBFixedPort` | pkg/itestkit/infra/mongodb/mongodb\_options.go | 0 tests | No tests |
| `\*MSSQLInfra.Endpoint` | pkg/itestkit/infra/mssql/mssql.go | 0 tests | No tests |
| `NewMSSQLInfraStub` | pkg/itestkit/infra/mssql/mssql.go | 0 tests | No tests |
| `NewMSSQLInfra` | pkg/itestkit/infra/mssql/mssql.go | 0 tests | No tests |
| `\*MSSQLInfra.Start` | pkg/itestkit/infra/mssql/mssql.go | 0 tests | No tests |
| `\*MSSQLInfra.DSN` | pkg/itestkit/infra/mssql/mssql.go | 0 tests | No tests |
| `\*MSSQLInfra.HostPort` | pkg/itestkit/infra/mssql/mssql.go | 0 tests | No tests |
| `\*MSSQLInfra.Terminate` | pkg/itestkit/infra/mssql/mssql.go | 0 tests | No tests |
| `WithMSSQLFixedPort` | pkg/itestkit/infra/mssql/mssql\_options.go | 0 tests | No tests |
| `NewMySQLInfra` | pkg/itestkit/infra/mysql/mysql.go | 0 tests | No tests |
| `\*MySQLInfra.Endpoint` | pkg/itestkit/infra/mysql/mysql.go | 0 tests | No tests |
| `\*MySQLInfra.Terminate` | pkg/itestkit/infra/mysql/mysql.go | 0 tests | No tests |
| `\*MySQLInfra.Start` | pkg/itestkit/infra/mysql/mysql.go | 0 tests | No tests |
| `\*MySQLInfra.DSN` | pkg/itestkit/infra/mysql/mysql.go | 0 tests | No tests |
| `\*MySQLInfra.HostPort` | pkg/itestkit/infra/mysql/mysql.go | 0 tests | No tests |
| `NewMySQLInfraStub` | pkg/itestkit/infra/mysql/mysql.go | 0 tests | No tests |
| `WithMySQLInitScript` | pkg/itestkit/infra/mysql/mysql\_options.go | 0 tests | No tests |
| `WithMySQLFixedPort` | pkg/itestkit/infra/mysql/mysql\_options.go | 0 tests | No tests |
| `\*OracleInfra.Endpoint` | pkg/itestkit/infra/oracle/oracle.go | 0 tests | No tests |
| `NewOracleInfraStub` | pkg/itestkit/infra/oracle/oracle.go | 0 tests | No tests |
| `\*OracleInfra.DSN` | pkg/itestkit/infra/oracle/oracle.go | 0 tests | No tests |
| `\*OracleInfra.GoDRORDSN` | pkg/itestkit/infra/oracle/oracle.go | 0 tests | No tests |
| `\*OracleInfra.HostPort` | pkg/itestkit/infra/oracle/oracle.go | 0 tests | No tests |
| `\*OracleInfra.Terminate` | pkg/itestkit/infra/oracle/oracle.go | 0 tests | No tests |
| `NewOracleInfra` | pkg/itestkit/infra/oracle/oracle.go | 0 tests | No tests |
| `\*OracleInfra.Start` | pkg/itestkit/infra/oracle/oracle.go | 0 tests | No tests |
| `WithOracleInitScript` | pkg/itestkit/infra/oracle/oracle\_options.go | 0 tests | No tests |
| `WithOracleFixedPort` | pkg/itestkit/infra/oracle/oracle\_options.go | 0 tests | No tests |
| `\*PostgresInfra.Start` | pkg/itestkit/infra/postgres/postgres.go | 0 tests | No tests |
| `\*PostgresInfra.DSN` | pkg/itestkit/infra/postgres/postgres.go | 0 tests | No tests |
| `\*PostgresInfra.HostPort` | pkg/itestkit/infra/postgres/postgres.go | 0 tests | No tests |
| `NewPostgresInfra` | pkg/itestkit/infra/postgres/postgres.go | 0 tests | No tests |
| `\*PostgresInfra.Endpoint` | pkg/itestkit/infra/postgres/postgres.go | 0 tests | No tests |
| `\*PostgresInfra.Terminate` | pkg/itestkit/infra/postgres/postgres.go | 0 tests | No tests |
| `NewPostgresInfraStub` | pkg/itestkit/infra/postgres/postgres.go | 0 tests | No tests |
| `\*PostgresInfra.ContainerHostPort` | pkg/itestkit/infra/postgres/postgres.go | 0 tests | No tests |
| `WithPGInitFile` | pkg/itestkit/infra/postgres/postgres\_options.go | 0 tests | No tests |
| `WithPGFixedPort` | pkg/itestkit/infra/postgres/postgres\_options.go | 0 tests | No tests |
| `WithRabbitFixedPort` | pkg/itestkit/infra/rabbitmq/rabbit\_options.go | 0 tests | No tests |
| `\*configReader.Read` | pkg/itestkit/infra/rabbitmq/rabbit\_options.go | 0 tests | No tests |
| `NewRabbitInfra` | pkg/itestkit/infra/rabbitmq/rabbitmq.go | 0 tests | No tests |
| `\*RabbitInfra.Start` | pkg/itestkit/infra/rabbitmq/rabbitmq.go | 0 tests | No tests |
| `\*RabbitInfra.Endpoint` | pkg/itestkit/infra/rabbitmq/rabbitmq.go | 0 tests | No tests |
| `\*RabbitInfra.AMQPURL` | pkg/itestkit/infra/rabbitmq/rabbitmq.go | 0 tests | No tests |
| `\*RabbitInfra.HostPort` | pkg/itestkit/infra/rabbitmq/rabbitmq.go | 0 tests | No tests |
| `\*RabbitInfra.Terminate` | pkg/itestkit/infra/rabbitmq/rabbitmq.go | 0 tests | No tests |
| `\*RabbitInfra.ContainerHostPort` | pkg/itestkit/infra/rabbitmq/rabbitmq.go | 0 tests | No tests |
| `NewRedisInfra` | pkg/itestkit/infra/redis/redis.go | 0 tests | No tests |
| `\*RedisInfra.Start` | pkg/itestkit/infra/redis/redis.go | 0 tests | No tests |
| `\*RedisInfra.HostPort` | pkg/itestkit/infra/redis/redis.go | 0 tests | No tests |
| `\*RedisInfra.Terminate` | pkg/itestkit/infra/redis/redis.go | 0 tests | No tests |
| `\*RedisInfra.Endpoint` | pkg/itestkit/infra/redis/redis.go | 0 tests | No tests |
| `\*RedisInfra.URL` | pkg/itestkit/infra/redis/redis.go | 0 tests | No tests |
| `\*RedisInfra.Addr` | pkg/itestkit/infra/redis/redis.go | 0 tests | No tests |
| `\*RedisInfra.ContainerHostPort` | pkg/itestkit/infra/redis/redis.go | 0 tests | No tests |
| `WithRedisFixedPort` | pkg/itestkit/infra/redis/redis\_options.go | 0 tests | No tests |
| `NewSeaweedFSInfra` | pkg/itestkit/infra/seaweedfs/seaweedfs.go | 3 tests | 3 tests |
| `\*SeaweedFSInfra.Start` | pkg/itestkit/infra/seaweedfs/seaweedfs.go | 0 tests | No tests |
| `\*SeaweedFSInfra.Endpoint` | pkg/itestkit/infra/seaweedfs/seaweedfs.go | 0 tests | No tests |
| `\*SeaweedFSInfra.URL` | pkg/itestkit/infra/seaweedfs/seaweedfs.go | 0 tests | No tests |
| `\*SeaweedFSInfra.HostPort` | pkg/itestkit/infra/seaweedfs/seaweedfs.go | 0 tests | No tests |
| `\*SeaweedFSInfra.Terminate` | pkg/itestkit/infra/seaweedfs/seaweedfs.go | 0 tests | No tests |
| `\*SeaweedFSInfra.ContainerHostPort` | pkg/itestkit/infra/seaweedfs/seaweedfs.go | 0 tests | No tests |
| `WithSeaweedFSFixedPort` | pkg/itestkit/infra/seaweedfs/seaweedfs\_options.go | 0 tests | No tests |
| `\*Builder.Build` | pkg/itestkit/suite.go | 0 tests | No tests |
| `\*Suite.Terminate` | pkg/itestkit/suite.go | 0 tests | No tests |
| `New` | pkg/itestkit/suite.go | 0 tests | No tests |
| `\*Builder.WithInfra` | pkg/itestkit/suite.go | 0 tests | No tests |
| `\*Builder.WithInfras` | pkg/itestkit/suite.go | 0 tests | No tests |
| `\*Suite.Network` | pkg/itestkit/suite.go | 0 tests | No tests |
| `\*DataSourceConfigMongoDB.GetSchemaInfo` | pkg/model/datasource/mongodb/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigMongoDB.Connect` | pkg/model/datasource/mongodb/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigMongoDB.Query` | pkg/model/datasource/mongodb/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigMySQL.Connect` | pkg/model/datasource/mysql/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigMySQL.Query` | pkg/model/datasource/mysql/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigMySQL.GetSchemaInfo` | pkg/model/datasource/mysql/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigOracle.Connect` | pkg/model/datasource/oracle/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigOracle.Query` | pkg/model/datasource/oracle/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigOracle.GetSchemaInfo` | pkg/model/datasource/oracle/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigPostgres.Connect` | pkg/model/datasource/postgres/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigPostgres.GetSchemaInfo` | pkg/model/datasource/postgres/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigPostgres.Query` | pkg/model/datasource/postgres/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigSQLServer.Query` | pkg/model/datasource/sqlserver/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigSQLServer.GetSchemaInfo` | pkg/model/datasource/sqlserver/datasource-config.go | 0 tests | No tests |
| `\*DataSourceConfigSQLServer.Connect` | pkg/model/datasource/sqlserver/datasource-config.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.ListUnassigned` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.AssignProduct` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.FindByID` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.FindByOrganizationAndName` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.FindByOrganizationAndDatabaseName` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.CountByProduct` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.buildQueryFilter` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `NewConnectionMongoDBRepository` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.Create` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.Update` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.Delete` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.FindByConfigNames` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.List` | pkg/mongodb/connection/connection.mongodb.go | 0 tests | No tests |
| `\*MockmongoDatabaseProvider.Client` | pkg/mongodb/connection/connection.mongodb.mock.go | 0 tests | No tests |
| `\*MockmongoDatabaseProviderMockRecorder.Client` | pkg/mongodb/connection/connection.mongodb.mock.go | 0 tests | No tests |
| `TestConnectionMongoDBRepository\_Create` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `newConnectionRepository` | pkg/mongodb/connection/connection.mongodb\_test.go | 30 tests | 30 tests |
| `clearConnectionsCollection` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `TestConnectionMongoDBRepository\_Update` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `TestConnectionMongoDBRepository\_FindByID` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `TestConnectionMongoDBRepository\_EnsureIndexes` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `TestConnectionMongoDBRepository\_Delete` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `TestConnectionMongoDBRepository\_FindByOrganizationAndDatabaseName` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `TestConnectionMongoDBRepository\_FindByConfigNames` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `TestConnectionMongoDBRepository\_DropIndexes` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `TestConnectionMongoDBRepository\_List` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `TestMain` | pkg/mongodb/connection/connection.mongodb\_test.go | 0 tests | No tests |
| `stubConnectionSpanAttributes` | pkg/mongodb/connection/connection.mongodb\_test.go | 3 tests | 3 tests |
| `TestConnectionMongoDBRepository\_FindByOrganizationAndName` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `TestNewConnectionMongoDBRepository\_NilClient` | pkg/mongodb/connection/connection.mongodb\_test.go | 8 tests | 8 tests |
| `\*ConnectionMongoDBRepository.EnsureIndexes` | pkg/mongodb/connection/indexes.go | 0 tests | No tests |
| `\*ConnectionMongoDBRepository.DropIndexes` | pkg/mongodb/connection/indexes.go | 0 tests | No tests |
| `NewDataSourceRepository` | pkg/mongodb/datasource.mongodb.go | 2 tests | 2 tests |
| `\*ExternalDataSource.Query` | pkg/mongodb/datasource.mongodb.go | 0 tests | No tests |
| `\*ExternalDataSource.QueryWithAdvancedFilters` | pkg/mongodb/datasource.mongodb.go | 0 tests | No tests |
| `\*ExternalDataSource.GetDatabaseSchema` | pkg/mongodb/datasource.mongodb.go | 0 tests | No tests |
| `\*ExternalDataSource.processQueryResults` | pkg/mongodb/datasource.mongodb.go | 0 tests | No tests |
| `\*ExternalDataSource.CloseConnection` | pkg/mongodb/datasource.mongodb.go | 0 tests | No tests |
| `NewMockDatasource` | pkg/mongodb/datasource.mongodb.mock.go | 0 tests | No tests |
| `\*MockDatasourceMockRecorder.CloseConnection` | pkg/mongodb/datasource.mongodb.mock.go | 0 tests | No tests |
| `\*MockDatasource.GetDatabaseSchema` | pkg/mongodb/datasource.mongodb.mock.go | 0 tests | No tests |
| `\*MockDatasource.Query` | pkg/mongodb/datasource.mongodb.mock.go | 0 tests | No tests |
| `\*MockDatasource.EXPECT` | pkg/mongodb/datasource.mongodb.mock.go | 0 tests | No tests |
| `\*MockDatasource.CloseConnection` | pkg/mongodb/datasource.mongodb.mock.go | 0 tests | No tests |
| `\*MockDatasourceMockRecorder.GetDatabaseSchema` | pkg/mongodb/datasource.mongodb.mock.go | 0 tests | No tests |
| `\*MockDatasourceMockRecorder.Query` | pkg/mongodb/datasource.mongodb.mock.go | 0 tests | No tests |
| `\*MockDatasource.QueryWithAdvancedFilters` | pkg/mongodb/datasource.mongodb.mock.go | 0 tests | No tests |
| `\*MockDatasourceMockRecorder.QueryWithAdvancedFilters` | pkg/mongodb/datasource.mongodb.mock.go | 0 tests | No tests |
| `testLogger` | pkg/mongodb/datasource.mongodb\_test.go | 2 tests | 2 tests |
| `\*JobMongoDBRepository.EnsureIndexes` | pkg/mongodb/job/indexes.go | 0 tests | No tests |
| `\*JobMongoDBRepository.DropIndexes` | pkg/mongodb/job/indexes.go | 0 tests | No tests |
| `\*JobMongoDBRepository.Create` | pkg/mongodb/job/job.mongodb.go | 0 tests | No tests |
| `\*JobMongoDBRepository.UpdateStatus` | pkg/mongodb/job/job.mongodb.go | 0 tests | No tests |
| `\*JobMongoDBRepository.FindByRequestHashWithinWindow` | pkg/mongodb/job/job.mongodb.go | 0 tests | No tests |
| `NewJobMongoDBRepository` | pkg/mongodb/job/job.mongodb.go | 0 tests | No tests |
| `\*JobMongoDBRepository.Update` | pkg/mongodb/job/job.mongodb.go | 0 tests | No tests |
| `\*JobMongoDBRepository.ExistsRunningByMappedFieldKey` | pkg/mongodb/job/job.mongodb.go | 0 tests | No tests |
| `\*JobMongoDBRepository.scanJobs` | pkg/mongodb/job/job.mongodb.go | 0 tests | No tests |
| `\*JobMongoDBRepository.FindByID` | pkg/mongodb/job/job.mongodb.go | 0 tests | No tests |
| `\*JobMongoDBRepository.List` | pkg/mongodb/job/job.mongodb.go | 0 tests | No tests |
| `TestJobMongoDBRepository\_List` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `TestJobMongoDBRepository\_ExistsRunningByMappedFieldKey` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `TestJobMongoDBRepository\_Update` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `TestJobMongoDBRepository\_FindByID` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `TestJobMongoDBRepository\_UpdateStatus` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `TestDropIndexesDatabaseError` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `stubJobSpanAttributes` | pkg/mongodb/job/job.mongodb\_test.go | 4 tests | 4 tests |
| `TestJobMongoDBRepository\_FindByRequestHashWithinWindow` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `TestRepositoryConstructorValidatesDB` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `TestMain` | pkg/mongodb/job/job.mongodb\_test.go | 0 tests | No tests |
| `newJobRepository` | pkg/mongodb/job/job.mongodb\_test.go | 45 tests | 45 tests |
| `TestEnsureIndexesDatabaseError` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `clearJobsCollection` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `TestJobMongoDBRepository\_Create` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `TestNewJobMongoDBRepository\_NilClient` | pkg/mongodb/job/job.mongodb\_test.go | 8 tests | 8 tests |
| `MapMongoErrorToResponse` | pkg/mongodb/mongo.go | 0 tests | No tests |
| `PingMongo` | pkg/mongodb/mongo.go | 4 tests | 4 tests |
| `\*MockMongoClientProvider.Client` | pkg/mongodb/mongo\_client\_provider.mock.go | 0 tests | No tests |
| `\*MockMongoClientProviderMockRecorder.Client` | pkg/mongodb/mongo\_client\_provider.mock.go | 0 tests | No tests |
| `TestPingMongo` | pkg/mongodb/mongo\_test.go | 8 tests | 8 tests |
| `\*ProductMongoDBRepository.EnsureIndexes` | pkg/mongodb/product/indexes.go | 0 tests | No tests |
| `\*ProductMongoDBRepository.DropIndexes` | pkg/mongodb/product/indexes.go | 0 tests | No tests |
| `\*ProductMongoDBRepository.List` | pkg/mongodb/product/product.mongodb.go | 0 tests | No tests |
| `\*ProductMongoDBRepository.buildQueryFilter` | pkg/mongodb/product/product.mongodb.go | 0 tests | No tests |
| `NewProductMongoDBRepository` | pkg/mongodb/product/product.mongodb.go | 1 tests | 1 tests |
| `\*ProductMongoDBRepository.Create` | pkg/mongodb/product/product.mongodb.go | 0 tests | No tests |
| `\*ProductMongoDBRepository.Update` | pkg/mongodb/product/product.mongodb.go | 0 tests | No tests |
| `\*ProductMongoDBRepository.Delete` | pkg/mongodb/product/product.mongodb.go | 0 tests | No tests |
| `\*ProductMongoDBRepository.FindByID` | pkg/mongodb/product/product.mongodb.go | 0 tests | No tests |
| `\*ProductMongoDBRepository.FindByCode` | pkg/mongodb/product/product.mongodb.go | 0 tests | No tests |
| `newProductMongoDBRepository` | pkg/mongodb/product/product.mongodb.go | 1 tests | 1 tests |
| `parseJSONField` | pkg/mysql/datasource.mysql.go | 1 tests | 1 tests |
| `\*ExternalDataSource.GetDatabaseSchema` | pkg/mysql/datasource.mysql.go | 0 tests | No tests |
| `\*ExternalDataSource.buildSchema` | pkg/mysql/datasource.mysql.go | 0 tests | No tests |
| `\*ExternalDataSource.CloseConnection` | pkg/mysql/datasource.mysql.go | 0 tests | No tests |
| `\*ExternalDataSource.scanColumns` | pkg/mysql/datasource.mysql.go | 0 tests | No tests |
| `NewDataSourceRepository` | pkg/mysql/datasource.mysql.go | 0 tests | No tests |
| `\*ExternalDataSource.Query` | pkg/mysql/datasource.mysql.go | 0 tests | No tests |
| `\*ExternalDataSource.buildTableSchema` | pkg/mysql/datasource.mysql.go | 0 tests | No tests |
| `scanRows` | pkg/mysql/datasource.mysql.go | 0 tests | No tests |
| `createRowMap` | pkg/mysql/datasource.mysql.go | 1 tests | 1 tests |
| `\*ExternalDataSource.ValidateTableAndFields` | pkg/mysql/datasource.mysql.go | 0 tests | No tests |
| `\*ExternalDataSource.QueryWithAdvancedFilters` | pkg/mysql/datasource.mysql.go | 0 tests | No tests |
| `testContext` | pkg/mysql/datasource.mysql\_test.go | 8 tests | 8 tests |
| `\*Connection.GetDB` | pkg/mysql/mysql.go | 0 tests | No tests |
| `\*Connection.Connect` | pkg/mysql/mysql.go | 0 tests | No tests |
| `TestQueryHeaderMetadata` | pkg/net/http/http-utils\_test.go | 8 tests | 8 tests |
| `TestValidateParameters` | pkg/net/http/http-utils\_test.go | 8 tests | 8 tests |
| `TestValidateParametersNonMetadataKeys` | pkg/net/http/http-utils\_test.go | 8 tests | 8 tests |
| `WithRecover` | pkg/net/http/with\_recover.go | 0 tests | No tests |
| `\*ExternalDataSource.CloseConnection` | pkg/oracle/datasource.oracle.go | 0 tests | No tests |
| `\*ExternalDataSource.GetDatabaseSchema` | pkg/oracle/datasource.oracle.go | 0 tests | No tests |
| `\*ExternalDataSource.buildSchema` | pkg/oracle/datasource.oracle.go | 0 tests | No tests |
| `\*ExternalDataSource.ValidateTableAndFields` | pkg/oracle/datasource.oracle.go | 0 tests | No tests |
| `createRowMap` | pkg/oracle/datasource.oracle.go | 0 tests | No tests |
| `\*ExternalDataSource.Query` | pkg/oracle/datasource.oracle.go | 0 tests | No tests |
| `\*ExternalDataSource.scanColumns` | pkg/oracle/datasource.oracle.go | 0 tests | No tests |
| `\*ExternalDataSource.QueryWithAdvancedFilters` | pkg/oracle/datasource.oracle.go | 0 tests | No tests |
| `NewDataSourceRepository` | pkg/oracle/datasource.oracle.go | 2 tests | 2 tests |


## Uncovered New Code



- `\*ConnectionHandler.ListConnections` at `components/manager/internal/adapters/http/in/connection.go` - **No tests found**
- `\*ConnectionHandler.GetConnectionSchema` at `components/manager/internal/adapters/http/in/connection.go` - **No tests found**
- `\*ConnectionHandler.CreateConnection` at `components/manager/internal/adapters/http/in/connection.go` - **No tests found**
- `\*ConnectionHandler.GetConnection` at `components/manager/internal/adapters/http/in/connection.go` - **No tests found**
- `\*ConnectionHandler.TestConnection` at `components/manager/internal/adapters/http/in/connection.go` - **No tests found**
- `\*ConnectionHandler.UpdateConnection` at `components/manager/internal/adapters/http/in/connection.go` - **No tests found**
- `\*ConnectionHandler.DeleteConnection` at `components/manager/internal/adapters/http/in/connection.go` - **No tests found**
- `\*ConnectionHandler.ValidateSchema` at `components/manager/internal/adapters/http/in/connection.go` - **No tests found**
- `\*FetcherHandler.GetJob` at `components/manager/internal/adapters/http/in/fetcher.go` - **No tests found**
- `\*FetcherHandler.CreateJob` at `components/manager/internal/adapters/http/in/fetcher.go` - **No tests found**
- `\*MigrationHandler.AssignConnectionToProduct` at `components/manager/internal/adapters/http/in/migration.go` - **No tests found**
- `\*MigrationHandler.ListUnassignedConnections` at `components/manager/internal/adapters/http/in/migration.go` - **No tests found**
- `\*ProductHandler.CreateProduct` at `components/manager/internal/adapters/http/in/product.go` - **No tests found**
- `\*ProductHandler.ListProducts` at `components/manager/internal/adapters/http/in/product.go` - **No tests found**
- `\*ProductHandler.GetProduct` at `components/manager/internal/adapters/http/in/product.go` - **No tests found**
- `\*ProductHandler.UpdateProduct` at `components/manager/internal/adapters/http/in/product.go` - **No tests found**
- `\*ProductHandler.DeleteProduct` at `components/manager/internal/adapters/http/in/product.go` - **No tests found**
- `NewRoutes` at `components/manager/internal/adapters/http/in/routes.go` - **No tests found**
- `InitServers` at `components/manager/internal/bootstrap/config.go` - **No tests found**
- `initLoggerAndTelemetry` at `components/manager/internal/bootstrap/config.go` - **No tests found**
- `initPlatformDependencies` at `components/manager/internal/bootstrap/config.go` - **No tests found**
- `assembleService` at `components/manager/internal/bootstrap/config.go` - **No tests found**
- `buildMongoSource` at `components/manager/internal/bootstrap/config.go` - **No tests found**
- `buildRabbitMQSource` at `components/manager/internal/bootstrap/config.go` - **No tests found**
- `resolveZapEnvironment` at `components/manager/internal/bootstrap/config.go` - **No tests found**
- `loadConfig` at `components/manager/internal/bootstrap/config.go` - **No tests found**
- `initMongoRepositories` at `components/manager/internal/bootstrap/config.go` - **No tests found**
- `initCrypto` at `components/manager/internal/bootstrap/config.go` - **No tests found**
- `\*Server.Run` at `components/manager/internal/bootstrap/server.go` - **No tests found**
- `\*AssignConnection.Execute` at `components/manager/internal/services/command/assign\_connection.go` - **No tests found**
- `\*CreateConnection.Execute` at `components/manager/internal/services/command/create\_connection.go` - **No tests found**
- `NewCreateFetcherJob` at `components/manager/internal/services/command/create\_fetcher\_job.go` - **No tests found**
- `\*CreateFetcherJob.Execute` at `components/manager/internal/services/command/create\_fetcher\_job.go` - **No tests found**
- `\*CreateFetcherJob.validateProductOwnership` at `components/manager/internal/services/command/create\_fetcher\_job.go` - **No tests found**
- `\*CreateFetcherJob.TestConnection` at `components/manager/internal/services/command/create\_fetcher\_job.go` - **No tests found**
- `\*CreateProduct.Execute` at `components/manager/internal/services/command/create\_product.go` - **No tests found**
- `\*DeleteProduct.Execute` at `components/manager/internal/services/command/delete\_product.go` - **No tests found**
- `\*UpdateConnection.Execute` at `components/manager/internal/services/command/update\_connection.go` - **No tests found**
- `\*UpdateProduct.Execute` at `components/manager/internal/services/command/update\_product.go` - **No tests found**
- `\*GetConnectionSchema.Execute` at `components/manager/internal/services/query/get\_connection\_schema.go` - **No tests found**
- `\*ListConnections.Execute` at `components/manager/internal/services/query/list\_connections.go` - **No tests found**
- `\*ListProducts.Execute` at `components/manager/internal/services/query/list\_products.go` - **No tests found**
- `\*ListUnassignedConnections.Execute` at `components/manager/internal/services/query/list\_unassigned\_connections.go` - **No tests found**
- `\*TestConnection.Execute` at `components/manager/internal/services/query/test\_connection.go` - **No tests found**
- `\*ValidateSchema.Execute` at `components/manager/internal/services/query/validate\_schema.go` - **No tests found**
- `\*ValidateSchema.getOrFetchSchema` at `components/manager/internal/services/query/validate\_schema.go` - **No tests found**
- `\*ConsumerRoutes.Shutdown` at `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go` - **No tests found**
- `NewConsumerRoutes` at `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go` - **No tests found**
- `\*ConsumerRoutes.RunConsumers` at `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go` - **No tests found**
- `NewPublisherRoutes` at `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go` - **No tests found**
- `\*PublisherRoutes.Publish` at `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go` - **No tests found**
- `\*PublisherRoutes.Shutdown` at `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go` - **No tests found**
- `resolveZapEnvironment` at `components/worker/internal/bootstrap/config.go` - **No tests found**
- `must` at `components/worker/internal/bootstrap/config.go` - **No tests found**
- `\*MultiQueueConsumer.Run` at `components/worker/internal/bootstrap/consumer.go` - **No tests found**
- `\*MultiQueueConsumer.handlerGenerateReport` at `components/worker/internal/bootstrap/consumer.go` - **No tests found**
- `\*Service.Run` at `components/worker/internal/bootstrap/service.go` - **No tests found**
- `\*UseCase.ExtractExternalData` at `components/worker/internal/services/extract-data.go` - **No tests found**
- `\*UseCase.saveExternalDataToSeaweedFS` at `components/worker/internal/services/extract-data.go` - **No tests found**
- `\*UseCase.shouldSkipProcessing` at `components/worker/internal/services/extract-data.go` - **No tests found**
- `\*UseCase.parseMessage` at `components/worker/internal/services/extract-data.go` - **No tests found**
- `\*UseCase.extractJobIDFromMultipleSources` at `components/worker/internal/services/extract-data.go` - **No tests found**
- `\*UseCase.extractJobIDFromPartialJSON` at `components/worker/internal/services/extract-data.go` - **No tests found**
- `\*UseCase.queryDatabase` at `components/worker/internal/services/extract-data.go` - **No tests found**
- `\*UseCase.handleErrorWithUpdate` at `components/worker/internal/services/extract-data.go` - **No tests found**
- `\*UseCase.encryptDataForSeaweedFS` at `components/worker/internal/services/extract-data.go` - **No tests found**
- `\*UseCase.checkReportStatus` at `components/worker/internal/services/extract-data.go` - **No tests found**
- `\*UseCase.transformPluginCRMAdvancedFilters` at `components/worker/internal/services/extract\_crm\_data.go` - **No tests found**
- `\*UseCase.decryptPluginCRMData` at `components/worker/internal/services/extract\_crm\_data.go` - **No tests found**
- `\*UseCase.QueryPluginCRM` at `components/worker/internal/services/extract\_crm\_data.go` - **No tests found**
- `\*UseCase.queryPluginCRMCollectionWithFilters` at `components/worker/internal/services/extract\_crm\_data.go` - **No tests found**
- `\*UseCase.processPluginCRMCollection` at `components/worker/internal/services/extract\_crm\_data.go` - **No tests found**
- `\*UseCase.publishJobNotification` at `components/worker/internal/services/job\_notification.go` - **No tests found**
- `NewLoggerFromContext` at `pkg/context.go` - **No tests found**
- `newDataSourceConfigMongoDB` at `pkg/datasource/datasource-factory.go` - **No tests found**
- `newDataSourceConfigPostgres` at `pkg/datasource/datasource-factory.go` - **No tests found**
- `newDataSourceConfigOracle` at `pkg/datasource/datasource-factory.go` - **No tests found**
- `newDataSourceConfigMySQL` at `pkg/datasource/datasource-factory.go` - **No tests found**
- `newDataSourceConfigSQLServer` at `pkg/datasource/datasource-factory.go` - **No tests found**
- `NewDataSourceFromConnectionWithLogger` at `pkg/datasource/datasource-factory.go` - **No tests found**
- `resolveBuildSecretSource` at `pkg/itestkit/addons/e2ekit/build\_secrets.go` - **No tests found**
- `createBuildSecretTempFile` at `pkg/itestkit/addons/e2ekit/build\_secrets.go` - **No tests found**
- `\*Builder.WithRewriter` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `localhostToHostGatewayRewriter.Rewrite` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `\*Builder.WithImage` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `\*Builder.Run` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `dumpRecentLogs` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `\*Builder.WithEnv` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `\*Builder.ExposePort` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `\*Builder.DisableDefaultLocalhostRewrite` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `\*Builder.WithLogsOnFailureMaxBytes` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `\*Builder.WithWait` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `waitHTTP.Configure` at `pkg/itestkit/addons/e2ekit/builder.go` - **No tests found**
- `\*ChaosAssertions.TimeoutsBelow` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ChaosAssertions.ThroughputAbove` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ChaosAssertions.FailedResults` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ChaosAssertions.P50Below` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ChaosAssertions.MinRequestsReached` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ChaosAssertions.Summary` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ChaosAssertions.AverageLatencyBelow` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ChaosAssertions.FailuresBelow` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ChaosAssertions.SuccessRateAbove` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ChaosAssertions.P99Below` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ChaosAssertions.P95Below` at `pkg/itestkit/addons/metricskit/assertions.go` - **No tests found**
- `\*ErrorClassifier.GetCategoryCounts` at `pkg/itestkit/addons/metricskit/error\_classifier.go` - **No tests found**
- `\*ChaosMetrics.GetTotalRequests` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.SuccessRate` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.StartTest` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.StartChaos` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.GetErrorCounts` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.ChaosThroughputRPS` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.Percentile` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.EndChaos` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.RecordRequest` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.GetTimeoutRequests` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.SuccessfulThroughputRPS` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.ChaosDuration` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.TestDuration` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.GetFailedRequests` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.GetMinLatency` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.AverageLatency` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.ThroughputRPS` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*ChaosMetrics.EndTest` at `pkg/itestkit/addons/metricskit/metrics.go` - **No tests found**
- `\*Reporter.WriteReport` at `pkg/itestkit/addons/metricskit/reporter.go` - **No tests found**
- `\*Reporter.String` at `pkg/itestkit/addons/metricskit/reporter.go` - **No tests found**
- `\*Reporter.CompactSummary` at `pkg/itestkit/addons/metricskit/reporter.go` - **No tests found**
- `\*AMQPConsumerBuilder.WithPrefetch` at `pkg/itestkit/addons/queuekit/amqp.go` - **No tests found**
- `\*AMQPConsumer.Close` at `pkg/itestkit/addons/queuekit/amqp.go` - **No tests found**
- `\*AMQPPublisher.connect` at `pkg/itestkit/addons/queuekit/amqp.go` - **No tests found**
- `\*AMQPPublisher.Close` at `pkg/itestkit/addons/queuekit/amqp.go` - **No tests found**
- `\*AMQPConsumerBuilder.BindTo` at `pkg/itestkit/addons/queuekit/amqp.go` - **No tests found**
- `\*AMQPConsumerBuilder.WithQueueDeclare` at `pkg/itestkit/addons/queuekit/amqp.go` - **No tests found**
- `NewAMQPConsumer` at `pkg/itestkit/addons/queuekit/amqp.go` - **No tests found**
- `\*AMQPConsumer.connect` at `pkg/itestkit/addons/queuekit/amqp.go` - **No tests found**
- `\*Assertions\[T\].HasHeaderKey` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ResultAssertions\[T\].At` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*MessageSequence\[T\].FilterBy` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ExpectMessagesHelper\[T\].OrFatal` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*Assertions\[T\].PayloadSatisfies` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ResultAssertions\[T\].UnmatchedCount` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*Assertions\[T\].HasRoutingKey` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*Assertions\[T\].HasMessageID` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*Assertions\[T\].HasContentType` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ResultAssertions\[T\].First` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `AssertResult` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `AssertJSONEqual` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*MessageSequence\[T\].GroupBy` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ExpectMessagesHelper\[T\].ToContainWhere` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `AssertMessage` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*Assertions\[T\].HasHeader` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ResultAssertions\[T\].HasAtLeast` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `JSONEqual` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ExpectMessagesHelper\[T\].ToHaveCount` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*Assertions\[T\].HasCorrelationID` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*Assertions\[T\].PayloadEquals` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ResultAssertions\[T\].HasCount` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ResultAssertions\[T\].DidNotTimeout` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ResultAssertions\[T\].HasNoErrors` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ResultAssertions\[T\].All` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*MessageSequence\[T\].RoutingKeysInOrder` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*ExpectMessagesHelper\[T\].ToSucceed` at `pkg/itestkit/addons/queuekit/assertions.go` - **No tests found**
- `\*Consumer\[T\].CaptureAll` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `\*ConsumerBuilder\[T\].WithMatcher` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `\*ConsumerBuilder\[T\].WithTimeout` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `\*Consumer\[T\].DrainQueue` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `\*Consumer\[T\].captureMessage` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `NewConsumer` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `\*ConsumerBuilder\[T\].WithUnmarshaler` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `\*ConsumerBuilder\[T\].Build` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `\*Consumer\[T\].WaitForMessages` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `\*Consumer\[T\].GetCaptured` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `\*Consumer\[T\].ClearCaptured` at `pkg/itestkit/addons/queuekit/consumer.go` - **No tests found**
- `MatchHeaderExists` at `pkg/itestkit/addons/queuekit/matcher.go` - **No tests found**
- `MatchJSONField` at `pkg/itestkit/addons/queuekit/matcher.go` - **No tests found**
- `MatchJSONFieldPattern` at `pkg/itestkit/addons/queuekit/matcher.go` - **No tests found**
- `hasNestedValue` at `pkg/itestkit/addons/queuekit/matcher.go` - **No tests found**
- `MatchAll` at `pkg/itestkit/addons/queuekit/matcher.go` - **No tests found**
- `MatchAny` at `pkg/itestkit/addons/queuekit/matcher.go` - **No tests found**
- `MatchRoutingKeyPattern` at `pkg/itestkit/addons/queuekit/matcher.go` - **No tests found**
- `applyPublishOptions` at `pkg/itestkit/addons/queuekit/queuekit.go` - **No tests found**
- `WaitResult\[T\].First` at `pkg/itestkit/addons/queuekit/queuekit.go` - **No tests found**
- `\*toxiproxyChaos.RemoveToxic` at `pkg/itestkit/chaos\_toxiproxy.go` - **No tests found**
- `\*toxiproxyChaos.RemoveAllToxics` at `pkg/itestkit/chaos\_toxiproxy.go` - **No tests found**
- `\*toxiproxyChaos.Close` at `pkg/itestkit/chaos\_toxiproxy.go` - **No tests found**
- `NewToxiproxyChaos` at `pkg/itestkit/chaos\_toxiproxy.go` - **No tests found**
- `\*toxiproxyChaos.CreateProxy` at `pkg/itestkit/chaos\_toxiproxy.go` - **No tests found**
- `\*toxiproxyChaos.AddBandwidth` at `pkg/itestkit/chaos\_toxiproxy.go` - **No tests found**
- `\*toxiproxyChaos.AddLatency` at `pkg/itestkit/chaos\_toxiproxy.go` - **No tests found**
- `\*toxiproxyChaos.CutConnection` at `pkg/itestkit/chaos\_toxiproxy.go` - **No tests found**
- `\*toxiproxyChaos.AddTimeout` at `pkg/itestkit/chaos\_toxiproxy.go` - **No tests found**
- `WaitListeningPort.Apply` at `pkg/itestkit/container\_generic.go` - **No tests found**
- `\*Builder.WithContainerCustomize` at `pkg/itestkit/container\_generic.go` - **No tests found**
- `\*genericContainerInfra.Start` at `pkg/itestkit/container\_generic.go` - **No tests found**
- `\*genericContainerInfra.Terminate` at `pkg/itestkit/container\_generic.go` - **No tests found**
- `portKey` at `pkg/itestkit/container\_generic.go` - **No tests found**
- `CustomizerFunc.Customize` at `pkg/itestkit/customizer.go` - **No tests found**
- `MergeCustomizers` at `pkg/itestkit/customizer.go` - **No tests found**
- `CEnvFromOS` at `pkg/itestkit/customizer\_options.go` - **No tests found**
- `uniqueAppendMany` at `pkg/itestkit/customizer\_options.go` - **No tests found**
- `CEnvs` at `pkg/itestkit/customizer\_options.go` - **No tests found**
- `CAll` at `pkg/itestkit/customizer\_options.go` - **No tests found**
- `CNetworkAliases` at `pkg/itestkit/customizer\_options.go` - **No tests found**
- `HostGatewayIP` at `pkg/itestkit/hostport.go` - **No tests found**
- `ParseHostPort` at `pkg/itestkit/hostport.go` - **No tests found**
- `ResolveHostHostPort` at `pkg/itestkit/hostport.go` - **No tests found**
- `validateUniqueInfraNames` at `pkg/itestkit/infra.go` - **No tests found**
- `NewMongoDBInfra` at `pkg/itestkit/infra/mongodb/mongodb.go` - **No tests found**
- `\*MongoDBInfra.HostPort` at `pkg/itestkit/infra/mongodb/mongodb.go` - **No tests found**
- `NewMongoDBInfraStub` at `pkg/itestkit/infra/mongodb/mongodb.go` - **No tests found**
- `\*MongoDBInfra.Start` at `pkg/itestkit/infra/mongodb/mongodb.go` - **No tests found**
- `\*MongoDBInfra.Endpoint` at `pkg/itestkit/infra/mongodb/mongodb.go` - **No tests found**
- `\*MongoDBInfra.URI` at `pkg/itestkit/infra/mongodb/mongodb.go` - **No tests found**
- `\*MongoDBInfra.Terminate` at `pkg/itestkit/infra/mongodb/mongodb.go` - **No tests found**
- `\*MongoDBInfra.ContainerHostPort` at `pkg/itestkit/infra/mongodb/mongodb.go` - **No tests found**
- `WithMongoDBFixedPort` at `pkg/itestkit/infra/mongodb/mongodb\_options.go` - **No tests found**
- `\*MSSQLInfra.Endpoint` at `pkg/itestkit/infra/mssql/mssql.go` - **No tests found**
- `NewMSSQLInfraStub` at `pkg/itestkit/infra/mssql/mssql.go` - **No tests found**
- `NewMSSQLInfra` at `pkg/itestkit/infra/mssql/mssql.go` - **No tests found**
- `\*MSSQLInfra.Start` at `pkg/itestkit/infra/mssql/mssql.go` - **No tests found**
- `\*MSSQLInfra.DSN` at `pkg/itestkit/infra/mssql/mssql.go` - **No tests found**
- `\*MSSQLInfra.HostPort` at `pkg/itestkit/infra/mssql/mssql.go` - **No tests found**
- `\*MSSQLInfra.Terminate` at `pkg/itestkit/infra/mssql/mssql.go` - **No tests found**
- `WithMSSQLFixedPort` at `pkg/itestkit/infra/mssql/mssql\_options.go` - **No tests found**
- `NewMySQLInfra` at `pkg/itestkit/infra/mysql/mysql.go` - **No tests found**
- `\*MySQLInfra.Endpoint` at `pkg/itestkit/infra/mysql/mysql.go` - **No tests found**
- `\*MySQLInfra.Terminate` at `pkg/itestkit/infra/mysql/mysql.go` - **No tests found**
- `\*MySQLInfra.Start` at `pkg/itestkit/infra/mysql/mysql.go` - **No tests found**
- `\*MySQLInfra.DSN` at `pkg/itestkit/infra/mysql/mysql.go` - **No tests found**
- `\*MySQLInfra.HostPort` at `pkg/itestkit/infra/mysql/mysql.go` - **No tests found**
- `NewMySQLInfraStub` at `pkg/itestkit/infra/mysql/mysql.go` - **No tests found**
- `WithMySQLInitScript` at `pkg/itestkit/infra/mysql/mysql\_options.go` - **No tests found**
- `WithMySQLFixedPort` at `pkg/itestkit/infra/mysql/mysql\_options.go` - **No tests found**
- `\*OracleInfra.Endpoint` at `pkg/itestkit/infra/oracle/oracle.go` - **No tests found**
- `NewOracleInfraStub` at `pkg/itestkit/infra/oracle/oracle.go` - **No tests found**
- `\*OracleInfra.DSN` at `pkg/itestkit/infra/oracle/oracle.go` - **No tests found**
- `\*OracleInfra.GoDRORDSN` at `pkg/itestkit/infra/oracle/oracle.go` - **No tests found**
- `\*OracleInfra.HostPort` at `pkg/itestkit/infra/oracle/oracle.go` - **No tests found**
- `\*OracleInfra.Terminate` at `pkg/itestkit/infra/oracle/oracle.go` - **No tests found**
- `NewOracleInfra` at `pkg/itestkit/infra/oracle/oracle.go` - **No tests found**
- `\*OracleInfra.Start` at `pkg/itestkit/infra/oracle/oracle.go` - **No tests found**
- `WithOracleInitScript` at `pkg/itestkit/infra/oracle/oracle\_options.go` - **No tests found**
- `WithOracleFixedPort` at `pkg/itestkit/infra/oracle/oracle\_options.go` - **No tests found**
- `\*PostgresInfra.Start` at `pkg/itestkit/infra/postgres/postgres.go` - **No tests found**
- `\*PostgresInfra.DSN` at `pkg/itestkit/infra/postgres/postgres.go` - **No tests found**
- `\*PostgresInfra.HostPort` at `pkg/itestkit/infra/postgres/postgres.go` - **No tests found**
- `NewPostgresInfra` at `pkg/itestkit/infra/postgres/postgres.go` - **No tests found**
- `\*PostgresInfra.Endpoint` at `pkg/itestkit/infra/postgres/postgres.go` - **No tests found**
- `\*PostgresInfra.Terminate` at `pkg/itestkit/infra/postgres/postgres.go` - **No tests found**
- `NewPostgresInfraStub` at `pkg/itestkit/infra/postgres/postgres.go` - **No tests found**
- `\*PostgresInfra.ContainerHostPort` at `pkg/itestkit/infra/postgres/postgres.go` - **No tests found**
- `WithPGInitFile` at `pkg/itestkit/infra/postgres/postgres\_options.go` - **No tests found**
- `WithPGFixedPort` at `pkg/itestkit/infra/postgres/postgres\_options.go` - **No tests found**
- `WithRabbitFixedPort` at `pkg/itestkit/infra/rabbitmq/rabbit\_options.go` - **No tests found**
- `\*configReader.Read` at `pkg/itestkit/infra/rabbitmq/rabbit\_options.go` - **No tests found**
- `NewRabbitInfra` at `pkg/itestkit/infra/rabbitmq/rabbitmq.go` - **No tests found**
- `\*RabbitInfra.Start` at `pkg/itestkit/infra/rabbitmq/rabbitmq.go` - **No tests found**
- `\*RabbitInfra.Endpoint` at `pkg/itestkit/infra/rabbitmq/rabbitmq.go` - **No tests found**
- `\*RabbitInfra.AMQPURL` at `pkg/itestkit/infra/rabbitmq/rabbitmq.go` - **No tests found**
- `\*RabbitInfra.HostPort` at `pkg/itestkit/infra/rabbitmq/rabbitmq.go` - **No tests found**
- `\*RabbitInfra.Terminate` at `pkg/itestkit/infra/rabbitmq/rabbitmq.go` - **No tests found**
- `\*RabbitInfra.ContainerHostPort` at `pkg/itestkit/infra/rabbitmq/rabbitmq.go` - **No tests found**
- `NewRedisInfra` at `pkg/itestkit/infra/redis/redis.go` - **No tests found**
- `\*RedisInfra.Start` at `pkg/itestkit/infra/redis/redis.go` - **No tests found**
- `\*RedisInfra.HostPort` at `pkg/itestkit/infra/redis/redis.go` - **No tests found**
- `\*RedisInfra.Terminate` at `pkg/itestkit/infra/redis/redis.go` - **No tests found**
- `\*RedisInfra.Endpoint` at `pkg/itestkit/infra/redis/redis.go` - **No tests found**
- `\*RedisInfra.URL` at `pkg/itestkit/infra/redis/redis.go` - **No tests found**
- `\*RedisInfra.Addr` at `pkg/itestkit/infra/redis/redis.go` - **No tests found**
- `\*RedisInfra.ContainerHostPort` at `pkg/itestkit/infra/redis/redis.go` - **No tests found**
- `WithRedisFixedPort` at `pkg/itestkit/infra/redis/redis\_options.go` - **No tests found**
- `\*SeaweedFSInfra.Start` at `pkg/itestkit/infra/seaweedfs/seaweedfs.go` - **No tests found**
- `\*SeaweedFSInfra.Endpoint` at `pkg/itestkit/infra/seaweedfs/seaweedfs.go` - **No tests found**
- `\*SeaweedFSInfra.URL` at `pkg/itestkit/infra/seaweedfs/seaweedfs.go` - **No tests found**
- `\*SeaweedFSInfra.HostPort` at `pkg/itestkit/infra/seaweedfs/seaweedfs.go` - **No tests found**
- `\*SeaweedFSInfra.Terminate` at `pkg/itestkit/infra/seaweedfs/seaweedfs.go` - **No tests found**
- `\*SeaweedFSInfra.ContainerHostPort` at `pkg/itestkit/infra/seaweedfs/seaweedfs.go` - **No tests found**
- `WithSeaweedFSFixedPort` at `pkg/itestkit/infra/seaweedfs/seaweedfs\_options.go` - **No tests found**
- `\*Builder.Build` at `pkg/itestkit/suite.go` - **No tests found**
- `\*Suite.Terminate` at `pkg/itestkit/suite.go` - **No tests found**
- `New` at `pkg/itestkit/suite.go` - **No tests found**
- `\*Builder.WithInfra` at `pkg/itestkit/suite.go` - **No tests found**
- `\*Builder.WithInfras` at `pkg/itestkit/suite.go` - **No tests found**
- `\*Suite.Network` at `pkg/itestkit/suite.go` - **No tests found**
- `\*DataSourceConfigMongoDB.GetSchemaInfo` at `pkg/model/datasource/mongodb/datasource-config.go` - **No tests found**
- `\*DataSourceConfigMongoDB.Connect` at `pkg/model/datasource/mongodb/datasource-config.go` - **No tests found**
- `\*DataSourceConfigMongoDB.Query` at `pkg/model/datasource/mongodb/datasource-config.go` - **No tests found**
- `\*DataSourceConfigMySQL.Connect` at `pkg/model/datasource/mysql/datasource-config.go` - **No tests found**
- `\*DataSourceConfigMySQL.Query` at `pkg/model/datasource/mysql/datasource-config.go` - **No tests found**
- `\*DataSourceConfigMySQL.GetSchemaInfo` at `pkg/model/datasource/mysql/datasource-config.go` - **No tests found**
- `\*DataSourceConfigOracle.Connect` at `pkg/model/datasource/oracle/datasource-config.go` - **No tests found**
- `\*DataSourceConfigOracle.Query` at `pkg/model/datasource/oracle/datasource-config.go` - **No tests found**
- `\*DataSourceConfigOracle.GetSchemaInfo` at `pkg/model/datasource/oracle/datasource-config.go` - **No tests found**
- `\*DataSourceConfigPostgres.Connect` at `pkg/model/datasource/postgres/datasource-config.go` - **No tests found**
- `\*DataSourceConfigPostgres.GetSchemaInfo` at `pkg/model/datasource/postgres/datasource-config.go` - **No tests found**
- `\*DataSourceConfigPostgres.Query` at `pkg/model/datasource/postgres/datasource-config.go` - **No tests found**
- `\*DataSourceConfigSQLServer.Query` at `pkg/model/datasource/sqlserver/datasource-config.go` - **No tests found**
- `\*DataSourceConfigSQLServer.GetSchemaInfo` at `pkg/model/datasource/sqlserver/datasource-config.go` - **No tests found**
- `\*DataSourceConfigSQLServer.Connect` at `pkg/model/datasource/sqlserver/datasource-config.go` - **No tests found**
- `\*ConnectionMongoDBRepository.ListUnassigned` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.AssignProduct` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.FindByID` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.FindByOrganizationAndName` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.FindByOrganizationAndDatabaseName` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.CountByProduct` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.buildQueryFilter` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `NewConnectionMongoDBRepository` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.Create` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.Update` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.Delete` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.FindByConfigNames` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*ConnectionMongoDBRepository.List` at `pkg/mongodb/connection/connection.mongodb.go` - **No tests found**
- `\*MockmongoDatabaseProvider.Client` at `pkg/mongodb/connection/connection.mongodb.mock.go` - **No tests found**
- `\*MockmongoDatabaseProviderMockRecorder.Client` at `pkg/mongodb/connection/connection.mongodb.mock.go` - **No tests found**
- `TestMain` at `pkg/mongodb/connection/connection.mongodb\_test.go` - **No tests found**
- `\*ConnectionMongoDBRepository.EnsureIndexes` at `pkg/mongodb/connection/indexes.go` - **No tests found**
- `\*ConnectionMongoDBRepository.DropIndexes` at `pkg/mongodb/connection/indexes.go` - **No tests found**
- `\*ExternalDataSource.Query` at `pkg/mongodb/datasource.mongodb.go` - **No tests found**
- `\*ExternalDataSource.QueryWithAdvancedFilters` at `pkg/mongodb/datasource.mongodb.go` - **No tests found**
- `\*ExternalDataSource.GetDatabaseSchema` at `pkg/mongodb/datasource.mongodb.go` - **No tests found**
- `\*ExternalDataSource.processQueryResults` at `pkg/mongodb/datasource.mongodb.go` - **No tests found**
- `\*ExternalDataSource.CloseConnection` at `pkg/mongodb/datasource.mongodb.go` - **No tests found**
- `NewMockDatasource` at `pkg/mongodb/datasource.mongodb.mock.go` - **No tests found**
- `\*MockDatasourceMockRecorder.CloseConnection` at `pkg/mongodb/datasource.mongodb.mock.go` - **No tests found**
- `\*MockDatasource.GetDatabaseSchema` at `pkg/mongodb/datasource.mongodb.mock.go` - **No tests found**
- `\*MockDatasource.Query` at `pkg/mongodb/datasource.mongodb.mock.go` - **No tests found**
- `\*MockDatasource.EXPECT` at `pkg/mongodb/datasource.mongodb.mock.go` - **No tests found**
- `\*MockDatasource.CloseConnection` at `pkg/mongodb/datasource.mongodb.mock.go` - **No tests found**
- `\*MockDatasourceMockRecorder.GetDatabaseSchema` at `pkg/mongodb/datasource.mongodb.mock.go` - **No tests found**
- `\*MockDatasourceMockRecorder.Query` at `pkg/mongodb/datasource.mongodb.mock.go` - **No tests found**
- `\*MockDatasource.QueryWithAdvancedFilters` at `pkg/mongodb/datasource.mongodb.mock.go` - **No tests found**
- `\*MockDatasourceMockRecorder.QueryWithAdvancedFilters` at `pkg/mongodb/datasource.mongodb.mock.go` - **No tests found**
- `\*JobMongoDBRepository.EnsureIndexes` at `pkg/mongodb/job/indexes.go` - **No tests found**
- `\*JobMongoDBRepository.DropIndexes` at `pkg/mongodb/job/indexes.go` - **No tests found**
- `\*JobMongoDBRepository.Create` at `pkg/mongodb/job/job.mongodb.go` - **No tests found**
- `\*JobMongoDBRepository.UpdateStatus` at `pkg/mongodb/job/job.mongodb.go` - **No tests found**
- `\*JobMongoDBRepository.FindByRequestHashWithinWindow` at `pkg/mongodb/job/job.mongodb.go` - **No tests found**
- `NewJobMongoDBRepository` at `pkg/mongodb/job/job.mongodb.go` - **No tests found**
- `\*JobMongoDBRepository.Update` at `pkg/mongodb/job/job.mongodb.go` - **No tests found**
- `\*JobMongoDBRepository.ExistsRunningByMappedFieldKey` at `pkg/mongodb/job/job.mongodb.go` - **No tests found**
- `\*JobMongoDBRepository.scanJobs` at `pkg/mongodb/job/job.mongodb.go` - **No tests found**
- `\*JobMongoDBRepository.FindByID` at `pkg/mongodb/job/job.mongodb.go` - **No tests found**
- `\*JobMongoDBRepository.List` at `pkg/mongodb/job/job.mongodb.go` - **No tests found**
- `TestMain` at `pkg/mongodb/job/job.mongodb\_test.go` - **No tests found**
- `MapMongoErrorToResponse` at `pkg/mongodb/mongo.go` - **No tests found**
- `\*MockMongoClientProvider.Client` at `pkg/mongodb/mongo\_client\_provider.mock.go` - **No tests found**
- `\*MockMongoClientProviderMockRecorder.Client` at `pkg/mongodb/mongo\_client\_provider.mock.go` - **No tests found**
- `\*ProductMongoDBRepository.EnsureIndexes` at `pkg/mongodb/product/indexes.go` - **No tests found**
- `\*ProductMongoDBRepository.DropIndexes` at `pkg/mongodb/product/indexes.go` - **No tests found**
- `\*ProductMongoDBRepository.List` at `pkg/mongodb/product/product.mongodb.go` - **No tests found**
- `\*ProductMongoDBRepository.buildQueryFilter` at `pkg/mongodb/product/product.mongodb.go` - **No tests found**
- `\*ProductMongoDBRepository.Create` at `pkg/mongodb/product/product.mongodb.go` - **No tests found**
- `\*ProductMongoDBRepository.Update` at `pkg/mongodb/product/product.mongodb.go` - **No tests found**
- `\*ProductMongoDBRepository.Delete` at `pkg/mongodb/product/product.mongodb.go` - **No tests found**
- `\*ProductMongoDBRepository.FindByID` at `pkg/mongodb/product/product.mongodb.go` - **No tests found**
- `\*ProductMongoDBRepository.FindByCode` at `pkg/mongodb/product/product.mongodb.go` - **No tests found**
- `\*ExternalDataSource.GetDatabaseSchema` at `pkg/mysql/datasource.mysql.go` - **No tests found**
- `\*ExternalDataSource.buildSchema` at `pkg/mysql/datasource.mysql.go` - **No tests found**
- `\*ExternalDataSource.CloseConnection` at `pkg/mysql/datasource.mysql.go` - **No tests found**
- `\*ExternalDataSource.scanColumns` at `pkg/mysql/datasource.mysql.go` - **No tests found**
- `NewDataSourceRepository` at `pkg/mysql/datasource.mysql.go` - **No tests found**
- `\*ExternalDataSource.Query` at `pkg/mysql/datasource.mysql.go` - **No tests found**
- `\*ExternalDataSource.buildTableSchema` at `pkg/mysql/datasource.mysql.go` - **No tests found**
- `scanRows` at `pkg/mysql/datasource.mysql.go` - **No tests found**
- `\*ExternalDataSource.ValidateTableAndFields` at `pkg/mysql/datasource.mysql.go` - **No tests found**
- `\*ExternalDataSource.QueryWithAdvancedFilters` at `pkg/mysql/datasource.mysql.go` - **No tests found**
- `\*Connection.GetDB` at `pkg/mysql/mysql.go` - **No tests found**
- `\*Connection.Connect` at `pkg/mysql/mysql.go` - **No tests found**
- `WithRecover` at `pkg/net/http/with\_recover.go` - **No tests found**
- `\*ExternalDataSource.CloseConnection` at `pkg/oracle/datasource.oracle.go` - **No tests found**
- `\*ExternalDataSource.GetDatabaseSchema` at `pkg/oracle/datasource.oracle.go` - **No tests found**
- `\*ExternalDataSource.buildSchema` at `pkg/oracle/datasource.oracle.go` - **No tests found**
- `\*ExternalDataSource.ValidateTableAndFields` at `pkg/oracle/datasource.oracle.go` - **No tests found**
- `createRowMap` at `pkg/oracle/datasource.oracle.go` - **No tests found**
- `\*ExternalDataSource.Query` at `pkg/oracle/datasource.oracle.go` - **No tests found**
- `\*ExternalDataSource.scanColumns` at `pkg/oracle/datasource.oracle.go` - **No tests found**
- `\*ExternalDataSource.QueryWithAdvancedFilters` at `pkg/oracle/datasource.oracle.go` - **No tests found**


## Focus Areas

Based on analysis, pay special attention to:

1. **Uncovered Code** - 376 modified functions without test coverage

