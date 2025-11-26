package domain

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/google/uuid"
)

type Type string

const (
	TypeOracle     Type = "ORACLE"
	TypeSQLServer  Type = "SQL_SERVER"
	TypePostgreSQL Type = "POSTGRESQL"
	TypeMongoDB    Type = "MONGODB"
	TypeMySQL      Type = "MYSQL"
)

var validTypes = map[Type]struct{}{
	TypeOracle:     {},
	TypeSQLServer:  {},
	TypePostgreSQL: {},
	TypeMongoDB:    {},
	TypeMySQL:      {},
}

func (t Type) IsValid() bool {
	_, ok := validTypes[t]
	return ok
}

func NewTypeFromString(s string) (Type, error) {
	typ := Type(strings.ToUpper(strings.TrimSpace(s)))
	if !typ.IsValid() {
		return "", errors.New("invalid connection type")
	}
	return typ, nil
}

var defaultPorts = map[Type]int{
	TypeOracle:     1521,
	TypeSQLServer:  1433,
	TypePostgreSQL: 5432,
	TypeMongoDB:    27017,
	TypeMySQL:      3306,
}

// Encryptor defines the contract needed to encrypt a plain password.
type Encryptor interface {
	Encrypt(ctx context.Context, plain string) (cipherTextBase64 string, keyVersion string, err error)
}

// Repository defines the domain port for connections.
type Repository interface {
	Create(ctx context.Context, conn *Connection) (*Connection, error)
	Update(ctx context.Context, conn *Connection) (*Connection, error)
	Delete(ctx context.Context, id, organizationID uuid.UUID, deletedAt time.Time) error
	FindByID(ctx context.Context, id, organizationID uuid.UUID) (*Connection, error)
	FindByOrganizationAndName(ctx context.Context, organizationID uuid.UUID, configName string) (*Connection, error)
	FindByOrganizationAndDatabaseName(ctx context.Context, organizationID uuid.UUID, databaseName string) (*Connection, error)
	List(ctx context.Context, filters *ListFilterParams) ([]*Connection, error)
}

// CreateParams holds parameters for creating a Connection.
type CreateParams struct {
	OrganizationID    uuid.UUID
	ConfigName        string
	Type              string
	Host              string
	Port              int
	DatabaseName      string
	Username          string
	Password          string
	PasswordEncrypted string
	KeyVersion        string
	SSL               *SSLConfig
}

// ListFilterParams holds filtering parameters for listing connections.
type ListFilterParams struct {
	OrganizationID uuid.UUID
	ConfigName     string
	Types          []Type
	Host           string
	DatabaseName   string
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	IncludeDeleted bool
	Limit          int
	Page           int
	SortOrder      constant.Order
}

// NewListFilter creates a ListFilterParams from input parameters for listing connections.
func NewListFilter(
	orgID uuid.UUID,
	configName, host, dbName, typ, sortOrder, createdAt string,
	page, limit int,
) (*ListFilterParams, error) {
	if orgID == uuid.Nil {
		return nil, errors.New("organization ID is required")
	}

	f := &ListFilterParams{
		OrganizationID: orgID,
		ConfigName:     configName,
		Host:           host,
		DatabaseName:   dbName,
		Limit:          limit,
		Page:           page,
		SortOrder: func(sortOrder string) constant.Order {
			if strings.EqualFold(sortOrder, string(constant.Asc)) {
				return constant.Asc
			}
			return constant.Desc
		}(sortOrder),
	}

	if typ != "" {
		ct := Type(strings.ToUpper(strings.TrimSpace(typ)))
		if !ct.IsValid() {
			return nil, errors.New("invalid connection type")
		}
		f.Types = []Type{ct}
	}

	if createdAt != "" {
		t, err := time.Parse("2006-01-02", createdAt)
		if err != nil {
			return nil, errors.New("invalid createdAt format, expected YYYY-MM-DD")
		}
		start := t
		end := t.Add(24 * time.Hour)
		f.CreatedFrom = &start
		f.CreatedTo = &end
	}

	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 1000 {
		f.Limit = 1000
	}
	if f.Page < 0 {
		f.Page = 0
	}

	trimmedConfig := strings.TrimSpace(f.ConfigName)
	trimmedHost := strings.TrimSpace(f.Host)
	trimmedDB := strings.TrimSpace(f.DatabaseName)
	if trimmedConfig == "" && len(f.Types) == 0 && trimmedHost == "" && trimmedDB == "" && f.CreatedFrom == nil && f.CreatedTo == nil {
		return nil, errors.New("at least one filter must be provided")
	}

	return f, nil
}

// Connection is the domain entity.
type Connection struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	ConfigName        string
	Type              Type
	Host              string
	Port              int
	DatabaseName      string
	Username          string
	PasswordEncrypted string
	KeyVersion        string
	SSL               *SSLConfig
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

// SSLConfig holds SSL configuration for a connection.
type SSLConfig struct {
	Mode string
	CA   string
	Cert string
	Key  string
}

// NewWithEncrypted creates a new Connection with an already encrypted password.
func NewWithEncrypted(params CreateParams) (*Connection, error) {
	return newConnection(params)
}

