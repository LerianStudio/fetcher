package in

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/lib-commons/v5/commons/net/http/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type humaValidationProbeInput struct {
	Name string `query:"name" required:"true"`
}

type humaValidationProbeOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func TestProblemInstall_PreservesValidationDetailsAndScrubsServerErrors(t *testing.T) {
	app := fiber.New()
	api := BuildHumaAPI(app, false)
	huma.Register(api, huma.Operation{
		OperationID: "validationProbe",
		Method:      http.MethodGet,
		Path:        "/validation",
		Errors:      []int{http.StatusUnprocessableEntity},
	}, func(context.Context, *humaValidationProbeInput) (*humaValidationProbeOutput, error) {
		return &humaValidationProbeOutput{}, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "serverErrorProbe",
		Method:      http.MethodGet,
		Path:        "/server-error",
		Errors:      []int{http.StatusInternalServerError},
	}, func(context.Context, *emptyInput) (*humaValidationProbeOutput, error) {
		return nil, huma.Error500InternalServerError("secret database address", errors.New("secret cause"))
	})

	validationResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/validation", nil))
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, validationResp.Body.Close())
	}()
	var validationDetail problem.Detail
	require.NoError(t, json.NewDecoder(validationResp.Body).Decode(&validationDetail))
	assert.Equal(t, http.StatusUnprocessableEntity, validationResp.StatusCode)
	assert.NotEmpty(t, validationDetail.Errors)
	assert.Empty(t, validationDetail.Code)

	serverResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/server-error", nil))
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, serverResp.Body.Close())
	}()
	var serverDetail problem.Detail
	require.NoError(t, json.NewDecoder(serverResp.Body).Decode(&serverDetail))
	assert.Equal(t, http.StatusInternalServerError, serverResp.StatusCode)
	assert.Equal(t, "internal error", serverDetail.Detail)
	assert.Nil(t, serverDetail.Errors)
}

func TestGenerateCanonicalSpec_IsDeterministic(t *testing.T) {
	first, err := GenerateCanonicalSpec()
	require.NoError(t, err)
	second, err := GenerateCanonicalSpec()
	require.NoError(t, err)

	assert.True(t, bytes.Equal(first, second))
	assert.Contains(t, string(first), "openapi: 3.1.0")
	assert.Contains(t, string(first), "Elastic License 2.0 (Source Available)")
}

func TestRegisterHumaOperations_ExecutesTypedCallback(t *testing.T) {
	app := fiber.New()
	api := BuildHumaAPI(app, false)
	wantID := uuid.New()

	handlers := OperationHandlers{
		GetJob: func(_ context.Context, input *idInput) (*getJobOutput, error) {
			assert.Equal(t, wantID.String(), input.ID)
			return &getJobOutput{Body: model.JobResponse{ID: wantID, Status: "PENDING"}}, nil
		},
	}
	RegisterHumaOperations(app, api, handlers, nil)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/fetcher/"+wantID.String(), nil))
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, resp.Body.Close())
	}()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body model.JobResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, wantID, body.ID)
	assert.Equal(t, "PENDING", body.Status)
}

func TestRegisterHumaOperations_DocumentsCompleteBusinessContract(t *testing.T) {
	app := fiber.New()
	api := BuildHumaAPI(app, true)
	RegisterHumaOperations(app, api, OperationHandlers{}, nil)

	tests := []struct {
		method        string
		path          string
		successStatus string
	}{
		{http.MethodPost, "/v1/management/connections", "201"},
		{http.MethodGet, "/v1/management/connections", "200"},
		{http.MethodPost, "/v1/management/connections/validate-schema", "200"},
		{http.MethodGet, "/v1/management/connections/unassigned", "200"},
		{http.MethodPost, "/v1/management/connections/{id}/assign", "200"},
		{http.MethodGet, "/v1/management/connections/{id}", "200"},
		{http.MethodPost, "/v1/management/connections/{id}/test", "200"},
		{http.MethodGet, "/v1/management/connections/{id}/schema", "200"},
		{http.MethodPatch, "/v1/management/connections/{id}", "200"},
		{http.MethodDelete, "/v1/management/connections/{id}", "204"},
		{http.MethodPost, "/v1/fetcher", "202"},
		{http.MethodGet, "/v1/fetcher/{id}", "200"},
	}

	doc := api.OpenAPI()
	require.Len(t, doc.Paths, 9)
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			item := doc.Paths[tc.path]
			require.NotNil(t, item)

			var operation any
			switch tc.method {
			case http.MethodGet:
				operation = item.Get
			case http.MethodPost:
				operation = item.Post
			case http.MethodPatch:
				operation = item.Patch
			case http.MethodDelete:
				operation = item.Delete
			}
			require.NotNil(t, operation)

			var responses map[string]*huma.Response
			switch op := operation.(type) {
			case *huma.Operation:
				responses = op.Responses
			default:
				t.Fatalf("unexpected operation type %T", operation)
			}
			require.Contains(t, responses, tc.successStatus)
			require.Contains(t, responses, "500")
		})
	}

	schemas := doc.Components.Schemas.Map()
	require.Contains(t, schemas, "Detail")
	require.Contains(t, schemas["Detail"].Properties, "code")
	require.Contains(t, schemas, "ConnectionInput")
	configName := schemas["ConnectionInput"].Properties["configName"]
	require.NotNil(t, configName)
	assert.NotEmpty(t, configName.Description)
	assert.NotNil(t, configName.Examples)
}

