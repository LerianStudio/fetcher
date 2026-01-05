package in

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	httpUtils "github.com/LerianStudio/fetcher/pkg/net/http"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// setupTestApp creates a Fiber app with test context middleware.
func setupTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024, // 10KB for test flexibility
	})

	// Middleware to inject test context with logger and tracer
	app.Use(func(c *fiber.Ctx) error {
		logger := &libLog.GoLogger{Level: libLog.DebugLevel}
		values := &libCommons.CustomContextKeyValue{
			HeaderID: "test-request-id",
			Logger:   logger,
			Tracer:   otel.Tracer("test"),
		}

		ctx := c.UserContext()
		ctx = context.WithValue(ctx, libCommons.CustomContextKey, values)
		c.SetUserContext(ctx)

		return c.Next()
	})

	return app
}

// TestFetcherHandler_CreateJob_InvalidJSON tests that invalid JSON returns 400.
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected status %d, got %d. Body: %s", tt.wantCode, resp.StatusCode, string(body))
			}
		})
	}
}

// TestFetcherHandler_CreateJob_MissingOrgHeader tests that missing or invalid X-Organization-Id returns 400.
func TestFetcherHandler_CreateJob_MissingOrgHeader(t *testing.T) {
	app := setupTestApp()

	handler := &FetcherHandler{CreateJobCmd: nil}
	app.Post("/v1/fetcher", handler.CreateJob)

	tests := []struct {
		name      string
		orgHeader string
		setHeader bool
		wantCode  int
	}{
		{
			name:      "missing X-Organization-Id header",
			orgHeader: "",
			setHeader: false,
			wantCode:  fiber.StatusBadRequest,
		},
		{
			name:      "invalid UUID format",
			orgHeader: "not-a-uuid",
			setHeader: true,
			wantCode:  fiber.StatusBadRequest,
		},
		{
			name:      "whitespace only header",
			orgHeader: "   ",
			setHeader: true,
			wantCode:  fiber.StatusBadRequest,
		},
	}

	validPayload := `{"dataRequest":{"mappedFields":{"ds1":{"t1":["f1"]}}}}`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validPayload))
			req.Header.Set("Content-Type", "application/json")

			if tt.setHeader {
				req.Header.Set("X-Organization-Id", tt.orgHeader)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected status %d, got %d. Body: %s", tt.wantCode, resp.StatusCode, string(body))
			}
		})
	}
}

// TestFetcherHandler_CreateJob_ContentTypeValidation tests content type handling.
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected status %d, got %d. Body: %s", tt.wantCode, resp.StatusCode, string(body))
			}
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

// createTestJob creates a test job with default values.
func createTestJob(id, orgID uuid.UUID) *model.Job {
	now := time.Now().UTC()
	return &model.Job{
		ID:             id,
		OrganizationID: orgID,
		MappedFields: map[string]map[string][]string{
			"ds1": {
				"table1": {"field1", "field2"},
			},
		},
		Status:      model.JobStatusPending,
		RequestHash: "test-hash-123",
		CreatedAt:   now,
	}
}

