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
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewDataSourceFromConnection creates a DataSource implementation based on the connection type.
// This factory function encapsulates the logic for converting a Connection entity
// into the appropriate DataSourceConfig implementation (MongoDB, PostgreSQL, Oracle, MySQL, or SQL Server).
// The cryptor is required to decrypt the connection password before creating the data source.
func NewDataSourceFromConnection(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor, logger log.Logger) (datasource.DataSource, error) {
	if conn == nil {
		return nil, fmt.Errorf("connection cannot be nil")
	}

	if cryptor == nil {
		return nil, fmt.Errorf("cryptor cannot be nil")
	}

	baseConfig := newDataSourceConfigFromConnection(conn)
	switch conn.Type {
	case model.TypeMongoDB:
		return newDataSourceConfigMongoDB(ctx, baseConfig, conn, cryptor, logger)
	case model.TypePostgreSQL:
		return newDataSourceConfigPostgres(ctx, baseConfig, conn, cryptor, logger)
	case model.TypeOracle:
		return newDataSourceConfigOracle(ctx, baseConfig, conn, cryptor, logger)
	case model.TypeMySQL:
		return newDataSourceConfigMySQL(ctx, baseConfig, conn, cryptor, logger)
	case model.TypeSQLServer:
		return newDataSourceConfigSQLServer(ctx, baseConfig, conn, cryptor, logger)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", conn.Type)
	}
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
func newDataSourceConfigMongoDB(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger log.Logger) (*datsourceMongoConfig.DataSourceConfigMongoDB, error) {
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

	var params []string
	if conn.SSL != nil && conn.SSL.Mode == "true" {
		params = append(params, "ssl=true")
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
		logger.Errorf("Failed to connect to MongoDB [%s]: %v", conn.ConfigName, errConnect)
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", errConnect)
	}

	if errPing := client.Ping(testCtx, nil); errPing != nil {
		_ = client.Disconnect(testCtx)

		logger.Errorf("Failed to ping MongoDB [%s]: %v", conn.ConfigName, errPing)

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
func newDataSourceConfigPostgres(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger log.Logger) (*datsourcePostgresConfig.DataSourceConfigPostgres, error) {
	// Decrypt password before using it in connection string
	password, err := conn.GetPasswordDecrypted(ctx, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password for PostgreSQL connection: %w", err)
	}

	sslMode := "disable"
	if conn.SSL != nil && conn.SSL.Mode != "" {
		sslMode = conn.SSL.Mode
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
		logger.Errorf("Failed to connect to Postgres [%s]: %v", conn.ConfigName, errConnect)
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
func newDataSourceConfigOracle(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger log.Logger) (*datsourceOracleConfig.DataSourceConfigOracle, error) {
	// Decrypt password before using it in connection string
	password, err := conn.GetPasswordDecrypted(ctx, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password for Oracle connection: %w", err)
	}

	// TODO: Implementar um metadata talvez para o tipo oracle pra passar o service
	serviceName := "XEPDB1"
	connectionString := fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		conn.Username,
		url.QueryEscape(password),
		conn.Host,
		conn.Port,
		serviceName,
	)

	logger.Infof("Attempting Oracle connection [%s] - Service: %s, Username: %s (will access schema: %s)",
		conn.ConfigName, serviceName, conn.Username, strings.ToUpper(conn.Username))
	logger.Debugf("Oracle connection string: oracle://%s:***@%s:%d/%s", conn.Username, conn.Host, conn.Port, serviceName)

	oraConnection := &oracle.Connection{
		ConnectionString:   connectionString,
		DBName:             conn.DatabaseName,
		Logger:             logger,
		MaxOpenConnections: constant.OracleMaxOpenConns,
		MaxIdleConnections: constant.OracleMaxIdleConns,
	}

	if errConnect := oraConnection.Connect(); errConnect != nil {
		logger.Errorf("Failed to connect to Oracle [%s] with service name [%s]: %v", conn.ConfigName, conn.DatabaseName, errConnect)
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
func newDataSourceConfigMySQL(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger log.Logger) (*datsourceMySQLConfig.DataSourceConfigMySQL, error) {
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
		logger.Errorf("Failed to connect to MySQL [%s]: %v", conn.ConfigName, errConnect)
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
func newDataSourceConfigSQLServer(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, cryptor crypto.Cryptor, logger log.Logger) (*datsourceSQLServerConfig.DataSourceConfigSQLServer, error) {
	// Decrypt password before using it in connection string
	password, err := conn.GetPasswordDecrypted(ctx, cryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password for SQL Server connection: %w", err)
	}

	// SQL Server connection string format: sqlserver://user:password@host:port?database=database&encrypt=disable
	sslMode := "disable"

	if conn.SSL != nil && conn.SSL.Mode != "" {
		if conn.SSL.Mode == "true" || conn.SSL.Mode == "require" {
			sslMode = "true"
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

	sqlServerConnection := &sqlserver.Connection{
		ConnectionString:   connectionString,
		DBName:             conn.DatabaseName,
		Logger:             logger,
		MaxOpenConnections: constant.SQLServerMaxOpenConns,
		MaxIdleConnections: constant.SQLServerMaxIdleConns,
	}

	if errConnect := sqlServerConnection.Connect(); errConnect != nil {
		logger.Errorf("Failed to connect to SQL Server [%s]: %v", conn.ConfigName, errConnect)
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
