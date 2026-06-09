package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	libLog "github.com/LerianStudio/lib-observability/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

const (
	// DefaultPingTimeout is the default timeout for health check pings
	DefaultPingTimeout = 5 * time.Second
)

// ValidateFieldsInSchemaMongo validate if all fields exist on mongo DB collection
func ValidateFieldsInSchemaMongo(expectedFields []string, schema CollectionSchema, countIfTableExist *int32) (missing []string) {
	columnSet := make(map[string]struct{}, len(schema.Fields))
	for _, col := range schema.Fields {
		columnSet[strings.ToLower(col.Name)] = struct{}{}
	}

	for _, field := range expectedFields {
		if _, exists := columnSet[strings.ToLower(field)]; !exists {
			missing = append(missing, field)
		} else {
			*countIfTableExist++
		}
	}

	return
}

//nolint:gocyclo // High complexity is inherent to comprehensive MongoDB error handling across multiple error types
func MapMongoErrorToResponse(err error, ctx context.Context) error {
	logger := observability.NewLoggerFromContext(ctx)

	// Client-side cancellation / deadlines (HTTP layer)
	// If the client closed the connection, you often can't write a response anyway.
	if errors.Is(err, context.Canceled) {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB request canceled by client: %v", err))
		return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	// Timeouts (driver/helpers)
	if errors.Is(err, context.DeadlineExceeded) || mongo.IsTimeout(err) {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB timeout error: %v", err))
		return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	// Server selection / network.
	// Note: mongo-driver v2 removed the topology.ErrServerSelectionTimeout sentinel.
	// Server-selection timeouts now surface as context.DeadlineExceeded (handled above
	// under the timeout branch) or as a topology.ServerSelectionError (handled below).
	var sse topology.ServerSelectionError
	if errors.As(err, &sse) {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB server selection error: %v", err))
		return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	if mongo.IsNetworkError(err) {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB network error: %v", err))
		return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	// Common query/result semantics
	if errors.Is(err, mongo.ErrNoDocuments) {
		logger.Log(ctx, libLog.LevelDebug, fmt.Sprintf("MongoDB document not found: %v", err))
		return pkg.ValidateInternalError(constant.ErrNotFound, "")
	}

	// Duplicate key -> 409
	if mongo.IsDuplicateKeyError(err) {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB duplicate key error: %v", err))
		return pkg.ValidateInternalError(constant.ErrConflict, "")
	}

	// Command errors from MongoDB
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		switch cmdErr.Code {
		case 13, 18: // Unauthorized / AuthenticationFailed
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB unauthorized error: %v", err))
			return pkg.ValidateInternalError(constant.ErrInternalServer, "")
		case 50: // ExceededTimeLimit
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB exceeded time limit error: %v", err))
			return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
		case 6, 7, 89, 91: // HostUnreachable/HostNotFound/Shutdown
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB service unavailable error: %v", err))
			return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
		case 9: // FailedToParse
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB bad request error: %v", err))
			return pkg.ValidateInternalError(constant.ErrInternalServer, "")
		case 26: // NamespaceNotFound
			logger.Log(ctx, libLog.LevelDebug, fmt.Sprintf("MongoDB namespace not found error: %v", err))
			return pkg.ValidateInternalError(constant.ErrInternalServer, "")
		}
	}

	// Write exceptions (bulk/insert/update) - map duplicate here too, plus other codes if desired
	var we mongo.WriteException
	if errors.As(err, &we) {
		for _, e := range we.WriteErrors {
			if e.Code == 11000 || e.Code == 11001 {
				logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB duplicate key error in write exception: %v", err))
				return pkg.ValidateInternalError(constant.ErrConflict, "")
			}
		}

		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB write exception error: %v", err))

		return pkg.ValidateInternalError(constant.ErrInternalServer, "")
	}

	// Decode / BSON issues
	var decodeErr bson.ValueDecoderError
	if errors.As(err, &decodeErr) {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB decode error: %v", err))
		return pkg.ValidateInternalError(constant.ErrInternalServer, "")
	}

	// Multi-tenant sentinel errors
	if errors.Is(err, tmcore.ErrTenantContextRequired) {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB tenant context required: %v", err))
		return pkg.ValidateBusinessError(constant.ErrTenantContextRequired, "tenant")
	}

	if errors.Is(err, tmcore.ErrTenantNotFound) {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB tenant not found: %v", err))
		return pkg.ValidateBusinessError(constant.ErrTenantNotFound, "tenant")
	}

	if errors.Is(err, tmcore.ErrCircuitBreakerOpen) {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB tenant circuit breaker open: %v", err))
		return pkg.ValidateBusinessError(constant.ErrTenantCircuitBreaker, "tenant")
	}

	logger.Log(ctx, libLog.LevelError, fmt.Sprintf("MongoDB unknown error: %v", err))

	return pkg.ValidateInternalError(constant.ErrInternalServer, "")
}

//go:generate mockgen --destination=mongo_client_provider.mock.go --package=mongodb . MongoClientProvider

// MongoClientProvider is an interface for obtaining a MongoDB client.
// This allows for easy testing and dependency injection.
type MongoClientProvider interface {
	Client(ctx context.Context) (*mongo.Client, error)
}

// PingMongo performs a health check ping on the MongoDB connection.
// It uses the provided timeout, defaulting to DefaultPingTimeout if timeout is 0.
// This is useful for Kubernetes readiness probes.
func PingMongo(ctx context.Context, provider MongoClientProvider, timeout time.Duration) error {
	if provider == nil {
		return errors.New("mongo client provider is nil")
	}

	if timeout <= 0 {
		timeout = DefaultPingTimeout
	}

	client, err := provider.Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		return fmt.Errorf("mongodb ping failed: %w", err)
	}

	return nil
}
