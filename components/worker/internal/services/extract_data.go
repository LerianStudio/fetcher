package services

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	modelJob "github.com/LerianStudio/fetcher/v2/pkg/model/job"
	tms3 "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/s3"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ExtractExternalDataMessage contains the information needed to extract external data
type ExtractExternalDataMessage struct {
	// JobID is the unique identifier of the job extract.
	JobID uuid.UUID `json:"jobId"`

	// DataQueries maps database names to tables and their fields.
	// Format: map[databaseName]map[tableName][]fieldName.
	// Example: {"onboarding": {"organization": ["name"], "ledger": ["id"]}}.
	MappedFields map[string]map[string][]string `json:"mappedFields"`

	// Filters specify advanced filtering criteria using FilterCondition for complex queries.
	// Format: map[databaseName]map[tableName]map[fieldName]model.FilterCondition
	// Example: {"db": {"table": {"created_at": {"gte": ["2025-06-01"], "lte": ["2025-06-30"]}}}}
	Filters map[string]map[string]map[string]modelJob.FilterCondition `json:"filters"`

	// Metadata contains additional metadata for the report.
	Metadata map[string]any `json:"metadata"`
}

// ExtractExternalData handles the extraction of data from external sources.
func (uc *UseCase) ExtractExternalData(ctx context.Context, body []byte, headers map[string]any) error {
	startTime := time.Now() // Track execution start time

	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	message, err := uc.parseMessage(ctx, body, headers, span, logger)
	if err != nil {
		jobID := uc.extractJobIDFromMultipleSources(body, headers, logger)
		if jobID != uuid.Nil {
			failedMessage := ExtractExternalDataMessage{
				JobID:    jobID,
				Metadata: map[string]any{},
			}

			if terminalErr := uc.handleErrorWithUpdate(ctx, jobID, failedMessage, span, "Invalid extraction message", err, logger); terminalErr != nil {
				return terminalErr
			}
		} else {
			logger.Log(ctx, libLog.LevelWarn, "dropping invalid extraction message without terminal job event: no usable job id found")
		}

		return err
	}

	if skip, retryErr := uc.shouldSkipProcessing(ctx, message.JobID, logger); skip {
		if retryErr != nil {
			return retryErr
		}

		return nil
	}

	job, errJob := uc.JobRepository.FindByID(ctx, message.JobID)
	if errJob != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, *message, span, "Error finding job by ID in database", errJob, logger)
	}

	// Check if job exists, if not, update job status to failed
	if job == nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, *message, span, "Job not found in database", nil, logger)
	}

	// Best-effort CAS: skip if another worker already moved this job past PENDING
	if job.Status != model.JobStatusPending {
		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Job %s status is %s (expected pending), skipping", message.JobID, job.Status))
		return nil
	}

	if err := uc.JobRepository.UpdateStatus(ctx, message.JobID, model.JobStatusProcessing, "", "", nil); err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, *message, span, "Error updating job status to processing", err, logger)
	}

	// Extract config names from mappedFields
	configNames := extractConfigNamesFromMappedFields(message.MappedFields)

	// Find connections by config names (use resolver if available, fallback to direct repo lookup)
	var connections []*model.Connection
	if uc.ConnectionResolver != nil {
		connections, err = uc.ConnectionResolver.ResolveConnections(ctx, configNames)
	} else {
		connections, err = uc.ConnectionRepository.FindByConfigNames(ctx, configNames)
	}

	if err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, *message, span, "Error finding connections by config names", err, logger)
	}

	// Check if connections exist, if not, update job status to failed
	if len(connections) == 0 {
		return uc.handleErrorWithUpdate(ctx, message.JobID, *message, span, "No connections found for config names", nil, logger)
	}

	result := make(map[string]map[string][]map[string]any)

	reuse, err := uc.extractInto(ctx, *message, connections, result)
	if err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, *message, span, "Error querying external data", err, logger)
	}

	resultData, err := uc.saveExternalData(ctx, tracer, *message, result, reuse, span, logger)
	if err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, *message, span, "Error saving external data to storage", err, logger)
	}

	return uc.completeJob(ctx, *message, resultData, startTime, span, logger)
}

