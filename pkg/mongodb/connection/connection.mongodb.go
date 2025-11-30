package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
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
	defaultConnectionPageLimit = 10
	maxConnectionPageLimit     = 50
)

// ListFilter controls pagination and filtering for List queries.
type ListFilter struct {
	OrganizationID uuid.UUID
	ConfigName     string
	Types          []ConnectionType
	IncludeDeleted bool
	Limit          int
	Page           int
	SortOrder      constant.Order
}

// Repository defines the MongoDB contract for the connections collection.
//
//go:generate mockgen --destination=connection.mongodb.mock.go --package=connection . Repository
type Repository interface {
	Create(ctx context.Context, connection *Connection) (*Connection, error)
	Update(ctx context.Context, connection *Connection) (*Connection, error)
	Delete(ctx context.Context, id, organizationID uuid.UUID, deletedAt time.Time) error
	FindByID(ctx context.Context, id, organizationID uuid.UUID) (*Connection, error)
	FindByOrganizationAndName(ctx context.Context, organizationID uuid.UUID, configName string) (*Connection, error)
	FindByConfigNames(ctx context.Context, organizationID uuid.UUID, configNames []string) ([]*Connection, error)
	List(ctx context.Context, filters *ListFilter) ([]*Connection, error)
}

type mongoDatabaseProvider interface {
	GetDB(ctx context.Context) (*mongo.Client, error)
}

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
func (cr *ConnectionMongoDBRepository) Create(ctx context.Context, conn *Connection) (*Connection, error) {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.create_connection")
	defer span.End()

	if conn == nil {
		err := errors.New("connection is required")
		libOpentelemetry.HandleSpanError(&span, "Connection payload is nil", err)

		return nil, err
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
	}

	if conn.OrganizationID != uuid.Nil {
		attributes = append(attributes, attribute.String("app.request.organization_id", conn.OrganizationID.String()))
	}

	if conn.ID != uuid.Nil {
		attributes = append(attributes, attribute.String("app.request.connection_id", conn.ID.String()))
	}

	span.SetAttributes(attributes...)

	if err := conn.ValidateForCreate(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Invalid connection payload", err)
		return nil, err
	}

	if err := setSpanAttributesFromStruct(&span, "app.request.payload", conn); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert connection payload to JSON", err)
	}

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))
	record := &ConnectionMongoDBModel{}

	if err := record.FromEntity(conn); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert entity to MongoDB model", err)
		return nil, err
	}

	ctx, spanInsert := tracer.Start(ctx, "mongodb.create_connection.insert")
	defer spanInsert.End()

	spanInsert.SetAttributes(attributes...)
	spanInsert.SetAttributes(attribute.String("app.request.connection_id", record.ID.String()))

	if err := setSpanAttributesFromStruct(&spanInsert, "app.request.repository_input", NewConnectionTelemetryFromMongoDBModel(record)); err != nil {
		libOpentelemetry.HandleSpanError(&spanInsert, "Failed to convert record to JSON", err)
	}

	if _, err := coll.InsertOne(ctx, record); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			err := fmt.Errorf("connection with config_name '%s' already exists for organization '%s'", conn.ConfigName, conn.OrganizationID.String())
			libOpentelemetry.HandleSpanError(&spanInsert, "Duplicate connection", err)

			return nil, err
		}

		libOpentelemetry.HandleSpanError(&spanInsert, "Failed to insert connection", err)

		return nil, err
	}

	return record.ToEntity(), nil
}

// Update overwrites mutable fields of an existing connection and returns the saved entity.
func (cr *ConnectionMongoDBRepository) Update(ctx context.Context, conn *Connection) (*Connection, error) {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.update_connection")
	defer span.End()

	if conn == nil {
		err := errors.New("connection is required")
		libOpentelemetry.HandleSpanError(&span, "Connection payload is nil", err)

		return nil, err
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
		attribute.String("app.request.connection_id", conn.ID.String()),
		attribute.String("app.request.organization_id", conn.OrganizationID.String()),
	}
	span.SetAttributes(attributes...)

	if err := conn.ValidateForUpdate(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Invalid connection payload", err)
		return nil, err
	}

	conn.UpdatedAt = time.Now().UTC()

	if err := setSpanAttributesFromStruct(&span, "app.request.payload", conn); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert connection payload to JSON", err)
	}

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
			"ssl":                conn.SSL,
			"updated_at":         conn.UpdatedAt,
		},
	}

	ctx, spanUpdate := tracer.Start(ctx, "mongodb.update_connection.find_one_and_update")
	defer spanUpdate.End()

	spanUpdate.SetAttributes(attributes...)

	if err := setSpanAttributesFromStruct(&spanUpdate, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(&spanUpdate, "Failed to convert filter to JSON", err)
	}

	var record ConnectionMongoDBModel

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	if err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&record); err != nil {
		libOpentelemetry.HandleSpanError(&spanUpdate, "Failed to update connection", err)
		return nil, err
	}

	return record.ToEntity(), nil
}

