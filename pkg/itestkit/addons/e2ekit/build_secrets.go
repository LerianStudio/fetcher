package e2ekit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// BuildSecret represents a BuildKit secret source for docker build.
// Use either Src (file path) or Env (environment variable), not both.
//
// Example Dockerfile usage:
//
//	RUN --mount=type=secret,id=github_token \
//	    GITHUB_TOKEN=$(cat /run/secrets/github_token) && \
//	    go mod download
type BuildSecret struct {
	// ID is the secret identifier used in the Dockerfile's --mount=type=secret,id=<ID>
	ID string

	// Src is the path to a file containing the secret value.
	// Mutually exclusive with Env.
	Src string

	// Env is the name of an environment variable containing the secret value.
	// A temporary file will be created to pass the secret to docker build.
	// Mutually exclusive with Src.
	Env string
}

// buildImageWithSecrets builds a Docker image using the docker CLI with BuildKit secrets.
// This is necessary because testcontainers-go's FromDockerfile doesn't support BuildKit secrets.
//
// Returns the image tag that was built.
func buildImageWithSecrets(ctx context.Context, cfg BuildConfig) (string, error) {
	if cfg.ContextDir == "" {
		return "", fmt.Errorf("e2ekit: BuildConfig.ContextDir is required")
	}

	dockerfile := cfg.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}

	tag := cfg.Tag
	if tag == "" {
		tag = generateImageTag()
	}

	// Build the docker build command (always use --no-cache for fresh builds)
	args := []string{"build", "--no-cache"}

	// Add dockerfile path
	args = append(args, "-f", filepath.Join(cfg.ContextDir, dockerfile))

	// Add tag
	args = append(args, "-t", tag)

	// Add build args
	for k, v := range cfg.BuildArgs {
		if v != nil {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, *v))
		}
	}

	// Process secrets and create cleanup function
	var tempFiles []string

	defer func() {
		for _, f := range tempFiles {
			_ = os.Remove(f)
		}
	}()

	for _, secret := range cfg.Secrets {
		if secret.ID == "" {
			return "", fmt.Errorf("e2ekit: BuildSecret.ID is required")
		}

		var srcPath string

		switch {
		case secret.Src != "" && secret.Env != "":
			return "", fmt.Errorf("e2ekit: BuildSecret %q has both Src and Env set; use only one", secret.ID)

		case secret.Src != "":
			// Use file path directly
			srcPath = secret.Src

		case secret.Env != "":
			// Read from environment and create temp file
			value := os.Getenv(secret.Env)
			if value == "" {
				return "", fmt.Errorf("e2ekit: environment variable %q for secret %q is empty or not set", secret.Env, secret.ID)
			}

			tmpFile, err := os.CreateTemp("", fmt.Sprintf("e2ekit-secret-%s-*", secret.ID))
			if err != nil {
				return "", fmt.Errorf("e2ekit: failed to create temp file for secret %q: %w", secret.ID, err)
			}

			if _, err := tmpFile.WriteString(value); err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())

				return "", fmt.Errorf("e2ekit: failed to write secret %q to temp file: %w", secret.ID, err)
			}

			tmpFile.Close()

			srcPath = tmpFile.Name()
			tempFiles = append(tempFiles, srcPath)

		default:
			return "", fmt.Errorf("e2ekit: BuildSecret %q has neither Src nor Env set", secret.ID)
		}

		args = append(args, "--secret", fmt.Sprintf("id=%s,src=%s", secret.ID, srcPath))
	}

	// Add context directory
	args = append(args, cfg.ContextDir)

	// Execute docker build with BuildKit enabled
	cmd := exec.CommandContext(ctx, "docker", args...)

	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("e2ekit: docker build failed: %w", err)
	}

	return tag, nil
}

// generateImageTag creates a unique image tag for builds.
func generateImageTag() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)

	return fmt.Sprintf("e2ekit-build-%s", hex.EncodeToString(b))
}

// hasSecrets returns true if the BuildConfig has any secrets configured.
func (cfg *BuildConfig) hasSecrets() bool {
	return cfg != nil && len(cfg.Secrets) > 0
}
