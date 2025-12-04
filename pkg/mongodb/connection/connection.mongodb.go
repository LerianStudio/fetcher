package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libMongo "github.com/LerianStudio/lib-commons/v2/commons/mongo"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

// Repository defines the domain port for connections.
type Repository interface {
	Create(ctx context.Context, conn *model.Connection) (*model.Connection, error)
	Update(ctx context.Context, conn *model.Connection) (*model.Connection, error)
	Delete(ctx context.Context, id, organizationID uuid.UUID, deletedAt time.Time) error
	FindByID(ctx context.Context, id, organizationID uuid.UUID) (*model.Connection, error)
	FindByOrganizationAndName(ctx context.Context, organizationID uuid.UUID, configName string) (*model.Connection, error)
	FindByOrganizationAndDatabaseName(ctx context.Context, organizationID uuid.UUID, databaseName string) (*model.Connection, error)
	FindByConfigNames(ctx context.Context, organizationID uuid.UUID, configNames []string) ([]*model.Connection, error)
	List(ctx context.Context, organizationID uuid.UUID, filters http.QueryHeader) ([]*model.Connection, error)
}

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
func NewConnectionMongoDBRepository(mc *libMongo.MongoConnection) (*ConnectionMongoDBRepository, error) {
	repo := &ConnectionMongoDBRepository{
		connection: mc,
		Database:   mc.Database,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := repo.connection.GetDB(ctx); err != nil {
		return nil, err
	}

	return repo, nil
}

// Create inserts a new connection respecting the unique constraint per organization.
func (cr *ConnectionMongoDBRepository) Create(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.create_connection")
	defer span.End()

	if conn == nil {
		err := errors.New("connection is required")
		libOpentelemetry.HandleSpanError(&span, "Connection payload is nil", err)

		return nil, pkg.ValidateInternalError(err, "connection")
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", conn.OrganizationID.String()),
		attribute.String("app.request.config_name", conn.ConfigName),
		attribute.String("app.request.connection_id", conn.ID.String()),
	)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	record := NewConnectionMongoDBModelFromDomain(conn)
	if err := setSpanAttributesFromStruct(&span, "app.request.payload", record.ToMapWithMask()); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to JSON", err)
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	if _, err := coll.InsertOne(ctx, record); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			err := fmt.Errorf("connection with config_name '%s' already exists for organization '%s'", conn.ConfigName, conn.OrganizationID.String())
			libOpentelemetry.HandleSpanError(&span, "Duplicate connection", err)

			return nil, pkg.EntityConflictError{
				EntityType: "connection",
				Code:       constant.ErrEntityConflict.Error(),
				Title:      "Conflict",
				Message:    "Connection with the same name already exists",
			}
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to insert connection", err)

		return nil, pkg.ValidateInternalError(err, "connection")
	}

	connection, err := record.ToDomain()
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return connection, nil
}

