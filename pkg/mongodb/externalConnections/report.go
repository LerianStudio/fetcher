package report

import (
	"time"

	"github.com/google/uuid"
)

type ExternalConnection struct {
	ID          uuid.UUID  `json:"id" example:"00000000-0000-0000-0000-000000000000"`
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt"`
}

type ExternalConnectionMongoDBModel struct {
	ID             uuid.UUID  `bson:"_id"`
	OrganizationID uuid.UUID  `bson:"organization_id"`
	CompletedAt    *time.Time `bson:"completed_at"`
	CreatedAt      time.Time  `bson:"created_at"`
	UpdatedAt      time.Time  `bson:"updated_at"`
	DeletedAt      *time.Time `bson:"deleted_at"`
}

// ToEntity converts ReportMongoDBModel to Report
func (rm *ExternalConnectionMongoDBModel) ToEntity() *ExternalConnection {
	return &ExternalConnection{
		ID:          rm.ID,
		CompletedAt: rm.CompletedAt,
		CreatedAt:   rm.CreatedAt,
		UpdatedAt:   rm.UpdatedAt,
		DeletedAt:   rm.DeletedAt,
	}
}

// ToEntityFindByID converts ReportMongoDBModel to Report
func (rm *ExternalConnectionMongoDBModel) ToEntityFindByID() *ExternalConnection {
	return &ExternalConnection{
		ID:          rm.ID,
		CompletedAt: rm.CompletedAt,
		CreatedAt:   rm.CreatedAt,
		UpdatedAt:   rm.UpdatedAt,
		DeletedAt:   rm.DeletedAt,
	}
}

// FromEntity converts Report to ReportMongoDBModel
func (rm *ExternalConnectionMongoDBModel) FromEntity(r *ExternalConnection, organizationID uuid.UUID) error {
	dateNow := time.Now()
	rm.ID = r.ID
	rm.OrganizationID = organizationID
	rm.CompletedAt = nil
	rm.CreatedAt = dateNow
	rm.UpdatedAt = dateNow
	rm.DeletedAt = nil

	return nil
}
