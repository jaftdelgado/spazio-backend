package payments

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// @Summary Confirm a pending payment
// @Description Manually transition a 'Pending' payment (like OXXO) to 'Completed'.
// @Tags Payments
// @Accept json
// @Produce json
// @Param uuid path string true "Payment UUID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/payments/{uuid}/confirm [patch]
func (h *Handler) confirmPendingPayment(c *gin.Context) {
	userID, ok := resolveAuthenticatedUserID(c)
	if !ok {
		return
	}

	uuidStr := c.Param("uuid")
	paymentUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	err = h.service.ConfirmPendingPayment(c.Request.Context(), userID, paymentUUID)
	if err != nil {
		// F6: Proper error mapping
		errMsg := err.Error()
		if strings.Contains(errMsg, "encontrado") {
			shared.NotFound(c, errMsg)
			return
		}
		if strings.Contains(errMsg, "no autorizada") {
			shared.Forbidden(c, errMsg)
			return
		}
		if strings.Contains(errMsg, "estado pendiente") || strings.Contains(errMsg, "expirado") {
			shared.BadRequest(c, err)
			return
		}
		shared.InternalError(c, errMsg)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pago confirmado correctamente"})
}
