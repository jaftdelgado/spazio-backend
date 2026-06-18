package properties

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHandler_ListProperties_CU12(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setAuth        bool
		mockResult     ListPropertiesResult
		mockErr        error
		wantStatus     int
		expectMinPrice float64
		expectMaxPrice float64
		expectBedrooms int32
		expectUserID   int32
		expectRoleID   int32
	}{
		{
			name:        "valid search with all CU-12 filters",
			queryParams: "?min_price=1000&max_price=5000&min_bedrooms=3&q=xalapa",
			setAuth:     true,
			mockResult: ListPropertiesResult{
				Data: []PropertyCardData{{
					PropertyUUID:   "abc",
					Title:          "Casa Xalapa",
					CoverPhotoURL:  ptrString("https://pub-ab9b26339b564d53b2f5ec019d1ca830.r2.dev/properties/abc/photos/cover.webp"),
					AddressSummary: "Av. Principal 45, Centro, Xalapa, Veracruz, Mexico",
					AssignedAgent: &PropertyAgentData{
						UserID:    21,
						UserUUID:  "agent-uuid",
						FirstName: "Ada",
						LastName:  "Lovelace",
					},
					Location: PropertyCardLocationData{
						CountryID:   1,
						CountryName: "Mexico",
						StateID:     30,
						StateName:   "Veracruz",
						CityID:      3001,
						CityName:    "Xalapa",
					},
				}},
				Meta: ListPropertiesMeta{TotalCount: 1, TotalPages: 1},
			},
			wantStatus:     http.StatusOK,
			expectMinPrice: 1000,
			expectMaxPrice: 5000,
			expectBedrooms: 3,
			expectUserID:   10,
			expectRoleID:   RoleAdminID,
		},
		{
			name:         "allows guest reader when auth context is missing",
			queryParams:  "",
			wantStatus:   http.StatusOK,
			expectUserID: 0,
			expectRoleID: RoleClientID,
		},
		{
			name:        "invalid min_price format",
			queryParams: "?min_price=invalid",
			setAuth:     true,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid min_bedrooms format",
			queryParams: "?min_bedrooms=-1",
			setAuth:     true,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "page size too large",
			queryParams: "?page_size=101",
			setAuth:     true,
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockPropertyService{
				listPropertiesFunc: func(ctx context.Context, input ListPropertiesInput) (ListPropertiesResult, error) {
					if tt.wantStatus == http.StatusOK {
						if input.UserID != tt.expectUserID || input.RoleID != tt.expectRoleID {
							return ListPropertiesResult{}, errors.New("auth context mismatch")
						}
					}

					if tt.name == "valid search with all CU-12 filters" {
						if input.MinPrice != tt.expectMinPrice || input.MaxPrice != tt.expectMaxPrice || input.MinBedrooms != tt.expectBedrooms {
							return ListPropertiesResult{}, errors.New("filter mismatch")
						}
					}
					return tt.mockResult, tt.mockErr
				},
			}

			h := NewHandler(mockSvc)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/properties"+tt.queryParams, nil)
			c.Request = req
			if tt.setAuth {
				c.Set("user_id", int32(10))
				c.Set("role_id", int32(RoleAdminID))
				c.Set("user_role", "Admin")
			}

			h.listProperties(c)

			if w.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var res ListPropertiesResult
				if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if len(res.Data) != len(tt.mockResult.Data) {
					t.Fatalf("data length: got %d, want %d", len(res.Data), len(tt.mockResult.Data))
				}
				if tt.name == "valid search with all CU-12 filters" {
					if res.Data[0].Location.CityName != "Xalapa" {
						t.Fatalf("location.city_name = %q, want Xalapa", res.Data[0].Location.CityName)
					}
					if res.Data[0].AddressSummary == "" {
						t.Fatal("address_summary should not be empty")
					}
					if res.Data[0].CoverPhotoURL == nil || !strings.HasPrefix(*res.Data[0].CoverPhotoURL, "https://") {
						t.Fatalf("cover_photo_url = %#v, want public https url", res.Data[0].CoverPhotoURL)
					}
					if res.Data[0].AssignedAgent == nil || res.Data[0].AssignedAgent.UserID != 21 {
						t.Fatalf("assigned_agent = %#v, want user_id 21", res.Data[0].AssignedAgent)
					}
				}
			}
		})
	}
}

