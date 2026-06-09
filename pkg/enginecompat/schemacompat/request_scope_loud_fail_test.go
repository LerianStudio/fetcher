// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package schemacompat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelDatasource "github.com/LerianStudio/fetcher/pkg/model/datasource"

	"github.com/stretchr/testify/require"
)

// CHARACTERIZATION — Seam 3: a MISSING request-scoped connection seed must fail
// LOUDLY.
//
// The schema engine's request-scoped ConnectionStore
// (schemacompat.NewConnectionStore) resolves connections ONLY from the context
// seed planted by WithResolvedConnections. When the seed is absent its
// FindConnection reports found=false. These tests pin the END-TO-END behavior
// through the engine: the engine maps that absence to a clear
// CategoryNotFound *EngineError ("connection not found") — it does NOT silently
// return an empty schema snapshot, which would be a misleading success.
//
// If a future refactor made the missing seed return an empty snapshot with a nil
// error, these tests would fail — which is the point: a silent not-found is a
// latent bug, and this seam currently fails loudly.

// loudFailRegistry is a wildcard ConnectorRegistry mirroring the production
// schemaConnectorRegistry: it resolves the single schema ConnectorFactory for any
// datasource type. It is here only to satisfy engine.New; these tests never reach
// a Build because resolution fails BEFORE connector construction when the seed is
// absent.
type loudFailRegistry struct {
	factory engine.ConnectorFactory
}

func (r loudFailRegistry) Connector(string) (engine.ConnectorFactory, bool) {
	return r.factory, true
}

// newSchemaEngineUnderTest builds an engine wired exactly like the Manager's
// schemaEngine: a wildcard schemacompat ConnectorFactory registry and the
// request-scoped schemacompat ConnectionStore. No SchemaCache is wired, so a
// resolution that PASSES would proceed to live discovery — but the point of these
// tests is that resolution FAILS first, before any connector or cache is touched.
func newSchemaEngineUnderTest(t *testing.T) *engine.Engine {
	t.Helper()

	// A datasource factory that fails loudly if ever invoked: a correct loud-fail
	// on the missing seed must short-circuit BEFORE any connect.
	dsFactory := func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		t.Fatal("datasource factory must not be reached when the request seed is absent")
		return nil, errors.New("unreachable")
	}

	eng, err := engine.New(
		engine.WithConnectorRegistry(loudFailRegistry{
			factory: schemacompat.NewConnectorFactory(dsFactory, nil),
		}),
		engine.WithConnectionStore(schemacompat.NewConnectionStore()),
	)
	require.NoError(t, err)

	return eng
}

func requireLoudNotFound(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err, "a missing request seed must fail loudly, not return a silent empty result")

	var engErr *engine.EngineError
	require.True(t, errors.As(err, &engErr), "expected *engine.EngineError, got %T: %v", err, err)
	require.Equalf(t, engine.CategoryNotFound, engErr.Category,
		"expected a loud CategoryNotFound for the absent seed, got %q (%v)", engErr.Category, err)
}

func TestDiscoverSchema_MissingSeed_FailsLoudly(t *testing.T) {
	t.Parallel()

	eng := newSchemaEngineUnderTest(t)

	// NO WithResolvedConnections on the context: the seed is absent.
	snapshot, err := eng.DiscoverSchema(context.Background(), mustTenant(t), "db1")

	requireLoudNotFound(t, err)

	// And crucially: the returned snapshot is the zero value, NOT a misleading
	// partially-populated success.
	require.Empty(t, snapshot.Tables, "a loud-fail must not also return a non-empty snapshot")
}

func TestDiscoverSchemaFresh_MissingSeed_FailsLoudly(t *testing.T) {
	t.Parallel()

	eng := newSchemaEngineUnderTest(t)

	_, err := eng.DiscoverSchemaFresh(context.Background(), mustTenant(t), "db1")

	requireLoudNotFound(t, err)
}

// TestDiscoverSchema_WrongConfigNameInSeed_FailsLoudly proves the loud-fail is
// keyed on the EXACT config name: a seed for "db1" does not satisfy a discovery
// for "db2". A mismatched seed is just as much an absence as no seed at all, and
// must fail loudly rather than silently returning an empty result.
func TestDiscoverSchema_WrongConfigNameInSeed_FailsLoudly(t *testing.T) {
	t.Parallel()

	eng := newSchemaEngineUnderTest(t)

	conn := &model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL, Host: "h"}
	ctx := schemacompat.WithResolvedConnections(context.Background(), []*model.Connection{conn})

	_, err := eng.DiscoverSchema(ctx, mustTenant(t), "db2")

	requireLoudNotFound(t, err)
}
