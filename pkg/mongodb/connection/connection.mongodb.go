package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/fetcher/pkg/net/http"
	portsConnection "github.com/LerianStudio/fetcher/pkg/ports/connection"

	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Repository is an alias for the domain port interface defined in pkg/ports/connection.
type Repository = portsConnection.Repository

// mongoDatabaseProvider defines the interface for obtaining a MongoDB client.
//
//go:generate mockgen --destination=mock_db_provider_test.go --package=connection . mongoDatabaseProvider
type mongoDatabaseProvider interface {
	Client(ctx context.Context) (*mongo.Client, error)
}

const (
	// DefaultInitTimeout is the default timeout for repository initialization
	DefaultInitTimeout = 10 * time.Second
)

// setSpanAttributesFromValue adapts lib-commons v4 span attribute helpers for local stubbing in tests.
var setSpanAttributesFromValue = func(span trace.Span, key string, value any) error {
	return libOpentelemetry.SetSpanAttributesFromValue(span, key, value, nil)
}

// RepositoryConfig holds configuration options for the repository.
type RepositoryConfig struct {
	// InitTimeout is the timeout for repository initialization.
	// Default: DefaultInitTimeout (10s)
	InitTimeout time.Duration
}

// ConnectionMongoDBRepository implements Repository backed by MongoDB.
// NOTE: Span names in this file use the pattern "mongodb.verb_entity" (e.g., "mongodb.create_connection").
// The preferred convention is "mongodb.entity.operation" (e.g., "mongodb.connection.create").
// This inconsistency is tracked for a future rename when dashboards and alerts can be updated.
type ConnectionMongoDBRepository struct {
	connection mongoDatabaseProvider
	Database   string
	config     RepositoryConfig
}

// NewConnectionMongoDBRepository provisions a repository using the given client.
// Accepts an optional RepositoryConfig; if nil, defaults are used.
// The provider must implement GetDB(ctx) (*mongo.Client, error). When the provider
// also implements tmcore.MultiTenantChecker (IsMultiTenant() bool), ResolveMongo will
// return ErrTenantContextRequired instead of silently falling back to the default DB.
func NewConnectionMongoDBRepository(ctx context.Context, provider mongodb.MongoClientProvider, dbName string, cfg ...RepositoryConfig) (*ConnectionMongoDBRepository, error) {
	config := RepositoryConfig{
		InitTimeout: DefaultInitTimeout,
	}
	if len(cfg) > 0 {
		config = cfg[0]
		if config.InitTimeout <= 0 {
			config.InitTimeout = DefaultInitTimeout
		}
	}

	repo := &ConnectionMongoDBRepository{
		connection: provider,
		Database:   dbName,
		config:     config,
	}

	ctx, cancel := context.WithTimeout(ctx, config.InitTimeout)
	defer cancel()

	if _, err := repo.connection.Client(ctx); err != nil {
		return nil, err
	}

	return repo, nil
}

// getDatabase returns a *mongo.Database for the current request context.
// In multi-tenant mode, it retrieves the tenant-specific database from context
// via tmcore.GetMongoForTenant. In single-tenant mode (no tenant in context),
// it falls back to the static connection using cr.connection.GetDB.
func (cr *ConnectionMongoDBRepository) getDatabase(ctx context.Context) (*mongo.Database, error) {
	return mongodb.ResolveDatabase(ctx, cr.connection, cr.Database)
}

// Create inserts a new connection respecting the unique constraint per organization.
func (cr *ConnectionMongoDBRepository) Create(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.create_connection")
	defer span.End()

	if conn == nil {
		err := errors.New("connection is required")
		libOpentelemetry.HandleSpanError(span, "Connection payload is nil", err)

		return nil, pkg.ValidateInternalError(err, "connection")
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.config_name", conn.ConfigName),
		attribute.String("app.request.connection_id", conn.ID.String()),
	)

	db, err := cr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	record := NewConnectionMongoDBModelFromDomain(conn)
	if err := setSpanAttributesFromValue(span, "app.request.payload", record.ToMapWithMask()); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert record to JSON", err)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))
	if _, err := coll.InsertOne(ctx, record); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			err := fmt.Errorf("connection with config_name '%s' already exists", conn.ConfigName)
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Duplicate connection", err)

			return nil, pkg.ValidateBusinessError(
				constant.ErrEntityConflict,
				"connection",
			)
		}

		libOpentelemetry.HandleSpanError(span, "Failed to insert connection", err)

		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	connection, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return connection, nil
}

