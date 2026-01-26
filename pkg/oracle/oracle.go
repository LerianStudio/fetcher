package oracle

import (
	"database/sql"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/constant"

	"github.com/LerianStudio/lib-commons/v2/commons/log"
	_ "github.com/sijms/go-ora/v2" // Oracle driver
)

// DefaultSchema represents Oracle's default schema behavior.
// Unlike PostgreSQL ("public") or SQL Server ("dbo"), Oracle uses the
// current user as the default schema. This constant is empty because
// Oracle's default schema is dynamic and determined at runtime.
// Table name normalization for Oracle is handled differently - see
// normalizeTableNameForValidation in validate_schema.go.
const DefaultSchema = ""

// Connection is a hub which deals with Oracle connections.
type Connection struct {
	ConnectionString   string
	DBName             string
	ConnectionDB       *sql.DB
	Connected          bool
	Logger             log.Logger
	MaxOpenConnections int
	MaxIdleConnections int
}

// Connect initializes the connection with the Oracle DB.
func (c *Connection) Connect() error {
	c.Logger.Info("Connecting to Oracle...")

	db, err := sql.Open("oracle", c.ConnectionString)
	if err != nil {
		c.Logger.Errorf("Error opening connection: %v", err)
		return err
	}

	if err := db.Ping(); err != nil {
		closeErr := db.Close()
		if closeErr != nil {
			c.Logger.Errorf("Error closing connection: %v", closeErr)
		}

		c.Logger.Errorf("Error pinging Oracle: %v", err)

		return err
	}

	db.SetMaxOpenConns(c.MaxOpenConnections)
	db.SetMaxIdleConns(c.MaxIdleConnections)
	db.SetConnMaxLifetime(constant.OracleConnMaxLifetime)
	db.SetConnMaxIdleTime(constant.OracleConnMaxIdleTime)

	c.ConnectionDB = db
	c.Connected = true

	c.Logger.Infof("Connected to Oracle [%s] with pool settings (maxOpen: %d, maxIdle: %d, maxLifetime: %v, maxIdleTime: %v)",
		c.DBName, c.MaxOpenConnections, c.MaxIdleConnections, constant.OracleConnMaxLifetime, constant.OracleConnMaxIdleTime)

	return nil
}

// GetDB returns a pointer to the Oracle connection, initializing it if necessary.
func (oc *Connection) GetDB() (*sql.DB, error) {
	if oc.ConnectionDB == nil {
		if err := oc.Connect(); err != nil {
			oc.Logger.Infof("ERRCONECT %s", err)
			return nil, err
		}
	}

	return oc.ConnectionDB, nil
}

// ValidateFieldsInSchemaOracle validate if all fields exist on oracle schema table
func ValidateFieldsInSchemaOracle(expectedFields []string, schema TableSchema, countIfTableExist *int32) (missing []string) {
	columnSet := make(map[string]struct{}, len(schema.Columns))
	for _, col := range schema.Columns {
		columnSet[strings.ToUpper(col.Name)] = struct{}{}
	}

	for _, field := range expectedFields {
		*countIfTableExist++

		if _, exists := columnSet[strings.ToUpper(field)]; !exists {
			missing = append(missing, field)
		}
	}

	return
}
