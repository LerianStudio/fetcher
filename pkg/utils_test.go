package pkg

import (
	"math"
	"testing"
)

func TestSafeIntToInt32(t *testing.T) {
	tests := []struct {
		name    string
		val     int
		want    int32
		wantErr bool
	}{
		{
			name:    "valid positive value",
			val:     100,
			want:    100,
			wantErr: false,
		},
		{
			name:    "valid negative value",
			val:     -100,
			want:    -100,
			wantErr: false,
		},
		{
			name:    "zero value",
			val:     0,
			want:    0,
			wantErr: false,
		},
		{
			name:    "max int32 value",
			val:     math.MaxInt32,
			want:    math.MaxInt32,
			wantErr: false,
		},
		{
			name:    "min int32 value",
			val:     math.MinInt32,
			want:    math.MinInt32,
			wantErr: false,
		},
		{
			name:    "overflow positive",
			val:     math.MaxInt32 + 1,
			want:    0,
			wantErr: true,
		},
		{
			name:    "overflow negative",
			val:     math.MinInt32 - 1,
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeIntToInt32(tt.val)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeIntToInt32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SafeIntToInt32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNilOrEmpty(t *testing.T) {
	tests := []struct {
		name string
		s    *string
		want bool
	}{
		{
			name: "nil pointer",
			s:    nil,
			want: true,
		},
		{
			name: "empty string",
			s:    strPtr(""),
			want: true,
		},
		{
			name: "whitespace only",
			s:    strPtr("   "),
			want: true,
		},
		{
			name: "null string",
			s:    strPtr("null"),
			want: true,
		},
		{
			name: "nil string",
			s:    strPtr("nil"),
			want: true,
		},
		{
			name: "valid string",
			s:    strPtr("hello"),
			want: false,
		},
		{
			name: "valid string with spaces",
			s:    strPtr("  hello  "),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNilOrEmpty(tt.s); got != tt.want {
				t.Errorf("IsNilOrEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateServerAddress(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "valid address with port",
			value: "localhost:8080",
			want:  "localhost:8080",
		},
		{
			name:  "valid IP with port",
			value: "192.168.1.1:5432",
			want:  "192.168.1.1:5432",
		},
		{
			name:  "valid hostname with port",
			value: "db.example.com:3306",
			want:  "db.example.com:3306",
		},
		{
			name:  "missing port",
			value: "localhost",
			want:  "",
		},
		{
			name:  "missing address",
			value: ":8080",
			want:  "",
		},
		{
			name:  "empty string",
			value: "",
			want:  "",
		},
		{
			name:  "invalid format - no colon",
			value: "localhost8080",
			want:  "",
		},
		{
			name:  "invalid format - non-numeric port",
			value: "localhost:abc",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateServerAddress(tt.value); got != tt.want {
				t.Errorf("ValidateServerAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "non-empty secret",
			value: "supersecret",
			want:  "[REDACTED]",
		},
		{
			name:  "empty secret",
			value: "",
			want:  "",
		},
		{
			name:  "whitespace secret",
			value: "   ",
			want:  "[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskSecret(tt.value); got != tt.want {
				t.Errorf("MaskSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaskSecretPtr(t *testing.T) {
	tests := []struct {
		name  string
		value *string
		want  *string
	}{
		{
			name:  "non-empty secret",
			value: strPtr("supersecret"),
			want:  strPtr("[REDACTED]"),
		},
		{
			name:  "empty secret",
			value: strPtr(""),
			want:  strPtr(""),
		},
		{
			name:  "nil pointer",
			value: nil,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskSecretPtr(tt.value)
			if tt.want == nil {
				if got != nil {
					t.Errorf("MaskSecretPtr() = %v, want nil", *got)
				}
				return
			}
			if got == nil {
				t.Errorf("MaskSecretPtr() = nil, want %v", *tt.want)
				return
			}
			if *got != *tt.want {
				t.Errorf("MaskSecretPtr() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}
