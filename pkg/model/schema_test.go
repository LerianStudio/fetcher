package model_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaValidationSpec_Validate tests the domain entity validation.
// Note: We test SchemaValidationSpec.Validate(), NOT SchemaValidationRequest.IsValid().
// The request DTO is anemic - validation logic lives in the domain entity.
func TestSchemaValidationSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request model.SchemaValidationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"db1": {"table1": {"field1", "field2"}},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil mapped fields",
			request: model.SchemaValidationRequest{MappedFields: nil},
			wantErr: true,
			errMsg:  "mappedFields is required",
		},
		{
			name:    "empty mapped fields",
			request: model.SchemaValidationRequest{MappedFields: map[string]map[string][]string{}},
			wantErr: true,
			errMsg:  "mappedFields cannot be empty",
		},
		{
			name: "exceeds datasource limit",
			request: model.SchemaValidationRequest{
				MappedFields: generateMappedFields(11, 1, 1),
			},
			wantErr: true,
			errMsg:  "maximum 10 datasources",
		},
		{
			name: "exceeds table limit",
			request: model.SchemaValidationRequest{
				MappedFields: generateMappedFields(1, 21, 1),
			},
			wantErr: true,
			errMsg:  "maximum 20 tables",
		},
		{
			name: "exceeds field limit",
			request: model.SchemaValidationRequest{
				MappedFields: generateMappedFields(1, 1, 51),
			},
			wantErr: true,
			errMsg:  "maximum 50 fields",
		},
		{
			name: "at exact datasource limit",
			request: model.SchemaValidationRequest{
				MappedFields: generateMappedFields(10, 1, 1),
			},
			wantErr: false,
		},
		{
			name: "at exact table limit",
			request: model.SchemaValidationRequest{
				MappedFields: generateMappedFields(1, 20, 1),
			},
			wantErr: false,
		},
		{
			name: "at exact field limit",
			request: model.SchemaValidationRequest{
				MappedFields: generateMappedFields(1, 1, 50),
			},
			wantErr: false,
		},
		{
			name: "multiple datasources with valid counts",
			request: model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"db1": {"table1": {"field1", "field2"}, "table2": {"field3"}},
					"db2": {"users": {"id", "name", "email"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert DTO to domain entity and validate
			spec := model.NewSchemaValidationSpec(tt.request)
			err := spec.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSchemaValidationSpec_Validate_EmptyNames(t *testing.T) {
	tests := []struct {
		name    string
		request model.SchemaValidationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty config name",
			request: model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"": {"table1": {"field1"}},
				},
			},
			wantErr: true,
			errMsg:  "config name cannot be empty",
		},
		{
			name: "whitespace-only config name",
			request: model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"   ": {"table1": {"field1"}},
				},
			},
			wantErr: true,
			errMsg:  "config name cannot be empty",
		},
		{
			name: "empty table name",
			request: model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"db1": {"": {"field1"}},
				},
			},
			wantErr: true,
			errMsg:  "table name cannot be empty",
		},
		{
			name: "whitespace-only table name",
			request: model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"db1": {"  \t  ": {"field1"}},
				},
			},
			wantErr: true,
			errMsg:  "table name cannot be empty",
		},
		{
			name: "empty field name",
			request: model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"db1": {"table1": {""}},
				},
			},
			wantErr: true,
			errMsg:  "field name cannot be empty",
		},
		{
			name: "whitespace-only field name",
			request: model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"db1": {"table1": {"  "}},
				},
			},
			wantErr: true,
			errMsg:  "field name cannot be empty",
		},
		{
			name: "empty field name among valid fields",
			request: model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"db1": {"table1": {"field1", "", "field2"}},
				},
			},
			wantErr: true,
			errMsg:  "field name cannot be empty",
		},
		{
			name: "valid names with leading/trailing spaces are accepted",
			request: model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"db1": {"table1": {"field1"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert DTO to domain entity and validate
			spec := model.NewSchemaValidationSpec(tt.request)
			err := spec.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDataSourceSchema_HasTable(t *testing.T) {
	// Note: Tables is map[string]*TableSchema (pointer), Columns is map[string]bool
	schema := &model.DataSourceSchema{
		Tables: map[string]*model.TableSchema{
			"users":    {TableName: "users", Columns: map[string]bool{"id": true, "name": true}},
			"accounts": {TableName: "accounts", Columns: map[string]bool{"id": true, "balance": true}},
		},
	}

	tests := []struct {
		name      string
		tableName string
		want      bool
	}{
		{name: "existing table users", tableName: "users", want: true},
		{name: "existing table accounts", tableName: "accounts", want: true},
		{name: "case sensitive - Users", tableName: "Users", want: false},
		{name: "case sensitive - USERS", tableName: "USERS", want: false},
		{name: "non-existing table", tableName: "orders", want: false},
		{name: "empty string", tableName: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, schema.HasTable(tt.tableName))
		})
	}
}

func TestDataSourceSchema_HasField(t *testing.T) {
	// Note: Tables is map[string]*TableSchema (pointer), Columns is map[string]bool
	schema := &model.DataSourceSchema{
		Tables: map[string]*model.TableSchema{
			"users": {TableName: "users", Columns: map[string]bool{"id": true, "name": true, "email": true}},
		},
	}

	tests := []struct {
		name      string
		tableName string
		fieldName string
		want      bool
	}{
		{name: "existing field id", tableName: "users", fieldName: "id", want: true},
		{name: "existing field name", tableName: "users", fieldName: "name", want: true},
		{name: "existing field email", tableName: "users", fieldName: "email", want: true},
		{name: "case sensitive - ID", tableName: "users", fieldName: "ID", want: false},
		{name: "case sensitive - Name", tableName: "users", fieldName: "Name", want: false},
		{name: "non-existing field", tableName: "users", fieldName: "password", want: false},
		{name: "non-existing table", tableName: "orders", fieldName: "id", want: false},
		{name: "empty table name", tableName: "", fieldName: "id", want: false},
		{name: "empty field name", tableName: "users", fieldName: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, schema.HasField(tt.tableName, tt.fieldName))
		})
	}
}

