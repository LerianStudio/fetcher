package bootstrap

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/testutil"
	tmconsumer "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/consumer"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/tenantcache"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWorkerTenantLoader struct {
	cache    *tenantcache.TenantCache
	err      error
	onLoaded func(context.Context, string)
	calls    atomic.Int32
}

func (l *fakeWorkerTenantLoader) LoadTenant(ctx context.Context, tenantID string) (*tmcore.TenantConfig, error) {
	l.calls.Add(1)
	if l.err != nil {
		return nil, l.err
	}

	config := &tmcore.TenantConfig{ID: tenantID, Service: "fetcher"}
	if l.cache != nil {
		l.cache.Set(tenantID, config, time.Minute)
	}
	if l.onLoaded != nil {
		l.onLoaded(ctx, tenantID)
	}

	return config, nil
}

type fakeWorkerRabbitMQManager struct {
	channel *fakeWorkerRabbitMQChannel
	err     error
	calls   atomic.Int32
	started chan string
}

func (m *fakeWorkerRabbitMQManager) GetChannel(_ context.Context, tenantID string) (workerTenantRabbitMQChannel, error) {
	m.calls.Add(1)
	if m.started != nil {
		select {
		case m.started <- tenantID:
		default:
		}
	}
	if m.err != nil {
		return nil, m.err
	}

	return m.channel, nil
}

type fakeWorkerRabbitMQChannel struct {
	messages      chan amqp.Delivery
	notify        chan *amqp.Error
	qosCount      atomic.Int32
	consumeCount  atomic.Int32
	closeCount    atomic.Int32
	prefetchCount atomic.Int32
	qosErr        error
	consumeErr    error
	closeMessages bool
	closeOnce     sync.Once
}

func newFakeWorkerRabbitMQChannel() *fakeWorkerRabbitMQChannel {
	return &fakeWorkerRabbitMQChannel{
		messages:      make(chan amqp.Delivery, 16),
		notify:        make(chan *amqp.Error),
		closeMessages: true,
	}
}

func (c *fakeWorkerRabbitMQChannel) Qos(prefetchCount, _ int, _ bool) error {
	c.qosCount.Add(1)
	c.prefetchCount.Store(int32(prefetchCount))
	if c.qosErr != nil {
		return c.qosErr
	}

	return nil
}

func (c *fakeWorkerRabbitMQChannel) Consume(string, string, bool, bool, bool, bool, amqp.Table) (<-chan amqp.Delivery, error) {
	c.consumeCount.Add(1)
	if c.consumeErr != nil {
		return nil, c.consumeErr
	}

	return c.messages, nil
}

func (c *fakeWorkerRabbitMQChannel) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	return receiver
}

func (c *fakeWorkerRabbitMQChannel) Close() error {
	c.closeCount.Add(1)
	c.closeOnce.Do(func() {
		close(c.notify)
		if c.closeMessages {
			close(c.messages)
		}
	})

	return nil
}

type recordingAcknowledger struct {
	acks        atomic.Int32
	nacks       atomic.Int32
	rejects     atomic.Int32
	nackRequeue atomic.Bool
}

func (a *recordingAcknowledger) Ack(uint64, bool) error {
	a.acks.Add(1)

	return nil
}

func (a *recordingAcknowledger) Nack(_ uint64, _ bool, requeue bool) error {
	a.nacks.Add(1)
	a.nackRequeue.Store(requeue)

	return nil
}

func (a *recordingAcknowledger) Reject(uint64, bool) error {
	a.rejects.Add(1)

	return nil
}

func TestWorkerMultiTenantConsumer_EnsureConsumerStarted_ReentrantLoaderCallbackDoesNotDeadlock(t *testing.T) {
	t.Parallel()

	cache := tenantcache.NewTenantCache()
	channel := newFakeWorkerRabbitMQChannel()
	manager := &fakeWorkerRabbitMQManager{channel: channel, started: make(chan string, 2)}
	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		TenantCache: cache,
		RabbitMQ:    manager,
		Logger:      testBootstrapLogger(),
	})
	loader := &fakeWorkerTenantLoader{cache: cache}
	loader.onLoaded = func(ctx context.Context, tenantID string) {
		consumer.EnsureConsumerStarted(ctx, tenantID)
	}
	consumer.loader = loader
	require.NoError(t, consumer.Register("jobs", func(context.Context, amqp.Delivery) error { return nil }))

	done := make(chan struct{})
	go func() {
		consumer.EnsureConsumerStarted(context.Background(), "tenant-reentrant")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("EnsureConsumerStarted deadlocked during reentrant tenant loader callback")
	}

	assert.Eventually(t, func() bool { return manager.calls.Load() == 1 }, time.Second, 10*time.Millisecond)
	assert.Equal(t, int32(1), loader.calls.Load())
	require.NoError(t, consumer.Close())
}

