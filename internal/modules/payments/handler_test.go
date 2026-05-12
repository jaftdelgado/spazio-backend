package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

type MockService struct {
	mock.Mock
}

func (m *MockService) ProcessPayment(ctx context.Context, userID int32, req RegisterPaymentRequest) (PaymentResponse, error) {
	args := m.Called(ctx, userID, req)
	return args.Get(0).(PaymentResponse), args.Error(1)
}

func (m *MockService) ConfirmPendingPayment(ctx context.Context, userID int32, paymentUUID uuid.UUID) error {
	args := m.Called(ctx, userID, paymentUUID)
	return args.Error(0)
}

func (m *MockService) HandleWebhook(ctx context.Context, xSignature string, xRequestID string, body []byte) error {
	args := m.Called(ctx, xSignature, xRequestID, body)
	return args.Error(0)
}

func (m *MockService) ListPayments(ctx context.Context, userID int32, input ListPaymentsInput) (ListPaymentsResult, error) {
	args := m.Called(ctx, userID, input)
	return args.Get(0).(ListPaymentsResult), args.Error(1)
}

func (m *MockService) GetPaymentByID(ctx context.Context, userID int32, paymentID int32) (PaymentDetail, error) {
	args := m.Called(ctx, userID, paymentID)
	return args.Get(0).(PaymentDetail), args.Error(1)
}

