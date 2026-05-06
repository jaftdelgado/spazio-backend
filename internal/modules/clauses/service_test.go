package clauses

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type mockClausesRepository struct {
	listClausesFunc   func(ctx context.Context, modalityID, pageSize, pageOffset int32) ([]Clause, int64, error)
	searchClausesFunc func(ctx context.Context, modalityID int32, query string, pageSize, pageOffset int32) ([]Clause, int64, error)
}

func (m *mockClausesRepository) ListClauses(ctx context.Context, modalityID, pageSize, pageOffset int32) ([]Clause, int64, error) {
	if m.listClausesFunc != nil {
		return m.listClausesFunc(ctx, modalityID, pageSize, pageOffset)
	}
	return nil, 0, nil
}

func (m *mockClausesRepository) SearchClauses(ctx context.Context, modalityID int32, query string, pageSize, pageOffset int32) ([]Clause, int64, error) {
	if m.searchClausesFunc != nil {
		return m.searchClausesFunc(ctx, modalityID, query, pageSize, pageOffset)
	}
	return nil, 0, nil
}

func TestService_ListClauses(t *testing.T) {
	repositoryErr := errors.New("repo fail")

	tests := []struct {
		name       string
		input      ListClausesInput
		repo       *mockClausesRepository
		wantMeta   ListClausesMeta
		wantData   []Clause
		wantErr    bool
		wantOffset int32
	}{
		{
			name:  "returns data and correct meta",
			input: ListClausesInput{ModalityID: 1, Page: 2, PageSize: 20},
			repo: &mockClausesRepository{
				listClausesFunc: func(ctx context.Context, modalityID, pageSize, pageOffset int32) ([]Clause, int64, error) {
					return []Clause{{ClauseID: 1, Code: "pets", SortOrder: 1}}, 45, nil
				},
			},
			wantData:   []Clause{{ClauseID: 1, Code: "pets", SortOrder: 1}},
			wantMeta:   ListClausesMeta{Total: 45, Page: 2, PageSize: 20, TotalPages: 3, Query: nil},
			wantOffset: 20,
		},
		{
			name:  "empty result returns empty slice and zero meta",
			input: ListClausesInput{ModalityID: 2, Page: 1, PageSize: 20},
			repo: &mockClausesRepository{
				listClausesFunc: func(ctx context.Context, modalityID, pageSize, pageOffset int32) ([]Clause, int64, error) {
					return []Clause{}, 0, nil
				},
			},
			wantData:   []Clause{},
			wantMeta:   ListClausesMeta{Total: 0, Page: 1, PageSize: 20, TotalPages: 0, Query: nil},
			wantOffset: 0,
		},
		{
			name:  "repository error is wrapped",
			input: ListClausesInput{ModalityID: 3, Page: 1, PageSize: 20},
			repo: &mockClausesRepository{
				listClausesFunc: func(ctx context.Context, modalityID, pageSize, pageOffset int32) ([]Clause, int64, error) {
					return nil, 0, repositoryErr
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calledOffset int32 = -1
			// wrap repo to capture offset
			repo := tt.repo
			orig := repo.listClausesFunc
			repo.listClausesFunc = func(ctx context.Context, modalityID, pageSize, pageOffset int32) ([]Clause, int64, error) {
				calledOffset = pageOffset
				return orig(ctx, modalityID, pageSize, pageOffset)
			}

			svc := NewService(repo)
			got, err := svc.ListClauses(context.Background(), tt.input)

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
				t.Fatalf("expected non-nil data slice")
			}
			if !reflect.DeepEqual(got.Data, tt.wantData) {
				t.Fatalf("data mismatch: got %#v want %#v", got.Data, tt.wantData)
			}
			if !reflect.DeepEqual(got.Meta, tt.wantMeta) {
				t.Fatalf("meta mismatch: got %#v want %#v", got.Meta, tt.wantMeta)
			}
			if tt.wantOffset >= 0 && calledOffset != tt.wantOffset {
				t.Fatalf("offset mismatch: got %d want %d", calledOffset, tt.wantOffset)
			}
		})
	}
}

