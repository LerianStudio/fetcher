package connection

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	libMongo "github.com/LerianStudio/lib-commons/v2/commons/mongo"
	"github.com/google/uuid"
	"github.com/tryvium-travels/memongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
)

var (
	testMongoServer *memongo.Server
	testMongoConn   *libMongo.MongoConnection
)

const testDatabaseName = "fetcher_test"

func TestMain(m *testing.M) {
	server, err := memongo.Start("6.0.6")
	if err != nil {
		log.Fatalf("failed to start memongo: %v", err)
	}
	testMongoServer = server
	testMongoConn = &libMongo.MongoConnection{
		ConnectionStringSource: server.URI(),
		Database:               testDatabaseName,
		Logger:                 &libLog.GoLogger{Level: libLog.ErrorLevel},
		MaxPoolSize:            5,
	}

	code := m.Run()

	server.Stop()
	os.Exit(code)
}

func newRepository(t *testing.T) *ConnectionMongoDBRepository {
	t.Helper()
	clearConnectionsCollection(t)
	repo, err := NewConnectionMongoDBRepository(context.Background(), testMongoConn)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	return repo
}

func clearConnectionsCollection(t *testing.T) {
	t.Helper()
	if testMongoConn == nil {
		t.Fatalf("mongo connection not initialized")
	}
	client, err := testMongoConn.GetDB(context.Background())
	if err != nil {
		t.Fatalf("failed to get db: %v", err)
	}
	coll := client.Database(strings.ToLower(testMongoConn.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	if err := coll.Drop(context.Background()); err != nil {
		var cmdErr mongo.CommandError
		if errors.As(err, &cmdErr) && cmdErr.Code == 26 {
			return
		}
		t.Fatalf("failed to drop collection: %v", err)
	}
}

func connectionFixture() *Connection {
	return &Connection{
		OrganizationID:    uuid.New(),
		ConfigName:        fmt.Sprintf("cfg-%s", uuid.NewString()),
		Type:              ConnectionTypePostgreSQL,
		Host:              "db.test.internal",
		Port:              5432,
		DatabaseName:      "db",
		Username:          "user",
		PasswordEncrypted: "secret",
	}
}

func createConnection(t *testing.T, repo *ConnectionMongoDBRepository, conn *Connection) *Connection {
	t.Helper()
	created, err := repo.Create(context.Background(), conn)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	return created
}

func stubSpanAttributes(t *testing.T, retErr error) {
	original := setSpanAttributesFromStruct
	setSpanAttributesFromStruct = func(span *trace.Span, key string, valueStruct any) error {
		return retErr
	}
	t.Cleanup(func() {
		setSpanAttributesFromStruct = original
	})
}

type fakeMongoConnection struct {
	err error
}

func (f *fakeMongoConnection) GetDB(ctx context.Context) (*mongo.Client, error) {
	return nil, f.err
}

func TestConnectionMongoDBRepository_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newRepository(t)
		conn := connectionFixture()
		created, err := repo.Create(context.Background(), conn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.ID == uuid.Nil {
			t.Fatalf("expected generated ID")
		}
	})

	t.Run("validation error", func(t *testing.T) {
		repo := newRepository(t)
		conn := connectionFixture()
		conn.Port = 0
		if _, err := repo.Create(context.Background(), conn); err == nil {
			t.Fatalf("expected validation error")
		}
	})

	t.Run("nil payload", func(t *testing.T) {
		repo := newRepository(t)
		if _, err := repo.Create(context.Background(), nil); err == nil {
			t.Fatalf("expected error for nil payload")
		}
	})

	t.Run("span attribute error ignored", func(t *testing.T) {
		repo := newRepository(t)
		stubSpanAttributes(t, errors.New("span failure"))
		if _, err := repo.Create(context.Background(), connectionFixture()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("duplicate insert fails", func(t *testing.T) {
		repo := newRepository(t)
		conn := connectionFixture()
		created := createConnection(t, repo, conn)
		// Reuse same struct to attempt duplicate insert.
		if _, err := repo.Create(context.Background(), created); err == nil {
			t.Fatalf("expected duplicate key error")
		}
	})

	t.Run("from entity error surfaces", func(t *testing.T) {
		repo := newRepository(t)
		original := generateConnectionUUID
		defer func() { generateConnectionUUID = original }()
		generateConnectionUUID = func() (uuid.UUID, error) {
			return uuid.Nil, errors.New("uuid failure")
		}
		if _, err := repo.Create(context.Background(), connectionFixture()); err == nil || err.Error() != "uuid failure" {
			t.Fatalf("expected uuid failure, got %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeMongoConnection{err: errors.New("db down")},
			Database:   testDatabaseName,
		}
		if _, err := repo.Create(context.Background(), connectionFixture()); err == nil || err.Error() != "db down" {
			t.Fatalf("expected db error, got %v", err)
		}
	})
}

func TestConnectionMongoDBRepository_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newRepository(t)
		created := createConnection(t, repo, connectionFixture())
		created.ConfigName = "  Updated  "
		created.Host = "  updated.internal  "
		updated, err := repo.Update(context.Background(), created)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.ConfigName != "Updated" {
			t.Fatalf("expected trimmed config name")
		}
		if updated.Host != "updated.internal" {
			t.Fatalf("expected trimmed host")
		}
	})

	t.Run("nil payload", func(t *testing.T) {
		repo := newRepository(t)
		if _, err := repo.Update(context.Background(), nil); err == nil {
			t.Fatalf("expected error for nil payload")
		}
	})

	t.Run("validation error", func(t *testing.T) {
		repo := newRepository(t)
		created := createConnection(t, repo, connectionFixture())
		created.Host = ""
		if _, err := repo.Update(context.Background(), created); err == nil {
			t.Fatalf("expected validation error")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newRepository(t)
		missing := connectionFixture()
		missing.ID = uuid.New()
		if _, err := repo.Update(context.Background(), missing); err == nil {
			t.Fatalf("expected not found error")
		}
	})

	t.Run("span attribute error ignored", func(t *testing.T) {
		repo := newRepository(t)
		created := createConnection(t, repo, connectionFixture())
		stubSpanAttributes(t, errors.New("span failure"))
		created.Host = "updated"
		if _, err := repo.Update(context.Background(), created); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeMongoConnection{err: errors.New("db down")},
			Database:   testDatabaseName,
		}
		conn := connectionFixture()
		conn.ID = uuid.New()
		if _, err := repo.Update(context.Background(), conn); err == nil || err.Error() != "db down" {
			t.Fatalf("expected db error, got %v", err)
		}
	})
}

func TestConnectionMongoDBRepository_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newRepository(t)
		created := createConnection(t, repo, connectionFixture())
		if err := repo.Delete(context.Background(), created.ID, created.OrganizationID, time.Time{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Ensure deleted record is no longer returned by FindByID.
		if _, err := repo.FindByID(context.Background(), created.ID, created.OrganizationID); err == nil {
			t.Fatalf("expected find to fail after delete")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newRepository(t)
		if err := repo.Delete(context.Background(), uuid.New(), uuid.New(), time.Time{}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeMongoConnection{err: errors.New("db down")},
			Database:   testDatabaseName,
		}
		if err := repo.Delete(context.Background(), uuid.New(), uuid.New(), time.Now()); err == nil || err.Error() != "db down" {
			t.Fatalf("expected db error, got %v", err)
		}
	})

	t.Run("update failure surfaces", func(t *testing.T) {
		repo := newRepository(t)
		created := createConnection(t, repo, connectionFixture())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := repo.Delete(ctx, created.ID, created.OrganizationID, time.Now()); err == nil {
			t.Fatalf("expected context error")
		}
	})
}

func TestConnectionMongoDBRepository_FindByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newRepository(t)
		created := createConnection(t, repo, connectionFixture())
		found, err := repo.FindByID(context.Background(), created.ID, created.OrganizationID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected IDs to match")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newRepository(t)
		if _, err := repo.FindByID(context.Background(), uuid.New(), uuid.New()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeMongoConnection{err: errors.New("db down")},
			Database:   testDatabaseName,
		}
		if _, err := repo.FindByID(context.Background(), uuid.New(), uuid.New()); err == nil || err.Error() != "db down" {
			t.Fatalf("expected db error, got %v", err)
		}
	})
}

