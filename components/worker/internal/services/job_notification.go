package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg"
	libOutbox "github.com/LerianStudio/lib-commons/v6/commons/outbox"
	"github.com/LerianStudio/lib-commons/v6/commons/tenant-manager/core"
	observability "github.com/LerianStudio/lib-observability/v2"
	streaming "github.com/LerianStudio/lib-streaming/v3"

	libLog "github.com/LerianStudio/lib-observability/v2/log"

	"github.com/google/uuid"
)

// JobResultIntegrity is the event-payload projection of the canonical T-007
// integrity declaration (engine.ResultIntegrity). It exists so the published
// job event contract owns its own snake_case json tags independently of the
// engine package's internal serialization.
type JobResultIntegrity struct {
	// Algorithm names the integrity scheme (e.g. "HMAC-SHA256").
	Algorithm string `json:"algorithm,omitempty"`
	// Digest is the hex-encoded unkeyed content hash, when integrity is a digest.
	Digest string `json:"digest,omitempty"`
	// Signature is the hex-encoded keyed MAC, when integrity is a signature.
	Signature string `json:"signature,omitempty"`
}

// JobResultProtection is the event-payload projection of the canonical T-007
// confidentiality declaration (engine.ResultProtection), with snake_case json
// tags owned by the job event contract.
type JobResultProtection struct {
	// Encrypted reports whether the result bytes are encrypted at rest.
	Encrypted bool `json:"encrypted"`
	// KeyVersion records which key version protected the bytes, when known.
	KeyVersion int `json:"key_version,omitempty"`
	// Mode names the encryption mode when applicable (e.g. "adapter-managed").
	Mode string `json:"mode,omitempty"`
	// AppliedBy names the accountable layer ("engine", "adapter", or "host").
	AppliedBy engine.ProtectionApplier `json:"applied_by,omitempty"`
}

// JobResultData contains information about the extraction result.
// All fields use omitempty to only include data when provided.
type JobResultData struct {
	// Path is the SeaweedFS path where result data is stored.
	Path string `json:"path,omitempty"`

	// SizeBytes is the size of the result data in bytes (before encryption).
	SizeBytes int64 `json:"size_bytes,omitempty"`

	// RowCount is the total number of records extracted across all tables.
	RowCount int64 `json:"row_count,omitempty"`

	// Format is the output format (e.g., "json").
	Format string `json:"format,omitempty"`

	// HMAC is the HMAC-SHA256 signature of the result data (before encryption).
	// Consumers can use this to verify data integrity using the external HMAC key.
	HMAC string `json:"hmac,omitempty"`

	// Integrity is the canonical T-007 integrity declaration over the stored result.
	// It makes the tacit "what does the HMAC sign" convention EXPLICIT: the signature
	// is the keyed HMAC-SHA256 over the PLAINTEXT extraction JSON (the same value as
	// HMAC above). It is a keyed signature, never an unkeyed Digest.
	Integrity *JobResultIntegrity `json:"integrity,omitempty"`

	// Protection is the canonical T-007 confidentiality declaration over the stored
	// result bytes. It describes the STORED EXTRACTION RESULT only — never connection
	// credentials (T-007 invariant: result-protection != credential-protection).
	Protection *JobResultProtection `json:"protection,omitempty"`
}

// JobNotificationOptions contains optional data for job notifications.
type JobNotificationOptions struct {
	// Result contains extraction result data (path, size, rowCount, format).
	Result *JobResultData

	// ExecutionTimeMs is the total execution time in milliseconds.
	ExecutionTimeMs int64

	// CompletedAt is the timestamp when the job completed.
	CompletedAt *time.Time
}