func TestWorkerMultiTenantConsumer_Register_StartsKnownTenantDiscoveredBeforeHandlers(t *testing.T) {
	t.Parallel()

	cache := tenantcache.NewTenantCache()
	channel := newFakeWorkerRabbitMQChannel()
	manager := &fakeWorkerRabbitMQManager{channel: channel, started: make(chan string, 1)}
	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		TenantCache: cache,
		RabbitMQ:    manager,
		Logger:      testBootstrapLogger(),
	})
	consumer.markTenantKnown("tenant-before-register")

	consumer.EnsureConsumerStarted(context.Background(), "tenant-before-register")
	assert.Equal(t, int32(0), manager.calls.Load(), "consumer must not start with zero handlers")

	require.NoError(t, consumer.Register("jobs", func(context.Context, amqp.Delivery) error { return nil }))

	select {
	case got := <-manager.started:
		assert.Equal(t, "tenant-before-register", got)
	case <-time.After(time.Second):
		t.Fatal("Register did not start queue consumption for tenant discovered before handler registration")
	}
	require.NoError(t, consumer.Close())
}

func TestWorkerMultiTenantConsumer_EnsureConsumerStarted_PreventsDuplicateStarts(t *testing.T) {
	t.Parallel()

	cache := tenantcache.NewTenantCache()
	cache.Set("tenant-dup", &tmcore.TenantConfig{ID: "tenant-dup"}, time.Minute)
	channel := newFakeWorkerRabbitMQChannel()
	manager := &fakeWorkerRabbitMQManager{channel: channel, started: make(chan string, 2)}
	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		TenantCache: cache,
		RabbitMQ:    manager,
		Logger:      testBootstrapLogger(),
	})
	require.NoError(t, consumer.Register("jobs", func(context.Context, amqp.Delivery) error { return nil }))

	consumer.EnsureConsumerStarted(context.Background(), "tenant-dup")
	consumer.EnsureConsumerStarted(context.Background(), "tenant-dup")

	assert.Eventually(t, func() bool { return manager.calls.Load() == 1 }, time.Second, 10*time.Millisecond)
	assert.Never(t, func() bool { return manager.calls.Load() > 1 }, 100*time.Millisecond, 10*time.Millisecond)
	require.NoError(t, consumer.Close())
}

func TestWorkerMultiTenantConsumer_EnsureConsumerStarted_LazyLoadFailureDoesNotStartConsumer(t *testing.T) {
	t.Parallel()

	manager := &fakeWorkerRabbitMQManager{channel: newFakeWorkerRabbitMQChannel(), started: make(chan string, 1)}
	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		TenantLoader: &fakeWorkerTenantLoader{err: errors.New("tenant-manager unavailable")},
		RabbitMQ:     manager,
		Logger:       testBootstrapLogger(),
	})
	require.NoError(t, consumer.Register("jobs", func(context.Context, amqp.Delivery) error { return nil }))

	consumer.EnsureConsumerStarted(context.Background(), "tenant-load-fails")

	assert.Equal(t, int32(0), manager.calls.Load())
	assert.False(t, consumer.OwnsTenant("tenant-load-fails"))
}

