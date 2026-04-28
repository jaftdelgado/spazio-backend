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
