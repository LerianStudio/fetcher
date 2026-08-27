package etc

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rabbitMQDefinitions struct {
	Exchanges []struct {
		Name string `json:"name"`
	} `json:"exchanges"`
	Queues []struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"queues"`
	Bindings []struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		RoutingKey  string `json:"routing_key"`
	} `json:"bindings"`
}

func loadDefinitions(t *testing.T) rabbitMQDefinitions {
	t.Helper()

	contents, err := os.ReadFile("definitions.json")
	require.NoError(t, err)

	var definitions rabbitMQDefinitions
	require.NoError(t, json.Unmarshal(contents, &definitions))

	return definitions
}

func TestDefinitionsDoNotDeclareRetiredWorkExchange(t *testing.T) {
	t.Parallel()

	definitions := loadDefinitions(t)
	for _, exchange := range definitions.Exchanges {
		assert.NotEqual(t, "fetcher.extract-external-data.exchange", exchange.Name)
	}
	for _, binding := range definitions.Bindings {
		assert.NotEqual(t, "fetcher.extract-external-data.exchange", binding.Source)
	}
}

func TestDefinitionsRouteDeadLettersToConfiguredQueue(t *testing.T) {
	t.Parallel()

	definitions := loadDefinitions(t)

	var workQueueRoutingKey any
	for _, queue := range definitions.Queues {
		if queue.Name == "fetcher.extract-external-data.queue" {
			workQueueRoutingKey = queue.Arguments["x-dead-letter-routing-key"]
		}
	}
	assert.Equal(t, "fetcher.dlq", workQueueRoutingKey)

	foundBinding := false
	for _, binding := range definitions.Bindings {
		if binding.Source == "fetcher.dlx" && binding.Destination == "fetcher.dlq" && binding.RoutingKey == "fetcher.dlq" {
			foundBinding = true
		}
	}
	assert.True(t, foundBinding, "fetcher.dlx must route fetcher.dlq to the dead-letter queue")
}
