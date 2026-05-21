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

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	portDS "github.com/LerianStudio/fetcher/pkg/ports/datasource"
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
			notificationMessage := ExtractExternalDataMessage{
				JobID:    jobID,
				Metadata: make(map[string]any),
			}

			errorMetadata := map[string]any{
				"message": err.Error(),
			}
			if errNotify := uc.publishJobNotification(ctx, tracer, notificationMessage, "failed", errorMetadata, nil, logger); errNotify != nil {
				logger.Log(ctx, libLog.LevelWarn, "failed to publish job failure notification after parse error", libLog.Err(errNotify))
			}
		}

		return err
	}

	if skip := uc.shouldSkipProcessing(ctx, message.JobID, logger); skip {
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
	if err := uc.queryExternalData(ctx, *message, connections, result); err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, *message, span, "Error querying external data", err, logger)
	}

	resultData, err := uc.saveExternalData(ctx, tracer, *message, result, span, logger)
	if err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, *message, span, "Error saving external data to storage", err, logger)
	}

	return uc.completeJob(ctx, tracer, *message, resultData, startTime, span, logger)
}

// completeJob persists the completed status and publishes a completion notification.
func (uc *UseCase) completeJob(
	ctx context.Context,
	tracer trace.Tracer,
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

	if err := uc.publishJobNotification(ctx, tracer, message, "completed", nil, notificationOpts, logger); err != nil {
		libOtel.HandleSpanError(span, "Error publishing job completion notification", err)
		logger.Log(ctx, libLog.LevelError, "failed to publish required job completion notification",
			libLog.String("job_id", message.JobID.String()),
			libLog.Err(err),
		)

		return fmt.Errorf("publish required job completion notification: %w", err)
	}

	if err := uc.JobRepository.UpdateStatus(ctx, message.JobID, model.JobStatusCompleted, resultData.Path, resultData.HMAC, nil); err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message, span, "Error updating job status to completed", err, logger)
	}

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

		jobID := uc.extractJobIDFromMultipleSources(body, headers, logger)
		if jobID != uuid.Nil {
			updateErr := uc.updateJobWithErrors(ctx, jobID, fmt.Sprintf("Failed to parse message: %v", err))
			if updateErr != nil {
				logger.Log(ctx, libLog.LevelError, "failed to update job status after parse error",
					libLog.String("job_id", jobID.String()),
					libLog.Err(updateErr),
				)
			} else {
				logger.Log(ctx, libLog.LevelInfo, "updated job to failed status due to parse error",
					libLog.String("job_id", jobID.String()),
				)
			}
		} else {
			logger.Log(ctx, libLog.LevelWarn, "could not extract job id from headers or partial json; job status will not be updated")
		}

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

		if jobID != uuid.Nil {
			updateErr := uc.updateJobWithErrors(ctx, jobID, wrappedErr.Error())
			if updateErr != nil {
				logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to update job status after payload validation error: %v", updateErr))
			}
		} else {
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

	if errUpdate := uc.updateJobWithErrors(ctx, jobID, err.Error()); errUpdate != nil {
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

	// Publish job failure notification to RabbitMQ topic exchange.
	// Sanitize the error message to avoid leaking internal details.
	errorMetadata := map[string]any{
		"message": sanitizeErrorForNotification(err.Error()),
	}

	// Ensure message has correct IDs (in case it was partially parsed)
	message.JobID = jobID

	if errNotify := uc.publishJobNotification(ctx, nil, message, "failed", errorMetadata, nil, logger); errNotify != nil {
		logger.Log(ctx, libLog.LevelWarn, "failed to publish job failure notification",
			libLog.String("job_id", jobID.String()),
			libLog.Err(errNotify),
		)
	}

	return err
}

// updateJobWithErrors updates the status of a job to "Error" with the provided error message.
func (uc *UseCase) updateJobWithErrors(ctx context.Context, jobID uuid.UUID, errorMessage string) error {
	metadata := map[string]any{
		"error": errorMessage,
	}

	errUpdate := uc.JobRepository.UpdateStatus(ctx, jobID, model.JobStatusFailed, "", "", metadata)
	if errUpdate != nil {
		return fmt.Errorf("failed to update job status to failed: %w", errUpdate)
	}

	return nil
}

// queryExternalData retrieves data from external data sources specified in the message and populates the result map.
func (uc *UseCase) queryExternalData(ctx context.Context, message ExtractExternalDataMessage, connections []*model.Connection, result map[string]map[string][]map[string]any) error {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data.query_external_data")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	for databaseName, tables := range message.MappedFields {
		if err := uc.queryDatabase(ctx, databaseName, tables, connections, message.Filters, result, logger, tracer); err != nil {
			return fmt.Errorf("failed to query database %s: %w", databaseName, err)
		}
	}

	return nil
}

