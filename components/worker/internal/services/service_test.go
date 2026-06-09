package services

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/model/datasource"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// stubEngineRunner is a no-op EngineRunner used to satisfy the mandatory-runner
// wiring guard in tests that only need a non-nil runner.
type stubEngineRunner struct{}

func (stubEngineRunner) RunExtraction(context.Context, engine.TenantContext, string, engine.ExtractionRequest) (engine.ExtractionResult, error) {
	return engine.ExtractionResult{}, nil
}

// stubDataSourceFactory is a no-op datasource factory used to satisfy the
// mandatory-factory wiring guard in tests that only need a non-nil factory.
func stubDataSourceFactory(context.Context, *model.Connection, crypto.Cryptor) (datasource.DataSource, error) {
	return nil, nil
}

// TestUseCase_Validate_RequiresEngineRunner locks the strangler completion plus
// the plugin_crm wiring guard: the legacy generic extraction path is gone, so the
// Engine runner is MANDATORY; and CreateDataSource (still used by plugin_crm)
// dereferences dataSourceFactory unconditionally, so that factory is MANDATORY
// too. Validate must reject either nil dependency at wiring time (fail fast at
// construction) rather than nil-panicking deep in extraction.
func TestUseCase_Validate_RequiresEngineRunner(t *testing.T) {
	t.Run("nil engine runner is rejected", func(t *testing.T) {
		uc := &UseCase{dataSourceFactory: stubDataSourceFactory}

		err := uc.Validate()
		if err == nil {
			t.Fatal("expected Validate to reject a nil EngineRunner, got nil")
		}
	})

	t.Run("nil dataSourceFactory is rejected", func(t *testing.T) {
		uc := &UseCase{EngineRunner: stubEngineRunner{}}

		err := uc.Validate()
		if err == nil {
			t.Fatal("expected Validate to reject a nil dataSourceFactory, got nil")
		}
	})

	t.Run("both dependencies wired passes", func(t *testing.T) {
		uc := &UseCase{EngineRunner: stubEngineRunner{}, dataSourceFactory: stubDataSourceFactory}

		if err := uc.Validate(); err != nil {
			t.Fatalf("expected Validate to pass with EngineRunner and dataSourceFactory, got %v", err)
		}
	})
}

func TestSetStorageEncryptDerivedKey(t *testing.T) {
	t.Run("sets derived key", func(t *testing.T) {
		uc := &UseCase{}
		key := []byte("12345678901234567890123456789012")
		uc.SetStorageEncryptDerivedKey(key)

		if len(uc.storageEncryptDerivedKey) != 32 {
			t.Errorf("storageEncryptDerivedKey length = %d, want 32", len(uc.storageEncryptDerivedKey))
		}
	})

	t.Run("sets nil key", func(t *testing.T) {
		uc := &UseCase{}
		uc.SetStorageEncryptDerivedKey(nil)

		if uc.storageEncryptDerivedKey != nil {
			t.Error("storageEncryptDerivedKey should be nil")
		}
	})
}

func TestSetCRMSecrets(t *testing.T) {
	tests := []struct {
		name       string
		encryptKey string
		hashKey    string
	}{
		{
			name:       "sets both keys",
			encryptKey: "crm-encrypt-key",
			hashKey:    "crm-hash-key",
		},
		{
			name:       "sets empty keys",
			encryptKey: "",
			hashKey:    "",
		},
		{
			name:       "overwrites previously set keys",
			encryptKey: "new-encrypt-key",
			hashKey:    "new-hash-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &UseCase{}

			// Set initial values for the overwrite test
			if tt.name == "overwrites previously set keys" {
				uc.SetCRMSecrets("old-encrypt", "old-hash")
			}

			uc.SetCRMSecrets(tt.encryptKey, tt.hashKey)

			if uc.crmEncryptSecretKey != tt.encryptKey {
				t.Errorf("crmEncryptSecretKey = %q, want %q", uc.crmEncryptSecretKey, tt.encryptKey)
			}
			if uc.crmHashSecretKey != tt.hashKey {
				t.Errorf("crmHashSecretKey = %q, want %q", uc.crmHashSecretKey, tt.hashKey)
			}
		})
	}
}

func TestSetDataSourceFactory(t *testing.T) {
	t.Run("sets factory function", func(t *testing.T) {
		uc := &UseCase{}

		factory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
			return nil, nil
		}

		uc.SetDataSourceFactory(factory)

		if uc.dataSourceFactory == nil {
			t.Fatal("dataSourceFactory should not be nil after SetDataSourceFactory")
		}
	})

	t.Run("factory is callable after being set", func(t *testing.T) {
		uc := &UseCase{}

		called := false
		factory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
			called = true
			return nil, nil
		}

		uc.SetDataSourceFactory(factory)

		_, _ = uc.dataSourceFactory(context.Background(), nil, nil)
		if !called {
			t.Fatal("factory function was not called")
		}
	})
}

func TestCreateDataSource(t *testing.T) {
	t.Run("delegates to factory with valid connection", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDS := datasource.NewMockDataSource(ctrl)
		mockCryptor := crypto.NewMockCryptor(ctrl)

		conn := &model.Connection{
			ID:           uuid.New(),
			ConfigName:   "test-conn",
			Host:         "localhost",
			Port:         5432,
			DatabaseName: "testdb",
			Username:     "user",
			Type:         model.TypePostgreSQL,
		}

		uc := &UseCase{
			Cryptor: mockCryptor,
		}

		factoryCalled := false
		uc.SetDataSourceFactory(func(ctx context.Context, c *model.Connection, cr crypto.Cryptor) (datasource.DataSource, error) {
			factoryCalled = true
			if c != conn {
				t.Error("factory received wrong connection")
			}
			if cr != mockCryptor {
				t.Error("factory received wrong cryptor")
			}
			return mockDS, nil
		})

		ctx := testContext()
		ds, err := uc.CreateDataSource(ctx, conn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ds != mockDS {
			t.Error("expected mock datasource to be returned")
		}
		if !factoryCalled {
			t.Error("factory function was not called")
		}
	})

	t.Run("returns error from factory", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCryptor := crypto.NewMockCryptor(ctrl)
		expectedErr := errors.New("factory error")

		uc := &UseCase{
			Cryptor: mockCryptor,
		}

		uc.SetDataSourceFactory(func(ctx context.Context, c *model.Connection, cr crypto.Cryptor) (datasource.DataSource, error) {
			return nil, expectedErr
		})

		ctx := testContext()
		ds, err := uc.CreateDataSource(ctx, &model.Connection{})
		if err == nil {
			t.Fatal("expected error from factory")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
		if ds != nil {
			t.Error("expected nil datasource on error")
		}
	})

	t.Run("panics with nil factory", func(t *testing.T) {
		uc := &UseCase{}

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic when factory is nil")
			}
		}()

		ctx := testContext()
		_, _ = uc.CreateDataSource(ctx, &model.Connection{})
	})
}
