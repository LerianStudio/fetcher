package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	portDS "github.com/LerianStudio/fetcher/pkg/ports/datasource"
	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libCrypto "github.com/LerianStudio/lib-commons/v2/commons/crypto"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ExtractExternalDataMessage contains the information needed to extract external data
type ExtractExternalDataMessage struct {
	// JobID is the unique identifier of the job extract.
	JobID uuid.UUID `json:"jobId"`

	// OrganizationID is the unique identifier of the organization.
	OrganizationID uuid.UUID `json:"organizationId"`

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

	logger, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	message, err := uc.parseMessage(ctx, body, headers, &span, logger)
	if err != nil {
		jobID, orgID := uc.extractJobIDFromMultipleSources(body, headers, logger)
		if jobID != uuid.Nil {
			notificationMessage := ExtractExternalDataMessage{
				JobID:          jobID,
				OrganizationID: orgID,
				Metadata:       make(map[string]any),
			}

			errorMetadata := map[string]any{
				"message": err.Error(),
			}
			if errNotify := uc.publishJobNotification(ctx, tracer, notificationMessage, "failed", errorMetadata, nil, logger); errNotify != nil {
				logger.Warnf("Failed to publish job failure notification after parse error: %v", errNotify)
			}
		}

		return err
	}

	if skip := uc.shouldSkipProcessing(ctx, message.JobID, message.OrganizationID, logger); skip {
		return nil
	}

	job, errJob := uc.JobRepository.FindByID(ctx, message.JobID, message.OrganizationID)
	if errJob != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message.OrganizationID, *message, &span, "Error finding job by ID in database", errJob, logger)
	}

	// Check if job exists, if not, update job status to failed
	if job == nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message.OrganizationID, *message, &span, "Job not found in database", nil, logger)
	}

	// Extract config names from mappedFields
	configNames := extractConfigNamesFromMappedFields(message.MappedFields)

	// Find connections by config names
	connections, err := uc.ConnectionRepository.FindByConfigNames(ctx, message.OrganizationID, configNames)
	if err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message.OrganizationID, *message, &span, "Error finding connections by config names", err, logger)
	}

	// Check if connections exist, if not, update job status to failed
	if len(connections) == 0 {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message.OrganizationID, *message, &span, "No connections found for config names", nil, logger)
	}

	result := make(map[string]map[string][]map[string]any)
	if err := uc.queryExternalData(ctx, *message, connections, result); err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message.OrganizationID, *message, &span, "Error querying external data", err, logger)
	}

	resultData, err := uc.saveExternalDataToSeaweedFS(ctx, tracer, *message, result, &span, logger)
	if err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message.OrganizationID, *message, &span, "Error saving external data to SeaweedFS", err, logger)
	}

	// Update job status to completed in MongoDB with the resultPath and resultHMAC
	if err := uc.JobRepository.UpdateStatus(ctx, message.JobID, message.OrganizationID, model.JobStatusCompleted, resultData.Path, resultData.HMAC, nil); err != nil {
		libOtel.HandleSpanError(&span, "Error updating job status to completed", err)
		logger.Errorf("Failed to update job status to completed: %v", err)
	}

	// Calculate execution metrics
	completedAt := time.Now()
	executionTimeMs := completedAt.Sub(startTime).Milliseconds()

	// Publish job completion notification with result data and metrics
	notificationOpts := &JobNotificationOptions{
		Result:          resultData,
		ExecutionTimeMs: executionTimeMs,
		CompletedAt:     &completedAt,
	}

	if err := uc.publishJobNotification(ctx, tracer, *message, "completed", nil, notificationOpts, logger); err != nil {
		libOtel.HandleSpanError(&span, "Error publishing job completion notification", err)
		logger.Warnf("Failed to publish job completion notification: %v", err)
	}

	return nil
}

