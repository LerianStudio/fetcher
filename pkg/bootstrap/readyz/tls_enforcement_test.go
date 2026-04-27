package readyz

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ValidateSaaSTLS behavioural contract (Gate 4 of ring:dev-readyz):
//
//   - DEPLOYMENT_MODE != "saas"         -> nil (no enforcement)
//   - DEPLOYMENT_MODE == "saas":
//       - configured URL + TLS          -> nil
//       - configured URL + non-TLS      -> error wrapping ErrSaaSTLSRequired
//       - configured URL + parse error  -> wrapped parse error
//       - empty URL                     -> skip (not an error)
//   - MultiTenantEnabled == false       -> multi-tenant Redis + Tenant Manager skipped
//   - AllowInsecureHTTPTM == true       -> Tenant Manager TLS check skipped
//   - HasS3 == false                    -> S3 check skipped even when endpoint is http
//
// The ordering of checks is stable (documented in the godoc for
// ValidateSaaSTLS): mongodb, rabbitmq, redis, multi_tenant_redis, s3,
// tenant_manager. The first failing dep is returned.

// --- DEPLOYMENT_MODE gate ---------------------------------------------------

func TestValidateSaaSTLS_LocalMode_NoEnforcement(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "local",
		MongoURI:       "mongodb://host/db",
		RedisURL:       "redis://host:6379/0",
		RabbitMQURL:    "amqp://host:5672/",
	}

	require.NoError(t, ValidateSaaSTLS(cfg))
}

func TestValidateSaaSTLS_BYOCMode_NoEnforcement(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "byoc",
		MongoURI:       "mongodb://host/db",
	}

	require.NoError(t, ValidateSaaSTLS(cfg))
}

func TestValidateSaaSTLS_EmptyMode_NoEnforcement(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "",
		MongoURI:       "mongodb://host/db",
	}

	require.NoError(t, ValidateSaaSTLS(cfg))
}

func TestValidateSaaSTLS_CaseInsensitiveMode(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"saas", "SaaS", "SAAS", "Saas"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			cfg := SaaSTLSConfig{
				DeploymentMode: mode,
				MongoURI:       "mongodb://host/db",
			}

			err := ValidateSaaSTLS(cfg)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrSaaSTLSRequired)
		})
	}
}

// --- SaaS + empty URLs ------------------------------------------------------

func TestValidateSaaSTLS_SaaS_AllEmpty_NoError(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode:     "saas",
		MultiTenantEnabled: false,
		HasS3:              false,
	}

	require.NoError(t, ValidateSaaSTLS(cfg))
}

// --- SaaS + all TLS ---------------------------------------------------------

func TestValidateSaaSTLS_SaaS_AllTLS_NoError(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode:      "saas",
		MongoURI:            "mongodb+srv://user:pass@cluster0.example.com/db",
		RedisURL:            "rediss://cache.example.com:6380/0",
		MultiTenantRedisURL: "rediss://cache-mt.example.com:6380/0",
		RabbitMQURL:         "amqps://user:pass@rabbit.example.com:5671/",
		S3Endpoint:          "https://s3.amazonaws.com",
		TenantManagerURL:    "https://tenant-manager.lerian.io",
		MultiTenantEnabled:  true,
		HasS3:               true,
	}

	require.NoError(t, ValidateSaaSTLS(cfg))
}

func TestValidateSaaSTLS_SaaS_MongoTLSViaQueryParam(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		MongoURI:       "mongodb://host/db?tls=true",
	}

	require.NoError(t, ValidateSaaSTLS(cfg))
}

// --- Per-dep failure cases --------------------------------------------------

func TestValidateSaaSTLS_MongoNonTLS_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		MongoURI:       "mongodb://host/db",
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSaaSTLSRequired)
	assert.Contains(t, err.Error(), "mongodb")
}

func TestValidateSaaSTLS_RedisNonTLS_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		RedisURL:       "redis://host:6379/0",
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSaaSTLSRequired)
	assert.Contains(t, err.Error(), "redis")
}

func TestValidateSaaSTLS_RabbitMQNonTLS_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		RabbitMQURL:    "amqp://host:5672/",
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSaaSTLSRequired)
	assert.Contains(t, err.Error(), "rabbitmq")
}