// Update overwrites mutable fields of an existing connection and returns the saved entity.
func (cr *ConnectionMongoDBRepository) Update(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.update_connection")
	defer span.End()

	if conn == nil {
		err := errors.New("connection is required")
		libOpentelemetry.HandleSpanError(span, "Connection payload is nil", err)

		return nil, pkg.ValidateInternalError(err, "connection")
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", conn.ID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := cr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))
	filter := bson.M{
		"_id":        conn.ID,
		"deleted_at": bson.D{{Key: "$eq", Value: nil}},
	}

	update := bson.M{
		"$set": bson.M{
			"config_name":            conn.ConfigName,
			"type":                   conn.Type,
			"host":                   conn.Host,
			"port":                   conn.Port,
			"database_name":          conn.DatabaseName,
			"schema":                 conn.Schema,
			"username":               conn.Username,
			"password_encrypted":     conn.PasswordEncrypted,
			"encryption_key_version": conn.EncryptionKeyVersion,
			"updated_at":             conn.UpdatedAt,
			"metadata":               conn.Metadata,
		},
	}

	mongoRecord := NewConnectionMongoDBModelFromDomain(conn)
	if mongoRecord.SSL != nil {
		update["$set"].(bson.M)["ssl"] = mongoRecord.SSL
	} else {
		update["$set"].(bson.M)["ssl"] = nil
	}

	span.SetAttributes(attributes...)

	if err := setSpanAttributesFromValue(span, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert filter to JSON", err)
	}

	var record ConnectionMongoDBModel

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	if err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		if mongo.IsDuplicateKeyError(err) {
			err := fmt.Errorf("connection with config_name '%s' already exists", conn.ConfigName)
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Duplicate connection", err)

			return nil, pkg.ValidateBusinessError(
				constant.ErrEntityConflict,
				"connection",
			)
		}

		libOpentelemetry.HandleSpanError(span, "Failed to update connection", err)

		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	connection, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return connection, nil
}

// Delete performs a soft delete by stamping deleted_at and updated_at fields.
func (cr *ConnectionMongoDBRepository) Delete(ctx context.Context, connectionID uuid.UUID, deletedAt time.Time) error {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.delete_connection")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := cr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return mongodb.MapMongoErrorToResponse(err, ctx)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))
	filter := bson.M{
		"_id":        connectionID,
		"deleted_at": bson.D{{Key: "$eq", Value: nil}},
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": deletedAt,
			"updated_at": deletedAt,
		},
	}

	res, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to soft delete connection", err)
		return mongodb.MapMongoErrorToResponse(err, ctx)
	}

	if res.MatchedCount == 0 {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection not found for delete", mongo.ErrNoDocuments)

		return pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	return nil
}

// FindByID fetches a connection by its ID scoped to an organization.
func (cr *ConnectionMongoDBRepository) FindByID(ctx context.Context, connectionID uuid.UUID) (*model.Connection, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_id")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := cr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	var record ConnectionMongoDBModel

	filter := bson.M{
		"_id":        connectionID,
		"deleted_at": bson.D{{Key: "$eq", Value: nil}},
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))
	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(span, "Failed to find connection", err)

		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	connection, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return connection, nil
}

