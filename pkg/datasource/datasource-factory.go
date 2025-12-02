package datasource

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	datsourceMongoConfig "github.com/LerianStudio/fetcher/pkg/model/datasource/mongodb"
	datsourcePostgresConfig "github.com/LerianStudio/fetcher/pkg/model/datasource/postgres"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/fetcher/pkg/postgres"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewDataSourceFromConnection creates a DataSource implementation based on the connection type.
// This factory function encapsulates the logic for converting a Connection entity
// into the appropriate DataSourceConfig implementation (MongoDB or PostgreSQL).
func NewDataSourceFromConnection(ctx context.Context, conn *model.Connection, logger log.Logger) (datasource.DataSource, error) {
	if conn == nil {
		return nil, fmt.Errorf("connection cannot be nil")
	}

	baseConfig := newDataSourceConfigFromConnection(conn)
	switch conn.Type {
	case model.TypeMongoDB:
		return newDataSourceConfigMongoDB(ctx, baseConfig, conn, logger)
	case model.TypePostgreSQL:
		return newDataSourceConfigPostgres(baseConfig, conn, logger)
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
		Type:              strings.ToLower(string(conn.Type)),
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
func newDataSourceConfigMongoDB(ctx context.Context, base datasource.DataSourceConfig, conn *model.Connection, logger log.Logger) (*datsourceMongoConfig.DataSourceConfigMongoDB, error) {
	optionsStr := "authSource=admin&directConnection=true"
	mongoURI := fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		strings.ToLower(string(conn.Type)),
		conn.Username,
		conn.PasswordEncrypted,
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

	client, err := mongo.Connect(testCtx, clientOpts)
	if err != nil {
		logger.Errorf("Failed to connect to MongoDB [%s]: %v", conn.ConfigName, err)
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(testCtx, nil); err != nil {
		_ = client.Disconnect(testCtx)

		logger.Errorf("Failed to ping MongoDB [%s]: %v", conn.ConfigName, err)

		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	_ = client.Disconnect(testCtx)

	// Create repository
	repo, err := mongodb.NewDataSourceRepository(mongoURI, conn.DatabaseName, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB repository: %w", err)
	}

	return &datsourceMongoConfig.DataSourceConfigMongoDB{
		DataSourceConfig:  base,
		MongoDBRepository: repo,
		MongoURI:          mongoURI,
		Options:           optionsStr,
	}, nil
}

// newDataSourceConfigPostgres creates a PostgreSQL-specific DataSourceConfig.
func newDataSourceConfigPostgres(base datasource.DataSourceConfig, conn *model.Connection, logger log.Logger) (*datsourcePostgresConfig.DataSourceConfigPostgres, error) {
	sslMode := "disable"
	if conn.SSL != nil && conn.SSL.Mode != "" {
		sslMode = conn.SSL.Mode
	}

	connectionString := fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=%s",
		strings.ToLower(string(conn.Type)),
		conn.Username,
		url.QueryEscape(conn.PasswordEncrypted),
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

	if err := pgConnection.Connect(); err != nil {
		logger.Errorf("Failed to connect to Postgres [%s]: %v", conn.ConfigName, err)
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Create repository
	repo, err := postgres.NewDataSourceRepository(pgConnection)
	if err != nil {
		_ = pgConnection.ConnectionDB.Close()
		return nil, fmt.Errorf("failed to create PostgreSQL repository: %w", err)
	}

	return &datsourcePostgresConfig.DataSourceConfigPostgres{
		DataSourceConfig:   base,
		PostgresConnection: pgConnection,
		PostgresRepository: repo,
	}, nil
}
