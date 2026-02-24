// Package storage defines the domain port interface for object storage operations.
package storage

import (
	"context"
)

// Repository provides an interface for object storage operations (get/put).
//
//go:generate mockgen --destination=repository.mock.go --package=storage . Repository
type Repository interface {
	Get(ctx context.Context, objectName string) ([]byte, error)
	Put(ctx context.Context, objectName string, data []byte) error
}
