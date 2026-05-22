package payments

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

// listPayments godoc
// @Summary      List payments
// @Description  Returns payments visible to the authenticated user resolved from the bearer token. Supports filtering by property, status, due date range, and offset pagination.
// @Tags         Payments
// @Produce      json
// @Param        Authorization  header    string              true   "Bearer access token"
// @Param        property_id    query     int                 false  "Filter by property ID"
// @Param        status_id      query     int                 false  "Filter by payment status ID"
// @Param        date_from      query     string              false  "Minimum due date in YYYY-MM-DD format"
// @Param        date_to        query     string              false  "Maximum due date in YYYY-MM-DD format"
// @Param        limit          query     int                 false  "Maximum number of results to return" default(20)
// @Param        offset         query     int                 false  "Pagination offset" default(0)
// @Success      200            {object}  ListPaymentsResult  "List of payments"
// @Failure      400            {object}  shared.ErrorResponse "Invalid query params"
// @Failure      401            {object}  shared.ErrorResponse "Invalid or expired session"
// @Failure      403            {object}  shared.ErrorResponse "Unsupported role or forbidden scope"
// @Failure      500            {object}  shared.ErrorResponse "Internal error"
// @Router       /api/v1/payments [get]
func (h *Handler) listPayments(c *gin.Context) {
	userID, ok := resolveAuthenticatedUserID(c)
	if !ok {
		return
	}
	roleID, ok := resolveAuthenticatedRoleID(c)
	if !ok {
		return
	}

	input, err := resolveListPaymentsInput(c)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	result, err := h.service.ListPayments(c.Request.Context(), userID, roleID, input)
	if err != nil {
		if errors.Is(err, ErrPaymentForbidden) || errors.Is(err, ErrUnsupportedRole) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		shared.InternalError(c, "could not list payments")
		return
	}

	c.JSON(http.StatusOK, result)
}

// getPaymentByID godoc
// @Summary      Get payment detail
// @Description  Returns one payment detail visible to the authenticated user resolved from the bearer token. The path parameter is the public payment UUID.
// @Tags         Payments
// @Produce      json
// @Param        Authorization  header    string                 true  "Bearer access token"
// @Param        payment_uuid   path      string                 true  "Payment UUID"
// @Success      200            {object}  PaymentDetailResponse  "Payment detail"
// @Failure      400            {object}  shared.ErrorResponse   "Invalid payment UUID"
// @Failure      401            {object}  shared.ErrorResponse   "Invalid or expired session"
// @Failure      403            {object}  shared.ErrorResponse   "Forbidden"
// @Failure      404            {object}  shared.ErrorResponse   "Payment not found"
// @Failure      500            {object}  shared.ErrorResponse   "Internal error"
// @Router       /api/v1/payments/{payment_uuid} [get]
func (h *Handler) getPaymentByID(c *gin.Context) {
	userID, ok := resolveAuthenticatedUserID(c)
	if !ok {
		return
	}
	roleID, ok := resolveAuthenticatedRoleID(c)
	if !ok {
		return
	}

	paymentUUID, err := resolveRequiredUUID(strings.TrimSpace(c.Param("payment_uuid")), "payment_uuid")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	result, err := h.service.GetPaymentByUUID(c.Request.Context(), userID, roleID, paymentUUID)
	if err != nil {
		switch {
		case errors.Is(err, ErrPaymentNotFound):
			shared.NotFound(c, err.Error())
		case errors.Is(err, ErrPaymentForbidden), errors.Is(err, ErrUnsupportedRole):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			shared.InternalError(c, "could not get payment")
		}
		return
	}

	c.JSON(http.StatusOK, result)
}
