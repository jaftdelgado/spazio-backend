package visits

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// @Summary List user visits
// @Description List visits according to the authenticated user role (Admin, Agent, Client). Supports filtering by status, property and date.
// @Tags visits
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Param status_id query int false "Status ID"
// @Param property_id query int false "Property ID"
// @Param date query string false "Date (YYYY-MM-DD)"
// @Success 200 {array} VisitResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/visits [get]
func (h *Handler) listVisits(c *gin.Context) {
	userID, roleID, ok := resolveAuthenticatedIdentity(c)
	if !ok {
		return
	}

	filter := ListVisitsFilter{}
	if sID := c.Query("status_id"); sID != "" {
		val, _ := strconv.Atoi(sID)
		v32 := int32(val)
		filter.StatusID = &v32
	}
	if pID := c.Query("property_id"); pID != "" {
		val, _ := strconv.Atoi(pID)
		v32 := int32(val)
		filter.PropertyID = &v32
	}
	if dateStr := c.Query("date"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			filter.Date = &t
		}
	}

	visits, err := h.service.ListUserVisits(c.Request.Context(), userID, roleID, filter)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, visits)
}
