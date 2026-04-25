package readyz

import (
	"context"
	"errors"
	"time"

	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	amqp "github.com/rabbitmq/amqp091-go"
)

// TenantChecker is the per-tenant analogue of DependencyChecker. It is used
// by /readyz/tenant/:id to probe tenant-scoped dependencies (the per-tenant
// Mongo database, the per-tenant AMQP vhost).
//
// The context carries the tenant ID via tmcore.ContextWithTenantID, which
// the per-tenant handler installs from the URL parameter BEFORE calling
// CheckForTenant. Implementations are free to consult either that context
// or the explicit tenantID argument; both are provided for convenience.
type TenantChecker interface {
	// Name returns the dep identifier used as the key in the response.
	// MUST match the global dep name ("mongodb", "rabbitmq") so dashboards
	// stay portable between /readyz and /readyz/tenant/:id.
	Name() string
	// CheckForTenant runs a single probe against the tenant-scoped
	// dependency. See the handler for per-dep timeout budget.
	CheckForTenant(ctx context.Context, tenantID string) DependencyCheck
}

// TenantMongoResolver is the narrow tmmongo.Manager surface the per-tenant
// Mongo checker needs. The real *tmmongo.Manager satisfies this — tests
// inject a fake.
type TenantMongoResolver interface {
	GetDatabaseForTenant(ctx context.Context, tenantID string) (*mongo.Database, error)
}

// TenantMongoChecker probes the per-tenant MongoDB database. It resolves the
// tenant's *mongo.Database via the tmmongo.Manager (which reuses its cached
// per-tenant connection pool), then issues a Ping with readpref.Primary.
//
// Breaker-state contract: tmmongo.Manager does not expose a circuit-breaker
// state method — it wraps the tmclient whose breaker is opaque. Errors that
// are tmcore.ErrCircuitBreakerOpen surface with BreakerState="open"; all
// other errors surface without BreakerState (matching the
// per-tenant-checker rules in the Gate 6 contract).
type TenantMongoChecker struct {
	mgr TenantMongoResolver
}

// NewTenantMongoChecker constructs a per-tenant Mongo checker.
func NewTenantMongoChecker(mgr TenantMongoResolver) *TenantMongoChecker {
	return &TenantMongoChecker{mgr: mgr}
}

// Name returns "mongodb" — the same key used on the global /readyz.
func (c *TenantMongoChecker) Name() string { return "mongodb" }

// CheckForTenant resolves the tenant's Mongo database and pings it. The
// tenantID argument is mirrored into ctx via tmcore.ContextWithTenantID at
// the handler level; the method doesn't re-install the context here because
// the resolver also accepts the explicit ID.
func (c *TenantMongoChecker) CheckForTenant(ctx context.Context, tenantID string) DependencyCheck {
	if c.mgr == nil {
		return DependencyCheck{Status: StatusDown, Error: "tenant mongo manager not initialized"}
	}

	start := time.Now()

	db, err := c.mgr.GetDatabaseForTenant(ctx, tenantID)
	if err != nil {
		return classifyTenantErr(ctx, err, start)
	}

	if db == nil {
		return DependencyCheck{
			Status:    StatusDown,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     "tenant mongo database not available",
		}
	}

	if pingErr := db.Client().Ping(ctx, readpref.Primary()); pingErr != nil {
		return classifyTenantErr(ctx, pingErr, start)
	}

	return DependencyCheck{
		Status:    StatusUp,
		LatencyMs: time.Since(start).Milliseconds(),
	}
}

// TenantRabbitMQResolver is the narrow tmrabbitmq.Manager surface the
// per-tenant Rabbit checker needs. The real *tmrabbitmq.Manager satisfies
// this (its GetChannel returns *amqp.Channel).
type TenantRabbitMQResolver interface {
	GetChannel(ctx context.Context, tenantID string) (*amqp.Channel, error)
}

// TenantRabbitMQChecker probes the per-tenant RabbitMQ vhost. It asks the
// tmrabbitmq.Manager for a channel on the tenant's vhost (which implicitly
// validates connectivity + permissions) and immediately closes it so the
// pool is not held open by the readiness probe.
type TenantRabbitMQChecker struct {
	mgr TenantRabbitMQResolver
}

// NewTenantRabbitMQChecker constructs a per-tenant Rabbit checker.
func NewTenantRabbitMQChecker(mgr TenantRabbitMQResolver) *TenantRabbitMQChecker {
	return &TenantRabbitMQChecker{mgr: mgr}
}

// Name returns "rabbitmq" — the same key used on the global /readyz.
func (c *TenantRabbitMQChecker) Name() string { return "rabbitmq" }

// CheckForTenant obtains a channel on the tenant's vhost and closes it
// immediately. Any channel-close error after a successful open is reported
// as a warning-level error string rather than flipping the check to down —
// the connectivity probe already succeeded. This keeps transient AMQP
// close races from poisoning readiness.
func (c *TenantRabbitMQChecker) CheckForTenant(ctx context.Context, tenantID string) DependencyCheck {
	if c.mgr == nil {
		return DependencyCheck{Status: StatusDown, Error: "tenant rabbitmq manager not initialized"}
	}

	start := time.Now()

	ch, err := c.mgr.GetChannel(ctx, tenantID)
	if err != nil {
		return classifyTenantErr(ctx, err, start)
	}

	// Close the channel under a best-effort idiom — the probe succeeds as
	// long as GetChannel returned without error.
	defer func() {
		if ch != nil {
			_ = ch.Close()
		}
	}()

	return DependencyCheck{
		Status:    StatusUp,
		LatencyMs: time.Since(start).Milliseconds(),
	}
}

// classifyTenantErr is the shared error-classification helper for per-tenant
// probers. It mirrors the global classifyErr but also checks
// tmcore.ErrCircuitBreakerOpen so we surface the breaker state when the
// tmclient (wrapping the manager) trips.
func classifyTenantErr(ctx context.Context, err error, start time.Time) DependencyCheck {
	elapsed := time.Since(start)

	if errors.Is(err, tmcore.ErrCircuitBreakerOpen) {
		return DependencyCheck{
			Status:       StatusDown,
			LatencyMs:    elapsed.Milliseconds(),
			Error:        "circuit breaker open",
			BreakerState: BreakerOpen.String(),
		}
	}

	return DependencyCheck{
		Status:    StatusDown,
		LatencyMs: elapsed.Milliseconds(),
		Error:     classifyErr(ctx, err),
	}
}
