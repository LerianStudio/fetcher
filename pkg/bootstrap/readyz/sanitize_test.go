package readyz

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize_RedactsUserinfo(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple user:pass@host",
			in:   "error: user:secret@host:27017 unreachable",
			want: "error: ***@host:27017 unreachable",
		},
		{
			name: "full mongo URI",
			in:   "dial mongodb://admin:hunter2@mongo.prod:27017/db",
			want: "dial mongodb://***@mongo.prod:27017/db",
		},
		{
			name: "amqp URI",
			in:   "failed: amqp://rmq:topsecret@rabbit:5672/%2F",
			want: "failed: amqp://***@rabbit:5672/%2F",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, sanitize(tc.in))
		})
	}
}

func TestSanitize_RedactsPasswordKV(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "password=...",
			in:   "postgres: sslmode=require password=secret host=db",
			want: "postgres: sslmode=require password=*** host=db",
		},
		{
			name: "PASSWORD=... case-insensitive",
			in:   "PASSWORD=abc123",
			want: "PASSWORD=***",
		},
		{
			name: "pwd=...",
			in:   "conn string: user=bob pwd=letmein",
			want: "conn string: user=bob pwd=***",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, sanitize(tc.in))
		})
	}
}

func TestSanitize_NonMatchingErrorsUnchanged(t *testing.T) {
	cases := []string{
		"dial tcp: connection refused",
		"context deadline exceeded",
		"bucket not found",
	}
	for _, in := range cases {
		assert.Equal(t, in, sanitize(in))
	}
}

func TestSanitize_EmptyInput(t *testing.T) {
	assert.Equal(t, "", sanitize(""))
}

func TestSanitize_TruncatesLongStrings(t *testing.T) {
	in := strings.Repeat("x", 1024)
	out := sanitize(in)
	assert.LessOrEqual(t, len(out), 512)
}

func TestSanitize_PreservesPortColons(t *testing.T) {
	// A "host:port" colon at the end of an error must not be mistaken for
	// userinfo — there's no "@" terminator.
	in := "connect: mongo.prod:27017 timeout"
	assert.Equal(t, in, sanitize(in))
}

func TestTLSOrFalse(t *testing.T) {
	// The helper collapses (bool, err) by discarding the error. The error
	// signalling is surfaced elsewhere (Gate 4 fails bootstrap on SaaS
	// parse errors); for the informational TLS field we follow the bool.
	assert.True(t, tlsOrFalse(true, nil))
	assert.False(t, tlsOrFalse(false, nil))
	assert.True(t, tlsOrFalse(true, assert.AnError))
	assert.False(t, tlsOrFalse(false, assert.AnError))
}