// FindByName retrieves a connection by configName.
func (cr *ConnectionMongoDBRepository) FindByName(ctx context.Context, configName string) (*model.Connection, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_name")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.config_name", configName),
	}
	span.SetAttributes(attributes...)

	db, err := cr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	var record ConnectionMongoDBModel

	filter := bson.M{
		"config_name": configName,
		"deleted_at":  bson.D{{Key: "$eq", Value: nil}},
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))
	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(span, "Failed to find connection by config_name", err)

		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	conn, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	if errSpan := setSpanAttributesFromValue(span, "app.response.payload", conn.ToMapWithMask()); errSpan != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert connection to JSON", errSpan)
	}

	return conn, nil
}

// FindByDatabaseName retrieves a connection by databaseName.
func (cr *ConnectionMongoDBRepository) FindByDatabaseName(ctx context.Context, databaseName string) (*model.Connection, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_database")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.database_name", databaseName),
	}
	span.SetAttributes(attributes...)

	if strings.TrimSpace(databaseName) == "" {
		err := errors.New("database_name cannot be empty")
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Invalid database_name", err)

		return nil, pkg.ValidateBusinessError(constant.ErrInvalidDataRequest, "connection", err.Error())
	}

	db, err := cr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))

	var record ConnectionMongoDBModel

	filter := bson.M{
		"database_name": databaseName,
		"deleted_at":    bson.D{{Key: "$eq", Value: nil}},
	}

	if errFind := coll.FindOne(ctx, filter).Decode(&record); errFind != nil {
		if errors.Is(errFind, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(span, "Failed to find connection by database_name", errFind)

		return nil, mongodb.MapMongoErrorToResponse(errFind, ctx)
	}

	connection, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return connection, nil
}

// FindByConfigNames retrieves connections that match any of the provided config names for the given organization.
func (cr *ConnectionMongoDBRepository) FindByConfigNames(ctx context.Context, configNames []string) ([]*model.Connection, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connections_by_config_names")
	defer span.End()

	if len(configNames) == 0 {
		return []*model.Connection{}, nil
	}

	// Trim and filter empty config names
	trimmedNames := make([]string, 0, len(configNames))
	for _, name := range configNames {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			trimmedNames = append(trimmedNames, trimmed)
		}
	}

	if len(trimmedNames) == 0 {
		return []*model.Connection{}, nil
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.Int("app.request.config_names_count", len(trimmedNames)),
	}
	span.SetAttributes(attributes...)

	db, err := cr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))

	filter := bson.M{
		"config_name": bson.M{"$in": trimmedNames},
		"deleted_at":  bson.D{{Key: "$eq", Value: nil}},
	}

	if err := setSpanAttributesFromValue(span, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert filter to JSON", err)
	}

	cur, err := coll.Find(ctx, filter)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to find connections by config names", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}
	defer cur.Close(ctx)

	connections := make([]*model.Connection, 0)

	for cur.Next(ctx) {
		var record ConnectionMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to decode connection record", err)
			return nil, mongodb.MapMongoErrorToResponse(err, ctx)
		}

		recordConvert, errDomain := record.ToEntity()
		if errDomain != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to convert connection model", errDomain)
			return nil, pkg.ValidateInternalError(errDomain, "connection")
		}

		connections = append(connections, recordConvert)
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to iterate over connections", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	span.SetAttributes(attribute.Int("app.response.connections_count", len(connections)))

	return connections, nil
}

// List returns a paginated set of connections for the given organization.
func (rm *ConnectionMongoDBRepository) List(ctx context.Context, filters http.QueryHeader) ([]*model.Connection, int64, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.list_connections")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", filters, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert filters to JSON string", err)
	}

	db, err := rm.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, 0, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	queryFilter := rm.buildQueryFilter(filters)
	opts, limit := mongodb.BuildPaginationOptions(filters)

	err = libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.repository_filter", queryFilter, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert filters to JSON string", err)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))

	totalCount, err := coll.CountDocuments(ctx, queryFilter)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to count connections", err)
		return nil, 0, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	cur, err := coll.Find(ctx, queryFilter, opts)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to list connections", err)
		return nil, 0, mongodb.MapMongoErrorToResponse(err, ctx)
	}
	defer cur.Close(ctx)

	connections := make([]*model.Connection, 0, int(limit))

	for cur.Next(ctx) {
		var record ConnectionMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to decode connection record", err)
			return nil, 0, mongodb.MapMongoErrorToResponse(err, ctx)
		}

		connection, err := record.ToEntity()
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to convert record to domain", err)
			return nil, 0, pkg.ValidateInternalError(err, "connection")
		}

		connections = append(connections, connection)
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to iterate over connections", err)
		return nil, 0, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	span.SetAttributes(attribute.Int64("app.response.total_count", totalCount))

	return connections, totalCount, nil
}

