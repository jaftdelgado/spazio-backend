package visits

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// @Summary Get property availability
// @Description Get available 1-hour slots for a property on a specific date
// @Tags visits
// @Accept json
// @Produce json
// @Param uuid path int true "Property ID"
// @Param date query string false "Date (YYYY-MM-DD). Defaults to today."
// @Success 200 {array} TimeSlot
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/properties/{uuid}/availability [get]
func (h *Handler) getAvailability(c *gin.Context) {
	idStr := c.Param("uuid")
	propertyID, err := strconv.Atoi(idStr)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	slots, err := h.service.GetAvailableSlots(c.Request.Context(), int32(propertyID), date)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, slots)
}