// queryDatabase handles data retrieval for a specific database.
func (uc *UseCase) queryDatabase(
	ctx context.Context,
	databaseName string,
	tables map[string][]string,
	connections []*model.Connection,
	allFilters map[string]map[string]map[string]modelJob.FilterCondition,
	result map[string]map[string][]map[string]any,
	logger libLog.Logger,
	tracer trace.Tracer,
) error {
	ctx, dbSpan := tracer.Start(ctx, "service.extract_external_data.query_external_data.database")
	defer dbSpan.End()

	logger.Log(ctx, libLog.LevelInfo, "querying database", libLog.String("database_name", databaseName))

	var foundConnection *model.Connection

	for _, conn := range connections {
		if conn != nil && conn.ConfigName == databaseName {
			foundConnection = conn
			break
		}
	}

	if foundConnection == nil {
		err := pkg.ValidationError{Code: "FET-0054", Title: "Connection Not Found", Message: fmt.Sprintf("connection not found for database: %s", databaseName)}
		libOtel.HandleSpanBusinessErrorEvent(dbSpan, "Connection not found", err)

		return err
	}

	// Create DataSource using injected factory
	dataSource, err := uc.CreateDataSource(ctx, foundConnection)
	if err != nil {
		libOtel.HandleSpanError(dbSpan, "Error creating data source", err)
		return fmt.Errorf("failed to create data source for %s: %w", databaseName, err)
	}

	// Establish connection
	if err := dataSource.Connect(ctx, logger); err != nil {
		libOtel.HandleSpanError(dbSpan, "Error connecting to data source", err)
		return fmt.Errorf("failed to connect to %s: %w", databaseName, err)
	}

	// Ensure connection is closed after query
	defer func() {
		if closeErr := dataSource.Close(ctx); closeErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "error closing connection",
				libLog.String("database_name", databaseName),
				libLog.Err(closeErr),
			)
		}
	}()

	// Prepare a result map for this database
	if _, databaseExists := result[databaseName]; !databaseExists {
		result[databaseName] = make(map[string][]map[string]any)
	}

	databaseFilters := allFilters[databaseName]

	if foundConnection.Type == model.TypeMongoDB && databaseName == "plugin_crm" {
		crmDS, ok := dataSource.(portDS.CRMQueryable)
		if !ok {
			return pkg.ValidationError{Code: "FET-0055", Title: "Unsupported Operation", Message: "data source for plugin_crm does not support CRM queries"}
		}

		return uc.QueryPluginCRM(ctx, crmDS, databaseName, tables, databaseFilters, result, logger)
	}

	queryResult, errQuery := dataSource.Query(ctx, tables, databaseFilters, logger)
	if errQuery != nil {
		libOtel.HandleSpanError(dbSpan, "Error querying data source", errQuery)
		return fmt.Errorf("failed to query %s: %w", databaseName, errQuery)
	}

	// Merge query results into the result map
	for tableOrCollection, tableResult := range queryResult {
		result[databaseName][tableOrCollection] = tableResult
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

// saveExternalData converts the result map to JSON, encrypts it, and saves it to storage.
func (uc *UseCase) saveExternalData(
	ctx context.Context,
	tracer trace.Tracer,
	message ExtractExternalDataMessage,
	result map[string]map[string][]map[string]any,
	span trace.Span,
	logger libLog.Logger,
) (*JobResultData, error) {
	ctx, spanSave := tracer.Start(ctx, "service.extract_external_data.save_external_data")
	defer spanSave.End()

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		libOtel.HandleSpanError(span, "Error marshalling result to JSON", err)
		logger.Log(ctx, libLog.LevelError, "error marshalling result to json", libLog.Err(err))

		return nil, pkg.FailedPreconditionError{Code: "FET-0060", Title: "Data Serialization Failed", Message: fmt.Sprintf("marshalling result to JSON: %s", err.Error()), Err: err}
	}

	// Calculate metrics before encryption (original data size)
	sizeBytes := int64(len(jsonData))
	rowCount := countTotalRows(result)

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
		Path:      resultPath,
		SizeBytes: sizeBytes,
		RowCount:  rowCount,
		Format:    "json",
		HMAC:      documentHMAC,
	}, nil
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
func (uc *UseCase) shouldSkipProcessing(ctx context.Context, jobID uuid.UUID, logger libLog.Logger) bool {
	jobStatus, err := uc.checkReportStatus(ctx, jobID, logger)
	if err == nil {
		if jobStatus == model.JobStatusCompleted {
			logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Job %s is already %s, skipping reprocessing", jobID, jobStatus))
			return true
		}

		if jobStatus == model.JobStatusProcessing {
			logger.Log(ctx, libLog.LevelWarn, fmt.Sprintf("Job %s is processing; reprocessing is allowed so required terminal events can be retried", jobID))
		}
	}

	return false
}

// checkReportStatus checks the current status of a report to implement idempotency.
func (uc *UseCase) checkReportStatus(ctx context.Context, jobID uuid.UUID, logger libLog.Logger) (model.JobStatus, error) {
	jobData, err := uc.JobRepository.FindByID(ctx, jobID)
	if err != nil {
		logger.Log(ctx, libLog.LevelDebug, "could not check job status; may be first attempt",
			libLog.String("job_id", jobID.String()),
			libLog.Err(err),
		)

		return "", fmt.Errorf("failed to check job status: %w", err)
	}

	if jobData == nil {
		logger.Log(ctx, libLog.LevelDebug, "no job data found", libLog.String("job_id", jobID.String()))
		return "", pkg.ValidationError{Code: "FET-0067", Title: "Job Not Found", Message: fmt.Sprintf("no job data found for %s", jobID)}
	}

	logger.Log(ctx, libLog.LevelDebug, "current job status",
		libLog.String("job_id", jobID.String()),
		libLog.String("status", string(jobData.Status)),
	)

	return jobData.Status, nil
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

// notificationURIPattern matches connection strings and URIs that may contain
// credentials or internal infrastructure details.
var notificationURIPattern = regexp.MustCompile(`\w+://[^\s]+`)

// sanitizeErrorForNotification strips connection strings, hostnames, and other
// internal infrastructure details from error messages.
func sanitizeErrorForNotification(msg string) string {
	return notificationURIPattern.ReplaceAllString(msg, "[redacted]")
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
