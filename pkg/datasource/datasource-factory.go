// Package datasource provides factory functions for creating database data source implementations
// from connection models, handling password decryption and connection string construction.
package datasource

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource/sslmode"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	datsourceMongoConfig "github.com/LerianStudio/fetcher/pkg/model/datasource/mongodb"
	datsourceMySQLConfig "github.com/LerianStudio/fetcher/pkg/model/datasource/mysql"
	datsourceOracleConfig "github.com/LerianStudio/fetcher/pkg/model/datasource/oracle"
	datsourcePostgresConfig "github.com/LerianStudio/fetcher/pkg/model/datasource/postgres"
	datsourceSQLServerConfig "github.com/LerianStudio/fetcher/pkg/model/datasource/sqlserver"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/fetcher/pkg/mysql"
	"github.com/LerianStudio/fetcher/pkg/oracle"
	"github.com/LerianStudio/fetcher/pkg/postgres"
	"github.com/LerianStudio/fetcher/pkg/sqlserver"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type dataSourceConfigBuilder func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error)

var (
	buildMongoDataSource dataSourceConfigBuilder = func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
		return newDataSourceConfigMongoDB(ctx, base, conn, cryptor, logger)
	}
	buildPostgresDataSource dataSourceConfigBuilder = func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
		return newDataSourceConfigPostgres(ctx, base, conn, cryptor, logger)
	}
	buildOracleDataSource dataSourceConfigBuilder = func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
		return newDataSourceConfigOracle(ctx, base, conn, cryptor, logger)
	}
	buildMySQLDataSource dataSourceConfigBuilder = func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
		return newDataSourceConfigMySQL(ctx, base, conn, cryptor, logger)
	}
	buildSQLServerDataSource dataSourceConfigBuilder = func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
		return newDataSourceConfigSQLServer(ctx, base, conn, cryptor, logger)
	}
)

var defaultDataSourceConfigBuilders = map[model.DBType]dataSourceConfigBuilder{
	model.TypeMongoDB: func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
		return buildMongoDataSource(ctx, base, conn, cryptor, logger)
	},
	model.TypePostgreSQL: func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
		return buildPostgresDataSource(ctx, base, conn, cryptor, logger)
	},
	model.TypeOracle: func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
		return buildOracleDataSource(ctx, base, conn, cryptor, logger)
	},
	model.TypeMySQL: func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
		return buildMySQLDataSource(ctx, base, conn, cryptor, logger)
	},
	model.TypeSQLServer: func(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
		return buildSQLServerDataSource(ctx, base, conn, cryptor, logger)
	},
}

// NewDataSourceFromConnection creates a DataSource implementation based on the connection type.
// This factory function encapsulates the logic for converting a Connection entity
// into the appropriate DataSourceConfig implementation (MongoDB, PostgreSQL, Oracle, MySQL, or SQL Server).
// The cryptor is required to decrypt the connection password before creating the data source.
func NewDataSourceFromConnection(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
	return newDataSourceFromConnection(ctx, conn, cryptor, logger, defaultDataSourceConfigBuilders)
}

func newDataSourceFromConnection(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger, builders map[model.DBType]dataSourceConfigBuilder) (datasource.DataSource, error) {
	if conn == nil {
		return nil, fmt.Errorf("connection cannot be nil")
	}

	if cryptor == nil {
		return nil, fmt.Errorf("cryptor cannot be nil")
	}

	baseConfig := newDataSourceConfigFromConnection(conn)

	builder, ok := builders[conn.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported database type: %s", conn.Type)
	}

	return builder(ctx, baseConfig, conn, cryptor, logger)
}

// newDataSourceConfigFromConnection creates a base DataSourceConfig from a Connection entity.
func newDataSourceConfigFromConnection(conn *model.Connection) datasource.DataSourceConfig {
	var sslConfig model.SSLConfig
	if conn.SSL != nil {
		sslConfig = *conn.SSL
	}

	return datasource.DataSourceConfig{
		ID:                conn.ID.String(),
		OrganizationID:    conn.OrganizationID.String(),
		ConfigName:        conn.ConfigName,
		Type:              strings.ToUpper(string(conn.Type)),
		Host:              conn.Host,
		Port:              fmt.Sprintf("%d", conn.Port),
		DatabaseName:      conn.DatabaseName,
		Username:          conn.Username,
		PasswordEncrypted: conn.PasswordEncrypted,
		SSL:               sslConfig,
		Status:            "",
	}
}

