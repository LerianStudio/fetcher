package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// JobResultData contains information about the extraction result.
type JobResultData struct {
	Path      string `json:"path,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
	RowCount  int64  `json:"rowCount,omitempty"`
	Format    string `json:"format,omitempty"`
}

// JobNotification represents a job event from RabbitMQ.
type JobNotification struct {
	JobID           string         `json:"jobId"`
	OrganizationID  string         `json:"organizationId"`
	Status          string         `json:"status"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	Result          *JobResultData `json:"result,omitempty"`
	ExecutionTimeMs int64          `json:"executionTimeMs,omitempty"`
	CompletedAt     *time.Time     `json:"completedAt,omitempty"`
}

// RabbitMQEventConsumer listens for job completion events.
type RabbitMQEventConsumer struct {
	amqpURL string
}

// NewRabbitMQEventConsumer creates a new event consumer.
func NewRabbitMQEventConsumer(amqpURL string) (*RabbitMQEventConsumer, error) {
	// Verify connection is possible
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	_ = conn.Close()

	return &RabbitMQEventConsumer{
		amqpURL: amqpURL,
	}, nil
}

// WaitForJobEvent waits for a specific job's completion or failure event.
// Uses the persistent test.job.events queue that is pre-bound during chaos.
func (c *RabbitMQEventConsumer) WaitForJobEvent(ctx context.Context, jobID string, timeout time.Duration) (*JobNotification, error) {
	// Create fresh connection for this wait operation
	conn, err := amqp.Dial(c.amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Consume from the persistent test queue (already bound to fetcher.job.events during setup)
	msgs, err := ch.Consume(
		"test.job.events", // Queue name
		"",                // Consumer tag (auto-generated)
		true,              // Auto-ack
		false,             // Not exclusive (allow multiple consumers)
		false,             // No-local
		false,             // No-wait
		nil,               // Arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to consume messages: %w", err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				return nil, fmt.Errorf("channel closed")
			}

			var notification JobNotification
			if err := json.Unmarshal(msg.Body, &notification); err != nil {
				continue // Skip invalid messages
			}

			if notification.JobID == jobID {
				return &notification, nil
			} else {
				// Log mismatch for debugging
				fmt.Printf("[DEBUG] Received event for different job: %s (waiting for %s)\n", notification.JobID, jobID)
			}
		case <-timer.C:
			fmt.Printf("[DEBUG] Timeout waiting for job event %s after %v\n", jobID, timeout)
			return nil, fmt.Errorf("timeout waiting for job event %s", jobID)
		}
	}
}

// Close is a no-op since connections are created per-call.
func (c *RabbitMQEventConsumer) Close() error {
	return nil
}