func TestService_SearchClauses(t *testing.T) {
	repositoryErr := errors.New("repo fail")

	tests := []struct {
		name     string
		input    SearchClausesInput
		repo     *mockClausesRepository
		wantMeta ListClausesMeta
		wantData []Clause
		wantErr  bool
	}{
		{
			name:  "returns data with query in meta",
			input: SearchClausesInput{ModalityID: 1, Query: "pets", Page: 1, PageSize: 20},
			repo: &mockClausesRepository{
				searchClausesFunc: func(ctx context.Context, modalityID int32, query string, pageSize, pageOffset int32) ([]Clause, int64, error) {
					return []Clause{{ClauseID: 1, Code: "pets"}}, 2, nil
				},
			},
			wantData: []Clause{{ClauseID: 1, Code: "pets"}},
			wantMeta: ListClausesMeta{Total: 2, Page: 1, PageSize: 20, TotalPages: 1, Query: func() *string { s := "pets"; return &s }()},
		},
		{
			name:  "empty search returns empty slice with query",
			input: SearchClausesInput{ModalityID: 1, Query: "dogs", Page: 1, PageSize: 20},
			repo: &mockClausesRepository{
				searchClausesFunc: func(ctx context.Context, modalityID int32, query string, pageSize, pageOffset int32) ([]Clause, int64, error) {
					return []Clause{}, 0, nil
				},
			},
			wantData: []Clause{},
			wantMeta: ListClausesMeta{Total: 0, Page: 1, PageSize: 20, TotalPages: 0, Query: func() *string { s := "dogs"; return &s }()},
		},
		{
			name:  "repository error is wrapped",
			input: SearchClausesInput{ModalityID: 1, Query: "fail", Page: 1, PageSize: 20},
			repo: &mockClausesRepository{
				searchClausesFunc: func(ctx context.Context, modalityID int32, query string, pageSize, pageOffset int32) ([]Clause, int64, error) {
					return nil, 0, repositoryErr
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)
			got, err := svc.SearchClauses(context.Background(), tt.input)

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
				t.Fatalf("expected non-nil data slice")
			}
			if !reflect.DeepEqual(got.Data, tt.wantData) {
				t.Fatalf("data mismatch: got %#v want %#v", got.Data, tt.wantData)
			}
			if tt.wantMeta.Query == nil {
				if got.Meta.Query != nil {
					t.Fatalf("expected nil query in meta, got %v", *got.Meta.Query)
				}
			} else {
				if got.Meta.Query == nil || *got.Meta.Query != *tt.wantMeta.Query {
					t.Fatalf("query mismatch: got %v want %v", got.Meta.Query, tt.wantMeta.Query)
				}
			}
		})
	}
}

func TestResolveOffsetAndTotalPages(t *testing.T) {
	if got := resolveOffset(1, 20); got != 0 {
		t.Fatalf("offset(1,20) = %d, want 0", got)
	}
	if got := resolveOffset(2, 20); got != 20 {
		t.Fatalf("offset(2,20) = %d, want 20", got)
	}
	if got := resolveOffset(3, 10); got != 20 {
		t.Fatalf("offset(3,10) = %d, want 20", got)
	}

	if got := resolveTotalPages(0, 20); got != 0 {
		t.Fatalf("totalPages(0,20) = %d, want 0", got)
	}
	if got := resolveTotalPages(20, 20); got != 1 {
		t.Fatalf("totalPages(20,20) = %d, want 1", got)
	}
	if got := resolveTotalPages(21, 20); got != 2 {
		t.Fatalf("totalPages(21,20) = %d, want 2", got)
	}
	if got := resolveTotalPages(1, 20); got != 1 {
		t.Fatalf("totalPages(1,20) = %d, want 1", got)
	}
	if got := resolveTotalPages(100, 20); got != 5 {
		t.Fatalf("totalPages(100,20) = %d, want 5", got)
	}
	if got := resolveTotalPages(101, 20); got != 6 {
		t.Fatalf("totalPages(101,20) = %d, want 6", got)
	}
}
