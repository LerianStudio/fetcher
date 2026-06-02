package in

import (
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	connPort "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/ports/job"

	"github.com/stretchr/testify/require"
)

// connectionEngineForConnRepo builds the connection-authority Engine the Manager
// bootstrap wires: a ConnectionStore over the connection repo (so Create
// persistence flows through the Engine) plus the active-execution checker over
// the job repo. Handler tests pass the same mock repos they assert on, so the
// Engine's pre-check + store write land on those mocks with the pre-delegation
// call shape.
func connectionEngineForConnRepo(t *testing.T, connRepo connPort.Repository, jobRepo job.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(noopConnectorRegistryForTest{}),
		engine.WithConnectionStore(connectioncompat.NewConnectionStore(connRepo)),
		engine.WithActiveExecutionChecker(connectioncompat.NewJobActiveExecutionChecker(jobRepo)),
	)
	require.NoError(t, err)

	return eng
}

// scopeAuthorityEngine builds the connection-authority Engine the read services
// route their PERSISTENCE through after the read-path deepening: Get's external
// fallback flows through FindByID and List through ListConnectionsPaged, both
// landing on the supplied connection repo via the connectioncompat adapter.
// Handler tests pass the same mock repo they assert on.
func scopeAuthorityEngine(t *testing.T, connRepo connPort.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(noopConnectorRegistryForTest{}),
		engine.WithConnectionStore(connectioncompat.NewConnectionStore(connRepo)),
	)
	require.NoError(t, err)

	return eng
}

// connectionEngineForJobRepo builds the connection-authority Engine the command
// services delegate to, wired to BOTH the supplied connection repo (so
// Update/Delete persistence flows through the Engine ID-addressed ops) and the
// job repository active-execution checker — mirroring the Manager bootstrap.
func connectionEngineForJobRepo(t *testing.T, connRepo connPort.Repository, jobRepo job.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(noopConnectorRegistryForTest{}),
		engine.WithConnectionStore(connectioncompat.NewConnectionStore(connRepo)),
		engine.WithActiveExecutionChecker(connectioncompat.NewJobActiveExecutionChecker(jobRepo)),
	)
	require.NoError(t, err)

	return eng
}

type noopConnectorRegistryForTest struct{}

func (noopConnectorRegistryForTest) Connector(string) (engine.ConnectorFactory, bool) {
	return nil, false
}
