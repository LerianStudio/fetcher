//go:build e2e

package extraction

// S3 storage verification tests.
//
// These tests require E2E_ENABLE_S3=true. They validate that extraction results
// are correctly persisted to S3-compatible object storage (MinIO in the test environment).
//
// Run:
//
//	E2E_ENABLE_S3=true go test -v -tags e2e ./tests/e2e -run TestS3Storage -timeout 10m
//
// When E2E_ENABLE_S3 is not set, all tests in this file are skipped.

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// s3ResultPath is the expected prefix for result paths when using the S3 storage provider.
// The Worker stores objects under /{bucket}/{jobID}.json, where bucket = constant.ExternalDataBucketName.
var s3ResultPathPrefix = "/" + constant.ExternalDataBucketName + "/"

// skipIfS3NotEnabled skips the test when E2E_ENABLE_S3 is not set (coreInfra.Minio is nil).
func skipIfS3NotEnabled(t *testing.T) {
	t.Helper()

	if coreInfra.Minio == nil {
		t.Skip("S3 storage tests skipped: set E2E_ENABLE_S3=true to enable")
	}
}

// s3ObjectKeyFromResultPath extracts the S3 object key from the result path returned by the API.
// The result path format is "/{bucket}/{jobID}.json"; the key is the part after the bucket prefix.
// Example: "/external-data/550e8400-xxxx.json" → "550e8400-xxxx.json"
func s3ObjectKeyFromResultPath(resultPath string) string {
	return strings.TrimPrefix(resultPath, s3ResultPathPrefix)
}

// TestS3Storage_JobCompletesSuccessfully validates the full extraction workflow when the
// Worker is configured with STORAGE_PROVIDER=s3. The job must complete successfully and
// return a non-empty result_path in the expected bucket.
//
// This test acts as a smoke test: if S3 is misconfigured (wrong endpoint, bad credentials,
// bucket missing), the Worker will fail to upload and the job will move to "failed" status.
func TestS3Storage_JobCompletesSuccessfully(t *testing.T) {
	t.Parallel()

	skipIfS3NotEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()
	connName := fmt.Sprintf("e2e-s3-smoke-%s", uuid.New().String()[:8])

	conn := e2eshared.CreateTestConnection(t, apiClient, ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   connName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				connName: {
					"transactions": {"id", "amount", "status"},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "s3-smoke",
		},
	})
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created job: %s", jobID)

	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, "completed", jobResult.Status, "job status should be completed")
	assert.NotEmpty(t, jobResult.ResultPath, "result path should be set")
	assert.True(t,
		strings.HasPrefix(jobResult.ResultPath, s3ResultPathPrefix),
		"result path %q should start with %q", jobResult.ResultPath, s3ResultPathPrefix,
	)

	t.Logf("Job completed: status=%s resultPath=%s", jobResult.Status, jobResult.ResultPath)
}

// TestS3Storage_ObjectExistsInBucket verifies that after a job completes the result
// object actually exists in the MinIO bucket, using HeadObject to confirm the object
// is addressable at the expected key.
func TestS3Storage_ObjectExistsInBucket(t *testing.T) {
	t.Parallel()

	skipIfS3NotEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()
	connName := fmt.Sprintf("e2e-s3-exists-%s", uuid.New().String()[:8])

	conn := e2eshared.CreateTestConnection(t, apiClient, ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   connName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				connName: {
					"transactions": {"id", "amount", "status"},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "s3-exists",
		},
	})
	require.NoError(t, err, "create fetcher job")

	jobResult := e2eshared.AssertJobCompleted(t, apiClient, fetcherResp.JobID.String(), e2eshared.DefaultJobTimeout)
	require.Equal(t, "completed", jobResult.Status, "job must complete before S3 verification")
	require.NotEmpty(t, jobResult.ResultPath, "result path must be set to locate the S3 object")

	objectKey := s3ObjectKeyFromResultPath(jobResult.ResultPath)
	t.Logf("Verifying S3 object exists: bucket=%s key=%s", coreInfra.Minio.Bucket(), objectKey)

	err = coreInfra.Minio.HeadObject(ctx, objectKey)
	assert.NoError(t, err, "object %q should exist in S3 bucket %q after job completion", objectKey, coreInfra.Minio.Bucket())
}

// TestS3Storage_ObjectHasContent verifies that the stored object is non-empty.
// The Worker encrypts data before uploading, so this test does not attempt to
// decrypt or parse the content — it only confirms that bytes were written.
func TestS3Storage_ObjectHasContent(t *testing.T) {
	t.Parallel()

	skipIfS3NotEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()
	connName := fmt.Sprintf("e2e-s3-content-%s", uuid.New().String()[:8])

	conn := e2eshared.CreateTestConnection(t, apiClient, ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   connName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				connName: {
					"transactions": {"id", "account_id", "amount", "currency", "type", "created_at"},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "s3-content",
		},
	})
	require.NoError(t, err, "create fetcher job")

	jobResult := e2eshared.AssertJobCompleted(t, apiClient, fetcherResp.JobID.String(), e2eshared.DefaultJobTimeout)
	require.Equal(t, "completed", jobResult.Status)
	require.NotEmpty(t, jobResult.ResultPath)

	objectKey := s3ObjectKeyFromResultPath(jobResult.ResultPath)

	data, err := coreInfra.Minio.GetObject(ctx, objectKey)
	require.NoError(t, err, "should be able to download object %q from S3", objectKey)

	assert.Greater(t, len(data), 0,
		"S3 object %q should contain non-zero bytes (encrypted extraction result)", objectKey)

	t.Logf("S3 object verified: key=%s size=%d bytes", objectKey, len(data))
}

