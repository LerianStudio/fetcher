package query

import (
	"errors"
	"fmt"
	gohttp "net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	productMock "github.com/LerianStudio/fetcher/pkg/mongodb/product"
	"github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

// newMockProductRepoForList creates a permissive product mock for list tests that don't test product filtering.
func newMockProductRepoForList(ctrl *gomock.Controller) *productMock.MockRepository {
	return productMock.NewMockRepository(ctrl)
}

// newListTestConnection creates a test Connection with the given parameters for list tests.
func newListTestConnection(orgID, connID uuid.UUID, configName string, dbType model.DBType) *model.Connection {
	return &model.Connection{
		ID:                   connID,
		OrganizationID:       orgID,
		ConfigName:           configName,
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

// TestListConnections_Execute_Success tests successful listing of connections.
func TestListConnections_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	conn1 := newListTestConnection(orgID, uuid.New(), "connection-1", model.TypePostgreSQL)
	conn2 := newListTestConnection(orgID, uuid.New(), "connection-2", model.TypeMySQL)
	expectedList := []*model.Connection{conn1, conn2}

	// Mock: list returns connections
	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(expectedList, nil)

	result, err := svc.Execute(ctx, orgID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if len(result) != len(expectedList) {
		t.Fatalf("expected %d connections, got %d", len(expectedList), len(result))
	}

	if result[0].ConfigName != "connection-1" {
		t.Fatalf("expected first connection name 'connection-1', got %s", result[0].ConfigName)
	}

	if result[1].ConfigName != "connection-2" {
		t.Fatalf("expected second connection name 'connection-2', got %s", result[1].ConfigName)
	}
}

// TestListConnections_Execute_EmptyList tests that empty list returns empty slice, not error.
func TestListConnections_Execute_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Mock: list returns nil (no connections found)
	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result (empty slice)")
	}

	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d connections", len(result))
	}
}

// TestListConnections_Execute_RepositoryError tests repository error handling.
func TestListConnections_Execute_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	dbError := errors.New("database connection failed")

	// Mock: list returns error
	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, nil, filters)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to be dbError, got %v", err)
	}
}

// TestNewListConnections verifies the constructor.
func TestNewListConnections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.connRepo == nil {
		t.Fatal("expected connRepo to be set")
	}

	if svc.productRepo == nil {
		t.Fatal("expected productRepo to be set")
	}
}

// TestListConnections_Execute_TableDriven uses table-driven tests for various scenarios.
func TestListConnections_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		filters         http.QueryHeader
		setupMocks      func(*connRepo.MockRepository, uuid.UUID, http.QueryHeader)
		wantErr         bool
		wantResultCount int
	}{
		{
			name: "successful list with multiple connections",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  1,
			},
			setupMocks: func(mock *connRepo.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				connections := []*model.Connection{
					newListTestConnection(orgID, uuid.New(), "conn-1", model.TypePostgreSQL),
					newListTestConnection(orgID, uuid.New(), "conn-2", model.TypeMySQL),
					newListTestConnection(orgID, uuid.New(), "conn-3", model.TypeMongoDB),
				}
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(connections, nil)
			},
			wantErr:         false,
			wantResultCount: 3,
		},
		{
			name: "empty list returns empty slice",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  1,
			},
			setupMocks: func(mock *connRepo.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(nil, nil)
			},
			wantErr:         false,
			wantResultCount: 0,
		},
		{
			name: "repository error",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  1,
			},
			setupMocks: func(mock *connRepo.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(nil, errors.New("database error"))
			},
			wantErr:         true,
			wantResultCount: 0,
		},
		{
			name: "list with page 2",
			filters: http.QueryHeader{
				Limit: 5,
				Page:  2,
			},
			setupMocks: func(mock *connRepo.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				connections := []*model.Connection{
					newListTestConnection(orgID, uuid.New(), "conn-6", model.TypePostgreSQL),
					newListTestConnection(orgID, uuid.New(), "conn-7", model.TypeMySQL),
				}
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(connections, nil)
			},
			wantErr:         false,
			wantResultCount: 2,
		},
		{
			name: "list with limit 1",
			filters: http.QueryHeader{
				Limit: 1,
				Page:  1,
			},
			setupMocks: func(mock *connRepo.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				connections := []*model.Connection{
					newListTestConnection(orgID, uuid.New(), "conn-1", model.TypePostgreSQL),
				}
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(connections, nil)
			},
			wantErr:         false,
			wantResultCount: 1,
		},
		{
			name: "list with large limit",
			filters: http.QueryHeader{
				Limit: 100,
				Page:  1,
			},
			setupMocks: func(mock *connRepo.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				connections := make([]*model.Connection, 50)
				for i := 0; i < 50; i++ {
					connections[i] = newListTestConnection(orgID, uuid.New(), "conn", model.TypePostgreSQL)
				}
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(connections, nil)
			},
			wantErr:         false,
			wantResultCount: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			ctx := testContext()
			orgID := uuid.New()

			tt.setupMocks(mockConnRepo, orgID, tt.filters)

			mockProductRepo := newMockProductRepoForList(ctrl)
			svc := NewListConnections(mockConnRepo, mockProductRepo)

			result, err := svc.Execute(ctx, orgID, nil, tt.filters)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil && tt.wantResultCount > 0 {
				t.Fatal("expected non-nil result")
			}

			if len(result) != tt.wantResultCount {
				t.Fatalf("expected %d connections, got %d", tt.wantResultCount, len(result))
			}
		})
	}
}

