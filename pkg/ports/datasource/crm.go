// Package datasource defines domain port interfaces for data source operations.
package datasource

import (
	"context"

	"github.com/LerianStudio/fetcher/v2/pkg/model/job"
)

// CRMQueryable defines the interface for data sources that support CRM-specific
// collection-level query operations. This allows service-layer code to use
// interface-based type assertions instead of depending on concrete MongoDB types.
//
// Implementations should delegate to the underlying database driver's query methods.
//
//go:generate mockgen --destination=crm.mock.go --package=datasource . CRMQueryable
type CRMQueryable interface {
	// QueryCollection queries a single collection with the specified fields and optional filter.
	// Pass nil for filter to query without filtering.
	QueryCollection(ctx context.Context, collection string, fields []string, filter map[string][]any) ([]map[string]any, error)

	// QueryCollectionWithAdvancedFilters queries a single collection using advanced FilterCondition filters.
	QueryCollectionWithAdvancedFilters(ctx context.Context, collection string, fields []string, filters map[string]job.FilterCondition) ([]map[string]any, error)

	// ListCollectionNames returns all collection names in the database.
	// Used by plugin_crm to discover collections by prefix (e.g., "holders_*").
	ListCollectionNames(ctx context.Context) ([]string, error)
}
