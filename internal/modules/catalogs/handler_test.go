package catalogs

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockCatalogsService struct {
	listModalitiesFunc    func(ctx context.Context) (ListModalitiesResult, error)
	listPropertyTypesFunc func(ctx context.Context) (ListPropertyTypesResult, error)
	listRentPeriodsFunc   func(ctx context.Context, propertyTypeID int32) (ListRentPeriodsResult, error)
	listOrientationsFunc  func(ctx context.Context) (ListOrientationsResult, error)
}

func (m *mockCatalogsService) ListModalities(ctx context.Context) (ListModalitiesResult, error) {
	if m.listModalitiesFunc != nil {
		return m.listModalitiesFunc(ctx)
	}

	return ListModalitiesResult{}, nil
}

func (m *mockCatalogsService) ListPropertyTypes(ctx context.Context) (ListPropertyTypesResult, error) {
	if m.listPropertyTypesFunc != nil {
		return m.listPropertyTypesFunc(ctx)
	}

	return ListPropertyTypesResult{}, nil
}

func (m *mockCatalogsService) ListRentPeriods(ctx context.Context, propertyTypeID int32) (ListRentPeriodsResult, error) {
	if m.listRentPeriodsFunc != nil {
		return m.listRentPeriodsFunc(ctx, propertyTypeID)
	}

	return ListRentPeriodsResult{}, nil
}

func (m *mockCatalogsService) ListOrientations(ctx context.Context) (ListOrientationsResult, error) {
	if m.listOrientationsFunc != nil {
		return m.listOrientationsFunc(ctx)
	}

	return ListOrientationsResult{}, nil
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func newTestContext(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, target, nil)
	ginContext.Request = request
	return ginContext, recorder
}

func assertStatusCode(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()

	if recorder.Code != want {
		t.Fatalf("status code: got %d, want %d", recorder.Code, want)
	}
}

func assertHasDataKey(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if _, ok := payload["data"]; !ok {
		t.Fatalf("expected response body to contain data key, got %s", recorder.Body.String())
	}
}