// TestListConnections_Execute_OrganizationIsolation tests that connections are isolated by organization.
func TestListConnections_Execute_OrganizationIsolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Only connections for the specified organization should be returned
	org1Connections := []*model.Connection{
		newListTestConnection(orgID, uuid.New(), "org1-conn-1", model.TypePostgreSQL),
		newListTestConnection(orgID, uuid.New(), "org1-conn-2", model.TypeMySQL),
	}

	// Mock: list returns only connections for the given organization
	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(org1Connections, nil)

	result, err := svc.Execute(ctx, orgID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(result))
	}

	// Verify all returned connections belong to the requested organization
	for _, conn := range result {
		if conn.OrganizationID != orgID {
			t.Fatalf("expected organization ID %s, got %s", orgID, conn.OrganizationID)
		}
	}
}

// TestListConnections_Execute_DifferentOrganizations tests listing connections for different organizations.
func TestListConnections_Execute_DifferentOrganizations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	org1ID := uuid.New()
	org2ID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Create connections for different organizations
	org1Connections := []*model.Connection{
		newListTestConnection(org1ID, uuid.New(), "org1-conn", model.TypePostgreSQL),
	}
	org2Connections := []*model.Connection{
		newListTestConnection(org2ID, uuid.New(), "org2-conn-1", model.TypeMySQL),
		newListTestConnection(org2ID, uuid.New(), "org2-conn-2", model.TypeMongoDB),
	}

	// Mock: list for org1
	mockConnRepo.EXPECT().
		List(gomock.Any(), org1ID, filters).
		Return(org1Connections, nil)

	result1, err := svc.Execute(ctx, org1ID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error for org1: %v", err)
	}

	if len(result1) != 1 {
		t.Fatalf("expected 1 connection for org1, got %d", len(result1))
	}

	// Mock: list for org2
	mockConnRepo.EXPECT().
		List(gomock.Any(), org2ID, filters).
		Return(org2Connections, nil)

	result2, err := svc.Execute(ctx, org2ID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error for org2: %v", err)
	}

	if len(result2) != 2 {
		t.Fatalf("expected 2 connections for org2, got %d", len(result2))
	}
}

