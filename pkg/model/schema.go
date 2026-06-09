package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
)

const (
	ErrTypeDataSourceNotFound = "DATA_SOURCE_NOT_FOUND"
	ErrTypeTableNotFound      = "TABLE_NOT_FOUND"
	ErrTypeFieldNotFound      = "FIELD_NOT_FOUND"
	ErrTypeDataSourceDown     = "DATA_SOURCE_DOWN"
)

const (
	MaxDataSourcesPerRequest = 10
	MaxTablesPerDataSource   = 20
	MaxFieldsPerTable        = 50
)

// Schema validation status constants.
const (
	StatusSuccess = "success"
	StatusFailure = "failure"
)

// SchemaValidationRequest represents the POST /v1/management/connections/validate-schema request body.
//
// swagger:model SchemaValidationRequest
//
// @Description Request body for schema validation containing mapped fields per datasource.
type SchemaValidationRequest struct {
	// MappedFields maps datasource config names to their tables and fields
	// Key: configName (e.g., "midaz_onboarding")
	// Value: map of table names to field names
	MappedFields map[string]map[string][]string `json:"mappedFields" validate:"required"`
}

// SchemaValidationResponse represents the response for schema validation.
type SchemaValidationResponse struct {
	Status  string                  `json:"status"` // "success" or "failure"
	Message string                  `json:"message"`
	Errors  []SchemaValidationError `json:"errors,omitempty"`
}

// SchemaValidationError represents a single validation error in the response.
type SchemaValidationError struct {
	Type         string `json:"type"`
	DataSourceID string `json:"dataSourceId"`
	Table        string `json:"table,omitempty"`
	Field        string `json:"field,omitempty"`
}

// SchemaValidationErrorResponse represents a validation failure response (HTTP 422).
// This follows the project's standard error response format (Title, Code, Message)
// with an additional Errors array containing detailed validation failures.
// Note: Success responses use SchemaValidationResponse with a different structure.
type SchemaValidationErrorResponse struct {
	Title   string                  `json:"title"`
	Code    string                  `json:"code"`
	Message string                  `json:"message"`
	Errors  []SchemaValidationError `json:"errors"`
}

// ToMapWithMask returns a masked version for logging.
func (r *SchemaValidationRequest) ToMapWithMask() map[string]any {
	return map[string]any{
		"mappedFields":    r.MappedFields,
		"datasourceCount": len(r.MappedFields),
	}
}

// DataSourceSchema represents the schema of a datasource.
type DataSourceSchema struct {
	ConfigName string                  `json:"configName"`
	Tables     map[string]*TableSchema `json:"tables"` // key: tableName
	CachedAt   time.Time               `json:"cachedAt"`
	ExpiresAt  time.Time               `json:"expiresAt"`
}

// TableSchema represents the schema of a single table/collection.
type TableSchema struct {
	TableName string          `json:"tableName"`
	Columns   map[string]bool `json:"columns"` // map para O(1) lookup
}

// NewDataSourceSchema creates a new DataSourceSchema entity.
func NewDataSourceSchema(configName string) *DataSourceSchema {
	return &DataSourceSchema{
		ConfigName: configName,
		Tables:     make(map[string]*TableSchema),
	}
}

// NewTableSchema creates a new TableSchema entity from a list of columns.
func NewTableSchema(tableName string, columns []string) *TableSchema {
	ts := &TableSchema{
		TableName: tableName,
		Columns:   make(map[string]bool, len(columns)),
	}
	for _, col := range columns {
		ts.Columns[col] = true
	}

	return ts
}

// AddTable adds a table to the schema.
func (s *DataSourceSchema) AddTable(tableName string, columns []string) {
	s.Tables[tableName] = NewTableSchema(tableName, columns)
}

// HasTable checks if the schema contains a specific table (case-sensitive).
func (s *DataSourceSchema) HasTable(tableName string) bool {
	_, exists := s.Tables[tableName]
	return exists
}

