# Pre-Analysis Context: Nil Safety

## Nil Source Analysis


| Variable | File | Line | Checked? | Risk |
|----------|------|------|----------|------|
| `message` | components/manager/internal/services/command/create\_fetcher\_job.go | 389 | No | high |
| `header` | components/manager/internal/services/command/create\_fetcher\_job.go | 407 | No | medium |
| `found` | components/manager/internal/services/command/create\_fetcher\_job.go | 231 | No | medium |
| `fields` | components/manager/internal/services/query/get\_connection\_schema.go | 125 | No | medium |
| `systemSchemas` | components/manager/internal/services/query/get\_connection\_schema.go | 263 | No | high |
| `systemTables` | components/manager/internal/services/query/get\_connection\_schema.go | 226 | No | high |
| `systemDatabases` | components/manager/internal/services/query/get\_connection\_schema.go | 290 | No | high |
| `found` | components/manager/internal/services/query/validate\_schema.go | 138 | No | medium |
| `tables` | components/manager/internal/services/query/validate\_schema.go | 149 | No | medium |
| `Tables` | components/manager/internal/services/query/validate\_schema.go | 276 | No | medium |
| `exists` | components/manager/internal/services/query/validate\_schema.go | 441 | No | medium |
| `configNames` | components/manager/internal/services/query/validate\_schema.go | 100 | No | medium |
| `schemas` | components/manager/internal/services/query/validate\_schema.go | 164 | No | high |
| `pluginCRMTableMapping` | components/manager/internal/services/query/validate\_schema.go | 35 | No | high |
| `encryptSecretKey` | components/worker/internal/services/extract-data.go | 528 | No | medium |
| `hashSecretKey` | components/worker/internal/services/extract-data.go | 533 | No | medium |
| `errorMetadata` | components/worker/internal/services/extract-data.go | 301 | No | medium |
| `exists` | components/worker/internal/services/extract-data.go | 190 | No | medium |
| `orgMatches` | components/worker/internal/services/extract-data.go | 256 | No | high |
| `existOrg` | components/worker/internal/services/extract-data.go | 198 | No | medium |
| `matches` | components/worker/internal/services/extract-data.go | 249 | No | high |
| `databaseExists` | components/worker/internal/services/extract-data.go | 407 | No | medium |
| `encryptedFields` | components/worker/internal/services/extract\_crm\_data.go | 314 | No | high |
| `exists` | components/worker/internal/services/extract\_crm\_data.go | 452 | No | medium |
| `encryptSecretKey` | components/worker/internal/services/extract\_crm\_data.go | 279 | No | medium |
| `hashSecretKey` | components/worker/internal/services/extract\_crm\_data.go | 277 | No | medium |
| `fieldMappings` | components/worker/internal/services/extract\_crm\_data.go | 180 | No | high |
| `defaultDataSourceConfigBuilders` | pkg/datasource/datasource-factory.go | 33 | No | medium |
| `validMongoDBModes` | pkg/datasource/sslmode/mongodb.go | 23 | No | high |
| `valid` | pkg/datasource/sslmode/mongodb.go | 45 | No | medium |
| `validMySQLModes` | pkg/datasource/sslmode/mysql.go | 22 | No | high |
| `valid` | pkg/datasource/sslmode/mysql.go | 43 | No | medium |
| `validOracleModes` | pkg/datasource/sslmode/oracle.go | 25 | No | high |
| `valid` | pkg/datasource/sslmode/oracle.go | 48 | No | medium |
| `validPostgreSQLModes` | pkg/datasource/sslmode/postgresql.go | 21 | No | high |
| `valid` | pkg/datasource/sslmode/postgresql.go | 44 | No | medium |
| `validSQLServerModes` | pkg/datasource/sslmode/sqlserver.go | 22 | No | high |
| `valid` | pkg/datasource/sslmode/sqlserver.go | 43 | No | medium |
| `value` | pkg/itestkit/addons/e2ekit/build\_secrets.go | 134 | No | medium |
| `result` | pkg/itestkit/addons/metricskit/metrics.go | 306 | No | medium |
| `errorCounts` | pkg/itestkit/addons/metricskit/reporter.go | 62 | No | medium |
| `e` | pkg/itestkit/addons/queuekit/assertions.go | 251 | No | high |
| `a` | pkg/itestkit/addons/queuekit/assertions.go | 256 | No | high |
| `m` | pkg/itestkit/addons/queuekit/assertions.go | 278 | No | high |
| `parsed` | pkg/itestkit/addons/queuekit/consumer.go | 249 | No | medium |
| `consumer` | pkg/itestkit/addons/queuekit/consumer.go | 392 | No | high |
| `result` | pkg/itestkit/addons/queuekit/consumer.go | 210 | No | high |
| `v` | pkg/itestkit/addons/queuekit/matcher.go | 190 | No | medium |
| `current` | pkg/itestkit/addons/queuekit/matcher.go | 213 | No | high |
| `exists` | pkg/itestkit/addons/queuekit/matcher.go | 229 | No | medium |
| `e` | pkg/itestkit/addons/queuekit/matcher.go | 258 | No | medium |
| `data` | pkg/itestkit/addons/queuekit/matcher.go | 181 | No | medium |
| `NetworkAliases` | pkg/itestkit/chaos\_toxiproxy.go | 58 | No | medium |
| `Containers` | pkg/itestkit/container\_generic.go | 79 | No | high |
| `NetworkAliases` | pkg/itestkit/customizer\_options.go | 112 | No | high |
| `override` | pkg/itestkit/hostport.go | 31 | No | medium |
| `seen` | pkg/itestkit/infra.go | 20 | No | high |
| `exists` | pkg/itestkit/infra.go | 36 | No | medium |
| `NetworkAliases` | pkg/itestkit/infra/oracle/oracle.go | 92 | No | medium |
| `schemas` | pkg/model/datasource/oracle/datasource-config.go | 65 | No | high |
| `schemas` | pkg/model/datasource/postgres/datasource-config.go | 65 | No | high |
| `exists` | pkg/model/datasource/postgres/datasource-config.go | 169 | No | medium |
| `schemas` | pkg/model/datasource/sqlserver/datasource-config.go | 68 | No | high |
| `exists` | pkg/model/datasource/sqlserver/datasource-config.go | 126 | No | medium |
| `config` | pkg/mongodb/connection/connection.mongodb.go | 83 | No | high |
| `opts` | pkg/mongodb/connection/connection.mongodb.go | 745 | No | medium |
| `errFind` | pkg/mongodb/connection/connection.mongodb.go | 446 | No | medium |
| `v` | pkg/mongodb/datasource.mongodb.go | 206 | No | high |
| `dataType` | pkg/mongodb/datasource.mongodb.go | 323 | No | medium |
| `exists` | pkg/mongodb/datasource.mongodb.go | 516 | No | medium |
| `typeHierarchy` | pkg/mongodb/datasource.mongodb.go | 559 | No | high |
| `regexPattern` | pkg/mongodb/datasource.mongodb.go | 839 | No | medium |
| `findOptions` | pkg/mongodb/datasource.mongodb.go | 655 | No | high |
| `newLevel` | pkg/mongodb/datasource.mongodb.go | 575 | No | medium |
| `currentLevel` | pkg/mongodb/datasource.mongodb.go | 576 | No | medium |
| `singleValueOps` | pkg/mongodb/datasource.mongodb.go | 869 | No | medium |
| `config` | pkg/mongodb/job/job.mongodb.go | 80 | No | high |
| `opts` | pkg/mongodb/job/job.mongodb.go | 430 | No | high |
| `exists` | pkg/mongodb/mongo.go | 33 | No | medium |
| `opts` | pkg/mongodb/product/product.mongodb.go | 216 | No | high |
| `config` | pkg/mongodb/product/product.mongodb.go | 82 | No | high |
| `exists` | pkg/mysql/datasource.mysql.go | 314 | No | medium |
| `jsonArray` | pkg/mysql/datasource.mysql.go | 381 | No | medium |
| `singleValueOps` | pkg/mysql/datasource.mysql.go | 761 | No | medium |
| `IsPrimaryKey` | pkg/mysql/datasource.mysql.go | 315 | No | medium |
| `jsonMap` | pkg/mysql/datasource.mysql.go | 375 | No | medium |
| `jsonString` | pkg/mysql/datasource.mysql.go | 387 | No | medium |
| `exists` | pkg/mysql/mysql.go | 82 | No | medium |
| `jsonArray` | pkg/oracle/datasource.oracle.go | 635 | No | medium |
| `jsonString` | pkg/oracle/datasource.oracle.go | 641 | No | medium |
| `singleValueOps` | pkg/oracle/datasource.oracle.go | 1054 | No | medium |
| `exists` | pkg/oracle/datasource.oracle.go | 569 | No | medium |
| `IsPrimaryKey` | pkg/oracle/datasource.oracle.go | 570 | No | medium |
| `jsonMap` | pkg/oracle/datasource.oracle.go | 629 | No | medium |
| `exists` | pkg/oracle/oracle.go | 90 | No | medium |
| `jsonArray` | pkg/postgres/datasource.postgres.go | 785 | No | medium |
| `jsonString` | pkg/postgres/datasource.postgres.go | 791 | No | medium |
| `singleValueOps` | pkg/postgres/datasource.postgres.go | 1410 | No | medium |
| `exists` | pkg/postgres/datasource.postgres.go | 655 | No | medium |
| `IsPrimaryKey` | pkg/postgres/datasource.postgres.go | 656 | No | medium |
| `jsonMap` | pkg/postgres/datasource.postgres.go | 779 | No | medium |
| `v` | pkg/rabbitmq/rabbitmq.go | 1158 | No | high |
| `found` | pkg/rabbitmq/rabbitmq.go | 967 | No | medium |
| `redisCache` | pkg/redis/factory.go | 44 | No | medium |
| `value` | pkg/redis/redis\_cache.go | 86 | No | medium |
| `exists` | pkg/sqlserver/datasource.sqlserver.go | 511 | No | medium |
| `IsPrimaryKey` | pkg/sqlserver/datasource.sqlserver.go | 512 | No | medium |
| `jsonMap` | pkg/sqlserver/datasource.sqlserver.go | 572 | No | medium |
| `jsonArray` | pkg/sqlserver/datasource.sqlserver.go | 578 | No | medium |
| `jsonString` | pkg/sqlserver/datasource.sqlserver.go | 584 | No | medium |
| `singleValueOps` | pkg/sqlserver/datasource.sqlserver.go | 977 | No | medium |
| `exists` | pkg/sqlserver/sqlserver.go | 82 | No | medium |
| `err` | tests/shared/assertions.go | 133 | No | medium |
| `val` | tests/shared/infra.go | 34 | No | medium |


