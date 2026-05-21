package in

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/components/manager/internal/services/query"
	"github.com/LerianStudio/fetcher/pkg/model"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/ports/messaging"

	libLog "github.com/LerianStudio/lib-observability/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.uber.org/mock/gomock"
)

// setupTestApp creates a Fiber app with test context middleware.
func setupTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024, // 10KB for test flexibility
	})

	// Middleware to inject test context with logger and tracer
	app.Use(func(c *fiber.Ctx) error {
		logger := &libLog.GoLogger{Level: libLog.LevelDebug}
		ctx := observability.ContextWithHeaderID(c.UserContext(), "test-request-id")
		ctx = observability.ContextWithLogger(ctx, logger)
		ctx = observability.ContextWithTracer(ctx, otel.Tracer("test"))
		c.SetUserContext(ctx)

		return c.Next()
	})

	return app
}

// createTestJob creates a test job with default values.
func createTestJob(id uuid.UUID) *model.Job {
	now := time.Now().UTC()
	return &model.Job{
		ID: id,
		MappedFields: map[string]map[string][]string{
			"ds1": {
				"table1": {"field1", "field2"},
			},
		},
		Metadata: map[string]any{
			"source":        "test-product",
			"correlationId": "test-123",
		},
		Status:      model.JobStatusPending,
		RequestHash: "test-hash-123",
		CreatedAt:   now,
	}
}

// validFetcherRequestPayload returns a valid FetcherRequest JSON payload.
// The handler requires metadata.source to be present.
func validFetcherRequestPayload() string {
	return `{
		"dataRequest": {
			"mappedFields": {
				"ds1": {
					"table1": ["field1", "field2"]
				}
			}
		},
		"metadata": {
			"source": "test-product",
			"correlationId": "test-123"
		}
	}`
}

// ============================================================================
// CreateJob Handler Tests - Validation (already use real handler)
// ============================================================================

