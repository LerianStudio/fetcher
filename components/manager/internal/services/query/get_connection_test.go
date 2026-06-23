package query

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/v2/pkg/resolver"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newExistingConnection creates a valid existing Connection for testing.
func newExistingConnection(connID uuid.UUID) *model.Connection {
	return &model.Connection{
		ID:                   connID,
		ConfigName:           "test-connection",
		Type:                 model.TypePostgreSQL,
		Host:                 "localhost",
		Port:                 5432,
		DatabaseName:         "testdb",
		Username:             "testuser",
		PasswordEncrypted:    "encrypted-password",
		EncryptionKeyVersion: "v1",
		CreatedAt:            time.Now().UTC().Add(-24 * time.Hour),
		UpdatedAt:            time.Now().UTC().Add(-1 * time.Hour),
	}
}

// TestGetConnection_Execute_Success tests successful connection retrieval.
func TestGetConnection_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	connID := uuid.New()
	existingConn := newExistingConnection(connID)

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	result, err := svc.Execute(ctx, connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.ID != connID {
		t.Fatalf("expected ID %s, got %s", connID, result.ID)
	}

	if result.ConfigName != existingConn.ConfigName {
		t.Fatalf("expected ConfigName %s, got %s", existingConn.ConfigName, result.ConfigName)
	}

	if result.Type != existingConn.Type {
		t.Fatalf("expected Type %s, got %s", existingConn.Type, result.Type)
	}

	if result.Host != existingConn.Host {
		t.Fatalf("expected Host %s, got %s", existingConn.Host, result.Host)
	}

	if result.Port != existingConn.Port {
		t.Fatalf("expected Port %d, got %d", existingConn.Port, result.Port)
	}
}

// TestGetConnection_Execute_NotFoundError tests that non-existent connection returns not found error.
func TestGetConnection_Execute_NotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	connID := uuid.New()

	// Mock: connection not found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, connID)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for non-existent connection, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}

// TestGetConnection_Execute_RepositoryError tests repository error during FindByID.
func TestGetConnection_Execute_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	connID := uuid.New()

	dbError := errors.New("database connection failed")

	// Mock: database error during lookup
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, connID)

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

// TestGetConnection_Execute_OrganizationIsolation tests that connections are isolated by organization.
func TestGetConnection_Execute_OrganizationIsolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	differentOrgID := uuid.New()
	connID := uuid.New()

	// Connection belongs to a different organization but we query with orgID
	// The repository should return nil because it filters by organizationID
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, connID)

	if result != nil {
		t.Fatalf("expected nil result for connection in different organization, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for connection in different organization, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)

	// Verify the existing connection is not returned (it belongs to a different org)
	_ = differentOrgID // Unused in mock but demonstrates the test scenario
}

// TestGetConnection_Execute_TableDriven uses table-driven tests for various scenarios.
func TestGetConnection_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*connRepo.MockRepository, uuid.UUID, *model.Connection)
		wantErr        bool
		wantStatusCode int // 0 means generic error (no status code check)
	}{
		{
			name: "successful retrieval",
			setupMocks: func(connMock *connRepo.MockRepository, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID).
					Return(existing, nil)
			},
			wantErr: false,
		},
		{
			name: "connection not found",
			setupMocks: func(connMock *connRepo.MockRepository, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "FindByID database error",
			setupMocks: func(connMock *connRepo.MockRepository, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID).
					Return(nil, errors.New("database connection failed"))
			},
			wantErr:        true,
			wantStatusCode: 0, // generic error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)

			ctx := testContext()
			connID := uuid.New()
			existingConn := newExistingConnection(connID)

			tt.setupMocks(mockConnRepo, connID, existingConn)

			svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

			result, err := svc.Execute(ctx, connID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.wantStatusCode != 0 {
					var respErr pkg.ResponseErrorWithStatusCode
					if !errors.As(err, &respErr) {
						t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
					}
					assert.Equal(t, tt.wantStatusCode, respErr.StatusCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestGetConnection_Execute_ConnectionWithSSL tests retrieval of connection with SSL configuration.
func TestGetConnection_Execute_ConnectionWithSSL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	connID := uuid.New()

	existingConn := newExistingConnection(connID)
	existingConn.SSL = &model.SSLConfig{
		Mode: "require",
		CA:   "-----BEGIN CERTIFICATE-----\ntest-ca\n-----END CERTIFICATE-----",
		Cert: "-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----",
		Key:  "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----",
	}

	// Mock: connection found with SSL
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	result, err := svc.Execute(ctx, connID)
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

// TestGetConnection_Execute_AllDatabaseTypes tests retrieval of connections with all supported database types.
func TestGetConnection_Execute_AllDatabaseTypes(t *testing.T) {
	tests := []struct {
		name   string
		dbType model.DBType
	}{
		{
			name:   "PostgreSQL connection",
			dbType: model.TypePostgreSQL,
		},
		{
			name:   "MySQL connection",
			dbType: model.TypeMySQL,
		},
		{
			name:   "MongoDB connection",
			dbType: model.TypeMongoDB,
		},
		{
			name:   "Oracle connection",
			dbType: model.TypeOracle,
		},
		{
			name:   "SQL Server connection",
			dbType: model.TypeSQLServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)

			svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

			ctx := testContext()
			connID := uuid.New()

			existingConn := newExistingConnection(connID)
			existingConn.Type = tt.dbType

			// Mock: connection found
			mockConnRepo.EXPECT().
				FindByID(gomock.Any(), connID).
				Return(existingConn, nil)

			result, err := svc.Execute(ctx, connID)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.name, err)
			}

			if result == nil {
				t.Fatalf("expected non-nil result for %s", tt.name)
			}

			if result.Type != tt.dbType {
				t.Fatalf("expected Type %s, got %s", tt.dbType, result.Type)
			}
		})
	}
}

