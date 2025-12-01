package connection

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	http "github.com/LerianStudio/fetcher/pkg/net/http"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	libMongo "github.com/LerianStudio/lib-commons/v2/commons/mongo"
	"github.com/google/uuid"
	"github.com/tryvium-travels/memongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
)

var (
	connectionTestMongoServer *memongo.Server
	connectionTestMongoConn   *libMongo.MongoConnection
)

const connectionTestDatabaseName = "fetcher_connection_test"

func TestMain(m *testing.M) {
	server, err := memongo.Start("6.0.6")
	if err != nil {
		log.Fatalf("failed to start memongo: %v", err)
	}
	connectionTestMongoServer = server
	connectionTestMongoConn = &libMongo.MongoConnection{
		ConnectionStringSource: server.URI(),
		Database:               connectionTestDatabaseName,
		Logger:                 &libLog.GoLogger{Level: libLog.ErrorLevel},
		MaxPoolSize:            5,
	}

	code := m.Run()

	server.Stop()
	os.Exit(code)
}

func newConnectionRepository(t *testing.T) *ConnectionMongoDBRepository {
	t.Helper()
	clearConnectionsCollection(t)
	repo, err := NewConnectionMongoDBRepository(context.Background(), connectionTestMongoConn)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("failed to ensure indexes: %v", err)
	}
	return repo
}

func clearConnectionsCollection(t *testing.T) {
	t.Helper()
	if connectionTestMongoConn == nil {
		t.Fatalf("mongo connection not initialized")
	}
	client, err := connectionTestMongoConn.GetDB(context.Background())
	if err != nil {
		t.Fatalf("failed to get db: %v", err)
	}
	coll := client.Database(strings.ToLower(connectionTestMongoConn.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	if err := coll.Drop(context.Background()); err != nil {
		var cmdErr mongo.CommandError
		if errors.As(err, &cmdErr) && cmdErr.Code == 26 {
			return
		}
		t.Fatalf("failed to drop collection: %v", err)
	}
}

func connectionFixture() *model.Connection {
	now := time.Now().UTC()
	return &model.Connection{
		ID:                   uuid.New(),
		OrganizationID:       uuid.New(),
		ConfigName:           "primary-db",
		Type:                 model.TypePostgreSQL,
		Host:                 "localhost",
		Port:                 5432,
		DatabaseName:         "db",
		Username:             "user",
		PasswordEncrypted:    "encrypted",
		EncryptionKeyVersion: "v1",
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func connectionWithSSLFixture() *model.Connection {
	conn := connectionFixture()
	conn.SSL = &model.SSLConfig{
		Mode: "require",
		CA:   "ca",
		Cert: "cert",
		Key:  "key",
	}
	return conn
}

func createConnection(t *testing.T, repo *ConnectionMongoDBRepository, conn *model.Connection) *model.Connection {
	t.Helper()
	created, err := repo.Create(context.Background(), conn)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	return created
}

func stubConnectionSpanAttributes(t *testing.T, retErr error) {
	t.Helper()
	original := setSpanAttributesFromStruct
	setSpanAttributesFromStruct = func(span *trace.Span, key string, valueStruct any) error {
		return retErr
	}
	t.Cleanup(func() {
		setSpanAttributesFromStruct = original
	})
}

type fakeConnectionMongoConnection struct {
	err error
}

func (f *fakeConnectionMongoConnection) GetDB(ctx context.Context) (*mongo.Client, error) {
	return nil, f.err
}

func TestConnectionMongoDBRepository_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newConnectionRepository(t)
		conn := connectionFixture()
		created, err := repo.Create(context.Background(), conn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.ID != conn.ID || created.ConfigName != conn.ConfigName {
			t.Fatalf("expected returned connection to match input")
		}
	})

	t.Run("nil payload returns error", func(t *testing.T) {
		repo := newConnectionRepository(t)
		if _, err := repo.Create(context.Background(), nil); err == nil {
			t.Fatalf("expected error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
		}
	})

	t.Run("duplicate config name returns conflict", func(t *testing.T) {
		repo := newConnectionRepository(t)
		base := createConnection(t, repo, connectionFixture())

		dup := connectionFixture()
		dup.OrganizationID = base.OrganizationID
		dup.ConfigName = base.ConfigName

		if _, err := repo.Create(context.Background(), dup); err == nil {
			t.Fatalf("expected conflict error")
		} else {
			var conflict pkg.EntityConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("expected conflict error, got %v", err)
			}
		}
	})

	t.Run("span attribute errors are ignored", func(t *testing.T) {
		repo := newConnectionRepository(t)
		stubConnectionSpanAttributes(t, errors.New("span failure"))
		if _, err := repo.Create(context.Background(), connectionFixture()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeConnectionMongoConnection{err: errors.New("db down")},
			Database:   connectionTestDatabaseName,
		}
		if _, err := repo.Create(context.Background(), connectionFixture()); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
			if internal.Err == nil || internal.Err.Error() != "db down" {
				t.Fatalf("expected wrapped db error, got %v", internal.Err)
			}
		}
	})
}

func TestConnectionMongoDBRepository_Update(t *testing.T) {
	t.Run("updates fields including ssl", func(t *testing.T) {
		repo := newConnectionRepository(t)
		created := createConnection(t, repo, connectionFixture())

		created.ConfigName = "updated-name"
		created.Host = "new-host"
		created.Port = 3306
		created.DatabaseName = "otherdb"
		created.Username = "other"
		created.PasswordEncrypted = "new-encrypted"
		created.EncryptionKeyVersion = "v2"
		created.SSL = &model.SSLConfig{Mode: "require", CA: "ca"}

		updated, err := repo.Update(context.Background(), created)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.ConfigName != "updated-name" || updated.Host != "new-host" || updated.Port != 3306 {
			t.Fatalf("expected fields updated")
		}
		if updated.SSL == nil || updated.SSL.Mode != "require" || updated.SSL.CA != "ca" {
			t.Fatalf("expected ssl persisted")
		}
	})

	t.Run("clears ssl when nil", func(t *testing.T) {
		repo := newConnectionRepository(t)
		created := createConnection(t, repo, connectionWithSSLFixture())
		created.SSL = nil
		updated, err := repo.Update(context.Background(), created)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.SSL != nil {
			t.Fatalf("expected ssl cleared")
		}
	})

	t.Run("nil payload returns error", func(t *testing.T) {
		repo := newConnectionRepository(t)
		if _, err := repo.Update(context.Background(), nil); err == nil {
			t.Fatalf("expected error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
		}
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		repo := newConnectionRepository(t)
		conn := connectionFixture()
		updated, err := repo.Update(context.Background(), conn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated != nil {
			t.Fatalf("expected nil when connection not found")
		}
	})

	t.Run("duplicate name returns conflict", func(t *testing.T) {
		repo := newConnectionRepository(t)
		first := createConnection(t, repo, connectionFixture())
		second := connectionFixture()
		second.OrganizationID = first.OrganizationID
		second.ConfigName = "secondary-db"
		createConnection(t, repo, second)

		second.ConfigName = first.ConfigName
		if _, err := repo.Update(context.Background(), second); err == nil {
			t.Fatalf("expected conflict")
		} else {
			var conflict pkg.EntityConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("expected conflict error, got %v", err)
			}
		}
	})

	t.Run("span attribute errors are ignored", func(t *testing.T) {
		repo := newConnectionRepository(t)
		created := createConnection(t, repo, connectionFixture())
		stubConnectionSpanAttributes(t, errors.New("span failure"))
		created.ConfigName = "another-name"
		if _, err := repo.Update(context.Background(), created); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeConnectionMongoConnection{err: errors.New("db down")},
			Database:   connectionTestDatabaseName,
		}
		conn := connectionFixture()
		conn.ID = uuid.New()
		conn.OrganizationID = uuid.New()
		if _, err := repo.Update(context.Background(), conn); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
			if internal.Err == nil || internal.Err.Error() != "db down" {
				t.Fatalf("expected wrapped db error, got %v", internal.Err)
			}
		}
	})
}