func TestFetcherHandler_CreateJob_InvalidJSON(t *testing.T) {
	app := setupTestApp()

	// Create handler without actual service (we expect early failure)
	handler := &FetcherHandler{CreateJobCmd: nil}

	app.Post("/v1/fetcher", handler.CreateJob)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "invalid JSON - missing closing brace",
			body:     `{"dataRequest": {"mappedFields": {"ds1": {"t1": ["f1"]}}`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid JSON - syntax error",
			body:     `{invalid}`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid JSON - empty string",
			body:     ``,
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestFetcherHandler_CreateJob_ContentTypeValidation(t *testing.T) {
	app := setupTestApp()

	handler := &FetcherHandler{CreateJobCmd: nil}
	app.Post("/v1/fetcher", handler.CreateJob)

	tests := []struct {
		name        string
		contentType string
		body        string
		wantCode    int
	}{
		{
			name:        "valid JSON content type but invalid body",
			contentType: "application/json",
			body:        `{invalid`,
			wantCode:    fiber.StatusBadRequest,
		},
		{
			name:        "JSON with charset but invalid body",
			contentType: "application/json; charset=utf-8",
			body:        `{invalid`,
			wantCode:    fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

// TestFetcherHandler_CreateJob_MetadataSourceValidation tests required metadata.source validation.
func TestFetcherHandler_CreateJob_MetadataSourceValidation(t *testing.T) {
	app := setupTestApp()

	handler := &FetcherHandler{CreateJobCmd: nil}
	app.Post("/v1/fetcher", handler.CreateJob)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "missing metadata",
			body:     `{"dataRequest":{"mappedFields":{"ds1":{"t1":["f1"]}}}}`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "missing metadata source",
			body:     `{"dataRequest":{"mappedFields":{"ds1":{"t1":["f1"]}}},"metadata":{}}`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "whitespace metadata source",
			body:     `{"dataRequest":{"mappedFields":{"ds1":{"t1":["f1"]}}},"metadata":{"source":"   "}}`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "non-string metadata source",
			body:     `{"dataRequest":{"mappedFields":{"ds1":{"t1":["f1"]}}},"metadata":{"source":123}}`,
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

// createLargePayload generates a valid JSON payload of approximately the specified size.
func createLargePayload(targetSize int) []byte {
	// Build a JSON structure with enough data to exceed targetSize
	var sb strings.Builder
	sb.WriteString(`{"dataRequest":{"mappedFields":{`)

	// Add datasources until we reach target size
	datasourceNum := 0
	for sb.Len() < targetSize-100 { // Leave room for closing braces
		if datasourceNum > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`"datasource`)
		sb.WriteString(strings.Repeat("x", 50)) // Pad datasource name
		sb.WriteString(`_`)
		sb.WriteString(string(rune('0' + (datasourceNum % 10))))
		sb.WriteString(`":{"table1":["field1","field2","field3","field4","field5","field6","field7","field8","field9","field10"]}`)
		datasourceNum++
	}

	sb.WriteString(`}},"metadata":{"padding":"`)
	// Add padding to reach exact size
	remaining := targetSize - sb.Len() - 5 // Account for closing characters
	if remaining > 0 {
		sb.WriteString(strings.Repeat("x", remaining))
	}
	sb.WriteString(`"}}`)

	return []byte(sb.String())
}

// ============================================================================
// CreateJob Handler Tests - Success Cases (refactored to use real handler + mocks)
// ============================================================================

func TestFetcherHandler_CreateJob_Success_NewJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)
	mockTester := command.NewMockConnectionTester(ctrl)

	testConn := &model.Connection{
		ID: uuid.New(),

		ProductName: "test-product",
		ConfigName:  "ds1",
		Type:        model.TypePostgreSQL,
	}

	// CreateFetcherJob service flow:
	// 1. ComputeRequestHash + NewJob + IsValid -> no mocks needed (pure model)
	// 2. Check for duplicate within deduplication window
	mockJobRepo.EXPECT().FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).Return(nil, nil)
	// 3. Find connections by datasource names
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), gomock.Any()).Return([]*model.Connection{testConn}, nil)
	// 4. Test each connection
	mockTester.EXPECT().TestConnection(gomock.Any(), testConn).Return(nil)
	// 5. Create the job in the repository
	mockJobRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, j *model.Job) (*model.Job, error) {
			return j, nil
		},
	)
	// 6. Publish to RabbitMQ queue
	mockRabbitMQ.EXPECT().ProducerDefault(gomock.Any(), "", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	createCmd := command.NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockCryptor, mockRabbitMQ, mockTester, "test-queue", nil)
	handler := &FetcherHandler{CreateJobCmd: createCmd}

	app := setupTestApp()
	app.Post("/v1/fetcher", handler.CreateJob)

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validFetcherRequestPayload()))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusAccepted, resp.StatusCode)

	var body model.FetcherResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.NotEmpty(t, body.JobID)
	assert.Equal(t, "pending", body.Status)
	assert.Equal(t, "Job created and queued for processing", body.Message)
}

func TestFetcherHandler_CreateJob_Success_DuplicateJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)
	mockTester := command.NewMockConnectionTester(ctrl)

	jobID := uuid.New()
	existingJob := createTestJob(jobID)

	// Service finds existing job within dedup window -> returns duplicate
	mockJobRepo.EXPECT().FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).Return(existingJob, nil)

	createCmd := command.NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockCryptor, mockRabbitMQ, mockTester, "test-queue", nil)
	handler := &FetcherHandler{CreateJobCmd: createCmd}

	app := setupTestApp()
	app.Post("/v1/fetcher", handler.CreateJob)

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validFetcherRequestPayload()))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.FetcherResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, jobID, body.JobID)
	assert.Contains(t, body.Message, "Duplicate")
}

// ============================================================================
// CreateJob Handler Tests - Error Cases (refactored to use real handler + mocks)
// ============================================================================

