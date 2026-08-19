package in

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	httpUtils "github.com/LerianStudio/fetcher/v2/pkg/net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const jsonContentType = "application/json"

type emptyInput struct{}

type idInput struct {
	ID string `path:"id" doc:"Resource identifier." example:"2f1f35f4-4f50-44f3-a8ab-2772d395f0c2"`
}

type rawJSONInput struct {
	RawBody []byte `contentType:"application/json"`
}

type rawJSONIDInput struct {
	ID      string `path:"id" doc:"Connection identifier." example:"2f1f35f4-4f50-44f3-a8ab-2772d395f0c2"`
	RawBody []byte `contentType:"application/json"`
}

type connectionPage struct {
	Items []*model.ConnectionResponse `json:"items" doc:"Connections returned for this page." example:"[{\"id\":\"018f47a6-3e5f-7b9a-8c1d-2e3f4a5b6c7d\",\"productName\":\"midaz\",\"configName\":\"production-db\",\"type\":\"POSTGRESQL\",\"host\":\"db.example.com\",\"port\":5432,\"databaseName\":\"mydatabase\",\"schema\":\"public\",\"userName\":\"dbuser\",\"ssl\":{\"mode\":\"require\"},\"metadata\":{\"environment\":\"production\",\"region\":\"us-east-1\"},\"createdAt\":\"2026-01-15T10:30:00Z\",\"updatedAt\":\"2026-01-16T14:45:00Z\"}]"`
	Page  int                         `json:"page,omitempty" doc:"Current page number." example:"1"`
	Limit int                         `json:"limit" doc:"Maximum items returned per page." example:"10"`
	Total int                         `json:"total" doc:"Total matching connections." example:"10"`
}

type createConnectionOutput struct {
	Body model.ConnectionResponse
}

type connectionOutput struct {
	Body model.ConnectionResponse
}

type listConnectionsOutput struct {
	Body connectionPage
}

type validateSchemaOutput struct {
	Body model.SchemaValidationResponse
}

type testConnectionOutput struct {
	Body model.ConnectionTestResponse
}

type connectionSchemaOutput struct {
	Body model.ConnectionSchemaResponse
}

type deleteConnectionOutput struct{}

type createJobOutput struct {
	Status int
	Body   model.FetcherResponse
}

type getJobOutput struct {
	Body model.JobResponse
}

// OperationHandlers is the typed callback set shared by runtime and spec-gen.
// A nil callback is valid only for spec generation; the registered schema-only
// handler returns a scrubbed 501 if it is accidentally executed.
type OperationHandlers struct {
	CreateConnection          func(context.Context, *rawJSONInput) (*createConnectionOutput, error)
	ListConnections           func(context.Context, *emptyInput) (*listConnectionsOutput, error)
	ValidateSchema            func(context.Context, *rawJSONInput) (*validateSchemaOutput, error)
	ListUnassignedConnections func(context.Context, *emptyInput) (*listConnectionsOutput, error)
	AssignConnection          func(context.Context, *idInput) (*connectionOutput, error)
	GetConnection             func(context.Context, *idInput) (*connectionOutput, error)
	TestConnection            func(context.Context, *idInput) (*testConnectionOutput, error)
	GetConnectionSchema       func(context.Context, *idInput) (*connectionSchemaOutput, error)
	UpdateConnection          func(context.Context, *rawJSONIDInput) (*connectionOutput, error)
	DeleteConnection          func(context.Context, *idInput) (*deleteConnectionOutput, error)
	CreateJob                 func(context.Context, *rawJSONInput) (*createJobOutput, error)
	GetJob                    func(context.Context, *idInput) (*getJobOutput, error)
}

type OperationMiddlewareFactory func(resource, action string) []fiber.Handler

// fiberChain adapts a non-empty handler chain to Fiber v3's route-registration
// shape, which takes the first handler positionally and the remainder as ...any.
// Execution order is preserved, so auth stays ahead of the tenant and callback
// middlewares.
//
// The remainder is widened one element at a time on purpose: Fiber panics on an
// argument it cannot convert to a Handler, so handing it a []fiber.Handler as a
// single value would fail at route-registration time rather than compile time.
func fiberChain(chain []fiber.Handler) (fiber.Handler, []any) {
	trailing := make([]any, len(chain)-1)
	for i, handler := range chain[1:] {
		trailing[i] = handler
	}

	return chain[0], trailing
}

