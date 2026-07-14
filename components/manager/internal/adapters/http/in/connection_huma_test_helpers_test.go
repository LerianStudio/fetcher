package in

import (
	"context"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/engine"
	pkgdatasource "github.com/LerianStudio/fetcher/v2/pkg/datasource"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	cacheRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/cache"
	"github.com/LerianStudio/fetcher/v2/pkg/testutil"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func setupConnectionTestApp() *fiber.App {
	app := fiber.New(fiber.Config{BodyLimit: 10 * 1024})
	app.Use(func(c *fiber.Ctx) error {
		logger := &libLog.GoLogger{Level: libLog.LevelDebug}
		ctx := observability.ContextWithHeaderID(testutil.TestContext(), "test-request-id")
		ctx = observability.ContextWithLogger(ctx, logger)
		ctx = observability.ContextWithTracer(ctx, otel.Tracer("test"))
		c.SetUserContext(ctx)

		return c.Next()
	})

	return app
}

func createTestConnection(id uuid.UUID) *model.Connection {
	now := time.Now().UTC()

	return &model.Connection{
		ID:                   id,
		ProductName:          "test-product",
		ConfigName:           "test-connection",
		Type:                 model.TypePostgreSQL,
		Host:                 "localhost",
		Port:                 5432,
		DatabaseName:         "testdb",
		Username:             "testuser",
		PasswordEncrypted:    "encrypted-password",
		EncryptionKeyVersion: "v1",
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func validConnectionInput() string {
	return `{
		"configName": "test-connection",
		"type": "POSTGRESQL",
		"host": "localhost",
		"port": 5432,
		"databaseName": "testdb",
		"username": "testuser",
		"password": "secretpassword"
	}`
}

func validSchemaValidationRequest() string {
	return `{
		"mappedFields": {
			"ds1": {
				"table1": ["field1", "field2"],
				"table2": ["field3"]
			}
		}
	}`
}

func newSchemaEngineForTest(
	t *testing.T,
	factory pkgdatasource.DataSourceFactory,
	cache cacheRepo.SchemaCacheRepository,
) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(schemaConnectorRegistryForTest{factory: schemacompat.NewConnectorFactory(factory, nil)}),
		engine.WithConnectionStore(schemacompat.NewConnectionStore()),
		engine.WithSchemaCache(schemacompat.NewSchemaCache(cache, 0)),
	)
	require.NoError(t, err)

	return eng
}

type schemaConnectorRegistryForTest struct {
	factory engine.ConnectorFactory
}

func (r schemaConnectorRegistryForTest) Connector(string) (engine.ConnectorFactory, bool) {
	return r.factory, true
}

type noopSchemaCache struct{}

func newNoopSchemaCache() *noopSchemaCache {
	return &noopSchemaCache{}
}

func (n *noopSchemaCache) Get(_ context.Context, _ string) (*model.DataSourceSchema, error) {
	return nil, nil
}

func (n *noopSchemaCache) Set(_ context.Context, _ string, _ *model.DataSourceSchema, _ time.Duration) error {
	return nil
}

func (n *noopSchemaCache) Delete(_ context.Context, _ string) error {
	return nil
}

func (n *noopSchemaCache) Clear(_ context.Context) error {
	return nil
}

func (n *noopSchemaCache) IsHealthy(_ context.Context) bool {
	return true
}

func (n *noopSchemaCache) Close() error {
	return nil
}

func TestNewConnectionHandlerPreservesDependencies(t *testing.T) {
	handler := NewConnectionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	assert.NotNil(t, handler)
	assert.Nil(t, handler.CreateCmd)
	assert.Nil(t, handler.UpdateCmd)
	assert.Nil(t, handler.DeleteCmd)
	assert.Nil(t, handler.GetQuery)
	assert.Nil(t, handler.ListQuery)
	assert.Nil(t, handler.TestQuery)
	assert.Nil(t, handler.ValidateSchemaQuery)
	assert.Nil(t, handler.GetSchemaQuery)
}
