# Pre-Analysis Context: Code Quality

## Static Analysis Findings (0 issues)


No static analysis findings.


## Semantic Changes


### Functions Modified (550)

#### `.\*ConnectionHandler.ListConnections`
**File:** `components/manager/internal/adapters/http/in/connection.go:154-216`
**Changes:** implementation changed



#### `.\*ConnectionHandler.GetConnectionSchema`
**File:** `components/manager/internal/adapters/http/in/connection.go:580-626`
**Changes:** implementation changed



#### `.\*ConnectionHandler.CreateConnection`
**File:** `components/manager/internal/adapters/http/in/connection.go:72-130`
**Changes:** implementation changed



#### `.\*ConnectionHandler.GetConnection`
**File:** `components/manager/internal/adapters/http/in/connection.go:232-278`
**Changes:** implementation changed



#### `.\*ConnectionHandler.TestConnection`
**File:** `components/manager/internal/adapters/http/in/connection.go:295-341`
**Changes:** implementation changed



#### `.\*ConnectionHandler.UpdateConnection`
**File:** `components/manager/internal/adapters/http/in/connection.go:360-430`
**Changes:** implementation changed



#### `.\*ConnectionHandler.DeleteConnection`
**File:** `components/manager/internal/adapters/http/in/connection.go:447-490`
**Changes:** implementation changed



#### `.\*ConnectionHandler.ValidateSchema`
**File:** `components/manager/internal/adapters/http/in/connection.go:511-564`
**Changes:** implementation changed



#### `.setupConnectionTestApp`
**File:** `components/manager/internal/adapters/http/in/connection\_test.go:29-51`
**Changes:** implementation changed



#### `.\*FetcherHandler.GetJob`
**File:** `components/manager/internal/adapters/http/in/fetcher.go:170-218`
**Changes:** implementation changed



#### `.\*FetcherHandler.CreateJob`
**File:** `components/manager/internal/adapters/http/in/fetcher.go:56-154`
**Changes:** implementation changed



#### `.setupTestApp`
**File:** `components/manager/internal/adapters/http/in/fetcher\_test.go:28-50`
**Changes:** implementation changed



#### `.setupMiddlewareTestApp`
**File:** `components/manager/internal/adapters/http/in/middlewares\_test.go:18-40`
**Changes:** implementation changed



#### `.\*MigrationHandler.AssignConnectionToProduct`
**File:** `components/manager/internal/adapters/http/in/migration.go:125-200`
**Changes:** implementation changed



#### `.\*MigrationHandler.ListUnassignedConnections`
**File:** `components/manager/internal/adapters/http/in/migration.go:54-106`
**Changes:** implementation changed



#### `.\*ProductHandler.CreateProduct`
**File:** `components/manager/internal/adapters/http/in/product.go:63-121`
**Changes:** implementation changed



#### `.\*ProductHandler.ListProducts`
**File:** `components/manager/internal/adapters/http/in/product.go:140-192`
**Changes:** implementation changed



#### `.\*ProductHandler.GetProduct`
**File:** `components/manager/internal/adapters/http/in/product.go:208-254`
**Changes:** implementation changed



#### `.\*ProductHandler.UpdateProduct`
**File:** `components/manager/internal/adapters/http/in/product.go:272-342`
**Changes:** implementation changed



#### `.\*ProductHandler.DeleteProduct`
**File:** `components/manager/internal/adapters/http/in/product.go:359-402`
**Changes:** implementation changed



#### `.NewRoutes`
**File:** `components/manager/internal/adapters/http/in/routes.go:24-86`
**Changes:** implementation changed



#### `.InitServers`
**File:** `components/manager/internal/bootstrap/config.go:108-124`
**Changes:** implementation changed



#### `.\*Server.Run`
**File:** `components/manager/internal/bootstrap/server.go:44-61`
**Changes:** implementation changed



#### `.\*AssignConnection.Execute`
**File:** `components/manager/internal/services/command/assign\_connection.go:30-82`
**Changes:** implementation changed



#### `.\*CreateConnection.Execute`
**File:** `components/manager/internal/services/command/create\_connection.go:34-124`
**Changes:** implementation changed



#### `.NewCreateFetcherJob`
**File:** `components/manager/internal/services/command/create\_fetcher\_job.go:65-85`
**Changes:** parameters changed, signature\_changed


```diff
- func NewCreateFetcherJob\(connectionRepo connRepo.Repository, jobRepository jobRepo.Repository, productRepository productRepo.Repository, cryptor crypto.Cryptor, rabbitMQ \*rabbitmq.RabbitMQAdapter, queueName string\) \*CreateFetcherJob
+ func NewCreateFetcherJob\(connectionRepo connRepo.Repository, jobRepository jobRepo.Repository, productRepository productRepo.Repository, cryptor crypto.Cryptor, rabbitMQ rabbitmq.Adapter, queueName string\) \*CreateFetcherJob
```


#### `.NewCreateFetcherJobWithTester`
**File:** `components/manager/internal/services/command/create\_fetcher\_job.go:91-114`
**Changes:** parameters changed, signature\_changed


```diff
- func NewCreateFetcherJobWithTester\(connectionRepo connRepo.Repository, jobRepository jobRepo.Repository, productRepository productRepo.Repository, cryptor crypto.Cryptor, rabbitMQ \*rabbitmq.RabbitMQAdapter, tester ConnectionTester, queueName string\) \*CreateFetcherJob
+ func NewCreateFetcherJobWithTester\(connectionRepo connRepo.Repository, jobRepository jobRepo.Repository, productRepository productRepo.Repository, cryptor crypto.Cryptor, rabbitMQ rabbitmq.Adapter, tester ConnectionTester, queueName string\) \*CreateFetcherJob
```


#### `.\*CreateFetcherJob.Execute`
**File:** `components/manager/internal/services/command/create\_fetcher\_job.go:119-303`
**Changes:** implementation changed



#### `.\*CreateFetcherJob.validateProductOwnership`
**File:** `components/manager/internal/services/command/create\_fetcher\_job.go:308-356`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*CreateFetcherJob\) \*CreateFetcherJob.validateProductOwnership\(ctx context.Context, span \*trace.Span, source string, organizationID uuid.UUID, connections \[\]\*model.Connection\) error
+ func \(\*CreateFetcherJob\) \*CreateFetcherJob.validateProductOwnership\(ctx context.Context, span trace.Span, source string, organizationID uuid.UUID, connections \[\]\*model.Connection\) error
```


#### `.\*CreateFetcherJob.TestConnection`
**File:** `components/manager/internal/services/command/create\_fetcher\_job.go:360-380`
**Changes:** implementation changed



#### `.\*CreateProduct.Execute`
**File:** `components/manager/internal/services/command/create\_product.go:27-83`
**Changes:** implementation changed



#### `.\*DeleteProduct.Execute`
**File:** `components/manager/internal/services/command/delete\_product.go:29-87`
**Changes:** implementation changed



#### `.testContext`
**File:** `components/manager/internal/services/command/helpers\_test.go:12-21`
**Changes:** implementation changed



#### `.\*UpdateConnection.Execute`
**File:** `components/manager/internal/services/command/update\_connection.go:34-132`
**Changes:** implementation changed



#### `.\*UpdateProduct.Execute`
**File:** `components/manager/internal/services/command/update\_product.go:27-89`
**Changes:** implementation changed



#### `.\*GetConnectionSchema.Execute`
**File:** `components/manager/internal/services/query/get\_connection\_schema.go:45-142`
**Changes:** implementation changed



#### `.testContext`
**File:** `components/manager/internal/services/query/helpers\_test.go:12-21`
**Changes:** implementation changed



#### `.\*ListConnections.Execute`
**File:** `components/manager/internal/services/query/list\_connections.go:29-70`
**Changes:** implementation changed



#### `.\*ListProducts.Execute`
**File:** `components/manager/internal/services/query/list\_products.go:25-51`
**Changes:** implementation changed



#### `.\*ListUnassignedConnections.Execute`
**File:** `components/manager/internal/services/query/list\_unassigned\_connections.go:25-51`
**Changes:** implementation changed



#### `.\*TestConnection.Execute`
**File:** `components/manager/internal/services/query/test\_connection.go:53-156`
**Changes:** implementation changed



#### `.NewTestConnection`
**File:** `components/manager/internal/services/query/test\_connection.go:44-51`
**Changes:** implementation changed



#### `.TestTestConnection\_Execute\_WithSSLConfiguration`
**File:** `components/manager/internal/services/query/test\_connection\_test.go:325-366`
**Changes:** implementation changed



#### `.TestTestConnection\_Execute\_AllDatabaseTypes`
**File:** `components/manager/internal/services/query/test\_connection\_test.go:369-440`
**Changes:** implementation changed



#### `.NewValidateSchema`
**File:** `components/manager/internal/services/query/validate\_schema.go:51-62`
**Changes:** implementation changed



#### `.\*ValidateSchema.Execute`
**File:** `components/manager/internal/services/query/validate\_schema.go:65-219`
**Changes:** implementation changed



#### `.\*ValidateSchema.getOrFetchSchema`
**File:** `components/manager/internal/services/query/validate\_schema.go:222-294`
**Changes:** implementation changed



#### `.TestValidateSchema\_CacheSetError`
**File:** `components/manager/internal/services/query/validate\_schema\_test.go:800-856`
**Changes:** implementation changed



#### `.TestValidateSchema\_CacheError\_ContinuesToFetch`
**File:** `components/manager/internal/services/query/validate\_schema\_test.go:323-380`
**Changes:** implementation changed



#### `.\*ConsumerRoutes.Shutdown`
**File:** `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go:100-114`
**Changes:** implementation changed



#### `.NewConsumerRoutes`
**File:** `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go:39-45`
**Changes:** parameters changed, signature\_changed


```diff
- func NewConsumerRoutes\(conn \*libRabbitmq.RabbitMQConnection, numWorkers int, logger log.Logger, telemetry \*opentelemetry.Telemetry, signer crypto.Signer\) \*ConsumerRoutes
+ func NewConsumerRoutes\(conn \*libRabbitmq.RabbitMQConnection, numWorkers int, logger libLog.Logger, telemetry \*opentelemetry.Telemetry, signer crypto.Signer\) \*ConsumerRoutes
```


#### `.NewConsumerRoutesWithAdapter`
**File:** `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go:48-62`
**Changes:** parameters changed, signature\_changed


```diff
- func NewConsumerRoutesWithAdapter\(adapter rabbitmq.Adapter, numWorkers int, logger log.Logger, telemetry \*opentelemetry.Telemetry\) \*ConsumerRoutes
+ func NewConsumerRoutesWithAdapter\(adapter rabbitmq.Adapter, numWorkers int, logger libLog.Logger, telemetry \*opentelemetry.Telemetry\) \*ConsumerRoutes
```


#### `.\*ConsumerRoutes.RunConsumers`
**File:** `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go:70-97`
**Changes:** implementation changed



#### `.TestNewConsumerRoutes`
**File:** `components/worker/internal/adapters/rabbitmq/consumer\_test.go:16-33`
**Changes:** implementation changed



#### `.TestConsumerRoutes\_Shutdown`
**File:** `components/worker/internal/adapters/rabbitmq/consumer\_test.go:72-104`
**Changes:** implementation changed



#### `.TestConsumerRoutes\_Shutdown\_Error`
**File:** `components/worker/internal/adapters/rabbitmq/consumer\_test.go:106-122`
**Changes:** implementation changed



#### `.TestConsumerRoutes\_RunConsumers`
**File:** `components/worker/internal/adapters/rabbitmq/consumer\_test.go:124-217`
**Changes:** implementation changed



#### `.NewPublisherRoutes`
**File:** `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go:32-38`
**Changes:** parameters changed, signature\_changed


```diff
- func NewPublisherRoutes\(conn \*libRabbitmq.RabbitMQConnection, logger log.Logger, telemetry \*opentelemetry.Telemetry, signer crypto.Signer\) \*PublisherRoutes
+ func NewPublisherRoutes\(conn \*libRabbitmq.RabbitMQConnection, logger libLog.Logger, telemetry \*opentelemetry.Telemetry, signer crypto.Signer\) \*PublisherRoutes
```


#### `.NewPublisherRoutesWithAdapter`
**File:** `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go:41-49`
**Changes:** parameters changed, signature\_changed


```diff
- func NewPublisherRoutesWithAdapter\(adapter rabbitmq.Adapter, logger log.Logger, telemetry \*opentelemetry.Telemetry\) \*PublisherRoutes
+ func NewPublisherRoutesWithAdapter\(adapter rabbitmq.Adapter, logger libLog.Logger, telemetry \*opentelemetry.Telemetry\) \*PublisherRoutes
```


#### `.\*PublisherRoutes.Publish`
**File:** `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go:52-74`
**Changes:** implementation changed



#### `.\*PublisherRoutes.Shutdown`
**File:** `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go:77-88`
**Changes:** implementation changed



#### `.TestPublisherRoutes\_Shutdown`
**File:** `components/worker/internal/adapters/rabbitmq/publisher\_test.go:58-82`
**Changes:** implementation changed



#### `.TestNewPublisherRoutes`
**File:** `components/worker/internal/adapters/rabbitmq/publisher\_test.go:15-26`
**Changes:** implementation changed



#### `.TestPublisherRoutes\_Publish`
**File:** `components/worker/internal/adapters/rabbitmq/publisher\_test.go:28-56`
**Changes:** implementation changed



#### `.InitWorker`
**File:** `components/worker/internal/bootstrap/config.go:81-230`
**Changes:** implementation changed



#### `.\*MultiQueueConsumer.Run`
**File:** `components/worker/internal/bootstrap/consumer.go:48-93`
**Changes:** implementation changed



#### `.\*MultiQueueConsumer.handlerGenerateReport`
**File:** `components/worker/internal/bootstrap/consumer.go:96-118`
**Changes:** implementation changed



#### `.\*Service.Run`
**File:** `components/worker/internal/bootstrap/service.go:31-45`
**Changes:** implementation changed



#### `.\*UseCase.ExtractExternalData`
**File:** `components/worker/internal/services/extract-data.go:52-148`
**Changes:** implementation changed



#### `.\*UseCase.saveExternalDataToSeaweedFS`
**File:** `components/worker/internal/services/extract-data.go:448-524`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.saveExternalDataToSeaweedFS\(ctx context.Context, tracer trace.Tracer, message ExtractExternalDataMessage, result map\[string\]map\[string\]\[\]map\[string\]any, span \*trace.Span, logger log.Logger\) \(\*JobResultData, error\)
+ func \(\*UseCase\) \*UseCase.saveExternalDataToSeaweedFS\(ctx context.Context, tracer trace.Tracer, message ExtractExternalDataMessage, result map\[string\]map\[string\]\[\]map\[string\]any, span trace.Span, logger libLog.Logger\) \(\*JobResultData, error\)
```


#### `.\*UseCase.shouldSkipProcessing`
**File:** `components/worker/internal/services/extract-data.go:559-569`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.shouldSkipProcessing\(ctx context.Context, jobID uuid.UUID, organizationID uuid.UUID, logger log.Logger\) bool
+ func \(\*UseCase\) \*UseCase.shouldSkipProcessing\(ctx context.Context, jobID uuid.UUID, organizationID uuid.UUID, logger libLog.Logger\) bool
```


#### `.\*UseCase.parseMessage`
**File:** `components/worker/internal/services/extract-data.go:152-185`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.parseMessage\(ctx context.Context, body \[\]byte, headers map\[string\]any, span \*trace.Span, logger log.Logger\) \(\*ExtractExternalDataMessage, error\)
+ func \(\*UseCase\) \*UseCase.parseMessage\(ctx context.Context, body \[\]byte, headers map\[string\]any, span trace.Span, logger libLog.Logger\) \(\*ExtractExternalDataMessage, error\)
```


