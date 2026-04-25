package readyz

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Deployment-mode constants. These mirror the ring:dev-readyz contract and
// drive SaaS-specific TLS enforcement in Gate 4.
const (
	DeploymentModeSaaS  = "saas"
	DeploymentModeBYOC  = "byoc"
	DeploymentModeLocal = "local"
)

// Default values applied by LoadConfig when the corresponding env var is unset
// or invalid.
const (
	defaultDeploymentMode = DeploymentModeLocal
	defaultHealthPort     = 4007
	defaultDrainDelay     = 12 * time.Second
	minDrainDelay         = 1 * time.Second
)

// Config holds the readyz-specific bootstrap configuration. It is deliberately
// self-contained so that both manager and worker can call LoadConfig()
// without reaching into their own Config structs — this avoids import cycles
// and keeps pkg/bootstrap/readyz dependency-free.
type Config struct {
	// DeploymentMode is one of "saas" / "byoc" / "local", from DEPLOYMENT_MODE.
	DeploymentMode string

	// HealthPort is the port the worker exposes /health, /readyz and
	// /metrics on, from HEALTH_PORT. The manager reuses SERVER_PORT and
	// ignores this field.
	HealthPort int

	// DrainDelay is how long the handler waits after SetDraining(true) before
	// the process actually starts tearing down connections. Gate 7 wires this
	// into the SIGTERM handler.
	DrainDelay time.Duration

	// Version is resolved once, at startup, via resolveVersion(). It is
	// emitted at the top of every /readyz response.
	Version string
}

// LoadConfig reads the readyz-related environment variables and returns a
// populated Config. Invalid or missing values fall back to sane defaults
// rather than erroring — /readyz MUST remain bootable even on a
// misconfigured host, because it is the signal operators rely on to spot
// misconfiguration.
func LoadConfig() *Config {
	return &Config{
		DeploymentMode: resolveDeploymentMode(os.Getenv("DEPLOYMENT_MODE")),
		HealthPort:     resolveHealthPort(os.Getenv("HEALTH_PORT")),
		DrainDelay:     resolveDrainDelay(os.Getenv("READYZ_DRAIN_DELAY_SEC")),
		Version:        resolveVersion(),
	}
}

// resolveDeploymentMode normalises the DEPLOYMENT_MODE env var. Unknown
// values collapse to "local" so a typo in the deployment manifest does not
// accidentally activate SaaS-mode TLS enforcement (which is strictly stricter
// than BYOC/local in Gate 4).
func resolveDeploymentMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case DeploymentModeSaaS:
		return DeploymentModeSaaS
	case DeploymentModeBYOC:
		return DeploymentModeBYOC
	case DeploymentModeLocal, "":
		return DeploymentModeLocal
	default:
		return defaultDeploymentMode
	}
}

// resolveHealthPort parses HEALTH_PORT. Invalid / non-positive values fall
// back to 4007 — the standard Lerian worker health port.
func resolveHealthPort(v string) int {
	if v == "" {
		return defaultHealthPort
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || parsed <= 0 || parsed > 65535 {
		return defaultHealthPort
	}

	return parsed
}

// resolveDrainDelay parses READYZ_DRAIN_DELAY_SEC. The value is treated as
// seconds. Zero and negative values are clamped up to minDrainDelay (1s) so
// draining always takes at least one readiness-probe interval — a 0-second
// drain would cause Kubernetes to route traffic to a pod that is about to
// close its listeners. A missing env var falls back to 12 seconds, which is
// the default kube-proxy sync-plus-probe window.
func resolveDrainDelay(v string) time.Duration {
	if v == "" {
		return defaultDrainDelay
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return defaultDrainDelay
	}

	if parsed <= 0 {
		return minDrainDelay
	}

	return time.Duration(parsed) * time.Second
}

// resolveVersion returns the service version string emitted in every /readyz
// response. Precedence: OTEL_RESOURCE_SERVICE_VERSION (primary, because it's
// the source of truth for OTEL resource attributes), then VERSION (legacy
// compatibility), then "unknown".
func resolveVersion() string {
	if v := strings.TrimSpace(os.Getenv("OTEL_RESOURCE_SERVICE_VERSION")); v != "" {
		return v
	}

	if v := strings.TrimSpace(os.Getenv("VERSION")); v != "" {
		return v
	}

	return "unknown"
}
