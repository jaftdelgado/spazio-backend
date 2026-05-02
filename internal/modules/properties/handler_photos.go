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

// getPhotos godoc
// @Summary      List property photos
// @Description  Returns the photo metadata linked to a property by UUID ordered by sort_order ascending. When the property has no photos, the response contains an empty data array.
// @Tags         Properties
// @Produce      json
// @Param        uuid  path     string                   true  "Property UUID"
// @Success      200   {object}  GetPropertyPhotosResult  "Property photos"
// @Failure      400   {object}  shared.ErrorResponse     "Invalid path parameter"
// @Failure      404   {object}  shared.ErrorResponse     "Property not found"
// @Failure      500   {object}  shared.ErrorResponse     "Internal error"
// @Router       /api/v1/properties/{uuid}/photos [get]
func (h *Handler) getPhotos(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	if _, err := uuid.Parse(propertyUUID); err != nil {
		shared.BadRequest(c, errors.New("uuid must be a valid UUID"))
		return
	}

	result, err := h.service.GetPhotos(c.Request.Context(), propertyUUID)
	if err != nil {
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("get property photos: %v", err)
		shared.InternalError(c, "could not get property photos")
		return
	}

	c.JSON(http.StatusOK, result)
}

// updatePhotos godoc
// @Summary      Replace property photos
// @Description  Replaces the stored metadata of the photos linked to a property by UUID. Exactly one photo must be marked as cover when the payload contains photos. When the payload omits photos or sends an empty array, all linked photos are removed.
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Param        uuid     path     string                     true  "Property UUID"
// @Param        request  body     UpdatePropertyPhotosInput  true  "Property photos payload"
// @Success      204
// @Failure      400      {object}  shared.ErrorResponse  "Invalid input"
// @Failure      404      {object}  shared.ErrorResponse  "Property not found"
// @Failure      500      {object}  shared.ErrorResponse  "Internal error"
// @Router       /api/v1/properties/{uuid}/photos [put]
func (h *Handler) updatePhotos(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	if _, err := uuid.Parse(propertyUUID); err != nil {
		shared.BadRequest(c, errors.New("uuid must be a valid UUID"))
		return
	}

	var req UpdatePropertyPhotosInput
	if err := c.ShouldBindJSON(&req); err != nil {
		if !errors.Is(err, io.EOF) {
			shared.BadRequest(c, err)
			return
		}
	}

	if err := h.service.UpdatePhotos(c.Request.Context(), propertyUUID, req); err != nil {
		var validationErr ValidationError
		if errors.As(err, &validationErr) {
			shared.BadRequest(c, validationErr)
			return
		}

		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("update property photos: %v", err)
		shared.InternalError(c, "could not update property photos")
		return
	}

	c.Status(http.StatusNoContent)
}
