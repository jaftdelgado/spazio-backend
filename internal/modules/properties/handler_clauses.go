package properties

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// getClauses godoc
// @Summary      List property clauses
// @Description  Returns the clauses linked to a property by UUID. When the property has no clauses, the response contains an empty data array.
// @Tags         Properties
// @Produce      json
// @Param        uuid  path     string                 true  "Property UUID"
// @Success      200   {object}  GetPropertyClausesResult  "Property clauses"
// @Failure      400   {object}  shared.ErrorResponse   "Invalid path parameter"
// @Failure      404   {object}  shared.ErrorResponse   "Property not found"
// @Failure      500   {object}  shared.ErrorResponse   "Internal error"
// @Router       /api/v1/properties/{uuid}/clauses [get]
func (h *Handler) getClauses(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	if _, err := uuid.Parse(propertyUUID); err != nil {
		shared.BadRequest(c, errors.New("uuid must be a valid UUID"))
		return
	}

	result, err := h.service.GetClauses(c.Request.Context(), propertyUUID)
	if err != nil {
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("get property clauses: %v", err)
		shared.InternalError(c, "could not get property clauses")
		return
	}

	c.JSON(http.StatusOK, result)
}

// updateClauses godoc
// @Summary      Replace property clauses
// @Description  Replaces all clauses linked to a property by UUID. When the payload omits clauses or sends an empty array, all linked clauses are removed.
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Param        uuid     path     string                   true  "Property UUID"
// @Param        request   body     UpdatePropertyClausesInput  true  "Property clauses payload"
// @Success      204
// @Failure      400      {object}  shared.ErrorResponse   "Invalid input"
// @Failure      404      {object}  shared.ErrorResponse   "Property not found"
// @Failure      500      {object}  shared.ErrorResponse   "Internal error"
// @Router       /api/v1/properties/{uuid}/clauses [put]
func (h *Handler) updateClauses(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	if _, err := uuid.Parse(propertyUUID); err != nil {
		shared.BadRequest(c, errors.New("uuid must be a valid UUID"))
		return
	}

	var req UpdatePropertyClausesInput
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := h.service.UpdateClauses(c.Request.Context(), propertyUUID, req); err != nil {
		var validationErr ValidationError
		if errors.As(err, &validationErr) {
			shared.BadRequest(c, validationErr)
			return
		}

		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("update property clauses: %v", err)
		shared.InternalError(c, "could not update property clauses")
		return
	}

	c.Status(http.StatusNoContent)
}
