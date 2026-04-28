package catalogs

import (
	"context"
	"fmt"
)

type service struct {
	repository CatalogsRepository
}

// NewService builds a catalogs service implementation.
func NewService(repository CatalogsRepository) CatalogsService {
	return &service{repository: repository}
}

func (s *service) ListModalities(ctx context.Context) (ListModalitiesResult, error) {
	items, err := s.repository.ListModalities(ctx)
	if err != nil {
		return ListModalitiesResult{}, fmt.Errorf("list modalities: %w", err)
	}

	return ListModalitiesResult{Data: items}, nil
}

func (s *service) ListPropertyTypes(ctx context.Context) (ListPropertyTypesResult, error) {
	items, err := s.repository.ListPropertyTypes(ctx)
	if err != nil {
		return ListPropertyTypesResult{}, fmt.Errorf("list property types: %w", err)
	}

	return ListPropertyTypesResult{Data: items}, nil
}
