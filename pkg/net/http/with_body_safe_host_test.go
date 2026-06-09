// External test package (http_test) is required here because the test imports
// pkg/datasource/hostsafety via the validator wiring. Keeping the test in
// http_test keeps the import graph clean and lets WithBody be consumed through
// the public API, exactly as production code does.
package http_test

import (
	"bytes"
	"encoding/json"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/datasource/hostsafety"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	fetcherhttp "github.com/LerianStudio/fetcher/v2/pkg/net/http"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hostInput is a minimal struct that exercises the `safe_host` validator tag
// without dragging in the full ConnectionInput surface (which has many other
// required fields). Mirrors how ConnectionInput.Host uses the tag at line 469
// of pkg/model/connection.go.
type hostInput struct {
	Host string `json:"host" validate:"required,hostname|ip,safe_host"`
}

func TestSafeHostValidatorTag(t *testing.T) {
	// The validator is initialized once per process via sync.Once. The
	// `safe_host` registered closure reads hostSafetyEnabled fresh on each
	// call, so toggling the flag between subtests works as expected.
	cases := []struct {
		name           string
		host           string
		safetyEnabled  bool
		wantStatusCode int
	}{
		{
			name:           "flag off allows loopback IP literal",
			host:           "127.0.0.1",
			safetyEnabled:  false,
			wantStatusCode: stdhttp.StatusOK,
		},
		{
			name:           "flag off allows RFC1918 IP literal",
			host:           "10.0.0.1",
			safetyEnabled:  false,
			wantStatusCode: stdhttp.StatusOK,
		},
		{
			name:           "flag on rejects loopback IP literal",
			host:           "127.0.0.1",
			safetyEnabled:  true,
			wantStatusCode: stdhttp.StatusBadRequest,
		},
		{
			name:           "flag on rejects RFC1918 IP literal",
			host:           "10.0.0.1",
			safetyEnabled:  true,
			wantStatusCode: stdhttp.StatusBadRequest,
		},
		{
			name:           "flag on rejects link-local IMDS literal",
			host:           "169.254.169.254",
			safetyEnabled:  true,
			wantStatusCode: stdhttp.StatusBadRequest,
		},
		{
			name:           "flag on accepts public IPv4 literal",
			host:           "8.8.8.8",
			safetyEnabled:  true,
			wantStatusCode: stdhttp.StatusOK,
		},
		{
			name:           "flag on accepts public hostname (DNS deferred to factory)",
			host:           "db.example.com",
			safetyEnabled:  true,
			wantStatusCode: stdhttp.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hostsafety.SetHostSafetyEnabled(tc.safetyEnabled)
			t.Cleanup(func() { hostsafety.SetHostSafetyEnabled(false) })

			app := fiber.New()
			app.Post("/test", fetcherhttp.WithBody(&hostInput{}, func(_ any, c *fiber.Ctx) error {
				return c.SendStatus(stdhttp.StatusOK)
			}))

			body := []byte(`{"host":"` + tc.host + `"}`)
			req := httptest.NewRequest(stdhttp.MethodPost, "/test", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatusCode, resp.StatusCode, "host=%q safety=%v", tc.host, tc.safetyEnabled)

			// When the guard rejects, the body must surface FET-0414 verbatim
			// so clients can branch on the code. Mensagem genérica intencional
			// — see docs/PROJECT_RULES.md § "Error Surface".
			if tc.wantStatusCode == stdhttp.StatusBadRequest && tc.safetyEnabled {
				assertSSRFErrorBody(t, resp.Body)
			}
		})
	}
}

// TestValidateStruct_SafeHostReturnsFET0414 exercises the call shape used by
// Manager HTTP handlers: c.BodyParser(&request) then ValidateStruct(&request).
// Before Fix 2 wires ValidateStruct into the handlers, this test is what
// guarantees the validator emits the correct error code when invoked directly.
// The switch case in ValidateStruct MUST map `safe_host` tag → FET-0414.
func TestValidateStruct_SafeHostReturnsFET0414(t *testing.T) {
	hostsafety.SetHostSafetyEnabled(true)
	t.Cleanup(func() { hostsafety.SetHostSafetyEnabled(false) })

	input := hostInput{Host: "127.0.0.1"}
	err := fetcherhttp.ValidateStruct(&input)
	require.Error(t, err, "ValidateStruct must reject denylisted IP literal")
	assert.Contains(t, err.Error(), "FET-0414", "ValidateStruct must surface FET-0414 for safe_host violations")
}

// hostUpdateInput mirrors the *string-pointer shape of ConnectionUpdateInput,
// exercised by the PATCH endpoint. Without `safe_host` on the tag, a tenant
// could update a previously-validated connection to point at an internal host.
type hostUpdateInput struct {
	Host *string `json:"host,omitempty" validate:"omitempty,hostname|ip,safe_host"`
}

// TestValidateStruct_SafeHostOnUpdateInput covers Fix 3: PATCH payloads must
// be guarded against the same SSRF vectors as POST. The pointer form must
// still trip the safe_host validator when a value is supplied.
func TestValidateStruct_SafeHostOnUpdateInput(t *testing.T) {
	hostsafety.SetHostSafetyEnabled(true)
	t.Cleanup(func() { hostsafety.SetHostSafetyEnabled(false) })

	loopback := "169.254.169.254"
	input := hostUpdateInput{Host: &loopback}
	err := fetcherhttp.ValidateStruct(&input)
	require.Error(t, err, "PATCH validator must reject denylisted IMDS literal")
	assert.Contains(t, err.Error(), "FET-0414")

	// Nil Host (most PATCH calls don't touch Host) must pass via omitempty.
	emptyInput := hostUpdateInput{}
	assert.NoError(t, fetcherhttp.ValidateStruct(&emptyInput),
		"omitempty must let nil Host pass through")
}

// TestValidateStruct_SafeHostOnRealConnectionUpdateInput is the production
// regression: the real model.ConnectionUpdateInput DTO must reject denylisted
// hosts in PATCH bodies. This binds the production-side tag to behavior; if
// someone strips `safe_host` from connection.go:545 in the future, this test
// fails immediately.
func TestValidateStruct_SafeHostOnRealConnectionUpdateInput(t *testing.T) {
	hostsafety.SetHostSafetyEnabled(true)
	t.Cleanup(func() { hostsafety.SetHostSafetyEnabled(false) })

	imds := "169.254.169.254"
	input := model.ConnectionUpdateInput{Host: &imds}
	err := fetcherhttp.ValidateStruct(&input)
	require.Error(t, err, "model.ConnectionUpdateInput must guard Host against SSRF in PATCH")
	assert.Contains(t, err.Error(), "FET-0414",
		"PATCH on ConnectionUpdateInput.Host must surface FET-0414")
}

// assertSSRFErrorBody confirms the response body emits the SSRF error code.
// The renderer (pkg.WithError) marshals the ValidationError as JSON; we only
// check for the substring so this test is robust to minor schema changes.
func assertSSRFErrorBody(t *testing.T, body io.Reader) {
	t.Helper()

	raw, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(raw), "FET-0414"),
		"body should surface FET-0414 error code, got: %s", string(raw))

	// Sanity-check: body must be valid JSON.
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded), "body must be valid JSON: %s", string(raw))
}
