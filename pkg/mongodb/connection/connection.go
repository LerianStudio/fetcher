package connection

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ConnectionType enumerates all supported connection providers.
type ConnectionType string

const (
	ConnectionTypeOracle     ConnectionType = "ORACLE"
	ConnectionTypeSQLServer  ConnectionType = "SQL_SERVER"
	ConnectionTypePostgreSQL ConnectionType = "POSTGRESQL"
	ConnectionTypeMongoDB    ConnectionType = "MONGODB"
	ConnectionTypeMySQL      ConnectionType = "MYSQL"
)

var validConnectionTypes = map[ConnectionType]struct{}{
	ConnectionTypeOracle:     {},
	ConnectionTypeSQLServer:  {},
	ConnectionTypePostgreSQL: {},
	ConnectionTypeMongoDB:    {},
	ConnectionTypeMySQL:      {},
}

var generateConnectionUUID = uuid.NewV7

// IsValid reports whether the connection type is part of the supported enum.
func (ct ConnectionType) IsValid() bool {
	_, ok := validConnectionTypes[ct]
	return ok
}

// SSLConfig captures TLS metadata stored inside ssl JSONB field.
type SSLConfig struct {
	Mode string `json:"mode" bson:"mode,omitempty"`
	CA   string `json:"ca" bson:"ca,omitempty"`
	Cert string `json:"cert" bson:"cert,omitempty"`
	Key  string `json:"key" bson:"key,omitempty"`
}

// Connection models the API payload used by the management layer.
type Connection struct {
	ID                uuid.UUID      `json:"id"`
	OrganizationID    uuid.UUID      `json:"organizationId"`
	ConfigName        string         `json:"configName"`
	Type              ConnectionType `json:"type"`
	Host              string         `json:"host"`
	Port              int            `json:"port"`
	DatabaseName      string         `json:"databaseName"`
	Username          string         `json:"username"`
	PasswordEncrypted string         `json:"passwordEncrypted"`
	SSL               *SSLConfig     `json:"ssl,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	DeletedAt         *time.Time     `json:"deletedAt,omitempty"`
}

// ValidateForCreate ensures the connection has the required fields for insertion.
// Note: This method trims whitespace from ConfigName and Host as a side effect.
func (conn *Connection) ValidateForCreate() error {
	if conn == nil {
		return errors.New("connection entity is required")
	}

	conn.ConfigName = strings.TrimSpace(conn.ConfigName)
	conn.Host = strings.TrimSpace(conn.Host)

	if conn.OrganizationID == uuid.Nil {
		return errors.New("organization ID is required")
	}

	if conn.ConfigName == "" {
		return errors.New("config name is required")
	}

	if !conn.Type.IsValid() {
		return errors.New("invalid connection type")
	}

	if conn.Port <= 0 {
		return errors.New("port must be a positive integer")
	}

	if conn.Host == "" {
		return errors.New("host is required")
	}

	if conn.PasswordEncrypted == "" {
		return errors.New("password_encrypted is required")
	}

	return nil
}

// ValidateForUpdate ensures the connection has all fields required for updates.
func (conn *Connection) ValidateForUpdate() error {
	if err := conn.ValidateForCreate(); err != nil {
		return err
	}

	if conn.ID == uuid.Nil {
		return errors.New("connection ID is required")
	}

	return nil
}

// ConnectionMongoDBModel represents how a connection is stored in MongoDB.
type ConnectionMongoDBModel struct {
	ID                uuid.UUID      `bson:"_id"`
	OrganizationID    uuid.UUID      `bson:"organization_id"`
	ConfigName        string         `bson:"config_name"`
	Type              ConnectionType `bson:"type"`
	Host              string         `bson:"host"`
	Port              int            `bson:"port"`
	DatabaseName      string         `bson:"database_name"`
	Username          string         `bson:"username"`
	PasswordEncrypted string         `bson:"password_encrypted"`
	SSL               *SSLConfig     `bson:"ssl,omitempty"`
	CreatedAt         time.Time      `bson:"created_at"`
	UpdatedAt         time.Time      `bson:"updated_at"`
	DeletedAt         *time.Time     `bson:"deleted_at"`
}

// ToEntity converts a MongoDB model into the public entity representation.
func (cm *ConnectionMongoDBModel) ToEntity() *Connection {
	if cm == nil {
		return nil
	}

	return &Connection{
		ID:                cm.ID,
		OrganizationID:    cm.OrganizationID,
		ConfigName:        cm.ConfigName,
		Type:              cm.Type,
		Host:              cm.Host,
		Port:              cm.Port,
		DatabaseName:      cm.DatabaseName,
		Username:          cm.Username,
		PasswordEncrypted: cm.PasswordEncrypted,
		SSL:               cm.SSL,
		CreatedAt:         cm.CreatedAt,
		UpdatedAt:         cm.UpdatedAt,
		DeletedAt:         cm.DeletedAt,
	}
}

// FromEntity prepares the MongoDB model for persistence using validation defaults.
// Note: This method mutates the input entity by setting ID (if uuid.Nil), CreatedAt, and UpdatedAt (if zero).
func (cm *ConnectionMongoDBModel) FromEntity(conn *Connection) error {
	if conn == nil {
		return errors.New("connection entity is required")
	}

	id := conn.ID
	if id == uuid.Nil {
		generated, err := generateConnectionUUID()
		if err != nil {
			return err
		}

		id = generated
		conn.ID = generated
	}

	now := time.Now().UTC()
	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = now
	}

	if conn.UpdatedAt.IsZero() {
		conn.UpdatedAt = now
	}

	cm.ID = id
	cm.OrganizationID = conn.OrganizationID
	cm.ConfigName = conn.ConfigName
	cm.Type = conn.Type
	cm.Host = conn.Host
	cm.Port = conn.Port
	cm.DatabaseName = conn.DatabaseName
	cm.Username = conn.Username
	cm.PasswordEncrypted = conn.PasswordEncrypted
	cm.SSL = conn.SSL
	cm.CreatedAt = conn.CreatedAt
	cm.UpdatedAt = conn.UpdatedAt
	cm.DeletedAt = conn.DeletedAt

	return nil
}
