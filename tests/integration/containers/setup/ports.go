package setup

// PortsConfig holds fixed port numbers for all infrastructure services.
// Using fixed ports enables:
// 1. Infrastructure reuse between test runs
// 2. Consistent VS Code debug configurations
// 3. Easier troubleshooting
type PortsConfig struct {
	// Infrastructure services
	MongoMain      string // Main MongoDB for fetcher-db
	MongoExternal  string // External MongoDB for test data
	RabbitMQ       string // RabbitMQ AMQP port
	SeaweedFSFiler string // SeaweedFS Filer HTTP port
	Redis          string // Redis/Valkey port

	// External databases
	Postgres  string // PostgreSQL port
	MySQL     string // MySQL port
	SQLServer string // SQL Server port
	Oracle    string // Oracle port

	// Applications
	Manager string // Manager API port
}

// FixedPorts contains the standard ports for integration tests.
// These match the ports typically used in development environments.
var FixedPorts = PortsConfig{
	// Infrastructure - using standard ports
	MongoMain:      "27017",
	MongoExternal:  "27018",
	RabbitMQ:       "5672",
	SeaweedFSFiler: "8888",
	Redis:          "6379",

	// External databases - using standard ports
	Postgres:  "5432",
	MySQL:     "3306",
	SQLServer: "1433",
	Oracle:    "1521",

	// Applications
	Manager: "4006",
}

// InfraConfigPath is the path where infrastructure configuration is saved.
// Tests check this file to detect already-running infrastructure.
const InfraConfigPath = "/tmp/fetcher-test-infra.json"
