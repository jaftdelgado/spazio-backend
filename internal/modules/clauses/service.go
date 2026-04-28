package clauses

import (
	"context"
	"fmt"
)

type service struct {
	repository ClausesRepository
}

// NewService builds a clauses catalog service implementation.
func NewService(repository ClausesRepository) ClausesService {
	return &service{repository: repository}
}

func (s *service) ListClauses(ctx context.Context, input ListClausesInput) (ListClausesResult, error) {
	items, total, err := s.repository.ListClauses(ctx, input.ModalityID, input.PageSize, resolveOffset(input.Page, input.PageSize))
	if err != nil {
		return ListClausesResult{}, fmt.Errorf("list clauses: %w", err)
	}

	return ListClausesResult{
		Data: items,
		Meta: ListClausesMeta{
			Total:      total,
			Page:       input.Page,
			PageSize:   input.PageSize,
			TotalPages: resolveTotalPages(total, input.PageSize),
		},
	}, nil
}

func (s *service) SearchClauses(ctx context.Context, input SearchClausesInput) (ListClausesResult, error) {
	items, total, err := s.repository.SearchClauses(ctx, input.ModalityID, input.Query, input.PageSize, resolveOffset(input.Page, input.PageSize))
	if err != nil {
		return ListClausesResult{}, fmt.Errorf("search clauses: %w", err)
	}

	return ListClausesResult{
		Data: items,
		Meta: ListClausesMeta{
			Total:      total,
			Page:       input.Page,
			PageSize:   input.PageSize,
			TotalPages: resolveTotalPages(total, input.PageSize),
			Query:      &input.Query,
		},
	}, nil
}

func resolveOffset(page, pageSize int32) int32 {
	return (page - 1) * pageSize
}

func resolveTotalPages(total int64, pageSize int32) int32 {
	if total == 0 {
		return 0
	}

	return int32((total + int64(pageSize) - 1) / int64(pageSize))
}