func TestConnectionMongoDBRepository_FindByOrganizationAndName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newRepository(t)
		created := createConnection(t, repo, connectionFixture())
		found, err := repo.FindByOrganizationAndName(context.Background(), created.OrganizationID, "  "+created.ConfigName+"  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected IDs to match")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newRepository(t)
		if _, err := repo.FindByOrganizationAndName(context.Background(), uuid.New(), "missing"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeMongoConnection{err: errors.New("db down")},
			Database:   testDatabaseName,
		}
		if _, err := repo.FindByOrganizationAndName(context.Background(), uuid.New(), "cfg"); err == nil || err.Error() != "db down" {
			t.Fatalf("expected db error, got %v", err)
		}
	})
}

func TestConnectionMongoDBRepository_List(t *testing.T) {
	t.Run("applies filters and pagination", func(t *testing.T) {
		repo := newRepository(t)
		orgID := uuid.New()
		otherOrg := uuid.New()

		alpha := connectionFixture()
		alpha.OrganizationID = orgID
		alpha.ConfigName = "alpha"
		alpha.Type = ConnectionTypeMySQL
		createConnection(t, repo, alpha)

		beta := connectionFixture()
		beta.OrganizationID = orgID
		beta.ConfigName = "beta"
		beta.Type = ConnectionTypePostgreSQL
		createdBeta := createConnection(t, repo, beta)

		gamma := connectionFixture()
		gamma.OrganizationID = otherOrg
		gamma.ConfigName = "gamma"
		createConnection(t, repo, gamma)

		filters := &ListFilter{
			OrganizationID: orgID,
			ConfigName:     "  beta  ",
			Types:          []ConnectionType{ConnectionTypePostgreSQL},
			Limit:          1,
			Page:           1,
			SortOrder:      constant.Asc,
		}

		list, err := repo.List(context.Background(), filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 || list[0].ID != createdBeta.ID {
			t.Fatalf("expected beta connection")
		}
	})

	t.Run("include deleted", func(t *testing.T) {
		repo := newRepository(t)
		conn := createConnection(t, repo, connectionFixture())
		if err := repo.Delete(context.Background(), conn.ID, conn.OrganizationID, time.Time{}); err != nil {
			t.Fatalf("unexpected delete error: %v", err)
		}
		filters := &ListFilter{OrganizationID: conn.OrganizationID, IncludeDeleted: true}
		list, err := repo.List(context.Background(), filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 || list[0].DeletedAt == nil {
			t.Fatalf("expected deleted record")
		}
	})

	t.Run("limit boundaries and nil filters", func(t *testing.T) {
		repo := newRepository(t)
		createConnection(t, repo, connectionFixture())
		// limit <= 0 uses default
		if _, err := repo.List(context.Background(), &ListFilter{OrganizationID: uuid.New(), Limit: -1}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// limit > max is clamped
		if _, err := repo.List(context.Background(), &ListFilter{OrganizationID: uuid.New(), Limit: 1000}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := repo.List(context.Background(), nil); err != nil {
			t.Fatalf("unexpected error listing with nil filters: %v", err)
		}
	})

	t.Run("span attribute error ignored", func(t *testing.T) {
		repo := newRepository(t)
		createConnection(t, repo, connectionFixture())
		stubSpanAttributes(t, errors.New("span failure"))
		if _, err := repo.List(context.Background(), &ListFilter{OrganizationID: uuid.New()}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeMongoConnection{err: errors.New("db down")},
			Database:   testDatabaseName,
		}
		if _, err := repo.List(context.Background(), &ListFilter{OrganizationID: uuid.New()}); err == nil || err.Error() != "db down" {
			t.Fatalf("expected db error, got %v", err)
		}
	})

	t.Run("find failure surfaces", func(t *testing.T) {
		repo := newRepository(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := repo.List(ctx, &ListFilter{OrganizationID: uuid.New()}); err == nil {
			t.Fatalf("expected error")
		}
	})

}
