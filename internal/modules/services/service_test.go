package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type mockServicesRepository struct {
	listPopularServicesFunc func(ctx context.Context, input ListPopularInput) ([]Service, int64, error)
	searchServicesFunc      func(ctx context.Context, input SearchInput) ([]Service, int64, error)
}

func (m *mockServicesRepository) ListPopularServices(ctx context.Context, input ListPopularInput) ([]Service, int64, error) {
	if m.listPopularServicesFunc != nil {
		return m.listPopularServicesFunc(ctx, input)
	}
	return nil, 0, nil
}

func (m *mockServicesRepository) SearchServices(ctx context.Context, input SearchInput) ([]Service, int64, error) {
	if m.searchServicesFunc != nil {
		return m.searchServicesFunc(ctx, input)
	}
	return nil, 0, nil
}

func TestService_ListPopularServices(t *testing.T) {
	repositoryErr := errors.New("repo fail")

	tests := []struct {
		name      string
		input     ListPopularInput
		repo      *mockServicesRepository
		wantData  []Service
		wantMeta  ListServicesMeta
		wantErr   bool
		wantInput ListPopularInput
	}{
		{
			name:  "returns data with correct shown and total",
			input: ListPopularInput{CategoryID: 2, Page: 2, PageSize: 12},
			repo: &mockServicesRepository{
				listPopularServicesFunc: func(ctx context.Context, input ListPopularInput) ([]Service, int64, error) {
					return []Service{{ServiceID: 1, Code: "wifi"}, {ServiceID: 2, Code: "pool"}}, 50, nil
				},
			},
			wantData:  []Service{{ServiceID: 1, Code: "wifi"}, {ServiceID: 2, Code: "pool"}},
			wantMeta:  ListServicesMeta{Total: 50, Shown: 2, Page: 2, PageSize: 12, TotalPages: 5, Query: nil},
			wantInput: ListPopularInput{CategoryID: 2, Page: 2, PageSize: 12},
		},
		{
			name:  "empty result returns empty slice and zero shown",
			input: ListPopularInput{Page: 1, PageSize: 12},
			repo: &mockServicesRepository{
				listPopularServicesFunc: func(ctx context.Context, input ListPopularInput) ([]Service, int64, error) {
					return []Service{}, 0, nil
				},
			},
			wantData:  []Service{},
			wantMeta:  ListServicesMeta{Total: 0, Shown: 0, Page: 1, PageSize: 12, TotalPages: 0, Query: nil},
			wantInput: ListPopularInput{Page: 1, PageSize: 12},
		},
		{
			name:  "repository error is wrapped",
			input: ListPopularInput{Page: 1, PageSize: 12},
			repo: &mockServicesRepository{
				listPopularServicesFunc: func(ctx context.Context, input ListPopularInput) ([]Service, int64, error) {
					return nil, 0, repositoryErr
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calledInput ListPopularInput
			// wrap to capture limit
			repo := tt.repo
			orig := repo.listPopularServicesFunc
			repo.listPopularServicesFunc = func(ctx context.Context, input ListPopularInput) ([]Service, int64, error) {
				calledInput = input
				return orig(ctx, input)
			}

			svc := NewService(repo)
			got, err := svc.ListPopularServices(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Data == nil {
				t.Fatal("expected non-nil data slice")
			}
			if !reflect.DeepEqual(got.Data, tt.wantData) {
				t.Fatalf("data mismatch: got %#v want %#v", got.Data, tt.wantData)
			}
			if !reflect.DeepEqual(got.Meta, tt.wantMeta) {
				t.Fatalf("meta mismatch: got %#v want %#v", got.Meta, tt.wantMeta)
			}
			if !reflect.DeepEqual(calledInput, tt.wantInput) {
				t.Fatalf("input mismatch: got %#v want %#v", calledInput, tt.wantInput)
			}
		})
	}
}

func TestService_SearchServices(t *testing.T) {
	repositoryErr := errors.New("repo fail")

	tests := []struct {
		name      string
		input     SearchInput
		repo      *mockServicesRepository
		wantData  []Service
		wantMeta  ListServicesMeta
		wantErr   bool
		wantInput SearchInput
	}{
		{
			name:  "returns data with query in meta",
			input: SearchInput{Query: "wifi", CategoryID: 3, Page: 2, PageSize: 10},
			repo: &mockServicesRepository{
				searchServicesFunc: func(ctx context.Context, input SearchInput) ([]Service, int64, error) {
					return []Service{{ServiceID: 1, Code: "wifi"}}, 1, nil
				},
			},
			wantData:  []Service{{ServiceID: 1, Code: "wifi"}},
			wantMeta:  ListServicesMeta{Total: 1, Shown: 1, Page: 2, PageSize: 10, TotalPages: 1, Query: func() *string { s := "wifi"; return &s }()},
			wantInput: SearchInput{Query: "wifi", CategoryID: 3, Page: 2, PageSize: 10},
		},
		{
			name:  "empty search returns empty slice with query",
			input: SearchInput{Query: "nonexistent", Page: 1, PageSize: 10},
			repo: &mockServicesRepository{
				searchServicesFunc: func(ctx context.Context, input SearchInput) ([]Service, int64, error) {
					return []Service{}, 0, nil
				},
			},
			wantData:  []Service{},
			wantMeta:  ListServicesMeta{Total: 0, Shown: 0, Page: 1, PageSize: 10, TotalPages: 0, Query: func() *string { s := "nonexistent"; return &s }()},
			wantInput: SearchInput{Query: "nonexistent", Page: 1, PageSize: 10},
		},
		{
			name:  "repository error is wrapped",
			input: SearchInput{Query: "fail", Page: 1, PageSize: 10},
			repo: &mockServicesRepository{
				searchServicesFunc: func(ctx context.Context, input SearchInput) ([]Service, int64, error) {
					return nil, 0, repositoryErr
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calledInput SearchInput
			repo := tt.repo
			orig := repo.searchServicesFunc
			repo.searchServicesFunc = func(ctx context.Context, input SearchInput) ([]Service, int64, error) {
				calledInput = input
				return orig(ctx, input)
			}

			svc := NewService(repo)
			got, err := svc.SearchServices(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Data == nil {
				t.Fatal("expected non-nil data slice")
			}
			if !reflect.DeepEqual(got.Data, tt.wantData) {
				t.Fatalf("data mismatch: got %#v want %#v", got.Data, tt.wantData)
			}
			// Compare Query pointers
			if tt.wantMeta.Query == nil {
				if got.Meta.Query != nil {
					t.Fatalf("expected nil query, got %v", *got.Meta.Query)
				}
			} else {
				if got.Meta.Query == nil || *got.Meta.Query != *tt.wantMeta.Query {
					t.Fatalf("query mismatch: got %v want %v", got.Meta.Query, tt.wantMeta.Query)
				}
			}
			if got.Meta.Total != tt.wantMeta.Total || got.Meta.Shown != tt.wantMeta.Shown {
				t.Fatalf("meta mismatch: got Total=%d Shown=%d, want Total=%d Shown=%d",
					got.Meta.Total, got.Meta.Shown, tt.wantMeta.Total, tt.wantMeta.Shown)
			}
			if got.Meta.Page != tt.wantMeta.Page || got.Meta.PageSize != tt.wantMeta.PageSize || got.Meta.TotalPages != tt.wantMeta.TotalPages {
				t.Fatalf("pagination meta mismatch: got Page=%d PageSize=%d TotalPages=%d, want Page=%d PageSize=%d TotalPages=%d",
					got.Meta.Page, got.Meta.PageSize, got.Meta.TotalPages, tt.wantMeta.Page, tt.wantMeta.PageSize, tt.wantMeta.TotalPages)
			}
			if !reflect.DeepEqual(calledInput, tt.wantInput) {
				t.Fatalf("input mismatch: got %#v want %#v", calledInput, tt.wantInput)
			}
		})
	}
}

func TestCalculateTotalPages(t *testing.T) {
	if got := calculateTotalPages(0, 10); got != 0 {
		t.Fatalf("calculateTotalPages(0,10) = %d, want 0", got)
	}
	if got := calculateTotalPages(1, 10); got != 1 {
		t.Fatalf("calculateTotalPages(1,10) = %d, want 1", got)
	}
	if got := calculateTotalPages(11, 10); got != 2 {
		t.Fatalf("calculateTotalPages(11,10) = %d, want 2", got)
	}
}
