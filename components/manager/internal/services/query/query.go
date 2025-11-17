package query

import (
	"github.com/LerianStudio/fetcher/components/manager/internal/adapters/postgres/example"
)

// ExampleQuery is a struct that aggregates various repositories for simplified access for implementation.
type ExampleQuery struct {
	// ExampleRepo provides an abstraction on top of the examples data source.
	ExampleRepo example.Repository
}
