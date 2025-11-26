package command

import (
	"context"
	"errors"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/connection"
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	domainConn "github.com/LerianStudio/fetcher/pkg/domain"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

type CreateConnection struct {
	repo   domainConn.Repository
	crypto crypto.Service
}

func NewCreateConnection(repo domainConn.Repository, cryptoSvc crypto.Service) *CreateConnection {
	return &CreateConnection{
		repo:   repo,
		crypto: cryptoSvc,
	}
}

func (s *CreateConnection) Execute(ctx context.Context, orgID uuid.UUID, in connection.ConnectionInput) (*domainConn.Connection, error) {
	entity, err := domainConn.NewWithPlain(ctx, s.crypto, domainConn.CreateParams{
		OrganizationID: orgID,
		ConfigName:     in.ConfigName,
		Type:           in.Type,
		Host:           in.Host,
		Port:           in.Port,
		DatabaseName:   in.DatabaseName,
		Username:       in.Username,
		Password:       in.Password,
		SSL:            connection.SSLInputToModel(in.SSL),
	})
	if err != nil {
		return nil, connection.ValidationError(err.Error())
	}

	if existing, errRepo := s.repo.FindByOrganizationAndName(ctx, entity.OrganizationID, entity.ConfigName); errRepo == nil && existing != nil {
		return nil, connection.ConflictError("a connection with this configName already exists")
	} else if errRepo != nil && !errors.Is(errRepo, mongo.ErrNoDocuments) {
		return nil, pkg.ValidateInternalError(errRepo, "connection")
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return created, nil
}
