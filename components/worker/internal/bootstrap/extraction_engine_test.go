package bootstrap

import (
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/datasource"
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestExtractionDeps builds the minimal dsFactory + cryptor pair the engine
// wiring needs. The factory/cryptor are never exercised at construction time —
// newExtractionEngine only validates ports — so a real (unused) factory and a
// crypto service over a zero key are sufficient.
func newTestExtractionDeps(t *testing.T) (datasource.DataSourceFactory, crypto.Cryptor) {
	t.Helper()

	dsFactory := datasource.NewDataSourceFromConnectionWithLogger(testBootstrapLogger())

	cryptor, err := crypto.NewAESGCMService(make([]byte, 32), "1")
	require.NoError(t, err)

	return dsFactory, cryptor
}

func TestNewExtractionEngine_DefaultLimitsWhenUnset(t *testing.T) {
	t.Parallel()

	dsFactory, cryptor := newTestExtractionDeps(t)

	// Zero (env unset) and negative both leave the engine at DefaultLimits.
	for _, maxResultBytes := range []int64{0, -1} {
		eng, err := newExtractionEngine(dsFactory, cryptor, maxResultBytes)
		require.NoError(t, err)
		require.NotNil(t, eng)

		assert.Equal(t, engine.DefaultMaxResultBytes, eng.Limits().MaxResultBytes,
			"non-positive override must leave the default result-size ceiling in force")
	}
}

func TestNewExtractionEngine_PositiveOverridesOnlyMaxResultBytes(t *testing.T) {
	t.Parallel()

	dsFactory, cryptor := newTestExtractionDeps(t)

	const override int64 = 64 * 1024 * 1024 // 64 MiB, below the 256 MiB default

	eng, err := newExtractionEngine(dsFactory, cryptor, override)
	require.NoError(t, err)
	require.NotNil(t, eng)

	got := eng.Limits()

	assert.Equal(t, override, got.MaxResultBytes,
		"positive override must set MaxResultBytes")

	// Every other limit must remain at its default — proving the override is
	// composed from DefaultLimits and WithLimits did not zero the struct.
	assert.Equal(t, engine.DefaultMaxDatasources, got.MaxDatasources)
	assert.Equal(t, engine.DefaultMaxTablesPerDatasource, got.MaxTablesPerDatasource)
	assert.Equal(t, engine.DefaultMaxFieldsPerTable, got.MaxFieldsPerTable)
	assert.Equal(t, engine.DefaultMaxConcurrency, got.MaxConcurrency)
	assert.Equal(t, engine.DefaultTimeout, got.Timeout)
}

func TestNewExtractionEngine_PositiveOverrideAboveDefaultIsAccepted(t *testing.T) {
	t.Parallel()

	dsFactory, cryptor := newTestExtractionDeps(t)

	// The engine default is the per-request ceiling, but the engine-construction
	// limit itself can be raised above the historical default by an operator.
	const override = engine.DefaultMaxResultBytes * 2

	eng, err := newExtractionEngine(dsFactory, cryptor, override)
	require.NoError(t, err)
	require.NotNil(t, eng)

	assert.Equal(t, override, eng.Limits().MaxResultBytes)
}

func TestConfig_EngineMaxResultBytes(t *testing.T) {
	t.Setenv("ENGINE_MAX_RESULT_BYTES", "134217728") // 128 MiB

	cfg := &Config{}
	require.NoError(t, libCommons.SetConfigFromEnvVars(cfg))

	assert.Equal(t, int64(134217728), cfg.EngineMaxResultBytes)
}

func TestConfig_EngineMaxResultBytes_UnsetIsZero(t *testing.T) {
	t.Setenv("ENGINE_MAX_RESULT_BYTES", "")

	cfg := &Config{}
	require.NoError(t, libCommons.SetConfigFromEnvVars(cfg))

	assert.Zero(t, cfg.EngineMaxResultBytes,
		"unset env must leave the field zero so the engine default stays in force")
}
