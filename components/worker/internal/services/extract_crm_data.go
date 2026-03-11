// Package services provides business logic for data extraction operations.
package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	datasourceMongoConfig "github.com/LerianStudio/fetcher/pkg/model/datasource/mongodb"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libCrypto "github.com/LerianStudio/lib-commons/v4/commons/crypto"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// QueryPluginCRM handles querying MongoDB plugin_crm database with special processing.
func (uc *UseCase) QueryPluginCRM(
	ctx context.Context,
	dataSource *datasourceMongoConfig.DataSourceConfigMongoDB,
	databaseName string,
	collections map[string][]string,
	databaseFilters map[string]map[string]modelJob.FilterCondition,
	organizationID uuid.UUID,
	result map[string]map[string][]map[string]any,
	logger libLog.Logger,
) error {
	_, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "service.extract_external_data.query_plugin_crm")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.database_name", databaseName),
		attribute.String("app.request.organization_id", organizationID.String()),
	)

	for collection, fields := range collections {
		collectionFilters := getTableFilters(databaseFilters, collection)

		if err := uc.processPluginCRMCollection(ctx, dataSource, collection, fields, collectionFilters, organizationID, result, logger); err != nil {
			libOtel.HandleSpanError(span, "Error processing plugin_crm collection", err)
			return err
		}
	}

	return nil
}

// processPluginCRMCollection handles plugin_crm specific collection processing.
func (uc *UseCase) processPluginCRMCollection(
	ctx context.Context,
	dataSource *datasourceMongoConfig.DataSourceConfigMongoDB,
	collection string,
	fields []string,
	collectionFilters map[string]modelJob.FilterCondition,
	organizationID uuid.UUID,
	result map[string]map[string][]map[string]any,
	logger libLog.Logger,
) error {
	// Transform collection name by appending organization ID suffix
	// e.g., "holders" becomes "holders_019b9df1-34eb-7dd0-afd5-53f859667e51"
	newCollection := collection + "_" + organizationID.String()
	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Transformed plugin_crm collection: %s -> %s", collection, newCollection))

	collectionResult, err := uc.queryPluginCRMCollectionWithFilters(ctx, dataSource, newCollection, fields, collectionFilters, logger)
	if err != nil {
		return err
	}

	if result["plugin_crm"] == nil {
		result["plugin_crm"] = make(map[string][]map[string]any)
	}

	result["plugin_crm"][collection] = collectionResult

	decryptedResult, err := uc.decryptPluginCRMData(logger, result["plugin_crm"][collection], fields)
	if err != nil {
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error decrypting data for collection %s: %s", collection, err.Error()))
		return fmt.Errorf("error decrypting data for collection %s: %w", collection, err)
	}

	result["plugin_crm"][collection] = decryptedResult

	return nil
}

// queryPluginCRMCollectionWithFilters queries a MongoDB collection with plugin_crm specific filter transformation.
func (uc *UseCase) queryPluginCRMCollectionWithFilters(
	ctx context.Context,
	dataSource *datasourceMongoConfig.DataSourceConfigMongoDB,
	collection string,
	fields []string,
	collectionFilters map[string]modelJob.FilterCondition,
	logger libLog.Logger,
) ([]map[string]any, error) {
	var (
		queryResult    []map[string]any
		errQueryResult error
	)

	if len(collectionFilters) > 0 {
		transformedFilter, err := uc.transformPluginCRMAdvancedFilters(collectionFilters, logger)
		if err != nil {
			return nil, fmt.Errorf("error transforming advanced filters for collection %s: %w", collection, err)
		}

		queryResult, errQueryResult = dataSource.MongoDBRepository.QueryWithAdvancedFilters(ctx, collection, fields, transformedFilter)
	} else {
		queryResult, errQueryResult = dataSource.MongoDBRepository.Query(ctx, collection, fields, nil)
	}

	if errQueryResult != nil {
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error querying collection %s: %s", collection, errQueryResult.Error()))
		return nil, errQueryResult
	}

	return queryResult, nil
}

