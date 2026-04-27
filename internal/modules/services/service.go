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
	items, total, err := s.repository.ListPopularServices(ctx, input.Limit)
	if err != nil {
		return ListServicesResult{}, fmt.Errorf("list popular services: %w", err)
	}

	return ListServicesResult{
		Data: items,
		Meta: ListServicesMeta{
			Total: total,
			Shown: len(items),
		},
	}, nil
}

func (s *service) SearchServices(ctx context.Context, input SearchInput) (ListServicesResult, error) {
	items, total, err := s.repository.SearchServices(ctx, input.Query, input.Limit)
	if err != nil {
		return ListServicesResult{}, fmt.Errorf("search services: %w", err)
	}

	return ListServicesResult{
		Data: items,
		Meta: ListServicesMeta{
			Total: total,
			Shown: len(items),
			Query: &input.Query,
		},
	}, nil
}
