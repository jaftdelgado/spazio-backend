package properties

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// getProperty godoc
// @Summary      Get property by UUID
// @Description  Returns property base data, subtype and location for the given UUID. Subtype-specific data included depends on subtype.
// @Tags         Properties
// @Produce      json
// @Param        uuid  path     string                 true  "Property UUID"
// @Success      200   {object}  GetPropertyResult    "Property data"
// @Failure      400   {object}  shared.ErrorResponse "Invalid path parameter"
// @Failure      404   {object}  shared.ErrorResponse "Property not found"
// @Failure      500   {object}  shared.ErrorResponse "Internal error"
// @Router       /api/v1/properties/{uuid} [get]
func (h *Handler) getProperty(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	result, err := h.service.GetProperty(c.Request.Context(), propertyUUID)
	if err != nil {
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("get property: %v", err)
		shared.InternalError(c, "could not get property")
		return
	}

	c.JSON(http.StatusOK, result)
}
