package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	libBackoff "github.com/LerianStudio/lib-commons/v5/commons/backoff"
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
)

type workerTenantLoader interface {
	LoadTenant(ctx context.Context, tenantID string) (*tmcore.TenantConfig, error)
}

type workerTenantRabbitMQ interface {
	GetChannel(ctx context.Context, tenantID string) (workerTenantRabbitMQChannel, error)
}

type workerTenantRabbitMQChannel interface {
	Qos(prefetchCount, prefetchSize int, global bool) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
	Close() error
}

type realWorkerTenantRabbitMQManager struct {
	manager *tmrabbitmq.Manager
}

func newRealWorkerTenantRabbitMQManager(manager *tmrabbitmq.Manager) *realWorkerTenantRabbitMQManager {
	if manager == nil {
		return nil
	}

	return &realWorkerTenantRabbitMQManager{manager: manager}
}

func (m *realWorkerTenantRabbitMQManager) GetChannel(ctx context.Context, tenantID string) (workerTenantRabbitMQChannel, error) {
	return m.manager.GetChannel(ctx, tenantID)
}

type workerMultiTenantConsumerConfig struct {
	TenantClient  *tmclient.Client
	TenantCache   *tenantcache.TenantCache
	TenantLoader  workerTenantLoader
	Dispatcher    *tmevent.EventDispatcher
	RabbitMQ      workerTenantRabbitMQ
	Service       string
	PrefetchCount int
	Logger        libLog.Logger
	RetryWait     func(context.Context, time.Duration) error
}

