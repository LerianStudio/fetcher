//go:build go1.18
// +build go1.18

package connection

import (
	"encoding/json"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/tests/fuzz/shared/generators"
)

func FuzzConnectionInputParsing(f *testing.F) {
	seeds := [][]byte{
		generators.GenerateConnectionInputSeed(),
		[]byte(`{"configName":"a","type":"POSTGRESQL","host":"h","port":1,"databaseName":"d","username":"u","password":"p"}`),
		[]byte(`{"configName":"test-config_123","type":"ORACLE","host":"db.example.com","port":1521,"databaseName":"ORCL","username":"admin","password":"secret123"}`),
		[]byte(`{"configName":"","type":"","host":"","port":0,"databaseName":"","username":"","password":""}`),
		[]byte(`{}`),
		[]byte(`{"configName":"x"}`),
		[]byte(`{"type":"INVALID_TYPE"}`),
		[]byte(`{"port":"not_a_number"}`),
		[]byte(`{"port":-1}`),
		[]byte(`{"port":65536}`),
		[]byte(`{"ssl":{"mode":"require","ca":"cert"}}`),
		[]byte(`{"metadata":{"key":"value"}}`),
		[]byte(`{"unknownField":"value"}`),
	}

	// Add security test seeds (SQL injection, XSS - OWASP A03:2021)
	seeds = append(seeds, generators.GetSecuritySeedBytes()...)

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var input model.ConnectionInput
		err := json.Unmarshal(data, &input)
		if err != nil {
			return
		}
		_ = len(input.ConfigName)
		_ = input.Type
		_ = input.Host
		_ = input.Port
		if input.SSL != nil {
			_ = input.SSL.Mode
			_ = input.SSL.IsEmpty()
		}
		if input.Metadata != nil {
			for k, v := range *input.Metadata {
				_ = k
				_ = v
			}
		}
		_ = input.ToMapWithMask()
	})
}
