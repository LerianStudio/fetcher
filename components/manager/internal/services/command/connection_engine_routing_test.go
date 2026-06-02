package command

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestCreateConnection_RoutesPersistenceThroughEngine proves Create's
// persistence flows THROUGH the Engine: the Engine's uniqueness pre-check lands
// on repo.FindByName and the Engine's store write lands on repo.Create — the
// pre-delegation call shape — while the rich record (ProductName / SSL / UUID /
// metadata) round-trips losslessly through the opaque host payload.
func TestCreateConnection_RoutesPersistenceThroughEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Encrypt(gomock.Any(), gomock.Any()).Return("enc", "v1", nil)

	var persisted *model.Connection

	gomock.InOrder(
		mockConnRepo.EXPECT().FindByName(gomock.Any(), "routed-conn").Return(nil, nil),
		mockConnRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, c *model.Connection) (*model.Connection, error) {
				persisted = c
				return c, nil
			}),
	)

	svc := NewCreateConnection(mockCrypto, engineForConnRepo(t, mockConnRepo, nil))

	input := model.ConnectionInput{
		ConfigName:   "routed-conn",
		Type:         "POSTGRESQL",
		Host:         "localhost",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "secret",
	}

	result, err := svc.Execute(testContext(), input, "plugin_crm")
	require.NoError(t, err)
	require.NotNil(t, result)

	// The returned connection is the rich record unpacked from the Engine
	// descriptor's opaque payload — ProductName preserved (a host attribute the
	// Engine never scoped on).
	assert.Equal(t, "plugin_crm", result.ProductName)
	assert.Equal(t, "routed-conn", result.ConfigName)
	require.NotNil(t, persisted)
	assert.Equal(t, result.ID, persisted.ID, "UUID identity round-trips through the Engine seam")
}

// TestCreateConnection_EngineConflictMapsTo409 proves the Engine's
// (tenantID, configName) uniqueness conflict surfaces as the Manager's existing
// ErrEntityConflict business error (HTTP 409), preserving the duplicate-create
// contract while the uniqueness RULE now lives in the Engine.
func TestCreateConnection_EngineConflictMapsTo409(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Encrypt(gomock.Any(), gomock.Any()).Return("enc", "v1", nil)

	// Engine pre-check finds an existing connection -> conflict; Create is never
	// reached.
	mockConnRepo.EXPECT().FindByName(gomock.Any(), "dup-conn").Return(&model.Connection{ConfigName: "dup-conn"}, nil)

	svc := NewCreateConnection(mockCrypto, engineForConnRepo(t, mockConnRepo, nil))

	input := model.ConnectionInput{
		ConfigName:   "dup-conn",
		Type:         "POSTGRESQL",
		Host:         "localhost",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "secret",
	}

	result, err := svc.Execute(testContext(), input, "p")
	require.Nil(t, result)
	require.Error(t, err)

	var respErr pkg.ResponseErrorWithStatusCode
	require.True(t, errors.As(err, &respErr), "duplicate must map to a business error, got %T", err)
	assert.Equal(t, http.StatusConflict, respErr.StatusCode)
}

// TestCreateConnection_RequiresEngineStore proves Create genuinely depends on
// the Engine: an Engine wired WITHOUT a connection store cannot persist, so the
// create surfaces the Engine's "store not configured" rule rather than silently
// bypassing the Engine.
func TestCreateConnection_RequiresEngineStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Encrypt(gomock.Any(), gomock.Any()).Return("enc", "v1", nil)

	storelessEngine, err := engine.New(engine.WithConnectorRegistry(stubConnectorRegistry{}))
	require.NoError(t, err)

	svc := NewCreateConnection(mockCrypto, storelessEngine)

	input := model.ConnectionInput{
		ConfigName:   "no-store",
		Type:         "POSTGRESQL",
		Host:         "localhost",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "secret",
	}

	result, err := svc.Execute(testContext(), input, "p")
	require.Nil(t, result)
	require.Error(t, err)
}

// recordingObservability records Engine span operations so update/delete can be
// proven to route their tenant-scope authority through the Engine.
type recordingObservability struct {
	mu  sync.Mutex
	ops []string
}

func (r *recordingObservability) StartSpan(ctx context.Context, operation string) (context.Context, func()) {
	r.mu.Lock()
	r.ops = append(r.ops, operation)
	r.mu.Unlock()

	return ctx, func() {}
}

func (r *recordingObservability) seen(op string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, got := range r.ops {
		if got == op {
			return true
		}
	}

	return false
}

func engineWithObservability(t *testing.T, obs engine.Observability, checker engine.ActiveExecutionChecker, connRepo connRepo.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(stubConnectorRegistry{}),
		engine.WithConnectionStore(connectioncompat.NewConnectionStore(connRepo)),
		engine.WithActiveExecutionChecker(checker),
		engine.WithObservability(obs),
	)
	require.NoError(t, err)

	return eng
}

// TestUpdateConnection_RoutesScopeAuthorityThroughEngine proves update consults
// the Engine's tenant-scope authority rule before mutating.
func TestUpdateConnection_RoutesScopeAuthorityThroughEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	obs := &recordingObservability{}
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockCrypto.EXPECT().Encrypt(gomock.Any(), gomock.Any()).Return("enc", "v1", nil).AnyTimes()

	spy := &spyActiveExecutionChecker{active: false}
	connID := uuid.New()
	existing := newExistingConnection(connID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(existing, nil)
	mockConnRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, c *model.Connection) (*model.Connection, error) { return c, nil })

	svc := NewUpdateConnection(mockConnRepo, nil, mockCrypto, engineWithObservability(t, obs, spy, mockConnRepo))

	_, err := svc.Execute(testContext(), connID, newUpdateConnectionInput())
	require.NoError(t, err)
	assert.True(t, obs.seen("engine.connection.authorize_access"), "update must route the scope authority through the Engine")
	assert.True(t, obs.seen("engine.connection.get_by_id"), "update must route its read PERSISTENCE through the Engine ID-addressed op")
	assert.True(t, obs.seen("engine.connection.update_by_id"), "update must route its write PERSISTENCE through the Engine ID-addressed op")
}

// TestDeleteConnection_RoutesScopeAuthorityThroughEngine proves delete consults
// the Engine's tenant-scope authority rule before the soft delete.
func TestDeleteConnection_RoutesScopeAuthorityThroughEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	obs := &recordingObservability{}
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	spy := &spyActiveExecutionChecker{active: false}
	connID := uuid.New()
	existing := newExistingConnectionForDelete(connID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(existing, nil)
	mockConnRepo.EXPECT().Delete(gomock.Any(), connID, gomock.Any()).Return(nil)

	svc := NewDeleteConnection(mockConnRepo, nil, engineWithObservability(t, obs, spy, mockConnRepo))

	require.NoError(t, svc.Execute(testContext(), connID))
	assert.True(t, obs.seen("engine.connection.authorize_access"), "delete must route the scope authority through the Engine")
	assert.True(t, obs.seen("engine.connection.get_by_id"), "delete must route its read PERSISTENCE through the Engine ID-addressed op")
	assert.True(t, obs.seen("engine.connection.delete_by_id"), "delete must route its SOFT delete PERSISTENCE through the Engine ID-addressed op")
}