// HasField checks if a table has a specific field (case-sensitive).
// Also recognizes parent objects: if "natural_person" is not a direct column
// but "natural_person.mother_name" exists, "natural_person" is valid.
func (s *DataSourceSchema) HasField(tableName, fieldName string) bool {
	table, exists := s.Tables[tableName]
	if !exists {
		return false
	}

	if table.Columns[fieldName] {
		return true
	}

	// Check if fieldName is a parent object with nested fields (e.g. "natural_person"
	// when "natural_person.mother_name" exists in the schema).
	prefix := fieldName + "."
	for col := range table.Columns {
		if strings.HasPrefix(col, prefix) {
			return true
		}
	}

	return false
}

// IsExpired checks if the cached schema has expired.
func (s *DataSourceSchema) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// SetCacheTTL sets the cache timestamps.
func (s *DataSourceSchema) SetCacheTTL(ttl time.Duration) {
	now := time.Now().UTC()
	s.CachedAt = now
	s.ExpiresAt = now.Add(ttl)
}

// GetColumnsList returns columns as a slice (for serialization).
func (t *TableSchema) GetColumnsList() []string {
	columns := make([]string, 0, len(t.Columns))
	for col := range t.Columns {
		columns = append(columns, col)
	}

	return columns
}

// SchemaValidationSpec represents the specification for schema validation.
type SchemaValidationSpec struct {
	MappedFields map[string]map[string][]string
}

// NewSchemaValidationSpec creates a new SchemaValidationSpec from a request DTO.
func NewSchemaValidationSpec(request SchemaValidationRequest) *SchemaValidationSpec {
	return &SchemaValidationSpec{
		MappedFields: request.MappedFields,
	}
}

// Validate validates the schema validation specification.
func (s *SchemaValidationSpec) Validate() error {
	if s.MappedFields == nil {
		return pkg.ValidationError{
			EntityType: "schema",
			Code:       constant.ErrMissingFieldsInRequest.Error(),
			Title:      "Missing Fields in Request",
			Message:    "mappedFields is required",
		}
	}

	if len(s.MappedFields) == 0 {
		return pkg.ValidationError{
			EntityType: "schema",
			Code:       constant.ErrInvalidDataRequest.Error(),
			Title:      "Invalid Data Request",
			Message:    "mappedFields cannot be empty",
		}
	}

	if len(s.MappedFields) > MaxDataSourcesPerRequest {
		return pkg.ValidationError{
			EntityType: "schema",
			Code:       constant.ErrSchemaValidationLimit.Error(),
			Title:      "Validation Limit Exceeded",
			Message:    fmt.Sprintf("maximum %d datasources allowed per validation request", MaxDataSourcesPerRequest),
		}
	}

	for configName, tables := range s.MappedFields {
		// Validate config name is not empty or whitespace-only
		if strings.TrimSpace(configName) == "" {
			return pkg.ValidationError{
				EntityType: "schema",
				Code:       constant.ErrInvalidDataRequest.Error(),
				Title:      "Invalid Data Request",
				Message:    "config name cannot be empty or whitespace-only",
			}
		}

		if len(tables) > MaxTablesPerDataSource {
			return pkg.ValidationError{
				EntityType: "schema",
				Code:       constant.ErrSchemaValidationLimit.Error(),
				Title:      "Validation Limit Exceeded",
				Message:    fmt.Sprintf("maximum %d tables allowed per datasource, got %d for '%s'", MaxTablesPerDataSource, len(tables), configName),
			}
		}

		for tableName, fields := range tables {
			// Validate table name is not empty or whitespace-only
			if strings.TrimSpace(tableName) == "" {
				return pkg.ValidationError{
					EntityType: "schema",
					Code:       constant.ErrInvalidDataRequest.Error(),
					Title:      "Invalid Data Request",
					Message:    fmt.Sprintf("table name cannot be empty or whitespace-only in datasource '%s'", configName),
				}
			}

			if len(fields) > MaxFieldsPerTable {
				return pkg.ValidationError{
					EntityType: "schema",
					Code:       constant.ErrSchemaValidationLimit.Error(),
					Title:      "Validation Limit Exceeded",
					Message:    fmt.Sprintf("maximum %d fields allowed per table, got %d for '%s.%s'", MaxFieldsPerTable, len(fields), configName, tableName),
				}
			}

			// Validate each field name is not empty or whitespace-only
			for _, fieldName := range fields {
				if strings.TrimSpace(fieldName) == "" {
					return pkg.ValidationError{
						EntityType: "schema",
						Code:       constant.ErrInvalidDataRequest.Error(),
						Title:      "Invalid Data Request",
						Message:    fmt.Sprintf("field name cannot be empty or whitespace-only in '%s.%s'", configName, tableName),
					}
				}
			}
		}
	}

	return nil
}

