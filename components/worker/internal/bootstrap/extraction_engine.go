// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package bootstrap

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/components/worker/internal/services"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/engine"
	enginecompatdatasource "github.com/LerianStudio/fetcher/pkg/enginecompat/datasource"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
)

// wireEngineRunner builds the embedded extraction Engine, sets it as the Worker
// UseCase's mandatory EngineRunner, and fails fast at startup if the resulting
// wiring is invalid. After the strangler completion (T-010) the legacy extraction
// path is gone, so a nil runner is a wiring bug that must surface here — not
// nil-panic during extraction. Extracting this keeps InitWorker's complexity down
// while making the bootstrap the sole guarantor that EngineRunner is set.
func wireEngineRunner(service *services.UseCase, dsFactory datasource.DataSourceFactory, cryptor crypto.Cryptor) error {
	extractionEngine, err := newExtractionEngine(dsFactory, cryptor)
	if err != nil {
		return fmt.Errorf("build extraction engine: %w", err)
	}

	service.EngineRunner = newWorkerEngineRunner(extractionEngine)

	if err := service.Validate(); err != nil {
		return fmt.Errorf("worker service wiring invalid: %w", err)
	}

	return nil
}

// extractionConnectorRegistry resolves the single extraction ConnectorFactory for
// any datasource type. Type validation happens inside the factory's Build (from
// the descriptor), so the registry resolves unconditionally rather than
// enumerating every type — mirroring the Manager's schemaConnectorRegistry.
type extractionConnectorRegistry struct {
	factory engine.ConnectorFactory
}

// Connector returns the one extraction ConnectorFactory for any type.
func (r extractionConnectorRegistry) Connector(string) (engine.ConnectorFactory, bool) {
	return r.factory, true
}

// newExtractionEngine builds the embedded Engine that the Worker drives for
// plan-then-execute extraction in DIRECT mode:
//
//   - ConnectorRegistry: the enginecompat datasource ConnectorFactory. The Worker
//     seeds the FULL resolved connection record into each descriptor, so the
//     factory's rich-record path connects with the exact SSL/schema/encrypted
//     credential the Worker resolved (the legacy direct-datasource credential path).
//     The CredentialResolver is a no-op fallback that is never reached on this path.
//   - ConnectionStore: the request-scoped schemacompat store, which returns the
//     connection the Worker already resolved (seeded via WithResolvedConnections)
//     so the Engine never re-resolves and tenant-manager stays out of Engine core.
//
// No ResultSink is wired: DIRECT mode returns the bytes inline and the Worker owns
// encrypt + store + HMAC (ST-02). No ExecutionStore/EventSink: the Worker owns the
// job lifecycle and lib-streaming events.
func newExtractionEngine(dsFactory datasource.DataSourceFactory, cryptor crypto.Cryptor) (*engine.Engine, error) {
	factory := enginecompatdatasource.NewConnectorFactory(
		enginecompatdatasource.DataSourceFactory(dsFactory),
		noopCredentialResolver,
		cryptor,
	)

	registry := extractionConnectorRegistry{factory: factory}

	return engine.New(
		engine.WithConnectorRegistry(registry),
		engine.WithConnectionStore(schemacompat.NewConnectionStore()),
	)
}

// noopCredentialResolver is the unreachable fallback resolver. The Worker always
// seeds the rich connection record into the descriptor, so the enginecompat
// connector uses its rich-record path and never calls the resolver. It returns an
// empty password defensively rather than panicking, should the rich record ever be
// absent (which would then fail at connect time, not here).
func noopCredentialResolver(context.Context, engine.ConnectionDescriptor) (string, error) {
	return "", nil
}

// workerEngineRunner implements services.EngineRunner. It plans then executes an
// extraction request in DIRECT mode through the embedded Engine. The resolved
// connections travel in the ctx (seeded by the UseCase via
// schemacompat.WithResolvedConnections), which the Engine's ConnectionStore reads.
type workerEngineRunner struct {
	engine *engine.Engine
}

// newWorkerEngineRunner wires the runner over a constructed extraction Engine.
func newWorkerEngineRunner(eng *engine.Engine) *workerEngineRunner {
	return &workerEngineRunner{engine: eng}
}

// RunExtraction plans then executes the request for the tenant in DIRECT mode and
// returns the inline ExtractionResult. PlanExtraction applies strict schema
// validation (the T-009-deferred target) against the host-normalized snapshot;
// ExecuteExtraction reads the rows through the connector. The plan's Mode is left
// ModeAuto, which resolves to DIRECT because no ResultSink is configured.
func (r *workerEngineRunner) RunExtraction(
	ctx context.Context,
	tenant engine.TenantContext,
	jobID string,
	request engine.ExtractionRequest,
) (engine.ExtractionResult, error) {
	plan, err := r.engine.PlanExtraction(ctx, tenant, request)
	if err != nil {
		return engine.ExtractionResult{}, err
	}

	plan.RequestID = jobID

	return r.engine.ExecuteExtraction(ctx, plan)
}
