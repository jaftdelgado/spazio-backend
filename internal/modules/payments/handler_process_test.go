package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestHandler_ProcessPayment(t *testing.T) {
	tests := []struct {
		name         string
		setupService func() *mockPaymentService
		reqBody      interface{}
		wantStatus   int
	}{
		{
			name: "400 Bad Request when validation fails",
			reqBody: RegisterPaymentRequest{
				ContractID: 0, // Invalid
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "403 Forbidden when unauthorized (F6 Mapping)",
			reqBody: RegisterPaymentRequest{
				ContractID: 1, PaymentMethodID: 1, GatewayID: 1, Amount: 10, Currency: "MXN", PayerEmail: "t@t.com",
			},
			setupService: func() *mockPaymentService {
				return &mockPaymentService{
					processPaymentFunc: func(ctx context.Context, uid int32, req RegisterPaymentRequest) (PaymentResponse, error) {
						return PaymentResponse{}, errors.New("operación no autorizada: este contrato no pertenece")
					},
				}
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "201 Created on success",
			reqBody: RegisterPaymentRequest{
				ContractID: 1, PaymentMethodID: 1, GatewayID: 1, Amount: 10, Currency: "MXN", PayerEmail: "t@t.com",
			},
			setupService: func() *mockPaymentService {
				return &mockPaymentService{
					processPaymentFunc: func(ctx context.Context, uid int32, req RegisterPaymentRequest) (PaymentResponse, error) {
						return PaymentResponse{Status: "Success"}, nil
					},
				}
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPaymentService{}
			if tt.setupService != nil {
				svc = tt.setupService()
			}
			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodPost, "/payments")
			setAuthenticatedContext(ctx, 10, 3)

			body, _ := json.Marshal(tt.reqBody)
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))

			h.processPayment(ctx)
			if tt.wantStatus != rec.Code {
				t.Errorf("expected %v, got %v", tt.wantStatus, rec.Code)
			}
		})
	}
}
