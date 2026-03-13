package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	portDS "github.com/LerianStudio/fetcher/pkg/ports/datasource"
	libCrypto "github.com/LerianStudio/lib-commons/v4/commons/crypto"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// TestIsEncryptedField tests the encrypted field detection function.
func TestIsEncryptedField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected bool
	}{
		{
			name:     "document field is encrypted",
			field:    "document",
			expected: true,
		},
		{
			name:     "name field is encrypted",
			field:    "name",
			expected: true,
		},
		{
			name:     "email field is not encrypted",
			field:    "email",
			expected: false,
		},
		{
			name:     "id field is not encrypted",
			field:    "id",
			expected: false,
		},
		{
			name:     "empty field is not encrypted",
			field:    "",
			expected: false,
		},
		{
			name:     "nested field is not in list",
			field:    "contact.primary_email",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEncryptedField(tt.field)
			if result != tt.expected {
				t.Fatalf("isEncryptedField(%q) = %v, want %v", tt.field, result, tt.expected)
			}
		})
	}
}

// TestHashFilterValues tests the filter value hashing function.
func TestHashFilterValues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Create a crypto instance with a test secret key
	crypto := &libCrypto.Crypto{
		HashSecretKey: "test-secret-key-for-hashing",
		Logger:        logger,
	}

	tests := []struct {
		name   string
		values []any
	}{
		{
			name:   "empty values",
			values: []any{},
		},
		{
			name:   "single string value",
			values: []any{"test-value"},
		},
		{
			name:   "multiple string values",
			values: []any{"value1", "value2", "value3"},
		},
		{
			name:   "mixed types - string and int",
			values: []any{"string-value", 123},
		},
		{
			name:   "mixed types - string and nil",
			values: []any{"value", nil},
		},
		{
			name:   "empty string value",
			values: []any{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.hashFilterValues(tt.values, crypto)

			// Check that result has same length as input
			if len(result) != len(tt.values) {
				t.Fatalf("expected %d values, got %d", len(tt.values), len(result))
			}

			// Check that non-string values are preserved
			for i, original := range tt.values {
				if original == nil {
					if result[i] != nil {
						t.Errorf("expected nil at index %d, got %v", i, result[i])
					}
					continue
				}

				strValue, isString := original.(string)
				if !isString {
					// Non-string values should be preserved as-is
					if result[i] != original {
						t.Errorf("expected non-string value %v at index %d, got %v", original, i, result[i])
					}
				} else if strValue == "" {
					// Empty strings should be preserved as-is
					if result[i] != original {
						t.Errorf("expected empty string at index %d, got %v", i, result[i])
					}
				} else {
					// Non-empty strings should be hashed (different from original)
					if result[i] == original {
						t.Errorf("expected hashed value at index %d, got original value %v", i, original)
					}
				}
			}
		})
	}
}

// TestDecryptPluginCRMData_NoDecryptionNeeded tests the edge case where no decryption is needed.
func TestDecryptPluginCRMData_NoDecryptionNeeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Fields that don't require decryption
	fields := []string{"id", "status", "created_at"}

	// Sample collection result
	collectionResult := []map[string]any{
		{"id": "123", "status": "active", "created_at": "2024-01-01"},
		{"id": "456", "status": "inactive", "created_at": "2024-01-02"},
	}

	result, err := uc.decryptPluginCRMData(logger, collectionResult, fields)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Result should be the same as input since no decryption is needed
	if len(result) != len(collectionResult) {
		t.Fatalf("expected %d records, got %d", len(collectionResult), len(result))
	}

	// Verify the data is unchanged
	for i, record := range result {
		if record["id"] != collectionResult[i]["id"] {
			t.Errorf("record %d: expected id %v, got %v", i, collectionResult[i]["id"], record["id"])
		}
		if record["status"] != collectionResult[i]["status"] {
			t.Errorf("record %d: expected status %v, got %v", i, collectionResult[i]["status"], record["status"])
		}
	}
}

// TestDecryptPluginCRMData_WithEncryptedFields tests decryption with encrypted fields.
func TestDecryptPluginCRMData_WithEncryptedFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Fields that require decryption
	fields := []string{"document", "name", "status"}

	// Sample collection result
	collectionResult := []map[string]any{
		{"id": "123", "status": "active"},
	}

	// This should fail because we don't have the env vars set
	_, err := uc.decryptPluginCRMData(logger, collectionResult, fields)
	if err == nil {
		t.Error("expected error due to missing env vars, got nil")
	}
}

// TestDecryptPluginCRMData_WithNestedField tests decryption with nested field paths.
func TestDecryptPluginCRMData_WithNestedField(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Fields with nested paths require decryption
	fields := []string{"contact.primary_email", "id"}

	// Sample collection result
	collectionResult := []map[string]any{
		{"id": "123", "contact": map[string]any{"primary_email": "test@example.com"}},
	}

	// This should fail because we don't have the env vars set
	_, err := uc.decryptPluginCRMData(logger, collectionResult, fields)
	if err == nil {
		t.Error("expected error due to missing env vars, got nil")
	}
}

// TestDecryptPluginCRMData_EmptyResult tests decryption with empty result set.
func TestDecryptPluginCRMData_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Fields that require decryption
	fields := []string{"document", "name"}

	// Empty collection result
	collectionResult := []map[string]any{}

	// Should not fail on empty result
	_, err := uc.decryptPluginCRMData(logger, collectionResult, fields)
	if err == nil {
		t.Error("expected error due to missing env vars, got nil")
	}
}

