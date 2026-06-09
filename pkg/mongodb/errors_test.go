package mongodb

import (
	"errors"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestIsIndexConflictError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "code 85 IndexOptionsConflict",
			err:  mongo.CommandError{Code: 85, Message: "Index options conflict"},
			want: true,
		},
		{
			name: "code 86 IndexKeySpecsConflict",
			err:  mongo.CommandError{Code: 86, Message: "Index key specs conflict"},
			want: true,
		},
		{
			name: "code 11000 DuplicateKey",
			err:  mongo.CommandError{Code: 11000, Message: "Duplicate key"},
			want: false,
		},
		{
			name: "code 0",
			err:  mongo.CommandError{Code: 0, Message: "Unknown"},
			want: false,
		},
		{
			name: "code 84 near miss",
			err:  mongo.CommandError{Code: 84, Message: "Some error"},
			want: false,
		},
		{
			name: "code 87 near miss",
			err:  mongo.CommandError{Code: 87, Message: "Some error"},
			want: false,
		},
		{
			name: "non-CommandError type - plain error",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "non-CommandError type - fmt wrapped",
			err:  fmt.Errorf("wrapped: %w", errors.New("inner")),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "wrapped CommandError code 85",
			err:  fmt.Errorf("index creation failed: %w", mongo.CommandError{Code: 85, Message: "conflict"}),
			want: true,
		},
		{
			name: "wrapped CommandError code 86",
			err:  fmt.Errorf("index creation failed: %w", mongo.CommandError{Code: 86, Message: "conflict"}),
			want: true,
		},
		{
			name: "wrapped non-conflict CommandError",
			err:  fmt.Errorf("failed: %w", mongo.CommandError{Code: 100, Message: "other"}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsIndexConflictError(tt.err)
			if got != tt.want {
				t.Errorf("IsIndexConflictError() = %v, want %v", got, tt.want)
			}
		})
	}
}
