# Gate 1 PRD: Fetcher Embedded Runtime

## Executive Summary

Fetcher currently works as a separately deployed Manager and Worker service. That shape is useful for standalone operation, but it creates deployment and integration drag for internal products that already own their own operational lifecycle.

The Fetcher Embedded Runtime turns Fetcher into an importable Engine that host applications can run in-process. The Engine provides canonical datasource and extraction capabilities; host apps decide how those capabilities are exposed, secured, observed, stored, scheduled, and deployed.

This is not greenfield work. It is a strangler extraction from existing Fetcher service internals into a reusable Engine, while preserving current Manager and Worker behavior as optional compatibility adapters.

## Problem Statement

Internal products that need extraction capabilities must currently integrate with and operate a separate Fetcher deployment. That means another service boundary, another runtime, another queue/state/storage stack, and another set of deployment concerns across cloud, on-premises, and self-hosted environments.

If teams avoid that overhead by reimplementing extraction logic inside their own products, Fetcher stops being the canonical owner of datasource behavior. That would fragment security, connector behavior, validation rules, schema handling, and extraction semantics.

The actual gap is not lack of abstractions. Fetcher already has service orchestration, ports, models, connectors, adapters, and tests. The gap is that reusable orchestration is locked behind `components/*/internal`, and there is no importable composition root for host applications.

## Product Vision

Fetcher becomes an embedded runtime for canonical data extraction. Host applications import the Fetcher Engine and run connection management, schema discovery, validation, query planning, extraction, result handling, and tenant-safe behavior in-process.

The Engine owns datasource/source capabilities. Host applications own operational shell decisions: HTTP, authentication, license checks, queues, state, caches, storage, telemetry exporters, health endpoints, deployment, and process lifecycle.

The current Fetcher Manager and Worker remain available as optional compatibility adapters over the Engine.

## Goals

- Enable internal applications such as Reporter and Matcher to embed Fetcher capabilities without deploying a standalone Fetcher service.
- Preserve Fetcher as the canonical owner of datasource connectors and extraction semantics.
- Support connection lifecycle, schema discovery/validation, query planning, extraction execution, canonical result handling, limits, errors, and tenant-safety primitives.
- Let host applications choose synchronous, asynchronous, or hybrid execution patterns.
- Reduce deployment complexity across hybrid, cloud, on-premises, and self-hosted models.
- Keep current Manager and Worker behavior representable as compatibility adapters over the Engine.

## Non-Goals

- Do not move host application authentication or license enforcement into Engine core.
- Do not require Fiber handlers, Swagger routes, health endpoints, readiness endpoints, telemetry exporters, queues, or process launchers in Engine core.
- Do not make RabbitMQ, MongoDB, Redis, Docker Compose, KEDA, S3, or SeaweedFS mandatory Engine core dependencies.
- Do not force host applications to expose a specific HTTP API.
- Do not rewrite all existing service code in one pass.
- Do not turn Fetcher into a stateless helper library that loses runtime semantics.

## Target Users

- Reporter engineering teams embedding extraction inside reporting workflows.
- Matcher engineering teams needing controlled datasource access without a separate Fetcher deployment.
- Platform teams maintaining one canonical extraction capability across products.
- Operations teams supporting hybrid, cloud, on-premises, and self-hosted deployment models.
- Security and compliance stakeholders responsible for credential protection, tenant safety, and data ownership.

## Capabilities

- Embedded connection management for create, update, delete, list, retrieve, and test behavior.
- Canonical credential protection semantics with host-controlled key provisioning.
- Connector registry and datasource factory for MongoDB, PostgreSQL, MySQL, Oracle, SQL Server, and future connectors.
- Schema discovery and validation for requested tables, fields, filters, and datasource-specific conventions.
- Query planning across one or more datasources.
- Extraction runner with cancellation, timeouts, limits, and controlled concurrency.
- Canonical result model with row count, size, format, integrity metadata, and result location or payload.
- Host-provided result sinks and state stores.
- Consistent error taxonomy that host apps can map to their own APIs or workflows.
- Observability hooks that connect to host logging, metrics, and tracing without owning host endpoints.
- Tenant-safety primitives for product and tenant isolation.