// NewWithPlain creates a new Connection, encrypting the password.
func NewWithPlain(ctx context.Context, enc Encryptor, params CreateParams) (*Connection, error) {
	if enc == nil {
		return nil, errors.New("encryptor is required")
	}

	cipher, keyVersion, err := enc.Encrypt(ctx, params.Password)
	if err != nil {
		return nil, err
	}
	params.PasswordEncrypted = cipher
	params.KeyVersion = keyVersion

	return newConnection(params)
}

// newConnection is a helper to create a Connection entity.
func newConnection(params CreateParams) (*Connection, error) {
	connType, err := NewTypeFromString(params.Type)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	conn := &Connection{
		ID:                uuid.Nil,
		OrganizationID:    params.OrganizationID,
		ConfigName:        params.ConfigName,
		Type:              connType,
		Host:              params.Host,
		Port:              params.Port,
		DatabaseName:      params.DatabaseName,
		Username:          params.Username,
		PasswordEncrypted: params.PasswordEncrypted,
		KeyVersion:        params.KeyVersion,
		SSL:               params.SSL,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if conn.Port == 0 {
		if def, ok := defaultPorts[conn.Type]; ok {
			conn.Port = def
		}
	}

	if err := conn.ValidateForCreate(); err != nil {
		return nil, err
	}

	return conn, nil
}

// ValidateForCreate trims and enforces required fields.
func (conn *Connection) ValidateForCreate() error {
	conn.ConfigName = strings.TrimSpace(conn.ConfigName)
	conn.Host = strings.TrimSpace(conn.Host)
	conn.DatabaseName = strings.TrimSpace(conn.DatabaseName)
	conn.Username = strings.TrimSpace(conn.Username)

	if conn.OrganizationID == uuid.Nil {
		return errors.New("organization ID is required")
	}
	if conn.ConfigName == "" {
		return errors.New("config name is required")
	}
	if len(conn.ConfigName) < 3 || len(conn.ConfigName) > 100 {
		return errors.New("config name must be between 3 and 100 characters")
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
	if conn.DatabaseName == "" {
		return errors.New("database name is required")
	}
	if conn.Username == "" {
		return errors.New("username is required")
	}
	if conn.PasswordEncrypted == "" {
		return errors.New("password_encrypted is required")
	}

	if conn.SSL != nil {
		if conn.SSL.Mode == "" {
			return errors.New("SSL mode is required")
		}
		if conn.SSL.CA == "" {
			return errors.New("SSL CA is required")
		}
	}

	return nil
}

// ValidateForUpdate trims and enforces required fields for update.
func (conn *Connection) ValidateForUpdate() error {
	if err := conn.ValidateForCreate(); err != nil {
		return err
	}
	if conn.ID == uuid.Nil {
		return errors.New("connection ID is required")
	}
	return nil
}

// ApplyPatch applies partial updates to the Connection.
func (conn *Connection) ApplyPatch(
	ctx context.Context,
	enc Encryptor,
	configName *string,
	typ *string,
	host *string,
	port *int,
	dbName *string,
	username *string,
	password *string,
	ssl *SSLConfig,
) error {
	if configName == nil &&
		typ == nil &&
		host == nil &&
		port == nil &&
		dbName == nil &&
		username == nil &&
		password == nil &&
		ssl == nil {
		return errors.New("no fields to update")
	}

	if configName != nil {
		conn.ConfigName = *configName
	}
	if typ != nil {
		connType, errParse := NewTypeFromString(*typ)
		if errParse != nil {
			return errParse
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
			return errors.New("encryptor is required to update password")
		}
		cipher, kv, err := enc.Encrypt(ctx, *password)
		if err != nil {
			return err
		}
		conn.PasswordEncrypted = cipher
		conn.KeyVersion = kv
	}
	if ssl != nil {
		conn.SSL = ssl
	}
	if conn.Port == 0 {
		if def, ok := defaultPorts[conn.Type]; ok {
			conn.Port = def
		}
	}

	conn.UpdatedAt = time.Now().UTC()
	return conn.ValidateForUpdate()
}

// SoftDelete marks the Connection as deleted.
func (conn *Connection) SoftDelete(ts time.Time) {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	conn.DeletedAt = &ts
	conn.UpdatedAt = ts
}

// GenerateID ensures an ID is present.
func (conn *Connection) GenerateID() error {
	if conn.ID != uuid.Nil {
		return nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	conn.ID = id
	return nil
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
		"id":                 conn.ID,
		"organization_id":    conn.OrganizationID,
		"config_name":        conn.ConfigName,
		"type":               string(conn.Type),
		"host":               conn.Host,
		"port":               conn.Port,
		"database_name":      conn.DatabaseName,
		"username":           conn.Username,
		"password_encrypted": pkg.MaskSecret(conn.PasswordEncrypted),
		"key_version":        conn.KeyVersion,
		"ssl":                ssl,
		"created_at":         conn.CreatedAt,
		"updated_at":         conn.UpdatedAt,
		"deleted_at":         conn.DeletedAt,
	}
}
