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

// GetExampleByID fetch a new example from the repository
func (ex *ExampleQuery) GetExampleByID(ctx context.Context, id uuid.UUID) (*model.ExampleOutput, error) {
	logger := libCommons.NewLoggerFromContext(ctx)
	tracer := libCommons.NewTracerFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.get_example_by_id")
	defer span.End()

	logger.Infof("Retrieving example for id: %s", id.String())

	example, err := ex.ExampleRepo.Find(ctx, id)
	if err != nil {
		libOtel.HandleSpanError(&span, "Failed to get example on repo by id", err)

		logger.Errorf("Error getting example on repo by id: %v", err)

		if errors.Is(err, services.ErrDatabaseItemNotFound) {
			return nil, pkg.ValidateBusinessError(constant.ErrEntityNotFound, reflect.TypeOf(model.Example{}).Name())
		}

		return nil, err
	}

	return example, nil
}
