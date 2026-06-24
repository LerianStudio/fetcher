package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
	connPort "github.com/LerianStudio/fetcher/pkg/ports/connection"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMultiTenantResolver_ResolveConnections_InternalDatasource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := connPort.NewMockRepository(ctrl)
	tenantID := uuid.New()

	mockTenantConfig := NewMockTenantConfigProvider(ctrl)

	mockTenantConfig.EXPECT().
		GetServiceConnection(gomock.Any(), tenantID.String(), "ledger", "onboarding").
		Return(&ServiceConnectionConfig{
			Host:     "tenant-midaz.db.internal",
			Port:     5432,
			Database: "midaz_onboarding",
			Username: "tenant_user",
			Password: "tenant_pass",
		}, nil)

	resolver := NewMultiTenantResolver(mockRepo, NewInternalDatasourceRegistry(), mockTenantConfig)

	ctx := tmcore.ContextWithTenantID(context.Background(), tenantID.String())

	conns, err := resolver.ResolveConnections(ctx, []string{"midaz_onboarding"})
	require.NoError(t, err)
	require.Len(t, conns, 1)
	assert.Equal(t, "tenant-midaz.db.internal", conns[0].Host)
	assert.Equal(t, model.TypePostgreSQL, conns[0].Type)
	assert.Equal(t, "midaz_onboarding", conns[0].ConfigName)
	assert.Equal(t, 5432, conns[0].Port)
	assert.Equal(t, "midaz_onboarding", conns[0].DatabaseName)
	assert.Equal(t, "tenant_user", conns[0].Username)
	// EncryptionKeyVersion should be empty for in-memory connections
	assert.Empty(t, conns[0].EncryptionKeyVersion)
}

func TestMultiTenantResolver_ResolveConnections_MixedInternalAndExternal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := connPort.NewMockRepository(ctrl)
	tenantID := uuid.New()
	mockTenantConfig := NewMockTenantConfigProvider(ctrl)

	mockTenantConfig.EXPECT().
		GetServiceConnection(gomock.Any(), tenantID.String(), "ledger", "onboarding").
		Return(&ServiceConnectionConfig{
			Host:     "tenant-db.internal",
			Port:     5432,
			Database: "midaz_onboarding",
			Username: "user",
			Password: "pass",
		}, nil)

	externalConn := &model.Connection{ConfigName: "my-oracle", Type: model.TypeOracle}
	mockRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"my-oracle"}).
		Return([]*model.Connection{externalConn}, nil)

	resolver := NewMultiTenantResolver(mockRepo, NewInternalDatasourceRegistry(), mockTenantConfig)
	ctx := tmcore.ContextWithTenantID(context.Background(), tenantID.String())

	conns, err := resolver.ResolveConnections(ctx, []string{"midaz_onboarding", "my-oracle"})
	require.NoError(t, err)
	assert.Len(t, conns, 2)
}