## High Risk Nil Sources


### `message` at `components/manager/internal/services/command/create\_fetcher\_job.go:389`
**Expression:** `message`
**Checked:** No
**Notes:** map\_lookup

### `systemSchemas` at `components/manager/internal/services/query/get\_connection\_schema.go:263`
**Expression:** `systemSchemas`
**Checked:** No
**Notes:** map\_lookup

### `systemTables` at `components/manager/internal/services/query/get\_connection\_schema.go:226`
**Expression:** `systemTables`
**Checked:** No
**Notes:** map\_lookup

### `systemDatabases` at `components/manager/internal/services/query/get\_connection\_schema.go:290`
**Expression:** `systemDatabases`
**Checked:** No
**Notes:** map\_lookup

### `schemas` at `components/manager/internal/services/query/validate\_schema.go:164`
**Expression:** `schemas`
**Checked:** No
**Notes:** lookup\_operation

### `pluginCRMTableMapping` at `components/manager/internal/services/query/validate\_schema.go:35`
**Expression:** `pluginCRMTableMapping`
**Checked:** No
**Notes:** map\_lookup

### `orgMatches` at `components/worker/internal/services/extract-data.go:256`
**Expression:** `orgMatches`
**Checked:** No
**Notes:** lookup\_operation

### `matches` at `components/worker/internal/services/extract-data.go:249`
**Expression:** `matches`
**Checked:** No
**Notes:** lookup\_operation

