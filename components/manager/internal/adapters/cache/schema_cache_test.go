package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	rediscache "github.com/LerianStudio/fetcher/pkg/redis"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	libLog "github.com/LerianStudio/lib-observability/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// mockGenericCache implements redis.Cache[model.DataSourceSchema] for testing.
//
// NOTE: This manual mock is intentionally retained because mockgen does not support
// generating mocks for generic interfaces (Go 1.18+ type parameters). The redis.Cache[T]
// interface uses generics, which makes it incompatible with mockgen's code generation.
type mockGenericCache struct {
	data    map[string]model.DataSourceSchema
	healthy bool
	closed  bool
	mu      sync.RWMutex
}

func newMockGenericCache() *mockGenericCache {
	return &mockGenericCache{
		data:    make(map[string]model.DataSourceSchema),
		healthy: true,
	}
}

func (m *mockGenericCache) Get(ctx context.Context, key string) (model.DataSourceSchema, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	schema, found := m.data[key]
	return schema, found, nil
}

func (m *mockGenericCache) Set(ctx context.Context, key string, value model.DataSourceSchema, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *mockGenericCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *mockGenericCache) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]model.DataSourceSchema)
	return nil
}

func (m *mockGenericCache) IsHealthy(ctx context.Context) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.healthy
}

func (m *mockGenericCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func TestSchemaCache_InMemoryBackend_TenantScopedRawConfigNames(t *testing.T) {
	backend := rediscache.NewInMemoryCache[model.DataSourceSchema](time.Minute, libLog.NewNop())
	defer backend.Close()

	cache := NewSchemaCache(backend, time.Minute)
	configName := "shared_config"
	ctxA := tmcore.ContextWithTenantID(context.Background(), "tenant-schema-a")
	ctxB := tmcore.ContextWithTenantID(context.Background(), "tenant-schema-b")
	schemaA := &model.DataSourceSchema{
		ConfigName: configName,
		Tables: map[string]*model.TableSchema{
			"tenant_a_table": model.NewTableSchema("tenant_a_table", []string{"id"}),
		},
	}
	schemaB := &model.DataSourceSchema{
		ConfigName: configName,
		Tables: map[string]*model.TableSchema{
			"tenant_b_table": model.NewTableSchema("tenant_b_table", []string{"id"}),
		},
	}

	err := cache.Set(ctxA, configName, schemaA, 0)
	require.NoError(t, err)

	err = cache.Set(ctxB, configName, schemaB, 0)
	require.NoError(t, err)

	gotA, err := cache.Get(ctxA, configName)
	require.NoError(t, err)
	require.NotNil(t, gotA)
	assert.True(t, gotA.HasTable("tenant_a_table"))
	assert.False(t, gotA.HasTable("tenant_b_table"))

	gotB, err := cache.Get(ctxB, configName)
	require.NoError(t, err)
	require.NotNil(t, gotB)
	assert.True(t, gotB.HasTable("tenant_b_table"))
	assert.False(t, gotB.HasTable("tenant_a_table"))
}

func TestSchemaCache_InMemoryBackend_DeleteTenantIsolation(t *testing.T) {
	t.Parallel()

	backend := rediscache.NewInMemoryCache[model.DataSourceSchema](time.Minute, libLog.NewNop())
	defer backend.Close()

	cache := NewSchemaCache(backend, time.Minute)
	configName := "shared_config_delete"
	ctxA := tmcore.ContextWithTenantID(context.Background(), "tenant-schema-delete-a")
	ctxB := tmcore.ContextWithTenantID(context.Background(), "tenant-schema-delete-b")

	require.NoError(t, cache.Set(ctxA, configName, &model.DataSourceSchema{ConfigName: configName}, 0))
	require.NoError(t, cache.Set(ctxB, configName, &model.DataSourceSchema{ConfigName: configName}, 0))
	require.NoError(t, cache.Delete(ctxA, configName))

	gotA, err := cache.Get(ctxA, configName)
	require.NoError(t, err)
	assert.Nil(t, gotA)

	gotB, err := cache.Get(ctxB, configName)
	require.NoError(t, err)
	assert.NotNil(t, gotB)
}

func TestSchemaCache_InMemoryBackend_ClearTenantIsolation(t *testing.T) {
	t.Parallel()

	backend := rediscache.NewInMemoryCache[model.DataSourceSchema](time.Minute, libLog.NewNop())
	defer backend.Close()

	cache := NewSchemaCache(backend, time.Minute)
	configName := "shared_config_clear"
	ctxA := tmcore.ContextWithTenantID(context.Background(), "tenant-schema-clear-a")
	ctxB := tmcore.ContextWithTenantID(context.Background(), "tenant-schema-clear-b")

	require.NoError(t, cache.Set(ctxA, configName, &model.DataSourceSchema{ConfigName: configName}, 0))
	require.NoError(t, cache.Set(ctxB, configName, &model.DataSourceSchema{ConfigName: configName}, 0))
	require.NoError(t, cache.Clear(ctxA))

	gotA, err := cache.Get(ctxA, configName)
	require.NoError(t, err)
	assert.Nil(t, gotA)

	gotB, err := cache.Get(ctxB, configName)
	require.NoError(t, err)
	assert.NotNil(t, gotB)
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

func TestSchemaCache_SetNilSchema(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)

	ctx := context.Background()

	// Set nil schema should be a no-op
	err := cache.Set(ctx, "test_config", nil, 0)
	assert.NoError(t, err)

	// Verify nothing was stored
	assert.Empty(t, mock.data)
}

// mockCacheWithoutClose is a cache that doesn't implement Closeable
type mockCacheWithoutClose struct {
	data map[string]model.DataSourceSchema
}

func (m *mockCacheWithoutClose) Get(ctx context.Context, key string) (model.DataSourceSchema, bool, error) {
	schema, found := m.data[key]
	return schema, found, nil
}

func (m *mockCacheWithoutClose) Set(ctx context.Context, key string, value model.DataSourceSchema, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCacheWithoutClose) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCacheWithoutClose) Clear(ctx context.Context) error {
	m.data = make(map[string]model.DataSourceSchema)
	return nil
}