#### `.\*UseCase.extractJobIDFromMultipleSources`
**File:** `components/worker/internal/services/extract-data.go:188-216`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.extractJobIDFromMultipleSources\(body \[\]byte, headers map\[string\]any, logger log.Logger\) \(uuid.UUID, uuid.UUID\)
+ func \(\*UseCase\) \*UseCase.extractJobIDFromMultipleSources\(ctx context.Context, body \[\]byte, headers map\[string\]any, logger libLog.Logger\) \(uuid.UUID, uuid.UUID\)
```


#### `.\*UseCase.extractJobIDFromPartialJSON`
**File:** `components/worker/internal/services/extract-data.go:220-268`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.extractJobIDFromPartialJSON\(body \[\]byte, logger log.Logger\) \(uuid.UUID, uuid.UUID\)
+ func \(\*UseCase\) \*UseCase.extractJobIDFromPartialJSON\(ctx context.Context, body \[\]byte, logger libLog.Logger\) \(uuid.UUID, uuid.UUID\)
```


#### `.\*UseCase.queryDatabase`
**File:** `components/worker/internal/services/extract-data.go:351-435`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.queryDatabase\(ctx context.Context, databaseName string, tables map\[string\]\[\]string, connections \[\]\*model.Connection, allFilters map\[string\]map\[string\]map\[string\]modelJob.FilterCondition, organizationID uuid.UUID, result map\[string\]map\[string\]\[\]map\[string\]any, logger log.Logger, tracer trace.Tracer\) error
+ func \(\*UseCase\) \*UseCase.queryDatabase\(ctx context.Context, databaseName string, tables map\[string\]\[\]string, connections \[\]\*model.Connection, allFilters map\[string\]map\[string\]map\[string\]modelJob.FilterCondition, organizationID uuid.UUID, result map\[string\]map\[string\]\[\]map\[string\]any, logger libLog.Logger, tracer trace.Tracer\) error
```


#### `.\*UseCase.handleErrorWithUpdate`
**File:** `components/worker/internal/services/extract-data.go:271-317`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.handleErrorWithUpdate\(ctx context.Context, jobID uuid.UUID, orgID uuid.UUID, message ExtractExternalDataMessage, span \*trace.Span, errorMsg string, err error, logger log.Logger\) error
+ func \(\*UseCase\) \*UseCase.handleErrorWithUpdate\(ctx context.Context, jobID uuid.UUID, orgID uuid.UUID, message ExtractExternalDataMessage, span trace.Span, errorMsg string, err error, logger libLog.Logger\) error
```


#### `.\*UseCase.encryptDataForSeaweedFS`
**File:** `components/worker/internal/services/extract-data.go:527-556`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.encryptDataForSeaweedFS\(data \[\]byte, logger log.Logger\) \(\[\]byte, error\)
+ func \(\*UseCase\) \*UseCase.encryptDataForSeaweedFS\(data \[\]byte, logger libLog.Logger\) \(\[\]byte, error\)
```


#### `.\*UseCase.checkReportStatus`
**File:** `components/worker/internal/services/extract-data.go:572-594`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.checkReportStatus\(ctx context.Context, jobID uuid.UUID, organizationID uuid.UUID, logger log.Logger\) \(model.JobStatus, error\)
+ func \(\*UseCase\) \*UseCase.checkReportStatus\(ctx context.Context, jobID uuid.UUID, organizationID uuid.UUID, logger libLog.Logger\) \(model.JobStatus, error\)
```


#### `.\*UseCase.transformPluginCRMAdvancedFilters`
**File:** `components/worker/internal/services/extract\_crm\_data.go:162-238`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.transformPluginCRMAdvancedFilters\(filter map\[string\]modelJob.FilterCondition, logger log.Logger\) \(map\[string\]modelJob.FilterCondition, error\)
+ func \(\*UseCase\) \*UseCase.transformPluginCRMAdvancedFilters\(filter map\[string\]modelJob.FilterCondition, logger libLog.Logger\) \(map\[string\]modelJob.FilterCondition, error\)
```


#### `.\*UseCase.decryptPluginCRMData`
**File:** `components/worker/internal/services/extract\_crm\_data.go:257-310`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.decryptPluginCRMData\(logger log.Logger, collectionResult \[\]map\[string\]any, fields \[\]string\) \(\[\]map\[string\]any, error\)
+ func \(\*UseCase\) \*UseCase.decryptPluginCRMData\(logger libLog.Logger, collectionResult \[\]map\[string\]any, fields \[\]string\) \(\[\]map\[string\]any, error\)
```


#### `.\*UseCase.QueryPluginCRM`
**File:** `components/worker/internal/services/extract\_crm\_data.go:57-88`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.QueryPluginCRM\(ctx context.Context, dataSource \*datasourceMongoConfig.DataSourceConfigMongoDB, databaseName string, collections map\[string\]\[\]string, databaseFilters map\[string\]map\[string\]modelJob.FilterCondition, organizationID uuid.UUID, result map\[string\]map\[string\]\[\]map\[string\]any, logger log.Logger\) error
+ func \(\*UseCase\) \*UseCase.QueryPluginCRM\(ctx context.Context, dataSource \*datasourceMongoConfig.DataSourceConfigMongoDB, databaseName string, collections map\[string\]\[\]string, databaseFilters map\[string\]map\[string\]modelJob.FilterCondition, organizationID uuid.UUID, result map\[string\]map\[string\]\[\]map\[string\]any, logger libLog.Logger\) error
```


#### `.\*UseCase.queryPluginCRMCollectionWithFilters`
**File:** `components/worker/internal/services/extract\_crm\_data.go:129-159`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.queryPluginCRMCollectionWithFilters\(ctx context.Context, dataSource \*datasourceMongoConfig.DataSourceConfigMongoDB, collection string, fields \[\]string, collectionFilters map\[string\]modelJob.FilterCondition, logger log.Logger\) \(\[\]map\[string\]any, error\)
+ func \(\*UseCase\) \*UseCase.queryPluginCRMCollectionWithFilters\(ctx context.Context, dataSource \*datasourceMongoConfig.DataSourceConfigMongoDB, collection string, fields \[\]string, collectionFilters map\[string\]modelJob.FilterCondition, logger libLog.Logger\) \(\[\]map\[string\]any, error\)
```


#### `.\*UseCase.processPluginCRMCollection`
**File:** `components/worker/internal/services/extract\_crm\_data.go:91-126`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.processPluginCRMCollection\(ctx context.Context, dataSource \*datasourceMongoConfig.DataSourceConfigMongoDB, collection string, fields \[\]string, collectionFilters map\[string\]modelJob.FilterCondition, organizationID uuid.UUID, result map\[string\]map\[string\]\[\]map\[string\]any, logger log.Logger\) error
+ func \(\*UseCase\) \*UseCase.processPluginCRMCollection\(ctx context.Context, dataSource \*datasourceMongoConfig.DataSourceConfigMongoDB, collection string, fields \[\]string, collectionFilters map\[string\]modelJob.FilterCondition, organizationID uuid.UUID, result map\[string\]map\[string\]\[\]map\[string\]any, logger libLog.Logger\) error
```


#### `.TestQueryPluginCRMCollectionWithFilters\_NoFilters`
**File:** `components/worker/internal/services/extract\_crm\_data\_test.go:1142-1174`
**Changes:** implementation changed



#### `.TestQueryPluginCRM\_WithOrganizationOnly`
**File:** `components/worker/internal/services/extract\_crm\_data\_test.go:1295-1337`
**Changes:** implementation changed



#### `.TestQueryPluginCRM\_WithFilters`
**File:** `components/worker/internal/services/extract\_crm\_data\_test.go:1673-1716`
**Changes:** implementation changed



#### `.TestProcessPluginCRMCollection\_WithValidOrganization`
**File:** `components/worker/internal/services/extract\_crm\_data\_test.go:1749-1777`
**Changes:** implementation changed



#### `.TestProcessPluginCRMCollection\_WithOrganizationID`
**File:** `components/worker/internal/services/extract\_crm\_data\_test.go:1085-1139`
**Changes:** implementation changed



#### `.TestExtractJobIDFromMultipleSources\_EdgeCases`
**File:** `components/worker/internal/services/extract\_data\_test.go:590-686`
**Changes:** implementation changed



#### `.TestExtractJobIDFromMultipleSources\_FromHeaders`
**File:** `components/worker/internal/services/extract\_data\_test.go:141-166`
**Changes:** implementation changed



#### `.TestExtractJobIDFromPartialJSON`
**File:** `components/worker/internal/services/extract\_data\_test.go:502-587`
**Changes:** implementation changed



#### `.TestExtractJobIDFromMultipleSources\_FromPartialJSON`
**File:** `components/worker/internal/services/extract\_data\_test.go:169-192`
**Changes:** implementation changed



#### `.TestExtractJobIDFromMultipleSources\_NoIDs`
**File:** `components/worker/internal/services/extract\_data\_test.go:195-214`
**Changes:** implementation changed



#### `.TestExtractJobIDFromPartialJSON\_ValidJobIDInvalidOrgID`
**File:** `components/worker/internal/services/extract\_data\_test.go:1275-1296`
**Changes:** implementation changed



#### `.TestExtractJobIDFromPartialJSON\_RegexFallback`
**File:** `components/worker/internal/services/extract\_data\_test.go:981-1000`
**Changes:** implementation changed



#### `.TestExtractJobIDFromMultipleSources\_HeaderPrecedence`
**File:** `components/worker/internal/services/extract\_data\_test.go:1322-1344`
**Changes:** implementation changed



#### `.\*UseCase.publishJobNotification`
**File:** `components/worker/internal/services/job\_notification.go:76-167`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*UseCase\) \*UseCase.publishJobNotification\(ctx context.Context, tracer trace.Tracer, message ExtractExternalDataMessage, status string, errorMetadata map\[string\]any, opts \*JobNotificationOptions, logger log.Logger\) error
+ func \(\*UseCase\) \*UseCase.publishJobNotification\(ctx context.Context, tracer trace.Tracer, message ExtractExternalDataMessage, status string, errorMetadata map\[string\]any, opts \*JobNotificationOptions, logger libLog.Logger\) error
```


#### `.testLogger`
**File:** `components/worker/internal/services/test\_helpers\_test.go:33-35`
**Changes:** implementation changed



#### `.NewLoggerFromContext`
**File:** `pkg/context.go:21-28`
**Changes:** implementation changed



#### `.TestCustomContextKey\_Type`
**File:** `pkg/context\_test.go:286-304`
**Changes:** implementation changed



#### `.TestNewLoggerFromContext`
**File:** `pkg/context\_test.go:15-68`
**Changes:** implementation changed



#### `.TestContextWithLogger`
**File:** `pkg/context\_test.go:124-176`
**Changes:** implementation changed



#### `.TestContextWithTracer`
**File:** `pkg/context\_test.go:178-230`
**Changes:** implementation changed



#### `.TestContextWithLoggerAndTracer\_Combined`
**File:** `pkg/context\_test.go:232-264`
**Changes:** implementation changed



#### `.TestCustomContextKeyValue\_Integration`
**File:** `pkg/context\_test.go:266-284`
**Changes:** implementation changed



#### `.newDataSourceConfigMongoDB`
**File:** `pkg/datasource/datasource-factory.go:101-188`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func newDataSourceConfigMongoDB\(ctx context.Context, base datasource.DataSourceConfig, conn \*model.Connection, cryptor crypto.Cryptor, logger log.Logger\) \(\*datsourceMongoConfig.DataSourceConfigMongoDB, error\)
+ func newDataSourceConfigMongoDB\(ctx context.Context, base datasource.DataSourceConfig, conn \*model.Connection, cryptor crypto.Cryptor, logger libLog.Logger\) \(\*datsourceMongoConfig.DataSourceConfigMongoDB, error\)
```


#### `.newDataSourceConfigPostgres`
**File:** `pkg/datasource/datasource-factory.go:191-243`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func newDataSourceConfigPostgres\(ctx context.Context, base datasource.DataSourceConfig, conn \*model.Connection, cryptor crypto.Cryptor, logger log.Logger\) \(\*datsourcePostgresConfig.DataSourceConfigPostgres, error\)
+ func newDataSourceConfigPostgres\(ctx context.Context, base datasource.DataSourceConfig, conn \*model.Connection, cryptor crypto.Cryptor, logger libLog.Logger\) \(\*datsourcePostgresConfig.DataSourceConfigPostgres, error\)
```


#### `.newDataSourceConfigOracle`
**File:** `pkg/datasource/datasource-factory.go:246-310`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func newDataSourceConfigOracle\(ctx context.Context, base datasource.DataSourceConfig, conn \*model.Connection, cryptor crypto.Cryptor, logger log.Logger\) \(\*datsourceOracleConfig.DataSourceConfigOracle, error\)
+ func newDataSourceConfigOracle\(ctx context.Context, base datasource.DataSourceConfig, conn \*model.Connection, cryptor crypto.Cryptor, logger libLog.Logger\) \(\*datsourceOracleConfig.DataSourceConfigOracle, error\)
```


#### `.newDataSourceConfigMySQL`
**File:** `pkg/datasource/datasource-factory.go:313-365`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func newDataSourceConfigMySQL\(ctx context.Context, base datasource.DataSourceConfig, conn \*model.Connection, cryptor crypto.Cryptor, logger log.Logger\) \(\*datsourceMySQLConfig.DataSourceConfigMySQL, error\)
+ func newDataSourceConfigMySQL\(ctx context.Context, base datasource.DataSourceConfig, conn \*model.Connection, cryptor crypto.Cryptor, logger libLog.Logger\) \(\*datsourceMySQLConfig.DataSourceConfigMySQL, error\)
```


#### `.newDataSourceConfigSQLServer`
**File:** `pkg/datasource/datasource-factory.go:368-443`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func newDataSourceConfigSQLServer\(ctx context.Context, base datasource.DataSourceConfig, conn \*model.Connection, cryptor crypto.Cryptor, logger log.Logger\) \(\*datsourceSQLServerConfig.DataSourceConfigSQLServer, error\)
+ func newDataSourceConfigSQLServer\(ctx context.Context, base datasource.DataSourceConfig, conn \*model.Connection, cryptor crypto.Cryptor, logger libLog.Logger\) \(\*datsourceSQLServerConfig.DataSourceConfigSQLServer, error\)
```


#### `.NewDataSourceFromConnectionWithLogger`
**File:** `pkg/datasource/datasource-factory.go:452-456`
**Changes:** parameters changed, signature\_changed


```diff
- func NewDataSourceFromConnectionWithLogger\(logger log.Logger\) func\(ctx context.Context, conn \*model.Connection, cryptor crypto.Cryptor\) \(datasource.DataSource, error\)
+ func NewDataSourceFromConnectionWithLogger\(logger libLog.Logger\) func\(ctx context.Context, conn \*model.Connection, cryptor crypto.Cryptor\) \(datasource.DataSource, error\)
```


#### `.NewDataSourceFromConnection`
**File:** `pkg/datasource/datasource-factory.go:55-57`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func NewDataSourceFromConnection\(ctx context.Context, conn \*model.Connection, cryptor crypto.Cryptor, logger log.Logger\) \(datasource.DataSource, error\)
+ func NewDataSourceFromConnection\(ctx context.Context, conn \*model.Connection, cryptor crypto.Cryptor, logger libLog.Logger\) \(datasource.DataSource, error\)
```


#### `.buildImageWithSecrets`
**File:** `pkg/itestkit/addons/e2ekit/build\_secrets.go:40-114`
**Changes:** implementation changed



#### `.generateImageTag`
**File:** `pkg/itestkit/addons/e2ekit/build\_secrets.go:166-171`
**Changes:** implementation changed



#### `.New`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:43-57`
**Changes:** implementation changed



#### `.WaitLog`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:313-319`
**Changes:** implementation changed



#### `.WaitPort`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:330-336`
**Changes:** implementation changed



#### `.\*Builder.WithRewriter`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:125-131`
**Changes:** implementation changed



#### `.WaitHTTP`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:284-294`
**Changes:** implementation changed



#### `.localhostToHostGatewayRewriter.Rewrite`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:370-377`
**Changes:** implementation changed



#### `.\*Builder.WithImage`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:64-69`
**Changes:** implementation changed



#### `.\*Builder.Run`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:161-249`
**Changes:** implementation changed



#### `.uniqueAppend`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:458-466`
**Changes:** implementation changed



#### `.rewriteLocalhostForContainer`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:379-427`
**Changes:** implementation changed



#### `.dumpRecentLogs`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:429-456`
**Changes:** implementation changed



#### `.\*Builder.WithEnv`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:76-82`
**Changes:** implementation changed



#### `.\*Builder.ExposePort`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:94-105`
**Changes:** implementation changed



