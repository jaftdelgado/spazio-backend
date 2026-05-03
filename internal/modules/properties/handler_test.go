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
	result             CreatePropertyResult
	err                error
	input              CreatePropertyInput
	called             bool
	listResult         ListPropertiesResult
	listErr            error
	listInput          ListPropertiesInput
	listCalled         bool
	propertyResult     GetPropertyResult
	propertyErr        error
	propertyCalled     bool
	fullPropertyResult GetPropertyFullResult
	fullPropertyErr    error
	fullPropertyCalled bool
}

func (m *mockService) CreateProperty(_ context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
	m.called = true
	m.input = input
	return m.result, m.err
}

func (m *mockService) ListProperties(_ context.Context, input ListPropertiesInput) (ListPropertiesResult, error) {
	m.listCalled = true
	m.listInput = input
	return m.listResult, m.listErr
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

func (m *mockService) GetPhotos(_ context.Context, _ string) (GetPropertyPhotosResult, error) {
	return GetPropertyPhotosResult{}, nil
}

func (m *mockService) UpdatePhotos(_ context.Context, _ string, _ UpdatePropertyPhotosInput) error {
	return nil
}

func (m *mockService) UpdateServices(_ context.Context, _ string, _ UpdatePropertyServicesInput) error {
	return nil
}

func (m *mockService) GetPrices(_ context.Context, _ string) (GetPropertyPricesResult, error) {
	return GetPropertyPricesResult{}, nil
}

func (m *mockService) UpdatePrices(_ context.Context, _ string, _ UpdatePropertyPricesInput) error {
	return nil
}

func (m *mockService) GetProperty(_ context.Context, _ string) (GetPropertyResult, error) {
	m.propertyCalled = true
	return m.propertyResult, m.propertyErr
}

func (m *mockService) GetFullProperty(_ context.Context, _ string) (GetPropertyFullResult, error) {
	m.fullPropertyCalled = true
	return m.fullPropertyResult, m.fullPropertyErr
}

func (m *mockService) UpdateProperty(_ context.Context, _ string, _ UpdatePropertyInput) (UpdatePropertyResult, error) {
	return UpdatePropertyResult{}, nil
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
				Subtype:        SubtypeOther,
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
				Subtype:        SubtypeOther,
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

func TestCreatePropertyRejectsForbiddenFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	payload := gin.H{
		"owner_id":         1,
		"category":         "other",
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
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/properties", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	mock := &mockService{}
	handler := NewHandler(mock)
	handler.createProperty(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	if mock.called {
		t.Fatalf("called = %v, want false", mock.called)
	}
}

func TestUpdatePropertyRejectsForbiddenFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name  string
		field string
	}{
		{name: "rejects category", field: "category"},
		{name: "rejects subtype", field: "subtype"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			payload := gin.H{
				tt.field: "other",
			}

			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}

			ctx.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/properties/test-uuid", bytes.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Params = gin.Params{{Key: "uuid", Value: "test-uuid"}}

			handler := NewHandler(&mockService{})
			handler.updateProperty(ctx)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestListProperties(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/properties?page=2&page_size=10&q=downtown&status_id=1&status_id=2&property_type_id=3&modality_id=2&country_id=1&state_id=4&city_id=9&sort=price&order=asc",
		nil,
	)

	mock := &mockService{
		listResult: ListPropertiesResult{
			Data: []PropertyCardData{
				{PropertyUUID: "123e4567-e89b-12d3-a456-426614174000", Title: "Apartment in Downtown"},
			},
			Meta: ListPropertiesMeta{
				TotalCount:  1,
				TotalPages:  1,
				CurrentPage: 2,
				PageSize:    10,
				HasNext:     false,
				HasPrev:     true,
			},
		},
	}

	handler := NewHandler(mock)
	handler.listProperties(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	if !mock.listCalled {
		t.Fatalf("listCalled = false, want true")
	}

	if mock.listInput.Page != 2 || mock.listInput.PageSize != 10 {
		t.Fatalf("unexpected pagination input: %+v", mock.listInput)
	}

	if mock.listInput.Query != "downtown" {
		t.Fatalf("query = %q, want %q", mock.listInput.Query, "downtown")
	}

	if len(mock.listInput.StatusIDs) != 2 || mock.listInput.StatusIDs[0] != 1 || mock.listInput.StatusIDs[1] != 2 {
		t.Fatalf("status_ids = %+v, want [1 2]", mock.listInput.StatusIDs)
	}

	if mock.listInput.Sort != "price" || mock.listInput.Order != "asc" {
		t.Fatalf("sort/order = %q/%q, want price/asc", mock.listInput.Sort, mock.listInput.Order)
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte("Apartment in Downtown")) {
		t.Fatalf("body %q does not contain %q", recorder.Body.String(), "Apartment in Downtown")
	}
}

func TestGetPropertyWithFullQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/properties/test-uuid?full=true", nil)
	ctx.Params = gin.Params{{Key: "uuid", Value: "test-uuid"}}

	mock := &mockService{
		fullPropertyResult: GetPropertyFullResult{
			Data: GetPropertyFullData{
				PropertyUUID: "test-uuid",
				Photos:       []PropertyPhotoData{},
				Services:     []int32{},
				Clauses:      []PropertyClauseData{},
				Prices: PropertyFullPricesData{
					Current: PropertyCurrentPricesData{Rent: []CurrentRentPriceDetailData{}},
					History: []PropertyPriceHistoryData{},
				},
			},
		},
	}

	handler := NewHandler(mock)
	handler.getProperty(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	if !mock.fullPropertyCalled {
		t.Fatalf("fullPropertyCalled = false, want true")
	}

	if mock.propertyCalled {
		t.Fatalf("propertyCalled = true, want false")
	}
}

func TestGetPropertyWithoutFullQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/properties/test-uuid", nil)
	ctx.Params = gin.Params{{Key: "uuid", Value: "test-uuid"}}

	mock := &mockService{
		propertyResult: GetPropertyResult{
			Data: GetPropertyData{
				PropertyUUID: "test-uuid",
				Title:        "Property title",
			},
		},
	}

	handler := NewHandler(mock)
	handler.getProperty(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	if !mock.propertyCalled {
		t.Fatalf("propertyCalled = false, want true")
	}

	if mock.fullPropertyCalled {
		t.Fatalf("fullPropertyCalled = true, want false")
	}
}

func TestGetPropertyNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("not found without full", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/properties/test-uuid", nil)
		ctx.Params = gin.Params{{Key: "uuid", Value: "test-uuid"}}

		mock := &mockService{
			propertyErr: ErrPropertyNotFound,
		}

		handler := NewHandler(mock)
		handler.getProperty(ctx)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNotFound)
		}

		if !bytes.Contains(recorder.Body.Bytes(), []byte(ErrPropertyNotFound.Error())) {
			t.Fatalf("body %q does not contain %q", recorder.Body.String(), ErrPropertyNotFound.Error())
		}
	})

	t.Run("not found with full", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/properties/test-uuid?full=true", nil)
		ctx.Params = gin.Params{{Key: "uuid", Value: "test-uuid"}}

		mock := &mockService{
			fullPropertyErr: ErrPropertyNotFound,
		}

		handler := NewHandler(mock)
		handler.getProperty(ctx)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNotFound)
		}

		if !bytes.Contains(recorder.Body.Bytes(), []byte(ErrPropertyNotFound.Error())) {
			t.Fatalf("body %q does not contain %q", recorder.Body.String(), ErrPropertyNotFound.Error())
		}
	})
}
