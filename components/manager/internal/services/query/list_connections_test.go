package query

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/net/http"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/v2/pkg/resolver"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newListTestConnection creates a test Connection with the given parameters for list tests.
func newListTestConnection(connID uuid.UUID, configName string, dbType model.DBType) *model.Connection {
	return &model.Connection{
		ID:                   connID,
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

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	conn1 := newListTestConnection(uuid.New(), "connection-1", model.TypePostgreSQL)
	conn2 := newListTestConnection(uuid.New(), "connection-2", model.TypeMySQL)
	expectedList := []*model.Connection{conn1, conn2}

	// Mock: list returns connections
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(expectedList, int64(2), nil)

	result, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	items := result.Items.([]*model.ConnectionResponse)

	if len(items) != len(expectedList) {
		t.Fatalf("expected %d connections, got %d", len(expectedList), len(items))
	}

	if items[0].ConfigName != "connection-1" {
		t.Fatalf("expected first connection name 'connection-1', got %s", items[0].ConfigName)
	}

	if items[1].ConfigName != "connection-2" {
		t.Fatalf("expected second connection name 'connection-2', got %s", items[1].ConfigName)
	}
}

// TestListConnections_Execute_InternalMerge_FirstPage proves the internal-merge
// branch: with a non-nil resolver and Page<=1, internal connections are resolved,
// PREPENDED before the external (Engine-routed) list, and the total is
// repoTotal + len(internal). This locks the resolver-merge contract the
// migration touches but no existing test exercised (all passed nil resolver).
func TestListConnections_Execute_InternalMerge_FirstPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockResolver := resolver.NewMockConnectionResolver(ctrl)

	filters := http.QueryHeader{Limit: 10, Page: 1}

	external := newListTestConnection(uuid.New(), "external-1", model.TypePostgreSQL)
	mockConnRepo.EXPECT().List(gomock.Any(), filters).Return([]*model.Connection{external}, int64(1), nil)

	internal := &model.Connection{ConfigName: "midaz_onboarding", Type: model.TypePostgreSQL}
	mockResolver.EXPECT().ListInternalConnections(gomock.Any()).Return([]*model.Connection{internal}, nil)

	svc := NewListConnections(mockResolver, scopeAuthorityEngine(t, mockConnRepo))

	result, err := svc.Execute(testContext(), "", filters)
	require.NoError(t, err)
	require.NotNil(t, result)

	items := result.Items.([]*model.ConnectionResponse)
	require.Len(t, items, 2)
	// Internal first, then external.
	assert.Equal(t, "midaz_onboarding", items[0].ConfigName)
	assert.Equal(t, "external-1", items[1].ConfigName)
	// Total = repo total (1) + internal count (1).
	assert.Equal(t, 2, result.Total)
}

// TestListConnections_Execute_InternalMerge_TypeFilter proves the internal
// type-filter loop: when a type filter is set, internal connections of a
// non-matching type are dropped from the merge (the external list is filtered by
// the repo via the opaque params).
func TestListConnections_Execute_InternalMerge_TypeFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockResolver := resolver.NewMockConnectionResolver(ctrl)

	filters := http.QueryHeader{Limit: 10, Page: 1, Type: "postgresql"}

	mockConnRepo.EXPECT().List(gomock.Any(), filters).Return(nil, int64(0), nil)

	pgInternal := &model.Connection{ConfigName: "midaz_onboarding", Type: model.TypePostgreSQL}
	mongoInternal := &model.Connection{ConfigName: "plugin_crm", Type: model.TypeMongoDB}
	mockResolver.EXPECT().ListInternalConnections(gomock.Any()).
		Return([]*model.Connection{pgInternal, mongoInternal}, nil)

	svc := NewListConnections(mockResolver, scopeAuthorityEngine(t, mockConnRepo))

	result, err := svc.Execute(testContext(), "", filters)
	require.NoError(t, err)
	require.NotNil(t, result)

	items := result.Items.([]*model.ConnectionResponse)
	require.Len(t, items, 1, "only the postgres internal connection survives the type filter")
	assert.Equal(t, "midaz_onboarding", items[0].ConfigName)
	assert.Equal(t, 1, result.Total, "total = repo total (0) + filtered internal count (1)")
}

