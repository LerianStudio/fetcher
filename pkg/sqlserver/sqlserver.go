package sqlserver

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/constant"

	"github.com/LerianStudio/lib-commons/v2/commons/log"
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
)

// Connection is a hub which deals with SQL Server connections.
type Connection struct {
	ConnectionString   string
	DBName             string
	ConnectionDB       *sql.DB
	Connected          bool
	Logger             log.Logger
	MaxOpenConnections int
	MaxIdleConnections int
}

// Connect initializes the connection with the SQL Server DB.
func (c *Connection) Connect() error {
	c.Logger.Info("Connecting to SQL Server...")

	db, err := sql.Open("sqlserver", c.ConnectionString)
	if err != nil {
		c.Logger.Errorf("Error opening connection: %v", err)
		return fmt.Errorf("failed to open SQL Server connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		closeErr := db.Close()
		if closeErr != nil {
			c.Logger.Errorf("Error closing connection: %v", closeErr)
		}

		c.Logger.Errorf("Error pinging SQL Server: %v", err)

		return fmt.Errorf("failed to ping SQL Server: %w", err)
	}

	db.SetMaxOpenConns(c.MaxOpenConnections)
	db.SetMaxIdleConns(c.MaxIdleConnections)
	db.SetConnMaxLifetime(constant.SQLServerConnMaxLifetime)
	db.SetConnMaxIdleTime(constant.SQLServerConnMaxIdleTime)

	c.ConnectionDB = db
	c.Connected = true

	c.Logger.Infof("Connected to SQL Server [%s] with pool settings (maxOpen: %d, maxIdle: %d, maxLifetime: %v, maxIdleTime: %v)",
		c.DBName, c.MaxOpenConnections, c.MaxIdleConnections, constant.SQLServerConnMaxLifetime, constant.SQLServerConnMaxIdleTime)

	return nil
}

// GetDB returns a pointer to the SQL Server connection, initializing it if necessary.
func (sc *Connection) GetDB() (*sql.DB, error) {
	if sc.ConnectionDB == nil {
		if err := sc.Connect(); err != nil {
			sc.Logger.Errorf("ERR_CONNECT: failed to connect to SQL Server: %v", err)
			return nil, err
		}
	}

	return sc.ConnectionDB, nil
}

// ValidateFieldsInSchemaSQLServer validate if all fields exist on sqlserver schema table
func ValidateFieldsInSchemaSQLServer(expectedFields []string, schema TableSchema, countIfTableExist *int32) (missing []string) {
	columnSet := make(map[string]struct{}, len(schema.Columns))
	for _, col := range schema.Columns {
		columnSet[strings.ToLower(col.Name)] = struct{}{}
	}

	for _, field := range expectedFields {
		*countIfTableExist++

		if _, exists := columnSet[strings.ToLower(field)]; !exists {
			missing = append(missing, field)
		}
	}

	return
}