func TestWorkerMultiTenantConsumer_ConsumeQueueOnce_ConfiguresQoSAndClosesChannelOnCancellation(t *testing.T) {
	t.Parallel()

	channel := newFakeWorkerRabbitMQChannel()
	manager := &fakeWorkerRabbitMQManager{channel: channel}
	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		RabbitMQ:      manager,
		PrefetchCount: 3,
		Logger:        testBootstrapLogger(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	channel.messages <- amqp.Delivery{Acknowledger: &recordingAcknowledger{}}
	cancel()

	keepRunning := consumer.consumeQueueOnce(ctx, "tenant-qos", "jobs", func(context.Context, amqp.Delivery) error { return nil })

	assert.True(t, keepRunning)
	assert.Equal(t, int32(1), channel.qosCount.Load())
	assert.Equal(t, int32(3), channel.prefetchCount.Load())
	assert.Equal(t, int32(1), channel.closeCount.Load())
}

func TestWorkerMultiTenantConsumer_ConsumeQueueOnce_QoSFailureClosesChannelAndRetries(t *testing.T) {
	t.Parallel()

	channel := newFakeWorkerRabbitMQChannel()
	channel.qosErr = errors.New("qos failed")
	var retryCalls atomic.Int32
	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		RabbitMQ: &fakeWorkerRabbitMQManager{channel: channel},
		Logger:   testBootstrapLogger(),
		RetryWait: func(context.Context, time.Duration) error {
			retryCalls.Add(1)

			return nil
		},
	})

	keepRunning := consumer.consumeQueueOnce(context.Background(), "tenant-qos-fails", "jobs", func(context.Context, amqp.Delivery) error { return nil })

	assert.True(t, keepRunning)
	assert.Equal(t, int32(1), channel.qosCount.Load())
	assert.Equal(t, int32(1), channel.closeCount.Load())
	assert.Equal(t, int32(1), retryCalls.Load())
}

func TestWorkerMultiTenantConsumer_ConsumeQueueOnce_ConsumeFailureClosesChannelAndRetries(t *testing.T) {
	t.Parallel()

	channel := newFakeWorkerRabbitMQChannel()
	channel.consumeErr = errors.New("consume failed")
	var retryCalls atomic.Int32
	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		RabbitMQ: &fakeWorkerRabbitMQManager{channel: channel},
		Logger:   testBootstrapLogger(),
		RetryWait: func(context.Context, time.Duration) error {
			retryCalls.Add(1)

			return nil
		},
	})

	keepRunning := consumer.consumeQueueOnce(context.Background(), "tenant-consume-fails", "jobs", func(context.Context, amqp.Delivery) error { return nil })

	assert.True(t, keepRunning)
	assert.Equal(t, int32(1), channel.consumeCount.Load())
	assert.Equal(t, int32(1), channel.closeCount.Load())
	assert.Equal(t, int32(1), retryCalls.Load())
}

func TestWorkerMultiTenantConsumer_ConsumeQueueOnce_SuspendedTenantCleansUpAndStops(t *testing.T) {
	t.Parallel()

	cache := tenantcache.NewTenantCache()
	cache.Set("tenant-suspended", &tmcore.TenantConfig{ID: "tenant-suspended"}, time.Minute)
	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		TenantCache: cache,
		RabbitMQ: &fakeWorkerRabbitMQManager{
			err: &tmcore.TenantSuspendedError{TenantID: "tenant-suspended", Status: "suspended", Message: "tenant suspended"},
		},
		Logger: testBootstrapLogger(),
	})
	consumer.markTenantKnown("tenant-suspended")

	keepRunning := consumer.consumeQueueOnce(context.Background(), "tenant-suspended", "jobs", func(context.Context, amqp.Delivery) error { return nil })

	assert.False(t, keepRunning)
	assert.False(t, consumer.OwnsTenant("tenant-suspended"))
	_, cached := cache.Get("tenant-suspended")
	assert.False(t, cached)
}

func TestWorkerMultiTenantConsumer_ProcessMessages_ClosesChannelOnContextCancellation(t *testing.T) {
	t.Parallel()

	channel := newFakeWorkerRabbitMQChannel()
	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{Logger: testBootstrapLogger()})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		consumer.processMessages(ctx, "tenant-close", "jobs", func(context.Context, amqp.Delivery) error { return nil }, channel.messages, channel)
		close(done)
	}()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("processMessages did not return after context cancellation")
	}
	assert.Equal(t, int32(1), channel.closeCount.Load())
}

