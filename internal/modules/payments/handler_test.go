package payments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

type handlerTestService struct {
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

func (m *handlerTestService) ListPayments(_ context.Context, userID int32, input ListPaymentsInput) (ListPaymentsResult, error) {
	m.listCalled = true
	m.listUserID = userID
	m.listInput = input
	return m.listResult, m.listErr
}

func (m *handlerTestService) GetPaymentByID(_ context.Context, userID int32, paymentID int32) (PaymentDetail, error) {
	m.detailCalled = true
	m.detailUserID = userID
	m.detailPaymentID = paymentID
	return m.detailResult, m.detailErr
}

type errorResponseBody struct {
	Error string `json:"error"`
}

type listPaymentsResponseBody struct {
	Data       []PaymentListItem `json:"data"`
	Pagination struct {
		Limit  int32 `json:"limit"`
		Offset int32 `json:"offset"`
		Total  int64 `json:"total"`
	} `json:"pagination"`
}

func TestHandler_ListPayments_MissingUserID(t *testing.T) {
	runListPaymentsHandlerAuthCase(t, "", http.StatusUnauthorized, "unauthorized")
}

func TestHandler_ListPayments_EmptyUserID(t *testing.T) {
	runListPaymentsHandlerAuthCase(t, "   ", http.StatusUnauthorized, "unauthorized")
}

func TestHandler_ListPayments_ZeroUserID(t *testing.T) {
	runListPaymentsHandlerAuthCase(t, "0", http.StatusBadRequest, "X-User-ID must be a positive integer")
}

func TestHandler_ListPayments_NegativeUserID(t *testing.T) {
	runListPaymentsHandlerAuthCase(t, "-1", http.StatusBadRequest, "X-User-ID must be a positive integer")
}

func TestHandler_ListPayments_NonNumericUserID(t *testing.T) {
	runListPaymentsHandlerAuthCase(t, "abc", http.StatusBadRequest, "X-User-ID must be a valid integer")
}

func TestHandler_ListPayments_LimitZero(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?limit=0", "limit must be greater than 0")
}

func TestHandler_ListPayments_LimitNegative(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?limit=-5", "limit must be greater than 0")
}

func TestHandler_ListPayments_LimitExceedsMax(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?limit=101", "limit must be less than or equal to 100")
}

func TestHandler_ListPayments_LimitNonNumeric(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?limit=abc", "limit must be a valid integer")
}

func TestHandler_ListPayments_OffsetNegative(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?offset=-1", "offset must be greater than or equal to 0")
}

func TestHandler_ListPayments_OffsetNonNumeric(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?offset=xyz", "offset must be a valid integer")
}

func TestHandler_ListPayments_DateFromInvalidFormat(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?date_from=31-12-2024", "date_from must use YYYY-MM-DD format")
}

func TestHandler_ListPayments_DateToInvalidFormat(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?date_to=2024/12/31", "date_to must use YYYY-MM-DD format")
}

func TestHandler_ListPayments_DateToBeforeDateFrom(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?date_from=2024-05-01&date_to=2024-01-01", "date_to must be greater than or equal to date_from")
}

func TestHandler_ListPayments_DateFromEqualsDateTo(t *testing.T) {
	service := &handlerTestService{
		listResult: ListPaymentsResult{
			Data:       []PaymentListItem{},
			Pagination: PaymentsPagination{Limit: 20, Offset: 0, Total: 0},
		},
	}

	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments?date_from=2024-03-01&date_to=2024-03-01")
	ctx.Request.Header.Set("X-User-ID", "1")

	handler := NewHandler(service)
	handler.listPayments(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !service.listCalled {
		t.Fatal("expected ListPayments to be called")
	}
	if service.listInput.DateFrom == nil || service.listInput.DateTo == nil {
		t.Fatal("expected both date filters to be set")
	}
	if !service.listInput.DateFrom.Equal(*service.listInput.DateTo) {
		t.Fatalf("date filters differ: from=%v to=%v", service.listInput.DateFrom, service.listInput.DateTo)
	}
}

func TestHandler_ListPayments_PropertyIDZero(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?property_id=0", "property_id must be a positive integer")
}

func TestHandler_ListPayments_PropertyIDNegative(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?property_id=-1", "property_id must be a positive integer")
}

func TestHandler_ListPayments_PropertyIDNonNumeric(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?property_id=abc", "property_id must be a valid integer")
}

func TestHandler_ListPayments_StatusIDZero(t *testing.T) {
	runListPaymentsHandlerInvalidQueryCase(t, "/api/v1/payments?status_id=0", "status_id must be a positive integer")
}

func TestHandler_ListPayments_ServiceReturnsUnsupportedRole(t *testing.T) {
	service := &handlerTestService{listErr: ErrUnsupportedRole}
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments")
	ctx.Request.Header.Set("X-User-ID", "1")

	handler := NewHandler(service)
	handler.listPayments(ctx)

	assertErrorResponse(t, recorder, http.StatusForbidden, "forbidden")
}

func TestHandler_ListPayments_ServiceReturnsEmptyList(t *testing.T) {
	service := &handlerTestService{
		listResult: ListPaymentsResult{
			Data:       []PaymentListItem{},
			Pagination: PaymentsPagination{Limit: 20, Offset: 0, Total: 0},
		},
	}
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments")
	ctx.Request.Header.Set("X-User-ID", "1")

	handler := NewHandler(service)
	handler.listPayments(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body listPaymentsResponseBody
	decodeJSONBody(t, recorder, &body)

	if len(body.Data) != 0 {
		t.Fatalf("len(data) = %d, want 0", len(body.Data))
	}
	if body.Pagination.Total != 0 {
		t.Fatalf("pagination.total = %d, want 0", body.Pagination.Total)
	}
}

func TestHandler_ListPayments_ServiceReturnsInternalError(t *testing.T) {
	service := &handlerTestService{listErr: errors.New("db down")}
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments")
	ctx.Request.Header.Set("X-User-ID", "1")

	handler := NewHandler(service)
	handler.listPayments(ctx)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "could not list payments")
}

func TestHandler_ListPayments_ServiceReturnsResults(t *testing.T) {
	paymentDate := time.Date(2024, time.March, 8, 14, 32, 0, 0, time.UTC)
	service := &handlerTestService{
		listResult: ListPaymentsResult{
			Data: []PaymentListItem{
				{
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
				},
				{
					PaymentID:     2,
					ContractID:    11,
					PropertyID:    6,
					BillingPeriod: "2024-04-01",
					DueDate:       "2024-04-10",
					Amount:        "1800.00",
					Currency:      "MXN",
					PaymentMethod: "Tarjeta",
					Status:        "Pendiente",
				},
			},
			Pagination: PaymentsPagination{Limit: 20, Offset: 0, Total: 2},
		},
	}
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments")
	ctx.Request.Header.Set("X-User-ID", "1")

	handler := NewHandler(service)
	handler.listPayments(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body listPaymentsResponseBody
	decodeJSONBody(t, recorder, &body)

	if len(body.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2", len(body.Data))
	}
	if body.Pagination.Total != 2 {
		t.Fatalf("pagination.total = %d, want 2", body.Pagination.Total)
	}
}

func TestHandler_GetPaymentByID_MissingUserID(t *testing.T) {
	service := &handlerTestService{}
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments/1")
	ctx.Params = gin.Params{{Key: "payment_id", Value: "1"}}

	handler := NewHandler(service)
	handler.getPaymentByID(ctx)

	assertErrorResponse(t, recorder, http.StatusUnauthorized, "unauthorized")
	if service.detailCalled {
		t.Fatal("did not expect GetPaymentByID to be called")
	}
}

func TestHandler_GetPaymentByID_PaymentIDZero(t *testing.T) {
	runGetPaymentByIDInvalidPathCase(t, "0", "payment_id must be a positive integer")
}

func TestHandler_GetPaymentByID_PaymentIDNegative(t *testing.T) {
	runGetPaymentByIDInvalidPathCase(t, "-1", "payment_id must be a positive integer")
}

func TestHandler_GetPaymentByID_PaymentIDNonNumeric(t *testing.T) {
	runGetPaymentByIDInvalidPathCase(t, "abc", "payment_id must be a valid integer")
}

func TestHandler_GetPaymentByID_PaymentNotFound(t *testing.T) {
	runGetPaymentByIDServiceErrorCase(t, ErrPaymentNotFound, http.StatusNotFound, "payment not found")
}

func TestHandler_GetPaymentByID_PaymentForbidden(t *testing.T) {
	runGetPaymentByIDServiceErrorCase(t, ErrPaymentForbidden, http.StatusForbidden, "forbidden")
}

func TestHandler_GetPaymentByID_UnsupportedRole(t *testing.T) {
	runGetPaymentByIDServiceErrorCase(t, ErrUnsupportedRole, http.StatusForbidden, "forbidden")
}

func TestHandler_GetPaymentByID_InternalError(t *testing.T) {
	runGetPaymentByIDServiceErrorCase(t, errors.New("db down"), http.StatusInternalServerError, "could not get payment")
}

func TestHandler_GetPaymentByID_Success(t *testing.T) {
	service := &handlerTestService{
		detailResult: PaymentDetail{
			PaymentID:       1,
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
	}
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments/1")
	ctx.Request.Header.Set("X-User-ID", "1")
	ctx.Params = gin.Params{{Key: "payment_id", Value: "1"}}

	handler := NewHandler(service)
	handler.getPaymentByID(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body PaymentDetail
	decodeJSONBody(t, recorder, &body)

	if body.PaymentID != 1 {
		t.Fatalf("payment_id = %d, want 1", body.PaymentID)
	}
	if !service.detailCalled {
		t.Fatal("expected GetPaymentByID to be called")
	}
}

func runListPaymentsHandlerAuthCase(t *testing.T, headerUserID string, wantStatus int, wantError string) {
	t.Helper()

	service := &handlerTestService{}
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments")
	if headerUserID != "" {
		ctx.Request.Header.Set("X-User-ID", headerUserID)
	}

	handler := NewHandler(service)
	handler.listPayments(ctx)

	assertErrorResponse(t, recorder, wantStatus, wantError)
	if service.listCalled {
		t.Fatal("did not expect ListPayments to be called")
	}
}

func runListPaymentsHandlerInvalidQueryCase(t *testing.T, target string, wantError string) {
	t.Helper()

	service := &handlerTestService{}
	recorder, ctx := newHandlerTestContext(http.MethodGet, target)
	ctx.Request.Header.Set("X-User-ID", "1")

	handler := NewHandler(service)
	handler.listPayments(ctx)

	assertErrorResponse(t, recorder, http.StatusBadRequest, wantError)
	if service.listCalled {
		t.Fatal("did not expect ListPayments to be called")
	}
}

func runGetPaymentByIDInvalidPathCase(t *testing.T, paymentID string, wantError string) {
	t.Helper()

	service := &handlerTestService{}
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments/"+paymentID)
	ctx.Request.Header.Set("X-User-ID", "1")
	ctx.Params = gin.Params{{Key: "payment_id", Value: paymentID}}

	handler := NewHandler(service)
	handler.getPaymentByID(ctx)

	assertErrorResponse(t, recorder, http.StatusBadRequest, wantError)
	if service.detailCalled {
		t.Fatal("did not expect GetPaymentByID to be called")
	}
}

func runGetPaymentByIDServiceErrorCase(t *testing.T, serviceErr error, wantStatus int, wantError string) {
	t.Helper()

	service := &handlerTestService{detailErr: serviceErr}
	recorder, ctx := newHandlerTestContext(http.MethodGet, "/api/v1/payments/1")
	ctx.Request.Header.Set("X-User-ID", "1")
	ctx.Params = gin.Params{{Key: "payment_id", Value: "1"}}

	handler := NewHandler(service)
	handler.getPaymentByID(ctx)

	assertErrorResponse(t, recorder, wantStatus, wantError)
	if !service.detailCalled {
		t.Fatal("expected GetPaymentByID to be called")
	}
}

func newHandlerTestContext(method string, target string) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, nil)
	return recorder, ctx
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantError string) {
	t.Helper()

	if recorder.Code != wantStatus {
		t.Fatalf("status code = %d, want %d", recorder.Code, wantStatus)
	}

	var body errorResponseBody
	decodeJSONBody(t, recorder, &body)

	if body.Error != wantError {
		t.Fatalf("error = %q, want %q", body.Error, wantError)
	}
}

func decodeJSONBody(t *testing.T, recorder *httptest.ResponseRecorder, target interface{}) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("unmarshal response body: %v; body=%s", err, recorder.Body.String())
	}
}

func stringPointer(value string) *string {
	return &value
}

var _ PaymentsService = (*handlerTestService)(nil)
var _ = strings.TrimSpace
