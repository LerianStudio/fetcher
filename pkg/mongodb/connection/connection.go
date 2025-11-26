package connection

import (
	"errors"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	domainConn "github.com/LerianStudio/fetcher/pkg/domain"
	"github.com/google/uuid"
)

var generateConnectionUUID = uuid.NewV7

// ConnectionMongoDBModel represents how a connection is stored in MongoDB.
type ConnectionMongoDBModel struct {
	ID                uuid.UUID              `bson:"_id"`
	OrganizationID    uuid.UUID              `bson:"organization_id"`
	ConfigName        string                 `bson:"config_name"`
	Type              string                 `bson:"type"`
	Host              string                 `bson:"host"`
	Port              int                    `bson:"port"`
	DatabaseName      string                 `bson:"database_name"`
	Username          string                 `bson:"username"`
	PasswordEncrypted string                 `bson:"password_encrypted"`
	KeyVersion        string                 `bson:"key_version"`
	SSL               *SSLConfigMongoDBModel `bson:"ssl,omitempty"`
	CreatedAt         time.Time              `bson:"created_at"`
	UpdatedAt         time.Time              `bson:"updated_at"`
	DeletedAt         *time.Time             `bson:"deleted_at"`
}

type SSLConfigMongoDBModel struct {
	Mode string `bson:"mode"`
	CA   string `bson:"ca,omitempty"`
	Cert string `bson:"cert,omitempty"`
	Key  string `bson:"key,omitempty"`
}

func NewConnectionMongoDBModelFromDomain(conn *domainConn.Connection) (*ConnectionMongoDBModel, error) {
	if conn == nil {
		return nil, errors.New("connection entity is required")
	}

	var cm ConnectionMongoDBModel
	if err := cm.FromDomain(conn); err != nil {
		return nil, err
	}

	return &cm, nil
}

// ToDomain converts a MongoDB model into the domain entity representation.
func (cm *ConnectionMongoDBModel) ToDomain() *domainConn.Connection {
	if cm == nil {
		return nil
	}

	connType, err := domainConn.NewTypeFromString(cm.Type)
	if err != nil {
		return nil
	}

	var ssl *domainConn.SSLConfig
	if cm.SSL != nil {
		ssl = &domainConn.SSLConfig{
			Mode: cm.SSL.Mode,
			CA:   cm.SSL.CA,
			Cert: cm.SSL.Cert,
			Key:  cm.SSL.Key,
		}
	}

	return &domainConn.Connection{
		ID:                cm.ID,
		OrganizationID:    cm.OrganizationID,
		ConfigName:        cm.ConfigName,
		Type:              connType,
		Host:              cm.Host,
		Port:              cm.Port,
		DatabaseName:      cm.DatabaseName,
		Username:          cm.Username,
		PasswordEncrypted: cm.PasswordEncrypted,
		KeyVersion:        cm.KeyVersion,
		SSL:               ssl,
		CreatedAt:         cm.CreatedAt,
		UpdatedAt:         cm.UpdatedAt,
		DeletedAt:         cm.DeletedAt,
	}
}

// FromDomain prepares the MongoDB model for persistence using the domain entity.
// Note: This mutates the domain entity to set ID and timestamps if missing.
func (cm *ConnectionMongoDBModel) FromDomain(conn *domainConn.Connection) error {
	if conn == nil {
		return errors.New("connection entity is required")
	}

	if conn.ID == uuid.Nil {
		generated, err := generateConnectionUUID()
		if err != nil {
			return err
		}
		conn.ID = generated
	}

	now := time.Now().UTC()
	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = now
	}
	if conn.UpdatedAt.IsZero() {
		conn.UpdatedAt = now
	}

	var ssl *SSLConfigMongoDBModel
	if conn.SSL != nil {
		ssl = &SSLConfigMongoDBModel{
			Mode: conn.SSL.Mode,
			CA:   conn.SSL.CA,
			Cert: conn.SSL.Cert,
			Key:  conn.SSL.Key,
		}
	}

	cm.ID = conn.ID
	cm.OrganizationID = conn.OrganizationID
	cm.ConfigName = conn.ConfigName
	cm.Type = string(conn.Type)
	cm.Host = conn.Host
	cm.Port = conn.Port
	cm.DatabaseName = conn.DatabaseName
	cm.Username = conn.Username
	cm.PasswordEncrypted = conn.PasswordEncrypted
	cm.KeyVersion = conn.KeyVersion
	cm.SSL = ssl
	cm.CreatedAt = conn.CreatedAt
	cm.UpdatedAt = conn.UpdatedAt
	cm.DeletedAt = conn.DeletedAt

	return nil
}

// ToMapWithMask converts the MongoDB model to a map with sensitive fields masked.
func (cm *ConnectionMongoDBModel) ToMapWithMask() map[string]interface{} {
	result := map[string]interface{}{
		"id":                 cm.ID,
		"organization_id":    cm.OrganizationID,
		"config_name":        cm.ConfigName,
		"type":               cm.Type,
		"host":               cm.Host,
		"port":               cm.Port,
		"database_name":      cm.DatabaseName,
		"username":           cm.Username,
		"password_encrypted": pkg.MaskSecret(cm.PasswordEncrypted),
		"key_version":        cm.KeyVersion,
		"created_at":         cm.CreatedAt,
		"updated_at":         cm.UpdatedAt,
		"deleted_at":         cm.DeletedAt,
	}

	if cm.SSL != nil {
		result["ssl"] = map[string]interface{}{
			"mode": cm.SSL.Mode,
			"ca":   pkg.MaskSecret(cm.SSL.CA),
			"cert": pkg.MaskSecret(cm.SSL.Cert),
			"key":  pkg.MaskSecret(cm.SSL.Key),
		}
	}

	return result
}