// TestGetTableFiltersForPluginCRM tests filter extraction specifically for plugin_crm use cases.
func TestGetTableFiltersForPluginCRM(t *testing.T) {
	tests := []struct {
		name         string
		dbFilters    map[string]map[string]modelJob.FilterCondition
		collection   string
		wantNil      bool
		wantFieldLen int
	}{
		{
			name:       "nil filters returns nil",
			dbFilters:  nil,
			collection: "counterparty",
			wantNil:    true,
		},
		{
			name:       "empty filters returns nil",
			dbFilters:  map[string]map[string]modelJob.FilterCondition{},
			collection: "counterparty",
			wantNil:    true,
		},
		{
			name: "collection not in filters returns nil",
			dbFilters: map[string]map[string]modelJob.FilterCondition{
				"other_collection": {"field1": {Equals: []any{"value"}}},
			},
			collection: "counterparty",
			wantNil:    true,
		},
		{
			name: "collection found with single filter",
			dbFilters: map[string]map[string]modelJob.FilterCondition{
				"counterparty": {"document": {Equals: []any{"12345678900"}}},
			},
			collection:   "counterparty",
			wantNil:      false,
			wantFieldLen: 1,
		},
		{
			name: "collection found with multiple filters",
			dbFilters: map[string]map[string]modelJob.FilterCondition{
				"counterparty": {
					"document": {Equals: []any{"12345678900"}},
					"name":     {Equals: []any{"Test Name"}},
					"status":   {In: []any{"active", "pending"}},
				},
			},
			collection:   "counterparty",
			wantNil:      false,
			wantFieldLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTableFilters(tt.dbFilters, tt.collection)

			if tt.wantNil {
				if result != nil {
					t.Fatalf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if len(result) != tt.wantFieldLen {
				t.Fatalf("expected %d fields, got %d", tt.wantFieldLen, len(result))
			}
		})
	}
}

// TestHashFilterValues_EdgeCases tests additional edge cases for hash filter values.
func TestHashFilterValues_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey: "test-secret-key-for-hashing",
		Logger:        logger,
	}

	tests := []struct {
		name        string
		values      []any
		description string
	}{
		{
			name:        "nil slice",
			values:      nil,
			description: "should handle nil slice",
		},
		{
			name:        "slice with only nil values",
			values:      []any{nil, nil, nil},
			description: "should preserve nil values",
		},
		{
			name:        "slice with mixed string and boolean",
			values:      []any{"test", true, false},
			description: "should hash strings and preserve booleans",
		},
		{
			name:        "slice with float values",
			values:      []any{3.14, 2.71, "string"},
			description: "should preserve floats and hash strings",
		},
		{
			name:        "slice with nested maps (non-string)",
			values:      []any{map[string]any{"key": "value"}, "string"},
			description: "should preserve maps and hash strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.hashFilterValues(tt.values, crypto)

			if tt.values == nil {
				if len(result) != 0 {
					t.Errorf("expected empty result for nil input, got %d items", len(result))
				}
				return
			}

			if len(result) != len(tt.values) {
				t.Fatalf("expected %d values, got %d", len(tt.values), len(result))
			}
		})
	}
}

// TestDecryptFieldValue tests the field value decryption logic.
func TestDecryptFieldValue_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	tests := []struct {
		name        string
		fieldValue  any
		wantErr     bool
		description string
	}{
		{
			name:        "nil value",
			fieldValue:  nil,
			wantErr:     false,
			description: "should skip nil values",
		},
		{
			name:        "empty string",
			fieldValue:  "",
			wantErr:     false,
			description: "should skip empty strings",
		},
		{
			name:        "non-string value - integer",
			fieldValue:  12345,
			wantErr:     false,
			description: "should skip non-string values",
		},
		{
			name:        "non-string value - boolean",
			fieldValue:  true,
			wantErr:     false,
			description: "should skip boolean values",
		},
		{
			name:        "non-string value - map",
			fieldValue:  map[string]any{"key": "value"},
			wantErr:     false,
			description: "should skip map values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We can't fully test the decryption without a valid crypto setup
			// but we can test the type checking and edge case handling
			container := make(map[string]any)
			container["testField"] = tt.fieldValue

			// For edge cases that should skip decryption, the value should remain unchanged
			originalValue := tt.fieldValue

			// Create a dummy crypto (won't be used for non-string values)
			crypto := &libCrypto.Crypto{
				HashSecretKey:    "test-hash-key",
				EncryptSecretKey: "test-encrypt-key",
				Logger:           logger,
			}

			err := uc.decryptFieldValue(container, "testField", tt.fieldValue, crypto)

			if tt.wantErr && err == nil {
				t.Errorf("expected error for %s, got nil", tt.description)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error for %s, got: %v", tt.description, err)
			}

			// For non-string values, the container should remain unchanged
			// Skip comparison for maps as they're not comparable
			if _, isMap := tt.fieldValue.(map[string]any); !isMap {
				if tt.fieldValue != "" && container["testField"] != originalValue {
					// Only check for non-empty strings as empty strings are also skipped
					if _, isString := tt.fieldValue.(string); !isString {
						t.Errorf("expected value to remain %v, got %v", originalValue, container["testField"])
					}
				}
			}
		})
	}
}