#### `.\*Builder.DisableDefaultLocalhostRewrite`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:133-146`
**Changes:** implementation changed



#### `.\*Builder.WithLogsOnFailureMaxBytes`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:153-159`
**Changes:** implementation changed



#### `.\*Builder.WithWait`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:117-123`
**Changes:** implementation changed



#### `.cloneMap`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:468-475`
**Changes:** implementation changed



#### `.WaitRunning`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:348-354`
**Changes:** implementation changed



#### `.waitHTTP.Configure`
**File:** `pkg/itestkit/addons/e2ekit/builder.go:296-306`
**Changes:** implementation changed



#### `.ProjectRoot`
**File:** `pkg/itestkit/addons/e2ekit/helpers.go:31-54`
**Changes:** implementation changed



#### `.ProjectRootFrom`
**File:** `pkg/itestkit/addons/e2ekit/helpers.go:61-81`
**Changes:** implementation changed



#### `.\*ChaosAssertions.TimeoutsBelow`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:159-176`
**Changes:** implementation changed



#### `.\*ChaosAssertions.ThroughputAbove`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:139-156`
**Changes:** implementation changed



#### `.\*ChaosAssertions.FailedResults`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:234-244`
**Changes:** implementation changed



#### `.\*ChaosAssertions.P50Below`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:99-116`
**Changes:** implementation changed



#### `.\*ChaosAssertions.MinRequestsReached`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:199-216`
**Changes:** implementation changed



#### `.\*ChaosAssertions.Summary`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:247-272`
**Changes:** implementation changed



#### `.\*ChaosAssertions.AverageLatencyBelow`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:119-136`
**Changes:** implementation changed



#### `.\*ChaosAssertions.FailuresBelow`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:179-196`
**Changes:** implementation changed



#### `.\*ChaosAssertions.SuccessRateAbove`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:39-56`
**Changes:** implementation changed



#### `.\*ChaosAssertions.P99Below`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:59-76`
**Changes:** implementation changed



#### `.\*ChaosAssertions.P95Below`
**File:** `pkg/itestkit/addons/metricskit/assertions.go:79-96`
**Changes:** implementation changed



#### `.\*ErrorClassifier.GetCategoryCounts`
**File:** `pkg/itestkit/addons/metricskit/error\_classifier.go:104-114`
**Changes:** implementation changed



#### `.\*ChaosMetrics.GetTotalRequests`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:134-139`
**Changes:** implementation changed



#### `.\*ChaosMetrics.SuccessRate`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:170-179`
**Changes:** implementation changed



#### `.\*ChaosMetrics.StartTest`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:51-56`
**Changes:** implementation changed



#### `.\*ChaosMetrics.StartChaos`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:67-72`
**Changes:** implementation changed



#### `.\*ChaosMetrics.GetErrorCounts`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:122-131`
**Changes:** implementation changed



#### `.\*ChaosMetrics.ChaosThroughputRPS`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:228-244`
**Changes:** implementation changed



#### `.\*ChaosMetrics.Percentile`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:272-310`
**Changes:** implementation changed



#### `.\*ChaosMetrics.EndChaos`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:75-80`
**Changes:** implementation changed



#### `.\*ChaosMetrics.RecordRequest`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:83-109`
**Changes:** implementation changed



#### `.\*ChaosMetrics.GetTimeoutRequests`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:150-155`
**Changes:** implementation changed



#### `.\*ChaosMetrics.SuccessfulThroughputRPS`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:211-225`
**Changes:** implementation changed



#### `.\*ChaosMetrics.ChaosDuration`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:247-256`
**Changes:** implementation changed



#### `.\*ChaosMetrics.TestDuration`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:259-268`
**Changes:** implementation changed



#### `.\*ChaosMetrics.GetFailedRequests`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:142-147`
**Changes:** implementation changed



#### `.\*ChaosMetrics.GetMinLatency`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:158-167`
**Changes:** implementation changed



#### `.\*ChaosMetrics.AverageLatency`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:182-191`
**Changes:** implementation changed



#### `.\*ChaosMetrics.ThroughputRPS`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:194-208`
**Changes:** implementation changed



#### `.\*ChaosMetrics.EndTest`
**File:** `pkg/itestkit/addons/metricskit/metrics.go:59-64`
**Changes:** implementation changed



#### `.\*Reporter.WriteReport`
**File:** `pkg/itestkit/addons/metricskit/reporter.go:25-80`
**Changes:** implementation changed



#### `.\*Reporter.String`
**File:** `pkg/itestkit/addons/metricskit/reporter.go:83-89`
**Changes:** implementation changed



#### `.\*Reporter.CompactSummary`
**File:** `pkg/itestkit/addons/metricskit/reporter.go:92-103`
**Changes:** implementation changed



#### `.\*AMQPConsumerBuilder.WithPrefetch`
**File:** `pkg/itestkit/addons/queuekit/amqp.go:420-426`
**Changes:** implementation changed



#### `.\*AMQPConsumer.Close`
**File:** `pkg/itestkit/addons/queuekit/amqp.go:199-228`
**Changes:** implementation changed



#### `.\*AMQPPublisher.connect`
**File:** `pkg/itestkit/addons/queuekit/amqp.go:270-297`
**Changes:** implementation changed



#### `.\*AMQPPublisher.Close`
**File:** `pkg/itestkit/addons/queuekit/amqp.go:342-371`
**Changes:** implementation changed



#### `.\*AMQPConsumerBuilder.BindTo`
**File:** `pkg/itestkit/addons/queuekit/amqp.go:400-405`
**Changes:** implementation changed



#### `.\*AMQPConsumerBuilder.WithQueueDeclare`
**File:** `pkg/itestkit/addons/queuekit/amqp.go:429-435`
**Changes:** implementation changed



#### `.NewAMQPConsumer`
**File:** `pkg/itestkit/addons/queuekit/amqp.go:56-73`
**Changes:** implementation changed



#### `.\*AMQPConsumer.connect`
**File:** `pkg/itestkit/addons/queuekit/amqp.go:76-145`
**Changes:** implementation changed



#### `.\*Assertions\[T\].HasHeaderKey`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:55-63`
**Changes:** implementation changed



#### `.\*ResultAssertions\[T\].At`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:214-223`
**Changes:** implementation changed



#### `.\*MessageSequence\[T\].FilterBy`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:327-337`
**Changes:** implementation changed



#### `.\*ExpectMessagesHelper\[T\].OrFatal`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:415-421`
**Changes:** implementation changed



#### `.\*Assertions\[T\].PayloadSatisfies`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:110-118`
**Changes:** implementation changed



#### `.\*ResultAssertions\[T\].UnmatchedCount`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:191-199`
**Changes:** implementation changed



#### `.\*Assertions\[T\].HasRoutingKey`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:27-35`
**Changes:** implementation changed



#### `.\*Assertions\[T\].HasMessageID`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:77-85`
**Changes:** implementation changed



#### `.\*Assertions\[T\].HasContentType`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:88-96`
**Changes:** implementation changed



#### `.\*ResultAssertions\[T\].First`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:202-211`
**Changes:** implementation changed



#### `.AssertResult`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:137-144`
**Changes:** implementation changed



#### `.AssertJSONEqual`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:265-271`
**Changes:** implementation changed



#### `.\*MessageSequence\[T\].GroupBy`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:340-349`
**Changes:** implementation changed



#### `.\*ExpectMessagesHelper\[T\].ToContainWhere`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:394-407`
**Changes:** implementation changed



#### `.AssertMessage`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:17-24`
**Changes:** implementation changed



#### `.\*Assertions\[T\].HasHeader`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:38-52`
**Changes:** implementation changed



#### `.\*ResultAssertions\[T\].HasAtLeast`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:158-166`
**Changes:** implementation changed



#### `.JSONEqual`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:247-262`
**Changes:** implementation changed



#### `.\*ExpectMessagesHelper\[T\].ToHaveCount`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:382-391`
**Changes:** implementation changed



#### `.\*Assertions\[T\].HasCorrelationID`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:66-74`
**Changes:** implementation changed



#### `.\*Assertions\[T\].PayloadEquals`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:99-107`
**Changes:** implementation changed



#### `.\*ResultAssertions\[T\].HasCount`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:147-155`
**Changes:** implementation changed



#### `.\*ResultAssertions\[T\].DidNotTimeout`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:180-188`
**Changes:** implementation changed



#### `.\*ResultAssertions\[T\].HasNoErrors`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:169-177`
**Changes:** implementation changed



#### `.\*ResultAssertions\[T\].All`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:226-234`
**Changes:** implementation changed



#### `.\*MessageSequence\[T\].RoutingKeysInOrder`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:300-307`
**Changes:** implementation changed



#### `.\*ExpectMessagesHelper\[T\].ToSucceed`
**File:** `pkg/itestkit/addons/queuekit/assertions.go:365-379`
**Changes:** implementation changed



#### `.\*Consumer\[T\].CaptureAll`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:203-256`
**Changes:** implementation changed



#### `.\*ConsumerBuilder\[T\].WithMatcher`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:50-56`
**Changes:** implementation changed



#### `.\*ConsumerBuilder\[T\].WithTimeout`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:68-74`
**Changes:** implementation changed



#### `.\*Consumer\[T\].DrainQueue`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:277-312`
**Changes:** implementation changed



#### `.\*Consumer\[T\].captureMessage`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:334-339`
**Changes:** implementation changed



#### `.NewConsumer`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:36-47`
**Changes:** implementation changed



#### `.truncateBody`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:352-358`
**Changes:** implementation changed



#### `.\*ConsumerBuilder\[T\].WithUnmarshaler`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:59-65`
**Changes:** implementation changed



#### `.\*ConsumerBuilder\[T\].Build`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:83-95`
**Changes:** implementation changed



#### `.\*Consumer\[T\].WaitForMessages`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:123-199`
**Changes:** implementation changed



#### `.\*Consumer\[T\].GetCaptured`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:315-323`
**Changes:** implementation changed



#### `.\*Consumer\[T\].ClearCaptured`
**File:** `pkg/itestkit/addons/queuekit/consumer.go:326-331`
**Changes:** implementation changed



#### `.compareValues`
**File:** `pkg/itestkit/addons/queuekit/matcher.go:248-274`
**Changes:** implementation changed



#### `.MatchHeader`
**File:** `pkg/itestkit/addons/queuekit/matcher.go:89-102`
**Changes:** implementation changed



#### `.MatchHeaderExists`
**File:** `pkg/itestkit/addons/queuekit/matcher.go:105-115`
**Changes:** implementation changed



#### `.MatchJSONField`
**File:** `pkg/itestkit/addons/queuekit/matcher.go:150-161`
**Changes:** implementation changed



#### `.MatchJSONFieldPattern`
**File:** `pkg/itestkit/addons/queuekit/matcher.go:176-197`
**Changes:** implementation changed



#### `.MatchBodyPattern`
**File:** `pkg/itestkit/addons/queuekit/matcher.go:139-145`
**Changes:** implementation changed



#### `.hasNestedValue`
**File:** `pkg/itestkit/addons/queuekit/matcher.go:223-245`
**Changes:** implementation changed



#### `.MatchAll`
**File:** `pkg/itestkit/addons/queuekit/matcher.go:15-25`
**Changes:** implementation changed



#### `.MatchAny`
**File:** `pkg/itestkit/addons/queuekit/matcher.go:28-42`
**Changes:** implementation changed



#### `.MatchRoutingKeyPattern`
**File:** `pkg/itestkit/addons/queuekit/matcher.go:80-86`
**Changes:** implementation changed



#### `.applyPublishOptions`
**File:** `pkg/itestkit/addons/queuekit/queuekit.go:128-137`
**Changes:** implementation changed



#### `.WaitResult\[T\].First`
**File:** `pkg/itestkit/addons/queuekit/queuekit.go:158-165`
**Changes:** implementation changed



#### `.\*toxiproxyChaos.RemoveToxic`
**File:** `pkg/itestkit/chaos\_toxiproxy.go:214-221`
**Changes:** implementation changed



#### `.\*toxiproxyChaos.RemoveAllToxics`
**File:** `pkg/itestkit/chaos\_toxiproxy.go:223-246`
**Changes:** implementation changed



#### `.\*toxiproxyChaos.Close`
**File:** `pkg/itestkit/chaos\_toxiproxy.go:248-258`
**Changes:** implementation changed



#### `.NewToxiproxyChaos`
**File:** `pkg/itestkit/chaos\_toxiproxy.go:36-96`
**Changes:** implementation changed



#### `.\*toxiproxyChaos.CreateProxy`
**File:** `pkg/itestkit/chaos\_toxiproxy.go:98-135`
**Changes:** implementation changed



#### `.\*toxiproxyChaos.AddBandwidth`
**File:** `pkg/itestkit/chaos\_toxiproxy.go:195-212`
**Changes:** implementation changed



#### `.\*toxiproxyChaos.AddLatency`
**File:** `pkg/itestkit/chaos\_toxiproxy.go:137-155`
**Changes:** implementation changed



#### `.\*toxiproxyChaos.CutConnection`
**File:** `pkg/itestkit/chaos\_toxiproxy.go:157-174`
**Changes:** implementation changed



#### `.\*toxiproxyChaos.AddTimeout`
**File:** `pkg/itestkit/chaos\_toxiproxy.go:176-193`
**Changes:** implementation changed



#### `.WaitListeningPort.Apply`
**File:** `pkg/itestkit/container\_generic.go:46-52`
**Changes:** implementation changed



#### `.\*Builder.WithContainerCustomize`
**File:** `pkg/itestkit/container\_generic.go:54-62`
**Changes:** implementation changed



#### `.\*genericContainerInfra.Start`
**File:** `pkg/itestkit/container\_generic.go:77-158`
**Changes:** implementation changed



#### `.\*genericContainerInfra.Terminate`
**File:** `pkg/itestkit/container\_generic.go:160-166`
**Changes:** implementation changed



#### `.portKey`
**File:** `pkg/itestkit/container\_generic.go:168-176`
**Changes:** implementation changed



#### `.CustomizerFunc.Customize`
**File:** `pkg/itestkit/customizer.go:9-15`
**Changes:** implementation changed



#### `.MergeCustomizers`
**File:** `pkg/itestkit/customizer.go:17-29`
**Changes:** implementation changed



#### `.CExposedPorts`
**File:** `pkg/itestkit/customizer\_options.go:29-34`
**Changes:** implementation changed



#### `.CEnvFromOS`
**File:** `pkg/itestkit/customizer\_options.go:83-90`
**Changes:** implementation changed



#### `.uniqueAppendMany`
**File:** `pkg/itestkit/customizer\_options.go:131-151`
**Changes:** implementation changed



#### `.CBindMount`
**File:** `pkg/itestkit/customizer\_options.go:122-129`
**Changes:** implementation changed



#### `.CEnvs`
**File:** `pkg/itestkit/customizer\_options.go:18-25`
**Changes:** implementation changed



#### `.CHostDockerInternal`
**File:** `pkg/itestkit/customizer\_options.go:61-70`
**Changes:** implementation changed



#### `.CNetworks`
**File:** `pkg/itestkit/customizer\_options.go:98-103`
**Changes:** implementation changed



#### `.CAll`
**File:** `pkg/itestkit/customizer\_options.go:72-81`
**Changes:** implementation changed



#### `.CNetworkAliases`
**File:** `pkg/itestkit/customizer\_options.go:105-120`
**Changes:** implementation changed



#### `.HostGatewayIP`
**File:** `pkg/itestkit/hostport.go:28-74`
**Changes:** implementation changed



#### `.validateUniqueInfraNames`
**File:** `pkg/itestkit/infra.go:19-44`
**Changes:** implementation changed



#### `.NewMongoDBInfra`
**File:** `pkg/itestkit/infra/mongodb/mongodb.go:42-56`
**Changes:** implementation changed



#### `.\*MongoDBInfra.HostPort`
**File:** `pkg/itestkit/infra/mongodb/mongodb.go:171-183`
**Changes:** implementation changed



#### `.NewMongoDBInfraStub`
**File:** `pkg/itestkit/infra/mongodb/mongodb.go:209-229`
**Changes:** implementation changed



#### `.\*MongoDBInfra.Start`
**File:** `pkg/itestkit/infra/mongodb/mongodb.go:58-150`
**Changes:** implementation changed