### `encryptedFields` at `components/worker/internal/services/extract\_crm\_data.go:314`
**Expression:** `encryptedFields`
**Checked:** No
**Notes:** map\_lookup

### `fieldMappings` at `components/worker/internal/services/extract\_crm\_data.go:180`
**Expression:** `fieldMappings`
**Checked:** No
**Notes:** map\_lookup

### `validMongoDBModes` at `pkg/datasource/sslmode/mongodb.go:23`
**Expression:** `validMongoDBModes`
**Checked:** No
**Notes:** map\_lookup

### `validMySQLModes` at `pkg/datasource/sslmode/mysql.go:22`
**Expression:** `validMySQLModes`
**Checked:** No
**Notes:** map\_lookup

### `validOracleModes` at `pkg/datasource/sslmode/oracle.go:25`
**Expression:** `validOracleModes`
**Checked:** No
**Notes:** map\_lookup

### `validPostgreSQLModes` at `pkg/datasource/sslmode/postgresql.go:21`
**Expression:** `validPostgreSQLModes`
**Checked:** No
**Notes:** map\_lookup

### `validSQLServerModes` at `pkg/datasource/sslmode/sqlserver.go:22`
**Expression:** `validSQLServerModes`
**Checked:** No
**Notes:** map\_lookup

### `e` at `pkg/itestkit/addons/queuekit/assertions.go:251`
**Expression:** `e`
**Checked:** No
**Notes:** json\_unmarshal

