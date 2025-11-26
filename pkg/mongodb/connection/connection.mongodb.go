package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	domainConn "github.com/LerianStudio/fetcher/pkg/domain"
	"github.com/LerianStudio/lib-commons/v2/commons"
	libMongo "github.com/LerianStudio/lib-commons/v2/commons/mongo"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

const (
	defaultConnectionPageLimit = 50
	maxConnectionPageLimit     = 1000
)

// mongoDatabaseProvider defines the interface for obtaining a MongoDB client.
type mongoDatabaseProvider interface {
	GetDB(ctx context.Context) (*mongo.Client, error)
}

// SetSpanAttributesFromStruct is a helper to set span attributes from a struct.
var setSpanAttributesFromStruct = libOpentelemetry.SetSpanAttributesFromStruct

// ConnectionMongoDBRepository implements Repository backed by MongoDB.
type ConnectionMongoDBRepository struct {
	connection mongoDatabaseProvider
	Database   string
}

// NewConnectionMongoDBRepository provisions a repository using the given client.
func NewConnectionMongoDBRepository(ctx context.Context, mc *libMongo.MongoConnection) (*ConnectionMongoDBRepository, error) {
	repo := &ConnectionMongoDBRepository{
		connection: mc,
		Database:   mc.Database,
	}

	if _, err := repo.connection.GetDB(ctx); err != nil {
		return nil, err
	}

	return repo, nil
}

// Create inserts a new connection respecting the unique constraint per organization.
func (cr *ConnectionMongoDBRepository) Create(ctx context.Context, conn *domainConn.Connection) (*domainConn.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)
	ctx, span := tracer.Start(ctx, "mongodb.create_connection")
	defer span.End()

	if conn == nil {
		err := errors.New("connection is required")
		libOpentelemetry.HandleSpanError(&span, "Connection payload is nil", err)
		return nil, err
	}

	if err := conn.ValidateForCreate(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Invalid connection payload", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("app.request.request_id", reqID))
	if conn.OrganizationID != uuid.Nil {
		span.SetAttributes(attribute.String("app.request.organization_id", conn.OrganizationID.String()))
	}
	if conn.ID != uuid.Nil {
		span.SetAttributes(attribute.String("app.request.connection_id", conn.ID.String()))
	}

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	record := &ConnectionMongoDBModel{}

	if err := record.FromDomain(conn); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert entity to MongoDB model", err)
		return nil, err
	}

	if err := setSpanAttributesFromStruct(&span, "app.request.payload", record.ToMapWithMask()); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to JSON", err)
	}

	if _, err := coll.InsertOne(ctx, record); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			err := fmt.Errorf("connection with config_name '%s' already exists for organization '%s'", conn.ConfigName, conn.OrganizationID.String())
			libOpentelemetry.HandleSpanError(&span, "Duplicate connection", err)
			return nil, err
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to insert connection", err)
		return nil, err
	}

	return record.ToDomain(), nil
}

// Update overwrites mutable fields of an existing connection and returns the saved entity.
func (cr *ConnectionMongoDBRepository) Update(ctx context.Context, conn *domainConn.Connection) (*domainConn.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.update_connection")
	defer span.End()

	if conn == nil {
		err := errors.New("connection is required")
		libOpentelemetry.HandleSpanError(&span, "Connection payload is nil", err)
		return nil, err
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", conn.ID.String()),
		attribute.String("app.request.organization_id", conn.OrganizationID.String()),
	}
	span.SetAttributes(attributes...)

	if err := conn.ValidateForUpdate(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Invalid connection payload", err)
		return nil, err
	}

	conn.UpdatedAt = time.Now().UTC()

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	filter := bson.M{
		"_id":             conn.ID,
		"organization_id": conn.OrganizationID,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	update := bson.M{
		"$set": bson.M{
			"config_name":        conn.ConfigName,
			"type":               conn.Type,
			"host":               conn.Host,
			"port":               conn.Port,
			"database_name":      conn.DatabaseName,
			"username":           conn.Username,
			"password_encrypted": conn.PasswordEncrypted,
			"key_version":        conn.KeyVersion,
			"ssl":                conn.SSL,
			"updated_at":         conn.UpdatedAt,
		},
	}

	span.SetAttributes(attributes...)
	if err := setSpanAttributesFromStruct(&span, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filter to JSON", err)
	}

	var record ConnectionMongoDBModel
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	if err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&record); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to update connection", err)
		return nil, err
	}

	return record.ToDomain(), nil
}

// Delete performs a soft delete by stamping deleted_at and updated_at fields.
func (cr *ConnectionMongoDBRepository) Delete(ctx context.Context, id, organizationID uuid.UUID, deletedAt time.Time) error {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.delete_connection")
	defer span.End()

	if deletedAt.IsZero() {
		deletedAt = time.Now().UTC()
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", id.String()),
		attribute.String("app.request.organization_id", organizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	filter := bson.M{
		"_id":             id,
		"organization_id": organizationID,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": deletedAt,
			"updated_at": deletedAt,
		},
	}

	res, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to soft delete connection", err)
		return err
	}

	if res.MatchedCount == 0 {
		err := mongo.ErrNoDocuments
		libOpentelemetry.HandleSpanError(&span, "Connection not found for delete", err)
		return err
	}

	return nil
}

