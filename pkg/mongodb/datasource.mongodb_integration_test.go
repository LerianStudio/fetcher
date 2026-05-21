//go:build integration
// +build integration

package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/LerianStudio/lib-observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// setupMongoContainer starts a MongoDB container for integration testing
func setupMongoContainer(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mongo:7",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForLog("Waiting for connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, "", err
	}

	port, err := container.MappedPort(ctx, "27017")
	if err != nil {
		return nil, "", err
	}

	uri := "mongodb://" + host + ":" + port.Port()
	return container, uri, nil
}

// seedTestData inserts test data into MongoDB
func seedTestData(ctx context.Context, uri, dbName string) error {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	defer client.Disconnect(ctx)

	db := client.Database(dbName)

	// Create users collection with test data
	usersCollection := db.Collection("users")
	_, err = usersCollection.InsertMany(ctx, []interface{}{
		bson.M{"_id": 1, "name": "Alice", "email": "alice@example.com", "age": 30, "status": "active"},
		bson.M{"_id": 2, "name": "Bob", "email": "bob@example.com", "age": 25, "status": "active"},
		bson.M{"_id": 3, "name": "Charlie", "email": "charlie@example.com", "age": 35, "status": "inactive"},
		bson.M{"_id": 4, "name": "Diana", "email": "diana@example.com", "age": 28, "status": "active"},
	})
	if err != nil {
		return err
	}

	// Create products collection with test data
	productsCollection := db.Collection("products")
	_, err = productsCollection.InsertMany(ctx, []interface{}{
		bson.M{"_id": 1, "name": "Product A", "price": 100.0, "stock": 50},
		bson.M{"_id": 2, "name": "Product B", "price": 200.0, "stock": 30},
		bson.M{"_id": 3, "name": "Product C", "price": 150.0, "stock": 0},
	})
	if err != nil {
		return err
	}

	return nil
}

