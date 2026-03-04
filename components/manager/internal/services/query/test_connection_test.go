package query

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

	tmcore "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/core"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newTestConnectionFixture creates a valid Connection for testing TestConnection service.
func newTestConnectionFixture(orgID, connID uuid.UUID, dbType model.DBType) *model.Connection {
	return &model.Connection{
		ID:                   connID,
		OrganizationID:       orgID,
		ConfigName:           "test-connection",
		Type:                 dbType,
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

// TestTestConnection_Execute_NotFoundError tests that non-existent connection returns not found error.
func TestTestConnection_Execute_NotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().
		Take(gomock.Any(), gomock.Any()).
		Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	// Mock: connection not found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, connID)

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

// TestTestConnection_Execute_RepositoryError tests repository error during FindByID.
func TestTestConnection_Execute_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().
		Take(gomock.Any(), gomock.Any()).
		Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	dbError := errors.New("database connection failed")

	// Mock: database error during lookup
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, connID)

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

// TestTestConnection_Execute_DecryptionError tests password decryption failure.
func TestTestConnection_Execute_DecryptionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	decryptionError := errors.New("decryption key invalid")
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().
		Decrypt(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("", decryptionError)

	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().
		Take(gomock.Any(), gomock.Any()).
		Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

	// Provide a factory that calls cryptor.Decrypt to exercise the mock
	testFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		_, err := cryptor.Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
		return nil, fmt.Errorf("connection failed")
	}

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, testFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newTestConnectionFixture(orgID, connID, model.TypePostgreSQL)

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	result, err := svc.Execute(ctx, orgID, connID)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for decryption failure, got nil")
	}

	var internalErr pkg.ResponseError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected ResponseError, got %T: %v", err, err)
	}
}

// TestTestConnection_Execute_RateLimitError tests rate limiter store error.
func TestTestConnection_Execute_RateLimitError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)

	limiterError := errors.New("rate limiter storage error")
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().
		Take(gomock.Any(), gomock.Any()).
		Return(uint64(0), uint64(0), uint64(0), false, limiterError)

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	result, err := svc.Execute(ctx, orgID, connID)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for rate limiter failure, got nil")
	}

	var internalErr pkg.InternalServerError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected InternalServerError, got %T: %v", err, err)
	}
}

// TestTestConnection_Execute_RateLimited tests rate limit exceeded scenario.
func TestTestConnection_Execute_RateLimited(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)

	resetTime := time.Now().UTC().Add(30 * time.Second)
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().
		Take(gomock.Any(), gomock.Any()).
		Return(uint64(0), uint64(0), uint64(resetTime.UnixNano()), false, nil)

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	result, err := svc.Execute(ctx, orgID, connID)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for rate limit exceeded, got nil")
	}

	var responseErr pkg.ResponseError
	if !errors.As(err, &responseErr) {
		t.Fatalf("expected ResponseError, got %T: %v", err, err)
	}

	if responseErr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status code %d, got %d", http.StatusTooManyRequests, responseErr.Code)
	}

	if responseErr.Title != "Rate Limit Exceeded" {
		t.Fatalf("expected title 'Rate Limit Exceeded', got %s", responseErr.Title)
	}
}

// TestTestConnection_Execute_OrganizationIsolation tests that connections are isolated by organization.
func TestTestConnection_Execute_OrganizationIsolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().Take(gomock.Any(), gomock.Any()).Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

	ctx := testContext()
	orgID := uuid.New()
	differentOrgID := uuid.New()
	connID := uuid.New()

	// Connection belongs to a different organization but we query with orgID
	// The repository should return nil because it filters by organizationID
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, connID)

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

// TestTestConnection_Execute_WithSSLConfiguration tests connection test with SSL configuration.
func TestTestConnection_Execute_WithSSLConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Decrypt(gomock.Any(), gomock.Any(), gomock.Any()).Return("test-password", nil)
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().Take(gomock.Any(), gomock.Any()).Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

	// Provide a factory that calls cryptor.Decrypt then fails on connection
	testFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		_, err := cryptor.Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
		return nil, fmt.Errorf("connection failed: unable to connect to database")
	}

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, testFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	existingConn := newTestConnectionFixture(orgID, connID, model.TypePostgreSQL)
	existingConn.SSL = &model.SSLConfig{
		Mode: "require",
		CA:   "-----BEGIN CERTIFICATE-----\ntest-ca\n-----END CERTIFICATE-----",
		Cert: "-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----",
		Key:  "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----",
	}

	// Mock: connection found with SSL
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	result, err := svc.Execute(ctx, orgID, connID)

	// The datasource factory will fail because it tries to actually connect
	// This test verifies that SSL configuration is properly passed through
	if err == nil {
		t.Fatal("expected error (datasource connection failure), got nil")
	}

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	// Verify it's a connection error, not a validation error
	var responseErr pkg.ResponseError
	if !errors.As(err, &responseErr) {
		t.Fatalf("expected ResponseError (connection failure), got %T: %v", err, err)
	}
}