// JobNotificationMessage represents the structure of a job event notification published to RabbitMQ.
type JobNotificationMessage struct {
	// JobID is the unique identifier of the job.
	JobID uuid.UUID `json:"job_id"`

	// Metadata contains additional metadata for the notification.
	// It should include "source" to identify which service requested the job.
	// For failed jobs, it may include "error" with error details.
	Metadata map[string]any `json:"metadata"`

	// Status indicates the job status: "completed" or "failed".
	Status string `json:"status"`

	// Result contains information about the extraction result (optional, only on success).
	Result *JobResultData `json:"result,omitempty"`

	// ExecutionTimeMs is the total execution time in milliseconds (optional).
	ExecutionTimeMs int64 `json:"execution_time_ms,omitempty"`

	// CompletedAt is the timestamp when the job completed (optional).
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

const (
	terminalEventPendingMetadataKey = "terminalEventPending"
	terminalEventStatusMetadataKey  = "terminalEventStatus"
	terminalEventPayloadMetadataKey = "terminalEventPayload"
)

func buildJobNotificationPayload(message ExtractExternalDataMessage, status string, errorMetadata map[string]any, opts *JobNotificationOptions) ([]byte, error) {
	notification := JobNotificationMessage{
		JobID:    message.JobID,
		Status:   status,
		Metadata: make(map[string]any),
	}

	if opts != nil {
		notification.Result = opts.Result
		notification.ExecutionTimeMs = opts.ExecutionTimeMs
		notification.CompletedAt = opts.CompletedAt
	}

	if message.Metadata != nil {
		for k, v := range message.Metadata {
			notification.Metadata[k] = v
		}
	}

	if status == "failed" && errorMetadata != nil {
		notification.Metadata["error"] = errorMetadata
	}

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("marshalling job notification: %w", err)
	}

	return notificationJSON, nil
}

func (uc *UseCase) emitJobNotificationEvent(ctx context.Context, status, subject string, payload []byte) error {
	if uc.JobEventEmitter == nil {
		return fmt.Errorf("mandatory lib-streaming job event emitter is not configured")
	}

	tenantID := core.GetTenantIDContext(ctx)
	if tenantID == "" {
		tenantID = "single-tenant"
	}

	outboxCtx, err := libOutbox.ContextWithTenantIDStrict(ctx, tenantID)
	if err != nil {
		return err
	}

	if err := uc.JobEventEmitter.Emit(outboxCtx, streaming.EmitRequest{
		DefinitionKey: fmt.Sprintf("job.%s", status),
		EventID:       deterministicJobEventID(status, subject),
		TenantID:      tenantID,
		Subject:       subject,
		Payload:       payload,
	}); err != nil {
		if streaming.IsCallerError(err) {
			return pkg.FailedPreconditionError{
				Code:    "FET-0060",
				Title:   "Job Event Configuration Error",
				Message: fmt.Sprintf("non-retryable lib-streaming job event configuration error: %s", err.Error()),
				Err:     err,
			}
		}

		return err
	}

	return nil
}

// deterministicJobEventID returns the stable CloudEvents id (ce-id) for a job
// terminal event: "fetcher.job.<status>.<jobID>". This id is the consumer
// idempotency key and the ONLY dedup anchor for the job event stream.
//
// Delivery is at-least-once: the terminal-event repairer re-emits when the
// terminalEventPending flag survives a crash or a failed flag-clear, and
// lib-streaming allocates a fresh outbox row UUID per emit. The ce-id, however,
// is stable across every re-emit of the same (status, jobID) pair. Consumers
// MUST dedup on ce-id; they must not rely on the outbox UUID or on
// exactly-once delivery at the broker.
func deterministicJobEventID(status, jobID string) string {
	return fmt.Sprintf("fetcher.job.%s.%s", status, jobID)
}

func normalizeJobNotificationLogger(ctx context.Context, logger libLog.Logger) libLog.Logger {
	if logger != nil {
		return logger
	}

	ctxLogger := observability.NewLoggerFromContext(ctx)
	if ctxLogger != nil {
		return ctxLogger
	}

	return &libLog.GoLogger{Level: libLog.LevelError}
}
