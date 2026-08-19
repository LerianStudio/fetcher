package in

import (
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	observability "github.com/LerianStudio/lib-observability/v2"
	libLog "github.com/LerianStudio/lib-observability/v2/log"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

func setupMigrationTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024,
	})

	app.Use(func(c fiber.Ctx) error {
		logger := &libLog.GoLogger{Level: libLog.LevelDebug}
		ctx := observability.ContextWithHeaderID(c.Context(), "test-request-id")
		ctx = observability.ContextWithLogger(ctx, logger)
		ctx = observability.ContextWithTracer(ctx, otel.Tracer("test"))
		c.SetContext(ctx)

		return c.Next()
	})

	return app
}

func createTestConnectionForMigration(id uuid.UUID) *model.Connection {
	now := time.Now().UTC()

	return &model.Connection{
		ID:                   id,
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
