package in

import (
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/ports/job"

	"github.com/stretchr/testify/require"
)

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
