// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package connectioncompat_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/model"
	connPort "github.com/LerianStudio/fetcher/pkg/ports/connection"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func richConnection(configName string) *model.Connection {
	id := uuid.New()
	schema := "public"
	meta := map[string]any{"source": "plugin_crm"}
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := time.Date(2026, 1, 2, 6, 7, 8, 0, time.UTC)

	return &model.Connection{
		ID:                   id,
		ProductName:          "plugin_crm",
		ConfigName:           configName,
		Type:                 model.TypePostgreSQL,
		Host:                 "db.internal",
		Port:                 5432,
		DatabaseName:         "ledger",
		Schema:               &schema,
		Username:             "svc",
		PasswordEncrypted:    "ciphertext",
		EncryptionKeyVersion: "v3",
		SSL:                  &model.SSLConfig{Mode: "verify-full", CA: "ca-pem", Cert: "cert-pem", Key: "key-pem"},
		Metadata:             &meta,
		CreatedAt:            created,
		UpdatedAt:            updated,
	}
}

// TestConnectionStore_RoundTripsRichModelLosslessly proves the rich Manager
// record survives a full Create -> FindConnection round-trip through the Engine
// ConnectionStore port WITHOUT field loss: ProductName, full SSL (CA/Cert/Key),
// uuid identity, metadata, timestamps, and EncryptionKeyVersion all preserved.
func TestConnectionStore_RoundTripsRichModelLosslessly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := connPort.NewMockRepository(ctrl)

	var persisted *model.Connection

	gomock.InOrder(
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, c *model.Connection) (*model.Connection, error) {
				persisted = c
				return c, nil
			}),
		// Read-back through the port: returns the persisted record.
		repo.EXPECT().FindByName(gomock.Any(), "pg-main").DoAndReturn(
			func(_ context.Context, _ string) (*model.Connection, error) {
				return persisted, nil
			}),
	)

	store := connectioncompat.NewConnectionStore(repo)
	tenant := mustTenant(t, "tenant-a")
	ctx := context.Background()

	original := richConnection("pg-main")
	descriptor := connectioncompat.DescriptorFromConnection(original)

	require.NoError(t, store.Create(ctx, tenant, descriptor, nil))
	require.NotNil(t, persisted)
	assert.Equal(t, original, persisted, "rich model must be persisted to the repo unchanged")

	got, found, err := store.FindConnection(ctx, tenant, "pg-main")
	require.NoError(t, err)
	require.True(t, found)

	roundTripped := connectioncompat.ConnectionFromDescriptor(got)
	require.NotNil(t, roundTripped)
	assert.Equal(t, original, roundTripped, "rich model must round-trip through the opaque seam losslessly")
}

// TestConnectionStore_CreateRequiresPackedRecord proves the adapter refuses a
// descriptor that carries no opaque host record (a programming error), mapping
// it to the Engine's validation rule rather than persisting an empty connection.
func TestConnectionStore_CreateRequiresPackedRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := connPort.NewMockRepository(ctrl)
	store := connectioncompat.NewConnectionStore(repo)

	// A bare descriptor with no HostAttributes record.
	err := store.Create(context.Background(), mustTenant(t, "tenant-a"), engine.ConnectionDescriptor{ConfigName: "pg-main"}, nil)
	require.Error(t, err)

	var engErr *engine.EngineError
	require.True(t, errors.As(err, &engErr))
	assert.Equal(t, engine.CategoryValidation, engErr.Category)
}

// TestConnectionStore_DeleteSoftDeletesViaUUID proves the Engine's
// config-name-keyed Delete maps to the Manager's UUID-keyed SOFT delete
// (deleted_at), never a hard delete.
func TestConnectionStore_DeleteSoftDeletesViaUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := connPort.NewMockRepository(ctrl)
	existing := richConnection("pg-main")

	repo.EXPECT().FindByName(gomock.Any(), "pg-main").Return(existing, nil)
	repo.EXPECT().Delete(gomock.Any(), existing.ID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, deletedAt time.Time) error {
			assert.False(t, deletedAt.IsZero(), "soft-delete must carry a deleted_at timestamp")
			return nil
		})

	store := connectioncompat.NewConnectionStore(repo)
	require.NoError(t, store.Delete(context.Background(), mustTenant(t, "tenant-a"), "pg-main"))
}

// TestConnectionStore_DeleteMissingMapsToNotFound proves a missing connection
// surfaces the Engine's not-found category.
func TestConnectionStore_DeleteMissingMapsToNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := connPort.NewMockRepository(ctrl)
	repo.EXPECT().FindByName(gomock.Any(), "ghost").Return(nil, nil)

	store := connectioncompat.NewConnectionStore(repo)
	err := store.Delete(context.Background(), mustTenant(t, "tenant-a"), "ghost")

	var engErr *engine.EngineError
	require.True(t, errors.As(err, &engErr))
	assert.Equal(t, engine.CategoryNotFound, engErr.Category)
}

// TestConnectionStore_UpdateMapsToRepoUpdate proves the Engine's
// config-name-keyed Update lands on the Manager's UUID-keyed repo Update with
// the rich record reconstructed from the opaque payload (no field loss).
func TestConnectionStore_UpdateMapsToRepoUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := connPort.NewMockRepository(ctrl)
	updatedRich := richConnection("pg-main")
	updatedRich.Host = "new-host.internal"

	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, c *model.Connection) (*model.Connection, error) {
			assert.Equal(t, "new-host.internal", c.Host)
			assert.Equal(t, updatedRich.ID, c.ID, "UUID identity must be preserved through the seam")
			return c, nil
		})

	store := connectioncompat.NewConnectionStore(repo)
	desc := connectioncompat.DescriptorFromConnection(updatedRich)

	require.NoError(t, store.Update(context.Background(), mustTenant(t, "tenant-a"), desc, nil))
}

