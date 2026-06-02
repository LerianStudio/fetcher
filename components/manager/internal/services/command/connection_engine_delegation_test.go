package command

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/model"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

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
// checker. A stub connector registry satisfies the Engine's only REQUIRED port;
// connection persistence stays in the Manager, so no ConnectionStore is wired.
func newTestEngine(t *testing.T, checker engine.ActiveExecutionChecker) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(stubConnectorRegistry{}),
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
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Encrypt(gomock.Any(), gomock.Any()).Return("enc", "v1", nil).AnyTimes()

	spy := &spyActiveExecutionChecker{active: false}
	eng := newTestEngine(t, spy)

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto, eng)

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
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Encrypt(gomock.Any(), gomock.Any()).Return("enc", "v1", nil).AnyTimes()

	spy := &spyActiveExecutionChecker{active: true}
	eng := newTestEngine(t, spy)

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto, eng)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newExistingConnection(connID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(existingConn, nil)

	result, err := svc.Execute(ctx, connID, newUpdateConnectionInput())
	require.Nil(t, result)
	require.Error(t, err)
	assert.True(t, spy.called)
}

// TestUpdateConnection_NilEngineSkipsGate proves a service constructed without
// an Engine (defensive guard) skips the gate rather than panicking: the update
// proceeds straight to persistence.
func TestUpdateConnection_NilEngineSkipsGate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Encrypt(gomock.Any(), gomock.Any()).Return("enc", "v1", nil).AnyTimes()

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto, nil)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newExistingConnection(connID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(existingConn, nil)
	mockConnRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, conn *model.Connection) (*model.Connection, error) {
			return conn, nil
		})

	result, err := svc.Execute(ctx, connID, newUpdateConnectionInput())
	require.NoError(t, err)
	require.NotNil(t, result)
}

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

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto, engineForJobRepo(t, mockJobRepo))

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
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	spy := &spyActiveExecutionChecker{active: false}
	eng := newTestEngine(t, spy)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo, eng)

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
