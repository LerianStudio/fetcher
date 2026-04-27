package readyz

import (
	"context"
	"errors"
	"time"
)

// MongoPinger is the narrow client surface the checker requires; lib-commons
// *mongo.Client satisfies it directly. Defined as an interface so tests can
// stand in a fake without a live driver.
type MongoPinger interface {
	Ping(ctx context.Context) error
}

// MongoClientChecker probes a MongoDB client via Ping under the caller's
// context, reports latency, and surfaces the TLS posture derived from the
// connection URI. Errors are sanitized before being returned so credentials
// from URIs are never leaked.
type MongoClientChecker struct {
	name   string
	client MongoPinger
	uri    string
}

// NewMongoClientChecker reports under the dep name "mongodb". The URI is used
// only for TLS posture detection.
func NewMongoClientChecker(client MongoPinger, uri string) *MongoClientChecker {
	return &MongoClientChecker{
		name:   "mongodb",
		client: client,
		uri:    uri,
	}
}

func (c *MongoClientChecker) Name() string { return c.name }

func (c *MongoClientChecker) Check(ctx context.Context) DependencyCheck {
	if c.client == nil {
		return DependencyCheck{
			Status: StatusDown,
			TLS:    TLSPtr(tlsOrFalse(detectMongoTLS(c.uri))),
			Error:  "mongo client not initialized",
		}
	}

	start := time.Now()
	err := c.client.Ping(ctx)
	elapsed := time.Since(start)

	tlsOn := tlsOrFalse(detectMongoTLS(c.uri))

	if err == nil {
		return DependencyCheck{
			Status:    StatusUp,
			LatencyMs: elapsed.Milliseconds(),
			TLS:       TLSPtr(tlsOn),
		}
	}

	return DependencyCheck{
		Status:    StatusDown,
		LatencyMs: elapsed.Milliseconds(),
		TLS:       TLSPtr(tlsOn),
		Error:     classifyErr(ctx, err),
	}
}

// classifyErr maps probe errors to a stable operator vocabulary. Context
// cancellation collapses to "timeout"/"canceled" so alerts can split retries
// from real failures; everything else is sanitized to redact credentials.
func classifyErr(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		switch {
		case errors.Is(ctxErr, context.DeadlineExceeded):
			return "timeout"
		case errors.Is(ctxErr, context.Canceled):
			return "canceled"
		}
	}

	return sanitize(err.Error())
}
