package properties

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type mockService struct {
	result CreatePropertyResult
	err    error
	input  CreatePropertyInput
	called bool
}

func (m *mockService) CreateProperty(_ context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
	m.called = true
	m.input = input
	return m.result, m.err
}

func TestCreateProperty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		payload          any
		mock             *mockService
		wantStatusCode   int
		wantBodyContains string
		wantCalled       bool
	}{
		{
			name: "creates property",
			payload: gin.H{
				"owner_id":         1,
				"title":            "Casa demo",
				"description":      "Una propiedad de prueba",
				"property_type_id": 2,
				"modality_id":      3,
				"status_id":        4,
				"cover_photo_url":  "https://example.com/cover.jpg",
			},
			mock: &mockService{
				result: CreatePropertyResult{
					PropertyID: 7,
					Title:      "Casa demo",
					CreatedAt:  time.Date(2026, time.April, 23, 12, 0, 0, 0, time.UTC),
				},
			},
			wantStatusCode:   http.StatusCreated,
			wantBodyContains: "\"property_id\":7",
			wantCalled:       true,
		},
		{
			name:             "rejects invalid payload",
			payload:          gin.H{"title": ""},
			mock:             &mockService{},
			wantStatusCode:   http.StatusBadRequest,
			wantBodyContains: "owner_id is required",
			wantCalled:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			body, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}

			ctx.Request = httptest.NewRequest(http.MethodPost, "/properties", bytes.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			handler := NewHandler(tt.mock)
			handler.createProperty(ctx)

			if recorder.Code != tt.wantStatusCode {
				t.Fatalf("status code = %d, want %d", recorder.Code, tt.wantStatusCode)
			}

			if tt.wantCalled != tt.mock.called {
				t.Fatalf("called = %v, want %v", tt.mock.called, tt.wantCalled)
			}

			if tt.wantBodyContains != "" && !bytes.Contains(recorder.Body.Bytes(), []byte(tt.wantBodyContains)) {
				t.Fatalf("body %q does not contain %q", recorder.Body.String(), tt.wantBodyContains)
			}
		})
	}
}

func TestValidateCreatePropertyRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     CreatePropertyInput
		wantErr string
	}{
		{
			name: "valid request",
			req: CreatePropertyInput{
				OwnerID:        1,
				Title:          "Casa demo",
				Description:    "Descripcion",
				PropertyTypeID: 2,
				ModalityID:     3,
				StatusID:       4,
				CoverPhotoURL:  "https://example.com/cover.jpg",
			},
		},
		{
			name:    "missing owner",
			req:     CreatePropertyInput{Title: "Casa demo"},
			wantErr: "owner_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreatePropertyRequest(tt.req)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("validateCreatePropertyRequest() error = %v, want nil", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("validateCreatePropertyRequest() error = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("validateCreatePropertyRequest() error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}
