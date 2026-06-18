package clauses

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

type mockClausesService struct {
	listClausesFunc   func(ctx context.Context, input ListClausesInput) (ListClausesResult, error)
	searchClausesFunc func(ctx context.Context, input SearchClausesInput) (ListClausesResult, error)

	// helpers to assert calls
	calledList   bool
	calledSearch bool
}

func (m *mockClausesService) ListClauses(ctx context.Context, input ListClausesInput) (ListClausesResult, error) {
	m.calledList = true
	if m.listClausesFunc != nil {
		return m.listClausesFunc(ctx, input)
	}
	return ListClausesResult{}, nil
}

func (m *mockClausesService) SearchClauses(ctx context.Context, input SearchClausesInput) (ListClausesResult, error) {
	m.calledSearch = true
	if m.searchClausesFunc != nil {
		return m.searchClausesFunc(ctx, input)
	}
	return ListClausesResult{}, nil
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

func TestHandler_ListClauses_ModalityValidation(t *testing.T) {
	cases := []struct {
		name            string
		query           string
		wantStatus      int
		wantMsgContains string
	}{
		{"missing", "/api/v1/clauses", http.StatusBadRequest, "modality_id is required"},
		{"non-integer", "/api/v1/clauses?modality_id=abc", http.StatusBadRequest, "must be a valid integer"},
		{"zero", "/api/v1/clauses?modality_id=0", http.StatusBadRequest, "must be greater than 0"},
		{"negative", "/api/v1/clauses?modality_id=-3", http.StatusBadRequest, "must be greater than 0"},
		{"whitespace", "/api/v1/clauses?modality_id=%20%20", http.StatusBadRequest, "modality_id is required"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, rec := newTestContext(http.MethodGet, tt.query)
			svc := &mockClausesService{}
			h := NewHandler(svc)
			h.listClauses(ctx)

			assertStatus(t, rec, tt.wantStatus)
			// error responses contain key "error"
			assertHasKeys(t, rec, "error")
			if !strings.Contains(rec.Body.String(), tt.wantMsgContains) {
				t.Fatalf("expected error message to contain %q, got %s", tt.wantMsgContains, rec.Body.String())
			}
		})
	}
}

func TestHandler_ListClauses_PaginationDefaultsAndValidation(t *testing.T) {
	// default page=1 page_size=20 path
	ctx, rec := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1")
	svc := &mockClausesService{
		listClausesFunc: func(ctx context.Context, input ListClausesInput) (ListClausesResult, error) {
			if input.Page != 1 || input.PageSize != 20 {
				return ListClausesResult{}, nil
			}
			return ListClausesResult{Data: []Clause{}, Meta: ListClausesMeta{Total: 0, Page: 1, PageSize: 20, TotalPages: 0}}, nil
		},
	}
	h := NewHandler(svc)
	h.listClauses(ctx)
	assertStatus(t, rec, http.StatusOK)
	assertHasKeys(t, rec, "data", "meta")

	// invalid page_size > max
	ctx2, rec2 := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1&page_size=201")
	h.listClauses(ctx2)
	assertStatus(t, rec2, http.StatusBadRequest)
	assertHasKeys(t, rec2, "error")

	// page=0
	ctx3, rec3 := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1&page=0")
	h.listClauses(ctx3)
	assertStatus(t, rec3, http.StatusBadRequest)
	assertHasKeys(t, rec3, "error")

	// page_size=0
	ctx4, rec4 := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1&page_size=0")
	h.listClauses(ctx4)
	assertStatus(t, rec4, http.StatusBadRequest)
	assertHasKeys(t, rec4, "error")

	// page non-integer
	ctx5, rec5 := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1&page=abc")
	h.listClauses(ctx5)
	assertStatus(t, rec5, http.StatusBadRequest)
	assertHasKeys(t, rec5, "error")

	// page_size non-integer
	ctx6, rec6 := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1&page_size=abc")
	h.listClauses(ctx6)
	assertStatus(t, rec6, http.StatusBadRequest)
	assertHasKeys(t, rec6, "error")
}

