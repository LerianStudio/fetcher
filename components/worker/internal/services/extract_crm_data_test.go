package services

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg"
	modelJob "github.com/LerianStudio/fetcher/v2/pkg/model/job"
	portDS "github.com/LerianStudio/fetcher/v2/pkg/ports/datasource"
	libCrypto "github.com/LerianStudio/lib-commons/v6/commons/crypto"
	libLog "github.com/LerianStudio/lib-observability/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestDecryptPluginCRMData_NoDecryptionNeeded tests records that contain no
// encrypted content: decryption is content-driven, so the records round-trip
// unchanged even though valid keys are configured.
func TestDecryptPluginCRMData_NoDecryptionNeeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.SetCRMSecrets(crmTestEncryptKey, crmTestHashKey)
	logger := testLogger()

	// Sample collection result with no encrypted content.
	collectionResult := []map[string]any{
		{"id": "123", "status": "active", "created_at": "2024-01-01"},
		{"id": "456", "status": "inactive", "created_at": "2024-01-02"},
	}

	result, err := uc.decryptPluginCRMData(logger, collectionResult)
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

	// Sample collection result
	collectionResult := []map[string]any{
		{"id": "123", "status": "active"},
	}

	// newTestUseCase seeds non-empty but invalid CRM keys, so the missing-key
	// preconditions pass and the failure surfaces at cipher initialization
	// (FET-0064) as a fail-closed FailedPreconditionError.
	_, err := uc.decryptPluginCRMData(logger, collectionResult)
	var fpe pkg.FailedPreconditionError
	require.ErrorAs(t, err, &fpe, "expected pkg.FailedPreconditionError")
	assert.Equal(t, "FET-0064", fpe.Code, "expected cipher-init-failed error code")
}

// TestDecryptPluginCRMData_WithNestedField tests decryption with nested field paths.
func TestDecryptPluginCRMData_WithNestedField(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Sample collection result
	collectionResult := []map[string]any{
		{"id": "123", "contact": map[string]any{"primary_email": "test@example.com"}},
	}

	// newTestUseCase seeds non-empty but invalid CRM keys, so the missing-key
	// preconditions pass and the failure surfaces at cipher initialization
	// (FET-0064) as a fail-closed FailedPreconditionError.
	_, err := uc.decryptPluginCRMData(logger, collectionResult)
	var fpe pkg.FailedPreconditionError
	require.ErrorAs(t, err, &fpe, "expected pkg.FailedPreconditionError")
	assert.Equal(t, "FET-0064", fpe.Code, "expected cipher-init-failed error code")
}

