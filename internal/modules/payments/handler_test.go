package payments

import (
	"bytes"
	"context"
	"encoding/json"
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

func (m *MockService) ListPayments(ctx context.Context, userID int32, input ListPaymentsInput) (ListPaymentsResult, error) {
	args := m.Called(ctx, userID, input)
	return args.Get(0).(ListPaymentsResult), args.Error(1)
}

func (m *MockService) GetPaymentByID(ctx context.Context, userID int32, paymentID int32) (PaymentDetail, error) {
	args := m.Called(ctx, userID, paymentID)
	return args.Get(0).(PaymentDetail), args.Error(1)
}

// --- UC-16 & UC-17 Tests (Processing) ---

func TestHandler_ProcessPayment(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	
	r := gin.New()
	handler.RegisterRoutes(r.Group(""))

	t.Run("Valid Request", func(t *testing.T) {
		reqBody := RegisterPaymentRequest{
			ContractID:      1,
			PaymentMethodID: 1,
			GatewayID:       1,
			Amount:          500.0,
		}
		
		mockService.On("ProcessPayment", mock.Anything, int32(10), reqBody).Return(PaymentResponse{
			Status: "Success",
		}, nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
		req.Header.Set("X-User-ID", "10")
		w := httptest.NewRecorder()
		
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp PaymentResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "Success", resp.Status)
	})
}

func TestHandler_ConfirmPendingPayment(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	r := gin.New()
	handler.RegisterRoutes(r.Group(""))

	t.Run("Valid Confirmation", func(t *testing.T) {
		pUUID := uuid.New()
		mockService.On("ConfirmPendingPayment", mock.Anything, int32(10), pUUID).Return(nil)

		req, _ := http.NewRequest(http.MethodPatch, "/payments/"+pUUID.String()+"/confirm", nil)
		req.Header.Set("X-User-ID", "10")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "pago confirmado correctamente")
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, "/payments/invalid-uuid/confirm", nil)
		req.Header.Set("X-User-ID", "10")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// --- UC-17 Tests (Consulting) ---

func TestHandler_ListPayments_MissingUserID(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments")
	handler.listPayments(ctx)
	assertErrorResponse(t, recorder, http.StatusUnauthorized, "unauthorized")
}

func TestHandler_ListPayments_ServiceReturnsUnsupportedRole(t *testing.T) {
	mockService := new(MockService)
	mockService.On("ListPayments", mock.Anything, int32(1), mock.Anything).Return(ListPaymentsResult{}, ErrUnsupportedRole)
	
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments")
	ctx.Request.Header.Set("X-User-ID", "1")

	handler := NewHandler(mockService)
	handler.listPayments(ctx)

	assertErrorResponse(t, recorder, http.StatusForbidden, "forbidden")
}

func TestHandler_GetPaymentByID_Success(t *testing.T) {
	mockService := new(MockService)
	mockService.On("GetPaymentByID", mock.Anything, int32(1), int32(1)).Return(PaymentDetail{
		PaymentID: 1,
		ClientID:  7,
		AgentID:   2,
	}, nil)

	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments/1")
	ctx.Request.Header.Set("X-User-ID", "1")
	ctx.Params = gin.Params{{Key: "payment_id", Value: "1"}}

	handler := NewHandler(mockService)
	handler.getPaymentByID(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

// --- Helpers ---

func newHandlerTestContext(method string, target string) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, nil)
	return recorder, ctx
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantError string) {
	if recorder.Code != wantStatus {
		t.Fatalf("status code = %d, want %d", recorder.Code, wantStatus)
	}

	var body struct {
		Error string `json:"error"`
	}
	json.Unmarshal(recorder.Body.Bytes(), &body)

	if body.Error != wantError {
		t.Fatalf("error = %q, want %q", body.Error, wantError)
	}
}
