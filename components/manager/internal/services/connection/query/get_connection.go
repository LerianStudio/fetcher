package query

import (
	"context"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/connection"
	"github.com/LerianStudio/fetcher/pkg"
	domainConn "github.com/LerianStudio/fetcher/pkg/domain"

	"github.com/google/uuid"
)

type GetConnection struct {
	repo domainConn.Repository
}

func NewGetConnection(repo domainConn.Repository) *GetConnection {
	return &GetConnection{repo: repo}
}

func (s *GetConnection) Execute(ctx context.Context, orgID, id uuid.UUID) (*domainConn.Connection, error) {
	if orgID == uuid.Nil || id == uuid.Nil {
		return nil, connection.ValidationError("invalid organization or connection id")
	}

	current, err := s.repo.FindByID(ctx, id, orgID)
	if err != nil {
		return nil, pkg.ValidateInternalError(err, "connection")
	}
	if current == nil {
		return nil, connection.NotFoundError()
	}

	return current, nil
}
