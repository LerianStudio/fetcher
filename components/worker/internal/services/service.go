package services

import (
	"context"
	"errors"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/v2/pkg/ports/job"
	"github.com/LerianStudio/fetcher/v2/pkg/ports/storage"
	"github.com/LerianStudio/fetcher/v2/pkg/resolver"
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
// message parsing, job lookup, ack/nack, and skip logic. EngineRunner is MANDATORY
// — the legacy direct-datasource extraction path was removed in the strangler
// completion (T-010) and UseCase.Validate rejects a nil runner at wiring time, so a
// nil EngineRunner is a bootstrap wiring bug, not a supported fallback.
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

	// EngineRunner routes generic-datasource extraction through the embedded Engine
	// (plan-then-execute, DIRECT mode). It is MANDATORY: the legacy direct-datasource
	// path was removed (strangler completion, T-010) and Validate rejects a nil
	// runner at bootstrap. A nil EngineRunner is a wiring bug, not a fallback.
	EngineRunner EngineRunner
}

// Validate verifies the UseCase is wired with the mandatory capabilities the
// extraction path requires. After the strangler completion (T-010) the legacy
// direct-datasource extraction path is gone, so the embedded Engine runner is
// MANDATORY: a nil EngineRunner would nil-panic deep in extraction. The guard
// fails fast at wiring time (bootstrap) with a clear error instead. The bootstrap
// is the sole guarantor that EngineRunner is set.
func (uc *UseCase) Validate() error {
	if uc.EngineRunner == nil {
		return errEngineRunnerRequired
	}

	if uc.dataSourceFactory == nil {
		return errDataSourceFactoryRequired
	}

	return nil
}

// errEngineRunnerRequired is returned by Validate when no Engine runner is wired.
var errEngineRunnerRequired = errors.New("worker UseCase requires a non-nil EngineRunner: the legacy extraction path has been removed")

// errDataSourceFactoryRequired is returned by Validate when no datasource factory is wired.
// The plugin_crm extraction path calls CreateDataSource, which dereferences this factory.
var errDataSourceFactoryRequired = errors.New("worker UseCase requires a non-nil dataSourceFactory for plugin_crm extraction")

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
