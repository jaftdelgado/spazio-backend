package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type mockServicesRepository struct {
	listPopularServicesFunc func(ctx context.Context, limit int32) ([]Service, int64, error)
	searchServicesFunc      func(ctx context.Context, query string, limit int32) ([]Service, int64, error)
}

func (m *mockServicesRepository) ListPopularServices(ctx context.Context, limit int32) ([]Service, int64, error) {
	if m.listPopularServicesFunc != nil {
		return m.listPopularServicesFunc(ctx, limit)
	}
	return nil, 0, nil
}

func (m *mockServicesRepository) SearchServices(ctx context.Context, query string, limit int32) ([]Service, int64, error) {
	if m.searchServicesFunc != nil {
		return m.searchServicesFunc(ctx, query, limit)
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
		wantLimit int32
	}{
		{
			name:  "returns data with correct shown and total",
			input: ListPopularInput{Limit: 12},
			repo: &mockServicesRepository{
				listPopularServicesFunc: func(ctx context.Context, limit int32) ([]Service, int64, error) {
					return []Service{{ServiceID: 1, Code: "wifi"}, {ServiceID: 2, Code: "pool"}}, 50, nil
				},
			},
			wantData:  []Service{{ServiceID: 1, Code: "wifi"}, {ServiceID: 2, Code: "pool"}},
			wantMeta:  ListServicesMeta{Total: 50, Shown: 2, Query: nil},
			wantLimit: 12,
		},
		{
			name:  "empty result returns empty slice and zero shown",
			input: ListPopularInput{Limit: 12},
			repo: &mockServicesRepository{
				listPopularServicesFunc: func(ctx context.Context, limit int32) ([]Service, int64, error) {
					return []Service{}, 0, nil
				},
			},
			wantData:  []Service{},
			wantMeta:  ListServicesMeta{Total: 0, Shown: 0, Query: nil},
			wantLimit: 12,
		},
		{
			name:  "repository error is wrapped",
			input: ListPopularInput{Limit: 12},
			repo: &mockServicesRepository{
				listPopularServicesFunc: func(ctx context.Context, limit int32) ([]Service, int64, error) {
					return nil, 0, repositoryErr
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calledLimit int32 = -1
			// wrap to capture limit
			repo := tt.repo
			orig := repo.listPopularServicesFunc
			repo.listPopularServicesFunc = func(ctx context.Context, limit int32) ([]Service, int64, error) {
				calledLimit = limit
				return orig(ctx, limit)
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
			if tt.wantLimit >= 0 && calledLimit != tt.wantLimit {
				t.Fatalf("limit mismatch: got %d want %d", calledLimit, tt.wantLimit)
			}
		})
	}
}

func TestService_SearchServices(t *testing.T) {
	repositoryErr := errors.New("repo fail")

	tests := []struct {
		name     string
		input    SearchInput
		repo     *mockServicesRepository
		wantData []Service
		wantMeta ListServicesMeta
		wantErr  bool
	}{
		{
			name:  "returns data with query in meta",
			input: SearchInput{Query: "wifi", Limit: 10},
			repo: &mockServicesRepository{
				searchServicesFunc: func(ctx context.Context, query string, limit int32) ([]Service, int64, error) {
					return []Service{{ServiceID: 1, Code: "wifi"}}, 1, nil
				},
			},
			wantData: []Service{{ServiceID: 1, Code: "wifi"}},
			wantMeta: ListServicesMeta{Total: 1, Shown: 1, Query: func() *string { s := "wifi"; return &s }()},
		},
		{
			name:  "empty search returns empty slice with query",
			input: SearchInput{Query: "nonexistent", Limit: 10},
			repo: &mockServicesRepository{
				searchServicesFunc: func(ctx context.Context, query string, limit int32) ([]Service, int64, error) {
					return []Service{}, 0, nil
				},
			},
			wantData: []Service{},
			wantMeta: ListServicesMeta{Total: 0, Shown: 0, Query: func() *string { s := "nonexistent"; return &s }()},
		},
		{
			name:  "repository error is wrapped",
			input: SearchInput{Query: "fail", Limit: 10},
			repo: &mockServicesRepository{
				searchServicesFunc: func(ctx context.Context, query string, limit int32) ([]Service, int64, error) {
					return nil, 0, repositoryErr
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)
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
		})
	}
}
