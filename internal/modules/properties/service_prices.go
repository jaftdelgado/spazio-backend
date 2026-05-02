package properties

import (
	"context"
	"fmt"
)

func (s *service) GetPrices(ctx context.Context, propertyUUID string) (GetPropertyPricesResult, error) {
	result, err := s.repository.GetPropertyPrices(ctx, propertyUUID)
	if err != nil {
		return GetPropertyPricesResult{}, fmt.Errorf("get property prices: %w", err)
	}

	return result, nil
}

func (s *service) UpdatePrices(ctx context.Context, propertyUUID string, input UpdatePropertyPricesInput) error {
	if err := validatePriceInputs(input); err != nil {
		return ValidationError{Message: err.Error()}
	}

	if err := s.repository.UpdatePropertyPrices(ctx, propertyUUID, input); err != nil {
		return fmt.Errorf("update property prices: %w", err)
	}

	return nil
}
