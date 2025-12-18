package query

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

const ConnectionTestTimeout = 10 * time.Second

// RateLimiterStore defines the interface for rate limiting operations.
// This interface maintains backward compatibility with tests.
type RateLimiterStore interface {
	Take(ctx context.Context, key string) (tokens, remaining, reset uint64, ok bool, err error)
}

type TestConnection struct {
	connRepo connRepo.Repository
	store    RateLimiterStore
	cryptor  crypto.Cryptor
}

// NewTestConnection creates a new TestConnection service.
// The store parameter accepts either *ratelimit.RateLimiter or any implementation
// of the RateLimiterStore interface for backward compatibility.
func NewTestConnection(connectionRepo connRepo.Repository, cryptor crypto.Cryptor, store RateLimiterStore) *TestConnection {
	return &TestConnection{
		connRepo: connectionRepo,
		store:    store,
		cryptor:  cryptor,
	}
}

func (s *TestConnection) Execute(ctx context.Context, organizationID, connectionID uuid.UUID) (*model.ConnectionTestResponse, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.test_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	key := connectionID.String()

	_, _, reset, ok, err := s.store.Take(ctx, key)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "connection test rate limiter error", err)
		logger.Errorf("connection test rate limiter error id=%s org=%s: %v", connectionID, organizationID, err)

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

		logger.Warnf("connection test rate limited id=%s org=%s", connectionID, organizationID)

		return nil, pkg.ResponseError{
			Code:    http.StatusTooManyRequests,
			Title:   "Rate Limit Exceeded",
			Message: fmt.Sprintf("Connection test limit reached. Try again in %d seconds.", waitSeconds),
		}
	}

	conn, err := s.connRepo.FindByID(ctx, connectionID, organizationID)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to find connection", err)
		return nil, err
	}

	if conn == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	testCtx, cancel := context.WithTimeout(ctx, ConnectionTestTimeout)
	defer cancel()

	start := time.Now()

	ds, err := datasource.NewDataSourceFromConnection(testCtx, conn, s.cryptor, logger)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to establish datasource connection", err)
		logger.Errorf("connection test failed id=%s org=%s", connectionID, organizationID)

		return nil, pkg.ResponseError{
			Code:    http.StatusInternalServerError,
			Title:   "Database Connection Error",
			Message: "The adapter failed to connect to the target data source. Check credentials and network access.",
		}
	}

	latencyMs := time.Since(start).Milliseconds()
	span.SetAttributes(attribute.Int64("app.connection_test.latency_ms", latencyMs))

	if err := ds.Close(testCtx); err != nil {
		logger.Warnf("connection test cleanup failed id=%s org=%s: %v", connectionID, organizationID, err)
	}

	return &model.ConnectionTestResponse{
		Status:    "success",
		Message:   "Connection successful",
		LatencyMs: latencyMs,
	}, nil
}
