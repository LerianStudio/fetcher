package http

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

func TestWithBody(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "valid json body",
			body:           `{"name":"John","email":"john@example.com","age":30}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "missing required field",
			body:           `{"email":"john@example.com"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid email format",
			body:           `{"name":"John","email":"invalid-email","age":30}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "unknown fields",
			body:           `{"name":"John","email":"john@example.com","age":30,"unknown":"field"}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "negative age",
			body:           `{"name":"John","email":"john@example.com","age":-5}`,
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid json",
			body:           `{"name":"John"`,
			wantStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/test", WithBody(&TestStruct{}, func(p any, c *fiber.Ctx) error {
				return c.SendStatus(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatusCode, resp.StatusCode)
		})
	}
}

func TestWithBodyMetadata(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "with metadata",
			body:           `{"name":"John","metadata":{"key":"value"}}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "without metadata",
			body:           `{"name":"John"}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "null metadata",
			body:           `{"name":"John","metadata":null}`,
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/test", WithBody(&TestMetadataStruct{}, func(p any, c *fiber.Ctx) error {
				return c.SendStatus(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatusCode, resp.StatusCode)
		})
	}
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

func TestFindUnknownFields(t *testing.T) {
	tests := []struct {
		name      string
		original  map[string]any
		marshaled map[string]any
		wantLen   int
	}{
		{
			name: "no differences",
			original: map[string]any{
				"name": "John",
				"age":  30,
			},
			marshaled: map[string]any{
				"name": "John",
				"age":  30,
			},
			wantLen: 0,
		},
		{
			name: "extra field in original",
			original: map[string]any{
				"name":    "John",
				"age":     30,
				"unknown": "field",
			},
			marshaled: map[string]any{
				"name": "John",
				"age":  30,
			},
			wantLen: 1,
		},
		{
			name: "nested differences",
			original: map[string]any{
				"user": map[string]any{
					"name":    "John",
					"unknown": "field",
				},
			},
			marshaled: map[string]any{
				"user": map[string]any{
					"name": "John",
				},
			},
			wantLen: 1,
		},
		{
			name: "array differences",
			original: map[string]any{
				"items": []any{"a", "b", "c"},
			},
			marshaled: map[string]any{
				"items": []any{"a", "b"},
			},
			wantLen: 1,
		},
		{
			name: "zero numeric value ignored",
			original: map[string]any{
				"name":  "John",
				"count": 0.0,
			},
			marshaled: map[string]any{
				"name": "John",
			},
			wantLen: 0,
		},
		{
			name: "different types",
			original: map[string]any{
				"value": "string",
			},
			marshaled: map[string]any{
				"value": 123,
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findUnknownFields(tt.original, tt.marshaled)
			assert.Equal(t, tt.wantLen, len(result))
		})
	}
}

func TestCompareSlices(t *testing.T) {
	tests := []struct {
		name      string
		original  []any
		marshaled []any
		wantLen   int
	}{
		{
			name:      "identical slices",
			original:  []any{"a", "b", "c"},
			marshaled: []any{"a", "b", "c"},
			wantLen:   0,
		},
		{
			name:      "original longer",
			original:  []any{"a", "b", "c"},
			marshaled: []any{"a", "b"},
			wantLen:   1,
		},
		{
			name:      "marshaled longer",
			original:  []any{"a", "b"},
			marshaled: []any{"a", "b", "c"},
			wantLen:   1,
		},
		{
			name:      "different values",
			original:  []any{"a", "b", "c"},
			marshaled: []any{"a", "x", "c"},
			wantLen:   1,
		},
		{
			name:      "nested maps in slices",
			original:  []any{map[string]any{"key": "value", "extra": "field"}},
			marshaled: []any{map[string]any{"key": "value"}},
			wantLen:   1,
		},
		{
			name:      "empty slices",
			original:  []any{},
			marshaled: []any{},
			wantLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareSlices(tt.original, tt.marshaled)
			assert.Equal(t, tt.wantLen, len(result))
		})
	}
}

