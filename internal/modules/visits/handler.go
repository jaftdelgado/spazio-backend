package visits

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/api/v1/properties/:uuid/availability", h.getAvailability)
	r.POST("/api/v1/visits", h.scheduleVisit)
	r.GET("/api/v1/visits", h.listVisits)
	r.PATCH("/api/v1/visits/:uuid/confirm", h.confirmVisit)
	r.PATCH("/api/v1/visits/:uuid/reschedule", h.rescheduleVisit)
	r.PATCH("/api/v1/visits/:uuid/complete", h.completeVisit)
}

// @Summary Get property availability
// @Description Get available 1-hour slots for a property on a specific date
// @Tags visits
// @Accept json
// @Produce json
// @Param id path int true "Property ID"
// @Param date query string false "Date (YYYY-MM-DD). Defaults to today."
// @Success 200 {array} TimeSlot
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/properties/{id}/availability [get]
func (h *Handler) getAvailability(c *gin.Context) {
	idStr := c.Param("id")
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

// @Summary Schedule a visit
// @Description Create a new visit request in 'Pending' status. Requires X-User-ID header.
// @Tags visits
// @Accept json
// @Produce json
// @Param X-User-ID header int true "User ID"
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

	userIDStr := c.GetHeader("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		shared.BadRequest(c, errors.New("X-User-ID debe ser un número válido"))
		return
	}

	visit, err := h.service.ScheduleVisit(c.Request.Context(), int32(userID), req.PropertyID, req.VisitDate)
	if err != nil {
		shared.InternalError(c, err.Error())
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

// @Summary List user visits
// @Description List visits according to user role (Admin, Agent, Client). Supports filtering by status, property and date. Requires X-User-ID header.
// @Tags visits
// @Accept json
// @Produce json
// @Param X-User-ID header int true "User ID"
// @Param status_id query int false "Status ID"
// @Param property_id query int false "Property ID"
// @Param date query string false "Date (YYYY-MM-DD)"
// @Success 200 {array} VisitResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/visits [get]
func (h *Handler) listVisits(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		shared.BadRequest(c, errors.New("X-User-ID debe ser un número válido"))
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

	visits, err := h.service.ListUserVisits(c.Request.Context(), int32(userID), filter)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, visits)
}

// @Summary Confirm a visit
// @Description Transition a visit status towards 'Confirmed' by Client or Agent. Requires X-User-ID header.
// @Tags visits
// @Accept json
// @Produce json
// @Param uuid path string true "Visit UUID"
// @Param X-User-ID header int true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/visits/{uuid}/confirm [patch]
func (h *Handler) confirmVisit(c *gin.Context) {
	uuidStr := c.Param("uuid")
	visitUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	userIDStr := c.GetHeader("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		shared.BadRequest(c, errors.New("X-User-ID debe ser un número válido"))
		return
	}

	err = h.service.ConfirmVisit(c.Request.Context(), int32(userID), visitUUID)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "visita confirmada correctamente"})
}

// @Summary Reschedule a visit
// @Description Cancels the old visit and creates a new one with the new date. Requires X-User-ID header.
// @Tags visits
// @Accept json
// @Produce json
// @Param uuid path string true "Old Visit UUID"
// @Param X-User-ID header int true "User ID"
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

	userIDStr := c.GetHeader("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		shared.BadRequest(c, errors.New("X-User-ID debe ser un número válido"))
		return
	}

	visit, err := h.service.RescheduleVisit(c.Request.Context(), int32(userID), visitUUID, req.VisitDate)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, visit)
}

// @Summary Complete a visit
// @Description Mark a confirmed visit as completed. Only for Agents or Admin. Requires X-User-ID header.
// @Tags visits
// @Accept json
// @Produce json
// @Param uuid path string true "Visit UUID"
// @Param X-User-ID header int true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/visits/{uuid}/complete [patch]
func (h *Handler) completeVisit(c *gin.Context) {
	uuidStr := c.Param("uuid")
	visitUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	userIDStr := c.GetHeader("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		shared.BadRequest(c, errors.New("X-User-ID debe ser un número válido"))
		return
	}

	err = h.service.CompleteVisit(c.Request.Context(), int32(userID), visitUUID)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "visita marcada como completada"})
}