// parseMessage parses the RabbitMQ message body into ExtractExternalDataMessage struct.
// If parsing fails, it attempts to extract jobID from headers or partial JSON to update job status.
func (uc *UseCase) parseMessage(ctx context.Context, body []byte, headers map[string]any, span *trace.Span, logger log.Logger) (*ExtractExternalDataMessage, error) {
	var message *ExtractExternalDataMessage

	err := json.Unmarshal(body, &message)
	if err != nil {
		libOtel.HandleSpanError(span, "Error unmarshalling message.", err)
		logger.Errorf("Error unmarshalling message: %s", err.Error())

		jobID, orgID := uc.extractJobIDFromMultipleSources(body, headers, logger)
		if jobID != uuid.Nil {
			updateErr := uc.updateJobWithErrors(ctx, jobID, orgID, fmt.Sprintf("Failed to parse message: %v", err))
			if updateErr != nil {
				logger.Errorf("Failed to update job status after parse error: %v", updateErr)
			} else {
				logger.Infof("Updated job %s to failed status due to parse error", jobID)
			}
		} else {
			logger.Warnf("Could not extract jobID from headers or partial JSON, job status will not be updated")
		}

		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	return message, nil
}

// extractJobIDFromMultipleSources attempts to extract jobID and organizationID from multiple sources.
func (uc *UseCase) extractJobIDFromMultipleSources(body []byte, headers map[string]any, logger log.Logger) (uuid.UUID, uuid.UUID) {
	if headers != nil {
		if jobIDHeader, exists := headers[constant.HeaderJobID]; exists {
			if jobIDStr, ok := jobIDHeader.(string); ok {
				jobID, err := uuid.Parse(jobIDStr)
				if err == nil {
					logger.Infof("Extracted jobID from header: %s", jobID)

					var orgID uuid.UUID

					if orgIDHeader, existOrg := headers["organizationId"]; existOrg {
						if orgIDStr, ok := orgIDHeader.(string); ok {
							orgID, _ = uuid.Parse(orgIDStr)
						}
					}

					return jobID, orgID
				}
			}
		}
	}

	jobID, orgID := uc.extractJobIDFromPartialJSON(body, logger)
	if jobID != uuid.Nil {
		return jobID, orgID
	}

	return uuid.Nil, uuid.Nil
}

// extractJobIDFromPartialJSON attempts to extract jobID and organizationID from a potentially malformed JSON.
// Uses regex to find UUIDs in the JSON structure, looking for "jobId" and "organizationId" fields.
func (uc *UseCase) extractJobIDFromPartialJSON(body []byte, logger log.Logger) (uuid.UUID, uuid.UUID) {
	bodyStr := string(body)

	var partial struct {
		JobID          *string `json:"jobId"`
		OrganizationID *string `json:"organizationId"`
	}

	decoder := json.NewDecoder(strings.NewReader(bodyStr))
	_ = decoder.Decode(&partial)

	if partial.JobID != nil {
		jobID, err := uuid.Parse(*partial.JobID)
		if err == nil {
			logger.Infof("Extracted jobID from partial JSON: %s", jobID)

			var orgID uuid.UUID
			if partial.OrganizationID != nil {
				orgID, _ = uuid.Parse(*partial.OrganizationID)
			}

			return jobID, orgID
		}
	}

	// Limit whitespace to prevent ReDoS (use {0,10} instead of * to cap backtracking)
	jobIDRegex := regexp.MustCompile(`"jobId"\s{0,10}:\s{0,10}"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)

	matches := jobIDRegex.FindStringSubmatch(bodyStr)
	if len(matches) > 1 {
		jobID, err := uuid.Parse(matches[1])
		if err == nil {
			logger.Infof("Extracted jobID from regex: %s", jobID)

			orgIDRegex := regexp.MustCompile(`"organizationId"\s{0,10}:\s{0,10}"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)
			orgMatches := orgIDRegex.FindStringSubmatch(bodyStr)

			var orgID uuid.UUID
			if len(orgMatches) > 1 {
				orgID, _ = uuid.Parse(orgMatches[1])
			}

			return jobID, orgID
		}
	}

	return uuid.Nil, uuid.Nil
}

