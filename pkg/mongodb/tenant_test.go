package mongodb

import (
	"context"
	"errors"
	"testing"

	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/mock/gomock"
)

// multiTenantProvider wraps MockMongoClientProvider and adds MultiTenantChecker behavior.
type multiTenantProvider struct {
	*MockMongoClientProvider
	multiTenant bool
}

func (m *multiTenantProvider) IsMultiTenant() bool {
	return m.multiTenant
}

func TestGetDatabaseForContext(t *testing.T) {
	tests := []struct {
		name          string
		setupCtx      func() context.Context
		setupProvider func(ctrl *gomock.Controller) MongoClientProvider
		dbName        string
		wantErr       bool
		wantErrIs     error
		wantErrMsg    string
	}{
		{
			name: "single-tenant fallback when provider errors",
			setupCtx: func() context.Context {
				return context.Background()
			},
			setupProvider: func(ctrl *gomock.Controller) MongoClientProvider {
				mock := NewMockMongoClientProvider(ctrl)
				mock.EXPECT().
					Client(gomock.Any()).
					Return(nil, errors.New("connection failed"))
				return mock
			},
			dbName:     "test_db",
			wantErr:    true,
			wantErrMsg: "connection failed",
		},
		{
			name: "multi-tenant returns ErrTenantContextRequired when no tenant in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			setupProvider: func(ctrl *gomock.Controller) MongoClientProvider {
				mock := NewMockMongoClientProvider(ctrl)
				// GetDB should NOT be called when multi-tenant and no tenant in context
				return &multiTenantProvider{
					MockMongoClientProvider: mock,
					multiTenant:             true,
				}
			},
			dbName:    "test_db",
			wantErr:   true,
			wantErrIs: tmcore.ErrTenantContextRequired,
		},
		{
			name: "single-tenant mode falls back to static provider when no tenant context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			setupProvider: func(ctrl *gomock.Controller) MongoClientProvider {
				mock := NewMockMongoClientProvider(ctrl)
				mock.EXPECT().
					Client(gomock.Any()).
					Return(nil, errors.New("static fallback error"))
				return mock
			},
			dbName:     "my_database",
			wantErr:    true,
			wantErrMsg: "static fallback error",
		},
		{
			name: "multi-tenant uses tenant DB from context",
			setupCtx: func() context.Context {
				// Inject a real-ish tenant DB into context.
				// We use mongo.Database obtained from a disconnected client.
				// This tests context extraction; actual DB operations are integration-tested.
				client, err := mongo.NewClient() //nolint:staticcheck // test-only disconnected client
				if err != nil {
					panic("mongo.NewClient() failed: " + err.Error())
				}
				db := client.Database("tenant_abc")
				return tmcore.ContextWithTenantMongo(context.Background(), db)
			},
			setupProvider: func(ctrl *gomock.Controller) MongoClientProvider {
				mock := NewMockMongoClientProvider(ctrl)
				// GetDB should NOT be called when tenant DB is in context
				return &multiTenantProvider{
					MockMongoClientProvider: mock,
					multiTenant:             true,
				}
			},
			dbName:  "default_db",
			wantErr: false,
		},
		{
			name: "single-tenant with tenant DB in context still uses tenant DB",
			setupCtx: func() context.Context {
				client, err := mongo.NewClient() //nolint:staticcheck // test-only disconnected client
				if err != nil {
					panic("mongo.NewClient() failed: " + err.Error())
				}
				db := client.Database("tenant_xyz")
				return tmcore.ContextWithTenantMongo(context.Background(), db)
			},
			setupProvider: func(ctrl *gomock.Controller) MongoClientProvider {
				mock := NewMockMongoClientProvider(ctrl)
				// GetDB should NOT be called when tenant DB is in context
				return mock
			},
			dbName:  "default_db",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := tt.setupCtx()
			provider := tt.setupProvider(ctrl)

			db, err := GetDatabaseForContext(ctx, provider, tt.dbName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, err, tt.wantErrIs)
				}
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
				assert.Nil(t, db)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, db)
			}
		})
	}
}