// TestListConnections_Execute_WithMetadataFilter tests listing with metadata filters.
func TestListConnections_Execute_WithMetadataFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()

	metadata := bson.M{"environment": "production"}
	filters := http.QueryHeader{
		Limit:       10,
		Page:        1,
		Metadata:    &metadata,
		UseMetadata: true,
	}

	connections := []*model.Connection{
		newListTestConnection(orgID, uuid.New(), "prod-conn", model.TypePostgreSQL),
	}

	// Mock: list with metadata filter
	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(connections, nil)

	result, err := svc.Execute(ctx, orgID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(result))
	}
}

// TestListConnections_Execute_WithDateFilters tests listing with date range filters.
func TestListConnections_Execute_WithDateFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()

	startDate := time.Now().UTC().Add(-7 * 24 * time.Hour) // 7 days ago
	endDate := time.Now().UTC()

	filters := http.QueryHeader{
		Limit:     10,
		Page:      1,
		StartDate: startDate,
		EndDate:   endDate,
	}

	connections := []*model.Connection{
		newListTestConnection(orgID, uuid.New(), "recent-conn", model.TypePostgreSQL),
	}

	// Mock: list with date filters
	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(connections, nil)

	result, err := svc.Execute(ctx, orgID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(result))
	}
}

// TestListConnections_Execute_WithSortOrder tests listing with sort order.
func TestListConnections_Execute_WithSortOrder(t *testing.T) {
	tests := []struct {
		name      string
		sortOrder string
	}{
		{
			name:      "ascending order",
			sortOrder: "asc",
		},
		{
			name:      "descending order",
			sortOrder: "desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)

			mockProductRepo := newMockProductRepoForList(ctrl)
			svc := NewListConnections(mockConnRepo, mockProductRepo)

			ctx := testContext()
			orgID := uuid.New()
			filters := http.QueryHeader{
				Limit:     10,
				Page:      1,
				SortOrder: tt.sortOrder,
			}

			connections := []*model.Connection{
				newListTestConnection(orgID, uuid.New(), "conn-1", model.TypePostgreSQL),
				newListTestConnection(orgID, uuid.New(), "conn-2", model.TypeMySQL),
			}

			// Mock: list with sort order
			mockConnRepo.EXPECT().
				List(gomock.Any(), orgID, filters).
				Return(connections, nil)

			result, err := svc.Execute(ctx, orgID, nil, filters)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != 2 {
				t.Fatalf("expected 2 connections, got %d", len(result))
			}
		})
	}
}

// TestListConnections_Execute_ConnectionTypes tests listing connections of different database types.
func TestListConnections_Execute_ConnectionTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Create connections of all supported types
	connections := []*model.Connection{
		newListTestConnection(orgID, uuid.New(), "postgres-conn", model.TypePostgreSQL),
		newListTestConnection(orgID, uuid.New(), "mysql-conn", model.TypeMySQL),
		newListTestConnection(orgID, uuid.New(), "mongo-conn", model.TypeMongoDB),
		newListTestConnection(orgID, uuid.New(), "oracle-conn", model.TypeOracle),
		newListTestConnection(orgID, uuid.New(), "sqlserver-conn", model.TypeSQLServer),
	}

	// Mock: list returns connections of all types
	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(connections, nil)

	result, err := svc.Execute(ctx, orgID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 5 {
		t.Fatalf("expected 5 connections, got %d", len(result))
	}

	// Verify all database types are present
	typeCount := make(map[model.DBType]int)
	for _, conn := range result {
		typeCount[conn.Type]++
	}

	expectedTypes := []model.DBType{
		model.TypePostgreSQL,
		model.TypeMySQL,
		model.TypeMongoDB,
		model.TypeOracle,
		model.TypeSQLServer,
	}

	for _, typ := range expectedTypes {
		if typeCount[typ] != 1 {
			t.Fatalf("expected 1 connection of type %s, got %d", typ, typeCount[typ])
		}
	}
}

// TestListConnections_Execute_WithCursor tests listing with cursor-based pagination.
func TestListConnections_Execute_WithCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit:  10,
		Cursor: "encoded-cursor-value",
	}

	connections := []*model.Connection{
		newListTestConnection(orgID, uuid.New(), "conn-after-cursor", model.TypePostgreSQL),
	}

	// Mock: list with cursor
	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(connections, nil)

	result, err := svc.Execute(ctx, orgID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(result))
	}
}