// Delete performs a soft delete by stamping deleted_at and updated_at fields.
func (cr *ConnectionMongoDBRepository) Delete(ctx context.Context, id, organizationID uuid.UUID, deletedAt time.Time) error {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.delete_connection")
	defer span.End()

	if deletedAt.IsZero() {
		deletedAt = time.Now().UTC()
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
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
func (cr *ConnectionMongoDBRepository) FindByID(ctx context.Context, id, organizationID uuid.UUID) (*Connection, error) {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_id")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
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
		libOpentelemetry.HandleSpanError(&span, "Failed to find connection", err)
		return nil, err
	}

	return record.ToEntity(), nil
}

// FindByOrganizationAndName retrieves a connection ensuring config_name uniqueness per org.
func (cr *ConnectionMongoDBRepository) FindByOrganizationAndName(ctx context.Context, organizationID uuid.UUID, configName string) (*Connection, error) {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connection_by_org_name")
	defer span.End()

	configName = strings.TrimSpace(configName)
	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
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

	return record.ToEntity(), nil
}

// FindByConfigNames retrieves connections that match any of the provided config names for the given organization.
func (cr *ConnectionMongoDBRepository) FindByConfigNames(ctx context.Context, organizationID uuid.UUID, configNames []string) ([]*Connection, error) {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_connections_by_config_names")
	defer span.End()

	if len(configNames) == 0 {
		return []*Connection{}, nil
	}

	// Trim and filter empty config names
	trimmedNames := make([]string, 0, len(configNames))
	for _, name := range configNames {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			trimmedNames = append(trimmedNames, trimmed)
		}
	}

	if len(trimmedNames) == 0 {
		return []*Connection{}, nil
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

	connections := make([]*Connection, 0)

	for cur.Next(ctx) {
		var record ConnectionMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to decode connection record", err)
			return nil, err
		}

		connections = append(connections, record.ToEntity())
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to iterate over connections", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("app.response.connections_count", len(connections)))

	return connections, nil
}

// List returns a paginated set of connections for the given organization.
func (cr *ConnectionMongoDBRepository) List(ctx context.Context, filters *ListFilter) ([]*Connection, error) {
	if filters == nil {
		filters = &ListFilter{}
	}

	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.list_connections")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
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
		queryFilter["config_name"] = trimmed
	}

	if len(filters.Types) > 0 {
		queryFilter["type"] = bson.M{"$in": filters.Types}
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = defaultConnectionPageLimit
	}

	if limit > maxConnectionPageLimit {
		limit = maxConnectionPageLimit
	}

	page := filters.Page
	if page <= 0 {
		page = 1
	}

	skip := int64((page - 1) * limit)
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

	connections := make([]*Connection, 0, limit)

	for cur.Next(ctx) {
		var record ConnectionMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to decode connection record", err)
			return nil, err
		}

		connections = append(connections, record.ToEntity())
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to iterate over connections", err)
		return nil, err
	}

	return connections, nil
}

// ConnectionTelemetry models a connection without sensitive fields for telemetry.
type ConnectionTelemetry struct {
	ID             uuid.UUID      `json:"id"`
	OrganizationID uuid.UUID      `json:"organizationId"`
	ConfigName     string         `json:"configName"`
	Type           ConnectionType `json:"type"`
	Host           string         `json:"host"`
	Port           int            `json:"port"`
	DatabaseName   string         `json:"databaseName"`
	Username       string         `json:"username"`
	// PasswordEncrypted OMITIDO
	// SSL OMITIDO
}

// NewConnectionTelemetryFromMongoDBModel creates a telemetry-safe struct from the MongoDB model.
func NewConnectionTelemetryFromMongoDBModel(conn *ConnectionMongoDBModel) *ConnectionTelemetry {
	if conn == nil {
		return nil
	}

	return &ConnectionTelemetry{
		ID:             conn.ID,
		OrganizationID: conn.OrganizationID,
		ConfigName:     conn.ConfigName,
		Type:           conn.Type,
		Host:           conn.Host,
		Port:           conn.Port,
		DatabaseName:   conn.DatabaseName,
		Username:       conn.Username,
	}
}
