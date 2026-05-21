package services

import (
	"context"
	"fmt"

	libLog "github.com/LerianStudio/lib-observability/log"
	streaming "github.com/LerianStudio/lib-streaming"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	publisherPort "github.com/LerianStudio/fetcher/pkg/ports/publisher"
	storagePort "github.com/LerianStudio/fetcher/pkg/ports/storage"
	"github.com/LerianStudio/fetcher/pkg/testutil"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// testContext creates a context with logger and tracer for testing.
func testContext() context.Context {
	return testutil.TestContext()
}

// testLogger creates a logger for testing that suppresses output.
func testLogger() *libLog.GoLogger {
	return &libLog.GoLogger{Level: libLog.LevelError}
}

// testMocks holds all mock dependencies for testing.
type testMocks struct {
	ctrl            *gomock.Controller
	jobRepo         *jobRepo.MockRepository
	connRepo        *connRepo.MockRepository
	seaweedFS       *storagePort.MockRepository
	cryptor         *crypto.MockCryptor
	rabbitPublisher *publisherPort.MockRepository
}

// newTestMocks creates and returns all mock dependencies.
func newTestMocks(ctrl *gomock.Controller) *testMocks {
	return &testMocks{
		ctrl:            ctrl,
		jobRepo:         jobRepo.NewMockRepository(ctrl),
		connRepo:        connRepo.NewMockRepository(ctrl),
		seaweedFS:       storagePort.NewMockRepository(ctrl),
		cryptor:         crypto.NewMockCryptor(ctrl),
		rabbitPublisher: publisherPort.NewMockRepository(ctrl),
	}
}

// newTestUseCase creates a UseCase with all mocked dependencies.
// Now that UseCase uses interfaces, we can inject mocks directly.
func newTestUseCase(mocks *testMocks) *UseCase {
	uc := &UseCase{
		ExternalDataStorage:      mocks.seaweedFS,
		JobRepository:            mocks.jobRepo,
		ConnectionRepository:     mocks.connRepo,
		Cryptor:                  mocks.cryptor,
		FileTTL:                  "1h",
		JobEventEmitter:          publisherBackedJobEmitter{publisher: mocks.rabbitPublisher, exchange: "test-exchange"},
		JobEventStreamingEnabled: true,
	}

	uc.SetStorageEncryptDerivedKey([]byte("test-seaweedfs-encrypt-key-32by"))
	uc.SetCRMSecrets("test-crm-encrypt-key", "test-crm-hash-key")

	return uc
}

// newTestJobID returns a new UUID for testing.
func newTestJobID() uuid.UUID {
	return uuid.New()
}

// newTestOrgID returns a new UUID for testing.
func newTestOrgID() uuid.UUID {
	return uuid.New()
}

// testTracer returns a tracer for testing.
func testTracer() trace.Tracer {
	return otel.Tracer("test")
}

type publisherBackedJobEmitter struct {
	publisher publisherPort.Repository
	exchange  string
}

func (e publisherBackedJobEmitter) Emit(ctx context.Context, request streaming.EmitRequest) error {
	if e.publisher == nil || e.exchange == "" {
		return fmt.Errorf("job event streaming publisher is not configured")
	}

	if request.DefinitionKey == "" {
		return fmt.Errorf("missing definition key")
	}

	return e.publisher.Publish(ctx, e.exchange, request.DefinitionKey, request.Payload)
}

func (e publisherBackedJobEmitter) Close() error { return nil }

func (e publisherBackedJobEmitter) Healthy(context.Context) error { return nil }
