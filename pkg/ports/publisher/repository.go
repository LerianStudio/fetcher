// Package publisher defines the domain port interface for message publishing.
package publisher

import (
	"context"
)

// Repository provides an interface for publishing messages to a message broker.
//
//go:generate mockgen --destination=repository.mock.go --package=publisher . Repository
type Repository interface {
	Publish(ctx context.Context, exchange, routingKey string, body []byte) error
	Shutdown(ctx context.Context) error
}