func TestHandler_ProcessPayment(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	r := gin.New()
	handler.RegisterRoutes(r.Group(""), r.Group(""))

	t.Run("Valid Request", func(t *testing.T) {
		reqBody := RegisterPaymentRequest{
			ContractID:      1,
			PaymentMethodID: 1,
			GatewayID:       1,
			Amount:          500.0,
			Currency:        "MXN",
			PayerEmail:      "test@example.com",
		}

		mockService.On("ProcessPayment", mock.Anything, int32(10), reqBody).Return(PaymentResponse{
			Status: "Success",
		}, nil).Once()

		body, _ := json.Marshal(reqBody)
		recorder, ctx := newHandlerTestContext(http.MethodPost, "/api/v1/payments")
		ctx.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBuffer(body))
		ctx.Request.Header.Set("X-User-ID", "10")

		handler.processPayment(ctx)

		assert.Equal(t, http.StatusCreated, recorder.Code)
	})

	t.Run("Validation Error", func(t *testing.T) {
		reqBody := RegisterPaymentRequest{
			ContractID: 1,
		}

		body, _ := json.Marshal(reqBody)
		recorder, ctx := newHandlerTestContext(http.MethodPost, "/api/v1/payments")
		ctx.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBuffer(body))
		ctx.Request.Header.Set("X-User-ID", "10")

		handler.processPayment(ctx)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		reqBody := RegisterPaymentRequest{
			ContractID: 1, PaymentMethodID: 1, GatewayID: 1, Amount: 10, Currency: "MXN", PayerEmail: "a@a.com",
		}
		mockService.On("ProcessPayment", mock.Anything, int32(10), reqBody).Return(PaymentResponse{}, errors.New("fail")).Once()

		body, _ := json.Marshal(reqBody)
		recorder, ctx := newHandlerTestContext(http.MethodPost, "/api/v1/payments")
		ctx.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBuffer(body))
		ctx.Request.Header.Set("X-User-ID", "10")

		handler.processPayment(ctx)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("Unauthorized - Missing User ID", func(t *testing.T) {
		recorder, ctx := newHandlerTestContext(http.MethodPost, "/api/v1/payments")
		handler.processPayment(ctx)
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("Bad Request - Non-numeric User ID", func(t *testing.T) {
		recorder, ctx := newHandlerTestContext(http.MethodPost, "/api/v1/payments")
		ctx.Request.Header.Set("X-User-ID", "abc")
		handler.processPayment(ctx)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Bad Request - Zero User ID", func(t *testing.T) {
		recorder, ctx := newHandlerTestContext(http.MethodPost, "/api/v1/payments")
		ctx.Request.Header.Set("X-User-ID", "0")
		handler.processPayment(ctx)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestHandler_ConfirmPendingPayment(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	t.Run("Valid Confirmation", func(t *testing.T) {
		pUUID := uuid.New()
		mockService.On("ConfirmPendingPayment", mock.Anything, int32(10), pUUID).Return(nil).Once()

		recorder, ctx := newHandlerTestContext(http.MethodPatch, "/api/v1/payments/"+pUUID.String()+"/confirm")
		ctx.Request.Header.Set("X-User-ID", "10")
		ctx.Params = gin.Params{{Key: "uuid", Value: pUUID.String()}}

		handler.confirmPendingPayment(ctx)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		recorder, ctx := newHandlerTestContext(http.MethodPatch, "/api/v1/payments/invalid/confirm")
		ctx.Request.Header.Set("X-User-ID", "10")
		ctx.Params = gin.Params{{Key: "uuid", Value: "invalid"}}

		handler.confirmPendingPayment(ctx)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		pUUID := uuid.New()
		mockService.On("ConfirmPendingPayment", mock.Anything, int32(10), pUUID).Return(errors.New("fail")).Once()

		recorder, ctx := newHandlerTestContext(http.MethodPatch, "/api/v1/payments/"+pUUID.String()+"/confirm")
		ctx.Request.Header.Set("X-User-ID", "10")
		ctx.Params = gin.Params{{Key: "uuid", Value: pUUID.String()}}

		handler.confirmPendingPayment(ctx)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestHandler_HandleWebhook(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		mockService.On("HandleWebhook", mock.Anything, "sig", "reqid", mock.Anything).Return(nil).Once()

		recorder, ctx := newHandlerTestContext(http.MethodPost, "/api/v1/payments/webhook")
		ctx.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewBuffer([]byte(`{}`)))
		ctx.Request.Header.Set("x-signature", "sig")
		ctx.Request.Header.Set("x-request-id", "reqid")

		handler.handleWebhook(ctx)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("Service Error Returns 200 with status error", func(t *testing.T) {
		mockService.On("HandleWebhook", mock.Anything, "sig", "reqid", mock.Anything).Return(errors.New("fail")).Once()

		recorder, ctx := newHandlerTestContext(http.MethodPost, "/api/v1/payments/webhook")
		ctx.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewBuffer([]byte(`{}`)))
		ctx.Request.Header.Set("x-signature", "sig")
		ctx.Request.Header.Set("x-request-id", "reqid")

		handler.handleWebhook(ctx)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestHandler_ListPayments(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	t.Run("Success with Full Filters", func(t *testing.T) {
		mockService.On("ListPayments", mock.Anything, int32(1), mock.Anything).Return(ListPaymentsResult{}, nil).Once()
		recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments?property_id=1&status_id=1&date_from=2024-01-01&date_to=2024-12-31&limit=50&offset=10")
		ctx.Request.Header.Set("X-User-ID", "1")
		handler.listPayments(ctx)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("Invalid date_from", func(t *testing.T) {
		recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments?date_from=invalid")
		ctx.Request.Header.Set("X-User-ID", "1")
		handler.listPayments(ctx)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Invalid limit", func(t *testing.T) {
		recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments?limit=abc")
		ctx.Request.Header.Set("X-User-ID", "1")
		handler.listPayments(ctx)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Limit too high", func(t *testing.T) {
		recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments?limit=1000")
		ctx.Request.Header.Set("X-User-ID", "1")
		handler.listPayments(ctx)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Date range error", func(t *testing.T) {
		recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments?date_from=2024-12-31&date_to=2024-01-01")
		ctx.Request.Header.Set("X-User-ID", "1")
		handler.listPayments(ctx)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		mockService.On("ListPayments", mock.Anything, int32(1), mock.Anything).Return(ListPaymentsResult{}, errors.New("fail")).Once()
		recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments")
		ctx.Request.Header.Set("X-User-ID", "1")
		handler.listPayments(ctx)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestHandler_GetPaymentByID_Errors(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	t.Run("Not Found", func(t *testing.T) {
		mockService.On("GetPaymentByID", mock.Anything, int32(1), int32(1)).Return(PaymentDetail{}, ErrPaymentNotFound).Once()
		recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments/1")
		ctx.Request.Header.Set("X-User-ID", "1")
		ctx.Params = gin.Params{{Key: "payment_id", Value: "1"}}
		handler.getPaymentByID(ctx)
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("Internal Error", func(t *testing.T) {
		mockService.On("GetPaymentByID", mock.Anything, int32(1), int32(1)).Return(PaymentDetail{}, errors.New("db fail")).Once()
		recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments/1")
		ctx.Request.Header.Set("X-User-ID", "1")
		ctx.Params = gin.Params{{Key: "payment_id", Value: "1"}}
		handler.getPaymentByID(ctx)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("Bad ID Parameter", func(t *testing.T) {
		recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments/abc")
		ctx.Request.Header.Set("X-User-ID", "1")
		ctx.Params = gin.Params{{Key: "payment_id", Value: "abc"}}
		handler.getPaymentByID(ctx)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Forbidden Error", func(t *testing.T) {
		mockService.On("GetPaymentByID", mock.Anything, int32(1), int32(1)).Return(PaymentDetail{}, ErrPaymentForbidden).Once()
		recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments/1")
		ctx.Request.Header.Set("X-User-ID", "1")
		ctx.Params = gin.Params{{Key: "payment_id", Value: "1"}}
		handler.getPaymentByID(ctx)
		assert.Equal(t, http.StatusForbidden, recorder.Code)
	})
}

func newHandlerTestContext(method string, target string) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request, _ = http.NewRequest(method, target, nil)
	return recorder, ctx
}
