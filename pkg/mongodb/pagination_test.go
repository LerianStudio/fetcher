package mongodb

import (
	"testing"
	"time"

	http "github.com/LerianStudio/fetcher/pkg/net/http"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestBuildPaginationOptions(t *testing.T) {
	tests := []struct {
		name          string
		filters       http.QueryHeader
		wantLimit     int64
		wantSkip      int64
		wantSortValue int
	}{
		{
			name: "default descending sort",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  1,
			},
			wantLimit:     10,
			wantSkip:      0,
			wantSortValue: -1,
		},
		{
			name: "ascending sort",
			filters: http.QueryHeader{
				Limit:     10,
				Page:      1,
				SortOrder: "asc",
			},
			wantLimit:     10,
			wantSkip:      0,
			wantSortValue: 1,
		},
		{
			name: "page 2 with limit 10",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  2,
			},
			wantLimit:     10,
			wantSkip:      10,
			wantSortValue: -1,
		},
		{
			name: "page 3 with limit 5",
			filters: http.QueryHeader{
				Limit: 5,
				Page:  3,
			},
			wantLimit:     5,
			wantSkip:      10,
			wantSortValue: -1,
		},
		{
			name: "zero page defaults to page 1",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  0,
			},
			wantLimit:     10,
			wantSkip:      0,
			wantSortValue: -1,
		},
		{
			name: "negative page defaults to page 1",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  -1,
			},
			wantLimit:     10,
			wantSkip:      0,
			wantSortValue: -1,
		},
		{
			name: "negative limit clamped to zero",
			filters: http.QueryHeader{
				Limit: -5,
				Page:  1,
			},
			wantLimit:     0,
			wantSkip:      0,
			wantSortValue: -1,
		},
		{
			name: "zero limit",
			filters: http.QueryHeader{
				Limit: 0,
				Page:  1,
			},
			wantLimit:     0,
			wantSkip:      0,
			wantSortValue: -1,
		},
		{
			name:          "empty filters defaults",
			filters:       http.QueryHeader{},
			wantLimit:     0,
			wantSkip:      0,
			wantSortValue: -1,
		},
		{
			name: "non-asc sort order treated as desc",
			filters: http.QueryHeader{
				Limit:     10,
				Page:      1,
				SortOrder: "desc",
			},
			wantLimit:     10,
			wantSkip:      0,
			wantSortValue: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, limit := BuildPaginationOptions(tt.filters)

			if limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", limit, tt.wantLimit)
			}

			// mongo-driver v2 options are builders; materialize the FindOptions
			// by applying the builder's List() to inspect the accumulated values.
			opts := &options.FindOptions{}
			for _, set := range builder.List() {
				if err := set(opts); err != nil {
					t.Fatalf("failed to apply find option: %v", err)
				}
			}

			if opts.Limit == nil {
				t.Fatal("expected Limit to be set")
			}
			if *opts.Limit != tt.wantLimit {
				t.Errorf("materialized Limit = %d, want %d", *opts.Limit, tt.wantLimit)
			}

			if opts.Skip == nil {
				t.Fatal("expected Skip to be set")
			}
			if *opts.Skip != tt.wantSkip {
				t.Errorf("Skip = %d, want %d", *opts.Skip, tt.wantSkip)
			}

			sort, ok := opts.Sort.(bson.D)
			if !ok {
				t.Fatalf("expected Sort to be bson.D, got %T", opts.Sort)
			}
			if len(sort) != 1 || sort[0].Key != "created_at" {
				t.Fatalf("expected sort key 'created_at', got %v", sort)
			}
			if sort[0].Value != tt.wantSortValue {
				t.Errorf("sort value = %v, want %d", sort[0].Value, tt.wantSortValue)
			}
		})
	}
}

func TestAddDateRangeFilter(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-24 * time.Hour)

	tests := []struct {
		name      string
		filters   http.QueryHeader
		wantGte   bool
		wantLte   bool
		wantNoKey bool
	}{
		{
			name: "both dates zero adds no filter",
			filters: http.QueryHeader{
				StartDate: time.Time{},
				EndDate:   time.Time{},
			},
			wantNoKey: true,
		},
		{
			name: "only start date adds gte",
			filters: http.QueryHeader{
				StartDate: earlier,
				EndDate:   time.Time{},
			},
			wantGte: true,
			wantLte: false,
		},
		{
			name: "only end date adds lte",
			filters: http.QueryHeader{
				StartDate: time.Time{},
				EndDate:   now,
			},
			wantGte: false,
			wantLte: true,
		},
		{
			name: "both dates adds gte and lte",
			filters: http.QueryHeader{
				StartDate: earlier,
				EndDate:   now,
			},
			wantGte: true,
			wantLte: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryFilter := bson.M{}
			AddDateRangeFilter(queryFilter, tt.filters)

			createdAt, exists := queryFilter["created_at"]
			if tt.wantNoKey {
				if exists {
					t.Errorf("expected no 'created_at' key, but got %v", createdAt)
				}
				return
			}

			if !exists {
				t.Fatal("expected 'created_at' key in filter")
			}

			dateFilter, ok := createdAt.(bson.M)
			if !ok {
				t.Fatalf("expected created_at to be bson.M, got %T", createdAt)
			}

			_, hasGte := dateFilter["$gte"]
			if tt.wantGte && !hasGte {
				t.Error("expected $gte in date filter")
			}
			if !tt.wantGte && hasGte {
				t.Error("unexpected $gte in date filter")
			}

			_, hasLte := dateFilter["$lte"]
			if tt.wantLte && !hasLte {
				t.Error("expected $lte in date filter")
			}
			if !tt.wantLte && hasLte {
				t.Error("unexpected $lte in date filter")
			}
		})
	}
}

func TestAddDateRangeFilter_PreservesExistingFilters(t *testing.T) {
	queryFilter := bson.M{
		"status": "active",
	}
	now := time.Now()

	AddDateRangeFilter(queryFilter, http.QueryHeader{
		StartDate: now,
	})

	if _, exists := queryFilter["status"]; !exists {
		t.Error("existing 'status' filter was removed")
	}
	if _, exists := queryFilter["created_at"]; !exists {
		t.Error("expected 'created_at' filter to be added")
	}
}
