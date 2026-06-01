// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"context"
	"errors"
	"testing"
)

// The fakes below are zero-behavior test doubles that exist only to satisfy
// the Engine port interfaces at construction time. ST-T002-02 validates
// CONSTRUCTOR behavior only; no operation is invoked on these collaborators,
// so there are no interactions to record. They are defined in the test file so
// pkg/engine stays importable without MongoDB/Redis/RabbitMQ/S3/SeaweedFS/Fiber
// or any database package (T-001 boundary).

type fakeConnectorRegistry struct{}

func (fakeConnectorRegistry) Connector(string) (Connector, bool) { return nil, false }

type fakeCredentialProtector struct{}

func (fakeCredentialProtector) Protect(context.Context, TenantContext, []byte) ([]byte, error) {
	return nil, nil
}

func (fakeCredentialProtector) Reveal(context.Context, TenantContext, []byte) ([]byte, error) {
	return nil, nil
}

type fakeConnectionStore struct{}

func (fakeConnectionStore) FindConnection(context.Context, TenantContext, string) (ConnectionDescriptor, bool, error) {
	return ConnectionDescriptor{}, false, nil
}

type fakeExecutionStore struct{}

func (fakeExecutionStore) SaveExecution(context.Context, TenantContext, ExecutionState) error {
	return nil
}

func (fakeExecutionStore) FindExecution(context.Context, TenantContext, string) (ExecutionState, bool, error) {
	return ExecutionState{}, false, nil
}

type fakeResultSink struct{}

func (fakeResultSink) PersistResult(context.Context, TenantContext, []byte) (ResultReference, error) {
	return ResultReference{}, nil
}

type fakeSchemaCache struct{}

func (fakeSchemaCache) GetSchema(context.Context, TenantContext, string) (SchemaSnapshot, bool, error) {
	return SchemaSnapshot{}, false, nil
}

func (fakeSchemaCache) PutSchema(context.Context, TenantContext, SchemaSnapshot) error {
	return nil
}

type fakeEventSink struct{}

func (fakeEventSink) Emit(context.Context, TenantContext, ExecutionState) error { return nil }

type fakeTenantResolver struct{}

func (fakeTenantResolver) Resolve(context.Context, TenantContext) (TenantContext, error) {
	return TenantContext{}, nil
}

type fakeObservability struct{}

func (fakeObservability) StartSpan(ctx context.Context, _ string) (context.Context, func()) {
	return ctx, func() {}
}

func validBaseOptions() []Option {
	return []Option{
		WithConnectorRegistry(fakeConnectorRegistry{}),
	}
}

func TestNew_RequiresConnectorRegistry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		opts         []Option
		wantErr      bool
		wantCategory ErrorCategory
	}{
		{
			name:         "missing connector registry returns validation error",
			opts:         nil,
			wantErr:      true,
			wantCategory: CategoryValidation,
		},
		{
			name:         "nil connector registry returns validation error",
			opts:         []Option{WithConnectorRegistry(nil)},
			wantErr:      true,
			wantCategory: CategoryValidation,
		},
		{
			name:    "connector registry present succeeds",
			opts:    []Option{WithConnectorRegistry(fakeConnectorRegistry{})},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eng, err := New(tt.opts...)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("New() error = nil, want %s error", tt.wantCategory)
				}

				var engErr *EngineError
				if !errors.As(err, &engErr) {
					t.Fatalf("New() error type = %T, want *EngineError", err)
				}
				if engErr.Category != tt.wantCategory {
					t.Fatalf("New() error category = %q, want %q", engErr.Category, tt.wantCategory)
				}
				if eng != nil {
					t.Fatalf("New() engine = %v, want nil on error", eng)
				}

				return
			}

			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}
			if eng == nil {
				t.Fatalf("New() engine = nil, want usable instance")
			}
		})
	}
}

func TestNew_EncryptedPersistenceRequiresCredentialProtector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		opts         []Option
		wantErr      bool
		wantCategory ErrorCategory
	}{
		{
			name: "encrypted persistence without protector returns validation error",
			opts: []Option{
				WithConnectorRegistry(fakeConnectorRegistry{}),
				WithEncryptedPersistence(true),
			},
			wantErr:      true,
			wantCategory: CategoryValidation,
		},
		{
			name: "encrypted persistence with nil protector returns validation error",
			opts: []Option{
				WithConnectorRegistry(fakeConnectorRegistry{}),
				WithEncryptedPersistence(true),
				WithCredentialProtector(nil),
			},
			wantErr:      true,
			wantCategory: CategoryValidation,
		},
		{
			name: "encrypted persistence with protector succeeds",
			opts: []Option{
				WithConnectorRegistry(fakeConnectorRegistry{}),
				WithEncryptedPersistence(true),
				WithCredentialProtector(fakeCredentialProtector{}),
			},
			wantErr: false,
		},
		{
			name: "encrypted persistence disabled does not require protector",
			opts: []Option{
				WithConnectorRegistry(fakeConnectorRegistry{}),
				WithEncryptedPersistence(false),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eng, err := New(tt.opts...)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("New() error = nil, want %s error", tt.wantCategory)
				}

				var engErr *EngineError
				if !errors.As(err, &engErr) {
					t.Fatalf("New() error type = %T, want *EngineError", err)
				}
				if engErr.Category != tt.wantCategory {
					t.Fatalf("New() error category = %q, want %q", engErr.Category, tt.wantCategory)
				}
				if eng != nil {
					t.Fatalf("New() engine = %v, want nil on error", eng)
				}

				return
			}

			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}
			if eng == nil {
				t.Fatalf("New() engine = nil, want usable instance")
			}
		})
	}
}

