package bootstrap

import (
	"time"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/engine"
	schemacompat "github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
	cachePort "github.com/LerianStudio/fetcher/pkg/ports/cache"
)

// schemaConnectorType is the datasource type the schema connector registry
// resolves. The schemacompat ConnectorFactory builds the rich connection carried
// in the descriptor regardless of its declared type, so a single registered
// factory under a wildcard-equivalent lookup serves every datasource type. The
// registry below ignores the type and always returns the one factory, because
// the factory itself validates the type from the descriptor at Build time.
type schemaConnectorRegistry struct {
	factory engine.ConnectorFactory
}

// Connector resolves the single schema ConnectorFactory for any datasource type.
// Type validation happens inside the factory's Build (from the descriptor), so
// the registry resolves unconditionally rather than enumerating every type.
func (r schemaConnectorRegistry) Connector(string) (engine.ConnectorFactory, bool) {
	return r.factory, true
}

// schemaEngine builds the embedded Engine that is the AUTHORITY for the Manager's
// schema DISCOVERY and CACHING. The Manager's GetConnectionSchema and
// ValidateSchema services resolve connections on their own hot path (internal via
// the resolver, external via the connection Engine's ID-addressed read), then
// route discovery through this Engine:
//
//   - ConnectorRegistry: the schemacompat ConnectorFactory, which rebuilds the
//     host-resolved connection from the descriptor's opaque payload and runs it
//     through the existing datasource factory (preserving decrypt + connect).
//   - ConnectionStore: the schemacompat request-scoped store, which returns the
//     connection the host already resolved (seeded into the request context) so
//     the Engine never re-resolves and tenant-manager stays out of Engine core.
//   - SchemaCache: the Manager's Redis-backed schema cache adapted behind the
//     engine.SchemaCache port. The Engine sees only the port; Redis stays Manager-
//     wired. A nil cache leaves the Engine to discover fresh every time.
//
// Credential protection and HTTP mapping deliberately stay in the Manager; no
// CredentialProtector is wired (the rich connection carries PasswordEncrypted and
// the host factory decrypts).
func schemaEngine(
	dsFactory datasource.DataSourceFactory,
	cryptor crypto.Cryptor,
	schemaCache cachePort.SchemaCacheRepository,
	schemaCacheTTL time.Duration,
) (*engine.Engine, error) {
	registry := schemaConnectorRegistry{
		factory: schemacompat.NewConnectorFactory(dsFactory, cryptor),
	}

	return engine.New(
		engine.WithConnectorRegistry(registry),
		engine.WithConnectionStore(schemacompat.NewConnectionStore()),
		engine.WithSchemaCache(schemacompat.NewSchemaCache(schemaCache, schemaCacheTTL)),
	)
}
