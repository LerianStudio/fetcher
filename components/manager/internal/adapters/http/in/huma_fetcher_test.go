package in

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/query"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	jobRepo "github.com/LerianStudio/fetcher/v2/pkg/mongodb/job"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	commonsopenapi "github.com/LerianStudio/lib-commons/v5/commons/net/http/openapi"
	"github.com/LerianStudio/lib-commons/v5/commons/net/http/problem"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRegisterFetcherOperationsServesTypedResponses(t *testing.T) {
	createdAt := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	jobID := uuid.MustParse("01980a89-21f0-7d7e-a109-564b5c6f53ac")

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		handlers   OperationHandlers
		wantStatus int
		wantBody   any
	}{
		{
			name:   "new job",
			method: http.MethodPost,
			path:   "/v1/fetcher",
			body:   `{"dataRequest":{"mappedFields":{"ledger":{"transactions":["id"]}}}}`,
			handlers: OperationHandlers{
				CreateJob: func(context.Context, *rawJSONInput) (*createJobOutput, error) {
					return &createJobOutput{
						Status: http.StatusAccepted,
						Body: model.FetcherResponse{
							JobID:     jobID,
							Status:    string(model.JobStatusPending),
							CreatedAt: createdAt,
							Message:   "Job created and queued for processing",
						},
					}, nil
				},
			},
			wantStatus: http.StatusAccepted,
			wantBody: model.FetcherResponse{
				JobID:     jobID,
				Status:    string(model.JobStatusPending),
				CreatedAt: createdAt,
				Message:   "Job created and queued for processing",
			},
		},
		{
			name:   "duplicate job",
			method: http.MethodPost,
			path:   "/v1/fetcher",
			body:   `{"dataRequest":{"mappedFields":{"ledger":{"transactions":["id"]}}}}`,
			handlers: OperationHandlers{
				CreateJob: func(context.Context, *rawJSONInput) (*createJobOutput, error) {
					return &createJobOutput{
						Status: http.StatusOK,
						Body: model.FetcherResponse{
							JobID:     jobID,
							Status:    string(model.JobStatusPending),
							CreatedAt: createdAt,
							Message:   "Duplicate request detected - returning existing job",
						},
					}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantBody: model.FetcherResponse{
				JobID:     jobID,
				Status:    string(model.JobStatusPending),
				CreatedAt: createdAt,
				Message:   "Duplicate request detected - returning existing job",
			},
		},
		{
			name:   "get job",
			method: http.MethodGet,
			path:   "/v1/fetcher/" + jobID.String(),
			handlers: OperationHandlers{
				GetJob: func(context.Context, *idInput) (*getJobOutput, error) {
					return &getJobOutput{Body: model.JobResponse{
						ID:           jobID,
						MappedFields: map[string]map[string][]string{"ledger": {"transactions": {"id"}}},
						Status:       string(model.JobStatusPending),
						CreatedAt:    createdAt,
					}}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantBody: model.JobResponse{
				ID:           jobID,
				MappedFields: map[string]map[string][]string{"ledger": {"transactions": {"id"}}},
				Status:       string(model.JobStatusPending),
				CreatedAt:    createdAt,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newFetcherTestAPI(t, tt.handlers, nil)
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(body))

			want, err := json.Marshal(tt.wantBody)
			require.NoError(t, err)
			var wantJSON, gotJSON any
			require.NoError(t, json.Unmarshal(want, &wantJSON))
			require.NoError(t, json.Unmarshal(body, &gotJSON))
			require.Equal(t, wantJSON, gotJSON)
		})
	}
}

func TestFetcherCreateCallbackPreservesNewAndDuplicateSemantics(t *testing.T) {
	t.Run("new job", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connections := connRepo.NewMockRepository(ctrl)
		jobs := jobRepo.NewMockRepository(ctrl)
		connection := &model.Connection{
			ID:          uuid.New(),
			ConfigName:  "ledger",
			ProductName: "test",
		}

		jobs.EXPECT().
			FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).
			Return(nil, nil)
		connections.EXPECT().
			FindByConfigNames(gomock.Any(), []string{"ledger"}).
			Return([]*model.Connection{connection}, nil)
		jobs.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, job *model.Job) (*model.Job, error) {
				return job, nil
			})

		createJob := command.NewCreateFetcherJob(connections, jobs, nil, nil, "", nil, nil)
		handler := NewFetcherHandler(createJob, nil)
		handlers := OperationHandlers{}
		bindFetcherHandlers(&handlers, handler)
		app, _ := newFetcherTestAPI(t, handlers, nil)

		resp, body := makeFetcherRequest(
			t,
			app,
			http.MethodPost,
			"/v1/fetcher",
			`{"dataRequest":{"mappedFields":{"ledger":{"transactions":["id"]}}},"metadata":{"source":"test"}}`,
		)
		require.Equal(t, http.StatusAccepted, resp.StatusCode, string(body))

		var output model.FetcherResponse
		require.NoError(t, json.Unmarshal(body, &output))
		require.NotEqual(t, uuid.Nil, output.JobID)
		require.Equal(t, string(model.JobStatusPending), output.Status)
		require.False(t, output.CreatedAt.IsZero())
		require.Equal(t, "Job created and queued for processing", output.Message)
	})

	t.Run("duplicate job", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		jobs := jobRepo.NewMockRepository(ctrl)
		createdAt := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
		existing := &model.Job{
			ID:           uuid.MustParse("01980a89-21f0-7d7e-a109-564b5c6f53ac"),
			MappedFields: map[string]map[string][]string{"ledger": {"transactions": {"id"}}},
			Status:       model.JobStatusPending,
			CreatedAt:    createdAt,
		}

		jobs.EXPECT().
			FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).
			Return(existing, nil)

		createJob := command.NewCreateFetcherJob(nil, jobs, nil, nil, "", nil, nil)
		handler := NewFetcherHandler(createJob, nil)
		handlers := OperationHandlers{}
		bindFetcherHandlers(&handlers, handler)
		app, _ := newFetcherTestAPI(t, handlers, nil)

		resp, body := makeFetcherRequest(
			t,
			app,
			http.MethodPost,
			"/v1/fetcher",
			`{"dataRequest":{"mappedFields":{"ledger":{"transactions":["id"]}}},"metadata":{"source":"test"}}`,
		)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

		var output model.FetcherResponse
		require.NoError(t, json.Unmarshal(body, &output))
		require.Equal(t, existing.ID, output.JobID)
		require.Equal(t, existing.CreatedAt, output.CreatedAt)
		require.Equal(t, "Duplicate request detected - returning existing job", output.Message)
	})
}

