package readyz

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3HeadBucketAPI matches s3.Client's HeadBucket signature so a real
// *s3.Client satisfies it directly; tests plug in a fake.
type S3HeadBucketAPI interface {
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

// S3BucketChecker probes bucket reachability via HeadBucket — cheap on AWS
// and on S3-compatible stacks (MinIO, SeaweedFS-S3) since it returns headers
// only, with no listing. Error classification matches on string signatures
// because the v2 SDK's typed errors are not exported at a stable path and
// vary between AWS and S3-compatible implementations.
type S3BucketChecker struct {
	name     string
	api      S3HeadBucketAPI
	bucket   string
	endpoint string
}

// NewS3BucketChecker reports under dep name "s3". endpoint is used only for
// TLS posture detection — empty endpoint means AWS default (always HTTPS).
func NewS3BucketChecker(api S3HeadBucketAPI, bucket, endpoint string) *S3BucketChecker {
	return &S3BucketChecker{
		name:     "s3",
		api:      api,
		bucket:   bucket,
		endpoint: endpoint,
	}
}

func (c *S3BucketChecker) Name() string { return c.name }

func (c *S3BucketChecker) Check(ctx context.Context) DependencyCheck {
	tlsOn := tlsOrFalse(detectS3TLS(c.endpoint))

	if c.api == nil {
		return DependencyCheck{
			Status: StatusDown,
			TLS:    TLSPtr(tlsOn),
			Error:  "s3 client not initialized",
		}
	}

	if c.bucket == "" {
		return DependencyCheck{
			Status: StatusDown,
			TLS:    TLSPtr(tlsOn),
			Error:  "s3 bucket not configured",
		}
	}

	start := time.Now()
	_, err := c.api.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(c.bucket)})
	elapsed := time.Since(start)

	if err == nil {
		return DependencyCheck{
			Status:    StatusUp,
			LatencyMs: elapsed.Milliseconds(),
			TLS:       TLSPtr(tlsOn),
		}
	}

	return DependencyCheck{
		Status:    StatusDown,
		LatencyMs: elapsed.Milliseconds(),
		TLS:       TLSPtr(tlsOn),
		Error:     classifyS3Err(ctx, err),
	}
}

// classifyS3Err collapses well-known HeadBucket failures to short labels so
// alerts can distinguish "bucket missing" (misconfiguration) from "access
// denied" (credential rotation / IAM drift) from generic network errors.
func classifyS3Err(ctx context.Context, err error) string {
	if ctxMsg := classifyErr(ctx, err); ctxMsg == "timeout" || ctxMsg == "canceled" {
		return ctxMsg
	}

	msg := err.Error()
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(msg, "NotFound"),
		strings.Contains(lower, "not found"),
		strings.Contains(lower, "nosuchbucket"):
		return "bucket not found"
	case strings.Contains(msg, "AccessDenied"),
		strings.Contains(lower, "access denied"),
		strings.Contains(lower, "forbidden"):
		return "access denied"
	}

	return sanitize(msg)
}
