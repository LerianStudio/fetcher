//go:build go1.18
// +build go1.18

package connection

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/tests/fuzz/shared/mocks"
)

func FuzzConnectionValidation(f *testing.F) {
	f.Add("test-product", "", "", "", 0, "", "", "")
	f.Add("test-product", "ab", "POSTGRESQL", "h", 1, "d", "u", "p")
	f.Add("test-product", "abc", "POSTGRESQL", "h", 1, "d", "u", "p")
	f.Add("", "abc", "POSTGRESQL", "h", 1, "d", "u", "p")
	f.Add("my-product", string(make([]byte, 100)), "POSTGRESQL", "h", 1, "d", "u", "p")
	f.Add("my-product", string(make([]byte, 101)), "POSTGRESQL", "h", 1, "d", "u", "p")
	f.Add("my-product", "test@db", "POSTGRESQL", "h", 1, "d", "u", "p")
	f.Add("my-product", "test_db-123", "POSTGRESQL", "h", 1, "d", "u", "p")
	f.Add("my-product", "test", "INVALID", "h", 1, "d", "u", "p")
	f.Add("my-product", "test", "postgresql", "h", 1, "d", "u", "p")
	f.Add("my-product", "test", "POSTGRESQL", "h", 0, "d", "u", "p")
	f.Add("my-product", "test", "POSTGRESQL", "h", -1, "d", "u", "p")
	f.Add("my-product", "test", "POSTGRESQL", "h", 65535, "d", "u", "p")
	f.Add("my-product", "test", "POSTGRESQL", "h", 65536, "d", "u", "p")

	f.Fuzz(func(t *testing.T, productName, configName, typ, host string, port int, dbName, username, password string) {
		ctx := context.Background()
		mockCryptor := &mocks.MockCryptor{}

		conn, err := model.NewConnection(ctx, mockCryptor, productName, configName, typ, host, port, dbName, nil, username, password, nil, nil, nil, nil, nil)
		if err != nil {
			return
		}
		if conn != nil {
			_ = conn.IsValid()
			_ = conn.ToMapWithMask()
		}
	})
}

func FuzzDBTypeValidation(f *testing.F) {
	f.Add("ORACLE")
	f.Add("SQL_SERVER")
	f.Add("POSTGRESQL")
	f.Add("MONGODB")
	f.Add("MYSQL")
	f.Add("")
	f.Add("oracle")
	f.Add("PostgreSQL")
	f.Add("INVALID")

	f.Fuzz(func(t *testing.T, typeStr string) {
		dbType, err := model.NewTypeFromString(typeStr)
		if err == nil {
			_ = dbType.IsValid()
		}
	})
}