## Ownership Boundaries

Engine owns:

- Connection model and validation semantics.
- Credential encryption/decryption semantics.
- Datasource connectors, connector registry, and factory behavior.
- Schema discovery and schema validation.
- Query planning and extraction execution.
- Canonical execution, result, and error models.
- Storage/result sink contracts, connection store contracts, and optional execution store contracts.
- Limits, cancellation semantics, observability hooks, and tenant-safety primitives.

Host applications own:

- HTTP route shape and compatibility API exposure.
- Authentication, authorization, and license middleware.
- Environment/config loading and key provisioning.
- Health, readiness, metrics endpoints, and telemetry exporters.
- Queueing, scheduling, async worker topology, state stores, schema caches, and storage choices.
- Deployment model, process lifecycle, Docker Compose, KEDA, Kubernetes, and runtime operations.
- Product-specific policy and UI.

## Functional Requirements

- The Engine must expose embedded operations for connection lifecycle management.
- The Engine must validate connection definitions before persistence or extraction.
- The Engine must protect credentials through canonical Fetcher semantics while accepting host-provided keys or credential protectors.
- The Engine must discover schemas from supported datasources.
- The Engine must validate requested tables, fields, and filters against discovered or provided schemas.
- The Engine must plan extraction work across one or more datasources.
- The Engine must execute extraction through canonical connector behavior.
- The Engine must return results through a canonical result model.
- The Engine must support host-provided result sinks and state stores.
- The Engine must expose lightweight execution tracking for in-process and host-scheduled execution.
- The Engine must expose errors that are stable enough for host API mapping.
- The Engine must support configurable limits for timeouts, result size, query scope, datasource count, table count, field count, and concurrency.
- The Engine must preserve adapter-facing seams for product-specific behavior without forcing connector forks. The first release keeps `plugin_crm` as explicit compatibility-adapter behavior, not generic Engine core extension behavior.

## Security and Data Ownership Requirements

- The Engine must preserve client ownership of data and infrastructure.
- The Engine must not require extracted data to leave the host application deployment boundary.
- The Engine must avoid logging raw credentials, connection secrets, or extracted sensitive data.
- The Engine must make authorization a host responsibility while preserving enough context for policy enforcement.
- The Engine must provide tenant and product isolation primitives.
- The Engine must keep datasource access semantics centralized to avoid inconsistent security behavior across products.

## Operational Requirements

- The Engine must run in-process inside host applications.
- The Engine must not require a separate Fetcher service deployment.
- The Engine must not require RabbitMQ, MongoDB, Redis, Fiber, Docker Compose, KEDA, S3, or SeaweedFS at core runtime level.
- The Engine must support hybrid, cloud, on-premises, and self-hosted deployments through host-controlled infrastructure.
- The Engine must support graceful cancellation and timeout behavior through host-provided context.
- The current Manager and Worker must remain viable compatibility adapters over the Engine.

## Success Metrics

- Reporter and Matcher can run Fetcher capabilities in-process without a standalone Fetcher service deployment.
- Supported datasource connector behavior remains centralized in Fetcher.
- Host applications do not reimplement MongoDB, PostgreSQL, MySQL, Oracle, or SQL Server extraction paths.
- Existing Fetcher Manager and Worker behavior can be expressed as adapters over the Engine.
- Security review confirms credential handling, tenant-safety primitives, and data ownership remain intact.
- Self-hosted and on-premises deployments remove at least one separately managed Fetcher runtime where embedded mode is adopted.

## Decisions

- Async execution is a host execution mode; RabbitMQ remains an adapter.
- Current Manager and Worker remain compatibility adapters preserving current behavior unless explicitly migrated.
- `plugin_crm` is adapter-level compatibility behavior for the first Engine release, not generic Engine core behavior.

## Open Questions

- What is the minimum embedded API surface needed by Reporter and Matcher?
- Which result sinks should ship as optional adapters in the first release?
- Which concrete key-provider adapter shape should implement host-owned key provisioning and rotation?
- Are any compatibility guarantees outside the TRD preservation list required for the first Engine release?
