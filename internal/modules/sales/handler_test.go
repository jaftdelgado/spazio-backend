package sales

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockSalesService struct {
	confirmFunc func(ctx context.Context, auth AuthContext, input SaleInput) (SaleResponse, error)
}

func (m *mockSalesService) ConfirmSale(ctx context.Context, auth AuthContext, input SaleInput) (SaleResponse, error) {
	if m.confirmFunc != nil {
		return m.confirmFunc(ctx, auth, input)
	}

	return SaleResponse{}, nil
}

func TestHandler_ConfirmSale(t *testing.T) {
	gin.SetMode(gin.TestMode)

	propertyUUID := uuid.New()
	service := &mockSalesService{
		confirmFunc: func(ctx context.Context, auth AuthContext, input SaleInput) (SaleResponse, error) {
			return SaleResponse{
				TransactionUUID: uuid.New().String(),
				ContractUUID:    uuid.New().String(),
				PropertyUUID:    input.PropertyUUID.String(),
				Status:          "formalized",
				FinalAmount:     input.AgreedAmount,
				Currency:        "MXN",
			}, nil
		},
	}
	handler := NewHandler(service)

	body, _ := json.Marshal(SaleRequest{
		PropertyUUID: propertyUUID.String(),
		AgreedAmount: 1500000,
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(body))
	ctx.Request.Header.Set("Authorization", "Bearer token")
	setSaleAuth(ctx, 7, roleAgentID)

	handler.confirmSale(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
}

func TestHandler_ConfirmSale_InvalidBody_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(&mockSalesService{})
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader([]byte(`{"property_uuid":"bad"`)))
	ctx.Request.Header.Set("Authorization", "Bearer token")
	setSaleAuth(ctx, 7, roleAgentID)

	handler.confirmSale(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandler_ConfirmSale_ServiceError_Returns422(t *testing.T) {
	gin.SetMode(gin.TestMode)

	propertyUUID := uuid.New()
	service := &mockSalesService{
		confirmFunc: func(ctx context.Context, auth AuthContext, input SaleInput) (SaleResponse, error) {
			return SaleResponse{}, newStatusError(http.StatusUnprocessableEntity, "agreed_amount must match current sale price exactly; expected 1500000.00 MXN")
		},
	}
	handler := NewHandler(service)

	body, _ := json.Marshal(SaleRequest{
		PropertyUUID: propertyUUID.String(),
		AgreedAmount: 1400000,
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(body))
	ctx.Request.Header.Set("Authorization", "Bearer token")
	setSaleAuth(ctx, 7, roleAgentID)

	handler.confirmSale(ctx)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
}

func setSaleAuth(ctx *gin.Context, userID int32, roleID int32) {
	ctx.Set("user_id", userID)
	ctx.Set("role_id", roleID)
}
