package rabbitmq

import (
	"context"
	"errors"
	"testing"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libConstants "github.com/LerianStudio/lib-commons/v2/commons/constants"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
)

func TestProducerDefaultPublishesMessage(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := &RabbitMQAdapter{conn: conn}

	body := []byte(`{"foo":"bar"}`)
	if err := adapter.ProducerDefault(testContextWithHeader("req-123"), "exchange", "key", body); err != nil {
		t.Fatalf("ProducerDefault returned error: %v", err)
	}

	if channel.publishAttempts != 1 {
		t.Fatalf("expected one publish attempt, got %d", channel.publishAttempts)
	}

	if len(channel.published) != 1 {
		t.Fatalf("expected one published message, got %d", len(channel.published))
	}

	record := channel.published[0]
	if record.exchange != "exchange" || record.key != "key" {
		t.Fatalf("unexpected routing: exchange=%s key=%s", record.exchange, record.key)
	}

	if got := string(record.message.Body); got != string(body) {
		t.Fatalf("unexpected message body: %s", got)
	}

	if record.message.ContentType != "application/json" {
		t.Fatalf("unexpected content type: %s", record.message.ContentType)
	}

	headerID, _ := record.message.Headers[libConstants.HeaderID].(string)
	if headerID != "req-123" {
		t.Fatalf("expected header id req-123, got %s", headerID)
	}

	if retry, _ := record.message.Headers["x-retry-count"].(int); retry != 0 {
		t.Fatalf("expected retry count 0, got %d", retry)
	}
}

func TestProducerDefaultRetriesWhenPublishFails(t *testing.T) {
	t.Parallel()

	first := newTestAMQPChannel()
	first.publishErr = errors.New("publish failed")

	second := newTestAMQPChannel()

	conn := &testRabbitConnection{
		channels: []amqpChannel{first, second},
	}
	adapter := &RabbitMQAdapter{conn: conn}

	if err := adapter.ProducerDefault(testContextWithHeader("req-10"), "ex", "rk", []byte(`{"hello":"world"}`)); err != nil {
		t.Fatalf("ProducerDefault returned error: %v", err)
	}

	if conn.calls != 2 {
		t.Fatalf("expected two EnsureChannel calls, got %d", conn.calls)
	}

	if first.publishAttempts != 1 {
		t.Fatalf("expected first channel publish attempt once, got %d", first.publishAttempts)
	}

	if second.publishAttempts != 1 || len(second.published) != 1 {
		t.Fatalf("expected second channel to publish message once; attempts=%d published=%d", second.publishAttempts, len(second.published))
	}

	if first.closeCalls != 1 {
		t.Fatalf("expected first channel to be closed, got %d close calls", first.closeCalls)
	}
}

