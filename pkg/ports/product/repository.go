// Package product defines the domain port interface for product repositories.
package product

import (
	"context"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/google/uuid"
)

// Repository defines the domain port for products.
//
//go:generate mockgen --destination=repository.mock.go --package=product . Repository
type Repository interface {
	Create(ctx context.Context, product *model.Product) (*model.Product, error)
	Update(ctx context.Context, product *model.Product) (*model.Product, error)
	Delete(ctx context.Context, id, organizationID uuid.UUID, deletedAt time.Time) error
	FindByID(ctx context.Context, id, organizationID uuid.UUID) (*model.Product, error)
	FindByCode(ctx context.Context, code string, organizationID uuid.UUID) (*model.Product, error)
	List(ctx context.Context, organizationID uuid.UUID, filters http.QueryHeader) ([]*model.Product, int64, error)
}
