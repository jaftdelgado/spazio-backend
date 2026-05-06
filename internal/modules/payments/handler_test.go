package payments

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type handlerMockService struct {
	listResult      ListPaymentsResult
	listErr         error
	detailResult    PaymentDetail
	detailErr       error
	listCalled      bool
	detailCalled    bool
	listUserID      int32
	listInput       ListPaymentsInput
	detailUserID    int32
	detailPaymentID int32
}

func (m *handlerMockService) ListPayments(_ context.Context, userID int32, input ListPaymentsInput) (ListPaymentsResult, error) {
	m.listCalled = true
	m.listUserID = userID
	m.listInput = input
	return m.listResult, m.listErr
}

func (m *handlerMockService) GetPaymentByID(_ context.Context, userID int32, paymentID int32) (PaymentDetail, error) {
	m.detailCalled = true
	m.detailUserID = userID
	m.detailPaymentID = paymentID
	return m.detailResult, m.detailErr
}

func TestListPayments(t *testing.T) {
	gin.SetMode(gin.TestMode)

	paymentDate := time.Date(2024, time.March, 8, 14, 32, 0, 0, time.UTC)

	tests := []struct {
		name             string
		headerUserID     string
		url              string
		mock             *handlerMockService
		wantStatusCode   int
		wantBodyContains string
		wantCalled       bool
		wantUserID       int32
	}{
		{
			name:         "lists payments as admin",
			headerUserID: "1",
			url:          "/api/v1/payments",
			mock: &handlerMockService{
				listResult: ListPaymentsResult{
					Data: []PaymentListItem{{
						PaymentID:     1,
						ContractID:    10,
						PropertyID:    5,
						BillingPeriod: "2024-03-01",
						DueDate:       "2024-03-10",
						Amount:        "1500.00",
						Currency:      "MXN",
						PaymentMethod: "Transferencia bancaria",
						Gateway:       stringPointer("Stripe"),
						Status:        "Pagado",
						PaymentDate:   &paymentDate,
					}},
					Pagination: PaymentsPagination{Limit: 20, Offset: 0, Total: 1},
				},
			},
			wantStatusCode:   http.StatusOK,
			wantBodyContains: "\"payment_id\":1",
			wantCalled:       true,
			wantUserID:       1,
		},
		{
			name:         "lists payments as agent",
			headerUserID: "2",
			url:          "/api/v1/payments?property_id=5&limit=10&offset=2",
			mock: &handlerMockService{
				listResult: ListPaymentsResult{
					Data:       []PaymentListItem{},
					Pagination: PaymentsPagination{Limit: 10, Offset: 2, Total: 0},
				},
			},
			wantStatusCode:   http.StatusOK,
			wantBodyContains: "\"limit\":10",
			wantCalled:       true,
			wantUserID:       2,
		},
		{
			name:         "lists payments as client",
			headerUserID: "3",
			url:          "/api/v1/payments?status_id=4",
			mock: &handlerMockService{
				listResult: ListPaymentsResult{
					Data:       []PaymentListItem{},
					Pagination: PaymentsPagination{Limit: 20, Offset: 0, Total: 0},
				},
			},
			wantStatusCode:   http.StatusOK,
			wantBodyContains: "\"total\":0",
			wantCalled:       true,
			wantUserID:       3,
		},
		{
			name:             "rejects invalid query params",
			headerUserID:     "1",
			url:              "/api/v1/payments?limit=-1",
			mock:             &handlerMockService{},
			wantStatusCode:   http.StatusBadRequest,
			wantBodyContains: "limit must be greater than 0",
			wantCalled:       false,
		},
		{
			name:             "rejects invalid date query params",
			headerUserID:     "1",
			url:              "/api/v1/payments?date_from=2024-13-01",
			mock:             &handlerMockService{},
			wantStatusCode:   http.StatusBadRequest,
			wantBodyContains: "date_from must use YYYY-MM-DD format",
			wantCalled:       false,
		},
		{
			name:             "returns unauthorized when auth is missing",
			headerUserID:     "",
			url:              "/api/v1/payments",
			mock:             &handlerMockService{},
			wantStatusCode:   http.StatusUnauthorized,
			wantBodyContains: "unauthorized",
			wantCalled:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.headerUserID != "" {
				ctx.Request.Header.Set("X-User-ID", tt.headerUserID)
			}

			handler := NewHandler(tt.mock)
			handler.listPayments(ctx)

			if recorder.Code != tt.wantStatusCode {
				t.Fatalf("status code = %d, want %d", recorder.Code, tt.wantStatusCode)
			}

			if tt.mock.listCalled != tt.wantCalled {
				t.Fatalf("listCalled = %v, want %v", tt.mock.listCalled, tt.wantCalled)
			}

			if tt.wantCalled && tt.mock.listUserID != tt.wantUserID {
				t.Fatalf("listUserID = %d, want %d", tt.mock.listUserID, tt.wantUserID)
			}

			if tt.wantBodyContains != "" && !strings.Contains(recorder.Body.String(), tt.wantBodyContains) {
				t.Fatalf("body %q does not contain %q", recorder.Body.String(), tt.wantBodyContains)
			}
		})
	}
}

func TestGetPaymentByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		headerUserID     string
		paymentID        string
		mock             *handlerMockService
		wantStatusCode   int
		wantBodyContains string
		wantCalled       bool
	}{
		{
			name:         "returns payment detail",
			headerUserID: "1",
			paymentID:    "7",
			mock: &handlerMockService{
				detailResult: PaymentDetail{
					PaymentID:       7,
					ContractID:      10,
					PropertyID:      5,
					TransactionID:   3,
					TransactionType: "rent",
					BillingPeriod:   "2024-03-01",
					DueDate:         "2024-03-10",
					AgreedAmount:    "15000.00",
					Amount:          "1500.00",
					Currency:        "MXN",
					PaymentMethod:   "Transferencia bancaria",
					Status:          "Pagado",
					ClientID:        7,
					AgentID:         2,
				},
			},
			wantStatusCode:   http.StatusOK,
			wantBodyContains: "\"transaction_type\":\"rent\"",
			wantCalled:       true,
		},
		{
			name:             "returns forbidden for foreign payment",
			headerUserID:     "2",
			paymentID:        "7",
			mock:             &handlerMockService{detailErr: ErrPaymentForbidden},
			wantStatusCode:   http.StatusForbidden,
			wantBodyContains: "forbidden",
			wantCalled:       true,
		},
		{
			name:             "returns not found for missing payment",
			headerUserID:     "1",
			paymentID:        "999",
			mock:             &handlerMockService{detailErr: ErrPaymentNotFound},
			wantStatusCode:   http.StatusNotFound,
			wantBodyContains: ErrPaymentNotFound.Error(),
			wantCalled:       true,
		},
		{
			name:             "rejects invalid payment id",
			headerUserID:     "1",
			paymentID:        "abc",
			mock:             &handlerMockService{},
			wantStatusCode:   http.StatusBadRequest,
			wantBodyContains: "payment_id must be a valid integer",
			wantCalled:       false,
		},
		{
			name:             "returns unauthorized without auth",
			headerUserID:     "",
			paymentID:        "7",
			mock:             &handlerMockService{},
			wantStatusCode:   http.StatusUnauthorized,
			wantBodyContains: "unauthorized",
			wantCalled:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payments/"+tt.paymentID, nil)
			if tt.headerUserID != "" {
				ctx.Request.Header.Set("X-User-ID", tt.headerUserID)
			}
			ctx.Params = gin.Params{{Key: "payment_id", Value: tt.paymentID}}

			handler := NewHandler(tt.mock)
			handler.getPaymentByID(ctx)

			if recorder.Code != tt.wantStatusCode {
				t.Fatalf("status code = %d, want %d", recorder.Code, tt.wantStatusCode)
			}

			if tt.mock.detailCalled != tt.wantCalled {
				t.Fatalf("detailCalled = %v, want %v", tt.mock.detailCalled, tt.wantCalled)
			}

			if tt.wantBodyContains != "" && !strings.Contains(recorder.Body.String(), tt.wantBodyContains) {
				t.Fatalf("body %q does not contain %q", recorder.Body.String(), tt.wantBodyContains)
			}
		})
	}
}

func TestResolveListPaymentsInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payments?property_id=5&status_id=4&date_from=2024-03-01&date_to=2024-03-31&limit=10&offset=2", nil)

	input, err := resolveListPaymentsInput(ctx)
	if err != nil {
		t.Fatalf("resolveListPaymentsInput() error = %v, want nil", err)
	}

	if input.Limit != 10 {
		t.Fatalf("limit = %d, want 10", input.Limit)
	}
	if input.Offset != 2 {
		t.Fatalf("offset = %d, want 2", input.Offset)
	}
	if input.PropertyID == nil || *input.PropertyID != 5 {
		t.Fatalf("property_id = %#v, want 5", input.PropertyID)
	}
	if input.StatusID == nil || *input.StatusID != 4 {
		t.Fatalf("status_id = %#v, want 4", input.StatusID)
	}
}

func stringPointer(value string) *string {
	return &value
}

var _ PaymentsService = (*handlerMockService)(nil)
var _ = errors.Is
