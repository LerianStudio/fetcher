package query

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/testutil"
)

// testContext creates a context with logger and tracer for testing.
func testContext() context.Context {
	return testutil.TestContext()
}