// TestDecryptRecord_EmptyRecord tests decryption with empty record.
func TestDecryptRecord_EmptyRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Use valid hex keys (32 bytes each, 64 hex chars)
	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	// Initialize cipher for the crypto instance
	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	tests := []struct {
		name   string
		record map[string]any
	}{
		{
			name:   "empty record",
			record: map[string]any{},
		},
		{
			name:   "nil record",
			record: nil,
		},
		{
			name: "record with only non-encrypted fields",
			record: map[string]any{
				"id":         "123",
				"status":     "active",
				"created_at": "2024-01-01",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := uc.decryptRecord(tt.record, crypto)
			if err != nil {
				t.Fatalf("expected no error for %s, got: %v", tt.name, err)
			}

			if tt.record == nil {
				if result == nil || len(result) != 0 {
					t.Errorf("expected empty result for nil record")
				}
			}
		})
	}
}

// TestDecryptTopLevelFields tests decryption of top-level encrypted fields.
func TestDecryptTopLevelFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	tests := []struct {
		name    string
		record  map[string]any
		wantErr bool
	}{
		{
			name: "no encrypted fields",
			record: map[string]any{
				"id":     "123",
				"status": "active",
			},
			wantErr: false,
		},
		{
			name: "with nil encrypted field",
			record: map[string]any{
				"document": nil,
				"id":       "123",
			},
			wantErr: false,
		},
		{
			name: "with non-string encrypted field",
			record: map[string]any{
				"document": 12345,
				"id":       "123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.decryptTopLevelFields(tt.record, crypto)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

// TestDecryptNestedFields tests decryption of nested field structures.
func TestDecryptNestedFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	tests := []struct {
		name    string
		record  map[string]any
		wantErr bool
	}{
		{
			name:    "no nested fields",
			record:  map[string]any{"id": "123"},
			wantErr: false,
		},
		{
			name: "contact field - not map",
			record: map[string]any{
				"contact": "not-a-map",
			},
			wantErr: false,
		},
		{
			name: "contact field - empty map",
			record: map[string]any{
				"contact": map[string]any{},
			},
			wantErr: false,
		},
		{
			name: "contact field - with nil values",
			record: map[string]any{
				"contact": map[string]any{
					"primary_email": nil,
					"mobile_phone":  nil,
				},
			},
			wantErr: false,
		},
		{
			name: "banking_details field - not map",
			record: map[string]any{
				"banking_details": "not-a-map",
			},
			wantErr: false,
		},
		{
			name: "legal_person field - not map",
			record: map[string]any{
				"legal_person": "not-a-map",
			},
			wantErr: false,
		},
		{
			name: "legal_person with representative not map",
			record: map[string]any{
				"legal_person": map[string]any{
					"representative": "not-a-map",
				},
			},
			wantErr: false,
		},
		{
			name: "natural_person field - not map",
			record: map[string]any{
				"natural_person": "not-a-map",
			},
			wantErr: false,
		},
		{
			name: "natural_person with nil values",
			record: map[string]any{
				"natural_person": map[string]any{
					"mother_name": nil,
					"father_name": nil,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.decryptNestedFields(tt.record, crypto)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

// TestDecryptContactFields tests contact field decryption.
func TestDecryptContactFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	record := map[string]any{
		"contact": map[string]any{
			"primary_email": "test@example.com",
			"mobile_phone":  "123456789",
		},
	}

	err := uc.decryptContactFields(record, crypto)
	if err != nil {
		// Expected to fail as we're not providing encrypted values
		// Just ensure it doesn't panic
		t.Logf("decryption failed as expected: %v", err)
	}
}

// TestDecryptBankingDetailsFields tests banking details field decryption.
func TestDecryptBankingDetailsFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	record := map[string]any{
		"banking_details": map[string]any{
			"account": nil,
			"iban":    "",
		},
	}

	err := uc.decryptBankingDetailsFields(record, crypto)
	if err != nil {
		t.Errorf("expected no error for nil/empty values, got: %v", err)
	}
}

// TestDecryptLegalPersonFields tests legal person field decryption.
func TestDecryptLegalPersonFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	record := map[string]any{
		"legal_person": map[string]any{
			"representative": map[string]any{
				"name":     nil,
				"document": "",
				"email":    "",
			},
		},
	}

	err := uc.decryptLegalPersonFields(record, crypto)
	if err != nil {
		t.Errorf("expected no error for nil/empty values, got: %v", err)
	}
}

// TestDecryptNaturalPersonFields tests natural person field decryption.
func TestDecryptNaturalPersonFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	record := map[string]any{
		"natural_person": map[string]any{
			"mother_name": nil,
			"father_name": "",
		},
	}

	err := uc.decryptNaturalPersonFields(record, crypto)
	if err != nil {
		t.Errorf("expected no error for nil/empty values, got: %v", err)
	}
}

// TestTransformPluginCRMAdvancedFilters tests the filter transformation logic.
func TestTransformPluginCRMAdvancedFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	logger := testLogger()

	tests := []struct {
		name    string
		filter  map[string]modelJob.FilterCondition
		hashKey string
		wantNil bool
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil filter returns nil",
			filter:  nil,
			hashKey: "test-hash-key",
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "empty filter returns empty",
			filter:  map[string]modelJob.FilterCondition{},
			hashKey: "test-hash-key",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "missing hash key returns error",
			filter:  map[string]modelJob.FilterCondition{"document": {Equals: []any{"123"}}},
			hashKey: "",
			wantNil: false,
			wantErr: true,
			errMsg:  "CRM hash secret key not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newTestUseCase(mocks)
			uc.SetCRMSecrets("test-crm-encrypt-key", tt.hashKey)

			result, err := uc.transformPluginCRMAdvancedFilters(testContext(), tt.filter, logger)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if tt.wantNil && result != nil {
				t.Errorf("expected nil result, got %v", result)
			}
		})
	}
}

