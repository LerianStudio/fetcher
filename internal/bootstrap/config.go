package bootstrap

import (
	"fmt"
	grcpcin "golang-plugin-boilerplate/internal/adapters/grpc/in"
	"golang-plugin-boilerplate/internal/adapters/http/in"
	"golang-plugin-boilerplate/internal/adapters/postgres/example"
	command "golang-plugin-boilerplate/internal/services/command"
	query "golang-plugin-boilerplate/internal/services/query"
	"golang-plugin-boilerplate/pkg"

	libOtel "github.com/LerianStudio/lib-commons/commons/opentelemetry"
	libPostgres "github.com/LerianStudio/lib-commons/commons/postgres"
	libZap "github.com/LerianStudio/lib-commons/commons/zap"
)

const ApplicationName = "example-boilerplate"

// Config is the top level configuration struct for the entire application.
type Config struct {
	EnvName                 string `env:"ENV_NAME"`
	ProtoAddress            string `env:"PROTO_ADDRESS"`
	ServerAddress           string `env:"SERVER_ADDRESS"`
	LogLevel                string `env:"LOG_LEVEL"`
	OtelServiceName         string `env:"OTEL_RESOURCE_SERVICE_NAME"`
	OtelLibraryName         string `env:"OTEL_LIBRARY_NAME"`
	OtelServiceVersion      string `env:"OTEL_RESOURCE_SERVICE_VERSION"`
	OtelDeploymentEnv       string `env:"OTEL_RESOURCE_DEPLOYMENT_ENVIRONMENT"`
	OtelColExporterEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	EnableTelemetry         bool   `env:"ENABLE_TELEMETRY"`
	PrimaryDBHost           string `env:"DB_HOST"`
	PrimaryDBUser           string `env:"DB_USER"`
	PrimaryDBPassword       string `env:"DB_PASSWORD"`
	PrimaryDBName           string `env:"DB_NAME"`
	PrimaryDBPort           string `env:"DB_PORT"`
	MigrationPath           string `env:"MIGRATIONS_PATH"`
	ReplicaDBHost           string `env:"DB_REPLICA_HOST"`
	ReplicaDBUser           string `env:"DB_REPLICA_USER"`
	ReplicaDBPassword       string `env:"DB_REPLICA_PASSWORD"`
	ReplicaDBName           string `env:"DB_REPLICA_NAME"`
	ReplicaDBPort           string `env:"DB_REPLICA_PORT"`
	ValkeyHost              string `env:"VALKEY_HOST"`
	ValkeyPort              string `env:"VALKEY_PORT"`
	ValkeyUser              string `env:"VALKEY_USER"`
	ValkeyPassword          string `env:"VALKEY_PASSWORD"`
}

// InitServers initiate http and grpc servers.
func InitServers() *Service {
	cfg := &Config{}

	if err := pkg.SetConfigFromEnvVars(cfg); err != nil {
		panic(err)
	}

	logger := libZap.InitializeLogger()

	// Init Open telemetry to control logs and flows
	telemetry := &libOtel.Telemetry{
		LibraryName:               cfg.OtelLibraryName,
		ServiceName:               cfg.OtelServiceName,
		ServiceVersion:            cfg.OtelServiceVersion,
		DeploymentEnv:             cfg.OtelDeploymentEnv,
		CollectorExporterEndpoint: cfg.OtelColExporterEndpoint,
		EnableTelemetry:           cfg.EnableTelemetry,
	}

	// Init database connection
	postgresSQLSourcePrimary := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.PrimaryDBHost, cfg.PrimaryDBUser, cfg.PrimaryDBPassword, cfg.PrimaryDBName, cfg.PrimaryDBPort)

	postgresSQLSourceReplica := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.ReplicaDBHost, cfg.ReplicaDBUser, cfg.ReplicaDBPassword, cfg.ReplicaDBName, cfg.ReplicaDBPort)

	postgresConnection := &libPostgres.PostgresConnection{
		ConnectionStringPrimary: postgresSQLSourcePrimary,
		ConnectionStringReplica: postgresSQLSourceReplica,
		PrimaryDBName:           cfg.PrimaryDBName,
		ReplicaDBName:           cfg.ReplicaDBName,
		Component:               ApplicationName,
		MigrationsPath:          cfg.MigrationPath,
		Logger:                  logger,
	}

	/* Init Valkey Cache

	valkeySource := fmt.Sprintf("%s:%s", cfg.ValkeyHost, cfg.ValkeyPort)

	valkeyConnection := &mredis.ValkeyConnection{
		Addr:     valkeySource,
		User:     cfg.ValkeyUser,
		Password: cfg.ValkeyPassword,
		DB:       0,
		Protocol: 3,
		Logger:   logger,
	}

	valkeyConsumerRepository := redis.NewConsumerRedis(redisConnection)

	*/

	examplePostgreSQLRepository := example.NewExamplePostgresSQLRepository(postgresConnection)

	exampleCommand := &command.ExampleCommand{
		ExampleRepo: examplePostgreSQLRepository,
	}

	exampleQuery := &query.ExampleQuery{
		ExampleRepo: examplePostgreSQLRepository,
	}

	exampleHandler := &in.ExampleHandler{
		ExampleCommand: exampleCommand,
		ExampleQuery:   exampleQuery,
	}

	httpApp := in.NewRoutes(logger, telemetry, exampleHandler)
	serverAPI := NewServer(cfg, httpApp, logger, telemetry)

	grpcApp := grcpcin.NewRouterGRPC(logger, telemetry, exampleQuery, exampleCommand)
	serverGRPC := NewServerGRPC(cfg, grpcApp, logger, telemetry)

	return &Service{
		Server:     serverAPI,
		ServerGRPC: serverGRPC,
		Logger:     logger,
	}
}