// newDataSourceConfigMongoDB creates a MongoDB-specific DataSourceConfig.
func newDataSourceConfigMongoDB(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (*datsourceMongoConfig.DataSourceConfigMongoDB, error) {
	// Decrypt password before using it in connection string
	password, err := conn.GetPasswordDecrypted(ctx, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password for MongoDB connection: %w", err)
	}

	optionsStr := "authSource=admin&directConnection=true"
	mongoURI := fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		strings.ToLower(string(conn.Type)),
		conn.Username,
		url.QueryEscape(password),
		conn.Host,
		conn.Port,
		conn.DatabaseName,
	)

	if optionsStr != "" {
		mongoURI += "?" + optionsStr
	}

	// Validate SSL mode to prevent injection attacks
	if conn.SSL != nil && conn.SSL.Mode != "" {
		if err := sslmode.ValidateMongoDBMode(conn.SSL.Mode); err != nil {
			return nil, err
		}
	}

	var params []string
	if conn.SSL != nil && conn.SSL.Mode != "" && conn.SSL.Mode != "false" && conn.SSL.Mode != "disable" {
		// Enable TLS for MongoDB connection
		params = append(params, "tls=true")
		// For self-signed certificates in local/test environments, we allow skipping verification.
		// In production, this should be avoided to prevent MitM attacks.
		if conn.SSL.Mode == "skip-verify" || conn.SSL.Mode == "insecure" {
			params = append(params, "tlsInsecure=true")
		}
	}

	if len(params) > 0 {
		if strings.Contains(mongoURI, "?") {
			mongoURI += "&" + strings.Join(params, "&")
		} else {
			mongoURI += "?" + strings.Join(params, "&")
		}
	}

	// Test connection with timeout
	testCtx, cancel := context.WithTimeout(ctx, constant.ConnectionTimeout)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(constant.MongoDBMaxPoolSize).
		SetMinPoolSize(constant.MongoDBMinPoolSize).
		SetMaxConnIdleTime(constant.MongoDBMaxConnIdleTime).
		SetConnectTimeout(constant.ConnectionTimeout).
		SetServerSelectionTimeout(constant.ConnectionTimeout)

	client, errConnect := mongo.Connect(testCtx, clientOpts)
	if errConnect != nil {
		logger.Log(testCtx, libLog.LevelError, fmt.Sprintf("Failed to connect to MongoDB [%s]: %v", conn.ConfigName, errConnect))
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", errConnect)
	}

	if errPing := client.Ping(testCtx, nil); errPing != nil {
		_ = client.Disconnect(testCtx)

		logger.Log(testCtx, libLog.LevelError, fmt.Sprintf("Failed to ping MongoDB [%s]: %v", conn.ConfigName, errPing))

		return nil, fmt.Errorf("failed to ping MongoDB: %w", errPing)
	}

	_ = client.Disconnect(testCtx)

	// Create repository
	repo, errRepo := mongodb.NewDataSourceRepository(mongoURI, conn.DatabaseName, logger)
	if errRepo != nil {
		return nil, fmt.Errorf("failed to create MongoDB repository: %w", errRepo)
	}

	return &datsourceMongoConfig.DataSourceConfigMongoDB{
		DataSourceConfig:  base,
		MongoDBRepository: repo,
		MongoURI:          mongoURI,
		Options:           optionsStr,
	}, nil
}