func TestFetcherCreateCallbackPreservesMetadataAndFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	connections := connRepo.NewMockRepository(ctrl)
	jobs := jobRepo.NewMockRepository(ctrl)
	connection := &model.Connection{
		ID:          uuid.New(),
		ConfigName:  "ledger",
		ProductName: "payments",
	}

	jobs.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).
		Return(nil, nil)
	connections.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"ledger"}).
		Return([]*model.Connection{connection}, nil)
	jobs.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, created *model.Job) (*model.Job, error) {
			require.Equal(t, map[string]any{
				"source":        "payments",
				"correlationId": "settlement-42",
			}, created.Metadata)
			require.Equal(t, []any{"approved"}, created.Filters["ledger"]["transactions"]["status"].Equals)

			return created, nil
		},
	)

	handler := NewFetcherHandler(command.NewCreateFetcherJob(connections, jobs, nil, nil, "", nil, nil), nil)
	handlers := OperationHandlers{}
	bindFetcherHandlers(&handlers, handler)
	app, _ := newFetcherTestAPI(t, handlers, nil)

	resp, body := makeFetcherRequest(
		t,
		app,
		http.MethodPost,
		"/v1/fetcher",
		`{
			"dataRequest": {
				"mappedFields": {"ledger": {"transactions": ["id", "status"]}},
				"filters": {"ledger": {"transactions": {"status": {"eq": ["approved"]}}}}
			},
			"metadata": {"source": "payments", "correlationId": "settlement-42"}
		}`,
	)
	require.Equal(t, http.StatusAccepted, resp.StatusCode, string(body))
	require.Contains(t, resp.Header.Get("Content-Type"), "application/json")
}

