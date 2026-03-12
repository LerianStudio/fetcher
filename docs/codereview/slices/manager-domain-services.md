# Slice: manager-domain-services

Description: Manager command/query service logic and related service-level tests.

## Files
- components/manager/internal/services/command/assign_connection.go
- components/manager/internal/services/command/create_fetcher_job.go
- components/manager/internal/services/command/create_product.go
- components/manager/internal/services/command/delete_product.go
- components/manager/internal/services/command/helpers_test.go
- components/manager/internal/services/command/update_connection.go
- components/manager/internal/services/command/update_product.go
- components/manager/internal/services/query/get_connection_schema.go
- components/manager/internal/services/query/helpers_test.go
- components/manager/internal/services/query/test_connection.go
- components/manager/internal/services/query/validate_schema.go

## Mithril Highlights
- Nil safety: high-risk unchecked values flagged at `components/manager/internal/services/command/create_fetcher_job.go:374`, `components/manager/internal/services/query/get_connection_schema.go:218`, `components/manager/internal/services/query/get_connection_schema.go:255`, `components/manager/internal/services/query/get_connection_schema.go:282`, `components/manager/internal/services/query/validate_schema.go:35`, and `components/manager/internal/services/query/validate_schema.go:148`.
- Nil safety medium signals also exist around `header`, `found`, `fields`, `configNames`, `tables`, and `exists` in the same service/query files.
- Testing: Mithril coverage table reports `0 tests` for modified command/query methods; helper functions `testContext` in command/query tests are heavily reused and should be checked for hidden coverage and shared-fixture fragility.
- Impact: `testContext` in `components/manager/internal/services/command/helpers_test.go` is referenced by 95 callers; call graph analysis was partial/time-boxed, so downstream impact may be under-reported.
- Context files available: `docs/codereview/context-code-reviewer.md`, `docs/codereview/context-business-logic-reviewer.md`, `docs/codereview/context-test-reviewer.md`, `docs/codereview/context-nil-safety-reviewer.md`, `docs/codereview/impact-summary-go.md`.
