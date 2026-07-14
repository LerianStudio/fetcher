package in

import (
	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/query"
)

type MigrationHandler struct {
	AssignCmd         *command.AssignConnection
	ListUnassignedQry *query.ListUnassignedConnections
}

func NewMigrationHandler(
	assignCmd *command.AssignConnection,
	listUnassignedQry *query.ListUnassignedConnections,
) *MigrationHandler {
	return &MigrationHandler{
		AssignCmd:         assignCmd,
		ListUnassignedQry: listUnassignedQry,
	}
}
