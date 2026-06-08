package query

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	ds "github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/resolver"

	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	valkey "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/valkey"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

const ConnectionTestTimeout = 10 * time.Second

//go:generate mockgen --destination=rate_limiter_store.mock.go --package=query . RateLimiterStore

// RateLimiterStore defines the interface for rate limiting operations.
// This interface maintains backward compatibility with tests.
type RateLimiterStore interface {
	Take(ctx context.Context, key string) (tokens, remaining, reset uint64, ok bool, err error)
}

type TestConnection struct {
	connRepo  connRepo.Repository
	store     RateLimiterStore
	cryptor   crypto.Cryptor
	dsFactory ds.DataSourceFactory
	resolver  resolver.ConnectionResolver          // nil-safe
	registry  *resolver.InternalDatasourceRegistry // nil-safe
}

// NewTestConnection creates a new TestConnection service.
// The store parameter accepts either *ratelimit.RateLimiter or any implementation
// of the RateLimiterStore interface for backward compatibility.
func NewTestConnection(
	connectionRepo connRepo.Repository,
	cryptor crypto.Cryptor,
	store RateLimiterStore,
	factory ds.DataSourceFactory,
	connResolver resolver.ConnectionResolver,
	dsRegistry *resolver.InternalDatasourceRegistry,
) *TestConnection {
	return &TestConnection{
		connRepo:  connectionRepo,
		store:     store,
		cryptor:   cryptor,
		dsFactory: factory,
		resolver:  connResolver,
		registry:  dsRegistry,
	}
}

func (s *TestConnection) Execute(ctx context.Context, connectionID uuid.UUID) (*model.ConnectionTestResponse, error) {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.test_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	key, err := valkey.GetKeyContext(ctx, connectionID.String())
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "connection test rate limiter key derivation error", err)
		logger.Log(ctx, libLog.LevelError, "connection test rate limiter key derivation error",
			libLog.String("connection_id", connectionID.String()),
			libLog.Err(err),
		)

		return nil, pkg.ValidateInternalError(err, "connection")
	}

	_, _, reset, ok, err := s.store.Take(ctx, key)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "connection test rate limiter error", err)
		logger.Log(ctx, libLog.LevelError, "connection test rate limiter error",
			libLog.String("connection_id", connectionID.String()),
			libLog.Err(err),
		)

		return nil, pkg.ValidateInternalError(err, "connection")
	} else if !ok {
		span.SetAttributes(attribute.Bool("app.connection_test.rate_limited", true))

		// reset is a Unix nanosecond timestamp from rate limiter, always positive and recent
		resetTime := time.Unix(0, int64(reset)) // #nosec G115 -- reset is always a valid positive timestamp from rate limiter
		retryAfter := max(time.Until(resetTime), 0)

		waitSeconds := int(retryAfter / time.Second)
		if retryAfter%time.Second != 0 {
			waitSeconds++
		}

		if waitSeconds <= 0 {
			waitSeconds = 1
		}

		logger.Log(ctx, libLog.LevelWarn, "connection test rate limited",
			libLog.String("connection_id", connectionID.String()),
			libLog.Int("wait_seconds", waitSeconds),
		)

		return nil, pkg.ResponseError{
			Code:    http.StatusTooManyRequests,
			Title:   "Rate Limit Exceeded",
			Message: fmt.Sprintf("Connection test limit reached. Try again in %d seconds.", waitSeconds),
		}
	}

	// Check if this is an internal datasource (deterministic UUID per tenant)
	var conn *model.Connection

	if s.registry != nil && s.resolver != nil {
		tenantID := tmcore.GetTenantIDContext(ctx)
		if configName, _, found := s.registry.FindConfigByID(connectionID, tenantID); found {
			resolved, resolveErr := s.resolver.ResolveInternalByConfigName(ctx, configName)
			if resolveErr != nil {
				libOpentelemetry.HandleSpanError(span, "failed to resolve internal datasource", resolveErr)
				return nil, fmt.Errorf("failed to resolve internal datasource '%s': %w", configName, resolveErr)
			}

			conn = resolved
		}
	}

	// Fallback to MongoDB lookup for external connections
	if conn == nil {
		var findErr error

		conn, findErr = s.connRepo.FindByID(ctx, connectionID)
		if findErr != nil {
			libOpentelemetry.HandleSpanError(span, "failed to find connection", findErr)
			return nil, fmt.Errorf("failed to find connection by id: %w", findErr)
		}

		if conn == nil {
			return nil, pkg.ValidateBusinessError(
				constant.ErrEntityNotFound,
				"connection",
			)
		}
	}

	testCtx, cancel := context.WithTimeout(ctx, ConnectionTestTimeout)
	defer cancel()

	start := time.Now()

	var connDS datasource.DataSource

	connDS, err = s.dsFactory(testCtx, conn, s.cryptor)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to establish datasource connection", err)
		logger.Log(testCtx, libLog.LevelError, "connection test failed",
			libLog.String("connection_id", connectionID.String()),
			libLog.Err(err),
		)

		// Preserve typed validation errors (e.g. FET-0414 from the SSRF host
		// safety guard) so the renderer maps them to HTTP 400 — masking with
		// a generic 500 would break the documented FET-0414 contract and drop
		// the audit signal that a tenant tried to reach a denylisted host.
		var ve pkg.ValidationError
		if errors.As(err, &ve) {
			return nil, err
		}

		return nil, pkg.ResponseError{
			Code:    http.StatusInternalServerError,
			Title:   "Database Connection Error",
			Message: "The adapter failed to connect to the target data source. Check credentials and network access.",
		}
	}

	latencyMs := time.Since(start).Milliseconds()
	span.SetAttributes(attribute.Int64("app.connection_test.latency_ms", latencyMs))

	if err := connDS.Close(testCtx); err != nil {
		logger.Log(testCtx, libLog.LevelWarn, "connection test cleanup failed",
			libLog.String("connection_id", connectionID.String()),
			libLog.Err(err),
		)
	}

	return &model.ConnectionTestResponse{
		Status:    "success",
		Message:   "Connection successful",
		LatencyMs: latencyMs,
	}, nil
}