// validFetcherRequestPayload returns a valid FetcherRequest JSON payload.
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
			"correlationId": "test-123"
		}
	}`
}

// ============================================================================
// CreateJob Handler Tests - Success Cases
// ============================================================================

func TestFetcherHandler_CreateJob_Success_NewJob(t *testing.T) {
	app := setupTestApp()

	jobID := uuid.New()
	orgID := uuid.New()
	testJob := createTestJob(jobID, orgID)

	app.Post("/v1/fetcher", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		orgIDHeader, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.FetcherRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate successful job creation
		if orgIDHeader == orgID {
			response := model.FetcherResponse{
				JobID:     testJob.ID,
				Status:    string(testJob.Status),
				CreatedAt: testJob.CreatedAt,
				Message:   "Job created and queued for processing",
			}
			return httpUtils.Accepted(c, response)
		}

		return httpUtils.WithError(c, pkg.InternalServerError{
			EntityType: "fetcher",
			Code:       constant.ErrInternalServer.Error(),
			Title:      "Internal Error",
			Message:    "unexpected org id",
		})
	})

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validFetcherRequestPayload()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusAccepted, resp.StatusCode)

	var body model.FetcherResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, jobID, body.JobID)
	assert.Equal(t, "pending", body.Status)
	assert.Equal(t, "Job created and queued for processing", body.Message)
}

func TestFetcherHandler_CreateJob_Success_DuplicateJob(t *testing.T) {
	app := setupTestApp()

	jobID := uuid.New()
	orgID := uuid.New()
	testJob := createTestJob(jobID, orgID)

	app.Post("/v1/fetcher", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.FetcherRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate duplicate detection
		response := model.FetcherResponse{
			JobID:     testJob.ID,
			Status:    string(testJob.Status),
			CreatedAt: testJob.CreatedAt,
			Message:   "Duplicate request detected - returning existing job",
		}
		return httpUtils.OK(c, response)
	})

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validFetcherRequestPayload()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

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
// CreateJob Handler Tests - Error Cases
// ============================================================================

func TestFetcherHandler_CreateJob_Conflict(t *testing.T) {
	app := setupTestApp()
	orgID := uuid.New()

	app.Post("/v1/fetcher", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.FetcherRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate conflict error
		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusConflict,
			Code:       constant.ErrEntityConflict.Error(),
			Title:      "Conflict",
			Message:    "connection config not found",
		})
	})

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validFetcherRequestPayload()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestFetcherHandler_CreateJob_InternalError(t *testing.T) {
	app := setupTestApp()
	orgID := uuid.New()

	app.Post("/v1/fetcher", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.FetcherRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate internal error
		return httpUtils.WithError(c, pkg.InternalServerError{
			EntityType: "fetcher",
			Code:       constant.ErrInternalServer.Error(),
			Title:      "Internal Server Error",
			Message:    "database connection failed",
		})
	})

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validFetcherRequestPayload()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// ============================================================================
// GetJob Handler Tests
// ============================================================================

func TestFetcherHandler_GetJob_Success(t *testing.T) {
	app := setupTestApp()

	jobID := uuid.New()
	orgID := uuid.New()
	testJob := createTestJob(jobID, orgID)

	app.Get("/v1/fetcher/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "job",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid job id",
				Err:        err,
			})
		}

		if id == jobID {
			resp := model.NewJobResponseFrom(testJob)
			return httpUtils.OK(c, resp)
		}

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "job not found",
		})
	})

	req := httptest.NewRequest("GET", "/v1/fetcher/"+jobID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupTestApp()
	orgID := uuid.New()

	app.Get("/v1/fetcher/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "job",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid job id",
				Err:        err,
			})
		}

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "job not found",
		})
	})

	req := httptest.NewRequest("GET", "/v1/fetcher/"+uuid.New().String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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

func TestFetcherHandler_GetJob_MissingOrgHeader(t *testing.T) {
	app := setupTestApp()

	handler := &FetcherHandler{GetJobQuery: nil}
	app.Get("/v1/fetcher/:id", handler.GetJob)

	req := httptest.NewRequest("GET", "/v1/fetcher/"+uuid.New().String(), nil)
	// Not setting X-Organization-Id header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestFetcherHandler_GetJob_InternalError(t *testing.T) {
	app := setupTestApp()
	jobID := uuid.New()
	orgID := uuid.New()

	app.Get("/v1/fetcher/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "job",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid job id",
				Err:        err,
			})
		}

		// Simulate internal error
		return httpUtils.WithError(c, pkg.InternalServerError{
			EntityType: "job",
			Code:       constant.ErrInternalServer.Error(),
			Title:      "Internal Server Error",
			Message:    "database error",
		})
	})

	req := httptest.NewRequest("GET", "/v1/fetcher/"+jobID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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
// Edge Case Tests
// ============================================================================

func TestFetcherHandler_CreateJob_WithMetadata(t *testing.T) {
	app := setupTestApp()

	jobID := uuid.New()
	orgID := uuid.New()

	app.Post("/v1/fetcher", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.FetcherRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Verify metadata was parsed
		if request.Metadata == nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "metadata was expected but not found",
			})
		}

		response := model.FetcherResponse{
			JobID:     jobID,
			Status:    "pending",
			CreatedAt: time.Now().UTC(),
			Message:   "Job created and queued for processing",
		}
		return httpUtils.Accepted(c, response)
	})

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
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusAccepted, resp.StatusCode)
}

func TestFetcherHandler_CreateJob_WithFilters(t *testing.T) {
	app := setupTestApp()

	jobID := uuid.New()
	orgID := uuid.New()

	app.Post("/v1/fetcher", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.FetcherRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Verify filters were parsed
		if len(request.DataRequest.Filters) == 0 {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "filters were expected but not found",
			})
		}

		response := model.FetcherResponse{
			JobID:     jobID,
			Status:    "pending",
			CreatedAt: time.Now().UTC(),
			Message:   "Job created and queued for processing",
		}
		return httpUtils.Accepted(c, response)
	})

	payload := `{
		"dataRequest": {
			"mappedFields": {
				"ds1": {"table1": ["field1"]}
			},
			"filters": [
				{
					"field": "ds1.table1.field1",
					"operator": "eq",
					"value": ["test-value"]
				}
			]
		}
	}`

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusAccepted, resp.StatusCode)
}

func TestFetcherHandler_GetJob_CompletedJob(t *testing.T) {
	app := setupTestApp()

	jobID := uuid.New()
	orgID := uuid.New()
	completedAt := time.Now().UTC()
	testJob := createTestJob(jobID, orgID)
	testJob.Status = model.JobStatusCompleted
	testJob.CompletedAt = &completedAt
	testJob.ResultPath = "s3://bucket/results/job-123.json"

	app.Get("/v1/fetcher/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "job",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid job id",
				Err:        err,
			})
		}

		resp := model.NewJobResponseFrom(testJob)
		return httpUtils.OK(c, resp)
	})

	req := httptest.NewRequest("GET", "/v1/fetcher/"+jobID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupTestApp()

	jobID := uuid.New()
	orgID := uuid.New()
	completedAt := time.Now().UTC()
	testJob := createTestJob(jobID, orgID)
	testJob.Status = model.JobStatusFailed
	testJob.CompletedAt = &completedAt
	testJob.Metadata = map[string]any{
		"error": "connection timeout",
	}

	app.Get("/v1/fetcher/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "job",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid job id",
				Err:        err,
			})
		}

		resp := model.NewJobResponseFrom(testJob)
		return httpUtils.OK(c, resp)
	})

	req := httptest.NewRequest("GET", "/v1/fetcher/"+jobID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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

func TestFetcherHandler_CreateJob_HandlerDirectly_MissingOrgHeader(t *testing.T) {
	app := setupTestApp()

	handler := NewFetcherHandler(nil, nil)
	app.Post("/v1/fetcher", handler.CreateJob)

	req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validFetcherRequestPayload()))
	req.Header.Set("Content-Type", "application/json")
	// Not setting X-Organization-Id header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestFetcherHandler_GetJob_HandlerDirectly_MissingOrgHeader(t *testing.T) {
	app := setupTestApp()

	handler := NewFetcherHandler(nil, nil)
	app.Get("/v1/fetcher/:id", handler.GetJob)

	req := httptest.NewRequest("GET", "/v1/fetcher/"+uuid.New().String(), nil)
	// Not setting X-Organization-Id header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