func TestFetcherGetCallbackValidatesIDAndFiltersInternalMetadata(t *testing.T) {
	t.Run("invalid identifier", func(t *testing.T) {
		handler := NewFetcherHandler(nil, nil)
		handlers := OperationHandlers{}
		bindFetcherHandlers(&handlers, handler)
		app, _ := newFetcherTestAPI(t, handlers, nil)

		resp, body := makeFetcherRequest(t, app, http.MethodGet, "/v1/fetcher/not-a-uuid", "")
		requireFetcherProblem(
			t,
			resp,
			body,
			http.StatusBadRequest,
			"FET-0404",
			"Bad Request",
			"invalid job id",
		)
	})

	t.Run("public response", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		jobs := jobRepo.NewMockRepository(ctrl)
		jobID := uuid.MustParse("01980a89-21f0-7d7e-a109-564b5c6f53ac")
		job := &model.Job{
			ID: jobID,
			Metadata: map[string]any{
				"source":               "settlement",
				"terminalEventPending": true,
				"terminalEventStatus":  "pending",
				"terminalEventPayload": map[string]any{"secret": "internal"},
			},
			MappedFields: map[string]map[string][]string{"ledger": {"transactions": {"id"}}},
			Status:       model.JobStatusCompleted,
			CreatedAt:    time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC),
		}
		jobs.EXPECT().FindByID(gomock.Any(), jobID).Return(job, nil)

		handler := NewFetcherHandler(nil, query.NewGetJob(jobs))
		handlers := OperationHandlers{}
		bindFetcherHandlers(&handlers, handler)
		app, _ := newFetcherTestAPI(t, handlers, nil)

		resp, body := makeFetcherRequest(t, app, http.MethodGet, "/v1/fetcher/"+jobID.String(), "")
		require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
		require.NotContains(t, string(body), "terminalEvent")
		require.NotContains(t, string(body), "internal")

		var output model.JobResponse
		require.NoError(t, json.Unmarshal(body, &output))
		require.Equal(t, map[string]any{"source": "settlement"}, output.Metadata)
	})
}

func TestFetcherCreateCallbackValidatesMetadataSource(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantDetail string
	}{
		{
			name:       "missing metadata",
			body:       `{"dataRequest":{"mappedFields":{"ledger":{"transactions":["id"]}}}}`,
			wantDetail: "metadata is required and must contain 'source' field",
		},
		{
			name:       "missing source",
			body:       `{"dataRequest":{"mappedFields":{"ledger":{"transactions":["id"]}}},"metadata":{}}`,
			wantDetail: "metadata.source is required for job notification routing",
		},
		{
			name:       "blank source",
			body:       `{"dataRequest":{"mappedFields":{"ledger":{"transactions":["id"]}}},"metadata":{"source":"   "}}`,
			wantDetail: "metadata.source must be a non-empty string",
		},
		{
			name:       "non-string source",
			body:       `{"dataRequest":{"mappedFields":{"ledger":{"transactions":["id"]}}},"metadata":{"source":123}}`,
			wantDetail: "metadata.source must be a non-empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewFetcherHandler(command.NewCreateFetcherJob(nil, nil, nil, nil, "", nil, nil), nil)
			handlers := OperationHandlers{}
			bindFetcherHandlers(&handlers, handler)
			app, _ := newFetcherTestAPI(t, handlers, nil)

			resp, body := makeFetcherRequest(t, app, http.MethodPost, "/v1/fetcher", tt.body)
			requireFetcherProblem(
				t,
				resp,
				body,
				http.StatusBadRequest,
				"FET-0402",
				"Bad Request",
				tt.wantDetail,
			)
		})
	}
}

