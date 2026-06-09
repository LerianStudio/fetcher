// Package datasource provides factory functions for creating database data source implementations
// from connection models, handling password decryption and connection string construction.
package datasource

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/datasource/hostsafety"
	"github.com/LerianStudio/fetcher/v2/pkg/datasource/sslmode"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/model/datasource"
	datasourceMongoConfig "github.com/LerianStudio/fetcher/v2/pkg/model/datasource/mongodb"
	datasourceMySQLConfig "github.com/LerianStudio/fetcher/v2/pkg/model/datasource/mysql"
	datasourceOracleConfig "github.com/LerianStudio/fetcher/v2/pkg/model/datasource/oracle"
	datasourcePostgresConfig "github.com/LerianStudio/fetcher/v2/pkg/model/datasource/postgres"
	datasourceSQLServerConfig "github.com/LerianStudio/fetcher/v2/pkg/model/datasource/sqlserver"
	"github.com/LerianStudio/fetcher/v2/pkg/mongodb"
	"github.com/LerianStudio/fetcher/v2/pkg/mysql"
	"github.com/LerianStudio/fetcher/v2/pkg/oracle"
	"github.com/LerianStudio/fetcher/v2/pkg/postgres"
	"github.com/LerianStudio/fetcher/v2/pkg/sqlserver"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

// NewDataSourceFromConnection creates a DataSource implementation based on the connection type.
// This factory function encapsulates the logic for converting a Connection entity
// into the appropriate DataSourceConfig implementation (MongoDB, PostgreSQL, Oracle, MySQL, or SQL Server).
// The cryptor is required to decrypt the connection password before creating the data source.
func NewDataSourceFromConnection(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (datasource.DataSource, error) {
	ctxLogger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)
	if logger == nil {
		logger = ctxLogger
	}

	if logger == nil {
		logger = libLog.NewNop()
	}

	ctx, span := tracer.Start(ctx, "factory.datasource.create")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	if conn == nil {
		err := fmt.Errorf("connection cannot be nil")
		libOpentelemetry.HandleSpanError(span, "nil connection", err)

		return nil, err
	}

	span.SetAttributes(attribute.String("app.datasource.connection_type", string(conn.Type)))

	// Allow nil cryptor for in-memory connections (from tenant-manager) that have
	// plaintext passwords. Connections with EncryptionKeyVersion set still require cryptor.
	// NOTE: EncryptionKeyVersion="" is the signal for "plaintext/internal". This is an
	// implicit contract — external connections persisted in MongoDB always have a non-empty
	// EncryptionKeyVersion set during creation. If this contract changes, add an explicit
	// IsEncrypted field to the Connection model.
	if cryptor == nil && conn.EncryptionKeyVersion != "" {
		err := fmt.Errorf("cryptor cannot be nil for encrypted connections")
		libOpentelemetry.HandleSpanError(span, "nil cryptor", err)

		return nil, err
	}

	if err := hostsafety.ValidateHostForConnection(ctx, conn); err != nil {
		libOpentelemetry.HandleSpanError(span, "host blocked by safety policy", err)
		return nil, err
	}

	baseConfig := newDataSourceConfigFromConnection(conn)

	var ds datasource.DataSource

	var err error

	switch conn.Type {
	case model.TypeMongoDB:
		ds, err = newDataSourceConfigMongoDB(ctx, baseConfig, conn, cryptor, logger)
	case model.TypePostgreSQL:
		ds, err = newDataSourceConfigPostgres(ctx, baseConfig, conn, cryptor, logger)
	case model.TypeOracle:
		ds, err = newDataSourceConfigOracle(ctx, baseConfig, conn, cryptor, logger)
	case model.TypeMySQL:
		ds, err = newDataSourceConfigMySQL(ctx, baseConfig, conn, cryptor, logger)
	case model.TypeSQLServer:
		ds, err = newDataSourceConfigSQLServer(ctx, baseConfig, conn, cryptor, logger)
	default:
		err = fmt.Errorf("unsupported database type: %s", conn.Type)
	}

	if err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to create datasource", err)
		return nil, err
	}

	return ds, nil
}

