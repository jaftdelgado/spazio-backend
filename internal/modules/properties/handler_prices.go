package properties

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// getPrices godoc
// @Summary      Get property prices
// @Description  Returns the active prices of a property by UUID. Sale price and rent prices include only currently active records (is_current=true). Guests and clients can only view prices from available properties.
// @Tags         Properties
// @Produce      json
// @Param        uuid  path      string                   true  "Property UUID"
// @Success      200   {object}  GetPropertyPricesResult  "Property prices"
// @Failure      400   {object}  shared.ErrorResponse     "Invalid path parameter"
// @Failure      403   {object}  shared.ErrorResponse     "Forbidden"
// @Failure      404   {object}  shared.ErrorResponse     "Property not found"
// @Failure      500   {object}  shared.ErrorResponse     "Internal error"
// @Router       /api/v1/properties/{uuid}/prices [get]
func (h *Handler) getPrices(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	if _, err := uuid.Parse(propertyUUID); err != nil {
		shared.BadRequest(c, errors.New("uuid must be a valid UUID"))
		return
	}

	if !h.ensureReadableProperty(c, propertyUUID) {
		return
	}

	result, err := h.service.GetPrices(c.Request.Context(), propertyUUID)
	if err != nil {
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		shared.InternalError(c, "could not get property prices")
		return
	}

	c.JSON(http.StatusOK, result)
}

// updatePrices godoc
// @Summary      Update property prices
// @Description  Updates property prices by UUID. Only processes prices in the payload; unmodified prices remain unchanged. When the amount of a price changes, a new price record is created and the old one is marked as inactive. Amounts must be greater than 0. Currency is not editable.
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid     path      string                     true  "Property UUID"
// @Param        request  body      UpdatePropertyPricesInput  true  "Property prices payload"
// @Success      204
// @Failure      400      {object}  shared.ErrorResponse       "Invalid input"
// @Failure      401      {object}  shared.ErrorResponse       "Missing or invalid authenticated session"
// @Failure      403      {object}  shared.ErrorResponse       "Forbidden"
// @Failure      404      {object}  shared.ErrorResponse       "Property not found"
// @Failure      500      {object}  shared.ErrorResponse       "Internal error"
// @Router       /api/v1/properties/{uuid}/prices [put]
func (h *Handler) updatePrices(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	if _, err := uuid.Parse(propertyUUID); err != nil {
		shared.BadRequest(c, errors.New("uuid must be a valid UUID"))
		return
	}

	var req UpdatePropertyPricesInput
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	attachActorContext(c, &req.Actor)

	if err := h.service.UpdatePrices(c.Request.Context(), propertyUUID, req); err != nil {
		var validationErr ValidationError
		if errors.As(err, &validationErr) {
			shared.BadRequest(c, validationErr)
			return
		}

		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		shared.InternalError(c, "could not update property prices")
		return
	}

	c.Status(http.StatusNoContent)
}