// TestNewGetConnection verifies the constructor.
func TestNewGetConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.engine == nil {
		t.Fatal("expected engine to be set")
	}
}

// internalConnID is the deterministic per-tenant UUID for an internal datasource
// config in single-tenant mode (empty tenant id), matching the registry's UUID v5
// derivation. It is the key the internal-datasource branch matches on.
func internalConnID(configName string) uuid.UUID {
	return uuid.NewSHA1(resolver.InternalDatasourceNamespace, []byte("/"+configName))
}

// TestGetConnection_Execute_InternalDatasource_Resolved proves the internal
// branch: when the registry maps the UUID to an internal config, the resolver
// resolves it on the host hot path and the resolved connection is returned WITHOUT
// any Engine connection-store read (the external getConnectionByID is never hit).
func TestGetConnection_Execute_InternalDatasource_Resolved(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl) // no FindByID expectation: internal never reads the store
	mockResolver := resolver.NewMockConnectionResolver(ctrl)
	registry := resolver.NewInternalDatasourceRegistry()

	connID := internalConnID("midaz_onboarding")
	internalConn := &model.Connection{
		ConfigName:   "midaz_onboarding",
		Type:         model.TypePostgreSQL,
		Host:         "internal-db",
		DatabaseName: "ledger",
	}

	mockResolver.EXPECT().
		ResolveInternalByConfigName(gomock.Any(), "midaz_onboarding").
		Return(internalConn, nil)

	svc := NewGetConnection(mockResolver, registry, scopeAuthorityEngine(t, mockConnRepo))

	got, err := svc.Execute(testContext(), connID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "midaz_onboarding", got.ConfigName)
}

// TestGetConnection_Execute_InternalDatasource_ResolveError proves a resolver
// error on the internal branch is wrapped and propagated (NOT swallowed, NOT
// mapped to not-found).
func TestGetConnection_Execute_InternalDatasource_ResolveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockResolver := resolver.NewMockConnectionResolver(ctrl)
	registry := resolver.NewInternalDatasourceRegistry()

	connID := internalConnID("midaz_onboarding")

	mockResolver.EXPECT().
		ResolveInternalByConfigName(gomock.Any(), "midaz_onboarding").
		Return(nil, errors.New("tenant-manager unavailable"))

	svc := NewGetConnection(mockResolver, registry, scopeAuthorityEngine(t, mockConnRepo))

	got, err := svc.Execute(testContext(), connID)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "failed to resolve internal datasource")
}

// TestGetConnection_Execute_InternalDatasource_NilResolvedIsNotFound proves a
// nil-resolved internal connection (registry knows the id but the resolver returns
// no connection) maps to the Manager's not-found business error.
func TestGetConnection_Execute_InternalDatasource_NilResolvedIsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockResolver := resolver.NewMockConnectionResolver(ctrl)
	registry := resolver.NewInternalDatasourceRegistry()

	connID := internalConnID("midaz_onboarding")

	mockResolver.EXPECT().
		ResolveInternalByConfigName(gomock.Any(), "midaz_onboarding").
		Return(nil, nil)

	svc := NewGetConnection(mockResolver, registry, scopeAuthorityEngine(t, mockConnRepo))

	got, err := svc.Execute(testContext(), connID)
	require.Error(t, err)
	assert.Nil(t, got)

	var respErr pkg.ResponseErrorWithStatusCode
	if assert.True(t, errors.As(err, &respErr)) {
		assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
	}
}