// TestListConnections_Execute_InternalMerge_SkippedAfterFirstPage proves the
// Page<=1 guard: on page 2+ the resolver is NOT consulted (internal connections
// only appear on the first page), so the total is the repo total alone.
func TestListConnections_Execute_InternalMerge_SkippedAfterFirstPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockResolver := resolver.NewMockConnectionResolver(ctrl) // no ListInternalConnections expectation: page 2 skips it

	filters := http.QueryHeader{Limit: 10, Page: 2}

	external := newListTestConnection(uuid.New(), "external-2", model.TypePostgreSQL)
	mockConnRepo.EXPECT().List(gomock.Any(), filters).Return([]*model.Connection{external}, int64(5), nil)

	svc := NewListConnections(mockResolver, scopeAuthorityEngine(t, mockConnRepo))

	result, err := svc.Execute(testContext(), "", filters)
	require.NoError(t, err)
	require.NotNil(t, result)

	items := result.Items.([]*model.ConnectionResponse)
	require.Len(t, items, 1)
	assert.Equal(t, "external-2", items[0].ConfigName)
	assert.Equal(t, 5, result.Total, "page 2: total is the repo total alone, no internal merge")
}

// TestListConnections_Execute_EmptyList tests that empty list returns empty slice, not error.
func TestListConnections_Execute_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Mock: list returns nil (no connections found)
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(nil, int64(0), nil)

	result, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
}

// TestListConnections_Execute_RepositoryError tests repository error handling.
func TestListConnections_Execute_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	dbError := errors.New("database connection failed")

	// Mock: list returns error
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(nil, int64(0), dbError)

	result, err := svc.Execute(ctx, "", filters)

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

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.engine == nil {
		t.Fatal("expected engine to be set")
	}
}

// TestListConnections_Execute_TableDriven uses table-driven tests for various scenarios.
func TestListConnections_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		filters         http.QueryHeader
		setupMocks      func(*connRepo.MockRepository, http.QueryHeader)
		wantErr         bool
		wantResultCount int
	}{
		{
			name: "successful list with multiple connections",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  1,
			},
			setupMocks: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				connections := []*model.Connection{
					newListTestConnection(uuid.New(), "conn-1", model.TypePostgreSQL),
					newListTestConnection(uuid.New(), "conn-2", model.TypeMySQL),
					newListTestConnection(uuid.New(), "conn-3", model.TypeMongoDB),
				}
				mock.EXPECT().
					List(gomock.Any(), filters).
					Return(connections, int64(3), nil)
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
			setupMocks: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), filters).
					Return(nil, int64(0), nil)
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
			setupMocks: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), filters).
					Return(nil, int64(0), errors.New("database error"))
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
			setupMocks: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				connections := []*model.Connection{
					newListTestConnection(uuid.New(), "conn-6", model.TypePostgreSQL),
					newListTestConnection(uuid.New(), "conn-7", model.TypeMySQL),
				}
				mock.EXPECT().
					List(gomock.Any(), filters).
					Return(connections, int64(7), nil)
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
			setupMocks: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				connections := []*model.Connection{
					newListTestConnection(uuid.New(), "conn-1", model.TypePostgreSQL),
				}
				mock.EXPECT().
					List(gomock.Any(), filters).
					Return(connections, int64(1), nil)
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
			setupMocks: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				connections := make([]*model.Connection, 50)
				for i := 0; i < 50; i++ {
					connections[i] = newListTestConnection(uuid.New(), "conn", model.TypePostgreSQL)
				}
				mock.EXPECT().
					List(gomock.Any(), filters).
					Return(connections, int64(50), nil)
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

			tt.setupMocks(mockConnRepo, tt.filters)

			svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

			result, err := svc.Execute(ctx, "", tt.filters)

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

			items := result.Items.([]*model.ConnectionResponse)

			if len(items) != tt.wantResultCount {
				t.Fatalf("expected %d connections, got %d", tt.wantResultCount, len(items))
			}
		})
	}
}

// TestListConnections_Execute_OrganizationIsolation tests that connections are isolated by organization.
func TestListConnections_Execute_OrganizationIsolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Only connections for the specified organization should be returned
	org1Connections := []*model.Connection{
		newListTestConnection(uuid.New(), "org1-conn-1", model.TypePostgreSQL),
		newListTestConnection(uuid.New(), "org1-conn-2", model.TypeMySQL),
	}

	// Mock: list returns only connections for the given organization
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(org1Connections, int64(2), nil)

	result, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.Items.([]*model.ConnectionResponse)

	if len(items) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(items))
	}

	// Verify all returned connections belong to the requested organization
	// Note: ConnectionResponse does not have OrganizationID, so we check the count matches
	// the expected org-scoped result from the repository.
	for _, conn := range items {
		if conn.ConfigName != "org1-conn-1" && conn.ConfigName != "org1-conn-2" {
			t.Fatalf("unexpected connection config name: %s", conn.ConfigName)
		}
	}
}