func TestFetcherGetCallbackPreservesTerminalJobRepresentations(t *testing.T) {
	jobID := uuid.MustParse("01980a89-21f0-7d7e-a109-564b5c6f53ac")
	createdAt := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	completedAt := createdAt.Add(5 * time.Minute)

	tests := []struct {
		name           string
		job            *model.Job
		wantStatus     model.JobStatus
		wantResultPath string
		wantError      string
	}{
		{
			name: "completed job",
			job: &model.Job{
				ID:          jobID,
				Metadata:    map[string]any{"source": "payments"},
				Status:      model.JobStatusCompleted,
				ResultPath:  "s3://fetcher-results/job.json",
				ResultHMAC:  "sha256-result",
				CreatedAt:   createdAt,
				CompletedAt: &completedAt,
			},
			wantStatus:     model.JobStatusCompleted,
			wantResultPath: "s3://fetcher-results/job.json",
		},
		{
			name: "failed job",
			job: &model.Job{
				ID:          jobID,
				Metadata:    map[string]any{"source": "payments", "error": "connection timeout"},
				Status:      model.JobStatusFailed,
				CreatedAt:   createdAt,
				CompletedAt: &completedAt,
			},
			wantStatus: model.JobStatusFailed,
			wantError:  "connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			jobs := jobRepo.NewMockRepository(ctrl)
			jobs.EXPECT().FindByID(gomock.Any(), jobID).Return(tt.job, nil)

			handler := NewFetcherHandler(nil, query.NewGetJob(jobs))
			handlers := OperationHandlers{}
			bindFetcherHandlers(&handlers, handler)
			app, _ := newFetcherTestAPI(t, handlers, nil)

			resp, body := makeFetcherRequest(t, app, http.MethodGet, "/v1/fetcher/"+jobID.String(), "")
			require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
			require.Contains(t, resp.Header.Get("Content-Type"), "application/json")

			var output model.JobResponse
			require.NoError(t, json.Unmarshal(body, &output))
			require.Equal(t, string(tt.wantStatus), output.Status)
			require.Equal(t, tt.wantResultPath, output.ResultPath)
			if tt.wantError == "" {
				require.NotContains(t, output.Metadata, "error")
			} else {
				require.Equal(t, tt.wantError, output.Metadata["error"])
			}
			require.Equal(t, &completedAt, output.CompletedAt)
		})
	}
}

func TestFetcherErrorsUseProblemContract(t *testing.T) {
	t.Run("malformed request", func(t *testing.T) {
		handler := NewFetcherHandler(nil, nil)
		handlers := OperationHandlers{}
		bindFetcherHandlers(&handlers, handler)
		app, _ := newFetcherTestAPI(t, handlers, nil)

		resp, body := makeFetcherRequest(t, app, http.MethodPost, "/v1/fetcher", `{`)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, string(body))
		requireFetcherProblem(
			t,
			resp,
			body,
			http.StatusBadRequest,
			"FET-0001",
			"Bad Request",
			"unable to parse request body",
		)
	})

	t.Run("unknown service error is scrubbed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		jobs := jobRepo.NewMockRepository(ctrl)
		jobID := uuid.MustParse("01980a89-21f0-7d7e-a109-564b5c6f53ac")
		jobs.EXPECT().FindByID(gomock.Any(), jobID).Return(nil, errors.New("database password leaked"))

		handler := NewFetcherHandler(nil, query.NewGetJob(jobs))
		handlers := OperationHandlers{}
		bindFetcherHandlers(&handlers, handler)
		app, _ := newFetcherTestAPI(t, handlers, nil)

		resp, body := makeFetcherRequest(t, app, http.MethodGet, "/v1/fetcher/"+jobID.String(), "")
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode, string(body))
		require.NotContains(t, string(body), "database password leaked")
		requireFetcherProblem(
			t,
			resp,
			body,
			http.StatusInternalServerError,
			"FET-0002",
			"Internal Server Error",
			"internal error",
		)
	})

	t.Run("missing job keeps its domain taxonomy", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		jobs := jobRepo.NewMockRepository(ctrl)
		jobID := uuid.MustParse("01980a89-21f0-7d7e-a109-564b5c6f53ac")
		jobs.EXPECT().FindByID(gomock.Any(), jobID).Return(nil, nil)

		handler := NewFetcherHandler(nil, query.NewGetJob(jobs))
		handlers := OperationHandlers{}
		bindFetcherHandlers(&handlers, handler)
		app, _ := newFetcherTestAPI(t, handlers, nil)

		resp, body := makeFetcherRequest(t, app, http.MethodGet, "/v1/fetcher/"+jobID.String(), "")
		require.Equal(t, http.StatusNotFound, resp.StatusCode, string(body))
		requireFetcherProblem(
			t,
			resp,
			body,
			http.StatusNotFound,
			"FET-1001",
			"Not Found",
			"It was not possible to find the job entity during the requested flow. Please review the data provided in the request.",
		)
	})
}

