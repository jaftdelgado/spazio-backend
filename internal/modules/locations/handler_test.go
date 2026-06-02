package locations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockLocationsService struct {
	listCountriesFunc func(ctx context.Context) (ListCountriesResult, error)
	listStatesFunc    func(ctx context.Context, input ListStatesInput) (ListStatesResult, error)
	listCitiesFunc    func(ctx context.Context, input ListCitiesInput) (ListCitiesResult, error)
}

func (m *mockLocationsService) ListCountries(ctx context.Context) (ListCountriesResult, error) {
	if m.listCountriesFunc != nil {
		return m.listCountriesFunc(ctx)
	}
	return ListCountriesResult{}, nil
}

func (m *mockLocationsService) ListStates(ctx context.Context, input ListStatesInput) (ListStatesResult, error) {
	if m.listStatesFunc != nil {
		return m.listStatesFunc(ctx, input)
	}
	return ListStatesResult{}, nil
}

func (m *mockLocationsService) ListCities(ctx context.Context, input ListCitiesInput) (ListCitiesResult, error) {
	if m.listCitiesFunc != nil {
		return m.listCitiesFunc(ctx, input)
	}
	return ListCitiesResult{}, nil
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func newTestContext(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(method, target, nil)
	ginCtx.Request = req
	return ginCtx, rec
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status: got %d want %d; body=%s", rec.Code, want, rec.Body.String())
	}
}

func assertHasKeys(t *testing.T, rec *httptest.ResponseRecorder, keys ...string) {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			t.Fatalf("expected key %q in response, body=%s", k, rec.Body.String())
		}
	}
}

func TestHandler_ListCountries(t *testing.T) {
	tests := []struct {
		name       string
		service    *mockLocationsService
		wantStatus int
		wantKeys   []string
	}{
		{
			name: "service success returns 200 with data",
			service: &mockLocationsService{
				listCountriesFunc: func(ctx context.Context) (ListCountriesResult, error) {
					return ListCountriesResult{Data: []Country{{CountryID: 1, Iso2Code: "US", Name: "United States"}}}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantKeys:   []string{"data"},
		},
		{
			name: "service error returns 500 with error",
			service: &mockLocationsService{
				listCountriesFunc: func(ctx context.Context) (ListCountriesResult, error) {
					return ListCountriesResult{}, errors.New("boom")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantKeys:   []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, rec := newTestContext(http.MethodGet, "/api/v1/locations/countries")
			h := NewHandler(tt.service)
			h.listCountries(ctx)

			assertStatus(t, rec, tt.wantStatus)
			assertHasKeys(t, rec, tt.wantKeys...)
		})
	}
}

func TestHandler_ListStates_CountryIDValidation(t *testing.T) {
	cases := []struct {
		name            string
		query           string
		wantStatus      int
		wantMsgContains string
	}{
		{"missing", "/api/v1/locations/states", http.StatusBadRequest, "country_id is required"},
		{"whitespace only", "/api/v1/locations/states?country_id=%20%20", http.StatusBadRequest, "country_id is required"},
		{"non-integer", "/api/v1/locations/states?country_id=abc", http.StatusBadRequest, "must be a valid integer"},
		{"valid", "/api/v1/locations/states?country_id=1", http.StatusOK, ""},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, rec := newTestContext(http.MethodGet, tt.query)
			svc := &mockLocationsService{
				listStatesFunc: func(ctx context.Context, input ListStatesInput) (ListStatesResult, error) {
					return ListStatesResult{Data: []State{}}, nil
				},
			}
			h := NewHandler(svc)
			h.listStates(ctx)

			assertStatus(t, rec, tt.wantStatus)
			if tt.wantStatus == http.StatusOK {
				assertHasKeys(t, rec, "data")
			} else {
				assertHasKeys(t, rec, "error")
				if tt.wantStatus == http.StatusBadRequest && !strings.Contains(rec.Body.String(), tt.wantMsgContains) {
					t.Fatalf("expected error to contain %q, got %s", tt.wantMsgContains, rec.Body.String())
				}
			}
		})
	}
}

func TestHandler_ListStates_SearchForwarding(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantSearch string
	}{
		{
			name:       "without search keeps empty string",
			target:     "/api/v1/locations/states?country_id=1",
			wantSearch: "",
		},
		{
			name:       "with search forwards trimmed value",
			target:     "/api/v1/locations/states?country_id=1&search=%20jal%20",
			wantSearch: "jal",
		},
		{
			name:       "blank search becomes empty string",
			target:     "/api/v1/locations/states?country_id=1&search=%20%20",
			wantSearch: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, rec := newTestContext(http.MethodGet, tt.target)
			svc := &mockLocationsService{
				listStatesFunc: func(ctx context.Context, input ListStatesInput) (ListStatesResult, error) {
					if input.CountryID != 1 {
						t.Fatalf("unexpected country id: %d", input.CountryID)
					}
					if input.Search != tt.wantSearch {
						t.Fatalf("search: got %q want %q", input.Search, tt.wantSearch)
					}
					return ListStatesResult{Data: []State{{StateID: 1, Name: "Jalisco"}}}, nil
				},
			}

			NewHandler(svc).listStates(ctx)

			assertStatus(t, rec, http.StatusOK)
			assertHasKeys(t, rec, "data")
		})
	}
}

