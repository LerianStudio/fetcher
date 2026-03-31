// Package services provides business logic for data extraction operations.
package services

import (
	"context"
	"fmt"
	"strings"

	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	portDS "github.com/LerianStudio/fetcher/pkg/ports/datasource"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libCrypto "github.com/LerianStudio/lib-commons/v4/commons/crypto"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"go.opentelemetry.io/otel/attribute"
)

var (
	processPluginCRMCollectionFn = func(
		uc *UseCase,
		ctx context.Context,
		dataSource portDS.CRMQueryable,
		collection string,
		fields []string,
		collectionFilters map[string]modelJob.FilterCondition,
		matchingCollections []string,
		result map[string]map[string][]map[string]any,
		logger libLog.Logger,
	) error {
		return uc.processPluginCRMCollection(ctx, dataSource, collection, fields, collectionFilters, matchingCollections, result, logger)
	}
	queryPluginCRMCollectionWithFiltersFn = func(
		uc *UseCase,
		ctx context.Context,
		dataSource portDS.CRMQueryable,
		collection string,
		fields []string,
		collectionFilters map[string]modelJob.FilterCondition,
		logger libLog.Logger,
	) ([]map[string]any, error) {
		return uc.queryPluginCRMCollectionWithFilters(ctx, dataSource, collection, fields, collectionFilters, logger)
	}
	decryptPluginCRMDataFn = func(uc *UseCase, logger libLog.Logger, collectionResult []map[string]any, fields []string) ([]map[string]any, error) {
		return uc.decryptPluginCRMData(logger, collectionResult, fields)
	}
)

// QueryPluginCRM handles querying MongoDB plugin_crm database with special processing.
// It lists all available collections once and dispatches matching collections to each
// logical collection processor, avoiding repeated ListCollectionNames calls.
func (uc *UseCase) QueryPluginCRM(
	ctx context.Context,
	dataSource portDS.CRMQueryable,
	databaseName string,
	collections map[string][]string,
	databaseFilters map[string]map[string]modelJob.FilterCondition,
	result map[string]map[string][]map[string]any,
	logger libLog.Logger,
) error {
	_, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data.query_plugin_crm")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.database_name", databaseName),
	)

	if len(collections) == 0 {
		return nil
	}

	// List all collections once for the entire plugin_crm database
	allCollectionNames, err := dataSource.ListCollectionNames(ctx)
	if err != nil {
		libOtel.HandleSpanError(span, "Error listing plugin_crm collections", err)
		return fmt.Errorf("failed to list collections in plugin_crm database: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("plugin_crm database has %d collection(s): %v", len(allCollectionNames), allCollectionNames))

	for collection, fields := range collections {
		collectionFilters := getTableFilters(databaseFilters, collection)

		// Filter collections matching the prefix "collection_"
		prefix := collection + "_"

		var matchingCollections []string

		for _, c := range allCollectionNames {
			if strings.HasPrefix(c, prefix) {
				matchingCollections = append(matchingCollections, c)
			}
		}

		if len(matchingCollections) == 0 {
			err := fmt.Errorf("no collections found matching prefix %q in plugin_crm database", prefix)
			libOtel.HandleSpanError(span, "No matching collections found", err)

			return err
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Prefix %q matched %d collection(s): %v", prefix, len(matchingCollections), matchingCollections))

		if err := processPluginCRMCollectionFn(uc, ctx, dataSource, collection, fields, collectionFilters, matchingCollections, result, logger); err != nil {
			libOtel.HandleSpanError(span, "Error processing plugin_crm collection", err)
			return fmt.Errorf("failed to process plugin_crm collection %s: %w", collection, err)
		}
	}

	return nil
}

// processPluginCRMCollection queries each matching physical collection and merges
// the results into a single dataset keyed by the original logical name.
// matchingCollections contains the pre-filtered list of real collection names
// (e.g., ["holders_06c4f684-...", "holders_abc123-..."]).
func (uc *UseCase) processPluginCRMCollection(
	ctx context.Context,
	dataSource portDS.CRMQueryable,
	collection string,
	fields []string,
	collectionFilters map[string]modelJob.FilterCondition,
	matchingCollections []string,
	result map[string]map[string][]map[string]any,
	logger libLog.Logger,
) error {
	// Query each matching collection and merge results
	var allResults []map[string]any

	for _, realCollection := range matchingCollections {
		collectionResult, err := queryPluginCRMCollectionWithFiltersFn(uc, ctx, dataSource, realCollection, fields, collectionFilters, logger)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Error querying collection %s: %s", realCollection, err.Error()))
			return fmt.Errorf("failed to query plugin_crm collection %s: %w", realCollection, err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Collection %s returned %d documents", realCollection, len(collectionResult)))

		allResults = append(allResults, collectionResult...)
	}

	if result["plugin_crm"] == nil {
		result["plugin_crm"] = make(map[string][]map[string]any)
	}

	result["plugin_crm"][collection] = allResults

	decryptedResult, err := decryptPluginCRMDataFn(uc, logger, result["plugin_crm"][collection], fields)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Error decrypting data for collection %s: %s", collection, err.Error()))
		return fmt.Errorf("error decrypting data for collection %s: %w", collection, err)
	}

	result["plugin_crm"][collection] = decryptedResult

	return nil
}

// queryPluginCRMCollectionWithFilters queries a MongoDB collection with plugin_crm specific filter transformation.
func (uc *UseCase) queryPluginCRMCollectionWithFilters(
	ctx context.Context,
	dataSource portDS.CRMQueryable,
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
		transformedFilter, err := uc.transformPluginCRMAdvancedFilters(ctx, collectionFilters, logger)
		if err != nil {
			return nil, fmt.Errorf("error transforming advanced filters for collection %s: %w", collection, err)
		}

		queryResult, errQueryResult = dataSource.QueryCollectionWithAdvancedFilters(ctx, collection, fields, transformedFilter)
	} else {
		queryResult, errQueryResult = dataSource.QueryCollection(ctx, collection, fields, nil)
	}

	if errQueryResult != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Error querying collection %s: %s", collection, errQueryResult.Error()))
		return nil, fmt.Errorf("failed to query collection %s: %w", collection, errQueryResult)
	}

	return queryResult, nil
}