func TestConnectionMongoDBRepository_Delete(t *testing.T) {
	t.Run("soft deletes connection", func(t *testing.T) {
		repo := newConnectionRepository(t)
		conn := createConnection(t, repo, connectionFixture())
		deletedAt := time.Now().UTC()
		if err := repo.Delete(context.Background(), conn.ID, conn.OrganizationID, deletedAt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		client, err := connectionTestMongoConn.GetDB(context.Background())
		if err != nil {
			t.Fatalf("failed to get db: %v", err)
		}
		coll := client.Database(strings.ToLower(connectionTestMongoConn.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
		var record ConnectionMongoDBModel
		if err := coll.FindOne(context.Background(), bson.M{"_id": conn.ID}).Decode(&record); err != nil {
			t.Fatalf("failed to fetch deleted record: %v", err)
		}
		if record.DeletedAt == nil {
			t.Fatalf("expected deleted_at set")
		}
	})

	t.Run("not found returns entity not found", func(t *testing.T) {
		repo := newConnectionRepository(t)
		if err := repo.Delete(context.Background(), uuid.New(), uuid.New(), time.Now()); err == nil {
			t.Fatalf("expected not found")
		} else {
			var notFound pkg.EntityNotFoundError
			if !errors.As(err, &notFound) {
				t.Fatalf("expected not found error, got %v", err)
			}
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeConnectionMongoConnection{err: errors.New("db down")},
			Database:   connectionTestDatabaseName,
		}
		if err := repo.Delete(context.Background(), uuid.New(), uuid.New(), time.Now()); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
			if internal.Err == nil || internal.Err.Error() != "db down" {
				t.Fatalf("expected wrapped db error, got %v", internal.Err)
			}
		}
	})
}

func TestConnectionMongoDBRepository_FindByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newConnectionRepository(t)
		created := createConnection(t, repo, connectionFixture())
		found, err := repo.FindByID(context.Background(), created.ID, created.OrganizationID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected matching id")
		}
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		repo := newConnectionRepository(t)
		found, err := repo.FindByID(context.Background(), uuid.New(), uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil when connection not found")
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeConnectionMongoConnection{err: errors.New("db down")},
			Database:   connectionTestDatabaseName,
		}
		if _, err := repo.FindByID(context.Background(), uuid.New(), uuid.New()); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
			if internal.Err == nil || internal.Err.Error() != "db down" {
				t.Fatalf("expected wrapped db error, got %v", internal.Err)
			}
		}
	})
}

