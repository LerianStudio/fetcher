// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package connectioncompat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	connPort "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/v2/pkg/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// engineForAccess builds a minimal Engine wired to the supplied connection repo
// through the connectioncompat ConnectionStore — the same assembly the Manager
// read/mutation services use, so the shared AuthorizeAccess / FindByID helpers
// are exercised end-to-end through the Engine gate.
func engineForAccess(t *testing.T, repo connPort.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(noopConnectorRegistry{}),
		engine.WithConnectionStore(connectioncompat.NewConnectionStore(repo)),
	)
	require.NoError(t, err)

	return eng
}

type noopConnectorRegistry struct{}

func (noopConnectorRegistry) Connector(string) (engine.ConnectorFactory, bool) { return nil, false }

// TestAuthorizeAccess_NilEngineIsNoOp proves the shared helper treats a nil
// engine (test-only construction) as a no-op rather than panicking — the single
// defensive guard both command and query packages now rely on.
func TestAuthorizeAccess_NilEngineIsNoOp(t *testing.T) {
	t.Parallel()

	require.NoError(t, connectioncompat.AuthorizeAccess(testutil.TestContext(), nil))
}

// TestAuthorizeAccess_RealEngineRoutesScope proves the shared helper drives the
// Engine's tenant-scope authority rule when a real engine is supplied (it does
// not touch the repo: authorization is a scope decision, not a read).
func TestAuthorizeAccess_RealEngineRoutesScope(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := connPort.NewMockRepository(ctrl) // no expectations: authorize must not read

	require.NoError(t, connectioncompat.AuthorizeAccess(testutil.TestContext(), engineForAccess(t, repo)))
}

// TestFindByID_FoundReturnsRichRecord proves the shared FindByID helper routes
// the read through the Engine ID-addressed op and unpacks the rich record from
// the opaque host payload.
func TestFindByID_FoundReturnsRichRecord(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connID := uuid.New()
	repo := connPort.NewMockRepository(ctrl)
	repo.EXPECT().FindByID(gomock.Any(), connID).Return(&model.Connection{ID: connID, ConfigName: "pg-main"}, nil)

	got, err := connectioncompat.FindByID(testutil.TestContext(), engineForAccess(t, repo), connID.String())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "pg-main", got.ConfigName)
	assert.Equal(t, connID, got.ID)
}

// TestFindByID_NotFoundMapsToNilNil locks the not-found-mapping contract in ONE
// place: the Engine's CategoryNotFound (repo.FindByID -> nil) must surface as
// (nil, nil) so every caller maps it to the Manager's existing not-found business
// error byte-identically.
func TestFindByID_NotFoundMapsToNilNil(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connID := uuid.New()
	repo := connPort.NewMockRepository(ctrl)
	repo.EXPECT().FindByID(gomock.Any(), connID).Return(nil, nil)

	got, err := connectioncompat.FindByID(testutil.TestContext(), engineForAccess(t, repo), connID.String())
	require.NoError(t, err)
	assert.Nil(t, got, "not-found must map to (nil, nil), not an error")
}

// TestFindByID_RepoErrorPropagates proves a genuine repo error (NOT a not-found)
// is surfaced rather than swallowed by the not-found mapping.
func TestFindByID_RepoErrorPropagates(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connID := uuid.New()
	repo := connPort.NewMockRepository(ctrl)
	repo.EXPECT().FindByID(gomock.Any(), connID).Return(nil, errors.New("db down"))

	_, err := connectioncompat.FindByID(context.Background(), engineForAccess(t, repo), connID.String())
	require.Error(t, err)
}
