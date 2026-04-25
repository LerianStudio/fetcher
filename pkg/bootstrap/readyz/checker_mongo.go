package readyz

import (
	"context"
	"errors"
	"time"
)

// MongoPinger is the narrow surface the MongoClientChecker needs from its
// underlying client. The real lib-commons *mongo.Client satisfies this
// directly (via its Ping(ctx context.Context) error method); the interface
// lets unit tests stand in a fake without pulling in a live driver.
type MongoPinger interface {
	Ping(ctx context.Context) error
}

// MongoClientChecker is the DependencyChecker implementation for the
// fetcher's PLATFORM MongoDB connection (single-tenant mode) or the
// single lib-commons *mongo.Client that underpins the tmmongo.Manager
// (multi-tenant mode — though in that mode the global /readyz uses an
// NAChecker; the real checker lives in the per-tenant prober).
//
// The checker issues a Ping against the primary read preference under the
// caller-supplied context, measures latency in milliseconds, and reports the
// configured TLS posture derived from the connection URI via the Gate 3
// detector. Credentials are never leaked: the error field runs through
// sanitize() before it is placed in the response.
type MongoClientChecker struct {
	name   string
	client MongoPinger
	uri    string
}

// NewMongoClientChecker constructs a checker reporting under the dep name
// "mongodb". The URI is used only for TLS posture detection; the checker
// does not parse it for connection parameters.
func NewMongoClientChecker(client MongoPinger, uri string) *MongoClientChecker {
	return &MongoClientChecker{
		name:   "mongodb",
		client: client,
		uri:    uri,
	}
}

// Name returns the stable dependency identifier ("mongodb").
func (c *MongoClientChecker) Name() string { return c.name }

// Check probes the Mongo connection using the caller-supplied context. The
// context deadline is expected to come from the /readyz handler
// (PerDepTimeout("mongodb") == 2s); the checker does not wrap it again.
//
// Error classification:
//   - nil err                     → Status=up
//   - ctx.DeadlineExceeded        → Status=down, Error="timeout"
//   - ctx.Canceled                → Status=down, Error="canceled"
//   - other                       → Status=down, Error=sanitize(err.Error())
//
// The TLS field is populated from detectMongoTLS(uri) — the checker refuses
// to inspect the live socket for TLS. The informational field reflects the
// configured posture, matching the contract.
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

// classifyErr picks the operator-visible error string for a failed probe.
// Context-cancellation errors short-circuit to stable vocabulary ("timeout",
// "canceled") so Grafana alerts can split retries from real failures; every
// other error falls through to sanitize() which redacts credentials.
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
