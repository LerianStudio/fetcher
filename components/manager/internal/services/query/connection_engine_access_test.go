package query

import (
	"context"
	"sync"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	netHTTP "github.com/LerianStudio/fetcher/v2/pkg/net/http"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// recordingObservability records the Engine span operations the read services
// trigger, so a test can prove the scope-authority gate (the Engine op named
// "engine.connection.authorize_access") is actually consulted on get/list.
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

func engineWithObservability(t *testing.T, obs engine.Observability, connRepo connRepo.Repository) *engine.Engine {
	t.Helper()

	eng, err := engine.New(
		engine.WithConnectorRegistry(stubConnectorRegistry{}),
		engine.WithConnectionStore(connectioncompat.NewConnectionStore(connRepo)),
		engine.WithObservability(obs),
	)
	require.NoError(t, err)

	return eng
}

// TestAuthorizeConnectionAccess_NilEngineIsNoOp proves a read service built
// without an Engine (test-only) skips the scope gate rather than panicking. It
// exercises the shared connectioncompat.AuthorizeAccess helper that command and
// query both delegate to.
func TestAuthorizeConnectionAccess_NilEngineIsNoOp(t *testing.T) {
	require.NoError(t, connectioncompat.AuthorizeAccess(testContext(), nil))
}

// TestGetConnection_RoutesScopeAuthorityThroughEngine proves the get read
// consults the Engine's tenant-scope authority rule before resolving the
// connection by UUID.
func TestGetConnection_RoutesScopeAuthorityThroughEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	obs := &recordingObservability{}
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	connID := uuid.New()
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(&model.Connection{ID: connID, ConfigName: "c"}, nil)

	svc := NewGetConnection(nil, nil, engineWithObservability(t, obs, mockConnRepo))

	_, err := svc.Execute(testContext(), connID)
	require.NoError(t, err)
	assert.True(t, obs.seen("engine.connection.authorize_access"), "get must route the scope authority through the Engine")
	assert.True(t, obs.seen("engine.connection.get_by_id"), "get must route the external read PERSISTENCE through the Engine ID-addressed op")
}

// TestListConnections_RoutesScopeAuthorityThroughEngine proves the list read
// consults the Engine's tenant-scope authority rule before the paginated repo
// read.
func TestListConnections_RoutesScopeAuthorityThroughEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	obs := &recordingObservability{}
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockConnRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(0), nil)

	svc := NewListConnections(nil, engineWithObservability(t, obs, mockConnRepo))

	_, err := svc.Execute(testContext(), "", netHTTP.QueryHeader{Limit: 10, Page: 1})
	require.NoError(t, err)
	assert.True(t, obs.seen("engine.connection.authorize_access"), "list must route the scope authority through the Engine")
	assert.True(t, obs.seen("engine.connection.list_paged"), "list must route the paginated read PERSISTENCE through the Engine ID-addressed op")
}