// transformPluginCRMAdvancedFilters transforms advanced FilterCondition filters for plugin_crm to use search fields.
func (uc *UseCase) transformPluginCRMAdvancedFilters(filter map[string]modelJob.FilterCondition, logger libLog.Logger) (map[string]modelJob.FilterCondition, error) {
	if filter == nil {
		return nil, nil
	}

	hashSecretKey := os.Getenv("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM")
	if hashSecretKey == "" {
		return nil, fmt.Errorf("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM environment variable not set")
	}

	crypto := &libCrypto.Crypto{
		HashSecretKey: hashSecretKey,
		Logger:        logger,
	}

	transformedFilter := make(map[string]modelJob.FilterCondition)

	// Define field mappings: encrypted field -> search field
	fieldMappings := map[string]string{
		"document":                "search.document",
		"name":                    "search.name",
		"banking_details.account": "search.banking_details_account",
		"banking_details.iban":    "search.banking_details_iban",
		"contact.primary_email":   "search.contact_primary_email",
		"contact.secondary_email": "search.contact_secondary_email",
		"contact.mobile_phone":    "search.contact_mobile_phone",
		"contact.other_phone":     "search.contact_other_phone",
	}

	for fieldName, condition := range filter {
		if searchField, exists := fieldMappings[fieldName]; exists {
			// Transform the condition by hashing string values
			transformedCondition := modelJob.FilterCondition{}

			if len(condition.Equals) > 0 {
				transformedCondition.Equals = uc.hashFilterValues(condition.Equals, crypto)
			}

			if len(condition.GreaterThan) > 0 {
				transformedCondition.GreaterThan = uc.hashFilterValues(condition.GreaterThan, crypto)
			}

			if len(condition.GreaterOrEqual) > 0 {
				transformedCondition.GreaterOrEqual = uc.hashFilterValues(condition.GreaterOrEqual, crypto)
			}

			if len(condition.LessThan) > 0 {
				transformedCondition.LessThan = uc.hashFilterValues(condition.LessThan, crypto)
			}

			if len(condition.LessOrEqual) > 0 {
				transformedCondition.LessOrEqual = uc.hashFilterValues(condition.LessOrEqual, crypto)
			}

			if len(condition.Between) > 0 {
				transformedCondition.Between = uc.hashFilterValues(condition.Between, crypto)
			}

			// Transform In values
			if len(condition.In) > 0 {
				transformedCondition.In = uc.hashFilterValues(condition.In, crypto)
			}

			if len(condition.NotIn) > 0 {
				transformedCondition.NotIn = uc.hashFilterValues(condition.NotIn, crypto)
			}

			transformedFilter[searchField] = transformedCondition

			logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Transformed advanced filter: %s -> %s", fieldName, searchField))
		} else {
			transformedFilter[fieldName] = condition
		}
	}

	return transformedFilter, nil
}

// hashFilterValues hashes string values in a filter condition array.
func (uc *UseCase) hashFilterValues(values []any, crypto *libCrypto.Crypto) []any {
	hashedValues := make([]any, len(values))

	for i, value := range values {
		if strValue, ok := value.(string); ok && strValue != "" {
			hash := crypto.GenerateHash(&strValue)
			hashedValues[i] = hash
		} else {
			hashedValues[i] = value // Keep non-string values as-is
		}
	}

	return hashedValues
}

// decryptPluginCRMData decrypts sensitive fields for plugin_crm database.
func (uc *UseCase) decryptPluginCRMData(logger libLog.Logger, collectionResult []map[string]any, fields []string) ([]map[string]any, error) {
	needsDecryption := false

	for _, field := range fields {
		if isEncryptedField(field) {
			needsDecryption = true
			break
		}

		if strings.Contains(field, ".") {
			needsDecryption = true
			break
		}
	}

	if !needsDecryption {
		return collectionResult, nil
	}

	// Initialize crypto instance
	hashSecretKey := os.Getenv("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM")

	encryptSecretKey := os.Getenv("CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM")
	if encryptSecretKey == "" {
		return nil, fmt.Errorf("CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM environment variable not set")
	}

	if hashSecretKey == "" {
		return nil, fmt.Errorf("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM environment variable not set")
	}

	crypto := &libCrypto.Crypto{
		HashSecretKey:    hashSecretKey,
		EncryptSecretKey: encryptSecretKey,
		Logger:           logger,
	}

	err := crypto.InitializeCipher()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cipher: %w", err)
	}

	// Process each record in the collection
	for i, record := range collectionResult {
		decryptedRecord, err := uc.decryptRecord(record, crypto)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt record %d: %w", i, err)
		}

		collectionResult[i] = decryptedRecord
	}

	return collectionResult, nil
}

// isEncryptedField checks if a field is known to be encrypted in plugin_crm.
func isEncryptedField(field string) bool {
	encryptedFields := map[string]bool{
		"document": true,
		"name":     true,
	}

	return encryptedFields[field]
}

// decryptRecord decrypts a single record's encrypted fields.
func (uc *UseCase) decryptRecord(record map[string]any, crypto *libCrypto.Crypto) (map[string]any, error) {
	decryptedRecord := make(map[string]any)
	for k, v := range record {
		decryptedRecord[k] = v
	}

	if err := uc.decryptTopLevelFields(decryptedRecord, crypto); err != nil {
		return nil, err
	}

	if err := uc.decryptNestedFields(decryptedRecord, crypto); err != nil {
		return nil, err
	}

	return decryptedRecord, nil
}

