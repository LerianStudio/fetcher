package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	datasourceMongoConfig "github.com/LerianStudio/fetcher/pkg/model/datasource/mongodb"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libCrypto "github.com/LerianStudio/lib-commons/v2/commons/crypto"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
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
	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data")
	defer span.End()

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
			if errNotify := uc.publishJobNotification(ctx, tracer, notificationMessage, "failed", errorMetadata, logger); errNotify != nil {
				logger.Warnf("Failed to publish job failure notification after parse error: %v", errNotify)
			}
		}
		return err
	}

	if skip := uc.shouldSkipProcessing(ctx, message.JobID, logger); skip {
		return nil
	}

	_, errJob := uc.JobRepository.FindByID(ctx, message.JobID, message.OrganizationID)
	if errJob != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message.OrganizationID, *message, &span, "Error finding job by ID", errJob, logger)
	}

	// Extract config names from mappedFields
	configNames := extractConfigNamesFromMappedFields(message.MappedFields)

	// Find connections by config names
	connections, err := uc.ConnectionRepository.FindByConfigNames(ctx, message.OrganizationID, configNames)
	if err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message.OrganizationID, *message, &span, "Error finding connections by config names", err, logger)
	}

	result := make(map[string]map[string][]map[string]any)
	if err := uc.queryExternalData(ctx, *message, connections, result); err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message.OrganizationID, *message, &span, "Error querying external data", err, logger)
	}

	if err := uc.saveExternalDataToSeaweedFS(ctx, tracer, *message, result, &span, logger); err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, message.OrganizationID, *message, &span, "Error saving external data to SeaweedFS", err, logger)
	}

	// Publish job completion notification to RabbitMQ topic exchange
	if err := uc.publishJobNotification(ctx, tracer, *message, "completed", nil, logger); err != nil {
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
	decoder.DisallowUnknownFields()
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

	jobIDRegex := regexp.MustCompile(`"jobId"\s*:\s*"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)

	matches := jobIDRegex.FindStringSubmatch(bodyStr)
	if len(matches) > 1 {
		jobID, err := uuid.Parse(matches[1])
		if err == nil {
			logger.Infof("Extracted jobID from regex: %s", jobID)

			orgIDRegex := regexp.MustCompile(`"organizationId"\s*:\s*"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)
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
	if errUpdate := uc.updateJobWithErrors(ctx, jobID, orgID, err.Error()); errUpdate != nil {
		libOtel.HandleSpanError(span, "Error to update report status with error.", errUpdate)
		logger.Errorf("Error update report status with error: %s", errUpdate.Error())

		return errUpdate
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

	if errNotify := uc.publishJobNotification(ctx, nil, message, "failed", errorMetadata, logger); errNotify != nil {
		logger.Warnf("Failed to publish job failure notification: %v", errNotify)
	}

	return err
}

// updateJobWithErrors updates the status of a job to "Error" with the provided error message.
func (uc *UseCase) updateJobWithErrors(ctx context.Context, jobID, orgID uuid.UUID, errorMessage string) error {
	metadata := make(map[string]any)
	metadata["error"] = errorMessage

	errUpdate := uc.JobRepository.UpdateStatus(ctx, jobID, orgID, job.JobStatusFailed, metadata)
	if errUpdate != nil {
		return errUpdate
	}

	return nil
}

// queryExternalData retrieves data from external data sources specified in the message and populates the result map.
func (uc *UseCase) queryExternalData(ctx context.Context, message ExtractExternalDataMessage, connections []*model.Connection, result map[string]map[string][]map[string]any) error {
	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data.query_external_data")
	defer span.End()

	for databaseName, tables := range message.MappedFields {
		if err := uc.queryDatabase(ctx, databaseName, tables, connections, message.Filters, result, logger, tracer); err != nil {
			return err
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
		libOtel.HandleSpanError(&dbSpan, "Connection not found", err)

		return err
	}

	// Create DataSource using factory pattern
	dataSource, err := datasource.NewDataSourceFromConnection(ctx, foundConnection, uc.Cryptor, logger)
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
		// MongoDB plugin_crm requires special handling (decryption, collection name transformation)
		mongoDS, ok := dataSource.(*datasourceMongoConfig.DataSourceConfigMongoDB)
		if !ok {
			return fmt.Errorf("invalid MongoDB data source type")
		}

		return uc.QueryPluginCRM(ctx, mongoDS, databaseName, tables, databaseFilters, result, logger)
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
func (uc *UseCase) saveExternalDataToSeaweedFS(
	ctx context.Context,
	tracer trace.Tracer,
	message ExtractExternalDataMessage,
	result map[string]map[string][]map[string]any,
	span *trace.Span,
	logger log.Logger,
) error {
	ctx, spanSave := tracer.Start(ctx, "service.extract_external_data.save_external_data")
	defer spanSave.End()

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		libOtel.HandleSpanError(span, "Error marshalling result to JSON", err)
		logger.Errorf("Error marshalling result to JSON: %s", err.Error())

		return fmt.Errorf("marshalling result to JSON: %w", err)
	}

	encryptedData, err := uc.encryptDataForSeaweedFS(jsonData, logger)
	if err != nil {
		libOtel.HandleSpanError(span, "Error encrypting data for SeaweedFS", err)
		logger.Errorf("Error encrypting data for SeaweedFS: %s", err.Error())

		return fmt.Errorf("encrypting data for SeaweedFS: %w", err)
	}

	objectName := fmt.Sprintf("%s.json", message.JobID.String())
	contentType := "application/json"

	if err := uc.ExternalDataSeaweedFS.Put(ctx, objectName, contentType, encryptedData); err != nil {
		libOtel.HandleSpanError(span, "Error saving external data to SeaweedFS", err)
		logger.Errorf("Error saving external data to SeaweedFS: %s", err.Error())

		return fmt.Errorf("saving external data to SeaweedFS: %w", err)
	}

	logger.Infof("Successfully saved encrypted external data to SeaweedFS: %s", objectName)

	return nil
}

// encryptDataForSeaweedFS encrypts the data using the crypto library before saving to SeaweedFS.
func (uc *UseCase) encryptDataForSeaweedFS(data []byte, logger log.Logger) ([]byte, error) {
	encryptSecretKey := os.Getenv("CRYPTO_ENCRYPT_SECRET_KEY_SEAWEEDFS")
	if encryptSecretKey == "" {
		return nil, fmt.Errorf("CRYPTO_ENCRYPT_SECRET_KEY_SEAWEEDFS environment variable not set")
	}

	hashSecretKey := os.Getenv("CRYPTO_HASH_SECRET_KEY_SEAWEEDFS")
	if hashSecretKey == "" {
		return nil, fmt.Errorf("CRYPTO_HASH_SECRET_KEY_SEAWEEDFS environment variable not set")
	}

	crypto := &libCrypto.Crypto{
		HashSecretKey:    hashSecretKey,
		EncryptSecretKey: encryptSecretKey,
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
func (uc *UseCase) shouldSkipProcessing(ctx context.Context, jobID uuid.UUID, logger log.Logger) bool {
	jobStatus, err := uc.checkReportStatus(ctx, jobID, logger)
	if err == nil {
		if jobStatus == job.JobStatusCompleted {
			logger.Infof("Job %s is already completed, skipping reprocessing", jobID)
			return true
		}
	}

	return false
}

// checkReportStatus checks the current status of a report to implement idempotency.
func (uc *UseCase) checkReportStatus(ctx context.Context, jobID uuid.UUID, logger log.Logger) (job.JobStatus, error) {
	zeroUUID := uuid.UUID{}

	jobData, err := uc.JobRepository.FindByID(ctx, jobID, zeroUUID)
	if err != nil {
		logger.Debugf("Could not check job status for %s (may be first attempt): %v", jobID, err)
		return "", err
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
