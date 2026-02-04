package services

import (
	"github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/fetcher/pkg/mongodb/job"
	externalData "github.com/LerianStudio/fetcher/pkg/seaweedfs/external"
)

// UseCase is a struct that coordinates the handling of template files, report storage, external data sources, and report data.
type UseCase struct {
	// ExternalDataSeaweedFS is a repository used to retrieve external data from SeaweedFS storage.
	ExternalDataSeaweedFS externalData.Repository

	// JobRepository is a repository used to retrieve job data from MongoDB storage.
	JobRepository job.Repository

	// ConnectionRepository is a repository used to retrieve connection data from MongoDB storage.
	ConnectionRepository connection.Repository

	// Cryptor is used to decrypt connection passwords when creating data sources.
	Cryptor crypto.Cryptor

	// DocumentSigner is used to compute HMAC signatures for extracted data before encryption.
	// External consumers can use this signature to verify data integrity.
	DocumentSigner crypto.Signer

	// FileTTL defines the Time To Live for file (e.g., "1m", "1h", "7d", "30d"). Empty means no TTL.
	FileTTL string

	// RabbitMQPublisher is used to publish job event notifications to RabbitMQ topic exchange.
	RabbitMQPublisher rabbitmq.PublisherRepository

	// JobEventsExchange is the name of the RabbitMQ topic exchange for job events.
	JobEventsExchange string
}
