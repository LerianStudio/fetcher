package redis

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisConnection(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() (*miniredis.Miniredis, RedisConfig)
		wantErr   bool
		wantConn  bool
		errSubstr string
	}{
		{
			name: "valid_config_returns_connection",
			setup: func() (*miniredis.Miniredis, RedisConfig) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				host, port := mr.Host(), mr.Port()
				return mr, RedisConfig{
					Host:     host,
					Port:     port,
					Password: "",
					DB:       0,
				}
			},
			wantErr:  false,
			wantConn: true,
		},
		{
			name: "invalid_host_returns_error",
			setup: func() (*miniredis.Miniredis, RedisConfig) {
				return nil, RedisConfig{
					Host:     "invalid-host-that-does-not-exist",
					Port:     "6379",
					Password: "",
					DB:       0,
				}
			},
			wantErr:   true,
			wantConn:  false,
			errSubstr: "failed to connect to Redis",
		},
		{
			name: "invalid_port_returns_error",
			setup: func() (*miniredis.Miniredis, RedisConfig) {
				return nil, RedisConfig{
					Host:     "localhost",
					Port:     "99999",
					Password: "",
					DB:       0,
				}
			},
			wantErr:   true,
			wantConn:  false,
			errSubstr: "failed to connect to Redis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr, cfg := tt.setup()
			if mr != nil {
				defer mr.Close()
			}

			conn, err := NewRedisConnection(cfg, &mockLogger{})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, conn)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, conn)
				assert.True(t, conn.Connected)
				assert.NotNil(t, conn.Client)
				assert.NotNil(t, conn.Logger)
				conn.Client.Close()
			}
		})
	}
}

func TestRedisConnection_Close(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *RedisConnection
		wantErr bool
	}{
		{
			name: "closes_client_successfully",
			setup: func() *RedisConnection {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				client := redis.NewClient(&redis.Options{
					Addr: mr.Addr(),
				})
				return &RedisConnection{
					Client:    client,
					Logger:    &mockLogger{},
					Connected: true,
				}
			},
			wantErr: false,
		},
		{
			name: "nil_client_returns_no_error",
			setup: func() *RedisConnection {
				return &RedisConnection{
					Client:    nil,
					Logger:    &mockLogger{},
					Connected: false,
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := tt.setup()

			err := conn.Close()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if conn.Client != nil {
					assert.False(t, conn.Connected)
				}
			}
		})
	}
}

func TestRedisConnection_Close_SetsConnectedFalse(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	conn := &RedisConnection{
		Client:    client,
		Logger:    &mockLogger{},
		Connected: true,
	}

	assert.True(t, conn.Connected)

	err = conn.Close()
	require.NoError(t, err)

	assert.False(t, conn.Connected)
}

func TestRedisConnection_IsConnected(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (*RedisConnection, func())
		want  bool
	}{
		{
			name: "healthy_client_returns_true",
			setup: func(t *testing.T) (*RedisConnection, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				client := redis.NewClient(&redis.Options{
					Addr: mr.Addr(),
				})
				return &RedisConnection{
						Client:    client,
						Logger:    &mockLogger{},
						Connected: true,
					}, func() {
						client.Close()
						mr.Close()
					}
			},
			want: true,
		},
		{
			name: "closed_server_returns_false",
			setup: func(t *testing.T) (*RedisConnection, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				client := redis.NewClient(&redis.Options{
					Addr: mr.Addr(),
				})
				mr.Close() // Close server before test
				return &RedisConnection{
						Client:    client,
						Logger:    &mockLogger{},
						Connected: true,
					}, func() {
						client.Close()
					}
			},
			want: false,
		},
		{
			name: "nil_client_returns_false",
			setup: func(t *testing.T) (*RedisConnection, func()) {
				return &RedisConnection{
					Client:    nil,
					Logger:    &mockLogger{},
					Connected: false,
				}, func() {}
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, cleanup := tt.setup(t)
			defer cleanup()

			got := conn.IsConnected()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisConnection_IsConnected_AfterClose(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	conn := &RedisConnection{
		Client:    client,
		Logger:    &mockLogger{},
		Connected: true,
	}

	// Should be connected initially
	assert.True(t, conn.IsConnected())

	// Close the client
	err = conn.Close()
	require.NoError(t, err)

	// Should not be connected after close
	assert.False(t, conn.IsConnected())
}

func TestRedisConfig_WithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    RedisConfig
		expected RedisConfig
	}{
		{
			name:  "all_zero_values_get_defaults",
			input: RedisConfig{Host: "localhost", Port: "6379"},
			expected: RedisConfig{
				Host:         "localhost",
				Port:         "6379",
				PoolSize:     DefaultPoolSize,
				MinIdleConns: DefaultMinIdleConns,
				DialTimeout:  DefaultDialTimeout,
				ReadTimeout:  DefaultReadTimeout,
				WriteTimeout: DefaultWriteTimeout,
			},
		},
		{
			name: "custom_values_preserved",
			input: RedisConfig{
				Host:         "localhost",
				Port:         "6379",
				PoolSize:     20,
				MinIdleConns: 5,
				DialTimeout:  10 * time.Second,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
			expected: RedisConfig{
				Host:         "localhost",
				Port:         "6379",
				PoolSize:     20,
				MinIdleConns: 5,
				DialTimeout:  10 * time.Second,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
		{
			name: "partial_custom_values",
			input: RedisConfig{
				Host:        "localhost",
				Port:        "6379",
				PoolSize:    50,
				DialTimeout: 15 * time.Second,
			},
			expected: RedisConfig{
				Host:         "localhost",
				Port:         "6379",
				PoolSize:     50,
				MinIdleConns: DefaultMinIdleConns,
				DialTimeout:  15 * time.Second,
				ReadTimeout:  DefaultReadTimeout,
				WriteTimeout: DefaultWriteTimeout,
			},
		},
		{
			name: "negative_values_get_defaults",
			input: RedisConfig{
				Host:         "localhost",
				Port:         "6379",
				PoolSize:     -1,
				MinIdleConns: -5,
				DialTimeout:  -1 * time.Second,
			},
			expected: RedisConfig{
				Host:         "localhost",
				Port:         "6379",
				PoolSize:     DefaultPoolSize,
				MinIdleConns: DefaultMinIdleConns,
				DialTimeout:  DefaultDialTimeout,
				ReadTimeout:  DefaultReadTimeout,
				WriteTimeout: DefaultWriteTimeout,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.WithDefaults()

			assert.Equal(t, tt.expected.Host, result.Host)
			assert.Equal(t, tt.expected.Port, result.Port)
			assert.Equal(t, tt.expected.PoolSize, result.PoolSize)
			assert.Equal(t, tt.expected.MinIdleConns, result.MinIdleConns)
			assert.Equal(t, tt.expected.DialTimeout, result.DialTimeout)
			assert.Equal(t, tt.expected.ReadTimeout, result.ReadTimeout)
			assert.Equal(t, tt.expected.WriteTimeout, result.WriteTimeout)
		})
	}
}

func TestNewRedisConnection_UsesConfigValues(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cfg := RedisConfig{
		Host:         mr.Host(),
		Port:         mr.Port(),
		PoolSize:     25,
		MinIdleConns: 3,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  6 * time.Second,
		WriteTimeout: 6 * time.Second,
	}

	conn, err := NewRedisConnection(cfg, &mockLogger{})
	require.NoError(t, err)
	defer conn.Close()

	// Verify connection was created successfully with custom config
	assert.True(t, conn.Connected)
	assert.True(t, conn.IsConnected())
}
