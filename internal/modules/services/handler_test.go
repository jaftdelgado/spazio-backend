package services

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

type mockServicesService struct {
	listPopularServicesFunc func(ctx context.Context, input ListPopularInput) (ListServicesResult, error)
	searchServicesFunc      func(ctx context.Context, input SearchInput) (ListServicesResult, error)

	calledListPopular bool
	calledSearch      bool
}

func (m *mockServicesService) ListPopularServices(ctx context.Context, input ListPopularInput) (ListServicesResult, error) {
	m.calledListPopular = true
	if m.listPopularServicesFunc != nil {
		return m.listPopularServicesFunc(ctx, input)
	}
	return ListServicesResult{}, nil
}

func (m *mockServicesService) SearchServices(ctx context.Context, input SearchInput) (ListServicesResult, error) {
	m.calledSearch = true
	if m.searchServicesFunc != nil {
		return m.searchServicesFunc(ctx, input)
	}
	return ListServicesResult{}, nil
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

func TestHandler_ListServices_LimitValidation(t *testing.T) {
	cases := []struct {
		name            string
		query           string
		wantStatus      int
		wantMsgContains string
		expectError     bool
	}{
		{"no limit no q", "/api/v1/services", http.StatusOK, "", false},
		{"no limit with q", "/api/v1/services?q=wifi", http.StatusOK, "", false},
		{"limit abc", "/api/v1/services?limit=abc", http.StatusBadRequest, "must be a valid integer", true},
		{"limit 0", "/api/v1/services?limit=0", http.StatusBadRequest, "page_size must be greater than 0", true},
		{"limit -1", "/api/v1/services?limit=-1", http.StatusBadRequest, "page_size must be greater than 0", true},
		{"limit whitespace", "/api/v1/services?limit=%20%20", http.StatusOK, "", false},
		{"limit 5 explicit", "/api/v1/services?limit=5", http.StatusOK, "", false},
		{"page invalid", "/api/v1/services?page=abc", http.StatusBadRequest, "page must be a valid integer", true},
		{"page_size invalid", "/api/v1/services?page_size=abc", http.StatusBadRequest, "page_size must be a valid integer", true},
		{"page zero", "/api/v1/services?page=0", http.StatusBadRequest, "page must be greater than 0", true},
		{"page_size too large", "/api/v1/services?page_size=51", http.StatusBadRequest, "page_size must be less than or equal to 50", true},
		{"category invalid", "/api/v1/services?category_id=abc", http.StatusBadRequest, "category_id must be a valid integer", true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, rec := newTestContext(http.MethodGet, tt.query)
			svc := &mockServicesService{
				listPopularServicesFunc: func(ctx context.Context, input ListPopularInput) (ListServicesResult, error) {
					return ListServicesResult{Data: []Service{}, Meta: ListServicesMeta{Total: 0, Shown: 0}}, nil
				},
				searchServicesFunc: func(ctx context.Context, input SearchInput) (ListServicesResult, error) {
					return ListServicesResult{Data: []Service{}, Meta: ListServicesMeta{Total: 0, Shown: 0}}, nil
				},
			}
			h := NewHandler(svc)
			h.listServices(ctx)

			assertStatus(t, rec, tt.wantStatus)
			if tt.expectError {
				assertHasKeys(t, rec, "error")
				if !strings.Contains(rec.Body.String(), tt.wantMsgContains) {
					t.Fatalf("expected error to contain %q, got %s", tt.wantMsgContains, rec.Body.String())
				}
			} else {
				assertHasKeys(t, rec, "data", "meta")
			}
		})
	}
}