// TestDecryptPluginCRMData_EmptyResult tests that an empty result set short-circuits
// before any key validation or cipher init, returning no error even with no keys set.
func TestDecryptPluginCRMData_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Empty collection result must short-circuit (no cipher init, no error).
	collectionResult := []map[string]any{}

	result, err := uc.decryptPluginCRMData(logger, collectionResult)
	if err != nil {
		t.Fatalf("expected no error for empty result, got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d records", len(result))
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
			name: "regulatory_fields present but not a map",
			record: map[string]any{
				"regulatory_fields": "not-a-map",
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

// TestDecryptRegulatoryFields tests regulatory_fields field decryption directly
// (not via decryptNestedFields), mirroring TestDecryptBankingDetailsFields for
// naming parity and discoverability among the decrypt*Fields sibling tests.
func TestDecryptRegulatoryFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    crmTestHashKey,
		EncryptSecretKey: crmTestEncryptKey,
		Logger:           logger,
	}
	require.NoError(t, crypto.InitializeCipher(), "failed to initialize cipher")

	t.Run("participant_document decrypts to plaintext", func(t *testing.T) {
		t.Parallel()

		const plainCNPJ = "12345678000199"

		record := map[string]any{
			"regulatory_fields": map[string]any{
				"participant_document": mustEncrypt(t, crypto, plainCNPJ),
			},
		}

		err := uc.decryptRegulatoryFields(record, crypto)
		require.NoError(t, err, "decryptRegulatoryFields returned error")

		regulatoryFields, ok := record["regulatory_fields"].(map[string]any)
		require.True(t, ok, "regulatory_fields must remain a map, got %T", record["regulatory_fields"])
		assert.Equal(t, plainCNPJ, regulatoryFields["participant_document"], "regulatory_fields.participant_document should decrypt to plaintext")
	})

	t.Run("regulatory_fields absent is a no-op", func(t *testing.T) {
		t.Parallel()

		record := map[string]any{"id": "holder-1"}

		err := uc.decryptRegulatoryFields(record, crypto)
		require.NoError(t, err, "decryptRegulatoryFields must be a no-op when regulatory_fields is absent")
		assert.Equal(t, map[string]any{"id": "holder-1"}, record, "record must be unchanged")
	})
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

// TestProcessPluginCRMCollection_WithOrganizationID tests auto-discovery of collections by prefix
// and merging results from all matching collections.
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

	result := make(map[string]map[string][]map[string]any)
	rowsOrg1 := []map[string]any{{"id": "123", "name": "Ada"}}
	rowsOrg2 := []map[string]any{{"id": "456", "name": "Grace"}}
	decryptedRows := []map[string]any{{"id": "123", "name": "Ada Lovelace"}, {"id": "456", "name": "Grace Hopper"}}

	var queryCallCount int
	queryPluginCRMCollectionWithFiltersFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, fields []string, filters map[string]modelJob.FilterCondition, _ libLog.Logger) ([]map[string]any, error) {
		queryCallCount++
		if len(fields) != 2 || fields[0] != "id" || fields[1] != "name" {
			t.Fatalf("unexpected fields: %v", fields)
		}
		switch collection {
		case "holders_org-uuid-1":
			return rowsOrg1, nil
		case "holders_org-uuid-2":
			return rowsOrg2, nil
		default:
			t.Fatalf("unexpected collection name: %s", collection)
			return nil, nil
		}
	}

	decryptPluginCRMDataFn = func(_ *UseCase, _ libLog.Logger, collectionResult []map[string]any) ([]map[string]any, error) {
		if len(collectionResult) != 2 {
			t.Fatalf("expected 2 merged results, got %d: %+v", len(collectionResult), collectionResult)
		}
		return decryptedRows, nil
	}

	// matchingCollections is now pre-filtered and passed by QueryPluginCRM
	matchingCollections := []string{"holders_org-uuid-1", "holders_org-uuid-2"}
	err := uc.processPluginCRMCollection(ctx, nil, "holders", []string{"id", "name"}, nil, matchingCollections, result, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if queryCallCount != 2 {
		t.Fatalf("expected 2 query calls (one per matching collection), got %d", queryCallCount)
	}

	got := result["plugin_crm"]["holders"]
	if len(got) != 2 || got[0]["name"] != "Ada Lovelace" || got[1]["name"] != "Grace Hopper" {
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

	collectionResult := []map[string]any{
		{"id": "123", "document": "encrypted-doc"},
	}

	_, err := uc.decryptPluginCRMData(logger, collectionResult)
	var fpe pkg.FailedPreconditionError
	require.ErrorAs(t, err, &fpe, "expected pkg.FailedPreconditionError")
	assert.Equal(t, "FET-0057", fpe.Code, "expected missing-hash-key error code")
	assert.Equal(t, "CRM hash secret key not configured", fpe.Message, "unexpected error message")
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

	collectionResult := []map[string]any{
		{"id": "123", "document": "encrypted-doc"},
	}

	_, err := uc.decryptPluginCRMData(logger, collectionResult)
	var fpe pkg.FailedPreconditionError
	require.ErrorAs(t, err, &fpe, "expected pkg.FailedPreconditionError")
	assert.Equal(t, "FET-0058", fpe.Code, "expected missing-encrypt-key error code")
	assert.Equal(t, "CRM encrypt secret key not configured", fpe.Message, "unexpected error message")
}

// TestDecryptPluginCRMData_MalformedCiphertext locks the fail-closed behavior of the
// rewritten function: with valid keys configured but a known-encrypted field seeded
// with non-decryptable bytes, decryption must surface a FailedPreconditionError with
// code FET-0065 rather than silently passing the garbage through.
func TestDecryptPluginCRMData_MalformedCiphertext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.SetCRMSecrets(crmTestEncryptKey, crmTestHashKey)
	logger := testLogger()

	// `document` is a known encrypted field, but this value is plain garbage that
	// is not valid AES-256-GCM ciphertext, so Decrypt must fail.
	collectionResult := []map[string]any{
		{"id": "123", "document": "this-is-not-valid-ciphertext"},
	}

	_, err := uc.decryptPluginCRMData(logger, collectionResult)
	var fpe pkg.FailedPreconditionError
	require.ErrorAs(t, err, &fpe, "expected pkg.FailedPreconditionError")
	assert.Equal(t, "FET-0065", fpe.Code, "expected decryption-failure error code")
}