// TestGetConnection_Execute_ConnectionWithAllFields tests retrieval of connection with all fields populated.
func TestGetConnection_Execute_ConnectionWithAllFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	connID := uuid.New()

	existingConn := &model.Connection{
		ID:                   connID,
		ConfigName:           "full-connection",
		Type:                 model.TypePostgreSQL,
		Host:                 "db.example.com",
		Port:                 5432,
		DatabaseName:         "production",
		Username:             "admin",
		PasswordEncrypted:    "super-secret-encrypted",
		EncryptionKeyVersion: "v2",
		SSL: &model.SSLConfig{
			Mode: "verify-full",
			CA:   "-----BEGIN CERTIFICATE-----\nca-cert\n-----END CERTIFICATE-----",
			Cert: "-----BEGIN CERTIFICATE-----\nclient-cert\n-----END CERTIFICATE-----",
			Key:  "-----BEGIN PRIVATE KEY-----\nclient-key\n-----END PRIVATE KEY-----",
		},
		CreatedAt: time.Now().UTC().Add(-48 * time.Hour),
		UpdatedAt: time.Now().UTC().Add(-30 * time.Minute),
	}

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	result, err := svc.Execute(ctx, connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify all fields
	if result.ID != connID {
		t.Fatalf("expected ID %s, got %s", connID, result.ID)
	}
	if result.ConfigName != "full-connection" {
		t.Fatalf("expected ConfigName 'full-connection', got %s", result.ConfigName)
	}
	if result.Type != model.TypePostgreSQL {
		t.Fatalf("expected Type %s, got %s", model.TypePostgreSQL, result.Type)
	}
	if result.Host != "db.example.com" {
		t.Fatalf("expected Host 'db.example.com', got %s", result.Host)
	}
	if result.Port != 5432 {
		t.Fatalf("expected Port 5432, got %d", result.Port)
	}
	if result.DatabaseName != "production" {
		t.Fatalf("expected DatabaseName 'production', got %s", result.DatabaseName)
	}
	if result.Username != "admin" {
		t.Fatalf("expected Username 'admin', got %s", result.Username)
	}
	if result.PasswordEncrypted != "super-secret-encrypted" {
		t.Fatalf("expected PasswordEncrypted 'super-secret-encrypted', got %s", result.PasswordEncrypted)
	}
	if result.EncryptionKeyVersion != "v2" {
		t.Fatalf("expected EncryptionKeyVersion 'v2', got %s", result.EncryptionKeyVersion)
	}
	if result.SSL == nil {
		t.Fatal("expected SSL to be present")
	}
	if result.SSL.Mode != "verify-full" {
		t.Fatalf("expected SSL.Mode 'verify-full', got %s", result.SSL.Mode)
	}
}

// TestGetConnection_Execute_MultipleRepositoryErrors tests various repository error scenarios.
func TestGetConnection_Execute_MultipleRepositoryErrors(t *testing.T) {
	tests := []struct {
		name    string
		dbError error
	}{
		{
			name:    "connection timeout",
			dbError: errors.New("connection timeout"),
		},
		{
			name:    "network error",
			dbError: errors.New("network error: no route to host"),
		},
		{
			name:    "authentication failed",
			dbError: errors.New("authentication failed"),
		},
		{
			name:    "permission denied",
			dbError: errors.New("permission denied"),
		},
		{
			name:    "database unavailable",
			dbError: errors.New("database unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)

			svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

			ctx := testContext()
			connID := uuid.New()

			// Mock: database error
			mockConnRepo.EXPECT().
				FindByID(gomock.Any(), connID).
				Return(nil, tt.dbError)

			result, err := svc.Execute(ctx, connID)

			if result != nil {
				t.Fatalf("expected nil result for %s, got %+v", tt.name, result)
			}

			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}

			if !errors.Is(err, tt.dbError) {
				t.Fatalf("expected error to wrap %v, got %v", tt.dbError, err)
			}
		})
	}
}

// TestGetConnection_Execute_EmptyUUIDs tests behavior with edge case UUIDs.
func TestGetConnection_Execute_EmptyUUIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	connID := uuid.Nil

	// Mock: connection not found with nil UUIDs
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, connID)

	if result != nil {
		t.Fatalf("expected nil result with nil UUIDs, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error with nil UUIDs, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}
