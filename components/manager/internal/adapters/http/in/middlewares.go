package in

import (
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var (
	UUIDPathParameter    = "id"
	OrgIDHeaderParameter = "X-Organization-Id"
)

// ParsePathParametersUUID convert and validate if the path parameter is UUID
func ParsePathParametersUUID(c *fiber.Ctx) error {
	pathParam := c.Params(UUIDPathParameter)

	if pkg.IsNilOrEmpty(&pathParam) {
		err := pkg.ValidateBusinessError(constant.ErrInvalidPathParameter, "", UUIDPathParameter)
		return http.WithError(c, err)
	}

	parsedPathUUID, errPath := uuid.Parse(pathParam)
	if errPath != nil {
		err := pkg.ValidateBusinessError(constant.ErrInvalidPathParameter, "", UUIDPathParameter)
		return http.WithError(c, err)
	}

	c.Locals(UUIDPathParameter, parsedPathUUID)

	return c.Next()
}

// ParseHeaderParameters convert and validate if the header parameters is UUID
func ParseHeaderParameters(c *fiber.Ctx) error {
	headerParam := c.Get(OrgIDHeaderParameter)

	if pkg.IsNilOrEmpty(&headerParam) {
		err := pkg.ValidateBusinessError(constant.ErrInvalidHeaderParameter, "", OrgIDHeaderParameter)
		return http.WithError(c, err)
	}

	parsedHeaderUUID, errHeader := uuid.Parse(headerParam)
	if errHeader != nil {
		err := pkg.ValidateBusinessError(constant.ErrInvalidHeaderParameter, "", OrgIDHeaderParameter)
		return http.WithError(c, err)
	}

	c.Locals(OrgIDHeaderParameter, parsedHeaderUUID)

	return c.Next()
}