func TestHandler_ListStates_ServiceError(t *testing.T) {
	ctx, rec := newTestContext(http.MethodGet, "/api/v1/locations/states?country_id=1")
	svc := &mockLocationsService{
		listStatesFunc: func(ctx context.Context, input ListStatesInput) (ListStatesResult, error) {
			return ListStatesResult{}, errors.New("boom")
		},
	}
	h := NewHandler(svc)
	h.listStates(ctx)

	assertStatus(t, rec, http.StatusInternalServerError)
	assertHasKeys(t, rec, "error")
}

func TestHandler_ListCities_StateIDValidation(t *testing.T) {
	cases := []struct {
		name            string
		query           string
		wantStatus      int
		wantMsgContains string
	}{
		{"missing", "/api/v1/locations/cities", http.StatusBadRequest, "state_id is required"},
		{"non-integer", "/api/v1/locations/cities?state_id=abc", http.StatusBadRequest, "must be a valid integer"},
		{"valid", "/api/v1/locations/cities?state_id=1", http.StatusOK, ""},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, rec := newTestContext(http.MethodGet, tt.query)
			svc := &mockLocationsService{
				listCitiesFunc: func(ctx context.Context, input ListCitiesInput) (ListCitiesResult, error) {
					return ListCitiesResult{Data: []City{}, Meta: ListCitiesMeta{Total: 0, Page: 1, PageSize: 50, TotalPages: 0}}, nil
				},
			}
			h := NewHandler(svc)
			h.listCities(ctx)

			assertStatus(t, rec, tt.wantStatus)
			if tt.wantStatus == http.StatusBadRequest {
				assertHasKeys(t, rec, "error")
				if !strings.Contains(rec.Body.String(), tt.wantMsgContains) {
					t.Fatalf("expected error to contain %q, got %s", tt.wantMsgContains, rec.Body.String())
				}
			}
		})
	}
}

func TestHandler_ListCities_PaginationDefaults(t *testing.T) {
	// no page or page_size -> uses defaults page=1, pageSize=50
	ctx, rec := newTestContext(http.MethodGet, "/api/v1/locations/cities?state_id=1")
	svc := &mockLocationsService{
		listCitiesFunc: func(ctx context.Context, input ListCitiesInput) (ListCitiesResult, error) {
			if input.Page != 1 || input.PageSize != 50 || input.Search != "" {
				t.Fatalf("expected page=1 pageSize=50 search empty, got page=%d pageSize=%d search=%q", input.Page, input.PageSize, input.Search)
			}
			return ListCitiesResult{Data: []City{}, Meta: ListCitiesMeta{Total: 0, Page: 1, PageSize: 50, TotalPages: 0}}, nil
		},
	}
	h := NewHandler(svc)
	h.listCities(ctx)

	assertStatus(t, rec, http.StatusOK)
	assertHasKeys(t, rec, "data", "meta")
}

func TestHandler_ListCities_SearchForwarding(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantInput  ListCitiesInput
		wantStatus int
	}{
		{
			name:       "without search keeps empty string",
			target:     "/api/v1/locations/cities?state_id=5&page=1&page_size=50",
			wantInput:  ListCitiesInput{StateID: 5, Page: 1, PageSize: 50, Search: ""},
			wantStatus: http.StatusOK,
		},
		{
			name:       "search with spaces is trimmed",
			target:     "/api/v1/locations/cities?state_id=5&search=%20san%20&page=2&page_size=10",
			wantInput:  ListCitiesInput{StateID: 5, Page: 2, PageSize: 10, Search: "san"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "blank search becomes empty string",
			target:     "/api/v1/locations/cities?state_id=5&search=%20%20&page=1&page_size=25",
			wantInput:  ListCitiesInput{StateID: 5, Page: 1, PageSize: 25, Search: ""},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, rec := newTestContext(http.MethodGet, tt.target)
			svc := &mockLocationsService{
				listCitiesFunc: func(ctx context.Context, input ListCitiesInput) (ListCitiesResult, error) {
					if input != tt.wantInput {
						t.Fatalf("input: got %#v want %#v", input, tt.wantInput)
					}
					return ListCitiesResult{
						Data: []City{{CityID: 1, Name: "San Pedro"}},
						Meta: ListCitiesMeta{Total: 1, Page: input.Page, PageSize: input.PageSize, TotalPages: 1},
					}, nil
				},
			}

			NewHandler(svc).listCities(ctx)

			assertStatus(t, rec, tt.wantStatus)
			assertHasKeys(t, rec, "data", "meta")
		})
	}
}