func TestFetcherCreateErrorsPreserveDomainTaxonomy(t *testing.T) {
	t.Run("missing datasource", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connections := connRepo.NewMockRepository(ctrl)
		jobs := jobRepo.NewMockRepository(ctrl)
		jobs.EXPECT().
			FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).
			Return(nil, nil)
		connections.EXPECT().FindByConfigNames(gomock.Any(), []string{"missing"}).Return(nil, nil)

		handler := NewFetcherHandler(command.NewCreateFetcherJob(connections, jobs, nil, nil, "", nil, nil), nil)
		handlers := OperationHandlers{}
		bindFetcherHandlers(&handlers, handler)
		app, _ := newFetcherTestAPI(t, handlers, nil)

		resp, body := makeFetcherRequest(
			t,
			app,
			http.MethodPost,
			"/v1/fetcher",
			`{"dataRequest":{"mappedFields":{"missing":{"transactions":["id"]}}},"metadata":{"source":"payments"}}`,
		)
		requireFetcherProblem(
			t,
			resp,
			body,
			http.StatusBadRequest,
			"FET-1020",
			"Bad Request",
			"No connections configured for the requested datasources",
		)
	})

	t.Run("unknown create failure is scrubbed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		jobs := jobRepo.NewMockRepository(ctrl)
		jobs.EXPECT().
			FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), command.DeduplicationWindowMinutes).
			Return(nil, errors.New("mongodb password leaked"))

		handler := NewFetcherHandler(command.NewCreateFetcherJob(nil, jobs, nil, nil, "", nil, nil), nil)
		handlers := OperationHandlers{}
		bindFetcherHandlers(&handlers, handler)
		app, _ := newFetcherTestAPI(t, handlers, nil)

		resp, body := makeFetcherRequest(
			t,
			app,
			http.MethodPost,
			"/v1/fetcher",
			`{"dataRequest":{"mappedFields":{"ledger":{"transactions":["id"]}}},"metadata":{"source":"payments"}}`,
		)
		require.NotContains(t, string(body), "mongodb password leaked")
		requireFetcherProblem(
			t,
			resp,
			body,
			http.StatusInternalServerError,
			"FET-0002",
			"Internal Server Error",
			"internal error",
		)
	})
}

func TestFetcherMalformedBodyAcceptsJSONCharset(t *testing.T) {
	handler := NewFetcherHandler(nil, nil)
	handlers := OperationHandlers{}
	bindFetcherHandlers(&handlers, handler)
	app, _ := newFetcherTestAPI(t, handlers, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/fetcher", strings.NewReader(`{broken`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	requireFetcherProblem(
		t,
		resp,
		body,
		http.StatusBadRequest,
		"FET-0001",
		"Bad Request",
		"unable to parse request body",
	)
}

func TestFetcherHumaRailReplacesLegacyFiberHandlers(t *testing.T) {
	handler := NewFetcherHandler(nil, nil)
	app, _ := newFetcherTestAPI(t, OperationHandlers{
		CreateJob: func(context.Context, *rawJSONInput) (*createJobOutput, error) {
			return &createJobOutput{Status: http.StatusAccepted}, nil
		},
	}, nil)

	resp, body := makeFetcherRequest(t, app, http.MethodPost, "/v1/fetcher", `{}`)
	require.Equal(t, http.StatusAccepted, resp.StatusCode, string(body))

	handlerType := reflect.TypeOf(handler)
	_, hasLegacyCreate := handlerType.MethodByName("CreateJob")
	_, hasLegacyGet := handlerType.MethodByName("GetJob")
	require.False(t, hasLegacyCreate, "FetcherHandler.CreateJob keeps the dead Fiber response rail callable")
	require.False(t, hasLegacyGet, "FetcherHandler.GetJob keeps the dead Fiber response rail callable")
}