// Update overwrites mutable fields of an existing connection and returns the saved entity.
func (cr *ConnectionMongoDBRepository) Update(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.update_connection")
	defer span.End()

	if conn == nil {
		err := errors.New("connection is required")
		libOpentelemetry.HandleSpanError(&span, "Connection payload is nil", err)

		return nil, pkg.ValidateInternalError(err, "connection")
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", conn.ID.String()),
		attribute.String("app.request.organization_id", conn.OrganizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	filter := bson.M{
		"_id":             conn.ID,
		"organization_id": conn.OrganizationID,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	update := bson.M{
		"$set": bson.M{
			"config_name":            conn.ConfigName,
			"type":                   conn.Type,
			"host":                   conn.Host,
			"port":                   conn.Port,
			"database_name":          conn.DatabaseName,
			"username":               conn.Username,
			"password_encrypted":     conn.PasswordEncrypted,
			"encryption_key_version": conn.EncryptionKeyVersion,
			"updated_at":             conn.UpdatedAt,
		},
	}

	mongoRecord := NewConnectionMongoDBModelFromDomain(conn)
	if mongoRecord.SSL != nil {
		update["$set"].(bson.M)["ssl"] = mongoRecord.SSL
	} else {
		update["$set"].(bson.M)["ssl"] = nil
	}

	span.SetAttributes(attributes...)

	if err := setSpanAttributesFromStruct(&span, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filter to JSON", err)
	}

	var record ConnectionMongoDBModel

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	if err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		if mongo.IsDuplicateKeyError(err) {
			err := fmt.Errorf("connection with config_name '%s' already exists for organization '%s'", conn.ConfigName, conn.OrganizationID.String())
			libOpentelemetry.HandleSpanError(&span, "Duplicate connection", err)

			return nil, pkg.EntityConflictError{
				EntityType: "connection",
				Code:       constant.ErrEntityConflict.Error(),
				Title:      "Conflict",
				Message:    "Connection with the same name already exists",
			}
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to update connection", err)

		return nil, pkg.ValidateInternalError(err, "connection")
	}

	connection, err := record.ToDomain()
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return connection, nil
}

// Delete performs a soft delete by stamping deleted_at and updated_at fields.
func (cr *ConnectionMongoDBRepository) Delete(ctx context.Context, connectionID, organizationID uuid.UUID, deletedAt time.Time) error {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.delete_connection")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
		attribute.String("app.request.organization_id", organizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return pkg.ValidateInternalError(err, "connection")
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	filter := bson.M{
		"_id":             connectionID,
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
		return pkg.ValidateInternalError(err, "connection")
	}

	if res.MatchedCount == 0 {
		libOpentelemetry.HandleSpanError(&span, "Connection not found for delete", mongo.ErrNoDocuments)

		return pkg.EntityNotFoundError{
			EntityType: "connection",
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		}
	}

	return nil
}

// FindByID fetches a connection by its ID scoped to an organization.
func (cr *ConnectionMongoDBRepository) FindByID(ctx context.Context, connectionID, organizationID uuid.UUID) (*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_id")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
		attribute.String("app.request.organization_id", organizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	var record ConnectionMongoDBModel

	filter := bson.M{
		"_id":             connectionID,
		"organization_id": organizationID,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to find connection", err)

		return nil, pkg.ValidateInternalError(err, "connection")
	}

	connection, err := record.ToDomain()
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return connection, nil
}

// FindByOrganizationAndName retrieves a connection by configName scoped to an organization.
func (cr *ConnectionMongoDBRepository) FindByOrganizationAndName(ctx context.Context, organizationID uuid.UUID, configName string) (*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_org_name")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.config_name", configName),
	}
	span.SetAttributes(attributes...)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	var record ConnectionMongoDBModel

	filter := bson.M{
		"organization_id": organizationID,
		"config_name":     configName,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to find connection by config_name", err)

		return nil, pkg.ValidateInternalError(err, "connection")
	}

	conn, err := record.ToDomain()
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	if errSpan := setSpanAttributesFromStruct(&span, "app.response.payload", conn.ToMapWithMask()); errSpan != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert connection to JSON", errSpan)
	}

	return conn, nil
}

// FindByOrganizationAndDatabaseName retrieves a connection by databaseName scoped to an organization.
func (cr *ConnectionMongoDBRepository) FindByOrganizationAndDatabaseName(ctx context.Context, organizationID uuid.UUID, databaseName string) (*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_org_database")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.database_name", databaseName),
	}
	span.SetAttributes(attributes...)

	if databaseName == "" {
		err := errors.New("database_name cannot be empty")
		libOpentelemetry.HandleSpanError(&span, "Invalid database_name", err)

		return nil, pkg.ValidateInternalError(err, "connection")
	}

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))

	var record ConnectionMongoDBModel

	filter := bson.M{
		"organization_id": organizationID,
		"database_name":   databaseName,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	if errFind := coll.FindOne(ctx, filter).Decode(&record); errFind != nil {
		if errors.Is(errFind, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to find connection by database_name", errFind)

		return nil, pkg.ValidateInternalError(errFind, "connection")
	}

	connection, err := record.ToDomain()
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return connection, nil
}

// FindByConfigNames retrieves connections that match any of the provided config names for the given organization.
func (cr *ConnectionMongoDBRepository) FindByConfigNames(ctx context.Context, organizationID uuid.UUID, configNames []string) ([]*model.Connection, error) {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

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
		attribute.String("app.request.request_id", reqId),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.Int("app.request.config_names_count", len(trimmedNames)),
	}
	span.SetAttributes(attributes...)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))

	filter := bson.M{
		"organization_id": organizationID,
		"config_name":     bson.M{"$in": trimmedNames},
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	if err := setSpanAttributesFromStruct(&span, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filter to JSON", err)
	}

	cur, err := coll.Find(ctx, filter)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to find connections by config names", err)
		return nil, err
	}
	defer cur.Close(ctx)

	connections := make([]*model.Connection, 0)

	for cur.Next(ctx) {
		var record ConnectionMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to decode connection record", err)
			return nil, err
		}

		recordConvert, errDomain := record.ToDomain()
		if errDomain != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to convert connection model", err)
			return nil, errDomain
		}

		connections = append(connections, recordConvert)
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to iterate over connections", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("app.response.connections_count", len(connections)))

	return connections, nil
}

// List returns a paginated set of connections for the given organization.
func (rm *ConnectionMongoDBRepository) List(ctx context.Context, organizationID uuid.UUID, filters http.QueryHeader) ([]*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.list_connections")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", filters)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filters to JSON string", err)
	}

	db, err := rm.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	queryFilter := bson.M{
		"organization_id": organizationID,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	if filters.Metadata != nil && filters.UseMetadata {
		for key, value := range *filters.Metadata {
			queryFilter[key] = value
		}
	}

	if !filters.StartDate.IsZero() || !filters.EndDate.IsZero() {
		createdAt := bson.M{}
		if !filters.StartDate.IsZero() {
			createdAt["$gte"] = filters.StartDate
		}

		if !filters.EndDate.IsZero() {
			createdAt["$lte"] = filters.EndDate
		}

		queryFilter["created_at"] = createdAt
	}

	limit := int64(filters.Limit)
	if limit < 0 {
		limit = 0
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}

	skip := int64(page*int(limit) - int(limit))
	if skip < 0 {
		skip = 0
	}

	opts := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  bson.D{{Key: "created_at", Value: -1}},
	}

	err = libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.repository_filter", queryFilter)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filters to JSON string", err)
	}

	coll := db.Database(strings.ToLower(rm.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))

	cur, err := coll.Find(ctx, queryFilter, &opts)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to list connections", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}
	defer cur.Close(ctx)

	connections := make([]*model.Connection, 0, int(limit))

	for cur.Next(ctx) {
		var record ConnectionMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(&span, "", err)
			return nil, pkg.ValidateInternalError(err, "connection")
		}

		connection, err := record.ToDomain()
		if err != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
			return nil, pkg.ValidateInternalError(err, "connection")
		}

		connections = append(connections, connection)
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to iterate over connections", err)
		return nil, pkg.ValidateInternalError(err, "connection")
	}

	return connections, nil
}
