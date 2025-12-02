package model

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/crypto"

	"github.com/google/uuid"
)

type Connection struct {
	ID                   uuid.UUID
	OrganizationID       uuid.UUID
	ConfigName           string
	Type                 DBType
	Host                 string
	Port                 int
	DatabaseName         string
	Username             string
	PasswordEncrypted    string
	EncryptionKeyVersion string
	SSL                  *SSLConfig
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            *time.Time
}

type SSLConfig struct {
	Mode string
	CA   string
	Cert string
	Key  string
}

func NewConnection(
	ctx context.Context,
	cryptor crypto.Cryptor,
	organizationID uuid.UUID,
	configName string,
	typ string,
	host string,
	port int,
	databaseName string,
	username string,
	password string,
	sslMode *string,
	sslCA *string,
	sslCert *string,
	sslKey *string,
) (*Connection, error) {
	var ssl *SSLConfig
	if sslMode != nil {
		ssl = &SSLConfig{}
		ssl.Mode = *sslMode
		if sslCA != nil {
			ssl.CA = *sslCA
		}
		if sslCert != nil {
			ssl.Cert = *sslCert
		}
		if sslKey != nil {
			ssl.Key = *sslKey
		}
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	var passwordEncrypted, encryptionKeyVersion string
	if password != "" {
		passwordEncrypted, encryptionKeyVersion, err = cryptor.Encrypt(ctx, password)
		if err != nil {
			return nil, pkg.ValidateInternalError(err, "connection")
		}
	}

	dbType, err := NewTypeFromString(typ)
	if err != nil {
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	connection := Connection{
		ID:                   id,
		OrganizationID:       organizationID,
		ConfigName:           configName,
		Type:                 dbType,
		Host:                 host,
		Port:                 port,
		DatabaseName:         databaseName,
		Username:             username,
		PasswordEncrypted:    passwordEncrypted,
		EncryptionKeyVersion: encryptionKeyVersion,
		SSL:                  ssl,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	return &connection, connection.IsValid()
}

// IsValid trims and enforces required fields.
func (conn *Connection) IsValid() error {
	conn.ConfigName = strings.TrimSpace(conn.ConfigName)
	conn.Host = strings.TrimSpace(conn.Host)
	conn.DatabaseName = strings.TrimSpace(conn.DatabaseName)
	conn.Username = strings.TrimSpace(conn.Username)

	var requiredFields = make(map[string]string)
	var knownInvalidFields = make(map[string]string)

	if !conn.Type.IsValid() {
		knownInvalidFields["type"] = "invalid connection type"
	}
	if conn.OrganizationID == uuid.Nil {
		requiredFields["organization_id"] = "organization ID is required"
	}
	if conn.ConfigName == "" {
		requiredFields["config_name"] = "config name is required"
	}
	if len(conn.ConfigName) < 3 || len(conn.ConfigName) > 100 {
		knownInvalidFields["config_name"] = "config name must be between 3 and 100 characters"
	}
	if conn.Port <= 0 {
		requiredFields["port"] = "port must be a positive integer"
	}
	if conn.Host == "" {
		requiredFields["host"] = "host is required"
	}
	if conn.DatabaseName == "" {
		requiredFields["database_name"] = "database name is required"
	}
	if conn.Username == "" {
		requiredFields["username"] = "username is required"
	}
	if conn.PasswordEncrypted == "" {
		requiredFields["password_encrypted"] = "password_encrypted is required"
	}
	if conn.ID == uuid.Nil {
		requiredFields["id"] = "connection ID is required"
	}

	if conn.SSL != nil {
		if conn.SSL.Mode == "" {
			requiredFields["ssl.mode"] = "SSL mode is required"
		}
		if conn.SSL.CA == "" {
			requiredFields["ssl.ca"] = "SSL CA is required"
		}
	}

	if len(requiredFields) == 0 && len(knownInvalidFields) == 0 {
		return nil
	} else {
		return pkg.ValidateBadRequestFieldsError(
			requiredFields,
			knownInvalidFields,
			"connection",
			nil,
		)
	}
}

// ApplyPatch applies partial updates to the Connection.
func (conn *Connection) ApplyPatch(
	ctx context.Context,
	enc crypto.Cryptor,
	configName *string,
	typ *string,
	host *string,
	port *int,
	dbName *string,
	username *string,
	password *string,
	sslMode *string,
	sslCA *string,
	sslCert *string,
	sslKey *string,
) error {
	if configName != nil {
		conn.ConfigName = *configName
	}
	if typ != nil {
		connType, errParse := NewTypeFromString(*typ)
		if errParse != nil {
			return pkg.ValidateInternalError(errParse, "connection")
		}
		conn.Type = connType
	}
	if host != nil {
		conn.Host = *host
	}
	if port != nil {
		conn.Port = *port
	}
	if dbName != nil {
		conn.DatabaseName = *dbName
	}
	if username != nil {
		conn.Username = *username
	}
	if password != nil {
		if enc == nil {
			return pkg.ValidateInternalError(errors.New("cryptor is required to encrypt password"), "connection")
		}
		passwordEncrypted, encryptionKeyVersion, err := enc.Encrypt(ctx, *password)
		if err != nil {
			return pkg.ValidateInternalError(err, "connection")
		}
		conn.PasswordEncrypted = passwordEncrypted
		conn.EncryptionKeyVersion = encryptionKeyVersion
	}

	if sslMode != nil {
		ssl := SSLConfig{}
		ssl.Mode = *sslMode
		if sslCA != nil {
			ssl.CA = *sslCA
		}
		if sslCert != nil {
			ssl.Cert = *sslCert
		}
		if sslKey != nil {
			ssl.Key = *sslKey
		}
		conn.SSL = &ssl
	}

	conn.UpdatedAt = time.Now().UTC()
	return conn.IsValid()
}

// SoftDelete marks the Connection as deleted.
func (conn *Connection) SoftDelete(ts time.Time) {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	conn.DeletedAt = &ts
	conn.UpdatedAt = ts
}

// GetPasswordDecrypted decrypts and returns the connection password.
func (conn *Connection) GetPasswordDecrypted(ctx context.Context, cryptor crypto.Cryptor) (string, error) {
	if cryptor == nil {
		return "", errors.New("cryptor is required to decrypt password")
	}
	plain, err := cryptor.Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion)
	if err != nil {
		return "", pkg.ValidateInternalError(err, "connection")
	}
	return plain, nil
}

// ToMapWithMask converts the Connection to a map with sensitive fields masked.
func (conn *Connection) ToMapWithMask() map[string]interface{} {
	var ssl map[string]interface{}
	if conn.SSL != nil {
		ssl = map[string]interface{}{
			"mode": conn.SSL.Mode,
			"ca":   pkg.MaskSecret(conn.SSL.CA),
			"cert": pkg.MaskSecret(conn.SSL.Cert),
			"key":  pkg.MaskSecret(conn.SSL.Key),
		}
	}
	return map[string]interface{}{
		"id":                     conn.ID,
		"organization_id":        conn.OrganizationID,
		"config_name":            conn.ConfigName,
		"type":                   string(conn.Type),
		"host":                   conn.Host,
		"port":                   conn.Port,
		"database_name":          conn.DatabaseName,
		"username":               conn.Username,
		"password_encrypted":     pkg.MaskSecret(conn.PasswordEncrypted),
		"encryption_key_version": pkg.MaskSecret(conn.EncryptionKeyVersion),
		"ssl":                    ssl,
		"created_at":             conn.CreatedAt,
		"updated_at":             conn.UpdatedAt,
		"deleted_at":             conn.DeletedAt,
	}
}

// ##############################################################################################################################################################################
// Request, Response DTOs And Value Objects

type ConnectionInput struct {
	ConfigName   string    `json:"configName" validate:"required" example:"production-db" minLength:"3" maxLength:"100"`
	Type         string    `json:"type" validate:"required,oneof=ORACLE SQL_SERVER POSTGRESQL MONGODB MYSQL" example:"POSTGRESQL"`
	Host         string    `json:"host" validate:"required,hostname|ip" example:"db.example.com"`
	Port         int       `json:"port" validate:"required,min=1,max=65535" example:"5432"`
	DatabaseName string    `json:"databaseName" validate:"required" example:"mydatabase"`
	Username     string    `json:"username" validate:"required" example:"dbuser"`
	Password     string    `json:"password" validate:"required" example:"secretpassword"`
	SSL          *SSLInput `json:"ssl,omitempty"`
}

type SSLInput struct {
	Mode string  `json:"mode" validate:"required" example:"require"`
	CA   string  `json:"ca" validate:"omitempty" example:"-----BEGIN CERTIFICATE-----\n..."`
	Cert *string `json:"cert"`
	Key  *string `json:"key"`
}

type ConnectionResponse struct {
	ID           uuid.UUID    `json:"id"`
	ConfigName   string       `json:"configName"`
	Type         string       `json:"type"`
	Host         string       `json:"host"`
	Port         int          `json:"port"`
	DatabaseName string       `json:"databaseName"`
	Username     string       `json:"username"`
	SSL          *SSLResponse `json:"ssl,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type SSLResponse struct {
	Mode string `json:"mode,omitempty"`
}

// NewConnectionResponseFrom maps a Connection to a ConnectionResponse.
func NewConnectionResponseFrom(conn *Connection) *ConnectionResponse {
	if conn == nil {
		return nil
	}
	resp := &ConnectionResponse{
		ID:           conn.ID,
		ConfigName:   conn.ConfigName,
		Type:         string(conn.Type),
		Host:         conn.Host,
		Port:         conn.Port,
		DatabaseName: conn.DatabaseName,
		Username:     conn.Username,
		CreatedAt:    conn.CreatedAt,
		UpdatedAt:    conn.UpdatedAt,
	}
	if conn.SSL != nil {
		resp.SSL = &SSLResponse{
			Mode: conn.SSL.Mode,
		}
	}
	return resp
}

type DBType string

const (
	TypeOracle     DBType = "ORACLE"
	TypeSQLServer  DBType = "SQL_SERVER"
	TypePostgreSQL DBType = "POSTGRESQL"
	TypeMongoDB    DBType = "MONGODB"
	TypeMySQL      DBType = "MYSQL"
)

var validTypes = map[DBType]struct{}{
	TypeOracle:     {},
	TypeSQLServer:  {},
	TypePostgreSQL: {},
	TypeMongoDB:    {},
	TypeMySQL:      {},
}

func (t DBType) IsValid() bool {
	_, ok := validTypes[t]
	return ok
}

func NewTypeFromString(s string) (DBType, error) {
	typ := DBType(strings.ToUpper(strings.TrimSpace(s)))
	if !typ.IsValid() {
		return "", errors.New("invalid connection type")
	}
	return typ, nil
}