// transformPluginCRMAdvancedFilters transforms advanced FilterCondition filters for plugin_crm to use search fields.
func (uc *UseCase) transformPluginCRMAdvancedFilters(ctx context.Context, filter map[string]modelJob.FilterCondition, logger libLog.Logger) (map[string]modelJob.FilterCondition, error) {
	if filter == nil {
		return nil, nil
	}

	if uc.crmHashSecretKey == "" {
		return nil, fmt.Errorf("CRM hash secret key not configured")
	}

	crypto := &libCrypto.Crypto{
		HashSecretKey: uc.crmHashSecretKey,
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

			logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Transformed advanced filter: %s -> %s", fieldName, searchField))
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
// Note: ctx is intentionally omitted as the crypto layer receives its logger at construction time.
// Add ctx if trace propagation is needed in the future.
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
	if uc.crmEncryptSecretKey == "" {
		return nil, fmt.Errorf("CRM encrypt secret key not configured")
	}

	if uc.crmHashSecretKey == "" {
		return nil, fmt.Errorf("CRM hash secret key not configured")
	}

	crypto := &libCrypto.Crypto{
		HashSecretKey:    uc.crmHashSecretKey,
		EncryptSecretKey: uc.crmEncryptSecretKey,
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
		return nil, fmt.Errorf("failed to decrypt top-level fields: %w", err)
	}

	if err := uc.decryptNestedFields(decryptedRecord, crypto); err != nil {
		return nil, fmt.Errorf("failed to decrypt nested fields: %w", err)
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
		return fmt.Errorf("failed to decrypt contact fields: %w", err)
	}

	if err := uc.decryptBankingDetailsFields(record, crypto); err != nil {
		return fmt.Errorf("failed to decrypt banking details fields: %w", err)
	}

	if err := uc.decryptLegalPersonFields(record, crypto); err != nil {
		return fmt.Errorf("failed to decrypt legal person fields: %w", err)
	}

	if err := uc.decryptNaturalPersonFields(record, crypto); err != nil {
		return fmt.Errorf("failed to decrypt natural person fields: %w", err)
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
		return fmt.Errorf("failed to decrypt value: %w", err)
	}

	container[fieldName] = *decryptedValue

	return nil
}