func TestHandler_ListServices_RoutingToServiceMethods(t *testing.T) {
	// no q -> ListPopularServices called
	ctx, rec := newTestContext(http.MethodGet, "/api/v1/services")
	svc := &mockServicesService{
		listPopularServicesFunc: func(ctx context.Context, input ListPopularInput) (ListServicesResult, error) {
			if input.Page != 1 || input.PageSize != 12 || input.CategoryID != 0 {
				t.Fatalf("unexpected input: %#v", input)
			}
			return ListServicesResult{Data: []Service{{ServiceID: 1}}, Meta: ListServicesMeta{Total: 1, Shown: 1}}, nil
		},
	}
	h := NewHandler(svc)
	h.listServices(ctx)
	assertStatus(t, rec, http.StatusOK)
	if !svc.calledListPopular || svc.calledSearch {
		t.Fatalf("no q: expected ListPopularServices called and SearchServices not called")
	}

	// q="wifi" -> SearchServices called
	ctx2, rec2 := newTestContext(http.MethodGet, "/api/v1/services?q=wifi")
	svc2 := &mockServicesService{
		searchServicesFunc: func(ctx context.Context, input SearchInput) (ListServicesResult, error) {
			if input.Query != "wifi" || input.Page != 1 || input.PageSize != 10 || input.CategoryID != 0 {
				t.Fatalf("unexpected input: %#v", input)
			}
			return ListServicesResult{Data: []Service{{ServiceID: 2}}, Meta: ListServicesMeta{Total: 1, Shown: 1, Query: &input.Query}}, nil
		},
	}
	h2 := NewHandler(svc2)
	h2.listServices(ctx2)
	assertStatus(t, rec2, http.StatusOK)
	if !svc2.calledSearch || svc2.calledListPopular {
		t.Fatalf("q=wifi: expected SearchServices called and ListPopularServices not called")
	}

	// q="  " whitespace -> trimmed to empty -> ListPopularServices called
	ctx3, rec3 := newTestContext(http.MethodGet, "/api/v1/services?q=%20%20")
	svc3 := &mockServicesService{
		listPopularServicesFunc: func(ctx context.Context, input ListPopularInput) (ListServicesResult, error) {
			return ListServicesResult{Data: []Service{}, Meta: ListServicesMeta{Total: 0, Shown: 0}}, nil
		},
	}
	h3 := NewHandler(svc3)
	h3.listServices(ctx3)
	assertStatus(t, rec3, http.StatusOK)
	if !svc3.calledListPopular {
		t.Fatalf("q=whitespace: expected ListPopularServices called")
	}

	// service ListPopularServices error -> 500
	ctx4, rec4 := newTestContext(http.MethodGet, "/api/v1/services")
	svc4 := &mockServicesService{
		listPopularServicesFunc: func(ctx context.Context, input ListPopularInput) (ListServicesResult, error) {
			return ListServicesResult{}, errors.New("boom")
		},
	}
	h4 := NewHandler(svc4)
	h4.listServices(ctx4)
	assertStatus(t, rec4, http.StatusInternalServerError)
	assertHasKeys(t, rec4, "error")

	// service SearchServices error -> 500
	ctx5, rec5 := newTestContext(http.MethodGet, "/api/v1/services?q=wifi")
	svc5 := &mockServicesService{
		searchServicesFunc: func(ctx context.Context, input SearchInput) (ListServicesResult, error) {
			return ListServicesResult{}, errors.New("boom")
		},
	}
	h5 := NewHandler(svc5)
	h5.listServices(ctx5)
	assertStatus(t, rec5, http.StatusInternalServerError)
	assertHasKeys(t, rec5, "error")
}

func TestHandler_ListServices_DefaultLimits(t *testing.T) {
	// no limit, no q -> uses defaultPopularLimit=12
	ctx, rec := newTestContext(http.MethodGet, "/api/v1/services")
	svc := &mockServicesService{
		listPopularServicesFunc: func(ctx context.Context, input ListPopularInput) (ListServicesResult, error) {
			if input.PageSize != 12 || input.Page != 1 {
				t.Fatalf("expected page=1 pageSize=12, got %#v", input)
			}
			return ListServicesResult{Data: []Service{}, Meta: ListServicesMeta{Total: 0, Shown: 0}}, nil
		},
	}
	h := NewHandler(svc)
	h.listServices(ctx)
	assertStatus(t, rec, http.StatusOK)

	// no limit, q="wifi" -> uses defaultSearchLimit=10
	ctx2, rec2 := newTestContext(http.MethodGet, "/api/v1/services?q=wifi")
	svc2 := &mockServicesService{
		searchServicesFunc: func(ctx context.Context, input SearchInput) (ListServicesResult, error) {
			if input.PageSize != 10 || input.Page != 1 {
				t.Fatalf("expected page=1 pageSize=10, got %#v", input)
			}
			return ListServicesResult{Data: []Service{}, Meta: ListServicesMeta{Total: 0, Shown: 0}}, nil
		},
	}
	h2 := NewHandler(svc2)
	h2.listServices(ctx2)
	assertStatus(t, rec2, http.StatusOK)
}

