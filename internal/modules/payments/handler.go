package payments

import (
	"net/http"
	"strconv"

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
	r.POST("/payments", h.processPayment)
	r.PATCH("/payments/:uuid/confirm", h.confirmPendingPayment)
}

// @Summary Confirm a pending payment
// @Description Manually transition a 'Pending' payment (like OXXO) to 'Completed'.
// @Tags payments
// @Accept json
// @Produce json
// @Param uuid path string true "Payment UUID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /payments/{uuid}/confirm [patch]
func (h *Handler) confirmPendingPayment(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	userID, _ := strconv.Atoi(userIDStr)

	uuidStr := c.Param("uuid")
	paymentUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	err = h.service.ConfirmPendingPayment(c.Request.Context(), int32(userID), paymentUUID)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pago confirmado correctamente"})
}

// @Summary Process a payment (Simulated)
// @Description Register a payment for a contract with simulation logic (terminates in 0000 for failure).
// @Tags payments
// @Accept json
// @Produce json
// @Param request body RegisterPaymentRequest true "Payment Details"
// @Success 201 {object} PaymentResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 500 {object} shared.ErrorResponse
// @Router /payments [post]
func (h *Handler) processPayment(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	userID, _ := strconv.Atoi(userIDStr)

	var req RegisterPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	if err := validatePaymentRequest(req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	payment, err := h.service.ProcessPayment(c.Request.Context(), int32(userID), req)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, payment)
}

func validatePaymentRequest(req RegisterPaymentRequest) error {
	return shared.Validate([]shared.ValidationRule{
		{Fail: req.ContractID <= 0, Msg: "contract_id is required"},
		{Fail: req.PaymentMethodID <= 0, Msg: "payment_method_id is required"},
		{Fail: req.GatewayID <= 0, Msg: "gateway_id is required"},
		{Fail: req.Amount <= 0, Msg: "amount must be greater than 0"},
	})
}
