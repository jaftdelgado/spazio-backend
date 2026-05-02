package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func TestHandler_ProcessPayment(t *testing.T) {
	gin.SetMode(gin.TestMode)
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
