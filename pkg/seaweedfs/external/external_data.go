package external

import (
	"context"
	"fmt"

	"github.com/LerianStudio/lib-observability"

	portStorage "github.com/LerianStudio/fetcher/pkg/ports/storage"
	"github.com/LerianStudio/fetcher/pkg/seaweedfs"
	tms3 "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/s3"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// Repository is an alias for the port interface.
// The canonical definition lives in pkg/ports/storage.
type Repository = portStorage.Repository

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
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "seaweedfs.external_data.get")
	defer span.End()

	// Apply tenant-prefixed key (multi-tenant: "{tenantId}/{objectName}", single-tenant: "{objectName}")
	tenantObjectName, err := tms3.GetS3KeyStorageContext(ctx, objectName)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to resolve tenant object key", err)
		return nil, fmt.Errorf("tenant object key for %s: %w", objectName, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("seaweedfs.object_name", tenantObjectName),
	)

	// Add .json extension for external data
	path := fmt.Sprintf("/%s/%s.json", repo.bucket, tenantObjectName)

	data, err := repo.client.DownloadFile(ctx, path)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to download file from SeaweedFS", err)
		return nil, fmt.Errorf("seaweedfs download failed for %s: %w", objectName, err)
	}

	span.SetAttributes(attribute.Int("seaweedfs.response_size", len(data)))

	return data, nil
}

// Put uploads data to the SeaweedFS storage with the given object name and content type.
func (repo *SimpleRepository) Put(ctx context.Context, objectName string, data []byte) error {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "seaweedfs.external_data.put")
	defer span.End()

	// Apply tenant-prefixed key (multi-tenant: "{tenantId}/{objectName}", single-tenant: "{objectName}")
	tenantObjectName, err := tms3.GetS3KeyStorageContext(ctx, objectName)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to resolve tenant object key", err)
		return fmt.Errorf("tenant object key for %s: %w", objectName, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("seaweedfs.object_name", tenantObjectName),
		attribute.Int("seaweedfs.data_size", len(data)),
	)

	path := fmt.Sprintf("/%s/%s", repo.bucket, tenantObjectName)

	err = repo.client.UploadFile(ctx, path, data)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to upload file to SeaweedFS", err)
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Error communicating with SeaweedFS: %v", err))

		return fmt.Errorf("seaweedfs upload failed for %s: %w", objectName, err)
	}

	return nil
}
