package properties

import (
	"context"
	"fmt"
)

func (s *service) GetServices(ctx context.Context, propertyUUID string) (GetPropertyServicesResult, error) {
	result, err := s.repository.GetPropertyServices(ctx, propertyUUID)
	if err != nil {
		return GetPropertyServicesResult{}, fmt.Errorf("get property services: %w", err)
	}

	return result, nil
}

func (s *service) UpdateServices(ctx context.Context, propertyUUID string, input UpdatePropertyServicesInput) error {
	if err := validateServiceIDs(input.ServiceIDs); err != nil {
		return ValidationError{Message: err.Error()}
	}

	if err := s.repository.UpdatePropertyServices(ctx, propertyUUID, input); err != nil {
		return fmt.Errorf("update property services: %w", err)
	}

	return nil
}
