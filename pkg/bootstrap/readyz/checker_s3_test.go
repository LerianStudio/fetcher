package readyz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeS3 struct {
	out      *s3.HeadBucketOutput
	err      error
	delay    time.Duration
	calls    int
	lastName string
}

func (f *fakeS3) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	f.calls++

	if params != nil && params.Bucket != nil {
		f.lastName = *params.Bucket
	}

	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return f.out, f.err
}

func TestS3BucketChecker_Up(t *testing.T) {
	fake := &fakeS3{}
	c := NewS3BucketChecker(fake, "my-bucket", "https://s3.amazonaws.com")

	assert.Equal(t, "s3", c.Name())

	res := c.Check(context.Background())
	assert.Equal(t, StatusUp, res.Status)
	assert.Empty(t, res.Error)
	if assert.NotNil(t, res.TLS) {
		assert.True(t, *res.TLS)
	}

	assert.Equal(t, "my-bucket", fake.lastName)
	assert.Equal(t, 1, fake.calls)
}

func TestS3BucketChecker_DefaultEndpointIsHTTPS(t *testing.T) {
	// Empty endpoint means AWS default, which is HTTPS.
	fake := &fakeS3{}
	c := NewS3BucketChecker(fake, "b", "")

	res := c.Check(context.Background())
	assert.Equal(t, StatusUp, res.Status)
	if assert.NotNil(t, res.TLS) {
		assert.True(t, *res.TLS)
	}
}

func TestS3BucketChecker_NotFound(t *testing.T) {
	fake := &fakeS3{err: errors.New("api error NotFound: Not Found")}
	c := NewS3BucketChecker(fake, "my-bucket", "http://minio:9000")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "bucket not found", res.Error)
	if assert.NotNil(t, res.TLS) {
		assert.False(t, *res.TLS, "http:// endpoint implies TLS=false")
	}
}

func TestS3BucketChecker_AccessDenied(t *testing.T) {
	fake := &fakeS3{err: errors.New("AccessDenied: User is not authorized to perform s3:ListBucket")}
	c := NewS3BucketChecker(fake, "my-bucket", "")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "access denied", res.Error)
}

func TestS3BucketChecker_NoSuchBucket(t *testing.T) {
	fake := &fakeS3{err: errors.New("NoSuchBucket: the specified bucket does not exist")}
	c := NewS3BucketChecker(fake, "ghost", "")

	res := c.Check(context.Background())
	assert.Equal(t, "bucket not found", res.Error)
}

func TestS3BucketChecker_Timeout(t *testing.T) {
	fake := &fakeS3{delay: 50 * time.Millisecond, err: context.DeadlineExceeded}
	c := NewS3BucketChecker(fake, "b", "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	res := c.Check(ctx)
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "timeout", res.Error)
}

func TestS3BucketChecker_NilClient(t *testing.T) {
	c := NewS3BucketChecker(nil, "b", "")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "s3 client not initialized", res.Error)
}

func TestS3BucketChecker_EmptyBucket(t *testing.T) {
	fake := &fakeS3{}
	c := NewS3BucketChecker(fake, "", "")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "s3 bucket not configured", res.Error)
	assert.Zero(t, fake.calls)
}

func TestS3BucketChecker_SanitizesGenericError(t *testing.T) {
	fake := &fakeS3{err: errors.New("signing failed for https://user:hunter2@endpoint:9000")}
	c := NewS3BucketChecker(fake, "b", "http://minio:9000")

	res := c.Check(context.Background())
	require.Equal(t, StatusDown, res.Status)
	assert.NotContains(t, res.Error, "hunter2")
}
