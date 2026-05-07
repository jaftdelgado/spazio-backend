package properties

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"errors"

	"github.com/gin-gonic/gin"
)

// minimalValidBody is the minimum valid JSON payload for createProperty.
const minimalValidBody = `{
	"owner_id": 1,
	"title": "Casa",
	"property_type_id": 1,
	"modality_id": 1,
	"location": {
		"city_id": 1,
		"street": "Av. Principal",
		"exterior_number": "45",
		"postal_code": "91000",
		"latitude": 19.5,
		"longitude": -96.9,
		"is_public_address": true
	}
}`

func TestHandler_CreateProperty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
	}{
		// Forbidden fields
		{
			name:       "returns bad request when category is forbidden",
			body:       `{"category":"x","owner_id":1}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when subtype is forbidden",
			body:       `{"subtype":"residential","owner_id":1}`,
			wantStatus: http.StatusBadRequest,
		},
		// Binding
		{
			name:       "returns bad request when json is invalid",
			body:       `{bad json}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when body is empty",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		// Base field validations
		{
			name:       "returns bad request when owner id is zero",
			body:       `{"owner_id":0,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when owner id is negative",
			body:       `{"owner_id":-1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when title is empty after sanitization",
			body:       `{"owner_id":1,"title":"   ","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when property type id is zero",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":0,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when modality id is zero",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":0,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when location is missing",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when location city id is zero",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":0,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when location street is empty",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"   ","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when location exterior number is empty",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"   ","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when location latitude is missing",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","longitude":-96.9,"is_public_address":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when location longitude is missing",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"is_public_address":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when location public address flag is missing",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9}}`,
			wantStatus: http.StatusBadRequest,
		},
		// Optional price validations
		{
			name:       "returns bad request when sale price amount is missing",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true},"sale_price":{"currency":"MXN","is_negotiable":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when sale price currency is missing",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true},"sale_price":{"sale_price":100000,"is_negotiable":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when sale price negotiable flag is missing",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true},"sale_price":{"sale_price":100000,"currency":"MXN"}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when rent price period id is zero",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true},"rent_prices":[{"period_id":0,"rent_price":8000,"currency":"MXN","is_negotiable":false}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when rent price amount is missing",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true},"rent_prices":[{"period_id":1,"currency":"MXN","is_negotiable":false}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when rent price currency is empty",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true},"rent_prices":[{"period_id":1,"rent_price":8000,"currency":"","is_negotiable":false}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when rent price negotiable flag is missing",
			body:       `{"owner_id":1,"title":"Casa","property_type_id":1,"modality_id":1,"location":{"city_id":1,"street":"Av","exterior_number":"1","postal_code":"91000","latitude":19.5,"longitude":-96.9,"is_public_address":true},"rent_prices":[{"period_id":1,"rent_price":8000,"currency":"MXN"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		// Service errors
		{
			name:       "returns bad request when validation fails",
			body:       minimalValidBody,
			mockErr:    ValidationError{Message: "subtype validation failed"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns internal server error when service fails",
			body:       minimalValidBody,
			mockErr:    errors.New("db"),
			wantStatus: http.StatusInternalServerError,
		},
		// Happy path
		{
			name:       "returns created when request is valid",
			body:       minimalValidBody,
			mockErr:    nil,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockServiceForClauses{
				createPropertyFunc: func(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
					if tt.mockErr != nil {
						return CreatePropertyResult{}, tt.mockErr
					}
					return CreatePropertyResult{Data: CreatePropertyResultData{PropertyUUID: "abc-123"}}, nil
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/properties", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx.Request = req

			handler := NewHandler(svcMock)
			handler.createProperty(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusCreated {
				var result CreatePropertyResult
				if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
					t.Fatalf("decode response body: %v", err)
				}
				if result.Data.PropertyUUID == "" {
					t.Fatal("expected non-empty property_uuid in response")
				}
			}
		})
	}
}
