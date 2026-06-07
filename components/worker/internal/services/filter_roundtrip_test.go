// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package services

import (
	"context"
	"testing"

	workerCrypto "github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/engine"
	enginecompatdatasource "github.com/LerianStudio/fetcher/pkg/enginecompat/datasource"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelDatasource "github.com/LerianStudio/fetcher/pkg/model/datasource"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	libLog "github.com/LerianStudio/lib-observability/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// singleFactoryRegistry resolves one ConnectorFactory for any type.
type singleFactoryRegistry struct{ factory engine.ConnectorFactory }

func (r singleFactoryRegistry) Connector(string) (engine.ConnectorFactory, bool) {
	return r.factory, true
}

// realEngineRunner wires a REAL engine.Engine over the production extraction
// ConnectorFactory + schemacompat ConnectionStore, exactly like the worker
// bootstrap's newWorkerEngineRunner, so the test drives the genuine
// PlanExtraction -> ExecuteExtraction -> connector path.
type realEngineRunner struct{ eng *engine.Engine }

func (r *realEngineRunner) RunExtraction(
	ctx context.Context,
	tenant engine.TenantContext,
	jobID string,
	request engine.ExtractionRequest,
) (engine.ExtractionResult, error) {
	plan, err := r.eng.PlanExtraction(ctx, tenant, request)
	if err != nil {
		return engine.ExtractionResult{}, err
	}

	plan.RequestID = jobID

	return r.eng.ExecuteExtraction(ctx, plan)
}

func newRealEngineRunner(t *testing.T, ds modelDatasource.DataSource) *realEngineRunner {
	t.Helper()

	factory := enginecompatdatasource.NewConnectorFactory(
		func(context.Context, *model.Connection, workerCrypto.Cryptor) (modelDatasource.DataSource, error) {
			return ds, nil
		},
		func(context.Context, engine.ConnectionDescriptor) (string, error) { return "", nil },
		nil,
	)

	eng, err := engine.New(
		engine.WithConnectorRegistry(singleFactoryRegistry{factory: factory}),
		engine.WithConnectionStore(schemacompat.NewConnectionStore()),
	)
	require.NoError(t, err)

	return &realEngineRunner{eng: eng}
}

// TestExtractInto_GenericFilters_ReachDataSourceQuery_EndToEnd is the FIX-1 (CRITICAL)
// regression guard. It drives the REAL engine end to end through the Worker mapper:
// a generic-datasource job carrying a filter must have that filter SURVIVE the
// PlanExtraction -> ExecuteExtraction round-trip and reach the underlying
// DataSource.Query (filtered), NOT be silently dropped (which would extract the full
// table). Before FIX-1, mapFilters emitted the typed shape, the planner's
// map[string]any assertion failed, step.Filters was never set, and Query received
// nil filters — this test fails. After FIX-1 (nested map[string]any out of the
// mapper + reconstruction in filtersForConfig) the filter reaches Query.
func TestExtractInto_GenericFilters_ReachDataSourceQuery_EndToEnd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	schema := model.NewDataSourceSchema("pg-main")
	schema.AddTable("users", []string{"id", "name", "status"})

	mockDS := modelDatasource.NewMockDataSource(ctrl)
	mockDS.EXPECT().GetSchemaInfo(gomock.Any(), gomock.Any()).Return(schema, nil).AnyTimes()
	mockDS.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()

	// Capture the filters the engine ultimately hands to DataSource.Query.
	var capturedFilters map[string]map[string]modelJob.FilterCondition
	mockDS.EXPECT().
		Query(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ map[string][]string, filters map[string]map[string]modelJob.FilterCondition, _ libLog.Logger) (map[string][]map[string]any, error) {
			capturedFilters = filters
			return map[string][]map[string]any{"users": {{"id": float64(1), "status": "active"}}}, nil
		})

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.SetStorageEncryptDerivedKey(storageKey)
	uc.EngineRunner = newRealEngineRunner(t, mockDS)

	conn := &model.Connection{ID: newTestJobID(), ConfigName: "pg-main", Type: model.TypePostgreSQL, Host: "db", Port: 5432, Username: "r"}
	conn.SetPlaintextPassword("p")

	message := ExtractExternalDataMessage{
		JobID:        newTestJobID(),
		MappedFields: map[string]map[string][]string{"pg-main": {"users": {"id", "name", "status"}}},
		Filters: map[string]map[string]map[string]modelJob.FilterCondition{
			"pg-main": {"users": {"status": {Equals: []any{"active"}}}},
		},
		Metadata: map[string]any{"source": "test"},
	}

	// Seed the resolved connection into ctx exactly as extractViaEngine does, since we
	// call extractInto directly here.
	ctx := schemacompat.WithResolvedConnections(testContext(), []*model.Connection{conn})

	result := make(map[string]map[string][]map[string]any)
	_, errExtractInto := uc.extractInto(ctx, message, []*model.Connection{conn}, result)
	require.NoError(t, errExtractInto)

	// The CRITICAL assertion: the filter survived plan->execute and reached Query.
	require.NotNil(t, capturedFilters, "filters were DROPPED before reaching DataSource.Query (FIX-1 regression)")
	usersFilters, ok := capturedFilters["users"]
	require.True(t, ok, "expected users-table filters at Query, got %#v", capturedFilters)
	statusCond, ok := usersFilters["status"]
	require.True(t, ok, "expected a status filter condition, got %#v", usersFilters)
	require.Len(t, statusCond.Equals, 1)
	assert.Equal(t, "active", statusCond.Equals[0])
}
