// Package messaging defines the domain port interface for message broker operations.
package messaging

import (
	"context"
)

// MessagePublisher defines the interface for publishing messages to a message broker.
// This is a subset of the full messaging adapter interface, exposing only the
// producer capability needed by service-layer code.
//
//go:generate mockgen --destination=publisher.mock.go --package=messaging . MessagePublisher
type MessagePublisher interface {
	ProducerDefault(ctx context.Context, exchange, key string, queueMessage []byte, header *map[string]any) error
}
