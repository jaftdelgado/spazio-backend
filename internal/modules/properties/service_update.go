package properties

import (
	"context"
	"fmt"
)

func (s *service) UpdateProperty(ctx context.Context, propertyUUID string, input UpdatePropertyInput) (UpdatePropertyResult, error) {
	if err := requireAdminActor(input.Actor); err != nil {
		return UpdatePropertyResult{}, err
	}

	res, err := s.repository.UpdateProperty(ctx, propertyUUID, input)
	if err != nil {
		return UpdatePropertyResult{}, fmt.Errorf("update property: %w", err)
	}

	return res, nil
}
