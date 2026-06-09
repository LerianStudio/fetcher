// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	modelDatasource "github.com/LerianStudio/fetcher/v2/pkg/model/datasource"
	connPort "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/v2/pkg/ports/job"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// These tests pin the WIRING CONTRACT of the Manager's two embedded engines.
// They construct each engine through the REAL bootstrap functions
// (connectionEngine, schemaEngine) with the intended port set and assert two
// things a future port-drop would break:
//
//  1. engine.New SUCCEEDS and returns a usable engine (no infra required); and
//  2. the SPECIFIC optional ports each bootstrap wires are present and ACTIVE,
//     proven by driving a public engine operation whose result depends on the
//     port being wired (a missing port would surface a different,
//     "port not configured" error or skip the gate entirely).
//
// No real Mongo/Redis is started: the connection store rides over a gomock
// connection repo, the active-execution checker over a tiny job-repo fake, and
// the schema cache over a fake cache port — matching the established mocking
// pattern in the sibling enginecompat tests.

const wiringTenant = "tenant-wiring"

func mustWiringTenant(t *testing.T) engine.TenantContext {
	t.Helper()

	tenant, err := engine.NewTenantContext(wiringTenant)
	if err != nil {
		t.Fatalf("NewTenantContext: %v", err)
	}

	return tenant
}

// stubJobRepo is a minimal job.Repository whose only meaningful method is
// ExistsRunningByMappedFieldKey — the single method the ActiveExecutionChecker
// adapter calls. The remaining methods are present only to satisfy the
// interface and fail loudly if the wiring ever routes through them unexpectedly.
type stubJobRepo struct {
	existsRunning bool
	existsErr     error
}

func (s *stubJobRepo) ExistsRunningByMappedFieldKey(_ context.Context, _ string) (bool, error) {
	return s.existsRunning, s.existsErr
}

func (s *stubJobRepo) Create(context.Context, *model.Job) (*model.Job, error) {
	panic("stubJobRepo.Create must not be called")
}

func (s *stubJobRepo) Update(context.Context, *model.Job) (*model.Job, error) {
	panic("stubJobRepo.Update must not be called")
}

func (s *stubJobRepo) UpdateStatus(context.Context, uuid.UUID, model.JobStatus, string, string, map[string]any) error {
	panic("stubJobRepo.UpdateStatus must not be called")
}

func (s *stubJobRepo) FindByID(context.Context, uuid.UUID) (*model.Job, error) {
	panic("stubJobRepo.FindByID must not be called")
}

func (s *stubJobRepo) FindByRequestHashWithinWindow(context.Context, string, int) (*model.Job, error) {
	panic("stubJobRepo.FindByRequestHashWithinWindow must not be called")
}

func (s *stubJobRepo) FindActiveByRequestHash(context.Context, string) (*model.Job, error) {
	panic("stubJobRepo.FindActiveByRequestHash must not be called")
}

func (s *stubJobRepo) List(context.Context, *job.ListFilter) ([]*model.Job, error) {
	panic("stubJobRepo.List must not be called")
}

// compile-time proof the stub satisfies the port the bootstrap adapter needs.
var _ job.Repository = (*stubJobRepo)(nil)

// fakeWiringCachePort is a minimal cache.SchemaCacheRepository used to prove the
// schema engine's SchemaCache port is wired: a Get hit short-circuits live
// discovery, so a successful DiscoverSchema with NO connector ever built proves
// the cache port carried the result.
type fakeWiringCachePort struct {
	getSchema *model.DataSourceSchema
	getErr    error
}

func (f *fakeWiringCachePort) Get(context.Context, string) (*model.DataSourceSchema, error) {
	return f.getSchema, f.getErr
}

func (f *fakeWiringCachePort) Set(context.Context, string, *model.DataSourceSchema, time.Duration) error {
	return nil
}

func (f *fakeWiringCachePort) Delete(context.Context, string) error { return nil }
func (f *fakeWiringCachePort) Clear(context.Context) error          { return nil }
func (f *fakeWiringCachePort) IsHealthy(context.Context) bool       { return true }
func (f *fakeWiringCachePort) Close() error                         { return nil }

// --- connectionEngine wiring -------------------------------------------------

func TestConnectionEngine_ConstructsSuccessfully(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connRepo := connPort.NewMockRepository(ctrl)
	jobRepo := &stubJobRepo{}

	eng, err := connectionEngine(connRepo, jobRepo)
	if err != nil {
		t.Fatalf("connectionEngine: unexpected error: %v", err)
	}

	if eng == nil {
		t.Fatal("connectionEngine returned a nil engine")
	}
}

func TestConnectionEngine_WiresConnectionStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connRepo := connPort.NewMockRepository(ctrl)
	// The store is ACTIVE when ListConnections routes through to the repo. If the
	// ConnectionStore port were dropped, the engine would short-circuit with a
	// "connection store is not configured" validation error and never call List.
	connRepo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return([]*model.Connection{
			{ConfigName: "db1", Type: model.TypePostgreSQL, Host: "h"},
		}, int64(1), nil)

	eng, err := connectionEngine(connRepo, &stubJobRepo{})
	if err != nil {
		t.Fatalf("connectionEngine: unexpected error: %v", err)
	}

	descriptors, err := eng.ListConnections(context.Background(), mustWiringTenant(t))
	if err != nil {
		t.Fatalf("ListConnections through wired store: unexpected error: %v", err)
	}

	if len(descriptors) != 1 || descriptors[0].ConfigName != "db1" {
		t.Fatalf("store not wired: got %+v, want one descriptor for db1", descriptors)
	}
}