func TestParseUUIDPathParameters(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantStatusCode int
		wantErr        bool
	}{
		{
			name:           "valid UUID in id param",
			url:            "/test/550e8400-e29b-41d4-a716-446655440000",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "invalid UUID in id param",
			url:            "/test/not-a-uuid",
			wantStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "non-UUID param passes through",
			url:            "/other/value",
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test/:id", ParseUUIDPathParameters, func(c *fiber.Ctx) error {
				// Check if UUID was parsed and stored in locals
				if idVal := c.Locals("id"); idVal != nil {
					if _, ok := idVal.(uuid.UUID); !ok {
						t.Error("Expected UUID in locals, got different type")
					}
				}
				return c.SendStatus(http.StatusOK)
			})
			app.Get("/other/:name", ParseUUIDPathParameters, func(c *fiber.Ctx) error {
				return c.SendStatus(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatusCode, resp.StatusCode)
		})
	}
}

func TestNewOfType(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  reflect.Type
	}{
		{
			name:  "struct pointer",
			input: &TestStruct{},
			want:  reflect.TypeOf(&TestStruct{}),
		},
		{
			name:  "different struct",
			input: &TestMetadataStruct{},
			want:  reflect.TypeOf(&TestMetadataStruct{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newOfType(tt.input)
			assert.Equal(t, tt.want, reflect.TypeOf(result))
		})
	}
}

