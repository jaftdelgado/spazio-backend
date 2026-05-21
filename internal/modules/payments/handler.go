package payments

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(protected, public *gin.RouterGroup) {
	protected.POST("/api/v1/payments", h.processPayment)
	protected.PATCH("/api/v1/payments/:uuid/confirm", h.confirmPendingPayment)
	protected.GET("/api/v1/payments", h.listPayments)
	protected.GET("/api/v1/payments/:payment_id", h.getPaymentByID)

	public.POST("/api/v1/payments/webhook", h.handleWebhook)
}

func resolveAuthenticatedUserID(c *gin.Context) (int32, bool) {
	userID, err := middleware.AuthenticatedUserID(c)
	if err != nil {
		shared.Unauthorized(c)
		return 0, false
	}

	return userID, true
}

func resolveAuthenticatedRoleID(c *gin.Context) (int32, bool) {
	roleID, err := middleware.AuthenticatedRoleID(c)
	if err != nil {
		shared.Unauthorized(c)
		return 0, false
	}

	return roleID, true
}

func validatePaymentRequest(req RegisterPaymentRequest) error {
	return shared.Validate([]shared.ValidationRule{
		{Fail: req.ContractID <= 0, Msg: "contract_id is required"},
		{Fail: req.PaymentMethodID <= 0, Msg: "payment_method_id is required"},
		{Fail: req.GatewayID <= 0, Msg: "gateway_id is required"},
		{Fail: req.Amount <= 0, Msg: "amount must be greater than 0"},
		{Fail: req.PayerEmail == "", Msg: "payer_email is required"},
	})
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
