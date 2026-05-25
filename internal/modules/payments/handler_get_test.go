package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestHandler_ListPayments(t *testing.T) {
	tests := []struct {
		name         string
		setupService func() *mockPaymentService
		query        string
		wantStatus   int
	}{
		{
			name:       "400 Bad Request on invalid limit",
			query:      "?limit=abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "200 OK on successful list",
			setupService: func() *mockPaymentService {
				return &mockPaymentService{
					listPaymentsFunc: func(ctx context.Context, uid, rid int32, in ListPaymentsInput) (ListPaymentsResult, error) {
						return ListPaymentsResult{Data: []PaymentListItem{}}, nil
					},
				}
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPaymentService{}
			if tt.setupService != nil {
				svc = tt.setupService()
			}
			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodGet, "/payments"+tt.query)
			setAuthenticatedContext(ctx, 10, 3)

			h.listPayments(ctx)
			if tt.wantStatus != rec.Code {
				t.Errorf("expected %v, got %v", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandler_ListPayments_PublicResponseOmitsPaymentID(t *testing.T) {
	tests := []struct {
		name   string
		roleID int32
	}{
		{name: "client response", roleID: roleClientID},
		{name: "agent response", roleID: roleAgentID},
		{name: "admin response", roleID: roleAdminID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paymentUUID := uuid.New()
			svc := &mockPaymentService{
				listPaymentsFunc: func(ctx context.Context, uid, rid int32, in ListPaymentsInput) (ListPaymentsResult, error) {
					return ListPaymentsResult{
						Data: []PaymentListItem{
							{
								PaymentID:     22,
								PaymentUUID:   paymentUUID,
								ContractID:    7,
								PropertyID:    4,
								BillingPeriod: "2026-06-01",
								DueDate:       "2026-05-22",
								Amount:        "500.00",
								Currency:      "MXN",
								PaymentMethod: "Credit Card",
								Status:        "Completed",
							},
						},
					}, nil
				},
			}

			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodGet, "/payments")
			setAuthenticatedContext(ctx, 10, tt.roleID)

			h.listPayments(ctx)

			if http.StatusOK != rec.Code {
				t.Errorf("expected %v, got %v", http.StatusOK, rec.Code)
			}

			var body struct {
				Data []map[string]any `json:"data"`
			}
			if json.Unmarshal(rec.Body.Bytes(), &body) != nil {
				t.Errorf("unexpected error: %v", json.Unmarshal(rec.Body.Bytes(), &body))
			}
			if len(body.Data) == 1 {
				_, hasPaymentID := body.Data[0]["payment_id"]
				if hasPaymentID {
					t.Errorf("expected false")
				}
				if paymentUUID.String() != body.Data[0]["payment_uuid"] {
					t.Errorf("expected %v, got %v", paymentUUID.String(), body.Data[0]["payment_uuid"])
				}
			}
		})
	}
}

func TestHandler_GetPaymentByID(t *testing.T) {
	tests := []struct {
		name         string
		setupService func() *mockPaymentService
		idParam      string
		wantStatus   int
	}{
		{
			name:       "400 Bad Request on invalid id",
			idParam:    "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "403 Forbidden on role error (F6 Mapping)",
			idParam: uuid.NewString(),
			setupService: func() *mockPaymentService {
				return &mockPaymentService{
					getPaymentByUUIDFunc: func(ctx context.Context, uid, rid int32, paymentUUID uuid.UUID) (PaymentDetailResponse, error) {
						return PaymentDetailResponse{}, ErrPaymentForbidden
					},
				}
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPaymentService{}
			if tt.setupService != nil {
				svc = tt.setupService()
			}
			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodGet, "/payments/"+tt.idParam)
			setAuthenticatedContext(ctx, 10, 3)
			ctx.Params = []gin.Param{{Key: "payment_uuid", Value: tt.idParam}}

			h.getPaymentByID(ctx)
			if tt.wantStatus != rec.Code {
				t.Errorf("expected %v, got %v", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandler_GetPaymentByID_PublicResponseOmitsPaymentID(t *testing.T) {
	tests := []struct {
		name     string
		roleID   int32
		response PaymentDetailResponse
	}{
		{
			name:   "client response",
			roleID: roleClientID,
			response: PaymentDetailResponse{
				PaymentUUID:   uuid.New(),
				ContractID:    7,
				PropertyID:    4,
				BillingPeriod: "2026-06-01",
				DueDate:       "2026-05-22",
				Amount:        "500.00",
				Currency:      "MXN",
				PaymentMethod: "Credit Card",
				Gateway:       stringPointer("Stripe Simulation"),
				Status:        "Completed",
			},
		},
		{
			name:   "agent response",
			roleID: roleAgentID,
			response: PaymentDetailResponse{
				PaymentUUID:   uuid.New(),
				ContractID:    7,
				PropertyID:    4,
				BillingPeriod: "2026-06-01",
				DueDate:       "2026-05-22",
				Amount:        "500.00",
				Currency:      "MXN",
				PaymentMethod: "Credit Card",
				Gateway:       stringPointer("Stripe Simulation"),
				Status:        "Completed",
			},
		},
		{
			name:   "admin response",
			roleID: roleAdminID,
			response: PaymentDetailResponse{
				PaymentUUID:   uuid.New(),
				ContractID:    7,
				PropertyID:    4,
				BillingPeriod: "2026-06-01",
				DueDate:       "2026-05-22",
				Amount:        "500.00",
				Currency:      "MXN",
				PaymentMethod: "Credit Card",
				Gateway:       stringPointer("Stripe Simulation"),
				Status:        "Completed",
				ClientID:      int32Pointer(12),
				AgentID:       int32Pointer(34),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPaymentService{
				getPaymentByUUIDFunc: func(ctx context.Context, uid, rid int32, paymentUUID uuid.UUID) (PaymentDetailResponse, error) {
					return tt.response, nil
				},
			}
			h := NewHandler(svc)
			rec, ctx := newHandlerTestContext(http.MethodGet, "/payments/"+tt.response.PaymentUUID.String())
			setAuthenticatedContext(ctx, 10, tt.roleID)
			ctx.Params = []gin.Param{{Key: "payment_uuid", Value: tt.response.PaymentUUID.String()}}

			h.getPaymentByID(ctx)

			if http.StatusOK != rec.Code {
				t.Errorf("expected %v, got %v", http.StatusOK, rec.Code)
			}

			var body map[string]any
			if json.Unmarshal(rec.Body.Bytes(), &body) != nil {
				t.Errorf("unexpected error: %v", json.Unmarshal(rec.Body.Bytes(), &body))
			}
			_, hasPaymentID := body["payment_id"]
			if hasPaymentID {
				t.Errorf("expected false")
			}
			if tt.response.PaymentUUID.String() != body["payment_uuid"] {
				t.Errorf("expected %v, got %v", tt.response.PaymentUUID.String(), body["payment_uuid"])
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}

func int32Pointer(value int32) *int32 {
	return &value
}