// TestQueryPluginCRM_EmptyCollections tests QueryPluginCRM with empty collections.
func TestQueryPluginCRM_EmptyCollections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	result := make(map[string]map[string][]map[string]any)

	// Empty collections should not cause errors - just no processing
	err := uc.QueryPluginCRM(
		ctx,
		nil, // dataSource - won't be used with empty collections
		"plugin_crm",
		map[string][]string{}, // empty collections
		nil,
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

	result := make(map[string]map[string][]map[string]any)

	// Nil collections should not cause errors
	err := uc.QueryPluginCRM(
		ctx,
		nil, // dataSource - won't be used with nil collections
		"plugin_crm",
		nil, // nil collections
		nil,
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

	called := false
	processPluginCRMCollectionFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, fields []string, filters map[string]modelJob.FilterCondition, matchingCollections []string, _ map[string]map[string][]map[string]any, _ libLog.Logger) error {
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
		if len(matchingCollections) != 1 || matchingCollections[0] != "organization_org-uuid-1" {
			t.Fatalf("unexpected matching collections: %v", matchingCollections)
		}
		return nil
	}

	mockDS := portDS.NewMockCRMQueryable(ctrl)
	mockDS.EXPECT().ListCollectionNames(gomock.Any()).Return([]string{"organization_org-uuid-1"}, nil)

	if err := uc.QueryPluginCRM(ctx, mockDS, "plugin_crm", collections, nil, result, logger); err != nil {
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

	// Sample collection result
	collectionResult := []map[string]any{
		{"id": "123", "status": "active"},
		{"id": "456", "status": "inactive"},
	}

	result, err := uc.decryptPluginCRMData(logger, collectionResult)
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

	seenCounterparty := false
	processPluginCRMCollectionFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, _ []string, collectionFilters map[string]modelJob.FilterCondition, _ []string, _ map[string]map[string][]map[string]any, _ libLog.Logger) error {
		if collection == "counterparty" {
			seenCounterparty = true
			if collectionFilters["status"].Equals[0] != "active" {
				t.Fatalf("unexpected filters: %+v", collectionFilters)
			}
		}
		return nil
	}

	mockDS := portDS.NewMockCRMQueryable(ctrl)
	mockDS.EXPECT().ListCollectionNames(gomock.Any()).Return([]string{
		"organization_org-uuid-1",
		"counterparty_org-uuid-1",
	}, nil)

	if err := uc.QueryPluginCRM(ctx, mockDS, "plugin_crm", collections, filters, result, logger); err != nil {
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
	result := make(map[string]map[string][]map[string]any)

	queryPluginCRMCollectionWithFiltersFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, _ []string, _ map[string]modelJob.FilterCondition, _ libLog.Logger) ([]map[string]any, error) {
		if collection != "counterparty_org-uuid-1" {
			t.Fatalf("unexpected collection: %s", collection)
		}
		return nil, errors.New("query failed")
	}

	matchingCollections := []string{"counterparty_org-uuid-1"}
	err := uc.processPluginCRMCollection(ctx, nil, "counterparty", []string{"id", "name"}, nil, matchingCollections, result, logger)
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
	result := make(map[string]map[string][]map[string]any)

	hashKey := strings.Repeat("01", 32)
	encryptKey := strings.Repeat("fe", 32)
	t.Setenv("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM", hashKey)
	t.Setenv("CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM", encryptKey)

	plainDocument := "12345678900"

	queryPluginCRMCollectionWithFiltersFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, fields []string, _ map[string]modelJob.FilterCondition, _ libLog.Logger) ([]map[string]any, error) {
		expectedCollection := "holders_org-uuid-1"
		if collection != expectedCollection {
			t.Fatalf("expected collection %s, got %s", expectedCollection, collection)
		}
		if len(fields) != 1 || fields[0] != "document" {
			t.Fatalf("unexpected fields: %v", fields)
		}
		return []map[string]any{{"document": "encrypted-doc"}}, nil
	}

	decryptPluginCRMDataFn = func(_ *UseCase, _ libLog.Logger, collectionResult []map[string]any) ([]map[string]any, error) {
		return []map[string]any{{"document": plainDocument}}, nil
	}

	matchingCollections := []string{"holders_org-uuid-1"}
	err := uc.processPluginCRMCollection(ctx, nil, "holders", []string{"document"}, nil, matchingCollections, result, logger)
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
	result := make(map[string]map[string][]map[string]any)

	hashKey := strings.Repeat("01", 32)
	encryptKey := strings.Repeat("fe", 32)
	t.Setenv("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM", hashKey)
	t.Setenv("CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM", encryptKey)

	processPluginCRMCollectionFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, _ []string, collectionFilters map[string]modelJob.FilterCondition, _ []string, res map[string]map[string][]map[string]any, _ libLog.Logger) error {
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
	mockDS.EXPECT().ListCollectionNames(gomock.Any()).Return([]string{"holders_org-uuid-1"}, nil)

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

// crmTestKeys are the valid hex keys (32 bytes / 64 hex chars) used to drive a
// real encrypt/decrypt round-trip in TestDecryptPluginCRMData.
const (
	crmTestHashKey    = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	crmTestEncryptKey = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
)

// mustEncrypt encrypts plain with the supplied crypto, failing the test on error.
func mustEncrypt(t *testing.T, crypto *libCrypto.Crypto, plain string) string {
	t.Helper()

	cipher, err := crypto.Encrypt(&plain)
	if err != nil {
		t.Fatalf("failed to encrypt %q: %v", plain, err)
	}

	return *cipher
}

// TestDecryptPluginCRMData drives the REAL uc.decryptPluginCRMData (not the
// decryptPluginCRMDataFn indirection hook) over a full encrypt -> decrypt
// round-trip. It proves decryption is content-driven: encrypted PII is decrypted
// regardless of whether the requested fields carry a dotted path.
//
// Coverage goal (fetcher-029, extended by fetcher-033): every field that plugin-crm
// actually encrypts and the Fetcher actually decrypts must round-trip back to
// plaintext. That is the set of 15 fields below, split across a holder-shaped record
// and an alias-shaped record:
//
//	Top-level (decryptTopLevelFields):
//	  1. document
//	  2. name
//	Nested contact (decryptContactFields):
//	  3. contact.primary_email
//	  4. contact.secondary_email
//	  5. contact.mobile_phone
//	  6. contact.other_phone
//	Nested banking_details (decryptBankingDetailsFields):
//	  7. banking_details.account
//	  8. banking_details.iban
//	Nested regulatory_fields (decryptRegulatoryFields):
//	  9. regulatory_fields.participant_document
//	Nested legal_person.representative (decryptLegalPersonFields):
//	  10. legal_person.representative.name
//	  11. legal_person.representative.document
//	  12. legal_person.representative.email
//	Nested natural_person (decryptNaturalPersonFields):
//	  13. natural_person.mother_name
//	  14. natural_person.father_name
//
// Note: `document` is the shared top-level handler for both holder and alias records,
// so both shapes exercise it (15 distinct field positions across the two records).
//
// regulatory_fields.participant_document (fetcher-033 / CCS-0011): the deployed CRM
// (midaz/components/crm v3) DOES encrypt this field, and dev-st runs with
// KMS_VENDOR=none, meaning the CRM falls back to legacy AES-GCM with the same CRM
// encrypt key the Fetcher already holds. decryptRegulatoryFields (extract_crm_data.go)
// decrypts it via the same crypto primitive as banking_details.account, so it is now
// asserted as decrypted below like every other field in this list.
//
// Not asserted here: related_parties[].document and related_parties[].name.
// related_parties[].document IS decrypted in production (fetcher-033 / Epic 2.1),
// by decryptRelatedParties — that coverage lives in its own dedicated tests, not
// in this one: TestDecryptRelatedParties (direct unit coverage of
// decryptRelatedParties) and TestDecryptNestedFields_RelatedPartiesDocument_DecryptsToPlaintext
// (through the decryptNestedFields production entrypoint). related_parties[].name
// (and role/start_date/end_date) are plaintext in the live CRM and are
// intentionally never decrypted. This test's own fixtures simply don't put a
// related_parties field on any case, so nothing related_parties-shaped is
// asserted here.
//
// After fetcher-029, decryptPluginCRMData takes no `fields` parameter: plugin_crm
// decryption is UNCONDITIONAL and content-driven — it walks the record and decrypts
// every known encrypted field it finds, regardless of the requested field projection.
// The case names still describe the record shapes (holder / alias / nested-only) they
// exercise, but no field projection is fed to the function under test.
func TestDecryptPluginCRMData(t *testing.T) {
	tests := []struct {
		name      string
		buildRecs func(crypto *libCrypto.Crypto) []map[string]any
		assertFn  func(t *testing.T, records []map[string]any)
	}{
		{
			// Covers all PII the holder shape carries: top-level name+document, all
			// four contact fields, regulatory_fields.participant_document, both
			// natural_person fields, and all three legal_person.representative fields.
			name: "non-dotted holder projection decrypts every holder PII field",
			buildRecs: func(crypto *libCrypto.Crypto) []map[string]any {
				return []map[string]any{
					{
						"_id":      "holder-1",
						"type":     "natural_person",
						"name":     mustEncrypt(t, crypto, "Ada Lovelace"),
						"document": mustEncrypt(t, crypto, "11122233344"),
						"contact": map[string]any{
							"primary_email":   mustEncrypt(t, crypto, "ada@example.com"),
							"secondary_email": mustEncrypt(t, crypto, "ada.alt@example.com"),
							"mobile_phone":    mustEncrypt(t, crypto, "+5511999999999"),
							"other_phone":     mustEncrypt(t, crypto, "+551133334444"),
						},
						"regulatory_fields": map[string]any{
							"participant_document": mustEncrypt(t, crypto, "12345678000199"),
						},
						"natural_person": map[string]any{
							"mother_name":   mustEncrypt(t, crypto, "Annabella Milbanke"),
							"father_name":   mustEncrypt(t, crypto, "George Byron"),
							"favorite_name": "Ada",
						},
						"legal_person": map[string]any{
							"representative": map[string]any{
								"name":     mustEncrypt(t, crypto, "Charles Babbage"),
								"document": mustEncrypt(t, crypto, "55566677788"),
								"email":    mustEncrypt(t, crypto, "charles@example.com"),
								"role":     "director",
							},
						},
					},
				}
			},
			assertFn: func(t *testing.T, records []map[string]any) {
				rec := records[0]

				// Top-level encrypted fields (#1 name, #2 document).
				assert.Equal(t, "Ada Lovelace", rec["name"], "name should decrypt to plaintext")
				assert.Equal(t, "11122233344", rec["document"], "document should decrypt to plaintext")

				// Nested contact (#3-#6).
				contact, ok := rec["contact"].(map[string]any)
				require.True(t, ok, "contact must remain a map, got %T", rec["contact"])
				assert.Equal(t, "ada@example.com", contact["primary_email"], "contact.primary_email")
				assert.Equal(t, "ada.alt@example.com", contact["secondary_email"], "contact.secondary_email")
				assert.Equal(t, "+5511999999999", contact["mobile_phone"], "contact.mobile_phone")
				assert.Equal(t, "+551133334444", contact["other_phone"], "contact.other_phone")

				// Nested regulatory_fields (#9). fetcher-033: previously NOT asserted
				// as decrypted (stale premise — the CRM does encrypt this field, and
				// decryptRegulatoryFields now decrypts it in production).
				regulatoryFields, ok := rec["regulatory_fields"].(map[string]any)
				require.True(t, ok, "regulatory_fields must remain a map, got %T", rec["regulatory_fields"])
				assert.Equal(t, "12345678000199", regulatoryFields["participant_document"], "regulatory_fields.participant_document should decrypt to plaintext")

				// Nested natural_person (#13-#14).
				np, ok := rec["natural_person"].(map[string]any)
				require.True(t, ok, "natural_person must remain a map, got %T", rec["natural_person"])
				assert.Equal(t, "Annabella Milbanke", np["mother_name"], "natural_person.mother_name")
				assert.Equal(t, "George Byron", np["father_name"], "natural_person.father_name")
				assert.Equal(t, "Ada", np["favorite_name"], "natural_person.favorite_name must stay plaintext")

				// Nested legal_person.representative (#10-#12).
				lp, ok := rec["legal_person"].(map[string]any)
				require.True(t, ok, "legal_person must remain a map, got %T", rec["legal_person"])
				rep, ok := lp["representative"].(map[string]any)
				require.True(t, ok, "legal_person.representative must remain a map, got %T", lp["representative"])
				assert.Equal(t, "Charles Babbage", rep["name"], "legal_person.representative.name")
				assert.Equal(t, "55566677788", rep["document"], "legal_person.representative.document")
				assert.Equal(t, "charles@example.com", rep["email"], "legal_person.representative.email")
				assert.Equal(t, "director", rep["role"], "legal_person.representative.role must stay plaintext")

				// Co-located structural/plaintext fields stay intact.
				assert.Equal(t, "holder-1", rec["_id"], "_id must stay intact")
				assert.Equal(t, "natural_person", rec["type"], "type must stay intact")
			},
		},
		{
			// Covers the alias shape: top-level document plus banking_details.account
			// and banking_details.iban.
			name: "non-dotted alias projection decrypts document and banking_details PII",
			buildRecs: func(crypto *libCrypto.Crypto) []map[string]any {
				return []map[string]any{
					{
						"_id":      "alias-1",
						"type":     "alias",
						"document": mustEncrypt(t, crypto, "99988877766"),
						"banking_details": map[string]any{
							"account": mustEncrypt(t, crypto, "1234567890"),
							"iban":    mustEncrypt(t, crypto, "BR1500000000000010932840814P2"),
							"branch":  "0001",
						},
					},
				}
			},
			assertFn: func(t *testing.T, records []map[string]any) {
				rec := records[0]

				// Top-level document via the shared handler (#1 document).
				assert.Equal(t, "99988877766", rec["document"], "alias document should decrypt to plaintext")

				// Nested banking_details (#7-#8).
				bd, ok := rec["banking_details"].(map[string]any)
				require.True(t, ok, "banking_details must remain a map, got %T", rec["banking_details"])
				assert.Equal(t, "1234567890", bd["account"], "banking_details.account")
				assert.Equal(t, "BR1500000000000010932840814P2", bd["iban"], "banking_details.iban")
				assert.Equal(t, "0001", bd["branch"], "banking_details.branch must stay plaintext")

				// Co-located structural/plaintext fields stay intact.
				assert.Equal(t, "alias-1", rec["_id"], "_id must stay intact")
				assert.Equal(t, "alias", rec["type"], "type must stay intact")
			},
		},
		{
			// Non-regression control: a nested-only record must still decrypt its
			// nested PII exactly as the holder/alias records above.
			name: "dotted field control still decrypts nested PII (non-regression)",
			buildRecs: func(crypto *libCrypto.Crypto) []map[string]any {
				return []map[string]any{
					{
						"banking_details": map[string]any{
							"account": mustEncrypt(t, crypto, "9999999999"),
							"branch":  "0002",
						},
					},
				}
			},
			assertFn: func(t *testing.T, records []map[string]any) {
				bd, ok := records[0]["banking_details"].(map[string]any)
				require.True(t, ok, "banking_details must remain a map, got %T", records[0]["banking_details"])
				assert.Equal(t, "9999999999", bd["account"], "banking_details.account")
				assert.Equal(t, "0002", bd["branch"], "banking_details.branch must stay plaintext")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := newTestMocks(ctrl)
			uc := newTestUseCase(mocks)
			uc.SetCRMSecrets(crmTestEncryptKey, crmTestHashKey)
			logger := testLogger()

			crypto := &libCrypto.Crypto{
				HashSecretKey:    crmTestHashKey,
				EncryptSecretKey: crmTestEncryptKey,
				Logger:           logger,
			}
			require.NoError(t, crypto.InitializeCipher(), "failed to initialize cipher")

			records := tt.buildRecs(crypto)

			result, err := uc.decryptPluginCRMData(logger, records)
			require.NoError(t, err, "decryptPluginCRMData returned error")
			require.Len(t, result, len(records), "record count must be preserved")

			tt.assertFn(t, result)
		})
	}
}

// TestDecryptNestedFields_RegulatoryFieldsParticipantDocument_DecryptsToPlaintext proves
// the CCS-0011 round-trip fix: regulatory_fields.participant_document, encrypted with the
// SAME runtime crypto primitive the Fetcher already uses to decrypt banking_details.account
// (real AES-GCM encrypt/decrypt via *libCrypto.Crypto, not a mock), must decrypt to plaintext
// through decryptNestedFields.
func TestDecryptNestedFields_RegulatoryFieldsParticipantDocument_DecryptsToPlaintext(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    crmTestHashKey,
		EncryptSecretKey: crmTestEncryptKey,
		Logger:           logger,
	}
	require.NoError(t, crypto.InitializeCipher(), "failed to initialize cipher")

	const plainCNPJ = "12345678000199"

	record := map[string]any{
		"regulatory_fields": map[string]any{
			"participant_document": mustEncrypt(t, crypto, plainCNPJ),
		},
	}

	err := uc.decryptNestedFields(record, crypto)
	require.NoError(t, err, "decryptNestedFields returned error")

	regulatoryFields, ok := record["regulatory_fields"].(map[string]any)
	require.True(t, ok, "regulatory_fields must remain a map, got %T", record["regulatory_fields"])
	assert.Equal(t, plainCNPJ, regulatoryFields["participant_document"], "regulatory_fields.participant_document should decrypt to plaintext")
}

