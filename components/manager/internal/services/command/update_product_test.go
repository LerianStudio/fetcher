package command

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	productMock "github.com/LerianStudio/fetcher/pkg/mongodb/product"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newExistingProductForUpdate creates a valid existing Product for testing updates,
// including metadata and a description. Reuses the same shape as newExistingProduct
// from assign_connection_test.go but adds fields relevant to update tests.
func newExistingProductForUpdate(orgID, productID uuid.UUID) *model.Product {
	return &model.Product{
		ID:             productID,
		OrganizationID: orgID,
		Code:           "test-product",
		Name:           "Test Product",
		Description:    "A test product for unit tests",
		Metadata:       &map[string]any{"env": "test"},
		CreatedAt:      time.Now().UTC().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().UTC().Add(-1 * time.Hour),
	}
}

// TestNewUpdateProduct verifies the constructor wires the repository.
func TestNewUpdateProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewUpdateProduct(mockRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.productRepo == nil {
		t.Fatal("expected productRepo to be set")
	}
}

// TestUpdateProduct_Execute_SuccessUpdateName tests a successful update of the name field.
func TestUpdateProduct_Execute_SuccessUpdateName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewUpdateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()
	existing := newExistingProductForUpdate(orgID, productID)

	newName := "Updated Product Name"
	input := model.ProductUpdateInput{
		Name: &newName,
	}

	// Mock: find existing product
	mockRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existing, nil)

	// Mock: update returns the updated product
	mockRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, p *model.Product) (*model.Product, error) {
			return p, nil
		})

	result, err := svc.Execute(ctx, orgID, productID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	assert.Equal(t, newName, result.Name, "Name should be updated")
	assert.Equal(t, "test-product", result.Code, "Code should remain unchanged")
	assert.Equal(t, "A test product for unit tests", result.Description, "Description should remain unchanged")
}

// TestUpdateProduct_Execute_SuccessUpdateDescription tests a partial update with only description.
func TestUpdateProduct_Execute_SuccessUpdateDescription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewUpdateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()
	existing := newExistingProductForUpdate(orgID, productID)

	originalName := existing.Name
	newDesc := "Updated description for the product"
	input := model.ProductUpdateInput{
		Description: &newDesc,
	}

	// Mock: find existing product
	mockRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existing, nil)

	// Mock: update returns the updated product
	mockRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, p *model.Product) (*model.Product, error) {
			return p, nil
		})

	result, err := svc.Execute(ctx, orgID, productID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	assert.Equal(t, newDesc, result.Description, "Description should be updated")
	assert.Equal(t, originalName, result.Name, "Name should remain unchanged")
	assert.Equal(t, "test-product", result.Code, "Code should remain unchanged")
}

// TestUpdateProduct_Execute_SuccessUpdateMetadata tests a partial update with metadata map.
func TestUpdateProduct_Execute_SuccessUpdateMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewUpdateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()
	existing := newExistingProductForUpdate(orgID, productID)

	originalName := existing.Name
	originalDesc := existing.Description
	newMetadata := map[string]any{"region": "us-east-1", "tier": "premium"}
	input := model.ProductUpdateInput{
		Metadata: &newMetadata,
	}

	// Mock: find existing product
	mockRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existing, nil)

	// Mock: update returns the updated product
	mockRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, p *model.Product) (*model.Product, error) {
			return p, nil
		})

	result, err := svc.Execute(ctx, orgID, productID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	assert.Equal(t, &newMetadata, result.Metadata, "Metadata should be updated")
	assert.Equal(t, originalName, result.Name, "Name should remain unchanged")
	assert.Equal(t, originalDesc, result.Description, "Description should remain unchanged")
}

// TestUpdateProduct_Execute_ProductNotFound tests that FindByID returning (nil, nil) yields ErrEntityNotFound.
func TestUpdateProduct_Execute_ProductNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewUpdateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	newName := "Does Not Matter"
	input := model.ProductUpdateInput{
		Name: &newName,
	}

	// Mock: product not found
	mockRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, productID, input)

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

	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}