// FindByID fetches a connection by its ID scoped to an organization.
func (cr *ConnectionMongoDBRepository) FindByID(ctx context.Context, id, organizationID uuid.UUID) (*domainConn.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_id")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", id.String()),
		attribute.String("app.request.organization_id", organizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))

	var record ConnectionMongoDBModel
	filter := bson.M{
		"_id":             id,
		"organization_id": organizationID,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		libOpentelemetry.HandleSpanError(&span, "Failed to find connection", err)
		return nil, err
	}

	return record.ToDomain(), nil
}

// FindByOrganizationAndName retrieves a connection ensuring config_name uniqueness per org.
func (cr *ConnectionMongoDBRepository) FindByOrganizationAndName(ctx context.Context, organizationID uuid.UUID, configName string) (*domainConn.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_org_name")
	defer span.End()

	configName = strings.ToLower(strings.TrimSpace(configName))
	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.config_name", configName),
	}
	span.SetAttributes(attributes...)

	if configName == "" {
		err := errors.New("config_name cannot be empty")
		libOpentelemetry.HandleSpanError(&span, "Invalid config_name", err)
		return nil, err
	}

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))

	var record ConnectionMongoDBModel
	filter := bson.M{
		"organization_id": organizationID,
		"config_name":     configName,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to find connection by config_name", err)
		return nil, err
	}

	return record.ToDomain(), nil
}

// FindByOrganizationAndDatabaseName retrieves a connection by databaseName scoped to an organization.
func (cr *ConnectionMongoDBRepository) FindByOrganizationAndDatabaseName(ctx context.Context, organizationID uuid.UUID, databaseName string) (*domainConn.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_org_database")
	defer span.End()

	databaseName = strings.ToLower(strings.TrimSpace(databaseName))
	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.database_name", databaseName),
	}
	span.SetAttributes(attributes...)

	if databaseName == "" {
		err := errors.New("database_name cannot be empty")
		libOpentelemetry.HandleSpanError(&span, "Invalid database_name", err)
		return nil, err
	}

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))

	var record ConnectionMongoDBModel
	filter := bson.M{
		"organization_id": organizationID,
		"database_name":   databaseName,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to find connection by database_name", err)
		return nil, err
	}

	return record.ToDomain(), nil
}

// List returns a paginated set of connections for the given organization.
func (cr *ConnectionMongoDBRepository) List(ctx context.Context, filters *domainConn.ListFilterParams) ([]*domainConn.Connection, error) {
	if filters == nil {
		filters = &domainConn.ListFilterParams{}
	}

	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.list_connections")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", filters.OrganizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))

	queryFilter := bson.M{
		"organization_id": filters.OrganizationID,
	}

	if !filters.IncludeDeleted {
		queryFilter["deleted_at"] = bson.D{{Key: "$eq", Value: nil}}
	}

	if trimmed := strings.TrimSpace(filters.ConfigName); trimmed != "" {
		queryFilter["config_name"] = strings.ToLower(trimmed)
	}

	if len(filters.Types) > 0 {
		queryFilter["type"] = bson.M{"$in": filters.Types}
	}

	if trimmedHost := strings.TrimSpace(filters.Host); trimmedHost != "" {
		queryFilter["host"] = trimmedHost
	}

	if trimmedDB := strings.TrimSpace(filters.DatabaseName); trimmedDB != "" {
		queryFilter["database_name"] = strings.ToLower(trimmedDB)
	}

	createdRange := bson.M{}
	if filters.CreatedFrom != nil {
		createdRange["$gte"] = *filters.CreatedFrom
	}
	if filters.CreatedTo != nil {
		createdRange["$lte"] = *filters.CreatedTo
	}
	if len(createdRange) > 0 {
		queryFilter["created_at"] = createdRange
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = defaultConnectionPageLimit
	}
	if limit > maxConnectionPageLimit {
		limit = maxConnectionPageLimit
	}

	page := filters.Page
	if page < 0 {
		page = 0
	}

	skip := int64(page * limit)
	limit64 := int64(limit)
	sortDirection := int32(-1)
	if strings.EqualFold(string(filters.SortOrder), string(constant.Asc)) {
		sortDirection = 1
	}

	opts := options.FindOptions{
		Limit: &limit64,
		Skip:  &skip,
		Sort:  bson.D{{Key: "created_at", Value: sortDirection}},
	}

	if err := setSpanAttributesFromStruct(&span, "app.request.repository_filter", queryFilter); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert list filter to JSON", err)
	}

	cur, err := coll.Find(ctx, queryFilter, &opts)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to list connections", err)
		return nil, err
	}
	defer cur.Close(ctx)

	connections := make([]*domainConn.Connection, 0, limit)
	for cur.Next(ctx) {
		var record ConnectionMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to decode connection record", err)
			return nil, err
		}
		connections = append(connections, record.ToDomain())
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to iterate over connections", err)
		return nil, err
	}

	return connections, nil
}
