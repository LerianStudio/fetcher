package query

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	connPort "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/testutil"

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
