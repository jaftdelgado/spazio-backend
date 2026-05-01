package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockServicesService struct {
	result        ListServicesResult
	err           error
	popularInput  ListPopularInput
	searchInput   SearchInput
	calledPopular bool
	calledSearch  bool
}

func (m *mockServicesService) ListPopularServices(_ context.Context, input ListPopularInput) (ListServicesResult, error) {
	m.calledPopular = true
	m.popularInput = input
	return m.result, m.err
}

func (m *mockServicesService) SearchServices(_ context.Context, input SearchInput) (ListServicesResult, error) {
	m.calledSearch = true
	m.searchInput = input
	return m.result, m.err
}

func TestListServices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		url               string
		mock              *mockServicesService
		wantStatusCode    int
		wantBodyContains  string
		wantCalledPopular bool
		wantCalledSearch  bool
		wantPopularInput  ListPopularInput
		wantSearchInput   SearchInput
	}{
		{
			name: "lists popular services by default",
			url:  "/api/v1/services",
			mock: &mockServicesService{
				result: ListServicesResult{
					Data: []Service{{ServiceID: 1, Code: "WIFI", Icon: "wifi", CategoryCode: "BASIC"}},
					Meta: ListServicesMeta{Total: 1, Shown: 1},
				},
			},
			wantStatusCode:    http.StatusOK,
			wantBodyContains:  "\"service_id\":1",
			wantCalledPopular: true,
			wantCalledSearch:  false,
			wantPopularInput:  ListPopularInput{Limit: 12},
		},
		{
			name: "searches services when q is provided",
			url:  "/api/v1/services?q=wifi&limit=2",
			mock: &mockServicesService{
				result: ListServicesResult{
					Data: []Service{{ServiceID: 1, Code: "WIFI", Icon: "wifi", CategoryCode: "BASIC"}},
					Meta: ListServicesMeta{Total: 1, Shown: 1},
				},
			},
			wantStatusCode:    http.StatusOK,
			wantBodyContains:  "\"service_id\":1",
			wantCalledPopular: false,
			wantCalledSearch:  true,
			wantSearchInput:   SearchInput{Query: "wifi", Limit: 2},
		},
		{
			name:              "rejects invalid limit",
			url:               "/api/v1/services?limit=foo",
			mock:              &mockServicesService{},
			wantStatusCode:    http.StatusBadRequest,
			wantBodyContains:  "limit must be a valid integer",
			wantCalledPopular: false,
			wantCalledSearch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, tt.url, nil)

			handler := NewHandler(tt.mock)
			handler.listServices(ctx)

			if recorder.Code != tt.wantStatusCode {
				t.Fatalf("status code = %d, want %d", recorder.Code, tt.wantStatusCode)
			}

			if tt.wantCalledPopular != tt.mock.calledPopular {
				t.Fatalf("calledPopular = %v, want %v", tt.mock.calledPopular, tt.wantCalledPopular)
			}

			if tt.wantCalledSearch != tt.mock.calledSearch {
				t.Fatalf("calledSearch = %v, want %v", tt.mock.calledSearch, tt.wantCalledSearch)
			}

			if tt.wantCalledPopular && tt.mock.popularInput != tt.wantPopularInput {
				t.Fatalf("popularInput = %#v, want %#v", tt.mock.popularInput, tt.wantPopularInput)
			}

			if tt.wantCalledSearch && tt.mock.searchInput != tt.wantSearchInput {
				t.Fatalf("searchInput = %#v, want %#v", tt.mock.searchInput, tt.wantSearchInput)
			}

			if tt.wantBodyContains != "" && !strings.Contains(recorder.Body.String(), tt.wantBodyContains) {
				t.Fatalf("body %q does not contain %q", recorder.Body.String(), tt.wantBodyContains)
			}
		})
	}
}

func TestValidateListServicesRequest(t *testing.T) {
	tests := []struct {
		name    string
		limit   int
		wantErr string
	}{
		{name: "valid", limit: 1},
		{name: "invalid", limit: 0, wantErr: "limit must be greater than 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateListServicesRequest(tt.limit)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("validateListServicesRequest() error = %v, want nil", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("validateListServicesRequest() error = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("validateListServicesRequest() error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}
