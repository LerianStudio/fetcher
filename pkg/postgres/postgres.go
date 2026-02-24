package postgres

import (
	"database/sql"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/constant"

	"github.com/LerianStudio/lib-commons/v2/commons/log"
	_ "github.com/jackc/pgx/v5/stdlib" // Registers the "pgx" driver with database/sql via init() – required for sql.Open("pgx", ...)
)

// Connection is a hub which deals with postgres connections.
type Connection struct {
	ConnectionString   string
	DBName             string
	ConnectionDB       *sql.DB
	Connected          bool
	Logger             log.Logger
	MaxOpenConnections int
	MaxIdleConnections int
}

// Connect initializes the connection with the PostgreSQL DB.
func (c *Connection) Connect() error {
	c.Logger.Info("Connecting to PostgreSQL...")

	db, err := sql.Open("pgx", c.ConnectionString)
	if err != nil {
		c.Logger.Errorf("Error opening connection: %v", err)
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		closeErr := db.Close()
		if closeErr != nil {
			c.Logger.Errorf("Error closing connection: %v", closeErr)
		}

		c.Logger.Errorf("Error pinging PostgreSQL: %v", err)

		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	db.SetMaxOpenConns(c.MaxOpenConnections)
	db.SetMaxIdleConns(c.MaxIdleConnections)
	db.SetConnMaxLifetime(constant.PostgresConnMaxLifetime)
	db.SetConnMaxIdleTime(constant.PostgresConnMaxIdleTime)

	c.ConnectionDB = db
	c.Connected = true

	c.Logger.Infof("Connected to PostgreSQL [%s] with pool settings (maxOpen: %d, maxIdle: %d, maxLifetime: %v, maxIdleTime: %v)",
		c.DBName, c.MaxOpenConnections, c.MaxIdleConnections, constant.PostgresConnMaxLifetime, constant.PostgresConnMaxIdleTime)

	return nil
}

// GetDB returns a pointer to the postgres connection, initializing it if necessary.
func (pc *Connection) GetDB() (*sql.DB, error) {
	if pc.ConnectionDB == nil {
		if err := pc.Connect(); err != nil {
			pc.Logger.Errorf("ERR_CONNECT: failed to connect to PostgreSQL: %v", err)
			return nil, err
		}
	}

	return pc.ConnectionDB, nil
}
