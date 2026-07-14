package model

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/datasource/sslmode"

	"github.com/google/uuid"
)

type Connection struct {
	ID                   uuid.UUID
	ProductName          string
	ConfigName           string
	Type                 DBType
	Host                 string
	Port                 int
	DatabaseName         string
	Schema               *string
	Username             string
	PasswordEncrypted    string
	EncryptionKeyVersion string
	SSL                  *SSLConfig
	Metadata             *map[string]any
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            *time.Time
	password             string
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
	productName string,
	configName string,
	typ string,
	host string,
	port int,
	databaseName string,
	schema *string,
	username string,
	password string,
	metadata *map[string]any,
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
		return nil, pkg.ValidateBadRequestFieldsError(
			nil,
			map[string]string{"type": "invalid connection type"},
			"connection",
			nil,
		)
	}

	connection := Connection{
		ID:                   id,
		ProductName:          productName,
		ConfigName:           configName,
		Type:                 dbType,
		Host:                 host,
		Port:                 port,
		DatabaseName:         databaseName,
		Schema:               schema,
		Username:             username,
		PasswordEncrypted:    passwordEncrypted,
		EncryptionKeyVersion: encryptionKeyVersion,
		Metadata:             metadata,
		SSL:                  ssl,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	return &connection, connection.IsValid()
}

// IsValid trims and enforces required fields.
func (conn *Connection) IsValid() error {
	conn.normalizeFields()

	requiredFields := conn.validateRequiredFields()
	knownInvalidFields := conn.validateFieldValues()

	if len(requiredFields) == 0 && len(knownInvalidFields) == 0 {
		return nil
	}

	return pkg.ValidateBadRequestFieldsError(
		requiredFields,
		knownInvalidFields,
		"connection",
		nil,
	)
}

// normalizeFields trims whitespace from string fields
func (conn *Connection) normalizeFields() {
	conn.ProductName = strings.TrimSpace(conn.ProductName)
	conn.ConfigName = strings.TrimSpace(conn.ConfigName)
	conn.Host = strings.TrimSpace(conn.Host)
	conn.DatabaseName = strings.TrimSpace(conn.DatabaseName)
	conn.Username = strings.TrimSpace(conn.Username)
}

