package constant

const (
	// DefaultPrefetchCount is the QoS prefetch count for multi-tenant RabbitMQ consumers.
	// Each per-tenant consumer goroutine processes this many messages at a time.
	DefaultPrefetchCount = 10
)
