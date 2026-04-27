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
	result ListServicesResult
	err    error
	input  ListServicesInput
	called bool
}

func (m *mockServicesService) ListServices(_ context.Context, input ListServicesInput) (ListServicesResult, error) {
	m.called = true
	m.input = input
	return m.result, m.err
}

func TestListServices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		url              string
		mock             *mockServicesService
		wantStatusCode   int
		wantBodyContains string
		wantCalled       bool
		wantInput        ListServicesInput
	}{
		{
			name: "lists popular services by default",
			url:  "/services",
			mock: &mockServicesService{
				result: ListServicesResult{
					Data: []Service{{ServiceID: 1, Code: "WIFI", Icon: "wifi", CategoryCode: "BASIC"}},
					Meta: ListServicesMeta{Total: 1, Shown: 1},
				},
			},
			wantStatusCode:   http.StatusOK,
			wantBodyContains: "\"service_id\":1",
			wantCalled:       true,
			wantInput:        ListServicesInput{Limit: 12},
		},
		{
			name: "searches services when q is provided",
			url:  "/services?q=wifi&limit=2",
			mock: &mockServicesService{
				result: ListServicesResult{
					Data: []Service{{ServiceID: 1, Code: "WIFI", Icon: "wifi", CategoryCode: "BASIC"}},
					Meta: ListServicesMeta{Total: 1, Shown: 1},
				},
			},
			wantStatusCode:   http.StatusOK,
			wantBodyContains: "\"service_id\":1",
			wantCalled:       true,
			wantInput:        ListServicesInput{Query: "wifi", Limit: 2},
		},
		{
			name:             "rejects invalid limit",
			url:              "/services?limit=foo",
			mock:             &mockServicesService{},
			wantStatusCode:   http.StatusBadRequest,
			wantBodyContains: "limit must be a valid integer",
			wantCalled:       false,
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

			if tt.wantCalled != tt.mock.called {
				t.Fatalf("called = %v, want %v", tt.mock.called, tt.wantCalled)
			}

			if tt.wantCalled && tt.mock.input != tt.wantInput {
				t.Fatalf("input = %#v, want %#v", tt.mock.input, tt.wantInput)
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
