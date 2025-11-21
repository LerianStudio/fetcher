package pkg

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/fetcher/pkg/postgres"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DataSourceConfig struct {
	ID                 string
	OrganizationID     string
	ConfigName         string
	Type               string
	Host               string
	Port               string
	DatabaseName       string
	Username           string
	PasswordEncrypted  string
	SSL                any
	PostgresConnection *postgres.Connection
	PostgresRepository *postgres.ExternalDataSource
	MongoDBRepository  *mongodb.ExternalDataSource
	MongoURI           string
	Options            string
	Status             string
}

// ConnectToDataSource establishes a connection to a data source if not already initialized.
func ConnectToDataSource(dataSource *DataSourceConfig, logger log.Logger) error {

	var err error

	switch dataSource.Type {
	case constant.PostgreSQLType:
		initPostgresDataSource(dataSource, logger)
		dataSource.PostgresRepository, err = postgres.NewDataSourceRepository(dataSource.PostgresConnection)
		if err != nil {
			dataSource.Status = libConstant.DataSourceStatusUnavailable
			logger.Errorf("Failed to establish PostgreSQL connection to %s: %v", dataSource.ConfigName, err)

			return fmt.Errorf("failed to establish PostgreSQL connection to %s: %w", dataSource.ConfigName, err)
		}

		logger.Infof("Established PostgreSQL connection to %s database", dataSource.ConfigName)

		dataSource.Status = libConstant.DataSourceStatusAvailable

	case constant.MongoDBType:
		initMongoDataSource(dataSource, logger)
		dataSource.MongoDBRepository, err = mongodb.NewDataSourceRepository(dataSource.MongoURI, dataSource.DatabaseName, logger)
		if err != nil {
			dataSource.Status = libConstant.DataSourceStatusUnavailable
			logger.Errorf("Failed to establish MongoDB connection to %s: %v", dataSource.DatabaseName, err)

			return fmt.Errorf("failed to establish MongoDB connection to %s: %w", dataSource.DatabaseName, err)
		}

		logger.Infof("Established MongoDB connection to %s database", dataSource.DatabaseName)

		dataSource.Status = libConstant.DataSourceStatusAvailable

	default:
		dataSource.Status = libConstant.DataSourceStatusUnavailable
		return fmt.Errorf("unsupported database type: %s for database: %s", dataSource.Type, dataSource.ConfigName)
	}

	return nil
}

// initMongoDataSource initializes a MongoDB connection.
func initMongoDataSource(dataSource *DataSourceConfig, logger log.Logger) {
	// TODO: Passar isso pra dentro a entity de connection
	dataSource.Options = "authSource=admin&directConnection=true"
	mongoURI := fmt.Sprintf("%s://%s:%s@%s:%s/%s",
		dataSource.Type, dataSource.Username, dataSource.PasswordEncrypted, dataSource.Host, dataSource.Port, dataSource.DatabaseName)
	if dataSource.Options != "" {
		mongoURI += "?" + dataSource.Options
	}

	var params []string
	if dataSource.SSL == "true" {
		params = append(params, "ssl=true")
	}

	//if dataSource.SSLCA != "" {
	//	params = append(params, "tlsCAFile="+url.QueryEscape(dataSource.SSLCA))
	//}

	if len(params) > 0 {
		if strings.Contains(mongoURI, "?") {
			mongoURI += "&" + strings.Join(params, "&")
		} else {
			mongoURI += "?" + strings.Join(params, "&")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), constant.ConnectionTimeout)
	defer cancel()

	// Configure MongoDB client with pool settings and shorter timeouts
	clientOpts := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(constant.MongoDBMaxPoolSize).
		SetMinPoolSize(constant.MongoDBMinPoolSize).
		SetMaxConnIdleTime(constant.MongoDBMaxConnIdleTime).
		SetConnectTimeout(constant.ConnectionTimeout).
		SetServerSelectionTimeout(constant.ConnectionTimeout)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		logger.Errorf("Failed to connect to MongoDB [%s]: %v", dataSource.ConfigName, err)
	} else if err := client.Ping(ctx, nil); err != nil {
		logger.Errorf("Failed to ping MongoDB [%s]: %v", dataSource.ConfigName, err)
	} else {
		logger.Infof("Successfully connected to MongoDB [%s] with pool config (max: %d, min: %d)",
			dataSource.ConfigName, constant.MongoDBMaxPoolSize, constant.MongoDBMinPoolSize)
	}

	// Only disconnect if client was successfully created
	if client != nil {
		_ = client.Disconnect(ctx)
	}

	dataSource.MongoURI = mongoURI
}

func initPostgresDataSource(dataSource *DataSourceConfig, logger log.Logger) {
	connectionString := fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=%s",
		dataSource.Type, dataSource.Username, url.QueryEscape(dataSource.PasswordEncrypted), dataSource.Host, dataSource.Port, dataSource.DatabaseName, dataSource.SSL)
	// TODO: Add SSL support
	//if dataSource.SSL != "" {
	//	connectionString += fmt.Sprintf("&sslrootcert=%s", url.QueryEscape(dataSource.SSLRootCert))
	//}

	connection := &postgres.Connection{
		ConnectionString:   connectionString,
		DBName:             dataSource.DatabaseName,
		Logger:             logger,
		MaxOpenConnections: constant.PostgresMaxOpenConns,
		MaxIdleConnections: constant.PostgresMaxIdleConns,
	}
	if err := connection.Connect(); err != nil {
		logger.Errorf("Failed to connect to Postgres [%s]: %v", dataSource.ConfigName, err)
	} else {
		logger.Infof("Successfully connected to Postgres [%s] with pool config (max: %d, idle: %d)",
			dataSource.ConfigName, constant.PostgresMaxOpenConns, constant.PostgresMaxIdleConns)
	}

	dataSource.PostgresConnection = connection
}
