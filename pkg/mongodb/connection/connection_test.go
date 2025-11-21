package connection

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newValidConnectionEntity() *Connection {
	return &Connection{
		OrganizationID:    uuid.New(),
		ConfigName:        "  Valid Config  ",
		Type:              ConnectionTypePostgreSQL,
		Host:              "  db.internal.local  ",
		Port:              5432,
		DatabaseName:      "main",
		Username:          "fetcher",
		PasswordEncrypted: "secret",
	}
}

func cloneConnectionEntity(src *Connection) *Connection {
	if src == nil {
		return nil
	}
	cp := *src
	if src.SSL != nil {
		sslCopy := *src.SSL
		cp.SSL = &sslCopy
	}
	return &cp
}

func TestConnectionTypeIsValid(t *testing.T) {
	if !ConnectionTypePostgreSQL.IsValid() {
		t.Fatalf("expected ConnectionTypePostgreSQL to be valid")
	}
	if ConnectionType("unknown").IsValid() {
		t.Fatalf("expected unknown type to be invalid")
	}
}

func TestConnectionValidateForCreate(t *testing.T) {
	t.Run("success trims fields", func(t *testing.T) {
		conn := newValidConnectionEntity()
		if err := conn.ValidateForCreate(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if conn.ConfigName != "Valid Config" {
			t.Fatalf("expected trimmed config name, got %q", conn.ConfigName)
		}
		if conn.Host != "db.internal.local" {
			t.Fatalf("expected trimmed host, got %q", conn.Host)
		}
	})

	tests := []struct {
		name string
		conn *Connection
		err  string
	}{
		{"nil connection", nil, "connection entity is required"},
		{"missing organization", func() *Connection { c := newValidConnectionEntity(); c.OrganizationID = uuid.Nil; return c }(), "organization ID is required"},
		{"missing config", func() *Connection { c := newValidConnectionEntity(); c.ConfigName = "   "; return c }(), "config name is required"},
		{"invalid type", func() *Connection { c := newValidConnectionEntity(); c.Type = ConnectionType("invalid"); return c }(), "invalid connection type"},
		{"invalid port", func() *Connection { c := newValidConnectionEntity(); c.Port = 0; return c }(), "port must be a positive integer"},
		{"missing host", func() *Connection { c := newValidConnectionEntity(); c.Host = "  "; return c }(), "host is required"},
		{"missing password", func() *Connection { c := newValidConnectionEntity(); c.PasswordEncrypted = ""; return c }(), "password_encrypted is required"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.conn == nil {
				var nilConn *Connection
				err = nilConn.ValidateForCreate()
			} else {
				err = cloneConnectionEntity(tt.conn).ValidateForCreate()
			}
			if err == nil || err.Error() != tt.err {
				t.Fatalf("expected error %q, got %v", tt.err, err)
			}
		})
	}
}

func TestConnectionValidateForUpdate(t *testing.T) {
	conn := newValidConnectionEntity()
	conn.ID = uuid.New()
	if err := conn.ValidateForUpdate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	connWithoutID := newValidConnectionEntity()
	if err := connWithoutID.ValidateForUpdate(); err == nil || err.Error() != "connection ID is required" {
		t.Fatalf("expected connection ID error, got %v", err)
	}
}

func TestConnectionMongoDBModelFromEntity(t *testing.T) {
	t.Run("nil entity", func(t *testing.T) {
		model := &ConnectionMongoDBModel{}
		if err := model.FromEntity(nil); err == nil {
			t.Fatalf("expected error for nil entity")
		}
	})

	t.Run("generates defaults", func(t *testing.T) {
		originalGen := generateConnectionUUID
		defer func() { generateConnectionUUID = originalGen }()

		expectedID := uuid.New()
		generateConnectionUUID = func() (uuid.UUID, error) {
			return expectedID, nil
		}

		conn := newValidConnectionEntity()
		conn.ID = uuid.Nil
		conn.CreatedAt = time.Time{}
		conn.UpdatedAt = time.Time{}

		model := &ConnectionMongoDBModel{}
		if err := model.FromEntity(conn); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.ID != expectedID {
			t.Fatalf("expected generated ID %v, got %v", expectedID, model.ID)
		}
		if conn.CreatedAt.IsZero() || conn.UpdatedAt.IsZero() {
			t.Fatalf("expected timestamps to be set")
		}
	})

	t.Run("propagates uuid generation error", func(t *testing.T) {
		originalGen := generateConnectionUUID
		defer func() { generateConnectionUUID = originalGen }()

		generateConnectionUUID = func() (uuid.UUID, error) {
			return uuid.Nil, errors.New("uuid error")
		}

		conn := newValidConnectionEntity()
		conn.ID = uuid.Nil
		model := &ConnectionMongoDBModel{}
		if err := model.FromEntity(conn); err == nil || err.Error() != "uuid error" {
			t.Fatalf("expected uuid error, got %v", err)
		}
	})

	t.Run("preserves provided values", func(t *testing.T) {
		now := time.Now().Add(-time.Hour)
		conn := newValidConnectionEntity()
		conn.ID = uuid.New()
		conn.CreatedAt = now
		conn.UpdatedAt = now

		model := &ConnectionMongoDBModel{}
		if err := model.FromEntity(conn); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.CreatedAt != now || model.UpdatedAt != now {
			t.Fatalf("expected timestamps to be preserved")
		}
		if conn.ID != model.ID {
			t.Fatalf("expected IDs to match")
		}
	})
}

func TestConnectionMongoDBModelToEntity(t *testing.T) {
	t.Run("nil model", func(t *testing.T) {
		var model *ConnectionMongoDBModel
		if model.ToEntity() != nil {
			t.Fatalf("expected nil entity for nil model")
		}
	})

	deletedAt := time.Now()
	model := &ConnectionMongoDBModel{
		ID:                uuid.New(),
		OrganizationID:    uuid.New(),
		ConfigName:        "cfg",
		Type:              ConnectionTypeMongoDB,
		Host:              "localhost",
		Port:              27017,
		DatabaseName:      "db",
		Username:          "user",
		PasswordEncrypted: "pwd",
		SSL:               &SSLConfig{Mode: "require"},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		DeletedAt:         &deletedAt,
	}

	entity := model.ToEntity()
	if entity == nil {
		t.Fatalf("expected entity")
	}
	if entity.ID != model.ID || entity.OrganizationID != model.OrganizationID {
		t.Fatalf("expected fields to be copied")
	}
	if entity.SSL == nil || entity.SSL.Mode != "require" {
		t.Fatalf("expected SSL to be copied")
	}
}
