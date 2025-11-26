package command

import (
	"context"
	"time"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/connection"
	"github.com/LerianStudio/fetcher/pkg"
	domainConn "github.com/LerianStudio/fetcher/pkg/domain"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"

	"github.com/google/uuid"
)

type DeleteConnection struct {
	repo    domainConn.Repository
	jobRepo job.Repository
}

func NewDeleteConnection(repo domainConn.Repository, jobRepo job.Repository) *DeleteConnection {
	return &DeleteConnection{
		repo:    repo,
		jobRepo: jobRepo,
	}
}

func (s *DeleteConnection) Execute(ctx context.Context, orgID, id uuid.UUID) error {
	if orgID == uuid.Nil || id == uuid.Nil {
		return connection.ValidationError("invalid organization or connection id")
	}

	active, err := s.jobRepo.ExistsRunningByConnection(ctx, orgID, id)
	if err != nil {
		return pkg.ValidateInternalError(err, "connection")
	}
	if active {
		return connection.ConflictError("active job exists for this connection, cannot delete now")
	}

	current, err := s.repo.FindByID(ctx, id, orgID)
	if err != nil {
		return pkg.ValidateInternalError(err, "connection")
	}
	if current == nil {
		return connection.NotFoundError()
	}

	if err := s.repo.Delete(ctx, id, orgID, time.Now().UTC()); err != nil {
		return pkg.ValidateInternalError(err, "connection")
	}

	return nil
}