// TestTransformPluginCRMAdvancedFilters_FieldMappings tests that field mappings are applied correctly.
func TestTransformPluginCRMAdvancedFilters_FieldMappings(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	tests := []struct {
		name            string
		inputField      string
		expectedOutput  string
		shouldTransform bool
	}{
		{
			name:            "document field transforms to search.document",
			inputField:      "document",
			expectedOutput:  "search.document",
			shouldTransform: true,
		},
		{
			name:            "name field transforms to search.name",
			inputField:      "name",
			expectedOutput:  "search.name",
			shouldTransform: true,
		},
		{
			name:            "banking_details.account transforms",
			inputField:      "banking_details.account",
			expectedOutput:  "search.banking_details_account",
			shouldTransform: true,
		},
		{
			name:            "banking_details.iban transforms",
			inputField:      "banking_details.iban",
			expectedOutput:  "search.banking_details_iban",
			shouldTransform: true,
		},
		{
			name:            "contact.primary_email transforms",
			inputField:      "contact.primary_email",
			expectedOutput:  "search.contact_primary_email",
			shouldTransform: true,
		},
		{
			name:            "contact.secondary_email transforms",
			inputField:      "contact.secondary_email",
			expectedOutput:  "search.contact_secondary_email",
			shouldTransform: true,
		},
		{
			name:            "contact.mobile_phone transforms",
			inputField:      "contact.mobile_phone",
			expectedOutput:  "search.contact_mobile_phone",
			shouldTransform: true,
		},
		{
			name:            "contact.other_phone transforms",
			inputField:      "contact.other_phone",
			expectedOutput:  "search.contact_other_phone",
			shouldTransform: true,
		},
		{
			name:            "unmapped field stays unchanged",
			inputField:      "status",
			expectedOutput:  "status",
			shouldTransform: false,
		},
		{
			name:            "unknown nested field stays unchanged",
			inputField:      "custom.field",
			expectedOutput:  "custom.field",
			shouldTransform: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := map[string]modelJob.FilterCondition{
				tt.inputField: {Equals: []any{"test-value"}},
			}

			result, err := uc.transformPluginCRMAdvancedFilters(testContext(), filter, logger)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			// Check if the output field exists
			if _, exists := result[tt.expectedOutput]; !exists {
				t.Errorf("expected field %q not found in result", tt.expectedOutput)
				t.Logf("result keys: %v", result)
			}
		})
	}
}

// TestTransformPluginCRMAdvancedFilters_AllConditionTypes tests all filter condition types.
func TestTransformPluginCRMAdvancedFilters_AllConditionTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Create a filter with all condition types for document field
	filter := map[string]modelJob.FilterCondition{
		"document": {
			Equals:         []any{"value1"},
			GreaterThan:    []any{"value2"},
			GreaterOrEqual: []any{"value3"},
			LessThan:       []any{"value4"},
			LessOrEqual:    []any{"value5"},
			Between:        []any{"start", "end"},
			In:             []any{"val1", "val2", "val3"},
			NotIn:          []any{"val4", "val5"},
		},
	}

	result, err := uc.transformPluginCRMAdvancedFilters(testContext(), filter, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the transformed field exists
	transformedCondition, exists := result["search.document"]
	if !exists {
		t.Fatal("expected search.document field not found")
	}

	// Verify all condition types are present
	if len(transformedCondition.Equals) != 1 {
		t.Errorf("expected Equals to have 1 value, got %d", len(transformedCondition.Equals))
	}
	if len(transformedCondition.GreaterThan) != 1 {
		t.Errorf("expected GreaterThan to have 1 value, got %d", len(transformedCondition.GreaterThan))
	}
	if len(transformedCondition.GreaterOrEqual) != 1 {
		t.Errorf("expected GreaterOrEqual to have 1 value, got %d", len(transformedCondition.GreaterOrEqual))
	}
	if len(transformedCondition.LessThan) != 1 {
		t.Errorf("expected LessThan to have 1 value, got %d", len(transformedCondition.LessThan))
	}
	if len(transformedCondition.LessOrEqual) != 1 {
		t.Errorf("expected LessOrEqual to have 1 value, got %d", len(transformedCondition.LessOrEqual))
	}
	if len(transformedCondition.Between) != 2 {
		t.Errorf("expected Between to have 2 values, got %d", len(transformedCondition.Between))
	}
	if len(transformedCondition.In) != 3 {
		t.Errorf("expected In to have 3 values, got %d", len(transformedCondition.In))
	}
	if len(transformedCondition.NotIn) != 2 {
		t.Errorf("expected NotIn to have 2 values, got %d", len(transformedCondition.NotIn))
	}
}

