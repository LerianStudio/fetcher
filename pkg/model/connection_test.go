package model

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/crypto"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

// TestDBType_IsValid tests the DBType.IsValid method.
func TestDBType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		dbType   DBType
		expected bool
	}{
		{
			name:     "valid POSTGRESQL",
			dbType:   TypePostgreSQL,
			expected: true,
		},
		{
			name:     "valid MYSQL",
			dbType:   TypeMySQL,
			expected: true,
		},
		{
			name:     "valid MONGODB",
			dbType:   TypeMongoDB,
			expected: true,
		},
		{
			name:     "valid ORACLE",
			dbType:   TypeOracle,
			expected: true,
		},
		{
			name:     "valid SQL_SERVER",
			dbType:   TypeSQLServer,
			expected: true,
		},
		{
			name:     "invalid empty type",
			dbType:   DBType(""),
			expected: false,
		},
		{
			name:     "invalid unknown type",
			dbType:   DBType("SQLITE"),
			expected: false,
		},
		{
			name:     "invalid lowercase postgresql",
			dbType:   DBType("postgresql"),
			expected: false,
		},
		{
			name:     "invalid random string",
			dbType:   DBType("INVALID_TYPE"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dbType.IsValid()
			if result != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestNewTypeFromString tests the NewTypeFromString function.
func TestNewTypeFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    DBType
		expectError bool
	}{
		{
			name:        "valid POSTGRESQL uppercase",
			input:       "POSTGRESQL",
			expected:    TypePostgreSQL,
			expectError: false,
		},
		{
			name:        "valid postgresql lowercase",
			input:       "postgresql",
			expected:    TypePostgreSQL,
			expectError: false,
		},
		{
			name:        "valid MYSQL",
			input:       "MYSQL",
			expected:    TypeMySQL,
			expectError: false,
		},
		{
			name:        "valid mysql lowercase",
			input:       "mysql",
			expected:    TypeMySQL,
			expectError: false,
		},
		{
			name:        "valid MONGODB",
			input:       "MONGODB",
			expected:    TypeMongoDB,
			expectError: false,
		},
		{
			name:        "valid ORACLE",
			input:       "ORACLE",
			expected:    TypeOracle,
			expectError: false,
		},
		{
			name:        "valid SQL_SERVER",
			input:       "SQL_SERVER",
			expected:    TypeSQLServer,
			expectError: false,
		},
		{
			name:        "valid with spaces",
			input:       "  POSTGRESQL  ",
			expected:    TypePostgreSQL,
			expectError: false,
		},
		{
			name:        "valid mixed case",
			input:       "PostgreSQL",
			expected:    TypePostgreSQL,
			expectError: false,
		},
		{
			name:        "invalid empty string",
			input:       "",
			expected:    DBType(""),
			expectError: true,
		},
		{
			name:        "invalid type",
			input:       "SQLITE",
			expected:    DBType(""),
			expectError: true,
		},
		{
			name:        "invalid whitespace only",
			input:       "   ",
			expected:    DBType(""),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewTypeFromString(tt.input)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestNewConnection tests the NewConnection constructor function.
func TestNewConnection(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().
		Encrypt(gomock.Any(), gomock.Any()).
		Return("encrypted-password", "v1", nil).
		AnyTimes()

	orgID := uuid.New()

	tests := []struct {
		name        string
		configName  string
		typ         string
		host        string
		port        int
		dbName      string
		username    string
		password    string
		sslMode     *string
		sslCA       *string
		sslCert     *string
		sslKey      *string
		cryptor     crypto.Cryptor
		expectError bool
		checkFields func(t *testing.T, conn *Connection)
	}{
		{
			name:        "valid connection without SSL",
			configName:  "test-connection",
			typ:         "POSTGRESQL",
			host:        "localhost",
			port:        5432,
			dbName:      "testdb",
			username:    "testuser",
			password:    "testpassword",
			sslMode:     nil,
			cryptor:     mockCrypto,
			expectError: false,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.ConfigName != "test-connection" {
					t.Fatalf("expected ConfigName 'test-connection', got %s", conn.ConfigName)
				}
				if conn.Type != TypePostgreSQL {
					t.Fatalf("expected Type POSTGRESQL, got %s", conn.Type)
				}
				if conn.Host != "localhost" {
					t.Fatalf("expected Host 'localhost', got %s", conn.Host)
				}
				if conn.Port != 5432 {
					t.Fatalf("expected Port 5432, got %d", conn.Port)
				}
				if conn.SSL != nil {
					t.Fatalf("expected SSL nil, got %+v", conn.SSL)
				}
			},
		},
		{
			name:       "valid connection with SSL",
			configName: "ssl-connection",
			typ:        "POSTGRESQL",
			host:       "localhost",
			port:       5432,
			dbName:     "testdb",
			username:   "testuser",
			password:   "testpassword",
			sslMode:    strPtr("require"),
			sslCA:      strPtr("ca-cert"),
			sslCert:    strPtr("client-cert"),
			sslKey:     strPtr("client-key"),
			cryptor:    mockCrypto,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.SSL == nil {
					t.Fatal("expected SSL to be set")
				}
				if conn.SSL.Mode != "require" {
					t.Fatalf("expected SSL Mode 'require', got %s", conn.SSL.Mode)
				}
				if conn.SSL.CA != "ca-cert" {
					t.Fatalf("expected SSL CA 'ca-cert', got %s", conn.SSL.CA)
				}
				if conn.SSL.Cert != "client-cert" {
					t.Fatalf("expected SSL Cert 'client-cert', got %s", conn.SSL.Cert)
				}
				if conn.SSL.Key != "client-key" {
					t.Fatalf("expected SSL Key 'client-key', got %s", conn.SSL.Key)
				}
			},
		},
		{
			name:        "valid connection with SSL mode only",
			configName:  "ssl-mode-only",
			typ:         "POSTGRESQL",
			host:        "localhost",
			port:        5432,
			dbName:      "testdb",
			username:    "testuser",
			password:    "testpassword",
			sslMode:     strPtr("require"),
			sslCA:       nil,
			sslCert:     nil,
			sslKey:      nil,
			cryptor:     mockCrypto,
			expectError: true, // SSL CA is required when SSL mode is set
		},
		{
			name:        "invalid database type",
			configName:  "test-connection",
			typ:         "INVALID_TYPE",
			host:        "localhost",
			port:        5432,
			dbName:      "testdb",
			username:    "testuser",
			password:    "testpassword",
			cryptor:     mockCrypto,
			expectError: true,
		},
		{
			name:        "invalid empty config name",
			configName:  "",
			typ:         "POSTGRESQL",
			host:        "localhost",
			port:        5432,
			dbName:      "testdb",
			username:    "testuser",
			password:    "testpassword",
			cryptor:     mockCrypto,
			expectError: true,
		},
		{
			name:        "invalid empty host",
			configName:  "test-connection",
			typ:         "POSTGRESQL",
			host:        "",
			port:        5432,
			dbName:      "testdb",
			username:    "testuser",
			password:    "testpassword",
			cryptor:     mockCrypto,
			expectError: true,
		},
		{
			name:        "invalid zero port",
			configName:  "test-connection",
			typ:         "POSTGRESQL",
			host:        "localhost",
			port:        0,
			dbName:      "testdb",
			username:    "testuser",
			password:    "testpassword",
			cryptor:     mockCrypto,
			expectError: true,
		},
		{
			name:        "invalid negative port",
			configName:  "test-connection",
			typ:         "POSTGRESQL",
			host:        "localhost",
			port:        -1,
			dbName:      "testdb",
			username:    "testuser",
			password:    "testpassword",
			cryptor:     mockCrypto,
			expectError: true,
		},
		{
			name:        "invalid empty database name",
			configName:  "test-connection",
			typ:         "POSTGRESQL",
			host:        "localhost",
			port:        5432,
			dbName:      "",
			username:    "testuser",
			password:    "testpassword",
			cryptor:     mockCrypto,
			expectError: true,
		},
		{
			name:        "invalid empty username",
			configName:  "test-connection",
			typ:         "POSTGRESQL",
			host:        "localhost",
			port:        5432,
			dbName:      "testdb",
			username:    "",
			password:    "testpassword",
			cryptor:     mockCrypto,
			expectError: true,
		},
		{
			name:        "invalid empty password",
			configName:  "test-connection",
			typ:         "POSTGRESQL",
			host:        "localhost",
			port:        5432,
			dbName:      "testdb",
			username:    "testuser",
			password:    "",
			cryptor:     mockCrypto,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup cryptor for encryption error test
			var testCryptor crypto.Cryptor
			if tt.name == "encryption error" {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				mockCrypto := crypto.NewMockCryptor(ctrl)
				mockCrypto.EXPECT().
					Encrypt(gomock.Any(), gomock.Any()).
					Return("", "", errors.New("encryption failed"))
				testCryptor = mockCrypto
			} else {
				testCryptor = tt.cryptor
			}

			conn, err := NewConnection(
				ctx,
				testCryptor,
				orgID,
				tt.configName,
				tt.typ,
				tt.host,
				tt.port,
				tt.dbName,
				tt.username,
				tt.password,
				&map[string]any{},
				tt.sslMode,
				tt.sslCA,
				tt.sslCert,
				tt.sslKey,
			)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if conn == nil {
				t.Fatal("expected non-nil connection")
			}

			// Verify common fields
			if conn.ID == uuid.Nil {
				t.Fatal("expected non-nil UUID for ID")
			}
			if conn.OrganizationID != orgID {
				t.Fatalf("expected OrganizationID %s, got %s", orgID, conn.OrganizationID)
			}
			if conn.CreatedAt.IsZero() {
				t.Fatal("expected CreatedAt to be set")
			}
			if conn.UpdatedAt.IsZero() {
				t.Fatal("expected UpdatedAt to be set")
			}

			if tt.checkFields != nil {
				tt.checkFields(t, conn)
			}
		})
	}
}

// TestConnection_IsValid tests the Connection.IsValid method.
func TestConnection_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		connection  Connection
		expectError bool
		errorField  string
	}{
		{
			name: "valid connection without SSL",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: false,
		},
		{
			name: "valid connection with SSL",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
				SSL: &SSLConfig{
					Mode: "require",
					CA:   "ca-cert",
				},
			},
			expectError: false,
		},
		{
			name: "missing organization ID",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.Nil,
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "organization_id",
		},
		{
			name: "missing ID",
			connection: Connection{
				ID:                uuid.Nil,
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "id",
		},
		{
			name: "empty config name",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "config_name",
		},
		{
			name: "config name too short",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "ab",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "config_name",
		},
		{
			name: "config name with invalid characters",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test@connection!",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "config_name",
		},
		{
			name: "invalid database type",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              DBType("INVALID"),
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "type",
		},
		{
			name: "empty host",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "host",
		},
		{
			name: "zero port",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              0,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "port",
		},
		{
			name: "negative port",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              -1,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "port",
		},
		{
			name: "empty database name",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "database_name",
		},
		{
			name: "empty username",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: true,
			errorField:  "username",
		},
		{
			name: "empty password encrypted",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "",
			},
			expectError: true,
			errorField:  "password_encrypted",
		},
		{
			name: "SSL without mode",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
				SSL: &SSLConfig{
					Mode: "",
					CA:   "ca-cert",
				},
			},
			expectError: true,
			errorField:  "ssl.mode",
		},
		{
			name: "SSL without CA",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
				SSL: &SSLConfig{
					Mode: "require",
					CA:   "",
				},
			},
			expectError: true,
			errorField:  "ssl.ca",
		},
		{
			name: "whitespace in config name gets trimmed",
			connection: Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "   test-connection   ",
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.connection.IsValid()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				var knownFieldsErr pkg.ValidationKnownFieldsError
				if errors.As(err, &knownFieldsErr) {
					if _, exists := knownFieldsErr.Fields[tt.errorField]; !exists {
						t.Fatalf("expected error field %s, got fields %v", tt.errorField, knownFieldsErr.Fields)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestConnection_ApplyPatch tests the Connection.ApplyPatch method.
func TestConnection_ApplyPatch(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().
		Encrypt(gomock.Any(), gomock.Any()).
		Return("encrypted-newpassword", "v1", nil).
		AnyTimes()

	baseConnection := func() *Connection {
		return &Connection{
			ID:                   uuid.New(),
			OrganizationID:       uuid.New(),
			ConfigName:           "original-connection",
			Type:                 TypePostgreSQL,
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "originaldb",
			Username:             "originaluser",
			PasswordEncrypted:    "encrypted-original",
			EncryptionKeyVersion: "v1",
			CreatedAt:            time.Now().UTC().Add(-1 * time.Hour),
			UpdatedAt:            time.Now().UTC().Add(-1 * time.Hour),
		}
	}

	tests := []struct {
		name        string
		conn        *Connection
		configName  *string
		typ         *string
		host        *string
		port        *int
		dbName      *string
		username    *string
		password    *string
		sslMode     *string
		sslCA       *string
		sslCert     *string
		sslKey      *string
		cryptor     crypto.Cryptor
		expectError bool
		checkFields func(t *testing.T, conn *Connection)
	}{
		{
			name:       "patch config name",
			conn:       baseConnection(),
			configName: strPtr("updated-connection"),
			cryptor:    mockCrypto,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.ConfigName != "updated-connection" {
					t.Fatalf("expected ConfigName 'updated-connection', got %s", conn.ConfigName)
				}
			},
		},
		{
			name:    "patch type",
			conn:    baseConnection(),
			typ:     strPtr("MYSQL"),
			cryptor: mockCrypto,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.Type != TypeMySQL {
					t.Fatalf("expected Type MYSQL, got %s", conn.Type)
				}
			},
		},
		{
			name:    "patch host",
			conn:    baseConnection(),
			host:    strPtr("newhost.example.com"),
			cryptor: mockCrypto,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.Host != "newhost.example.com" {
					t.Fatalf("expected Host 'newhost.example.com', got %s", conn.Host)
				}
			},
		},
		{
			name:    "patch port",
			conn:    baseConnection(),
			port:    intPtr(3306),
			cryptor: mockCrypto,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.Port != 3306 {
					t.Fatalf("expected Port 3306, got %d", conn.Port)
				}
			},
		},
		{
			name:    "patch database name",
			conn:    baseConnection(),
			dbName:  strPtr("newdb"),
			cryptor: mockCrypto,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.DatabaseName != "newdb" {
					t.Fatalf("expected DatabaseName 'newdb', got %s", conn.DatabaseName)
				}
			},
		},
		{
			name:     "patch username",
			conn:     baseConnection(),
			username: strPtr("newuser"),
			cryptor:  mockCrypto,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.Username != "newuser" {
					t.Fatalf("expected Username 'newuser', got %s", conn.Username)
				}
			},
		},
		{
			name:     "patch password",
			conn:     baseConnection(),
			password: strPtr("newpassword"),
			cryptor:  mockCrypto,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.PasswordEncrypted != "encrypted-newpassword" {
					t.Fatalf("expected PasswordEncrypted 'encrypted-newpassword', got %s", conn.PasswordEncrypted)
				}
			},
		},
		{
			name:    "patch SSL configuration",
			conn:    baseConnection(),
			sslMode: strPtr("require"),
			sslCA:   strPtr("new-ca"),
			sslCert: strPtr("new-cert"),
			sslKey:  strPtr("new-key"),
			cryptor: mockCrypto,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.SSL == nil {
					t.Fatal("expected SSL to be set")
				}
				if conn.SSL.Mode != "require" {
					t.Fatalf("expected SSL Mode 'require', got %s", conn.SSL.Mode)
				}
				if conn.SSL.CA != "new-ca" {
					t.Fatalf("expected SSL CA 'new-ca', got %s", conn.SSL.CA)
				}
			},
		},
		{
			name:        "patch invalid type",
			conn:        baseConnection(),
			typ:         strPtr("INVALID_TYPE"),
			cryptor:     mockCrypto,
			expectError: true,
		},
		{
			name:        "patch password without cryptor",
			conn:        baseConnection(),
			password:    strPtr("newpassword"),
			cryptor:     nil,
			expectError: true,
		},
		{
			name:        "patch config name to invalid short value",
			conn:        baseConnection(),
			configName:  strPtr("ab"),
			cryptor:     mockCrypto,
			expectError: true,
		},
		{
			name:       "updatedAt is modified",
			conn:       baseConnection(),
			configName: strPtr("new-name"),
			cryptor:    mockCrypto,
			checkFields: func(t *testing.T, conn *Connection) {
				if conn.UpdatedAt.Before(conn.CreatedAt) {
					t.Fatalf("expected UpdatedAt to be after CreatedAt")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup cryptor for encryption error test
			testCryptor := tt.cryptor
			if tt.name == "patch password with encryption error" {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				mockCrypto := crypto.NewMockCryptor(ctrl)
				mockCrypto.EXPECT().
					Encrypt(gomock.Any(), gomock.Any()).
					Return("", "", errors.New("encryption failed"))
				testCryptor = mockCrypto
			}

			err := tt.conn.ApplyPatch(
				ctx,
				testCryptor,
				tt.configName,
				tt.typ,
				tt.host,
				tt.port,
				tt.dbName,
				tt.username,
				tt.password,
				&map[string]any{},
				tt.sslMode,
				tt.sslCA,
				tt.sslCert,
				tt.sslKey,
			)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkFields != nil {
				tt.checkFields(t, tt.conn)
			}
		})
	}
}

