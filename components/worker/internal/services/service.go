package services

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/ports/job"
	"github.com/LerianStudio/fetcher/pkg/ports/storage"
	"github.com/LerianStudio/fetcher/pkg/resolver"
	streaming "github.com/LerianStudio/lib-streaming"
)

// EngineRunner is the Worker's seam onto the embedded Engine's synchronous
// extraction runner. The Worker maps a queued job into an engine.ExtractionRequest
// and hands it, the tenant scope (tenantId + requestId ONLY — owner decision B2),
// and the host job id to this seam, which plans then executes the extraction in
// DIRECT mode and returns the inline result.
//
// The resolved connections the Worker already looked up (internal + external)
// travel through the ctx (seeded via schemacompat.WithResolvedConnections), so the
// Engine never re-resolves and tenant-manager stays out of the Engine core — the
// same context-seed contract the Manager schema path uses.
//
// The seam keeps the Engine wiring (connector registry, request-scoped connection
// store, host-side table-name normalization) OUT of the UseCase: the UseCase owns
// message parsing, job lookup, ack/nack, and skip logic. When EngineRunner is nil
// the UseCase falls back to the legacy direct-datasource extraction path, so
// wiring is opt-in per deployment.
type EngineRunner interface {
	// RunExtraction plans and executes the request in DIRECT mode for the tenant,
	// returning the inline ExtractionResult. The jobID is the host execution
	// identity.
	RunExtraction(
		ctx context.Context,
		tenant engine.TenantContext,
		jobID string,
		request engine.ExtractionRequest,
	) (engine.ExtractionResult, error)
}

// UseCase is a struct that coordinates the handling of template files, report storage, external data sources, and report data.
type UseCase struct {
	// ExternalDataStorage is a repository used to retrieve external data from external data storage.
	ExternalDataStorage storage.Repository

	// JobRepository is a repository used to retrieve job data from MongoDB storage.
	JobRepository job.Repository

	// ConnectionRepository is a repository used to retrieve connection data from MongoDB storage.
	ConnectionRepository connection.Repository

	// Cryptor is used to decrypt connection passwords when creating data sources.
	Cryptor crypto.Cryptor

	// DocumentSigner is used to compute HMAC signatures for extracted data before encryption.
	// External consumers can use this signature to verify data integrity.
	DocumentSigner crypto.Signer

	// FileTTL defines the Time To Live for file (e.g., "1m", "1h", "7d", "30d"). Empty means no TTL.
	FileTTL string

	// JobEventEmitter publishes past-tense job business events through lib-streaming.
	// This is the public job notification event contract; legacy direct RabbitMQ
	// routing is not used for completed/failed business notifications.
	JobEventEmitter streaming.Emitter

	// JobEventStreamingEnabled indicates STREAMING_ENABLED=true produced a real emitter.
	JobEventStreamingEnabled bool

	// JobEventStreamingRequireTenant makes missing tenant IDs fail loudly in multi-tenant mode.
	JobEventStreamingRequireTenant bool

	// dataSourceFactory creates DataSource instances from connections.
	dataSourceFactory func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error)

	// storageEncryptDerivedKey is the HKDF-derived AES-256 key for encrypting
	// extracted data at rest. Derived from APP_ENC_KEY with context "fetcher-storage-encryption-v1".
	storageEncryptDerivedKey []byte

	// crmEncryptSecretKey is the encryption secret key for plugin_crm datasource.
	crmEncryptSecretKey string

	// crmHashSecretKey is the hash secret key for plugin_crm datasource.
	crmHashSecretKey string

	// ConnectionResolver resolves datasource connections (internal + external).
	// When nil, falls back to ConnectionRepository.FindByConfigNames (backwards compatible).
	ConnectionResolver resolver.ConnectionResolver

	// EngineRunner routes extraction through the embedded Engine (plan-then-execute,
	// DIRECT mode). When nil, the Worker uses the legacy direct-datasource path,
	// keeping the migration opt-in and every legacy test unchanged.
	EngineRunner EngineRunner
}

// SetStorageEncryptDerivedKey configures the HKDF-derived AES-256 key for storage encryption.
func (uc *UseCase) SetStorageEncryptDerivedKey(key []byte) {
	uc.storageEncryptDerivedKey = key
}

// SetCRMSecrets configures the CRM plugin encryption and hash secret keys.
func (uc *UseCase) SetCRMSecrets(encryptKey, hashKey string) {
	uc.crmEncryptSecretKey = encryptKey
	uc.crmHashSecretKey = hashKey
}

// SetDataSourceFactory configures the factory used to create DataSource instances.
func (uc *UseCase) SetDataSourceFactory(factory func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error)) {
	uc.dataSourceFactory = factory
}

// CreateDataSource creates a DataSource from a connection using the injected factory.
func (uc *UseCase) CreateDataSource(ctx context.Context, conn *model.Connection) (datasource.DataSource, error) {
	return uc.dataSourceFactory(ctx, conn, uc.Cryptor)
}
