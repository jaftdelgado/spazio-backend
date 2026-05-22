package payments

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tt.wantStatus, rec.Code)
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
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
