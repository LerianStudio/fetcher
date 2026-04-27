package readyz

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	amqp "github.com/rabbitmq/amqp091-go"
)

// fakeMongoResolver returns a static (db, err) pair without any driver state.
// We set db=nil for error-path tests; the checker reports "tenant mongo
// database not available" on nil without trying to ping.
type fakeMongoResolver struct {
	db    *mongo.Database
	err   error
	delay time.Duration
	calls int
}

func (f *fakeMongoResolver) GetDatabaseForTenant(ctx context.Context, _ string) (*mongo.Database, error) {
	f.calls++

	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return f.db, f.err
}

func TestTenantMongoChecker_ResolverError(t *testing.T) {
	res := NewTenantMongoChecker(&fakeMongoResolver{
		err: errors.New("dial tcp: connection refused"),
	}).CheckForTenant(context.Background(), "t1")

	assert.Equal(t, StatusDown, res.Status)
	assert.Contains(t, res.Error, "connection refused")
	assert.Empty(t, res.BreakerState, "generic errors don't set breaker state")
}

func TestTenantMongoChecker_BreakerOpen(t *testing.T) {
	res := NewTenantMongoChecker(&fakeMongoResolver{
		err: fmt.Errorf("resolve: %w", tmcore.ErrCircuitBreakerOpen),
	}).CheckForTenant(context.Background(), "t1")

	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "circuit breaker open", res.Error)
	assert.Equal(t, "open", res.BreakerState)
}

func TestTenantMongoChecker_NilManager(t *testing.T) {
	res := NewTenantMongoChecker(nil).CheckForTenant(context.Background(), "t1")

	assert.Equal(t, StatusDown, res.Status)
	assert.Contains(t, res.Error, "not initialized")
}

func TestTenantMongoChecker_NilDatabase(t *testing.T) {
	// Resolver returns nil db and nil err — a defensive case that should
	// surface as down rather than panic on the Ping.
	res := NewTenantMongoChecker(&fakeMongoResolver{db: nil, err: nil}).
		CheckForTenant(context.Background(), "t1")

	assert.Equal(t, StatusDown, res.Status)
	assert.Contains(t, res.Error, "not available")
}

func TestTenantMongoChecker_Timeout(t *testing.T) {
	res := NewTenantMongoChecker(&fakeMongoResolver{
		delay: 50 * time.Millisecond,
		err:   context.DeadlineExceeded,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	out := res.CheckForTenant(ctx, "t1")
	assert.Equal(t, StatusDown, out.Status)
	assert.Equal(t, "timeout", out.Error)
}

func TestTenantMongoChecker_Name(t *testing.T) {
	assert.Equal(t, "mongodb", NewTenantMongoChecker(nil).Name())
}

// fakeRabbitResolver simulates tmrabbitmq.Manager.GetChannel. Returning a
// real *amqp.Channel here is not possible without a live broker, so tests
// exercise the error paths; the "happy path" is covered by the integration
// suite in Gate 8.
type fakeRabbitResolver struct {
	ch    *amqp.Channel
	err   error
	delay time.Duration
	calls int
}

func (f *fakeRabbitResolver) GetChannel(ctx context.Context, _ string) (*amqp.Channel, error) {
	f.calls++

	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return f.ch, f.err
}

func TestTenantRabbitMQChecker_ResolverError(t *testing.T) {
	res := NewTenantRabbitMQChecker(&fakeRabbitResolver{
		err: errors.New("dial tcp: connection refused"),
	}).CheckForTenant(context.Background(), "t1")

	assert.Equal(t, StatusDown, res.Status)
	assert.Contains(t, res.Error, "connection refused")
}

func TestTenantRabbitMQChecker_BreakerOpen(t *testing.T) {
	res := NewTenantRabbitMQChecker(&fakeRabbitResolver{
		err: fmt.Errorf("resolve: %w", tmcore.ErrCircuitBreakerOpen),
	}).CheckForTenant(context.Background(), "t1")

	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "circuit breaker open", res.Error)
	assert.Equal(t, "open", res.BreakerState)
}

func TestTenantRabbitMQChecker_NilManager(t *testing.T) {
	res := NewTenantRabbitMQChecker(nil).CheckForTenant(context.Background(), "t1")

	assert.Equal(t, StatusDown, res.Status)
	assert.Contains(t, res.Error, "not initialized")
}

func TestTenantRabbitMQChecker_Timeout(t *testing.T) {
	res := NewTenantRabbitMQChecker(&fakeRabbitResolver{
		delay: 50 * time.Millisecond,
		err:   context.DeadlineExceeded,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	out := res.CheckForTenant(ctx, "t1")
	assert.Equal(t, StatusDown, out.Status)
	assert.Equal(t, "timeout", out.Error)
}

func TestTenantRabbitMQChecker_Name(t *testing.T) {
	assert.Equal(t, "rabbitmq", NewTenantRabbitMQChecker(nil).Name())
}

// TestTenantRabbitMQChecker_NilChannel_ReturnsDown is a regression for the
// case where the underlying tmrabbitmq.Manager returns (nil, nil) from
// GetChannel — the resolver succeeded (no error) but the channel was
// torn down between resolution and allocation. Before the fix the checker
// would proceed to defer-close, never observe the missing channel, and
// report StatusUp, masking a genuinely missing tenant channel. The probe
// must surface a "channel not available" Down so dashboards reflect the
// real state.
func TestTenantRabbitMQChecker_NilChannel_ReturnsDown(t *testing.T) {
	res := NewTenantRabbitMQChecker(&fakeRabbitResolver{ch: nil, err: nil}).
		CheckForTenant(context.Background(), "t1")

	assert.Equal(t, StatusDown, res.Status)
	assert.Contains(t, res.Error, "channel")
}
