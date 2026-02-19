package bootstrap

import (
	"testing"
)

func TestServer_ServerAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "standard address",
			address: "localhost:8080",
		},
		{
			name:    "wildcard address",
			address: "0.0.0.0:3000",
		},
		{
			name:    "empty address",
			address: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				serverAddress: tt.address,
			}

			got := s.ServerAddress()
			if got != tt.address {
				t.Errorf("ServerAddress() = %q, want %q", got, tt.address)
			}
		})
	}
}
