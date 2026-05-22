package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	tmconsumer "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/consumer"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	tmevent "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/event"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/rabbitmq"
	"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/tenantcache"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	obsRuntime "github.com/LerianStudio/lib-observability/runtime"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	workerMTDefaultPrefetchCount   = 10
	workerMTInitialBackoff         = 5 * time.Second
	workerMTMaxBackoff             = 40 * time.Second
	workerMTMaxRetryBeforeDegraded = 3
	workerMTBackoffMultiplier      = 2
	workerMTJitterMin              = 0.75
	workerMTJitterRange            = 0.5
)

type workerMultiTenantConsumerConfig struct {
	TenantClient  *tmclient.Client
	TenantCache   *tenantcache.TenantCache
	TenantLoader  *tenantcache.TenantLoader
	Dispatcher    *tmevent.EventDispatcher
	RabbitMQ      *tmrabbitmq.Manager
	Service       string
	PrefetchCount int
	Logger        libLog.Logger
}

type workerMultiTenantConsumer struct {
	tenantClient *tmclient.Client
	cache        *tenantcache.TenantCache
	loader       *tenantcache.TenantLoader
	dispatcher   *tmevent.EventDispatcher
	rabbitmq     *tmrabbitmq.Manager
	service      string
	prefetch     int
	logger       libLog.Logger

	mu           sync.RWMutex
	handlers     map[string]tmconsumer.HandlerFunc
	tenants      map[string]context.CancelFunc
	knownTenants map[string]bool
	parentCtx    context.Context
	closed       bool
	retryCounts  map[string]int
	locks        sync.Map
}

func newWorkerMultiTenantConsumer(cfg workerMultiTenantConsumerConfig) *workerMultiTenantConsumer {
	cache := cfg.TenantCache
	if cache == nil {
		cache = tenantcache.NewTenantCache()
	}

	logger := cfg.Logger
	if logger == nil {
		logger = libLog.NewNop()
	}

	prefetch := cfg.PrefetchCount
	if prefetch <= 0 {
		prefetch = workerMTDefaultPrefetchCount
	}

	return &workerMultiTenantConsumer{
		tenantClient: cfg.TenantClient,
		cache:        cache,
		loader:       cfg.TenantLoader,
		dispatcher:   cfg.Dispatcher,
		rabbitmq:     cfg.RabbitMQ,
		service:      cfg.Service,
		prefetch:     prefetch,
		logger:       logger,
		handlers:     make(map[string]tmconsumer.HandlerFunc),
		tenants:      make(map[string]context.CancelFunc),
		knownTenants: make(map[string]bool),
		retryCounts:  make(map[string]int),
	}
}

func (c *workerMultiTenantConsumer) Register(queueName string, handler tmconsumer.HandlerFunc) error {
	queueName = strings.TrimSpace(queueName)
	if queueName == "" {
		return fmt.Errorf("worker multi-tenant consumer: queue name is required")
	}

	if handler == nil {
		return fmt.Errorf("worker multi-tenant consumer: handler for queue %q is required", queueName)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[queueName] = handler
	c.logger.Log(context.Background(), libLog.LevelInfo, "registered multi-tenant worker handler", libLog.String("queue", queueName))

	return nil
}

func (c *workerMultiTenantConsumer) Run(ctx context.Context) error {
	_, tracer, _, _ := observability.NewTrackingFromContext(ctx) //nolint:dogsled // tracer is the only value needed here.

	ctx, span := tracer.Start(ctx, "consumer.worker_multi_tenant.run")
	defer span.End()

	c.mu.Lock()
	c.parentCtx = ctx
	c.mu.Unlock()

	c.logger.Log(ctx, libLog.LevelInfo, "worker multi-tenant consumer ready with circuit-breaker compliant tenant-manager client")

	return nil
}

func (c *workerMultiTenantConsumer) Close() error {
	c.mu.Lock()

	c.closed = true
	for tenantID, cancel := range c.tenants {
		c.logger.Log(context.Background(), libLog.LevelInfo, "stopping multi-tenant worker consumer", libLog.String("tenant_id", tenantID))
		cancel()
	}

	c.tenants = make(map[string]context.CancelFunc)
	c.knownTenants = make(map[string]bool)
	c.retryCounts = make(map[string]int)
	c.mu.Unlock()

	if c.tenantClient != nil {
		if err := c.tenantClient.Close(); err != nil {
			c.logger.Log(context.Background(), libLog.LevelWarn, "failed to close tenant-manager client", libLog.Err(err))
		}
	}

	return nil
}

func (c *workerMultiTenantConsumer) EnsureConsumerStarted(ctx context.Context, tenantID string) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return
	}

	_, tracer, _, _ := observability.NewTrackingFromContext(ctx) //nolint:dogsled // tracer is the only value needed here.

	ctx, span := tracer.Start(ctx, "consumer.worker_multi_tenant.ensure_consumer_started")
	defer span.End()

	lockValue, _ := c.locks.LoadOrStore(tenantID, &sync.Mutex{})

	tenantLock, ok := lockValue.(*sync.Mutex)
	if !ok {
		return
	}

	tenantLock.Lock()
	defer tenantLock.Unlock()

	if c.consumerActiveOrClosed(tenantID) {
		return
	}

	if !c.tenantKnown(tenantID) {
		if c.loader == nil {
			libOtel.HandleSpanError(span, "tenant loader is required", fmt.Errorf("tenant loader is required"))
			return
		}

		if _, err := c.loader.LoadTenant(ctx, tenantID); err != nil {
			libOtel.HandleSpanBusinessErrorEvent(span, "lazy-load tenant failed", err)
			c.logger.Log(ctx, libLog.LevelWarn, "lazy-load tenant failed", libLog.String("tenant_id", tenantID), libLog.Err(err))

			return
		}

		c.markTenantKnown(tenantID)
	}

	if c.rabbitmq == nil {
		return
	}

	startCtx := c.startContext(ctx)
	c.startTenantConsumer(startCtx, tenantID)
}

