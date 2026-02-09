package command

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newValidConnectionInput creates a valid ConnectionInput for testing.
func newValidConnectionInput() model.ConnectionInput {
	return model.ConnectionInput{
		ConfigName:   "test-connection",
		Type:         "POSTGRESQL",
		Host:         "localhost",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpassword",
	}
}

// TestCreateConnection_Execute_Success tests successful connection creation.
func TestCreateConnection_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().
		Encrypt(gomock.Any(), gomock.Any()).
		Return("encrypted-password", "v1", nil)

	svc := NewCreateConnection(mockConnRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	input := newValidConnectionInput()

	// Mock: no existing connection found
	mockConnRepo.EXPECT().
		FindByOrganizationAndName(gomock.Any(), orgID, input.ConfigName).
		Return(nil, nil)

	// Mock: create returns the connection
	mockConnRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
			return conn, nil
		})

	result, err := svc.Execute(ctx, orgID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.ConfigName != input.ConfigName {
		t.Fatalf("expected ConfigName %s, got %s", input.ConfigName, result.ConfigName)
	}

	if result.OrganizationID != orgID {
		t.Fatalf("expected OrganizationID %s, got %s", orgID, result.OrganizationID)
	}

	if result.Type != model.TypePostgreSQL {
		t.Fatalf("expected Type %s, got %s", model.TypePostgreSQL, result.Type)
	}

	if result.Host != input.Host {
		t.Fatalf("expected Host %s, got %s", input.Host, result.Host)
	}

	if result.Port != input.Port {
		t.Fatalf("expected Port %d, got %d", input.Port, result.Port)
	}
}

// TestCreateConnection_Execute_ConflictError tests that duplicate connection name returns conflict error.
func TestCreateConnection_Execute_ConflictError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().
		Encrypt(gomock.Any(), gomock.Any()).
		Return("encrypted-password", "v1", nil)

	svc := NewCreateConnection(mockConnRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	input := newValidConnectionInput()

	existingConnection := &model.Connection{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ConfigName:     input.ConfigName,
	}

	// Mock: existing connection found
	mockConnRepo.EXPECT().
		FindByOrganizationAndName(gomock.Any(), orgID, input.ConfigName).
		Return(existingConnection, nil)

	result, err := svc.Execute(ctx, orgID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for duplicate connection, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusConflict, respErr.StatusCode)
}

// TestCreateConnection_Execute_FindByOrganizationAndNameError tests repository error during lookup.
func TestCreateConnection_Execute_FindByOrganizationAndNameError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().
		Encrypt(gomock.Any(), gomock.Any()).
		Return("encrypted-password", "v1", nil)

	svc := NewCreateConnection(mockConnRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	input := newValidConnectionInput()

	dbError := errors.New("database connection failed")

	// Mock: database error during lookup
	mockConnRepo.EXPECT().
		FindByOrganizationAndName(gomock.Any(), orgID, input.ConfigName).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to wrap dbError, got %v", err)
	}
}

// TestCreateConnection_Execute_CreateError tests repository error during creation.
func TestCreateConnection_Execute_CreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().
		Encrypt(gomock.Any(), gomock.Any()).
		Return("encrypted-password", "v1", nil)

	svc := NewCreateConnection(mockConnRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	input := newValidConnectionInput()

	dbError := errors.New("failed to insert into database")

	// Mock: no existing connection found
	mockConnRepo.EXPECT().
		FindByOrganizationAndName(gomock.Any(), orgID, input.ConfigName).
		Return(nil, nil)

	// Mock: create returns error
	mockConnRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to wrap dbError, got %v", err)
	}
}

// TestCreateConnection_Execute_EncryptionError tests encryption failure.
func TestCreateConnection_Execute_EncryptionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	encryptionError := errors.New("encryption key invalid")
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().
		Encrypt(gomock.Any(), gomock.Any()).
		Return("", "", encryptionError)

	svc := NewCreateConnection(mockConnRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	input := newValidConnectionInput()

	result, err := svc.Execute(ctx, orgID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for encryption failure, got nil")
	}

	var internalErr pkg.InternalServerError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected InternalServerError, got %T: %v", err, err)
	}
}

