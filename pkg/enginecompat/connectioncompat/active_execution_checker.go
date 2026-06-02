// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package connectioncompat hosts the Manager-side compatibility adapters that
// let the live Manager connection services delegate shared, tenant-scoped
// connection policy to the embedded Engine (pkg/engine) without the Engine
// importing any Manager internals. The Manager imports this package; the Engine
// never does. This keeps the pkg/engine dependency boundary
// (engine/dependency_test.go) intact: the adapter wraps the host's existing job
// repository to satisfy the Engine's ActiveExecutionChecker port.
package connectioncompat

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/ports/job"
)

// JobActiveExecutionChecker adapts the Manager's job repository to the Engine's
// engine.ActiveExecutionChecker port. The Engine consults this seam BEFORE
// mutating a connection (update/delete) so the Manager's long-standing
// "block changes while jobs run against the connection" behavior is preserved —
// but the decision now flows through the Engine under a per-request tenant
// scope instead of being hard-coded in the Manager service.
//
// The adapter forwards to the SAME job-repository method
// (ExistsRunningByMappedFieldKey) the Manager called directly before this
// migration, keyed by the connection's config name. That is what lets the
// existing Manager service tests keep their jobRepo expectations unchanged: the
// call still lands on the job repository, only now via the Engine's gate.
type JobActiveExecutionChecker struct {
	jobRepo job.Repository
}

// NewJobActiveExecutionChecker builds the adapter from the host's job
// repository. A nil repository yields a nil adapter so the caller can treat
// "no job repo" as "no conflict gating" (the Engine port is optional).
func NewJobActiveExecutionChecker(jobRepo job.Repository) *JobActiveExecutionChecker {
	if jobRepo == nil {
		return nil
	}

	return &JobActiveExecutionChecker{jobRepo: jobRepo}
}

// HasActiveExecutions reports whether the named connection currently has active
// (running) jobs for the tenant. The connectionID is the connection's config
// name within the tenant scope — the same identity the Manager used as the
// mapped-field key before delegation. The tenant scope is carried by the
// Engine; the underlying job repository is already tenant-scoped via the
// request context, so the answer is correctly isolated per tenant.
func (c *JobActiveExecutionChecker) HasActiveExecutions(ctx context.Context, _ engine.TenantContext, connectionID string) (bool, error) {
	return c.jobRepo.ExistsRunningByMappedFieldKey(ctx, connectionID)
}