func TestFetcherHandler_CreateJob_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)
	mockTester := command.NewMockConnectionTester(ctrl)

	// No duplicate found
	mockJobRepo.EXPECT().FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).Return(nil, nil)
	// No connections found for the datasource -> returns "No Connections Found" error
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), gomock.Any()).Return(nil, nil)

	createCmd := command.NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockCryptor, mockRabbitMQ, mockTester, "test-queue", nil)
	handler := &FetcherHandler{CreateJobCmd: createCmd}

	app := setupTestApp()
	app.Post("/v1/fetcher", handler.CreateJob)

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validFetcherRequestPayload()))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// When no connections are found, the service returns a ValidationError
	// which maps to 400 Bad Request. The original test used an inline
	// closure returning 409 Conflict, but the real service path returns 400
	// for "missing data source" validation errors.
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestFetcherHandler_CreateJob_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)
	mockTester := command.NewMockConnectionTester(ctrl)

	// FindByRequestHashWithinWindow returns an error -> internal server error
	mockJobRepo.EXPECT().FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).Return(nil, assert.AnError)

	createCmd := command.NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockCryptor, mockRabbitMQ, mockTester, "test-queue", nil)
	handler := &FetcherHandler{CreateJobCmd: createCmd}

	app := setupTestApp()
	app.Post("/v1/fetcher", handler.CreateJob)

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validFetcherRequestPayload()))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// ============================================================================
// GetJob Handler Tests (refactored to use real handler + mocks)
// ============================================================================

func TestFetcherHandler_GetJob_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	jobID := uuid.New()

	testJob := createTestJob(jobID)

	mockJobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(testJob, nil)

	getJobQuery := query.NewGetJob(mockJobRepo)
	handler := &FetcherHandler{GetJobQuery: getJobQuery}

	app := setupTestApp()
	app.Get("/v1/fetcher/:id", handler.GetJob)

	req := httptest.NewRequest("GET", "/v1/fetcher/"+jobID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.JobResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, jobID, body.ID)
	assert.Equal(t, "pending", body.Status)
}

func TestFetcherHandler_GetJob_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	jobID := uuid.New()

	// Service returns nil -> not found
	mockJobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(nil, nil)

	getJobQuery := query.NewGetJob(mockJobRepo)
	handler := &FetcherHandler{GetJobQuery: getJobQuery}

	app := setupTestApp()
	app.Get("/v1/fetcher/:id", handler.GetJob)

	req := httptest.NewRequest("GET", "/v1/fetcher/"+jobID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestFetcherHandler_GetJob_InvalidID(t *testing.T) {
	app := setupTestApp()

	handler := &FetcherHandler{GetJobQuery: nil}
	app.Get("/v1/fetcher/:id", handler.GetJob)

	tests := []struct {
		name     string
		jobID    string
		wantCode int
	}{
		{
			name:     "invalid UUID format",
			jobID:    "not-a-uuid",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "partial UUID",
			jobID:    "550e8400-e29b",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "UUID with special characters",
			jobID:    "550e8400-e29b-<script>",
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/fetcher/"+tt.jobID, nil)
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestFetcherHandler_GetJob_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	jobID := uuid.New()

	// Repository returns an error -> internal server error
	mockJobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(nil, assert.AnError)

	getJobQuery := query.NewGetJob(mockJobRepo)
	handler := &FetcherHandler{GetJobQuery: getJobQuery}

	app := setupTestApp()
	app.Get("/v1/fetcher/:id", handler.GetJob)

	req := httptest.NewRequest("GET", "/v1/fetcher/"+jobID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// ============================================================================
// Handler Direct Tests
// ============================================================================

func TestFetcherHandler_GetJob_HandlerDirectly_InvalidUUID(t *testing.T) {
	app := setupTestApp()

	handler := &FetcherHandler{GetJobQuery: nil}
	app.Get("/v1/fetcher/:id", handler.GetJob)

	req := httptest.NewRequest("GET", "/v1/fetcher/invalid-uuid", nil)
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestFetcherHandler_CreateJob_HandlerDirectly_InvalidJSON(t *testing.T) {
	app := setupTestApp()

	handler := &FetcherHandler{CreateJobCmd: nil}
	app.Post("/v1/fetcher", handler.CreateJob)

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(`{broken`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "code")
}

// ============================================================================
// Edge Case Tests (refactored to use real handler + mocks)
// ============================================================================

func TestFetcherHandler_CreateJob_WithMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)
	mockTester := command.NewMockConnectionTester(ctrl)

	testConn := &model.Connection{
		ID: uuid.New(),

		ProductName: "test",
		ConfigName:  "ds1",
		Type:        model.TypePostgreSQL,
	}

	mockJobRepo.EXPECT().FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).Return(nil, nil)
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), gomock.Any()).Return([]*model.Connection{testConn}, nil)
	mockTester.EXPECT().TestConnection(gomock.Any(), testConn).Return(nil)
	mockJobRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, j *model.Job) (*model.Job, error) {
			return j, nil
		},
	)
	mockRabbitMQ.EXPECT().ProducerDefault(gomock.Any(), "", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	createCmd := command.NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockCryptor, mockRabbitMQ, mockTester, "test-queue", nil)
	handler := &FetcherHandler{CreateJobCmd: createCmd}

	app := setupTestApp()
	app.Post("/v1/fetcher", handler.CreateJob)

	payload := `{
		"dataRequest": {
			"mappedFields": {
				"ds1": {"table1": ["field1"]}
			}
		},
		"metadata": {
			"correlationId": "corr-123",
			"source": "test",
			"priority": 1
		}
	}`

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusAccepted, resp.StatusCode)
}

