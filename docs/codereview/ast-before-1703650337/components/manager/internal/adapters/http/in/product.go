package in

import (
	"github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/components/manager/internal/services/query"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	httpUtils "github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type ProductHandler struct {
	CreateCmd *command.CreateProduct
	UpdateCmd *command.UpdateProduct
	DeleteCmd *command.DeleteProduct
	GetQuery  *query.GetProduct
	ListQuery *query.ListProducts
}

func NewProductHandler(
	createCmd *command.CreateProduct,
	updateCmd *command.UpdateProduct,
	deleteCmd *command.DeleteProduct,
	getQuery *query.GetProduct,
	listQuery *query.ListProducts,
) *ProductHandler {
	return &ProductHandler{
		CreateCmd: createCmd,
		UpdateCmd: updateCmd,
		DeleteCmd: deleteCmd,
		GetQuery:  getQuery,
		ListQuery: listQuery,
	}
}

// CreateProduct creates a new product for the organization.
//
//	@Summary		Create product
//	@Description	Create a new product for the organization. Code must be unique per organization and is immutable after creation.
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			Authorization		header		string				false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string				true	"Organization ID"
//	@Param			product				body		model.ProductInput	true	"Product payload"
//	@Success		201					{object}	model.ProductResponse
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		409					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/products [post]
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.create_product")
	defer span.End()

	c.SetUserContext(ctx)

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	var request model.ProductInput
	if errParser := c.BodyParser(&request); errParser != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to parse payload", errParser)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "product",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "unable to parse request body",
			Err:        errParser,
		})
	}

	if request.IsEmpty() {
		err := pkg.ValidationError{
			EntityType: "product",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "empty request body",
		}

		libOpentelemetry.HandleSpanError(&span, "empty request body", err)

		return httpUtils.WithError(c, err)
	}

	product, err := h.CreateCmd.Execute(ctx, orgID, request)
	if err != nil {
		logger.Errorf("Failed to execute create product command, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to create product", err)

		return httpUtils.WithError(c, err)
	}

	resp := model.NewProductResponseFrom(product)
	logger.Infof("product created id=%s org=%s code=%s", resp.ID, orgID, resp.Code)

	return httpUtils.Created(c, resp)
}

// ListProducts lists products for the organization.
//
//	@Summary		List products
//	@Description	List products with pagination and filters.
//	@Tags			Products
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string	true	"Organization ID"
//	@Param			page				query		int		false	"Page number (minimum 1)"	default(1)
//	@Param			limit				query		int		false	"Page size (default 50, max 1000)"	default(50)
//	@Param			sortOrder			query		string	false	"Sort order"											Enums(asc, desc)	default(desc)
//	@Param			startDate			query		string	false	"Filter by start date (YYYY-MM-DD)"
//	@Param			endDate				query		string	false	"Filter by end date (YYYY-MM-DD)"
//	@Success		200					{object}	model.Pagination{items=[]model.ProductResponse,page=int,limit=int,total=int}
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/products [get]
func (h *ProductHandler) ListProducts(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.list_products")
	defer span.End()

	c.SetUserContext(ctx)

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	headerParams, err := httpUtils.ValidateParameters(c.Queries())
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to validate query parameters", err)
		logger.Errorf("Failed to validate query parameters, Error: %s", err.Error())

		return httpUtils.WithError(c, err)
	}

	pagination := model.Pagination{
		Limit: headerParams.Limit,
		Page:  headerParams.Page,
	}

	products, totalCount, err := h.ListQuery.Execute(ctx, orgID, *headerParams)
	if err != nil {
		logger.Errorf("Failed to execute list products query, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to list products", err)

		return httpUtils.WithError(c, err)
	}

	productResp := make([]*model.ProductResponse, 0, len(products))
	for _, p := range products {
		productResp = append(productResp, model.NewProductResponseFrom(p))
	}

	logger.Infof("products listed org=%s count=%d", orgID, len(productResp))

	pagination.SetItems(productResp)
	pagination.SetTotal(int(totalCount))

	return httpUtils.OK(c, pagination)
}

// GetProduct retrieves a product by ID.
//
//	@Summary		Get product
//	@Description	Get product details by ID for the given organization.
//	@Tags			Products
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string	true	"Organization ID"
//	@Param			id					path		string	true	"Product ID"
//	@Success		200					{object}	model.ProductResponse
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/products/{id} [get]
func (h *ProductHandler) GetProduct(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.get_product")
	defer span.End()

	c.SetUserContext(ctx)

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "product",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid product id",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.product_id", id.String()))

	product, err := h.GetQuery.Execute(ctx, orgID, id)
	if err != nil {
		logger.Errorf("Failed to execute get product query, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to get product", err)

		return httpUtils.WithError(c, err)
	}

	resp := model.NewProductResponseFrom(product)

	logger.Infof("product retrieved id=%s org=%s", id, orgID)

	return httpUtils.OK(c, resp)
}

// UpdateProduct partially updates a product.
//
//	@Summary		Update product
//	@Description	Apply a partial update to a product. Code is immutable and cannot be changed. Only include fields you want to change.
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			Authorization		header		string						false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string						true	"Organization ID"
//	@Param			id					path		string						true	"Product ID"
//	@Param			product				body		model.ProductUpdateInput		true	"Fields to update (only include fields you want to change)"
//	@Success		200					{object}	model.ProductResponse
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/products/{id} [patch]
func (h *ProductHandler) UpdateProduct(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.update_product")
	defer span.End()

	c.SetUserContext(ctx)

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "product",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid product id",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.product_id", id.String()))

	var request model.ProductUpdateInput
	if errParser := c.BodyParser(&request); errParser != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to parse payload", errParser)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "product",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "unable to parse request body",
			Err:        errParser,
		})
	}

	if request.IsEmpty() {
		err := pkg.ValidationError{
			EntityType: "product",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "empty request body",
		}

		libOpentelemetry.HandleSpanError(&span, "empty request body", err)

		return httpUtils.WithError(c, err)
	}

	product, err := h.UpdateCmd.Execute(ctx, orgID, id, request)
	if err != nil {
		logger.Errorf("Failed to execute update product command, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to update product", err)

		return httpUtils.WithError(c, err)
	}

	logger.Infof("product updated id=%s org=%s", id, orgID)

	return httpUtils.OK(c, model.NewProductResponseFrom(product))
}

// DeleteProduct performs a soft delete of a product.
//
//	@Summary		Delete product
//	@Description	Soft delete a product. Returns 409 if the product has associated connections.
//	@Tags			Products
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string	true	"Organization ID"
//	@Param			id					path		string	true	"Product ID"
//	@Success		204					"No Content"
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		409					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.delete_product")
	defer span.End()

	c.SetUserContext(ctx)

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "product",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid product id",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.product_id", id.String()))

	if err := h.DeleteCmd.Execute(ctx, orgID, id); err != nil {
		logger.Errorf("Failed to execute delete product command, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to delete product", err)

		return httpUtils.WithError(c, err)
	}

	logger.Infof("product deleted id=%s org=%s", id, orgID)

	return c.Status(fiber.StatusNoContent).Send(nil)
}
