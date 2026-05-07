package properties

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestService_ListProperties(t *testing.T) {
	tests := []struct {
		name            string
		repoItems       []PropertyCardData
		repoTotal       int64
		repoErr         error
		input           ListPropertiesInput
		wantErr         bool
		wantData        []PropertyCardData
		wantMeta        ListPropertiesMeta
		wantMetaSet     bool
		wantErrContains string
	}{
		{
			name:      "returns properties with pagination metadata",
			repoItems: []PropertyCardData{{PropertyUUID: "a"}, {PropertyUUID: "b"}},
			repoTotal: 45,
			input:     ListPropertiesInput{Page: 1, PageSize: 20},
			wantErr:   false,
			wantData:  []PropertyCardData{{PropertyUUID: "a"}, {PropertyUUID: "b"}},
			wantMeta: ListPropertiesMeta{
				TotalCount:  45,
				TotalPages:  3,
				CurrentPage: 1,
				PageSize:    20,
				HasNext:     true,
				HasPrev:     false,
			},
			wantMetaSet: true,
		},
		{
			name:        "returns empty properties when no results are found",
			repoItems:   []PropertyCardData{},
			repoTotal:   0,
			input:       ListPropertiesInput{Page: 1, PageSize: 20},
			wantErr:     false,
			wantData:    []PropertyCardData{},
			wantMeta:    ListPropertiesMeta{TotalCount: 0, TotalPages: 0, CurrentPage: 1, PageSize: 20, HasNext: false, HasPrev: false},
			wantMetaSet: true,
		},
		{
			name:        "returns properties with previous and next pages available",
			repoItems:   []PropertyCardData{{PropertyUUID: "c"}},
			repoTotal:   50,
			input:       ListPropertiesInput{Page: 2, PageSize: 20},
			wantErr:     false,
			wantData:    []PropertyCardData{{PropertyUUID: "c"}},
			wantMeta:    ListPropertiesMeta{TotalCount: 50, TotalPages: 3, CurrentPage: 2, PageSize: 20, HasNext: true, HasPrev: true},
			wantMetaSet: true,
		},
		{
			name:        "returns properties with no next page on last page",
			repoItems:   []PropertyCardData{{PropertyUUID: "d"}},
			repoTotal:   20,
			input:       ListPropertiesInput{Page: 1, PageSize: 20},
			wantErr:     false,
			wantData:    []PropertyCardData{{PropertyUUID: "d"}},
			wantMeta:    ListPropertiesMeta{TotalCount: 20, TotalPages: 1, CurrentPage: 1, PageSize: 20, HasNext: false, HasPrev: false},
			wantMetaSet: true,
		},
		{
			name:            "returns error when repository fails",
			repoErr:         errors.New("db"),
			input:           ListPropertiesInput{Page: 1, PageSize: 20},
			wantErr:         true,
			wantErrContains: "list properties:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				listPropertiesFunc: func(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error) {
					if tt.repoErr != nil {
						return nil, 0, tt.repoErr
					}
					return tt.repoItems, tt.repoTotal, nil
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.ListProperties(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error message: got %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Data == nil {
				t.Fatal("expected non-nil data slice")
			}

			if !reflect.DeepEqual(result.Data, tt.wantData) {
				t.Fatalf("data mismatch: got %#v want %#v", result.Data, tt.wantData)
			}

			if tt.wantMetaSet {
				if !reflect.DeepEqual(result.Meta, tt.wantMeta) {
					t.Fatalf("meta mismatch: got %#v want %#v", result.Meta, tt.wantMeta)
				}
			}
		})
	}
}

func TestService_GetProperty(t *testing.T) {
	tests := []struct {
		name            string
		repoResult      GetPropertyResult
		repoErr         error
		wantErr         bool
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:       "returns property when repository returns result",
			repoResult: GetPropertyResult{Data: GetPropertyData{PropertyUUID: "id"}},
			wantErr:    false,
		},
		{
			name:      "returns error when property is not found",
			repoErr:   ErrPropertyNotFound,
			wantErr:   true,
			wantErrIs: ErrPropertyNotFound,
		},
		{
			name:            "returns error when repository fails",
			repoErr:         errors.New("db"),
			wantErr:         true,
			wantErrContains: "get property:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getPropertyFunc: func(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
					if tt.repoErr != nil {
						return GetPropertyResult{}, tt.repoErr
					}
					return tt.repoResult, nil
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.GetProperty(context.Background(), "uuid")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("error type: got %v, want %v", err, tt.wantErrIs)
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error message: got %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.repoResult) {
				t.Fatalf("result mismatch: got %#v want %#v", result, tt.repoResult)
			}
		})
	}
}

func TestService_GetFullProperty(t *testing.T) {
	tests := []struct {
		name            string
		repoResult      GetPropertyFullResult
		repoErr         error
		wantErr         bool
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:       "returns full property when repository returns result",
			repoResult: GetPropertyFullResult{Data: GetPropertyFullData{PropertyUUID: "id"}},
			wantErr:    false,
		},
		{
			name:      "returns error when property is not found",
			repoErr:   ErrPropertyNotFound,
			wantErr:   true,
			wantErrIs: ErrPropertyNotFound,
		},
		{
			name:            "returns error when repository fails",
			repoErr:         errors.New("db"),
			wantErr:         true,
			wantErrContains: "get full property:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getFullPropertyFunc: func(ctx context.Context, propertyUUID string) (GetPropertyFullResult, error) {
					if tt.repoErr != nil {
						return GetPropertyFullResult{}, tt.repoErr
					}
					return tt.repoResult, nil
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.GetFullProperty(context.Background(), "uuid")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("error type: got %v, want %v", err, tt.wantErrIs)
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error message: got %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.repoResult) {
				t.Fatalf("result mismatch: got %#v want %#v", result, tt.repoResult)
			}
		})
	}
}

func TestResolvePropertiesTotalPages(t *testing.T) {
	if got := resolvePropertiesTotalPages(0, 20); got != 0 {
		t.Fatalf("resolvePropertiesTotalPages(0, 20) = %d, want 0", got)
	}
	if got := resolvePropertiesTotalPages(20, 20); got != 1 {
		t.Fatalf("resolvePropertiesTotalPages(20, 20) = %d, want 1", got)
	}
	if got := resolvePropertiesTotalPages(21, 20); got != 2 {
		t.Fatalf("resolvePropertiesTotalPages(21, 20) = %d, want 2", got)
	}
	if got := resolvePropertiesTotalPages(40, 20); got != 2 {
		t.Fatalf("resolvePropertiesTotalPages(40, 20) = %d, want 2", got)
	}
	if got := resolvePropertiesTotalPages(41, 20); got != 3 {
		t.Fatalf("resolvePropertiesTotalPages(41, 20) = %d, want 3", got)
	}
	if got := resolvePropertiesTotalPages(1, 20); got != 1 {
		t.Fatalf("resolvePropertiesTotalPages(1, 20) = %d, want 1", got)
	}
}