// TestCreateConnection_Execute_ValidationErrors tests validation failures using table-driven tests.
func TestCreateConnection_Execute_ValidationErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		input        model.ConnectionInput
		wantErrField string
	}{
		{
			name: "empty config name",
			input: model.ConnectionInput{
				ConfigName:   "",
				Type:         "POSTGRESQL",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpassword",
			},
			wantErrField: "config_name",
		},
		{
			name: "config name too short",
			input: model.ConnectionInput{
				ConfigName:   "ab",
				Type:         "POSTGRESQL",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpassword",
			},
			wantErrField: "config_name",
		},
		{
			name: "invalid config name with special characters",
			input: model.ConnectionInput{
				ConfigName:   "test@connection!",
				Type:         "POSTGRESQL",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpassword",
			},
			wantErrField: "config_name",
		},
		{
			name: "invalid database type",
			input: model.ConnectionInput{
				ConfigName:   "test-connection",
				Type:         "INVALID_TYPE",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpassword",
			},
			wantErrField: "type",
		},
		{
			name: "empty host",
			input: model.ConnectionInput{
				ConfigName:   "test-connection",
				Type:         "POSTGRESQL",
				Host:         "",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpassword",
			},
			wantErrField: "host",
		},
		{
			name: "invalid port (zero)",
			input: model.ConnectionInput{
				ConfigName:   "test-connection",
				Type:         "POSTGRESQL",
				Host:         "localhost",
				Port:         0,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpassword",
			},
			wantErrField: "port",
		},
		{
			name: "invalid port (negative)",
			input: model.ConnectionInput{
				ConfigName:   "test-connection",
				Type:         "POSTGRESQL",
				Host:         "localhost",
				Port:         -1,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpassword",
			},
			wantErrField: "port",
		},
		{
			name: "empty database name",
			input: model.ConnectionInput{
				ConfigName:   "test-connection",
				Type:         "POSTGRESQL",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "",
				Username:     "testuser",
				Password:     "testpassword",
			},
			wantErrField: "database_name",
		},
		{
			name: "empty username",
			input: model.ConnectionInput{
				ConfigName:   "test-connection",
				Type:         "POSTGRESQL",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "",
				Password:     "testpassword",
			},
			wantErrField: "username",
		},
		{
			name: "empty password",
			input: model.ConnectionInput{
				ConfigName:   "test-connection",
				Type:         "POSTGRESQL",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "",
			},
			wantErrField: "password_encrypted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockCrypto := crypto.NewMockCryptor(ctrl)
			mockCrypto.EXPECT().
				Encrypt(gomock.Any(), gomock.Any()).
				Return("encrypted-password", "v1", nil).
				AnyTimes()

			svc := NewCreateConnection(mockConnRepo, mockCrypto)

			ctx := testContext()
			orgID := uuid.New()

			result, err := svc.Execute(ctx, orgID, tt.input)

			if result != nil {
				t.Fatalf("expected nil result for invalid request, got %+v", result)
			}

			if err == nil {
				t.Fatal("expected error for invalid request, got nil")
			}

			var knownFieldsErr pkg.ValidationKnownFieldsError
			if errors.As(err, &knownFieldsErr) {
				if _, exists := knownFieldsErr.Fields[tt.wantErrField]; !exists {
					t.Fatalf("expected field %s in error fields, got %v", tt.wantErrField, knownFieldsErr.Fields)
				}
				return
			}

			var internalErr pkg.InternalServerError
			if errors.As(err, &internalErr) {
				// Some validation errors (like invalid type) come as InternalServerError
				return
			}

			t.Fatalf("expected ValidationKnownFieldsError or InternalServerError, got %T: %v", err, err)
		})
	}
}

// TestCreateConnection_Execute_WithSSL tests connection creation with SSL configuration.
func TestCreateConnection_Execute_WithSSL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().
		Encrypt(gomock.Any(), gomock.Any()).
		Return("encrypted-password", "v1", nil)

	svc := NewCreateConnection(mockConnRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	certValue := "-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----"
	keyValue := "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----"

	input := model.ConnectionInput{
		ConfigName:   "ssl-connection",
		Type:         "POSTGRESQL",
		Host:         "secure.example.com",
		Port:         5432,
		DatabaseName: "securedb",
		Username:     "secureuser",
		Password:     "securepassword",
		SSL: &model.SSLInput{
			Mode: "require",
			CA:   "-----BEGIN CERTIFICATE-----\ntest-ca\n-----END CERTIFICATE-----",
			Cert: &certValue,
			Key:  &keyValue,
		},
	}

	// Mock: no existing connection found
	mockConnRepo.EXPECT().
		FindByOrganizationAndName(gomock.Any(), orgID, input.ConfigName).
		Return(nil, nil)

	// Mock: create returns the connection
	mockConnRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
			// Verify SSL was set correctly
			if conn.SSL == nil {
				t.Error("expected SSL to be set")
			} else {
				if conn.SSL.Mode != "require" {
					t.Errorf("expected SSL mode 'require', got %s", conn.SSL.Mode)
				}
			}
			return conn, nil
		})

	result, err := svc.Execute(ctx, orgID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.SSL == nil {
		t.Fatal("expected SSL configuration to be present")
	}

	if result.SSL.Mode != "require" {
		t.Fatalf("expected SSL mode 'require', got %s", result.SSL.Mode)
	}
}