// handleErrorWithUpdate logs error, updates report status to error, and publishes failure notification.
func (uc *UseCase) handleErrorWithUpdate(
	ctx context.Context,
	jobID, orgID uuid.UUID,
	message ExtractExternalDataMessage,
	span *trace.Span,
	errorMsg string,
	err error,
	logger log.Logger,
) error {
	if err == nil {
		err = fmt.Errorf("operation failed: %s", errorMsg)
	}

	if errUpdate := uc.updateJobWithErrors(ctx, jobID, orgID, err.Error()); errUpdate != nil {
		libOtel.HandleSpanError(span, "Error to update report status with error.", errUpdate)
		logger.Errorf("Error update report status with error: %s", errUpdate.Error())

		return fmt.Errorf("failed to update job status: %w", errUpdate)
	}

	libOtel.HandleSpanError(span, errorMsg, err)
	logger.Errorf("%s: %s", errorMsg, err.Error())

	// Publish job failure notification to RabbitMQ topic exchange
	// Use the full message to preserve source and other metadata
	errorMetadata := map[string]any{
		"message": err.Error(),
	}

	// Ensure message has correct IDs (in case it was partially parsed)
	message.JobID = jobID
	message.OrganizationID = orgID

	if errNotify := uc.publishJobNotification(ctx, nil, message, "failed", errorMetadata, nil, logger); errNotify != nil {
		logger.Warnf("Failed to publish job failure notification: %v", errNotify)
	}

	return err
}

// updateJobWithErrors updates the status of a job to "Error" with the provided error message.
func (uc *UseCase) updateJobWithErrors(ctx context.Context, jobID, orgID uuid.UUID, errorMessage string) error {
	metadata := make(map[string]any)
	metadata["error"] = errorMessage

	errUpdate := uc.JobRepository.UpdateStatus(ctx, jobID, orgID, model.JobStatusFailed, "", "", metadata)
	if errUpdate != nil {
		return fmt.Errorf("failed to update job status to failed: %w", errUpdate)
	}

	return nil
}

// queryExternalData retrieves data from external data sources specified in the message and populates the result map.
func (uc *UseCase) queryExternalData(ctx context.Context, message ExtractExternalDataMessage, connections []*model.Connection, result map[string]map[string][]map[string]any) error {
	logger, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data.query_external_data")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	for databaseName, tables := range message.MappedFields {
		if err := uc.queryDatabase(ctx, databaseName, tables, connections, message.Filters, message.OrganizationID, result, logger, tracer); err != nil {
			return fmt.Errorf("failed to query database %s: %w", databaseName, err)
		}
	}

	return nil
}

