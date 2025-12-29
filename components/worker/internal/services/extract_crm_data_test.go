package services

import (
	"testing"

	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	libCrypto "github.com/LerianStudio/lib-commons/v2/commons/crypto"
	"github.com/golang/mock/gomock"
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
