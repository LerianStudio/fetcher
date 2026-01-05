package containers

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/sijms/go-ora/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// OracleContainer wraps an Oracle XE testcontainer with connection info.
type OracleContainer struct {
	Container    testcontainers.Container
	URL          string
	Host         string
	Port         string
	InternalHost string
	Internal     config.InternalDBConnection
}

// OracleOptions configures Oracle container startup.
type OracleOptions struct {
	NetworkName   string
	NetworkAlias  string
	FixedHostPort string
	Password      string
	InitScript    string
}

// DefaultOracleOptions returns default Oracle options.
func DefaultOracleOptions(networkName string) OracleOptions {
	return OracleOptions{
		NetworkName:  networkName,
		NetworkAlias: "oracle-external",
		Password:     "TestPass123",
	}
}

// StartOracle starts an Oracle XE container with the given options.
func StartOracle(ctx context.Context, opts OracleOptions) (*OracleContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "gvenzl/oracle-xe:21-slim-faststart",
		ExposedPorts: []string{"1521/tcp"},
		Env: map[string]string{
			"ORACLE_PASSWORD": opts.Password,
		},
		WaitingFor: wait.ForLog("DATABASE IS READY TO USE!").WithStartupTimeout(config.OracleStartupTimeout),
	}

	if opts.NetworkName != "" {
		req.Networks = []string{opts.NetworkName}
		req.NetworkAliases = map[string][]string{
			opts.NetworkName: {opts.NetworkAlias},
		}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start Oracle: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Oracle host: %w", err)
	}

	port, err := container.MappedPort(ctx, "1521")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Oracle port: %w", err)
	}

	connStr := fmt.Sprintf("oracle://system:%s@%s:%s/XEPDB1", opts.Password, host, port.Port())

	// Run init script if provided
	if opts.InitScript != "" {
		if err := runOracleInit(ctx, connStr, opts.InitScript); err != nil {
			_ = container.Terminate(ctx)
			return nil, fmt.Errorf("failed to run init script: %w", err)
		}
	}

	return &OracleContainer{
		Container:    container,
		URL:          connStr,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
		Internal: config.InternalDBConnection{
			Host:     opts.NetworkAlias,
			Port:     1521,
			Username: "system",
			Password: opts.Password,
			Database: "XEPDB1",
		},
	}, nil
}

// runOracleInit executes the init script on Oracle.
// It splits the script by / statements (PL/SQL block terminators) and semicolons.
// PL/SQL blocks with EXECUTE IMMEDIATE 'DROP TABLE...' are handled specially to ignore "table does not exist" errors.
func runOracleInit(ctx context.Context, connStr, script string) error {
	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer db.Close()

	// Split by / on its own line (PL/SQL block terminator)
	slashPattern := regexp.MustCompile(`(?m)^\s*/\s*$`)
	blocks := slashPattern.Split(script, -1)

	// Pattern to extract table name from EXECUTE IMMEDIATE 'DROP TABLE xxx'
	dropTablePattern := regexp.MustCompile(`(?i)EXECUTE\s+IMMEDIATE\s+'DROP\s+TABLE\s+(\w+)'`)

	// Pattern to find BEGIN blocks (to split mixed blocks)
	beginPattern := regexp.MustCompile(`(?mi)^BEGIN\s*$`)

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		// Check if block contains both regular SQL and a PL/SQL block
		upperBlock := strings.ToUpper(block)
		hasBegin := strings.Contains(upperBlock, "\nBEGIN\n") || strings.HasPrefix(upperBlock, "BEGIN\n")
		hasExecuteImmediate := strings.Contains(upperBlock, "EXECUTE IMMEDIATE")

		if hasBegin && hasExecuteImmediate {
			// Split at BEGIN to separate regular SQL from PL/SQL
			parts := beginPattern.Split(block, 2)

			// Process regular SQL part (before BEGIN)
			if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
				if err := executeOracleStatements(ctx, db, parts[0]); err != nil {
					return err
				}
			}

			// Handle the DROP TABLE from PL/SQL block
			if matches := dropTablePattern.FindStringSubmatch(block); len(matches) > 1 {
				tableName := matches[1]
				_, err := db.ExecContext(ctx, "DROP TABLE "+tableName)
				if err != nil && !strings.Contains(err.Error(), "ORA-00942") {
					return fmt.Errorf("failed to drop table %s: %w", tableName, err)
				}
			}
			continue
		}

		// Pure PL/SQL block with DROP TABLE
		if hasBegin && hasExecuteImmediate {
			if matches := dropTablePattern.FindStringSubmatch(block); len(matches) > 1 {
				tableName := matches[1]
				_, err := db.ExecContext(ctx, "DROP TABLE "+tableName)
				if err != nil && !strings.Contains(err.Error(), "ORA-00942") {
					return fmt.Errorf("failed to drop table %s: %w", tableName, err)
				}
			}
			continue
		}

		// Regular SQL block
		if err := executeOracleStatements(ctx, db, block); err != nil {
			return err
		}
	}

	return nil
}

// executeOracleStatements executes SQL statements from a block, splitting by semicolons.
func executeOracleStatements(ctx context.Context, db *sql.DB, block string) error {
	statements := strings.Split(block, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// Remove comment lines from the statement
		stmt = removeCommentLines(stmt)
		if stmt == "" {
			continue
		}

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to run statement: %w", err)
		}
	}
	return nil
}

// removeCommentLines removes SQL comment lines (starting with --) from a statement.
func removeCommentLines(stmt string) string {
	lines := strings.Split(stmt, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

// Stop terminates the Oracle container.
func (o *OracleContainer) Stop(ctx context.Context) error {
	if o.Container != nil {
		return o.Container.Terminate(ctx)
	}
	return nil
}