func (c *workerMultiTenantConsumer) StopConsumer(tenantID string) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return
	}

	lockValue, _ := c.locks.LoadOrStore(tenantID, &sync.Mutex{})

	tenantLock, ok := lockValue.(*sync.Mutex)
	if !ok {
		return
	}

	tenantLock.Lock()
	defer tenantLock.Unlock()

	c.mu.Lock()
	if cancel, exists := c.tenants[tenantID]; exists {
		cancel()
		delete(c.tenants, tenantID)
	}

	delete(c.knownTenants, tenantID)
	delete(c.retryCounts, tenantID)
	c.mu.Unlock()

	c.locks.Delete(tenantID)
}

func (c *workerMultiTenantConsumer) OwnsTenant(tenantID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.knownTenants[tenantID]
}

func (c *workerMultiTenantConsumer) consumerActiveOrClosed(tenantID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, active := c.tenants[tenantID]

	return active || c.closed
}

func (c *workerMultiTenantConsumer) tenantKnown(tenantID string) bool {
	c.mu.RLock()
	known := c.knownTenants[tenantID]
	c.mu.RUnlock()

	if known {
		return true
	}

	_, cached := c.cache.Get(tenantID)

	return cached
}

func (c *workerMultiTenantConsumer) markTenantKnown(tenantID string) {
	c.mu.Lock()
	c.knownTenants[tenantID] = true
	c.mu.Unlock()
}

func (c *workerMultiTenantConsumer) startContext(ctx context.Context) context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.parentCtx != nil {
		return c.parentCtx
	}

	return ctx
}

func (c *workerMultiTenantConsumer) startTenantConsumer(ctx context.Context, tenantID string) {
	c.mu.Lock()
	if _, exists := c.tenants[tenantID]; exists || c.closed {
		c.mu.Unlock()
		return
	}

	tenantCtx, cancel := context.WithCancel(tmcore.ContextWithTenantID(ctx, tenantID))
	c.tenants[tenantID] = cancel
	c.knownTenants[tenantID] = true
	c.mu.Unlock()

	obsRuntime.SafeGoWithContext(tenantCtx, c.logger, "worker-multi-tenant-consumer-"+tenantID, obsRuntime.KeepRunning, func(runCtx context.Context) {
		c.consumeTenant(runCtx, tenantID)
	})
}

func (c *workerMultiTenantConsumer) consumeTenant(ctx context.Context, tenantID string) {
	c.mu.RLock()

	handlers := make(map[string]tmconsumer.HandlerFunc, len(c.handlers))
	for queueName, handler := range c.handlers {
		handlers[queueName] = handler
	}

	c.mu.RUnlock()

	for queueName, handler := range handlers {
		queueName := queueName
		handler := handler

		obsRuntime.SafeGoWithContext(ctx, c.logger, "worker-multi-tenant-queue-"+tenantID+"-"+queueName, obsRuntime.KeepRunning, func(queueCtx context.Context) {
			c.consumeQueue(queueCtx, tenantID, queueName, handler)
		})
	}

	<-ctx.Done()
}

func (c *workerMultiTenantConsumer) consumeQueue(ctx context.Context, tenantID, queueName string, handler tmconsumer.HandlerFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if !c.consumeQueueOnce(ctx, tenantID, queueName, handler) {
			return
		}
	}
}

