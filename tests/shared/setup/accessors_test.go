package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockContainer struct {
	port string
	host string
	uri  string
}

func (m *mockContainer) GetPort() string { return m.port }
func (m *mockContainer) GetHost() string { return m.host }
func (m *mockContainer) GetURI() string  { return m.uri }

func TestGetPort_WithContainer(t *testing.T) {
	container := &mockContainer{port: "5432", host: "localhost"}
	port := GetPort(container)
	assert.Equal(t, "5432", port)
}

func TestGetPort_NilContainer(t *testing.T) {
	port := GetPort(nil)
	assert.Equal(t, "", port)
}

func TestGetPort_NoPortProvider(t *testing.T) {
	// A struct that doesn't implement PortProvider
	type noPort struct{}
	port := GetPort(&noPort{})
	assert.Equal(t, "", port)
}

func TestGetHost_WithContainer(t *testing.T) {
	container := &mockContainer{port: "5432", host: "localhost"}
	host := GetHost(container)
	assert.Equal(t, "localhost", host)
}

func TestGetHost_NilContainer(t *testing.T) {
	host := GetHost(nil)
	assert.Equal(t, "", host)
}

func TestGetHost_NoHostProvider(t *testing.T) {
	type noHost struct{}
	host := GetHost(&noHost{})
	assert.Equal(t, "", host)
}

func TestGetURI_WithContainer(t *testing.T) {
	container := &mockContainer{uri: "mongodb://localhost:27017"}
	uri := GetURI(container)
	assert.Equal(t, "mongodb://localhost:27017", uri)
}

func TestGetURI_NilContainer(t *testing.T) {
	uri := GetURI(nil)
	assert.Equal(t, "", uri)
}

func TestGetURI_NoURIProvider(t *testing.T) {
	type noURI struct{}
	uri := GetURI(&noURI{})
	assert.Equal(t, "", uri)
}

// Integration tests with actual container type patterns.
// These verify that the accessor interfaces work with the real container structs.

// PostgresContainerLike simulates the real PostgresContainer structure
type PostgresContainerLike struct {
	Host string
	Port string
	URL  string
}

func (p *PostgresContainerLike) GetHost() string { return p.Host }
func (p *PostgresContainerLike) GetPort() string { return p.Port }
func (p *PostgresContainerLike) GetURI() string  { return p.URL }

func TestAccessors_IntegrationWithContainerTypes(t *testing.T) {
	// Simulate a Postgres-like container
	postgres := &PostgresContainerLike{
		Host: "localhost",
		Port: "5432",
		URL:  "postgres://localhost:5432/testdb",
	}

	assert.Equal(t, "localhost", GetHost(postgres))
	assert.Equal(t, "5432", GetPort(postgres))
	assert.Equal(t, "postgres://localhost:5432/testdb", GetURI(postgres))
}

func TestAccessors_MixedContainerTypes(t *testing.T) {
	// Test that different container types all work through the same interface
	containers := []any{
		&PostgresContainerLike{Host: "pg-host", Port: "5432", URL: "pg-uri"},
		&mockContainer{host: "mock-host", port: "27017", uri: "mock-uri"},
	}

	for _, c := range containers {
		// All should return non-empty values
		assert.NotEmpty(t, GetHost(c))
		assert.NotEmpty(t, GetPort(c))
		assert.NotEmpty(t, GetURI(c))
	}
}