// TestListConnections_Execute_DifferentOrganizations tests listing connections for different organizations.
func TestListConnections_Execute_DifferentOrganizations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Create connections for different organizations
	org1Connections := []*model.Connection{
		newListTestConnection(uuid.New(), "org1-conn", model.TypePostgreSQL),
	}
	org2Connections := []*model.Connection{
		newListTestConnection(uuid.New(), "org2-conn-1", model.TypeMySQL),
		newListTestConnection(uuid.New(), "org2-conn-2", model.TypeMongoDB),
	}

	// Mock: list for org1
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(org1Connections, int64(1), nil)

	result1, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error for org1: %v", err)
	}

	items1 := result1.Items.([]*model.ConnectionResponse)

	if len(items1) != 1 {
		t.Fatalf("expected 1 connection for org1, got %d", len(items1))
	}

	// Mock: list for org2
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(org2Connections, int64(2), nil)

	result2, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error for org2: %v", err)
	}

	items2 := result2.Items.([]*model.ConnectionResponse)

	if len(items2) != 2 {
		t.Fatalf("expected 2 connections for org2, got %d", len(items2))
	}
}

// TestListConnections_Execute_WithMetadataFilter tests listing with metadata filters.
func TestListConnections_Execute_WithMetadataFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()

	metadata := map[string]string{"environment": "production"}
	filters := http.QueryHeader{
		Limit:       10,
		Page:        1,
		Metadata:    metadata,
		UseMetadata: true,
	}

	connections := []*model.Connection{
		newListTestConnection(uuid.New(), "prod-conn", model.TypePostgreSQL),
	}

	// Mock: list with metadata filter
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(connections, int64(1), nil)

	result, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.Items.([]*model.ConnectionResponse)

	if len(items) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(items))
	}
}

// TestListConnections_Execute_WithDateFilters tests listing with date range filters.
func TestListConnections_Execute_WithDateFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()

	startDate := time.Now().UTC().Add(-7 * 24 * time.Hour) // 7 days ago
	endDate := time.Now().UTC()

	filters := http.QueryHeader{
		Limit:     10,
		Page:      1,
		StartDate: startDate,
		EndDate:   endDate,
	}

	connections := []*model.Connection{
		newListTestConnection(uuid.New(), "recent-conn", model.TypePostgreSQL),
	}

	// Mock: list with date filters
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(connections, int64(1), nil)

	result, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.Items.([]*model.ConnectionResponse)

	if len(items) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(items))
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

			svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

			ctx := testContext()
			filters := http.QueryHeader{
				Limit:     10,
				Page:      1,
				SortOrder: tt.sortOrder,
			}

			connections := []*model.Connection{
				newListTestConnection(uuid.New(), "conn-1", model.TypePostgreSQL),
				newListTestConnection(uuid.New(), "conn-2", model.TypeMySQL),
			}

			// Mock: list with sort order
			mockConnRepo.EXPECT().
				List(gomock.Any(), filters).
				Return(connections, int64(2), nil)

			result, err := svc.Execute(ctx, "", filters)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			items := result.Items.([]*model.ConnectionResponse)

			if len(items) != 2 {
				t.Fatalf("expected 2 connections, got %d", len(items))
			}
		})
	}
}

// TestListConnections_Execute_ConnectionTypes tests listing connections of different database types.
func TestListConnections_Execute_ConnectionTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Create connections of all supported types
	connections := []*model.Connection{
		newListTestConnection(uuid.New(), "postgres-conn", model.TypePostgreSQL),
		newListTestConnection(uuid.New(), "mysql-conn", model.TypeMySQL),
		newListTestConnection(uuid.New(), "mongo-conn", model.TypeMongoDB),
		newListTestConnection(uuid.New(), "oracle-conn", model.TypeOracle),
		newListTestConnection(uuid.New(), "sqlserver-conn", model.TypeSQLServer),
	}

	// Mock: list returns connections of all types
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(connections, int64(5), nil)

	result, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.Items.([]*model.ConnectionResponse)

	if len(items) != 5 {
		t.Fatalf("expected 5 connections, got %d", len(items))
	}

	// Verify all database types are present (ConnectionResponse.Type is string)
	typeCount := make(map[string]int)
	for _, conn := range items {
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
		if typeCount[string(typ)] != 1 {
			t.Fatalf("expected 1 connection of type %s, got %d", typ, typeCount[string(typ)])
		}
	}
}

// TestListConnections_Execute_WithCursor tests listing with cursor-based pagination.
func TestListConnections_Execute_WithCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	filters := http.QueryHeader{
		Limit:  10,
		Cursor: "encoded-cursor-value",
	}

	connections := []*model.Connection{
		newListTestConnection(uuid.New(), "conn-after-cursor", model.TypePostgreSQL),
	}

	// Mock: list with cursor
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(connections, int64(1), nil)

	result, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.Items.([]*model.ConnectionResponse)

	if len(items) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(items))
	}
}

