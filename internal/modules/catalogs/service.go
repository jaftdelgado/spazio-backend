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

func (s *service) ListRentPeriods(ctx context.Context, propertyTypeID int32) (ListRentPeriodsResult, error) {
	items, err := s.repository.ListRentPeriodsByPropertyType(ctx, propertyTypeID)
	if err != nil {
		return ListRentPeriodsResult{}, fmt.Errorf("list rent periods by property type: %w", err)
	}

	return ListRentPeriodsResult{Data: items}, nil
}

func (s *service) ListOrientations(ctx context.Context) (ListOrientationsResult, error) {
	items, err := s.repository.ListOrientations(ctx)
	if err != nil {
		return ListOrientationsResult{}, fmt.Errorf("list orientations: %w", err)
	}

	return ListOrientationsResult{Data: items}, nil
}