func TestProducerDefaultReturnsErrorWhenEnsureChannelFails(t *testing.T) {
	t.Parallel()

	conn := &testRabbitConnection{err: errors.New("ensure failed")}
	adapter := &RabbitMQAdapter{conn: conn}

	if err := adapter.ProducerDefault(testContextWithHeader("req-err"), "ex", "key", []byte(`{}`)); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestProducerDefaultReturnsErrorAfterShutdown(t *testing.T) {
	t.Parallel()

	conn := &testRabbitConnection{}
	adapter := &RabbitMQAdapter{conn: conn}
	adapter.shutdown.Store(true)

	if err := adapter.ProducerDefault(testContextWithHeader("req-shutdown"), "ex", "key", []byte(`{}`)); err == nil {
		t.Fatalf("expected error when adapter is shut down")
	}
}

func TestConsumerLoopAckOnSuccess(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(testContextWithHeader("req-ack"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body:         []byte(`{"payload":"ok"}`),
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}
	adapter := &RabbitMQAdapter{conn: conn}

	handled := make(chan []byte, 1)
	handler := func(ctx context.Context, body []byte) error {
		handled <- body
		cancel()
		return nil
	}

	if err := adapter.ConsumerLoop(ctx, "queue-ack", 1, handler); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	select {
	case body := <-handled:
		if string(body) != `{"payload":"ok"}` {
			t.Fatalf("unexpected body, got %s", string(body))
		}
	case <-time.After(time.Second):
		t.Fatal("handler was not invoked")
	}

	waitUntil(t, func() bool { return ack.acks == 1 && ack.nacks == 0 }, time.Second)
}

func TestConsumerLoopNackOnHandlerError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(testContextWithHeader("req-nack"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body:         []byte(`{"payload":"fail"}`),
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}
	adapter := &RabbitMQAdapter{conn: conn}

	processed := make(chan struct{})
	handler := func(context.Context, []byte) error {
		close(processed)
		return errors.New("handler failed")
	}

	go func() {
		<-processed
		cancel()
	}()

	if err := adapter.ConsumerLoop(ctx, "queue-nack", 1, handler); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	waitUntil(t, func() bool { return ack.nacks == 1 && ack.acks == 0 }, time.Second)
}

func TestShutdownClosesResources(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := &RabbitMQAdapter{
		conn:    conn,
		channel: channel,
	}

	if err := adapter.Shutdown(testContextWithHeader("req-shutdown")); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	if !adapter.shutdown.Load() {
		t.Fatalf("expected shutdown flag to be set")
	}

	if adapter.channel != nil {
		t.Fatalf("expected channel to be cleared after shutdown")
	}

	if conn.closeCalls != 1 {
		t.Fatalf("expected connection close once, got %d", conn.closeCalls)
	}

	if channel.closeCalls != 1 || !channel.closed {
		t.Fatalf("expected channel closed once, got %d", channel.closeCalls)
	}
}

// Helpers/Mocks --------------------------------------------------------------------

type testRabbitConnection struct {
	channel    amqpChannel
	channels   []amqpChannel
	err        error
	calls      int
	closeCalls int
	closeErr   error
}

func (t *testRabbitConnection) EnsureChannel() (amqpChannel, error) {
	t.calls++
	if t.err != nil {
		return nil, t.err
	}

	if len(t.channels) > 0 {
		idx := t.calls - 1
		if idx >= len(t.channels) {
			idx = len(t.channels) - 1
		}
		ch := t.channels[idx]
		if tch, ok := ch.(*testAMQPChannel); ok {
			tch.closed = false
		}
		t.channel = ch
		return ch, nil
	}

	if t.channel == nil {
		t.channel = newTestAMQPChannel()
	}

	if ch, ok := t.channel.(*testAMQPChannel); ok {
		ch.closed = false
	}

	return t.channel, nil
}

func (t *testRabbitConnection) Close() error {
	t.closeCalls++
	return t.closeErr
}

type publishedRecord struct {
	exchange  string
	key       string
	mandatory bool
	immediate bool
	message   amqp.Publishing
}

type testAMQPChannel struct {
	publishErr      error
	publishErrs     []error
	publishAttempts int
	consumeErr      error
	qosErr          error

	deliveries chan amqp.Delivery
	published  []publishedRecord

	consumeQueue   string
	consumeAutoAck bool

	cancelCalls     int
	cancelConsumer  string
	cancelNoWait    bool
	closeCalls      int
	closeShouldFail bool

	notifyClose chan *amqp.Error
	closed      bool
}

func newTestAMQPChannel() *testAMQPChannel {
	return &testAMQPChannel{
		deliveries: make(chan amqp.Delivery, 1),
	}
}

func (t *testAMQPChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	t.publishAttempts++

	var err error
	if len(t.publishErrs) > 0 {
		err = t.publishErrs[0]
		t.publishErrs = t.publishErrs[1:]
	} else {
		err = t.publishErr
	}

	if err != nil {
		return err
	}

	t.published = append(t.published, publishedRecord{
		exchange:  exchange,
		key:       key,
		mandatory: mandatory,
		immediate: immediate,
		message:   msg,
	})

	return nil
}

func (t *testAMQPChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	if t.consumeErr != nil {
		return nil, t.consumeErr
	}

	t.consumeQueue = queue
	t.consumeAutoAck = autoAck
	t.cancelConsumer = consumer

	if t.deliveries == nil {
		t.deliveries = make(chan amqp.Delivery, 1)
	}

	return t.deliveries, nil
}

func (t *testAMQPChannel) IsClosed() bool {
	return t.closed
}

func (t *testAMQPChannel) Cancel(consumer string, noWait bool) error {
	t.cancelCalls++
	t.cancelConsumer = consumer
	t.cancelNoWait = noWait
	return nil
}

func (t *testAMQPChannel) Close() error {
	t.closeCalls++
	if t.closeShouldFail {
		return errors.New("close fail")
	}

	t.closed = true
	return nil
}

func (t *testAMQPChannel) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	t.notifyClose = receiver
	return receiver
}

func (t *testAMQPChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	if t.qosErr != nil {
		return t.qosErr
	}

	return nil
}

type testAcknowledger struct {
	acks  int
	nacks int
}

func (t *testAcknowledger) Ack(uint64, bool) error {
	t.acks++
	return nil
}

func (t *testAcknowledger) Nack(uint64, bool, bool) error {
	t.nacks++
	return nil
}

func (t *testAcknowledger) Reject(uint64, bool) error {
	t.nacks++
	return nil
}

func waitUntil(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	if !condition() {
		t.Fatalf("condition not met within %s", timeout)
	}
}

func testContextWithHeader(header string) context.Context {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	values := &libCommons.CustomContextKeyValue{
		HeaderID: header,
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}

	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}