// GetConfigNames returns all datasource config names.
func (s *SchemaValidationSpec) GetConfigNames() []string {
	names := make([]string, 0, len(s.MappedFields))
	for name := range s.MappedFields {
		names = append(names, name)
	}

	return names
}

// GetTables returns all tables names.
func (s *SchemaValidationSpec) GetTablesByConfigName(configName string) map[string][]string {
	tables := make(map[string][]string, len(s.MappedFields[configName]))

	for tableName, value := range s.MappedFields[configName] {
		tables[tableName] = value
	}

	return tables
}

// ValidateAgainstSchema validates the spec against a DataSourceSchema.
func (s *SchemaValidationSpec) ValidateAgainstSchema(
	configName string,
	schema *DataSourceSchema,
) []SchemaValidationError {
	var errors []SchemaValidationError

	tables, exists := s.MappedFields[configName]
	if !exists {
		return errors
	}

	// Validate each table and its fields against the schema
	for tableOrSchemaWithTable, fields := range tables {
		// Ensure that only the table name is returned
		_, tableName := splitSchemaTable(tableOrSchemaWithTable)

		// Check if table exists in schema
		if !schema.HasTable(tableName) {
			errors = append(errors, SchemaValidationError{
				Type:         ErrTypeTableNotFound,
				DataSourceID: configName,
				Table:        tableName,
			})

			continue
		}

		// Check if each field exists in the table schema
		for _, fieldName := range fields {
			if !schema.HasField(tableName, fieldName) {
				errors = append(errors, SchemaValidationError{
					Type:         ErrTypeFieldNotFound,
					DataSourceID: configName,
					Table:        tableName,
					Field:        fieldName,
				})
			}
		}
	}

	return errors
}

// NewSuccessResponse creates a success response.
func NewSuccessResponse() *SchemaValidationResponse {
	return &SchemaValidationResponse{
		Status:  StatusSuccess,
		Message: "Schema validated successfully. All datasources, tables, and fields exist.",
	}
}

// NewFailureResponse creates a failure response with errors.
func NewFailureResponse(errors []SchemaValidationError) *SchemaValidationResponse {
	return &SchemaValidationResponse{
		Status:  StatusFailure,
		Message: "Schema validation found inconsistencies.",
		Errors:  errors,
	}
}

// NewDataSourceNotFoundError creates a DATA_SOURCE_NOT_FOUND error.
func NewDataSourceNotFoundError(configName string) SchemaValidationError {
	return SchemaValidationError{
		Type:         ErrTypeDataSourceNotFound,
		DataSourceID: configName,
	}
}

// NewDataSourceDownError creates a DATA_SOURCE_DOWN error.
func NewDataSourceDownError(configName string) SchemaValidationError {
	return SchemaValidationError{
		Type:         ErrTypeDataSourceDown,
		DataSourceID: configName,
	}
}

func splitSchemaTable(qualified string) (schema string, table string) {
	qualified = strings.TrimSpace(qualified)
	if qualified == "" {
		return "", ""
	}

	schema, table, hasDot := strings.Cut(qualified, ".")
	if !hasDot {
		return "", qualified
	}

	schema = strings.TrimSpace(schema)
	table = strings.TrimSpace(table)

	if schema == "" || table == "" {
		return "", ""
	}

	return schema, table
}
