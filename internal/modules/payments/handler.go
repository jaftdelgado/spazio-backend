package payments

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

type Handler struct {
	service PaymentsService
}

func NewHandler(service PaymentsService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/api/v1/payments", h.listPayments)
	r.GET("/api/v1/payments/:payment_id", h.getPaymentByID)
}

// @Description  Returns payments visible to the authenticated user. Admin users can view every payment in the system, agent users can only view payments linked to their own transactions, and client users can only view payments linked to their own transactions. Optional filters by property, payment status, and due date range are applied only when provided.
// @Tags         Payments
// @Produce      json
// @Param        X-User-ID    header    int                          true   "Numeric ID of the authenticated user"
// @Param        property_id  query     int                          false  "Filter by property ID"
// @Param        status_id    query     int                          false  "Filter by payment status ID"
// @Param        date_from    query     string                       false  "Minimum due date in YYYY-MM-DD format"
// @Param        date_to      query     string                       false  "Maximum due date in YYYY-MM-DD format"
// @Param        limit        query     int                          false  "Maximum number of results to return, default 20, max 100" default(20)
// @Param        offset       query     int                          false  "Pagination offset, default 0" default(0)
// @Success      200          {object}  payments.ListPaymentsResult  "Paginated list of payments"

func (h *Handler) listPayments(c *gin.Context) {
	userID, ok := resolveAuthenticatedUserID(c)
	if !ok {
		return
	}

	input, err := resolveListPaymentsInput(c)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	result, err := h.service.ListPayments(c.Request.Context(), userID, input)
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
// @Description  Returns one payment detail when the payment exists and the authenticated user is allowed to access it. Agent and client users receive 403 Forbidden when the payment belongs to a different transaction.
// @Tags         Payments
// @Produce      json
// @Param        X-User-ID   header    int                     true  "Numeric ID of the authenticated user"
// @Param        payment_id  path      int                     true  "Payment ID"
// @Success      200         {object}  payments.PaymentDetail  "Payment detail"
// @Failure      400         {object}  shared.ErrorResponse "Invalid path params"
// @Failure      401         {object}  shared.ErrorResponse "Missing authentication"
// @Failure      403         {object}  shared.ErrorResponse "Forbidden"
// @Failure      404         {object}  shared.ErrorResponse "Payment not found"
// @Failure      500         {object}  shared.ErrorResponse "Internal error"
// @Router       /api/v1/payments/{payment_id} [get]
func (h *Handler) getPaymentByID(c *gin.Context) {
	userID, ok := resolveAuthenticatedUserID(c)
	if !ok {
		return
	}

	paymentID, err := resolveRequiredInt(strings.TrimSpace(c.Param("payment_id")), "payment_id")
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	result, err := h.service.GetPaymentByID(c.Request.Context(), userID, paymentID)
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

func resolveAuthenticatedUserID(c *gin.Context) (int32, bool) {
	rawUserID := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if rawUserID == "" {
		shared.Unauthorized(c)
		return 0, false
	}

	// TODO: connect claims extraction when auth middleware stores a numeric user_id in Gin context.
	// The current codebase only exposes user_uuid from JWT middleware, so this module follows the existing X-User-ID header pattern.
	userID, err := strconv.ParseInt(rawUserID, 10, 32)
	if err != nil {
		shared.BadRequest(c, errors.New("X-User-ID must be a valid integer"))
		return 0, false
	}

	if userID <= 0 {
		shared.BadRequest(c, errors.New("X-User-ID must be a positive integer"))
		return 0, false
	}

	return int32(userID), true
}

func resolveListPaymentsInput(c *gin.Context) (ListPaymentsInput, error) {
	propertyID, err := resolveOptionalInt(c.Query("property_id"), "property_id")
	if err != nil {
		return ListPaymentsInput{}, err
	}

	statusID, err := resolveOptionalInt(c.Query("status_id"), "status_id")
	if err != nil {
		return ListPaymentsInput{}, err
	}

	dateFrom, err := resolveOptionalDate(c.Query("date_from"), "date_from")
	if err != nil {
		return ListPaymentsInput{}, err
	}

	dateTo, err := resolveOptionalDate(c.Query("date_to"), "date_to")
	if err != nil {
		return ListPaymentsInput{}, err
	}

	limit, err := resolveLimit(c.Query("limit"))
	if err != nil {
		return ListPaymentsInput{}, err
	}

	offset, err := resolveOffset(c.Query("offset"))
	if err != nil {
		return ListPaymentsInput{}, err
	}

	if err := validateListPaymentsRequest(limit, offset, dateFrom, dateTo); err != nil {
		return ListPaymentsInput{}, err
	}

	return ListPaymentsInput{
		PropertyID: propertyID,
		StatusID:   statusID,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Limit:      int32(limit),
		Offset:     int32(offset),
	}, nil
}

func resolveRequiredInt(rawValue string, field string) (int32, error) {
	if rawValue == "" {
		return 0, errors.New(field + " is required")
	}

	value, err := strconv.ParseInt(rawValue, 10, 32)
	if err != nil {
		return 0, errors.New(field + " must be a valid integer")
	}

	if value <= 0 {
		return 0, errors.New(field + " must be a positive integer")
	}

	return int32(value), nil
}

func resolveOptionalInt(rawValue string, field string) (*int32, error) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return nil, nil
	}

	value, err := resolveRequiredInt(trimmed, field)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func resolveOptionalDate(rawValue string, field string) (*time.Time, error) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return nil, nil
	}

	value, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, errors.New(field + " must use YYYY-MM-DD format")
	}

	date := value.UTC()
	return &date, nil
}

func resolveLimit(rawValue string) (int, error) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return defaultListLimit, nil
	}

	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, errors.New("limit must be a valid integer")
	}

	return value, nil
}

func resolveOffset(rawValue string) (int, error) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return 0, nil
	}

	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, errors.New("offset must be a valid integer")
	}

	return value, nil
}

func validateListPaymentsRequest(limit int, offset int, dateFrom *time.Time, dateTo *time.Time) error {
	if err := shared.Validate([]shared.ValidationRule{
		{Fail: limit <= 0, Msg: "limit must be greater than 0"},
		{Fail: limit > maxListLimit, Msg: "limit must be less than or equal to 100"},
		{Fail: offset < 0, Msg: "offset must be greater than or equal to 0"},
	}); err != nil {
		return err
	}

	if dateFrom != nil && dateTo != nil && dateTo.Before(*dateFrom) {
		return errors.New("date_to must be greater than or equal to date_from")
	}

	return nil
}
