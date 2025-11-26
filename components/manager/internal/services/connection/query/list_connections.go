package query

import (
	"context"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/connection"
	"github.com/LerianStudio/fetcher/pkg"
	domainConn "github.com/LerianStudio/fetcher/pkg/domain"

	"github.com/google/uuid"
)

type ListConnections struct {
	repo domainConn.Repository
}

func NewListConnections(repo domainConn.Repository) *ListConnections {
	return &ListConnections{repo: repo}
}

func (s *ListConnections) Execute(ctx context.Context, orgID uuid.UUID, in connection.ListConnectionsInput) ([]*domainConn.Connection, error) {
	filter, err := domainConn.NewListFilter(
		orgID,
		in.ConfigName,
		in.Host,
		in.DatabaseName,
		in.Type,
		in.SortOrder,
		in.CreatedAt,
		in.Page,
		in.Limit,
	)
	if err != nil {
		return nil, connection.ValidationError(err.Error())
	}

	list, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return list, nil
}