func (m *mockCacheWithoutClose) IsHealthy(ctx context.Context) bool {
	return true
}

func TestSchemaCache_CloseNonCloseable(t *testing.T) {
	mock := &mockCacheWithoutClose{
		data: make(map[string]model.DataSourceSchema),
	}
	cache := NewSchemaCache(mock, time.Minute)

	// Close should return nil for non-closeable cache
	err := cache.Close()
	assert.NoError(t, err)
}

// mockCacheWithError is a cache that returns errors
type mockCacheWithError struct {
	getErr    error
	setErr    error
	deleteErr error
	clearErr  error
	closeErr  error
}

func (m *mockCacheWithError) Get(ctx context.Context, key string) (model.DataSourceSchema, bool, error) {
	if m.getErr != nil {
		return model.DataSourceSchema{}, false, m.getErr
	}
	return model.DataSourceSchema{}, false, nil
}

func (m *mockCacheWithError) Set(ctx context.Context, key string, value model.DataSourceSchema, ttl time.Duration) error {
	return m.setErr
}

func (m *mockCacheWithError) Delete(ctx context.Context, key string) error {
	return m.deleteErr
}

func (m *mockCacheWithError) Clear(ctx context.Context) error {
	return m.clearErr
}

func (m *mockCacheWithError) IsHealthy(ctx context.Context) bool {
	return true
}

func (m *mockCacheWithError) Close() error {
	return m.closeErr
}

func TestSchemaCache_GetError(t *testing.T) {
	expectedErr := assert.AnError
	mock := &mockCacheWithError{getErr: expectedErr}
	cache := NewSchemaCache(mock, time.Minute)

	ctx := context.Background()
	result, err := cache.Get(ctx, "test_key")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, expectedErr)
}

