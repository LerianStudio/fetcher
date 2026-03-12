# Slice: manager-api-handlers

Description: Manager HTTP entrypoints, routing, and component bootstrap/runtime wiring.

## Files
- components/manager/Dockerfile
- components/manager/internal/adapters/http/in/connection.go
- components/manager/internal/adapters/http/in/connection_test.go
- components/manager/internal/adapters/http/in/fetcher.go
- components/manager/internal/adapters/http/in/fetcher_test.go
- components/manager/internal/adapters/http/in/middlewares_test.go
- components/manager/internal/adapters/http/in/migration.go
- components/manager/internal/adapters/http/in/product.go
- components/manager/internal/adapters/http/in/routes.go
- components/manager/internal/bootstrap/config.go
- components/manager/internal/bootstrap/server.go
- components/manager/internal/bootstrap/service.go

## Mithril Highlights
- Security flows: high-risk http path parameter flows flagged in `connection.go`, `fetcher.go`, `migration.go`, and `product.go`; each source parses path UUIDs before service calls, so reviewers should verify whether the sink classification is a false positive or a real trust-boundary issue.
- Testing: Mithril coverage table reports `0 tests` for most modified handlers/bootstrap functions even though helper setup functions in nearby test files are heavily used; reviewers should verify real coverage instead of trusting the heuristic blindly.
- Impact: high-caller test setup functions include `setupConnectionTestApp` (46 callers), `setupTestApp` (20 callers), and `setupMiddlewareTestApp` (14 callers). Call graph analysis was partial/time-boxed.
- Nil safety: no direct high-risk nil source was surfaced for this slice, so reviewers should focus on request parsing, org/header propagation, and bootstrap config edges manually.
- Context files available: `docs/codereview/context-code-reviewer.md`, `docs/codereview/context-security-reviewer.md`, `docs/codereview/context-business-logic-reviewer.md`, `docs/codereview/context-test-reviewer.md`, `docs/codereview/context-nil-safety-reviewer.md`, `docs/codereview/impact-summary-go.md`, `docs/codereview/security-summary.md`.
