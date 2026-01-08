package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// InternalDBConnection holds connection info using Docker network hostnames.
type InternalDBConnection struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"userName"`
	Password string `json:"password"`
	Database string `json:"database"`

	// SSL configuration
	SSLEnabled    bool   `json:"sslEnabled,omitempty"`
	SSLMode       string `json:"sslMode,omitempty"`
	SSLCACert     string `json:"sslCaCert,omitempty"`
	SSLClientCert string `json:"sslClientCert,omitempty"`
	SSLClientKey  string `json:"sslClientKey,omitempty"`
}

// InfraPorts holds the mapped ports for infrastructure services.
type InfraPorts struct {
	MongoMain      string `json:"mongoMain"`
	MongoExternal  string `json:"mongoExternal"`
	RabbitMQ       string `json:"rabbitmq"`
	SeaweedFSFiler string `json:"seaweedfsFiler"`
	Redis          string `json:"redis"`
	Postgres       string `json:"postgres"`
	MySQL          string `json:"mysql"`
	SQLServer      string `json:"sqlserver"`
	Oracle         string `json:"oracle"`
}

// InfraConfig holds all infrastructure connection information.
// This is saved to a file by start-infra and read by tests to detect reuse.
type InfraConfig struct {
	NetworkName      string     `json:"networkName"`
	MongoMainURI     string     `json:"mongoMainUri"`
	MongoExternalURI string     `json:"mongoExternalUri"`
	RabbitMQURI      string     `json:"rabbitmqUri"`
	SeaweedFSURL     string     `json:"seaweedfsUrl"`
	RedisURL         string     `json:"redisUrl"`
	PostgresURL      string     `json:"postgresUrl"`
	MySQLURL         string     `json:"mysqlUrl"`
	SQLServerURL     string     `json:"sqlserverUrl"`
	OracleURL        string     `json:"oracleUrl"`
	Ports            InfraPorts `json:"ports"`

	// Connection info using Docker network hostnames.
	// Works for both containers (Docker DNS) and host (via /etc/hosts mapping).
	PostgresInternal      InternalDBConnection `json:"postgresInternal"`
	MySQLInternal         InternalDBConnection `json:"mysqlInternal"`
	SQLServerInternal     InternalDBConnection `json:"sqlserverInternal"`
	OracleInternal        InternalDBConnection `json:"oracleInternal"`
	MongoExternalInternal InternalDBConnection `json:"mongoExternalInternal"`
}

// Save writes the infrastructure config to a file.
func (c *InfraConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadInfraConfig loads infrastructure config from a file.
func LoadInfraConfig(path string) (*InfraConfig, error) {
	// #nosec G304 -- path is controlled test infrastructure config path, not user input
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config InfraConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// InfraConfigExists checks if infrastructure config file exists.
func InfraConfigExists() bool {
	_, err := os.Stat(InfraConfigPath)
	return err == nil
}

// RemoveInfraConfig removes the infrastructure config file.
func RemoveInfraConfig() error {
	if InfraConfigExists() {
		return os.Remove(InfraConfigPath)
	}

	return nil
}
