package locations

import (
	"context"
	"fmt"
)

type service struct {
	repository LocationsRepository
}

func NewService(repository LocationsRepository) LocationsService {
	return &service{
		repository: repository,
	}
}

func (s *service) ListCountries(ctx context.Context) (ListCountriesResult, error) {
	countries, err := s.repository.ListCountries(ctx)
	if err != nil {
		return ListCountriesResult{}, fmt.Errorf("list countries: %w", err)
	}

	if countries == nil {
		countries = []Country{}
	}

	return ListCountriesResult{
		Data: countries,
	}, nil
}

func (s *service) ListStates(ctx context.Context, input ListStatesInput) (ListStatesResult, error) {
	states, err := s.repository.ListStates(ctx, input.CountryID)
	if err != nil {
		return ListStatesResult{}, fmt.Errorf("list states: %w", err)
	}

	if states == nil {
		states = []State{}
	}

	return ListStatesResult{
		Data: states,
	}, nil
}

func (s *service) ListCities(ctx context.Context, input ListCitiesInput) (ListCitiesResult, error) {
	cities, total, err := s.repository.ListCities(ctx, input)
	if err != nil {
		return ListCitiesResult{}, fmt.Errorf("list cities: %w", err)
	}

	if cities == nil {
		cities = []City{}
	}

	return ListCitiesResult{
		Data: cities,
		Meta: ListCitiesMeta{
			Total:      total,
			Page:       input.Page,
			PageSize:   input.PageSize,
			TotalPages: calculateTotalPages(total, input.PageSize),
		},
	}, nil
}

func calculateTotalPages(total int64, pageSize int32) int32 {
	if total == 0 || pageSize <= 0 {
		return 0
	}

	return int32((total + int64(pageSize) - 1) / int64(pageSize))
}
