package constant

const (
	ApplicationName = "fetcher"
	HeaderJobID     = "jobId"

	// Module constants for multi-tenant service identity.
	ModuleManager = "fetcher-manager"
	ModuleWorker  = "fetcher-worker"

	// ModuleFetcherOperationalState is the shared Tenant Manager module key for
	// Fetcher's MongoDB operational state. Manager creates connections/jobs and
	// Worker consumes job IDs then reads/writes the same collections, so both
	// components must resolve the same tenant database for MongoDB operational data.
	ModuleFetcherOperationalState = ApplicationName
)
