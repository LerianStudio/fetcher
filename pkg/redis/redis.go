package redis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"time"

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
	Client    *redis.Client
	Logger    libLog.Logger
	Connected bool
}

// NewRedisConnection creates a new Redis connection.
// Configuration values default to sensible values if not specified.
// Use RedisConfig.WithDefaults() explicitly if you want to see the resolved values.
func NewRedisConnection(cfg RedisConfig, logger libLog.Logger) (*RedisConnection, error) {
	cfg = cfg.WithDefaults()
	addr := buildRedisAddr(cfg.Host, cfg.Port)

	opts := &redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	}

	if cfg.UseTLS {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}

		if cfg.CACert != "" {
			caBytes, err := base64.StdEncoding.DecodeString(cfg.CACert)
			if err != nil {
				return nil, fmt.Errorf("failed to decode Redis CA certificate: %w", err)
			}

			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(caBytes) {
				return nil, fmt.Errorf("failed to parse Redis CA certificate")
			}

			tlsCfg.RootCAs = pool
		}

		opts.TLSConfig = tlsCfg
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to connect to Redis at %s: %v", addr, err))
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Successfully connected to Redis at %s", addr))

	return &RedisConnection{
		Client:    client,
		Logger:    logger,
		Connected: true,
	}, nil
}

// Close closes the Redis connection.
func (r *RedisConnection) Close() error {
	if r.Client != nil {
		r.Logger.Log(context.Background(), libLog.LevelInfo, "Closing Redis connection...")

		err := r.Client.Close()
		if err != nil {
			r.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error closing Redis connection: %v", err))
			return err
		}

		r.Connected = false
		r.Logger.Log(context.Background(), libLog.LevelInfo, "Redis connection closed successfully.")
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