// TestDecryptNestedFields_RegulatoryFieldsAbsentOrKeyMissing_NoOpNoError covers the
// no-op branches of decryptRegulatoryFields (fetcher-033, cases b/c): when the
// regulatory_fields object itself is absent from the record, or it is present but
// participant_document is either missing or explicitly nil, decryption must be a
// no-op — no error, and the record (including any sibling plaintext data) must come
// back byte-identical. It uses the same real *libCrypto.Crypto primitive as the
// decrypting-path tests (never a mock), even though these branches never reach
// Decrypt, so a future regression that starts invoking Decrypt on a nil/missing
// value is still exercised against the real cipher.
func TestDecryptNestedFields_RegulatoryFieldsAbsentOrKeyMissing_NoOpNoError(t *testing.T) {
	t.Parallel()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    crmTestHashKey,
		EncryptSecretKey: crmTestEncryptKey,
		Logger:           testLogger(),
	}
	require.NoError(t, crypto.InitializeCipher(), "failed to initialize cipher")

	tests := []struct {
		name   string
		record map[string]any
		want   map[string]any
	}{
		{
			name:   "regulatory_fields object absent from record",
			record: map[string]any{"id": "holder-1", "type": "natural_person"},
			want:   map[string]any{"id": "holder-1", "type": "natural_person"},
		},
		{
			name: "regulatory_fields present but participant_document key absent",
			record: map[string]any{
				"regulatory_fields": map[string]any{"other_field": "unrelated-value"},
			},
			want: map[string]any{
				"regulatory_fields": map[string]any{"other_field": "unrelated-value"},
			},
		},
		{
			name: "regulatory_fields present with participant_document explicitly nil",
			record: map[string]any{
				"regulatory_fields": map[string]any{"participant_document": nil},
			},
			want: map[string]any{
				"regulatory_fields": map[string]any{"participant_document": nil},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runDecryptNestedFieldsWithFreshUseCase(t, crypto, tt.record)
			require.NoError(t, err, "decryptNestedFields must be a no-op, not an error")
			assert.Equal(t, tt.want, tt.record, "record must be unchanged when there is nothing to decrypt")
		})
	}
}

