package properties

import (
	"context"
	"fmt"
)

func (s *service) GetProperty(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
	result, err := s.repository.GetProperty(ctx, propertyUUID)
	if err != nil {
		return GetPropertyResult{}, fmt.Errorf("get property: %w", err)
	}

	return result, nil
}