// TestConnectionStore_ListPacksRichRecords proves the port-completeness List
// fetches the tenant set through the repo and packs each rich record into its
// descriptor's opaque payload, so an embedded host can unpack the full set.
func TestConnectionStore_ListPacksRichRecords(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := connPort.NewMockRepository(ctrl)
	a := richConnection("pg-a")
	b := richConnection("pg-b")

	repo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*model.Connection{a, b}, int64(2), nil)

	store := connectioncompat.NewConnectionStore(repo)
	descriptors, err := store.List(context.Background(), mustTenant(t, "tenant-a"))
	require.NoError(t, err)
	require.Len(t, descriptors, 2)

	assert.Equal(t, a, connectioncompat.ConnectionFromDescriptor(descriptors[0]))
	assert.Equal(t, b, connectioncompat.ConnectionFromDescriptor(descriptors[1]))
}

// TestConnectionStore_FindAndUpdatePropagateRepoErrors proves repository errors
// on the read and update paths surface unchanged (not masked), preserving the
// host's existing error-mapping contract.
func TestConnectionStore_FindAndUpdatePropagateRepoErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoErr := errors.New("mongo down")

	repo := connPort.NewMockRepository(ctrl)
	repo.EXPECT().FindByName(gomock.Any(), "pg-main").Return(nil, repoErr)

	store := connectioncompat.NewConnectionStore(repo)

	_, _, err := store.FindConnection(context.Background(), mustTenant(t, "tenant-a"), "pg-main")
	require.ErrorIs(t, err, repoErr)

	repo2 := connPort.NewMockRepository(ctrl)
	repo2.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, repoErr)
	store2 := connectioncompat.NewConnectionStore(repo2)
	require.ErrorIs(t,
		store2.Update(context.Background(), mustTenant(t, "tenant-a"), connectioncompat.DescriptorFromConnection(richConnection("pg-main")), nil),
		repoErr)
}

// TestConnectionStore_NilRepoYieldsNilAdapter proves a nil repo yields a nil
// adapter so the Engine treats it as "no connection store".
func TestConnectionStore_NilRepoYieldsNilAdapter(t *testing.T) {
	assert.Nil(t, connectioncompat.NewConnectionStore(nil))
}

// TestConnectionStore_UpdateRequiresPackedRecord proves Update refuses a bare
// descriptor (no opaque host record), mapping it to the Engine's validation rule.
func TestConnectionStore_UpdateRequiresPackedRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := connectioncompat.NewConnectionStore(connPort.NewMockRepository(ctrl))
	err := store.Update(context.Background(), mustTenant(t, "tenant-a"), engine.ConnectionDescriptor{ConfigName: "x"}, nil)

	var engErr *engine.EngineError
	require.True(t, errors.As(err, &engErr))
	assert.Equal(t, engine.CategoryValidation, engErr.Category)
}

// TestConnectionFromDescriptor_GuardsAgainstMissingOrWrongRecord proves the
// unpacker returns nil for a bare descriptor or one whose opaque slot holds a
// non-record value, rather than panicking on a bad type assertion.
func TestConnectionFromDescriptor_GuardsAgainstMissingOrWrongRecord(t *testing.T) {
	assert.Nil(t, connectioncompat.ConnectionFromDescriptor(engine.ConnectionDescriptor{}))
	assert.Nil(t, connectioncompat.ConnectionFromDescriptor(engine.ConnectionDescriptor{
		HostAttributes: map[string]any{"fetcher.connection.record": "not-a-connection"},
	}))
}

// TestConnectionStore_CreateAndDeletePropagateRepoErrors proves the write and
// delete paths surface repository errors unchanged.
func TestConnectionStore_CreateAndDeletePropagateRepoErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoErr := errors.New("mongo down")

	createRepo := connPort.NewMockRepository(ctrl)
	createRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, repoErr)
	require.ErrorIs(t,
		connectioncompat.NewConnectionStore(createRepo).Create(
			context.Background(), mustTenant(t, "tenant-a"),
			connectioncompat.DescriptorFromConnection(richConnection("pg-main")), nil),
		repoErr)

	deleteRepo := connPort.NewMockRepository(ctrl)
	deleteRepo.EXPECT().FindByName(gomock.Any(), "pg-main").Return(nil, repoErr)
	require.ErrorIs(t,
		connectioncompat.NewConnectionStore(deleteRepo).Delete(
			context.Background(), mustTenant(t, "tenant-a"), "pg-main"),
		repoErr)
}

// TestConnectionStore_FindReturnsNotFoundAsAbsent proves a nil connection from
// the repo is reported as found=false (the Engine maps it to not-found).
func TestConnectionStore_FindReturnsNotFoundAsAbsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := connPort.NewMockRepository(ctrl)
	repo.EXPECT().FindByName(gomock.Any(), "ghost").Return(nil, nil)

	store := connectioncompat.NewConnectionStore(repo)
	_, found, err := store.FindConnection(context.Background(), mustTenant(t, "tenant-a"), "ghost")
	require.NoError(t, err)
	assert.False(t, found)
}

func mustTenant(t *testing.T, id string) engine.TenantContext {
	t.Helper()

	tenant, err := engine.NewTenantContext(id)
	require.NoError(t, err)

	return tenant
}