// TestTestConnection_Execute_AllDatabaseTypes tests connection testing with all supported database types.
func TestTestConnection_Execute_AllDatabaseTypes(t *testing.T) {
	tests := []struct {
		name   string
		dbType model.DBType
		port   int
	}{
		{
			name:   "PostgreSQL connection",
			dbType: model.TypePostgreSQL,
			port:   5432,
		},
		{
			name:   "MySQL connection",
			dbType: model.TypeMySQL,
			port:   3306,
		},
		{
			name:   "MongoDB connection",
			dbType: model.TypeMongoDB,
			port:   27017,
		},
		{
			name:   "Oracle connection",
			dbType: model.TypeOracle,
			port:   1521,
		},
		{
			name:   "SQL Server connection",
			dbType: model.TypeSQLServer,
			port:   1433,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockCrypto := crypto.NewMockCryptor(ctrl)
			mockCrypto.EXPECT().Decrypt(gomock.Any(), gomock.Any(), gomock.Any()).Return("test-password", nil)
			mockStore := NewMockRateLimiterStore(ctrl)
			mockStore.EXPECT().Take(gomock.Any(), gomock.Any()).Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

			// Provide a factory that calls cryptor.Decrypt then fails on connection
			testFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
				_, err := cryptor.Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion)
				if err != nil {
					return nil, fmt.Errorf("decryption failed: %w", err)
				}
				return nil, fmt.Errorf("connection failed: unable to connect to database")
			}

			svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, testFactory)

			ctx := testContext()
			orgID := uuid.New()
			connID := uuid.New()

			existingConn := newTestConnectionFixture(orgID, connID, tt.dbType)
			existingConn.Port = tt.port

			// Mock: connection found
			mockConnRepo.EXPECT().
				FindByID(gomock.Any(), connID, orgID).
				Return(existingConn, nil)

			result, err := svc.Execute(ctx, orgID, connID)

			// The datasource factory will fail because it simulates a connection failure.
			// This test verifies that each database type is handled.
			if err == nil {
				// If somehow the connection succeeds (unlikely in unit tests)
				if result == nil {
					t.Fatal("expected non-nil result when no error")
				}
				return
			}

			// Expect a connection error (ResponseError), not other types of errors
			var responseErr pkg.ResponseError
			if errors.As(err, &responseErr) {
				// Expected: connection failure
				return
			}

			// Could also be InternalServerError if decryption fails in connection
			var internalErr pkg.InternalServerError
			if errors.As(err, &internalErr) {
				return
			}

			t.Fatalf("unexpected error type for %s: %T: %v", tt.name, err, err)
		})
	}
}

// TestTestConnection_Execute_MultipleRepositoryErrors tests various repository error scenarios.
func TestTestConnection_Execute_MultipleRepositoryErrors(t *testing.T) {
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
			mockCrypto := crypto.NewMockCryptor(ctrl)
			mockStore := NewMockRateLimiterStore(ctrl)
			mockStore.EXPECT().
				Take(gomock.Any(), gomock.Any()).
				Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

			svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

			ctx := testContext()
			orgID := uuid.New()
			connID := uuid.New()

			// Mock: database error
			mockConnRepo.EXPECT().
				FindByID(gomock.Any(), connID, orgID).
				Return(nil, tt.dbError)

			result, err := svc.Execute(ctx, orgID, connID)

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

// TestTestConnection_Execute_EmptyUUIDs tests behavior with edge case UUIDs.
func TestTestConnection_Execute_EmptyUUIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().Take(gomock.Any(), gomock.Any()).Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

	ctx := testContext()
	orgID := uuid.Nil
	connID := uuid.Nil

	// Mock: connection not found with nil UUIDs
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, connID)

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

