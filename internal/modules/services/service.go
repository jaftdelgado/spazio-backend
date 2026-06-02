package services

import (
	"context"
	"fmt"
)

type service struct {
	repository ServicesRepository
}

// NewService builds a services catalog service implementation.
func NewService(repository ServicesRepository) ServicesService {
	return &service{repository: repository}
}

func (s *service) ListPopularServices(ctx context.Context, input ListPopularInput) (ListServicesResult, error) {
	items, total, err := s.repository.ListPopularServices(ctx, input)
	if err != nil {
		return ListServicesResult{}, fmt.Errorf("list popular services: %w", err)
	}

	return ListServicesResult{
		Data: items,
		Meta: ListServicesMeta{
			Total:      total,
			Shown:      len(items),
			Page:       input.Page,
			PageSize:   input.PageSize,
			TotalPages: calculateTotalPages(total, input.PageSize),
		},
	}, nil
}

func (s *service) SearchServices(ctx context.Context, input SearchInput) (ListServicesResult, error) {
	items, total, err := s.repository.SearchServices(ctx, input)
	if err != nil {
		return ListServicesResult{}, fmt.Errorf("search services: %w", err)
	}

	return ListServicesResult{
		Data: items,
		Meta: ListServicesMeta{
			Total:      total,
			Shown:      len(items),
			Page:       input.Page,
			PageSize:   input.PageSize,
			TotalPages: calculateTotalPages(total, input.PageSize),
			Query:      &input.Query,
		},
	}, nil
}

func calculateTotalPages(total int64, pageSize int32) int32 {
	if total == 0 || pageSize <= 0 {
		return 0
	}

	return int32((total + int64(pageSize) - 1) / int64(pageSize))
}
