// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"context"
	"errors"
	"testing"
)

// TestEngineWithoutConnectionStore_DegradationSurface pins the real cost of
// omitting the optional ConnectionStore port. requireConnectionStore runs
// BEFORE tenant validation in every operation that guards on it, so the gate
// answers first and zero-value arguments are enough to observe it.
//
// The published port table is the contract an embedder plans against, so the
// split below is the contract: everything that resolves a connection — CRUD,
// connection testing, schema discovery/validation, extraction planning and
// execution — fails without the port. Only the three operations that touch no
// connection record keep working. A new operation that guards on the store must
// be added here, and the port table updated with it.
func TestEngineWithoutConnectionStore_DegradationSurface(t *testing.T) {
	t.Parallel()

	const notConfigured = "connection store is not configured"

	eng, err := New(WithConnectorRegistry(fakeConnectorRegistry{}))
	if err != nil {
		t.Fatalf("construct engine without a connection store: %v", err)
	}

	gated := map[string]func(context.Context) error{
		"CreateConnection": func(ctx context.Context) error {
			_, err := eng.CreateConnection(ctx, TenantContext{}, ConnectionInput{})
			return err
		},
		"GetConnection": func(ctx context.Context) error {
			_, err := eng.GetConnection(ctx, TenantContext{}, "")
			return err
		},
		"GetConnectionByID": func(ctx context.Context) error {
			_, err := eng.GetConnectionByID(ctx, TenantContext{}, "")
			return err
		},
		"ListConnections": func(ctx context.Context) error {
			_, err := eng.ListConnections(ctx, TenantContext{})
			return err
		},
		"ListConnectionsPaged": func(ctx context.Context) error {
			_, err := eng.ListConnectionsPaged(ctx, TenantContext{}, ConnectionListParams{})
			return err
		},
		"UpdateConnection": func(ctx context.Context) error {
			_, err := eng.UpdateConnection(ctx, TenantContext{}, "", ConnectionPatch{})
			return err
		},
		"UpdateConnectionByID": func(ctx context.Context) error {
			_, err := eng.UpdateConnectionByID(ctx, TenantContext{}, "", ConnectionDescriptor{}, ConnectionPatch{})
			return err
		},
		"DeleteConnection": func(ctx context.Context) error {
			return eng.DeleteConnection(ctx, TenantContext{}, "")
		},
		"DeleteConnectionByID": func(ctx context.Context) error {
			return eng.DeleteConnectionByID(ctx, TenantContext{}, "")
		},
		"TestConnection": func(ctx context.Context) error {
			_, err := eng.TestConnection(ctx, TenantContext{}, "")
			return err
		},
		"DiscoverSchema": func(ctx context.Context) error {
			_, err := eng.DiscoverSchema(ctx, TenantContext{}, "")
			return err
		},
		"DiscoverSchemaFresh": func(ctx context.Context) error {
			_, err := eng.DiscoverSchemaFresh(ctx, TenantContext{}, "")
			return err
		},
		"ValidateSchema": func(ctx context.Context) error {
			_, err := eng.ValidateSchema(ctx, TenantContext{}, SchemaValidationRequest{})
			return err
		},
		"PlanExtraction": func(ctx context.Context) error {
			_, err := eng.PlanExtraction(ctx, TenantContext{}, ExtractionRequest{})
			return err
		},
		"ExecuteExtraction": func(ctx context.Context) error {
			_, err := eng.ExecuteExtraction(ctx, ExtractionPlan{})
			return err
		},
	}

	for name, operation := range gated {
		name, operation := name, operation
		t.Run("gated/"+name, func(t *testing.T) {
			t.Parallel()

			err := operation(context.Background())
			if err == nil {
				t.Fatalf("%s must fail without a ConnectionStore", name)
			}

			var engineErr *EngineError
			if !errors.As(err, &engineErr) {
				t.Fatalf("%s without a ConnectionStore returned %T, want *EngineError", name, err)
			}

			if engineErr.Category != CategoryValidation || engineErr.Message != notConfigured {
				t.Fatalf(
					"%s without a ConnectionStore = [%s] %q, want [%s] %q",
					name, engineErr.Category, engineErr.Message, CategoryValidation, notConfigured,
				)
			}
		})
	}

	// The survivors: Limits reads construction-time configuration,
	// AuthorizeConnectionAccess validates a tenant scope and performs no I/O, and
	// CheckActiveExecutions consults the ActiveExecutionChecker port instead.
	t.Run("ungated/Limits", func(t *testing.T) {
		t.Parallel()

		if eng.Limits().MaxDatasources == 0 {
			t.Fatal("Limits must return the applied defaults without a ConnectionStore")
		}
	})

	survivors := map[string]func(context.Context) error{
		"AuthorizeConnectionAccess": func(ctx context.Context) error {
			return eng.AuthorizeConnectionAccess(ctx, TenantContext{TenantID: "tenant-1"})
		},
		"CheckActiveExecutions": func(ctx context.Context) error {
			return eng.CheckActiveExecutions(ctx, TenantContext{TenantID: "tenant-1"}, "connection-1")
		},
	}

	for name, operation := range survivors {
		name, operation := name, operation
		t.Run("ungated/"+name, func(t *testing.T) {
			t.Parallel()

			if err := operation(context.Background()); err != nil {
				t.Fatalf("%s must not depend on a ConnectionStore, got %v", name, err)
			}
		})
	}
}
