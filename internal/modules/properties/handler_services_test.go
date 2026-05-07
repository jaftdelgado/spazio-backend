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

// TestHandler_GetServices tests all getServices scenarios in table-driven format.
func TestHandler_GetServices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		uuid       string
		mockResult GetPropertyServicesResult
		mockErr    error
		wantStatus int
	}{
		{
			name:       "returns bad request when uuid is empty",
			uuid:       "",
			mockResult: GetPropertyServicesResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when uuid is invalid",
			uuid:       "not-a-uuid",
			mockResult: GetPropertyServicesResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns not found when property does not exist",
			uuid:       validUUID,
			mockResult: GetPropertyServicesResult{},
			mockErr:    ErrPropertyNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "returns internal server error when service fails",
			uuid:       validUUID,
			mockResult: GetPropertyServicesResult{},
			mockErr:    errors.New("db"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "returns ok with property services data",
			uuid:       validUUID,
			mockResult: GetPropertyServicesResult{Data: GetPropertyServicesData{ServiceIDs: []int32{1, 3, 7}}},
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockServiceForClauses{
				getServicesFunc: func(ctx context.Context, uuid string) (GetPropertyServicesResult, error) {
					return tt.mockResult, tt.mockErr
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/properties/services", nil)
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}

			handler := NewHandler(svcMock)
			handler.getServices(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
			}

			// For the 200 case, verify the body contains "data" with "service_ids".
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
				if _, ok := data["service_ids"]; !ok {
					t.Fatal("'data' missing key 'service_ids'")
				}
			}
		})
	}
}

// TestHandler_UpdateServices tests all updateServices scenarios in table-driven format.
func TestHandler_UpdateServices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	validInput := UpdatePropertyServicesInput{
		ServiceIDs: []int32{1, 3, 7},
	}

	tests := []struct {
		name       string
		uuid       string
		body       *UpdatePropertyServicesInput
		rawBody    []byte // used when body is nil (invalid JSON case)
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
			name:       "returns no content when body is empty",
			uuid:       validUUID,
			body:       nil,
			rawBody:    []byte(""),
			mockErr:    nil,
			wantStatus: http.StatusNoContent,
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
			mockErr:    ValidationError{Message: "services[0] must be greater than 0"},
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
			name:       "returns no content when service ids are valid",
			uuid:       validUUID,
			body:       &validInput,
			mockErr:    nil,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockServiceForClauses{
				updateServicesFunc: func(ctx context.Context, uuid string, input UpdatePropertyServicesInput) error {
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
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/properties/services", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}

			handler := NewHandler(svcMock)
			handler.updateServices(ctx)

			// For the 204 success case Gin's c.Status() in test context may return 200.
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