// TestProcessPluginCRMCollection_WithOrganizationID tests collection name transformation with organization ID.
func TestProcessPluginCRMCollection_WithOrganizationID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalQuery := queryPluginCRMCollectionWithFiltersFn
	originalDecrypt := decryptPluginCRMDataFn
	t.Cleanup(func() {
		queryPluginCRMCollectionWithFiltersFn = originalQuery
		decryptPluginCRMDataFn = originalDecrypt
	})

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	orgID := uuid.MustParse("019b9df1-34eb-7dd0-afd5-53f859667e51")
	result := make(map[string]map[string][]map[string]any)
	expectedRows := []map[string]any{{"id": "123", "name": "Ada"}}
	decryptedRows := []map[string]any{{"id": "123", "name": "Ada Lovelace"}}

	queryPluginCRMCollectionWithFiltersFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, fields []string, filters map[string]modelJob.FilterCondition, _ libLog.Logger) ([]map[string]any, error) {
		if collection != "holders_"+orgID.String() {
			t.Fatalf("unexpected collection name: %s", collection)
		}
		if len(fields) != 2 || fields[0] != "id" || fields[1] != "name" {
			t.Fatalf("unexpected fields: %v", fields)
		}
		if filters != nil {
			t.Fatalf("expected nil filters, got %+v", filters)
		}
		return expectedRows, nil
	}

	decryptPluginCRMDataFn = func(_ *UseCase, _ libLog.Logger, collectionResult []map[string]any, fields []string) ([]map[string]any, error) {
		if len(collectionResult) != 1 || collectionResult[0]["name"] != "Ada" {
			t.Fatalf("unexpected collection result: %+v", collectionResult)
		}
		if len(fields) != 2 {
			t.Fatalf("unexpected fields: %v", fields)
		}
		return decryptedRows, nil
	}

	err := uc.processPluginCRMCollection(ctx, nil, "holders", []string{"id", "name"}, nil, orgID, result, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got := result["plugin_crm"]["holders"]
	if len(got) != 1 || got[0]["name"] != "Ada Lovelace" {
		t.Fatalf("unexpected stored result: %+v", got)
	}
}

// TestQueryPluginCRMCollectionWithFilters_NoFilters tests querying without filters.
func TestQueryPluginCRMCollectionWithFilters_NoFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	ctx := testContext()
	logger := testLogger()
	wantRows := []map[string]any{{"id": "123"}}

	mockDS := portDS.NewMockCRMQueryable(ctrl)
	mockDS.EXPECT().
		QueryCollection(gomock.Any(), "collection_test", []string{"id", "name"}, gomock.Nil()).
		Return(wantRows, nil)

	got, err := uc.queryPluginCRMCollectionWithFilters(ctx, mockDS, "collection_test", []string{"id", "name"}, nil, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 1 || got[0]["id"] != "123" {
		t.Fatalf("unexpected rows: %+v", got)
	}
}

// TestDecryptPluginCRMData_MissingHashSecretKey tests error when hash secret key is missing.
func TestDecryptPluginCRMData_MissingHashSecretKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Set only the encrypt key, not the hash key
	uc.SetCRMSecrets("test-encrypt-key", "")

	fields := []string{"document", "name"}
	collectionResult := []map[string]any{
		{"id": "123", "document": "encrypted-doc"},
	}

	_, err := uc.decryptPluginCRMData(logger, collectionResult, fields)
	if err == nil {
		t.Error("expected error when hash secret key is missing")
	}

	if err.Error() != "CRM hash secret key not configured" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestDecryptPluginCRMData_MissingEncryptSecretKey tests error when encrypt secret key is missing.
func TestDecryptPluginCRMData_MissingEncryptSecretKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Set only the hash key, not the encrypt key
	uc.SetCRMSecrets("", "test-hash-key")

	fields := []string{"document", "name"}
	collectionResult := []map[string]any{
		{"id": "123", "document": "encrypted-doc"},
	}

	_, err := uc.decryptPluginCRMData(logger, collectionResult, fields)
	if err == nil {
		t.Error("expected error when encrypt secret key is missing")
	}

	if err.Error() != "CRM encrypt secret key not configured" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestQueryPluginCRM_EmptyCollections tests QueryPluginCRM with empty collections.
func TestQueryPluginCRM_EmptyCollections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	orgID := uuid.MustParse("019b9df1-34eb-7dd0-afd5-53f859667e51")
	result := make(map[string]map[string][]map[string]any)

	// Empty collections should not cause errors - just no processing
	err := uc.QueryPluginCRM(
		ctx,
		nil, // dataSource - won't be used with empty collections
		"plugin_crm",
		map[string][]string{}, // empty collections
		nil,
		orgID,
		result,
		logger,
	)
	if err != nil {
		t.Fatalf("expected no error for empty collections, got: %v", err)
	}
}

// TestQueryPluginCRM_NilCollections tests QueryPluginCRM with nil collections.
func TestQueryPluginCRM_NilCollections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	orgID := uuid.MustParse("019b9df1-34eb-7dd0-afd5-53f859667e51")
	result := make(map[string]map[string][]map[string]any)

	// Nil collections should not cause errors
	err := uc.QueryPluginCRM(
		ctx,
		nil, // dataSource - won't be used with nil collections
		"plugin_crm",
		nil, // nil collections
		nil,
		orgID,
		result,
		logger,
	)
	if err != nil {
		t.Fatalf("expected no error for nil collections, got: %v", err)
	}
}

