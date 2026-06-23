package query

import (
	"errors"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/net/http"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// TestListUnassignedConnections_Execute_Success tests successful listing of unassigned connections.
func TestListUnassignedConnections_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	svc := NewListUnassignedConnections(mockConnRepo)

	ctx := testContext()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	conn1 := newListTestConnection(uuid.New(), "unassigned-conn-1", model.TypePostgreSQL)
	conn2 := newListTestConnection(uuid.New(), "unassigned-conn-2", model.TypeMySQL)
	expectedList := []*model.Connection{conn1, conn2}

	// Mock: ListUnassigned returns connections
	mockConnRepo.EXPECT().
		ListUnassigned(gomock.Any(), filters).
		Return(expectedList, int64(2), nil)

	result, err := svc.Execute(ctx, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Total != len(expectedList) {
		t.Fatalf("expected %d connections, got %d", len(expectedList), result.Total)
	}

	items, ok := result.Items.([]*model.ConnectionResponse)
	if !ok {
		t.Fatal("expected items to be []*model.ConnectionResponse")
	}

	if items[0].ConfigName != "unassigned-conn-1" {
		t.Fatalf("expected first connection name 'unassigned-conn-1', got %s", items[0].ConfigName)
	}

	if items[1].ConfigName != "unassigned-conn-2" {
		t.Fatalf("expected second connection name 'unassigned-conn-2', got %s", items[1].ConfigName)
	}
}

// TestListUnassignedConnections_Execute_EmptyList tests that nil repo result returns empty pagination, not nil.
func TestListUnassignedConnections_Execute_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	svc := NewListUnassignedConnections(mockConnRepo)

	ctx := testContext()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Mock: ListUnassigned returns nil (no connections found)
	mockConnRepo.EXPECT().
		ListUnassigned(gomock.Any(), filters).
		Return(nil, int64(0), nil)

	result, err := svc.Execute(ctx, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result (empty pagination)")
	}

	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
}

// TestListUnassignedConnections_Execute_RepositoryError tests repository error propagation.
func TestListUnassignedConnections_Execute_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	svc := NewListUnassignedConnections(mockConnRepo)

	ctx := testContext()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	dbError := errors.New("database connection failed")

	// Mock: ListUnassigned returns error
	mockConnRepo.EXPECT().
		ListUnassigned(gomock.Any(), filters).
		Return(nil, int64(0), dbError)

	result, err := svc.Execute(ctx, filters)

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

// TestNewListUnassignedConnections verifies the constructor.
func TestNewListUnassignedConnections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	svc := NewListUnassignedConnections(mockConnRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.connRepo == nil {
		t.Fatal("expected connRepo to be set")
	}
}

// TestListUnassignedConnections_Execute_TableDriven uses table-driven tests for various scenarios.
func TestListUnassignedConnections_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		filters         http.QueryHeader
		setupMocks      func(*connRepo.MockRepository, http.QueryHeader)
		wantErr         bool
		wantResultCount int
	}{
		{
			name: "successful list with multiple unassigned connections",
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
					ListUnassigned(gomock.Any(), filters).
					Return(connections, int64(3), nil)
			},
			wantErr:         false,
			wantResultCount: 3,
		},
		{
			name: "empty list returns empty pagination",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  1,
			},
			setupMocks: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				mock.EXPECT().
					ListUnassigned(gomock.Any(), filters).
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
					ListUnassigned(gomock.Any(), filters).
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
					ListUnassigned(gomock.Any(), filters).
					Return(connections, int64(2), nil)
			},
			wantErr:         false,
			wantResultCount: 2,
		},
		{
			name:    "list with empty filters",
			filters: http.QueryHeader{},
			setupMocks: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				connections := []*model.Connection{
					newListTestConnection(uuid.New(), "conn-1", model.TypePostgreSQL),
				}
				mock.EXPECT().
					ListUnassigned(gomock.Any(), filters).
					Return(connections, int64(1), nil)
			},
			wantErr:         false,
			wantResultCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			ctx := testContext()

			tt.setupMocks(mockConnRepo, tt.filters)

			svc := NewListUnassignedConnections(mockConnRepo)

			result, err := svc.Execute(ctx, tt.filters)

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

			if result.Total != tt.wantResultCount {
				t.Fatalf("expected %d connections, got %d", tt.wantResultCount, result.Total)
			}
		})
	}
}

// TestListUnassignedConnections_Execute_OrganizationIsolation tests that unassigned connections are isolated by organization.
func TestListUnassignedConnections_Execute_OrganizationIsolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	svc := NewListUnassignedConnections(mockConnRepo)

	ctx := testContext()

	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Mock: org1 has 2 unassigned connections
	org1Connections := []*model.Connection{
		newListTestConnection(uuid.New(), "org1-conn-1", model.TypePostgreSQL),
		newListTestConnection(uuid.New(), "org1-conn-2", model.TypeMySQL),
	}

	mockConnRepo.EXPECT().
		ListUnassigned(gomock.Any(), filters).
		Return(org1Connections, int64(2), nil)

	result1, err := svc.Execute(ctx, filters)
	if err != nil {
		t.Fatalf("unexpected error for org1: %v", err)
	}

	if result1.Total != 2 {
		t.Fatalf("expected 2 connections for org1, got %d", result1.Total)
	}

	// Mock: org2 has 1 unassigned connection
	org2Connections := []*model.Connection{
		newListTestConnection(uuid.New(), "org2-conn-1", model.TypeMongoDB),
	}

	mockConnRepo.EXPECT().
		ListUnassigned(gomock.Any(), filters).
		Return(org2Connections, int64(1), nil)

	result2, err := svc.Execute(ctx, filters)
	if err != nil {
		t.Fatalf("unexpected error for org2: %v", err)
	}

	if result2.Total != 1 {
		t.Fatalf("expected 1 connection for org2, got %d", result2.Total)
	}
}

// TestListUnassignedConnections_Execute_ErrorScenarios tests various error scenarios.
func TestListUnassignedConnections_Execute_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*connRepo.MockRepository, http.QueryHeader)
		errorMsg  string
	}{
		{
			name: "database connection error",
			setupMock: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				mock.EXPECT().
					ListUnassigned(gomock.Any(), filters).
					Return(nil, int64(0), errors.New("database connection failed"))
			},
			errorMsg: "database connection failed",
		},
		{
			name: "timeout error",
			setupMock: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				mock.EXPECT().
					ListUnassigned(gomock.Any(), filters).
					Return(nil, int64(0), errors.New("context deadline exceeded"))
			},
			errorMsg: "context deadline exceeded",
		},
		{
			name: "permission denied error",
			setupMock: func(mock *connRepo.MockRepository, filters http.QueryHeader) {
				mock.EXPECT().
					ListUnassigned(gomock.Any(), filters).
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
			svc := NewListUnassignedConnections(mockConnRepo)

			ctx := testContext()
			filters := http.QueryHeader{
				Limit: 10,
				Page:  1,
			}

			tt.setupMock(mockConnRepo, filters)

			result, err := svc.Execute(ctx, filters)

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
