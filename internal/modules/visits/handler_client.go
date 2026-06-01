package visits

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// @Summary Schedule a visit
// @Description Create a new visit request in 'Pending' status for the authenticated user.
// @Tags Visits
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Param request body CreateVisitRequest true "Visit Details"
// @Success 201 {object} VisitResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/visits [post]
func (h *Handler) scheduleVisit(c *gin.Context) {
	var req CreateVisitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := validateCreateVisitRequest(req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	userID, roleID, ok := resolveAuthenticatedIdentity(c)
	if !ok {
		return
	}

	_ = roleID

	visit, err := h.service.ScheduleVisit(c.Request.Context(), userID, req.PropertyID, req.VisitDate)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	c.JSON(http.StatusCreated, visit)
}

func validateCreateVisitRequest(req CreateVisitRequest) error {
	return shared.Validate([]shared.ValidationRule{
		{Fail: req.PropertyID <= 0, Msg: "property_id is required"},
		{Fail: req.VisitDate.IsZero(), Msg: "visit_date is required"},
		{Fail: req.VisitDate.Minute() != 0 || req.VisitDate.Second() != 0, Msg: "las citas solo pueden ser agendadas en el minuto :00"},
		{Fail: req.VisitDate.Before(time.Now()), Msg: "visit_date must be in the future"},
	})
}

// @Summary Reschedule a visit
// @Description Cancels the old visit and creates a new one with the new date for the authenticated user.
// @Tags Visits
// @Accept json
// @Produce json
// @Param uuid path string true "Old Visit UUID"
// @Param Authorization header string true "Bearer access token"
// @Param request body CreateVisitRequest true "New Visit Details"
// @Success 201 {object} VisitResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/visits/{uuid}/reschedule [patch]
func (h *Handler) rescheduleVisit(c *gin.Context) {
	uuidStr := c.Param("uuid")
	visitUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	var req CreateVisitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := validateCreateVisitRequest(req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	userID, roleID, ok := resolveAuthenticatedIdentity(c)
	if !ok {
		return
	}

	visit, err := h.service.RescheduleVisit(c.Request.Context(), userID, roleID, visitUUID, req.VisitDate)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	c.JSON(http.StatusCreated, visit)
}
