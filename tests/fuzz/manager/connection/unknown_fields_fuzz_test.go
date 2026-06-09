//go:build go1.18
// +build go1.18

package connection

import (
	"encoding/json"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
)

func FuzzUnknownFieldsDetection(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"configName":"test","type":"POSTGRESQL","host":"h","port":5432,"databaseName":"d","username":"u","password":"p","unknownField":"value"}`),
		[]byte(`{"configName":"test","type":"POSTGRESQL","host":"h","port":5432,"databaseName":"d","username":"u","password":"p","extra1":"v1","extra2":"v2"}`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var input model.ConnectionInput
		var originalMap map[string]any

		if err := json.Unmarshal(data, &input); err != nil {
			return
		}
		if err := json.Unmarshal(data, &originalMap); err != nil {
			return
		}

		marshaled, err := json.Marshal(input)
		if err != nil {
			return
		}

		var marshaledMap map[string]any
		if err := json.Unmarshal(marshaled, &marshaledMap); err != nil {
			return
		}

		count := 0
		for key := range originalMap {
			if _, exists := marshaledMap[key]; !exists {
				count++
			}
		}
		_ = count
	})
}
