package itestkit

import "testing"

func TestResolveHostHostPort(t *testing.T) {
	t.Parallel()

	t.Run("prefers proxy listen for public endpoint", func(t *testing.T) {
		host, port, err := ResolveHostHostPort("127.0.0.1:15432", "postgres-default:5432")
		if err != nil {
			t.Fatalf("ResolveHostHostPort() error = %v", err)
		}

		if host != "127.0.0.1" || port != 15432 {
			t.Fatalf("ResolveHostHostPort() = %s:%d, want 127.0.0.1:15432", host, port)
		}
	})

	t.Run("falls back to upstream host endpoint", func(t *testing.T) {
		host, port, err := ResolveHostHostPort("", "localhost:5432")
		if err != nil {
			t.Fatalf("ResolveHostHostPort() error = %v", err)
		}

		if host != "localhost" || port != 5432 {
			t.Fatalf("ResolveHostHostPort() = %s:%d, want localhost:5432", host, port)
		}
	})
}

func TestResolveContainerHostPort(t *testing.T) {
	t.Parallel()

	t.Run("prefers in-network proxy endpoint", func(t *testing.T) {
		host, port, err := ResolveContainerHostPort("toxiproxy:10000", "postgres-default", 5432, "localhost:35432")
		if err != nil {
			t.Fatalf("ResolveContainerHostPort() error = %v", err)
		}

		if host != "toxiproxy" || port != 10000 {
			t.Fatalf("ResolveContainerHostPort() = %s:%d, want toxiproxy:10000", host, port)
		}
	})

	t.Run("uses shared-network alias without proxy", func(t *testing.T) {
		host, port, err := ResolveContainerHostPort("", "mongodb-default", 27017, "localhost:37017")
		if err != nil {
			t.Fatalf("ResolveContainerHostPort() error = %v", err)
		}

		if host != "mongodb-default" || port != 27017 {
			t.Fatalf("ResolveContainerHostPort() = %s:%d, want mongodb-default:27017", host, port)
		}
	})
}
