package properties

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// getServices godoc
// @Summary      List property services
// @Description  Returns the service IDs linked to a property by UUID. When the property has no services, the response contains an empty service_ids array.
// @Tags         Properties
// @Produce      json
// @Param        uuid  path     string                   true  "Property UUID"
// @Success      200   {object}  GetPropertyServicesResult  "Property services"
// @Failure      400   {object}  shared.ErrorResponse     "Invalid path parameter"
// @Failure      404   {object}  shared.ErrorResponse     "Property not found"
// @Failure      500   {object}  shared.ErrorResponse     "Internal error"
// @Router       /api/v1/properties/{uuid}/services [get]
func (h *Handler) getServices(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	if _, err := uuid.Parse(propertyUUID); err != nil {
		shared.BadRequest(c, errors.New("uuid must be a valid UUID"))
		return
	}

	result, err := h.service.GetServices(c.Request.Context(), propertyUUID)
	if err != nil {
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("get property services: %v", err)
		shared.InternalError(c, "could not get property services")
		return
	}

	c.JSON(http.StatusOK, result)
}

// updateServices godoc
// @Summary      Replace property services
// @Description  Replaces all services linked to a property by UUID. When the payload omits service_ids or sends an empty array, all linked services are removed.
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Param        uuid     path     string                    true  "Property UUID"
// @Param        request  body     UpdatePropertyServicesInput  true  "Property services payload"
// @Success      204
// @Failure      400      {object}  shared.ErrorResponse     "Invalid input"
// @Failure      404      {object}  shared.ErrorResponse     "Property not found"
// @Failure      500      {object}  shared.ErrorResponse     "Internal error"
// @Router       /api/v1/properties/{uuid}/services [put]
func (h *Handler) updateServices(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	if _, err := uuid.Parse(propertyUUID); err != nil {
		shared.BadRequest(c, errors.New("uuid must be a valid UUID"))
		return
	}

	var req UpdatePropertyServicesInput
	if err := c.ShouldBindJSON(&req); err != nil {
		if !errors.Is(err, io.EOF) {
			shared.BadRequest(c, err)
			return
		}
	}

	if err := h.service.UpdateServices(c.Request.Context(), propertyUUID, req); err != nil {
		var validationErr ValidationError
		if errors.As(err, &validationErr) {
			shared.BadRequest(c, validationErr)
			return
		}

		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("update property services: %v", err)
		shared.InternalError(c, "could not update property services")
		return
	}

	c.Status(http.StatusNoContent)
}