func TestWorkerMultiTenantConsumer_HandleDelivery_AcksSuccessAndPropagatesTenantContext(t *testing.T) {
	t.Parallel()

	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{Logger: testBootstrapLogger()})
	ack := &recordingAcknowledger{}

	consumer.handleDelivery(context.Background(), "tenant-handler", "jobs", func(ctx context.Context, _ amqp.Delivery) error {
		assert.Equal(t, "tenant-handler", tmcore.GetTenantIDContext(ctx))

		return nil
	}, amqp.Delivery{Acknowledger: ack, Headers: amqp.Table{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"}})

	assert.Equal(t, int32(1), ack.acks.Load())
	assert.Equal(t, int32(0), ack.nacks.Load())
}

func TestWorkerMultiTenantConsumer_HandleDelivery_NacksFailureWithoutImmediateRequeue(t *testing.T) {
	t.Parallel()

	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{Logger: testBootstrapLogger()})
	ack := &recordingAcknowledger{}

	consumer.handleDelivery(context.Background(), "tenant-handler", "jobs", func(context.Context, amqp.Delivery) error {
		return errors.New("handler failed")
	}, amqp.Delivery{Acknowledger: ack})

	assert.Equal(t, int32(0), ack.acks.Load())
	assert.Equal(t, int32(1), ack.nacks.Load())
	assert.False(t, ack.nackRequeue.Load(), "handler failures must DLQ instead of hot-looping with immediate requeue")
}

func TestWorkerMultiTenantConsumer_ProcessMessages_HandlesDeliveriesWithBoundedConcurrency(t *testing.T) {
	t.Parallel()

	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{PrefetchCount: 2, Logger: testBootstrapLogger()})
	channel := newFakeWorkerRabbitMQChannel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var active atomic.Int32
	var maxActive atomic.Int32
	processed := make(chan struct{}, 3)
	release := make(chan struct{})
	handler := func(context.Context, amqp.Delivery) error {
		current := active.Add(1)
		for {
			max := maxActive.Load()
			if current <= max || maxActive.CompareAndSwap(max, current) {
				break
			}
		}

		<-release
		active.Add(-1)
		processed <- struct{}{}

		return nil
	}

	done := make(chan struct{})
	go func() {
		consumer.processMessages(ctx, "tenant-concurrency", "jobs", handler, channel.messages, channel)
		close(done)
	}()

	for i := 0; i < 3; i++ {
		channel.messages <- amqp.Delivery{Acknowledger: &recordingAcknowledger{}}
	}

	assert.Eventually(t, func() bool { return active.Load() == 2 }, time.Second, 10*time.Millisecond)
	assert.Equal(t, int32(2), maxActive.Load())
	close(release)

	for i := 0; i < 3; i++ {
		select {
		case <-processed:
		case <-time.After(time.Second):
			t.Fatal("delivery was not processed")
		}
	}
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("processMessages did not drain bounded workers")
	}
}

func TestWorkerMultiTenantConsumer_KnownTenants(t *testing.T) {
	t.Parallel()

	t.Run("returns tenants after ensure and drops them after stop", func(t *testing.T) {
		t.Parallel()

		cache := tenantcache.NewTenantCache()
		channel := newFakeWorkerRabbitMQChannel()
		manager := &fakeWorkerRabbitMQManager{channel: channel, started: make(chan string, 4)}
		consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
			TenantCache: cache,
			RabbitMQ:    manager,
			Logger:      testBootstrapLogger(),
		})
		consumer.loader = &fakeWorkerTenantLoader{cache: cache}
		require.NoError(t, consumer.Register("jobs", func(context.Context, amqp.Delivery) error { return nil }))

		assert.Empty(t, consumer.KnownTenants(), "no tenants should be known before any ensure")

		for _, tenantID := range []string{"tenant-a", "tenant-b", "tenant-c"} {
			consumer.EnsureConsumerStarted(testutil.TestContext(), tenantID)
		}

		assert.ElementsMatch(t, []string{"tenant-a", "tenant-b", "tenant-c"}, consumer.KnownTenants())

		consumer.StopConsumer("tenant-b")

		assert.ElementsMatch(t, []string{"tenant-a", "tenant-c"}, consumer.KnownTenants(),
			"stopped tenant must disappear from the known set")

		require.NoError(t, consumer.Close())
	})

	t.Run("returns a defensive copy", func(t *testing.T) {
		t.Parallel()

		consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{Logger: testBootstrapLogger()})
		consumer.markTenantKnown("tenant-x")

		snapshot := consumer.KnownTenants()
		require.Equal(t, []string{"tenant-x"}, snapshot)

		// Mutating the returned slice must not affect internal state.
		snapshot[0] = "mutated"

		assert.Equal(t, []string{"tenant-x"}, consumer.KnownTenants(),
			"mutating the returned slice must not corrupt internal state")
		assert.True(t, consumer.OwnsTenant("tenant-x"))
		assert.False(t, consumer.OwnsTenant("mutated"))
	})
}

var _ tmconsumer.HandlerFunc = func(context.Context, amqp.Delivery) error { return nil }