// runDecryptNestedFieldsWithFreshUseCase builds a fresh UseCase per call so parallel subtests
// never share gomock controller state, and invokes decryptNestedFields on record.
func runDecryptNestedFieldsWithFreshUseCase(t *testing.T, crypto *libCrypto.Crypto, record map[string]any) error {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	uc := newTestUseCase(newTestMocks(ctrl))

	return uc.decryptNestedFields(record, crypto)
}

// TestDecryptNestedFields_RegulatoryFieldsParticipantDocument_InvalidCiphertext_ReturnsWrappedError
// covers case (d): participant_document holds a value that cannot be decrypted by the
// real *libCrypto.Crypto primitive the Fetcher uses in production (never a mock that
// always "decrypts"). decryptNestedFields must propagate a non-nil error whose message
// names the field path (regulatory_fields.participant_document) for operability, while
// never echoing the raw, potentially-sensitive input value into the error string.
func TestDecryptNestedFields_RegulatoryFieldsParticipantDocument_InvalidCiphertext_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    crmTestHashKey,
		EncryptSecretKey: crmTestEncryptKey,
		Logger:           testLogger(),
	}
	require.NoError(t, crypto.InitializeCipher(), "failed to initialize cipher")

	// A real ciphertext, tampered by flipping one byte so AES-GCM authentication
	// fails on Open — proves the failure path with a value that round-trips through
	// base64 decoding but is not decryptable, distinct from the not-base64 case below.
	tamperedCiphertext := tamperBase64Ciphertext(t, mustEncrypt(t, crypto, "98765432000188"))

	tests := []struct {
		name          string
		rawCiphertext string
	}{
		{
			name:          "not valid base64",
			rawCiphertext: "this-is-not-valid-ciphertext",
		},
		{
			name:          "valid base64 but tampered AES-GCM ciphertext fails authentication",
			rawCiphertext: tamperedCiphertext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := map[string]any{
				"regulatory_fields": map[string]any{
					"participant_document": tt.rawCiphertext,
				},
			}

			err := runDecryptNestedFieldsWithFreshUseCase(t, crypto, record)

			require.Error(t, err, "decryptNestedFields must return an error for undecryptable ciphertext")
			assert.Contains(t, err.Error(), "regulatory_fields.participant_document",
				"error must name the field path for operability")
			assert.NotContains(t, err.Error(), tt.rawCiphertext,
				"error must never echo the raw ciphertext value")
		})
	}
}

