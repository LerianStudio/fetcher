package connection

import (
	"errors"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/google/uuid"
)

// ConnectionMongoDBModel represents how a connection is stored in MongoDB.
type ConnectionMongoDBModel struct {
	ID                   uuid.UUID              `bson:"_id"`
	OrganizationID       uuid.UUID              `bson:"organization_id"`
	ConfigName           string                 `bson:"config_name"`
	Type                 string                 `bson:"type"`
	Host                 string                 `bson:"host"`
	Port                 int                    `bson:"port"`
	DatabaseName         string                 `bson:"database_name"`
	Username             string                 `bson:"username"`
	PasswordEncrypted    string                 `bson:"password_encrypted"`
	EncryptionKeyVersion string                 `bson:"encryption_key_version"`
	SSL                  *SSLConfigMongoDBModel `bson:"ssl,omitempty"`
	CreatedAt            time.Time              `bson:"created_at"`
	UpdatedAt            time.Time              `bson:"updated_at"`
	DeletedAt            *time.Time             `bson:"deleted_at"`
}

// SSLConfigMongoDBModel represents the SSL configuration stored in MongoDB.
type SSLConfigMongoDBModel struct {
	Mode string `bson:"mode"`
	CA   string `bson:"ca,omitempty"`
	Cert string `bson:"cert,omitempty"`
	Key  string `bson:"key,omitempty"`
}

// ToDomain converts a MongoDB model into the domain entity representation.
func (cm *ConnectionMongoDBModel) ToDomain() (*model.Connection, error) {
	if cm == nil {
		return nil, errors.New("cannot convert nil ConnectionMongoDBModel to domain")
	}

	connType, err := model.NewTypeFromString(cm.Type)
	if err != nil {
		return nil, err
	}

	var ssl *model.SSLConfig
	if cm.SSL != nil {
		ssl = &model.SSLConfig{
			Mode: cm.SSL.Mode,
			CA:   cm.SSL.CA,
			Cert: cm.SSL.Cert,
			Key:  cm.SSL.Key,
		}
	}

	return &model.Connection{
		ID:                   cm.ID,
		OrganizationID:       cm.OrganizationID,
		ConfigName:           cm.ConfigName,
		Type:                 connType,
		Host:                 cm.Host,
		Port:                 cm.Port,
		DatabaseName:         cm.DatabaseName,
		Username:             cm.Username,
		PasswordEncrypted:    cm.PasswordEncrypted,
		EncryptionKeyVersion: cm.EncryptionKeyVersion,
		SSL:                  ssl,
		CreatedAt:            cm.CreatedAt,
		UpdatedAt:            cm.UpdatedAt,
		DeletedAt:            cm.DeletedAt,
	}, nil
}

// NewConnectionMongoDBModelFromDomain creates a MongoDB model from the domain entity.
func NewConnectionMongoDBModelFromDomain(conn *model.Connection) *ConnectionMongoDBModel {
	var ssl *SSLConfigMongoDBModel
	if conn.SSL != nil {
		ssl = &SSLConfigMongoDBModel{
			Mode: conn.SSL.Mode,
			CA:   conn.SSL.CA,
			Cert: conn.SSL.Cert,
			Key:  conn.SSL.Key,
		}
	}

	var cm ConnectionMongoDBModel
	cm.ID = conn.ID
	cm.OrganizationID = conn.OrganizationID
	cm.ConfigName = conn.ConfigName
	cm.Type = string(conn.Type)
	cm.Host = conn.Host
	cm.Port = conn.Port
	cm.DatabaseName = conn.DatabaseName
	cm.Username = conn.Username
	cm.PasswordEncrypted = conn.PasswordEncrypted
	cm.EncryptionKeyVersion = conn.EncryptionKeyVersion
	cm.SSL = ssl
	cm.CreatedAt = conn.CreatedAt
	cm.UpdatedAt = conn.UpdatedAt
	cm.DeletedAt = conn.DeletedAt

	return &cm
}

// ToMapWithMask converts the MongoDB model to a map with sensitive fields masked.
func (cm *ConnectionMongoDBModel) ToMapWithMask() map[string]interface{} {
	result := map[string]interface{}{
		"id":                     cm.ID,
		"organization_id":        cm.OrganizationID,
		"config_name":            cm.ConfigName,
		"type":                   cm.Type,
		"host":                   cm.Host,
		"port":                   cm.Port,
		"database_name":          cm.DatabaseName,
		"username":               cm.Username,
		"password_encrypted":     pkg.MaskSecret(cm.PasswordEncrypted),
		"encryption_key_version": cm.EncryptionKeyVersion,
		"created_at":             cm.CreatedAt,
		"updated_at":             cm.UpdatedAt,
		"deleted_at":             cm.DeletedAt,
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