#### `.\*MongoDBInfra.Endpoint`
**File:** `pkg/itestkit/infra/mongodb/mongodb.go:152-158`
**Changes:** implementation changed



#### `.\*MongoDBInfra.URI`
**File:** `pkg/itestkit/infra/mongodb/mongodb.go:160-167`
**Changes:** implementation changed



#### `.\*MongoDBInfra.Terminate`
**File:** `pkg/itestkit/infra/mongodb/mongodb.go:195-201`
**Changes:** implementation changed



#### `.WithMongoDBFixedPort`
**File:** `pkg/itestkit/infra/mongodb/mongodb\_options.go:37-51`
**Changes:** implementation changed



#### `.\*MSSQLInfra.Endpoint`
**File:** `pkg/itestkit/infra/mssql/mssql.go:147-153`
**Changes:** implementation changed



#### `.NewMSSQLInfraStub`
**File:** `pkg/itestkit/infra/mssql/mssql.go:224-242`
**Changes:** implementation changed



#### `.NewMSSQLInfra`
**File:** `pkg/itestkit/infra/mssql/mssql.go:43-61`
**Changes:** implementation changed



#### `.\*MSSQLInfra.Start`
**File:** `pkg/itestkit/infra/mssql/mssql.go:63-145`
**Changes:** implementation changed



#### `.\*MSSQLInfra.DSN`
**File:** `pkg/itestkit/infra/mssql/mssql.go:155-162`
**Changes:** implementation changed



#### `.\*MSSQLInfra.HostPort`
**File:** `pkg/itestkit/infra/mssql/mssql.go:168-208`
**Changes:** implementation changed



#### `.\*MSSQLInfra.Terminate`
**File:** `pkg/itestkit/infra/mssql/mssql.go:210-216`
**Changes:** implementation changed



#### `.WithMSSQLFixedPort`
**File:** `pkg/itestkit/infra/mssql/mssql\_options.go:34-48`
**Changes:** implementation changed



#### `.NewMySQLInfra`
**File:** `pkg/itestkit/infra/mysql/mysql.go:45-75`
**Changes:** implementation changed



#### `.\*MySQLInfra.Endpoint`
**File:** `pkg/itestkit/infra/mysql/mysql.go:148-154`
**Changes:** implementation changed



#### `.\*MySQLInfra.Terminate`
**File:** `pkg/itestkit/infra/mysql/mysql.go:211-217`
**Changes:** implementation changed



#### `.\*MySQLInfra.Start`
**File:** `pkg/itestkit/infra/mysql/mysql.go:77-146`
**Changes:** implementation changed



#### `.\*MySQLInfra.DSN`
**File:** `pkg/itestkit/infra/mysql/mysql.go:156-163`
**Changes:** implementation changed



#### `.\*MySQLInfra.HostPort`
**File:** `pkg/itestkit/infra/mysql/mysql.go:169-209`
**Changes:** implementation changed



#### `.NewMySQLInfraStub`
**File:** `pkg/itestkit/infra/mysql/mysql.go:225-237`
**Changes:** implementation changed



#### `.WithMySQLInitScript`
**File:** `pkg/itestkit/infra/mysql/mysql\_options.go:31-47`
**Changes:** implementation changed



#### `.WithMySQLFixedPort`
**File:** `pkg/itestkit/infra/mysql/mysql\_options.go:52-66`
**Changes:** implementation changed



#### `.\*OracleInfra.Endpoint`
**File:** `pkg/itestkit/infra/oracle/oracle.go:159-165`
**Changes:** implementation changed



#### `.NewOracleInfraStub`
**File:** `pkg/itestkit/infra/oracle/oracle.go:250-262`
**Changes:** implementation changed



#### `.\*OracleInfra.DSN`
**File:** `pkg/itestkit/infra/oracle/oracle.go:167-174`
**Changes:** implementation changed



#### `.\*OracleInfra.GoDRORDSN`
**File:** `pkg/itestkit/infra/oracle/oracle.go:176-188`
**Changes:** implementation changed



#### `.\*OracleInfra.HostPort`
**File:** `pkg/itestkit/infra/oracle/oracle.go:194-234`
**Changes:** implementation changed



#### `.\*OracleInfra.Terminate`
**File:** `pkg/itestkit/infra/oracle/oracle.go:236-242`
**Changes:** implementation changed



#### `.NewOracleInfra`
**File:** `pkg/itestkit/infra/oracle/oracle.go:44-66`
**Changes:** implementation changed



#### `.\*OracleInfra.Start`
**File:** `pkg/itestkit/infra/oracle/oracle.go:68-157`
**Changes:** implementation changed



#### `.WithOracleInitScript`
**File:** `pkg/itestkit/infra/oracle/oracle\_options.go:31-47`
**Changes:** implementation changed



#### `.WithOracleFixedPort`
**File:** `pkg/itestkit/infra/oracle/oracle\_options.go:52-66`
**Changes:** implementation changed



#### `.\*PostgresInfra.Start`
**File:** `pkg/itestkit/infra/postgres/postgres.go:71-152`
**Changes:** implementation changed



#### `.\*PostgresInfra.DSN`
**File:** `pkg/itestkit/infra/postgres/postgres.go:162-169`
**Changes:** implementation changed



#### `.\*PostgresInfra.HostPort`
**File:** `pkg/itestkit/infra/postgres/postgres.go:173-185`
**Changes:** implementation changed



#### `.NewPostgresInfra`
**File:** `pkg/itestkit/infra/postgres/postgres.go:43-69`
**Changes:** implementation changed



#### `.\*PostgresInfra.Endpoint`
**File:** `pkg/itestkit/infra/postgres/postgres.go:154-160`
**Changes:** implementation changed



#### `.\*PostgresInfra.Terminate`
**File:** `pkg/itestkit/infra/postgres/postgres.go:197-203`
**Changes:** implementation changed



#### `.NewPostgresInfraStub`
**File:** `pkg/itestkit/infra/postgres/postgres.go:211-223`
**Changes:** implementation changed



#### `.WithPGInitFile`
**File:** `pkg/itestkit/infra/postgres/postgres\_options.go:37-53`
**Changes:** implementation changed



#### `.WithPGFixedPort`
**File:** `pkg/itestkit/infra/postgres/postgres\_options.go:58-72`
**Changes:** implementation changed



#### `.WithRabbitFixedPort`
**File:** `pkg/itestkit/infra/rabbitmq/rabbit\_options.go:39-53`
**Changes:** implementation changed



#### `.\*configReader.Read`
**File:** `pkg/itestkit/infra/rabbitmq/rabbit\_options.go:85-94`
**Changes:** implementation changed



#### `.NewRabbitInfra`
**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go:40-66`
**Changes:** implementation changed



#### `.\*RabbitInfra.Start`
**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go:68-147`
**Changes:** implementation changed



#### `.\*RabbitInfra.Endpoint`
**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go:149-155`
**Changes:** implementation changed



#### `.\*RabbitInfra.AMQPURL`
**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go:157-164`
**Changes:** implementation changed



#### `.\*RabbitInfra.HostPort`
**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go:168-175`
**Changes:** implementation changed



#### `.\*RabbitInfra.Terminate`
**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go:187-193`
**Changes:** implementation changed



#### `.NewRedisInfra`
**File:** `pkg/itestkit/infra/redis/redis.go:39-53`
**Changes:** implementation changed



#### `.\*RedisInfra.Start`
**File:** `pkg/itestkit/infra/redis/redis.go:55-138`
**Changes:** implementation changed



#### `.\*RedisInfra.HostPort`
**File:** `pkg/itestkit/infra/redis/redis.go:172-179`
**Changes:** implementation changed



#### `.\*RedisInfra.Terminate`
**File:** `pkg/itestkit/infra/redis/redis.go:191-197`
**Changes:** implementation changed



#### `.\*RedisInfra.Endpoint`
**File:** `pkg/itestkit/infra/redis/redis.go:140-146`
**Changes:** implementation changed



#### `.\*RedisInfra.URL`
**File:** `pkg/itestkit/infra/redis/redis.go:148-155`
**Changes:** implementation changed



#### `.\*RedisInfra.Addr`
**File:** `pkg/itestkit/infra/redis/redis.go:157-168`
**Changes:** implementation changed



#### `.WithRedisFixedPort`
**File:** `pkg/itestkit/infra/redis/redis\_options.go:37-51`
**Changes:** implementation changed



#### `.NewSeaweedFSInfra`
**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go:59-77`
**Changes:** implementation changed



#### `.\*SeaweedFSInfra.Start`
**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go:79-224`
**Changes:** implementation changed



#### `.\*SeaweedFSInfra.Endpoint`
**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go:227-233`
**Changes:** implementation changed



#### `.\*SeaweedFSInfra.URL`
**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go:236-243`
**Changes:** implementation changed



#### `.\*SeaweedFSInfra.HostPort`
**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go:247-260`
**Changes:** implementation changed



#### `.\*SeaweedFSInfra.Terminate`
**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go:272-304`
**Changes:** implementation changed



#### `.WithSeaweedFSFixedPort`
**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs\_options.go:19-31`
**Changes:** implementation changed



#### `.\*Builder.Build`
**File:** `pkg/itestkit/suite.go:72-122`
**Changes:** implementation changed



#### `.\*Suite.Terminate`
**File:** `pkg/itestkit/suite.go:128-142`
**Changes:** implementation changed



#### `.New`
**File:** `pkg/itestkit/suite.go:33-45`
**Changes:** implementation changed



#### `.\*Builder.WithInfra`
**File:** `pkg/itestkit/suite.go:47-53`
**Changes:** implementation changed



#### `.\*Builder.WithInfras`
**File:** `pkg/itestkit/suite.go:55-65`
**Changes:** implementation changed



#### `.\*Suite.Network`
**File:** `pkg/itestkit/suite.go:146-152`
**Changes:** implementation changed



#### `.\*DataSourceConfigMongoDB.GetSchemaInfo`
**File:** `pkg/model/datasource/mongodb/datasource-config.go:99-130`
**Changes:** implementation changed



#### `.\*DataSourceConfigMongoDB.Connect`
**File:** `pkg/model/datasource/mongodb/datasource-config.go:40-45`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*DataSourceConfigMongoDB\) \*DataSourceConfigMongoDB.Connect\(ctx context.Context, logger log.Logger\) error
+ func \(\*DataSourceConfigMongoDB\) \*DataSourceConfigMongoDB.Connect\(ctx context.Context, logger libLog.Logger\) error
```


#### `.\*DataSourceConfigMongoDB.Query`
**File:** `pkg/model/datasource/mongodb/datasource-config.go:61-87`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*DataSourceConfigMongoDB\) \*DataSourceConfigMongoDB.Query\(ctx context.Context, collections map\[string\]\[\]string, filters map\[string\]map\[string\]job.FilterCondition, logger log.Logger\) \(map\[string\]\[\]map\[string\]any, error\)
+ func \(\*DataSourceConfigMongoDB\) \*DataSourceConfigMongoDB.Query\(ctx context.Context, collections map\[string\]\[\]string, filters map\[string\]map\[string\]job.FilterCondition, logger libLog.Logger\) \(map\[string\]\[\]map\[string\]any, error\)
```


#### `.\*DataSourceConfigMySQL.Connect`
**File:** `pkg/model/datasource/mysql/datasource-config.go:39-44`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*DataSourceConfigMySQL\) \*DataSourceConfigMySQL.Connect\(ctx context.Context, logger log.Logger\) error
+ func \(\*DataSourceConfigMySQL\) \*DataSourceConfigMySQL.Connect\(ctx context.Context, logger libLog.Logger\) error
```


#### `.\*DataSourceConfigMySQL.Query`
**File:** `pkg/model/datasource/mysql/datasource-config.go:60-94`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*DataSourceConfigMySQL\) \*DataSourceConfigMySQL.Query\(ctx context.Context, tables map\[string\]\[\]string, filters map\[string\]map\[string\]job.FilterCondition, logger log.Logger\) \(map\[string\]\[\]map\[string\]any, error\)
+ func \(\*DataSourceConfigMySQL\) \*DataSourceConfigMySQL.Query\(ctx context.Context, tables map\[string\]\[\]string, filters map\[string\]map\[string\]job.FilterCondition, logger libLog.Logger\) \(map\[string\]\[\]map\[string\]any, error\)
```


#### `.\*DataSourceConfigMySQL.GetSchemaInfo`
**File:** `pkg/model/datasource/mysql/datasource-config.go:106-137`
**Changes:** implementation changed



#### `.\*DataSourceConfigOracle.Connect`
**File:** `pkg/model/datasource/oracle/datasource-config.go:40-45`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*DataSourceConfigOracle\) \*DataSourceConfigOracle.Connect\(ctx context.Context, logger log.Logger\) error
+ func \(\*DataSourceConfigOracle\) \*DataSourceConfigOracle.Connect\(ctx context.Context, logger libLog.Logger\) error
```


#### `.\*DataSourceConfigOracle.Query`
**File:** `pkg/model/datasource/oracle/datasource-config.go:61-102`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*DataSourceConfigOracle\) \*DataSourceConfigOracle.Query\(ctx context.Context, tables map\[string\]\[\]string, filters map\[string\]map\[string\]job.FilterCondition, logger log.Logger\) \(map\[string\]\[\]map\[string\]any, error\)
+ func \(\*DataSourceConfigOracle\) \*DataSourceConfigOracle.Query\(ctx context.Context, tables map\[string\]\[\]string, filters map\[string\]map\[string\]job.FilterCondition, logger libLog.Logger\) \(map\[string\]\[\]map\[string\]any, error\)
```


#### `.\*DataSourceConfigOracle.GetSchemaInfo`
**File:** `pkg/model/datasource/oracle/datasource-config.go:114-149`
**Changes:** implementation changed



#### `.\*DataSourceConfigPostgres.Connect`
**File:** `pkg/model/datasource/postgres/datasource-config.go:40-45`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*DataSourceConfigPostgres\) \*DataSourceConfigPostgres.Connect\(ctx context.Context, logger log.Logger\) error
+ func \(\*DataSourceConfigPostgres\) \*DataSourceConfigPostgres.Connect\(ctx context.Context, logger libLog.Logger\) error
```


#### `.\*DataSourceConfigPostgres.GetSchemaInfo`
**File:** `pkg/model/datasource/postgres/datasource-config.go:108-140`
**Changes:** implementation changed



#### `.\*DataSourceConfigPostgres.Query`
**File:** `pkg/model/datasource/postgres/datasource-config.go:61-105`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*DataSourceConfigPostgres\) \*DataSourceConfigPostgres.Query\(ctx context.Context, tables map\[string\]\[\]string, filters map\[string\]map\[string\]job.FilterCondition, logger log.Logger\) \(map\[string\]\[\]map\[string\]any, error\)
+ func \(\*DataSourceConfigPostgres\) \*DataSourceConfigPostgres.Query\(ctx context.Context, tables map\[string\]\[\]string, filters map\[string\]map\[string\]job.FilterCondition, logger libLog.Logger\) \(map\[string\]\[\]map\[string\]any, error\)
```


#### `.\*DataSourceConfigSQLServer.Query`
**File:** `pkg/model/datasource/sqlserver/datasource-config.go:64-101`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*DataSourceConfigSQLServer\) \*DataSourceConfigSQLServer.Query\(ctx context.Context, tables map\[string\]\[\]string, filters map\[string\]map\[string\]job.FilterCondition, logger log.Logger\) \(map\[string\]\[\]map\[string\]any, error\)
+ func \(\*DataSourceConfigSQLServer\) \*DataSourceConfigSQLServer.Query\(ctx context.Context, tables map\[string\]\[\]string, filters map\[string\]map\[string\]job.FilterCondition, logger libLog.Logger\) \(map\[string\]\[\]map\[string\]any, error\)
```


#### `.\*DataSourceConfigSQLServer.GetSchemaInfo`
**File:** `pkg/model/datasource/sqlserver/datasource-config.go:135-166`
**Changes:** implementation changed



