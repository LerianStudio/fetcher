package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/LerianStudio/lib-commons/v2/commons/log"
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
	Logger    log.Logger
	Connected bool
}

// NewRedisConnection creates a new Redis connection.
// Configuration values default to sensible values if not specified.
// Use RedisConfig.WithDefaults() explicitly if you want to see the resolved values.
func NewRedisConnection(cfg RedisConfig, logger log.Logger) (*RedisConnection, error) {
	cfg = cfg.WithDefaults()
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Errorf("Failed to connect to Redis at %s: %v", addr, err)
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Infof("Successfully connected to Redis at %s", addr)

	return &RedisConnection{
		Client:    client,
		Logger:    logger,
		Connected: true,
	}, nil
}

// Close closes the Redis connection.
func (r *RedisConnection) Close() error {
	if r.Client != nil {
		r.Logger.Info("Closing Redis connection...")
		err := r.Client.Close()
		if err != nil {
			r.Logger.Errorf("Error closing Redis connection: %v", err)
			return err
		}
		r.Connected = false
		r.Logger.Info("Redis connection closed successfully.")
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
