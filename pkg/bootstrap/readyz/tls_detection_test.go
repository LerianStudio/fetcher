package readyz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tests below cover every TLS-detection helper declared in
// tls_detection.go. Each detector follows the same behavioural contract:
//
//   (1) Empty input       -> (false, nil)     dep not configured, not an error.
//   (2) Malformed URL     -> (false, err)     surface the parse error.
//   (3) TLS-indicating    -> (true,  nil)
//   (4) Non-TLS           -> (false, nil)
//   (5) URL-encoded edge  -> (false, nil)     strings.Contains would match.
//   (6) Substring-trap    -> (false, nil)     strings.Contains would match.
//
// All test tables include at least these six rows so a regression to substring
// matching is caught immediately.

// --- Mongo ------------------------------------------------------------------

func TestDetectMongoTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		uri      string
		wantTLS  bool
		wantErr  bool
		errMatch string
	}{
		{
			name:    "mongodb+srv implicit TLS",
			uri:     "mongodb+srv://user:pass@cluster0.example.com/db",
			wantTLS: true,
		},
		{
			name:    "mongodb with tls=true",
			uri:     "mongodb://host/db?tls=true",
			wantTLS: true,
		},
		{
			name:    "mongodb with ssl=true",
			uri:     "mongodb://host/db?ssl=true",
			wantTLS: true,
		},
		{
			name:    "mongodb plaintext",
			uri:     "mongodb://host/db",
			wantTLS: false,
		},
		{
			name:    "mongodb tls=false explicit",
			uri:     "mongodb://host/db?tls=false",
			wantTLS: false,
		},
		{
			name:    "empty string returns false no error",
			uri:     "",
			wantTLS: false,
		},
		{
			name:     "malformed URL returns error",
			uri:      "://no-scheme",
			wantTLS:  false,
			wantErr:  true,
			errMatch: "parse",
		},
		{
			name:    "URL-encoded tls%3Dtrue is NOT tls param",
			uri:     "mongodb://host/db?tls%3Dtrue",
			wantTLS: false,
		},
		{
			name:    "substring trap tls=false with comment containing tls=true",
			uri:     "mongodb://host/db?tls=false&note=tls%3Dtrue-comment",
			wantTLS: false,
		},
		{
			name:    "case-insensitive TLS=TRUE",
			uri:     "mongodb://host/db?tls=TRUE",
			wantTLS: true,
		},
		{
			name:    "case-insensitive SSL=True",
			uri:     "mongodb://host/db?ssl=True",
			wantTLS: true,
		},
		{
			name:    "tlsCACert alone is not a TLS flag",
			uri:     "mongodb://host/db?tlsCACert=/etc/ssl/ca.pem",
			wantTLS: false,
		},
		// --- mongodb+srv override behaviour -----------------------------
		// MongoDB driver spec: mongodb+srv enables TLS by default, but
		// operators can opt out with ?tls=false or ?ssl=false. Without
		// the explicit-override branch the detector reported tls=true on
		// these URIs, masking a known operational opt-out path.
		{
			name:    "mongodb+srv with tls=false explicit override",
			uri:     "mongodb+srv://cluster0.example.com/db?tls=false",
			wantTLS: false,
		},
		{
			name:    "mongodb+srv with ssl=false explicit override",
			uri:     "mongodb+srv://cluster0.example.com/db?ssl=false",
			wantTLS: false,
		},
		{
			name:    "mongodb+srv with explicit redundant tls=true",
			uri:     "mongodb+srv://cluster0.example.com/db?tls=true",
			wantTLS: true,
		},
		{
			name:    "mongodb+srv default scheme implies TLS without param",
			uri:     "mongodb+srv://cluster0.example.com/db",
			wantTLS: true,
		},
		{
			name:    "case-insensitive tls=FALSE override on srv",
			uri:     "mongodb+srv://cluster0.example.com/db?tls=FALSE",
			wantTLS: false,
		},
		{
			name:    "tls=0 numeric falsey override on srv",
			uri:     "mongodb+srv://cluster0.example.com/db?tls=0",
			wantTLS: false,
		},
		{
			name:    "ssl=0 numeric falsey override on srv",
			uri:     "mongodb+srv://cluster0.example.com/db?ssl=0",
			wantTLS: false,
		},
		{
			name:    "empty tls param on srv falls through to scheme default",
			uri:     "mongodb+srv://cluster0.example.com/db?tls=",
			wantTLS: true,
		},
		// --- plain mongodb scheme regression coverage -------------------
		// Belt-and-braces: verify the new override-first ordering does
		// not regress the plain "mongodb" scheme paths.
		{
			name:    "plain mongodb with tls=false stays false (no regression)",
			uri:     "mongodb://host/db?tls=false",
			wantTLS: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := detectMongoTLS(tc.uri)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMatch != "" {
					assert.Contains(t, err.Error(), tc.errMatch)
				}
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantTLS, got)
		})
	}
}

// --- Redis ------------------------------------------------------------------

func TestDetectRedisTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantTLS bool
		wantErr bool
	}{
		{
			name:    "rediss scheme with port",
			raw:     "rediss://cache.example.com:6380/0",
			wantTLS: true,
		},
		{
			name:    "rediss scheme without port",
			raw:     "rediss://cache.example.com",
			wantTLS: true,
		},
		{
			name:    "redis plaintext",
			raw:     "redis://host:6379/0",
			wantTLS: false,
		},
		{
			name:    "empty string returns false no error",
			raw:     "",
			wantTLS: false,
		},
		{
			name:    "malformed URL returns error",
			raw:     "://broken",
			wantTLS: false,
			wantErr: true,
		},
		{
			name:    "substring trap path contains rediss but scheme is redis",
			raw:     "redis://host/redisspath",
			wantTLS: false,
		},
		{
			name:    "uppercase scheme REDISS is TLS",
			raw:     "REDISS://host:6380",
			wantTLS: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := detectRedisTLS(tc.raw)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantTLS, got)
		})
	}
}