type mockPropertyService struct {
	createPropertyFunc     func(ctx context.Context, userID int32, input CreatePropertyInput) (CreatePropertyResult, error)
	listPropertiesFunc     func(ctx context.Context, input ListPropertiesInput) (ListPropertiesResult, error)
	getPropertyForRoleFunc func(ctx context.Context, propertyUUID string, userID int32, roleID int32) (GetPropertyResult, error)
	getPricesHistoryFunc   func(ctx context.Context, propertyUUID string) (GetPropertyPricesHistoryResult, error)
	getPropertyHistoryFunc func(ctx context.Context, propertyUUID string) (GetPropertyHistoryResult, error)
}

func (m *mockPropertyService) CreateProperty(ctx context.Context, userID int32, input CreatePropertyInput) (CreatePropertyResult, error) {
	if m.createPropertyFunc != nil {
		return m.createPropertyFunc(ctx, userID, input)
	}
	return CreatePropertyResult{}, nil
}

func (m *mockPropertyService) ListProperties(ctx context.Context, input ListPropertiesInput) (ListPropertiesResult, error) {
	if m.listPropertiesFunc != nil {
		return m.listPropertiesFunc(ctx, input)
	}
	return ListPropertiesResult{}, nil
}

func (m *mockPropertyService) GetClauses(ctx context.Context, propertyUUID string) (GetPropertyClausesResult, error) {
	return GetPropertyClausesResult{}, nil
}

func (m *mockPropertyService) UpdateClauses(ctx context.Context, propertyUUID string, input UpdatePropertyClausesInput) error {
	return nil
}

func (m *mockPropertyService) GetPhotos(ctx context.Context, propertyUUID string) (GetPropertyPhotosResult, error) {
	return GetPropertyPhotosResult{}, nil
}

func (m *mockPropertyService) UpdatePhotos(ctx context.Context, propertyUUID string, input UpdatePropertyPhotosInput) error {
	return nil
}

func (m *mockPropertyService) GetServices(ctx context.Context, propertyUUID string) (GetPropertyServicesResult, error) {
	return GetPropertyServicesResult{}, nil
}

func (m *mockPropertyService) UpdateServices(ctx context.Context, propertyUUID string, input UpdatePropertyServicesInput) error {
	return nil
}

func (m *mockPropertyService) GetPrices(ctx context.Context, propertyUUID string) (GetPropertyPricesResult, error) {
	return GetPropertyPricesResult{}, nil
}

func (m *mockPropertyService) GetPricesHistory(ctx context.Context, propertyUUID string) (GetPropertyPricesHistoryResult, error) {
	if m.getPricesHistoryFunc != nil {
		return m.getPricesHistoryFunc(ctx, propertyUUID)
	}
	return GetPropertyPricesHistoryResult{}, nil
}

func (m *mockPropertyService) UpdatePrices(ctx context.Context, propertyUUID string, input UpdatePropertyPricesInput) error {
	return nil
}

func (m *mockPropertyService) GetPropertyForRole(ctx context.Context, propertyUUID string, userID int32, roleID int32) (GetPropertyResult, error) {
	if m.getPropertyForRoleFunc != nil {
		return m.getPropertyForRoleFunc(ctx, propertyUUID, userID, roleID)
	}
	return GetPropertyResult{}, nil
}

func (m *mockPropertyService) UpdateProperty(ctx context.Context, propertyUUID string, input UpdatePropertyInput) (UpdatePropertyResult, error) {
	return UpdatePropertyResult{}, nil
}