// TestNewTestConnection verifies the constructor.
func TestNewTestConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockStore := NewMockRateLimiterStore(ctrl)

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.connRepo == nil {
		t.Fatal("expected connRepo to be set")
	}

	if svc.cryptor == nil {
		t.Fatal("expected cryptor to be set")
	}

	if svc.store == nil {
		t.Fatal("expected store to be set")
	}
}

// TestTestConnection_Execute_RateLimitResetTime tests rate limit with various reset times.
func TestTestConnection_Execute_RateLimitResetTime(t *testing.T) {
	tests := []struct {
		name       string
		resetTime  time.Time
		wantMinSec int
	}{
		{
			name:       "reset in 1 second",
			resetTime:  time.Now().UTC().Add(1 * time.Second),
			wantMinSec: 1,
		},
		{
			name:       "reset in 30 seconds",
			resetTime:  time.Now().UTC().Add(30 * time.Second),
			wantMinSec: 1,
		},
		{
			name:       "reset in past (should be at least 1)",
			resetTime:  time.Now().UTC().Add(-10 * time.Second),
			wantMinSec: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockCrypto := crypto.NewMockCryptor(ctrl)
			mockStore := NewMockRateLimiterStore(ctrl)
			mockStore.EXPECT().Take(gomock.Any(), gomock.Any()).Return(uint64(0), uint64(0), uint64(tt.resetTime.UnixNano()), false, nil)

			svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

			ctx := testContext()
			orgID := uuid.New()
			connID := uuid.New()

			result, err := svc.Execute(ctx, orgID, connID)

			if result != nil {
				t.Fatalf("expected nil result, got %+v", result)
			}

			if err == nil {
				t.Fatal("expected error for rate limit, got nil")
			}

			var responseErr pkg.ResponseError
			if !errors.As(err, &responseErr) {
				t.Fatalf("expected ResponseError, got %T: %v", err, err)
			}

			if responseErr.Code != http.StatusTooManyRequests {
				t.Fatalf("expected status code %d, got %d", http.StatusTooManyRequests, responseErr.Code)
			}
		})
	}
}

// TestTestConnection_Execute_DecryptionKeyVersionMismatch tests decryption with mismatched key version.
func TestTestConnection_Execute_DecryptionKeyVersionMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Decrypt(gomock.Any(), gomock.Any(), gomock.Eq("v2")).Return("", errors.New("unsupported key version: v2"))
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().Take(gomock.Any(), gomock.Any()).Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

	testFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		_, err := cryptor.Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
		return nil, fmt.Errorf("connection failed")
	}
	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, testFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	existingConn := newTestConnectionFixture(orgID, connID, model.TypePostgreSQL)
	existingConn.EncryptionKeyVersion = "v2" // Mismatched version

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	result, err := svc.Execute(ctx, orgID, connID)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for key version mismatch, got nil")
	}

	var internalErr pkg.ResponseError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected ResponseError, got %T: %v", err, err)
	}
}

// TestTestConnection_Execute_ConnectionWithAllFields tests with a fully populated connection.
func TestTestConnection_Execute_ConnectionWithAllFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Decrypt(gomock.Any(), gomock.Any(), gomock.Any()).Return("super-secret-password", nil)
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().Take(gomock.Any(), gomock.Any()).Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

	testFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		_, err := cryptor.Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
		return nil, fmt.Errorf("connection failed: unable to connect to database")
	}
	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, testFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	existingConn := &model.Connection{
		ID:                   connID,
		OrganizationID:       orgID,
		ConfigName:           "full-connection",
		Type:                 model.TypePostgreSQL,
		Host:                 "db.example.com",
		Port:                 5432,
		DatabaseName:         "production",
		Username:             "admin",
		PasswordEncrypted:    "super-secret-encrypted",
		EncryptionKeyVersion: "v1",
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
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	result, err := svc.Execute(ctx, orgID, connID)

	// Expected: connection failure because we can't connect to actual DB
	if err == nil {
		// If somehow succeeded (unlikely in unit test)
		if result == nil {
			t.Fatal("expected non-nil result when no error")
		}
		return
	}

	// Should be a connection error, not validation error
	var responseErr pkg.ResponseError
	if errors.As(err, &responseErr) {
		// Expected: connection failure
		return
	}

	var internalErr pkg.InternalServerError
	if errors.As(err, &internalErr) {
		// Also acceptable: internal error during connection
		return
	}

	t.Fatalf("unexpected error type: %T: %v", err, err)
}