// TestUpdateProduct_Execute_FindByIDRepoError tests that a repository error from FindByID is propagated.
func TestUpdateProduct_Execute_FindByIDRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewUpdateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	newName := "Does Not Matter"
	input := model.ProductUpdateInput{
		Name: &newName,
	}

	dbError := errors.New("database connection failed")

	// Mock: database error during lookup
	mockRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, productID, input)

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

// TestUpdateProduct_Execute_UpdateRepoError tests that a repository error from Update is propagated.
func TestUpdateProduct_Execute_UpdateRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewUpdateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()
	existing := newExistingProductForUpdate(orgID, productID)

	newName := "Valid Name"
	input := model.ProductUpdateInput{
		Name: &newName,
	}

	dbError := errors.New("failed to update in database")

	// Mock: find existing product
	mockRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existing, nil)

	// Mock: update returns error
	mockRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, productID, input)

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

// TestUpdateProduct_Execute_UpdateReturnsNotFound tests that Update returning a not-found error
// (e.g. product soft-deleted between read and write) is propagated correctly.
func TestUpdateProduct_Execute_UpdateReturnsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewUpdateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()
	existing := newExistingProductForUpdate(orgID, productID)

	newName := "Valid Name"
	input := model.ProductUpdateInput{
		Name: &newName,
	}

	notFoundErr := pkg.ValidateBusinessError(constant.ErrEntityNotFound, "product")

	// Mock: find existing product
	mockRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existing, nil)

	// Mock: update returns not-found error (product soft-deleted between read and write)
	mockRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil, notFoundErr)

	result, err := svc.Execute(ctx, orgID, productID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for not-found update result, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}

	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}

// TestUpdateProduct_Execute_ValidationFailureEmptyName tests that patching with an empty name fails validation.
func TestUpdateProduct_Execute_ValidationFailureEmptyName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewUpdateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()
	existing := newExistingProductForUpdate(orgID, productID)

	emptyName := ""
	input := model.ProductUpdateInput{
		Name: &emptyName,
	}

	// Mock: find existing product
	mockRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existing, nil)

	// Update should NOT be called because ApplyPatch -> IsValid should fail
	result, err := svc.Execute(ctx, orgID, productID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected validation error for empty name, got nil")
	}

	// The error should be a ValidationKnownFieldsError (missing required field: name)
	var validationErr pkg.ValidationKnownFieldsError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationKnownFieldsError, got %T: %v", err, err)
	}
}

// TestUpdateProduct_Execute_ApplyPatchEmptyName verifies that ApplyPatch does NOT allow the Name field
// to be cleared via an empty string. When a patch provides Name="" (non-nil pointer to empty string),
// ApplyPatch applies it but IsValid rejects the result because Name is a required field.
// This ensures partial updates cannot accidentally erase required fields.
func TestUpdateProduct_Execute_ApplyPatchEmptyName(t *testing.T) {
	orgID := uuid.New()
	productID := uuid.New()
	existing := newExistingProductForUpdate(orgID, productID)

	originalName := existing.Name

	emptyName := ""
	err := existing.ApplyPatch(&emptyName, nil, nil)

	if err == nil {
		t.Fatal("expected validation error when patching Name to empty string, got nil")
	}

	// After the failed patch, the Name field was set to "" internally,
	// but the validation error prevents the caller from persisting the change.
	// Verify the error is a validation error for the "name" required field.
	var validationErr pkg.ValidationKnownFieldsError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationKnownFieldsError, got %T: %v", err, err)
	}

	if _, exists := validationErr.Fields["name"]; !exists {
		t.Fatalf("expected 'name' in validation error fields, got %v", validationErr.Fields)
	}

	// Confirm that a non-empty name still works (sanity check)
	existing.Name = originalName
	validName := "Valid Updated Name"
	err = existing.ApplyPatch(&validName, nil, nil)
	if err != nil {
		t.Fatalf("expected no error for valid name patch, got %v", err)
	}

	assert.Equal(t, validName, existing.Name, "Name should be updated when a valid value is provided")
}

