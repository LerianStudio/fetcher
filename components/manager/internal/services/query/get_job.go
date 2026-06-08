// Package query provides query services for the manager component following CQRS pattern.
package query

import (
	"context"
	"fmt"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	jobRepo "github.com/LerianStudio/fetcher/pkg/ports/job"

	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type GetJob struct {
	jobRepo jobRepo.Repository
}

func NewGetJob(jobRepository jobRepo.Repository) *GetJob {
	return &GetJob{jobRepo: jobRepository}
}

func (s *GetJob) Execute(ctx context.Context, jobID uuid.UUID) (*model.Job, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.get_job")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.job_id", jobID.String()),
	)

	job, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to find job by ID", err)
		return nil, fmt.Errorf("failed to find job by id: %w", err)
	}

	if job == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"job",
		)
	}

	return job, nil
}