// newDataSourceConfigPostgres creates a PostgreSQL-specific DataSourceConfig.
func newDataSourceConfigPostgres(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (*datsourcePostgresConfig.DataSourceConfigPostgres, error) {
	// Decrypt password before using it in connection string
	password, err := conn.GetPasswordDecrypted(ctx, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password for PostgreSQL connection: %w", err)
	}

	sslMode := "disable"
	if conn.SSL != nil && conn.SSL.Mode != "" {
		sslMode = conn.SSL.Mode
	}

	// Validate SSL mode to prevent injection attacks
	if err := sslmode.ValidatePostgreSQLMode(sslMode); err != nil {
		return nil, err
	}

	connectionString := fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=%s",
		strings.ToLower(string(conn.Type)),
		conn.Username,
		url.QueryEscape(password),
		conn.Host,
		conn.Port,
		conn.DatabaseName,
		sslMode,
	)

	pgConnection := &postgres.Connection{
		ConnectionString:   connectionString,
		DBName:             conn.DatabaseName,
		Logger:             logger,
		MaxOpenConnections: constant.PostgresMaxOpenConns,
		MaxIdleConnections: constant.PostgresMaxIdleConns,
	}

	if errConnect := pgConnection.Connect(); errConnect != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to connect to Postgres [%s]: %v", conn.ConfigName, errConnect))
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", errConnect)
	}

	// Create repository
	repo, errRepo := postgres.NewDataSourceRepository(pgConnection)
	if errRepo != nil {
		_ = pgConnection.ConnectionDB.Close()
		return nil, fmt.Errorf("failed to create PostgreSQL repository: %w", errRepo)
	}

	return &datsourcePostgresConfig.DataSourceConfigPostgres{
		DataSourceConfig:   base,
		PostgresConnection: pgConnection,
		PostgresRepository: repo,
	}, nil
}

// newDataSourceConfigOracle creates an Oracle-specific DataSourceConfig.
func newDataSourceConfigOracle(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (*datsourceOracleConfig.DataSourceConfigOracle, error) {
	// Decrypt password before using it in connection string
	password, err := conn.GetPasswordDecrypted(ctx, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password for Oracle connection: %w", err)
	}

	serviceName := ""

	if conn.Metadata != nil {
		// Try to get serviceName from metadata map if exists
		if service, ok := (*conn.Metadata)["serviceName"].(string); ok && service != "" {
			serviceName = service
		}
	}

	if serviceName == "" {
		return nil, fmt.Errorf("serviceName is required in metadata for Oracle connection")
	}

	// Validate SSL mode if provided to prevent injection attacks
	if conn.SSL != nil && conn.SSL.Mode != "" {
		if err := sslmode.ValidateOracleMode(conn.SSL.Mode); err != nil {
			return nil, err
		}
	}

	connectionString := fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		conn.Username,
		url.QueryEscape(password),
		conn.Host,
		conn.Port,
		serviceName,
	)

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Attempting Oracle connection [%s] - Service: %s, Username: %s (will access schema: %s)",
		conn.ConfigName, serviceName, conn.Username, strings.ToUpper(conn.Username)))
	logger.Log(ctx, libLog.LevelDebug, fmt.Sprintf("Oracle connection string: oracle://%s:***@%s:%d/%s", conn.Username, conn.Host, conn.Port, serviceName))

	oraConnection := &oracle.Connection{
		ConnectionString:   connectionString,
		DBName:             conn.DatabaseName,
		Logger:             logger,
		MaxOpenConnections: constant.OracleMaxOpenConns,
		MaxIdleConnections: constant.OracleMaxIdleConns,
	}

	if errConnect := oraConnection.Connect(); errConnect != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to connect to Oracle [%s] with service name [%s]: %v", conn.ConfigName, conn.DatabaseName, errConnect))
		return nil, fmt.Errorf("failed to connect to Oracle with service name '%s': %w. Verify that the service name is correct and registered with the Oracle listener", conn.DatabaseName, errConnect)
	}

	// Create repository
	repo, errRepo := oracle.NewDataSourceRepository(oraConnection)
	if errRepo != nil {
		_ = oraConnection.ConnectionDB.Close()
		return nil, fmt.Errorf("failed to create Oracle repository: %w", errRepo)
	}

	return &datsourceOracleConfig.DataSourceConfigOracle{
		DataSourceConfig: base,
		OracleConnection: oraConnection,
		OracleRepository: repo,
	}, nil
}