#### `.\*DataSourceConfigSQLServer.Connect`
**File:** `pkg/model/datasource/sqlserver/datasource-config.go:43-48`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*DataSourceConfigSQLServer\) \*DataSourceConfigSQLServer.Connect\(ctx context.Context, logger log.Logger\) error
+ func \(\*DataSourceConfigSQLServer\) \*DataSourceConfigSQLServer.Connect\(ctx context.Context, logger libLog.Logger\) error
```


#### `.\*ConnectionMongoDBRepository.ListUnassigned`
**File:** `pkg/mongodb/connection/connection.mongodb.go:625-704`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.AssignProduct`
**File:** `pkg/mongodb/connection/connection.mongodb.go:707-763`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.FindByID`
**File:** `pkg/mongodb/connection/connection.mongodb.go:310-355`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.FindByOrganizationAndName`
**File:** `pkg/mongodb/connection/connection.mongodb.go:358-407`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.FindByOrganizationAndDatabaseName`
**File:** `pkg/mongodb/connection/connection.mongodb.go:410-463`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.CountByProduct`
**File:** `pkg/mongodb/connection/connection.mongodb.go:766-801`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.buildQueryFilter`
**File:** `pkg/mongodb/connection/connection.mongodb.go:804-827`
**Changes:** implementation changed



#### `.NewConnectionMongoDBRepository`
**File:** `pkg/mongodb/connection/connection.mongodb.go:74-103`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func NewConnectionMongoDBRepository\(mc \*libMongo.MongoConnection, cfg ...RepositoryConfig\) \(\*ConnectionMongoDBRepository, error\)
+ func NewConnectionMongoDBRepository\(mc \*libMongo.Client, database string, cfg ...RepositoryConfig\) \(\*ConnectionMongoDBRepository, error\)
```


#### `.\*ConnectionMongoDBRepository.Create`
**File:** `pkg/mongodb/connection/connection.mongodb.go:106-161`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.Update`
**File:** `pkg/mongodb/connection/connection.mongodb.go:164-255`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.Delete`
**File:** `pkg/mongodb/connection/connection.mongodb.go:258-307`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.FindByConfigNames`
**File:** `pkg/mongodb/connection/connection.mongodb.go:466-546`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.List`
**File:** `pkg/mongodb/connection/connection.mongodb.go:549-622`
**Changes:** implementation changed



#### `.TestConnectionMongoDBRepository\_Create`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:153-230`
**Changes:** implementation changed



#### `.newConnectionRepository`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:62-73`
**Changes:** implementation changed



#### `.clearConnectionsCollection`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:75-92`
**Changes:** implementation changed



#### `.TestConnectionMongoDBRepository\_Update`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:232-353`
**Changes:** implementation changed



#### `.TestConnectionMongoDBRepository\_FindByID`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:418-467`
**Changes:** implementation changed



#### `.TestConnectionMongoDBRepository\_EnsureIndexes`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:720-756`
**Changes:** implementation changed



#### `.TestConnectionMongoDBRepository\_Delete`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:355-416`
**Changes:** implementation changed



#### `.TestConnectionMongoDBRepository\_FindByOrganizationAndDatabaseName`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:520-581`
**Changes:** implementation changed



#### `.TestConnectionMongoDBRepository\_FindByConfigNames`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:583-718`
**Changes:** implementation changed



#### `.TestConnectionMongoDBRepository\_DropIndexes`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:758-788`
**Changes:** implementation changed



#### `.TestConnectionMongoDBRepository\_List`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:820-915`
**Changes:** implementation changed



#### `.TestMain`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:36-60`
**Changes:** implementation changed



#### `.stubConnectionSpanAttributes`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:132-141`
**Changes:** implementation changed



#### `.TestConnectionMongoDBRepository\_FindByOrganizationAndName`
**File:** `pkg/mongodb/connection/connection.mongodb\_test.go:469-518`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.EnsureIndexes`
**File:** `pkg/mongodb/connection/indexes.go:27-141`
**Changes:** implementation changed



#### `.\*ConnectionMongoDBRepository.DropIndexes`
**File:** `pkg/mongodb/connection/indexes.go:144-180`
**Changes:** implementation changed



#### `.NewDataSourceRepository`
**File:** `pkg/mongodb/datasource.mongodb.go:56-73`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func NewDataSourceRepository\(mongoURI string, dbName string, logger log.Logger\) \(\*ExternalDataSource, error\)
+ func NewDataSourceRepository\(mongoURI string, dbName string, logger libLog.Logger\) \(\*ExternalDataSource, error\)
```


#### `.\*ExternalDataSource.Query`
**File:** `pkg/mongodb/datasource.mongodb.go:100-190`
**Changes:** implementation changed



#### `.\*ExternalDataSource.QueryWithAdvancedFilters`
**File:** `pkg/mongodb/datasource.mongodb.go:582-623`
**Changes:** implementation changed



#### `.\*ExternalDataSource.GetDatabaseSchema`
**File:** `pkg/mongodb/datasource.mongodb.go:259-341`
**Changes:** implementation changed



#### `.\*ExternalDataSource.processQueryResults`
**File:** `pkg/mongodb/datasource.mongodb.go:690-718`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.processQueryResults\(queryCtx context.Context, cursor \*mongo.Cursor, collection string, logger log.Logger\) \(\[\]map\[string\]any, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.processQueryResults\(queryCtx context.Context, cursor \*mongo.Cursor, collection string, logger libLog.Logger\) \(\[\]map\[string\]any, error\)
```


#### `.\*ExternalDataSource.CloseConnection`
**File:** `pkg/mongodb/datasource.mongodb.go:76-97`
**Changes:** implementation changed



#### `.testLogger`
**File:** `pkg/mongodb/datasource.mongodb\_test.go:1139-1141`
**Changes:** implementation changed



#### `.\*JobMongoDBRepository.EnsureIndexes`
**File:** `pkg/mongodb/job/indexes.go:27-121`
**Changes:** implementation changed



#### `.\*JobMongoDBRepository.DropIndexes`
**File:** `pkg/mongodb/job/indexes.go:124-160`
**Changes:** implementation changed



#### `.\*JobMongoDBRepository.Create`
**File:** `pkg/mongodb/job/job.mongodb.go:103-170`
**Changes:** implementation changed



#### `.\*JobMongoDBRepository.UpdateStatus`
**File:** `pkg/mongodb/job/job.mongodb.go:248-337`
**Changes:** implementation changed



#### `.\*JobMongoDBRepository.FindByRequestHashWithinWindow`
**File:** `pkg/mongodb/job/job.mongodb.go:390-450`
**Changes:** implementation changed



#### `.NewJobMongoDBRepository`
**File:** `pkg/mongodb/job/job.mongodb.go:71-100`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func NewJobMongoDBRepository\(mc \*libMongo.MongoConnection, cfg ...RepositoryConfig\) \(\*JobMongoDBRepository, error\)
+ func NewJobMongoDBRepository\(mc \*libMongo.Client, database string, cfg ...RepositoryConfig\) \(\*JobMongoDBRepository, error\)
```


#### `.\*JobMongoDBRepository.Update`
**File:** `pkg/mongodb/job/job.mongodb.go:173-245`
**Changes:** implementation changed



#### `.\*JobMongoDBRepository.ExistsRunningByMappedFieldKey`
**File:** `pkg/mongodb/job/job.mongodb.go:454-509`
**Changes:** implementation changed



#### `.\*JobMongoDBRepository.scanJobs`
**File:** `pkg/mongodb/job/job.mongodb.go:654-679`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*JobMongoDBRepository\) \*JobMongoDBRepository.scanJobs\(ctx context.Context, cur \*mongo.Cursor, span \*trace.Span, limit int\) \(\[\]\*model.Job, error\)
+ func \(\*JobMongoDBRepository\) \*JobMongoDBRepository.scanJobs\(ctx context.Context, cur \*mongo.Cursor, span trace.Span, limit int\) \(\[\]\*model.Job, error\)
```


#### `.\*JobMongoDBRepository.FindByID`
**File:** `pkg/mongodb/job/job.mongodb.go:340-385`
**Changes:** implementation changed



#### `.\*JobMongoDBRepository.List`
**File:** `pkg/mongodb/job/job.mongodb.go:512-556`
**Changes:** implementation changed



#### `.TestJobMongoDBRepository\_List`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:350-439`
**Changes:** implementation changed



#### `.TestJobMongoDBRepository\_ExistsRunningByMappedFieldKey`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:718-877`
**Changes:** implementation changed



#### `.TestJobMongoDBRepository\_Update`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:218-305`
**Changes:** implementation changed



#### `.TestJobMongoDBRepository\_FindByID`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:307-348`
**Changes:** implementation changed



#### `.TestJobMongoDBRepository\_UpdateStatus`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:441-585`
**Changes:** implementation changed



#### `.TestDropIndexesDatabaseError`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:954-970`
**Changes:** implementation changed



#### `.stubJobSpanAttributes`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:120-129`
**Changes:** implementation changed



#### `.TestJobMongoDBRepository\_FindByRequestHashWithinWindow`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:587-716`
**Changes:** implementation changed



#### `.TestRepositoryConstructorValidatesDB`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:972-980`
**Changes:** implementation changed



#### `.TestMain`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:35-59`
**Changes:** implementation changed



#### `.newJobRepository`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:61-69`
**Changes:** implementation changed



#### `.TestEnsureIndexesDatabaseError`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:936-952`
**Changes:** implementation changed



#### `.clearJobsCollection`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:71-88`
**Changes:** implementation changed



#### `.TestJobMongoDBRepository\_Create`
**File:** `pkg/mongodb/job/job.mongodb\_test.go:141-216`
**Changes:** implementation changed



#### `.MapMongoErrorToResponse`
**File:** `pkg/mongodb/mongo.go:44-136`
**Changes:** implementation changed



#### `.PingMongo`
**File:** `pkg/mongodb/mongo.go:149-171`
**Changes:** implementation changed



#### `.TestPingMongo`
**File:** `pkg/mongodb/mongo\_test.go:279-327`
**Changes:** implementation changed



#### `.\*ProductMongoDBRepository.EnsureIndexes`
**File:** `pkg/mongodb/product/indexes.go:27-95`
**Changes:** implementation changed



#### `.\*ProductMongoDBRepository.DropIndexes`
**File:** `pkg/mongodb/product/indexes.go:98-134`
**Changes:** implementation changed



#### `.\*ProductMongoDBRepository.List`
**File:** `pkg/mongodb/product/product.mongodb.go:387-460`
**Changes:** implementation changed



#### `.\*ProductMongoDBRepository.buildQueryFilter`
**File:** `pkg/mongodb/product/product.mongodb.go:463-482`
**Changes:** implementation changed



#### `.NewProductMongoDBRepository`
**File:** `pkg/mongodb/product/product.mongodb.go:65-71`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func NewProductMongoDBRepository\(mc \*libMongo.MongoConnection, cfg ...RepositoryConfig\) \(\*ProductMongoDBRepository, error\)
+ func NewProductMongoDBRepository\(mc \*libMongo.Client, database string, cfg ...RepositoryConfig\) \(\*ProductMongoDBRepository, error\)
```


#### `.\*ProductMongoDBRepository.Create`
**File:** `pkg/mongodb/product/product.mongodb.go:105-165`
**Changes:** implementation changed



#### `.\*ProductMongoDBRepository.Update`
**File:** `pkg/mongodb/product/product.mongodb.go:168-236`
**Changes:** implementation changed



#### `.\*ProductMongoDBRepository.Delete`
**File:** `pkg/mongodb/product/product.mongodb.go:239-288`
**Changes:** implementation changed



#### `.\*ProductMongoDBRepository.FindByID`
**File:** `pkg/mongodb/product/product.mongodb.go:291-336`
**Changes:** implementation changed



#### `.\*ProductMongoDBRepository.FindByCode`
**File:** `pkg/mongodb/product/product.mongodb.go:339-384`
**Changes:** implementation changed



#### `.parseJSONField`
**File:** `pkg/mysql/datasource.mysql.go:365-396`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func parseJSONField\(value any, logger log.Logger\) any
+ func parseJSONField\(value any, logger libLog.Logger\) any
```


#### `.\*ExternalDataSource.GetDatabaseSchema`
**File:** `pkg/mysql/datasource.mysql.go:146-179`
**Changes:** implementation changed