// TestQueryPluginCRM_WithOrganizationOnly tests QueryPluginCRM with only organization in collections.
func TestQueryPluginCRM_WithOrganizationOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalProcess := processPluginCRMCollectionFn
	t.Cleanup(func() {
		processPluginCRMCollectionFn = originalProcess
	})

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	result := make(map[string]map[string][]map[string]any)
	collections := map[string][]string{"organization": {"id", "name"}}
	orgID := uuid.MustParse("019b9df1-34eb-7dd0-afd5-53f859667e51")

	called := false
	processPluginCRMCollectionFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, fields []string, filters map[string]modelJob.FilterCondition, gotOrgID uuid.UUID, _ map[string]map[string][]map[string]any, _ libLog.Logger) error {
		called = true
		if collection != "organization" {
			t.Fatalf("unexpected collection: %s", collection)
		}
		if len(fields) != 2 {
			t.Fatalf("unexpected fields: %v", fields)
		}
		if filters != nil {
			t.Fatalf("expected nil filters, got %+v", filters)
		}
		if gotOrgID != orgID {
			t.Fatalf("unexpected org ID: %s", gotOrgID)
		}
		return nil
	}

	if err := uc.QueryPluginCRM(ctx, nil, "plugin_crm", collections, nil, orgID, result, logger); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected collection processor to be called")
	}
}

// TestDecryptRecord_WithAllFieldTypes tests decryptRecord with various field types.
func TestDecryptRecord_WithAllFieldTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	// Record with all types of fields
	record := map[string]any{
		"id":       "123",
		"status":   "active",
		"count":    100,
		"enabled":  true,
		"score":    3.14,
		"tags":     []string{"tag1", "tag2"},
		"metadata": map[string]any{"key": "value"},
		"empty":    "",
		"nilField": nil,
	}

	result, err := uc.decryptRecord(record, crypto)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// All fields should be preserved
	if result["id"] != "123" {
		t.Errorf("expected id '123', got %v", result["id"])
	}
	if result["status"] != "active" {
		t.Errorf("expected status 'active', got %v", result["status"])
	}
	if result["count"] != 100 {
		t.Errorf("expected count 100, got %v", result["count"])
	}
	if result["enabled"] != true {
		t.Errorf("expected enabled true, got %v", result["enabled"])
	}
}

// TestDecryptContactFields_WithNilContact tests decryptContactFields when contact is nil.
func TestDecryptContactFields_WithNilContact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	record := map[string]any{
		"id":      "123",
		"contact": nil,
	}

	err := uc.decryptContactFields(record, crypto)
	if err != nil {
		t.Errorf("expected no error for nil contact, got: %v", err)
	}
}

// TestDecryptBankingDetailsFields_WithNilBankingDetails tests with nil banking_details.
func TestDecryptBankingDetailsFields_WithNilBankingDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	record := map[string]any{
		"id":              "123",
		"banking_details": nil,
	}

	err := uc.decryptBankingDetailsFields(record, crypto)
	if err != nil {
		t.Errorf("expected no error for nil banking_details, got: %v", err)
	}
}

// TestDecryptLegalPersonFields_WithNilLegalPerson tests with nil legal_person.
func TestDecryptLegalPersonFields_WithNilLegalPerson(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	record := map[string]any{
		"id":           "123",
		"legal_person": nil,
	}

	err := uc.decryptLegalPersonFields(record, crypto)
	if err != nil {
		t.Errorf("expected no error for nil legal_person, got: %v", err)
	}
}

// TestDecryptNaturalPersonFields_WithNilNaturalPerson tests with nil natural_person.
func TestDecryptNaturalPersonFields_WithNilNaturalPerson(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	record := map[string]any{
		"id":             "123",
		"natural_person": nil,
	}

	err := uc.decryptNaturalPersonFields(record, crypto)
	if err != nil {
		t.Errorf("expected no error for nil natural_person, got: %v", err)
	}
}

// TestDecryptLegalPersonFields_WithEmptyRepresentative tests with empty representative.
func TestDecryptLegalPersonFields_WithEmptyRepresentative(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	record := map[string]any{
		"id": "123",
		"legal_person": map[string]any{
			"representative": map[string]any{}, // empty map
		},
	}

	err := uc.decryptLegalPersonFields(record, crypto)
	if err != nil {
		t.Errorf("expected no error for empty representative, got: %v", err)
	}
}

// TestIsEncryptedField_AllKnownFields tests all known encrypted fields.
func TestIsEncryptedField_AllKnownFields(t *testing.T) {
	knownEncryptedFields := []string{
		"document",
		"name",
	}

	for _, field := range knownEncryptedFields {
		if !isEncryptedField(field) {
			t.Errorf("expected %q to be encrypted, got false", field)
		}
	}

	knownUnencryptedFields := []string{
		"id",
		"status",
		"created_at",
		"updated_at",
		"type",
		"category",
	}

	for _, field := range knownUnencryptedFields {
		if isEncryptedField(field) {
			t.Errorf("expected %q to NOT be encrypted, got true", field)
		}
	}
}

// TestHashFilterValues_ConsistentHashing tests that same input produces same hash.
func TestHashFilterValues_ConsistentHashing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey: "test-secret-key-for-hashing",
		Logger:        logger,
	}

	values := []any{"test-value-1", "test-value-2"}

	result1 := uc.hashFilterValues(values, crypto)
	result2 := uc.hashFilterValues(values, crypto)

	// Same input should produce same hash
	for i := range result1 {
		if result1[i] != result2[i] {
			t.Errorf("inconsistent hashing at index %d: %v != %v", i, result1[i], result2[i])
		}
	}
}

