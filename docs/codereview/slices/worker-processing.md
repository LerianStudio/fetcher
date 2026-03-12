# Slice: worker-processing

Description: Worker consumer/publisher adapters, worker services, and worker bootstrap/runtime wiring.

## Files
- components/worker/Dockerfile
- components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go
- components/worker/internal/adapters/rabbitmq/consumer_test.go
- components/worker/internal/adapters/rabbitmq/publisher.rabbitmq.go
- components/worker/internal/adapters/rabbitmq/publisher_test.go
- components/worker/internal/bootstrap/config.go
- components/worker/internal/bootstrap/consumer.go
- components/worker/internal/bootstrap/service.go
- components/worker/internal/services/extract-data.go
- components/worker/internal/services/extract_crm_data.go
- components/worker/internal/services/extract_crm_data_test.go
- components/worker/internal/services/job_notification.go
- components/worker/internal/services/test_helpers_test.go

## Mithril Highlights
- Security scanner findings: `components/worker/internal/bootstrap/config.go:187` flagged `gosec G115` for `int -> uint64` conversion overflow risk; `components/worker/internal/adapters/rabbitmq/consumer.rabbitmq.go:81` flagged `gosec G118` for spawning work with `context.Background/TODO` while scoped context exists.
- Nil safety: high-risk values flagged at `extract-data.go:147`, `extract-data.go:232`, `extract-data.go:239`, `extract_crm_data.go:144`, and `extract_crm_data.go:278`; medium-risk values include `errorMetadata`, `databaseExists`, `encryptSecretKey`, `hashSecretKey`, and `exists` branches.
- Testing: Mithril sees only light direct tests around consumer/publisher constructors and shutdown paths; worker bootstrap and service flows mostly show `0 tests`, so reviewers should inspect error-path and message-shape coverage carefully.
- Impact: worker runtime, queue consumers, and extraction paths changed together, so contract drift between bootstrap, adapters, and services is a likely ripple-effect zone.
- Context files available: `docs/codereview/context-code-reviewer.md`, `docs/codereview/context-security-reviewer.md`, `docs/codereview/context-test-reviewer.md`, `docs/codereview/context-nil-safety-reviewer.md`, `docs/codereview/impact-summary-go.md`, `docs/codereview/security-summary.md`.
