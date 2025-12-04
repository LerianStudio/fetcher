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

	limiter "github.com/sethvargo/go-limiter"
)

type TestConnection struct {
	connRepo connRepo.Repository
	store    limiter.Store
	cryptor  crypto.Cryptor
}

func NewTestConnection(connRepo connRepo.Repository, cryptor crypto.Cryptor, store limiter.Store) *TestConnection {
	return &TestConnection{
		connRepo: connRepo,
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

		resetTime := time.Unix(0, int64(reset))
		retryAfter := time.Until(resetTime)
		if retryAfter < 0 {
			retryAfter = 0
		}

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
		return nil, pkg.EntityNotFoundError{
			EntityType: "connection",
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		}
	}

	err = conn.DecryptPassword(ctx, s.cryptor)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to decrypt connection password", err)
		logger.Errorf("failed to decrypt connection password id=%s org=%s: %v", connectionID, organizationID, err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	testCtx, cancel := context.WithTimeout(ctx, constant.ConnectionTestTimeout)
	defer cancel()

	start := time.Now()
	ds, err := datasource.NewDataSourceFromConnectionWithTimeout(testCtx, conn, logger, constant.ConnectionTestTimeout)
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