// newDataSourceConfigMySQL creates a MySQL-specific DataSourceConfig.
func newDataSourceConfigMySQL(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (*datsourceMySQLConfig.DataSourceConfigMySQL, error) {
	// Decrypt password before using it in connection string
	password, err := conn.GetPasswordDecrypted(ctx, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password for MySQL connection: %w", err)
	}

	// MySQL connection string format: user:password@tcp(host:port)/database?params
	sslMode := "false"
	if conn.SSL != nil && conn.SSL.Mode != "" {
		sslMode = conn.SSL.Mode
	}

	// Validate SSL mode to prevent injection attacks
	if err := sslmode.ValidateMySQLMode(sslMode); err != nil {
		return nil, err
	}

	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%s",
		conn.Username,
		url.QueryEscape(password),
		conn.Host,
		conn.Port,
		conn.DatabaseName,
		sslMode,
	)

	mysqlConnection := &mysql.Connection{
		ConnectionString:   connectionString,
		DBName:             conn.DatabaseName,
		Logger:             logger,
		MaxOpenConnections: constant.MySQLMaxOpenConns,
		MaxIdleConnections: constant.MySQLMaxIdleConns,
	}

	if errConnect := mysqlConnection.Connect(); errConnect != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to connect to MySQL [%s]: %v", conn.ConfigName, errConnect))
		return nil, fmt.Errorf("failed to connect to MySQL: %w", errConnect)
	}

	// Create repository
	repo, errRepo := mysql.NewDataSourceRepository(mysqlConnection)
	if errRepo != nil {
		_ = mysqlConnection.ConnectionDB.Close()
		return nil, fmt.Errorf("failed to create MySQL repository: %w", errRepo)
	}

	return &datsourceMySQLConfig.DataSourceConfigMySQL{
		DataSourceConfig: base,
		MySQLConnection:  mysqlConnection,
		MySQLRepository:  repo,
	}, nil
}

// newDataSourceConfigSQLServer creates a SQL Server-specific DataSourceConfig.
func newDataSourceConfigSQLServer(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (*datsourceSQLServerConfig.DataSourceConfigSQLServer, error) {
	// Decrypt password before using it in connection string
	password, err := conn.GetPasswordDecrypted(ctx, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password for SQL Server connection: %w", err)
	}

	// SQL Server connection string format: sqlserver://user:password@host:port?database=database&encrypt=mode
	// SSL modes:
	// - "disable": No encryption
	// - "true"/"require": Encryption with TrustServerCertificate=true (for self-signed certs)
	// - "strict": Strict encryption with certificate validation
	sslMode := "disable"
	trustServerCert := false

	if conn.SSL != nil && conn.SSL.Mode != "" {
		// Validate SSL mode to prevent injection attacks
		if err := sslmode.ValidateSQLServerMode(conn.SSL.Mode); err != nil {
			return nil, err
		}

		switch conn.SSL.Mode {
		case "true", "require":
			sslMode = "true"
			trustServerCert = true // Required for self-signed certificates in test environments
		case "strict":
			sslMode = "strict"
			trustServerCert = false // Strict mode validates certificates
		default:
			// For backwards compatibility, treat unknown modes as encrypted with trust
			if conn.SSL.Mode != "disable" {
				sslMode = "true"
				trustServerCert = true
			}
		}
	}

	connectionString := fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s&encrypt=%s",
		conn.Username,
		url.QueryEscape(password),
		conn.Host,
		conn.Port,
		conn.DatabaseName,
		sslMode,
	)

	if trustServerCert {
		connectionString += "&TrustServerCertificate=true"
	}

	sqlServerConnection := &sqlserver.Connection{
		ConnectionString:   connectionString,
		DBName:             conn.DatabaseName,
		Logger:             logger,
		MaxOpenConnections: constant.SQLServerMaxOpenConns,
		MaxIdleConnections: constant.SQLServerMaxIdleConns,
	}

	if errConnect := sqlServerConnection.Connect(); errConnect != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to connect to SQL Server [%s]: %v", conn.ConfigName, errConnect))
		return nil, fmt.Errorf("failed to connect to SQL Server: %w", errConnect)
	}

	// Create repository
	repo, errRepo := sqlserver.NewDataSourceRepository(sqlServerConnection)
	if errRepo != nil {
		_ = sqlServerConnection.ConnectionDB.Close()
		return nil, fmt.Errorf("failed to create SQL Server repository: %w", errRepo)
	}

	return &datsourceSQLServerConfig.DataSourceConfigSQLServer{
		DataSourceConfig:    base,
		SQLServerConnection: sqlServerConnection,
		SQLServerRepository: repo,
	}, nil
}

// DataSourceFactory is a function type for creating DataSource instances.
// This allows for dependency injection and easier testing.
type DataSourceFactory func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error)

// NewDataSourceFromConnectionWithLogger returns a factory function that creates DataSource
// implementations with a pre-configured logger. This is useful for dependency injection
// where the logger needs to be captured at initialization time.
func NewDataSourceFromConnectionWithLogger(logger libLog.Logger) func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
	return func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return NewDataSourceFromConnection(ctx, conn, cryptor, logger)
	}
}
