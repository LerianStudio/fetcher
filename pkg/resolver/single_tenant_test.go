package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	connPort "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSingleTenantResolver_ResolveConnections_MixedInternalAndExternal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := connPort.NewMockRepository(ctrl)

	envConnections := map[string]*model.Connection{
		"midaz_onboarding": {
			ConfigName:   "midaz_onboarding",
			Type:         model.TypePostgreSQL,
			Host:         "localhost",
			Port:         5432,
			DatabaseName: "midaz_onboarding",
			Username:     "user",
		},
	}

	resolver := NewSingleTenantResolver(mockRepo, NewInternalDatasourceRegistry(), envConnections)

	externalConn := &model.Connection{
		ConfigName: "my-oracle",
		Type:       model.TypeOracle,
		Host:       "oracle.example.com",
		Port:       1521,
	}

	mockRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"my-oracle"}).
		Return([]*model.Connection{externalConn}, nil)

	conns, err := resolver.ResolveConnections(context.Background(), []string{"midaz_onboarding", "my-oracle"})
	require.NoError(t, err)
	assert.Len(t, conns, 2)

	// Verify internal was resolved from env
	var internalConn *model.Connection
	for _, c := range conns {
		if c.ConfigName == "midaz_onboarding" {
			internalConn = c
		}
	}

	require.NotNil(t, internalConn)
	assert.Equal(t, "localhost", internalConn.Host)
}

func TestSingleTenantResolver_ResolveConnections_OnlyExternal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := connPort.NewMockRepository(ctrl)
	resolver := NewSingleTenantResolver(mockRepo, NewInternalDatasourceRegistry(), nil)

	externalConn := &model.Connection{ConfigName: "my-db", Type: model.TypePostgreSQL}
	mockRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"my-db"}).
		Return([]*model.Connection{externalConn}, nil)

	conns, err := resolver.ResolveConnections(context.Background(), []string{"my-db"})
	require.NoError(t, err)
	assert.Len(t, conns, 1)
	assert.Equal(t, "my-db", conns[0].ConfigName)
}

func TestSingleTenantResolver_ResolveConnections_InternalNotConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := connPort.NewMockRepository(ctrl)
	// No env connections configured
	resolver := NewSingleTenantResolver(mockRepo, NewInternalDatasourceRegistry(), nil)

	_, err := resolver.ResolveConnections(context.Background(), []string{"midaz_onboarding"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured via environment variables")
}

func TestSingleTenantResolver_ResolveConnections_ExternalRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := connPort.NewMockRepository(ctrl)
	resolver := NewSingleTenantResolver(mockRepo, NewInternalDatasourceRegistry(), nil)

	mockRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"external-db"}).
		Return(nil, errors.New("db connection lost"))

	_, err := resolver.ResolveConnections(context.Background(), []string{"external-db"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve external connections")
}

func TestSingleTenantResolver_IsInternalDatasource(t *testing.T) {
	resolver := NewSingleTenantResolver(nil, NewInternalDatasourceRegistry(), nil)
	assert.True(t, resolver.IsInternalDatasource("midaz_onboarding"))
	assert.True(t, resolver.IsInternalDatasource("plugin_crm"))
	assert.False(t, resolver.IsInternalDatasource("random-db"))
}
