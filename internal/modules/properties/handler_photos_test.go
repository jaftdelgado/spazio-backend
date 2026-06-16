package properties

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestHandler_GetPhotos tests all getPhotos scenarios in table-driven format.
func TestHandler_GetPhotos(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		uuid       string
		mockResult GetPropertyPhotosResult
		mockErr    error
		wantStatus int
	}{
		{
			name:       "returns bad request when uuid is empty",
			uuid:       "",
			mockResult: GetPropertyPhotosResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when uuid is invalid",
			uuid:       "not-a-uuid",
			mockResult: GetPropertyPhotosResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns not found when property does not exist",
			uuid:       validUUID,
			mockResult: GetPropertyPhotosResult{},
			mockErr:    ErrPropertyNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "returns internal server error when service fails",
			uuid:       validUUID,
			mockResult: GetPropertyPhotosResult{},
			mockErr:    errors.New("db"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "returns ok with property photos data",
			uuid: validUUID,
			mockResult: GetPropertyPhotosResult{Data: []PropertyPhotoData{{
				PhotoID:    1,
				StorageKey: "properties/123e4567-e89b-12d3-a456-426614174000/photos/1.webp",
				URL:        "https://pub-ab9b26339b564d53b2f5ec019d1ca830.r2.dev/properties/123e4567-e89b-12d3-a456-426614174000/photos/1.webp",
			}}},
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockServiceForClauses{
				getPhotosFunc: func(ctx context.Context, uuid string) (GetPropertyPhotosResult, error) {
					return tt.mockResult, tt.mockErr
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/properties/photos", nil)
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}

			handler := NewHandler(svcMock)
			handler.getPhotos(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var body GetPropertyPhotosResult
				if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
					t.Fatalf("decode response body: %v", err)
				}
				if len(body.Data) != len(tt.mockResult.Data) {
					t.Fatalf("data length: got %d, want %d", len(body.Data), len(tt.mockResult.Data))
				}
				if len(body.Data) > 0 {
					if body.Data[0].StorageKey == "" {
						t.Fatal("storage_key should not be empty")
					}
					if !strings.HasPrefix(body.Data[0].URL, "https://") {
						t.Fatalf("url = %q, want public https url", body.Data[0].URL)
					}
				}
			}
		})
	}
}

// TestHandler_UpdatePhotos tests all updatePhotos scenarios in table-driven format.
func TestHandler_UpdatePhotos(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	validInput := UpdatePropertyPhotosInput{
		Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}},
	}

	tests := []struct {
		name       string
		uuid       string
		body       *UpdatePropertyPhotosInput
		rawBody    []byte
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
			mockErr:    ValidationError{Message: "validation error"},
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
			name:       "returns no content when photos are valid",
			uuid:       validUUID,
			body:       &validInput,
			mockErr:    nil,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockServiceForClauses{
				updatePhotosFunc: func(ctx context.Context, uuid string, input UpdatePropertyPhotosInput) error {
					return tt.mockErr
				},
			}

			// Build request body
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
			req := httptest.NewRequest(http.MethodPut, "/api/v1/properties/photos", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}

			handler := NewHandler(svcMock)
			handler.updatePhotos(ctx)

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