func TestConnectionMongoDBRepository_FindByOrganizationAndName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newConnectionRepository(t)
		created := createConnection(t, repo, connectionFixture())
		found, err := repo.FindByOrganizationAndName(context.Background(), created.OrganizationID, created.ConfigName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected matching id")
		}
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		repo := newConnectionRepository(t)
		found, err := repo.FindByOrganizationAndName(context.Background(), uuid.New(), "missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil when connection not found")
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeConnectionMongoConnection{err: errors.New("db down")},
			Database:   connectionTestDatabaseName,
		}
		if _, err := repo.FindByOrganizationAndName(context.Background(), uuid.New(), "name"); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
			if internal.Err == nil || internal.Err.Error() != "db down" {
				t.Fatalf("expected wrapped db error, got %v", internal.Err)
			}
		}
	})
}

func TestConnectionMongoDBRepository_FindByOrganizationAndDatabaseName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newConnectionRepository(t)
		created := createConnection(t, repo, connectionFixture())
		found, err := repo.FindByOrganizationAndDatabaseName(context.Background(), created.OrganizationID, created.DatabaseName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected matching id")
		}
	})

	t.Run("empty database name returns validation error", func(t *testing.T) {
		repo := newConnectionRepository(t)
		if _, err := repo.FindByOrganizationAndDatabaseName(context.Background(), uuid.New(), ""); err == nil {
			t.Fatalf("expected error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
		}
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		repo := newConnectionRepository(t)
		found, err := repo.FindByOrganizationAndDatabaseName(context.Background(), uuid.New(), "missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil when connection not found")
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeConnectionMongoConnection{err: errors.New("db down")},
			Database:   connectionTestDatabaseName,
		}
		if _, err := repo.FindByOrganizationAndDatabaseName(context.Background(), uuid.New(), "db"); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
			if internal.Err == nil || internal.Err.Error() != "db down" {
				t.Fatalf("expected wrapped db error, got %v", internal.Err)
			}
		}
	})
}

func TestConnectionMongoDBRepository_List(t *testing.T) {
	t.Run("returns paginated results ordered by created_at desc", func(t *testing.T) {
		repo := newConnectionRepository(t)
		org := uuid.New()

		older := connectionFixture()
		older.OrganizationID = org
		older.ConfigName = "older"
		older.CreatedAt = time.Now().Add(-2 * time.Hour)
		older.UpdatedAt = older.CreatedAt
		createConnection(t, repo, older)

		newer := connectionFixture()
		newer.OrganizationID = org
		newer.ConfigName = "newer"
		newer.CreatedAt = time.Now().Add(-1 * time.Hour)
		newer.UpdatedAt = newer.CreatedAt
		createConnection(t, repo, newer)

		otherOrg := connectionFixture()
		otherOrg.OrganizationID = uuid.New()
		otherOrg.ConfigName = "other-org"
		createConnection(t, repo, otherOrg)

		filters := http.QueryHeader{Limit: 1, Page: 1}
		list, err := repo.List(context.Background(), org, filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 || list[0].ConfigName != "newer" {
			t.Fatalf("expected newest connection for org")
		}
	})

	t.Run("filters by created_at range", func(t *testing.T) {
		repo := newConnectionRepository(t)
		org := uuid.New()

		outOfRange := connectionFixture()
		outOfRange.OrganizationID = org
		outOfRange.ConfigName = "too-old"
		outOfRange.CreatedAt = time.Now().Add(-48 * time.Hour)
		outOfRange.UpdatedAt = outOfRange.CreatedAt
		createConnection(t, repo, outOfRange)

		inRange := connectionFixture()
		inRange.OrganizationID = org
		inRange.ConfigName = "in-range"
		inRange.CreatedAt = time.Now().Add(-1 * time.Hour)
		inRange.UpdatedAt = inRange.CreatedAt
		createConnection(t, repo, inRange)

		start := inRange.CreatedAt.Add(-30 * time.Minute)
		end := inRange.CreatedAt.Add(30 * time.Minute)
		filters := http.QueryHeader{
			Limit:     5,
			Page:      1,
			StartDate: start,
			EndDate:   end,
		}

		list, err := repo.List(context.Background(), org, filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 || list[0].ConfigName != "in-range" {
			t.Fatalf("expected only in-range connection")
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		repo := &ConnectionMongoDBRepository{
			connection: &fakeConnectionMongoConnection{err: errors.New("db down")},
			Database:   connectionTestDatabaseName,
		}
		if _, err := repo.List(context.Background(), uuid.New(), http.QueryHeader{}); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
			if internal.Err == nil || internal.Err.Error() != "db down" {
				t.Fatalf("expected wrapped db error, got %v", internal.Err)
			}
		}
	})
}
