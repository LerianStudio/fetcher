// Package query provides query services for the manager component following CQRS pattern.
package query

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"

	"github.com/LerianStudio/lib-commons/v4/commons"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type GetJob struct {
	jobRepo jobRepo.Repository
}

func NewGetJob(jobRepository jobRepo.Repository) *GetJob {
	return &GetJob{jobRepo: jobRepository}
}

func (s *GetJob) Execute(ctx context.Context, organizationID, jobID uuid.UUID) (*model.Job, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.get_job")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.job_id", jobID.String()),
	)

	job, err := s.jobRepo.FindByID(ctx, jobID, organizationID)
	if err != nil {
		return nil, err
	}

	if job == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"job",
		)
	}

	return job, nil
}
