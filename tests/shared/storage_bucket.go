package shared

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// EnsureS3BucketAtHost creates the configured object-storage bucket on the
// host-mapped endpoint, idempotently. It exists to unblock the Worker's
// startup self-probe (Gate 7 of ring:dev-readyz): the s3 dep checker uses
// HeadBucket, which returns 404 until the bucket exists, and self-probe
// failure pins /health to 503 forever (until pod restart).
//
// In production parity terms: this mirrors the operator step "provision the
// bucket before deploying the worker". MinIO's startup already does this via
// MinioInfra.ensureBucket, but SeaweedFS does not — its Filer accepts S3 calls
// without bootstrapping a bucket, and the Worker's PutObject path
// auto-creates the prefix on first use, leaving HeadBucket as the only
// failing call at boot.
//
// Parameters:
//   - hostEndpoint: a URL reachable from the test runner host
//     (e.g. http://localhost:<mapped-seaweedfs-port>). NOT the container
//     network alias — the test process runs outside the docker network.
//   - bucket / accessKey / secretKey: same values the Worker container will
//     use for OBJECT_STORAGE_*; we accept any creds for SeaweedFS.
//
// Returns nil when the bucket exists or was just created. Treats
// BucketAlreadyOwnedByYou and BucketAlreadyExists as success.
func EnsureS3BucketAtHost(ctx context.Context, hostEndpoint, bucket, accessKey, secretKey string) error {
	if bucket == "" {
		return errors.New("bucket name is empty")
	}

	if hostEndpoint == "" {
		return errors.New("host endpoint is empty")
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion("us-east-1"),
		awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(hostEndpoint)
		o.UsePathStyle = true // SeaweedFS + MinIO require path-style addressing
	})

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		return nil
	}

	// Idempotency: an already-owned bucket is success.
	var ownedByYou *s3types.BucketAlreadyOwnedByYou
	if errors.As(err, &ownedByYou) {
		return nil
	}

	var alreadyExists *s3types.BucketAlreadyExists
	if errors.As(err, &alreadyExists) {
		return nil
	}

	return fmt.Errorf("create bucket %q on %s: %w", bucket, hostEndpoint, err)
}
