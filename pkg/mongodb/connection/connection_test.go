package connection

import (
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/google/uuid"
)

// TestConnectionMongoDBModel_ToDomain tests the ToDomain method.
func TestConnectionMongoDBModel_ToDomain(t *testing.T) {
	now := time.Now().UTC()
	connID := uuid.New()

	tests := []struct {
		name        string
		model       *ConnectionMongoDBModel
		wantErr     bool
		errContains string
		checkResult func(t *testing.T, conn *model.Connection)
	}{
		{
			name:        "nil model returns error",
			model:       nil,
			wantErr:     true,
			errContains: "cannot convert nil",
		},
		{
			name: "valid model without SSL",
			model: &ConnectionMongoDBModel{
				ID: connID,

				ConfigName:           "test-connection",
				Type:                 "POSTGRESQL",
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
				SSL:                  nil,
				CreatedAt:            now,
				UpdatedAt:            now,
				DeletedAt:            nil,
			},
			wantErr: false,
			checkResult: func(t *testing.T, conn *model.Connection) {
				if conn.ID != connID {
					t.Fatalf("expected ID %s, got %s", connID, conn.ID)
				}
				if conn.ConfigName != "test-connection" {
					t.Fatalf("expected ConfigName 'test-connection', got %s", conn.ConfigName)
				}
				if conn.Type != model.TypePostgreSQL {
					t.Fatalf("expected Type POSTGRESQL, got %s", conn.Type)
				}
				if conn.Host != "localhost" {
					t.Fatalf("expected Host 'localhost', got %s", conn.Host)
				}
				if conn.Port != 5432 {
					t.Fatalf("expected Port 5432, got %d", conn.Port)
				}
				if conn.DatabaseName != "testdb" {
					t.Fatalf("expected DatabaseName 'testdb', got %s", conn.DatabaseName)
				}
				if conn.Username != "testuser" {
					t.Fatalf("expected Username 'testuser', got %s", conn.Username)
				}
				if conn.PasswordEncrypted != "encrypted-password" {
					t.Fatalf("expected PasswordEncrypted 'encrypted-password', got %s", conn.PasswordEncrypted)
				}
				if conn.EncryptionKeyVersion != "v1" {
					t.Fatalf("expected EncryptionKeyVersion 'v1', got %s", conn.EncryptionKeyVersion)
				}
				if conn.SSL != nil {
					t.Fatalf("expected SSL nil, got %+v", conn.SSL)
				}
				if !conn.CreatedAt.Equal(now) {
					t.Fatalf("expected CreatedAt %v, got %v", now, conn.CreatedAt)
				}
				if !conn.UpdatedAt.Equal(now) {
					t.Fatalf("expected UpdatedAt %v, got %v", now, conn.UpdatedAt)
				}
				if conn.DeletedAt != nil {
					t.Fatalf("expected DeletedAt nil, got %v", conn.DeletedAt)
				}
			},
		},
		{
			name: "valid model with SSL",
			model: &ConnectionMongoDBModel{
				ID: connID,

				ConfigName:           "ssl-connection",
				Type:                 "POSTGRESQL",
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
				SSL: &SSLConfigMongoDBModel{
					Mode: "require",
					CA:   "ca-cert-content",
					Cert: "client-cert-content",
					Key:  "client-key-content",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: false,
			checkResult: func(t *testing.T, conn *model.Connection) {
				if conn.SSL == nil {
					t.Fatal("expected SSL to be set")
				}
				if conn.SSL.Mode != "require" {
					t.Fatalf("expected SSL Mode 'require', got %s", conn.SSL.Mode)
				}
				if conn.SSL.CA != "ca-cert-content" {
					t.Fatalf("expected SSL CA 'ca-cert-content', got %s", conn.SSL.CA)
				}
				if conn.SSL.Cert != "client-cert-content" {
					t.Fatalf("expected SSL Cert 'client-cert-content', got %s", conn.SSL.Cert)
				}
				if conn.SSL.Key != "client-key-content" {
					t.Fatalf("expected SSL Key 'client-key-content', got %s", conn.SSL.Key)
				}
			},
		},
		{
			name: "valid model with DeletedAt",
			model: &ConnectionMongoDBModel{
				ID: connID,

				ConfigName:           "deleted-connection",
				Type:                 "POSTGRESQL",
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
				CreatedAt:            now,
				UpdatedAt:            now,
				DeletedAt:            &now,
			},
			wantErr: false,
			checkResult: func(t *testing.T, conn *model.Connection) {
				if conn.DeletedAt == nil {
					t.Fatal("expected DeletedAt to be set")
				}
				if !conn.DeletedAt.Equal(now) {
					t.Fatalf("expected DeletedAt %v, got %v", now, *conn.DeletedAt)
				}
			},
		},
		{
			name: "invalid database type",
			model: &ConnectionMongoDBModel{
				ID: connID,

				ConfigName:           "invalid-type-connection",
				Type:                 "INVALID_TYPE",
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
				CreatedAt:            now,
				UpdatedAt:            now,
			},
			wantErr:     true,
			errContains: "invalid connection type",
		},
		{
			name: "empty database type",
			model: &ConnectionMongoDBModel{
				ID: connID,

				ConfigName:           "empty-type-connection",
				Type:                 "",
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
				CreatedAt:            now,
				UpdatedAt:            now,
			},
			wantErr:     true,
			errContains: "invalid connection type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := tt.model.ToEntity()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing '%s', got '%s'", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if conn == nil {
				t.Fatal("expected non-nil connection")
			}

			if tt.checkResult != nil {
				tt.checkResult(t, conn)
			}
		})
	}
}

// TestConnectionMongoDBModel_ToDomain_AllDatabaseTypes tests that all database types are correctly converted.
func TestConnectionMongoDBModel_ToDomain_AllDatabaseTypes(t *testing.T) {
	now := time.Now().UTC()

	dbTypes := []struct {
		mongoType  string
		domainType model.DBType
	}{
		{"POSTGRESQL", model.TypePostgreSQL},
		{"MYSQL", model.TypeMySQL},
		{"MONGODB", model.TypeMongoDB},
		{"ORACLE", model.TypeOracle},
		{"SQL_SERVER", model.TypeSQLServer},
	}

	for _, tt := range dbTypes {
		t.Run(tt.mongoType, func(t *testing.T) {
			mongoModel := &ConnectionMongoDBModel{
				ID: uuid.New(),

				ConfigName:           "test-" + tt.mongoType,
				Type:                 tt.mongoType,
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted",
				EncryptionKeyVersion: "v1",
				CreatedAt:            now,
				UpdatedAt:            now,
			}

			conn, err := mongoModel.ToEntity()
			if err != nil {
				t.Fatalf("unexpected error for type %s: %v", tt.mongoType, err)
			}

			if conn.Type != tt.domainType {
				t.Fatalf("expected Type %s, got %s", tt.domainType, conn.Type)
			}
		})
	}
}

// TestNewConnectionMongoDBModelFromDomain tests the NewConnectionMongoDBModelFromDomain function.
func TestNewConnectionMongoDBModelFromDomain(t *testing.T) {
	now := time.Now().UTC()
	connID := uuid.New()

	tests := []struct {
		name        string
		connection  *model.Connection
		checkResult func(t *testing.T, mongoModel *ConnectionMongoDBModel)
	}{
		{
			name: "valid connection without SSL",
			connection: &model.Connection{
				ID: connID,

				ConfigName:           "test-connection",
				Type:                 model.TypePostgreSQL,
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
				SSL:                  nil,
				CreatedAt:            now,
				UpdatedAt:            now,
				DeletedAt:            nil,
			},
			checkResult: func(t *testing.T, mongoModel *ConnectionMongoDBModel) {
				if mongoModel.ID != connID {
					t.Fatalf("expected ID %s, got %s", connID, mongoModel.ID)
				}
				if mongoModel.ConfigName != "test-connection" {
					t.Fatalf("expected ConfigName 'test-connection', got %s", mongoModel.ConfigName)
				}
				if mongoModel.Type != "POSTGRESQL" {
					t.Fatalf("expected Type 'POSTGRESQL', got %s", mongoModel.Type)
				}
				if mongoModel.Host != "localhost" {
					t.Fatalf("expected Host 'localhost', got %s", mongoModel.Host)
				}
				if mongoModel.Port != 5432 {
					t.Fatalf("expected Port 5432, got %d", mongoModel.Port)
				}
				if mongoModel.DatabaseName != "testdb" {
					t.Fatalf("expected DatabaseName 'testdb', got %s", mongoModel.DatabaseName)
				}
				if mongoModel.Username != "testuser" {
					t.Fatalf("expected Username 'testuser', got %s", mongoModel.Username)
				}
				if mongoModel.PasswordEncrypted != "encrypted-password" {
					t.Fatalf("expected PasswordEncrypted 'encrypted-password', got %s", mongoModel.PasswordEncrypted)
				}
				if mongoModel.EncryptionKeyVersion != "v1" {
					t.Fatalf("expected EncryptionKeyVersion 'v1', got %s", mongoModel.EncryptionKeyVersion)
				}
				if mongoModel.SSL != nil {
					t.Fatalf("expected SSL nil, got %+v", mongoModel.SSL)
				}
				if !mongoModel.CreatedAt.Equal(now) {
					t.Fatalf("expected CreatedAt %v, got %v", now, mongoModel.CreatedAt)
				}
				if !mongoModel.UpdatedAt.Equal(now) {
					t.Fatalf("expected UpdatedAt %v, got %v", now, mongoModel.UpdatedAt)
				}
				if mongoModel.DeletedAt != nil {
					t.Fatalf("expected DeletedAt nil, got %v", mongoModel.DeletedAt)
				}
			},
		},
		{
			name: "valid connection with SSL",
			connection: &model.Connection{
				ID: connID,

				ConfigName:           "ssl-connection",
				Type:                 model.TypePostgreSQL,
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
				SSL: &model.SSLConfig{
					Mode: "require",
					CA:   "ca-cert-content",
					Cert: "client-cert-content",
					Key:  "client-key-content",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			checkResult: func(t *testing.T, mongoModel *ConnectionMongoDBModel) {
				if mongoModel.SSL == nil {
					t.Fatal("expected SSL to be set")
				}
				if mongoModel.SSL.Mode != "require" {
					t.Fatalf("expected SSL Mode 'require', got %s", mongoModel.SSL.Mode)
				}
				if mongoModel.SSL.CA != "ca-cert-content" {
					t.Fatalf("expected SSL CA 'ca-cert-content', got %s", mongoModel.SSL.CA)
				}
				if mongoModel.SSL.Cert != "client-cert-content" {
					t.Fatalf("expected SSL Cert 'client-cert-content', got %s", mongoModel.SSL.Cert)
				}
				if mongoModel.SSL.Key != "client-key-content" {
					t.Fatalf("expected SSL Key 'client-key-content', got %s", mongoModel.SSL.Key)
				}
			},
		},
		{
			name: "valid connection with DeletedAt",
			connection: &model.Connection{
				ID: connID,

				ConfigName:           "deleted-connection",
				Type:                 model.TypePostgreSQL,
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
				CreatedAt:            now,
				UpdatedAt:            now,
				DeletedAt:            &now,
			},
			checkResult: func(t *testing.T, mongoModel *ConnectionMongoDBModel) {
				if mongoModel.DeletedAt == nil {
					t.Fatal("expected DeletedAt to be set")
				}
				if !mongoModel.DeletedAt.Equal(now) {
					t.Fatalf("expected DeletedAt %v, got %v", now, *mongoModel.DeletedAt)
				}
			},
		},
		{
			name: "valid connection with partial SSL (only mode and CA)",
			connection: &model.Connection{
				ID: connID,

				ConfigName:           "partial-ssl-connection",
				Type:                 model.TypePostgreSQL,
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
				SSL: &model.SSLConfig{
					Mode: "verify-ca",
					CA:   "ca-cert-content",
					Cert: "",
					Key:  "",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			checkResult: func(t *testing.T, mongoModel *ConnectionMongoDBModel) {
				if mongoModel.SSL == nil {
					t.Fatal("expected SSL to be set")
				}
				if mongoModel.SSL.Mode != "verify-ca" {
					t.Fatalf("expected SSL Mode 'verify-ca', got %s", mongoModel.SSL.Mode)
				}
				if mongoModel.SSL.CA != "ca-cert-content" {
					t.Fatalf("expected SSL CA 'ca-cert-content', got %s", mongoModel.SSL.CA)
				}
				if mongoModel.SSL.Cert != "" {
					t.Fatalf("expected SSL Cert empty, got %s", mongoModel.SSL.Cert)
				}
				if mongoModel.SSL.Key != "" {
					t.Fatalf("expected SSL Key empty, got %s", mongoModel.SSL.Key)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mongoModel := NewConnectionMongoDBModelFromDomain(tt.connection)

			if mongoModel == nil {
				t.Fatal("expected non-nil MongoDB model")
			}

			if tt.checkResult != nil {
				tt.checkResult(t, mongoModel)
			}
		})
	}
}

// TestNewConnectionMongoDBModelFromDomain_AllDatabaseTypes tests that all database types are correctly converted.
func TestNewConnectionMongoDBModelFromDomain_AllDatabaseTypes(t *testing.T) {
	now := time.Now().UTC()

	dbTypes := []struct {
		domainType model.DBType
		mongoType  string
	}{
		{model.TypePostgreSQL, "POSTGRESQL"},
		{model.TypeMySQL, "MYSQL"},
		{model.TypeMongoDB, "MONGODB"},
		{model.TypeOracle, "ORACLE"},
		{model.TypeSQLServer, "SQL_SERVER"},
	}

	for _, tt := range dbTypes {
		t.Run(string(tt.domainType), func(t *testing.T) {
			conn := &model.Connection{
				ID: uuid.New(),

				ConfigName:           "test-" + string(tt.domainType),
				Type:                 tt.domainType,
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted",
				EncryptionKeyVersion: "v1",
				CreatedAt:            now,
				UpdatedAt:            now,
			}

			mongoModel := NewConnectionMongoDBModelFromDomain(conn)

			if mongoModel.Type != tt.mongoType {
				t.Fatalf("expected Type %s, got %s", tt.mongoType, mongoModel.Type)
			}
		})
	}
}

// TestConnectionMongoDBModel_RoundTrip tests that domain -> mongo -> domain conversion preserves data.
func TestConnectionMongoDBModel_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond) // Truncate to avoid precision issues
	connID := uuid.New()

	tests := []struct {
		name       string
		connection *model.Connection
	}{
		{
			name: "round trip without SSL",
			connection: &model.Connection{
				ID: connID,

				ConfigName:           "roundtrip-no-ssl",
				Type:                 model.TypePostgreSQL,
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
				SSL:                  nil,
				CreatedAt:            now,
				UpdatedAt:            now,
				DeletedAt:            nil,
			},
		},
		{
			name: "round trip with SSL",
			connection: &model.Connection{
				ID: connID,

				ConfigName:           "roundtrip-with-ssl",
				Type:                 model.TypeMySQL,
				Host:                 "db.example.com",
				Port:                 3306,
				DatabaseName:         "mydb",
				Username:             "admin",
				PasswordEncrypted:    "super-encrypted",
				EncryptionKeyVersion: "v2",
				SSL: &model.SSLConfig{
					Mode: "require",
					CA:   "ca-content",
					Cert: "cert-content",
					Key:  "key-content",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "round trip with DeletedAt",
			connection: &model.Connection{
				ID: connID,

				ConfigName:           "roundtrip-deleted",
				Type:                 model.TypeMongoDB,
				Host:                 "mongo.example.com",
				Port:                 27017,
				DatabaseName:         "mongodb",
				Username:             "mongouser",
				PasswordEncrypted:    "mongo-encrypted",
				EncryptionKeyVersion: "v1",
				CreatedAt:            now,
				UpdatedAt:            now,
				DeletedAt:            &now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Domain -> MongoDB
			mongoModel := NewConnectionMongoDBModelFromDomain(tt.connection)

			// MongoDB -> Domain
			result, err := mongoModel.ToEntity()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Compare fields
			if result.ID != tt.connection.ID {
				t.Fatalf("ID mismatch: expected %s, got %s", tt.connection.ID, result.ID)
			}
			if result.ConfigName != tt.connection.ConfigName {
				t.Fatalf("ConfigName mismatch: expected %s, got %s", tt.connection.ConfigName, result.ConfigName)
			}
			if result.Type != tt.connection.Type {
				t.Fatalf("Type mismatch: expected %s, got %s", tt.connection.Type, result.Type)
			}
			if result.Host != tt.connection.Host {
				t.Fatalf("Host mismatch: expected %s, got %s", tt.connection.Host, result.Host)
			}
			if result.Port != tt.connection.Port {
				t.Fatalf("Port mismatch: expected %d, got %d", tt.connection.Port, result.Port)
			}
			if result.DatabaseName != tt.connection.DatabaseName {
				t.Fatalf("DatabaseName mismatch: expected %s, got %s", tt.connection.DatabaseName, result.DatabaseName)
			}
			if result.Username != tt.connection.Username {
				t.Fatalf("Username mismatch: expected %s, got %s", tt.connection.Username, result.Username)
			}
			if result.PasswordEncrypted != tt.connection.PasswordEncrypted {
				t.Fatalf("PasswordEncrypted mismatch: expected %s, got %s", tt.connection.PasswordEncrypted, result.PasswordEncrypted)
			}
			if result.EncryptionKeyVersion != tt.connection.EncryptionKeyVersion {
				t.Fatalf("EncryptionKeyVersion mismatch: expected %s, got %s", tt.connection.EncryptionKeyVersion, result.EncryptionKeyVersion)
			}

			// Compare SSL
			if tt.connection.SSL == nil {
				if result.SSL != nil {
					t.Fatalf("SSL mismatch: expected nil, got %+v", result.SSL)
				}
			} else {
				if result.SSL == nil {
					t.Fatalf("SSL mismatch: expected %+v, got nil", tt.connection.SSL)
				}
				if result.SSL.Mode != tt.connection.SSL.Mode {
					t.Fatalf("SSL.Mode mismatch: expected %s, got %s", tt.connection.SSL.Mode, result.SSL.Mode)
				}
				if result.SSL.CA != tt.connection.SSL.CA {
					t.Fatalf("SSL.CA mismatch: expected %s, got %s", tt.connection.SSL.CA, result.SSL.CA)
				}
				if result.SSL.Cert != tt.connection.SSL.Cert {
					t.Fatalf("SSL.Cert mismatch: expected %s, got %s", tt.connection.SSL.Cert, result.SSL.Cert)
				}
				if result.SSL.Key != tt.connection.SSL.Key {
					t.Fatalf("SSL.Key mismatch: expected %s, got %s", tt.connection.SSL.Key, result.SSL.Key)
				}
			}

			// Compare timestamps
			if !result.CreatedAt.Equal(tt.connection.CreatedAt) {
				t.Fatalf("CreatedAt mismatch: expected %v, got %v", tt.connection.CreatedAt, result.CreatedAt)
			}
			if !result.UpdatedAt.Equal(tt.connection.UpdatedAt) {
				t.Fatalf("UpdatedAt mismatch: expected %v, got %v", tt.connection.UpdatedAt, result.UpdatedAt)
			}

			// Compare DeletedAt
			if tt.connection.DeletedAt == nil {
				if result.DeletedAt != nil {
					t.Fatalf("DeletedAt mismatch: expected nil, got %v", result.DeletedAt)
				}
			} else {
				if result.DeletedAt == nil {
					t.Fatalf("DeletedAt mismatch: expected %v, got nil", tt.connection.DeletedAt)
				}
				if !result.DeletedAt.Equal(*tt.connection.DeletedAt) {
					t.Fatalf("DeletedAt mismatch: expected %v, got %v", *tt.connection.DeletedAt, *result.DeletedAt)
				}
			}
		})
	}
}

// TestConnectionMongoDBModel_ToMapWithMask tests the ToMapWithMask method.
func TestConnectionMongoDBModel_ToMapWithMask(t *testing.T) {
	now := time.Now().UTC()
	connID := uuid.New()

	t.Run("model without SSL", func(t *testing.T) {
		mongoModel := &ConnectionMongoDBModel{
			ID: connID,

			ConfigName:           "test-connection",
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "super-secret-password",
			EncryptionKeyVersion: "v1",
			SSL:                  nil,
			CreatedAt:            now,
			UpdatedAt:            now,
			DeletedAt:            nil,
		}

		result := mongoModel.ToMapWithMask()

		if result["id"] != connID {
			t.Fatalf("expected id %v, got %v", connID, result["id"])
		}
		if result["config_name"] != "test-connection" {
			t.Fatalf("expected config_name 'test-connection', got %v", result["config_name"])
		}
		if result["type"] != "POSTGRESQL" {
			t.Fatalf("expected type 'POSTGRESQL', got %v", result["type"])
		}
		if result["host"] != "localhost" {
			t.Fatalf("expected host 'localhost', got %v", result["host"])
		}
		if result["port"] != 5432 {
			t.Fatalf("expected port 5432, got %v", result["port"])
		}
		if result["database_name"] != "testdb" {
			t.Fatalf("expected database_name 'testdb', got %v", result["database_name"])
		}
		if result["username"] != "testuser" {
			t.Fatalf("expected username 'testuser', got %v", result["username"])
		}
		// Password should be masked
		if result["password_encrypted"] != "[REDACTED]" {
			t.Fatalf("expected password_encrypted '[REDACTED]', got %v", result["password_encrypted"])
		}
		if result["encryption_key_version"] != "v1" {
			t.Fatalf("expected encryption_key_version 'v1', got %v", result["encryption_key_version"])
		}
		// SSL should not be present in the map
		if _, exists := result["ssl"]; exists && result["ssl"] != nil {
			t.Fatalf("expected ssl to be nil or not present, got %v", result["ssl"])
		}
	})

	t.Run("model with SSL", func(t *testing.T) {
		mongoModel := &ConnectionMongoDBModel{
			ID: connID,

			ConfigName:           "ssl-connection",
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "super-secret-password",
			EncryptionKeyVersion: "v1",
			SSL: &SSLConfigMongoDBModel{
				Mode: "require",
				CA:   "secret-ca-cert",
				Cert: "secret-client-cert",
				Key:  "secret-client-key",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		result := mongoModel.ToMapWithMask()

		ssl, ok := result["ssl"].(map[string]any)
		if !ok {
			t.Fatalf("expected ssl to be map[string]any, got %T", result["ssl"])
		}

		if ssl["mode"] != "require" {
			t.Fatalf("expected ssl.mode 'require', got %v", ssl["mode"])
		}
		// SSL sensitive fields should be masked
		if ssl["ca"] != "[REDACTED]" {
			t.Fatalf("expected ssl.ca '[REDACTED]', got %v", ssl["ca"])
		}
		if ssl["cert"] != "[REDACTED]" {
			t.Fatalf("expected ssl.cert '[REDACTED]', got %v", ssl["cert"])
		}
		if ssl["key"] != "[REDACTED]" {
			t.Fatalf("expected ssl.key '[REDACTED]', got %v", ssl["key"])
		}
	})

	t.Run("model with empty password does not mask", func(t *testing.T) {
		mongoModel := &ConnectionMongoDBModel{
			ID: connID,

			ConfigName:           "empty-password-connection",
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "",
			EncryptionKeyVersion: "v1",
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		result := mongoModel.ToMapWithMask()

		// Empty password should remain empty (not masked)
		if result["password_encrypted"] != "" {
			t.Fatalf("expected password_encrypted '', got %v", result["password_encrypted"])
		}
	})

	t.Run("model with DeletedAt", func(t *testing.T) {
		deletedTime := now.Add(time.Hour)
		mongoModel := &ConnectionMongoDBModel{
			ID: connID,

			ConfigName:           "deleted-connection",
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "encrypted",
			EncryptionKeyVersion: "v1",
			CreatedAt:            now,
			UpdatedAt:            deletedTime,
			DeletedAt:            &deletedTime,
		}

		result := mongoModel.ToMapWithMask()

		deletedAt, ok := result["deleted_at"].(*time.Time)
		if !ok {
			t.Fatalf("expected deleted_at to be *time.Time, got %T", result["deleted_at"])
		}
		if !deletedAt.Equal(deletedTime) {
			t.Fatalf("expected deleted_at %v, got %v", deletedTime, deletedAt)
		}
	})

	t.Run("model with SSL and empty optional fields", func(t *testing.T) {
		mongoModel := &ConnectionMongoDBModel{
			ID: connID,

			ConfigName:           "partial-ssl-connection",
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "encrypted",
			EncryptionKeyVersion: "v1",
			SSL: &SSLConfigMongoDBModel{
				Mode: "verify-ca",
				CA:   "ca-content",
				Cert: "",
				Key:  "",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		result := mongoModel.ToMapWithMask()

		ssl, ok := result["ssl"].(map[string]any)
		if !ok {
			t.Fatalf("expected ssl to be map[string]any, got %T", result["ssl"])
		}

		if ssl["mode"] != "verify-ca" {
			t.Fatalf("expected ssl.mode 'verify-ca', got %v", ssl["mode"])
		}
		if ssl["ca"] != "[REDACTED]" {
			t.Fatalf("expected ssl.ca '[REDACTED]', got %v", ssl["ca"])
		}
		// Empty fields should remain empty (not masked)
		if ssl["cert"] != "" {
			t.Fatalf("expected ssl.cert '', got %v", ssl["cert"])
		}
		if ssl["key"] != "" {
			t.Fatalf("expected ssl.key '', got %v", ssl["key"])
		}
	})
}

// TestConnectionMongoDBModel_EdgeCases tests edge cases for the model.
func TestConnectionMongoDBModel_EdgeCases(t *testing.T) {
	now := time.Now().UTC()

	t.Run("special characters in fields", func(t *testing.T) {
		mongoModel := &ConnectionMongoDBModel{
			ID: uuid.New(),

			ConfigName:           "test-connection_123",
			Type:                 "POSTGRESQL",
			Host:                 "db.example.com",
			Port:                 5432,
			DatabaseName:         "my_database",
			Username:             "user@domain.com",
			PasswordEncrypted:    "p@ssw0rd!#$%^&*(){}[]|\\:\";<>,.?/~`",
			EncryptionKeyVersion: "v1-beta-2",
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		conn, err := mongoModel.ToEntity()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if conn.Username != "user@domain.com" {
			t.Fatalf("expected Username 'user@domain.com', got %s", conn.Username)
		}

		// Round-trip should preserve special characters
		backToMongo := NewConnectionMongoDBModelFromDomain(conn)
		if backToMongo.PasswordEncrypted != mongoModel.PasswordEncrypted {
			t.Fatalf("expected PasswordEncrypted preserved, got %s", backToMongo.PasswordEncrypted)
		}
	})

	t.Run("zero UUID values", func(t *testing.T) {
		mongoModel := &ConnectionMongoDBModel{
			ID: uuid.Nil,

			ConfigName:           "nil-uuid-connection",
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "encrypted",
			EncryptionKeyVersion: "v1",
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		conn, err := mongoModel.ToEntity()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if conn.ID != uuid.Nil {
			t.Fatalf("expected ID to be uuid.Nil, got %s", conn.ID)
		}
	})

	t.Run("zero port value", func(t *testing.T) {
		mongoModel := &ConnectionMongoDBModel{
			ID: uuid.New(),

			ConfigName:           "zero-port-connection",
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 0,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "encrypted",
			EncryptionKeyVersion: "v1",
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		conn, err := mongoModel.ToEntity()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if conn.Port != 0 {
			t.Fatalf("expected Port 0, got %d", conn.Port)
		}
	})

	t.Run("empty string fields", func(t *testing.T) {
		mongoModel := &ConnectionMongoDBModel{
			ID: uuid.New(),

			ConfigName:           "",
			Type:                 "POSTGRESQL",
			Host:                 "",
			Port:                 5432,
			DatabaseName:         "",
			Username:             "",
			PasswordEncrypted:    "",
			EncryptionKeyVersion: "",
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		conn, err := mongoModel.ToEntity()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if conn.ConfigName != "" {
			t.Fatalf("expected ConfigName '', got %s", conn.ConfigName)
		}
		if conn.Host != "" {
			t.Fatalf("expected Host '', got %s", conn.Host)
		}
		if conn.DatabaseName != "" {
			t.Fatalf("expected DatabaseName '', got %s", conn.DatabaseName)
		}
		if conn.Username != "" {
			t.Fatalf("expected Username '', got %s", conn.Username)
		}
	})

	t.Run("very long string values", func(t *testing.T) {
		longString := make([]byte, 10000)
		for i := range longString {
			longString[i] = 'a'
		}

		mongoModel := &ConnectionMongoDBModel{
			ID: uuid.New(),

			ConfigName:           string(longString),
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    string(longString),
			EncryptionKeyVersion: "v1",
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		conn, err := mongoModel.ToEntity()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(conn.ConfigName) != 10000 {
			t.Fatalf("expected ConfigName length 10000, got %d", len(conn.ConfigName))
		}
		if len(conn.PasswordEncrypted) != 10000 {
			t.Fatalf("expected PasswordEncrypted length 10000, got %d", len(conn.PasswordEncrypted))
		}
	})

	t.Run("negative port value", func(t *testing.T) {
		mongoModel := &ConnectionMongoDBModel{
			ID: uuid.New(),

			ConfigName:           "negative-port-connection",
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 -1,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "encrypted",
			EncryptionKeyVersion: "v1",
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		conn, err := mongoModel.ToEntity()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if conn.Port != -1 {
			t.Fatalf("expected Port -1, got %d", conn.Port)
		}
	})

	t.Run("max port value", func(t *testing.T) {
		mongoModel := &ConnectionMongoDBModel{
			ID: uuid.New(),

			ConfigName:           "max-port-connection",
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 65535,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "encrypted",
			EncryptionKeyVersion: "v1",
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		conn, err := mongoModel.ToEntity()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if conn.Port != 65535 {
			t.Fatalf("expected Port 65535, got %d", conn.Port)
		}
	})

	t.Run("zero time values", func(t *testing.T) {
		zeroTime := time.Time{}
		mongoModel := &ConnectionMongoDBModel{
			ID: uuid.New(),

			ConfigName:           "zero-time-connection",
			Type:                 "POSTGRESQL",
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "encrypted",
			EncryptionKeyVersion: "v1",
			CreatedAt:            zeroTime,
			UpdatedAt:            zeroTime,
		}

		conn, err := mongoModel.ToEntity()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !conn.CreatedAt.IsZero() {
			t.Fatalf("expected CreatedAt to be zero, got %v", conn.CreatedAt)
		}
		if !conn.UpdatedAt.IsZero() {
			t.Fatalf("expected UpdatedAt to be zero, got %v", conn.UpdatedAt)
		}
	})
}

// TestConnectionMongoDBModel_ToDomain_TypeCaseInsensitive tests that type conversion handles case variations.
func TestConnectionMongoDBModel_ToDomain_TypeCaseInsensitive(t *testing.T) {
	now := time.Now().UTC()

	// Note: The NewTypeFromString function converts to uppercase before validation,
	// so lowercase types should work
	tests := []struct {
		inputType   string
		expectValid bool
	}{
		{"POSTGRESQL", true},
		{"postgresql", true},
		{"PostgreSQL", true},
		{"  POSTGRESQL  ", true},
		{"MYSQL", true},
		{"mysql", true},
		{"MONGODB", true},
		{"mongodb", true},
		{"ORACLE", true},
		{"oracle", true},
		{"SQL_SERVER", true},
		{"sql_server", true},
		{"INVALID", false},
		{"", false},
		{"   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.inputType, func(t *testing.T) {
			mongoModel := &ConnectionMongoDBModel{
				ID: uuid.New(),

				ConfigName:           "test-connection",
				Type:                 tt.inputType,
				Host:                 "localhost",
				Port:                 5432,
				DatabaseName:         "testdb",
				Username:             "testuser",
				PasswordEncrypted:    "encrypted",
				EncryptionKeyVersion: "v1",
				CreatedAt:            now,
				UpdatedAt:            now,
			}

			conn, err := mongoModel.ToEntity()

			if tt.expectValid {
				if err != nil {
					t.Fatalf("expected valid, got error: %v", err)
				}
				if conn == nil {
					t.Fatal("expected non-nil connection")
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			}
		})
	}
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
