package properties

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestHandler_DeleteProperty(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	validBody := DeletePropertyInput{Confirm: true, ChangedByUserID: 1}

	tests := []struct {
		name             string
		uuid             string
		body             any
		rawBody          string
		serviceErr       error
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:       "returns bad request when uuid is empty",
			uuid:       "",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when json is invalid",
			uuid:       validUUID,
			rawBody:    `{"confirm": true, "changed_by_user_id":}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when confirm is false",
			uuid:       validUUID,
			body:       DeletePropertyInput{Confirm: false, ChangedByUserID: 1},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when changed by user id is zero",
			uuid:       validUUID,
			body:       DeletePropertyInput{Confirm: true, ChangedByUserID: 0},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when validation fails",
			uuid:       validUUID,
			body:       validBody,
			serviceErr: ValidationError{Message: "validation failed"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns not found when property does not exist",
			uuid:       validUUID,
			body:       validBody,
			serviceErr: ErrPropertyNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "returns internal server error when service fails",
			uuid:       validUUID,
			body:       validBody,
			serviceErr: errors.New("db error"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:             "returns ok when property is deleted successfully",
			uuid:             validUUID,
			body:             validBody,
			wantStatus:       http.StatusOK,
			wantBodyContains: `"message":"property deleted successfully"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customMock := &mockServiceForClauses{
				deletePropertyFunc: func(_ context.Context, _ string, _ DeletePropertyInput) error {
					return tt.serviceErr
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}

			var bodyBytes []byte
			if tt.rawBody != "" {
				bodyBytes = []byte(tt.rawBody)
			} else {
				payload, err := json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("marshal body: %v", err)
				}
				bodyBytes = payload
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/properties/"+tt.uuid, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx.Request = req

			handler := NewHandler(customMock)
			handler.deleteProperty(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
			}

			if tt.wantBodyContains != "" && !bytes.Contains(recorder.Body.Bytes(), []byte(tt.wantBodyContains)) {
				t.Fatalf("body %q does not contain %q", recorder.Body.String(), tt.wantBodyContains)
			}
		})
	}
}

func TestHandler_DeleteProperty_ValidateDeletePropertyRequest(t *testing.T) {
	tests := []struct {
		name       string
		input      DeletePropertyInput
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "confirm must be true",
			input:      DeletePropertyInput{Confirm: false, ChangedByUserID: 1},
			wantErr:    true,
			wantErrMsg: "confirm must be true",
		},
		{
			name:       "changed_by_user_id must be greater than 0",
			input:      DeletePropertyInput{Confirm: true, ChangedByUserID: 0},
			wantErr:    true,
			wantErrMsg: "changed_by_user_id must be greater than 0",
		},
		{
			name:    "valid request",
			input:   DeletePropertyInput{Confirm: true, ChangedByUserID: 10},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDeletePropertyRequest(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tt.wantErrMsg {
					t.Fatalf("error message: got %q, want %q", err.Error(), tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