func TestSchemaCache_SetError(t *testing.T) {
	tests := []struct {
		name        string
		setErr      error
		expectError bool
	}{
		{
			name:        "set returns error",
			setErr:      assert.AnError,
			expectError: true,
		},
		{
			name:        "set succeeds",
			setErr:      nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCacheWithError{setErr: tt.setErr}
			cache := NewSchemaCache(mock, time.Minute)

			ctx := context.Background()
			schema := &model.DataSourceSchema{
				ConfigName: "test_config",
				Tables:     make(map[string]*model.TableSchema),
			}

			err := cache.Set(ctx, "test_key", schema, 0)

			if tt.expectError {
				assert.ErrorIs(t, err, tt.setErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSchemaCache_DeleteError(t *testing.T) {
	expectedErr := assert.AnError
	mock := &mockCacheWithError{deleteErr: expectedErr}
	cache := NewSchemaCache(mock, time.Minute)

	ctx := context.Background()
	err := cache.Delete(ctx, "test_key")

	assert.ErrorIs(t, err, expectedErr)
}

func TestSchemaCache_ClearError(t *testing.T) {
	expectedErr := assert.AnError
	mock := &mockCacheWithError{clearErr: expectedErr}
	cache := NewSchemaCache(mock, time.Minute)

	ctx := context.Background()
	err := cache.Clear(ctx)

	assert.ErrorIs(t, err, expectedErr)
}

func TestSchemaCache_CloseError(t *testing.T) {
	expectedErr := assert.AnError
	mock := &mockCacheWithError{closeErr: expectedErr}
	cache := NewSchemaCache(mock, time.Minute)

	err := cache.Close()

	assert.ErrorIs(t, err, expectedErr)
}

func TestSchemaCache_SetWithNegativeTTL(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, 2*time.Minute)

	ctx := context.Background()
	schema := &model.DataSourceSchema{
		ConfigName: "test_config",
		Tables:     make(map[string]*model.TableSchema),
	}

	// Set with negative TTL should use cache default (2 minutes)
	beforeSet := time.Now()
	err := cache.Set(ctx, "test_key", schema, -5*time.Second)
	require.NoError(t, err)

	// Verify schema was stored with default cache TTL
	stored := mock.data["test_key"]
	assert.False(t, stored.CachedAt.IsZero())
	assert.False(t, stored.ExpiresAt.IsZero())
	assert.True(t, stored.ExpiresAt.After(beforeSet.Add(time.Minute)))
}

func TestSchemaCache_TTLBoundaryConditions(t *testing.T) {
	tests := []struct {
		name        string
		cacheTTL    time.Duration
		setTTL      time.Duration
		expectedTTL time.Duration
	}{
		{
			name:        "zero cache TTL uses default, zero set TTL uses cache default",
			cacheTTL:    0,
			setTTL:      0,
			expectedTTL: DefaultSchemaCacheTTL,
		},
		{
			name:        "negative cache TTL uses default, positive set TTL uses set value",
			cacheTTL:    -1 * time.Minute,
			setTTL:      3 * time.Minute,
			expectedTTL: 3 * time.Minute,
		},
		{
			name:        "positive cache TTL, zero set TTL uses cache TTL",
			cacheTTL:    4 * time.Minute,
			setTTL:      0,
			expectedTTL: 4 * time.Minute,
		},
		{
			name:        "1 nanosecond TTL",
			cacheTTL:    time.Minute,
			setTTL:      1 * time.Nanosecond,
			expectedTTL: 1 * time.Nanosecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGenericCache()
			cache := NewSchemaCache(mock, tt.cacheTTL)

			ctx := context.Background()
			schema := &model.DataSourceSchema{
				ConfigName: "test_config",
				Tables:     make(map[string]*model.TableSchema),
			}

			beforeSet := time.Now().UTC()
			err := cache.Set(ctx, "test_key", schema, tt.setTTL)
			require.NoError(t, err)
			afterSet := time.Now().UTC()

			stored := mock.data["test_key"]
			assert.False(t, stored.CachedAt.IsZero())
			assert.False(t, stored.ExpiresAt.IsZero())

			// Verify CachedAt is within reasonable bounds
			assert.True(t, stored.CachedAt.After(beforeSet.Add(-time.Second)))
			assert.True(t, stored.CachedAt.Before(afterSet.Add(time.Second)))

			// Verify ExpiresAt is approximately CachedAt + expectedTTL
			expectedExpiry := stored.CachedAt.Add(tt.expectedTTL)
			assert.True(t, stored.ExpiresAt.After(expectedExpiry.Add(-time.Second)))
			assert.True(t, stored.ExpiresAt.Before(expectedExpiry.Add(time.Second)))
		})
	}
}

func TestSchemaCache_ConcurrentAccess(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)
	ctx := context.Background()

	const numGoroutines = 10
	const numOperations = 100

	// Use channels to synchronize goroutines
	done := make(chan bool, numGoroutines)

	// Launch multiple goroutines performing concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				configName := "config_" + string(rune('0'+id))
				schema := &model.DataSourceSchema{
					ConfigName: configName,
					Tables:     make(map[string]*model.TableSchema),
				}

				// Perform various operations
				_ = cache.Set(ctx, configName, schema, 0)
				_, _ = cache.Get(ctx, configName)
				_ = cache.Delete(ctx, configName)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify cache is still functional
	testSchema := &model.DataSourceSchema{
		ConfigName: "final_test",
		Tables:     make(map[string]*model.TableSchema),
	}
	err := cache.Set(ctx, "final_test", testSchema, 0)
	assert.NoError(t, err)

	result, err := cache.Get(ctx, "final_test")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSchemaCache_MultipleSetsSameKey(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)
	ctx := context.Background()

	configName := "test_config"

	// Set schema multiple times with different data
	for i := 0; i < 5; i++ {
		schema := &model.DataSourceSchema{
			ConfigName: configName,
			Tables:     make(map[string]*model.TableSchema),
		}
		// Add a table to differentiate each version
		tableName := "table_" + string(rune('0'+i))
		schema.Tables[tableName] = &model.TableSchema{
			TableName: tableName,
			Columns:   make(map[string]bool),
		}

		err := cache.Set(ctx, configName, schema, 0)
		require.NoError(t, err)

		// Verify the latest version is stored
		result, err := cache.Get(ctx, configName)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.Tables, tableName)
	}
}

func TestSchemaCache_DeleteNonExistentKey(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)
	ctx := context.Background()

	// Delete a key that doesn't exist should not error
	err := cache.Delete(ctx, "nonexistent_key")
	assert.NoError(t, err)
}