// TestConnection_ApplyPatch_Metadata tests metadata patching specifically.
func TestConnection_ApplyPatch_Metadata(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrypto := crypto.NewMockCryptor(ctrl)

	t.Run("patch metadata updates connection metadata", func(t *testing.T) {
		conn := &Connection{
			ID:                   uuid.New(),
			OrganizationID:       uuid.New(),
			ConfigName:           "oracle-connection",
			Type:                 TypeOracle,
			Host:                 "localhost",
			Port:                 1521,
			DatabaseName:         "ORCL",
			Username:             "testuser",
			PasswordEncrypted:    "encrypted-password",
			EncryptionKeyVersion: "v1",
			CreatedAt:            time.Now().UTC().Add(-1 * time.Hour),
			UpdatedAt:            time.Now().UTC().Add(-1 * time.Hour),
			Metadata:             nil,
		}

		newMetadata := &map[string]any{
			"service_name": "ORCL_SERVICE",
			"sid":          "ORCL",
		}

		err := conn.ApplyPatch(
			ctx,
			mockCrypto,
			nil, // configName
			nil, // typ
			nil, // host
			nil, // port
			nil, // dbName
			nil, // username
			nil, // password
			newMetadata,
			nil, // sslMode
			nil, // sslCA
			nil, // sslCert
			nil, // sslKey
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if conn.Metadata == nil {
			t.Fatal("expected Metadata to be set")
		}

		if (*conn.Metadata)["service_name"] != "ORCL_SERVICE" {
			t.Fatalf("expected service_name 'ORCL_SERVICE', got %v", (*conn.Metadata)["service_name"])
		}

		if (*conn.Metadata)["sid"] != "ORCL" {
			t.Fatalf("expected sid 'ORCL', got %v", (*conn.Metadata)["sid"])
		}
	})

	t.Run("patch nil metadata does not clear existing metadata", func(t *testing.T) {
		existingMetadata := &map[string]any{
			"service_name": "EXISTING_SERVICE",
		}

		conn := &Connection{
			ID:                   uuid.New(),
			OrganizationID:       uuid.New(),
			ConfigName:           "oracle-connection",
			Type:                 TypeOracle,
			Host:                 "localhost",
			Port:                 1521,
			DatabaseName:         "ORCL",
			Username:             "testuser",
			PasswordEncrypted:    "encrypted-password",
			EncryptionKeyVersion: "v1",
			Metadata:             existingMetadata,
			CreatedAt:            time.Now().UTC().Add(-1 * time.Hour),
			UpdatedAt:            time.Now().UTC().Add(-1 * time.Hour),
		}

		err := conn.ApplyPatch(
			ctx,
			mockCrypto,
			nil, nil, nil, nil, nil, nil, nil,
			nil, // nil metadata should NOT clear existing
			nil, nil, nil, nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if conn.Metadata == nil {
			t.Fatal("expected Metadata to remain set")
		}

		if (*conn.Metadata)["service_name"] != "EXISTING_SERVICE" {
			t.Fatalf("expected service_name 'EXISTING_SERVICE', got %v", (*conn.Metadata)["service_name"])
		}
	})
}

// TestConnection_SoftDelete tests the Connection.SoftDelete method.
func TestConnection_SoftDelete(t *testing.T) {
	t.Run("soft delete with specific timestamp", func(t *testing.T) {
		conn := &Connection{
			ID:                uuid.New(),
			OrganizationID:    uuid.New(),
			ConfigName:        "test-connection",
			Type:              TypePostgreSQL,
			Host:              "localhost",
			Port:              5432,
			DatabaseName:      "testdb",
			Username:          "testuser",
			PasswordEncrypted: "encrypted-password",
			CreatedAt:         time.Now().UTC().Add(-1 * time.Hour),
			UpdatedAt:         time.Now().UTC().Add(-1 * time.Hour),
		}

		deleteTime := time.Now().UTC()
		conn.SoftDelete(deleteTime)

		if conn.DeletedAt == nil {
			t.Fatal("expected DeletedAt to be set")
		}
		if !conn.DeletedAt.Equal(deleteTime) {
			t.Fatalf("expected DeletedAt %v, got %v", deleteTime, *conn.DeletedAt)
		}
		if !conn.UpdatedAt.Equal(deleteTime) {
			t.Fatalf("expected UpdatedAt %v, got %v", deleteTime, conn.UpdatedAt)
		}
	})

	t.Run("soft delete with zero timestamp uses current time", func(t *testing.T) {
		conn := &Connection{
			ID:                uuid.New(),
			OrganizationID:    uuid.New(),
			ConfigName:        "test-connection",
			Type:              TypePostgreSQL,
			Host:              "localhost",
			Port:              5432,
			DatabaseName:      "testdb",
			Username:          "testuser",
			PasswordEncrypted: "encrypted-password",
			CreatedAt:         time.Now().UTC().Add(-1 * time.Hour),
			UpdatedAt:         time.Now().UTC().Add(-1 * time.Hour),
		}

		beforeDelete := time.Now().UTC()
		conn.SoftDelete(time.Time{})
		afterDelete := time.Now().UTC()

		if conn.DeletedAt == nil {
			t.Fatal("expected DeletedAt to be set")
		}
		if conn.DeletedAt.Before(beforeDelete) || conn.DeletedAt.After(afterDelete) {
			t.Fatalf("DeletedAt should be between %v and %v, got %v", beforeDelete, afterDelete, *conn.DeletedAt)
		}
	})
}

// TestConnection_GetPasswordDecrypted tests the Connection.GetPasswordDecrypted method.
func TestConnection_GetPasswordDecrypted(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		expectError bool
		expected    string
		setupMock   func(ctrl *gomock.Controller) crypto.Cryptor
	}{
		{
			name:        "successful decryption",
			expectError: false,
			expected:    "decrypted-password",
			setupMock: func(ctrl *gomock.Controller) crypto.Cryptor {
				mockCrypto := crypto.NewMockCryptor(ctrl)
				mockCrypto.EXPECT().
					Decrypt(gomock.Any(), gomock.Any(), gomock.Any()).
					Return("decrypted-password", nil)
				return mockCrypto
			},
		},
		{
			name:        "nil cryptor",
			expectError: true,
			setupMock:   func(ctrl *gomock.Controller) crypto.Cryptor { return nil },
		},
		{
			name:        "decryption error",
			expectError: true,
			setupMock: func(ctrl *gomock.Controller) crypto.Cryptor {
				mockCrypto := crypto.NewMockCryptor(ctrl)
				mockCrypto.EXPECT().
					Decrypt(gomock.Any(), gomock.Any(), gomock.Any()).
					Return("", errors.New("decryption failed"))
				return mockCrypto
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cryptor := tt.setupMock(ctrl)
			conn := &Connection{
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
			}

			result, err := conn.GetPasswordDecrypted(ctx, cryptor)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestConnection_DecryptPassword tests the Connection.DecryptPassword method.
func TestConnection_DecryptPassword(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		expectError bool
		setupMock   func(ctrl *gomock.Controller) crypto.Cryptor
	}{
		{
			name:        "successful decryption",
			expectError: false,
			setupMock: func(ctrl *gomock.Controller) crypto.Cryptor {
				mockCrypto := crypto.NewMockCryptor(ctrl)
				mockCrypto.EXPECT().
					Decrypt(gomock.Any(), gomock.Any(), gomock.Any()).
					Return("decrypted-password", nil)
				return mockCrypto
			},
		},
		{
			name:        "nil cryptor",
			expectError: true,
			setupMock:   func(ctrl *gomock.Controller) crypto.Cryptor { return nil },
		},
		{
			name:        "decryption error",
			expectError: true,
			setupMock: func(ctrl *gomock.Controller) crypto.Cryptor {
				mockCrypto := crypto.NewMockCryptor(ctrl)
				mockCrypto.EXPECT().
					Decrypt(gomock.Any(), gomock.Any(), gomock.Any()).
					Return("", errors.New("decryption failed"))
				return mockCrypto
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cryptor := tt.setupMock(ctrl)
			conn := &Connection{
				PasswordEncrypted:    "encrypted-password",
				EncryptionKeyVersion: "v1",
			}

			err := conn.DecryptPassword(ctx, cryptor)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestConnection_ToMapWithMask tests the Connection.ToMapWithMask method.
func TestConnection_ToMapWithMask(t *testing.T) {
	t.Run("connection without SSL", func(t *testing.T) {
		connID := uuid.New()
		orgID := uuid.New()
		now := time.Now()

		conn := &Connection{
			ID:                   connID,
			OrganizationID:       orgID,
			ConfigName:           "test-connection",
			Type:                 TypePostgreSQL,
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "secret-encrypted-password",
			EncryptionKeyVersion: "v1",
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		result := conn.ToMapWithMask()

		if result["id"] != connID {
			t.Fatalf("expected ID %v, got %v", connID, result["id"])
		}
		if result["organization_id"] != orgID {
			t.Fatalf("expected organization_id %v, got %v", orgID, result["organization_id"])
		}
		if result["config_name"] != "test-connection" {
			t.Fatalf("expected config_name 'test-connection', got %v", result["config_name"])
		}
		if result["type"] != string(TypePostgreSQL) {
			t.Fatalf("expected type 'POSTGRESQL', got %v", result["type"])
		}
		if result["password_encrypted"] != "[REDACTED]" {
			t.Fatalf("expected password_encrypted '[REDACTED]', got %v", result["password_encrypted"])
		}
		if result["encryption_key_version"] != "[REDACTED]" {
			t.Fatalf("expected encryption_key_version '[REDACTED]', got %v", result["encryption_key_version"])
		}
		// When SSL is nil, the ToMapWithMask returns a nil map[string]any for ssl
		// which may be represented as nil or empty map depending on context
		ssl := result["ssl"]
		if ssl != nil {
			if sslMap, ok := ssl.(map[string]any); ok && len(sslMap) > 0 {
				t.Fatalf("expected ssl nil or empty map, got %v", result["ssl"])
			}
		}
	})

	t.Run("connection with SSL", func(t *testing.T) {
		conn := &Connection{
			ID:                   uuid.New(),
			OrganizationID:       uuid.New(),
			ConfigName:           "ssl-connection",
			Type:                 TypePostgreSQL,
			Host:                 "localhost",
			Port:                 5432,
			DatabaseName:         "testdb",
			Username:             "testuser",
			PasswordEncrypted:    "secret-encrypted-password",
			EncryptionKeyVersion: "v1",
			SSL: &SSLConfig{
				Mode: "require",
				CA:   "secret-ca-cert",
				Cert: "secret-client-cert",
				Key:  "secret-client-key",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		result := conn.ToMapWithMask()

		ssl, ok := result["ssl"].(map[string]any)
		if !ok {
			t.Fatalf("expected ssl to be map[string]any, got %T", result["ssl"])
		}

		if ssl["mode"] != "require" {
			t.Fatalf("expected ssl.mode 'require', got %v", ssl["mode"])
		}
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
}

// TestConnectionInput_ToMapWithMask tests the ConnectionInput.ToMapWithMask method.
func TestConnectionInput_ToMapWithMask(t *testing.T) {
	t.Run("input without SSL", func(t *testing.T) {
		input := &ConnectionInput{
			ConfigName:   "test-connection",
			Type:         "POSTGRESQL",
			Host:         "localhost",
			Port:         5432,
			DatabaseName: "testdb",
			Username:     "testuser",
			Password:     "secret-password",
		}

		result := input.ToMapWithMask()

		if result["config_name"] != "test-connection" {
			t.Fatalf("expected config_name 'test-connection', got %v", result["config_name"])
		}
		if result["password"] != "[REDACTED]" {
			t.Fatalf("expected password '[REDACTED]', got %v", result["password"])
		}
		// When SSL is nil, the ToMapWithMask returns a nil map[string]any for ssl
		// which may be represented as nil or empty map depending on context
		ssl := result["ssl"]
		if ssl != nil {
			if sslMap, ok := ssl.(map[string]any); ok && len(sslMap) > 0 {
				t.Fatalf("expected ssl nil or empty map, got %v", result["ssl"])
			}
		}
	})

	t.Run("input with SSL", func(t *testing.T) {
		certValue := "secret-cert"
		keyValue := "secret-key"

		input := &ConnectionInput{
			ConfigName:   "ssl-connection",
			Type:         "POSTGRESQL",
			Host:         "localhost",
			Port:         5432,
			DatabaseName: "testdb",
			Username:     "testuser",
			Password:     "secret-password",
			SSL: &SSLInput{
				Mode: "require",
				CA:   "secret-ca",
				Cert: &certValue,
				Key:  &keyValue,
			},
		}

		result := input.ToMapWithMask()

		ssl, ok := result["ssl"].(map[string]any)
		if !ok {
			t.Fatalf("expected ssl to be map[string]any, got %T", result["ssl"])
		}

		if ssl["ca"] != "[REDACTED]" {
			t.Fatalf("expected ssl.ca '[REDACTED]', got %v", ssl["ca"])
		}
	})
}

// TestNewConnectionResponseFrom tests the NewConnectionResponseFrom function.
func TestNewConnectionResponseFrom(t *testing.T) {
	t.Run("nil connection returns nil", func(t *testing.T) {
		result := NewConnectionResponseFrom(nil)
		if result != nil {
			t.Fatalf("expected nil, got %+v", result)
		}
	})

	t.Run("connection without SSL", func(t *testing.T) {
		connID := uuid.New()
		now := time.Now()

		conn := &Connection{
			ID:                uuid.New(),
			OrganizationID:    uuid.New(),
			ConfigName:        "test-connection",
			Type:              TypePostgreSQL,
			Host:              "localhost",
			Port:              5432,
			DatabaseName:      "testdb",
			Username:          "testuser",
			PasswordEncrypted: "encrypted-password",
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		conn.ID = connID

		result := NewConnectionResponseFrom(conn)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.ID != connID {
			t.Fatalf("expected ID %v, got %v", connID, result.ID)
		}
		if result.ConfigName != "test-connection" {
			t.Fatalf("expected ConfigName 'test-connection', got %s", result.ConfigName)
		}
		if result.Type != string(TypePostgreSQL) {
			t.Fatalf("expected Type 'POSTGRESQL', got %s", result.Type)
		}
		if result.SSL != nil {
			t.Fatalf("expected SSL nil, got %+v", result.SSL)
		}
	})

	t.Run("connection with SSL", func(t *testing.T) {
		conn := &Connection{
			ID:                uuid.New(),
			OrganizationID:    uuid.New(),
			ConfigName:        "ssl-connection",
			Type:              TypePostgreSQL,
			Host:              "localhost",
			Port:              5432,
			DatabaseName:      "testdb",
			Username:          "testuser",
			PasswordEncrypted: "encrypted-password",
			SSL: &SSLConfig{
				Mode: "require",
				CA:   "ca-cert",
				Cert: "client-cert",
				Key:  "client-key",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		result := NewConnectionResponseFrom(conn)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.SSL == nil {
			t.Fatal("expected SSL to be set")
		}
		if result.SSL.Mode != "require" {
			t.Fatalf("expected SSL Mode 'require', got %s", result.SSL.Mode)
		}
	})
}

// TestConnection_ConfigNameEdgeCases tests edge cases for config name validation.
func TestConnection_ConfigNameEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		configName  string
		expectValid bool
	}{
		{
			name:        "valid with underscore",
			configName:  "test_connection",
			expectValid: true,
		},
		{
			name:        "valid with hyphen",
			configName:  "test-connection",
			expectValid: true,
		},
		{
			name:        "valid with numbers",
			configName:  "test123connection",
			expectValid: true,
		},
		{
			name:        "valid mixed",
			configName:  "Test_Connection-123",
			expectValid: true,
		},
		{
			name:        "exactly 3 characters",
			configName:  "abc",
			expectValid: true,
		},
		{
			name:        "100 characters (max length)",
			configName:  "a" + string(make([]byte, 99)),
			expectValid: false, // Will fail because of non-alphanumeric characters in make([]byte, 99)
		},
		{
			name:        "with spaces",
			configName:  "test connection",
			expectValid: false,
		},
		{
			name:        "with dots",
			configName:  "test.connection",
			expectValid: false,
		},
		{
			name:        "with at symbol",
			configName:  "test@connection",
			expectValid: false,
		},
		{
			name:        "2 characters (too short)",
			configName:  "ab",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        tt.configName,
				Type:              TypePostgreSQL,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			}

			err := conn.IsValid()

			if tt.expectValid && err != nil {
				t.Fatalf("expected valid, got error: %v", err)
			}
			if !tt.expectValid && err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

// TestConnection_AllDatabaseTypes tests that all database types are properly validated.
func TestConnection_AllDatabaseTypes(t *testing.T) {
	dbTypes := []struct {
		name   string
		dbType DBType
	}{
		{"PostgreSQL", TypePostgreSQL},
		{"MySQL", TypeMySQL},
		{"MongoDB", TypeMongoDB},
		{"Oracle", TypeOracle},
		{"SQLServer", TypeSQLServer},
	}

	for _, tt := range dbTypes {
		t.Run(tt.name, func(t *testing.T) {
			conn := Connection{
				ID:                uuid.New(),
				OrganizationID:    uuid.New(),
				ConfigName:        "test-connection",
				Type:              tt.dbType,
				Host:              "localhost",
				Port:              5432,
				DatabaseName:      "testdb",
				Username:          "testuser",
				PasswordEncrypted: "encrypted-password",
			}

			err := conn.IsValid()
			if err != nil {
				t.Fatalf("expected valid connection for type %s, got error: %v", tt.dbType, err)
			}
		})
	}
}

// Helper functions

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
