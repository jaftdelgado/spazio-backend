package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// mockPaymentService is a manual mock implementation of the Service interface.
type mockPaymentService struct {
	processPaymentFunc        func(ctx context.Context, userID int32, req RegisterPaymentRequest) (PaymentResponse, error)
	confirmPendingPaymentFunc func(ctx context.Context, userID int32, paymentUUID uuid.UUID) error
	handleWebhookFunc         func(ctx context.Context, xSignature string, xRequestID string, body []byte) error
	listPaymentsFunc          func(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) (ListPaymentsResult, error)
	getPaymentByUUIDFunc      func(ctx context.Context, userID int32, roleID int32, paymentUUID uuid.UUID) (PaymentDetailResponse, error)
}

func (m *mockPaymentService) ProcessPayment(ctx context.Context, userID int32, req RegisterPaymentRequest) (PaymentResponse, error) {
	if m.processPaymentFunc != nil {
		return m.processPaymentFunc(ctx, userID, req)
	}
	return PaymentResponse{}, nil
}

func (m *mockPaymentService) ConfirmPendingPayment(ctx context.Context, userID int32, paymentUUID uuid.UUID) error {
	if m.confirmPendingPaymentFunc != nil {
		return m.confirmPendingPaymentFunc(ctx, userID, paymentUUID)
	}
	return nil
}

func (m *mockPaymentService) HandleWebhook(ctx context.Context, xSignature string, xRequestID string, body []byte) error {
	if m.handleWebhookFunc != nil {
		return m.handleWebhookFunc(ctx, xSignature, xRequestID, body)
	}
	return nil
}

func (m *mockPaymentService) ListPayments(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) (ListPaymentsResult, error) {
	if m.listPaymentsFunc != nil {
		return m.listPaymentsFunc(ctx, userID, roleID, input)
	}
	return ListPaymentsResult{}, nil
}

func (m *mockPaymentService) GetPaymentByUUID(ctx context.Context, userID int32, roleID int32, paymentUUID uuid.UUID) (PaymentDetailResponse, error) {
	if m.getPaymentByUUIDFunc != nil {
		return m.getPaymentByUUIDFunc(ctx, userID, roleID, paymentUUID)
	}
	return PaymentDetailResponse{}, nil
}

func newHandlerTestContext(method string, target string) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request, _ = http.NewRequest(method, target, nil)
	return recorder, ctx
}

func setAuthenticatedContext(ctx *gin.Context, userID int32, roleID int32) {
	ctx.Set("user_id", userID)
	ctx.Set("role_id", roleID)
	ctx.Set("user_role", "client")
}

func assertErrorResponse(t *testing.T, body []byte, wantMsg string) {
	var resp struct {
		Error string `json:"error"`
	}
	json.Unmarshal(body, &resp)
	if resp.Error == "" {
		// Try status/message pattern for webhooks
		var webhookResp struct {
			Message string `json:"message"`
		}
		json.Unmarshal(body, &webhookResp)
		if webhookResp.Message == "" {
			t.Errorf("expected error message %q, but got empty body or different structure: %s", wantMsg, string(body))
		}
	}
}
