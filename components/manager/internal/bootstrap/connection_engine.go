package bootstrap

import (
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/ports/job"
)

// connectionEngine builds the embedded Engine the Manager connection command
// services use to delegate their shared, tenant-scoped active-execution
// conflict gate (the "block update/delete while jobs run" rule).
//
// Only the gate is delegated, so the Engine is wired with the minimum required
// surface:
//   - ConnectorRegistry: REQUIRED by engine.New, but unused here because this
//     Engine never plans or executes extraction. A no-op registry that resolves
//     nothing satisfies the constructor without pulling any datasource driver
//     into this construction path.
//   - ActiveExecutionChecker: the Manager job repository adapted through
//     connectioncompat, which is what makes the gate consult real running jobs.
//
// Connection persistence, credential protection, and HTTP mapping deliberately
// stay in the Manager; no ConnectionStore or CredentialProtector is wired.
func connectionEngine(jobRepo job.Repository) (*engine.Engine, error) {
	return engine.New(
		engine.WithConnectorRegistry(noopConnectorRegistry{}),
		engine.WithActiveExecutionChecker(connectioncompat.NewJobActiveExecutionChecker(jobRepo)),
	)
}

// noopConnectorRegistry satisfies the Engine's REQUIRED ConnectorRegistry port
// for a connection-gate-only Engine. It resolves no connector types because
// this Engine never builds connectors.
type noopConnectorRegistry struct{}

func (noopConnectorRegistry) Connector(string) (engine.ConnectorFactory, bool) {
	return nil, false
}
