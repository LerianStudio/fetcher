package bootstrap

import (
	"testing"
	"time"

	cacheRepo "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
)

func TestGetSchemaCacheTTL(t *testing.T) {
	tests := []struct {
		name   string
		ttlStr string
		want   time.Duration
	}{
		{
			name:   "empty string returns default",
			ttlStr: "",
			want:   cacheRepo.DefaultSchemaCacheTTL,
		},
		{
			name:   "valid seconds",
			ttlStr: "300",
			want:   300 * time.Second,
		},
		{
			name:   "invalid string returns default",
			ttlStr: "invalid",
			want:   cacheRepo.DefaultSchemaCacheTTL,
		},
		{
			name:   "zero seconds",
			ttlStr: "0",
			want:   0,
		},
		{
			name:   "negative value",
			ttlStr: "-100",
			want:   -100 * time.Second,
		},
		{
			name:   "large value",
			ttlStr: "86400",
			want:   86400 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSchemaCacheTTL(tt.ttlStr)
			if got != tt.want {
				t.Errorf("getSchemaCacheTTL(%q) = %v, want %v", tt.ttlStr, got, tt.want)
			}
		})
	}
}

func TestGetRedisDB(t *testing.T) {
	tests := []struct {
		name  string
		dbStr string
		want  int
	}{
		{
			name:  "empty string returns 0",
			dbStr: "",
			want:  0,
		},
		{
			name:  "valid db number",
			dbStr: "5",
			want:  5,
		},
		{
			name:  "invalid string returns 0",
			dbStr: "invalid",
			want:  0,
		},
		{
			name:  "zero",
			dbStr: "0",
			want:  0,
		},
		{
			name:  "negative value",
			dbStr: "-1",
			want:  -1,
		},
		{
			name:  "large value",
			dbStr: "15",
			want:  15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRedisDB(tt.dbStr)
			if got != tt.want {
				t.Errorf("getRedisDB(%q) = %d, want %d", tt.dbStr, got, tt.want)
			}
		})
	}
}