### `a` at `pkg/itestkit/addons/queuekit/assertions.go:256`
**Expression:** `a`
**Checked:** No
**Notes:** json\_unmarshal

### `m` at `pkg/itestkit/addons/queuekit/assertions.go:278`
**Expression:** `m`
**Checked:** No
**Notes:** json\_unmarshal

### `consumer` at `pkg/itestkit/addons/queuekit/consumer.go:392`
**Expression:** `consumer`
**Checked:** No
**Notes:** map\_lookup

### `result` at `pkg/itestkit/addons/queuekit/consumer.go:210`
**Expression:** `result`
**Checked:** No
**Notes:** map\_lookup

### `current` at `pkg/itestkit/addons/queuekit/matcher.go:213`
**Expression:** `current`
**Checked:** No
**Notes:** map\_lookup

### `Containers` at `pkg/itestkit/container\_generic.go:79`
**Expression:** `Containers`
**Checked:** No
**Notes:** map\_lookup

### `NetworkAliases` at `pkg/itestkit/customizer\_options.go:112`
**Expression:** `NetworkAliases`
**Checked:** No
**Notes:** map\_lookup

### `seen` at `pkg/itestkit/infra.go:20`
**Expression:** `seen`
**Checked:** No
**Notes:** map\_lookup

### `schemas` at `pkg/model/datasource/oracle/datasource-config.go:65`
**Expression:** `schemas`
**Checked:** No
**Notes:** lookup\_operation

### `schemas` at `pkg/model/datasource/postgres/datasource-config.go:65`
**Expression:** `schemas`
**Checked:** No
**Notes:** lookup\_operation

### `schemas` at `pkg/model/datasource/sqlserver/datasource-config.go:68`
**Expression:** `schemas`
**Checked:** No
**Notes:** lookup\_operation

### `config` at `pkg/mongodb/connection/connection.mongodb.go:83`
**Expression:** `config`
**Checked:** No
**Notes:** map\_lookup

### `v` at `pkg/mongodb/datasource.mongodb.go:206`
**Expression:** `v`
**Checked:** No
**Notes:** type\_assertion

### `typeHierarchy` at `pkg/mongodb/datasource.mongodb.go:559`
**Expression:** `typeHierarchy`
**Checked:** No
**Notes:** map\_lookup

### `findOptions` at `pkg/mongodb/datasource.mongodb.go:655`
**Expression:** `findOptions`
**Checked:** No
**Notes:** lookup\_operation

### `config` at `pkg/mongodb/job/job.mongodb.go:80`
**Expression:** `config`
**Checked:** No
**Notes:** map\_lookup

### `opts` at `pkg/mongodb/job/job.mongodb.go:430`
**Expression:** `opts`
**Checked:** No
**Notes:** lookup\_operation

### `opts` at `pkg/mongodb/product/product.mongodb.go:216`
**Expression:** `opts`
**Checked:** No
**Notes:** lookup\_operation

### `config` at `pkg/mongodb/product/product.mongodb.go:82`
**Expression:** `config`
**Checked:** No
**Notes:** map\_lookup

### `v` at `pkg/rabbitmq/rabbitmq.go:1158`
**Expression:** `v`
**Checked:** No
**Notes:** type\_assertion


## Focus Areas

Based on analysis, pay special attention to:

1. **Unchecked Nil Sources** - 114 potential nil values without checks
2. **High-Risk Nil Sources** - 36 high-risk nil sources require immediate attention

