package product

import (
	"errors"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/google/uuid"
)

// ProductMongoDBModel represents how a product is stored in MongoDB.
type ProductMongoDBModel struct {
	ID             uuid.UUID      `bson:"_id"`
	OrganizationID uuid.UUID      `bson:"organization_id"`
	Code           string         `bson:"code"`
	Name           string         `bson:"name"`
	Description    string         `bson:"description,omitempty"`
	Metadata       map[string]any `bson:"metadata,omitempty"`
	CreatedAt      time.Time      `bson:"created_at"`
	UpdatedAt      time.Time      `bson:"updated_at"`
	DeletedAt      *time.Time     `bson:"deleted_at"`
}

// ToEntity converts a MongoDB model into the domain entity representation.
func (pm *ProductMongoDBModel) ToEntity() (*model.Product, error) {
	if pm == nil {
		return nil, errors.New("cannot convert nil ProductMongoDBModel to domain")
	}

	var metadata *map[string]any
	if len(pm.Metadata) > 0 {
		metadata = &pm.Metadata
	}

	return &model.Product{
		ID:             pm.ID,
		OrganizationID: pm.OrganizationID,
		Code:           pm.Code,
		Name:           pm.Name,
		Description:    pm.Description,
		Metadata:       metadata,
		CreatedAt:      pm.CreatedAt,
		UpdatedAt:      pm.UpdatedAt,
		DeletedAt:      pm.DeletedAt,
	}, nil
}

// FromEntity populates the MongoDB model from a domain entity.
func (pm *ProductMongoDBModel) FromEntity(p *model.Product) error {
	if p == nil {
		return errors.New("product entity is required")
	}

	pm.ID = p.ID
	pm.OrganizationID = p.OrganizationID
	pm.Code = p.Code
	pm.Name = p.Name
	pm.Description = p.Description

	if p.Metadata != nil {
		pm.Metadata = *p.Metadata
	}

	pm.CreatedAt = p.CreatedAt
	pm.UpdatedAt = p.UpdatedAt
	pm.DeletedAt = p.DeletedAt

	return nil
}

// NewProductMongoDBModelFromDomain creates a MongoDB model from the domain entity.
func NewProductMongoDBModelFromDomain(p *model.Product) (*ProductMongoDBModel, error) {
	pm := &ProductMongoDBModel{}
	if err := pm.FromEntity(p); err != nil {
		return nil, err
	}

	return pm, nil
}

// ToMapWithMask converts the MongoDB model to a map for logging/telemetry.
func (pm *ProductMongoDBModel) ToMapWithMask() map[string]any {
	return map[string]any{
		"id":              pm.ID,
		"organization_id": pm.OrganizationID,
		"code":            pm.Code,
		"name":            pm.Name,
		"description":     pm.Description,
		"metadata":        pm.Metadata,
		"created_at":      pm.CreatedAt,
		"updated_at":      pm.UpdatedAt,
		"deleted_at":      pm.DeletedAt,
	}
}