func TestConnectionEngine_WiresActiveExecutionChecker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connRepo := connPort.NewMockRepository(ctrl)
	// The checker is ACTIVE: a running job for the connection makes
	// CheckActiveExecutions return a CategoryConflict. If the
	// ActiveExecutionChecker port were dropped, CheckActiveExecutions would
	// short-circuit to nil (no gate), so the conflict proves the port is wired.
	jobRepo := &stubJobRepo{existsRunning: true}

	eng, err := connectionEngine(connRepo, jobRepo)
	if err != nil {
		t.Fatalf("connectionEngine: unexpected error: %v", err)
	}

	err = eng.CheckActiveExecutions(context.Background(), mustWiringTenant(t), "db1")
	if err == nil {
		t.Fatal("checker not wired: CheckActiveExecutions returned nil despite a running job")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryConflict {
		t.Fatalf("want CategoryConflict from wired checker, got %v", err)
	}
}

func TestConnectionEngine_ActiveExecutionCheckerPropagatesRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connRepo := connPort.NewMockRepository(ctrl)
	jobRepo := &stubJobRepo{existsErr: errors.New("mongo down")}

	eng, err := connectionEngine(connRepo, jobRepo)
	if err != nil {
		t.Fatalf("connectionEngine: unexpected error: %v", err)
	}

	if err := eng.CheckActiveExecutions(context.Background(), mustWiringTenant(t), "db1"); err == nil {
		t.Fatal("checker not wired: a repo error must surface through the gate, got nil")
	}
}

// --- schemaEngine wiring -----------------------------------------------------

func TestSchemaEngine_ConstructsSuccessfully(t *testing.T) {
	eng, err := schemaEngine(
		func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
			return nil, errors.New("must not connect in this test")
		},
		nil,
		&fakeWiringCachePort{},
		time.Minute,
	)
	if err != nil {
		t.Fatalf("schemaEngine: unexpected error: %v", err)
	}

	if eng == nil {
		t.Fatal("schemaEngine returned a nil engine")
	}
}

func TestSchemaEngine_WiresConnectionStoreAndSchemaCache(t *testing.T) {
	// A cached schema short-circuits live discovery: a successful DiscoverSchema
	// that NEVER reaches the datasource factory proves BOTH the request-scoped
	// ConnectionStore (resolves the seeded connection) AND the SchemaCache port
	// (returns the cached snapshot) are wired. The factory below panics if the
	// cache port were missing and the engine fell through to a live connect.
	cached := model.NewDataSourceSchema("db1")
	cached.AddTable("users", []string{"id", "name"})

	factoryCalled := false
	dsFactory := func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		factoryCalled = true
		return nil, errors.New("cache must short-circuit discovery; factory must not run")
	}

	eng, err := schemaEngine(dsFactory, nil, &fakeWiringCachePort{getSchema: cached}, time.Minute)
	if err != nil {
		t.Fatalf("schemaEngine: unexpected error: %v", err)
	}

	conn := &model.Connection{ConfigName: "db1", Type: model.TypePostgreSQL, Host: "db.example.com"}
	ctx := schemacompat.WithResolvedConnections(context.Background(), []*model.Connection{conn})

	snapshot, err := eng.DiscoverSchema(ctx, mustWiringTenant(t), "db1")
	if err != nil {
		t.Fatalf("DiscoverSchema through wired store+cache: unexpected error: %v", err)
	}

	if factoryCalled {
		t.Fatal("schema cache not wired: discovery fell through to the live datasource factory")
	}

	if snapshot.ConfigName != "db1" || len(snapshot.Tables) != 1 || snapshot.Tables[0].Name != "users" {
		t.Fatalf("cache port did not carry the snapshot: got %+v", snapshot)
	}
}

func TestSchemaEngine_WiresConnectorFactory(t *testing.T) {
	// On a cache MISS the engine resolves the wired ConnectorFactory and calls
	// its Build. The schemacompat factory validates the descriptor's datasource
	// type OFFLINE in Build, so an UNKNOWN type fails the build WITHOUT ever
	// reaching the datasource factory (no connect). The engine redacts any Build
	// failure to CategoryUnavailable "failed to build connector for connection".
	// Proving the build failed offline (the datasource factory was never invoked)
	// is what pins the ConnectorFactory wiring: discovery reached Build through the
	// wired registry rather than short-circuiting.
	connectAttempted := false
	dsFactory := func(context.Context, *model.Connection, crypto.Cryptor) (modelDatasource.DataSource, error) {
		connectAttempted = true
		return nil, errors.New("must not connect: build rejects the unknown type first")
	}

	eng, err := schemaEngine(dsFactory, nil, &fakeWiringCachePort{getSchema: nil}, time.Minute)
	if err != nil {
		t.Fatalf("schemaEngine: unexpected error: %v", err)
	}

	// A connection whose declared type is not a known datasource type. The
	// schemacompat ConnectorFactory.Build rejects it offline.
	conn := &model.Connection{ConfigName: "db1", Type: model.DBType("NOT_A_REAL_DB"), Host: "h"}
	ctx := schemacompat.WithResolvedConnections(context.Background(), []*model.Connection{conn})

	_, err = eng.DiscoverSchema(ctx, mustWiringTenant(t), "db1")
	if err == nil {
		t.Fatal("connector factory not wired: discovery succeeded on an unknown datasource type")
	}

	if connectAttempted {
		t.Fatal("offline build invariant broken: discovery connected before rejecting the unknown type")
	}

	// The engine redacts the build failure to a safe CategoryUnavailable error;
	// reaching it at all proves Build ran through the wired registry.
	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("want CategoryUnavailable (redacted build failure) from wired factory, got %v", err)
	}
}
