package mongodb

import (
	http "github.com/LerianStudio/fetcher/pkg/net/http"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// BuildPaginationOptions builds MongoDB find options from query header filters.
// It extracts limit, page, and sort_order to construct pagination.
// Default sort is by created_at descending. If filters.SortOrder == "asc",
// sort ascending (value 1); otherwise descending (value -1).
//
// It returns the builder alongside the effective limit so callers can pre-size
// result slices: mongo-driver v2's builder does not expose its fields for
// read-back the way the v1 FindOptions struct did.
func BuildPaginationOptions(filters http.QueryHeader) (*options.FindOptionsBuilder, int64) {
	limit := int64(filters.Limit)
	if limit < 0 {
		limit = 0
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}

	skip := int64(page*int(limit) - int(limit))
	if skip < 0 {
		skip = 0
	}

	sortValue := -1
	if filters.SortOrder == "asc" {
		sortValue = 1
	}

	opts := options.Find().
		SetLimit(limit).
		SetSkip(skip).
		SetSort(bson.D{{Key: "created_at", Value: sortValue}})

	return opts, limit
}

// AddDateRangeFilter adds created_at $gte/$lte conditions to the query filter
// based on filters.StartDate and filters.EndDate. If both dates are zero, no
// filter is added.
func AddDateRangeFilter(queryFilter bson.M, filters http.QueryHeader) {
	if filters.StartDate.IsZero() && filters.EndDate.IsZero() {
		return
	}

	createdAt := bson.M{}
	if !filters.StartDate.IsZero() {
		createdAt["$gte"] = filters.StartDate
	}

	if !filters.EndDate.IsZero() {
		createdAt["$lte"] = filters.EndDate
	}

	queryFilter["created_at"] = createdAt
}
