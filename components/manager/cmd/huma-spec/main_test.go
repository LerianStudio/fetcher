package main

import (
	"os"
	"path/filepath"
	"testing"

	in "github.com/LerianStudio/fetcher/v2/components/manager/internal/adapters/http/in"

	"github.com/stretchr/testify/require"
)

func TestWriteSpec_WritesCanonicalBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "openapi.yaml")
	require.NoError(t, writeSpec(path))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	want, err := in.GenerateCanonicalSpec()
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestCommittedSpec_IsCurrent(t *testing.T) {
	committed, err := os.ReadFile(filepath.Join("..", "..", "api", "openapi.yaml"))
	require.NoError(t, err)
	want, err := in.GenerateCanonicalSpec()
	require.NoError(t, err)

	require.Equal(t, want, committed, "run `make generate-docs`")
}