// validateRequiredFields validates that all required fields are present
func (conn *Connection) validateRequiredFields() map[string]string {
	requiredFields := make(map[string]string)

	if conn.ProductName == "" {
		requiredFields["product_name"] = "product name is required"
	}

	if conn.ConfigName == "" {
		requiredFields["config_name"] = "config name is required"
	} else {
		configNameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		if !configNameRegex.MatchString(conn.ConfigName) {
			requiredFields["config_name"] = "config name can only contain alphanumeric characters, underscores, and hyphens"
		}
	}

	if conn.Port <= 0 {
		requiredFields["port"] = "port must be a positive integer"
	} else if conn.Port > 65535 {
		requiredFields["port"] = "port must be between 1 and 65535"
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

	conn.validateSSLRequiredFields(requiredFields)

	return requiredFields
}

// validateSSLRequiredFields validates SSL-related required fields
func (conn *Connection) validateSSLRequiredFields(requiredFields map[string]string) {
	if conn.SSL == nil {
		return
	}

	if conn.SSL.Mode == "" {
		requiredFields["ssl.mode"] = "SSL mode is required"
	}

	if conn.SSL.CA == "" {
		requiredFields["ssl.ca"] = "SSL CA is required"
	}
}

// validateFieldValues validates field values and formats
func (conn *Connection) validateFieldValues() map[string]string {
	knownInvalidFields := make(map[string]string)

	if !conn.Type.IsValid() {
		knownInvalidFields["type"] = "invalid connection type"
	}

	if len(conn.ConfigName) < 3 || len(conn.ConfigName) > 100 {
		knownInvalidFields["config_name"] = "config name must be between 3 and 100 characters"
	}

	if conn.SSL != nil && conn.SSL.Mode != "" {
		if err := conn.validateSSLModeForType(); err != nil {
			knownInvalidFields["ssl.mode"] = err.Error()
		}
	}

	return knownInvalidFields
}

// validateSSLModeForType validates that the SSL mode is valid for the connection's database type.
// Each database driver has its own set of valid SSL/TLS modes.
func (conn *Connection) validateSSLModeForType() error {
	var err error

	switch conn.Type {
	case TypeMySQL:
		err = sslmode.ValidateMySQLMode(conn.SSL.Mode)
	case TypePostgreSQL:
		err = sslmode.ValidatePostgreSQLMode(conn.SSL.Mode)
	case TypeOracle:
		err = sslmode.ValidateOracleMode(conn.SSL.Mode)
	case TypeMongoDB:
		err = sslmode.ValidateMongoDBMode(conn.SSL.Mode)
	case TypeSQLServer:
		err = sslmode.ValidateSQLServerMode(conn.SSL.Mode)
	default:
		return nil
	}

	return err
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
	schema *string,
	username *string,
	password *string,
	metadata *map[string]any,
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
			return pkg.ValidateBadRequestFieldsError(
				nil,
				map[string]string{"type": "invalid connection type"},
				"connection",
				nil,
			)
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

	if schema != nil {
		conn.Schema = schema
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

	if metadata != nil {
		conn.Metadata = metadata
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

// AssignProductName associates a legacy (unassigned) connection to a product name.
// This is a one-time operation for migration purposes.
func (conn *Connection) AssignProductName(productName string) error {
	if conn.ProductName != "" {
		return pkg.ValidateBusinessError(
			constant.ErrConnectionAlreadyAssigned,
			"connection",
		)
	}

	conn.ProductName = productName
	conn.UpdatedAt = time.Now().UTC()

	return nil
}

// SetPlaintextPassword sets the internal password field directly.
// Used for in-memory connections resolved from tenant-manager where
// the password is not encrypted.
func (conn *Connection) SetPlaintextPassword(password string) {
	conn.password = password
}

// GetPlaintextPassword returns the internal password field.
// For in-memory connections (EncryptionKeyVersion == ""), this returns the plaintext password
// set by SetPlaintextPassword. For encrypted connections, this returns the decrypted password
// after DecryptPassword or GetPasswordDecrypted has been called.
func (conn *Connection) GetPlaintextPassword() string {
	return conn.password
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

	conn.password = plain

	return plain, nil
}

// DecryptPassword decrypts and stores the connection password internally.
func (conn *Connection) DecryptPassword(ctx context.Context, cryptor crypto.Cryptor) error {
	if cryptor == nil {
		return errors.New("cryptor is required to decrypt password")
	}

	plain, err := cryptor.Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion)
	if err != nil {
		return pkg.ValidateInternalError(err, "connection")
	}

	conn.password = plain

	return nil
}

// ToMapWithMask converts the Connection to a map with sensitive fields masked.
func (conn *Connection) ToMapWithMask() map[string]any {
	var ssl map[string]any
	if conn.SSL != nil {
		ssl = map[string]any{
			"mode": conn.SSL.Mode,
			"ca":   pkg.MaskSecret(conn.SSL.CA),
			"cert": pkg.MaskSecret(conn.SSL.Cert),
			"key":  pkg.MaskSecret(conn.SSL.Key),
		}
	}

	return map[string]any{
		"id":                     conn.ID,
		"product_name":           conn.ProductName,
		"config_name":            conn.ConfigName,
		"type":                   string(conn.Type),
		"host":                   conn.Host,
		"port":                   conn.Port,
		"database_name":          conn.DatabaseName,
		"schema":                 conn.Schema,
		"username":               conn.Username,
		"metadata":               conn.Metadata,
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
	ConfigName   string          `json:"configName" validate:"required" doc:"Unique configuration name used to reference this connection." example:"production-db"`
	Type         string          `json:"type" validate:"required,oneof=ORACLE SQL_SERVER POSTGRESQL MONGODB MYSQL" doc:"Database engine type." example:"POSTGRESQL"`
	Host         string          `json:"host" validate:"required,hostname|ip,safe_host" doc:"Database server hostname or IP address." example:"db.example.com"`
	Port         int             `json:"port" validate:"required,min=1,max=65535" doc:"Database server TCP port." example:"5432"`
	DatabaseName string          `json:"databaseName" validate:"required" doc:"Database name to connect to." example:"mydatabase"`
	Schema       string          `json:"schema,omitempty" doc:"Optional default database schema." example:"public"`
	Username     string          `json:"userName" validate:"required" doc:"Database authentication username." example:"dbuser"`
	Password     string          `json:"password" validate:"required" doc:"Database authentication password." example:"secretpassword"`
	SSL          *SSLInput       `json:"ssl,omitempty" doc:"Optional SSL/TLS connection settings." example:"{\"mode\":\"require\",\"ca\":\"certificate-authority\"}"`
	Metadata     *map[string]any `json:"metadata,omitempty" doc:"Optional application-defined connection metadata." example:"{\"environment\":\"production\",\"region\":\"us-east-1\"}"`
}

type SSLInput struct {
	Mode string  `json:"mode" validate:"omitempty" doc:"Driver-specific SSL/TLS mode." example:"require"`
	CA   string  `json:"ca" validate:"omitempty" doc:"PEM-encoded certificate authority used to verify the server." example:"-----BEGIN CERTIFICATE-----\n..."`
	Cert *string `json:"cert" doc:"Optional PEM-encoded client certificate." example:"-----BEGIN CERTIFICATE-----\n..."`
	Key  *string `json:"key" doc:"Optional PEM-encoded private key for the client certificate." example:"-----BEGIN PRIVATE KEY-----\n..."`
}

func (conn *ConnectionInput) ToMapWithMask() map[string]any {
	var ssl map[string]any
	if conn.SSL != nil {
		ssl = map[string]any{
			"mode": conn.SSL.Mode,
			"ca":   pkg.MaskSecret(conn.SSL.CA),
			"cert": pkg.MaskSecretPtr(conn.SSL.Cert),
			"key":  pkg.MaskSecretPtr(conn.SSL.Key),
		}
	}

	return map[string]any{
		"config_name":   conn.ConfigName,
		"type":          conn.Type,
		"host":          conn.Host,
		"port":          conn.Port,
		"database_name": conn.DatabaseName,
		"schema":        conn.Schema,
		"username":      conn.Username,
		"password":      pkg.MaskSecret(conn.Password),
		"metadata":      conn.Metadata,
		"ssl":           ssl,
	}
}

// IsEmpty returns true if all fields are empty/nil.
func (conn *ConnectionInput) IsEmpty() bool {
	if conn == nil {
		return true
	}

	return conn.ConfigName == "" &&
		conn.Type == "" &&
		conn.Host == "" &&
		conn.Port == 0 &&
		conn.DatabaseName == "" &&
		conn.Schema == "" &&
		conn.Username == "" &&
		conn.Password == "" &&
		(conn.SSL == nil || conn.SSL.IsEmpty()) &&
		conn.Metadata == nil
}

// IsEmpty returns true if all SSL fields are empty/nil.
// This is used to treat "ssl": {} as if SSL was not provided at all.
func (s *SSLInput) IsEmpty() bool {
	if s == nil {
		return true
	}

	return s.Mode == "" && s.CA == "" && s.Cert == nil && s.Key == nil
}

// ConnectionUpdateInput is the DTO for PATCH /connections/:id requests.
// All fields are pointers to distinguish between "not provided" (nil) and "provided with value".
// This enables true RFC 7396 JSON Merge Patch semantics.
type ConnectionUpdateInput struct {
	ConfigName   *string         `json:"configName,omitempty" validate:"omitempty,min=3,max=100" doc:"New configuration name for the connection." example:"production-db"`
	Type         *string         `json:"type,omitempty" validate:"omitempty,oneof=ORACLE SQL_SERVER POSTGRESQL MONGODB MYSQL" doc:"New database engine type." example:"POSTGRESQL"`
	Host         *string         `json:"host,omitempty" validate:"omitempty,hostname|ip,safe_host" doc:"New database server hostname or IP address." example:"db.example.com"`
	Port         *int            `json:"port,omitempty" validate:"omitempty,min=1,max=65535" doc:"New database server TCP port." example:"5432"`
	DatabaseName *string         `json:"databaseName,omitempty" validate:"omitempty" doc:"New database name to connect to." example:"mydatabase"`
	Schema       *string         `json:"schema,omitempty" doc:"New default database schema." example:"public"`
	Username     *string         `json:"userName,omitempty" validate:"omitempty" doc:"New database authentication username." example:"dbuser"`
	Password     *string         `json:"password,omitempty" validate:"omitempty" doc:"New database authentication password." example:"secretpassword"`
	SSL          *SSLUpdateInput `json:"ssl,omitempty" doc:"SSL/TLS settings to update." example:"{\"mode\":\"verify-full\"}"`
	Metadata     *map[string]any `json:"metadata,omitempty" doc:"Application-defined metadata that replaces the current metadata." example:"{\"environment\":\"production\",\"region\":\"us-east-1\"}"`
}

// SSLUpdateInput is the nested DTO for SSL configuration in PATCH requests.
// All fields are pointers for partial update semantics.
type SSLUpdateInput struct {
	Mode *string `json:"mode,omitempty" validate:"omitempty" doc:"New driver-specific SSL/TLS mode." example:"require"`
	CA   *string `json:"ca,omitempty" validate:"omitempty" doc:"New PEM-encoded certificate authority used to verify the server." example:"-----BEGIN CERTIFICATE-----\n..."`
	Cert *string `json:"cert,omitempty" doc:"New PEM-encoded client certificate." example:"-----BEGIN CERTIFICATE-----\n..."`
	Key  *string `json:"key,omitempty" doc:"New PEM-encoded private key for the client certificate." example:"-----BEGIN PRIVATE KEY-----\n..."`
}

// ToMapWithMask converts the ConnectionUpdateInput to a map with sensitive fields masked.
// Used for logging and telemetry without exposing secrets.
func (conn *ConnectionUpdateInput) ToMapWithMask() map[string]any {
	result := make(map[string]any)

	if conn.ConfigName != nil {
		result["config_name"] = *conn.ConfigName
	}

	if conn.Type != nil {
		result["type"] = *conn.Type
	}

	if conn.Host != nil {
		result["host"] = *conn.Host
	}

	if conn.Port != nil {
		result["port"] = *conn.Port
	}

	if conn.DatabaseName != nil {
		result["database_name"] = *conn.DatabaseName
	}

	if conn.Schema != nil {
		result["schema"] = *conn.Schema
	}

	if conn.Username != nil {
		result["username"] = *conn.Username
	}

	if conn.Password != nil {
		result["password"] = pkg.MaskSecret(*conn.Password)
	}

	if conn.Metadata != nil {
		result["metadata"] = conn.Metadata
	}

	if conn.SSL != nil {
		ssl := make(map[string]any)
		if conn.SSL.Mode != nil {
			ssl["mode"] = *conn.SSL.Mode
		}

		if conn.SSL.CA != nil {
			ssl["ca"] = pkg.MaskSecret(*conn.SSL.CA)
		}

		if conn.SSL.Cert != nil {
			ssl["cert"] = pkg.MaskSecret(*conn.SSL.Cert)
		}

		if conn.SSL.Key != nil {
			ssl["key"] = pkg.MaskSecret(*conn.SSL.Key)
		}

		result["ssl"] = ssl
	}

	return result
}

// IsEmpty returns true if all fields are empty/nil.
func (conn *ConnectionUpdateInput) IsEmpty() bool {
	if conn == nil {
		return true
	}

	if conn.ConfigName != nil ||
		conn.Type != nil ||
		conn.Host != nil ||
		conn.Port != nil ||
		conn.DatabaseName != nil ||
		conn.Schema != nil ||
		conn.Username != nil ||
		conn.Password != nil ||
		conn.Metadata != nil {
		return false
	}

	if conn.SSL != nil {
		return conn.SSL.IsEmpty()
	}

	return true
}

// IsEmpty returns true if all SSL fields are empty/nil.
func (s *SSLUpdateInput) IsEmpty() bool {
	if s == nil {
		return true
	}

	return s.Mode == nil && s.CA == nil && s.Cert == nil && s.Key == nil
}

type ConnectionResponse struct {
	ID           uuid.UUID       `json:"id" doc:"Unique connection identifier." example:"018f47a6-3e5f-7b9a-8c1d-2e3f4a5b6c7d"`
	ProductName  string          `json:"productName,omitempty" doc:"Product that owns the connection." example:"midaz"`
	ConfigName   string          `json:"configName" doc:"Configuration name used to reference the connection." example:"production-db"`
	Type         string          `json:"type" doc:"Database engine type." example:"POSTGRESQL"`
	Host         string          `json:"host" doc:"Database server hostname or IP address." example:"db.example.com"`
	Port         int             `json:"port" doc:"Database server TCP port." example:"5432"`
	DatabaseName string          `json:"databaseName" doc:"Connected database name." example:"mydatabase"`
	Schema       *string         `json:"schema,omitempty" doc:"Default database schema, when configured." example:"public"`
	Username     string          `json:"userName" doc:"Database authentication username." example:"dbuser"`
	SSL          *SSLResponse    `json:"ssl,omitempty" doc:"Configured SSL/TLS settings with secrets omitted." example:"{\"mode\":\"require\"}"`
	Metadata     *map[string]any `json:"metadata,omitempty" doc:"Application-defined connection metadata." example:"{\"environment\":\"production\",\"region\":\"us-east-1\"}"`
	CreatedAt    time.Time       `json:"createdAt" doc:"Timestamp when the connection was created." example:"2026-01-15T10:30:00Z"`
	UpdatedAt    time.Time       `json:"updatedAt" doc:"Timestamp when the connection was last updated." example:"2026-01-16T14:45:00Z"`
}

type ConnectionTestResponse struct {
	Status    string `json:"status" doc:"Connectivity test outcome." example:"success"`
	Message   string `json:"message" doc:"Human-readable connectivity test result." example:"Connection established successfully"`
	LatencyMs int64  `json:"latencyMs" doc:"Round-trip connection latency in milliseconds." example:"18"`
}

type SSLResponse struct {
	Mode string `json:"mode,omitempty" doc:"Configured driver-specific SSL/TLS mode." example:"require"`
}

// NewConnectionResponseFrom maps a Connection to a ConnectionResponse.
func NewConnectionResponseFrom(conn *Connection) *ConnectionResponse {
	if conn == nil {
		return nil
	}

	resp := &ConnectionResponse{
		ID:           conn.ID,
		ProductName:  conn.ProductName,
		ConfigName:   conn.ConfigName,
		Type:         string(conn.Type),
		Host:         conn.Host,
		Port:         conn.Port,
		DatabaseName: conn.DatabaseName,
		Schema:       conn.Schema,
		Username:     conn.Username,
		Metadata:     conn.Metadata,
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

// ConnectionSchemaResponse is the response DTO for GET /v1/management/connections/{id}/schema.
// It contains the connection details along with the list of tables/collections and their fields.
type ConnectionSchemaResponse struct {
	ID           string         `json:"id" doc:"Unique connection identifier." example:"018f47a6-3e5f-7b9a-8c1d-2e3f4a5b6c7d"`
	ConfigName   string         `json:"configName" doc:"Configuration name used to reference the connection." example:"production-db"`
	DatabaseName string         `json:"databaseName" doc:"Database whose schema was discovered." example:"mydatabase"`
	Type         string         `json:"type" doc:"Database engine type." example:"POSTGRESQL"`
	Tables       []TableDetails `json:"tables" doc:"Discovered tables or collections and their fields." example:"[{\"name\":\"public.accounts\",\"fields\":[\"id\",\"name\",\"created_at\"]}]"`
}

// TableDetails contains information about a table or collection.
// The Name field is the qualified name (e.g., "schema.table" for SQL databases
// or "database.collection" for MongoDB).
type TableDetails struct {
	Name   string   `json:"name" doc:"Qualified table or collection name." example:"public.accounts"`
	Fields []string `json:"fields" doc:"Fields discovered in the table or collection." example:"[\"id\",\"name\",\"created_at\"]"`
}

// NewConnectionSchemaFrom creates a ConnectionSchemaResponse from a Connection and a list of tables.
func NewConnectionSchemaFrom(conn *Connection, tables []TableDetails) *ConnectionSchemaResponse {
	if conn == nil {
		return nil
	}

	if tables == nil {
		tables = []TableDetails{}
	}

	return &ConnectionSchemaResponse{
		ID:           conn.ID.String(),
		ConfigName:   conn.ConfigName,
		DatabaseName: conn.DatabaseName,
		Type:         string(conn.Type),
		Tables:       tables,
	}
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
