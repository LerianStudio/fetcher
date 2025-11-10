package query

import (
	"context"
	"errors"
	servicesExample "golang-plugin-boilerplate/internal/services"
	"golang-plugin-boilerplate/pkg"
	"golang-plugin-boilerplate/pkg/constant"
	"golang-plugin-boilerplate/pkg/model"
	"golang-plugin-boilerplate/pkg/net/http"
	"reflect"

	libCommons "github.com/LerianStudio/lib-commons/commons"
	libOtel "github.com/LerianStudio/lib-commons/commons/opentelemetry"
)

// GetAllExample fetch all Examples from the repository
func (ex *ExampleQuery) GetAllExample(ctx context.Context, filter http.QueryHeader) ([]*model.ExampleOutput, error) {
	logger := libCommons.NewLoggerFromContext(ctx)
	tracer := libCommons.NewTracerFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.get_all_examples")
	defer span.End()

	logger.Infof("Retrieving examples")

	examples, err := ex.ExampleRepo.FindAll(ctx, filter.ToOffsetPagination())
	if err != nil {
		libOtel.HandleSpanError(&span, "Failed to get examples on repo", err)

		logger.Errorf("Error getting examples on repo: %v", err)

		if errors.Is(err, servicesExample.ErrDatabaseItemNotFound) {
			return nil, pkg.ValidateBusinessError(constant.ErrEntityNotFound, reflect.TypeOf(model.Example{}).Name())
		}

		return nil, err
	}

	return examples, nil
}
