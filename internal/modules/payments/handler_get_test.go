package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

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

func TestHandler_ListPayments_OmitsPaymentIDFromResponse(t *testing.T) {
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
				Pagination: PaymentsPagination{Limit: 20, Offset: 0, Total: 1},
			}, nil
		},
	}

	h := NewHandler(svc)
	rec, ctx := newHandlerTestContext(http.MethodGet, "/payments")
	setAuthenticatedContext(ctx, 10, roleClientID)

	h.listPayments(ctx)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	data, ok := body["data"].([]interface{})
	if assert.True(t, ok) && assert.Len(t, data, 1) {
		item := data[0].(map[string]interface{})
		_, hasPaymentID := item["payment_id"]
		assert.False(t, hasPaymentID)
		assert.Equal(t, paymentUUID.String(), item["payment_uuid"])
	}
}

func TestHandler_GetPaymentByID_OmitsPaymentIDFromResponse(t *testing.T) {
	paymentDate := time.Date(2026, 5, 21, 21, 36, 40, 0, time.UTC)
	svc := &mockPaymentService{
		getPaymentByUUIDFunc: func(ctx context.Context, uid, rid int32, paymentUUID uuid.UUID) (PaymentDetailResponse, error) {
			return PaymentDetailResponse{
				ContractID:      7,
				PropertyID:      4,
				TransactionID:   3,
				TransactionType: "rent",
				BillingPeriod:   "2026-06-01",
				DueDate:         "2026-05-22",
				AgreedAmount:    "500.00",
				Amount:          "500.00",
				Currency:        "MXN",
				PaymentMethod:   "Credit Card",
				Status:          "Completed",
				PaymentDate:     &paymentDate,
			}, nil
		},
	}

	h := NewHandler(svc)
	idParam := uuid.NewString()
	rec, ctx := newHandlerTestContext(http.MethodGet, "/payments/"+idParam)
	setAuthenticatedContext(ctx, 10, roleClientID)
	ctx.Params = []gin.Param{{Key: "payment_uuid", Value: idParam}}

	h.getPaymentByID(ctx)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	_, hasPaymentID := body["payment_id"]
	assert.False(t, hasPaymentID)
}
