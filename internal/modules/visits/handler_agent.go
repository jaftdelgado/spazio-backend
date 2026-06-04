package visits

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// @Summary Confirm a visit
// @Description Transition a visit status towards 'Confirmed' by Client or Agent using the authenticated session.
// @Tags Visits
// @Accept json
// @Produce json
// @Param uuid path string true "Visit UUID"
// @Param Authorization header string true "Bearer access token"
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

	userID, roleID, ok := resolveAuthenticatedIdentity(c)
	if !ok {
		return
	}

	err = h.service.ConfirmVisit(c.Request.Context(), userID, roleID, visitUUID)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "visita confirmada correctamente"})
}

// @Summary Complete a visit
// @Description Mark a confirmed visit as completed. Only for Agents or Admin from the authenticated session.
// @Tags Visits
// @Accept json
// @Produce json
// @Param uuid path string true "Visit UUID"
// @Param Authorization header string true "Bearer access token"
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

	userID, roleID, ok := resolveAuthenticatedIdentity(c)
	if !ok {
		return
	}

	err = h.service.CompleteVisit(c.Request.Context(), userID, roleID, visitUUID)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "visita marcada como completada"})
}

// @Summary Cancel a visit
// @Description Transition a visit status to 'Cancelled'. Only allowed before full confirmation.
// @Tags Visits
// @Accept json
// @Produce json
// @Param uuid path string true "Visit UUID"
// @Param Authorization header string true "Bearer access token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/visits/{uuid}/cancel [patch]
func (h *Handler) cancelVisit(c *gin.Context) {
	uuidStr := c.Param("uuid")
	visitUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	userID, roleID, ok := resolveAuthenticatedIdentity(c)
	if !ok {
		return
	}

	err = h.service.CancelVisit(c.Request.Context(), userID, roleID, visitUUID)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "visita cancelada correctamente"})
}
