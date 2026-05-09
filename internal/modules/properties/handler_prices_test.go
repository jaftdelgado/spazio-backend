package properties

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestHandler_GetPrices tests all getPrices scenarios in table-driven format.
func TestHandler_GetPrices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		uuid       string
		mockResult GetPropertyPricesResult
		mockErr    error
		wantStatus int
	}{
		{
			name:       "returns bad request when uuid is empty",
			uuid:       "",
			mockResult: GetPropertyPricesResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when uuid is invalid",
			uuid:       "not-a-uuid",
			mockResult: GetPropertyPricesResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns not found when property does not exist",
			uuid:       validUUID,
			mockResult: GetPropertyPricesResult{},
			mockErr:    ErrPropertyNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "returns internal server error when service fails",
			uuid:       validUUID,
			mockResult: GetPropertyPricesResult{},
			mockErr:    errors.New("db"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "returns ok with property prices data",
			uuid: validUUID,
			mockResult: GetPropertyPricesResult{
				Data: GetPropertyPricesData{
					SalePrice: &ActiveSalePriceData{
						SalePrice:    1500000,
						Currency:     "MXN",
						IsNegotiable: true,
					},
					RentPrices: []ActiveRentPriceData{
						{PeriodID: 3, RentPrice: 8000, Currency: "MXN"},
					},
				},
			},
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockServiceForClauses{
				getPricesFunc: func(ctx context.Context, uuid string) (GetPropertyPricesResult, error) {
					return tt.mockResult, tt.mockErr
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/properties/prices", nil)
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}

			handler := NewHandler(svcMock)
			handler.getPrices(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
			}

			// For the 200 case, verify the body contains "data" with "sale_price" and "rent_prices".
			if tt.wantStatus == http.StatusOK {
				var body map[string]json.RawMessage
				if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
					t.Fatalf("decode response body: %v", err)
				}
				if _, ok := body["data"]; !ok {
					t.Fatal("response body missing key 'data'")
				}
				var data map[string]json.RawMessage
				if err := json.Unmarshal(body["data"], &data); err != nil {
					t.Fatalf("decode 'data' field: %v", err)
				}
				if _, ok := data["sale_price"]; !ok {
					t.Fatal("'data' missing key 'sale_price'")
				}
				if _, ok := data["rent_prices"]; !ok {
					t.Fatal("'data' missing key 'rent_prices'")
				}
			}
		})
	}
}

// TestHandler_UpdatePrices tests all updatePrices scenarios in table-driven format.
func TestHandler_UpdatePrices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	validInput := UpdatePropertyPricesInput{
		SalePrice: &UpdateSalePriceInput{
			SalePrice:    1500000,
			IsNegotiable: true,
		},
	}

	tests := []struct {
		name       string
		uuid       string
		body       *UpdatePropertyPricesInput
		rawBody    []byte // For testing bad JSON
		mockErr    error
		wantStatus int
	}{
		{
			name:       "returns bad request when uuid is empty",
			uuid:       "",
			body:       &validInput,
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when uuid is invalid",
			uuid:       "not-a-uuid",
			body:       &validInput,
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when json is invalid",
			uuid:       validUUID,
			body:       nil,
			rawBody:    []byte(`{bad json}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when validation fails",
			uuid:       validUUID,
			body:       &validInput,
			mockErr:    ValidationError{Message: "sale_price.sale_price must be greater than 0"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns not found when property does not exist",
			uuid:       validUUID,
			body:       &validInput,
			mockErr:    ErrPropertyNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "returns internal server error when service fails",
			uuid:       validUUID,
			body:       &validInput,
			mockErr:    errors.New("db"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "returns no content when request is valid",
			uuid:       validUUID,
			body:       &validInput,
			mockErr:    nil,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockServiceForClauses{
				updatePricesFunc: func(ctx context.Context, uuid string, input UpdatePropertyPricesInput) error {
					return tt.mockErr
				},
			}

			// Build request body.
			var reqBody []byte
			if tt.rawBody != nil {
				reqBody = tt.rawBody
			} else if tt.body != nil {
				var err error
				reqBody, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("marshal body: %v", err)
				}
			} else {
				reqBody = []byte(`{}`)
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/properties/prices", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}

			handler := NewHandler(svcMock)
			handler.updatePrices(ctx)

			// For the 204 success case Gin's c.Status() in test context may return 200.
			// Accept both, identical to the pattern in handler_clauses_test.go.
			if tt.wantStatus == http.StatusNoContent {
				if recorder.Code != http.StatusNoContent && recorder.Code != http.StatusOK {
					t.Fatalf("status: got %d, want 204 or 200 (handler indicates success)", recorder.Code)
				}
			} else {
				if recorder.Code != tt.wantStatus {
					t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
				}
			}
		})
	}
}