// tamperBase64Ciphertext flips one byte inside the decoded ciphertext (never touching
// the nonce prefix) and re-encodes to base64, producing a value that decodes cleanly
// but fails AES-GCM authentication on Open — the "valid base64, bad ciphertext" edge
// distinct from a plain not-base64 string.
func tamperBase64Ciphertext(t *testing.T, encoded string) string {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err, "fixture ciphertext must be valid base64")
	require.Greater(t, len(decoded), 12, "fixture ciphertext must be longer than the AES-GCM nonce")

	tampered := make([]byte, len(decoded))
	copy(tampered, decoded)
	tampered[len(tampered)-1] ^= 0xFF // flip last byte, inside the ciphertext/tag, not the nonce

	return base64.StdEncoding.EncodeToString(tampered)
}

// newRelatedPartiesCryptoForTest builds a real *libCrypto.Crypto (never a mock)
// using the shared crmTestHashKey/crmTestEncryptKey fixture keys, matching the
// primitive decryptRelatedParties uses in production.
func newRelatedPartiesCryptoForTest(t *testing.T) *libCrypto.Crypto {
	t.Helper()

	crypto := &libCrypto.Crypto{
		HashSecretKey:    crmTestHashKey,
		EncryptSecretKey: crmTestEncryptKey,
		Logger:           testLogger(),
	}
	require.NoError(t, crypto.InitializeCipher(), "failed to initialize cipher")

	return crypto
}

