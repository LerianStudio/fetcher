package services

import (
	"github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	externalData "github.com/LerianStudio/fetcher/pkg/seaweedfs/external_data"
)

// UseCase is a struct that coordinates the handling of template files, report storage, external data sources, and report data.
type UseCase struct {
	// ExternalDataSeaweedFS is a repository used to retrieve external data from SeaweedFS storage.
	ExternalDataSeaweedFS externalData.Repository

	// JobRepository is a repository used to retrieve job data from MongoDB storage.
	JobRepository *job.JobMongoDBRepository

	// ConnectionRepository is a repository used to retrieve connection data from MongoDB storage.
	ConnectionRepository *connection.ConnectionMongoDBRepository

	// FileTTL defines the Time To Live for file (e.g., "1m", "1h", "7d", "30d"). Empty means no TTL.
	FileTTL string
}
