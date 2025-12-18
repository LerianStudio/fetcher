package datasource

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
)

//go:generate mockgen --destination=datasource-config.mock.go --package=datasource . DataSource

// DataSourceConfig represents the base configuration for all data sources.
type DataSourceConfig struct {
	ID                string
	OrganizationID    string
	ConfigName        string
	Type              string
	Host              string
	Port              string
	DatabaseName      string
	Username          string
	PasswordEncrypted string
	SSL               model.SSLConfig
	Status            string
}

// DataSource defines a common interface for all data source types.
type DataSource interface {
	// GetConfig returns the base configuration for the data source.
	GetConfig() DataSourceConfig

	// Connect establishes a connection to the data source.
	// Returns an error if the connection cannot be established.
	Connect(ctx context.Context, logger log.Logger) error

	// Close closes the connection to the data source.
	// Returns an error if the connection cannot be closed properly.
	Close(ctx context.Context) error

	// GetType returns the database type (e.g., "mongodb", "postgresql").
	GetType() string

	// Query executes a query on the specified table/collection with the given fields and filters.
	Query(ctx context.Context, tables map[string][]string, filters map[string]map[string]job.FilterCondition, logger log.Logger) (map[string][]map[string]any, error)

	// GetSchemaInfo returns the schema information for the datasource.
	// Returns tables and their columns for schema validation.
	GetSchemaInfo(ctx context.Context) (*model.DataSourceSchema, error)
}
