package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	portStorage "github.com/LerianStudio/fetcher/pkg/ports/storage"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	tms3 "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"go.opentelemetry.io/otel/attribute"
)

// S3Config contains the configuration for S3-compatible storage.
// Works with AWS S3, MinIO, SeaweedFS S3, and other S3-compatible services.
type S3Config struct {
	// Endpoint is the custom S3-compatible endpoint URL.
	// Empty string uses AWS S3 defaults. Examples: "http://minio:9000", "http://seaweedfs:8333"
	// SSL is controlled by the URL scheme: "http://" disables SSL, "https://" enables it.
	Endpoint string

	// Region is the AWS region. Default: "us-east-1".
	Region string

	// Bucket is the S3 bucket name for storing objects.
	Bucket string

	// KeyPrefix is an optional prefix prepended to all object keys.
	KeyPrefix string

	// AccessKeyID is the AWS access key ID for authentication.
	AccessKeyID string

	// SecretAccessKey is the AWS secret access key for authentication.
	SecretAccessKey string

	// UsePathStyle enables path-style addressing (required for MinIO/SeaweedFS S3).
	UsePathStyle bool
}

// S3Repository implements storage.Repository using the AWS SDK v2.
type S3Repository struct {
	s3Client *s3.Client
	cfg      S3Config
}

// NewS3Repository creates a new S3-compatible storage repository.
func NewS3Repository(ctx context.Context, cfg S3Config) (*S3Repository, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket name is required")
	}

	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	var opts []func(*awsConfig.LoadOptions) error

	opts = append(opts, awsConfig.WithRegion(cfg.Region))

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	} else if cfg.AccessKeyID != "" || cfg.SecretAccessKey != "" {
		return nil, pkg.ValidateBusinessError(constant.ErrInvalidDataRequest, "s3_credentials", "both S3 access key ID and secret access key must be provided together (got partial credentials)")
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}

	var clientOpts []func(*s3.Options)

	// If Endpoint is empty, AWS SDK v2 uses the default AWS S3 endpoint (s3.amazonaws.com).
	// Set Endpoint for S3-compatible services: "http://minio:9000" (MinIO), "http://seaweedfs:8333" (SeaweedFS S3).
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	if cfg.UsePathStyle {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	s3Client := s3.NewFromConfig(awsCfg, clientOpts...)

	return &S3Repository{
		s3Client: s3Client,
		cfg:      cfg,
	}, nil
}

// Client exposes the underlying *s3.Client so /readyz can probe with
// HeadBucket without constructing a second AWS SDK client.
func (r *S3Repository) Client() *s3.Client {
	if r == nil {
		return nil
	}

	return r.s3Client
}

func (r *S3Repository) Bucket() string {
	if r == nil {
		return ""
	}

	return r.cfg.Bucket
}

// Endpoint returns the configured endpoint URL; empty means the AWS
// default endpoint (HTTPS).
func (r *S3Repository) Endpoint() string {
	if r == nil {
		return ""
	}

	return r.cfg.Endpoint
}

// Get downloads the object identified by objectName from the S3 bucket.
func (r *S3Repository) Get(ctx context.Context, objectName string) ([]byte, error) {
	_, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "s3.external_data.get")
	defer span.End()

	tenantObjectName, err := tms3.GetS3KeyStorageContext(ctx, objectName)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to resolve tenant object key", err)
		return nil, fmt.Errorf("tenant object key for %s: %w", objectName, err)
	}

	key := r.cfg.KeyPrefix + tenantObjectName

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("s3.object_name", key),
		attribute.String("s3.bucket", r.cfg.Bucket),
	)

	input := &s3.GetObjectInput{
		Bucket: aws.String(r.cfg.Bucket),
		Key:    aws.String(key),
	}

	result, err := r.s3Client.GetObject(ctx, input)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			libOpentelemetry.HandleSpanError(span, "Object not found in S3", err)
			return nil, fmt.Errorf("object not found: %s", objectName)
		}

		libOpentelemetry.HandleSpanError(span, "Failed to download object from S3", err)

		return nil, fmt.Errorf("s3 download failed for %s: %w", objectName, err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to read S3 response body", err)
		return nil, fmt.Errorf("s3 read response failed for %s: %w", objectName, err)
	}

	span.SetAttributes(attribute.Int("s3.response_size", len(data)))

	return data, nil
}

// Put uploads data to the S3 bucket under the given objectName.
func (r *S3Repository) Put(ctx context.Context, objectName string, data []byte) error {
	logger, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "s3.external_data.put")
	defer span.End()

	tenantObjectName, err := tms3.GetS3KeyStorageContext(ctx, objectName)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to resolve tenant object key", err)
		return fmt.Errorf("tenant object key for %s: %w", objectName, err)
	}

	key := r.cfg.KeyPrefix + tenantObjectName

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("s3.object_name", key),
		attribute.String("s3.bucket", r.cfg.Bucket),
		attribute.Int("s3.data_size", len(data)),
	)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.cfg.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/octet-stream"),
	}

	if _, err := r.s3Client.PutObject(ctx, input); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to upload object to S3", err)
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Error communicating with S3: %v", err))

		return fmt.Errorf("s3 upload failed for %s: %w", objectName, err)
	}

	return nil
}

// Compile-time interface check.
var _ portStorage.Repository = (*S3Repository)(nil)