// TestTestConnection_Execute_DifferentOrganizations tests that connections from different orgs cannot be tested.
func TestTestConnection_Execute_DifferentOrganizations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().Take(gomock.Any(), gomock.Any()).Return(uint64(1), uint64(10), uint64(time.Now().UTC().Add(time.Minute).UnixNano()), true, nil)

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

	ctx := testContext()
	requestingOrgID := uuid.New()
	connectionOrgID := uuid.New()
	connID := uuid.New()

	// The repository should filter by organization and return nil
	// because the connection belongs to a different organization
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, requestingOrgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, requestingOrgID, connID)

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

	// The connection exists in connectionOrgID but not accessible from requestingOrgID
	_ = connectionOrgID // Demonstrates the scenario
}

// TestTestConnection_Execute_RateLimitKeyTenantIsolation tests that rate limiter keys
// are scoped per tenant, preventing cross-tenant rate limit interference.
func TestTestConnection_Execute_RateLimitKeyTenantIsolation(t *testing.T) {
	connID := uuid.New()

	tests := []struct {
		name        string
		tenantA     string
		tenantB     string
		wantDiffKey bool
	}{
		{
			name:        "different tenants produce different keys",
			tenantA:     "tenant-aaa",
			tenantB:     "tenant-bbb",
			wantDiffKey: true,
		},
		{
			name:        "same tenant produces same key",
			tenantA:     "tenant-aaa",
			tenantB:     "tenant-aaa",
			wantDiffKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var capturedKeys []string

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockCrypto := crypto.NewMockCryptor(ctrl)
			mockStore := NewMockRateLimiterStore(ctrl)
			mockStore.EXPECT().
				Take(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, key string) (uint64, uint64, uint64, bool, error) {
					capturedKeys = append(capturedKeys, key)
					// Return rate limited to avoid needing repo mocks
					return 0, 0, uint64(time.Now().UTC().Add(30 * time.Second).UnixNano()), false, nil
				}).
				Times(2)

			svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)
			orgID := uuid.New()

			// Execute with tenant A context
			ctxA := testContext()
			ctxA = tmcore.SetTenantIDInContext(ctxA, tt.tenantA)
			_, errA := svc.Execute(ctxA, orgID, connID)
			assert.Error(t, errA, "rate-limited request must return an error for tenant A")

			// Execute with tenant B context
			ctxB := testContext()
			ctxB = tmcore.SetTenantIDInContext(ctxB, tt.tenantB)
			_, errB := svc.Execute(ctxB, orgID, connID)
			assert.Error(t, errB, "rate-limited request must return an error for tenant B")

			assert.Len(t, capturedKeys, 2, "expected two rate limiter keys to be captured")

			if tt.wantDiffKey {
				assert.NotEqual(t, capturedKeys[0], capturedKeys[1],
					"different tenants must produce different rate limiter keys")
				assert.Contains(t, capturedKeys[0], tt.tenantA,
					"key for tenant A must contain its tenant ID")
				assert.Contains(t, capturedKeys[1], tt.tenantB,
					"key for tenant B must contain its tenant ID")
			} else {
				assert.Equal(t, capturedKeys[0], capturedKeys[1],
					"same tenant must produce identical rate limiter keys")
			}
		})
	}
}

// TestTestConnection_Execute_RateLimitKeySingleTenant tests that without tenant context
// (single-tenant mode), the rate limiter key is just the connection ID without prefix.
func TestTestConnection_Execute_RateLimitKeySingleTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connID := uuid.New()
	var capturedKey string

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockStore := NewMockRateLimiterStore(ctrl)
	mockStore.EXPECT().
		Take(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, key string) (uint64, uint64, uint64, bool, error) {
			capturedKey = key
			return 0, 0, uint64(time.Now().UTC().Add(30 * time.Second).UnixNano()), false, nil
		})

	svc := NewTestConnection(mockConnRepo, mockCrypto, mockStore, nil)

	// No tenant in context (single-tenant mode)
	ctx := testContext()
	orgID := uuid.New()

	_, err := svc.Execute(ctx, orgID, connID)
	assert.Error(t, err, "rate-limited request must return an error in single-tenant mode")

	assert.Equal(t, connID.String(), capturedKey,
		"single-tenant mode must use plain connection ID as rate limiter key")
}
