package properties

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// updateProperty godoc
// @Summary      Update property
// @Description  Update property base data, subtype and location. Only editable fields are allowed.
// @Tags         Properties
// @Accept       json
// @Produce      json
// @Param        uuid     path     string               true  "Property UUID"
// @Param        request  body     UpdatePropertyInput  true  "Update payload"
// @Success      200      {object} UpdatePropertyResult
// @Failure      400      {object} shared.ErrorResponse "Invalid input"
// @Failure      404      {object} shared.ErrorResponse "Property not found"
// @Failure      500      {object} shared.ErrorResponse "Internal error"
// @Router       /api/v1/properties/{uuid} [patch]
func (h *Handler) updateProperty(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("uuid"))
	if propertyUUID == "" {
		shared.BadRequest(c, errors.New("uuid is required"))
		return
	}

	var req UpdatePropertyInput
	if err := rejectForbiddenPayloadFields(c, "category", "subtype"); err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	// Basic validations: title not empty if present, lot_area > 0 if present
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			shared.BadRequest(c, errors.New("title cannot be empty"))
			return
		}
	}
	if req.LotArea != nil {
		if *req.LotArea <= 0 {
			shared.BadRequest(c, errors.New("lot_area must be greater than 0"))
			return
		}
	}

	// Validate residential/commercial/location payload completeness
	if req.Residential != nil {
		// all fields required for residential
		if req.Residential.Bedrooms == nil || req.Residential.Bathrooms == nil || req.Residential.Beds == nil || req.Residential.Floors == nil || req.Residential.ParkingSpots == nil || req.Residential.BuiltArea == nil || req.Residential.ConstructionYear == nil || req.Residential.OrientationID == nil || req.Residential.IsFurnished == nil {
			shared.BadRequest(c, errors.New("all residential fields are required when residential is provided"))
			return
		}
		if *req.Residential.OrientationID <= 0 {
			shared.BadRequest(c, errors.New("residential.orientation_id must be greater than 0"))
			return
		}
	}

	if req.Commercial != nil {
		if req.Commercial.CeilingHeight == nil || req.Commercial.LoadingDocks == nil || req.Commercial.InternalOffices == nil || req.Commercial.ThreePhasePower == nil || req.Commercial.LandUse == nil || strings.TrimSpace(*req.Commercial.LandUse) == "" {
			shared.BadRequest(c, errors.New("all commercial fields are required when commercial is provided"))
			return
		}
	}

	if req.Location != nil {
		if req.Location.CityID == nil || req.Location.Neighborhood == nil || req.Location.Street == nil || req.Location.ExteriorNumber == nil || req.Location.PostalCode == nil || req.Location.Latitude == nil || req.Location.Longitude == nil || req.Location.IsPublicAddress == nil {
			shared.BadRequest(c, errors.New("all location fields except interior_number are required when location is provided"))
			return
		}
	}

	result, err := h.service.UpdateProperty(c.Request.Context(), propertyUUID, req)
	if err != nil {
		var validationErr ValidationError
		if errors.As(err, &validationErr) {
			shared.BadRequest(c, validationErr)
			return
		}

		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}

		log.Printf("update property: %v", err)
		shared.InternalError(c, "could not update property")
		return
	}

	c.JSON(http.StatusOK, result)
}
