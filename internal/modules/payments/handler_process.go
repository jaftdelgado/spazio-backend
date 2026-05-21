package payments

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// @Summary Process a payment (Simulated)
// @Description Register a payment for a contract with simulation logic (terminates in 0000 for failure).
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body RegisterPaymentRequest true "Payment Details"
// @Success 201 {object} PaymentResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /api/v1/payments [post]
func (h *Handler) processPayment(c *gin.Context) {
	userID, ok := resolveAuthenticatedUserID(c)
	if !ok {
		return
	}

	var req RegisterPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := validatePaymentRequest(req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	payment, err := h.service.ProcessPayment(c.Request.Context(), userID, req)
	if err != nil {
		// F6: Proper error mapping for business logic failures
		errMsg := err.Error()
		if strings.Contains(errMsg, "no pertenece") || strings.Contains(errMsg, "no autorizada") {
			shared.Forbidden(c, errMsg)
			return
		}
		// Availability, currency, amount, etc.
		if strings.Contains(errMsg, "no coincide") || strings.Contains(errMsg, "disponible") || strings.Contains(errMsg, "pasado") || strings.Contains(errMsg, "bloqueado") || strings.Contains(errMsg, "terminado") {
			shared.BadRequest(c, err)
			return
		}
		shared.InternalError(c, errMsg)
		return
	}

	c.JSON(http.StatusCreated, payment)
}
