package in

import (
	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/query"
)

type ConnectionHandler struct {
	CreateCmd           *command.CreateConnection
	UpdateCmd           *command.UpdateConnection
	DeleteCmd           *command.DeleteConnection
	GetQuery            *query.GetConnection
	ListQuery           *query.ListConnections
	TestQuery           *query.TestConnection
	ValidateSchemaQuery *query.ValidateSchema
	GetSchemaQuery      *query.GetConnectionSchema
}

func NewConnectionHandler(
	createCmd *command.CreateConnection,
	updateCmd *command.UpdateConnection,
	deleteCmd *command.DeleteConnection,
	getQuery *query.GetConnection,
	listQuery *query.ListConnections,
	testQuery *query.TestConnection,
	validateSchemaQuery *query.ValidateSchema,
	getSchemaQuery *query.GetConnectionSchema,
) *ConnectionHandler {
	return &ConnectionHandler{
		CreateCmd:           createCmd,
		UpdateCmd:           updateCmd,
		DeleteCmd:           deleteCmd,
		GetQuery:            getQuery,
		ListQuery:           listQuery,
		TestQuery:           testQuery,
		ValidateSchemaQuery: validateSchemaQuery,
		GetSchemaQuery:      getSchemaQuery,
	}
}
