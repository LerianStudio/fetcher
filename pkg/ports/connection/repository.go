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
	Delete(ctx context.Context, id, organizationID uuid.UUID, deletedAt time.Time) error
	FindByID(ctx context.Context, id, organizationID uuid.UUID) (*model.Connection, error)
	FindByOrganizationAndName(ctx context.Context, organizationID uuid.UUID, configName string) (*model.Connection, error)
	FindByOrganizationAndDatabaseName(ctx context.Context, organizationID uuid.UUID, databaseName string) (*model.Connection, error)
	FindByConfigNames(ctx context.Context, organizationID uuid.UUID, configNames []string) ([]*model.Connection, error)
	List(ctx context.Context, organizationID uuid.UUID, filters http.QueryHeader) ([]*model.Connection, int64, error)
	ListUnassigned(ctx context.Context, organizationID uuid.UUID, filters http.QueryHeader) ([]*model.Connection, int64, error)
	AssignProductName(ctx context.Context, connectionID, organizationID uuid.UUID, productName string) (*model.Connection, error)
}