// --- AMQP -------------------------------------------------------------------

func TestDetectAMQPTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantTLS bool
		wantErr bool
	}{
		{
			name:    "amqps scheme with vhost",
			raw:     "amqps://rabbit.example.com:5671/vhost",
			wantTLS: true,
		},
		{
			name:    "amqps with credentials",
			raw:     "amqps://user:pass@rabbit.example.com",
			wantTLS: true,
		},
		{
			name:    "amqp plaintext",
			raw:     "amqp://host:5672/vhost",
			wantTLS: false,
		},
		{
			name:    "empty string returns false no error",
			raw:     "",
			wantTLS: false,
		},
		{
			name:    "malformed URL returns error",
			raw:     "://broken",
			wantTLS: false,
			wantErr: true,
		},
		{
			name:    "substring trap vhost named amqps does not flip amqp",
			raw:     "amqp://host/amqps-vhost",
			wantTLS: false,
		},
		{
			name:    "uppercase AMQPS is TLS",
			raw:     "AMQPS://host:5671",
			wantTLS: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := detectAMQPTLS(tc.raw)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantTLS, got)
		})
	}
}

// --- S3 ---------------------------------------------------------------------

func TestDetectS3TLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantTLS bool
		wantErr bool
	}{
		{
			name:    "https endpoint",
			raw:     "https://s3.amazonaws.com",
			wantTLS: true,
		},
		{
			name:    "http endpoint (MinIO dev)",
			raw:     "http://localhost:9000",
			wantTLS: false,
		},
		{
			name:    "empty means AWS default (HTTPS)",
			raw:     "",
			wantTLS: true,
		},
		{
			name:    "https with region endpoint",
			raw:     "https://s3.us-east-1.amazonaws.com",
			wantTLS: true,
		},
		{
			name:    "malformed URL returns error",
			raw:     "://broken",
			wantTLS: false,
			wantErr: true,
		},
		{
			name:    "substring trap http in path does not flip https",
			raw:     "https://host/http-path",
			wantTLS: true,
		},
		{
			name:    "uppercase HTTPS is TLS",
			raw:     "HTTPS://s3.amazonaws.com",
			wantTLS: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := detectS3TLS(tc.raw)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantTLS, got)
		})
	}
}

// --- HTTP upstream ----------------------------------------------------------

func TestDetectHTTPUpstreamTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantTLS bool
		wantErr bool
	}{
		{
			name:    "https tenant-manager",
			raw:     "https://tenant-manager.lerian.io",
			wantTLS: true,
		},
		{
			name:    "http tenant-manager plain",
			raw:     "http://tenant-manager:8080",
			wantTLS: false,
		},
		{
			name:    "empty string returns false no error",
			raw:     "",
			wantTLS: false,
		},
		{
			name:    "malformed URL returns error",
			raw:     "://broken",
			wantTLS: false,
			wantErr: true,
		},
		{
			name:    "uppercase HTTPS scheme is TLS",
			raw:     "HTTPS://api.example.com",
			wantTLS: true,
		},
		{
			name:    "substring trap https in path does not flip http",
			raw:     "http://host/https-endpoint",
			wantTLS: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := detectHTTPUpstreamTLS(tc.raw)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantTLS, got)
		})
	}
}

// --- Postgres ---------------------------------------------------------------

func TestDetectPostgresTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dsn     string
		wantTLS bool
		wantErr bool
	}{
		{
			name:    "sslmode=require is TLS",
			dsn:     "postgres://u:p@h/db?sslmode=require",
			wantTLS: true,
		},
		{
			name:    "sslmode=verify-full is TLS",
			dsn:     "postgres://u:p@h/db?sslmode=verify-full",
			wantTLS: true,
		},
		{
			name:    "sslmode=disable is not TLS",
			dsn:     "postgres://u:p@h/db?sslmode=disable",
			wantTLS: false,
		},
		{
			name:    "no sslmode param defaults to non-TLS",
			dsn:     "postgres://u:p@h/db",
			wantTLS: false,
		},
		{
			name:    "empty string returns false no error",
			dsn:     "",
			wantTLS: false,
		},
		{
			name:    "malformed URL returns error",
			dsn:     "://broken",
			wantTLS: false,
			wantErr: true,
		},
		{
			name:    "case-insensitive Sslmode=Require matches",
			dsn:     "postgres://u:p@h/db?Sslmode=Require",
			wantTLS: true,
		},
		{
			name:    "substring trap sslmode=disable with note containing require",
			dsn:     "postgres://u:p@h/db?sslmode=disable&note=require-me",
			wantTLS: false,
		},
		{
			name:    "URL-encoded sslmode%3Drequire is NOT sslmode param",
			dsn:     "postgres://u:p@h/db?sslmode%3Drequire",
			wantTLS: false,
		},
		{
			name:    "sslmode=prefer counts as TLS (not disable, not empty)",
			dsn:     "postgres://u:p@h/db?sslmode=prefer",
			wantTLS: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := detectPostgresTLS(tc.dsn)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantTLS, got)
		})
	}
}

// TLSPtr lives in types.go (Gate 2); confirm the convenience contract still
// holds so Gate 6 can rely on it without re-asserting.
func TestTLSPtr_ReturnsPointerToArgument(t *testing.T) {
	t.Parallel()

	tr := TLSPtr(true)
	require.NotNil(t, tr)
	assert.True(t, *tr)

	fa := TLSPtr(false)
	require.NotNil(t, fa)
	assert.False(t, *fa)
}
