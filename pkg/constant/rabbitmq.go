package constant

// RabbitMQ message signing headers
const (
	// HeaderMessageSignature contains the HMAC-SHA256 signature of the message payload
	HeaderMessageSignature = "x-message-signature"

	// HeaderSignatureTimestamp contains the Unix timestamp when the signature was created
	HeaderSignatureTimestamp = "t"

	// HeaderSignatureVersion contains the version of the signature algorithm (e.g., "v1")
	HeaderSignatureVersion = "signature-version"
)