type workerMultiTenantConsumer struct {
	tenantClient *tmclient.Client
	cache        *tenantcache.TenantCache
	loader       workerTenantLoader
	dispatcher   *tmevent.EventDispatcher
	rabbitmq     workerTenantRabbitMQ
	service      string
	prefetch     int
	logger       libLog.Logger
	retryWait    func(context.Context, time.Duration) error

	mu           sync.RWMutex
	handlers     map[string]tmconsumer.HandlerFunc
	tenants      map[string]context.CancelFunc
	tenantCtxs   map[string]context.Context
	tenantQueues map[string]map[string]bool
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

	retryWait := cfg.RetryWait
	if retryWait == nil {
		retryWait = libBackoff.WaitContext
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
		retryWait:    retryWait,
		handlers:     make(map[string]tmconsumer.HandlerFunc),
		tenants:      make(map[string]context.CancelFunc),
		tenantCtxs:   make(map[string]context.Context),
		tenantQueues: make(map[string]map[string]bool),
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

	startTenants := make([]string, 0)

	c.mu.Lock()

	c.handlers[queueName] = handler

	for tenantID := range c.knownTenants {
		if _, active := c.tenants[tenantID]; active || !c.closed {
			startTenants = append(startTenants, tenantID)
		}
	}

	for _, tenantID := range c.cache.TenantIDs() {
		if !c.knownTenants[tenantID] {
			c.knownTenants[tenantID] = true
			startTenants = append(startTenants, tenantID)
		}
	}
	c.mu.Unlock()

	c.logger.Log(context.Background(), libLog.LevelInfo, "registered multi-tenant worker handler", libLog.String("queue", queueName))

	startCtx := c.startContext(context.Background())

	for _, tenantID := range startTenants {
		c.startTenantConsumer(startCtx, tenantID)
	}

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
	c.tenantCtxs = make(map[string]context.Context)
	c.tenantQueues = make(map[string]map[string]bool)
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
	if c.consumerActiveOrClosed(tenantID) {
		tenantLock.Unlock()
		return
	}

	known := c.tenantKnown(tenantID)
	hasHandlers := c.hasHandlers()
	tenantLock.Unlock()

	if !known {
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

	if !hasHandlers {
		c.logger.Log(ctx, libLog.LevelWarn, "multi-tenant worker consumer start deferred until handlers are registered", libLog.String("tenant_id", tenantID))
		return
	}

	if c.rabbitmq == nil {
		return
	}

	tenantLock.Lock()
	defer tenantLock.Unlock()

	if c.consumerActiveOrClosed(tenantID) {
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

	delete(c.tenantCtxs, tenantID)
	delete(c.tenantQueues, tenantID)
	delete(c.knownTenants, tenantID)
	delete(c.retryCounts, tenantID)
	c.mu.Unlock()

	c.locks.Delete(tenantID)
}

func (c *workerMultiTenantConsumer) hasHandlers() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.handlers) > 0
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
	type queueStart struct {
		name    string
		handler tmconsumer.HandlerFunc
	}

	c.mu.Lock()
	if c.closed || len(c.handlers) == 0 {
		c.mu.Unlock()
		return
	}

	if _, exists := c.tenantCtxs[tenantID]; !exists {
		var cancel context.CancelFunc

		tenantCtx, cancel := context.WithCancel(tmcore.ContextWithTenantID(ctx, tenantID))
		c.tenants[tenantID] = cancel
		c.tenantCtxs[tenantID] = tenantCtx
	}

	if c.tenantQueues[tenantID] == nil {
		c.tenantQueues[tenantID] = make(map[string]bool)
	}

	queues := make([]queueStart, 0, len(c.handlers))
	for queueName, handler := range c.handlers {
		if c.tenantQueues[tenantID][queueName] {
			continue
		}

		c.tenantQueues[tenantID][queueName] = true
		queues = append(queues, queueStart{name: queueName, handler: handler})
	}

	c.knownTenants[tenantID] = true
	c.mu.Unlock()

	for _, queue := range queues {
		queueName := queue.name
		handler := queue.handler

		obsRuntime.SafeGoWithContext(ctx, c.logger, "worker-multi-tenant-queue-"+tenantID+"-"+queueName, obsRuntime.KeepRunning, func(queueCtx context.Context) {
			c.consumeQueue(queueCtx, tenantID, queueName, handler)
		})
	}
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

func (c *workerMultiTenantConsumer) processMessages(ctx context.Context, tenantID, queueName string, handler tmconsumer.HandlerFunc, messages <-chan amqp.Delivery, channel workerTenantRabbitMQChannel) {
	defer func() {
		if err := channel.Close(); err != nil {
			c.logger.Log(ctx, libLog.LevelWarn, "failed to close tenant RabbitMQ channel", libLog.String("tenant_id", tenantID), libLog.String("queue", queueName), libLog.Err(err))
		}
	}()

	notifyClose := make(chan *amqp.Error, 1)
	channel.NotifyClose(notifyClose)

	limit := c.prefetch
	if limit <= 0 {
		limit = 1
	}

	semaphore := make(chan struct{}, limit)

	var wg sync.WaitGroup

	defer wg.Wait()

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

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			case <-notifyClose:
				return
			}

			wg.Add(1)
			obsRuntime.SafeGoWithContext(ctx, c.logger, "worker-multi-tenant-message-"+tenantID+"-"+queueName, obsRuntime.KeepRunning, func(msgCtx context.Context) {
				defer func() {
					<-semaphore
					wg.Done()
				}()

				c.handleDelivery(msgCtx, tenantID, queueName, handler, delivery)
			})
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

		if nackErr := delivery.Nack(false, false); nackErr != nil {
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

	if waitErr := c.retryWait(ctx, delay); waitErr != nil {
		c.logger.Log(ctx, libLog.LevelDebug, "multi-tenant worker consumer retry wait interrupted", libLog.String("tenant_id", tenantID), libLog.Err(waitErr))
	}
}

func (c *workerMultiTenantConsumer) nextRetryDelay(tenantID string) time.Duration {
	c.mu.Lock()
	retryCount := c.retryCounts[tenantID]
	c.retryCounts[tenantID] = retryCount + 1
	c.mu.Unlock()

	delay := libBackoff.ExponentialWithJitter(workerMTInitialBackoff, retryCount)
	if delay > workerMTMaxBackoff {
		delay = libBackoff.FullJitter(workerMTMaxBackoff)
	}

	if retryCount+1 >= workerMTMaxRetryBeforeDegraded {
		c.logger.Log(context.Background(), libLog.LevelWarn, "multi-tenant worker consumer tenant degraded", libLog.String("tenant_id", tenantID))
	}

	return delay
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
