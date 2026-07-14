package in

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LerianStudio/lib-commons/v5/commons/net/http/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHumaRawJSONOperations_EmptyBodyPreservesFET0001(t *testing.T) {
	app, _ := newHumaContractTestAPI(t, false)

	tests := []struct {
		name    string
		method  string
		path    string
		headers map[string]string
	}{
		{
			name:    "create connection",
			method:  http.MethodPost,
			path:    "/v1/management/connections",
			headers: map[string]string{"X-Product-Name": "test-product"},
		},
		{
			name:   "validate schema",
			method: http.MethodPost,
			path:   "/v1/management/connections/validate-schema",
		},
		{
			name:   "update connection",
			method: http.MethodPatch,
			path:   "/v1/management/connections/" + uuid.NewString(),
		},
		{
			name:   "create fetcher job",
			method: http.MethodPost,
			path:   "/v1/fetcher",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			req.Header.Set("Content-Type", "application/json")
			for name, value := range tt.headers {
				req.Header.Set(name, value)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode, string(body))
			assert.Contains(t, resp.Header.Get("Content-Type"), "application/problem+json")

			var detail problem.Detail
			require.NoError(t, json.Unmarshal(body, &detail), "problem body: %s", body)
			assert.Equal(t, problem.BaseURI+"/FET-0001", detail.Type)
			assert.Equal(t, "Bad Request", detail.Title)
			assert.Equal(t, http.StatusBadRequest, detail.Status)
			assert.Equal(t, "unable to parse request body", detail.Detail)
			assert.Equal(t, "FET-0001", detail.Code)
			assert.Empty(t, detail.Errors)
		})
	}
}

func TestHumaProtectedOperations_DocumentBearerProblems(t *testing.T) {
	_, api := newHumaContractTestAPI(t, true)
	require.Equal(t, []map[string][]string{{"BearerAuth": {}}}, api.OpenAPI().Security)

	scheme, ok := api.OpenAPI().Components.SecuritySchemes["BearerAuth"]
	require.True(t, ok)
	require.NotNil(t, scheme)
	assert.Equal(t, "http", scheme.Type)
	assert.Equal(t, "bearer", scheme.Scheme)
	assert.Equal(t, "JWT", scheme.BearerFormat)
	assert.Equal(t, "JWT bearer token issued by the identity provider.", scheme.Description)

	operations := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/management/connections"},
		{http.MethodGet, "/v1/management/connections"},
		{http.MethodPost, "/v1/management/connections/validate-schema"},
		{http.MethodGet, "/v1/management/connections/unassigned"},
		{http.MethodPost, "/v1/management/connections/{id}/assign"},
		{http.MethodGet, "/v1/management/connections/{id}"},
		{http.MethodPost, "/v1/management/connections/{id}/test"},
		{http.MethodGet, "/v1/management/connections/{id}/schema"},
		{http.MethodPatch, "/v1/management/connections/{id}"},
		{http.MethodDelete, "/v1/management/connections/{id}"},
		{http.MethodPost, "/v1/fetcher"},
		{http.MethodGet, "/v1/fetcher/{id}"},
	}

	for _, item := range operations {
		operation := operationAt(api, item.method, item.path)
		if !assert.NotNil(t, operation, "%s %s is missing", item.method, item.path) {
			continue
		}
		assert.Nil(t, operation.Security, "%s %s must inherit global BearerAuth", item.method, item.path)

		for _, status := range []string{"401", "403"} {
			response, ok := operation.Responses[status]
			if !assert.True(t, ok, "%s %s is missing response %s", item.method, item.path, status) {
				continue
			}
			mediaType, ok := response.Content["application/problem+json"]
			if !assert.True(t, ok, "%s %s response %s must use application/problem+json", item.method, item.path, status) {
				continue
			}
			require.NotNil(t, mediaType.Schema)
			assert.Equal(t, "#/components/schemas/Detail", mediaType.Schema.Ref)
		}
	}
}

