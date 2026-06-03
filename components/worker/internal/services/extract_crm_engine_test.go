// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package services

import (
	"context"
	"testing"

	workerCrypto "github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelDatasource "github.com/LerianStudio/fetcher/pkg/model/datasource"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	portDS "github.com/LerianStudio/fetcher/pkg/ports/datasource"
	libLog "github.com/LerianStudio/lib-observability/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// crmCompatDataSource is a combined test double that satisfies BOTH the generic
// datasource.DataSource interface AND the CRM-specific portDS.CRMQueryable
// interface, mirroring the production MongoDB datasource which implements both.
// The factory returns it for a plugin_crm connection so the Worker's CRM branch
// (the dataSource.(CRMQueryable) cast) succeeds.
type crmCompatDataSource struct {
	*portDS.MockCRMQueryable
}

func (d crmCompatDataSource) GetConfig() modelDatasource.DataSourceConfig {
	return modelDatasource.DataSourceConfig{Type: string(model.TypeMongoDB), ConfigName: "plugin_crm"}
}
func (d crmCompatDataSource) GetType() string { return string(model.TypeMongoDB) }
func (d crmCompatDataSource) Connect(context.Context, libLog.Logger) error { return nil }
func (d crmCompatDataSource) Close(context.Context) error                  { return nil }
func (d crmCompatDataSource) Query(context.Context, map[string][]string, map[string]map[string]modelJob.FilterCondition, libLog.Logger) (map[string][]map[string]any, error) {
	// Generic Query must NEVER be used for plugin_crm: the CRM compat path uses the
	// CRMQueryable collection methods. Returning a sentinel makes a wrong route loud.
	panic("plugin_crm must not use generic DataSource.Query")
}
func (d crmCompatDataSource) GetSchemaInfo(context.Context, []string) (*model.DataSourceSchema, error) {
	return model.NewDataSourceSchema("plugin_crm"), nil
}

// crmCallTrackingRunner records whether the Engine runner was invoked and for
// which datasource config names, so a test can prove the CRM datasource never
// reaches the generic Engine runner.
type crmCallTrackingRunner struct {
	configNames []string
}

func (r *crmCallTrackingRunner) RunExtraction(
	_ context.Context,
	_ engine.TenantContext,
	_ string,
	request engine.ExtractionRequest,
) (engine.ExtractionResult, error) {
	for cfg := range request.MappedFields {
		r.configNames = append(r.configNames, cfg)
	}

	return engine.ExtractionResult{Direct: &engine.DirectResult{Data: []byte(`{}`), Format: "json"}}, nil
}

// TestExtractInto_PluginCRM_CRMCompatibility_RoutesThroughCRMPath proves that with
// the Engine runner WIRED, a plugin_crm datasource still extracts through the
// EXPLICIT CRM compatibility path (ListCollectionNames + prefix fan-out +
// QueryPluginCRM), NOT the generic Engine runner. CRM extraction behavior is
// preserved byte-identical because it reuses the unchanged QueryPluginCRM chain.
func TestExtractInto_PluginCRM_CRMCompatibility_RoutesThroughCRMPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalProcess := processPluginCRMCollectionFn
	t.Cleanup(func() { processPluginCRMCollectionFn = originalProcess })

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	runner := &crmCallTrackingRunner{}
	uc.EngineRunner = runner

	// CRM datasource: combined DataSource + CRMQueryable, returned by the factory.
	mockCRM := portDS.NewMockCRMQueryable(ctrl)
	mockCRM.EXPECT().ListCollectionNames(gomock.Any()).Return([]string{"holders_org-1"}, nil)
	uc.SetDataSourceFactory(func(context.Context, *model.Connection, workerCrypto.Cryptor) (modelDatasource.DataSource, error) {
		return crmCompatDataSource{MockCRMQueryable: mockCRM}, nil
	})

	processed := false
	processPluginCRMCollectionFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, _ []string, _ map[string]modelJob.FilterCondition, matching []string, result map[string]map[string][]map[string]any, _ libLog.Logger) error {
		processed = true
		assert.Equal(t, "holders", collection)
		assert.Equal(t, []string{"holders_org-1"}, matching, "CRM prefix fan-out must resolve physical collections")
		if result["plugin_crm"] == nil {
			result["plugin_crm"] = map[string][]map[string]any{}
		}
		result["plugin_crm"][collection] = []map[string]any{{"id": "1"}}
		return nil
	}

	message := ExtractExternalDataMessage{
		JobID:        newTestJobID(),
		MappedFields: map[string]map[string][]string{"plugin_crm": {"holders": {"id", "document"}}},
		Metadata:     map[string]any{"source": "plugin_crm"},
	}
	connections := []*model.Connection{{ConfigName: "plugin_crm", Type: model.TypeMongoDB}}
	result := make(map[string]map[string][]map[string]any)

	require.NoError(t, uc.extractInto(testContext(), message, connections, result))

	assert.True(t, processed, "plugin_crm must extract through the CRM compatibility path")
	assert.NotContains(t, runner.configNames, "plugin_crm",
		"plugin_crm must NOT be routed through the generic Engine runner")
	assert.Len(t, result["plugin_crm"]["holders"], 1, "CRM extraction result must be present")
}