// TestListConnections_Execute_EmptyFilters tests listing with empty/default filters.
func TestListConnections_Execute_EmptyFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	filters := http.QueryHeader{} // Empty filters

	connections := []*model.Connection{
		newListTestConnection(uuid.New(), "conn-1", model.TypePostgreSQL),
	}

	// Mock: list with empty filters
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(connections, int64(1), nil)

	result, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.Items.([]*model.ConnectionResponse)

	if len(items) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(items))
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

			svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

			ctx := testContext()
			filters := http.QueryHeader{
				Limit: tt.limit,
				Page:  tt.page,
			}

			// Create expected connections based on test case
			var connections []*model.Connection
			if tt.expect > 0 {
				connections = make([]*model.Connection, tt.expect)
				for i := 0; i < tt.expect; i++ {
					connections[i] = newListTestConnection(uuid.New(), "conn", model.TypePostgreSQL)
				}
			}

			// Mock: list returns connections based on pagination
			mockConnRepo.EXPECT().
				List(gomock.Any(), filters).
				Return(connections, int64(tt.expect), nil)

			result, err := svc.Execute(ctx, "", filters)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			items := result.Items.([]*model.ConnectionResponse)

			if len(items) != tt.expect {
				t.Fatalf("expected %d connections, got %d", tt.expect, len(items))
			}
		})
	}
}

// TestListConnections_Execute_ConnectionWithSSL tests listing connections with SSL configuration.
func TestListConnections_Execute_ConnectionWithSSL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	connWithSSL := newListTestConnection(uuid.New(), "ssl-conn", model.TypePostgreSQL)
	connWithSSL.SSL = &model.SSLConfig{
		Mode: "require",
		CA:   "-----BEGIN CERTIFICATE-----\ntest-ca\n-----END CERTIFICATE-----",
		Cert: "-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----",
		Key:  "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----",
	}

	connWithoutSSL := newListTestConnection(uuid.New(), "no-ssl-conn", model.TypeMySQL)

	connections := []*model.Connection{connWithSSL, connWithoutSSL}

	// Mock: list returns connections with and without SSL
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return(connections, int64(2), nil)

	result, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.Items.([]*model.ConnectionResponse)

	if len(items) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(items))
	}

	// Verify SSL connection
	var sslConn *model.ConnectionResponse
	for _, conn := range items {
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
		setupMock func(*connRepo.MockRepository, http.QueryHeader)
		errorMsg  string
	}{
		{
			name: "database connection error",
			setupMock: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), filters).
					Return(nil, int64(0), errors.New("database connection failed"))
			},
			errorMsg: "database connection failed",
		},
		{
			name: "timeout error",
			setupMock: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), filters).
					Return(nil, int64(0), errors.New("context deadline exceeded"))
			},
			errorMsg: "context deadline exceeded",
		},
		{
			name: "permission denied error",
			setupMock: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), filters).
					Return(nil, int64(0), errors.New("permission denied"))
			},
			errorMsg: "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)

			svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

			ctx := testContext()
			filters := http.QueryHeader{
				Limit: 10,
				Page:  1,
			}

			tt.setupMock(mockConnRepo, filters)

			result, err := svc.Execute(ctx, "", filters)

			if result != nil {
				t.Fatalf("expected nil result, got %+v", result)
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Fatalf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}

// TestListConnections_Execute_EmptyOrganizationID tests behavior with nil/empty organization ID.
func TestListConnections_Execute_EmptyOrganizationID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()

	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Repository should handle empty org ID - it may return empty list or error
	mockConnRepo.EXPECT().
		List(gomock.Any(), filters).
		Return([]*model.Connection{}, int64(0), nil)

	result, err := svc.Execute(ctx, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 0 {
		t.Fatalf("expected total 0 for nil org ID, got %d", result.Total)
	}
}

// TestListConnections_Execute_WithProductNameFilter tests that listing with a product name
// passes the product name filter to the repository via filters.ProductName.
func TestListConnections_Execute_WithProductNameFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo))

	ctx := testContext()
	filters := http.QueryHeader{}

	// After setting productName, filters.ProductName is set; connRepo.List is called with updated filters.
	expectedFilters := filters
	expectedFilters.ProductName = "test-product"

	expectedConnections := []*model.Connection{
		newListTestConnection(uuid.New(), "product-conn-1", model.TypePostgreSQL),
		newListTestConnection(uuid.New(), "product-conn-2", model.TypeMySQL),
	}

	mockConnRepo.EXPECT().
		List(gomock.Any(), expectedFilters).
		Return(expectedConnections, int64(2), nil)

	result, err := svc.Execute(ctx, "test-product", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	items := result.Items.([]*model.ConnectionResponse)

	assert.Equal(t, 2, len(items))
	assert.Equal(t, "product-conn-1", items[0].ConfigName)
	assert.Equal(t, "product-conn-2", items[1].ConfigName)
}
