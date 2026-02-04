package mocks

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/google/uuid"
)

// Compile-time interface compliance verification
var _ crypto.Cryptor = (*MockCryptor)(nil)

// MockCryptor implements crypto.Cryptor for testing
type MockCryptor struct {
	EncryptFunc    func(ctx context.Context, plaintext string) (string, string, error)
	DecryptFunc    func(ctx context.Context, ciphertext, keyVersion string) (string, error)
	KeyVersionFunc func() string
}

func (m *MockCryptor) Encrypt(ctx context.Context, plaintext string) (string, string, error) {
	if m.EncryptFunc != nil {
		return m.EncryptFunc(ctx, plaintext)
	}

	return "encrypted-" + plaintext, "v1", nil
}

func (m *MockCryptor) Decrypt(ctx context.Context, ciphertext, keyVersion string) (string, error) {
	if m.DecryptFunc != nil {
		return m.DecryptFunc(ctx, ciphertext, keyVersion)
	}

	return "decrypted", nil
}

func (m *MockCryptor) KeyVersion() string {
	if m.KeyVersionFunc != nil {
		return m.KeyVersionFunc()
	}

	return "v1"
}

// MockConnectionRepository implements connection.Repository for testing
type MockConnectionRepository struct {
	FindByIDFunc          func(ctx context.Context, id, orgID uuid.UUID) (*model.Connection, error)
	FindByConfigNamesFunc func(ctx context.Context, orgID uuid.UUID, names []string) ([]*model.Connection, error)
	CreateFunc            func(ctx context.Context, conn *model.Connection) error
	UpdateFunc            func(ctx context.Context, conn *model.Connection) error
	DeleteFunc            func(ctx context.Context, id, orgID uuid.UUID) error
	ListFunc              func(ctx context.Context, orgID uuid.UUID, params any) ([]*model.Connection, error)
}

func (m *MockConnectionRepository) FindByID(ctx context.Context, id, orgID uuid.UUID) (*model.Connection, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id, orgID)
	}

	return nil, nil
}

func (m *MockConnectionRepository) FindByConfigNames(ctx context.Context, orgID uuid.UUID, names []string) ([]*model.Connection, error) {
	if m.FindByConfigNamesFunc != nil {
		return m.FindByConfigNamesFunc(ctx, orgID, names)
	}

	return nil, nil
}

func (m *MockConnectionRepository) Create(ctx context.Context, conn *model.Connection) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, conn)
	}

	return nil
}

func (m *MockConnectionRepository) Update(ctx context.Context, conn *model.Connection) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, conn)
	}

	return nil
}

func (m *MockConnectionRepository) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id, orgID)
	}

	return nil
}

func (m *MockConnectionRepository) List(ctx context.Context, orgID uuid.UUID, params any) ([]*model.Connection, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, orgID, params)
	}

	return nil, nil
}

// MockJobRepository implements job.Repository for testing
type MockJobRepository struct {
	FindByIDFunc     func(ctx context.Context, id, orgID uuid.UUID) (*model.Job, error)
	CreateFunc       func(ctx context.Context, job *model.Job) error
	UpdateStatusFunc func(ctx context.Context, id, orgID uuid.UUID, status model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error
}

func (m *MockJobRepository) FindByID(ctx context.Context, id, orgID uuid.UUID) (*model.Job, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id, orgID)
	}

	return nil, nil
}

func (m *MockJobRepository) Create(ctx context.Context, job *model.Job) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, job)
	}

	return nil
}

func (m *MockJobRepository) UpdateStatus(ctx context.Context, id, orgID uuid.UUID, status model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, orgID, status, resultPath, resultHMAC, metadata)
	}

	return nil
}