func TestHandler_ListModalities(t *testing.T) {
	tests := []struct {
		name       string
		service    *mockCatalogsService
		wantStatus int
		wantData   bool
	}{
		{
			name: "returns ok",
			service: &mockCatalogsService{
				listModalitiesFunc: func(ctx context.Context) (ListModalitiesResult, error) {
					return ListModalitiesResult{Data: []Modality{{ModalityID: 1, Name: "Rent"}}}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantData:   true,
		},
		{
			name: "service error returns internal server error",
			service: &mockCatalogsService{
				listModalitiesFunc: func(ctx context.Context) (ListModalitiesResult, error) {
					return ListModalitiesResult{}, errors.New("boom")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, recorder := newTestContext(http.MethodGet, "/api/v1/catalogs/modalities")
			h := NewHandler(tt.service)
			h.listModalities(ctx)

			assertStatusCode(t, recorder, tt.wantStatus)
			if tt.wantData {
				assertHasDataKey(t, recorder)
			}
		})
	}
}

func TestHandler_ListPropertyTypes(t *testing.T) {
	tests := []struct {
		name       string
		service    *mockCatalogsService
		wantStatus int
		wantData   bool
	}{
		{
			name: "returns ok",
			service: &mockCatalogsService{
				listPropertyTypesFunc: func(ctx context.Context) (ListPropertyTypesResult, error) {
					icon := "/icons/apartment.svg"
					return ListPropertyTypesResult{Data: []PropertyType{{PropertyTypeID: 1, Name: "Apartment", Icon: &icon}}}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantData:   true,
		},
		{
			name: "service error returns internal server error",
			service: &mockCatalogsService{
				listPropertyTypesFunc: func(ctx context.Context) (ListPropertyTypesResult, error) {
					return ListPropertyTypesResult{}, errors.New("boom")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, recorder := newTestContext(http.MethodGet, "/api/v1/catalogs/property-types")
			h := NewHandler(tt.service)
			h.listPropertyTypes(ctx)

			assertStatusCode(t, recorder, tt.wantStatus)
			if tt.wantData {
				assertHasDataKey(t, recorder)
			}
		})
	}
}

func TestHandler_ListOrientations(t *testing.T) {
	tests := []struct {
		name       string
		service    *mockCatalogsService
		wantStatus int
		wantData   bool
	}{
		{
			name: "returns ok",
			service: &mockCatalogsService{
				listOrientationsFunc: func(ctx context.Context) (ListOrientationsResult, error) {
					return ListOrientationsResult{Data: []Orientation{{OrientationID: 1, Name: "North"}}}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantData:   true,
		},
		{
			name: "service error returns internal server error",
			service: &mockCatalogsService{
				listOrientationsFunc: func(ctx context.Context) (ListOrientationsResult, error) {
					return ListOrientationsResult{}, errors.New("boom")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, recorder := newTestContext(http.MethodGet, "/api/v1/catalogs/orientations")
			h := NewHandler(tt.service)
			h.listOrientations(ctx)

			assertStatusCode(t, recorder, tt.wantStatus)
			if tt.wantData {
				assertHasDataKey(t, recorder)
			}
		})
	}
}

func TestHandler_ListRentPeriods(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		service         *mockCatalogsService
		wantStatus      int
		wantData        bool
		wantServiceID   int32
		wantServiceCall bool
	}{
		{
			name:       "missing property type id returns bad request",
			query:      "/api/v1/catalogs/rent-periods",
			service:    &mockCatalogsService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non integer returns bad request",
			query:      "/api/v1/catalogs/rent-periods?property_type_id=abc",
			service:    &mockCatalogsService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero returns bad request",
			query:      "/api/v1/catalogs/rent-periods?property_type_id=0",
			service:    &mockCatalogsService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative returns bad request",
			query:      "/api/v1/catalogs/rent-periods?property_type_id=-5",
			service:    &mockCatalogsService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "whitespace only returns bad request",
			query:      "/api/v1/catalogs/rent-periods?property_type_id=%20%20",
			service:    &mockCatalogsService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "valid id returns ok",
			query: "/api/v1/catalogs/rent-periods?property_type_id=7",
			service: &mockCatalogsService{
				listRentPeriodsFunc: func(ctx context.Context, propertyTypeID int32) (ListRentPeriodsResult, error) {
					return ListRentPeriodsResult{Data: []RentPeriod{{PeriodID: 1, Name: "Monthly"}}}, nil
				},
			},
			wantStatus:      http.StatusOK,
			wantData:        true,
			wantServiceID:   7,
			wantServiceCall: true,
		},
		{
			name:  "valid id and service error returns internal server error",
			query: "/api/v1/catalogs/rent-periods?property_type_id=7",
			service: &mockCatalogsService{
				listRentPeriodsFunc: func(ctx context.Context, propertyTypeID int32) (ListRentPeriodsResult, error) {
					return ListRentPeriodsResult{}, errors.New("boom")
				},
			},
			wantStatus:      http.StatusInternalServerError,
			wantServiceID:   7,
			wantServiceCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, recorder := newTestContext(http.MethodGet, tt.query)
			var gotServiceID int32
			service := tt.service
			if tt.wantServiceCall {
				originalFunc := service.listRentPeriodsFunc
				service.listRentPeriodsFunc = func(ctx context.Context, propertyTypeID int32) (ListRentPeriodsResult, error) {
					gotServiceID = propertyTypeID
					return originalFunc(ctx, propertyTypeID)
				}
			}

			h := NewHandler(service)
			h.listRentPeriods(ctx)

			assertStatusCode(t, recorder, tt.wantStatus)
			if tt.wantServiceCall && gotServiceID != tt.wantServiceID {
				t.Fatalf("service called with propertyTypeID %d, want %d", gotServiceID, tt.wantServiceID)
			}
			if tt.wantData {
				assertHasDataKey(t, recorder)
			}
		})
	}
}

func TestResolveRequiredInt(t *testing.T) {
	tests := []struct {
		name      string
		rawValue  string
		field     string
		wantValue int32
		wantErr   bool
	}{
		{name: "empty string", rawValue: "", field: "property_type_id", wantErr: true},
		{name: "whitespace only", rawValue: "   ", field: "property_type_id", wantErr: true},
		{name: "non integer", rawValue: "abc", field: "property_type_id", wantErr: true},
		{name: "zero", rawValue: "0", field: "property_type_id", wantErr: true},
		{name: "negative", rawValue: "-5", field: "property_type_id", wantErr: true},
		{name: "overflow int64", rawValue: "9223372036854775808", field: "property_type_id", wantErr: true},
		{name: "valid positive one", rawValue: "1", field: "property_type_id", wantValue: 1},
		{name: "valid positive large", rawValue: "2147483647", field: "property_type_id", wantValue: 2147483647},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := resolveRequiredInt(tt.rawValue, tt.field)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if value != tt.wantValue {
				t.Fatalf("unexpected value: got %d, want %d", value, tt.wantValue)
			}
		})
	}
}
