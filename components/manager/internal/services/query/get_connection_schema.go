package query

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/resolver"
	observability "github.com/LerianStudio/lib-observability"

	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// GetConnectionSchema retrieves the database schema for a connection.
//
// Schema DISCOVERY is delegated to the Engine: after the host resolves the
// connection (internal datasources via the resolver/registry, external via the
// Engine's ID-addressed read), discovery flows through Engine.DiscoverSchema —
// cache-first via the host-wired SchemaCache port, falling back to a live
// connector build. System-table exclusion is preserved in the host connector
// adapter (schemacompat) so it happens BEFORE the snapshot crosses the Engine
// boundary; the Manager only maps the canonical snapshot into the current
// response shape.
type GetConnectionSchema struct {
	resolver           resolver.ConnectionResolver          // nil-safe
	registry           *resolver.InternalDatasourceRegistry // nil-safe
	connectionEngine   *engine.Engine                       // tenant-scope authority + ID-addressed external read
	schemaEngine       *engine.Engine                       // schema discovery authority (cache + connector)
	multiTenantEnabled bool
}

// NewGetConnectionSchema creates a new GetConnectionSchema service.
func NewGetConnectionSchema(
	connResolver resolver.ConnectionResolver,
	dsRegistry *resolver.InternalDatasourceRegistry,
	connectionEng *engine.Engine,
	schemaEng *engine.Engine,
	multiTenantEnabled bool,
) *GetConnectionSchema {
	return &GetConnectionSchema{
		resolver:           connResolver,
		registry:           dsRegistry,
		connectionEngine:   connectionEng,
		schemaEngine:       schemaEng,
		multiTenantEnabled: multiTenantEnabled,
	}
}

// Execute retrieves the database schema for the specified connection.
func (s *GetConnectionSchema) Execute(ctx context.Context, connectionID uuid.UUID) (*model.ConnectionSchemaResponse, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.get_connection_schema")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	conn, err := s.resolveConnection(ctx, connectionID, span)
	if err != nil {
		return nil, err
	}

	// Apply the host's PostgreSQL schema-resolution heuristic BEFORE discovery.
	// Priority: explicit Schema field > username-based (internal multi-tenant
	// connections) > unset (the connector adapter defaults to "public"). The
	// username-as-schema heuristic only holds in multi-tenant deployments where
	// tenant-manager provisions schemas named after the database user; in
	// single-tenant the adapter's "public" default is correct and we leave Schema
	// unset to let it apply.
	applySchemaResolutionHeuristic(conn, s.multiTenantEnabled)

	// GET .../schema is ALWAYS-FRESH: forceRefresh bypasses the schema cache
	// entirely (no read, no write), preserving the pre-embedded-Engine contract
	// where this endpoint reflected the live datasource on every call. ValidateSchema
	// keeps its cache-first path.
	schema, err := discoverSchemaViaEngine(ctx, s.schemaEngine, conn, nil, true)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to discover schema", err)

		// Preserve typed validation errors (e.g. FET-0414 host safety) so they
		// reach the renderer as HTTP 400 instead of being masked by a generic 500.
		var ve pkg.ValidationError
		if errors.As(err, &ve) {
			return nil, err
		}

		return nil, pkg.ResponseError{
			Code:    http.StatusInternalServerError,
			Title:   "Schema Retrieval Error",
			Message: "Failed to retrieve database schema information.",
		}
	}

	// Convert schema to response DTO. System tables are already excluded by the
	// host connector adapter before the snapshot crossed the Engine boundary.
	tables := make([]model.TableDetails, 0)

	for tableName, tableSchema := range schema.Tables {
		fields := tableSchema.GetColumnsList()
		sort.Strings(fields) // Sort for consistent output

		tables = append(tables, model.TableDetails{
			Name:   tableName,
			Fields: fields,
		})
	}

	// Sort tables by name for consistent output
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})

	span.SetAttributes(attribute.Int("app.schema.table_count", len(tables)))

	return model.NewConnectionSchemaFrom(conn, tables), nil
}

// resolveConnection finds a connection by ID, checking internal datasources first
// then the Engine's ID-addressed external read. Internal datasources are a host
// resolver concern (tenant-manager backed) and never flow through the Engine's
// connection store; the external read routes its persistence through the Engine.
func (s *GetConnectionSchema) resolveConnection(ctx context.Context, connectionID uuid.UUID, span trace.Span) (*model.Connection, error) {
	// Route the per-request tenant-scope authority through the Engine before any
	// read (mirrors GetConnection): the internal branch below is a resolver
	// concern, and the external read re-validates scope while resolving through
	// the store.
	if err := connectioncompat.AuthorizeAccess(ctx, s.connectionEngine); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to authorize tenant scope", err)
		return nil, err
	}

	if s.registry != nil && s.resolver != nil {
		tenantID := tmcore.GetTenantIDContext(ctx)
		if configName, _, found := s.registry.FindConfigByID(connectionID, tenantID); found {
			resolved, resolveErr := s.resolver.ResolveInternalByConfigName(ctx, configName)
			if resolveErr != nil {
				libOpentelemetry.HandleSpanError(span, "failed to resolve internal datasource", resolveErr)
				return nil, fmt.Errorf("failed to resolve internal datasource '%s': %w", configName, resolveErr)
			}

			if resolved == nil {
				return nil, pkg.ValidateBusinessError(
					constant.ErrEntityNotFound,
					"connection",
					connectionID,
				)
			}

			return resolved, nil
		}
	}

	conn, findErr := connectioncompat.FindByID(ctx, s.connectionEngine, connectionID.String())
	if findErr != nil {
		libOpentelemetry.HandleSpanError(span, "failed to find connection", findErr)
		return nil, fmt.Errorf("failed to find connection by id: %w", findErr)
	}

	if conn == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	return conn, nil
}

// applySchemaResolutionHeuristic sets the connection's Schema field to the
// username for internal multi-tenant PostgreSQL connections, preserving the
// legacy GetConnectionSchema behavior. The connector adapter reads conn.Schema
// to scope discovery; leaving it unset lets the adapter apply its default.
func applySchemaResolutionHeuristic(conn *model.Connection, multiTenantEnabled bool) {
	if conn.Schema != nil && *conn.Schema != "" {
		return
	}

	if multiTenantEnabled && conn.EncryptionKeyVersion == "" && conn.Username != "" &&
		conn.Type == model.TypePostgreSQL {
		username := conn.Username
		conn.Schema = &username
	}
}
