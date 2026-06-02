// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "time"

// ExecutionStatus is the stable lifecycle status of an extraction execution.
// Values are stable string constants; changing them is a breaking change.
type ExecutionStatus string

const (
	// StatusPending indicates the execution is accepted but not yet started.
	StatusPending ExecutionStatus = "pending"
	// StatusRunning indicates the execution is in progress.
	StatusRunning ExecutionStatus = "running"
	// StatusCompleted indicates the execution finished successfully.
	StatusCompleted ExecutionStatus = "completed"
	// StatusFailed indicates the execution finished with an error.
	StatusFailed ExecutionStatus = "failed"
	// StatusCanceled indicates the execution was canceled before completion.
	StatusCanceled ExecutionStatus = "canceled"
)

var validExecutionStatuses = map[ExecutionStatus]struct{}{
	StatusPending:   {},
	StatusRunning:   {},
	StatusCompleted: {},
	StatusFailed:    {},
	StatusCanceled:  {},
}

var terminalExecutionStatuses = map[ExecutionStatus]struct{}{
	StatusCompleted: {},
	StatusFailed:    {},
	StatusCanceled:  {},
}

// IsValid reports whether the status is a known, stable lifecycle status.
func (s ExecutionStatus) IsValid() bool {
	_, ok := validExecutionStatuses[s]
	return ok
}

// IsTerminal reports whether the status represents a finished execution.
func (s ExecutionStatus) IsTerminal() bool {
	_, ok := terminalExecutionStatuses[s]
	return ok
}

// FieldSelection maps a qualified table name to the field names to extract.
type FieldSelection map[string][]string

// ExtractionRequest is the input contract describing what to extract. It is a
// pure data contract: it owns no orchestration, persistence, or transport.
// MappedFields keys are datasource config names.
type ExtractionRequest struct {
	// MappedFields selects, per datasource, the tables and fields to extract.
	MappedFields map[string]FieldSelection `json:"mappedFields"`
	// Filters carries opaque, host-defined filter criteria keyed by datasource.
	// It is intentionally untyped at the contract layer; adapters interpret it.
	Filters map[string]any `json:"filters,omitempty"`
	// Metadata carries safe, non-secret request metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Overrides optionally requests per-request resource bounds. It is applied
	// over the Engine's default Limits with copy semantics: only fields set to a
	// positive value override the corresponding default, every override must be
	// within (<=) the Engine maximum, and the Engine's defaults are never mutated.
	// A nil Overrides leaves the Engine defaults in force.
	Overrides *Limits `json:"overrides,omitempty"`
}

// PlanStep describes a single planned datasource extraction unit. It is a
// deterministic, secret-free per-datasource work item: tables are sorted, the
// selected fields per table are sorted, and any matching filters are attached on
// their owning table path. It carries NO credential material — the host resolves
// credentials at execute time behind the connector seam.
type PlanStep struct {
	// ConfigName identifies the target datasource.
	ConfigName string `json:"configName"`
	// Tables lists the qualified tables the step will read, in sorted order.
	Tables []string `json:"tables"`
	// Fields maps each qualified table to its selected field names, sorted.
	Fields map[string][]string `json:"fields,omitempty"`
	// Filters carries the host-defined filter criteria for this datasource,
	// attached on the matching table path. It is intentionally untyped at the
	// contract layer; adapters interpret it. It carries no credential material.
	Filters map[string]map[string]any `json:"filters,omitempty"`
}

// ExtractionPlan is the contract describing how a request will be executed. It
// is derived from an ExtractionRequest by PlanExtraction: a deterministic,
// secret-free executable plan. The plan carries the tenant identity (tenantId
// only — never organization or product) plus an optional request id for
// downstream adapters and observability, and preserves safe request metadata
// (including metadata.source) for compatibility paths.
type ExtractionPlan struct {
	// TenantID is the isolation boundary the plan was built under. It is the
	// SOLE tenant identity the plan carries; the Engine never carries an
	// organization or product concept.
	TenantID string `json:"tenantId"`
	// RequestID correlates the plan with a single host request. Optional.
	RequestID string `json:"requestId,omitempty"`
	// Steps enumerates the planned per-datasource extraction units, in sorted
	// order by config name.
	Steps []PlanStep `json:"steps"`
	// Metadata carries safe, non-secret request metadata preserved for
	// compatibility paths (e.g. metadata.source = plugin_crm).
	Metadata map[string]any `json:"metadata,omitempty"`
	// Limits records the effective bounds applied to the plan.
	Limits Limits `json:"limits"`
}

// ExecutionState is the contract describing the live state of an execution.
// The zero value carries an empty (invalid) status, which callers treat as
// "not yet initialized".
type ExecutionState struct {
	// JobID is the host-assigned identifier for the execution.
	JobID string `json:"jobId,omitempty"`
	// Status is the current lifecycle status.
	Status ExecutionStatus `json:"status,omitempty"`
	// StartedAt records when execution began (optional).
	StartedAt time.Time `json:"startedAt,omitempty"`
	// CompletedAt records when execution finished (optional).
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	// FailureMessage carries a safe, credential-free failure description.
	FailureMessage string `json:"failureMessage,omitempty"`
}

// ResultReference is the contract pointing to a persisted extraction result.
// It carries no payload bytes and no secrets — only a location and integrity
// metadata that the host can use to retrieve and verify the result.
type ResultReference struct {
	// Path locates the persisted result in the host's storage.
	Path string `json:"path,omitempty"`
	// HMAC is the integrity signature over the persisted result.
	HMAC string `json:"hmac,omitempty"`
	// SizeBytes records the serialized result size.
	SizeBytes int64 `json:"sizeBytes,omitempty"`
}

// ExtractionResult is the output contract summarizing a finished extraction.
type ExtractionResult struct {
	// State is the terminal execution state.
	State ExecutionState `json:"state"`
	// Reference points to the persisted result payload, if any.
	Reference ResultReference `json:"reference"`
	// RowCounts records rows extracted per qualified table (optional).
	RowCounts map[string]int64 `json:"rowCounts,omitempty"`
}