// queryDatabase handles data retrieval for a specific database.
// It finds the connection, creates the appropriate DataSource using the factory pattern,
// and delegates to the specific database query method.
func (uc *UseCase) queryDatabase(
	ctx context.Context,
	databaseName string,
	tables map[string][]string,
	connections []*model.Connection,
	allFilters map[string]map[string]map[string]modelJob.FilterCondition,
	organizationID uuid.UUID,
	result map[string]map[string][]map[string]any,
	logger log.Logger,
	tracer trace.Tracer,
) error {
	ctx, dbSpan := tracer.Start(ctx, "service.extract_external_data.query_external_data.database")
	defer dbSpan.End()

	logger.Infof("Querying database %s", databaseName)

	// Find the connection for this database
	var foundConnection *model.Connection

	for _, conn := range connections {
		if conn.ConfigName == databaseName {
			foundConnection = conn
			break
		}
	}

	if foundConnection == nil {
		err := fmt.Errorf("connection not found for database: %s", databaseName)
		libOtel.HandleSpanBusinessErrorEvent(&dbSpan, "Connection not found", err)

		return err
	}

	// Create DataSource using injected factory
	dataSource, err := uc.CreateDataSource(ctx, foundConnection)
	if err != nil {
		libOtel.HandleSpanError(&dbSpan, "Error creating data source", err)
		return fmt.Errorf("failed to create data source for %s: %w", databaseName, err)
	}

	// Establish connection
	if err := dataSource.Connect(ctx, logger); err != nil {
		libOtel.HandleSpanError(&dbSpan, "Error connecting to data source", err)
		return fmt.Errorf("failed to connect to %s: %w", databaseName, err)
	}

	// Ensure connection is closed after query
	defer func() {
		if closeErr := dataSource.Close(ctx); closeErr != nil {
			logger.Warnf("Error closing connection for %s: %v", databaseName, closeErr)
		}
	}()

	// Prepare a result map for this database
	if _, databaseExists := result[databaseName]; !databaseExists {
		result[databaseName] = make(map[string][]map[string]any)
	}

	databaseFilters := allFilters[databaseName]

	if foundConnection.Type == model.TypeMongoDB && databaseName == "plugin_crm" {
		// MongoDB plugin_crm requires special handling (decryption, collection name transformation).
		// Use interface-based assertion to avoid coupling to concrete MongoDB type.
		crmDS, ok := dataSource.(portDS.CRMQueryable)
		if !ok {
			return fmt.Errorf("data source for plugin_crm does not support CRM queries")
		}

		return uc.QueryPluginCRM(ctx, crmDS, databaseName, tables, databaseFilters, organizationID, result, logger)
	}

	queryResult, errQuery := dataSource.Query(ctx, tables, databaseFilters, logger)
	if errQuery != nil {
		libOtel.HandleSpanError(&dbSpan, "Error querying data source", errQuery)
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

// saveExternalDataToSeaweedFS converts the result map to JSON, encrypts it, and saves it to SeaweedFS storage.
// Returns result data (path, sizeBytes, rowCount, format) for use in notifications and job status updates.
func (uc *UseCase) saveExternalDataToSeaweedFS(
	ctx context.Context,
	tracer trace.Tracer,
	message ExtractExternalDataMessage,
	result map[string]map[string][]map[string]any,
	span *trace.Span,
	logger log.Logger,
) (*JobResultData, error) {
	ctx, spanSave := tracer.Start(ctx, "service.extract_external_data.save_external_data")
	defer spanSave.End()

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		libOtel.HandleSpanError(span, "Error marshalling result to JSON", err)
		logger.Errorf("Error marshalling result to JSON: %s", err.Error())

		return nil, fmt.Errorf("marshalling result to JSON: %w", err)
	}

	// Calculate metrics before encryption (original data size)
	sizeBytes := int64(len(jsonData))
	rowCount := countTotalRows(result)

	// Compute HMAC of plaintext data before encryption for external verification
	var documentHMAC string

	if uc.DocumentSigner == nil {
		// Document signing is disabled (DocumentSigner not configured).
		// This is expected when external HMAC verification is not required.
		// To enable, configure the external HMAC key in worker bootstrap.
		logger.Infof("Document signing skipped: DocumentSigner not configured (HMAC verification disabled)")
	} else {
		hmac, errHMAC := uc.DocumentSigner.SignReader(bytes.NewReader(jsonData))
		if errHMAC != nil {
			libOtel.HandleSpanError(span, "Error computing document HMAC", errHMAC)
			logger.Errorf("Error computing document HMAC: %s", errHMAC.Error())

			return nil, fmt.Errorf("computing document HMAC: %w", errHMAC)
		}

		documentHMAC = hmac

		logger.Infof("Document HMAC computed successfully for job result")
	}

	encryptedData, err := uc.encryptDataForSeaweedFS(jsonData, logger)
	if err != nil {
		libOtel.HandleSpanError(span, "Error encrypting data for SeaweedFS", err)
		logger.Errorf("Error encrypting data for SeaweedFS: %s", err.Error())

		return nil, fmt.Errorf("encrypting data for SeaweedFS: %w", err)
	}

	objectName := fmt.Sprintf("%s.json", message.JobID.String())
	if err := uc.ExternalDataSeaweedFS.Put(ctx, objectName, encryptedData); err != nil {
		libOtel.HandleSpanError(span, "Error saving external data to SeaweedFS", err)
		logger.Errorf("Error saving external data to SeaweedFS: %s", err.Error())

		return nil, fmt.Errorf("saving external data to SeaweedFS: %w", err)
	}

	// Construct the full result path for job status updates
	resultPath := fmt.Sprintf("/%s/%s", constant.ExternalDataBucketName, objectName)
	logger.Infof("Successfully saved encrypted external data to SeaweedFS: %s (size=%d bytes, rows=%d)",
		resultPath, sizeBytes, rowCount)

	return &JobResultData{
		Path:      resultPath,
		SizeBytes: sizeBytes,
		RowCount:  rowCount,
		Format:    "json",
		HMAC:      documentHMAC,
	}, nil
}

// encryptDataForSeaweedFS encrypts the data using the crypto library before saving to SeaweedFS.
func (uc *UseCase) encryptDataForSeaweedFS(data []byte, logger log.Logger) ([]byte, error) {
	if uc.seaweedFSEncryptSecretKey == "" {
		return nil, fmt.Errorf("SeaweedFS encrypt secret key not configured")
	}

	if uc.seaweedFSHashSecretKey == "" {
		return nil, fmt.Errorf("SeaweedFS hash secret key not configured")
	}

	crypto := &libCrypto.Crypto{
		HashSecretKey:    uc.seaweedFSHashSecretKey,
		EncryptSecretKey: uc.seaweedFSEncryptSecretKey,
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		return nil, fmt.Errorf("failed to initialize cipher: %w", err)
	}

	dataStr := string(data)

	encryptedStr, err := crypto.Encrypt(&dataStr)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	return []byte(*encryptedStr), nil
}

// shouldSkipProcessing checks if job should be skipped due to idempotency.
func (uc *UseCase) shouldSkipProcessing(ctx context.Context, jobID, organizationID uuid.UUID, logger log.Logger) bool {
	jobStatus, err := uc.checkReportStatus(ctx, jobID, organizationID, logger)
	if err == nil {
		if jobStatus == model.JobStatusCompleted {
			logger.Infof("Job %s is already completed, skipping reprocessing", jobID)
			return true
		}
	}

	return false
}

// checkReportStatus checks the current status of a report to implement idempotency.
func (uc *UseCase) checkReportStatus(ctx context.Context, jobID, organizationID uuid.UUID, logger log.Logger) (model.JobStatus, error) {
	jobData, err := uc.JobRepository.FindByID(ctx, jobID, organizationID)
	if err != nil {
		logger.Debugf("Could not check job status for %s (may be first attempt): %v", jobID, err)
		return "", fmt.Errorf("failed to check job status: %w", err)
	}

	if jobData == nil {
		logger.Debugf("No job data found for %s", jobID)
		return "", fmt.Errorf("no job data found for %s", jobID)
	}

	logger.Debugf("Report %s current status: %s", jobID, jobData.Status)

	return jobData.Status, nil
}

// extractConfigNamesFromMappedFields extracts the first-level keys from mappedFields.
// Returns an array of strings containing the database/config names.
// Example: {"plugin_crm": {...}, "midaz_onboarding": {...}} -> ["plugin_crm", "midaz_onboarding"]
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

// countTotalRows counts the total number of records in the result map.
// This is performant as it only iterates through the top-level structure.
func countTotalRows(result map[string]map[string][]map[string]any) int64 {
	var count int64

	for _, tables := range result {
		for _, rows := range tables {
			count += int64(len(rows))
		}
	}

	return count
}
