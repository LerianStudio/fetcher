package http

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestDecodeCursor(t *testing.T) {
	tests := []struct {
		name       string
		cursor     string
		wantID     string
		wantPoints bool
		wantErr    bool
	}{
		{
			name:       "valid cursor pointing next",
			cursor:     encodeCursor(Cursor{ID: "abc123", PointsNext: true}),
			wantID:     "abc123",
			wantPoints: true,
			wantErr:    false,
		},
		{
			name:       "valid cursor pointing previous",
			cursor:     encodeCursor(Cursor{ID: "xyz789", PointsNext: false}),
			wantID:     "xyz789",
			wantPoints: false,
			wantErr:    false,
		},
		{
			name:       "empty cursor ID",
			cursor:     encodeCursor(Cursor{ID: "", PointsNext: true}),
			wantID:     "",
			wantPoints: true,
			wantErr:    false,
		},
		{
			name:    "invalid base64",
			cursor:  "not-valid-base64!!!",
			wantErr: true,
		},
		{
			name:    "valid base64 but invalid JSON",
			cursor:  base64.StdEncoding.EncodeToString([]byte("not json")),
			wantErr: true,
		},
		{
			name:    "empty string",
			cursor:  "",
			wantErr: true,
		},
		{
			name:       "uuid cursor",
			cursor:     encodeCursor(Cursor{ID: "550e8400-e29b-41d4-a716-446655440000", PointsNext: true}),
			wantID:     "550e8400-e29b-41d4-a716-446655440000",
			wantPoints: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeCursor(tt.cursor)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeCursor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.ID != tt.wantID {
				t.Errorf("DecodeCursor().ID = %v, want %v", got.ID, tt.wantID)
			}
			if got.PointsNext != tt.wantPoints {
				t.Errorf("DecodeCursor().PointsNext = %v, want %v", got.PointsNext, tt.wantPoints)
			}
		})
	}
}

// Helper function to encode cursor for tests
func encodeCursor(c Cursor) string {
	data, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(data)
}