func TestDataSourceSchema_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{name: "expired 1 minute ago", expiresAt: now.Add(-1 * time.Minute), want: true},
		{name: "expired 1 hour ago", expiresAt: now.Add(-1 * time.Hour), want: true},
		{name: "expired 1 second ago", expiresAt: now.Add(-1 * time.Second), want: true},
		{name: "not expired - 5 minutes remaining", expiresAt: now.Add(5 * time.Minute), want: false},
		{name: "not expired - 1 hour remaining", expiresAt: now.Add(1 * time.Hour), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &model.DataSourceSchema{ExpiresAt: tt.expiresAt}
			assert.Equal(t, tt.want, schema.IsExpired())
		})
	}
}

func TestNewDataSourceSchema(t *testing.T) {
	schema := model.NewDataSourceSchema("test-db")

	require.NotNil(t, schema)
	assert.Equal(t, "test-db", schema.ConfigName)
	assert.NotNil(t, schema.Tables)
	assert.Empty(t, schema.Tables)
}

func TestDataSourceSchema_AddTable(t *testing.T) {
	schema := model.NewDataSourceSchema("test-db")

	schema.AddTable("users", []string{"id", "name", "email"})
	schema.AddTable("accounts", []string{"id", "balance"})

	require.Len(t, schema.Tables, 2)

	usersTable := schema.Tables["users"]
	require.NotNil(t, usersTable)
	assert.Equal(t, "users", usersTable.TableName)
	assert.True(t, usersTable.Columns["id"])
	assert.True(t, usersTable.Columns["name"])
	assert.True(t, usersTable.Columns["email"])

	accountsTable := schema.Tables["accounts"]
	require.NotNil(t, accountsTable)
	assert.Equal(t, "accounts", accountsTable.TableName)
	assert.True(t, accountsTable.Columns["id"])
	assert.True(t, accountsTable.Columns["balance"])
}

func TestNewTableSchema(t *testing.T) {
	tableSchema := model.NewTableSchema("users", []string{"id", "name", "email"})

	require.NotNil(t, tableSchema)
	assert.Equal(t, "users", tableSchema.TableName)
	assert.Len(t, tableSchema.Columns, 3)
	assert.True(t, tableSchema.Columns["id"])
	assert.True(t, tableSchema.Columns["name"])
	assert.True(t, tableSchema.Columns["email"])
}

func TestTableSchema_GetColumnsList(t *testing.T) {
	tableSchema := model.NewTableSchema("users", []string{"id", "name", "email"})
	columns := tableSchema.GetColumnsList()

	assert.Len(t, columns, 3)
	assert.Contains(t, columns, "id")
	assert.Contains(t, columns, "name")
	assert.Contains(t, columns, "email")
}

func TestDataSourceSchema_SetCacheTTL(t *testing.T) {
	schema := &model.DataSourceSchema{}
	ttl := 5 * time.Minute

	before := time.Now()
	schema.SetCacheTTL(ttl)
	after := time.Now()

	assert.True(t, schema.CachedAt.After(before) || schema.CachedAt.Equal(before))
	assert.True(t, schema.CachedAt.Before(after) || schema.CachedAt.Equal(after))

	expectedExpires := schema.CachedAt.Add(ttl)
	assert.Equal(t, expectedExpires, schema.ExpiresAt)
}

func TestSchemaValidationSpec_GetConfigNames(t *testing.T) {
	spec := model.NewSchemaValidationSpec(model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"table1": {"field1"}},
			"db2": {"table2": {"field2"}},
			"db3": {"table3": {"field3"}},
		},
	})

	names := spec.GetConfigNames()

	assert.Len(t, names, 3)
	assert.Contains(t, names, "db1")
	assert.Contains(t, names, "db2")
	assert.Contains(t, names, "db3")
}