// TestUpdateProduct_Execute_CodeImmutability verifies that ProductUpdateInput has no Code field (structural test).
func TestUpdateProduct_Execute_CodeImmutability(t *testing.T) {
	// Verify via reflection that ProductUpdateInput does NOT contain a Code field.
	// This ensures immutability of the Code at the DTO level.
	typ := reflect.TypeOf(model.ProductUpdateInput{})

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "Code" {
			t.Fatal("ProductUpdateInput must NOT contain a Code field; Code is immutable after creation")
		}
	}
}

// TestUpdateProduct_Execute_TableDriven uses table-driven tests for various scenarios.
func TestUpdateProduct_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*productMock.MockRepository, uuid.UUID, uuid.UUID, *model.Product)
		input          model.ProductUpdateInput
		wantErr        bool
		wantStatusCode int // 0 means no status code check
		validateResult func(*testing.T, *model.Product)
	}{
		{
			name: "successful update with all mutable fields",
			setupMocks: func(repo *productMock.MockRepository, orgID, productID uuid.UUID, existing *model.Product) {
				repo.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(existing, nil)
				repo.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, p *model.Product) (*model.Product, error) {
						return p, nil
					})
			},
			input: func() model.ProductUpdateInput {
				name := "All Fields Updated"
				desc := "New description"
				meta := map[string]any{"key": "value"}
				return model.ProductUpdateInput{
					Name:        &name,
					Description: &desc,
					Metadata:    &meta,
				}
			}(),
			wantErr: false,
			validateResult: func(t *testing.T, result *model.Product) {
				assert.Equal(t, "All Fields Updated", result.Name)
				assert.Equal(t, "New description", result.Description)
				assert.Equal(t, "test-product", result.Code, "Code must remain unchanged")
			},
		},
		{
			name: "product not found returns 404",
			setupMocks: func(repo *productMock.MockRepository, orgID, productID uuid.UUID, existing *model.Product) {
				repo.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(nil, nil)
			},
			input: func() model.ProductUpdateInput {
				name := "Any Name"
				return model.ProductUpdateInput{Name: &name}
			}(),
			wantErr:        true,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "FindByID error propagated",
			setupMocks: func(repo *productMock.MockRepository, orgID, productID uuid.UUID, existing *model.Product) {
				repo.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(nil, errors.New("connection refused"))
			},
			input: func() model.ProductUpdateInput {
				name := "Any Name"
				return model.ProductUpdateInput{Name: &name}
			}(),
			wantErr: true,
		},
		{
			name: "Update error propagated",
			setupMocks: func(repo *productMock.MockRepository, orgID, productID uuid.UUID, existing *model.Product) {
				repo.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(existing, nil)
				repo.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("write conflict"))
			},
			input: func() model.ProductUpdateInput {
				name := "Valid Name"
				return model.ProductUpdateInput{Name: &name}
			}(),
			wantErr: true,
		},
		{
			name: "Update returns not-found error yields 404",
			setupMocks: func(repo *productMock.MockRepository, orgID, productID uuid.UUID, existing *model.Product) {
				repo.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(existing, nil)
				repo.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil, pkg.ValidateBusinessError(constant.ErrEntityNotFound, "product"))
			},
			input: func() model.ProductUpdateInput {
				name := "Valid Name"
				return model.ProductUpdateInput{Name: &name}
			}(),
			wantErr:        true,
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := productMock.NewMockRepository(ctrl)

			ctx := testContext()
			orgID := uuid.New()
			productID := uuid.New()
			existing := newExistingProduct(orgID, productID)

			tt.setupMocks(mockRepo, orgID, productID, existing)

			svc := NewUpdateProduct(mockRepo)

			result, err := svc.Execute(ctx, orgID, productID, tt.input)

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

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}
