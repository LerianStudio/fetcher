package external

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/seaweedfs"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
)

// Repository provides an interface for SeaweedFS storage operations
//
//go:generate mockgen --destination=repository.mock.go --package=external . Repository
type Repository interface {
	Get(ctx context.Context, objectName string) ([]byte, error)
	Put(ctx context.Context, objectName string, data []byte) error
}

// SimpleRepository provides access to SeaweedFS storage for file operations using direct HTTP.
type SimpleRepository struct {
	client seaweedfs.Client
	bucket string
}

// NewSimpleRepository creates a new instance of SimpleRepository with the given HTTP client and bucket name.
func NewSimpleRepository(client seaweedfs.Client, bucket string) *SimpleRepository {
	return &SimpleRepository{
		client: client,
		bucket: bucket,
	}
}

// Get the content of an external JSON file from the SeaweedFS storage.
// A .json extension is appended to the objectName for retrieval.
func (repo *SimpleRepository) Get(ctx context.Context, objectName string) ([]byte, error) {
	// Add .json extension for external data
	path := fmt.Sprintf("/%s/%s.json", repo.bucket, objectName)

	data, err := repo.client.DownloadFile(ctx, path)
	if err != nil {
		return nil, pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	return data, nil
}

// Put uploads data to the SeaweedFS storage with the given object name and content type.
func (repo *SimpleRepository) Put(ctx context.Context, objectName string, data []byte) error {
	logger := pkg.NewLoggerFromContext(ctx)

	path := fmt.Sprintf("/%s/%s", repo.bucket, objectName)

	err := repo.client.UploadFile(ctx, path, data)
	if err != nil {
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error communicating with SeaweedFS: %v", err))
		return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	return nil
}
