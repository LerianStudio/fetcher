package mocks

import (
	"context"

	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	jobRepo "github.com/LerianStudio/fetcher/v2/pkg/mongodb/job"
	"github.com/google/uuid"
)

// Compile-time interface compliance verification
var (
	_ crypto.Cryptor     = (*MockCryptor)(nil)
	_ jobRepo.Repository = (*MockJobRepository)(nil)
)

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
	FindByIDFunc          func(ctx context.Context, id uuid.UUID) (*model.Connection, error)
	FindByConfigNamesFunc func(ctx context.Context, names []string) ([]*model.Connection, error)
	CreateFunc            func(ctx context.Context, conn *model.Connection) error
	UpdateFunc            func(ctx context.Context, conn *model.Connection) error
	DeleteFunc            func(ctx context.Context, id uuid.UUID) error
	ListFunc              func(ctx context.Context, params any) ([]*model.Connection, error)
}

func (m *MockConnectionRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Connection, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}

	return nil, nil
}

func (m *MockConnectionRepository) FindByConfigNames(ctx context.Context, names []string) ([]*model.Connection, error) {
	if m.FindByConfigNamesFunc != nil {
		return m.FindByConfigNamesFunc(ctx, names)
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

func (m *MockConnectionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}

	return nil
}

func (m *MockConnectionRepository) List(ctx context.Context, params any) ([]*model.Connection, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}

	return nil, nil
}

// MockJobRepository implements job.Repository for testing
type MockJobRepository struct {
	CreateFunc                        func(ctx context.Context, job *model.Job) (*model.Job, error)
	UpdateFunc                        func(ctx context.Context, job *model.Job) (*model.Job, error)
	UpdateStatusFunc                  func(ctx context.Context, id uuid.UUID, status model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error
	FindByIDFunc                      func(ctx context.Context, id uuid.UUID) (*model.Job, error)
	FindByRequestHashWithinWindowFunc func(ctx context.Context, requestHash string, windowMinutes int) (*model.Job, error)
	FindActiveByRequestHashFunc       func(ctx context.Context, requestHash string) (*model.Job, error)
	ListFunc                          func(ctx context.Context, filters *jobRepo.ListFilter) ([]*model.Job, error)
	ExistsRunningByMappedFieldKeyFunc func(ctx context.Context, keyPattern string) (bool, error)
}

func (m *MockJobRepository) Create(ctx context.Context, job *model.Job) (*model.Job, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, job)
	}

	return job, nil
}

func (m *MockJobRepository) Update(ctx context.Context, job *model.Job) (*model.Job, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, job)
	}

	return job, nil
}

func (m *MockJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status, resultPath, resultHMAC, metadata)
	}

	return nil
}

func (m *MockJobRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Job, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}

	return nil, nil
}

func (m *MockJobRepository) FindByRequestHashWithinWindow(ctx context.Context, requestHash string, windowMinutes int) (*model.Job, error) {
	if m.FindByRequestHashWithinWindowFunc != nil {
		return m.FindByRequestHashWithinWindowFunc(ctx, requestHash, windowMinutes)
	}

	return nil, nil
}

func (m *MockJobRepository) FindActiveByRequestHash(ctx context.Context, requestHash string) (*model.Job, error) {
	if m.FindActiveByRequestHashFunc != nil {
		return m.FindActiveByRequestHashFunc(ctx, requestHash)
	}

	return nil, nil
}

func (m *MockJobRepository) List(ctx context.Context, filters *jobRepo.ListFilter) ([]*model.Job, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filters)
	}

	return nil, nil
}

func (m *MockJobRepository) ExistsRunningByMappedFieldKey(ctx context.Context, keyPattern string) (bool, error) {
	if m.ExistsRunningByMappedFieldKeyFunc != nil {
		return m.ExistsRunningByMappedFieldKeyFunc(ctx, keyPattern)
	}

	return false, nil
}
