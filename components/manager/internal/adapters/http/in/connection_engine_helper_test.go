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

// scopeAuthorityEngine builds the minimal Engine read services route their
// tenant-scope authority through. Get/List keep their own persistence, so no
// ConnectionStore is wired.
func scopeAuthorityEngine(t *testing.T) *engine.Engine {
	t.Helper()

	eng, err := engine.New(engine.WithConnectorRegistry(noopConnectorRegistryForTest{}))
	require.NoError(t, err)

	return eng
}

// connectionEngineForJobRepo builds the connection-gate Engine the command
// services delegate to, wired to the supplied job repository through the
// connectioncompat adapter — mirroring the Manager bootstrap. Handler tests use
// it to supply the Engine dependency with constructor-wiring changes only.
func connectionEngineForJobRepo(t *testing.T, jobRepo job.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(noopConnectorRegistryForTest{}),
		engine.WithActiveExecutionChecker(connectioncompat.NewJobActiveExecutionChecker(jobRepo)),
	)
	require.NoError(t, err)

	return eng
}

type noopConnectorRegistryForTest struct{}

func (noopConnectorRegistryForTest) Connector(string) (engine.ConnectorFactory, bool) {
	return nil, false
}