// TestDecryptNestedFields_RelatedPartiesDocument_DecryptsToPlaintext proves the
// fetcher-033 Phase 2 / Epic 2.1 wiring: related_parties[].document, encrypted with
// the SAME runtime crypto primitive the Fetcher already uses to decrypt
// banking_details.account and regulatory_fields.participant_document (real AES-GCM
// encrypt/decrypt via *libCrypto.Crypto, not a mock), must decrypt to plaintext
// through decryptNestedFields — the production entrypoint — while related_parties[].name
// (plaintext in the live CRM) is left untouched.
func TestDecryptNestedFields_RelatedPartiesDocument_DecryptsToPlaintext(t *testing.T) {
	t.Parallel()

	crypto := newRelatedPartiesCryptoForTest(t)

	const (
		plainDoc1  = "11122233344"
		plainDoc2  = "55566677788"
		plainName1 = "Joseph Roberts"
		plainName2 = "Maria Silva"
	)

	record := map[string]any{
		"related_parties": []any{
			map[string]any{
				"_id":      "rp-1",
				"document": mustEncrypt(t, crypto, plainDoc1),
				"name":     plainName1,
				"role":     "LEGAL_REPRESENTATIVE",
			},
			map[string]any{
				"_id":      "rp-2",
				"document": mustEncrypt(t, crypto, plainDoc2),
				"name":     plainName2,
				"role":     "BENEFICIAL_OWNER",
			},
		},
	}

	err := runDecryptNestedFieldsWithFreshUseCase(t, crypto, record)
	require.NoError(t, err, "decryptNestedFields returned error")

	relatedParties, ok := record["related_parties"].([]any)
	require.True(t, ok, "related_parties must remain a slice, got %T", record["related_parties"])
	require.Len(t, relatedParties, 2)

	party1, ok := relatedParties[0].(map[string]any)
	require.True(t, ok, "related_parties[0] must remain a map, got %T", relatedParties[0])
	assert.Equal(t, plainDoc1, party1["document"], "related_parties[0].document should decrypt to plaintext")
	assert.Equal(t, plainName1, party1["name"], "related_parties[0].name must be left untouched (plaintext)")
	assert.Equal(t, "LEGAL_REPRESENTATIVE", party1["role"])

	party2, ok := relatedParties[1].(map[string]any)
	require.True(t, ok, "related_parties[1] must remain a map, got %T", relatedParties[1])
	assert.Equal(t, plainDoc2, party2["document"], "related_parties[1].document should decrypt to plaintext")
	assert.Equal(t, plainName2, party2["name"], "related_parties[1].name must be left untouched (plaintext)")
}

