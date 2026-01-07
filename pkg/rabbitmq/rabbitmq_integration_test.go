//go:build integration
// +build integration

package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	libRabbitmq "github.com/LerianStudio/lib-commons/v2/commons/rabbitmq"
)

// stressTestConfig groups the RabbitMQ configuration required for the integration test.
type stressTestConfig struct {
	URI        string
	Host       string
	PortAMQP   string
	PortHost   string
	User       string
	Pass       string
	Exchange   string
	Queue      string
	RoutingKey string
	HealthURL  string
}

// loadStressConfig builds the configuration from environment variables with sensible defaults for the test hints.
func loadStressConfig(t *testing.T) stressTestConfig {
	t.Helper()

	return stressTestConfig{
		URI:        getenv("RABBITMQ_URI", "amqp"),
		Host:       getenv("RABBITMQ_HOST", "fetcher-rabbitmq"),
		PortAMQP:   getenv("RABBITMQ_PORT_AMQP", "3007"),
		PortHost:   getenv("RABBITMQ_PORT_HOST", "3008"),
		User:       getenv("RABBITMQ_DEFAULT_USER", "plugin"),
		Pass:       getenv("RABBITMQ_DEFAULT_PASS", "Lerian@123"),
		Exchange:   getenv("RABBITMQ_EXCHANGE", "fetcher.extract-external-data.exchange"),
		Queue:      getenv("RABBITMQ_FETCHER_WORK_QUEUE", "fetcher.extract-external-data.queue"),
		RoutingKey: getenv("RABBITMQ_FETCHER_WORK_KEY", "fetcher.extract-external-data.key"),
		HealthURL:  getenv("RABBITMQ_HEALTH_CHECK_URL", "http://fetcher-rabbitmq:3008"),
	}
}

// getenv retrieves the value of the environment variable named by the key.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// stressMessage represents the structure of messages used in the stress test.
type stressMessage struct {
	ID string `json:"id"`
	N  int    `json:"n"`
}

