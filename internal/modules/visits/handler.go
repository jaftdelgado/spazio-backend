package visits

import (
	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
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

func resolveAuthenticatedIdentity(c *gin.Context) (int32, int32, bool) {
	userID, err := middleware.AuthenticatedUserID(c)
	if err != nil {
		shared.Unauthorized(c)
		return 0, 0, false
	}

	roleID, err := middleware.AuthenticatedRoleID(c)
	if err != nil {
		shared.Unauthorized(c)
		return 0, 0, false
	}

	return userID, roleID, true
}
