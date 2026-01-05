package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/lib-commons/v2/commons"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
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
	logger := commons.NewLoggerFromContext(ctx)

	// Client-side cancellation / deadlines (HTTP layer)
	// If the client closed the connection, you often can't write a response anyway.
	if errors.Is(err, context.Canceled) {
		logger.Errorf("MongoDB request canceled by client: %v", err)
		return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	// Timeouts (driver/helpers)
	if errors.Is(err, context.DeadlineExceeded) || mongo.IsTimeout(err) {
		logger.Errorf("MongoDB timeout error: %v", err)
		return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	// Server selection / network
	if errors.Is(err, topology.ErrServerSelectionTimeout) {
		logger.Errorf("MongoDB server selection timeout: %v", err)
		return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	var sse topology.ServerSelectionError
	if errors.As(err, &sse) {
		logger.Errorf("MongoDB server selection error: %v", err)
		return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	if mongo.IsNetworkError(err) {
		logger.Errorf("MongoDB network error: %v", err)
		return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
	}

	// Common query/result semantics
	if errors.Is(err, mongo.ErrNoDocuments) {
		logger.Debugf("MongoDB document not found: %v", err)
		return pkg.ValidateInternalError(constant.ErrNotFound, "")
	}

	// Duplicate key -> 409
	if mongo.IsDuplicateKeyError(err) {
		logger.Errorf("MongoDB duplicate key error: %v", err)
		return pkg.ValidateInternalError(constant.ErrConflict, "")
	}

	// Command errors from MongoDB
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		switch cmdErr.Code {
		case 13, 18: // Unauthorized / AuthenticationFailed
			logger.Errorf("MongoDB unauthorized error: %v", err)
			return pkg.ValidateInternalError(constant.ErrInternalServer, "")
		case 50: // ExceededTimeLimit
			logger.Errorf("MongoDB exceeded time limit error: %v", err)
			return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
		case 6, 7, 89, 91: // HostUnreachable/HostNotFound/Shutdown
			logger.Errorf("MongoDB service unavailable error: %v", err)
			return pkg.ValidateInternalError(constant.ErrServiceUnavailable, "")
		case 9: // FailedToParse
			logger.Errorf("MongoDB bad request error: %v", err)
			return pkg.ValidateInternalError(constant.ErrInternalServer, "")
		case 26: // NamespaceNotFound
			logger.Debugf("MongoDB namespace not found error: %v", err)
			return pkg.ValidateInternalError(constant.ErrInternalServer, "")
		}
	}

	// Write exceptions (bulk/insert/update) - map duplicate here too, plus other codes if desired
	var we mongo.WriteException
	if errors.As(err, &we) {
		for _, e := range we.WriteErrors {
			if e.Code == 11000 || e.Code == 11001 {
				logger.Errorf("MongoDB duplicate key error in write exception: %v", err)
				return pkg.ValidateInternalError(constant.ErrConflict, "")
			}
		}

		logger.Errorf("MongoDB write exception error: %v", err)

		return pkg.ValidateInternalError(constant.ErrInternalServer, "")
	}

	// Decode / BSON issues
	var decodeErr bsoncodec.ValueDecoderError
	if errors.As(err, &decodeErr) {
		logger.Errorf("MongoDB decode error: %v", err)
		return pkg.ValidateInternalError(constant.ErrInternalServer, "")
	}

	logger.Errorf("MongoDB unknown error: %v", err)

	return pkg.ValidateInternalError(constant.ErrInternalServer, "")
}

//go:generate mockgen --destination=mongo_client_provider.mock.go --package=mongodb . MongoClientProvider

// MongoClientProvider is an interface for obtaining a MongoDB client.
// This allows for easy testing and dependency injection.
type MongoClientProvider interface {
	GetDB(ctx context.Context) (*mongo.Client, error)
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

	client, err := provider.GetDB(ctx)
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
