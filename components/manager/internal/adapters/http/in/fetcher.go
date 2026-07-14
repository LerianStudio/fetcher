package in

import (
	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/query"
)

// FetcherHandler handles HTTP requests for the fetcher API.
type FetcherHandler struct {
	CreateJobCmd *command.CreateFetcherJob
	GetJobQuery  *query.GetJob
}

// NewFetcherHandler creates a new FetcherHandler.
func NewFetcherHandler(createJobCmd *command.CreateFetcherJob, getJobQuery *query.GetJob) *FetcherHandler {
	return &FetcherHandler{
		CreateJobCmd: createJobCmd,
		GetJobQuery:  getJobQuery,
	}
}
