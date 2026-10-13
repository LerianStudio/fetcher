package in

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	httpUtils "github.com/LerianStudio/fetcher/v2/pkg/net/http"
	"github.com/LerianStudio/lib-commons/v7/commons"
	"github.com/LerianStudio/lib-commons/v7/commons/net/http/problem"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProblemMiddlewareChain_NormalizesShortCircuitResponse(t *testing.T) {
	app := fiber.New()
	chain := problemMiddlewareChain(func(c fiber.Ctx) error {
		return c.Status(http.StatusUnauthorized).SendString("Missing Token")
	})
	first, trailing := fiberChain(append(chain, func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	}))
	app.Get("/test", first, trailing...)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get(fiber.HeaderContentType))
	var detail problem.Detail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	assert.Equal(t, http.StatusUnauthorized, detail.Status)
	assert.Equal(t, http.StatusText(http.StatusUnauthorized), detail.Title)
	assert.Equal(t, "Missing Token", detail.Detail)
	assert.Empty(t, detail.Type)
	assert.Empty(t, detail.Code)
	assert.Empty(t, detail.Errors)
}

func TestProblemMiddlewareChain_DoesNotRewriteDownstreamProblem(t *testing.T) {
	app := fiber.New()
	chain := problemMiddlewareChain(func(c fiber.Ctx) error { return c.Next() })
	first, trailing := fiberChain(append(chain, func(c fiber.Ctx) error {
		return httpUtils.WithError(c, pkg.ValidationError{
			Code:    constant.ErrBadRequest.Error(),
			Message: "domain validation failed",
		})
	}))
	app.Get("/test", first, trailing...)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

	var detail problem.Detail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrBadRequest.Error(), detail.Type)
	assert.Equal(t, http.StatusText(http.StatusBadRequest), detail.Title)
	assert.Equal(t, http.StatusBadRequest, detail.Status)
	assert.Equal(t, constant.ErrBadRequest.Error(), detail.Code)
	assert.Equal(t, "domain validation failed", detail.Detail)
	assert.Empty(t, detail.Errors)
}

func TestProblemMiddlewareChain_NestedAuthPassDoesNotMaskTenantFailure(t *testing.T) {
	app := fiber.New()
	authChain := problemMiddlewareChain(func(c fiber.Ctx) error {
		return c.Next()
	})
	tenantChain := problemMiddlewareChain(func(c fiber.Ctx) error {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"code":    constant.ErrBadRequest.Error(),
			"message": "tenant header is required",
		})
	})
	handlers := append(authChain, tenantChain...)
	handlers = append(handlers, func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	})
	first, trailing := fiberChain(handlers)
	app.Get("/test", first, trailing...)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get(fiber.HeaderContentType))
	var detail problem.Detail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrBadRequest.Error(), detail.Type)
	assert.Equal(t, http.StatusText(http.StatusBadRequest), detail.Title)
	assert.Equal(t, http.StatusBadRequest, detail.Status)
	assert.Equal(t, constant.ErrBadRequest.Error(), detail.Code)
	assert.Equal(t, "tenant header is required", detail.Detail)
	assert.Empty(t, detail.Errors)
}

// accessManagerRefusalStub mirrors the shape lib-auth v5 refuses with when the
// Access Manager answered with a coded error: a *fiber.Error carrying the status
// and a commons.Response carrying the domain code, both reachable through
// errors.As. The concrete lib-auth type is unexported, so the test reproduces
// its documented contract.
type accessManagerRefusalStub struct {
	fiberErr *fiber.Error
	response commons.Response
}

func (e accessManagerRefusalStub) Error() string { return e.fiberErr.Error() }

func (e accessManagerRefusalStub) Unwrap() []error { return []error{e.fiberErr, e.response} }

func TestProblemMiddlewareChain_MapsReturnedFiberErrorRefusal(t *testing.T) {
	app := fiber.New()
	chain := problemMiddlewareChain(func(fiber.Ctx) error {
		return fiber.NewError(http.StatusForbidden, http.StatusText(http.StatusForbidden))
	})
	first, trailing := fiberChain(append(chain, func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	}))
	app.Get("/test", first, trailing...)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get(fiber.HeaderContentType))
	var detail problem.Detail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	assert.Equal(t, http.StatusForbidden, detail.Status)
	assert.Equal(t, http.StatusText(http.StatusForbidden), detail.Detail)
	assert.Empty(t, detail.Code)
}

func TestProblemMiddlewareChain_KeepsAccessManagerCodeFromReturnedRefusal(t *testing.T) {
	app := fiber.New()
	chain := problemMiddlewareChain(func(fiber.Ctx) error {
		return accessManagerRefusalStub{
			fiberErr: fiber.NewError(http.StatusUnauthorized, "the partner of this credential is suspended"),
			response: commons.Response{
				Code:    "AUT-1009",
				Title:   "Partner Suspended",
				Message: "the partner of this credential is suspended",
			},
		}
	})
	first, trailing := fiberChain(append(chain, func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	}))
	app.Get("/test", first, trailing...)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get(fiber.HeaderContentType))
	var detail problem.Detail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	assert.Equal(t, http.StatusUnauthorized, detail.Status)
	assert.Equal(t, "AUT-1009", detail.Code)
	assert.Equal(t, "the partner of this credential is suspended", detail.Detail)
}