func TestMultiTenantResolver_ResolveConnections_NoTenantInContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := connPort.NewMockRepository(ctrl)
	mockTenantConfig := NewMockTenantConfigProvider(ctrl)

	resolver := NewMultiTenantResolver(mockRepo, NewInternalDatasourceRegistry(), mockTenantConfig)

	// No tenant ID set in context
	_, err := resolver.ResolveConnections(context.Background(), []string{"midaz_onboarding"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID not found in context")
}

func TestMultiTenantResolver_ResolveConnections_TenantProviderError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := connPort.NewMockRepository(ctrl)
	tenantID := uuid.New()
	mockTenantConfig := NewMockTenantConfigProvider(ctrl)

	mockTenantConfig.EXPECT().
		GetServiceConnection(gomock.Any(), tenantID.String(), "ledger", "onboarding").
		Return(nil, errors.New("tenant-manager unavailable"))

	resolver := NewMultiTenantResolver(mockRepo, NewInternalDatasourceRegistry(), mockTenantConfig)
	ctx := tmcore.ContextWithTenantID(context.Background(), tenantID.String())

	_, err := resolver.ResolveConnections(ctx, []string{"midaz_onboarding"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant-manager lookup")
}

func TestMultiTenantResolver_ResolveConnections_SSLModeSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := connPort.NewMockRepository(ctrl)
	tenantID := uuid.New()
	mockTenantConfig := NewMockTenantConfigProvider(ctrl)

	mockTenantConfig.EXPECT().
		GetServiceConnection(gomock.Any(), tenantID.String(), "ledger", "onboarding").
		Return(&ServiceConnectionConfig{
			Host:     "db.internal",
			Port:     5432,
			Database: "midaz",
			Username: "user",
			Password: "pass",
			SSLMode:  "require",
		}, nil)

	resolver := NewMultiTenantResolver(mockRepo, NewInternalDatasourceRegistry(), mockTenantConfig)
	ctx := tmcore.ContextWithTenantID(context.Background(), tenantID.String())

	conns, err := resolver.ResolveConnections(ctx, []string{"midaz_onboarding"})
	require.NoError(t, err)
	require.Len(t, conns, 1)
	require.NotNil(t, conns[0].SSL)
	assert.Equal(t, "require", conns[0].SSL.Mode)
}

func TestMultiTenantResolver_ResolveConnections_MongoDBTLSAndMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := connPort.NewMockRepository(ctrl)
	tenantID := uuid.New()
	mockTenantConfig := NewMockTenantConfigProvider(ctrl)

	mockTenantConfig.EXPECT().
		GetServiceConnection(gomock.Any(), tenantID.String(), "plugin-crm", "crm-api").
		Return(&ServiceConnectionConfig{
			Host:             "docdb.cluster.amazonaws.com",
			Port:             27017,
			Database:         "crm_tenant1",
			Username:         "crm_user",
			Password:         "crm_pass",
			SSLMode:          "insecure",
			DirectConnection: true,
			AuthSource:       "crm_tenant1",
		}, nil)

	resolver := NewMultiTenantResolver(mockRepo, NewInternalDatasourceRegistry(), mockTenantConfig)
	ctx := tmcore.ContextWithTenantID(context.Background(), tenantID.String())

	conns, err := resolver.ResolveConnections(ctx, []string{"plugin_crm"})
	require.NoError(t, err)
	require.Len(t, conns, 1)

	conn := conns[0]
	assert.Equal(t, model.TypeMongoDB, conn.Type)
	assert.Equal(t, "plugin_crm", conn.ConfigName)
	assert.Equal(t, "docdb.cluster.amazonaws.com", conn.Host)
	assert.Equal(t, 27017, conn.Port)

	// SSL should be set with insecure mode for DocumentDB with TLSSkipVerify
	require.NotNil(t, conn.SSL)
	assert.Equal(t, "insecure", conn.SSL.Mode)

	// Metadata should carry directConnection and authSource for the datasource factory
	require.NotNil(t, conn.Metadata)
	assert.Equal(t, "true", (*conn.Metadata)["directConnection"])
	assert.Equal(t, "crm_tenant1", (*conn.Metadata)["authSource"])
}

func TestBuildMongoServiceConfig_TLSFromURI(t *testing.T) {
	cfg := &tmcore.MongoDBConfig{
		Host:     "shared-stg-docdb.cluster.amazonaws.com",
		Port:     27017,
		Database: "crm_tenant1",
		Username: "crm_user",
		Password: "crm_pass",
		URI:      "mongodb://crm_user:crm_pass@shared-stg-docdb.cluster.amazonaws.com:27017/crm_tenant1?authSource=crm_tenant1&tls=true&tlsInsecure=true&retryWrites=false",
		// TLS and TLSSkipVerify are NOT set (zero values) — TLS info is only in the URI.
	}

	result := buildMongoServiceConfig(cfg)

	assert.Equal(t, "insecure", result.SSLMode, "SSLMode should be parsed from URI tls+tlsInsecure params")
	assert.Equal(t, "crm_tenant1", result.AuthSource, "AuthSource should be parsed from URI")
	assert.Equal(t, "shared-stg-docdb.cluster.amazonaws.com", result.Host)
	assert.Equal(t, 27017, result.Port)
}

func TestBuildMongoServiceConfig_ExplicitFieldsTakePrecedence(t *testing.T) {
	cfg := &tmcore.MongoDBConfig{
		Host:          "docdb.amazonaws.com",
		Port:          27017,
		Database:      "mydb",
		Username:      "user",
		Password:      "pass",
		TLS:           true,
		TLSSkipVerify: false,
		URI:           "mongodb://user:pass@docdb.amazonaws.com:27017/mydb?tls=true&tlsInsecure=true",
	}

	result := buildMongoServiceConfig(cfg)

	// Explicit TLS=true + TLSSkipVerify=false should produce "enable", NOT "insecure" from URI
	assert.Equal(t, "enable", result.SSLMode)
}

func TestBuildMongoServiceConfig_NoTLS(t *testing.T) {
	cfg := &tmcore.MongoDBConfig{
		Host:     "localhost",
		Port:     27017,
		Database: "testdb",
		Username: "user",
		Password: "pass",
	}

	result := buildMongoServiceConfig(cfg)

	assert.Empty(t, result.SSLMode)
	assert.False(t, result.DirectConnection)
	assert.Empty(t, result.AuthSource)
}

func TestMultiTenantResolver_IsInternalDatasource(t *testing.T) {
	resolver := NewMultiTenantResolver(nil, NewInternalDatasourceRegistry(), nil)
	assert.True(t, resolver.IsInternalDatasource("midaz_onboarding"))
	assert.True(t, resolver.IsInternalDatasource("plugin_crm"))
	assert.False(t, resolver.IsInternalDatasource("random-db"))
}
