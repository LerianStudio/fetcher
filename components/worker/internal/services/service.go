package services

import (
	externalConnection "github.com/LerianStudio/fetcher/pkg/mongodb/externalConnections"
	externalData "github.com/LerianStudio/fetcher/pkg/seaweedfs/external_data"
)

// UseCase is a struct that coordinates the handling of template files, report storage, external data sources, and report data.
type UseCase struct {
	// ExternalDataSeaweedFS is a repository used to retrieve external data from SeaweedFS storage.
	ExternalDataSeaweedFS externalData.Repository

	// ExternalDataSources holds a map of external data sources identified by their names, each mapped to a DataSource object.
	ExternalConnectionMongoDB externalConnection.Repository

	// FileTTL defines the Time To Live for file (e.g., "1m", "1h", "7d", "30d"). Empty means no TTL.
	FileTTL string
}
