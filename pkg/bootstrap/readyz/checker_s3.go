package readyz

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3HeadBucketAPI is the narrow surface the S3BucketChecker needs from the
// AWS SDK. It matches s3.Client's HeadBucket method signature, so a real
// *s3.Client satisfies this directly. Tests plug in a fake.
type S3HeadBucketAPI interface {
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

// S3BucketChecker verifies that the worker's S3 bucket exists and is
// reachable. This is a worker-only dependency (the manager does not speak S3
// directly), so only the worker bootstrap registers this checker.
//
// Probe strategy: HeadBucket with the caller's context. The HeadBucket call
// is deliberately cheap on both AWS S3 and S3-compatible stacks (MinIO,
// SeaweedFS-S3) — it only returns headers, no object listing.
//
// Error classification uses the AWS SDK error string because the v2 SDK's
// typed errors are not exported at a stable package path and vary between
// AWS and S3-compatible implementations. We look for two well-known
// signatures ("NotFound" / "AccessDenied") and fall back to a sanitised
// error for everything else.
type S3BucketChecker struct {
	name     string
	api      S3HeadBucketAPI
	bucket   string
	endpoint string
}

// NewS3BucketChecker constructs a checker reporting under dep name "s3".
// endpoint is used only for TLS posture detection — empty endpoint means
// "AWS default", which is always HTTPS per detectS3TLS.
func NewS3BucketChecker(api S3HeadBucketAPI, bucket, endpoint string) *S3BucketChecker {
	return &S3BucketChecker{
		name:     "s3",
		api:      api,
		bucket:   bucket,
		endpoint: endpoint,
	}
}

// Name returns the stable dep identifier ("s3").
func (c *S3BucketChecker) Name() string { return c.name }

// Check runs HeadBucket. See the package-level comment for the error
// classification contract.
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

// classifyS3Err picks the operator-visible error string for a failed
// HeadBucket. Well-known signatures collapse to short labels so Grafana
// alerts can split "bucket missing" (misconfiguration) from "access denied"
// (credential rotation / IAM drift) from generic network errors.
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
