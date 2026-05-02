package properties

import (
	"context"
	"fmt"
)

func (s *service) UpdateProperty(ctx context.Context, propertyUUID string, input UpdatePropertyInput) (UpdatePropertyResult, error) {
	// Basic business validations that require service level can be added here.
	// For now assume handler performed field-level validations and delegate persistence to repository.

	res, err := s.repository.UpdateProperty(ctx, propertyUUID, input)
	if err != nil {
		return UpdatePropertyResult{}, fmt.Errorf("update property: %w", err)
	}

	return res, nil
}
