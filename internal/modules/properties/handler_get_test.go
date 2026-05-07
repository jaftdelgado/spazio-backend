package properties

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		mockResult     ListPropertiesResult
		mockErr        error
		wantStatus     int
		expectMinPrice float64
		expectMaxPrice float64
		expectBedrooms int32
	}{
		{
			name:        "valid search with all CU-12 filters",
			queryParams: "?min_price=1000&max_price=5000&min_bedrooms=3&q=xalapa",
			mockResult: ListPropertiesResult{
				Data: []PropertyCardData{{PropertyUUID: "abc", Title: "Casa Xalapa"}},
				Meta: ListPropertiesMeta{TotalCount: 1, TotalPages: 1},
			},
			wantStatus:     http.StatusOK,
			expectMinPrice: 1000,
			expectMaxPrice: 5000,
			expectBedrooms: 3,
		},
		{
			name:        "invalid min_price format",
			queryParams: "?min_price=invalid",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid min_bedrooms format",
			queryParams: "?min_bedrooms=-1",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "page size too large",
			queryParams: "?page_size=101",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockPropertyService{
				listPropertiesFunc: func(ctx context.Context, input ListPropertiesInput) (ListPropertiesResult, error) {
					if tt.name == "valid search with all CU-12 filters" {
						if input.MinPrice != tt.expectMinPrice || input.MaxPrice != tt.expectMaxPrice || input.MinBedrooms != tt.expectBedrooms {
							return ListPropertiesResult{}, fmt.Errorf("params mismatch: got price[%v-%v] beds[%v]", input.MinPrice, input.MaxPrice, input.MinBedrooms)
						}
					}
					return tt.mockResult, tt.mockErr
				},
			}

			h := NewHandler(mockSvc)
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/api/v1/properties", h.listProperties)
			
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/properties"+tt.queryParams, nil)
			c.Request = req
			
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var res ListPropertiesResult
				if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if len(res.Data) != len(tt.mockResult.Data) {
					t.Errorf("data length: got %d, want %d", len(res.Data), len(tt.mockResult.Data))
				}
			}
		})
	}
}

// Mock definitions for Service
type mockPropertyService struct {
	PropertyService // Embed to satisfy interface
	listPropertiesFunc func(ctx context.Context, input ListPropertiesInput) (ListPropertiesResult, error)
	getPropertyFunc    func(ctx context.Context, propertyUUID string) (GetPropertyResult, error)
	getFullPropertyFunc func(ctx context.Context, propertyUUID string) (GetPropertyFullResult, error)
	getPropertyHistoryFunc func(ctx context.Context, propertyUUID string, requesterID int32, requesterRoleID int32) (GetPropertyHistoryResult, error)
}

func (m *mockPropertyService) ListProperties(ctx context.Context, input ListPropertiesInput) (ListPropertiesResult, error) {
	return m.listPropertiesFunc(ctx, input)
}

func (m *mockPropertyService) GetProperty(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
	return m.getPropertyFunc(ctx, propertyUUID)
}

func (m *mockPropertyService) GetFullProperty(ctx context.Context, propertyUUID string) (GetPropertyFullResult, error) {
	return m.getFullPropertyFunc(ctx, propertyUUID)
}

func (m *mockPropertyService) GetPropertyHistory(ctx context.Context, propertyUUID string, requesterID int32, requesterRoleID int32) (GetPropertyHistoryResult, error) {
	return m.getPropertyHistoryFunc(ctx, propertyUUID, requesterID, requesterRoleID)
}

func TestHandler_GetPropertyHistory_CU18(t *testing.T) {
	tests := []struct {
		name        string
		propertyUUID string
		headers     map[string]string
		mockResult  GetPropertyHistoryResult
		mockErr     error
		wantStatus  int
	}{
		{
			name:         "returns history successfully",
			propertyUUID: "abc-123",
			headers:      map[string]string{"X-User-ID": "1", "X-Role-ID": "1"},
			mockResult:   GetPropertyHistoryResult{Data: []PropertyStatusHistoryData{{HistoryID: 1}}},
			wantStatus:   http.StatusOK,
		},
		{
			name:         "missing X-User-ID header",
			propertyUUID: "abc-123",
			headers:      map[string]string{"X-Role-ID": "1"},
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "forbidden error from service",
			propertyUUID: "abc-123",
			headers:      map[string]string{"X-User-ID": "2", "X-Role-ID": "3"},
			mockErr:      fmt.Errorf("forbidden: access denied"),
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "property not found",
			propertyUUID: "non-existent",
			headers:      map[string]string{"X-User-ID": "1", "X-Role-ID": "1"},
			mockErr:      ErrPropertyNotFound,
			wantStatus:   http.StatusNotFound,
		},
		{
			name:         "internal server error",
			propertyUUID: "abc-123",
			headers:      map[string]string{"X-User-ID": "1", "X-Role-ID": "1"},
			mockErr:      fmt.Errorf("random error"),
			wantStatus:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockPropertyService{
				getPropertyHistoryFunc: func(ctx context.Context, propertyUUID string, requesterID int32, requesterRoleID int32) (GetPropertyHistoryResult, error) {
					return tt.mockResult, tt.mockErr
				},
			}

			h := NewHandler(mockSvc)
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/api/v1/properties/:uuid/history", h.getPropertyHistory)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/properties/"+tt.propertyUUID+"/history", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