#### `.\*ExternalDataSource.buildSchema`
**File:** `pkg/mysql/datasource.mysql.go:253-266`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.buildSchema\(ctx context.Context, tables \[\]string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger\) \(\[\]TableSchema, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.buildSchema\(ctx context.Context, tables \[\]string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger\) \(\[\]TableSchema, error\)
```


#### `.\*ExternalDataSource.CloseConnection`
**File:** `pkg/mysql/datasource.mysql.go:67-83`
**Changes:** implementation changed



#### `.\*ExternalDataSource.scanColumns`
**File:** `pkg/mysql/datasource.mysql.go:301-326`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.scanColumns\(colRows \*sql.Rows, tableName string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger\) \(\[\]ColumnInformation, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.scanColumns\(colRows \*sql.Rows, tableName string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger\) \(\[\]ColumnInformation, error\)
```


#### `.NewDataSourceRepository`
**File:** `pkg/mysql/datasource.mysql.go:52-64`
**Changes:** implementation changed



#### `.\*ExternalDataSource.Query`
**File:** `pkg/mysql/datasource.mysql.go:87-142`
**Changes:** implementation changed



#### `.\*ExternalDataSource.buildTableSchema`
**File:** `pkg/mysql/datasource.mysql.go:269-298`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.buildTableSchema\(ctx context.Context, tableName string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger\) \(TableSchema, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.buildTableSchema\(ctx context.Context, tableName string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger\) \(TableSchema, error\)
```


#### `.scanRows`
**File:** `pkg/mysql/datasource.mysql.go:329-350`
**Changes:** parameters changed, signature\_changed


```diff
- func scanRows\(rows \*sql.Rows, logger log.Logger\) \(\[\]map\[string\]any, error\)
+ func scanRows\(rows \*sql.Rows, logger libLog.Logger\) \(\[\]map\[string\]any, error\)
```


#### `.createRowMap`
**File:** `pkg/mysql/datasource.mysql.go:353-362`
**Changes:** parameters changed, signature\_changed


```diff
- func createRowMap\(columns \[\]string, values \[\]any, logger log.Logger\) map\[string\]any
+ func createRowMap\(columns \[\]string, values \[\]any, logger libLog.Logger\) map\[string\]any
```


#### `.\*ExternalDataSource.ValidateTableAndFields`
**File:** `pkg/mysql/datasource.mysql.go:401-480`
**Changes:** implementation changed



#### `.\*ExternalDataSource.QueryWithAdvancedFilters`
**File:** `pkg/mysql/datasource.mysql.go:523-581`
**Changes:** implementation changed



#### `.testContext`
**File:** `pkg/mysql/datasource.mysql\_test.go:21-29`
**Changes:** implementation changed



#### `.\*Connection.GetDB`
**File:** `pkg/mysql/mysql.go:61-70`
**Changes:** implementation changed



#### `.\*Connection.Connect`
**File:** `pkg/mysql/mysql.go:26-58`
**Changes:** implementation changed



#### `.TestQueryHeaderMetadata`
**File:** `pkg/net/http/http-utils\_test.go:605-651`
**Changes:** implementation changed



#### `.TestValidateParameters`
**File:** `pkg/net/http/http-utils\_test.go:60-197`
**Changes:** implementation changed



#### `.TestValidateParametersNonMetadataKeys`
**File:** `pkg/net/http/http-utils\_test.go:813-832`
**Changes:** implementation changed



#### `.WithRecover`
**File:** `pkg/net/http/with\_recover.go:43-78`
**Changes:** implementation changed



#### `.\*ExternalDataSource.CloseConnection`
**File:** `pkg/oracle/datasource.oracle.go:69-85`
**Changes:** implementation changed



#### `.\*ExternalDataSource.GetDatabaseSchema`
**File:** `pkg/oracle/datasource.oracle.go:149-182`
**Changes:** implementation changed



#### `.\*ExternalDataSource.buildSchema`
**File:** `pkg/oracle/datasource.oracle.go:410-423`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.buildSchema\(ctx context.Context, tables \[\]string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger, schemas \[\]string\) \(\[\]TableSchema, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.buildSchema\(ctx context.Context, tables \[\]string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger, schemas \[\]string\) \(\[\]TableSchema, error\)
```


#### `.\*ExternalDataSource.ValidateTableAndFields`
**File:** `pkg/oracle/datasource.oracle.go:655-746`
**Changes:** implementation changed



#### `.createRowMap`
**File:** `pkg/oracle/datasource.oracle.go:608-617`
**Changes:** parameters changed, signature\_changed


```diff
- func createRowMap\(columns \[\]string, values \[\]any, logger log.Logger\) map\[string\]any
+ func createRowMap\(columns \[\]string, values \[\]any, logger libLog.Logger\) map\[string\]any
```


#### `.\*ExternalDataSource.Query`
**File:** `pkg/oracle/datasource.oracle.go:89-145`
**Changes:** implementation changed



#### `.\*ExternalDataSource.scanColumns`
**File:** `pkg/oracle/datasource.oracle.go:550-581`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.scanColumns\(colRows \*sql.Rows, tableName string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger\) \(\[\]ColumnInformation, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.scanColumns\(colRows \*sql.Rows, tableName string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger\) \(\[\]ColumnInformation, error\)
```


#### `.\*ExternalDataSource.QueryWithAdvancedFilters`
**File:** `pkg/oracle/datasource.oracle.go:800-863`
**Changes:** implementation changed



#### `.NewDataSourceRepository`
**File:** `pkg/oracle/datasource.oracle.go:54-66`
**Changes:** implementation changed



#### `.scanRows`
**File:** `pkg/oracle/datasource.oracle.go:584-605`
**Changes:** parameters changed, signature\_changed


```diff
- func scanRows\(rows \*sql.Rows, logger log.Logger\) \(\[\]map\[string\]any, error\)
+ func scanRows\(rows \*sql.Rows, logger libLog.Logger\) \(\[\]map\[string\]any, error\)
```


#### `.parseJSONField`
**File:** `pkg/oracle/datasource.oracle.go:620-650`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func parseJSONField\(value any, logger log.Logger\) any
+ func parseJSONField\(value any, logger libLog.Logger\) any
```


#### `.\*ExternalDataSource.buildTableSchema`
**File:** `pkg/oracle/datasource.oracle.go:427-528`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.buildTableSchema\(ctx context.Context, tableName string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger, schemas \[\]string\) \(TableSchema, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.buildTableSchema\(ctx context.Context, tableName string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger, schemas \[\]string\) \(TableSchema, error\)
```


#### `.\*Connection.Connect`
**File:** `pkg/oracle/oracle.go:34-66`
**Changes:** implementation changed



#### `.\*Connection.GetDB`
**File:** `pkg/oracle/oracle.go:69-78`
**Changes:** implementation changed



#### `.\*ExternalDataSource.Query`
**File:** `pkg/postgres/datasource.postgres.go:225-280`
**Changes:** implementation changed



#### `.parseJSONBField`
**File:** `pkg/postgres/datasource.postgres.go:769-800`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func parseJSONBField\(value any, logger log.Logger\) any
+ func parseJSONBField\(value any, logger libLog.Logger\) any
```


#### `.\*ExternalDataSource.buildSchema`
**File:** `pkg/postgres/datasource.postgres.go:503-516`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.buildSchema\(ctx context.Context, tables \[\]string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger\) \(\[\]TableSchema, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.buildSchema\(ctx context.Context, tables \[\]string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger\) \(\[\]TableSchema, error\)
```


#### `.\*ExternalDataSource.scanColumns`
**File:** `pkg/postgres/datasource.postgres.go:642-667`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.scanColumns\(colRows \*sql.Rows, tableName string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger\) \(\[\]ColumnInformation, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.scanColumns\(colRows \*sql.Rows, tableName string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger\) \(\[\]ColumnInformation, error\)
```


#### `.createRowMap`
**File:** `pkg/postgres/datasource.postgres.go:737-746`
**Changes:** parameters changed, signature\_changed


```diff
- func createRowMap\(columns \[\]string, values \[\]any, logger log.Logger\) map\[string\]any
+ func createRowMap\(columns \[\]string, values \[\]any, logger libLog.Logger\) map\[string\]any
```


#### `.\*ExternalDataSource.GetDatabaseSchema`
**File:** `pkg/postgres/datasource.postgres.go:308-341`
**Changes:** implementation changed



#### `.\*ExternalDataSource.CloseConnection`
**File:** `pkg/postgres/datasource.postgres.go:176-192`
**Changes:** implementation changed



#### `.\*ExternalDataSource.buildTableSchema`
**File:** `pkg/postgres/datasource.postgres.go:539-574`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.buildTableSchema\(ctx context.Context, tableName string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger\) \(TableSchema, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.buildTableSchema\(ctx context.Context, tableName string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger\) \(TableSchema, error\)
```


#### `.scanRows`
**File:** `pkg/postgres/datasource.postgres.go:692-717`
**Changes:** parameters changed, signature\_changed


```diff
- func scanRows\(rows \*sql.Rows, logger log.Logger\) \(\[\]map\[string\]any, error\)
+ func scanRows\(rows \*sql.Rows, logger libLog.Logger\) \(\[\]map\[string\]any, error\)
```


#### `.\*ExternalDataSource.ValidateTableAndFields`
**File:** `pkg/postgres/datasource.postgres.go:827-911`
**Changes:** implementation changed



#### `.\*ExternalDataSource.QueryWithAdvancedFilters`
**File:** `pkg/postgres/datasource.postgres.go:1023-1081`
**Changes:** implementation changed



#### `.NewDataSourceRepository`
**File:** `pkg/postgres/datasource.postgres.go:143-155`
**Changes:** implementation changed



#### `.testContext`
**File:** `pkg/postgres/datasource.postgres\_test.go:19-27`
**Changes:** implementation changed



#### `.testLogger`
**File:** `pkg/postgres/datasource.postgres\_test.go:30-32`
**Changes:** implementation changed



#### `.\*Connection.Connect`
**File:** `pkg/postgres/postgres.go:25-57`
**Changes:** implementation changed



#### `.\*Connection.GetDB`
**File:** `pkg/postgres/postgres.go:60-69`
**Changes:** implementation changed



#### `.\*RabbitMQAdapter.verifyMessageSignature`
**File:** `pkg/rabbitmq/rabbitmq.go:1133-1222`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*RabbitMQAdapter\) \*RabbitMQAdapter.verifyMessageSignature\(body \[\]byte, headers map\[string\]any, logger libLog.Logger, span \*trace.Span\) error
+ func \(\*RabbitMQAdapter\) \*RabbitMQAdapter.verifyMessageSignature\(body \[\]byte, headers map\[string\]any, logger libLog.Logger, span trace.Span\) error
```


#### `.\*RabbitMQAdapter.ensureChannel`
**File:** `pkg/rabbitmq/rabbitmq.go:576-629`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*RabbitMQAdapter\) \*RabbitMQAdapter.ensureChannel\(span \*trace.Span, logger libLog.Logger\) \(amqpChannel, error\)
+ func \(\*RabbitMQAdapter\) \*RabbitMQAdapter.ensureChannel\(span trace.Span, logger libLog.Logger\) \(amqpChannel, error\)
```


#### `.\*RabbitMQAdapter.processDelivery`
**File:** `pkg/rabbitmq/rabbitmq.go:952-1078`
**Changes:** implementation changed



#### `.\*RabbitMQAdapter.startChannelWatcher`
**File:** `pkg/rabbitmq/rabbitmq.go:535-554`
**Changes:** implementation changed



#### `.\*RabbitMQAdapter.initMetrics`
**File:** `pkg/rabbitmq/rabbitmq.go:384-462`
**Changes:** implementation changed



#### `.\*RabbitMQAdapter.ConsumerLoop`
**File:** `pkg/rabbitmq/rabbitmq.go:780-816`
**Changes:** implementation changed



#### `.\*RabbitMQAdapter.Shutdown`
**File:** `pkg/rabbitmq/rabbitmq.go:1082-1128`
**Changes:** implementation changed



#### `.\*RabbitMQAdapter.ProducerDefault`
**File:** `pkg/rabbitmq/rabbitmq.go:632-776`
**Changes:** implementation changed



#### `.\*RabbitMQAdapter.dispatchDeliveries`
**File:** `pkg/rabbitmq/rabbitmq.go:889-949`
**Changes:** implementation changed



#### `.NewRabbitMQAdapterWithOptions`
**File:** `pkg/rabbitmq/rabbitmq.go:356-381`
**Changes:** implementation changed



#### `.\*RabbitMQAdapter.invalidateChannel`
**File:** `pkg/rabbitmq/rabbitmq.go:521-532`
**Changes:** implementation changed



#### `.\*RabbitMQAdapter.runConsumerCycle`
**File:** `pkg/rabbitmq/rabbitmq.go:819-886`
**Changes:** implementation changed



#### `.TestRabbitMQStressProducerAndConsumer`
**File:** `pkg/rabbitmq/rabbitmq\_integration\_test.go:70-212`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_ConsumerLoop\_NacksOnVersionMismatch`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2200-2256`
**Changes:** implementation changed



#### `.TestVerifyMessageSignature\_TimestampAsInt64`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2427-2452`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_InvalidateChannel\_WithNilChannel`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:1149-1162`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_EnsureChannel\_ReturnsExistingChannel`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:1761-1774`
**Changes:** implementation changed



#### `.TestVerifyMessageSignature\_MissingSignatureHeader`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2322-2346`
**Changes:** implementation changed



#### `.TestVerifyMessageSignature\_NonStringVersion`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2508-2533`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_StartChannelWatcher\_HandlesNilChannel`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:1182-1193`
**Changes:** implementation changed



#### `.TestVerifyMessageSignature\_MissingVersionHeader`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2374-2398`
**Changes:** implementation changed



#### `.TestVerifyMessageSignature\_UnsupportedTimestampType`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2535-2559`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_InvalidateChannel\_ClosesAndNullifiesChannel`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:1097-1147`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_StartChannelWatcher\_NullifiesChannelOnClose`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:1195-1214`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_StartChannelWatcher\_HandlesNilError`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:1216-1235`
**Changes:** implementation changed



#### `.TestVerifyMessageSignature\_MissingTimestampHeader`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2348-2372`
**Changes:** implementation changed



#### `.testContextWithHeader`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:991-1000`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_EnsureChannel\_ReconnectsWhenChannelClosed`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:1776-1795`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_ConsumerLoop\_VerifiesSignatureSuccessfully`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2022-2083`
**Changes:** implementation changed



#### `.TestVerifyMessageSignature\_InvalidTimestampFormat`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2400-2425`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_ConsumerLoop\_NacksOnInvalidSignature`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2141-2198`
**Changes:** implementation changed



#### `.TestRabbitMQAdapter\_EnsureChannel\_RetriesOnFailure`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:1797-1821`
**Changes:** implementation changed



#### `.TestVerifyMessageSignature\_NonStringSignature`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2481-2506`
**Changes:** implementation changed



#### `.TestVerifyMessageSignature\_TimestampAsInt`
**File:** `pkg/rabbitmq/rabbitmq\_test.go:2454-2479`
**Changes:** implementation changed



#### `.NewCacheWithFallback`
**File:** `pkg/redis/factory.go:25-47`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func NewCacheWithFallback\(cfg RedisConfig, logger log.Logger, ttl time.Duration, keyPrefix string\) \(Cache\[T\], error\)
+ func NewCacheWithFallback\(cfg RedisConfig, logger libLog.Logger, ttl time.Duration, keyPrefix string\) \(Cache\[T\], error\)
```


#### `.MustNewCacheWithFallback`
**File:** `pkg/redis/factory.go:51-63`
**Changes:** parameters changed, signature\_changed


```diff
- func MustNewCacheWithFallback\(cfg RedisConfig, logger log.Logger, ttl time.Duration, keyPrefix string\) Cache\[T\]
+ func MustNewCacheWithFallback\(cfg RedisConfig, logger libLog.Logger, ttl time.Duration, keyPrefix string\) Cache\[T\]
```


#### `.TestNewCacheWithFallback\_ZeroTTL\_UsesDefault`
**File:** `pkg/redis/factory\_test.go:41-63`
**Changes:** implementation changed



#### `.TestNewCacheWithFallback\_RedisAvailable\_ReturnsFallbackCache`
**File:** `pkg/redis/factory\_test.go:65-89`
**Changes:** implementation changed



#### `.TestMustNewCacheWithFallback\_RedisUnavailable\_ReturnsMemoryOnlyCache`
**File:** `pkg/redis/factory\_test.go:91-115`
**Changes:** implementation changed



#### `.TestMustNewCacheWithFallback\_ZeroTTL\_UsesDefault`
**File:** `pkg/redis/factory\_test.go:117-136`
**Changes:** implementation changed



#### `.TestNewCacheWithFallback\_NegativeTTL\_UsesDefault`
**File:** `pkg/redis/factory\_test.go:138-158`
**Changes:** implementation changed



#### `.TestNewCacheWithFallback\_RedisUnavailable\_ReturnsMemoryOnlyCache`
**File:** `pkg/redis/factory\_test.go:12-39`
**Changes:** implementation changed



#### `.\*FallbackCache\[T\].Set`
**File:** `pkg/redis/fallback\_cache.go:90-121`
**Changes:** implementation changed



#### `.\*FallbackCache\[T\].Delete`
**File:** `pkg/redis/fallback\_cache.go:124-152`
**Changes:** implementation changed



#### `.\*FallbackCache\[T\].Clear`
**File:** `pkg/redis/fallback\_cache.go:155-180`
**Changes:** implementation changed



#### `.\*FallbackCache\[T\].Close`
**File:** `pkg/redis/fallback\_cache.go:197-221`
**Changes:** implementation changed



#### `.\*FallbackCache\[T\].monitorRedisHealth`
**File:** `pkg/redis/fallback\_cache.go:224-254`
**Changes:** implementation changed



#### `.NewFallbackCache`
**File:** `pkg/redis/fallback\_cache.go:36-55`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func NewFallbackCache\(redisCache \*RedisCache\[T\], logger log.Logger, ttl time.Duration\) \*FallbackCache\[T\]
+ func NewFallbackCache\(redisCache \*RedisCache\[T\], logger libLog.Logger, ttl time.Duration\) \*FallbackCache\[T\]
```


#### `.\*FallbackCache\[T\].Get`
**File:** `pkg/redis/fallback\_cache.go:58-87`
**Changes:** implementation changed



#### `.\*InMemoryCache\[T\].cleanupExpired`
**File:** `pkg/redis/memory\_cache.go:173-195`
**Changes:** implementation changed



#### `.NewInMemoryCache`
**File:** `pkg/redis/memory\_cache.go:36-51`
**Changes:** parameters changed, signature\_changed


```diff
- func NewInMemoryCache\(ttl time.Duration, logger log.Logger\) \*InMemoryCache\[T\]
+ func NewInMemoryCache\(ttl time.Duration, logger libLog.Logger\) \*InMemoryCache\[T\]
```


#### `.\*InMemoryCache\[T\].Get`
**File:** `pkg/redis/memory\_cache.go:54-89`
**Changes:** implementation changed



#### `.\*InMemoryCache\[T\].Set`
**File:** `pkg/redis/memory\_cache.go:92-119`
**Changes:** implementation changed



#### `.NewRedisConnection`
**File:** `pkg/redis/redis.go:73-104`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func NewRedisConnection\(cfg RedisConfig, logger log.Logger\) \(\*RedisConnection, error\)
+ func NewRedisConnection\(cfg RedisConfig, logger libLog.Logger\) \(\*RedisConnection, error\)
```


#### `.\*RedisConnection.Close`
**File:** `pkg/redis/redis.go:107-122`
**Changes:** implementation changed



#### `.\*RedisCache\[T\].Set`
**File:** `pkg/redis/redis\_cache.go:101-136`
**Changes:** implementation changed



#### `.\*RedisCache\[T\].Delete`
**File:** `pkg/redis/redis\_cache.go:139-164`
**Changes:** implementation changed



#### `.\*RedisCache\[T\].Clear`
**File:** `pkg/redis/redis\_cache.go:166-200`
**Changes:** implementation changed



#### `.\*RedisCache\[T\].Get`
**File:** `pkg/redis/redis\_cache.go:54-98`
**Changes:** implementation changed



#### `.\*SimpleRepository.Put`
**File:** `pkg/seaweedfs/external/external-data.go:50-62`
**Changes:** implementation changed



#### `.\*ExternalDataSource.buildSchema`
**File:** `pkg/sqlserver/datasource.sqlserver.go:357-370`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.buildSchema\(ctx context.Context, tables \[\]string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger, schemas \[\]string\) \(\[\]TableSchema, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.buildSchema\(ctx context.Context, tables \[\]string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger, schemas \[\]string\) \(\[\]TableSchema, error\)
```


#### `.\*ExternalDataSource.QueryWithAdvancedFilters`
**File:** `pkg/sqlserver/datasource.sqlserver.go:732-795`
**Changes:** implementation changed



#### `.\*ExternalDataSource.ValidateTableAndFields`
**File:** `pkg/sqlserver/datasource.sqlserver.go:598-682`
**Changes:** implementation changed



#### `.\*ExternalDataSource.Query`
**File:** `pkg/sqlserver/datasource.sqlserver.go:91-147`
**Changes:** implementation changed



#### `.\*ExternalDataSource.scanColumns`
**File:** `pkg/sqlserver/datasource.sqlserver.go:492-523`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.scanColumns\(colRows \*sql.Rows, tableName string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger\) \(\[\]ColumnInformation, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.scanColumns\(colRows \*sql.Rows, tableName string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger\) \(\[\]ColumnInformation, error\)
```


#### `.\*ExternalDataSource.buildTableSchema`
**File:** `pkg/sqlserver/datasource.sqlserver.go:374-469`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*ExternalDataSource\) \*ExternalDataSource.buildTableSchema\(ctx context.Context, tableName string, primaryKeys map\[string\]map\[string\]bool, logger log.Logger, schemas \[\]string\) \(TableSchema, error\)
+ func \(\*ExternalDataSource\) \*ExternalDataSource.buildTableSchema\(ctx context.Context, tableName string, primaryKeys map\[string\]map\[string\]bool, logger libLog.Logger, schemas \[\]string\) \(TableSchema, error\)
```


#### `.scanRows`
**File:** `pkg/sqlserver/datasource.sqlserver.go:526-547`
**Changes:** parameters changed, signature\_changed


```diff
- func scanRows\(rows \*sql.Rows, logger log.Logger\) \(\[\]map\[string\]any, error\)
+ func scanRows\(rows \*sql.Rows, logger libLog.Logger\) \(\[\]map\[string\]any, error\)
```


#### `.NewDataSourceRepository`
**File:** `pkg/sqlserver/datasource.sqlserver.go:56-68`
**Changes:** implementation changed



#### `.createRowMap`
**File:** `pkg/sqlserver/datasource.sqlserver.go:550-559`
**Changes:** parameters changed, signature\_changed


```diff
- func createRowMap\(columns \[\]string, values \[\]any, logger log.Logger\) map\[string\]any
+ func createRowMap\(columns \[\]string, values \[\]any, logger libLog.Logger\) map\[string\]any
```


#### `.\*ExternalDataSource.CloseConnection`
**File:** `pkg/sqlserver/datasource.sqlserver.go:71-87`
**Changes:** implementation changed



#### `.\*ExternalDataSource.GetDatabaseSchema`
**File:** `pkg/sqlserver/datasource.sqlserver.go:151-184`
**Changes:** implementation changed



#### `.parseJSONField`
**File:** `pkg/sqlserver/datasource.sqlserver.go:562-593`
**Changes:** parameters changed, implementation changed, signature\_changed


```diff
- func parseJSONField\(value any, logger log.Logger\) any
+ func parseJSONField\(value any, logger libLog.Logger\) any
```


#### `.testContext`
**File:** `pkg/sqlserver/datasource.sqlserver\_test.go:22-31`
**Changes:** implementation changed



#### `.\*Connection.Connect`
**File:** `pkg/sqlserver/sqlserver.go:26-58`
**Changes:** implementation changed



#### `.\*Connection.GetDB`
**File:** `pkg/sqlserver/sqlserver.go:61-70`
**Changes:** implementation changed



#### `.\*MockLogger.Sync`
**File:** `pkg/testutil/mocks.go:27-27`
**Changes:** parameters changed, signature\_changed


```diff
- func \(\*MockLogger\) \*MockLogger.Sync\(\) error
+ func \(\*MockLogger\) \*MockLogger.Sync\(\_ context.Context\) error
```


#### `.buildProxiedAppEnv`
**File:** `tests/chaos/helpers\_test.go:181-220`
**Changes:** implementation changed



#### `.BuildAppEnv`
**File:** `tests/shared/apps.go:63-100`
**Changes:** implementation changed



#### `.\*AppEnv.ManagerEnv`
**File:** `tests/shared/apps.go:105-134`
**Changes:** implementation changed



#### `.RequireNoError`
**File:** `tests/shared/assertions.go:147-150`
**Changes:** parameters changed, signature\_changed


```diff
- func RequireNoError\(t \*testing.T, err error, msgAndArgs ...interface{}\)
+ func RequireNoError\(t \*testing.T, err error, msgAndArgs ...any\)
```


#### `.AssertJobCompleted`
**File:** `tests/shared/assertions.go:22-71`
**Changes:** implementation changed



#### `.checkStatus`
**File:** `tests/shared/client.go:110-118`
**Changes:** implementation changed



#### `.ListConnectionsParams.toQueryString`
**File:** `tests/shared/client.go:241-268`
**Changes:** implementation changed



#### `.isFixedPortEnabled`
**File:** `tests/shared/infra.go:33-39`
**Changes:** implementation changed



#### `.definitionsPath`
**File:** `tests/shared/infra.go:87-94`
**Changes:** implementation changed




### Functions Added (125)

- `.TestFetcherHandler\_CreateJob\_MetadataSourceValidation` at `components/manager/internal/adapters/http/in/fetcher\_test.go:209`

- `.TestNewRoutes\_RegistersCriticalStaticRoutesBeforeIDRoutes` at `components/manager/internal/adapters/http/in/routes\_test.go:30`

- `.indexOfRoute` at `components/manager/internal/adapters/http/in/routes\_test.go:80`

- `.initLoggerAndTelemetry` at `components/manager/internal/bootstrap/config.go:135`

- `.initPlatformDependencies` at `components/manager/internal/bootstrap/config.go:215`

- `.assembleService` at `components/manager/internal/bootstrap/config.go:261`

- `.buildMongoSource` at `components/manager/internal/bootstrap/config.go:335`

- `.buildRabbitMQSource` at `components/manager/internal/bootstrap/config.go:342`

- `.resolveZapEnvironment` at `components/manager/internal/bootstrap/config.go:379`

- `.loadConfig` at `components/manager/internal/bootstrap/config.go:126`

- `.initMongoRepositories` at `components/manager/internal/bootstrap/config.go:165`

- `.initCrypto` at `components/manager/internal/bootstrap/config.go:195`

- `.must` at `components/manager/internal/bootstrap/config.go:396`

- `.TestGetSchemaCacheTTL` at `components/manager/internal/bootstrap/config\_test.go:32`

- `.TestGetRedisDB` at `components/manager/internal/bootstrap/config\_test.go:38`

- `.TestResolveZapEnvironment` at `components/manager/internal/bootstrap/config\_test.go:44`

- `.TestCreateFetcherJob\_Execute\_PublishFailureMarksJobFailed` at `components/manager/internal/services/command/create\_fetcher\_job\_test.go:697`

- `.TestTestConnection\_Execute\_Success` at `components/manager/internal/services/query/test\_connection\_test.go:203`

- `.TestValidateSchema\_NilSchemaFromDatasource` at `components/manager/internal/services/query/validate\_schema\_test.go:858`

- `.resolveZapEnvironment` at `components/worker/internal/bootstrap/config.go:232`

- `.must` at `components/worker/internal/bootstrap/config.go:249`

- `.TestInitWorker\_PanicsWhenTelemetryGlobalsFail` at `components/worker/internal/bootstrap/config\_test.go:139`

- `.testBootstrapLogger` at `components/worker/internal/bootstrap/config\_test.go:14`

- `.TestResolveZapEnvironment` at `components/worker/internal/bootstrap/config\_test.go:18`

- `.TestMust` at `components/worker/internal/bootstrap/config\_test.go:45`

- `.TestInitWorker\_PanicsWhenConfigLoadFails` at `components/worker/internal/bootstrap/config\_test.go:82`

- `.TestInitWorker\_PanicsWhenLoggerInitFails` at `components/worker/internal/bootstrap/config\_test.go:106`

- `.TestParseMessage\_JSONNull` at `components/worker/internal/services/extract\_data\_test.go:86`

- `.newDataSourceFromConnection` at `pkg/datasource/datasource-factory.go:59`

- `.resolveBuildSecretSource` at `pkg/itestkit/addons/e2ekit/build\_secrets.go:116`

- `.createBuildSecretTempFile` at `pkg/itestkit/addons/e2ekit/build\_secrets.go:133`

- `.ParseHostPort` at `pkg/itestkit/hostport.go:95`

- `.ResolveHostHostPort` at `pkg/itestkit/hostport.go:111`

- `.ResolveContainerHostPort` at `pkg/itestkit/hostport.go:124`

- `.\*MongoDBInfra.ContainerHostPort` at `pkg/itestkit/infra/mongodb/mongodb.go:186`

- `.\*PostgresInfra.ContainerHostPort` at `pkg/itestkit/infra/postgres/postgres.go:188`

- `.\*RabbitInfra.ContainerHostPort` at `pkg/itestkit/infra/rabbitmq/rabbitmq.go:178`

- `.\*RedisInfra.ContainerHostPort` at `pkg/itestkit/infra/redis/redis.go:182`

- `.\*SeaweedFSInfra.ContainerHostPort` at `pkg/itestkit/infra/seaweedfs/seaweedfs.go:263`

- `.\*MockmongoDatabaseProvider.Client` at `pkg/mongodb/connection/connection.mongodb.mock.go:239`

- `.\*MockmongoDatabaseProviderMockRecorder.Client` at `pkg/mongodb/connection/connection.mongodb.mock.go:248`

- `.TestNewConnectionMongoDBRepository\_NilClient` at `pkg/mongodb/connection/connection.mongodb\_test.go:143`

- `.NewMockDatasource` at `pkg/mongodb/datasource.mongodb.mock.go:33`

- `.\*MockDatasourceMockRecorder.CloseConnection` at `pkg/mongodb/datasource.mongodb.mock.go:53`

- `.\*MockDatasource.GetDatabaseSchema` at `pkg/mongodb/datasource.mongodb.mock.go:59`

- `.\*MockDatasource.Query` at `pkg/mongodb/datasource.mongodb.mock.go:74`

- `.\*MockDatasource.EXPECT` at `pkg/mongodb/datasource.mongodb.mock.go:40`

- `.\*MockDatasource.CloseConnection` at `pkg/mongodb/datasource.mongodb.mock.go:45`

- `.\*MockDatasourceMockRecorder.GetDatabaseSchema` at `pkg/mongodb/datasource.mongodb.mock.go:68`

- `.\*MockDatasourceMockRecorder.Query` at `pkg/mongodb/datasource.mongodb.mock.go:83`

- `.\*MockDatasource.QueryWithAdvancedFilters` at `pkg/mongodb/datasource.mongodb.mock.go:89`

- `.\*MockDatasourceMockRecorder.QueryWithAdvancedFilters` at `pkg/mongodb/datasource.mongodb.mock.go:98`

- `.TestNewJobMongoDBRepository\_NilClient` at `pkg/mongodb/job/job.mongodb\_test.go:131`

- `.\*MockMongoClientProvider.Client` at `pkg/mongodb/mongo\_client\_provider.mock.go:45`

- `.\*MockMongoClientProviderMockRecorder.Client` at `pkg/mongodb/mongo\_client\_provider.mock.go:54`

- `.newProductMongoDBRepository` at `pkg/mongodb/product/product.mongodb.go:73`

- `.\*MockLogger.Log` at `pkg/testutil/mocks.go:23`

- `.\*MockLogger.With` at `pkg/testutil/mocks.go:24`

- `.\*MockLogger.WithGroup` at `pkg/testutil/mocks.go:25`

- `.\*MockLogger.Enabled` at `pkg/testutil/mocks.go:26`

- `.createTestProduct` at `components/manager/internal/adapters/http/in/real\_handler\_test.go:28`

- `.TestConnectionHandler\_CreateConnection\_RealHandlerSuccess` at `components/manager/internal/adapters/http/in/real\_handler\_test.go:41`

- `.TestConnectionHandler\_UpdateConnection\_RealHandlerSuccess` at `components/manager/internal/adapters/http/in/real\_handler\_test.go:136`

- `.TestConnectionHandler\_ValidateSchema\_RealHandlerFailureResponse` at `components/manager/internal/adapters/http/in/real\_handler\_test.go:179`

- `.TestProductHandler\_CreateProduct\_RealHandlerSuccess` at `components/manager/internal/adapters/http/in/real\_handler\_test.go:256`

- `.TestMigrationHandler\_AssignConnectionToProduct\_RealHandlerSuccess` at `components/manager/internal/adapters/http/in/real\_handler\_test.go:347`

- `.TestConnectionHandler\_ListConnections\_RealHandlerSuccess` at `components/manager/internal/adapters/http/in/real\_handler\_test.go:97`

- `.TestConnectionHandler\_GetConnectionSchema\_RealHandlerSuccess` at `components/manager/internal/adapters/http/in/real\_handler\_test.go:218`

- `.TestProductHandler\_ListProducts\_RealHandlerSuccess` at `components/manager/internal/adapters/http/in/real\_handler\_test.go:289`

- `.TestMigrationHandler\_ListUnassignedConnections\_RealHandlerSuccess` at `components/manager/internal/adapters/http/in/real\_handler\_test.go:318`

- `.\*fakeLicenseTerminator.Terminate` at `components/worker/internal/bootstrap/consumer\_service\_test.go:28`

- `.TestNewMultiQueueConsumerRegistersQueue` at `components/worker/internal/bootstrap/consumer\_service\_test.go:33`

- `.TestHandlerGenerateReport\_DelegatesToUseCase` at `components/worker/internal/bootstrap/consumer\_service\_test.go:61`

- `.TestMultiQueueConsumerRun` at `components/worker/internal/bootstrap/consumer\_service\_test.go:111`

- `.TestServiceRun` at `components/worker/internal/bootstrap/consumer\_service\_test.go:169`

- `.contextWithBootstrapTracking` at `components/worker/internal/bootstrap/consumer\_service\_test.go:229`

- `.TestExtractExternalData\_JobNotFoundMarksFailed` at `components/worker/internal/services/extract\_data\_additional\_test.go:17`

- `.TestExtractExternalData\_NoConnectionsMarksFailed` at `components/worker/internal/services/extract\_data\_additional\_test.go:54`

- `.TestExtractExternalData\_SuccessWithNonFatalWarnings` at `components/worker/internal/services/extract\_data\_additional\_test.go:93`

- `.TestQueryDatabase\_DataSourceFactoryAndLifecycleErrors` at `components/worker/internal/services/extract\_data\_additional\_test.go:151`

- `.TestSaveExternalDataToSeaweedFS\_DocumentSignerError` at `components/worker/internal/services/extract\_data\_additional\_test.go:224`

- `.mustMarshalMessage` at `components/worker/internal/services/extract\_data\_additional\_test.go:254`

- `.\*stubDataSource.Close` at `pkg/datasource/datasource-factory\_test.go:32`

- `.\*stubDataSource.GetType` at `pkg/datasource/datasource-factory\_test.go:36`

- `.\*stubDataSource.GetSchemaInfo` at `pkg/datasource/datasource-factory\_test.go:44`

- `.testConnection` at `pkg/datasource/datasource-factory\_test.go:146`

- `.newMockCryptor` at `pkg/datasource/datasource-factory\_test.go:162`

- `.\*stubDataSource.GetConfig` at `pkg/datasource/datasource-factory\_test.go:24`

- `.\*stubDataSource.Connect` at `pkg/datasource/datasource-factory\_test.go:28`

- `.\*stubDataSource.Query` at `pkg/datasource/datasource-factory\_test.go:40`

- `.TestNewDataSourceFromConnection` at `pkg/datasource/datasource-factory\_test.go:48`

- `.TestBuildImageWithSecretsValidation` at `pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:9`

- `.TestBuildConfigHelpers` at `pkg/itestkit/addons/e2ekit/build\_secrets\_test.go:73`

- `.stubRewriter.Rewrite` at `pkg/itestkit/addons/e2ekit/helpers\_test.go:16`

- `.TestBuilderHelpersAndProjectRoot` at `pkg/itestkit/addons/e2ekit/helpers\_test.go:25`

- `.deterministicMetrics` at `pkg/itestkit/addons/metricskit/metrics\_test.go:10`

- `.TestChaosMetricsAndReporting` at `pkg/itestkit/addons/metricskit/metrics\_test.go:27`

- `.sampleParsedMessage` at `pkg/itestkit/addons/queuekit/assertions\_test.go:16`

- `.TestQueueAssertionsHelpers` at `pkg/itestkit/addons/queuekit/assertions\_test.go:31`

- `.applyCustomizer` at `pkg/itestkit/customizer\_options\_test.go:10`

- `.TestCustomizerOptions\_MutateContainerRequest` at `pkg/itestkit/customizer\_options\_test.go:22`

- `.TestResolveHostHostPort` at `pkg/itestkit/hostport\_test.go:5`

- `.TestResolveContainerHostPort` at `pkg/itestkit/hostport\_test.go:31`

- `.TestSeaweedFSHelpersWithoutDocker` at `pkg/itestkit/infra/seaweedfs/seaweedfs\_test.go:13`

- `.\*fakeInfra.Terminate` at `pkg/itestkit/suite\_test.go:19`

- `.\*fakeChaos.CreateProxy` at `pkg/itestkit/suite\_test.go:42`

- `.\*fakeChaos.RemoveToxic` at `pkg/itestkit/suite\_test.go:51`

- `.fakeNamedInfra.Start` at `pkg/itestkit/suite\_test.go:32`

- `.fakeNamedInfra.Terminate` at `pkg/itestkit/suite\_test.go:33`

- `.fakeNamedInfra.InfraKind` at `pkg/itestkit/suite\_test.go:34`

- `.\*fakeChaos.AddLatency` at `pkg/itestkit/suite\_test.go:45`

- `.\*fakeChaos.RemoveAllToxics` at `pkg/itestkit/suite\_test.go:52`

- `.\*fakeInfra.Start` at `pkg/itestkit/suite\_test.go:17`

- `.fakeNamedInfra.InfraName` at `pkg/itestkit/suite\_test.go:35`

- `.\*fakeChaos.AddBandwidth` at `pkg/itestkit/suite\_test.go:49`

- `.\*fakeChaos.CutConnection` at `pkg/itestkit/suite\_test.go:50`

- `.\*fakeChaos.Close` at `pkg/itestkit/suite\_test.go:53`

- `.\*fakeChaos.AddTimeout` at `pkg/itestkit/suite\_test.go:48`

- `.TestBuilderConfigurationAndSuiteLifecycle` at `pkg/itestkit/suite\_test.go:58`

- `.stubMongoDatabaseProvider.Client` at `pkg/mongodb/product/product.mongodb\_test.go:24`

- `.TestNewProductMongoDBRepository\_NilClient` at `pkg/mongodb/product/product.mongodb\_test.go:32`

- `.TestNewProductMongoDBRepository\_ConfigAndInitialization` at `pkg/mongodb/product/product.mongodb\_test.go:42`

- `.TestProductMongoDBRepository\_Create\_ErrorPaths` at `pkg/mongodb/product/product.mongodb\_test.go:118`

- `.TestProductMongoDBRepository\_BuildQueryFilter` at `pkg/mongodb/product/product.mongodb\_test.go:186`

- `.testProduct` at `pkg/mongodb/product/product.mongodb\_test.go:239`


### Types Modified (24)

#### `Server`
**File:** `components/manager/internal/bootstrap/server.go`

| Field | Before | After |
|-------|--------|-------|
| license | \*libCommonsLicense.ManagerShutdown | licenseTerminator |

#### `CreateFetcherJob`
**File:** `components/manager/internal/services/command/create\_fetcher\_job.go`

| Field | Before | After |
|-------|--------|-------|
| rabbitMQ | \*rabbitmq.RabbitMQAdapter | rabbitmq.Adapter |

#### `TestConnection`
**File:** `components/manager/internal/services/query/test\_connection.go`

| Field | Before | After |
|-------|--------|-------|
| dataSource | - | dataSourceFactory (added) |

#### `ValidateSchema`
**File:** `components/manager/internal/services/query/validate\_schema.go`

| Field | Before | After |
|-------|--------|-------|
| dataSource | - | dataSourceFactory (added) |

#### `ConsumerRoutes`
**File:** `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go`

| Field | Before | After |
|-------|--------|-------|
| libLog.Logger | - | libLog.Logger (added) |
| log.Logger | log.Logger | (deleted) |

#### `PublisherRoutes`
**File:** `components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go`

| Field | Before | After |
|-------|--------|-------|
| libLog.Logger | - | libLog.Logger (added) |
| log.Logger | log.Logger | (deleted) |

#### `Service`
**File:** `components/worker/internal/bootstrap/service.go`

| Field | Before | After |
|-------|--------|-------|
| licenseShutdown | \*libCommonsLicense.ManagerShutdown | licenseTerminator |
| libLog.Logger | - | libLog.Logger (added) |
| log.Logger | log.Logger | (deleted) |

#### `ProxyRef`
**File:** `pkg/itestkit/chaos.go`

| Field | Before | After |
|-------|--------|-------|
| InNetworkListenAddr | - | string (added) |

#### `MongoDBEndpoint`
**File:** `pkg/itestkit/infra/mongodb/mongodb.go`

| Field | Before | After |
|-------|--------|-------|
| ProxyListenInNetwork | - | string (added) |

#### `PostgresEndpoint`
**File:** `pkg/itestkit/infra/postgres/postgres.go`

| Field | Before | After |
|-------|--------|-------|
| ProxyListenInNetwork | - | string (added) |

#### `RabbitEndpoint`
**File:** `pkg/itestkit/infra/rabbitmq/rabbitmq.go`

| Field | Before | After |
|-------|--------|-------|
| ProxyListenInNetwork | - | string (added) |

#### `RedisEndpoint`
**File:** `pkg/itestkit/infra/redis/redis.go`

| Field | Before | After |
|-------|--------|-------|
| ProxyListenInNetwork | - | string (added) |

#### `SeaweedFSEndpoint`
**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go`

| Field | Before | After |
|-------|--------|-------|
| ProxyListenInNetwork | - | string (added) |

#### `SeaweedFSInfra`
**File:** `pkg/itestkit/infra/seaweedfs/seaweedfs.go`

| Field | Before | After |
|-------|--------|-------|
| network | testcontainers.Network | \*testcontainers.DockerNetwork |

#### `Suite`
**File:** `pkg/itestkit/suite.go`

| Field | Before | After |
|-------|--------|-------|
| network | testcontainers.Network | \*testcontainers.DockerNetwork |

#### `ExternalDataSource`
**File:** `pkg/mongodb/datasource.mongodb.go`

| Field | Before | After |
|-------|--------|-------|
| connection | \*libMongo.MongoConnection | \*libMongo.Client |
| logger | - | libLog.Logger (added) |

#### `MockMongoClientProvider`
**File:** `pkg/mongodb/mongo\_client\_provider.mock.go`

| Field | Before | After |
|-------|--------|-------|
| isgomock | - | struct{} (added) |

#### `Connection`
**File:** `pkg/mysql/mysql.go`

| Field | Before | After |
|-------|--------|-------|
| Logger | log.Logger | libLog.Logger |

#### `Connection`
**File:** `pkg/oracle/oracle.go`

| Field | Before | After |
|-------|--------|-------|
| Logger | log.Logger | libLog.Logger |

#### `Connection`
**File:** `pkg/postgres/postgres.go`

| Field | Before | After |
|-------|--------|-------|
| Logger | log.Logger | libLog.Logger |

#### `FallbackCache`
**File:** `pkg/redis/fallback\_cache.go`

| Field | Before | After |
|-------|--------|-------|
| logger | log.Logger | libLog.Logger |

#### `InMemoryCache`
**File:** `pkg/redis/memory\_cache.go`

| Field | Before | After |
|-------|--------|-------|
| logger | log.Logger | libLog.Logger |

#### `RedisConnection`
**File:** `pkg/redis/redis.go`

| Field | Before | After |
|-------|--------|-------|
| Logger | log.Logger | libLog.Logger |

#### `Connection`
**File:** `pkg/sqlserver/sqlserver.go`

| Field | Before | After |
|-------|--------|-------|
| Logger | log.Logger | libLog.Logger |



## Focus Areas

Based on analysis, pay special attention to:

1. **Signature change in NewCreateFetcherJob** - Function signature modified - verify caller compatibility
2. **Signature change in NewCreateFetcherJobWithTester** - Function signature modified - verify caller compatibility
3. **Signature change in \*CreateFetcherJob.validateProductOwnership** - Function signature modified - verify caller compatibility
4. **Signature change in NewConsumerRoutes** - Function signature modified - verify caller compatibility
5. **Signature change in NewConsumerRoutesWithAdapter** - Function signature modified - verify caller compatibility
6. **Signature change in NewPublisherRoutes** - Function signature modified - verify caller compatibility
7. **Signature change in NewPublisherRoutesWithAdapter** - Function signature modified - verify caller compatibility
8. **Signature change in \*UseCase.saveExternalDataToSeaweedFS** - Function signature modified - verify caller compatibility
9. **Signature change in \*UseCase.shouldSkipProcessing** - Function signature modified - verify caller compatibility
10. **Signature change in \*UseCase.parseMessage** - Function signature modified - verify caller compatibility
11. **Signature change in \*UseCase.extractJobIDFromMultipleSources** - Function signature modified - verify caller compatibility
12. **Signature change in \*UseCase.extractJobIDFromPartialJSON** - Function signature modified - verify caller compatibility
13. **Signature change in \*UseCase.queryDatabase** - Function signature modified - verify caller compatibility
14. **Signature change in \*UseCase.handleErrorWithUpdate** - Function signature modified - verify caller compatibility
15. **Signature change in \*UseCase.encryptDataForSeaweedFS** - Function signature modified - verify caller compatibility
16. **Signature change in \*UseCase.checkReportStatus** - Function signature modified - verify caller compatibility
17. **Signature change in \*UseCase.transformPluginCRMAdvancedFilters** - Function signature modified - verify caller compatibility
18. **Signature change in \*UseCase.decryptPluginCRMData** - Function signature modified - verify caller compatibility
19. **Signature change in \*UseCase.QueryPluginCRM** - Function signature modified - verify caller compatibility
20. **Signature change in \*UseCase.queryPluginCRMCollectionWithFilters** - Function signature modified - verify caller compatibility
21. **Signature change in \*UseCase.processPluginCRMCollection** - Function signature modified - verify caller compatibility
22. **Signature change in \*UseCase.publishJobNotification** - Function signature modified - verify caller compatibility
23. **Signature change in newDataSourceConfigMongoDB** - Function signature modified - verify caller compatibility
24. **Signature change in newDataSourceConfigPostgres** - Function signature modified - verify caller compatibility
25. **Signature change in newDataSourceConfigOracle** - Function signature modified - verify caller compatibility
26. **Signature change in newDataSourceConfigMySQL** - Function signature modified - verify caller compatibility
27. **Signature change in newDataSourceConfigSQLServer** - Function signature modified - verify caller compatibility
28. **Signature change in NewDataSourceFromConnectionWithLogger** - Function signature modified - verify caller compatibility
29. **Signature change in NewDataSourceFromConnection** - Function signature modified - verify caller compatibility
30. **Signature change in \*DataSourceConfigMongoDB.Connect** - Function signature modified - verify caller compatibility
31. **Signature change in \*DataSourceConfigMongoDB.Query** - Function signature modified - verify caller compatibility
32. **Signature change in \*DataSourceConfigMySQL.Connect** - Function signature modified - verify caller compatibility
33. **Signature change in \*DataSourceConfigMySQL.Query** - Function signature modified - verify caller compatibility
34. **Signature change in \*DataSourceConfigOracle.Connect** - Function signature modified - verify caller compatibility
35. **Signature change in \*DataSourceConfigOracle.Query** - Function signature modified - verify caller compatibility
36. **Signature change in \*DataSourceConfigPostgres.Connect** - Function signature modified - verify caller compatibility
37. **Signature change in \*DataSourceConfigPostgres.Query** - Function signature modified - verify caller compatibility
38. **Signature change in \*DataSourceConfigSQLServer.Query** - Function signature modified - verify caller compatibility
39. **Signature change in \*DataSourceConfigSQLServer.Connect** - Function signature modified - verify caller compatibility
40. **Signature change in NewConnectionMongoDBRepository** - Function signature modified - verify caller compatibility
41. **Signature change in NewDataSourceRepository** - Function signature modified - verify caller compatibility
42. **Signature change in \*ExternalDataSource.processQueryResults** - Function signature modified - verify caller compatibility
43. **Signature change in NewJobMongoDBRepository** - Function signature modified - verify caller compatibility
44. **Signature change in \*JobMongoDBRepository.scanJobs** - Function signature modified - verify caller compatibility
45. **Signature change in NewProductMongoDBRepository** - Function signature modified - verify caller compatibility
46. **Signature change in parseJSONField** - Function signature modified - verify caller compatibility
47. **Signature change in \*ExternalDataSource.buildSchema** - Function signature modified - verify caller compatibility
48. **Signature change in \*ExternalDataSource.scanColumns** - Function signature modified - verify caller compatibility
49. **Signature change in \*ExternalDataSource.buildTableSchema** - Function signature modified - verify caller compatibility
50. **Signature change in scanRows** - Function signature modified - verify caller compatibility
51. **Signature change in createRowMap** - Function signature modified - verify caller compatibility
52. **Signature change in \*ExternalDataSource.buildSchema** - Function signature modified - verify caller compatibility
53. **Signature change in createRowMap** - Function signature modified - verify caller compatibility
54. **Signature change in \*ExternalDataSource.scanColumns** - Function signature modified - verify caller compatibility
55. **Signature change in scanRows** - Function signature modified - verify caller compatibility
56. **Signature change in parseJSONField** - Function signature modified - verify caller compatibility
57. **Signature change in \*ExternalDataSource.buildTableSchema** - Function signature modified - verify caller compatibility
58. **Signature change in parseJSONBField** - Function signature modified - verify caller compatibility
59. **Signature change in \*ExternalDataSource.buildSchema** - Function signature modified - verify caller compatibility
60. **Signature change in \*ExternalDataSource.scanColumns** - Function signature modified - verify caller compatibility
61. **Signature change in createRowMap** - Function signature modified - verify caller compatibility
62. **Signature change in \*ExternalDataSource.buildTableSchema** - Function signature modified - verify caller compatibility
63. **Signature change in scanRows** - Function signature modified - verify caller compatibility
64. **Signature change in \*RabbitMQAdapter.verifyMessageSignature** - Function signature modified - verify caller compatibility
65. **Signature change in \*RabbitMQAdapter.ensureChannel** - Function signature modified - verify caller compatibility
66. **Signature change in NewCacheWithFallback** - Function signature modified - verify caller compatibility
67. **Signature change in MustNewCacheWithFallback** - Function signature modified - verify caller compatibility
68. **Signature change in NewFallbackCache** - Function signature modified - verify caller compatibility
69. **Signature change in NewInMemoryCache** - Function signature modified - verify caller compatibility
70. **Signature change in NewRedisConnection** - Function signature modified - verify caller compatibility
71. **Signature change in \*ExternalDataSource.buildSchema** - Function signature modified - verify caller compatibility
72. **Signature change in \*ExternalDataSource.scanColumns** - Function signature modified - verify caller compatibility
73. **Signature change in \*ExternalDataSource.buildTableSchema** - Function signature modified - verify caller compatibility
74. **Signature change in scanRows** - Function signature modified - verify caller compatibility
75. **Signature change in createRowMap** - Function signature modified - verify caller compatibility
76. **Signature change in parseJSONField** - Function signature modified - verify caller compatibility
77. **Signature change in \*MockLogger.Sync** - Function signature modified - verify caller compatibility
78. **Signature change in RequireNoError** - Function signature modified - verify caller compatibility