// ListUnassigned returns a paginated set of connections that have no product assigned for the given organization.
func (rm *ConnectionMongoDBRepository) ListUnassigned(ctx context.Context, filters http.QueryHeader) ([]*model.Connection, int64, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.list_unassigned_connections")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", filters, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert filters to JSON string", err)
	}

	db, err := rm.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, 0, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	queryFilter := bson.M{
		"deleted_at": bson.D{{Key: "$eq", Value: nil}},
		"$or": []bson.M{
			{"product_name": ""},
			{"product_name": bson.M{"$eq": nil}},
			{"product_name": bson.M{"$exists": false}},
		},
	}

	mongodb.AddDateRangeFilter(queryFilter, filters)
	opts, limit := mongodb.BuildPaginationOptions(filters)

	err = libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.repository_filter", queryFilter, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert filters to JSON string", err)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))

	totalCount, err := coll.CountDocuments(ctx, queryFilter)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to count unassigned connections", err)
		return nil, 0, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	cur, err := coll.Find(ctx, queryFilter, opts)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to list unassigned connections", err)
		return nil, 0, mongodb.MapMongoErrorToResponse(err, ctx)
	}
	defer cur.Close(ctx)

	connections := make([]*model.Connection, 0, int(limit))

	for cur.Next(ctx) {
		var record ConnectionMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to decode connection record", err)
			return nil, 0, mongodb.MapMongoErrorToResponse(err, ctx)
		}

		connection, err := record.ToEntity()
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to convert record to domain", err)
			return nil, 0, pkg.ValidateInternalError(err, "connection")
		}

		connections = append(connections, connection)
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to iterate over connections", err)
		return nil, 0, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	span.SetAttributes(attribute.Int64("app.response.total_count", totalCount))

	return connections, totalCount, nil
}

// AssignProductName associates a legacy (unassigned) connection to a product by name. Returns the updated connection.
func (cr *ConnectionMongoDBRepository) AssignProductName(ctx context.Context, connectionID uuid.UUID, productName string) (*model.Connection, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.assign_connection_product")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
		attribute.String("app.request.product_name", productName),
	)

	db, err := cr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))

	filter := bson.M{
		"_id":        connectionID,
		"deleted_at": bson.D{{Key: "$eq", Value: nil}},
		"$or": []bson.M{
			{"product_name": ""},
			{"product_name": bson.M{"$eq": nil}},
			{"product_name": bson.M{"$exists": false}},
		},
	}

	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"product_name": productName,
			"updated_at":   now,
		},
	}

	var record ConnectionMongoDBModel

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	if err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(span, "Failed to assign product to connection", err)

		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	connection, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return connection, nil
}

// buildQueryFilter builds the MongoDB query filter from filters
func (rm *ConnectionMongoDBRepository) buildQueryFilter(filters http.QueryHeader) bson.M {
	queryFilter := bson.M{
		"deleted_at": bson.D{{Key: "$eq", Value: nil}},
	}

	if filters.ProductName != "" {
		queryFilter["product_name"] = filters.ProductName
	}

	if filters.Type != "" {
		queryFilter["type"] = filters.Type
	}

	if len(filters.Metadata) > 0 && filters.UseMetadata {
		for key, value := range filters.Metadata {
			queryFilter[key] = value
		}
	}

	mongodb.AddDateRangeFilter(queryFilter, filters)

	return queryFilter
}
