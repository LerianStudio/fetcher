package query

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/datasource"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/schemacompat"
	cacheRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/cache"
	connPort "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/v2/pkg/testutil"

	"github.com/stretchr/testify/require"
)

// testContext creates a context with logger and tracer for testing.
func testContext() context.Context {
	return testutil.TestContext()
}

// scopeAuthorityEngine builds the connection-authority Engine the read services
// route their PERSISTENCE through after the read-path deepening: Get's external
// fallback flows through the Engine's ID-addressed FindByID and List flows
// through ListConnectionsPaged, both landing on the SAME connection repo passed
// here via the connectioncompat ConnectionStore adapter. Tests therefore set
// their repo.FindByID / repo.List expectations on the same mock and they are
// satisfied through the Engine gate. The connector registry satisfies the
// required engine.New port; this Engine never plans or executes extraction.
func scopeAuthorityEngine(t *testing.T, connRepo connPort.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(stubConnectorRegistry{}),
		engine.WithConnectionStore(connectioncompat.NewConnectionStore(connRepo)),
	)
	require.NoError(t, err)

	return eng
}

type stubConnectorRegistry struct{}

func (stubConnectorRegistry) Connector(string) (engine.ConnectorFactory, bool) { return nil, false }

// schemaDiscoveryEngine builds the schema-discovery Engine the schema services
// route DISCOVERY through. It mirrors the production schemaEngine wiring: a
// schemacompat ConnectorFactory over the supplied datasource factory + cryptor,
// the request-scoped schemacompat ConnectionStore, and the supplied schema cache
// adapted behind the engine.SchemaCache port. A nil cache disables caching (the
// Engine discovers fresh every time); the existing datasource and cache mocks
// therefore exercise the SAME hit/miss/Set behavior through the Engine port.
func schemaDiscoveryEngine(
	t *testing.T,
	factory datasource.DataSourceFactory,
	cryptor crypto.Cryptor,
	schemaCache cacheRepo.SchemaCacheRepository,
) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(schemaConnectorRegistryStub{factory: schemacompat.NewConnectorFactory(factory, cryptor)}),
		engine.WithConnectionStore(schemacompat.NewConnectionStore()),
		engine.WithSchemaCache(schemacompat.NewSchemaCache(schemaCache, 0)),
	)
	require.NoError(t, err)

	return eng
}

// schemaConnectorRegistryStub resolves the single schema ConnectorFactory for any
// datasource type, matching the production schema registry (type validation lives
// in the factory's Build, from the descriptor).
type schemaConnectorRegistryStub struct {
	factory engine.ConnectorFactory
}

func (r schemaConnectorRegistryStub) Connector(string) (engine.ConnectorFactory, bool) {
	return r.factory, true
}
