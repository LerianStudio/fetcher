// Package job defines the domain port interface for job repositories.
package job

import (
	"context"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/model"

	"github.com/google/uuid"
)

// ListFilter controls pagination and filtering for job listings.
type ListFilter struct {
	Status        model.JobStatus
	Statuses      []model.JobStatus
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	CompletedFrom *time.Time
	CompletedTo   *time.Time
	Limit         int
	Page          int
	SortOrder     constant.Order
}

// Repository defines the domain port for jobs.
//
//go:generate mockgen --destination=repository.mock.go --package=job . Repository
type Repository interface {
	Create(ctx context.Context, job *model.Job) (*model.Job, error)
	Update(ctx context.Context, job *model.Job) (*model.Job, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Job, error)
	FindByRequestHashWithinWindow(ctx context.Context, requestHash string, windowMinutes int) (*model.Job, error)
	FindActiveByRequestHash(ctx context.Context, requestHash string) (*model.Job, error)
	List(ctx context.Context, filters *ListFilter) ([]*model.Job, error)
	ExistsRunningByMappedFieldKey(ctx context.Context, keyPattern string) (bool, error)
}
