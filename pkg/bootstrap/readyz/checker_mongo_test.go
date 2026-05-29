package readyz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMongoPinger is a hand-rolled fake to keep tests hermetic. We don't
// use a mock framework because the surface is one method and the behaviour
// under test is strictly the checker's classification logic.
type fakeMongoPinger struct {
	err    error
	delay  time.Duration
	called int
}

func (f *fakeMongoPinger) Ping(ctx context.Context) error {
	f.called++

	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return f.err
}

func TestMongoClientChecker_Up(t *testing.T) {
	fake := &fakeMongoPinger{err: nil}
	c := NewMongoClientChecker(fake, "mongodb://host:27017/db?tls=true")

	assert.Equal(t, "mongodb", c.Name())

	res := c.Check(context.Background())
	assert.Equal(t, StatusUp, res.Status)
	assert.Empty(t, res.Error)
	if assert.NotNil(t, res.TLS) {
		assert.True(t, *res.TLS)
	}

	assert.Equal(t, 1, fake.called)
}

func TestMongoClientChecker_Down_GenericError(t *testing.T) {
	fake := &fakeMongoPinger{err: errors.New("dial tcp 10.0.0.1:27017: connection refused")}
	c := NewMongoClientChecker(fake, "mongodb://host:27017/db")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Contains(t, res.Error, "connection refused")
	if assert.NotNil(t, res.TLS) {
		assert.False(t, *res.TLS)
	}
}

func TestMongoClientChecker_Down_Timeout(t *testing.T) {
	fake := &fakeMongoPinger{delay: 50 * time.Millisecond, err: context.DeadlineExceeded}
	c := NewMongoClientChecker(fake, "mongodb+srv://host/db")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	res := c.Check(ctx)
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "timeout", res.Error)
	// mongodb+srv is implicitly TLS.
	if assert.NotNil(t, res.TLS) {
		assert.True(t, *res.TLS)
	}
}

func TestMongoClientChecker_Down_Canceled(t *testing.T) {
	fake := &fakeMongoPinger{err: context.Canceled}
	c := NewMongoClientChecker(fake, "mongodb://host:27017")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := c.Check(ctx)
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "canceled", res.Error)
}

func TestMongoClientChecker_NilClient(t *testing.T) {
	c := NewMongoClientChecker(nil, "mongodb://host:27017")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "mongo client not initialized", res.Error)
}

func TestMongoClientChecker_SanitizesCredentialsInError(t *testing.T) {
	fake := &fakeMongoPinger{err: errors.New("auth failed for mongodb://admin:hunter2@mongo:27017")}
	c := NewMongoClientChecker(fake, "mongodb://host:27017")

	res := c.Check(context.Background())
	require.Equal(t, StatusDown, res.Status)
	assert.NotContains(t, res.Error, "hunter2", "password must be redacted")
	assert.Contains(t, res.Error, "***@mongo:27017")
}

func TestMongoClientChecker_TLSFromURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{"plain mongodb", "mongodb://host:27017", false},
		{"tls query", "mongodb://host:27017/?tls=true", true},
		{"ssl query", "mongodb://host:27017/?ssl=true", true},
		{"srv implicit", "mongodb+srv://host/db", true},
		{"empty uri", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeMongoPinger{err: nil}
			c := NewMongoClientChecker(fake, tc.uri)

			res := c.Check(context.Background())
			if assert.NotNil(t, res.TLS) {
				assert.Equal(t, tc.want, *res.TLS, tc.uri)
			}
		})
	}
}

func TestMongoClientChecker_LatencyMsPopulatedOnUp(t *testing.T) {
	fake := &fakeMongoPinger{delay: 2 * time.Millisecond}
	c := NewMongoClientChecker(fake, "mongodb://host:27017")

	res := c.Check(context.Background())
	assert.Equal(t, StatusUp, res.Status)
	assert.GreaterOrEqual(t, res.LatencyMs, int64(0))
}
