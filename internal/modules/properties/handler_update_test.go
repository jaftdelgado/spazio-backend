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

// TestHandler_UpdateProperty tests the updateProperty handler scenarios in table-driven format.
func TestHandler_UpdateProperty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	validResidential := &UpdateResidentialInput{
		Bedrooms:         ptrInt16(3),
		Bathrooms:        ptrInt16(2),
		Beds:             ptrInt16(4),
		Floors:           ptrInt16(2),
		ParkingSpots:     ptrInt16(1),
		BuiltArea:        ptrFloat64(120.5),
		ConstructionYear: ptrInt16(2010),
		OrientationID:    ptrInt32(2),
		IsFurnished:      ptrBool(true),
	}

	validCommercial := &UpdateCommercialInput{
		CeilingHeight:   ptrFloat64(4.5),
		LoadingDocks:    ptrInt16(2),
		InternalOffices: ptrInt16(3),
		ThreePhasePower: ptrBool(true),
		LandUse:         ptrString("Retail"),
	}

	validLocation := &UpdateLocationInput{
		CityID:          ptrInt32(1),
		Neighborhood:    ptrString("Centro"),
		Street:          ptrString("Principal"),
		ExteriorNumber:  ptrString("123"),
		PostalCode:      ptrString("91000"),
		Latitude:        ptrFloat64(19.5),
		Longitude:       ptrFloat64(-96.9),
		IsPublicAddress: ptrBool(true),
	}

	tests := []struct {
		name       string
		uuid       string
		rawBody    []byte
		mockErr    error
		wantStatus int
	}{
		{
			name:       "returns bad request when uuid is empty",
			uuid:       "",
			rawBody:    []byte(`{}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when category is forbidden",
			uuid:       validUUID,
			rawBody:    []byte(`{"category":"x"}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when subtype is forbidden",
			uuid:       validUUID,
			rawBody:    []byte(`{"subtype":"residential"}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when json is invalid",
			uuid:       validUUID,
			rawBody:    []byte(`{bad json}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when title is empty",
			uuid:       validUUID,
			rawBody:    []byte(`{"title":""}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when title contains only spaces",
			uuid:       validUUID,
			rawBody:    []byte(`{"title":"   "}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when lot area is zero",
			uuid:       validUUID,
			rawBody:    []byte(`{"lot_area":0}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when lot area is negative",
			uuid:       validUUID,
			rawBody:    []byte(`{"lot_area":-1}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when residential field is missing",
			uuid:       validUUID,
			rawBody:    []byte(`{"residential":{"bathrooms":1}}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when residential orientation id is zero",
			uuid:       validUUID,
			rawBody:    mustMarshal(UpdatePropertyInput{Residential: &UpdateResidentialInput{Bedrooms: ptrInt16(3), Bathrooms: ptrInt16(2), Beds: ptrInt16(4), Floors: ptrInt16(2), ParkingSpots: ptrInt16(1), BuiltArea: ptrFloat64(120.5), ConstructionYear: ptrInt16(2010), IsFurnished: ptrBool(true), OrientationID: ptrInt32(0)}}),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when commercial field is missing",
			uuid:       validUUID,
			rawBody:    []byte(`{"commercial":{"loading_docks":1}}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when commercial land use is empty",
			uuid:       validUUID,
			rawBody:    mustMarshal(UpdatePropertyInput{Commercial: &UpdateCommercialInput{CeilingHeight: ptrFloat64(4.5), LoadingDocks: ptrInt16(2), InternalOffices: ptrInt16(3), ThreePhasePower: ptrBool(true), LandUse: ptrString("")}}),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when location field is missing",
			uuid:       validUUID,
			rawBody:    []byte(`{"location":{"neighborhood":"Centro"}}`),
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when validation error occurs",
			uuid:       validUUID,
			rawBody:    []byte(`{"title":"Casa"}`),
			mockErr:    ValidationError{Message: "validation err"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns not found when property does not exist",
			uuid:       validUUID,
			rawBody:    []byte(`{"title":"Casa"}`),
			mockErr:    ErrPropertyNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "returns internal server error when service fails",
			uuid:       validUUID,
			rawBody:    []byte(`{"title":"Casa"}`),
			mockErr:    errors.New("db"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "returns ok when payload is empty",
			uuid:       validUUID,
			rawBody:    []byte(`{}`),
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns ok when only title is provided",
			uuid:       validUUID,
			rawBody:    []byte(`{"title":"Casa"}`),
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns ok when residential data is valid",
			uuid:       validUUID,
			rawBody:    mustMarshal(UpdatePropertyInput{Residential: validResidential}),
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns ok when commercial data is valid",
			uuid:       validUUID,
			rawBody:    mustMarshal(UpdatePropertyInput{Commercial: validCommercial}),
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns ok when location data is valid",
			uuid:       validUUID,
			rawBody:    mustMarshal(UpdatePropertyInput{Location: validLocation}),
			mockErr:    nil,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockServiceForClauses{
				updatePropertyFunc: func(ctx context.Context, uuid string, input UpdatePropertyInput) (UpdatePropertyResult, error) {
					if tt.mockErr != nil {
						return UpdatePropertyResult{}, tt.mockErr
					}
					return UpdatePropertyResult{Message: "updated"}, nil
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/properties/123", bytes.NewReader(tt.rawBody))
			req.Header.Set("Content-Type", "application/json")
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}

			handler := NewHandler(svcMock)
			handler.updateProperty(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var body map[string]interface{}
				if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
					t.Fatalf("decode response body: %v", err)
				}
				if _, ok := body["message"]; !ok {
					t.Fatal("response body missing key 'message'")
				}
			}
		})
	}
}

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