func TestHumaSSLInput_DocumentsClientCredentialsAsOptional(t *testing.T) {
	_, api := newHumaContractTestAPI(t, false)

	schema := api.OpenAPI().Components.Schemas.Map()["SSLInput"]
	require.NotNil(t, schema)
	assert.NotContains(t, schema.Required, "cert")
	assert.NotContains(t, schema.Required, "key")
}

func TestHumaConnectionLists_DocumentOnlyEffectiveFilters(t *testing.T) {
	_, api := newHumaContractTestAPI(t, false)

	list := operationAt(api, http.MethodGet, "/v1/management/connections")
	require.NotNil(t, list)
	assert.ElementsMatch(t, []string{
		"header:X-Product-Name",
		"query:page",
		"query:limit",
		"query:sortOrder",
		"query:type",
		"query:startDate",
		"query:endDate",
	}, operationParameterKeys(list))
	assert.NotContains(t, operationParameterKeys(list), "query:cursor")
	assert.NotContains(t, operationParameterKeys(list), "query:host")
	assert.NotContains(t, operationParameterKeys(list), "query:databaseName")
	assert.False(t, hasMetadataParameter(list))
	assert.Contains(t, list.Description, "metadata.region=br")

	unassigned := operationAt(api, http.MethodGet, "/v1/management/connections/unassigned")
	require.NotNil(t, unassigned)
	assert.ElementsMatch(t, []string{
		"query:page",
		"query:limit",
		"query:sortOrder",
		"query:startDate",
		"query:endDate",
	}, operationParameterKeys(unassigned))
	assert.NotContains(t, operationParameterKeys(unassigned), "query:cursor")
	assert.NotContains(t, operationParameterKeys(unassigned), "query:type")
	assert.False(t, hasMetadataParameter(unassigned))
	assert.NotContains(t, strings.ToLower(unassigned.Description), "cursor")
	assert.NotContains(t, strings.ToLower(unassigned.Description), "metadata")
}

func TestFetcherRequestSchema_DocumentsRequiredMetadataSource(t *testing.T) {
	_, api := newHumaContractTestAPI(t, false)
	document := openAPIDocumentMap(t, api)

	components := document["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	fetcherRequest := schemas["FetcherRequest"].(map[string]any)
	assert.ElementsMatch(t, []string{"dataRequest", "metadata"}, jsonStringSlice(fetcherRequest["required"]))

	properties := fetcherRequest["properties"].(map[string]any)
	metadata := properties["metadata"].(map[string]any)
	assert.ElementsMatch(t, []string{"source"}, jsonStringSlice(metadata["required"]))
	assert.Contains(t, metadata, "additionalProperties")
	assert.NotEqual(t, false, metadata["additionalProperties"])

	metadataProperties, ok := metadata["properties"].(map[string]any)
	if !assert.True(t, ok, "metadata properties are missing") {
		return
	}
	source, ok := metadataProperties["source"].(map[string]any)
	if !assert.True(t, ok, "metadata.source is missing") {
		return
	}
	assert.Equal(t, "string", source["type"])
	assert.NotEmpty(t, source["description"])
	assert.NotEmpty(t, source["examples"])
}

func newHumaContractTestAPI(t *testing.T, authEnabled bool) (*fiber.App, huma.API) {
	t.Helper()

	problem.Install()
	app := setupConnectionTestApp()
	api := BuildHumaAPI(app, authEnabled)
	handlers := NewOperationHandlers(&ConnectionHandler{}, &MigrationHandler{}, &FetcherHandler{})
	RegisterHumaOperations(app, api, handlers, nil)

	return app, api
}

func operationParameterKeys(operation *huma.Operation) []string {
	keys := make([]string, 0, len(operation.Parameters))
	for _, parameter := range operation.Parameters {
		keys = append(keys, parameter.In+":"+parameter.Name)
	}

	return keys
}

func hasMetadataParameter(operation *huma.Operation) bool {
	for _, parameter := range operation.Parameters {
		if strings.Contains(strings.ToLower(parameter.Name), "metadata") {
			return true
		}
	}

	return false
}

func jsonStringSlice(value any) []string {
	items, _ := value.([]any)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}

	return result
}
