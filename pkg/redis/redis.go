package redis

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"time"

	libRedis "github.com/LerianStudio/lib-commons/v5/commons/redis"
	libLog "github.com/LerianStudio/lib-observability/log"
	"github.com/redis/go-redis/v9"
)

// Default values for Redis connection pool.
const (
	DefaultPoolSize     = 10
	DefaultMinIdleConns = 2
	DefaultDialTimeout  = 5 * time.Second
	DefaultReadTimeout  = 3 * time.Second
	DefaultWriteTimeout = 3 * time.Second
)

// RedisConfig holds the configuration for Redis connection.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int

	// TLS settings (optional - disabled by default)
	UseTLS bool   // Enable TLS for Redis connection
	CACert string // Base64-encoded CA certificate for TLS verification

	// Connection pool settings (optional - uses defaults if zero)
	PoolSize     int           // Default: 10
	MinIdleConns int           // Default: 2
	DialTimeout  time.Duration // Default: 5s
	ReadTimeout  time.Duration // Default: 3s
	WriteTimeout time.Duration // Default: 3s
}

// WithDefaults returns a copy of the config with default values applied
// for any zero-value fields.
func (c RedisConfig) WithDefaults() RedisConfig {
	cfg := c
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = DefaultPoolSize
	}

	if cfg.MinIdleConns <= 0 {
		cfg.MinIdleConns = DefaultMinIdleConns
	}

	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = DefaultDialTimeout
	}

	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = DefaultReadTimeout
	}

	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = DefaultWriteTimeout
	}

	return cfg
}

// RedisConnection manages the Redis client connection.
type RedisConnection struct {
	Client          *redis.Client
	CanonicalClient *libRedis.Client
	Logger          libLog.Logger
	Connected       bool
}

// NewRedisConnection creates a new Redis connection.
// Configuration values default to sensible values if not specified.
// Use RedisConfig.WithDefaults() explicitly if you want to see the resolved values.
func NewRedisConnection(cfg RedisConfig, logger libLog.Logger) (*RedisConnection, error) {
	cfg = cfg.WithDefaults()
	addr := buildRedisAddr(cfg.Host, cfg.Port)

	canonicalCfg := libRedis.Config{
		Topology: libRedis.Topology{Standalone: &libRedis.StandaloneTopology{Address: addr}},
		Auth:     libRedis.Auth{StaticPassword: &libRedis.StaticPasswordAuth{Password: cfg.Password}},
		Options: libRedis.ConnectionOptions{
			DB:              cfg.DB,
			PoolSize:        cfg.PoolSize,
			MinIdleConns:    cfg.MinIdleConns,
			DialTimeout:     cfg.DialTimeout,
			ReadTimeout:     cfg.ReadTimeout,
			WriteTimeout:    cfg.WriteTimeout,
			MinRetryBackoff: 8 * time.Millisecond,
			MaxRetryBackoff: 512 * time.Millisecond,
		},
		Logger: logger,
	}

	if cfg.UseTLS {
		if cfg.CACert != "" {
			caBytes, err := base64.StdEncoding.DecodeString(cfg.CACert)
			if err != nil {
				return nil, fmt.Errorf("failed to decode Redis CA certificate: %w", err)
			}

			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(caBytes) {
				return nil, fmt.Errorf("failed to parse Redis CA certificate")
			}
		}

		canonicalCfg.TLS = &libRedis.TLSConfig{CACertBase64: cfg.CACert}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	canonicalClient, err := libRedis.New(ctx, canonicalCfg)
	if err != nil {
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to connect to Redis at %s: %v", addr, err))
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	universalClient, err := canonicalClient.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis client: %w", err)
	}

	client, ok := universalClient.(*redis.Client)
	if !ok {
		_ = canonicalClient.Close()
		return nil, fmt.Errorf("unexpected Redis client type %T for standalone cache connection", universalClient)
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Successfully connected to Redis at %s", addr))

	return &RedisConnection{
		Client:          client,
		CanonicalClient: canonicalClient,
		Logger:          logger,
		Connected:       true,
	}, nil
}

// Close closes the Redis connection.
func (r *RedisConnection) Close() error {
	if r.CanonicalClient != nil {
		r.Logger.Log(context.Background(), libLog.LevelInfo, "Closing Redis connection...")

		err := r.CanonicalClient.Close()
		if err != nil {
			r.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error closing Redis connection: %v", err))
			return err
		}

		r.Connected = false
		r.Logger.Log(context.Background(), libLog.LevelInfo, "Redis connection closed successfully.")

		return nil
	}

	if r.Client != nil {
		if err := r.Client.Close(); err != nil {
			return err
		}

		r.Connected = false
	}

	return nil
}

// IsConnected returns whether the Redis connection is active.
func (r *RedisConnection) IsConnected() bool {
	if r.Client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	return r.Client.Ping(ctx).Err() == nil
}

// buildRedisAddr constructs the Redis address from host and port.
// If the host already contains a port (e.g., "redis.example.com:6379"),
// the explicit port parameter is ignored to avoid duplicate ports like
// "redis.example.com:6379:6379".
func buildRedisAddr(host, port string) string {
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}

	if port != "" {
		return net.JoinHostPort(host, port)
	}

	return host
}
