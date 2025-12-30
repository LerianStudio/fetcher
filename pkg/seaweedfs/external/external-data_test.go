package external

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/seaweedfs"
	gomock "github.com/golang/mock/gomock"
)

func TestNewSimpleRepository(t *testing.T) {
	client := &seaweedfs.SeaweedFSClient{}
	bucket := "test-bucket"

	repo := NewSimpleRepository(client, bucket)

	if repo == nil {
		t.Fatal("NewSimpleRepository() returned nil")
	}
	if repo.bucket != bucket {
		t.Errorf("bucket = %q, want %q", repo.bucket, bucket)
	}
	if repo.client != client {
		t.Error("client not set correctly")
	}
}

func TestSimpleRepository_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := seaweedfs.NewMockSeaweedFSClient(ctrl)
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		expectedData := []byte(`{"key":"value"}`)
		mockClient.EXPECT().
			DownloadFile(gomock.Any(), "/test-bucket/object-name.json").
			Return(expectedData, nil)

		repo := &SimpleRepository{
			client: mockClient,
			bucket: "test-bucket",
		}

		data, err := repo.Get(ctx, "object-name")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(data) != string(expectedData) {
			t.Errorf("Get() = %q, want %q", data, expectedData)
		}
	})

	t.Run("get with error", func(t *testing.T) {
		mockClient.EXPECT().
			DownloadFile(gomock.Any(), "/test-bucket/missing.json").
			Return(nil, errors.New("not found"))

		repo := &SimpleRepository{
			client: mockClient,
			bucket: "test-bucket",
		}

		_, err := repo.Get(ctx, "missing")
		if err == nil {
			t.Error("Get() expected error, got nil")
		}
	})
}

func TestSimpleRepository_Put(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := seaweedfs.NewMockSeaweedFSClient(ctrl)

	t.Run("successful put", func(t *testing.T) {
		data := []byte(`{"key":"value"}`)
		mockClient.EXPECT().
			UploadFile(gomock.Any(), "/test-bucket/new-object", data).
			Return(nil)

		repo := &SimpleRepository{
			client: mockClient,
			bucket: "test-bucket",
		}

		// Create context with logger
		ctx := context.Background()

		err := repo.Put(ctx, "new-object", data)
		if err != nil {
			t.Errorf("Put() error = %v", err)
		}
	})

	t.Run("put with error", func(t *testing.T) {
		data := []byte(`{"key":"value"}`)
		mockClient.EXPECT().
			UploadFile(gomock.Any(), "/test-bucket/failed-object", data).
			Return(errors.New("upload failed"))

		repo := &SimpleRepository{
			client: mockClient,
			bucket: "test-bucket",
		}

		ctx := context.Background()
		err := repo.Put(ctx, "failed-object", data)
		if err == nil {
			t.Error("Put() expected error, got nil")
		}
	})
}