func TestValidateSaaSTLS_MongoMalformedURI_ReturnsWrappedParseError(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		MongoURI:       "://no-scheme",
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	// The parse error must NOT be classified as a SaaS-TLS-required error —
	// it is a configuration-parse failure. But it MUST mention the dep.
	assert.False(t, errors.Is(err, ErrSaaSTLSRequired),
		"parse error must not be wrapped as ErrSaaSTLSRequired")
	assert.Contains(t, err.Error(), "mongodb")
	assert.Contains(t, err.Error(), "parse")
}

// --- Multi-tenant gate ------------------------------------------------------

func TestValidateSaaSTLS_MTEnabled_MultiTenantRedisNonTLS_ReturnsError(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode:      "saas",
		MultiTenantRedisURL: "redis://mt-host:6379/0",
		MultiTenantEnabled:  true,
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSaaSTLSRequired)
	// Dep name for the MT Redis must be distinguishable from the main Redis.
	assert.Contains(t, err.Error(), "multi_tenant_redis")
}

func TestValidateSaaSTLS_MTDisabled_NonTLSMultiTenantRedis_Skipped(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode:      "saas",
		MultiTenantRedisURL: "redis://mt-host:6379/0",
		TenantManagerURL:    "http://tenant-manager:8080",
		MultiTenantEnabled:  false,
	}

	require.NoError(t, ValidateSaaSTLS(cfg))
}

func TestValidateSaaSTLS_MTEnabled_TenantManagerHTTP_WithAllowInsecure_Skipped(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode:      "saas",
		TenantManagerURL:    "http://tenant-manager:8080",
		MultiTenantEnabled:  true,
		AllowInsecureHTTPTM: true,
	}

	require.NoError(t, ValidateSaaSTLS(cfg))
}

func TestValidateSaaSTLS_MTEnabled_TenantManagerHTTP_NoEscapeHatch_ReturnsError(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode:      "saas",
		TenantManagerURL:    "http://tenant-manager:8080",
		MultiTenantEnabled:  true,
		AllowInsecureHTTPTM: false,
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSaaSTLSRequired)
	assert.Contains(t, err.Error(), "tenant_manager")
}

// --- S3 gate ----------------------------------------------------------------

func TestValidateSaaSTLS_HasS3_EmptyEndpoint_NoError(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		HasS3:          true,
		S3Endpoint:     "",
	}

	require.NoError(t, ValidateSaaSTLS(cfg))
}

func TestValidateSaaSTLS_HasS3_HTTPEndpoint_ReturnsError(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		HasS3:          true,
		S3Endpoint:     "http://minio:9000",
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSaaSTLSRequired)
	assert.Contains(t, err.Error(), "s3")
}

func TestValidateSaaSTLS_HasS3False_HTTPEndpoint_Skipped(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		HasS3:          false,
		S3Endpoint:     "http://minio:9000",
	}

	require.NoError(t, ValidateSaaSTLS(cfg))
}

// --- Ordering stability -----------------------------------------------------

func TestValidateSaaSTLS_MultipleFailures_ReportsFirstInStableOrder(t *testing.T) {
	t.Parallel()

	// The documented order is: mongodb, rabbitmq, redis, multi_tenant_redis,
	// s3, tenant_manager. When multiple deps fail, mongodb must be reported
	// first.
	cfg := SaaSTLSConfig{
		DeploymentMode:      "saas",
		MongoURI:            "mongodb://host/db",
		RabbitMQURL:         "amqp://host/",
		RedisURL:            "redis://host:6379/0",
		MultiTenantRedisURL: "redis://mt:6379/0",
		S3Endpoint:          "http://minio:9000",
		TenantManagerURL:    "http://tenant-manager:8080",
		MultiTenantEnabled:  true,
		HasS3:               true,
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSaaSTLSRequired)
	assert.Contains(t, err.Error(), "mongodb")
	// The error must mention only ONE dep (the first failing one).
	assert.NotContains(t, err.Error(), "rabbitmq")
	assert.NotContains(t, err.Error(), "redis")
	assert.NotContains(t, err.Error(), "tenant_manager")
}