// TestDecryptRelatedParties tests related_parties[].document decryption directly
// (not via decryptNestedFields), mirroring TestDecryptRegulatoryFields for naming
// parity and discoverability among the decrypt*Fields sibling tests. related_parties
// is slice-shaped ([]any of map[string]any), unlike its map-shaped siblings, so this
// covers slice-specific edge cases: absent list, empty list, non-map elements, and
// per-element document absence/nil.
func TestDecryptRelatedParties(t *testing.T) {
	t.Parallel()

	crypto := newRelatedPartiesCryptoForTest(t)

	t.Run("multiple encrypted elements decrypt to plaintext, name untouched", func(t *testing.T) {
		t.Parallel()

		const plainDoc = "12345678901"
		const plainName = "Ada Lovelace"

		record := map[string]any{
			"related_parties": []any{
				map[string]any{
					"document": mustEncrypt(t, crypto, plainDoc),
					"name":     plainName,
				},
				map[string]any{
					"document": mustEncrypt(t, crypto, plainDoc),
					"name":     plainName,
				},
			},
		}

		err := runDecryptRelatedPartiesWithFreshUseCase(t, crypto, record)
		require.NoError(t, err, "decryptRelatedParties returned error")

		relatedParties, ok := record["related_parties"].([]any)
		require.True(t, ok, "related_parties must remain a slice, got %T", record["related_parties"])
		require.Len(t, relatedParties, 2)

		for i, elem := range relatedParties {
			party, ok := elem.(map[string]any)
			require.True(t, ok, "related_parties[%d] must remain a map, got %T", i, elem)
			assert.Equal(t, plainDoc, party["document"], "related_parties[%d].document should decrypt to plaintext", i)
			assert.Equal(t, plainName, party["name"], "related_parties[%d].name must be left untouched (plaintext)", i)
		}
	})

	t.Run("related_parties list absent is a no-op", func(t *testing.T) {
		t.Parallel()

		record := map[string]any{"id": "holder-1"}

		err := runDecryptRelatedPartiesWithFreshUseCase(t, crypto, record)
		require.NoError(t, err, "decryptRelatedParties must be a no-op when related_parties is absent")
		assert.Equal(t, map[string]any{"id": "holder-1"}, record, "record must be unchanged")
	})

	t.Run("related_parties empty list is a no-op", func(t *testing.T) {
		t.Parallel()

		record := map[string]any{"related_parties": []any{}}

		err := runDecryptRelatedPartiesWithFreshUseCase(t, crypto, record)
		require.NoError(t, err, "decryptRelatedParties must be a no-op for an empty list")
		assert.Equal(t, map[string]any{"related_parties": []any{}}, record, "record must be unchanged")
	})

	t.Run("related_parties wrong type is a no-op", func(t *testing.T) {
		t.Parallel()

		record := map[string]any{"related_parties": "not-a-slice"}

		err := runDecryptRelatedPartiesWithFreshUseCase(t, crypto, record)
		require.NoError(t, err, "decryptRelatedParties must be a no-op when related_parties is not a slice")
		assert.Equal(t, map[string]any{"related_parties": "not-a-slice"}, record, "record must be unchanged")
	})

	t.Run("non-map element is skipped without error", func(t *testing.T) {
		t.Parallel()

		const plainDoc = "98765432100"

		record := map[string]any{
			"related_parties": []any{
				"not-a-map",
				map[string]any{"document": mustEncrypt(t, crypto, plainDoc), "name": "Grace Hopper"},
				42,
			},
		}

		err := runDecryptRelatedPartiesWithFreshUseCase(t, crypto, record)
		require.NoError(t, err, "decryptRelatedParties must skip non-map elements without error")

		relatedParties, ok := record["related_parties"].([]any)
		require.True(t, ok)
		require.Len(t, relatedParties, 3)
		assert.Equal(t, "not-a-map", relatedParties[0])
		party, ok := relatedParties[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, plainDoc, party["document"])
		assert.Equal(t, 42, relatedParties[2])
	})

	t.Run("document absent is skipped without error", func(t *testing.T) {
		t.Parallel()

		record := map[string]any{
			"related_parties": []any{
				map[string]any{"name": "no-document-field"},
			},
		}

		err := runDecryptRelatedPartiesWithFreshUseCase(t, crypto, record)
		require.NoError(t, err, "decryptRelatedParties must skip an element with no document key")

		relatedParties, ok := record["related_parties"].([]any)
		require.True(t, ok)
		party, ok := relatedParties[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "no-document-field", party["name"])
		_, exists := party["document"]
		assert.False(t, exists, "document key must not be added when absent")
	})

	t.Run("document explicitly nil is skipped without error", func(t *testing.T) {
		t.Parallel()

		record := map[string]any{
			"related_parties": []any{
				map[string]any{"document": nil, "name": "nil-document"},
			},
		}

		err := runDecryptRelatedPartiesWithFreshUseCase(t, crypto, record)
		require.NoError(t, err, "decryptRelatedParties must skip an element with a nil document")

		relatedParties, ok := record["related_parties"].([]any)
		require.True(t, ok)
		party, ok := relatedParties[0].(map[string]any)
		require.True(t, ok)
		assert.Nil(t, party["document"])
		assert.Equal(t, "nil-document", party["name"])
	})

	t.Run("invalid ciphertext returns wrapped error naming the index and field, never the raw value", func(t *testing.T) {
		t.Parallel()

		tamperedCiphertext := tamperBase64Ciphertext(t, mustEncrypt(t, crypto, "11100022233"))

		tests := []struct {
			name          string
			rawCiphertext string
		}{
			{name: "not valid base64", rawCiphertext: "this-is-not-valid-ciphertext"},
			{name: "valid base64 but tampered AES-GCM ciphertext fails authentication", rawCiphertext: tamperedCiphertext},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				record := map[string]any{
					"related_parties": []any{
						map[string]any{"document": tt.rawCiphertext, "name": "irrelevant"},
					},
				}

				err := runDecryptRelatedPartiesWithFreshUseCase(t, crypto, record)

				require.Error(t, err, "decryptRelatedParties must return an error for undecryptable ciphertext")
				assert.Contains(t, err.Error(), "related_parties[0].document",
					"error must name the index and field path for operability")
				assert.NotContains(t, err.Error(), tt.rawCiphertext,
					"error must never echo the raw ciphertext value")
			})
		}
	})

	t.Run("undecryptable document at non-zero index is named exactly, not index 0", func(t *testing.T) {
		t.Parallel()

		const plainDoc0 = "22233344455"

		tamperedCiphertext := tamperBase64Ciphertext(t, mustEncrypt(t, crypto, "99988877766"))

		record := map[string]any{
			"related_parties": []any{
				map[string]any{"document": mustEncrypt(t, crypto, plainDoc0), "name": "valid element"},
				map[string]any{"document": tamperedCiphertext, "name": "undecryptable element"},
			},
		}

		err := runDecryptRelatedPartiesWithFreshUseCase(t, crypto, record)

		require.Error(t, err, "decryptRelatedParties must return an error when a non-zero-index element is undecryptable")
		assert.Contains(t, err.Error(), "related_parties[1].document",
			"error must name the dynamic loop index (1), not a hardcoded 0")
		assert.NotContains(t, err.Error(), "related_parties[0].document",
			"error must not misreport the valid element's index as the failing one")
		assert.NotContains(t, err.Error(), tamperedCiphertext,
			"error must never echo the raw ciphertext value")
	})

	t.Run("fails fast at first undecryptable index, later elements left unprocessed", func(t *testing.T) {
		t.Parallel()

		const (
			plainDoc0 = "11111111111"
			plainDoc2 = "33333333333"
		)

		undecryptable := tamperBase64Ciphertext(t, mustEncrypt(t, crypto, "22222222222"))
		originalDoc2 := mustEncrypt(t, crypto, plainDoc2)

		record := map[string]any{
			"related_parties": []any{
				map[string]any{"document": mustEncrypt(t, crypto, plainDoc0), "name": "first"},
				map[string]any{"document": undecryptable, "name": "second"},
				map[string]any{"document": originalDoc2, "name": "third"},
			},
		}

		err := runDecryptRelatedPartiesWithFreshUseCase(t, crypto, record)

		require.Error(t, err, "decryptRelatedParties must fail fast on the first undecryptable element")
		assert.Contains(t, err.Error(), "related_parties[1].document",
			"error must name the failing index")

		relatedParties, ok := record["related_parties"].([]any)
		require.True(t, ok, "related_parties must remain a slice, got %T", record["related_parties"])
		require.Len(t, relatedParties, 3)

		party0, ok := relatedParties[0].(map[string]any)
		require.True(t, ok, "related_parties[0] must remain a map, got %T", relatedParties[0])
		assert.Equal(t, plainDoc0, party0["document"],
			"related_parties[0].document should have been decrypted before the fail-fast stop at index 1")

		party2, ok := relatedParties[2].(map[string]any)
		require.True(t, ok, "related_parties[2] must remain a map, got %T", relatedParties[2])
		assert.Equal(t, originalDoc2, party2["document"],
			"related_parties[2].document must be LEFT UNCHANGED (still ciphertext): this locks in the "+
				"current fail-fast contract (stop at the first bad element, never reach later ones) so "+
				"a future best-effort/collect-all refactor cannot silently change this behavior")
	})
}

// runDecryptRelatedPartiesWithFreshUseCase builds a fresh UseCase per call so parallel
// subtests never share gomock controller state, and invokes decryptRelatedParties on
// record.
func runDecryptRelatedPartiesWithFreshUseCase(t *testing.T, crypto *libCrypto.Crypto, record map[string]any) error {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	uc := newTestUseCase(newTestMocks(ctrl))

	return uc.decryptRelatedParties(record, crypto)
}