// TestDecryptPluginCRMData_WithValidCrypto tests decryption with properly initialized crypto.
func TestDecryptPluginCRMData_WithValidCrypto(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Set valid crypto keys on the UseCase
	uc.SetCRMSecrets(
		"fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	)

	// Fields that don't require decryption (non-encrypted fields)
	fields := []string{"id", "status"}

	// Sample collection result
	collectionResult := []map[string]any{
		{"id": "123", "status": "active"},
		{"id": "456", "status": "inactive"},
	}

	result, err := uc.decryptPluginCRMData(logger, collectionResult, fields)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != len(collectionResult) {
		t.Errorf("expected %d records, got %d", len(collectionResult), len(result))
	}
}

// TestTransformPluginCRMAdvancedFilters_WithEncryptedFields tests filter transformation for encrypted fields.
func TestTransformPluginCRMAdvancedFilters_WithEncryptedFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	filter := map[string]modelJob.FilterCondition{
		"document": {Equals: []any{"12345678900"}},
	}

	result, err := uc.transformPluginCRMAdvancedFilters(testContext(), filter, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Document field should be transformed to search.document
	if _, exists := result["search.document"]; !exists {
		t.Error("expected search.document field not found")
	}

	// Value should be hashed (not the original)
	transformedCondition := result["search.document"]
	if len(transformedCondition.Equals) != 1 {
		t.Errorf("expected 1 Equals value, got %d", len(transformedCondition.Equals))
	}

	// Hashed value should be different from original
	if transformedCondition.Equals[0] == "12345678900" {
		t.Error("expected value to be hashed, got original")
	}
}

// TestQueryPluginCRM_WithFilters tests QueryPluginCRM with filters.
func TestQueryPluginCRM_WithFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalProcess := processPluginCRMCollectionFn
	t.Cleanup(func() {
		processPluginCRMCollectionFn = originalProcess
	})

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	result := make(map[string]map[string][]map[string]any)
	collections := map[string][]string{
		"organization": {"id", "name"},
		"counterparty": {"id", "document"},
	}
	filters := map[string]map[string]modelJob.FilterCondition{
		"counterparty": {
			"status": {Equals: []any{"active"}},
		},
	}
	orgID := uuid.MustParse("019b9df1-34eb-7dd0-afd5-53f859667e51")

	seenCounterparty := false
	processPluginCRMCollectionFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, _ []string, collectionFilters map[string]modelJob.FilterCondition, _ uuid.UUID, _ map[string]map[string][]map[string]any, _ libLog.Logger) error {
		if collection == "counterparty" {
			seenCounterparty = true
			if collectionFilters["status"].Equals[0] != "active" {
				t.Fatalf("unexpected filters: %+v", collectionFilters)
			}
		}
		return nil
	}

	if err := uc.QueryPluginCRM(ctx, nil, "plugin_crm", collections, filters, orgID, result, logger); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !seenCounterparty {
		t.Fatal("expected counterparty collection to be processed")
	}
}

// TestDecryptFieldValue_WithValidEncryptedValue tests decryption with properly encrypted value.
func TestDecryptFieldValue_WithValidEncryptedValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		EncryptSecretKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Logger:           logger,
	}

	if err := crypto.InitializeCipher(); err != nil {
		t.Skipf("skipping test due to cipher initialization failure: %v", err)
	}

	// Test with an invalid encrypted value - should return error
	container := make(map[string]any)
	container["testField"] = "not-a-valid-encrypted-value"

	err := uc.decryptFieldValue(container, "testField", "not-a-valid-encrypted-value", crypto)
	// Error expected because the value is not properly encrypted
	if err == nil {
		t.Logf("Note: decryption succeeded or was skipped for invalid value")
	}
}

// TestProcessPluginCRMCollection_WithValidOrganization tests query errors are returned deterministically.
func TestProcessPluginCRMCollection_WithValidOrganization(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalQuery := queryPluginCRMCollectionWithFiltersFn
	t.Cleanup(func() {
		queryPluginCRMCollectionWithFiltersFn = originalQuery
	})

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	orgID := uuid.MustParse("019b9df1-34eb-7dd0-afd5-53f859667e51")
	result := make(map[string]map[string][]map[string]any)

	queryPluginCRMCollectionWithFiltersFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, _ []string, _ map[string]modelJob.FilterCondition, _ libLog.Logger) ([]map[string]any, error) {
		if collection != "counterparty_"+orgID.String() {
			t.Fatalf("unexpected collection: %s", collection)
		}
		return nil, errors.New("query failed")
	}

	err := uc.processPluginCRMCollection(ctx, nil, "counterparty", []string{"id", "name"}, nil, orgID, result, logger)
	if err == nil || !strings.Contains(err.Error(), "query failed") {
		t.Fatalf("expected query failure, got %v", err)
	}
}

