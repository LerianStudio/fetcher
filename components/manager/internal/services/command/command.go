package query

import (
	"github.com/LerianStudio/fetcher/components/manager/internal/adapters/postgres/example"
)

// ExampleCommand is a struct that aggregates various repositories for simplified access for implementation.
type ExampleCommand struct {
	// ExampleRepo provides an abstraction on top of the examples data source.
	ExampleRepo example.Repository
}