func TestNewDataSourceRepository_Integration(t *testing.T) {
	ctx := context.Background()

	container, uri, err := setupMongoContainer(ctx)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	// Wait a bit for MongoDB to be fully ready
	time.Sleep(2 * time.Second)

	logger := observability.NewLoggerFromContext(ctx)
	dbName := "test_db"

	t.Run("successfully creates data source repository", func(t *testing.T) {
		ds, err := NewDataSourceRepository(uri, dbName, logger)
		require.NoError(t, err)
		assert.NotNil(t, ds)
		assert.Equal(t, dbName, ds.Database)

		// Clean up
		err = ds.CloseConnection(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns error for invalid URI", func(t *testing.T) {
		ds, err := NewDataSourceRepository("invalid://uri", dbName, logger)
		assert.Error(t, err)
		assert.Nil(t, ds)
	})
}

func TestCloseConnection_Integration(t *testing.T) {
	ctx := context.Background()

	container, uri, err := setupMongoContainer(ctx)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logger := observability.NewLoggerFromContext(ctx)
	dbName := "test_db"

	t.Run("successfully closes connection", func(t *testing.T) {
		ds, err := NewDataSourceRepository(uri, dbName, logger)
		require.NoError(t, err)

		err = ds.CloseConnection(ctx)
		assert.NoError(t, err)
	})

	t.Run("handles multiple close calls gracefully", func(t *testing.T) {
		ds, err := NewDataSourceRepository(uri, dbName, logger)
		require.NoError(t, err)

		err = ds.CloseConnection(ctx)
		assert.NoError(t, err)

		// Second close should not error
		err = ds.CloseConnection(ctx)
		assert.NoError(t, err)
	})
}

func TestQuery_Integration(t *testing.T) {
	ctx := context.Background()

	container, uri, err := setupMongoContainer(ctx)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logger := observability.NewLoggerFromContext(ctx)
	dbName := "test_db"

	err = seedTestData(ctx, uri, dbName)
	require.NoError(t, err)

	ds, err := NewDataSourceRepository(uri, dbName, logger)
	require.NoError(t, err)
	defer ds.CloseConnection(ctx)

	t.Run("queries all documents with wildcard", func(t *testing.T) {
		results, err := ds.Query(ctx, "users", []string{"*"}, map[string][]any{})
		require.NoError(t, err)
		assert.Len(t, results, 4)
	})

	t.Run("queries with specific fields", func(t *testing.T) {
		results, err := ds.Query(ctx, "users", []string{"name", "email"}, map[string][]any{})
		require.NoError(t, err)
		assert.Len(t, results, 4)

		// Check that only requested fields are present (plus _id which is always included)
		for _, result := range results {
			assert.Contains(t, result, "name")
			assert.Contains(t, result, "email")
		}
	})

	t.Run("queries with single value filter", func(t *testing.T) {
		results, err := ds.Query(ctx, "users", []string{"*"}, map[string][]any{
			"status": {"active"},
		})
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("queries with multiple value filter", func(t *testing.T) {
		results, err := ds.Query(ctx, "users", []string{"*"}, map[string][]any{
			"name": {"Alice", "Bob"},
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("queries empty collection", func(t *testing.T) {
		results, err := ds.Query(ctx, "empty_collection", []string{"*"}, map[string][]any{})
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("queries with no matching filter", func(t *testing.T) {
		results, err := ds.Query(ctx, "users", []string{"*"}, map[string][]any{
			"status": {"deleted"},
		})
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestGetDatabaseSchema_Integration(t *testing.T) {
	ctx := context.Background()

	container, uri, err := setupMongoContainer(ctx)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logger := observability.NewLoggerFromContext(ctx)
	dbName := "test_db"

	err = seedTestData(ctx, uri, dbName)
	require.NoError(t, err)

	ds, err := NewDataSourceRepository(uri, dbName, logger)
	require.NoError(t, err)
	defer ds.CloseConnection(ctx)

	t.Run("retrieves schema for all collections", func(t *testing.T) {
		schema, err := ds.GetDatabaseSchema(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(schema), 2, "should have at least users and products collections")

		// Find users collection schema
		var usersSchema *CollectionSchema
		for i := range schema {
			if schema[i].CollectionName == "users" {
				usersSchema = &schema[i]
				break
			}
		}

		require.NotNil(t, usersSchema, "users collection should be in schema")
		assert.NotEmpty(t, usersSchema.Fields)

		// Check that common fields are discovered
		fieldNames := make(map[string]bool)
		for _, field := range usersSchema.Fields {
			fieldNames[field.Name] = true
		}

		assert.True(t, fieldNames["_id"], "should discover _id field")
		assert.True(t, fieldNames["name"], "should discover name field")
		assert.True(t, fieldNames["email"], "should discover email field")
	})

	t.Run("infers correct data types", func(t *testing.T) {
		schema, err := ds.GetDatabaseSchema(ctx)
		require.NoError(t, err)

		var usersSchema *CollectionSchema
		for i := range schema {
			if schema[i].CollectionName == "users" {
				usersSchema = &schema[i]
				break
			}
		}

		require.NotNil(t, usersSchema)

		// Check data types
		fieldTypes := make(map[string]string)
		for _, field := range usersSchema.Fields {
			fieldTypes[field.Name] = field.DataType
		}

		// _id is int in our test data
		assert.Contains(t, []string{"number", "unknown"}, fieldTypes["_id"])
		assert.Equal(t, "string", fieldTypes["name"])
		assert.Equal(t, "string", fieldTypes["email"])
		assert.Contains(t, []string{"number", "unknown"}, fieldTypes["age"])
	})
}

func TestPingMongo_Integration(t *testing.T) {
	ctx := context.Background()

	container, uri, err := setupMongoContainer(ctx)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	time.Sleep(2 * time.Second)

	logger := observability.NewLoggerFromContext(ctx)
	dbName := "test_db"

	t.Run("pings successfully with valid connection", func(t *testing.T) {
		ds, err := NewDataSourceRepository(uri, dbName, logger)
		require.NoError(t, err)
		defer ds.CloseConnection(ctx)

		err = PingMongo(ctx, ds.connection, DefaultPingTimeout)
		assert.NoError(t, err)
	})

	t.Run("pings successfully with custom timeout", func(t *testing.T) {
		ds, err := NewDataSourceRepository(uri, dbName, logger)
		require.NoError(t, err)
		defer ds.CloseConnection(ctx)

		err = PingMongo(ctx, ds.connection, 10*time.Second)
		assert.NoError(t, err)
	})

	t.Run("uses default timeout when zero", func(t *testing.T) {
		ds, err := NewDataSourceRepository(uri, dbName, logger)
		require.NoError(t, err)
		defer ds.CloseConnection(ctx)

		err = PingMongo(ctx, ds.connection, 0)
		assert.NoError(t, err)
	})
}
