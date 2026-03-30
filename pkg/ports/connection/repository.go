// Package connection defines the domain port interface for connection repositories.
package connection

import (
	"context"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/google/uuid"
)

// Repository defines the domain port for connections.
//
//go:generate mockgen --destination=repository.mock.go --package=connection . Repository
type Repository interface {
	Create(ctx context.Context, conn *model.Connection) (*model.Connection, error)
	Update(ctx context.Context, conn *model.Connection) (*model.Connection, error)
	Delete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Connection, error)
	FindByName(ctx context.Context, configName string) (*model.Connection, error)
	FindByDatabaseName(ctx context.Context, databaseName string) (*model.Connection, error)
	FindByConfigNames(ctx context.Context, configNames []string) ([]*model.Connection, error)
	List(ctx context.Context, filters http.QueryHeader) ([]*model.Connection, int64, error)
	ListUnassigned(ctx context.Context, filters http.QueryHeader) ([]*model.Connection, int64, error)
	AssignProductName(ctx context.Context, connectionID uuid.UUID, productName string) (*model.Connection, error)
}
