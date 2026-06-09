package command

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
	connPort "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/v2/pkg/ports/job"
	"github.com/LerianStudio/fetcher/v2/pkg/testutil"

	"github.com/stretchr/testify/require"
)

// testContext creates a context with logger and tracer for testing.
func testContext() context.Context {
	return testutil.TestContext()
}

// engineForConnRepo builds the connection-authority Engine the Manager bootstrap
// wires: a ConnectionStore over the connection repo (so Create persistence flows
// through the Engine) plus the active-execution checker over the job repo. Tests
// pass the SAME mock connection/job repos they assert on, so the Engine's
// pre-check + store write land on those mocks with the pre-delegation call shape.
func engineForConnRepo(t *testing.T, connRepo connPort.Repository, jobRepo job.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(stubConnectorRegistry{}),
		engine.WithConnectionStore(connectioncompat.NewConnectionStore(connRepo)),
		engine.WithActiveExecutionChecker(connectioncompat.NewJobActiveExecutionChecker(jobRepo)),
	)
	require.NoError(t, err)

	return eng
}
