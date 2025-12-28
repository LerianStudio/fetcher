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