// resolvePassword returns the connection password, either from the plaintext field
// (for in-memory connections from tenant-manager) or by decrypting the encrypted password.
// Connections with empty EncryptionKeyVersion are assumed to have plaintext passwords.
func resolvePassword(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (string, error) {
	if conn.EncryptionKeyVersion == "" {
		// In-memory connection with plaintext password (from tenant-manager or env vars)
		return conn.GetPlaintextPassword(), nil
	}

	// Encrypted connection — decrypt using cryptor
	return conn.GetPasswordDecrypted(ctx, cryptor)
}

// newDataSourceConfigFromConnection creates a base DataSourceConfig from a Connection entity.
func newDataSourceConfigFromConnection(conn *model.Connection) datasource.DataSourceConfig {
	var sslConfig model.SSLConfig
	if conn.SSL != nil {
		sslConfig = *conn.SSL
	}

	return datasource.DataSourceConfig{
		ID:                conn.ID.String(),
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
func newDataSourceConfigMongoDB(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (*datasourceMongoConfig.DataSourceConfigMongoDB, error) {
	// Resolve password (plaintext for internal datasources, decrypt for encrypted)
	password, err := resolvePassword(ctx, conn, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve password for MongoDB connection: %w", err)
	}

	optionParts := buildMongoDBOptions(conn)

	mongoURI := fmt.Sprintf("%s://%s:%s@%s:%d/%s?%s",
		strings.ToLower(string(conn.Type)),
		conn.Username,
		url.QueryEscape(password),
		conn.Host,
		conn.Port,
		conn.DatabaseName,
		strings.Join(optionParts, "&"),
	)

	// Validate SSL mode to prevent injection attacks
	if conn.SSL != nil && conn.SSL.Mode != "" {
		if err := sslmode.ValidateMongoDBMode(conn.SSL.Mode); err != nil {
			return nil, err
		}
	}

	mongoURI = appendMongoDBSSLParams(mongoURI, conn)

	// Test connection with timeout
	if err := testMongoDBConnection(ctx, mongoURI, conn.ConfigName, logger); err != nil {
		return nil, err
	}

	// Create repository
	repo, errRepo := mongodb.NewDataSourceRepository(mongoURI, conn.DatabaseName, logger)
	if errRepo != nil {
		return nil, fmt.Errorf("failed to create MongoDB repository: %w", errRepo)
	}

	return &datasourceMongoConfig.DataSourceConfigMongoDB{
		DataSourceConfig:  base,
		MongoDBRepository: repo,
		MongoURI:          mongoURI,
		Options:           strings.Join(optionParts, "&"),
	}, nil
}

// buildMongoDBOptions builds the connection options from metadata.
// For internal connections (EncryptionKeyVersion empty), authSource defaults to
// the database name itself (per tenant-manager provisioning convention).
// For external connections, authSource defaults to "admin".
func buildMongoDBOptions(conn *model.Connection) []string {
	defaultAuthSource := "admin"
	if conn.EncryptionKeyVersion == "" && conn.DatabaseName != "" {
		defaultAuthSource = conn.DatabaseName
	}

	optionParts := []string{"authSource=" + defaultAuthSource}

	if conn.Metadata != nil {
		if dc, ok := (*conn.Metadata)["directConnection"].(string); ok && dc == "true" {
			optionParts = append(optionParts, "directConnection=true")
		}

		if authSrc, ok := (*conn.Metadata)["authSource"].(string); ok && authSrc != "" {
			optionParts[0] = "authSource=" + authSrc
		}
	}

	return optionParts
}

// appendMongoDBSSLParams appends TLS/SSL query parameters to the MongoDB URI
// based on the connection's SSL configuration.
func appendMongoDBSSLParams(mongoURI string, conn *model.Connection) string {
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

	return mongoURI
}

// testMongoDBConnection verifies connectivity to a MongoDB instance by performing
// a connect and ping operation, then immediately disconnects.
func testMongoDBConnection(ctx context.Context, mongoURI, configName string, logger libLog.Logger) error {
	testCtx, cancel := context.WithTimeout(ctx, constant.ConnectionTimeout)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(constant.MongoDBMaxPoolSize).
		SetMinPoolSize(constant.MongoDBMinPoolSize).
		SetMaxConnIdleTime(constant.MongoDBMaxConnIdleTime).
		SetConnectTimeout(constant.ConnectionTimeout).
		SetServerSelectionTimeout(constant.ConnectionTimeout)

	client, errConnect := mongo.Connect(clientOpts)
	if errConnect != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to connect to MongoDB [%s]: %v", configName, errConnect))
		return fmt.Errorf("failed to connect to MongoDB: %w", errConnect)
	}

	if errPing := client.Ping(testCtx, nil); errPing != nil {
		_ = client.Disconnect(testCtx)

		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to ping MongoDB [%s]: %v", configName, errPing))

		return fmt.Errorf("failed to ping MongoDB: %w", errPing)
	}

	_ = client.Disconnect(testCtx)

	return nil
}

// newDataSourceConfigPostgres creates a PostgreSQL-specific DataSourceConfig.
func newDataSourceConfigPostgres(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (*datasourcePostgresConfig.DataSourceConfigPostgres, error) {
	// Resolve password (plaintext for internal datasources, decrypt for encrypted)
	password, err := resolvePassword(ctx, conn, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve password for PostgreSQL connection: %w", err)
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

	return &datasourcePostgresConfig.DataSourceConfigPostgres{
		DataSourceConfig:   base,
		PostgresConnection: pgConnection,
		PostgresRepository: repo,
	}, nil
}

// newDataSourceConfigOracle creates an Oracle-specific DataSourceConfig.
func newDataSourceConfigOracle(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (*datasourceOracleConfig.DataSourceConfigOracle, error) {
	// Resolve password (plaintext for internal datasources, decrypt for encrypted)
	password, err := resolvePassword(ctx, conn, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve password for Oracle connection: %w", err)
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

	logger.Log(ctx, libLog.LevelDebug, fmt.Sprintf("Attempting Oracle connection [%s] - Service: %s, Username: %s (will access schema: %s)", conn.ConfigName, serviceName, conn.Username, strings.ToUpper(conn.Username)))
	logger.Log(ctx, libLog.LevelDebug, fmt.Sprintf("Oracle connection string: oracle://%s:***@%s:%d/%s", conn.Username, conn.Host, conn.Port, serviceName))

	oraConnection := &oracle.Connection{
		ConnectionString:   connectionString,
		DBName:             conn.DatabaseName,
		Logger:             logger,
		MaxOpenConnections: constant.OracleMaxOpenConns,
		MaxIdleConnections: constant.OracleMaxIdleConns,
	}

	if errConnect := oraConnection.Connect(); errConnect != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to connect to Oracle [%s] with service name [%s]: %v", conn.ConfigName, serviceName, errConnect))
		return nil, fmt.Errorf("failed to connect to Oracle with service name '%s': %w. Verify that the service name is correct and registered with the Oracle listener", serviceName, errConnect)
	}

	// Create repository
	repo, errRepo := oracle.NewDataSourceRepository(oraConnection)
	if errRepo != nil {
		_ = oraConnection.ConnectionDB.Close()
		return nil, fmt.Errorf("failed to create Oracle repository: %w", errRepo)
	}

	return &datasourceOracleConfig.DataSourceConfigOracle{
		DataSourceConfig: base,
		OracleConnection: oraConnection,
		OracleRepository: repo,
	}, nil
}

// newDataSourceConfigMySQL creates a MySQL-specific DataSourceConfig.
func newDataSourceConfigMySQL(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (*datasourceMySQLConfig.DataSourceConfigMySQL, error) {
	// Resolve password (plaintext for internal datasources, decrypt for encrypted)
	password, err := resolvePassword(ctx, conn, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve password for MySQL connection: %w", err)
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

	return &datasourceMySQLConfig.DataSourceConfigMySQL{
		DataSourceConfig: base,
		MySQLConnection:  mysqlConnection,
		MySQLRepository:  repo,
	}, nil
}

// newDataSourceConfigSQLServer creates a SQL Server-specific DataSourceConfig.
func newDataSourceConfigSQLServer(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger libLog.Logger) (*datasourceSQLServerConfig.DataSourceConfigSQLServer, error) {
	// Resolve password (plaintext for internal datasources, decrypt for encrypted)
	password, err := resolvePassword(ctx, conn, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve password for SQL Server connection: %w", err)
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

	if errConnect := sqlServerConnection.Connect(ctx); errConnect != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to connect to SQL Server [%s]: %v", conn.ConfigName, errConnect))
		return nil, fmt.Errorf("failed to connect to SQL Server: %w", errConnect)
	}

	// Create repository
	repo, errRepo := sqlserver.NewDataSourceRepository(sqlServerConnection)
	if errRepo != nil {
		_ = sqlServerConnection.ConnectionDB.Close()
		return nil, fmt.Errorf("failed to create SQL Server repository: %w", errRepo)
	}

	return &datasourceSQLServerConfig.DataSourceConfigSQLServer{
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
