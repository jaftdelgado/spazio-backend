package properties

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// deleteProperty godoc
// @Summary      Delete property
// @Description  Soft deletes a property by UUID. The request must include confirm=true and a valid changed_by_user_id. The operation deletes linked photo objects from storage before applying database updates. Only properties with available status can be deleted.
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Param        uuid     path      string               true  "Property UUID"
// @Param        request  body      DeletePropertyInput  true  "Delete payload"
// @Success      200      {object}  map[string]string    "Property deleted"
// @Failure      400      {object}  shared.ErrorResponse "Invalid input"
// @Failure      404      {object}  shared.ErrorResponse "Property not found"
// @Failure      500      {object}  shared.ErrorResponse "Internal error"
// @Router       /api/v1/properties/{uuid} [delete]
func (h *Handler) deleteProperty(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	var req DeletePropertyInput
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := validateDeletePropertyRequest(req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := h.service.DeleteProperty(c.Request.Context(), propertyUUID, req); err != nil {
		var validationErr ValidationError
		if errors.As(err, &validationErr) {
			shared.BadRequest(c, validationErr)
			return
		}

		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("delete property: %v", err)
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "property deleted successfully"})
}

func validateDeletePropertyRequest(req DeletePropertyInput) error {
	if !req.Confirm {
		return errors.New("confirm must be true")
	}

	if req.ChangedByUserID <= 0 {
		return errors.New("changed_by_user_id must be greater than 0")
	}

	return nil
}
