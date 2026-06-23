// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package engine is the embedded runtime core for canonical data extraction.
//
// It owns the rules of Fetcher — connection lifecycle, schema discovery and
// validation, query planning, extraction execution, result and error
// contracts, limits, and tenant-safety — behind host-provided port interfaces.
// It depends on no infrastructure: a build-enforced boundary (dependency_test.go)
// forbids importing HTTP frameworks, message brokers, database drivers, object
// storage SDKs, tenant-runtime middleware, or the Manager/Worker internals, so
// any host application can embed it in-process.
//
// The Fetcher Manager and Worker now run over this engine; the legacy
// worker extraction path has been removed.
package engine