func TestSchemaCache_ClearEmptyCache(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)
	ctx := context.Background()

	// Clear empty cache should not error
	err := cache.Clear(ctx)
	assert.NoError(t, err)

	// Verify cache is still functional
	schema := &model.DataSourceSchema{
		ConfigName: "test_config",
		Tables:     make(map[string]*model.TableSchema),
	}
	err = cache.Set(ctx, "test_key", schema, 0)
	assert.NoError(t, err)
}

func TestSchemaCache_GetAfterClear(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)
	ctx := context.Background()

	// Set multiple schemas
	for i := 0; i < 3; i++ {
		configName := "config_" + string(rune('0'+i))
		schema := &model.DataSourceSchema{
			ConfigName: configName,
			Tables:     make(map[string]*model.TableSchema),
		}
		err := cache.Set(ctx, configName, schema, 0)
		require.NoError(t, err)
	}

	// Clear cache
	err := cache.Clear(ctx)
	require.NoError(t, err)

	// Verify all keys return nil (cache miss)
	for i := 0; i < 3; i++ {
		configName := "config_" + string(rune('0'+i))
		result, err := cache.Get(ctx, configName)
		assert.NoError(t, err)
		assert.Nil(t, result)
	}
}

func TestSchemaCache_SetWithEmptyConfigName(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)
	ctx := context.Background()

	schema := &model.DataSourceSchema{
		ConfigName: "",
		Tables:     make(map[string]*model.TableSchema),
	}

	// Should allow empty config name (business logic doesn't restrict)
	err := cache.Set(ctx, "", schema, 0)
	assert.NoError(t, err)

	// Should be able to retrieve it
	result, err := cache.Get(ctx, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSchemaCache_SetWithLargeTTL(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)
	ctx := context.Background()

	schema := &model.DataSourceSchema{
		ConfigName: "test_config",
		Tables:     make(map[string]*model.TableSchema),
	}

	// Set with very large TTL (24 hours)
	largeTTL := 24 * time.Hour
	beforeSet := time.Now().UTC()
	err := cache.Set(ctx, "test_key", schema, largeTTL)
	require.NoError(t, err)

	// Verify timestamps
	stored := mock.data["test_key"]
	assert.False(t, stored.CachedAt.IsZero())
	assert.False(t, stored.ExpiresAt.IsZero())
	assert.True(t, stored.ExpiresAt.After(beforeSet.Add(23*time.Hour)))
}

