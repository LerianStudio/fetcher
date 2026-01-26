package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestBuilder_Post(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-org", r.Header.Get("X-Organization-Id"))

		body, _ := io.ReadAll(r.Body)
		var input map[string]string
		json.Unmarshal(body, &input)
		assert.Equal(t, "value", input["key"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"result": "ok"})
	}))
	defer server.Close()

	builder := NewRequestBuilder(http.DefaultClient)
	var result map[string]string

	err := builder.
		Post(server.URL+"/test").
		WithOrganizationID("test-org").
		WithJSONBody(map[string]string{"key": "value"}).
		ExpectStatus(http.StatusCreated).
		Execute(context.Background(), &result)

	require.NoError(t, err)
	assert.Equal(t, "ok", result["result"])
}

func TestRequestBuilder_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"data": "test"})
	}))
	defer server.Close()

	builder := NewRequestBuilder(http.DefaultClient)
	var result map[string]string

	err := builder.
		Get(server.URL+"/test").
		ExpectStatus(http.StatusOK).
		Execute(context.Background(), &result)

	require.NoError(t, err)
	assert.Equal(t, "test", result["data"])
}

func TestRequestBuilder_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	builder := NewRequestBuilder(http.DefaultClient)

	err := builder.
		Delete(server.URL+"/test").
		ExpectStatus(http.StatusNoContent).
		Execute(context.Background(), nil)

	require.NoError(t, err)
}

func TestRequestBuilder_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "validation failed"}`))
	}))
	defer server.Close()

	builder := NewRequestBuilder(http.DefaultClient)

	err := builder.
		Post(server.URL+"/test").
		ExpectStatus(http.StatusCreated).
		Execute(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "validation failed")
}

func TestRequestBuilder_MarshalError(t *testing.T) {
	// Create a type that cannot be marshaled to JSON
	type unmarshalable struct {
		Ch chan int
	}

	builder := NewRequestBuilder(http.DefaultClient)

	err := builder.
		Post("http://example.com/test").
		WithJSONBody(unmarshalable{Ch: make(chan int)}).
		Execute(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal request body")
}