func TestFetcherOpenAPIHasTypedBodyAndBothCreateStatuses(t *testing.T) {
	_, api := newFetcherTestAPI(t, OperationHandlers{}, nil)
	spec := fetcherOpenAPIDocument(t, api)
	paths := spec["paths"].(map[string]any)
	post := paths["/v1/fetcher"].(map[string]any)["post"].(map[string]any)
	responses := post["responses"].(map[string]any)
	require.Contains(t, responses, "200")
	require.Contains(t, responses, "202")
	_, hasOperationSecurity := post["security"]
	require.False(t, hasOperationSecurity, "Fetcher operations must inherit the global BearerAuth requirement")

	requestBody := post["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	schema := content["application/json"].(map[string]any)["schema"].(map[string]any)
	require.Equal(t, "#/components/schemas/FetcherRequest", schema["$ref"])
}

func TestFetcherOpenAPISchemasHaveDescriptionsAndExamples(t *testing.T) {
	_, api := newFetcherTestAPI(t, OperationHandlers{}, nil)
	spec := fetcherOpenAPIDocument(t, api)

	wantFields := map[string][]string{
		"FetcherRequest":  {"dataRequest", "metadata"},
		"DataRequest":     {"mappedFields", "filters"},
		"FetcherResponse": {"jobId", "status", "createdAt", "message"},
		"JobResponse": {
			"id",
			"metadata",
			"mappedFields",
			"filters",
			"status",
			"resultPath",
			"resultHmac",
			"requestHash",
			"createdAt",
			"completedAt",
		},
	}

	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)

	for schemaName, fields := range wantFields {
		schema := schemas[schemaName].(map[string]any)
		properties := schema["properties"].(map[string]any)

		for _, field := range fields {
			property := properties[field].(map[string]any)
			require.NotEmpty(t, property["description"], "%s.%s is missing documentation", schemaName, field)
			require.NotEmpty(t, property["examples"], "%s.%s is missing an example", schemaName, field)
		}
	}
}

func TestFetcherOperationsApplyTheirAuthorizationMiddleware(t *testing.T) {
	var registered []string
	middlewareRuns := 0
	factory := func(resource, action string) []fiber.Handler {
		registered = append(registered, resource+":"+action)

		return []fiber.Handler{func(c *fiber.Ctx) error {
			middlewareRuns++

			return c.Next()
		}}
	}
	handlers := OperationHandlers{
		CreateJob: func(context.Context, *rawJSONInput) (*createJobOutput, error) {
			return &createJobOutput{Status: http.StatusAccepted}, nil
		},
		GetJob: func(context.Context, *idInput) (*getJobOutput, error) {
			return &getJobOutput{}, nil
		},
	}
	app, _ := newFetcherTestAPI(t, handlers, factory)

	postResponse, postBody := makeFetcherRequest(t, app, http.MethodPost, "/v1/fetcher", `{}`)
	require.Equal(t, http.StatusAccepted, postResponse.StatusCode, string(postBody))
	getResponse, getBody := makeFetcherRequest(
		t,
		app,
		http.MethodGet,
		"/v1/fetcher/01980a89-21f0-7d7e-a109-564b5c6f53ac",
		"",
	)
	require.Equal(t, http.StatusOK, getResponse.StatusCode, string(getBody))
	require.Equal(t, []string{"fetcher:post", "fetcher:get"}, registered)
	require.Equal(t, 2, middlewareRuns)
}

func fetcherOpenAPIDocument(t *testing.T, api huma.API) map[string]any {
	t.Helper()

	document, err := json.Marshal(api.OpenAPI())
	require.NoError(t, err)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(document, &spec))

	return spec
}

func makeFetcherRequest(
	t *testing.T,
	app *fiber.App,
	method string,
	path string,
	body string,
) (*http.Response, []byte) {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, responseBody
}

func newFetcherTestAPI(
	t *testing.T,
	handlers OperationHandlers,
	middlewareFactory OperationMiddlewareFactory,
) (*fiber.App, huma.API) {
	t.Helper()

	app := fiber.New()
	api := commonsopenapi.New(app, app.Group(""), commonsopenapi.Config{
		Title:   "Fetcher Test API",
		Version: "test",
	})
	problem.Install()
	registerFetcherOperations(app, api, handlers, middlewareFactory)

	return app, api
}

func requireFetcherProblem(
	t *testing.T,
	resp *http.Response,
	body []byte,
	wantStatus int,
	wantCode string,
	wantTitle string,
	wantDetail string,
) {
	t.Helper()

	require.Equal(t, wantStatus, resp.StatusCode, string(body))
	require.Contains(t, resp.Header.Get("Content-Type"), "application/problem+json")

	var detail problem.Detail
	require.NoError(t, json.Unmarshal(body, &detail))
	require.Equal(t, wantStatus, detail.Status)
	require.Equal(t, problem.BaseURI+"/"+wantCode, detail.Type)
	require.Equal(t, wantTitle, detail.Title)
	require.Equal(t, wantDetail, detail.Detail)
	require.Equal(t, wantCode, detail.Code)
	require.Empty(t, detail.Errors)
}
