package streaming

import (
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireRosterSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured string
		wantErr    bool
	}{
		{name: "roster name", configured: constant.ApplicationName},
		{name: "empty", configured: "", wantErr: true},
		// Every case below is grammar-legal under lib-streaming's own source
		// validation, and every one of them is still wrong: the provisioned
		// topic, DLQ and ACL are named for the roster name alone.
		{name: "module name instead of application name", configured: "fetcher-worker", wantErr: true},
		{name: "suffixed", configured: "fetcherx", wantErr: true},
		{name: "underscored variant", configured: "fetcher_v2", wantErr: true},
		// Rejected by lib-streaming too, but this gate must catch it first so a
		// disabled config cannot hide it: lib-streaming skips source validation
		// entirely when streaming is off.
		{name: "legacy pre-v3 uri source", configured: "//lerian.fetcher/worker", wantErr: true},
		{name: "untrimmed", configured: " fetcher ", wantErr: true},
		{name: "wrong case", configured: "Fetcher", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := RequireRosterSource(tt.configured, constant.ApplicationName)
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrSourceNotRoster)
				assert.Contains(t, err.Error(), "STREAMING_CLOUDEVENTS_SOURCE")

				return
			}

			require.NoError(t, err)
		})
	}
}

// TestRosterNameIsTheApplicationName pins the value the gate enforces. The
// roster name is the literal topic segment and ACL scope the platform
// provisions, so changing it is a provisioning event, never a rename.
func TestRosterNameIsTheApplicationName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "fetcher", constant.ApplicationName)
}
