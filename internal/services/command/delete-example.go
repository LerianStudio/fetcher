package query

import (
	"context"
	"errors"
	"golang-plugin-boilerplate/internal/services"
	"golang-plugin-boilerplate/pkg"
	"golang-plugin-boilerplate/pkg/constant"
	"golang-plugin-boilerplate/pkg/model"
	"reflect"

	libCommons "github.com/LerianStudio/lib-commons/commons"
	libOtel "github.com/LerianStudio/lib-commons/commons/opentelemetry"
	"github.com/google/uuid"
)

// DeleteExampleByID fetch a new example from the repository
func (ex *ExampleCommand) DeleteExampleByID(ctx context.Context, id uuid.UUID) error {
	logger := libCommons.NewLoggerFromContext(ctx)
	tracer := libCommons.NewTracerFromContext(ctx)

	ctx, span := tracer.Start(ctx, "example_command.delete_example_by_id")
	defer span.End()

	logger.Infof("Remove example for id: %s", id)

	if err := ex.ExampleRepo.Delete(ctx, id); err != nil {
		libOtel.HandleSpanError(&span, "Failed to delete example on repo by id", err)

		logger.Errorf("Error deleting example on repo by id: %v", err)

		if errors.Is(err, services.ErrDatabaseItemNotFound) {
			return pkg.ValidateBusinessError(constant.ErrEntityNotFound, reflect.TypeOf(model.Example{}).Name())
		}

		return err
	}

	return nil
}
