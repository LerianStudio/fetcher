package command

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	jobRepo "github.com/LerianStudio/fetcher/v2/pkg/mongodb/job"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// spyActiveExecutionChecker records the (tenantID, connectionID) it was asked
// about and returns a programmable answer. It proves the Manager's
// update/delete conflict gate is delegated THROUGH the Engine (not decided
// inline) and that the Engine call is scoped by the request's tenant.
type spyActiveExecutionChecker struct {
	gotTenantID     string
	gotConnectionID string
	called          bool
	active          bool
	err             error
}

func (s *spyActiveExecutionChecker) HasActiveExecutions(_ context.Context, tenant engine.TenantContext, connectionID string) (bool, error) {
	s.called = true
	s.gotTenantID = tenant.TenantID
	s.gotConnectionID = connectionID

	return s.active, s.err
}

// newTestEngine builds an Engine wired with the supplied active-execution
// checker AND a ConnectionStore over the supplied connection repo. After the
// read-path deepening, Update/Delete route their persistence (FindByID/Update/
// DeleteByID) through the Engine, so the store must be present for the mock
// repo expectations to land through the Engine gate. A stub connector registry
// satisfies the Engine's only other REQUIRED port.
func newTestEngine(t *testing.T, checker engine.ActiveExecutionChecker, connRepo connRepo.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(stubConnectorRegistry{}),
		engine.WithConnectionStore(connectioncompat.NewConnectionStore(connRepo)),
		engine.WithActiveExecutionChecker(checker),
	)
	require.NoError(t, err)

	return eng
}

type stubConnectorRegistry struct{}

func (stubConnectorRegistry) Connector(string) (engine.ConnectorFactory, bool) { return nil, false }

// TestUpdateConnection_DelegatesConflictGateToEngine proves the active-job
// conflict check now flows through the Engine's ActiveExecutionChecker, scoped
// by the request tenant, rather than being decided inline in the service.
func TestUpdateConnection_DelegatesConflictGateToEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Encrypt(gomock.Any(), gomock.Any()).Return("enc", "v1", nil).AnyTimes()

	spy := &spyActiveExecutionChecker{active: false}
	eng := newTestEngine(t, spy, mockConnRepo)

	svc := NewUpdateConnection(mockCrypto, eng)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newExistingConnection(connID)
	// Capture the pre-patch config name: the gate must run on the connection's
	// CURRENT identity, before ApplyPatch can rename it.
	currentConfigName := existingConn.ConfigName
	input := newUpdateConnectionInput()

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(existingConn, nil)
	mockConnRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, conn *model.Connection) (*model.Connection, error) {
			return conn, nil
		})

	result, err := svc.Execute(ctx, connID, input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, spy.called, "Engine active-execution checker must be consulted")
	assert.Equal(t, currentConfigName, spy.gotConnectionID, "gate must be keyed by the connection config name")
	assert.Equal(t, connectioncompat.SingleTenantID, spy.gotTenantID, "gate must be scoped by the derived request tenant")
}

// TestUpdateConnection_EngineGateBlocksActiveJobs proves an Engine-reported
// conflict yields the same HTTP-mapped conflict behavior as before.
func TestUpdateConnection_EngineGateBlocksActiveJobs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Encrypt(gomock.Any(), gomock.Any()).Return("enc", "v1", nil).AnyTimes()

	spy := &spyActiveExecutionChecker{active: true}
	eng := newTestEngine(t, spy, mockConnRepo)

	svc := NewUpdateConnection(mockCrypto, eng)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newExistingConnection(connID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(existingConn, nil)

	result, err := svc.Execute(ctx, connID, newUpdateConnectionInput())
	require.Nil(t, result)
	require.Error(t, err)
	assert.True(t, spy.called)
}

// NOTE: the pre-deepening TestUpdateConnection_NilEngineSkipsGate was removed in
// this commit. Its premise — "a nil Engine skips the gate and the update
// proceeds straight to the Manager's own persistence" — no longer exists once
// Update routes its PERSISTENCE (FindByID/Update) through the Engine. There is no
// Manager-side persistence path to fall back to, so a nil Engine is not a valid
// Update construction; the assembled Manager always wires the connection-authority
// Engine (see bootstrap.connectionEngine).

// TestUpdateConnection_EngineCheckerFailureWrapsError proves a checker (job
// repo) failure surfaced through the Engine is wrapped so the existing
// errors.Is(err, repoErr) contract holds — the gate does not mask infra errors
// as conflicts.
func TestUpdateConnection_EngineCheckerFailureWrapsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Encrypt(gomock.Any(), gomock.Any()).Return("enc", "v1", nil).AnyTimes()

	repoErr := errors.New("job store down")
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), gomock.Any()).
		Return(false, repoErr)

	svc := NewUpdateConnection(mockCrypto, engineForConnRepo(t, mockConnRepo, mockJobRepo))

	ctx := testContext()
	connID := uuid.New()
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(newExistingConnection(connID), nil)

	result, err := svc.Execute(ctx, connID, newUpdateConnectionInput())
	require.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr, "infra error from the gate must be wrapped, not replaced by a conflict")
}

// TestDeleteConnection_DelegatesConflictGateToEngine proves delete also routes
// its active-job gate through the Engine under the request tenant scope.
func TestDeleteConnection_DelegatesConflictGateToEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	spy := &spyActiveExecutionChecker{active: false}
	eng := newTestEngine(t, spy, mockConnRepo)

	svc := NewDeleteConnection(eng)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newExistingConnectionForDelete(connID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(existingConn, nil)
	mockConnRepo.EXPECT().Delete(gomock.Any(), connID, gomock.Any()).Return(nil)

	err := svc.Execute(ctx, connID)
	require.NoError(t, err)

	assert.True(t, spy.called, "Engine active-execution checker must be consulted on delete")
	assert.Equal(t, existingConn.ConfigName, spy.gotConnectionID)
	assert.Equal(t, connectioncompat.SingleTenantID, spy.gotTenantID)
}