func TestHandler_ListClauses_RoutingToService(t *testing.T) {
	// ListClauses called when no q
	ctx, rec := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1")
	svc := &mockClausesService{
		listClausesFunc: func(ctx context.Context, input ListClausesInput) (ListClausesResult, error) {
			return ListClausesResult{Data: []Clause{{ClauseID: 1}}, Meta: ListClausesMeta{Total: 1, Page: 1, PageSize: 20, TotalPages: 1}}, nil
		},
	}
	h := NewHandler(svc)
	h.listClauses(ctx)
	assertStatus(t, rec, http.StatusOK)
	if !svc.calledList || svc.calledSearch {
		t.Fatalf("expected ListClauses called and SearchClauses not called")
	}

	// SearchClauses called when q non-empty
	ctx2, rec2 := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1&q=pets")
	svc2 := &mockClausesService{
		searchClausesFunc: func(ctx context.Context, input SearchClausesInput) (ListClausesResult, error) {
			return ListClausesResult{Data: []Clause{{ClauseID: 2}}, Meta: ListClausesMeta{Total: 1, Page: 1, PageSize: 20, TotalPages: 1, Query: &input.Query}}, nil
		},
	}
	h2 := NewHandler(svc2)
	h2.listClauses(ctx2)
	assertStatus(t, rec2, http.StatusOK)
	if !svc2.calledSearch || svc2.calledList {
		t.Fatalf("expected SearchClauses called and ListClauses not called")
	}

	// whitespace-only q -> treated as empty -> ListClauses
	ctx3, rec3 := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1&q=%20%20")
	svc3 := &mockClausesService{
		listClausesFunc: func(ctx context.Context, input ListClausesInput) (ListClausesResult, error) {
			return ListClausesResult{Data: []Clause{}, Meta: ListClausesMeta{Total: 0, Page: 1, PageSize: 20, TotalPages: 0}}, nil
		},
	}
	h3 := NewHandler(svc3)
	h3.listClauses(ctx3)
	assertStatus(t, rec3, http.StatusOK)
	if !svc3.calledList {
		t.Fatalf("expected ListClauses called for whitespace q")
	}

	// service ListClauses error -> 500
	ctx4, rec4 := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1")
	svc4 := &mockClausesService{
		listClausesFunc: func(ctx context.Context, input ListClausesInput) (ListClausesResult, error) {
			return ListClausesResult{}, errors.New("boom")
		},
	}
	h4 := NewHandler(svc4)
	h4.listClauses(ctx4)
	assertStatus(t, rec4, http.StatusInternalServerError)
	assertHasKeys(t, rec4, "error")

	// service SearchClauses error -> 500
	ctx5, rec5 := newTestContext(http.MethodGet, "/api/v1/clauses?modality_id=1&q=pets")
	svc5 := &mockClausesService{
		searchClausesFunc: func(ctx context.Context, input SearchClausesInput) (ListClausesResult, error) {
			return ListClausesResult{}, errors.New("boom")
		},
	}
	h5 := NewHandler(svc5)
	h5.listClauses(ctx5)
	assertStatus(t, rec5, http.StatusInternalServerError)
	assertHasKeys(t, rec5, "error")
}

func TestResolveRequiredAndOptionalInts(t *testing.T) {
	// resolveRequiredInt
	if _, err := resolveRequiredInt("", "modality_id"); err == nil {
		t.Fatal("expected error for empty required")
	}
	if _, err := resolveRequiredInt("abc", "modality_id"); err == nil {
		t.Fatal("expected error for non-integer")
	}
	if v, err := resolveRequiredInt("0", "m"); err != nil || v != 0 {
		t.Fatalf("expected 0, got %d err %v", v, err)
	}
	if v, err := resolveRequiredInt("-1", "m"); err != nil || v != -1 {
		t.Fatalf("expected -1, got %d err %v", v, err)
	}
	if v, err := resolveRequiredInt("5", "m"); err != nil || v != 5 {
		t.Fatalf("expected 5, got %d err %v", v, err)
	}

	// resolveOptionalInt
	if v, err := resolveOptionalInt("", 7, "page"); err != nil || v != 7 {
		t.Fatalf("expected fallback 7, got %d err %v", v, err)
	}
	if _, err := resolveOptionalInt("abc", 1, "page"); err == nil {
		t.Fatal("expected error for non-integer optional")
	}
	if v, err := resolveOptionalInt("10", 1, "page"); err != nil || v != 10 {
		t.Fatalf("expected 10, got %d err %v", v, err)
	}
	if v, err := resolveOptionalInt("0", 1, "page"); err != nil || v != 0 {
		t.Fatalf("expected 0, got %d err %v", v, err)
	}
}

func TestValidateListClausesRequest(t *testing.T) {
	if err := validateListClausesRequest(0, 1, 20); err == nil {
		t.Fatal("expected error when modalityID=0")
	}
	if err := validateListClausesRequest(1, 0, 20); err == nil {
		t.Fatal("expected error when page=0")
	}
	if err := validateListClausesRequest(1, 1, 0); err == nil {
		t.Fatal("expected error when pageSize=0")
	}
	if err := validateListClausesRequest(1, 1, 201); err == nil {
		t.Fatal("expected error when pageSize>max")
	}
	if err := validateListClausesRequest(1, 1, 20); err != nil {
		t.Fatalf("expected no error for valid inputs, got %v", err)
	}
}
