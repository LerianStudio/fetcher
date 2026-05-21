package readyz

import (
	"context"
	"errors"
	"time"

	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// TenantChecker is the per-tenant analogue of DependencyChecker, used by
// /readyz/tenant/:id. The context carries the tenant ID via
// tmcore.ContextWithTenantID; the explicit tenantID argument is provided
// for convenience. Name() must match the global dep name ("mongodb",
// "rabbitmq") so dashboards stay portable between /readyz and
// /readyz/tenant/:id.
type TenantChecker interface {
	Name() string
	CheckForTenant(ctx context.Context, tenantID string) DependencyCheck
}

// TenantMongoResolver is the narrow tmmongo.Manager surface required by the
// per-tenant Mongo checker.
type TenantMongoResolver interface {
	GetDatabaseForTenant(ctx context.Context, tenantID string) (*mongo.Database, error)
}

// TenantMongoChecker pings the tenant's *mongo.Database resolved via the
// tmmongo.Manager. The underlying tmclient breaker is opaque, so only
// tmcore.ErrCircuitBreakerOpen surfaces with BreakerState="open"; other
// errors surface without it.
type TenantMongoChecker struct {
	mgr TenantMongoResolver
}

func NewTenantMongoChecker(mgr TenantMongoResolver) *TenantMongoChecker {
	return &TenantMongoChecker{mgr: mgr}
}

func (c *TenantMongoChecker) Name() string { return "mongodb" }

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

// TenantRabbitMQResolver is the narrow tmrabbitmq.Manager surface required
// by the per-tenant Rabbit checker.
type TenantRabbitMQResolver interface {
	GetChannel(ctx context.Context, tenantID string) (*amqp.Channel, error)
}

// TenantRabbitMQChecker opens and immediately closes a channel on the
// tenant's vhost — opening implicitly validates connectivity and
// permissions, and closing avoids holding pool capacity. Channel-close
// errors after a successful open do not flip the check to down: transient
// close races must not poison readiness once the probe already succeeded.
type TenantRabbitMQChecker struct {
	mgr TenantRabbitMQResolver
}

func NewTenantRabbitMQChecker(mgr TenantRabbitMQResolver) *TenantRabbitMQChecker {
	return &TenantRabbitMQChecker{mgr: mgr}
}

func (c *TenantRabbitMQChecker) Name() string { return "rabbitmq" }

func (c *TenantRabbitMQChecker) CheckForTenant(ctx context.Context, tenantID string) DependencyCheck {
	if c.mgr == nil {
		return DependencyCheck{Status: StatusDown, Error: "tenant rabbitmq manager not initialized"}
	}

	start := time.Now()

	ch, err := c.mgr.GetChannel(ctx, tenantID)
	if err != nil {
		return classifyTenantErr(ctx, err, start)
	}

	// GetChannel may return (nil, nil) on edge cases (e.g. pool resolved
	// the tenant but the underlying connection was reaped between resolve
	// and channel allocation). Returning StatusUp in that scenario would
	// mask a genuinely missing channel — surface it as down.
	if ch == nil {
		return DependencyCheck{
			Status:    StatusDown,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     "tenant rabbitmq channel not available",
		}
	}

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

// classifyTenantErr extends classifyErr with detection of
// tmcore.ErrCircuitBreakerOpen so the breaker state surfaces when the
// tmclient wrapping the manager trips.
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
