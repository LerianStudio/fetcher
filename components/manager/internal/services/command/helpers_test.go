package command

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	connPort "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/ports/job"
	"github.com/LerianStudio/fetcher/pkg/testutil"

	"github.com/stretchr/testify/require"
)

// testContext creates a context with logger and tracer for testing.
func testContext() context.Context {
	return testutil.TestContext()
}

// engineForJobRepo builds an Engine wired to the supplied job repository
// through the connectioncompat ActiveExecutionChecker adapter. This is the same
// wiring the Manager bootstrap uses, so a test that sets jobRepo expectations
// keeps them satisfied through the Engine gate after delegation.
func engineForJobRepo(t *testing.T, jobRepo job.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(stubConnectorRegistry{}),
		engine.WithActiveExecutionChecker(connectioncompat.NewJobActiveExecutionChecker(jobRepo)),
	)
	require.NoError(t, err)

	return eng
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
