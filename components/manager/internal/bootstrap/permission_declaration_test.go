package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	fetcher "github.com/LerianStudio/fetcher/v2"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"

	"github.com/LerianStudio/lib-auth/v5/auth/declaration"
	"github.com/LerianStudio/lib-auth/v5/auth/middleware"
	libLog "github.com/LerianStudio/lib-observability/v4/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// declarationEnv lists every variable WireFromEnv reads, canonical names and
// their deprecated aliases, so each test starts from a known-empty contract.
var declarationEnv = []string{
	"IDP_DECLARATION_ENABLED", "DECLARATION_ENABLED",
	"IDP_HOST", "PLUGIN_IDENTITY_HOST",
	"IDP_M2M_CLIENT_ID", "M2M_CLIENT_ID",
	"IDP_M2M_CLIENT_SECRET", "M2M_CLIENT_SECRET",
	"PLUGIN_AUTH_ENABLED", "PLUGIN_AUTH_HOST", "PLUGIN_AUTH_ADDRESS",
}

func clearDeclarationEnv(t *testing.T) {
	t.Helper()

	for _, name := range declarationEnv {
		t.Setenv(name, "")
	}
}

// unavailableServer answers every request with 503: the access manager health
// check and the identity PUT both fail, which is the runtime-failure class.
func unavailableServer(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	return srv
}

func enableDeclaration(t *testing.T, identityHost string) {
	t.Helper()

	t.Setenv("IDP_DECLARATION_ENABLED", "true")
	t.Setenv("IDP_HOST", identityHost)
	t.Setenv("IDP_M2M_CLIENT_ID", "fetcher-m2m")
	t.Setenv("IDP_M2M_CLIENT_SECRET", "not-a-real-secret")
}

func TestDeclarationInput_UsesTheRouteSlugTheEmbeddedManifestAndTheServiceAuthClient(t *testing.T) {
	authClient := &middleware.AuthClient{Address: "http://auth.local", Enabled: true}
	logger := libLog.NewNop()

	in := declarationInput(logger, authClient)

	assert.Equal(t, constant.ApplicationName, in.Slug)
	assert.Equal(t, "fetcher", in.Slug, "the slug must be the application name the routes authorize under")
	assert.NotEmpty(t, in.Manifest)
	assert.Equal(t, fetcher.PermissionManifest, in.Manifest)
	assert.Same(t, authClient, in.AuthClient, "the publisher must mint with the Manager's own client")
	assert.Equal(t, logger, in.Logger)
}

func TestInitPermissionDeclaration_OffByDefaultIsANoop(t *testing.T) {
	clearDeclarationEnv(t)

	// Auth disabled and no address: had the switch read anything else, the
	// injected client would fail the boot.
	stop, err := initPermissionDeclaration(context.Background(), libLog.NewNop(), &middleware.AuthClient{})

	require.NoError(t, err)
	require.NotNil(t, stop)
	stop()
}

func TestInitPermissionDeclaration_FailsClosedOnConfigurationErrors(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(t *testing.T)
		authClient *middleware.AuthClient
		wantInErr  string
	}{
		{
			name: "identity host missing",
			setup: func(t *testing.T) {
				enableDeclaration(t, "")
			},
			authClient: &middleware.AuthClient{Address: "http://auth.local", Enabled: true},
			wantInErr:  "IDP_HOST",
		},
		{
			name: "m2m client id missing",
			setup: func(t *testing.T) {
				enableDeclaration(t, "http://identity.local")
				t.Setenv("IDP_M2M_CLIENT_ID", "")
			},
			authClient: &middleware.AuthClient{Address: "http://auth.local", Enabled: true},
			wantInErr:  "IDP_M2M_CLIENT_ID",
		},
		{
			name: "m2m client secret missing",
			setup: func(t *testing.T) {
				enableDeclaration(t, "http://identity.local")
				t.Setenv("IDP_M2M_CLIENT_SECRET", "")
			},
			authClient: &middleware.AuthClient{Address: "http://auth.local", Enabled: true},
			wantInErr:  "IDP_M2M_CLIENT_SECRET",
		},
		{
			name: "auth disabled",
			setup: func(t *testing.T) {
				enableDeclaration(t, "http://identity.local")
			},
			authClient: &middleware.AuthClient{Address: "http://auth.local", Enabled: false},
			wantInErr:  "Enabled must be true",
		},
		{
			name: "auth address empty",
			setup: func(t *testing.T) {
				enableDeclaration(t, "http://identity.local")
			},
			authClient: &middleware.AuthClient{Enabled: true},
			wantInErr:  "Address is required",
		},
		{
			name: "identity host not an absolute url",
			setup: func(t *testing.T) {
				enableDeclaration(t, "identity.local")
			},
			authClient: &middleware.AuthClient{Address: "http://auth.local", Enabled: true},
			wantInErr:  "IdentityAddr",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearDeclarationEnv(t)
			tc.setup(t)

			stop, err := initPermissionDeclaration(context.Background(), libLog.NewNop(), tc.authClient)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantInErr)
			assert.Contains(t, err.Error(), "wire access manager permission declaration")
			require.NotNil(t, stop, "the stop hook must be safe to call on every error path")
			stop()
		})
	}
}

func TestInitPermissionDeclaration_FailsOpenWhenIdentityIsUnavailable(t *testing.T) {
	clearDeclarationEnv(t)

	srv := unavailableServer(t)
	enableDeclaration(t, srv.URL)

	authClient := middleware.NewAuthClient(srv.URL, true, libLog.NewNop())

	// Reaching Start also proves the embedded manifest parses, validates and
	// names this slug: New rejects any of those before a publish is attempted.
	done := make(chan struct{})

	var (
		stop func()
		err  error
	)

	go func() {
		defer close(done)

		stop, err = initPermissionDeclaration(context.Background(), libLog.NewNop(), authClient)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the boot must not block on identity reachability")
	}

	require.NoError(t, err)
	require.NotNil(t, stop)
	stop()
}

func TestInitPermissionDeclaration_RejectsASlugTheManifestDoesNotName(t *testing.T) {
	clearDeclarationEnv(t)
	enableDeclaration(t, "http://identity.local")

	in := declarationInput(libLog.NewNop(), &middleware.AuthClient{Address: "http://auth.local", Enabled: true})
	in.Slug = "not-fetcher"

	stop, err := declaration.WireFromEnv(context.Background(), in)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must equal manifest.service")
	require.NotNil(t, stop)
}

func TestInitPermissionDeclaration_NormalizesANilStopAndWrapsErrors(t *testing.T) {
	original := wireDeclarationFromEnv
	t.Cleanup(func() { wireDeclarationFromEnv = original })

	var got declaration.WireInput

	wireDeclarationFromEnv = func(_ context.Context, in declaration.WireInput) (func(), error) {
		got = in

		return nil, nil
	}

	authClient := &middleware.AuthClient{}

	stop, err := initPermissionDeclaration(context.Background(), libLog.NewNop(), authClient)
	require.NoError(t, err)
	require.NotNil(t, stop)
	stop()
	assert.Same(t, authClient, got.AuthClient)

	sentinel := errors.New("boom")
	wireDeclarationFromEnv = func(context.Context, declaration.WireInput) (func(), error) {
		return nil, sentinel
	}

	stop, err = initPermissionDeclaration(context.Background(), libLog.NewNop(), authClient)
	require.ErrorIs(t, err, sentinel)
	require.NotNil(t, stop)
	stop()
}
