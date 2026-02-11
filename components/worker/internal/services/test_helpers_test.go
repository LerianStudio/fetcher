package services

import (
	"context"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"
	externalData "github.com/LerianStudio/fetcher/pkg/seaweedfs/external"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// testContext creates a context with logger and tracer for testing.
func testContext() context.Context {
	logger := testLogger()
	values := &libCommons.CustomContextKeyValue{
		HeaderID: "test-request-id",
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}

	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}

// testLogger creates a logger for testing that suppresses output.
func testLogger() *libLog.GoLogger {
	return &libLog.GoLogger{Level: libLog.ErrorLevel}
}

// testMocks holds all mock dependencies for testing.
type testMocks struct {
	ctrl            *gomock.Controller
	jobRepo         *jobRepo.MockRepository
	connRepo        *connRepo.MockRepository
	seaweedFS       *externalData.MockRepository
	cryptor         *crypto.MockCryptor
	rabbitPublisher *rabbitmq.MockPublisherRepository
}

// newTestMocks creates and returns all mock dependencies.
func newTestMocks(ctrl *gomock.Controller) *testMocks {
	return &testMocks{
		ctrl:            ctrl,
		jobRepo:         jobRepo.NewMockRepository(ctrl),
		connRepo:        connRepo.NewMockRepository(ctrl),
		seaweedFS:       externalData.NewMockRepository(ctrl),
		cryptor:         crypto.NewMockCryptor(ctrl),
		rabbitPublisher: rabbitmq.NewMockPublisherRepository(ctrl),
	}
}

// newTestUseCase creates a UseCase with all mocked dependencies.
// Now that UseCase uses interfaces, we can inject mocks directly.
func newTestUseCase(mocks *testMocks) *UseCase {
	return &UseCase{
		ExternalDataSeaweedFS: mocks.seaweedFS,
		JobRepository:         mocks.jobRepo,
		ConnectionRepository:  mocks.connRepo,
		Cryptor:               mocks.cryptor,
		FileTTL:               "1h",
		RabbitMQPublisher:     mocks.rabbitPublisher,
		JobEventsExchange:     "test-exchange",
	}
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
