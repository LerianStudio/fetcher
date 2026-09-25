package readyz

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/LerianStudio/lib-commons/v7/commons/buildinfo"
)

// DeploymentMode values drive SaaS-specific TLS enforcement.
const (
	DeploymentModeSaaS  = "saas"
	DeploymentModeBYOC  = "byoc"
	DeploymentModeLocal = "local"
)

const (
	defaultDeploymentMode = DeploymentModeLocal
	defaultHealthPort     = 4007
	defaultDrainDelay     = 12 * time.Second
	minDrainDelay         = 1 * time.Second
)

// Config is the readyz-specific bootstrap configuration. Self-contained so
// both manager and worker can call LoadConfig() without reaching into their
// own Config structs — this keeps pkg/bootstrap/readyz dependency-free.
type Config struct {
	// DeploymentMode is "saas", "byoc", or "local" (DEPLOYMENT_MODE).
	DeploymentMode string

	// HealthPort is the worker /health, /readyz, /metrics port (HEALTH_PORT).
	// The manager reuses SERVER_PORT and ignores this field.
	HealthPort int

	// DrainDelay is the wait between SetDraining(true) and connection
	// teardown.
	DrainDelay time.Duration

	// Identity is the compiled build identity, emitted in every /readyz
	// response.
	Identity
}

// LoadConfig falls back to sane defaults on missing or invalid input rather
// than erroring — /readyz must remain bootable on a misconfigured host
// because it is the signal operators rely on to spot misconfiguration.
func LoadConfig() *Config {
	return &Config{
		DeploymentMode: resolveDeploymentMode(os.Getenv("DEPLOYMENT_MODE")),
		HealthPort:     resolveHealthPort(os.Getenv("HEALTH_PORT")),
		DrainDelay:     resolveDrainDelay(os.Getenv("READYZ_DRAIN_DELAY_SEC")),
		Identity:       IdentityFromBuild(),
	}
}

// resolveDeploymentMode collapses unknown values to "local" so a typo in
// the deployment manifest cannot accidentally activate the stricter
// SaaS-mode TLS enforcement.
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

// resolveDrainDelay reads READYZ_DRAIN_DELAY_SEC (in seconds). Non-positive
// values are clamped to minDrainDelay so draining always covers at least
// one readiness-probe interval — a zero drain would cause Kubernetes to
// route traffic to a pod that is closing its listeners. The default
// matches the typical kube-proxy sync-plus-probe window.
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

// IdentityFromBuild reads the identity linked into the binary. Callers that
// build a Config by hand use it so every /readyz answers the same values as
// /version.
func IdentityFromBuild() Identity {
	b := buildinfo.Get()

	return Identity{Version: b.Version, Revision: b.Revision, BuildTime: b.BuildTime}
}