// TestListConnections_Execute_EmptyFilters tests listing with empty/default filters.
func TestListConnections_Execute_EmptyFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{} // Empty filters

	connections := []*model.Connection{
		newListTestConnection(orgID, uuid.New(), "conn-1", model.TypePostgreSQL),
	}

	// Mock: list with empty filters
	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(connections, nil)

	result, err := svc.Execute(ctx, orgID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(result))
	}
}

// TestListConnections_Execute_Pagination tests various pagination scenarios.
func TestListConnections_Execute_Pagination(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		page   int
		expect int
	}{
		{
			name:   "first page with limit 5",
			limit:  5,
			page:   1,
			expect: 5,
		},
		{
			name:   "second page with limit 5",
			limit:  5,
			page:   2,
			expect: 3,
		},
		{
			name:   "large page number with no results",
			limit:  10,
			page:   100,
			expect: 0,
		},
		{
			name:   "zero limit uses default",
			limit:  0,
			page:   1,
			expect: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)

			mockProductRepo := newMockProductRepoForList(ctrl)
			svc := NewListConnections(mockConnRepo, mockProductRepo)

			ctx := testContext()
			orgID := uuid.New()
			filters := http.QueryHeader{
				Limit: tt.limit,
				Page:  tt.page,
			}

			// Create expected connections based on test case
			var connections []*model.Connection
			if tt.expect > 0 {
				connections = make([]*model.Connection, tt.expect)
				for i := 0; i < tt.expect; i++ {
					connections[i] = newListTestConnection(orgID, uuid.New(), "conn", model.TypePostgreSQL)
				}
			}

			// Mock: list returns connections based on pagination
			mockConnRepo.EXPECT().
				List(gomock.Any(), orgID, filters).
				Return(connections, nil)

			result, err := svc.Execute(ctx, orgID, nil, filters)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != tt.expect {
				t.Fatalf("expected %d connections, got %d", tt.expect, len(result))
			}
		})
	}
}

// TestListConnections_Execute_ConnectionWithSSL tests listing connections with SSL configuration.
func TestListConnections_Execute_ConnectionWithSSL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	connWithSSL := newListTestConnection(orgID, uuid.New(), "ssl-conn", model.TypePostgreSQL)
	connWithSSL.SSL = &model.SSLConfig{
		Mode: "require",
		CA:   "-----BEGIN CERTIFICATE-----\ntest-ca\n-----END CERTIFICATE-----",
		Cert: "-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----",
		Key:  "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----",
	}

	connWithoutSSL := newListTestConnection(orgID, uuid.New(), "no-ssl-conn", model.TypeMySQL)

	connections := []*model.Connection{connWithSSL, connWithoutSSL}

	// Mock: list returns connections with and without SSL
	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(connections, nil)

	result, err := svc.Execute(ctx, orgID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(result))
	}

	// Verify SSL connection
	var sslConn *model.Connection
	for _, conn := range result {
		if conn.ConfigName == "ssl-conn" {
			sslConn = conn
			break
		}
	}

	if sslConn == nil {
		t.Fatal("expected to find ssl-conn")
	}

	if sslConn.SSL == nil {
		t.Fatal("expected SSL configuration to be present")
	}

	if sslConn.SSL.Mode != "require" {
		t.Fatalf("expected SSL mode 'require', got %s", sslConn.SSL.Mode)
	}
}

