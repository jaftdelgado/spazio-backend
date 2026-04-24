package properties

import (
	"context"
	"fmt"
)

type service struct {
	repository PropertyRepository
}

// NewService builds a property service implementation.
func NewService(repository PropertyRepository) PropertyService {
	return &service{repository: repository}
}

func (s *service) CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
	result, err := s.repository.CreateProperty(ctx, input)
	if err != nil {
		return CreatePropertyResult{}, fmt.Errorf("create property: %w", err)
	}

	return result, nil
}
