package bootstrap

import (
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
	connPort "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/v2/pkg/ports/job"
)

// connectionEngine builds the embedded Engine that is the AUTHORITY for the
// Manager's connection rules. All five connection services route their
// tenant-scoped policy through it:
//   - Create persists fully through the Engine (tenant-scope + (tenantID,
//     configName) uniqueness, then write via the ConnectionStore adapter);
//   - Update/Delete route the active-execution conflict gate and the tenant-scope
//     authority through the Engine while keeping their UUID-keyed persistence;
//   - Get/List route the tenant-scope authority through the Engine while keeping
//     their UUID-keyed / paginated reads.
//
// The Engine is wired with:
//   - ConnectorRegistry: REQUIRED by engine.New, but unused here because this
//     Engine never plans or executes extraction. A no-op registry satisfies the
//     constructor without pulling any datasource driver into this path.
//   - ConnectionStore: the Manager's RICH connection repository adapted through
//     connectioncompat, so Create's persistence flows through the Engine. The
//     rich record (ProductName / full SSL / uuid / metadata / timestamps) rides
//     through the descriptor's opaque host payload, so no field is dropped and
//     ProductName never becomes an Engine scope dimension.
//   - ActiveExecutionChecker: the Manager job repository adapted through
//     connectioncompat, which makes the conflict gate consult real running jobs.
//
// Credential protection and HTTP mapping deliberately stay in the Manager; no
// CredentialProtector is wired (encrypted persistence stays Manager-side via the
// rich model's PasswordEncrypted).
func connectionEngine(connRepo connPort.Repository, jobRepo job.Repository) (*engine.Engine, error) {
	return engine.New(
		engine.WithConnectorRegistry(noopConnectorRegistry{}),
		engine.WithConnectionStore(connectioncompat.NewConnectionStore(connRepo)),
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