func TestValidateSaaSTLS_OrderingWithMongoTLSOnly_ReportsRabbitMQNext(t *testing.T) {
	t.Parallel()

	// Mongo is TLS; next in order is rabbitmq, which is non-TLS.
	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		MongoURI:       "mongodb+srv://host/db",
		RabbitMQURL:    "amqp://host:5672/",
		RedisURL:       "redis://host:6379/0",
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSaaSTLSRequired)
	assert.Contains(t, err.Error(), "rabbitmq")
	assert.NotContains(t, err.Error(), "redis")
}

// --- Error message hygiene --------------------------------------------------

func TestValidateSaaSTLS_ErrorMessage_DoesNotLeakCredentials(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		MongoURI:       "mongodb://super-secret-user:super-secret-pass@host/db",
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "super-secret-user")
	assert.NotContains(t, err.Error(), "super-secret-pass")
}

func TestValidateSaaSTLS_ErrorMessage_IncludesDeploymentModeContext(t *testing.T) {
	t.Parallel()

	cfg := SaaSTLSConfig{
		DeploymentMode: "saas",
		MongoURI:       "mongodb://host/db",
	}

	err := ValidateSaaSTLS(cfg)
	require.Error(t, err)
	// The error must be actionable: operators need to see that SaaS mode is
	// the reason. Case-insensitive check — we only care the substring is
	// present somewhere.
	assert.True(t, strings.Contains(strings.ToLower(err.Error()), "saas"),
		"error should mention SaaS context, got: %s", err.Error())
}

// --- ErrSaaSTLSRequired sentinel --------------------------------------------

func TestErrSaaSTLSRequired_IsNonNil(t *testing.T) {
	t.Parallel()
	require.NotNil(t, ErrSaaSTLSRequired)
}

// --- ComposeRedisURL --------------------------------------------------------

func TestComposeRedisURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		port string
		tls  bool
		want string
	}{
		{
			name: "empty host returns empty",
			host: "",
			port: "6379",
			want: "",
		},
		{
			name: "plain redis with port",
			host: "cache",
			port: "6379",
			tls:  false,
			want: "redis://cache:6379",
		},
		{
			name: "rediss when tls=true",
			host: "cache.example.com",
			port: "6380",
			tls:  true,
			want: "rediss://cache.example.com:6380",
		},
		{
			name: "no port still composes",
			host: "cache",
			port: "",
			tls:  false,
			want: "redis://cache",
		},
		{
			name: "ipv6 host is bracketed",
			host: "::1",
			port: "6379",
			tls:  false,
			want: "redis://[::1]:6379",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ComposeRedisURL(tc.host, tc.port, tc.tls)
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- ComposeRabbitMQURL -----------------------------------------------------

func TestComposeRabbitMQURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		port string
		user string
		pass string
		tls  bool
		want string
	}{
		{
			name: "empty host returns empty",
			host: "",
			port: "5672",
			user: "guest",
			pass: "guest",
			want: "",
		},
		{
			name: "amqp with credentials",
			host: "rabbit",
			port: "5672",
			user: "guest",
			pass: "guest",
			tls:  false,
			want: "amqp://guest:guest@rabbit:5672",
		},
		{
			name: "amqps when tls=true",
			host: "rabbit.example.com",
			port: "5671",
			user: "user",
			pass: "pass",
			tls:  true,
			want: "amqps://user:pass@rabbit.example.com:5671",
		},
		{
			name: "special characters in password are URL-encoded",
			host: "rabbit",
			port: "5672",
			user: "admin",
			pass: "p@ss:word/!",
			tls:  false,
			// url.UserPassword + net/url encodes these safely.
			want: "amqp://admin:p%40ss%3Aword%2F%21@rabbit:5672",
		},
		{
			name: "no credentials composes without @",
			host: "rabbit",
			port: "5672",
			user: "",
			pass: "",
			tls:  false,
			want: "amqp://rabbit:5672",
		},
		{
			name: "no port composes host only",
			host: "rabbit",
			port: "",
			user: "guest",
			pass: "guest",
			tls:  false,
			want: "amqp://guest:guest@rabbit",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ComposeRabbitMQURL(tc.host, tc.port, tc.user, tc.pass, tc.tls)
			assert.Equal(t, tc.want, got)
		})
	}
}
