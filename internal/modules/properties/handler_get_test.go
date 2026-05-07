package properties

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandler_ListProperties(t *testing.T) {
	tests := []struct {
		name              string
		query             url.Values
		serviceErr        error
		wantStatus        int
		wantServiceCalled bool
		wantCheckInput    bool
		wantInput         ListPropertiesInput
	}{
		{
			name:              "returns ok with default values when query params are empty",
			query:             url.Values{},
			wantStatus:        http.StatusOK,
			wantServiceCalled: true,
			wantCheckInput:    true,
			wantInput: ListPropertiesInput{
				Page:      1,
				PageSize:  20,
				StatusIDs: []int32{},
				Order:     "desc",
			},
		},
		{
			name:       "returns bad request when page is invalid",
			query:      url.Values{"page": []string{"abc"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when page size is invalid",
			query:      url.Values{"page_size": []string{"abc"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when page size exceeds limit",
			query:      url.Values{"page_size": []string{"101"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when page is zero",
			query:      url.Values{"page": []string{"0"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when status id is invalid",
			query:      url.Values{"status_id": []string{"abc"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when status id is zero",
			query:      url.Values{"status_id": []string{"0"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when property type id is invalid",
			query:      url.Values{"property_type_id": []string{"abc"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when property type id is zero",
			query:      url.Values{"property_type_id": []string{"0"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when sort is invalid",
			query:      url.Values{"sort": []string{"invalid"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when order is invalid",
			query:      url.Values{"order": []string{"invalid"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:              "returns internal server error when service fails",
			query:             url.Values{"page": []string{"1"}, "page_size": []string{"20"}},
			serviceErr:        errors.New("db"),
			wantStatus:        http.StatusInternalServerError,
			wantServiceCalled: true,
		},
		{
			name: "returns ok when request is valid",
			query: url.Values{
				"page":             []string{"2"},
				"page_size":        []string{"50"},
				"q":                []string{"search"},
				"status_id":        []string{"1", "2"},
				"property_type_id": []string{"3"},
				"modality_id":      []string{"4"},
				"country_id":       []string{"5"},
				"state_id":         []string{"6"},
				"city_id":          []string{"7"},
				"sort":             []string{"price"},
				"order":            []string{"desc"},
			},
			wantStatus:        http.StatusOK,
			wantServiceCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var calledInput ListPropertiesInput

			customMock := &mockServiceForClauses{
				listPropertiesFunc: func(ctx context.Context, input ListPropertiesInput) (ListPropertiesResult, error) {
					called = true
					calledInput = input
					if tt.serviceErr != nil {
						return ListPropertiesResult{}, tt.serviceErr
					}
					return ListPropertiesResult{}, nil
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/properties?"+tt.query.Encode(), nil)
			ctx.Request = req

			handler := NewHandler(customMock)
			handler.listProperties(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
			}

			if called != tt.wantServiceCalled {
				t.Fatalf("service called: got %v, want %v", called, tt.wantServiceCalled)
			}

			if tt.wantCheckInput {
				if !reflect.DeepEqual(calledInput, tt.wantInput) {
					t.Fatalf("input mismatch: got %#v want %#v", calledInput, tt.wantInput)
				}
			}
		})
	}
}

func TestHandler_GetProperty(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name                  string
		uuid                  string
		full                  string
		getPropertyResult     GetPropertyResult
		getFullPropertyResult GetPropertyFullResult
		serviceErr            error
		wantStatus            int
		wantGetPropertyCalled bool
		wantGetFullCalled     bool
	}{
		{
			name:       "returns bad request when uuid is empty",
			uuid:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns bad request when full query param is invalid",
			uuid:       validUUID,
			full:       "maybe",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:                  "returns ok with default full value when query param is absent",
			uuid:                  validUUID,
			full:                  "",
			getPropertyResult:     GetPropertyResult{Data: GetPropertyData{PropertyUUID: validUUID}},
			wantStatus:            http.StatusOK,
			wantGetPropertyCalled: true,
			wantGetFullCalled:     false,
		},
		{
			name:                  "returns ok when full is false",
			uuid:                  validUUID,
			full:                  "false",
			getPropertyResult:     GetPropertyResult{Data: GetPropertyData{PropertyUUID: validUUID}},
			wantStatus:            http.StatusOK,
			wantGetPropertyCalled: true,
		},
		{
			name:                  "returns ok when full is true",
			uuid:                  validUUID,
			full:                  "true",
			getFullPropertyResult: GetPropertyFullResult{Data: GetPropertyFullData{PropertyUUID: validUUID}},
			wantStatus:            http.StatusOK,
			wantGetFullCalled:     true,
		},
		{
			name:                  "returns not found when property does not exist",
			uuid:                  validUUID,
			full:                  "false",
			serviceErr:            ErrPropertyNotFound,
			wantStatus:            http.StatusNotFound,
			wantGetPropertyCalled: true,
		},
		{
			name:                  "returns internal server error when service fails",
			uuid:                  validUUID,
			full:                  "false",
			serviceErr:            errors.New("db"),
			wantStatus:            http.StatusInternalServerError,
			wantGetPropertyCalled: true,
		},
		{
			name:              "returns not found when full property does not exist",
			uuid:              validUUID,
			full:              "true",
			serviceErr:        ErrPropertyNotFound,
			wantStatus:        http.StatusNotFound,
			wantGetFullCalled: true,
		},
		{
			name:              "returns internal server error when full property service fails",
			uuid:              validUUID,
			full:              "true",
			serviceErr:        errors.New("db"),
			wantStatus:        http.StatusInternalServerError,
			wantGetFullCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getPropertyCalled := false
			getFullCalled := false

			customMock := &mockServiceForClauses{
				getPropertyFunc: func(ctx context.Context, uuid string) (GetPropertyResult, error) {
					getPropertyCalled = true
					if tt.serviceErr != nil {
						return GetPropertyResult{}, tt.serviceErr
					}
					return tt.getPropertyResult, nil
				},
				getFullPropertyFunc: func(ctx context.Context, uuid string) (GetPropertyFullResult, error) {
					getFullCalled = true
					if tt.serviceErr != nil {
						return GetPropertyFullResult{}, tt.serviceErr
					}
					return tt.getFullPropertyResult, nil
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Params = gin.Params{{Key: "uuid", Value: tt.uuid}}

			query := url.Values{}
			if tt.full != "" {
				query.Set("full", tt.full)
			}
			path := "/api/v1/properties/" + tt.uuid
			if encoded := query.Encode(); encoded != "" {
				path += "?" + encoded
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			ctx.Request = req

			handler := NewHandler(customMock)
			handler.getProperty(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status: got %d, want %d", recorder.Code, tt.wantStatus)
			}

			if getPropertyCalled != tt.wantGetPropertyCalled {
				t.Fatalf("get property called: got %v, want %v", getPropertyCalled, tt.wantGetPropertyCalled)
			}
			if getFullCalled != tt.wantGetFullCalled {
				t.Fatalf("get full property called: got %v, want %v", getFullCalled, tt.wantGetFullCalled)
			}
		})
	}
}
