# Slice: infrastructure-and-test-support

Description: Build/module configuration plus shared integration-test infrastructure and helpers.

## Files
- Makefile
- go.mod
- go.sum
- pkg/itestkit/addons/e2ekit/build_secrets.go
- pkg/itestkit/addons/e2ekit/builder.go
- pkg/itestkit/addons/e2ekit/helpers.go
- pkg/itestkit/addons/metricskit/assertions.go
- pkg/itestkit/addons/metricskit/error_classifier.go
- pkg/itestkit/addons/metricskit/metrics.go
- pkg/itestkit/addons/metricskit/reporter.go
- pkg/itestkit/addons/queuekit/amqp.go
- pkg/itestkit/addons/queuekit/assertions.go
- pkg/itestkit/addons/queuekit/consumer.go
- pkg/itestkit/addons/queuekit/matcher.go
- pkg/itestkit/addons/queuekit/queuekit.go
- pkg/itestkit/chaos_toxiproxy.go
- pkg/itestkit/container_generic.go
- pkg/itestkit/customizer.go
- pkg/itestkit/customizer_options.go
- pkg/itestkit/hostport.go
- pkg/itestkit/infra.go
- pkg/itestkit/infra/mongodb/mongodb.go
- pkg/itestkit/infra/mongodb/mongodb_options.go
- pkg/itestkit/infra/mssql/mssql.go
- pkg/itestkit/infra/mssql/mssql_options.go
- pkg/itestkit/infra/mysql/mysql.go
- pkg/itestkit/infra/mysql/mysql_options.go
- pkg/itestkit/infra/oracle/oracle.go
- pkg/itestkit/infra/oracle/oracle_options.go
- pkg/itestkit/infra/postgres/postgres.go
- pkg/itestkit/infra/postgres/postgres_options.go
- pkg/itestkit/infra/rabbitmq/rabbit_options.go
- pkg/itestkit/infra/rabbitmq/rabbitmq.go
- pkg/itestkit/infra/redis/redis.go
- pkg/itestkit/infra/redis/redis_options.go
- pkg/itestkit/infra/seaweedfs/seaweedfs.go
- pkg/itestkit/infra/seaweedfs/seaweedfs_options.go
- pkg/itestkit/suite.go
- pkg/testutil/mocks.go
- tests/shared/apps.go
- tests/shared/assertions.go
- tests/shared/client.go
- tests/shared/infra.go

## Mithril Highlights
- Security scanner findings: `pkg/itestkit/addons/e2ekit/build_secrets.go:107`, `:108`, and `:113` were flagged with `gosec G104` for unhandled errors.
- Nil safety hot spots include `pkg/itestkit/addons/queuekit/assertions.go`, `pkg/itestkit/addons/queuekit/consumer.go`, `pkg/itestkit/addons/queuekit/matcher.go`, `pkg/itestkit/container_generic.go:79`, `pkg/itestkit/customizer_options.go:112`, `pkg/itestkit/infra.go:20`, and `tests/shared/apps.go:104`.
- Testing: many newly changed itestkit helpers show `0 tests` in Mithril, with only a few matcher/consumer helpers and some shared assertions carrying direct coverage.
- Impact: this slice reshapes shared test/build infrastructure used across end-to-end and integration flows, so reviewers should look for hidden breakage caused by environment, container, or secret-handling assumptions.
- Context files available: `docs/codereview/context-code-reviewer.md`, `docs/codereview/context-security-reviewer.md`, `docs/codereview/context-test-reviewer.md`, `docs/codereview/context-nil-safety-reviewer.md`, `docs/codereview/impact-summary-go.md`, `docs/codereview/security-summary.md`.
