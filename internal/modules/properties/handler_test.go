package properties

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func (m *mockService) GetClauses(_ context.Context, _ string) (GetPropertyClausesResult, error) {
	return GetPropertyClausesResult{}, nil
}

func (m *mockService) UpdateClauses(_ context.Context, _ string, _ UpdatePropertyClausesInput) error {
	return nil
}

func (m *mockService) GetServices(_ context.Context, _ string) (GetPropertyServicesResult, error) {
	return GetPropertyServicesResult{}, nil
}

func (m *mockService) UpdateServices(_ context.Context, _ string, _ UpdatePropertyServicesInput) error {
	return nil
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
				"category":         "land",
				"title":            "Terreno demo",
				"description":      "Una propiedad de prueba",
				"property_type_id": 2,
				"modality_id":      3,
				"lot_area":         200,
				"is_featured":      false,
				"location": gin.H{
					"city_id":           1,
					"street":            "Av. Principal",
					"exterior_number":   "45",
					"postal_code":       "91000",
					"latitude":          19.5438,
					"longitude":         -96.9102,
					"is_public_address": true,
				},
			},
			mock: &mockService{
				result: CreatePropertyResult{
					Data: CreatePropertyResultData{
						PropertyUUID: "123e4567-e89b-12d3-a456-426614174000",
					},
				},
			},
			wantStatusCode:   http.StatusCreated,
			wantBodyContains: "\"property_uuid\":\"123e4567-e89b-12d3-a456-426614174000\"",
			wantCalled:       true,
		},
		{
			name: "rejects invalid payload",
			payload: gin.H{
				"category":         "land",
				"title":            "Terreno demo",
				"property_type_id": 2,
				"modality_id":      3,
				"location": gin.H{
					"city_id":           1,
					"street":            "Av. Principal",
					"exterior_number":   "45",
					"latitude":          19.5438,
					"longitude":         -96.9102,
					"is_public_address": true,
				},
			},
			mock:             &mockService{},
			wantStatusCode:   http.StatusBadRequest,
			wantBodyContains: "owner_id must be greater than 0",
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

			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/properties", bytes.NewReader(body))
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
	isPublicAddress := true
	latitude := 19.5438
	longitude := -96.9102

	tests := []struct {
		name    string
		req     CreatePropertyInput
		wantErr string
	}{
		{
			name: "valid request",
			req: CreatePropertyInput{
				OwnerID:        1,
				Category:       CategoryLand,
				Title:          "Terreno demo",
				Description:    "Descripcion",
				PropertyTypeID: 2,
				ModalityID:     3,
				Location: &CreateLocationInput{
					CityID:          1,
					Street:          "Av. Principal",
					ExteriorNumber:  "45",
					Latitude:        &latitude,
					Longitude:       &longitude,
					IsPublicAddress: &isPublicAddress,
				},
			},
		},
		{
			name: "missing owner",
			req: CreatePropertyInput{
				Category:       CategoryLand,
				Title:          "Terreno demo",
				PropertyTypeID: 2,
				ModalityID:     3,
				Location: &CreateLocationInput{
					CityID:          1,
					Street:          "Av. Principal",
					ExteriorNumber:  "45",
					Latitude:        &latitude,
					Longitude:       &longitude,
					IsPublicAddress: &isPublicAddress,
				},
			},
			wantErr: "owner_id must be greater than 0",
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