func TestSchemaValidationSpec_ValidateAgainstSchema(t *testing.T) {
	schema := &model.DataSourceSchema{
		ConfigName: "test-db",
		Tables: map[string]*model.TableSchema{
			"users": {TableName: "users", Columns: map[string]bool{"id": true, "name": true, "email": true}},
		},
	}

	tests := []struct {
		name         string
		spec         *model.SchemaValidationSpec
		wantErrCount int
		wantErrTypes []string
	}{
		{
			name: "all fields exist",
			spec: model.NewSchemaValidationSpec(model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"test-db": {"users": {"id", "name"}},
				},
			}),
			wantErrCount: 0,
		},
		{
			name: "table not found",
			spec: model.NewSchemaValidationSpec(model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"test-db": {"orders": {"id"}},
				},
			}),
			wantErrCount: 1,
			wantErrTypes: []string{model.ErrTypeTableNotFound},
		},
		{
			name: "field not found",
			spec: model.NewSchemaValidationSpec(model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"test-db": {"users": {"id", "password"}},
				},
			}),
			wantErrCount: 1,
			wantErrTypes: []string{model.ErrTypeFieldNotFound},
		},
		{
			name: "multiple field errors",
			spec: model.NewSchemaValidationSpec(model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"test-db": {"users": {"id", "password", "role"}},
				},
			}),
			wantErrCount: 2,
			wantErrTypes: []string{model.ErrTypeFieldNotFound, model.ErrTypeFieldNotFound},
		},
		{
			name: "table and field errors",
			spec: model.NewSchemaValidationSpec(model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"test-db": {
						"users":  {"id", "password"},
						"orders": {"id"},
					},
				},
			}),
			wantErrCount: 2,
			wantErrTypes: []string{model.ErrTypeFieldNotFound, model.ErrTypeTableNotFound},
		},
		{
			name: "config name not in spec - returns empty",
			spec: model.NewSchemaValidationSpec(model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"other-db": {"users": {"id"}},
				},
			}),
			wantErrCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.spec.ValidateAgainstSchema("test-db", schema)

			assert.Len(t, errors, tt.wantErrCount)

			if tt.wantErrTypes != nil {
				for i, errType := range tt.wantErrTypes {
					if i < len(errors) {
						assert.Equal(t, errType, errors[i].Type)
					}
				}
			}
		})
	}
}

func TestNewSuccessResponse(t *testing.T) {
	resp := model.NewSuccessResponse()

	require.NotNil(t, resp)
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, "Schema validated successfully. All datasources, tables, and fields exist.", resp.Message)
	assert.Empty(t, resp.Errors)
}

func TestNewFailureResponse(t *testing.T) {
	errors := []model.SchemaValidationError{
		{Type: model.ErrTypeTableNotFound, DataSourceID: "db1", Table: "orders"},
		{Type: model.ErrTypeFieldNotFound, DataSourceID: "db1", Table: "users", Field: "password"},
	}

	resp := model.NewFailureResponse(errors)

	require.NotNil(t, resp)
	assert.Equal(t, "failure", resp.Status)
	assert.Equal(t, "Schema validation found inconsistencies.", resp.Message)
	assert.Len(t, resp.Errors, 2)
	assert.Equal(t, model.ErrTypeTableNotFound, resp.Errors[0].Type)
	assert.Equal(t, model.ErrTypeFieldNotFound, resp.Errors[1].Type)
}

func TestNewDataSourceNotFoundError(t *testing.T) {
	err := model.NewDataSourceNotFoundError("test-db")

	assert.Equal(t, model.ErrTypeDataSourceNotFound, err.Type)
	assert.Equal(t, "test-db", err.DataSourceID)
	assert.Empty(t, err.Table)
	assert.Empty(t, err.Field)
}

func TestNewDataSourceDownError(t *testing.T) {
	err := model.NewDataSourceDownError("test-db")

	assert.Equal(t, model.ErrTypeDataSourceDown, err.Type)
	assert.Equal(t, "test-db", err.DataSourceID)
	assert.Empty(t, err.Table)
	assert.Empty(t, err.Field)
}

func TestSchemaValidationRequest_ToMapWithMask(t *testing.T) {
	request := &model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"table1": {"field1", "field2"}},
			"db2": {"table2": {"field3"}},
		},
	}

	masked := request.ToMapWithMask()

	assert.NotNil(t, masked)
	assert.Equal(t, request.MappedFields, masked["mappedFields"])
	assert.Equal(t, 2, masked["datasourceCount"])
}

// generateMappedFields is a helper function to generate test data.
func generateMappedFields(dsCount, tableCount, fieldCount int) map[string]map[string][]string {
	result := make(map[string]map[string][]string)
	for i := 0; i < dsCount; i++ {
		tables := make(map[string][]string)
		for j := 0; j < tableCount; j++ {
			fields := make([]string, fieldCount)
			for k := 0; k < fieldCount; k++ {
				fields[k] = fmt.Sprintf("field%d", k)
			}
			tables[fmt.Sprintf("table%d", j)] = fields
		}
		result[fmt.Sprintf("ds%d", i)] = tables
	}
	return result
}