// completeJob persists the completed status and publishes a completion notification.
func (uc *UseCase) completeJob(
	ctx context.Context,
	message ExtractExternalDataMessage,
	resultData *JobResultData,
	startTime time.Time,
	span trace.Span,
	logger libLog.Logger,
) error {
	if resultData == nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message, span,
			"Cannot complete job: result data is nil", nil, logger)
	}

	completedAt := time.Now()
	executionTimeMs := completedAt.Sub(startTime).Milliseconds()

	notificationOpts := &JobNotificationOptions{
		Result:          resultData,
		ExecutionTimeMs: executionTimeMs,
		CompletedAt:     &completedAt,
	}

	notificationPayload, err := buildJobNotificationPayload(message, "completed", nil, notificationOpts)
	if err != nil {
		libOtel.HandleSpanError(span, "Error marshalling job completion notification", err)
		return fmt.Errorf("build required job completion notification: %w", err)
	}

	metadata := terminalEventPendingMetadata("completed", notificationPayload)
	if err := uc.JobRepository.UpdateStatus(ctx, message.JobID, model.JobStatusCompleted, resultData.Path, resultData.HMAC, metadata); err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message, span, "Error updating job status to completed", err, logger)
	}

	if err := uc.publishJobNotificationPayload(ctx, "completed", message.JobID.String(), notificationPayload, logger); err != nil {
		libOtel.HandleSpanError(span, "Error publishing job completion notification", err)
		logger.Log(ctx, libLog.LevelError, "failed to publish required job completion notification",
			libLog.String("job_id", message.JobID.String()),
			libLog.Err(err),
		)

		return fmt.Errorf("publish required job completion notification: %w", err)
	}

	uc.clearPendingTerminalEvent(ctx, message.JobID, logger)

	return nil
}