// TestExtractInto_NonCRM_NeverExecutesCRMCompatibility proves a generic (non-CRM)
// job routes through the Engine runner and NEVER touches CRM compatibility code:
// processPluginCRMCollectionFn is never invoked.
func TestExtractInto_NonCRM_NeverExecutesCRMCompatibility(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalProcess := processPluginCRMCollectionFn
	t.Cleanup(func() { processPluginCRMCollectionFn = originalProcess })

	crmTouched := false
	processPluginCRMCollectionFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, _ string, _ []string, _ map[string]modelJob.FilterCondition, _ []string, _ map[string]map[string][]map[string]any, _ libLog.Logger) error {
		crmTouched = true
		return nil
	}

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	runner := &crmCallTrackingRunner{}
	uc.EngineRunner = runner

	message := ExtractExternalDataMessage{
		JobID:        newTestJobID(),
		MappedFields: map[string]map[string][]string{"pg": {"users": {"id"}}},
		Metadata:     map[string]any{"source": "some-product"},
	}
	connections := []*model.Connection{{ConfigName: "pg", Type: model.TypePostgreSQL}}
	result := make(map[string]map[string][]map[string]any)

	require.NoError(t, uc.extractInto(testContext(), message, connections, result))

	assert.False(t, crmTouched, "non-CRM job must NEVER execute CRM compatibility code")
	assert.Contains(t, runner.configNames, "pg", "non-CRM datasource must route through the Engine runner")
}

// TestExtractInto_PluginCRM_CRMCompatibility_ByteIdenticalToLegacy proves the CRM
// extraction RESULT is byte-identical whether the Worker runs legacy (EngineRunner
// nil) or engine-enabled (EngineRunner set, which splits plugin_crm to the same CRM
// path). Same CRM rows in -> same result map out.
func TestExtractInto_PluginCRM_CRMCompatibility_ByteIdenticalToLegacy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalProcess := processPluginCRMCollectionFn
	t.Cleanup(func() { processPluginCRMCollectionFn = originalProcess })

	// Deterministic CRM processor: same rows regardless of the routing path.
	processPluginCRMCollectionFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, _ []string, _ map[string]modelJob.FilterCondition, _ []string, result map[string]map[string][]map[string]any, _ libLog.Logger) error {
		if result["plugin_crm"] == nil {
			result["plugin_crm"] = map[string][]map[string]any{}
		}
		result["plugin_crm"][collection] = []map[string]any{{"id": "1", "document": "decrypted-doc"}}
		return nil
	}

	message := ExtractExternalDataMessage{
		JobID:        newTestJobID(),
		MappedFields: map[string]map[string][]string{"plugin_crm": {"holders": {"id", "document"}}},
		Metadata:     map[string]any{"source": "plugin_crm"},
	}
	connections := []*model.Connection{{ConfigName: "plugin_crm", Type: model.TypeMongoDB}}

	newCRMUC := func() *UseCase {
		mocks := newTestMocks(ctrl)
		uc := newTestUseCase(mocks)
		mockCRM := portDS.NewMockCRMQueryable(ctrl)
		mockCRM.EXPECT().ListCollectionNames(gomock.Any()).Return([]string{"holders_org-1"}, nil)
		uc.SetDataSourceFactory(func(context.Context, *model.Connection, workerCrypto.Cryptor) (modelDatasource.DataSource, error) {
			return crmCompatDataSource{MockCRMQueryable: mockCRM}, nil
		})
		return uc
	}

	// Legacy path (EngineRunner nil).
	legacyUC := newCRMUC()
	legacyResult := make(map[string]map[string][]map[string]any)
	require.NoError(t, legacyUC.extractInto(testContext(), message, connections, legacyResult))

	// Engine-enabled path (EngineRunner set -> split routes plugin_crm to CRM path).
	engineUC := newCRMUC()
	engineUC.EngineRunner = &crmCallTrackingRunner{}
	engineResult := make(map[string]map[string][]map[string]any)
	require.NoError(t, engineUC.extractInto(testContext(), message, connections, engineResult))

	assert.Equal(t, legacyResult, engineResult, "CRM extraction result must be byte-identical across legacy and engine-enabled paths")
}

