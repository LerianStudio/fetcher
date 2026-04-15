package e2ekit

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestBuildImageWithSecretsValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     BuildConfig
		prepare func(t *testing.T)
		wantErr string
	}{
		{
			name:    "missing context dir",
			cfg:     BuildConfig{},
			wantErr: "BuildConfig.ContextDir is required",
		},
		{
			name: "missing secret id",
			cfg: BuildConfig{
				ContextDir: t.TempDir(),
				Secrets:    []BuildSecret{{Env: "GITHUB_TOKEN"}},
			},
			prepare: func(t *testing.T) {
				t.Setenv("GITHUB_TOKEN", "token")
			},
			wantErr: "BuildSecret.ID is required",
		},
		{
			name: "secret with both src and env",
			cfg: BuildConfig{
				ContextDir: t.TempDir(),
				Secrets:    []BuildSecret{{ID: "github", Src: "/tmp/token", Env: "GITHUB_TOKEN"}},
			},
			wantErr: "has both Src and Env set",
		},
		{
			name: "secret missing both src and env",
			cfg: BuildConfig{
				ContextDir: t.TempDir(),
				Secrets:    []BuildSecret{{ID: "github"}},
			},
			wantErr: "has neither Src nor Env set",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepare != nil {
				tt.prepare(t)
			}

			_, err := buildImageWithSecrets(context.Background(), tt.cfg)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCreateSecretTempFile_AllowsMissingEnv(t *testing.T) {
	t.Parallel()

	path, err := createSecretTempFile(BuildSecret{ID: "github", Env: "GITHUB_TOKEN"})
	if err != nil {
		t.Fatalf("expected missing env to produce empty secret file, got %v", err)
	}

	t.Cleanup(func() {
		_ = os.Remove(path)
	})

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected to read secret temp file, got %v", err)
	}

	if len(contents) != 0 {
		t.Fatalf("expected empty secret file for missing env, got %q", string(contents))
	}
}

func TestBuildConfigHelpers(t *testing.T) {
	t.Parallel()

	withSecrets := (&BuildConfig{Secrets: []BuildSecret{{ID: "token", Env: "GITHUB_TOKEN"}}}).hasSecrets()
	withoutSecrets := (&BuildConfig{}).hasSecrets()
	withNilConfig := (*BuildConfig)(nil).hasSecrets()

	if !withSecrets {
		t.Fatalf("expected config with secrets to report true")
	}

	if withoutSecrets || withNilConfig {
		t.Fatalf("expected empty or nil config to report false")
	}

	tagA := generateImageTag()
	tagB := generateImageTag()

	if !strings.HasPrefix(tagA, "e2ekit-build-") {
		t.Fatalf("expected generated tag prefix, got %q", tagA)
	}

	if len(tagA) != len("e2ekit-build-")+16 {
		t.Fatalf("expected 8 random bytes encoded as hex, got %q", tagA)
	}

	if tagA == tagB {
		t.Fatalf("expected generated tags to be unique, got duplicate %q", tagA)
	}
}
