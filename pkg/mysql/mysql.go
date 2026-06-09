package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	libLog "github.com/LerianStudio/lib-observability/log"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

// Connection is a hub which deals with MySQL connections.
type Connection struct {
	ConnectionString   string
	DBName             string
	ConnectionDB       *sql.DB
	Connected          bool
	Logger             libLog.Logger
	MaxOpenConnections int
	MaxIdleConnections int
}

// Connect initializes the connection with the MySQL DB.
func (c *Connection) Connect() error {
	c.Logger.Log(context.Background(), libLog.LevelInfo, "Connecting to MySQL...")

	db, err := sql.Open("mysql", c.ConnectionString)
	if err != nil {
		c.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error opening connection: %v", err))
		return err
	}

	if err := db.Ping(); err != nil {
		closeErr := db.Close()
		if closeErr != nil {
			c.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error closing connection: %v", closeErr))
		}

		c.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error pinging MySQL: %v", err))

		return err
	}

	db.SetMaxOpenConns(c.MaxOpenConnections)
	db.SetMaxIdleConns(c.MaxIdleConnections)
	db.SetConnMaxLifetime(constant.MySQLConnMaxLifetime)
	db.SetConnMaxIdleTime(constant.MySQLConnMaxIdleTime)

	c.ConnectionDB = db
	c.Connected = true

	c.Logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Connected to MySQL [%s] with pool settings (maxOpen: %d, maxIdle: %d, maxLifetime: %v, maxIdleTime: %v)",
		c.DBName, c.MaxOpenConnections, c.MaxIdleConnections, constant.MySQLConnMaxLifetime, constant.MySQLConnMaxIdleTime))

	return nil
}

// GetDB returns a pointer to the MySQL connection, initializing it if necessary.
func (mc *Connection) GetDB() (*sql.DB, error) {
	if mc.ConnectionDB == nil {
		if err := mc.Connect(); err != nil {
			mc.Logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("ERRCONECT %s", err))
			return nil, err
		}
	}

	return mc.ConnectionDB, nil
}

// ValidateFieldsInSchemaMySQL validate if all fields exist on mysql schema table
func ValidateFieldsInSchemaMySQL(expectedFields []string, schema TableSchema, countIfTableExist *int32) (missing []string) {
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