// TestExtractInto_MixedCRMAndGeneric_CRMCompatibilityRoutesBoth proves a job mixing
// plugin_crm AND a generic datasource splits correctly: the CRM datasource extracts
// via the CRM path and the generic datasource via the Engine runner, merging into
// one result. The Engine runner sees ONLY the generic config, never plugin_crm.
func TestExtractInto_MixedCRMAndGeneric_CRMCompatibilityRoutesBoth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalProcess := processPluginCRMCollectionFn
	t.Cleanup(func() { processPluginCRMCollectionFn = originalProcess })

	processPluginCRMCollectionFn = func(_ *UseCase, _ context.Context, _ portDS.CRMQueryable, collection string, _ []string, _ map[string]modelJob.FilterCondition, _ []string, result map[string]map[string][]map[string]any, _ libLog.Logger) error {
		if result["plugin_crm"] == nil {
			result["plugin_crm"] = map[string][]map[string]any{}
		}
		result["plugin_crm"][collection] = []map[string]any{{"id": "crm-1"}}
		return nil
	}

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	mockCRM := portDS.NewMockCRMQueryable(ctrl)
	mockCRM.EXPECT().ListCollectionNames(gomock.Any()).Return([]string{"holders_org-1"}, nil)
	uc.SetDataSourceFactory(func(context.Context, *model.Connection, workerCrypto.Cryptor) (modelDatasource.DataSource, error) {
		return crmCompatDataSource{MockCRMQueryable: mockCRM}, nil
	})

	// Engine runner returns the generic datasource rows.
	uc.EngineRunner = &fixedGenericRunner{
		data: []byte(`{"pg":{"users":[{"id":"pg-1"}]}}`),
		seen: &[]string{},
	}

	message := ExtractExternalDataMessage{
		JobID: newTestJobID(),
		MappedFields: map[string]map[string][]string{
			"plugin_crm": {"holders": {"id"}},
			"pg":         {"users": {"id"}},
		},
		Metadata: map[string]any{"source": "plugin_crm"},
	}
	connections := []*model.Connection{
		{ConfigName: "plugin_crm", Type: model.TypeMongoDB},
		{ConfigName: "pg", Type: model.TypePostgreSQL},
	}
	result := make(map[string]map[string][]map[string]any)

	require.NoError(t, uc.extractInto(testContext(), message, connections, result))

	// Both datasources merged into one result.
	assert.Len(t, result["plugin_crm"]["holders"], 1, "CRM datasource extracted via CRM path")
	assert.Len(t, result["pg"]["users"], 1, "generic datasource extracted via Engine runner")

	// Engine runner saw ONLY the generic config, never plugin_crm.
	runner := uc.EngineRunner.(*fixedGenericRunner)
	assert.Contains(t, *runner.seen, "pg")
	assert.NotContains(t, *runner.seen, "plugin_crm", "Engine runner must never receive plugin_crm")
}

// fixedGenericRunner returns fixed direct bytes and records which config names it
// was asked to extract.
type fixedGenericRunner struct {
	data []byte
	seen *[]string
}

func (r *fixedGenericRunner) RunExtraction(
	_ context.Context,
	_ engine.TenantContext,
	_ string,
	request engine.ExtractionRequest,
) (engine.ExtractionResult, error) {
	for cfg := range request.MappedFields {
		*r.seen = append(*r.seen, cfg)
	}

	return engine.ExtractionResult{Direct: &engine.DirectResult{Data: r.data, Format: "json"}}, nil
}