// NewOperationHandlers binds the HTTP contract to the existing application
// services without retaining the legacy Fiber response rail.
func NewOperationHandlers(
	connectionHandler *ConnectionHandler,
	migrationHandler *MigrationHandler,
	fetcherHandler *FetcherHandler,
) OperationHandlers {
	handlers := OperationHandlers{}
	bindConnectionHandlers(&handlers, connectionHandler)
	bindMigrationHandlers(&handlers, migrationHandler)
	bindFetcherHandlers(&handlers, fetcherHandler)

	return handlers
}

// RegisterHumaOperations registers all twelve client operations. Runtime and
// spec generation call this exact function; only callbacks/middleware differ.
func RegisterHumaOperations(
	app *fiber.App,
	api huma.API,
	handlers OperationHandlers,
	middlewareFactory OperationMiddlewareFactory,
) {
	registerMigrationOperations(app, api, handlers, middlewareFactory)
	registerConnectionOperations(app, api, handlers, middlewareFactory)
	registerFetcherOperations(app, api, handlers, middlewareFactory)
}

func registerTypedOperation[I, O any](
	app *fiber.App,
	api huma.API,
	op huma.Operation,
	resource string,
	action string,
	middlewareFactory OperationMiddlewareFactory,
	handler func(context.Context, *I) (*O, error),
) {
	if middlewareFactory != nil {
		middlewares := middlewareFactory(resource, action)
		if len(middlewares) > 0 {
			first, trailing := fiberChain(middlewares)
			app.Add([]string{op.Method}, fiberPath(op.Path), first, trailing...)
		}
	}

	op.Middlewares = append(op.Middlewares, captureFiberContext)

	if handler == nil {
		handler = func(context.Context, *I) (*O, error) {
			return nil, huma.Error501NotImplemented("operation handler is not configured")
		}
	}

	huma.Register(api, op, handler)
}

// captureFiberContext republishes the Fiber request context under
// fiberContextKey so Huma operation handlers can reach it.
//
// Unwrap (not UnwrapV2) is required: the API is built by the lib-commons Huma
// wrapper, which binds the Fiber v3 adapter, and the v2 unwrap would not match
// the context that adapter creates. The mismatch is invisible to the compiler
// on both sides — UnwrapV2 returns a Fiber v2 Ctx whose Context() is the
// fasthttp request context rather than the Go one, and the fiberContext type
// assertion below would then simply fail at runtime on every request.
func captureFiberContext(humaCtx huma.Context, next func(huma.Context)) {
	fiberCtx := humafiber.Unwrap(humaCtx)
	ctx := context.WithValue(fiberCtx.Context(), fiberContextKey{}, fiberCtx)
	next(huma.WithContext(humaCtx, ctx))
}

type fiberContextKey struct{}

func fiberContext(ctx context.Context) (fiber.Ctx, error) {
	fiberCtx, ok := ctx.Value(fiberContextKey{}).(fiber.Ctx)
	if !ok || fiberCtx == nil {
		return nil, httpUtils.MapError(pkg.InternalServerError{
			Code:    constant.ErrInternalServer.Error(),
			Message: "fiber request context is unavailable",
		})
	}

	return fiberCtx, nil
}

func decodeJSON[T any](raw []byte, entity string) (T, error) {
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, httpUtils.MapError(pkg.ValidationError{
			EntityType: entity,
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "unable to parse request body",
			Err:        err,
		})
	}

	return value, nil
}

func validateStruct(value any) error {
	if err := httpUtils.ValidateStruct(value); err != nil {
		return httpUtils.MapError(err)
	}

	return nil
}

func parseID(raw, entity string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpUtils.MapError(pkg.ValidationError{
			EntityType: entity,
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid " + entity + " id",
			Err:        err,
		})
	}

	return id, nil
}

func mapServiceError(err error) error {
	if err == nil {
		return nil
	}

	return httpUtils.MapError(err)
}

func connectionPageFrom(pagination *model.Pagination) connectionPage {
	page := connectionPage{}
	if pagination == nil {
		return page
	}

	page.Page = pagination.Page
	page.Limit = pagination.Limit
	page.Total = pagination.Total

	switch items := pagination.Items.(type) {
	case []*model.ConnectionResponse:
		page.Items = items
	case []model.ConnectionResponse:
		page.Items = make([]*model.ConnectionResponse, 0, len(items))
		for i := range items {
			page.Items = append(page.Items, &items[i])
		}
	}

	if page.Items == nil {
		page.Items = []*model.ConnectionResponse{}
	}

	return page
}

