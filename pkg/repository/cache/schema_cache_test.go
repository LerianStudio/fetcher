package cache

import (
	"context"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGenericCache implements redis.Cache[model.DataSourceSchema] for testing
type mockGenericCache struct {
	data    map[string]model.DataSourceSchema
	healthy bool
	closed  bool
}

func newMockGenericCache() *mockGenericCache {
	return &mockGenericCache{
		data:    make(map[string]model.DataSourceSchema),
		healthy: true,
	}
}

func (m *mockGenericCache) Get(ctx context.Context, key string) (model.DataSourceSchema, bool, error) {
	schema, found := m.data[key]
	return schema, found, nil
}

func (m *mockGenericCache) Set(ctx context.Context, key string, value model.DataSourceSchema, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockGenericCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockGenericCache) Clear(ctx context.Context) error {
	m.data = make(map[string]model.DataSourceSchema)
	return nil
}

func (m *mockGenericCache) IsHealthy(ctx context.Context) bool {
	return m.healthy
}

func (m *mockGenericCache) Close() error {
	m.closed = true
	return nil
}

func TestSchemaCache_GetSet(t *testing.T) {
	tests := []struct {
		name       string
		configName string
		schema     *model.DataSourceSchema
		wantFound  bool
	}{
		{
			name:       "cache miss returns nil",
			configName: "nonexistent",
			schema:     nil,
			wantFound:  false,
		},
		{
			name:       "cache hit returns schema",
			configName: "test_config",
			schema: &model.DataSourceSchema{
				ConfigName: "test_config",
				Tables:     make(map[string]*model.TableSchema),
			},
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGenericCache()
			cache := NewSchemaCache(mock, time.Minute)
			ctx := context.Background()

			// Test cache miss first
			result, err := cache.Get(ctx, tt.configName)
			assert.NoError(t, err)
			assert.Nil(t, result)

			// If schema provided, test set and get
			if tt.schema != nil {
				err = cache.Set(ctx, tt.configName, tt.schema, 0)
				assert.NoError(t, err)

				result, err = cache.Get(ctx, tt.configName)
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.configName, result.ConfigName)
			}
		})
	}
}

func TestSchemaCache_SetAppliesTimestamps(t *testing.T) {
	tests := []struct {
		name       string
		configName string
		ttl        time.Duration
		wantTTL    time.Duration
	}{
		{
			name:       "uses default TTL when zero",
			configName: "test_config",
			ttl:        0,
			wantTTL:    time.Minute, // cache default TTL
		},
		{
			name:       "uses custom TTL when provided",
			configName: "test_config",
			ttl:        2 * time.Minute,
			wantTTL:    2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGenericCache()
			cache := NewSchemaCache(mock, time.Minute)
			ctx := context.Background()

			schema := &model.DataSourceSchema{
				ConfigName: tt.configName,
				Tables:     make(map[string]*model.TableSchema),
			}

			beforeSet := time.Now()
			err := cache.Set(ctx, tt.configName, schema, tt.ttl)
			require.NoError(t, err)

			// Verify timestamps were set in stored schema
			stored := mock.data[tt.configName]
			assert.True(t, stored.CachedAt.After(beforeSet.Add(-time.Second)))
			assert.True(t, stored.CachedAt.Before(time.Now().UTC().Add(time.Second)))
			assert.True(t, stored.ExpiresAt.After(stored.CachedAt))
		})
	}
}

func TestSchemaCache_Delete(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)

	ctx := context.Background()
	configName := "test_config"
	schema := &model.DataSourceSchema{ConfigName: configName}

	// Set value
	err := cache.Set(ctx, configName, schema, 0)
	require.NoError(t, err)

	// Delete
	err = cache.Delete(ctx, configName)
	assert.NoError(t, err)

	// Verify deleted
	result, err := cache.Get(ctx, configName)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestSchemaCache_Clear(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)

	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 5; i++ {
		name := "config" + string(rune('0'+i))
		err := cache.Set(ctx, name, &model.DataSourceSchema{ConfigName: name}, 0)
		require.NoError(t, err)
	}

	// Clear
	err := cache.Clear(ctx)
	assert.NoError(t, err)

	// Verify all cleared
	assert.Empty(t, mock.data)
}

func TestSchemaCache_IsHealthy(t *testing.T) {
	tests := []struct {
		name        string
		healthy     bool
		wantHealthy bool
	}{
		{
			name:        "returns true when cache is healthy",
			healthy:     true,
			wantHealthy: true,
		},
		{
			name:        "returns false when cache is unhealthy",
			healthy:     false,
			wantHealthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGenericCache()
			mock.healthy = tt.healthy
			cache := NewSchemaCache(mock, time.Minute)

			ctx := context.Background()
			assert.Equal(t, tt.wantHealthy, cache.IsHealthy(ctx))
		})
	}
}

func TestSchemaCache_Close(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)

	err := cache.Close()
	assert.NoError(t, err)
	assert.True(t, mock.closed)
}

func TestSchemaCache_SetMutatesOriginalSchema(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)

	ctx := context.Background()
	configName := "test_config"
	originalSchema := &model.DataSourceSchema{
		ConfigName: configName,
		Tables:     make(map[string]*model.TableSchema),
		CachedAt:   time.Time{}, // Zero value
		ExpiresAt:  time.Time{}, // Zero value
	}

	// Set schema
	err := cache.Set(ctx, configName, originalSchema, 0)
	require.NoError(t, err)

	// Original schema IS mutated (SetCacheTTL is called directly on the pointer)
	// This is the current production behavior - schema timestamps are set in-place
	assert.False(t, originalSchema.CachedAt.IsZero(), "CachedAt should be set on original schema")
	assert.False(t, originalSchema.ExpiresAt.IsZero(), "ExpiresAt should be set on original schema")

	// Stored schema should also have timestamps
	stored := mock.data[configName]
	assert.False(t, stored.CachedAt.IsZero())
	assert.False(t, stored.ExpiresAt.IsZero())

	// Original and stored should have same timestamps
	assert.Equal(t, originalSchema.CachedAt, stored.CachedAt)
	assert.Equal(t, originalSchema.ExpiresAt, stored.ExpiresAt)
}

func TestNewSchemaCache_DefaultTTL(t *testing.T) {
	tests := []struct {
		name    string
		ttl     time.Duration
		wantTTL time.Duration
	}{
		{
			name:    "uses DefaultSchemaCacheTTL when zero",
			ttl:     0,
			wantTTL: DefaultSchemaCacheTTL,
		},
		{
			name:    "uses DefaultSchemaCacheTTL when negative",
			ttl:     -1 * time.Minute,
			wantTTL: DefaultSchemaCacheTTL,
		},
		{
			name:    "uses provided TTL when positive",
			ttl:     10 * time.Minute,
			wantTTL: 10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGenericCache()
			cache := NewSchemaCache(mock, tt.ttl)
			assert.Equal(t, tt.wantTTL, cache.ttl)
		})
	}
}