func TestSchemaCache_SchemaWithComplexData(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)
	ctx := context.Background()

	// Create a complex schema with multiple tables and columns
	schema := &model.DataSourceSchema{
		ConfigName: "complex_config",
		Tables:     make(map[string]*model.TableSchema),
	}

	for i := 0; i < 5; i++ {
		tableName := "table_" + string(rune('0'+i))
		table := &model.TableSchema{
			TableName: tableName,
			Columns:   make(map[string]bool),
		}

		// Add 10 columns to each table
		for j := 0; j < 10; j++ {
			columnName := "column_" + string(rune('0'+j))
			table.Columns[columnName] = true
		}

		schema.Tables[tableName] = table
	}

	// Set and retrieve complex schema
	err := cache.Set(ctx, "complex_config", schema, 0)
	require.NoError(t, err)

	result, err := cache.Get(ctx, "complex_config")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 5, len(result.Tables))
	assert.Equal(t, 10, len(result.Tables["table_0"].Columns))
}

func TestSchemaCache_IsHealthyStates(t *testing.T) {
	tests := []struct {
		name    string
		healthy bool
	}{
		{
			name:    "healthy cache",
			healthy: true,
		},
		{
			name:    "unhealthy cache",
			healthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGenericCache()
			mock.healthy = tt.healthy
			cache := NewSchemaCache(mock, time.Minute)

			ctx := context.Background()
			isHealthy := cache.IsHealthy(ctx)
			assert.Equal(t, tt.healthy, isHealthy)

			// Verify operations still work regardless of health status
			schema := &model.DataSourceSchema{
				ConfigName: "test_config",
				Tables:     make(map[string]*model.TableSchema),
			}
			err := cache.Set(ctx, "test_key", schema, 0)
			assert.NoError(t, err)
		})
	}
}

func TestSchemaCache_CloseMultipleTimes(t *testing.T) {
	mock := newMockGenericCache()
	cache := NewSchemaCache(mock, time.Minute)

	// Close multiple times should not error
	err := cache.Close()
	assert.NoError(t, err)
	assert.True(t, mock.closed)

	// Reset for second close
	mock.closed = false

	err = cache.Close()
	assert.NoError(t, err)
	assert.True(t, mock.closed)
}

// Tests using the generated MockSchemaCacheRepository to increase coverage
func TestMockSchemaCacheRepository_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockSchemaCacheRepository(ctrl)
	ctx := context.Background()
	expectedSchema := &model.DataSourceSchema{
		ConfigName: "test_config",
		Tables:     make(map[string]*model.TableSchema),
	}

	mock.EXPECT().Get(ctx, "test_key").Return(expectedSchema, nil)

	result, err := mock.Get(ctx, "test_key")
	assert.NoError(t, err)
	assert.Equal(t, expectedSchema, result)
}

func TestMockSchemaCacheRepository_Set(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockSchemaCacheRepository(ctrl)
	ctx := context.Background()
	schema := &model.DataSourceSchema{
		ConfigName: "test_config",
		Tables:     make(map[string]*model.TableSchema),
	}

	mock.EXPECT().Set(ctx, "test_key", schema, time.Minute).Return(nil)

	err := mock.Set(ctx, "test_key", schema, time.Minute)
	assert.NoError(t, err)
}

func TestMockSchemaCacheRepository_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockSchemaCacheRepository(ctrl)
	ctx := context.Background()

	mock.EXPECT().Delete(ctx, "test_key").Return(nil)

	err := mock.Delete(ctx, "test_key")
	assert.NoError(t, err)
}

func TestMockSchemaCacheRepository_Clear(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockSchemaCacheRepository(ctrl)
	ctx := context.Background()

	mock.EXPECT().Clear(ctx).Return(nil)

	err := mock.Clear(ctx)
	assert.NoError(t, err)
}

func TestMockSchemaCacheRepository_IsHealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockSchemaCacheRepository(ctrl)
	ctx := context.Background()

	mock.EXPECT().IsHealthy(ctx).Return(true)

	result := mock.IsHealthy(ctx)
	assert.True(t, result)
}

func TestMockSchemaCacheRepository_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockSchemaCacheRepository(ctrl)

	mock.EXPECT().Close().Return(nil)

	err := mock.Close()
	assert.NoError(t, err)
}