func TestRegisterHumaOperations_DocumentsEveryPublicDTOField(t *testing.T) {
	app := fiber.New()
	api := BuildHumaAPI(app, true)
	RegisterHumaOperations(app, api, OperationHandlers{}, nil)

	publicDTOFields := map[string][]string{
		"ConnectionInput":          {"configName", "type", "host", "port", "databaseName", "schema", "userName", "password", "ssl", "metadata"},
		"SSLInput":                 {"mode", "ca", "cert", "key"},
		"ConnectionUpdateInput":    {"configName", "type", "host", "port", "databaseName", "schema", "userName", "password", "ssl", "metadata"},
		"SSLUpdateInput":           {"mode", "ca", "cert", "key"},
		"ConnectionResponse":       {"id", "productName", "configName", "type", "host", "port", "databaseName", "schema", "userName", "ssl", "metadata", "createdAt", "updatedAt"},
		"SSLResponse":              {"mode"},
		"ConnectionPage":           {"items", "page", "limit", "total"},
		"ConnectionTestResponse":   {"status", "message", "latencyMs"},
		"ConnectionSchemaResponse": {"id", "configName", "databaseName", "type", "tables"},
		"TableDetails":             {"name", "fields"},
		"SchemaValidationRequest":  {"mappedFields"},
		"SchemaValidationResponse": {"status", "message", "errors"},
		"SchemaValidationError":    {"type", "dataSourceId", "table", "field"},
		"FetcherRequest":           {"dataRequest", "metadata"},
		"DataRequest":              {"mappedFields", "filters"},
		"FilterCondition":          {"eq", "gt", "gte", "lt", "lte", "between", "in", "nin", "ne", "like"},
		"FetcherResponse":          {"jobId", "status", "createdAt", "message"},
		"JobResponse":              {"id", "metadata", "mappedFields", "filters", "status", "resultPath", "resultHmac", "requestHash", "createdAt", "completedAt"},
	}

	schemas := api.OpenAPI().Components.Schemas.Map()
	require.Len(t, schemas, len(publicDTOFields)+2)
	require.Contains(t, schemas, "Detail")
	require.Contains(t, schemas, "ErrorDetail")
	for schemaName, fields := range publicDTOFields {
		t.Run(schemaName, func(t *testing.T) {
			schema := schemas[schemaName]
			require.NotNil(t, schema, "public DTO schema is not reachable from the registered operations")
			for _, fieldName := range fields {
				field := schema.Properties[fieldName]
				require.NotNil(t, field, "%s.%s is not documented", schemaName, fieldName)
				assert.NotEmpty(t, field.Description, "%s.%s is missing a description", schemaName, fieldName)
				assert.NotEmpty(t, field.Examples, "%s.%s is missing an example", schemaName, fieldName)
			}
		})
	}
}

func TestBuildHumaAPI_ConfiguresCanonicalDocument(t *testing.T) {
	api := BuildHumaAPI(fiber.New(), true)
	doc := api.OpenAPI()

	require.NotNil(t, doc.Info.Contact)
	assert.Equal(t, "Lerian", doc.Info.Contact.Name)
	assert.Equal(t, "contact@lerian.studio", doc.Info.Contact.Email)
	require.NotNil(t, doc.Info.License)
	assert.Equal(t, "Elastic License 2.0 (Source Available)", doc.Info.License.Name)
	assert.Equal(t, "Elastic-2.0", doc.Info.License.Identifier)
	assert.Empty(t, doc.Info.License.URL)
	require.Len(t, doc.Servers, 1)
	assert.Equal(t, "http://localhost:4006", doc.Servers[0].URL)
	require.Contains(t, doc.Components.SecuritySchemes, bearerAuthSecurityScheme)
	assert.Equal(t, []map[string][]string{{bearerAuthSecurityScheme: {}}}, doc.Security)
	require.Len(t, doc.Tags, 3)
	for _, tag := range doc.Tags {
		assert.NotEmpty(t, tag.Description)
	}

	mapped := problem.MapError(nil, nil, nil, "FET-0002")
	assert.Contains(t, mapped.Error(), "internal error")
}

func TestBuildHumaAPI_AuthDisabledHasNoSecurityRequirement(t *testing.T) {
	doc := BuildHumaAPI(fiber.New(), false).OpenAPI()

	require.Contains(t, doc.Components.SecuritySchemes, bearerAuthSecurityScheme)
	assert.Empty(t, doc.Security)
}

func TestServeHumaSpec_IsExplicitlyGated(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		wantStatus int
	}{
		{name: "disabled", enabled: false, wantStatus: http.StatusNotFound},
		{name: "enabled", enabled: true, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			api := BuildHumaAPI(app, true)
			ServeHumaSpec(app, api, nil, tc.enabled)

			for _, path := range []string{"/swagger/openapi.json", "/swagger/openapi.yaml", "/swagger/docs"} {
				resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
				require.NoError(t, err)
				assert.NoError(t, resp.Body.Close())
				assert.Equal(t, tc.wantStatus, resp.StatusCode, path)
			}
		})
	}
}
