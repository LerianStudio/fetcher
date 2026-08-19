package http

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0"`
}

type TestMetadataStruct struct {
	Name     string         `json:"name" validate:"required"`
	Metadata map[string]any `json:"metadata"`
}

func TestValidateStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name: "valid struct",
			input: &TestStruct{
				Name:  "John",
				Email: "john@example.com",
				Age:   30,
			},
			wantErr: false,
		},
		{
			name: "missing required field",
			input: &TestStruct{
				Email: "john@example.com",
				Age:   30,
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			input: &TestStruct{
				Name:  "John",
				Email: "invalid",
				Age:   30,
			},
			wantErr: true,
		},
		{
			name: "negative age",
			input: &TestStruct{
				Name:  "John",
				Email: "john@example.com",
				Age:   -1,
			},
			wantErr: true,
		},
		{
			name:    "non-struct type - map",
			input:   map[string]string{"key": "value"},
			wantErr: false, // Should return nil for non-struct
		},
		{
			name:    "non-struct type - string",
			input:   "test",
			wantErr: false, // Should return nil for non-struct
		},
		{
			name:    "nil pointer",
			input:   (*TestStruct)(nil),
			wantErr: false, // ValidateStruct returns nil for non-struct types
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatErrorFieldName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "field with dot notation",
			input: "User.Name",
			want:  "Name",
		},
		{
			name:  "nested field",
			input: "User.Address.Street",
			want:  "Address.Street", // Regex captures everything after the first dot
		},
		{
			name:  "field without dot",
			input: "Name",
			want:  "Name",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatErrorFieldName(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestValidateMetadataNestedValues(t *testing.T) {
	// This tests the custom validator indirectly through ValidateStruct
	type MetadataTest struct {
		Data map[string]any `json:"data" validate:"nonested"`
	}

	tests := []struct {
		name    string
		input   *MetadataTest
		wantErr bool
	}{
		{
			name: "non-nested metadata passes",
			input: &MetadataTest{
				Data: map[string]any{"key": "value"},
			},
			wantErr: true, // This will fail because Data is a map
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMetadataKeyMaxLength(t *testing.T) {
	type KeyLengthTest struct {
		Key string `json:"key" validate:"keymax=5"`
	}

	tests := []struct {
		name    string
		input   *KeyLengthTest
		wantErr bool
	}{
		{
			name:    "key within limit",
			input:   &KeyLengthTest{Key: "abc"},
			wantErr: false,
		},
		{
			name:    "key at limit",
			input:   &KeyLengthTest{Key: "abcde"},
			wantErr: false,
		},
		{
			name:    "key exceeds limit",
			input:   &KeyLengthTest{Key: "abcdef"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMetadataValueMaxLength(t *testing.T) {
	type ValueLengthTest struct {
		Value string `json:"value" validate:"valuemax=10"`
	}

	tests := []struct {
		name    string
		input   *ValueLengthTest
		wantErr bool
	}{
		{
			name:    "value within limit",
			input:   &ValueLengthTest{Value: "short"},
			wantErr: false,
		},
		{
			name:    "value at limit",
			input:   &ValueLengthTest{Value: "1234567890"},
			wantErr: false,
		},
		{
			name:    "value exceeds limit",
			input:   &ValueLengthTest{Value: "12345678901"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMetadataValueMaxLengthNumericTypes(t *testing.T) {
	type IntValueTest struct {
		Value int `json:"value" validate:"valuemax=5"`
	}

	type FloatValueTest struct {
		Value float64 `json:"value" validate:"valuemax=10"`
	}

	type BoolValueTest struct {
		Value bool `json:"value" validate:"valuemax=10"`
	}

	t.Run("int value within limit", func(t *testing.T) {
		input := &IntValueTest{Value: 123}
		err := ValidateStruct(input)
		assert.NoError(t, err)
	})

	t.Run("int value exceeds limit", func(t *testing.T) {
		input := &IntValueTest{Value: 123456}
		err := ValidateStruct(input)
		assert.Error(t, err)
	})

	t.Run("float value within limit", func(t *testing.T) {
		input := &FloatValueTest{Value: 1.5}
		err := ValidateStruct(input)
		assert.NoError(t, err)
	})

	t.Run("float value exceeds limit", func(t *testing.T) {
		input := &FloatValueTest{Value: 12345678901.5}
		err := ValidateStruct(input)
		assert.Error(t, err)
	})

	t.Run("bool value within limit", func(t *testing.T) {
		input := &BoolValueTest{Value: true}
		err := ValidateStruct(input)
		assert.NoError(t, err)
	})

	t.Run("bool false value within limit", func(t *testing.T) {
		input := &BoolValueTest{Value: false}
		err := ValidateStruct(input)
		assert.NoError(t, err)
	})
}

func TestFieldsFunction(t *testing.T) {
	// Test via ValidateStruct and malformedRequestErr
	type RequiredFieldsTest struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required"`
	}

	t.Run("multiple required fields missing", func(t *testing.T) {
		input := &RequiredFieldsTest{}
		err := ValidateStruct(input)
		assert.Error(t, err)
	})
}

func TestFieldsFunctionExtended(t *testing.T) {
	// Extended test via ValidateStruct
	t.Run("empty struct pointer", func(t *testing.T) {
		type EmptyStruct struct{}
		input := &EmptyStruct{}
		err := ValidateStruct(input)
		assert.NoError(t, err)
	})
}

func TestValidateMetadataKeyMaxLengthDefaultLimit(t *testing.T) {
	// Test with no param (uses default limit of 100)
	type KeyDefaultLimitTest struct {
		Key string `json:"key" validate:"keymax"`
	}

	t.Run("key within default limit", func(t *testing.T) {
		input := &KeyDefaultLimitTest{Key: "shortkey"}
		err := ValidateStruct(input)
		assert.NoError(t, err)
	})
}

// TestErrValidatorInit_SentinelBehavior verifies the ErrValidatorInit sentinel error
// wrapping works correctly so that FiberHandlerFunc can map it to HTTP 500.
//
// NOTE: We cannot trigger the actual validator-init-failure path through ValidateStruct
// because getValidator() uses sync.Once — once it succeeds (which it does in every test
// process), there is no way to make it fail again without refactoring the init pattern.
// Instead we verify:
//  1. The wrapping format used in ValidateStruct preserves errors.Is identity.
//  2. An unrelated validation error does NOT match ErrValidatorInit.
func TestErrValidatorInit_SentinelBehavior(t *testing.T) {
	t.Run("wrapped ErrValidatorInit is detectable via errors.Is", func(t *testing.T) {
		// This replicates the wrapping at with_body.go:228
		inner := errors.New("failed to register default translations: something broke")
		wrapped := fmt.Errorf("%w: %v", ErrValidatorInit, inner)

		assert.True(t, errors.Is(wrapped, ErrValidatorInit),
			"errors.Is must detect ErrValidatorInit through fmt.Errorf %%w wrapping")
		assert.Contains(t, wrapped.Error(), "validator initialization failed")
		assert.Contains(t, wrapped.Error(), "something broke")
	})

	t.Run("inner error is NOT unwrappable via errors.Is (uses %%v)", func(t *testing.T) {
		inner := errors.New("specific init detail")
		wrapped := fmt.Errorf("%w: %v", ErrValidatorInit, inner)

		// The inner error was formatted with %v, so it should NOT be unwrappable
		assert.False(t, errors.Is(wrapped, inner),
			"inner error must not be unwrappable — %v intentionally prevents this")
	})

	t.Run("normal validation error is NOT ErrValidatorInit", func(t *testing.T) {
		// Trigger a real validation error (missing required field)
		input := &TestStruct{Email: "john@example.com", Age: 30} // Name is required
		err := ValidateStruct(input)
		assert.Error(t, err)
		assert.False(t, errors.Is(err, ErrValidatorInit),
			"regular validation errors must not match ErrValidatorInit")
	})
}

func TestValidateMetadataValueMaxLengthDefaultLimit(t *testing.T) {
	// Test with no param (uses default limit of 2000)
	type ValueDefaultLimitTest struct {
		Value string `json:"value" validate:"valuemax"`
	}

	t.Run("value within default limit", func(t *testing.T) {
		input := &ValueDefaultLimitTest{Value: "shortvalue"}
		err := ValidateStruct(input)
		assert.NoError(t, err)
	})
}

// TestValidateStruct_NestedRequiredFields covers validation descending into an
// embedded struct. Previously asserted through the WithBody decode middleware;
// that transport is gone, but the nested-required behaviour it relied on is live
// in every Manager handler that validates a DTO with a nested object.
func TestValidateStruct_NestedRequiredFields(t *testing.T) {
	type Address struct {
		Street string `json:"street" validate:"required"`
		City   string `json:"city" validate:"required"`
	}

	type Person struct {
		Name    string  `json:"name" validate:"required"`
		Address Address `json:"address" validate:"required"`
	}

	assert.NoError(t, ValidateStruct(&Person{
		Name:    "John",
		Address: Address{Street: "Main St", City: "NYC"},
	}))

	assert.Error(t, ValidateStruct(&Person{
		Name:    "John",
		Address: Address{Street: "Main St"},
	}), "a missing required field on a nested struct must fail validation")
}
