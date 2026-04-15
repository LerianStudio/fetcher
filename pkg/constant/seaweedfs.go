package constant

const (
	// ExternalDataKeyPrefix is the folder prefix inside the storage bucket
	// where extracted data files are stored (e.g., "external-data/{jobId}.json").
	ExternalDataKeyPrefix = "external-data"

	// Deprecated: ExternalDataBucketName is kept for backward compatibility.
	// Use ExternalDataKeyPrefix instead — "external-data" is a key prefix, not a bucket name.
	ExternalDataBucketName = ExternalDataKeyPrefix
)