func TestHandler_ListCities_PageValidation(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		wantStatus int
		wantMsgKey string
	}{
		{"page 0", "/api/v1/locations/cities?state_id=1&page=0", http.StatusBadRequest, "error"},
		{"page -1", "/api/v1/locations/cities?state_id=1&page=-1", http.StatusBadRequest, "error"},
		{"page non-integer", "/api/v1/locations/cities?state_id=1&page=abc", http.StatusBadRequest, "error"},
		{"page 1 valid", "/api/v1/locations/cities?state_id=1&page=1", http.StatusOK, "data"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, rec := newTestContext(http.MethodGet, tt.query)
			svc := &mockLocationsService{
				listCitiesFunc: func(ctx context.Context, input ListCitiesInput) (ListCitiesResult, error) {
					return ListCitiesResult{Data: []City{}, Meta: ListCitiesMeta{Total: 0, Page: 1, PageSize: 50, TotalPages: 0}}, nil
				},
			}
			h := NewHandler(svc)
			h.listCities(ctx)

			assertStatus(t, rec, tt.wantStatus)
			assertHasKeys(t, rec, tt.wantMsgKey)
		})
	}
}

func TestHandler_ListCities_PageSizeValidation(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		wantStatus int
		wantMsgKey string
	}{
		{"pageSize 0", "/api/v1/locations/cities?state_id=1&page_size=0", http.StatusBadRequest, "error"},
		{"pageSize 101", "/api/v1/locations/cities?state_id=1&page_size=101", http.StatusBadRequest, "error"},
		{"pageSize -1", "/api/v1/locations/cities?state_id=1&page_size=-1", http.StatusBadRequest, "error"},
		{"pageSize non-integer", "/api/v1/locations/cities?state_id=1&page_size=abc", http.StatusBadRequest, "error"},
		{"pageSize 100 boundary", "/api/v1/locations/cities?state_id=1&page_size=100", http.StatusOK, "data"},
		{"pageSize 1 boundary", "/api/v1/locations/cities?state_id=1&page_size=1", http.StatusOK, "data"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, rec := newTestContext(http.MethodGet, tt.query)
			svc := &mockLocationsService{
				listCitiesFunc: func(ctx context.Context, input ListCitiesInput) (ListCitiesResult, error) {
					return ListCitiesResult{Data: []City{}, Meta: ListCitiesMeta{Total: 0, Page: 1, PageSize: 50, TotalPages: 0}}, nil
				},
			}
			h := NewHandler(svc)
			h.listCities(ctx)

			assertStatus(t, rec, tt.wantStatus)
			assertHasKeys(t, rec, tt.wantMsgKey)
		})
	}
}

func TestHandler_ListCities_ServiceError(t *testing.T) {
	ctx, rec := newTestContext(http.MethodGet, "/api/v1/locations/cities?state_id=1")
	svc := &mockLocationsService{
		listCitiesFunc: func(ctx context.Context, input ListCitiesInput) (ListCitiesResult, error) {
			return ListCitiesResult{}, errors.New("boom")
		},
	}
	h := NewHandler(svc)
	h.listCities(ctx)

	assertStatus(t, rec, http.StatusInternalServerError)
	assertHasKeys(t, rec, "error")
}

func TestResolvePage(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{"empty", "", 1, false},
		{"valid 1", "1", 1, false},
		{"valid 5", "5", 5, false},
		{"zero", "0", 0, true},
		{"negative", "-1", 0, true},
		{"non-integer", "abc", 0, true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePage(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "page must be an integer greater than 0") {
					t.Fatalf("unexpected error message: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d want %d", got, tt.want)
			}
		})
	}
}

func TestResolvePageSize(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{"empty", "", 50, false},
		{"valid 1", "1", 1, false},
		{"valid 100", "100", 100, false},
		{"valid 50", "50", 50, false},
		{"zero", "0", 0, true},
		{"over max 101", "101", 0, true},
		{"negative", "-1", 0, true},
		{"non-integer", "abc", 0, true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePageSize(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "page_size must be an integer between 1 and 100") {
					t.Fatalf("unexpected error message: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d want %d", got, tt.want)
			}
		})
	}
}