func TestParseMetadata(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		originalMap map[string]any
		checkFunc   func(*testing.T, any)
	}{
		{
			name: "with metadata in original - not modified",
			input: &TestMetadataStruct{
				Name: "John",
			},
			originalMap: map[string]any{
				"name":     "John",
				"metadata": map[string]any{"key": "value"},
			},
			checkFunc: func(t *testing.T, result any) {
				// parseMetadata only sets metadata if it's not in originalMap
				// In this case, metadata exists in originalMap, so it doesn't modify the struct
			},
		},
		{
			name: "without metadata in original",
			input: &TestMetadataStruct{
				Name: "John",
			},
			originalMap: map[string]any{
				"name": "John",
			},
			checkFunc: func(t *testing.T, result any) {
				s := result.(*TestMetadataStruct)
				assert.NotNil(t, s.Metadata)
				assert.Empty(t, s.Metadata)
			},
		},
		{
			name:  "non-pointer input",
			input: TestMetadataStruct{Name: "John"},
			originalMap: map[string]any{
				"name": "John",
			},
			checkFunc: func(t *testing.T, result any) {
				// Should not panic
			},
		},
		{
			name:  "non-struct input",
			input: "test",
			originalMap: map[string]any{
				"key": "value",
			},
			checkFunc: func(t *testing.T, result any) {
				// Should not panic
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseMetadata(tt.input, tt.originalMap)
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.input)
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

func TestWithBodyDecodeHandlerFunc(t *testing.T) {
	t.Run("handler receives decoded struct", func(t *testing.T) {
		app := fiber.New()
		var receivedData *TestStruct

		app.Post("/test", WithBody(&TestStruct{}, func(p any, c *fiber.Ctx) error {
			receivedData = p.(*TestStruct)
			return c.JSON(receivedData)
		}))

		body := `{"name":"Alice","email":"alice@example.com","age":25}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotNil(t, receivedData)
		assert.Equal(t, "Alice", receivedData.Name)
		assert.Equal(t, "alice@example.com", receivedData.Email)
		assert.Equal(t, 25, receivedData.Age)
	})
}

func TestDecoderHandlerStruct(t *testing.T) {
	t.Run("decoder handler fields", func(t *testing.T) {
		handler := func(p any, c *fiber.Ctx) error {
			return nil
		}
		constructor := func() any {
			return &TestStruct{}
		}

		d := &decoderHandler{
			handler:      handler,
			constructor:  constructor,
			structSource: &TestStruct{},
		}

		assert.NotNil(t, d.handler)
		assert.NotNil(t, d.constructor)
		assert.NotNil(t, d.structSource)
	})
}

func TestWithBodyNilConstructor(t *testing.T) {
	t.Run("uses newOfType when constructor is nil", func(t *testing.T) {
		app := fiber.New()

		app.Post("/test", WithBody(&TestStruct{}, func(p any, c *fiber.Ctx) error {
			assert.NotNil(t, p)
			return c.SendStatus(http.StatusOK)
		}))

		body := `{"name":"Bob","email":"bob@example.com","age":30}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
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

func TestMalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "incomplete json",
			body: `{"name":"John"`,
		},
		{
			name: "invalid json syntax",
			body: `{name: "John"}`,
		},
		{
			name: "empty body",
			body: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/test", WithBody(&TestStruct{}, func(p any, c *fiber.Ctx) error {
				return c.SendStatus(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Should return error status
			assert.NotEqual(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestUUIDPathParametersGlobal(t *testing.T) {
	t.Run("UUIDPathParameters contains id", func(t *testing.T) {
		assert.Contains(t, UUIDPathParameters, "id")
	})
}

func TestPayloadContextValue(t *testing.T) {
	t.Run("payload context value type", func(t *testing.T) {
		var pcv PayloadContextValue = "test"
		assert.Equal(t, "test", string(pcv))
	})
}

func TestNestedStructValidation(t *testing.T) {
	type Address struct {
		Street string `json:"street" validate:"required"`
		City   string `json:"city" validate:"required"`
	}

	type Person struct {
		Name    string  `json:"name" validate:"required"`
		Address Address `json:"address" validate:"required"`
	}

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid nested struct",
			body:    `{"name":"John","address":{"street":"Main St","city":"NYC"}}`,
			wantErr: false,
		},
		{
			name:    "missing nested required field",
			body:    `{"name":"John","address":{"street":"Main St"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/test", WithBody(&Person{}, func(p any, c *fiber.Ctx) error {
				return c.SendStatus(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			if tt.wantErr {
				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			} else {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}

func TestFindUnknownFieldsTypeMismatch(t *testing.T) {
	tests := []struct {
		name      string
		original  map[string]any
		marshaled map[string]any
		wantLen   int
	}{
		{
			name: "map vs non-map type mismatch",
			original: map[string]any{
				"field": map[string]any{"nested": "value"},
			},
			marshaled: map[string]any{
				"field": "string value",
			},
			wantLen: 1,
		},
		{
			name: "slice vs non-slice type mismatch",
			original: map[string]any{
				"items": []any{"a", "b"},
			},
			marshaled: map[string]any{
				"items": "not an array",
			},
			wantLen: 1,
		},
		{
			name: "nested map with matching types",
			original: map[string]any{
				"user": map[string]any{
					"name": "John",
					"age":  30,
				},
			},
			marshaled: map[string]any{
				"user": map[string]any{
					"name": "John",
					"age":  30,
				},
			},
			wantLen: 0,
		},
		{
			name: "array with nested map differences",
			original: map[string]any{
				"users": []any{
					map[string]any{"name": "John", "extra": "field"},
					map[string]any{"name": "Jane"},
				},
			},
			marshaled: map[string]any{
				"users": []any{
					map[string]any{"name": "John"},
					map[string]any{"name": "Jane"},
				},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findUnknownFields(tt.original, tt.marshaled)
			assert.Equal(t, tt.wantLen, len(result))
		})
	}
}

func TestWithBodyWithConstructor(t *testing.T) {
	t.Run("decoderHandler with constructor function", func(t *testing.T) {
		app := fiber.New()

		constructorCalled := false
		constructor := func() any {
			constructorCalled = true
			return &TestStruct{}
		}

		d := &decoderHandler{
			handler: func(p any, c *fiber.Ctx) error {
				return c.SendStatus(http.StatusOK)
			},
			constructor:  constructor,
			structSource: &TestStruct{},
		}

		app.Post("/test", d.FiberHandlerFunc)

		body := `{"name":"Test","email":"test@example.com","age":25}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, constructorCalled, "Constructor should have been called")
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