// TestListConnections_Execute_ErrorScenarios tests various error scenarios.
func TestListConnections_Execute_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*connRepo.MockRepository, uuid.UUID, http.QueryHeader)
		errorMsg  string
	}{
		{
			name: "database connection error",
			setupMock: func(mock *connRepo.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(nil, errors.New("database connection failed"))
			},
			errorMsg: "database connection failed",
		},
		{
			name: "timeout error",
			setupMock: func(mock *connRepo.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(nil, errors.New("context deadline exceeded"))
			},
			errorMsg: "context deadline exceeded",
		},
		{
			name: "permission denied error",
			setupMock: func(mock *connRepo.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(nil, errors.New("permission denied"))
			},
			errorMsg: "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)

			mockProductRepo := newMockProductRepoForList(ctrl)
			svc := NewListConnections(mockConnRepo, mockProductRepo)

			ctx := testContext()
			orgID := uuid.New()
			filters := http.QueryHeader{
				Limit: 10,
				Page:  1,
			}

			tt.setupMock(mockConnRepo, orgID, filters)

			result, err := svc.Execute(ctx, orgID, nil, filters)

			if result != nil {
				t.Fatalf("expected nil result, got %+v", result)
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if err.Error() != tt.errorMsg {
				t.Fatalf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}

// TestListConnections_Execute_EmptyOrganizationID tests behavior with nil/empty organization ID.
func TestListConnections_Execute_EmptyOrganizationID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockProductRepo := newMockProductRepoForList(ctrl)
	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	emptyOrgID := uuid.Nil
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Repository should handle empty org ID - it may return empty list or error
	mockConnRepo.EXPECT().
		List(gomock.Any(), emptyOrgID, filters).
		Return([]*model.Connection{}, nil)

	result, err := svc.Execute(ctx, emptyOrgID, nil, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected empty result for nil org ID, got %d connections", len(result))
	}
}

// TestListConnections_Execute_WithProductFilter_Success tests that listing with a valid product ID
// validates the product exists and then returns connections filtered by that product.
func TestListConnections_Execute_WithProductFilter_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	pid := uuid.New()
	productID := &pid
	filters := http.QueryHeader{}

	existingProduct := &model.Product{
		ID:             pid,
		OrganizationID: orgID,
		Code:           "test-product",
		Name:           "Test Product",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Mock: product exists
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), pid, orgID).
		Return(existingProduct, nil)

	// After product validation, filters.ProductID is set; connRepo.List is called with updated filters.
	expectedFilters := filters
	expectedFilters.ProductID = productID

	expectedConnections := []*model.Connection{
		newListTestConnection(orgID, uuid.New(), "product-conn-1", model.TypePostgreSQL),
		newListTestConnection(orgID, uuid.New(), "product-conn-2", model.TypeMySQL),
	}

	mockConnRepo.EXPECT().
		List(gomock.Any(), orgID, expectedFilters).
		Return(expectedConnections, nil)

	result, err := svc.Execute(ctx, orgID, productID, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	assert.Equal(t, 2, len(result))
	assert.Equal(t, "product-conn-1", result[0].ConfigName)
	assert.Equal(t, "product-conn-2", result[1].ConfigName)
}

// TestListConnections_Execute_WithProductFilter_NotFound tests that listing with a product ID
// that does not exist returns a 404 error and does not call connRepo.List.
func TestListConnections_Execute_WithProductFilter_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	pid := uuid.New()
	productID := &pid
	filters := http.QueryHeader{}

	// Mock: product not found (nil, nil)
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), pid, orgID).
		Return(nil, nil)

	// connRepo.List should NOT be called — no expectation set means gomock will fail if called.

	result, err := svc.Execute(ctx, orgID, productID, filters)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for non-existent product, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}

	assert.Equal(t, gohttp.StatusNotFound, respErr.StatusCode)
}

// TestListConnections_Execute_WithProductFilter_RepoError tests that a repository error
// from productRepo.FindByID is propagated and connRepo.List is not called.
func TestListConnections_Execute_WithProductFilter_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewListConnections(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	pid := uuid.New()
	productID := &pid
	filters := http.QueryHeader{}

	dbError := fmt.Errorf("database error")

	// Mock: productRepo returns error
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), pid, orgID).
		Return(nil, dbError)

	// connRepo.List should NOT be called — no expectation set means gomock will fail if called.

	result, err := svc.Execute(ctx, orgID, productID, filters)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	assert.Equal(t, dbError, err)
}