// TestRabbitMQStressProducerAndConsumer publishes 1000 messages and consumes all with concurrency=10,
// exercising ProducerDefault and ConsumerLoop against a real RabbitMQ instance.
func TestRabbitMQStressProducerAndConsumer(t *testing.T) {
	if os.Getenv("RUN_RABBITMQ_INTEGRATION") == "" {
		t.Skip("set RUN_RABBITMQ_INTEGRATION=1 to run this integration test")
	}

	cfg := loadStressConfig(t)
	logger := &libLog.GoLogger{Level: libLog.InfoLevel}

	escapedUser := url.PathEscape(cfg.User)
	escapedPass := url.QueryEscape(cfg.Pass)
	connectionString := fmt.Sprintf("%s://%s:%s@%s:%s", cfg.URI, escapedUser, escapedPass, cfg.Host, cfg.PortAMQP)

	conn := &libRabbitmq.RabbitMQConnection{
		ConnectionStringSource: connectionString,
		HealthCheckURL:         cfg.HealthURL,
		Host:                   cfg.Host,
		Port:                   cfg.PortHost,
		User:                   cfg.User,
		Pass:                   cfg.Pass,
		Queue:                  cfg.Queue,
		Logger:                 logger,
	}

	if err := conn.EnsureChannel(); err != nil {
		t.Fatalf("failed to ensure channel: %v", err)
	}

	ch := conn.Channel
	queueName := fmt.Sprintf("%s.integration-%d", cfg.Queue, time.Now().UTC().UnixNano())

	// Declare a temporary queue for the test messages
	if _, err := ch.QueueDeclare(queueName, false, true, false, false, nil); err != nil {
		t.Fatalf("failed to declare queue: %v", err)
	}

	// Bind the queue to the exchange with the routing key
	if err := ch.QueueBind(queueName, cfg.RoutingKey, cfg.Exchange, false, nil); err != nil {
		t.Fatalf("failed to bind queue to exchange: %v", err)
	}

	adapter := NewRabbitMQAdapter(conn)
	consumeCtx, consumeCancel := context.WithTimeout(testContextWithHeader("stress-consume"), 300*time.Second)
	defer consumeCancel()

	// Total messages to send and receive. Adjust as needed for stress level.
	const totalMessages = 1000
	const concurrency = 10
	const consumerSleep = 500 * time.Millisecond // Simulated processing time per message

	var (
		received   int64
		duplicates int64
		invalid    int64
	)

	// Mutex to protect access to the 'seen' map
	var mu sync.Mutex
	// Map to track seen message IDs for duplicate detection
	seen := make(map[string]struct{})

	// WaitGroup to track received messages
	recvWG := sync.WaitGroup{}
	recvWG.Add(totalMessages) // We expect to receive 'totalMessages' messages

	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		defer recvWG.Done()

		var msg stressMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			atomic.AddInt64(&invalid, 1)
			return nil
		}

		time.Sleep(consumerSleep)

		mu.Lock()
		if _, ok := seen[msg.ID]; ok {
			atomic.AddInt64(&duplicates, 1)
		} else {
			seen[msg.ID] = struct{}{}
		}
		mu.Unlock()

		// Check if we've received all messages and cancel the context if so to stop consuming
		if atomic.AddInt64(&received, 1) == totalMessages {
			consumeCancel()
		}

		return nil
	}

	consumerErrCh := make(chan error, 1)
	go func() {
		consumerErrCh <- adapter.ConsumerLoop(consumeCtx, queueName, concurrency, handler)
	}()

	producerErrCh := make(chan error, 1)
	go func() {
		producerErrCh <- runProducerStress(adapter, cfg.Exchange, cfg.RoutingKey, totalMessages)
	}()

	waitCh := make(chan struct{})
	go func() {
		// Wait for all messages to be received
		recvWG.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
	case <-consumeCtx.Done():
		// If we canceled the consume context after receiving everything, just wait
		// for the wait group to finish instead of failing the test.
		if atomic.LoadInt64(&received) != totalMessages {
			t.Fatalf("timeout waiting messages: received=%d/%d duplicates=%d invalid=%d err=%v",
				received, totalMessages, duplicates, invalid, consumeCtx.Err())
		}

		// Wait for the wait group to ensure all messages are processed
		<-waitCh
	}

	if err := <-producerErrCh; err != nil {
		t.Fatalf("producer failed: %v", err)
	}

	if err := <-consumerErrCh; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("consumer failed: %v", err)
	}

	// Final verification of results after all processing is done
	// Check for duplicates and invalid messages and total received count
	if duplicates != 0 || invalid != 0 || received != totalMessages {
		t.Fatalf("unexpected results: received=%d/%d duplicates=%d invalid=%d", received, totalMessages, duplicates, invalid)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(testContextWithHeader("stress-shutdown"), 5*time.Second)
	defer shutdownCancel()

	if err := adapter.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("failed to shutdown adapter: %v", err)
	}
}

// runProducerStress publishes 'total' messages concurrently using 'workers' goroutines.
func runProducerStress(adapter *RabbitMQAdapter, exchange, routingKey string, total int) error {
	ctx, cancel := context.WithTimeout(testContextWithHeader("stress-produce"), 20*time.Second)
	defer cancel()

	const workers = 5
	runID := time.Now().UTC().UnixNano()

	type result struct {
		err error
	}

	jobs := make(chan stressMessage, workers)
	results := make(chan result, workers)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range jobs {
				body, err := json.Marshal(msg)
				if err != nil {
					results <- result{err: err}
					return
				}

				if err := adapter.ProducerDefault(ctx, exchange, routingKey, body, nil); err != nil {
					results <- result{err: err}
					return
				}
			}
		}()
	}

	for i := 0; i < total; i++ {
		jobs <- stressMessage{
			ID: fmt.Sprintf("msg-%d-%d", runID, i),
			N:  i,
		}
	}
	close(jobs)

	wg.Wait()
	close(results)

	for res := range results {
		if res.err != nil {
			return res.err
		}
	}

	return nil
}