func TestHandler_ListServices_PaginationAndCategoryForwarding(t *testing.T) {
	ctx, rec := newTestContext(http.MethodGet, "/api/v1/services?q=wifi&category_id=3&page=2&page_size=7")
	svc := &mockServicesService{
		searchServicesFunc: func(ctx context.Context, input SearchInput) (ListServicesResult, error) {
			if input.Query != "wifi" || input.CategoryID != 3 || input.Page != 2 || input.PageSize != 7 {
				t.Fatalf("unexpected input: %#v", input)
			}
			return ListServicesResult{Data: []Service{}, Meta: ListServicesMeta{Total: 0, Shown: 0}}, nil
		},
	}

	NewHandler(svc).listServices(ctx)
	assertStatus(t, rec, http.StatusOK)
	assertHasKeys(t, rec, "data", "meta")
}

func TestResolveDefaultLimit(t *testing.T) {
	if got := resolveDefaultLimit(""); got != 12 {
		t.Fatalf("resolveDefaultLimit(\"\") = %d, want 12", got)
	}
	if got := resolveDefaultLimit("  "); got != 10 {
		t.Fatalf("resolveDefaultLimit(\"  \") = %d, want 10", got)
	}
	if got := resolveDefaultLimit("wifi"); got != 10 {
		t.Fatalf("resolveDefaultLimit(\"wifi\") = %d, want 10", got)
	}
}

func TestResolveLimit(t *testing.T) {
	if got, err := resolveLimit(""); err != nil || got != 0 {
		t.Fatalf("resolveLimit(\"\") = %d err %v, want 0 nil", got, err)
	}

	if got, err := resolveLimit("  "); err != nil || got != 0 {
		t.Fatalf("resolveLimit(\"  \") = %d err %v, want 0 nil", got, err)
	}

	if _, err := resolveLimit("abc"); err == nil {
		t.Fatal("resolveLimit(\"abc\") expected error, got nil")
	}

	if got, err := resolveLimit("5"); err != nil || got != 5 {
		t.Fatalf("resolveLimit(\"5\") = %d err %v, want 5 nil", got, err)
	}
}

func TestResolvePagination(t *testing.T) {
	tests := []struct {
		name         string
		page         string
		pageSize     string
		limit        string
		fallback     int
		wantPage     int
		wantPageSize int
		wantErr      bool
	}{
		{name: "defaults without q", fallback: 12, wantPage: 1, wantPageSize: 12},
		{name: "legacy limit", limit: "5", fallback: 12, wantPage: 1, wantPageSize: 5},
		{name: "explicit page and page_size", page: "2", pageSize: "7", limit: "5", fallback: 12, wantPage: 2, wantPageSize: 7},
		{name: "page only uses fallback", page: "3", fallback: 10, wantPage: 3, wantPageSize: 10},
		{name: "invalid page", page: "abc", fallback: 12, wantErr: true},
		{name: "invalid page_size", pageSize: "abc", fallback: 12, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPage, gotPageSize, err := resolvePagination(tt.page, tt.pageSize, tt.limit, tt.fallback)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotPage != tt.wantPage || gotPageSize != tt.wantPageSize {
				t.Fatalf("got page=%d pageSize=%d want page=%d pageSize=%d", gotPage, gotPageSize, tt.wantPage, tt.wantPageSize)
			}
		})
	}
}

func TestResolveCategoryID(t *testing.T) {
	if got, err := resolveCategoryID(""); err != nil || got != 0 {
		t.Fatalf("resolveCategoryID(\"\") = %d err %v, want 0 nil", got, err)
	}
	if got, err := resolveCategoryID("3"); err != nil || got != 3 {
		t.Fatalf("resolveCategoryID(\"3\") = %d err %v, want 3 nil", got, err)
	}
	if _, err := resolveCategoryID("abc"); err == nil {
		t.Fatal("expected error for invalid category_id")
	}
}

func TestValidateListServicesRequest(t *testing.T) {
	if err := validateListServicesRequest(0, 10, 0); err == nil {
		t.Fatal("expected error when page=0")
	}
	if err := validateListServicesRequest(1, 0, 0); err == nil {
		t.Fatal("expected error when page_size=0")
	}
	if err := validateListServicesRequest(1, 51, 0); err == nil {
		t.Fatal("expected error when page_size=51")
	}
	if err := validateListServicesRequest(1, 10, -1); err == nil {
		t.Fatal("expected error when category_id=-1")
	}
	if err := validateListServicesRequest(1, 10, 2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
