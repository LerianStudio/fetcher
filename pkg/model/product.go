package model

import (
	"regexp"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"

	"github.com/google/uuid"
)

// codeRegex validates slug format: lowercase alphanumeric with hyphens, no leading/trailing hyphens.
var codeRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Product represents a Lerian product registered in Fetcher for datasource isolation.
type Product struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Code           string
	Name           string
	Description    string
	Metadata       *map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

// NewProduct creates a new Product with validation.
func NewProduct(
	organizationID uuid.UUID,
	code string,
	name string,
	description string,
	metadata *map[string]any,
) (*Product, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, pkg.ValidateInternalError(err, "product")
	}

	now := time.Now().UTC()

	product := &Product{
		ID:             id,
		OrganizationID: organizationID,
		Code:           code,
		Name:           name,
		Description:    description,
		Metadata:       metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return product, product.IsValid()
}

// IsValid trims and enforces required fields.
func (p *Product) IsValid() error {
	p.normalizeFields()

	requiredFields := p.validateRequiredFields()
	knownInvalidFields := p.validateFieldValues()

	if len(requiredFields) == 0 && len(knownInvalidFields) == 0 {
		return nil
	}

	return pkg.ValidateBadRequestFieldsError(
		requiredFields,
		knownInvalidFields,
		"product",
		nil,
	)
}

// normalizeFields trims whitespace from string fields.
func (p *Product) normalizeFields() {
	p.Code = strings.TrimSpace(strings.ToLower(p.Code))
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
}

// validateRequiredFields validates that all required fields are present.
func (p *Product) validateRequiredFields() map[string]string {
	requiredFields := make(map[string]string)

	if p.OrganizationID == uuid.Nil {
		requiredFields["organization_id"] = "organization ID is required"
	}

	if p.ID == uuid.Nil {
		requiredFields["id"] = "product ID is required"
	}

	if p.Code == "" {
		requiredFields["code"] = "code is required"
	}

	if p.Name == "" {
		requiredFields["name"] = "name is required"
	}

	return requiredFields
}

// validateFieldValues validates field values and formats.
func (p *Product) validateFieldValues() map[string]string {
	knownInvalidFields := make(map[string]string)

	if p.Code != "" && !codeRegex.MatchString(p.Code) {
		knownInvalidFields["code"] = "code must be lowercase alphanumeric with hyphens (slug format, e.g. 'my-product')"
	} else if p.Code != "" && (len(p.Code) < 2 || len(p.Code) > 50) {
		knownInvalidFields["code"] = "code must be between 2 and 50 characters"
	}

	if len(p.Name) > 100 {
		knownInvalidFields["name"] = "name must be at most 100 characters"
	}

	if len(p.Description) > 500 {
		knownInvalidFields["description"] = "description must be at most 500 characters"
	}

	return knownInvalidFields
}

// ApplyPatch applies partial updates to the Product. Code is immutable.
func (p *Product) ApplyPatch(
	name *string,
	description *string,
	metadata *map[string]any,
) error {
	if name != nil {
		p.Name = *name
	}

	if description != nil {
		p.Description = *description
	}

	if metadata != nil {
		p.Metadata = metadata
	}

	p.UpdatedAt = time.Now().UTC()

	return p.IsValid()
}

// ToMapWithMask converts the Product to a map for logging/telemetry.
func (p *Product) ToMapWithMask() map[string]any {
	return map[string]any{
		"id":              p.ID,
		"organization_id": p.OrganizationID,
		"code":            p.Code,
		"name":            p.Name,
		"description":     p.Description,
		"metadata":        p.Metadata,
		"created_at":      p.CreatedAt,
		"updated_at":      p.UpdatedAt,
		"deleted_at":      p.DeletedAt,
	}
}

// ##############################################################################################################################################################################
// Request, Response DTOs And Value Objects

// ProductInput is the DTO for POST /products requests.
type ProductInput struct {
	Code        string          `json:"code" validate:"required" example:"reporter" minLength:"2" maxLength:"50"`
	Name        string          `json:"name" validate:"required" example:"Reporter" minLength:"1" maxLength:"100"`
	Description string          `json:"description,omitempty" example:"Reporting product" maxLength:"500"`
	Metadata    *map[string]any `json:"metadata,omitempty"`
}

// IsEmpty returns true if all fields are empty/nil.
func (p *ProductInput) IsEmpty() bool {
	if p == nil {
		return true
	}

	return p.Code == "" &&
		p.Name == "" &&
		p.Description == "" &&
		p.Metadata == nil
}

// ProductUpdateInput is the DTO for PATCH /products/:id requests.
// All fields are pointers to distinguish between "not provided" (nil) and "provided with value".
type ProductUpdateInput struct {
	Name        *string         `json:"name,omitempty" validate:"omitempty,min=1,max=100" example:"Reporter" minLength:"1" maxLength:"100"`
	Description *string         `json:"description,omitempty" validate:"omitempty,max=500" example:"Reporting product" maxLength:"500"`
	Metadata    *map[string]any `json:"metadata,omitempty"`
}

// IsEmpty returns true if all fields are empty/nil.
func (p *ProductUpdateInput) IsEmpty() bool {
	if p == nil {
		return true
	}

	return p.Name == nil &&
		p.Description == nil &&
		p.Metadata == nil
}

// ProductResponse is the DTO for GET /products responses.
type ProductResponse struct {
	ID          uuid.UUID       `json:"id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Metadata    *map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// NewProductResponseFrom maps a Product to a ProductResponse.
func NewProductResponseFrom(p *Product) *ProductResponse {
	if p == nil {
		return nil
	}

	return &ProductResponse{
		ID:          p.ID,
		Code:        p.Code,
		Name:        p.Name,
		Description: p.Description,
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