func TestFetcherHandler_CreateJob_WithFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)
	mockTester := command.NewMockConnectionTester(ctrl)

	testConn := &model.Connection{
		ID: uuid.New(),

		ProductName: "test-product",
		ConfigName:  "ds1",
		Type:        model.TypePostgreSQL,
	}

	mockJobRepo.EXPECT().FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).Return(nil, nil)
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), gomock.Any()).Return([]*model.Connection{testConn}, nil)
	mockTester.EXPECT().TestConnection(gomock.Any(), testConn).Return(nil)
	mockJobRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, j *model.Job) (*model.Job, error) {
			return j, nil
		},
	)
	mockRabbitMQ.EXPECT().ProducerDefault(gomock.Any(), "", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	createCmd := command.NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockCryptor, mockRabbitMQ, mockTester, "test-queue", nil)
	handler := &FetcherHandler{CreateJobCmd: createCmd}

	app := setupTestApp()
	app.Post("/v1/fetcher", handler.CreateJob)

	payload := `{
		"dataRequest": {
			"mappedFields": {
				"ds1": {"table1": ["field1"]}
			},
			"filters": {
				"ds1": {
					"table1": {
						"field1": {"eq": ["test-value"]}
					}
				}
			}
		},
		"metadata": {
			"source": "test-product"
		}
	}`

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusAccepted, resp.StatusCode)
}

func TestFetcherHandler_GetJob_CompletedJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	jobID := uuid.New()

	completedAt := time.Now().UTC()
	testJob := createTestJob(jobID)
	testJob.Status = model.JobStatusCompleted
	testJob.CompletedAt = &completedAt
	testJob.ResultPath = "s3://bucket/results/job-123.json"

	mockJobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(testJob, nil)

	getJobQuery := query.NewGetJob(mockJobRepo)
	handler := &FetcherHandler{GetJobQuery: getJobQuery}

	app := setupTestApp()
	app.Get("/v1/fetcher/:id", handler.GetJob)

	req := httptest.NewRequest("GET", "/v1/fetcher/"+jobID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.JobResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "completed", body.Status)
	assert.NotNil(t, body.CompletedAt)
	assert.Equal(t, "s3://bucket/results/job-123.json", body.ResultPath)
}

func TestFetcherHandler_GetJob_FailedJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	jobID := uuid.New()

	completedAt := time.Now().UTC()
	testJob := createTestJob(jobID)
	testJob.Status = model.JobStatusFailed
	testJob.CompletedAt = &completedAt
	testJob.Metadata = map[string]any{
		"error": "connection timeout",
	}

	mockJobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(testJob, nil)

	getJobQuery := query.NewGetJob(mockJobRepo)
	handler := &FetcherHandler{GetJobQuery: getJobQuery}

	app := setupTestApp()
	app.Get("/v1/fetcher/:id", handler.GetJob)

	req := httptest.NewRequest("GET", "/v1/fetcher/"+jobID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.JobResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "failed", body.Status)
	assert.NotNil(t, body.Metadata)
	assert.Equal(t, "connection timeout", body.Metadata["error"])
}

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNewFetcherHandler(t *testing.T) {
	// Test with nil dependencies (basic constructor test)
	handler := NewFetcherHandler(nil, nil)
	assert.NotNil(t, handler)
	assert.Nil(t, handler.CreateJobCmd)
	assert.Nil(t, handler.GetJobQuery)
}

// ============================================================================
// Handler Direct Tests - Missing Org Header
// ============================================================================
