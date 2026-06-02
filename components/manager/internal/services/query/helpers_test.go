package query

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/testutil"

	"github.com/stretchr/testify/require"
)

// testContext creates a context with logger and tracer for testing.
func testContext() context.Context {
	return testutil.TestContext()
}

// scopeAuthorityEngine builds the minimal Engine the read services route their
// tenant-scope authority through. Get/List keep their own UUID-keyed / paginated
// persistence, so the Engine needs no ConnectionStore — only the required
// connector registry to satisfy engine.New.
func scopeAuthorityEngine(t *testing.T) *engine.Engine {
	t.Helper()

	eng, err := engine.New(engine.WithConnectorRegistry(stubConnectorRegistry{}))
	require.NoError(t, err)

	return eng
}

type stubConnectorRegistry struct{}

func (stubConnectorRegistry) Connector(string) (engine.ConnectorFactory, bool) { return nil, false }