func (c *workerMultiTenantConsumer) consumeQueueOnce(ctx context.Context, tenantID, queueName string, handler tmconsumer.HandlerFunc) bool {
	_, tracer, _, _ := observability.NewTrackingFromContext(ctx) //nolint:dogsled // tracer is the only value needed here.

	ctx, span := tracer.Start(ctx, "consumer.worker_multi_tenant.consume_queue")
	defer span.End()

	channel, err := c.rabbitmq.GetChannel(ctx, tenantID)
	if err != nil {
		if tmcore.IsTenantSuspendedError(err) || tmcore.IsTenantPurgedError(err) {
			libOtel.HandleSpanBusinessErrorEvent(span, "tenant suspended or purged", err)
			c.removeTenant(tenantID)

			return false
		}

		c.waitRetry(ctx, tenantID, "get tenant RabbitMQ channel", err)

		return true
	}

	if err := channel.Qos(c.prefetch, 0, false); err != nil {
		_ = channel.Close()

		c.waitRetry(ctx, tenantID, "configure tenant RabbitMQ QoS", err)

		return true
	}

	messages, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()

		c.waitRetry(ctx, tenantID, "consume tenant RabbitMQ queue", err)

		return true
	}

	c.resetRetry(tenantID)
	c.processMessages(ctx, tenantID, queueName, handler, messages, channel)

	return true
}

func (c *workerMultiTenantConsumer) processMessages(ctx context.Context, tenantID, queueName string, handler tmconsumer.HandlerFunc, messages <-chan amqp.Delivery, channel *amqp.Channel) {
	notifyClose := make(chan *amqp.Error, 1)
	channel.NotifyClose(notifyClose)

	for {
		select {
		case <-ctx.Done():
			return
		case <-notifyClose:
			return
		case delivery, ok := <-messages:
			if !ok {
				return
			}

			c.handleDelivery(ctx, tenantID, queueName, handler, delivery)
		}
	}
}

func (c *workerMultiTenantConsumer) handleDelivery(ctx context.Context, tenantID, queueName string, handler tmconsumer.HandlerFunc, delivery amqp.Delivery) {
	msgCtx := tmcore.ContextWithTenantID(ctx, tenantID)
	msgCtx = libOtel.ExtractTraceContextFromQueueHeaders(msgCtx, delivery.Headers)

	_, tracer, _, _ := observability.NewTrackingFromContext(msgCtx) //nolint:dogsled // tracer is the only value needed here.

	msgCtx, span := tracer.Start(msgCtx, "consumer.worker_multi_tenant.handle_message")
	defer span.End()

	if err := handler(msgCtx, delivery); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "worker multi-tenant handler failed", err)
		c.logger.Log(msgCtx, libLog.LevelError, "multi-tenant worker handler failed", libLog.String("queue", queueName), libLog.Err(err))

		if nackErr := delivery.Nack(false, true); nackErr != nil {
			c.logger.Log(msgCtx, libLog.LevelError, "failed to nack multi-tenant worker message", libLog.Err(nackErr))
		}

		return
	}

	if ackErr := delivery.Ack(false); ackErr != nil {
		c.logger.Log(msgCtx, libLog.LevelError, "failed to ack multi-tenant worker message", libLog.Err(ackErr))
	}
}

func (c *workerMultiTenantConsumer) waitRetry(ctx context.Context, tenantID, operation string, err error) {
	delay := c.nextRetryDelay(tenantID)
	c.logger.Log(ctx, libLog.LevelWarn, "multi-tenant worker consumer retry scheduled", libLog.String("tenant_id", tenantID), libLog.String("operation", operation), libLog.String("delay", delay.String()), libLog.Err(err))

	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}

func (c *workerMultiTenantConsumer) nextRetryDelay(tenantID string) time.Duration {
	c.mu.Lock()
	retryCount := c.retryCounts[tenantID]
	c.retryCounts[tenantID] = retryCount + 1
	c.mu.Unlock()

	delay := workerMTInitialBackoff
	for range retryCount {
		delay *= workerMTBackoffMultiplier
		if delay > workerMTMaxBackoff {
			delay = workerMTMaxBackoff
			break
		}
	}

	if retryCount+1 >= workerMTMaxRetryBeforeDegraded {
		c.logger.Log(context.Background(), libLog.LevelWarn, "multi-tenant worker consumer tenant degraded", libLog.String("tenant_id", tenantID))
	}

	return applyWorkerMTJitter(delay)
}

func applyWorkerMTJitter(delay time.Duration) time.Duration {
	var randomBytes [8]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return delay
	}

	jitter := workerMTJitterMin + float64(binary.LittleEndian.Uint64(randomBytes[:]))/(1<<64)*workerMTJitterRange

	return time.Duration(float64(delay) * jitter)
}

func (c *workerMultiTenantConsumer) resetRetry(tenantID string) {
	c.mu.Lock()
	delete(c.retryCounts, tenantID)
	c.mu.Unlock()
}

func (c *workerMultiTenantConsumer) removeTenant(tenantID string) {
	c.StopConsumer(tenantID)
	c.cache.Delete(tenantID)
}