func (m *mockPropertyService) DeleteProperty(ctx context.Context, propertyUUID string, input DeletePropertyInput) error {
	return nil
}

func (m *mockPropertyService) GetPropertyHistory(ctx context.Context, propertyUUID string) (GetPropertyHistoryResult, error) {
	if m.getPropertyHistoryFunc != nil {
		return m.getPropertyHistoryFunc(ctx, propertyUUID)
	}
	return GetPropertyHistoryResult{}, nil
}

func TestHandler_GetProperty(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name         string
		uuid         string
		setAuth      bool
		mockResult   GetPropertyResult
		mockErr      error
		wantStatus   int
		expectUserID int32
		expectRoleID int32
	}{
		{
			name:       "returns bad request when uuid is empty",
			uuid:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "allows guest reader when auth context is missing",
			uuid: validUUID,
			mockResult: GetPropertyResult{
				Data: GetPropertyData{
					PropertyUUID: validUUID,
					Title:        "Casa pública",
					AssignedAgent: &PropertyAgentData{
						UserID:    21,
						UserUUID:  "agent-uuid",
						FirstName: "Ada",
						LastName:  "Lovelace",
					},
					Location: &LocationData{
						CountryName: "Mexico",
						StateName:   "Veracruz",
						CityName:    "Xalapa",
						Latitude:    19.5438,
						Longitude:   -96.9102,
					},
				},
			},
			wantStatus:   http.StatusOK,
			expectUserID: 0,
			expectRoleID: RoleClientID,
		},
		{
			name:         "returns forbidden when property is not assigned to agent",
			uuid:         validUUID,
			setAuth:      true,
			mockErr:      errors.New("forbidden: property not assigned to agent"),
			wantStatus:   http.StatusForbidden,
			expectUserID: 10,
			expectRoleID: RoleAdminID,
		},
		{
			name:         "returns not found when property does not exist",
			uuid:         validUUID,
			setAuth:      true,
			mockErr:      ErrPropertyNotFound,
			wantStatus:   http.StatusNotFound,
			expectUserID: 10,
			expectRoleID: RoleAdminID,
		},
		{
			name:         "returns internal server error when service fails",
			uuid:         validUUID,
			setAuth:      true,
			mockErr:      errors.New("db"),
			wantStatus:   http.StatusInternalServerError,
			expectUserID: 10,
			expectRoleID: RoleAdminID,
		},
		{
			name:         "returns ok with property data",
			uuid:         validUUID,
			setAuth:      true,
			expectUserID: 10,
			expectRoleID: RoleAdminID,
			mockResult: GetPropertyResult{
				Data: GetPropertyData{
					PropertyUUID: validUUID,
					Title:        "Casa",
					RegisteredBy: "Admin User",
					AssignedAgent: &PropertyAgentData{
						UserID:    21,
						UserUUID:  "agent-uuid",
						FirstName: "Ada",
						LastName:  "Lovelace",
					},
					Location: &LocationData{
						CountryID:       1,
						CountryName:     "Mexico",
						StateID:         30,
						StateName:       "Veracruz",
						CityID:          3001,
						CityName:        "Xalapa",
						Neighborhood:    "Centro",
						Street:          "Av. Principal",
						ExteriorNumber:  "45",
						PostalCode:      "91000",
						Latitude:        19.5438,
						Longitude:       -96.9102,
						IsPublicAddress: true,
					},
				},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockPropertyService{
				getPropertyForRoleFunc: func(ctx context.Context, propertyUUID string, userID int32, roleID int32) (GetPropertyResult, error) {
					if tt.uuid != "" {
						if userID != tt.expectUserID || roleID != tt.expectRoleID {
							t.Fatalf(
								"auth values: got userID=%d roleID=%d, want userID=%d roleID=%d",
								userID,
								roleID,
								tt.expectUserID,
								tt.expectRoleID,
							)
						}
					}
					return tt.mockResult, tt.mockErr
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/properties/"+tt.uuid, nil)
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}
			if tt.setAuth {
				ctx.Set("user_id", int32(10))
				ctx.Set("role_id", int32(RoleAdminID))
				ctx.Set("user_role", "Admin")
			}

			handler := NewHandler(svcMock)
			handler.getProperty(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var res GetPropertyResult
				if err := json.NewDecoder(recorder.Body).Decode(&res); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if res.Data.Location == nil {
					t.Fatal("location should not be nil")
				}
				if res.Data.Location.CountryName != "Mexico" {
					t.Fatalf("location.country_name = %q, want Mexico", res.Data.Location.CountryName)
				}
				if res.Data.Location.StateName != "Veracruz" {
					t.Fatalf("location.state_name = %q, want Veracruz", res.Data.Location.StateName)
				}
				if res.Data.Location.CityName != "Xalapa" {
					t.Fatalf("location.city_name = %q, want Xalapa", res.Data.Location.CityName)
				}
				if res.Data.Location.Latitude != 19.5438 {
					t.Fatalf("location.latitude = %v, want 19.5438", res.Data.Location.Latitude)
				}
				if res.Data.Location.Longitude != -96.9102 {
					t.Fatalf("location.longitude = %v, want -96.9102", res.Data.Location.Longitude)
				}
				if res.Data.AssignedAgent == nil || res.Data.AssignedAgent.UserID != 21 {
					t.Fatalf("assigned_agent = %#v, want user_id 21", res.Data.AssignedAgent)
				}
			}
		})
	}
}