func schemaFor[T any](api huma.API, hint string) *huma.Schema {
	return api.OpenAPI().Components.Schemas.Schema(reflect.TypeFor[T](), true, hint)
}

func replaceJSONRequestSchema(api huma.API, method, path, description string, schema *huma.Schema) {
	runtimeOperation := operationAt(api, method, path)
	if runtimeOperation == nil {
		return
	}

	// Huma marks every RawBody field as required and rejects a zero-byte body
	// before the callback can map it into Fetcher's FET-0001 error rail. Keep
	// the published operation strict while detaching the runtime operation so
	// an empty body reaches decodeJSON and preserves the service taxonomy.
	documentedOperation := *runtimeOperation
	documentedOperation.RequestBody = &huma.RequestBody{
		Description: description,
		Required:    true,
		Content: map[string]*huma.MediaType{
			jsonContentType: {Schema: schema},
		},
	}

	if runtimeOperation.RequestBody != nil {
		runtimeRequestBody := *runtimeOperation.RequestBody
		runtimeRequestBody.Required = false
		runtimeOperation.RequestBody = &runtimeRequestBody
	}

	setOperationAt(api, method, path, &documentedOperation)
}

func operationAt(api huma.API, method, path string) *huma.Operation {
	item := api.OpenAPI().Paths[path]
	if item == nil {
		return nil
	}

	switch method {
	case http.MethodGet:
		return item.Get
	case http.MethodPost:
		return item.Post
	case http.MethodPatch:
		return item.Patch
	case http.MethodDelete:
		return item.Delete
	default:
		return nil
	}
}

func setOperationAt(api huma.API, method, path string, operation *huma.Operation) {
	item := api.OpenAPI().Paths[path]
	if item == nil {
		return
	}

	switch method {
	case http.MethodGet:
		item.Get = operation
	case http.MethodPost:
		item.Post = operation
	case http.MethodPatch:
		item.Patch = operation
	case http.MethodDelete:
		item.Delete = operation
	}
}

func protectedErrors(statuses ...int) []int {
	result := make([]int, 0, len(statuses)+2)
	seen := make(map[int]struct{}, len(statuses)+2)

	for _, status := range append(statuses, http.StatusUnauthorized, http.StatusForbidden) {
		if _, exists := seen[status]; exists {
			continue
		}

		seen[status] = struct{}{}
		result = append(result, status)
	}

	return result
}

func productNameParameter(required bool) *huma.Param {
	return &huma.Param{
		Name:        "X-Product-Name",
		In:          "header",
		Description: "Product name used for connection isolation.",
		Required:    required,
		Schema:      &huma.Schema{Type: "string", Examples: []any{"midaz"}},
	}
}

func paginationParameters() []*huma.Param {
	return []*huma.Param{
		{Name: "page", In: "query", Description: "Page number; defaults to 1.", Schema: &huma.Schema{Type: "integer", Default: 1, Examples: []any{1}}},
		{Name: "limit", In: "query", Description: "Page size; defaults to 10 and is capped at 100.", Schema: &huma.Schema{Type: "integer", Default: 10, Examples: []any{10}}},
		{Name: "sortOrder", In: "query", Description: "Sort direction.", Schema: &huma.Schema{Type: "string", Default: "desc", Enum: []any{"asc", "desc"}, Examples: []any{"desc"}}},
	}
}

func connectionListParameters() []*huma.Param {
	params := paginationParameters()

	return append(params,
		&huma.Param{Name: "type", In: "query", Description: "Database type filter.", Schema: &huma.Schema{Type: "string", Enum: []any{"ORACLE", "SQL_SERVER", "POSTGRESQL", "MONGODB", "MYSQL"}, Examples: []any{"POSTGRESQL"}}},
		&huma.Param{Name: "startDate", In: "query", Description: "Inclusive creation date in YYYY-MM-DD format.", Schema: &huma.Schema{Type: "string", Examples: []any{"2026-01-01"}}},
		&huma.Param{Name: "endDate", In: "query", Description: "Inclusive final creation date in YYYY-MM-DD format.", Schema: &huma.Schema{Type: "string", Examples: []any{"2026-01-31"}}},
	)
}

func fiberPath(path string) string {
	path = strings.ReplaceAll(path, "{", ":")
	return strings.ReplaceAll(path, "}", "")
}
