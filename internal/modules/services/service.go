package services

import (
	"context"
	"fmt"
	"strings"
)

type service struct {
	repository ServicesRepository
}

// NewService builds a services catalog service implementation.
func NewService(repository ServicesRepository) ServicesService {
	return &service{repository: repository}
}

func (s *service) ListServices(ctx context.Context, input ListServicesInput) (ListServicesResult, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
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

	items, total, err := s.repository.SearchServices(ctx, query, input.Limit)
	if err != nil {
		return ListServicesResult{}, fmt.Errorf("search services: %w", err)
	}

	return ListServicesResult{
		Data: items,
		Meta: ListServicesMeta{
			Total: total,
			Shown: len(items),
			Query: &query,
		},
	}, nil
}
