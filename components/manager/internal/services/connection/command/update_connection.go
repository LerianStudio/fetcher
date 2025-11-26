package command

import (
	"context"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/connection"
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	domainConn "github.com/LerianStudio/fetcher/pkg/domain"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"

	"github.com/google/uuid"
)

type UpdateConnection struct {
	repo    domainConn.Repository
	jobRepo job.Repository
	crypto  crypto.Service
}

func NewUpdateConnection(repo domainConn.Repository, jobRepo job.Repository, cryptoSvc crypto.Service) *UpdateConnection {
	return &UpdateConnection{
		repo:    repo,
		jobRepo: jobRepo,
		crypto:  cryptoSvc,
	}
}

func (s *UpdateConnection) Execute(ctx context.Context, orgID, id uuid.UUID, in connection.ConnectionInput) (*domainConn.Connection, error) {
	if orgID == uuid.Nil || id == uuid.Nil {
		return nil, connection.ValidationError("invalid organization or connection id")
	}

	active, err := s.jobRepo.ExistsRunningByConnection(ctx, orgID, id)
	if err != nil {
		return nil, pkg.ValidateInternalError(err, "connection")
	}
	if active {
		return nil, connection.ConflictError("active job exists for this connection, cannot update now")
	}

	current, err := s.repo.FindByID(ctx, id, orgID)
	if err != nil {
		return nil, pkg.ValidateInternalError(err, "connection")
	}
	if current == nil {
		return nil, connection.NotFoundError()
	}

	if errPatch := current.ApplyPatch(
		ctx,
		s.crypto,
		&in.ConfigName,
		&in.Type,
		&in.Host,
		&in.Port,
		&in.DatabaseName,
		&in.Username,
		&in.Password,
		connection.SSLInputToModel(in.SSL),
	); errPatch != nil {
		return nil, connection.ValidationError(errPatch.Error())
	}

	// TODO: Test the database connection with the new data before persisting. If it fails, return the failure to the user; only update when the connection test passes.

	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return updated, nil
}