func TestNew_RejectsTypedNilConnectorRegistry(t *testing.T) {
	t.Parallel()

	// A typed nil: a non-nil ConnectorRegistry interface value that wraps a nil
	// *fakeConnectorRegistry pointer. A plain `== nil` check would pass it
	// through, so New must use a typed-nil-aware guard and reject it.
	var typedNil *fakeConnectorRegistry

	eng, err := New(WithConnectorRegistry(typedNil))

	if err == nil {
		t.Fatalf("New() error = nil, want %s error for typed-nil connector registry", CategoryValidation)
	}

	var engErr *EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("New() error type = %T, want *EngineError", err)
	}
	if engErr.Category != CategoryValidation {
		t.Fatalf("New() error category = %q, want %q", engErr.Category, CategoryValidation)
	}
	if eng != nil {
		t.Fatalf("New() engine = %v, want nil on error", eng)
	}
}

func TestNew_RejectsTypedNilCredentialProtectorWithEncryptedPersistence(t *testing.T) {
	t.Parallel()

	// A typed nil credential protector must be rejected exactly like a literal
	// nil when encrypted persistence is enabled.
	var typedNil *fakeCredentialProtector

	eng, err := New(
		WithConnectorRegistry(fakeConnectorRegistry{}),
		WithEncryptedPersistence(true),
		WithCredentialProtector(typedNil),
	)

	if err == nil {
		t.Fatalf("New() error = nil, want %s error for typed-nil credential protector", CategoryValidation)
	}

	var engErr *EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("New() error type = %T, want *EngineError", err)
	}
	if engErr.Category != CategoryValidation {
		t.Fatalf("New() error category = %q, want %q", engErr.Category, CategoryValidation)
	}
	if eng != nil {
		t.Fatalf("New() engine = %v, want nil on error", eng)
	}
}

func TestNew_AppliesSafeDefaultLimits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []Option
		want Limits
	}{
		{
			name: "no limits supplied falls back to safe defaults",
			opts: validBaseOptions(),
			want: DefaultLimits(),
		},
		{
			name: "zero-value limits supplied falls back to safe defaults",
			opts: append(validBaseOptions(), WithLimits(Limits{})),
			want: DefaultLimits(),
		},
		{
			name: "custom limits are preserved",
			opts: append(validBaseOptions(), WithLimits(Limits{
				MaxDatasources:         3,
				MaxTablesPerDatasource: 7,
				MaxFieldsPerTable:      11,
				MaxConcurrency:         2,
				Timeout:                DefaultTimeout,
				MaxResultBytes:         1024,
			})),
			want: Limits{
				MaxDatasources:         3,
				MaxTablesPerDatasource: 7,
				MaxFieldsPerTable:      11,
				MaxConcurrency:         2,
				Timeout:                DefaultTimeout,
				MaxResultBytes:         1024,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eng, err := New(tt.opts...)
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}
			if eng == nil {
				t.Fatalf("New() engine = nil, want usable instance")
			}

			got := eng.Limits()
			if got != tt.want {
				t.Fatalf("Engine.Limits() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNew_OptionalPortsAreOptional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []Option
	}{
		{
			name: "only required connector registry",
			opts: validBaseOptions(),
		},
		{
			name: "with result sink",
			opts: append(validBaseOptions(), WithResultSink(fakeResultSink{})),
		},
		{
			name: "with execution store",
			opts: append(validBaseOptions(), WithExecutionStore(fakeExecutionStore{})),
		},
		{
			name: "with connection store",
			opts: append(validBaseOptions(), WithConnectionStore(fakeConnectionStore{})),
		},
		{
			name: "with schema cache",
			opts: append(validBaseOptions(), WithSchemaCache(fakeSchemaCache{})),
		},
		{
			name: "with event sink",
			opts: append(validBaseOptions(), WithEventSink(fakeEventSink{})),
		},
		{
			name: "with tenant resolver",
			opts: append(validBaseOptions(), WithTenantResolver(fakeTenantResolver{})),
		},
		{
			name: "with observability",
			opts: append(validBaseOptions(), WithObservability(fakeObservability{})),
		},
		{
			name: "with every optional port",
			opts: append(validBaseOptions(),
				WithResultSink(fakeResultSink{}),
				WithExecutionStore(fakeExecutionStore{}),
				WithConnectionStore(fakeConnectionStore{}),
				WithSchemaCache(fakeSchemaCache{}),
				WithEventSink(fakeEventSink{}),
				WithTenantResolver(fakeTenantResolver{}),
				WithObservability(fakeObservability{}),
			),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eng, err := New(tt.opts...)
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}
			if eng == nil {
				t.Fatalf("New() engine = nil, want usable instance")
			}
		})
	}
}

func TestNew_ValidFakeCollaboratorsProduceUsableEngine(t *testing.T) {
	t.Parallel()

	eng, err := New(
		WithConnectorRegistry(fakeConnectorRegistry{}),
		WithEncryptedPersistence(true),
		WithCredentialProtector(fakeCredentialProtector{}),
		WithConnectionStore(fakeConnectionStore{}),
		WithExecutionStore(fakeExecutionStore{}),
		WithResultSink(fakeResultSink{}),
		WithSchemaCache(fakeSchemaCache{}),
		WithEventSink(fakeEventSink{}),
		WithTenantResolver(fakeTenantResolver{}),
		WithObservability(fakeObservability{}),
		WithLimits(DefaultLimits()),
	)
	if err != nil {
		t.Fatalf("New() unexpected error = %v", err)
	}
	if eng == nil {
		t.Fatalf("New() engine = nil, want usable instance")
	}

	if eng.Limits() != DefaultLimits() {
		t.Fatalf("Engine.Limits() = %+v, want defaults %+v", eng.Limits(), DefaultLimits())
	}
}