// parseMessage parses the RabbitMQ message body into ExtractExternalDataMessage struct.
func (uc *UseCase) parseMessage(ctx context.Context, body []byte, headers map[string]any, span trace.Span, logger libLog.Logger) (*ExtractExternalDataMessage, error) {
	var message *ExtractExternalDataMessage

	err := json.Unmarshal(body, &message)
	if err == nil && message == nil {
		err = fmt.Errorf("empty message payload")
	}

	if err != nil {
		libOtel.HandleSpanError(span, "Error unmarshalling message.", err)
		logger.Log(ctx, libLog.LevelError, "error unmarshalling message", libLog.Err(err))

		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	if validationErr := validateExtractExternalDataMessage(message); validationErr != nil {
		wrappedErr := fmt.Errorf("invalid message payload: %w", validationErr)
		libOtel.HandleSpanError(span, "Invalid message payload", wrappedErr)
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Invalid message payload: %s", wrappedErr.Error()))

		var jobID uuid.UUID

		if message != nil {
			jobID = message.JobID
		}

		if jobID == uuid.Nil {
			extractedJobID := uc.extractJobIDFromMultipleSources(body, headers, logger)
			jobID = extractedJobID
		}

		if jobID == uuid.Nil {
			logger.Log(ctx, libLog.LevelWarn, "Could not extract job ID from payload, job status will not be updated")
		}

		return nil, wrappedErr
	}

	return message, nil
}

func validateExtractExternalDataMessage(message *ExtractExternalDataMessage) error {
	if message == nil {
		return pkg.ValidationError{Code: "FET-0050", Title: "Invalid Message", Message: "message payload is null"}
	}

	if message.JobID == uuid.Nil {
		return pkg.ValidationError{Code: "FET-0051", Title: "Invalid Message", Message: "jobId is required"}
	}

	if len(message.MappedFields) == 0 {
		return pkg.ValidationError{Code: "FET-0052", Title: "Invalid Message", Message: "mappedFields is required"}
	}

	for db, tables := range message.MappedFields {
		if len(tables) == 0 {
			return pkg.ValidationError{Code: "FET-0053", Title: "Invalid Message", Message: fmt.Sprintf("mappedFields[%q] has no tables", db)}
		}
	}

	return nil
}

// extractJobIDFromMultipleSources attempts to extract jobID from multiple sources.
func (uc *UseCase) extractJobIDFromMultipleSources(body []byte, headers map[string]any, logger libLog.Logger) uuid.UUID {
	if headers != nil {
		if jobIDHeader, exists := headers[constant.HeaderJobID]; exists {
			if jobIDStr, ok := jobIDHeader.(string); ok {
				jobID, err := uuid.Parse(jobIDStr)
				if err == nil {
					logger.Log(context.Background(), libLog.LevelInfo, "extracted job id from header", libLog.String("job_id", jobID.String()))
					return jobID
				}
			}
		}
	}

	jobID := uc.extractJobIDFromPartialJSON(body, logger)
	if jobID != uuid.Nil {
		return jobID
	}

	return uuid.Nil
}

// extractJobIDFromPartialJSON attempts to extract jobID from a potentially malformed JSON.
func (uc *UseCase) extractJobIDFromPartialJSON(body []byte, logger libLog.Logger) uuid.UUID {
	bodyStr := string(body)

	var partial struct {
		JobID *string `json:"jobId"`
	}

	decoder := json.NewDecoder(strings.NewReader(bodyStr))
	_ = decoder.Decode(&partial)

	if partial.JobID != nil {
		jobID, err := uuid.Parse(*partial.JobID)
		if err == nil {
			logger.Log(context.Background(), libLog.LevelInfo, "extracted job id from partial json", libLog.String("job_id", jobID.String()))
			return jobID
		}
	}

	// Limit whitespace to prevent ReDoS (use {0,10} instead of * to cap backtracking)
	jobIDRegex := regexp.MustCompile(`"jobId"\s{0,10}:\s{0,10}"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)

	matches := jobIDRegex.FindStringSubmatch(bodyStr)
	if len(matches) > 1 {
		jobID, err := uuid.Parse(matches[1])
		if err == nil {
			logger.Log(context.Background(), libLog.LevelInfo, "extracted job id from regex", libLog.String("job_id", jobID.String()))
			return jobID
		}
	}

	return uuid.Nil
}

// handleErrorWithUpdate logs error, updates report status to error, and publishes failure notification.
func (uc *UseCase) handleErrorWithUpdate(
	ctx context.Context,
	jobID uuid.UUID,
	message ExtractExternalDataMessage,
	span trace.Span,
	errorMsg string,
	err error,
	logger libLog.Logger,
) error {
	if err == nil {
		err = fmt.Errorf("operation failed: %s", errorMsg)
	}

	errorMetadata := map[string]any{
		"message": sanitizeErrorForNotification(err.Error()),
	}

	// Ensure message has correct IDs (in case it was partially parsed)
	message.JobID = jobID

	notificationPayload, payloadErr := buildJobNotificationPayload(message, "failed", errorMetadata, nil)
	if payloadErr != nil {
		libOtel.HandleSpanError(span, "Error marshalling job failure notification", payloadErr)
		return fmt.Errorf("build required job failure notification: %w", payloadErr)
	}

	metadata := terminalEventPendingMetadata("failed", notificationPayload)
	metadata["error"] = sanitizeErrorForNotification(err.Error())

	if errUpdate := uc.JobRepository.UpdateStatus(ctx, jobID, model.JobStatusFailed, "", "", metadata); errUpdate != nil {
		libOtel.HandleSpanError(span, "Error to update report status with error.", errUpdate)
		logger.Log(ctx, libLog.LevelError, "error updating report status with error",
			libLog.String("job_id", jobID.String()),
			libLog.Err(errUpdate),
		)

		return fmt.Errorf("failed to update job status: %w", errUpdate)
	}

	libOtel.HandleSpanError(span, errorMsg, err)
	logger.Log(ctx, libLog.LevelError, errorMsg,
		libLog.String("job_id", jobID.String()),
		libLog.Err(err),
	)

	if errNotify := uc.publishJobNotificationPayload(ctx, "failed", jobID.String(), notificationPayload, logger); errNotify != nil {
		logger.Log(ctx, libLog.LevelError, "failed to publish required job failure notification",
			libLog.String("job_id", jobID.String()),
			libLog.Err(errNotify),
		)

		return fmt.Errorf("publish required job failure notification: %w", errNotify)
	}

	uc.clearPendingTerminalEvent(ctx, jobID, logger)

	return err
}

func terminalEventPendingMetadata(status string, payload []byte) map[string]any {
	return map[string]any{
		terminalEventPendingMetadataKey: true,
		terminalEventStatusMetadataKey:  status,
		terminalEventPayloadMetadataKey: string(payload),
	}
}

func (uc *UseCase) retryPendingTerminalEventForJob(ctx context.Context, job *model.Job, logger libLog.Logger) (bool, error) {
	status, payload, err := pendingTerminalEvent(job)
	if err != nil {
		return true, err
	}

	if !isTerminalStatus(job.Status) {
		err := fmt.Errorf("job %s has pending terminal event %q but non-terminal status %q", job.ID, status, job.Status)
		logger.Log(ctx, libLog.LevelError, "refusing to outbox terminal event without committed terminal state",
			libLog.String("job_id", job.ID.String()),
			libLog.Err(err),
		)

		return true, err
	}

	if terminalStatusForEvent(status) != job.Status {
		err := fmt.Errorf("job %s terminal event %q does not match committed status %q", job.ID, status, job.Status)
		logger.Log(ctx, libLog.LevelError, "refusing to outbox contradictory terminal event",
			libLog.String("job_id", job.ID.String()),
			libLog.Err(err),
		)

		return true, err
	}

	if err := uc.publishJobNotificationPayload(ctx, status, job.ID.String(), []byte(payload), logger); err != nil {
		return true, fmt.Errorf("publish required job %s notification: %w", status, err)
	}

	uc.clearPendingTerminalEvent(ctx, job.ID, logger)

	logger.Log(ctx, libLog.LevelInfo, "retried pending terminal job event",
		libLog.String("job_id", job.ID.String()),
		libLog.String("status", status),
	)

	return true, nil
}

type terminalEventMetadataClearer interface {
	ClearTerminalEventMetadata(ctx context.Context, id uuid.UUID) error
}

func (uc *UseCase) clearPendingTerminalEvent(ctx context.Context, jobID uuid.UUID, logger libLog.Logger) {
	clearer, ok := uc.JobRepository.(terminalEventMetadataClearer)
	if !ok {
		return
	}

	if err := clearer.ClearTerminalEventMetadata(ctx, jobID); err != nil {
		// Error, not Warn: a failed clear leaves the pending-terminal-event marker set,
		// so the terminal-event repairer will re-emit the (already-published) terminal
		// event — a duplicate-event window. That operational hazard must be visible.
		logger.Log(ctx, libLog.LevelError, "failed to clear terminal event retry metadata",
			libLog.String("job_id", jobID.String()),
			libLog.Err(err),
		)
	}
}

func hasPendingTerminalEvent(metadata map[string]any) bool {
	pending, _ := metadata[terminalEventPendingMetadataKey].(bool)
	return pending
}

func pendingTerminalEvent(job *model.Job) (string, string, error) {
	status, ok := job.Metadata[terminalEventStatusMetadataKey].(string)
	if !ok || status == "" {
		return "", "", fmt.Errorf("job %s missing terminal event status metadata", job.ID)
	}

	payload, ok := job.Metadata[terminalEventPayloadMetadataKey].(string)
	if !ok || payload == "" {
		return "", "", fmt.Errorf("job %s missing terminal event payload metadata", job.ID)
	}

	return status, payload, nil
}

func isTerminalStatus(status model.JobStatus) bool {
	return status == model.JobStatusCompleted || status == model.JobStatusFailed
}

func terminalStatusForEvent(status string) model.JobStatus {
	switch status {
	case "completed":
		return model.JobStatusCompleted
	case "failed":
		return model.JobStatusFailed
	default:
		return ""
	}
}

func (uc *UseCase) publishJobNotificationPayload(ctx context.Context, status, jobID string, payload []byte, logger libLog.Logger) error {
	logger = normalizeJobNotificationLogger(ctx, logger)

	if !uc.JobEventStreamingEnabled {
		return fmt.Errorf("mandatory lib-streaming job event emission is disabled")
	}

	logger.Log(ctx, libLog.LevelInfo, "publishing job notification",
		libLog.String("job_id", jobID),
		libLog.String("status", status),
		libLog.String("event_key", fmt.Sprintf("job.%s", status)),
	)

	if err := uc.emitJobNotificationEvent(ctx, status, jobID, payload); err != nil {
		logger.Log(ctx, libLog.LevelError, "error emitting job notification with lib-streaming", libLog.Err(err))
		return fmt.Errorf("emitting job notification event: %w", err)
	}

	return nil
}

// getTableFilters extracts filters for a specific table/collection
func getTableFilters(databaseFilters map[string]map[string]modelJob.FilterCondition, tableName string) map[string]modelJob.FilterCondition {
	if databaseFilters == nil {
		return nil
	}

	return databaseFilters[tableName]
}

// resultPlaintext returns the plaintext bytes to persist plus the total row count.
//
// On the generic-only fast path (reuse non-nil) it reuses the engine's already-
// serialized indented bytes verbatim and the engine's authoritative RowCount, after
// verifying the engine's SHA-256 digest over those bytes — the one place the engine's
// integrity digest, previously discarded, is finally checked. This eliminates the
// decode + re-marshal round-trip the engine path used to pay.
//
// Otherwise (CRM-only or mixed) it serializes the in-memory result map with the EXACT
// json.MarshalIndent(result, "", "  ") shape the stored artifact has always used, and
// counts rows by walking the map. The two-space indent matches the engine's serialized
// form, so the fast path and the fallback produce byte-identical artifacts for the
// same generic-only data.
func resultPlaintext(
	ctx context.Context,
	result map[string]map[string][]map[string]any,
	reuse *directReuse,
	span trace.Span,
	logger libLog.Logger,
) ([]byte, int64, error) {
	if reuse != nil {
		if err := reuse.verifyIntegrity(); err != nil {
			libOtel.HandleSpanError(span, "engine direct result integrity check failed", err)
			logger.Log(ctx, libLog.LevelError, "engine direct result integrity check failed", libLog.Err(err))

			return nil, 0, pkg.FailedPreconditionError{Code: "FET-0068", Title: "Result Integrity Mismatch", Message: err.Error(), Err: err}
		}

		return reuse.plaintext, reuse.rowCount, nil
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		libOtel.HandleSpanError(span, "Error marshalling result to JSON", err)
		logger.Log(ctx, libLog.LevelError, "error marshalling result to json", libLog.Err(err))

		return nil, 0, pkg.FailedPreconditionError{Code: "FET-0060", Title: "Data Serialization Failed", Message: fmt.Sprintf("marshalling result to JSON: %s", err.Error()), Err: err}
	}

	return jsonData, countTotalRows(result), nil
}

// saveExternalData serializes the extraction result, encrypts it, and saves it to
// storage. When reuse is non-nil (a generic-only job) the engine's already-serialized
// indented bytes ARE the plaintext: the worker-side re-marshal and row recount are
// skipped, and the engine's RowCount is authoritative. When reuse is nil (CRM-only or
// mixed) the in-memory result map is serialized via json.MarshalIndent, exactly as
// before — the engine bytes cannot stand in for a map the CRM path also contributed to.
func (uc *UseCase) saveExternalData(
	ctx context.Context,
	tracer trace.Tracer,
	message ExtractExternalDataMessage,
	result map[string]map[string][]map[string]any,
	reuse *directReuse,
	span trace.Span,
	logger libLog.Logger,
) (*JobResultData, error) {
	ctx, spanSave := tracer.Start(ctx, "service.extract_external_data.save_external_data")
	defer spanSave.End()

	jsonData, rowCount, err := resultPlaintext(ctx, result, reuse, span, logger)
	if err != nil {
		return nil, err
	}

	// Calculate metrics before encryption (original data size)
	sizeBytes := int64(len(jsonData))

	// Compute HMAC of plaintext data before encryption for external verification
	var documentHMAC string

	if uc.DocumentSigner == nil {
		logger.Log(ctx, libLog.LevelInfo, "document signing skipped; document signer not configured")
	} else {
		hmac, errHMAC := uc.DocumentSigner.SignReader(bytes.NewReader(jsonData))
		if errHMAC != nil {
			libOtel.HandleSpanError(span, "Error computing document HMAC", errHMAC)
			logger.Log(ctx, libLog.LevelError, "error computing document hmac", libLog.Err(errHMAC))

			return nil, pkg.FailedPreconditionError{Code: "FET-0061", Title: "HMAC Computation Failed", Message: fmt.Sprintf("computing document HMAC: %s", errHMAC.Error()), Err: errHMAC}
		}

		documentHMAC = hmac

		logger.Log(ctx, libLog.LevelInfo, "document hmac computed successfully for job result")
	}

	encryptedData, err := uc.encryptData(jsonData, logger)
	if err != nil {
		libOtel.HandleSpanError(span, "Error encrypting data for storage", err)
		logger.Log(ctx, libLog.LevelError, "error encrypting data for storage", libLog.Err(err))

		return nil, fmt.Errorf("encrypting data for storage: %w", err)
	}

	objectName := fmt.Sprintf("%s/%s.json", constant.ExternalDataKeyPrefix, message.JobID.String())
	if err := uc.ExternalDataStorage.Put(ctx, objectName, encryptedData); err != nil {
		libOtel.HandleSpanError(span, "Error saving external data to storage", err)
		logger.Log(ctx, libLog.LevelError, "error saving external data to storage", libLog.Err(err))

		return nil, fmt.Errorf("saving external data to storage: %w", err)
	}

	// Construct the result path (S3 key) for job status updates and notifications.
	// The tenant-aware key matches what Put() stored, so consumers can download directly.
	tenantObjectName, tenantKeyErr := tms3.GetS3KeyStorageContext(ctx, objectName)
	if tenantKeyErr != nil {
		return nil, pkg.FailedPreconditionError{Code: "FET-0066", Title: "Tenant Path Resolution Failed", Message: fmt.Sprintf("resolving tenant storage path: %s", tenantKeyErr.Error()), Err: tenantKeyErr}
	}

	resultPath := tenantObjectName
	logger.Log(ctx, libLog.LevelInfo, "saved encrypted external data to storage",
		libLog.String("result_path", resultPath),
		libLog.Any("size_bytes", sizeBytes),
		libLog.Any("row_count", rowCount),
	)

	return &JobResultData{
		Path:       resultPath,
		SizeBytes:  sizeBytes,
		RowCount:   rowCount,
		Format:     "json",
		HMAC:       documentHMAC,
		Integrity:  resultIntegrity(documentHMAC),
		Protection: resultProtection(),
	}, nil
}

// resultStorageProtectionMode names the protection mode for the stored extraction
// result. The Worker storage adapter manages encryption (AES-256-GCM via the
// HKDF-derived storage key), so the canonical mode is "adapter-managed" rather than
// a specific cipher string — the Engine never sees or names the cipher.
const resultStorageProtectionMode = "adapter-managed"

// resultIntegrity declares the canonical T-007 integrity over the stored result.
// It turns the tacit HMAC convention into an explicit field: the signature is the
// EXACT documentHMAC the Worker already computed — HMAC-SHA256 over the PLAINTEXT
// extraction JSON (before encryption). When no signer is configured (documentHMAC
// empty), no integrity is declared rather than an empty, misleading one.
//
// It is a keyed signature (Signature), never an unkeyed content hash (Digest):
// HMAC is one possible integrity SIGNATURE in the T-007 model.
func resultIntegrity(documentHMAC string) *engine.ResultIntegrity {
	if documentHMAC == "" {
		return nil
	}

	return &engine.ResultIntegrity{
		Algorithm: "HMAC-SHA256",
		Signature: documentHMAC,
	}
}

// resultProtection declares the canonical T-007 confidentiality over the stored
// result bytes. The bytes are ALWAYS encrypted on the store path (encryptData
// errors otherwise), the Worker storage layer applied it (AppliedBy = adapter), and
// it is adapter-managed. KeyVersion is intentionally omitted: the HKDF-derived
// storage key the Worker uses carries no version label today (see Gate-8 seam), and
// the field is omitempty so an unset version is honestly absent rather than a
// fabricated zero. This describes the STORED RESULT only, never credentials.
func resultProtection() *engine.ResultProtection {
	return &engine.ResultProtection{
		Encrypted: true,
		AppliedBy: engine.ProtectionAppliedByAdapter,
		Mode:      resultStorageProtectionMode,
	}
}

// encryptData encrypts extracted data using AES-GCM with the HKDF-derived storage key.
// Output format: Base64(nonce[12] || ciphertext + auth_tag).
// Compatible with Reporter's decryptFetcherData which uses the same derived key.
func (uc *UseCase) encryptData(data []byte, _ libLog.Logger) ([]byte, error) {
	if len(uc.storageEncryptDerivedKey) == 0 {
		return nil, pkg.FailedPreconditionError{Code: "FET-0056", Title: "Encryption Not Configured", Message: "storage encrypt secret key not configured"}
	}

	block, err := aes.NewCipher(uc.storageEncryptDerivedKey)
	if err != nil {
		return nil, pkg.FailedPreconditionError{Code: "FET-0062", Title: "Cipher Creation Failed", Message: fmt.Sprintf("encrypting data: create AES cipher: %s", err.Error()), Err: err}
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, pkg.FailedPreconditionError{Code: "FET-0063", Title: "Cipher Creation Failed", Message: fmt.Sprintf("encrypting data: create GCM: %s", err.Error()), Err: err}
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("encrypting data: generate nonce: %w", err)
	}

	// Seal prepends nonce to ciphertext: nonce || ciphertext + auth_tag
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Base64-encode for storage
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return []byte(encoded), nil
}

// shouldSkipProcessing checks if job should be skipped due to idempotency.
func (uc *UseCase) shouldSkipProcessing(ctx context.Context, jobID uuid.UUID, logger libLog.Logger) (bool, error) {
	jobData, err := uc.getJobForStatusCheck(ctx, jobID, logger)
	if err != nil || jobData == nil {
		//nolint:nilerr // status preflight failure falls through to the main repository load, which marks the job failed with full context.
		return false, nil
	}

	if hasPendingTerminalEvent(jobData.Metadata) {
		return uc.retryPendingTerminalEventForJob(ctx, jobData, logger)
	}

	if jobData.Status == model.JobStatusCompleted {
		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Job %s is already %s, skipping reprocessing", jobID, jobData.Status))
		return true, nil
	}

	if jobData.Status == model.JobStatusProcessing {
		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Job %s is already processing; skipping extraction replay until terminal event marker is present", jobID))
		return true, nil
	}

	return false, nil
}

func (uc *UseCase) getJobForStatusCheck(ctx context.Context, jobID uuid.UUID, logger libLog.Logger) (*model.Job, error) {
	jobData, err := uc.JobRepository.FindByID(ctx, jobID)
	if err != nil {
		logger.Log(ctx, libLog.LevelDebug, "could not check job status; may be first attempt",
			libLog.String("job_id", jobID.String()),
			libLog.Err(err),
		)

		return nil, fmt.Errorf("failed to check job status: %w", err)
	}

	if jobData == nil {
		logger.Log(ctx, libLog.LevelDebug, "no job data found", libLog.String("job_id", jobID.String()))
		return nil, pkg.ValidationError{Code: "FET-0067", Title: "Job Not Found", Message: fmt.Sprintf("no job data found for %s", jobID)}
	}

	logger.Log(ctx, libLog.LevelDebug, "current job status",
		libLog.String("job_id", jobID.String()),
		libLog.String("status", string(jobData.Status)),
	)

	return jobData, nil
}

// extractConfigNamesFromMappedFields extracts the first-level keys from mappedFields.
func extractConfigNamesFromMappedFields(mappedFields map[string]map[string][]string) []string {
	if len(mappedFields) == 0 {
		return []string{}
	}

	configNames := make([]string, 0, len(mappedFields))
	for configName := range mappedFields {
		configNames = append(configNames, configName)
	}

	return configNames
}

var (
	// notificationURIPattern matches scheme://... connection strings that may
	// carry credentials or internal infrastructure details.
	notificationURIPattern = regexp.MustCompile(`\w+://[^\s]+`)
	// notificationNetAddrPattern matches the address operand of Go net-stack
	// errors that embed an internal endpoint — e.g. "dial tcp 10.0.0.5:5432",
	// "read tcp 10.0.0.5:5432->10.0.0.6:5432", "lookup mongo.internal". The
	// operation keyword is preserved; the host/IP:port operand is redacted.
	notificationNetAddrPattern = regexp.MustCompile(`\b(dial (?:tcp|udp)|read tcp|write tcp|lookup)\s+\S+`)
	// notificationIPPattern catches any remaining bare IPv4 address (optionally
	// with :port) not already covered by the patterns above.
	notificationIPPattern = regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}(?::\d+)?\b`)
	// notificationMongoAddrPattern matches the "Addr: host:port" operand of Mongo
	// driver topology errors (e.g. "Addr: mongo-crm.internal:27017"), which the
	// net-stack/IP patterns above do not cover. The host:port is redacted.
	notificationMongoAddrPattern = regexp.MustCompile(`Addr:\s+\S+`)
)

// sanitizeErrorForNotification strips connection strings, internal endpoints
// (host:port from Go net-stack / DB-driver errors), and bare IP addresses from
// error messages before they are published to notification consumers or
// persisted in terminal-event metadata. It is deliberately anchored on known
// leak shapes (URIs, net-op operands, IPv4) rather than redacting arbitrary
// bare hostnames, to keep operator-facing errors useful while not leaking
// internal topology.
func sanitizeErrorForNotification(msg string) string {
	msg = notificationURIPattern.ReplaceAllString(msg, "[redacted]")
	msg = notificationNetAddrPattern.ReplaceAllString(msg, "$1 [redacted]")
	msg = notificationMongoAddrPattern.ReplaceAllString(msg, "Addr: [redacted]")
	msg = notificationIPPattern.ReplaceAllString(msg, "[redacted]")

	return msg
}

// countTotalRows counts the total number of records in the result map.
func countTotalRows(result map[string]map[string][]map[string]any) int64 {
	var count int64

	for _, tables := range result {
		for _, rows := range tables {
			count += int64(len(rows))
		}
	}

	return count
}