func TestHandler_GetPricesHistory(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		uuid       string
		mockResult GetPropertyPricesHistoryResult
		mockErr    error
		wantStatus int
	}{
		{
			name:       "returns bad request when uuid is empty",
			uuid:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns not found when property does not exist",
			uuid:       validUUID,
			mockErr:    ErrPropertyNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "returns internal server error when service fails",
			uuid:       validUUID,
			mockErr:    errors.New("db"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "returns ok with prices history data",
			uuid: validUUID,
			mockResult: GetPropertyPricesHistoryResult{
				Data: []PropertyPriceHistoryData{{PriceType: "sale", Amount: 1000}},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockPropertyService{
				getPricesHistoryFunc: func(ctx context.Context, propertyUUID string) (GetPropertyPricesHistoryResult, error) {
					return tt.mockResult, tt.mockErr
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/properties/"+tt.uuid+"/prices/history", nil)
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}

			handler := NewHandler(svcMock)
			handler.getPricesHistory(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_GetPropertyHistory_CU18(t *testing.T) {
	tests := []struct {
		name         string
		propertyUUID string
		mockResult   GetPropertyHistoryResult
		mockErr      error
		wantStatus   int
	}{
		{
			name:         "returns history successfully",
			propertyUUID: "abc-123",
			mockResult:   GetPropertyHistoryResult{Data: []PropertyStatusHistoryData{{HistoryID: 1}}},
			wantStatus:   http.StatusOK,
		},
		{
			name:         "property not found",
			propertyUUID: "non-existent",
			mockErr:      ErrPropertyNotFound,
			wantStatus:   http.StatusNotFound,
		},
		{
			name:         "internal server error",
			propertyUUID: "abc-123",
			mockErr:      errors.New("random error"),
			wantStatus:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockPropertyService{
				getPropertyHistoryFunc: func(ctx context.Context, propertyUUID string) (GetPropertyHistoryResult, error) {
					return tt.mockResult, tt.mockErr
				},
			}

			h := NewHandler(mockSvc)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/properties/"+tt.propertyUUID+"/history", nil)
			c.Request = req
			c.Params = gin.Params{{Key: "uuid", Value: tt.propertyUUID}}

			h.getPropertyHistory(c)

			if w.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