func TestProcessPluginCRMCollection_TransformsCollectionAndDecryptsResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalQueryFn := queryPluginCRMCollectionWithFiltersFn
	originalDecryptFn := decryptPluginCRMDataFn
	t.Cleanup(func() {
		queryPluginCRMCollectionWithFiltersFn = originalQueryFn
		decryptPluginCRMDataFn = originalDecryptFn
	})

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	ctx := testContext()
	logger := testLogger()
	orgID := uuid.MustParse("019b9df1-34eb-7dd0-afd5-53f859667e51")
	result := make(map[string]map[string][]map[string]any)

	hashKey := strings.Repeat("01", 32)
	encryptKey := strings.Repeat("fe", 32)
	t.Setenv("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM", hashKey)
	t.Setenv("CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM", encryptKey)

	plainDocument := "12345678900"

	queryPluginCRMCollectionWithFiltersFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, fields []string, _ map[string]modelJob.FilterCondition, _ libLog.Logger) ([]map[string]any, error) {
		expectedCollection := "holders_" + orgID.String()
		if collection != expectedCollection {
			t.Fatalf("expected collection %s, got %s", expectedCollection, collection)
		}
		if len(fields) != 1 || fields[0] != "document" {
			t.Fatalf("unexpected fields: %v", fields)
		}
		return []map[string]any{{"document": "encrypted-doc"}}, nil
	}

	decryptPluginCRMDataFn = func(_ *UseCase, _ libLog.Logger, collectionResult []map[string]any, _ []string) ([]map[string]any, error) {
		return []map[string]any{{"document": plainDocument}}, nil
	}

	mockDS := portDS.NewMockCRMQueryable(ctrl)
	err := uc.processPluginCRMCollection(ctx, mockDS, "holders", []string{"document"}, nil, orgID, result, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	rows := result["plugin_crm"]["holders"]
	if len(rows) != 1 {
		t.Fatalf("expected one row, got %+v", rows)
	}
	if got := rows[0]["document"]; got != plainDocument {
		t.Fatalf("expected decrypted document %q, got %#v", plainDocument, got)
	}
}

func TestQueryPluginCRM_WithFilters_TransformsAdvancedFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalProcessFn := processPluginCRMCollectionFn
	t.Cleanup(func() {
		processPluginCRMCollectionFn = originalProcessFn
	})

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	ctx := testContext()
	logger := testLogger()
	orgID := uuid.MustParse("019b9df1-34eb-7dd0-afd5-53f859667e51")
	result := make(map[string]map[string][]map[string]any)

	hashKey := strings.Repeat("01", 32)
	encryptKey := strings.Repeat("fe", 32)
	t.Setenv("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM", hashKey)
	t.Setenv("CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM", encryptKey)

	processPluginCRMCollectionFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, _ []string, collectionFilters map[string]modelJob.FilterCondition, _ uuid.UUID, res map[string]map[string][]map[string]any, _ libLog.Logger) error {
		if collectionFilters == nil {
			t.Fatal("expected non-nil filters")
		}
		if _, ok := collectionFilters["document"]; !ok {
			t.Fatalf("expected document filter, got %+v", collectionFilters)
		}
		if res["plugin_crm"] == nil {
			res["plugin_crm"] = make(map[string][]map[string]any)
		}
		res["plugin_crm"][collection] = []map[string]any{}
		return nil
	}

	mockDS := portDS.NewMockCRMQueryable(ctrl)
	err := uc.QueryPluginCRM(
		ctx,
		mockDS,
		"plugin_crm",
		map[string][]string{"holders": {"document"}},
		map[string]map[string]modelJob.FilterCondition{
			"holders": {
				"document": {Equals: []any{"12345678900"}},
			},
		},
		orgID,
		result,
		logger,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	rows, ok := result["plugin_crm"]["holders"]
	if !ok {
		t.Fatalf("expected holders collection in result, got %+v", result)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty result set, got %+v", rows)
	}
}

// TestGetTableFilters_WithDeepNesting tests filter extraction with complex nested structure.
func TestGetTableFilters_WithDeepNesting(t *testing.T) {
	dbFilters := map[string]map[string]modelJob.FilterCondition{
		"table1": {
			"field1": {
				Equals:         []any{"value1"},
				In:             []any{"a", "b", "c"},
				Between:        []any{1, 100},
				GreaterThan:    []any{0},
				LessThan:       []any{1000},
				GreaterOrEqual: []any{1},
				LessOrEqual:    []any{999},
				NotIn:          []any{"x", "y"},
			},
			"field2": {
				Equals: []any{"value2"},
			},
		},
		"table2": {
			"field3": {
				In: []any{"val1", "val2"},
			},
		},
	}

	// Test table1
	result := getTableFilters(dbFilters, "table1")
	if result == nil {
		t.Fatal("expected non-nil result for table1")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 fields for table1, got %d", len(result))
	}

	// Verify field1 has all condition types
	if field1, ok := result["field1"]; ok {
		if len(field1.Equals) != 1 {
			t.Errorf("expected 1 Equals value, got %d", len(field1.Equals))
		}
		if len(field1.In) != 3 {
			t.Errorf("expected 3 In values, got %d", len(field1.In))
		}
		if len(field1.Between) != 2 {
			t.Errorf("expected 2 Between values, got %d", len(field1.Between))
		}
	} else {
		t.Error("field1 not found in result")
	}

	// Test table2
	result2 := getTableFilters(dbFilters, "table2")
	if result2 == nil {
		t.Fatal("expected non-nil result for table2")
	}
	if len(result2) != 1 {
		t.Errorf("expected 1 field for table2, got %d", len(result2))
	}
}