// decryptTopLevelFields decrypts top-level encrypted fields.
func (uc *UseCase) decryptTopLevelFields(record map[string]any, crypto *libCrypto.Crypto) error {
	for fieldName, fieldValue := range record {
		if isEncryptedField(fieldName) && fieldValue != nil {
			if err := uc.decryptFieldValue(record, fieldName, fieldValue, crypto); err != nil {
				return fmt.Errorf("failed to decrypt field %s: %w", fieldName, err)
			}
		}
	}

	return nil
}

// decryptNestedFields decrypts nested encrypted fields in the record.
func (uc *UseCase) decryptNestedFields(record map[string]any, crypto *libCrypto.Crypto) error {
	if err := uc.decryptContactFields(record, crypto); err != nil {
		return err
	}

	if err := uc.decryptBankingDetailsFields(record, crypto); err != nil {
		return err
	}

	if err := uc.decryptLegalPersonFields(record, crypto); err != nil {
		return err
	}

	if err := uc.decryptNaturalPersonFields(record, crypto); err != nil {
		return err
	}

	return nil
}

// decryptContactFields decrypts fields within the contact object.
func (uc *UseCase) decryptContactFields(record map[string]any, crypto *libCrypto.Crypto) error {
	contact, ok := record["contact"].(map[string]any)
	if !ok {
		return nil
	}

	contactFields := []string{"primary_email", "secondary_email", "mobile_phone", "other_phone"}
	for _, fieldName := range contactFields {
		if fieldValue, exists := contact[fieldName]; exists && fieldValue != nil {
			if err := uc.decryptFieldValue(contact, fieldName, fieldValue, crypto); err != nil {
				return fmt.Errorf("failed to decrypt contact.%s: %w", fieldName, err)
			}
		}
	}

	record["contact"] = contact

	return nil
}

// decryptBankingDetailsFields decrypts fields within the banking_details object.
func (uc *UseCase) decryptBankingDetailsFields(record map[string]any, crypto *libCrypto.Crypto) error {
	bankingDetails, ok := record["banking_details"].(map[string]any)
	if !ok {
		return nil
	}

	bankingFields := []string{"account", "iban"}
	for _, fieldName := range bankingFields {
		if fieldValue, exists := bankingDetails[fieldName]; exists && fieldValue != nil {
			if err := uc.decryptFieldValue(bankingDetails, fieldName, fieldValue, crypto); err != nil {
				return fmt.Errorf("failed to decrypt banking_details.%s: %w", fieldName, err)
			}
		}
	}

	record["banking_details"] = bankingDetails

	return nil
}

// decryptLegalPersonFields decrypts fields within the legal_person object.
func (uc *UseCase) decryptLegalPersonFields(record map[string]any, crypto *libCrypto.Crypto) error {
	legalPerson, ok := record["legal_person"].(map[string]any)
	if !ok {
		return nil
	}

	representative, ok := legalPerson["representative"].(map[string]any)
	if !ok {
		return nil
	}

	representativeFields := []string{"name", "document", "email"}
	for _, fieldName := range representativeFields {
		if fieldValue, exists := representative[fieldName]; exists && fieldValue != nil {
			if err := uc.decryptFieldValue(representative, fieldName, fieldValue, crypto); err != nil {
				return fmt.Errorf("failed to decrypt legal_person.representative.%s: %w", fieldName, err)
			}
		}
	}

	legalPerson["representative"] = representative
	record["legal_person"] = legalPerson

	return nil
}

// decryptNaturalPersonFields decrypts fields within the natural_person object.
func (uc *UseCase) decryptNaturalPersonFields(record map[string]any, crypto *libCrypto.Crypto) error {
	naturalPerson, ok := record["natural_person"].(map[string]any)
	if !ok {
		return nil
	}

	naturalPersonFields := []string{"mother_name", "father_name"}
	for _, fieldName := range naturalPersonFields {
		if fieldValue, exists := naturalPerson[fieldName]; exists && fieldValue != nil {
			if err := uc.decryptFieldValue(naturalPerson, fieldName, fieldValue, crypto); err != nil {
				return fmt.Errorf("failed to decrypt natural_person.%s: %w", fieldName, err)
			}
		}
	}

	record["natural_person"] = naturalPerson

	return nil
}

// decryptFieldValue decrypts a single field value if it's a non-empty string.
func (uc *UseCase) decryptFieldValue(container map[string]any, fieldName string, fieldValue any, crypto *libCrypto.Crypto) error {
	strValue, ok := fieldValue.(string)
	if !ok || strValue == "" {
		return nil
	}

	decryptedValue, err := crypto.Decrypt(&strValue)
	if err != nil {
		return err
	}

	container[fieldName] = *decryptedValue

	return nil
}
