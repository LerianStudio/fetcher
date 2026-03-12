package datasource

import (
	"context"
	"testing"
	"time"

	cryptoPkg "github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	modeldatasource "github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewDataSourceFromConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		conn          *model.Connection
		cryptor       func(t *testing.T) cryptoPkg.Cryptor
		assertOutcome func(t *testing.T, got modeldatasource.DataSource, err error)
	}{
		{
			name:    "rejects nil connection",
			conn:    nil,
			cryptor: func(t *testing.T) cryptoPkg.Cryptor { return newMockCryptor(t) },
			assertOutcome: func(t *testing.T, got modeldatasource.DataSource, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "connection cannot be nil")
				assert.Nil(t, got)
			},
		},
		{
			name:    "rejects nil cryptor",
			conn:    testConnection(model.TypePostgreSQL),
			cryptor: func(t *testing.T) cryptoPkg.Cryptor { return nil },
			assertOutcome: func(t *testing.T, got modeldatasource.DataSource, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "cryptor cannot be nil")
				assert.Nil(t, got)
			},
		},
		{
			name:    "rejects unsupported database type",
			conn:    testConnection(model.DBType("UNSUPPORTED")),
			cryptor: func(t *testing.T) cryptoPkg.Cryptor { return newMockCryptor(t) },
			assertOutcome: func(t *testing.T, got modeldatasource.DataSource, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "unsupported database type: UNSUPPORTED")
				assert.Nil(t, got)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewDataSourceFromConnection(context.Background(), tt.conn, tt.cryptor(t), nil)
			tt.assertOutcome(t, got, err)
		})
	}
}

func testConnection(dbType model.DBType) *model.Connection {
	return &model.Connection{
		ID:                uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		OrganizationID:    uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		ConfigName:        "analytics",
		Type:              dbType,
		Host:              "db.internal",
		Port:              5432,
		DatabaseName:      "warehouse",
		Username:          "fetcher",
		PasswordEncrypted: "enc-password",
		CreatedAt:         time.Unix(1700000000, 0).UTC(),
		UpdatedAt:         time.Unix(1700000000, 0).UTC(),
	}
}

func newMockCryptor(t *testing.T) cryptoPkg.Cryptor {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	return cryptoPkg.NewMockCryptor(ctrl)
}