// TestCreateConnection_Execute_AllDatabaseTypes tests connection creation with all supported database types.
func TestCreateConnection_Execute_AllDatabaseTypes(t *testing.T) {
	tests := []struct {
		name         string
		dbType       string
		expectedType model.DBType
	}{
		{
			name:         "PostgreSQL",
			dbType:       "POSTGRESQL",
			expectedType: model.TypePostgreSQL,
		},
		{
			name:         "MySQL",
			dbType:       "MYSQL",
			expectedType: model.TypeMySQL,
		},
		{
			name:         "MongoDB",
			dbType:       "MONGODB",
			expectedType: model.TypeMongoDB,
		},
		{
			name:         "Oracle",
			dbType:       "ORACLE",
			expectedType: model.TypeOracle,
		},
		{
			name:         "SQL Server",
			dbType:       "SQL_SERVER",
			expectedType: model.TypeSQLServer,
		},
		{
			name:         "lowercase postgresql",
			dbType:       "postgresql",
			expectedType: model.TypePostgreSQL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockCrypto := crypto.NewMockCryptor(ctrl)
			mockCrypto.EXPECT().
				Encrypt(gomock.Any(), gomock.Any()).
				Return("encrypted-password", "v1", nil)

			svc := NewCreateConnection(mockConnRepo, mockCrypto)

			ctx := testContext()
			orgID := uuid.New()

			input := model.ConnectionInput{
				ConfigName:   "test-" + tt.dbType,
				Type:         tt.dbType,
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpassword",
			}

			// Mock: no existing connection found
			mockConnRepo.EXPECT().
				FindByOrganizationAndName(gomock.Any(), orgID, input.ConfigName).
				Return(nil, nil)

			// Mock: create returns the connection
			mockConnRepo.EXPECT().
				Create(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
					return conn, nil
				})

			result, err := svc.Execute(ctx, orgID, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Type != tt.expectedType {
				t.Fatalf("expected Type %s, got %s", tt.expectedType, result.Type)
			}
		})
	}
}

// TestNewCreateConnection verifies the constructor.
func TestNewCreateConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)

	svc := NewCreateConnection(mockConnRepo, mockCrypto)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.connRepo == nil {
		t.Fatal("expected connRepo to be set")
	}

	if svc.cryptor == nil {
		t.Fatal("expected cryptor to be set")
	}
}

// TestCreateConnection_Execute_ConfigNameEdgeCases tests edge cases for config name validation.
func TestCreateConnection_Execute_ConfigNameEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		configName string
		shouldPass bool
	}{
		{
			name:       "valid with underscore",
			configName: "test_connection",
			shouldPass: true,
		},
		{
			name:       "valid with hyphen",
			configName: "test-connection",
			shouldPass: true,
		},
		{
			name:       "valid with numbers",
			configName: "test123connection",
			shouldPass: true,
		},
		{
			name:       "valid mixed",
			configName: "Test_Connection-123",
			shouldPass: true,
		},
		{
			name:       "exactly 3 characters",
			configName: "abc",
			shouldPass: true,
		},
		{
			name:       "whitespace only",
			configName: "   ",
			shouldPass: false,
		},
		{
			name:       "with spaces",
			configName: "test connection",
			shouldPass: false,
		},
		{
			name:       "with dots",
			configName: "test.connection",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockCrypto := crypto.NewMockCryptor(ctrl)

			// Setup Encrypt expectation - encryption is always called before validation
			mockCrypto.EXPECT().
				Encrypt(gomock.Any(), gomock.Any()).
				Return("encrypted-password", "v1", nil).
				AnyTimes()

			svc := NewCreateConnection(mockConnRepo, mockCrypto)

			ctx := testContext()
			orgID := uuid.New()

			input := model.ConnectionInput{
				ConfigName:   tt.configName,
				Type:         "POSTGRESQL",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpassword",
			}

			if tt.shouldPass {
				// Mock: no existing connection found
				mockConnRepo.EXPECT().
					FindByOrganizationAndName(gomock.Any(), orgID, gomock.Any()).
					Return(nil, nil)

				// Mock: create returns the connection
				mockConnRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
						return conn, nil
					})
			}

			result, err := svc.Execute(ctx, orgID, input)

			if tt.shouldPass {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				if result == nil {
					t.Fatal("expected non-nil result")
				}
			} else {
				if err == nil {
					t.Fatal("expected error for invalid config name, got nil")
				}
				if result != nil {
					t.Fatalf("expected nil result, got %+v", result)
				}
			}
		})
	}
}
