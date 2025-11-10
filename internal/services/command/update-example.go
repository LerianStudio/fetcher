package query

import (
	"context"
	"errors"
	"golang-plugin-boilerplate/internal/services"
	"golang-plugin-boilerplate/pkg"
	"golang-plugin-boilerplate/pkg/constant"
	"golang-plugin-boilerplate/pkg/model"
	"reflect"

	"github.com/google/uuid"

	libCommons "github.com/LerianStudio/lib-commons/commons"
	libOtel "github.com/LerianStudio/lib-commons/commons/opentelemetry"
)

// UpdateExampleByID update an example from the repository.
func (ex *ExampleCommand) UpdateExampleByID(ctx context.Context, id uuid.UUID, uex *model.UpdateExampleInput) (*model.ExampleOutput, error) {
	logger := libCommons.NewLoggerFromContext(ctx)
	tracer := libCommons.NewTracerFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.update_example_by_id")
	defer span.End()

	logger.Infof("Trying to update example: %v", uex)

	example := &model.Example{
		Name: uex.Name,
		Age:  uex.Age,
	}

	organizationUpdated, err := ex.ExampleRepo.Update(ctx, id, example)
	if err != nil {
		libOtel.HandleSpanError(&span, "Failed to update organization on repo by id", err)

		logger.Errorf("Error updating organization on repo by id: %v", err)

		if errors.Is(err, services.ErrDatabaseItemNotFound) {
			return nil, pkg.ValidateBusinessError(constant.ErrEntityNotFound, reflect.TypeOf(model.Example{}).Name())
		}

		return nil, err
	}

	return organizationUpdated, nil
}
