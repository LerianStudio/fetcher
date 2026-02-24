package bootstrap

import (
	"testing"
)

func TestMultiQueueConsumer_StructFields(t *testing.T) {
	// MultiQueueConsumer can be partially constructed without infrastructure.
	// NewMultiQueueConsumer requires real ConsumerRoutes, so we test the struct directly.
	consumer := &MultiQueueConsumer{}

	if consumer.consumerRoutes != nil {
		t.Error("consumerRoutes should be nil by default")
	}
	if consumer.UseCase != nil {
		t.Error("UseCase should be nil by default")
	}
}

