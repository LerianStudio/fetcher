//go:build go1.18
// +build go1.18

package connection

import (
	"testing"

	"github.com/google/uuid"
)

func FuzzUUIDParsing(f *testing.F) {
	f.Add(uuid.New().String())
	f.Add("00000000-0000-0000-0000-000000000000")
	f.Add("ffffffff-ffff-ffff-ffff-ffffffffffff")
	f.Add("")
	f.Add("not-a-uuid")
	f.Add("12345678-1234-1234-1234-123456789012")
	f.Add("12345678-1234-1234-1234-12345678901")
	f.Add("gggggggg-gggg-gggg-gggg-gggggggggggg")
	f.Add("12345678123412341234123456789012")
	f.Add(" 12345678-1234-1234-1234-123456789012")

	f.Fuzz(func(t *testing.T, uuidStr string) {
		parsedUUID, err := uuid.Parse(uuidStr)
		if err == nil {
			_ = parsedUUID.String()
			_ = parsedUUID == uuid.Nil
		}
	})
}