// TestS3Storage_ObjectKeyMatchesResultPath confirms the relationship between the
// result_path returned by the API and the actual S3 object key.
//
// The Worker stores the object at key="{jobID}.json" (no key prefix in the test env)
// and records result_path="/{bucket}/{jobID}.json" in MongoDB. The test verifies both:
//  1. The result_path format conforms to the expected pattern
//  2. The derived key corresponds to an existing S3 object
func TestS3Storage_ObjectKeyMatchesResultPath(t *testing.T) {
	t.Parallel()

	skipIfS3NotEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()
	connName := fmt.Sprintf("e2e-s3-key-%s", uuid.New().String()[:8])

	conn := e2eshared.CreateTestConnection(t, apiClient, ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   connName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				connName: {
					"transactions": {"id", "amount"},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "s3-key-match",
		},
	})
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()

	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)
	require.Equal(t, "completed", jobResult.Status)

	// Verify result_path format: "/{bucket}/{jobID}.json"
	expectedResultPath := fmt.Sprintf("/%s/%s.json", constant.ExternalDataBucketName, jobID)
	assert.Equal(t, expectedResultPath, jobResult.ResultPath,
		"result_path should be /%s/{jobID}.json", constant.ExternalDataBucketName)

	// Derive object key and confirm it exists in S3
	expectedKey := fmt.Sprintf("%s.json", jobID)
	objectKey := s3ObjectKeyFromResultPath(jobResult.ResultPath)
	assert.Equal(t, expectedKey, objectKey,
		"object key derived from result_path should be {jobID}.json")

	err = coreInfra.Minio.HeadObject(ctx, objectKey)
	assert.NoError(t, err, "object %q should exist in S3 bucket", objectKey)

	t.Logf("Key validation passed: resultPath=%s objectKey=%s", jobResult.ResultPath, objectKey)
}

// TestS3Storage_MultipleJobsStoredSeparately validates that concurrent extractions
// produce distinct S3 objects — each job has its own key derived from its job ID.
func TestS3Storage_MultipleJobsStoredSeparately(t *testing.T) {
	t.Parallel()

	skipIfS3NotEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	const jobCount = 2
	type jobEntry struct {
		jobID      string
		objectKey  string
		resultPath string
	}

	entries := make([]jobEntry, jobCount)

	for i := range jobCount {
		productName := e2eshared.GenerateProductName()
		connName := fmt.Sprintf("e2e-s3-multi-%d-%s", i, uuid.New().String()[:8])

		conn := e2eshared.CreateTestConnection(t, apiClient, ctx, productName, e2eshared.ConnectionInput{
			ConfigName:   connName,
			Type:         e2eshared.DBTypePostgreSQL,
			Host:         pgHost,
			Port:         pgPort,
			DatabaseName: "testdb",
			Username:     "testuser",
			Password:     "testpass",
		})

		err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
		require.NoError(t, err, "wait for connection %d", i)

		fetcherResp, err := apiClient.CreateFetcherJob(ctx, model.FetcherRequest{
			DataRequest: model.DataRequest{
				MappedFields: map[string]map[string][]string{
					connName: {
						"transactions": {"id", "amount"},
					},
				},
			},
			Metadata: map[string]any{
				"source": productName,
				"test":   fmt.Sprintf("s3-multi-%d", i),
			},
		})
		require.NoError(t, err, "create fetcher job %d", i)

		entries[i].jobID = fetcherResp.JobID.String()
	}

	// Wait for all jobs to complete and collect object keys
	for i := range jobCount {
		jobResult := e2eshared.AssertJobCompleted(t, apiClient, entries[i].jobID, e2eshared.DefaultJobTimeout)
		require.Equal(t, "completed", jobResult.Status, "job %d should complete", i)
		require.NotEmpty(t, jobResult.ResultPath, "job %d should have result path", i)

		entries[i].resultPath = jobResult.ResultPath
		entries[i].objectKey = s3ObjectKeyFromResultPath(jobResult.ResultPath)
	}

	// Assert all keys are distinct and all objects exist in S3
	keysSeen := make(map[string]bool, jobCount)

	for i, e := range entries {
		assert.False(t, keysSeen[e.objectKey],
			"object key %q for job %d is a duplicate (jobs must have distinct S3 keys)", e.objectKey, i)

		keysSeen[e.objectKey] = true

		err := coreInfra.Minio.HeadObject(ctx, e.objectKey)
		assert.NoError(t, err, "job %d object %q should exist in S3", i, e.objectKey)
	}

	t.Logf("Multiple jobs stored separately: %d distinct S3 objects verified", jobCount)
}
