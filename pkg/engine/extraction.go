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

// ProtectionApplier names which layer applied result-byte protection. It is a
// CLOSED enumeration: only the three values below are valid, so a reviewer can
// see exactly who is accountable for the protection state on any result.
type ProtectionApplier string

const (
	// ProtectionAppliedByEngine indicates the Engine core applied protection.
	ProtectionAppliedByEngine ProtectionApplier = "engine"
	// ProtectionAppliedByAdapter indicates a host storage adapter applied it.
	ProtectionAppliedByAdapter ProtectionApplier = "adapter"
	// ProtectionAppliedByHost indicates the host process applied it outside the
	// Engine and its adapters.
	ProtectionAppliedByHost ProtectionApplier = "host"
)

var validProtectionAppliers = map[ProtectionApplier]struct{}{
	ProtectionAppliedByEngine:  {},
	ProtectionAppliedByAdapter: {},
	ProtectionAppliedByHost:    {},
}

// IsValid reports whether the applier is one of the closed, accountable values.
func (a ProtectionApplier) IsValid() bool {
	_, ok := validProtectionAppliers[a]
	return ok
}

// ResultIntegrity is the canonical integrity model over extracted RESULT bytes.
// It is DISTINCT from credential protection (see ProtectedCredential): it
// describes the integrity of a result payload, never secret material.
//
// When integrity applies, Algorithm names the scheme and EXACTLY one of Digest
// or Signature carries the value: a Digest is an unkeyed content hash (e.g.
// SHA-256), a Signature is a keyed MAC (e.g. HMAC-SHA-256). HMAC is therefore
// ONE possible integrity signature — it is not the result identity itself.
type ResultIntegrity struct {
	// Algorithm names the integrity scheme (e.g. "SHA-256", "HMAC-SHA-256").
	Algorithm string `json:"algorithm,omitempty"`
	// Digest is the hex-encoded unkeyed content hash, when integrity is a digest.
	Digest string `json:"digest,omitempty"`
	// Signature is the hex-encoded keyed MAC, when integrity is a signature.
	Signature string `json:"signature,omitempty"`
}

// IsPresent reports whether integrity metadata actually carries a value
// (a digest or a signature). An empty struct means no integrity was applied.
func (i ResultIntegrity) IsPresent() bool {
	return i.Digest != "" || i.Signature != ""
}

// ResultProtection is the canonical confidentiality model over extracted RESULT
// bytes. It is its OWN type — it deliberately does NOT reuse the credential
// encryption sidecar (ProtectedCredential) or its terminology. It describes the
// protection STATE of a result, not credential ciphertext, and embeds no bytes.
type ResultProtection struct {
	// Encrypted reports whether the result bytes are encrypted at rest.
	Encrypted bool `json:"encrypted"`
	// KeyVersion records which key version protected the bytes, when known.
	// It is result-key metadata and is unrelated to credential key versions.
	KeyVersion int `json:"keyVersion,omitempty"`
	// Mode names the encryption mode when applicable (e.g. "AES-256-GCM").
	Mode string `json:"mode,omitempty"`
	// AppliedBy names the accountable layer; it MUST be a valid ProtectionApplier
	// ("engine", "adapter", or "host").
	AppliedBy ProtectionApplier `json:"appliedBy,omitempty"`
}

// DirectResult is the DIRECT-mode output: the extracted bytes returned inline,
// with no sink reference. It carries the bytes plus canonical metadata and,
// optionally, canonical integrity and protection applied by the Engine or host.
type DirectResult struct {
	// Data is the serialized result payload returned inline to the host.
	Data []byte `json:"data,omitempty"`
	// Format is the output format (e.g. "json").
	Format string `json:"format,omitempty"`
	// RowCount is the total number of rows extracted across all tables.
	RowCount int64 `json:"rowCount,omitempty"`
	// PlaintextSize records the byte size of the result before any protection.
	PlaintextSize int64 `json:"plaintextSize,omitempty"`
	// Integrity optionally describes integrity over Data, when applied.
	Integrity *ResultIntegrity `json:"integrity,omitempty"`
	// Protection optionally describes confidentiality over Data, when applied.
	Protection *ResultProtection `json:"protection,omitempty"`
	// CompletedAt records when the extraction finished.
	CompletedAt time.Time `json:"completedAt,omitempty"`
}

// ResultReference is the STORE-mode contract pointing to a persisted extraction
// result. It carries no payload bytes and no secrets — only a LOGICAL location
// plus canonical metadata that the host uses to retrieve and verify the result.
// It intentionally exposes no physical backend type (no S3/SeaweedFS/Mongo
// fields): the physical storage adapter lives in the host, behind the ResultSink
// port, and resolves the logical Path to its own backend.
type ResultReference struct {
	// Path is the LOGICAL reference to the persisted result in host storage.
	Path string `json:"path,omitempty"`
	// Format is the output format of the persisted bytes (e.g. "json").
	Format string `json:"format,omitempty"`
	// RowCount is the total number of rows in the persisted result.
	RowCount int64 `json:"rowCount,omitempty"`
	// SizeBytes records the serialized (written) result size.
	SizeBytes int64 `json:"sizeBytes,omitempty"`
	// Integrity optionally describes integrity over the written bytes.
	Integrity *ResultIntegrity `json:"integrity,omitempty"`
	// Protection optionally describes confidentiality over the written bytes.
	Protection *ResultProtection `json:"protection,omitempty"`
	// CompletedAt records when the extraction finished.
	CompletedAt time.Time `json:"completedAt,omitempty"`
}

// ExtractionResult is the output contract summarizing a finished extraction. It
// models the two result shapes SEPARATELY: Direct carries inline bytes (direct
// mode), Reference points to persisted bytes (store mode). Exactly one is
// populated for a successful extraction depending on whether a ResultSink is
// configured.
type ExtractionResult struct {
	// State is the terminal execution state.
	State ExecutionState `json:"state"`
	// Direct carries the inline result payload in direct mode, if any.
	Direct *DirectResult `json:"direct,omitempty"`
	// Reference points to the persisted result payload in store mode, if any.
	Reference ResultReference `json:"reference"`
	// RowCounts records rows extracted per qualified table (optional).
	RowCounts map[string]int64 `json:"rowCounts,omitempty"`
}
